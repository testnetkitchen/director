package core

import (
	"director/m/v2/store"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
)

// AddressBook returns an addrbook.json file for a testnet
func AddressBook(ctx *rpctypes.Context, chainID string) (*store.AddrBookJSON, error) {
	return stateMachine.GetAddressBook(chainID)
}
