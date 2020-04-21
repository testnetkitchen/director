package store

import (
	"bytes"
	"director/m/v2/config"
	"director/m/v2/types"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
	"time"
)

// NewStore creates a new DB and load the data from the file system.
func NewStore(db dbm.DB, testnetstomlconfig map[string]config.TestnetsTOMLConfig) *TestnetDB {
	testnets := map[string]*TestnetConfig{}
	var err error
	for key := range testnetstomlconfig {
		testnets[key], err = loadTestnetConfig(db, key)
		if err != nil {
			panic(fmt.Sprintf("error while loading testnet config %s: %v", key, err))
		}
	}
	return &TestnetDB{
		db:        db,
		testnets:  testnets,
		startTime: time.Now(),
		config:    testnetstomlconfig,
	}
}

// GlobalStateCheck goes through all testnets and set the state when necessary.
func (s *TestnetDB) GlobalStateCheck() (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	for chainID := range s.config {
		err = s.checkAndChangeStateToServer(chainID)
	}
	return
}

// RegisterValidator registers a new validator on a testnet in DB.
func (s *TestnetDB) RegisterValidator(chainID string, validator ValidatorConfig) (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.isRegisteredTestnet(chainID) {
		return errors.New("unregistered testnet")
	}
	if s.testnets[chainID].State != types.Gather {
		return errors.New("testnet not accepting new registrations")
	}
	s.testnets[chainID].Validators[validator.PubKey] = &validator
	err = s.checkAndChangeStateToServer(chainID)
	return
}

// GetGenesis gets genesis file from DB.
func (s *TestnetDB) GetGenesis(chainID string) (*tmctypes.ResultGenesis, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if !s.isRegisteredTestnet(chainID) {
		return nil, errors.New("unregistered testnet")
	}
	if s.testnets[chainID].State != types.Serve {
		return nil, errors.New("testnet not ready")
	}
	if s.testnets[chainID].Genesis == nil {
		return nil, errors.New("no genesis for testnet")
	}
	return s.testnets[chainID].Genesis, nil
}

// GetAddressBook gets address book from DB.
func (s *TestnetDB) GetAddressBook(chainID string) (*AddrBookJSON, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if !s.isRegisteredTestnet(chainID) {
		return nil, errors.New("unregistered testnet")
	}
	if s.testnets[chainID].State != types.Serve {
		return nil, errors.New("testnet not ready")
	}
	if s.testnets[chainID].AddressBook == nil {
		return nil, errors.New("no address book for testnet")
	}
	return s.testnets[chainID].AddressBook, nil
}

////////////////////////////////////////////////////////////////
// Internal functions
////////////////////////////////////////////////////////////////

// loadTestnetConfig loads testnet config from file into DB
func loadTestnetConfig(db dbm.DB, chainID string) (*TestnetConfig, error) {
	result := &TestnetConfig{
		State:      types.Gather,
		Validators: map[string]*ValidatorConfig{},
	}
	testnetconfigbytearray, err := db.Get([]byte(chainID))
	if err != nil {
		return nil, err
	}
	if testnetconfigbytearray == nil {
		return result, nil
	}
	gob.Register(ed25519.PubKeyEd25519{})
	dec := gob.NewDecoder(bytes.NewBuffer(testnetconfigbytearray))
	err = dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Not thread safe.
func (s *TestnetDB) saveTestnetConfig(chainID string, testnetconfig *TestnetConfig) (err error) {

	var buffer bytes.Buffer
	gob.Register(ed25519.PubKeyEd25519{})
	enc := gob.NewEncoder(&buffer)
	err = enc.Encode(testnetconfig)
	if err != nil {
		return
	}
	err = s.db.Set([]byte(chainID), buffer.Bytes())
	return
}

// Not thread safe.
func (s *TestnetDB) saveStore() error {
	for key, value := range s.testnets {
		err := s.saveTestnetConfig(key, value)
		if err != nil {
			return err
		}
	}

	// Flush
	err := s.db.SetSync(nil, nil)
	return err
}

// Not thread safe.
func (s *TestnetDB) isRegisteredTestnet(chainID string) bool {
	_, ok := s.testnets[chainID]
	return ok
}

// Not thread safe.
func (s *TestnetDB) checkAndChangeStateToServer(chainID string) error {
	if !s.isRegisteredTestnet(chainID) {
		return errors.New("unregistered testnet")
	}

	// Check if need to change state
	if !(s.testnets[chainID].State != types.Serve &&
		(len(s.testnets[chainID].Validators) >= int(s.config[chainID].RequiredValidators) ||
			s.config[chainID].Timeout < time.Since(s.startTime))) {
		return nil
	}

	// State = Serve
	s.testnets[chainID].State = types.Serve

	now := time.Now()

	// Generate Genesis
	var validators []tmtypes.GenesisValidator

	for pubKey, validator := range s.testnets[chainID].Validators {
		pubBytes, err := base64.StdEncoding.DecodeString(pubKey)
		if err != nil {
			// This should not happen, it is checked during registration. (Maybe old data in database.)
			continue
		}
		ed := ed25519.PubKeyEd25519{}
		copy(ed[:], pubBytes)
		validators = append(validators, tmtypes.GenesisValidator{
			Address: ed.Address(),
			Power:   10,
			PubKey:  ed,
			Name:    validator.Name,
		})
	}

	// Generate Address Book
	addrs := make([]*knownAddress, 0, len(s.testnets[chainID].Validators))
	for _, ka := range s.testnets[chainID].Validators {
		addrs = append(addrs, &knownAddress{
			Addr:        ka.NetAddress,
			Src:         ka.NetAddress,
			Buckets:     []int{1},
			Attempts:    0,
			BucketType:  bucketTypeNew,
			LastAttempt: now,
			LastSuccess: now,
		})
	}

	// If no validators signed up, no genesis or address book is generated
	if len(validators) > 0 {
		s.testnets[chainID].AddressBook = &AddrBookJSON{
			Key:   crypto.CRandHex(24),
			Addrs: addrs,
		}
		s.testnets[chainID].Genesis = &tmctypes.ResultGenesis{
			Genesis: &tmtypes.GenesisDoc{
				GenesisTime:     now,
				ChainID:         chainID,
				ConsensusParams: tmtypes.DefaultConsensusParams(),
				Validators:      validators,
			},
		}
	}

	// Save
	return s.saveStore()
}
