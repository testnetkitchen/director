package config

import (
	"github.com/pkg/errors"
	tmcfg "github.com/tendermint/tendermint/config"
	"path/filepath"
	"time"
)

const (
	// LogFormatPlain is a format for colored text
	LogFormatPlain = "plain"
	// LogFormatJSON is a format for json output
	LogFormatJSON = "json"
)

var (
	// DefaultDirectorDir is the default $HOME folder
	DefaultDirectorDir   = ".director"
	defaultConfigDir     = "config"
	defaultDataDir       = "data"
	defaultListenAddress = "tcp://127.0.0.1:27001"

	defaultConfigFileName = "config.toml"

	defaultConfigFilePath = filepath.Join(defaultConfigDir, defaultConfigFileName)

	// StateMachineHeartbeat defines an interval when the state machine gets a regular update
	StateMachineHeartbeat = 15 * time.Second
)

// Config defines the top level configuration for the director
type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`

	// Options for services
	RPC *tmcfg.RPCConfig `mapstructure:"rpc"`

	// Testnet descriptions
	Testnets *map[string]TestnetsTOMLConfig `mapstructure:"testnets"`

	// State machine heartbeat
	StateMachineHeartbeat *time.Duration `mapstructure:"statemachine_heartbeat"`
}

// DefaultConfig returns a default configuration struct
func DefaultConfig() *Config {
	return &Config{
		BaseConfig:            DefaultBaseConfig(),
		RPC:                   DefaultRPCConfig(),
		Testnets:              DefaultTestnetsTOMLConfig(),
		StateMachineHeartbeat: defaultStateMachineHeartbeat(),
	}
}

// SetRoot sets the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	cfg.RPC.RootDir = root
	for _, testnet := range *cfg.Testnets {
		testnet.RootDir = root
	}
	return cfg
}

// ValidateBasic performs basic validation (checking param bounds, etc.) and
// returns an error if any check fails.
func (cfg *Config) ValidateBasic() error {
	if err := cfg.BaseConfig.ValidateBasic(); err != nil {
		return err
	}
	if err := cfg.RPC.ValidateBasic(); err != nil {
		return errors.Wrap(err, "error in [rpc] section")
	}
	for _, testnet := range *cfg.Testnets {
		if err := testnet.ValidateBasic(); err != nil {
			return err
		}
	}
	if *cfg.StateMachineHeartbeat <= 0 {
		return errors.New("no heartbeat set")
	}
	return nil
}

func defaultStateMachineHeartbeat() *time.Duration {
	return &StateMachineHeartbeat
}

//-----------------------------------------------------------------------------
// RPCConfig

// DefaultRPCConfig returns a default configuration for the RPC server
func DefaultRPCConfig() *tmcfg.RPCConfig {
	result := tmcfg.DefaultRPCConfig()
	result.ListenAddress = defaultListenAddress
	return result
}

//-----------------------------------------------------------------------------
// TestnetsTOMLConfig

// TestnetsTOMLConfig defines the configuration options for the Tendermint RPC server
type TestnetsTOMLConfig struct {
	RootDir string `mapstructure:"home"`

	// Timeout before director enters the 'serve' state
	Timeout time.Duration `mapstructure:"timeout,omitempty"`

	// Required minimum number of validators before director enters the 'serve' state
	RequiredValidators uint `mapstructure:"required_validators,omitempty"`

	// Todo: Low priority. Find a way to include ConsensusParams and AppState. Possibly separate JSON input.
	// Genesis consensus parameters
	//ConsensusParams string `mapstructure:"consensus_params,omitempty"`

	// AppState JSON
	//AppState json.RawMessage `mapstructure:"app_state"`
}

// DefaultTestnetsTOMLConfig returns a default configuration for a Testnet
func DefaultTestnetsTOMLConfig() *map[string]TestnetsTOMLConfig {
	return &map[string]TestnetsTOMLConfig{}
}

// ValidateBasic performs basic validation (checking param bounds, etc.) and
// returns an error if any check fails.
func (cfg *TestnetsTOMLConfig) ValidateBasic() error {
	if cfg.Timeout == time.Duration(0) && cfg.RequiredValidators == 0 {
		return errors.New("at least Timeout or RequiredValidators must be set greater than 0")
	}
	return nil
}

//-----------------------------------------------------------------------------
// Utils

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
