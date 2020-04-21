package state

import (
	"director/m/v2/store"
	"github.com/tendermint/tendermint/consensus"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"reflect"
	"runtime/debug"
	"time"
)

var (
	msgQueueSize = 1000
)

// Machine is the state machine struct definition
type Machine struct {
	service.BaseService
	testnetDB *store.TestnetDB

	// state changes may be triggered by: msgs from peers,
	// msgs from ourself, or by timeouts
	peerMsgQueue    chan msgInfo
	timeoutTicker   TimeoutTicker
	timeoutInterval time.Duration
}

// MachineOption is additional parameters to Machine
type MachineOption func(*Machine)

// msgs from the reactor which may update the state
type msgInfo struct {
	Msg consensus.Message `json:"msg"`
}

// internally generated messages which may update the state
type timeoutInfo struct {
	Duration time.Duration `json:"duration"`
}

// NewMachine returns a new state machine object
func NewMachine(testnetDB *store.TestnetDB, logger log.Logger, timeoutInterval time.Duration, options ...MachineOption) *Machine {
	m := &Machine{
		testnetDB:       testnetDB,
		peerMsgQueue:    make(chan msgInfo, msgQueueSize),
		timeoutTicker:   NewTimeoutTicker(),
		timeoutInterval: timeoutInterval,
	}
	m.BaseService = *service.NewBaseService(logger, "StateMachine", m)
	m.timeoutTicker.SetLogger(logger)
	for _, option := range options {
		option(m)
	}

	return m
}

// OnStart implements start function for the state machine service
func (m *Machine) OnStart() error {
	if err := m.timeoutTicker.Start(); err != nil {
		return err
	}

	// Todo: Make the smallest common denominator of all testnets that need a timer the next timeout.
	//for chainID, duration := range m.testnetDB.GetTimedTestnets() {
	//	m.timeoutTicker.ScheduleTimeout(timeoutInfo{
	//		Duration: duration,
	//		ChainID:  chainID,
	//	})
	//}
	m.timeoutTicker.ScheduleTimeout(timeoutInfo{
		Duration: 0,
	})

	go m.receiveRoutine()

	m.Logger.Info("State machine started")
	return nil
}

// OnStop implements stop function for the state machine service
func (m *Machine) OnStop() {
	_ = m.timeoutTicker.Stop()
	m.Logger.Info("State machine stopped")
}

func (m *Machine) receiveRoutine() {
	defer func() {
		if r := recover(); r != nil {
			m.Logger.Error("StateMachine failure", "err", r, "stack", string(debug.Stack()))
			// stop gracefully
		}
	}()

	for {
		var mi msgInfo

		select {
		case mi = <-m.peerMsgQueue:
			m.handleMsg(mi)
		case ti := <-m.timeoutTicker.Chan(): // tockChan:
			m.handleTimeout(ti)
		case <-m.Quit():
			return
		}
	}
}

// state transitions
func (m *Machine) handleMsg(mi msgInfo) {
	//Todo: Decide if we need a Mutex in Machine to make sure we only process one message at a time. (Right now, because testnetDB is thread-safe, most messages can be handled in parallel.)
	//m.mtx.Lock()
	//defer m.mtx.Unlock()

	var (
		err error
	)
	msg := mi.Msg
	m.Logger.Debug("Received message", "msg", reflect.TypeOf(msg))
	switch msg := msg.(type) {
	case *RegisterValidator:
		// Coming from the Register endpoint when a validator is registering on a testnet.
		err = m.testnetDB.RegisterValidator(msg.ChainID, msg.Validator)
	case *GlobalCheckAndSetState:
		// Coming from the Timer, when all testnet states should be checked for timeout.
		err = m.testnetDB.GlobalStateCheck()
	default:
		m.Logger.Error("Unknown msg type", "type", reflect.TypeOf(msg))
		return
	}

	if err != nil { // nolint:staticcheck
		// Causes TestReactorValidatorSetChanges to timeout
		// https://github.com/tendermint/tendermint/issues/3406
		m.Logger.Error("Error with msg", "err", err, "msg", msg)
	}
}

func (m *Machine) handleTimeout(ti timeoutInfo) {
	m.Logger.Debug("Received tock", "timeout", ti.Duration)
	m.SendMessage(&GlobalCheckAndSetState{})
	m.timeoutTicker.ScheduleTimeout(timeoutInfo{
		Duration: m.timeoutInterval,
	})
}

// SendMessage sends a channel message to the state machine
func (m *Machine) SendMessage(msg interface{}) {
	m.peerMsgQueue <- msgInfo{Msg: msg.(consensus.Message)}
}

// RegisterValidator registers a new validator in the state machine database struct
func (m *Machine) RegisterValidator(chainID string, validator store.ValidatorConfig) error {
	return m.testnetDB.RegisterValidator(chainID, validator)
}

// GetGenesis returns the genesis file of a testnet from the state machine database struct
func (m *Machine) GetGenesis(chainID string) (*tmctypes.ResultGenesis, error) {
	return m.testnetDB.GetGenesis(chainID)
}

// GetAddressBook returns the address book file of a testnet from the state machine database struct
func (m *Machine) GetAddressBook(chainID string) (*store.AddrBookJSON, error) {
	return m.testnetDB.GetAddressBook(chainID)
}
