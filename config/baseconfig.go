package config

import (
	"errors"
	"fmt"
)

//-----------------------------------------------------------------------------
// BaseConfig

// BaseConfig defines the base configuration for Director
type BaseConfig struct { //nolint: maligned
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	// Database backend: goleveldb | cleveldb | boltdb
	// * goleveldb (github.com/syndtr/goleveldb - most popular implementation)
	//   - pure go
	//   - stable
	// * cleveldb (uses levigo wrapper)
	//   - fast
	//   - requires gcc
	//   - use cleveldb build tag (go build -tags cleveldb)
	// * boltdb (uses etcd's fork of bolt - github.com/etcd-io/bbolt)
	//   - EXPERIMENTAL
	//   - may be faster is some use-cases (random reads - indexer)
	//   - use boltdb build tag (go build -tags boltdb)
	DBBackend string `mapstructure:"db_backend"`

	// Database directory
	DBPath string `mapstructure:"db_dir"`

	// Output level for logging
	LogLevel string `mapstructure:"log_level"`

	// Output format: 'plain' (colored text) or 'json'
	LogFormat string `mapstructure:"log_format"`
}

// DefaultBaseConfig returns a default base configuration for the Director
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		LogLevel:  DefaultPackageLogLevels(),
		LogFormat: LogFormatPlain,
		DBBackend: "goleveldb",
		DBPath:    "data",
	}
}

// DBDir returns the full path to the database directory
func (cfg BaseConfig) DBDir() string {
	return rootify(cfg.DBPath, cfg.RootDir)
}

// ValidateBasic performs basic validation (checking param bounds, etc.) and
// returns an error if any check fails.
func (cfg BaseConfig) ValidateBasic() error {
	switch cfg.LogFormat {
	case LogFormatPlain, LogFormatJSON:
	default:
		return errors.New("unknown log_format (must be 'plain' or 'json')")
	}
	return nil
}

// DefaultLogLevel returns a default log level of "error"
func DefaultLogLevel() string {
	return "error"
}

// DefaultPackageLogLevels returns a default log level setting so all packages
// log at "error", while the `state` and `main` packages log at "info"
func DefaultPackageLogLevels() string {
	return fmt.Sprintf("main:info,state:info,*:%s", DefaultLogLevel())
}
