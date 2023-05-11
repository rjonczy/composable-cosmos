package transfermiddleware_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	customibctesting "github.com/notional-labs/banksy/v2/app/ibctesting"
	"github.com/stretchr/testify/suite"
)

// TODO: use testsuite here.
type TransferMiddlewareTestSuite struct {
	suite.Suite

	coordinator *customibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *customibctesting.TestChain
	chainB *customibctesting.TestChain
	chainC *customibctesting.TestChain
}

func (suite *TransferMiddlewareTestSuite) SetupTest() {
	suite.coordinator = customibctesting.NewCoordinator(suite.T(), 4)
	suite.chainA = suite.coordinator.GetChain(customibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(customibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(customibctesting.GetChainID(3))

}

func NewTransferPath(chainA, chainB *customibctesting.TestChain) *customibctesting.Path {
	path := customibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = customibctesting.TransferPort
	path.EndpointB.ChannelConfig.PortID = customibctesting.TransferPort
	path.EndpointA.ChannelConfig.Version = ibctransfertypes.Version
	path.EndpointB.ChannelConfig.Version = ibctransfertypes.Version

	return path

}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(TransferMiddlewareTestSuite))
}

var keyCounter uint64

// we need to make this deterministic (same every test run), as encoded address size and thus gas cost,
// depends on the actual bytes (due to ugly CanonicalAddress encoding)
func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	keyCounter++
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, keyCounter)

	key := ed25519.GenPrivKeyFromSecret(seed)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

func RandomAccountAddress(_ testing.TB) sdk.AccAddress {
	_, _, addr := keyPubAddr()
	return addr
}

func RandomBech32AccountAddress(t testing.TB) string {
	return RandomAccountAddress(t).String()
}

func (suite *TransferMiddlewareTestSuite) TestSendTransfer() {
	var (
		transferAmount = sdk.NewInt(1000000000)
		// when transfer via sdk transfer from A (module) -> B (contract)
		timeoutHeight = clienttypes.NewHeight(1, 110)
		pathAtoB      *customibctesting.Path
		pathBtoC      *customibctesting.Path
		path          *customibctesting.Path
		srcPort       string
		srcChannel    string
		chain         *customibctesting.TestChain
		// pathBtoC      = NewTransferPath(suite.chainB, suite.chainC)
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"Receiver is source chain",
			func() {
				path = pathAtoB
				srcPort = pathAtoB.EndpointB.ChannelConfig.PortID
				srcChannel = pathAtoB.EndpointB.ChannelID
				chain = suite.chainA
			},
		},
		{
			"Receiver is sink chain",
			func() {
				path = pathBtoC
				srcPort = pathBtoC.EndpointB.ChannelConfig.PortID
				srcChannel = pathBtoC.EndpointB.ChannelID
				chain = suite.chainC
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			pathAtoB = NewTransferPath(suite.chainA, suite.chainB)
			pathBtoC = NewTransferPath(suite.chainB, suite.chainC)
			suite.coordinator.Setup(pathAtoB)
			suite.coordinator.Setup(pathBtoC)
			// Add parachain token info
			chainBtransMiddleware := suite.chainB.TransferMiddleware()
			err := chainBtransMiddleware.AddParachainIBCInfo(suite.chainB.GetContext(), "ibc/C053D637CCA2A2BA030E2C5EE1B28A16F71CCB0E45E8BE52766DC1B241B77878", pathAtoB.EndpointB.ChannelID, sdk.DefaultBondDenom)
			suite.Require().NoError(err)
			// send coin from A to B

			msg := ibctransfertypes.NewMsgTransfer(
				pathAtoB.EndpointA.ChannelConfig.PortID,
				pathAtoB.EndpointA.ChannelID,
				sdk.NewCoin(sdk.DefaultBondDenom, transferAmount),
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				timeoutHeight,
				0,
				"",
			)
			_, err = suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err)
			suite.Require().NoError(err, pathAtoB.EndpointB.UpdateClient())

			// then
			suite.Require().Equal(1, len(suite.chainA.PendingSendPackets))
			suite.Require().Equal(0, len(suite.chainB.PendingSendPackets))

			// and when relay to chain B and handle Ack on chain A
			err = suite.coordinator.RelayAndAckPendingPackets(pathAtoB)
			suite.Require().NoError(err)

			// then
			suite.Require().Equal(0, len(suite.chainA.PendingSendPackets))
			suite.Require().Equal(0, len(suite.chainB.PendingSendPackets))

			// balance := suite.chainB.AllBalances(suite.chainB.SenderAccount.GetAddress())

			tc.malleate()

			testAcc2 := RandomAccountAddress(suite.T())
			msg = ibctransfertypes.NewMsgTransfer(
				srcPort,
				srcChannel,
				sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000)),
				suite.chainB.SenderAccount.GetAddress().String(),
				chain.SenderAccount.GetAddress().String(),
				timeoutHeight,
				0,
				"",
			)
			_, err = suite.chainB.SendMsgs(msg)
			suite.Require().NoError(err)
			suite.Require().NoError(err, path.EndpointB.UpdateClient())

			// and when relay to chain B and handle Ack on chain A
			err = suite.coordinator.RelayAndAckPendingPackets(path)
			suite.Require().NoError(err)

			fmt.Printf("-------%v\n", chain.AllBalances(testAcc2))
		})
	}
}

// TODO: use testsuite here.
func (suite *TransferMiddlewareTestSuite) TestOnrecvPacket() {
	var (
		transferAmount = sdk.NewInt(1000000000)
		// when transfer via sdk transfer from A (module) -> B (contract)
		coinToSendToB = sdk.NewCoin(sdk.DefaultBondDenom, transferAmount)
		timeoutHeight = clienttypes.NewHeight(1, 110)
	)
	var (
		expChainBBalanceDiff sdk.Coin
		path                 = NewTransferPath(suite.chainA, suite.chainB)
	)

	testCases := []struct {
		name                 string
		expChainABalanceDiff sdk.Coin
		malleate             func()
	}{
		{
			"Transfer with no pre-set ParachainIBCTokenInfo",
			sdk.NewCoin(sdk.DefaultBondDenom, transferAmount),
			func() {
				expChainBBalanceDiff = ibctransfertypes.GetTransferCoin(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, coinToSendToB.Denom, transferAmount)

			},
		},
		{
			"Transfer with pre-set ParachainIBCTokenInfo",
			sdk.NewCoin(sdk.DefaultBondDenom, transferAmount),
			func() {
				// Add parachain token info
				chainBtransMiddleware := suite.chainB.TransferMiddleware()
				expChainBBalanceDiff = sdk.NewCoin(sdk.DefaultBondDenom, transferAmount)
				err := chainBtransMiddleware.AddParachainIBCInfo(suite.chainB.GetContext(), "ibc/C053D637CCA2A2BA030E2C5EE1B28A16F71CCB0E45E8BE52766DC1B241B77878", "channel-0", sdk.DefaultBondDenom)
				suite.Require().NoError(err)
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			tc.malleate()

			originalChainABalance := suite.chainA.AllBalances(suite.chainA.SenderAccount.GetAddress())
			// chainB.SenderAccount: 10000000000000000000stake
			originalChainBBalance := suite.chainB.AllBalances(suite.chainB.SenderAccount.GetAddress())

			fmt.Println("chainB.AllBalances(chainB.SenderAccount.GetAddress())", suite.chainB.AllBalances(suite.chainB.SenderAccount.GetAddress()))
			msg := ibctransfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coinToSendToB, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
			_, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err)
			suite.Require().NoError(err, path.EndpointB.UpdateClient())

			// then
			suite.Require().Equal(1, len(suite.chainA.PendingSendPackets))
			suite.Require().Equal(0, len(suite.chainB.PendingSendPackets))

			// and when relay to chain B and handle Ack on chain A
			err = suite.coordinator.RelayAndAckPendingPackets(path)
			suite.Require().NoError(err)

			// then
			suite.Require().Equal(0, len(suite.chainA.PendingSendPackets))
			suite.Require().Equal(0, len(suite.chainB.PendingSendPackets))

			// and source chain balance was decreased
			newChainABalance := suite.chainA.AllBalances(suite.chainA.SenderAccount.GetAddress())
			suite.Require().Equal(originalChainABalance.Sub(tc.expChainABalanceDiff), newChainABalance)

			// and dest chain balance contains voucher
			expBalance := originalChainBBalance.Add(expChainBBalanceDiff)
			gotBalance := suite.chainB.AllBalances(suite.chainB.SenderAccount.GetAddress())
			fmt.Println("expBalance", expBalance)
			fmt.Println("gotBalance", gotBalance)
			suite.Require().Equal(expBalance, gotBalance)
		})
	}
}
