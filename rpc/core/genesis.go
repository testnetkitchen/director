package core

import (
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
)

// Genesis returns the genesis.json file for a testnet
func Genesis(ctx *rpctypes.Context, chainID string) (*tmctypes.ResultGenesis, error) {
	return stateMachine.GetGenesis(chainID)
}
