#!/bin/bash
set -euoxa pipefail


_delete_balance() {
  declare -r address="$1"

  jq 'del(.app_state.bank.balances[] | select(.address == "'"$address"'"))' genesis_modified.json >genesis_modified_tmp.json && mv genesis_modified_tmp.json genesis_modified.json
}

_nullify_minter() {
  jq '.app_state.mint.minter.norm_time_passed="0.470000000000000000" |
    .app_state.mint.minter.total_minted="0"' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
}

# when supply is [] in genesis, it will be auto calculated on chain's first start
_cleanup_supply_and_godify() {
  declare -r code_upload_addr_new=$1

  jq '.app_state.bank.supply = []' genesis_modified.json >genesis_modified_tmp.json && mv genesis_modified_tmp.json genesis_modified.json

  jq --arg address $code_upload_addr_new '.app_state.bank.balances |= map(if .address == $address and any(.coins[]; .denom == "unls") then .coins |= map(if .denom == "unls" then .amount =  "99999999999999999999999999999" else . end) else . end)' genesis_modified.json >genesis_modified_tmp.json && mv genesis_modified_tmp.json genesis_modified.json

  jq '.app_state.gov.voting_params.voting_period="300s" |
    .app_state.wasm.params.code_upload_access.permission="AnyOfAddresses" |
    .app_state.wasm.params.code_upload_access.addresses=["'$code_upload_addr_new'"]' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
}

# Remove existing mentions of validators in the genesis
jq 'del(.app_state.genutil.gen_txs[])' test-exported-genesis.json >genesis_modified.json
jq 'del(.app_state.staking.redelegations[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.staking.delegations[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.staking.validators[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.staking.unbonding_delegations[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.staking.last_validator_powers[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.distribution.delegator_starting_infos[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.distribution.validator_slash_events[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.distribution.validator_current_rewards[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.distribution.validator_accumulated_commissions[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.distribution.validator_historical_rewards[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.distribution.delegator_withdraw_infos[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.distribution.delegator_withdraw_infos[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.validators[])' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq --arg chain_id "test-1" '.chain_id = $chain_id' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq '.app_state.staking.params.min_commission_rate = "0.000000000000000000"' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq '.app_state.gov.proposals = []' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq '.app_state.contractmanager.failures_list = []' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq '.app_state.wasm.codes |= map(.code_info.instantiate_config |= del(.address))' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json
jq 'del(.app_state.wasm.params.code_upload_access.address)' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json

jq '.app_state.gov += {
  "params": {
    "min_deposit": [
      {
        "denom": "ppica",
        "amount": "10000000"
      }
    ],
    "max_deposit_period": "172800s",
    "voting_period": "172800s",
    "quorum": "0.334000000000000000",
    "threshold": "0.500000000000000000",
    "veto_threshold": "0.334000000000000000",
    "min_initial_deposit_ratio": "0.000000000000000000",
    "burn_vote_quorum": false,
    "burn_proposal_deposit_prevote": false,
    "burn_vote_veto": true
  }
}' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json

jq '.app_state.ibc.connection_genesis.connections |= map(select(.client_id | contains("localhost") | not))' genesis_modified.json >tmp_genesis.json && mv tmp_genesis.json genesis_modified.json


# prepare minter for new local chain
# _nullify_minter

# delete balances for bonded and not bonded tokens
BONDED_TOKENS_ADDR=$(jq -r '.app_state.auth.accounts[] | select(.name == "bonded_tokens_pool") | .base_account.address' genesis_modified.json)
NOT_BONDED_TOKENS_ADDR=$(jq -r '.app_state.auth.accounts[] | select(.name == "not_bonded_tokens_pool") | .base_account.address' genesis_modified.json)
_delete_balance "$BONDED_TOKENS_ADDR"
_delete_balance "$NOT_BONDED_TOKENS_ADDR"

# Add balances to god acc, let god acc have permissions for storing contracts, lower proposals voting period, clean supply []
GOD_ADDRESS='centauri1hj5fveer5cjtn4wd6wstzugjfdxzl0xpzxlwgs'
_cleanup_supply_and_godify "$GOD_ADDRESS"