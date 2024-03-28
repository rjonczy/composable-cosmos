package v6

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v6 "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/migrations/v6"
)

const (
	// UpgradeName defines the on-chain upgrade name for the SimApp v6 upgrade.
	UpgradeName = "v6"
)

// CreateUpgradeHandler creates an upgrade handler for the v6 SimApp upgrade.
// NOTE: The v6.MigrateICS27ChannelCapabiliity function can be omitted if chains do not yet implement an ICS27 controller module
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	cdc codec.BinaryCodec,
	capabilityStoreKey *storetypes.KVStoreKey,
	capabilityKeeper *capabilitykeeper.Keeper,
	moduleName string,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkctx := sdk.UnwrapSDKContext(ctx)
		if err := v6.MigrateICS27ChannelCapability(sdkctx, cdc, capabilityStoreKey, capabilityKeeper, moduleName); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
