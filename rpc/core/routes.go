package core

import (
	rpc "github.com/tendermint/tendermint/rpc/lib/server"
)

// Routes defines RPC endpoints
var Routes = map[string]*rpc.RPCFunc{
	// API
	"register": rpc.NewRPCFunc(Register, "chain_id,name,pub_key,net_address"),
	"genesis":  rpc.NewRPCFunc(Genesis, "chain_id"),
	"addrbook": rpc.NewRPCFunc(AddressBook, "chain_id"),
}
