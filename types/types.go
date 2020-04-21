package types

import (
	"bytes"
	"github.com/tendermint/tendermint/p2p"
	"net"
)

// ServerState defines the state machine state type
type ServerState int

const (
	// Gather state
	Gather ServerState = iota
	// Serve state
	Serve
)

////////////////////////////////////////////////////////////////
// Copied from tendermint/tendermint@0.33.0/p2p/netaddress.go (because of https://gitlab.com/testnetkitchen/director/issues/11)
////////////////////////////////////////////////////////////////

// NetAddress defines information about a peer on the network
// including its ID, IP address, and port.
type NetAddress struct {
	ID   p2p.ID `json:"id"`
	IP   IP     `json:"ip"`
	Port uint16 `json:"port"`

	// TODO:
	// Name string `json:"name"` // optional DNS name

	// memoize .String()
	str string
}

// IP is a net.IP struct that implements MarshalJSON and UnmarshalJSON. This works around the problem that Amino does not check MarshalText.
type IP struct {
	net.IP
}

// MarshalJSON extends IP by using MarshalText
func (ip IP) MarshalJSON() ([]byte, error) {
	text, err := ip.MarshalText()
	if err != nil {
		return text, err
	}
	return append(append([]byte{'"'}, text...), '"'), nil
}

// UnmarshalJSON extends IP by using UnmarshalText.
func (ip *IP) UnmarshalJSON(text []byte) error {
	trimmed := bytes.Trim(text, "\"")
	return ip.UnmarshalText(trimmed)
}

// NewNetAddressString returns a new NetAddress using the provided address in
// the form of "ID@IP:Port".
// Also resolves the host if host is not an IP.
// Errors are of type ErrNetAddressXxx where Xxx is in (NoID, Invalid, Lookup)
func NewNetAddressString(addr string) (*NetAddress, error) {
	// Convert NetAddress to p2p.NetAddress
	original, err := p2p.NewNetAddressString(addr)
	if err != nil {
		return nil, err
	}
	converted := &NetAddress{
		ID: original.ID,
		IP: IP{
			IP: original.IP,
		},
		Port: original.Port,
		str:  "",
	}
	return converted, nil
}

// Valid implements net.IP.Valid
func (na *NetAddress) Valid() error {
	// Convert NetAddress to p2p.NetAddress
	p2pNetAddress := p2p.NetAddress{
		ID:   na.ID,
		IP:   net.ParseIP(na.IP.String()),
		Port: na.Port,
	}
	return p2pNetAddress.Valid()
}

////////////////////////////////////////////////////////////////
