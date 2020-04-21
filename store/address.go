package store

import (
	"director/m/v2/types"
	"time"
)

////////////////////////////////////////////////////////////////
// From: github.com/tendermint/tendermint/p2p
////////////////////////////////////////////////////////////////

const (
	bucketTypeNew = 0x01
	bucketTypeOld = 0x02
)

////////////////////////////////////////////////////////////////
// From: https://github.com/tendermint/tendermint/p2p/pex/known_address.go
////////////////////////////////////////////////////////////////

// knownAddress tracks information about a known network address
// that is used to determine how viable an address is.
type knownAddress struct {
	Addr        *types.NetAddress `json:"addr"`
	Src         *types.NetAddress `json:"src"`
	Buckets     []int             `json:"buckets"`
	Attempts    int32             `json:"attempts"`
	BucketType  byte              `json:"bucket_type"`
	LastAttempt time.Time         `json:"last_attempt"`
	LastSuccess time.Time         `json:"last_success"`
}

////////////////////////////////////////////////////////////////
// From: https://github.com/tendermint/tendermint/p2p/pex/file.go
////////////////////////////////////////////////////////////////

// AddrBookJSON is addrBookJSON, only it's exported
type AddrBookJSON struct {
	Key   string          `json:"key"`
	Addrs []*knownAddress `json:"addrs"`
}
