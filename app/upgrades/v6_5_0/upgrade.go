package v6_5_0

import (
	"context"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/notional-labs/composable/v6/app/keepers"
	"github.com/notional-labs/composable/v6/app/upgrades"
	ibctransfermiddleware "github.com/notional-labs/composable/v6/x/ibctransfermiddleware/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ upgrades.BaseAppParamManager,
	_ codec.Codec,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkctx := sdk.UnwrapSDKContext(ctx)
		custommiddlewareparams := ibctransfermiddleware.DefaultGenesisState()
		keepers.IbcTransferMiddlewareKeeper.SetParams(sdkctx, custommiddlewareparams.Params)

		// remove broken proposals
		//BrokenProposals := [3]uint64{2, 6, 11}
		//for _, proposal_id := range BrokenProposals {
		//	_, err := keepers.GovKeeper.Proposals.Get(sdkctx, proposal_id)
		//	if err != nil {
		//		keepers.GovKeeper.DeleteProposal(sdkctx, proposal_id)
		//	}
		//
		//}

		// burn extra ppica in escrow account
		// this ppica is unused because it is a native token stored in escrow account
		// it was unnecessarily minted to match pica escrowed on picasso to ppica minted
		// in genesis, to make initial native ppica transferrable to picasso
		amount, ok := sdkmath.NewIntFromString("1066669217167120000000")
		if ok {
			coins := sdk.Coins{sdk.NewCoin("ppica", amount)}
			keepers.BankKeeper.SendCoinsFromAccountToModule(ctx, sdk.MustAccAddressFromBech32("centauri12k2pyuylm9t7ugdvz67h9pg4gmmvhn5vmvgw48"), "gov", coins)
			keepers.BankKeeper.BurnCoins(ctx, "gov", coins)
		}
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
