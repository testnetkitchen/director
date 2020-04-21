package core

import (
	"director/m/v2/store"
	"director/m/v2/types"
	"encoding/base64"
	"errors"
	"github.com/tendermint/tendermint/crypto/ed25519"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
)

// Register a node for a testnet
func Register(ctx *rpctypes.Context, chainID string, name string, pubKey string, netAddress string) (*rpctypes.RPCError, error) {
	// Check ed25519 compatibiliy
	pubBytes, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return nil, err
	}
	if len(pubBytes) != ed25519.PubKeyEd25519Size {
		return nil, errors.New("invalid ed25519 public key length")
	}

	// Validate network address
	netAddressStruct, err := types.NewNetAddressString(netAddress)
	if err != nil {
		return nil, err
	}
	err = netAddressStruct.Valid()
	if err != nil {
		return nil, err
	}
	// Sync registration
	err = stateMachine.RegisterValidator(chainID, store.ValidatorConfig{
		NetAddress: netAddressStruct,
		Name:       name,
		PubKey:     pubKey,
	})
	if err != nil {
		return nil, err
	}
	// Success registering
	return &rpctypes.RPCError{
		Code:    0,
		Message: "Registered",
		Data:    "",
	}, nil
}
