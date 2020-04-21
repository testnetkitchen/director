package core

import (
	"director/m/v2/state"
	"time"

	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	// SubscribeTimeout is the maximum time we wait to subscribe for an event.
	// must be less than the server's write timeout (see rpcserver.DefaultConfig)
	SubscribeTimeout = 5 * time.Second
)

//----------------------------------------------
// These package level globals come with setters
// that are expected to be called only once, on startup

var (
	// interfaces defined in types and above

	stateMachine *state.Machine

	logger log.Logger

	config cfg.RPCConfig
)

// SetLogger sets the RPC logger
func SetLogger(l log.Logger) {
	logger = l
}

// SetStateMachine sets the RPC state machine
func SetStateMachine(m *state.Machine) {
	stateMachine = m
}

// SetConfig sets an RPCConfig.
func SetConfig(c cfg.RPCConfig) {
	config = c
}
