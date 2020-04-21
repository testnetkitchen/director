package state

import (
	"director/m/v2/store"
	"errors"
)

// CheckAndSetState is sent when the state of a testnet needs to be checked and if the conditions are correct, the new state has to be set
type CheckAndSetState struct {
	ChainID string
}

// ValidateBasic validates a CheckAndSetState message
func (c *CheckAndSetState) ValidateBasic() error {
	if c.ChainID == "" {
		return errors.New("message CheckAndSetState error: empty chain ID")
	}
	return nil
}

// GlobalCheckAndSetState is sent when all testnets needs to be checked and if the conditions are correct, the new state has to be set
type GlobalCheckAndSetState struct {
}

// ValidateBasic validates a GlobalCheckAndSetState message
func (c *GlobalCheckAndSetState) ValidateBasic() error {
	return nil
}

// RegisterValidator is sent when a new validator registers itself
type RegisterValidator struct {
	ChainID   string
	Validator store.ValidatorConfig
}

// ValidateBasic validates a RegisterValidator message
func (r *RegisterValidator) ValidateBasic() error {
	// Todo: Implement checks
	return nil
}
