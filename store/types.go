package store

import (
	"director/m/v2/config"
	"director/m/v2/types"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	dbm "github.com/tendermint/tm-db"
	"sync"
	"time"
)

// TestnetDB struct for database
type TestnetDB struct {
	db        dbm.DB
	testnets  map[string]*TestnetConfig
	startTime time.Time
	config    map[string]config.TestnetsTOMLConfig

	// Use this mutex to indicate access to testnets (Lock or RLock)
	mtx sync.RWMutex
}

// TestnetConfig entry in the database
type TestnetConfig struct {
	State       types.ServerState `json:"state"`
	Validators  map[string]*ValidatorConfig
	Genesis     *tmctypes.ResultGenesis
	AddressBook *AddrBookJSON
}

// ValidatorConfig entry in the database
type ValidatorConfig struct {
	NetAddress *types.NetAddress `json:"net_address"`
	Name       string            `json:"name"`
	PubKey     string            `json:"pub_key"`
}
