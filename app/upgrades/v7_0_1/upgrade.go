package v7_0_1

import (
	"context"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"encoding/hex"
	"fmt"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/notional-labs/composable/v6/app/keepers"
	"github.com/notional-labs/composable/v6/app/upgrades"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ upgrades.BaseAppParamManager,
	codec codec.Codec,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(goCtx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		fmt.Println("start v7.0.1 upgrade")
		ctx := sdk.UnwrapSDKContext(goCtx)
		store := runtime.NewKVStoreService(keepers.GetKVStoreKey()[types.StoreKey]).OpenKVStore(ctx)
		checksumStr, err := hex.DecodeString("58c7623a3ab78f4cb2e4c5d02876ac36c3b38bb472118173a7ec7faa688a66d2")

		checksum := types.Checksums{Checksums: [][]byte{checksumStr}}
		bz, err := codec.Marshal(&checksum)
		if err != nil {
			panic(err)
		}
		err = store.Set([]byte(types.KeyChecksums), bz)
		if err != nil {
			panic(err)
		}
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
