package node

import (
	cfg "director/m/v2/config"
	"director/m/v2/rpc/core"
	rpccore "director/m/v2/rpc/core"
	"director/m/v2/state"
	"director/m/v2/store"
	"director/m/v2/version"
	"github.com/rs/cors"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p/pex"
	rpccoretypes "github.com/tendermint/tendermint/rpc/core/types"
	grpccore "github.com/tendermint/tendermint/rpc/grpc"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	dbm "github.com/tendermint/tm-db"
	"net"
	"net/http"
	"strings"
	"time"
)

// DBContext specifies config information for loading a new DB.
type DBContext struct {
	ID     string
	Config *cfg.Config
}

// DBProvider takes a DBContext and returns an instantiated DB.
type DBProvider func(*DBContext) (dbm.DB, error)

// DefaultDBProvider returns a database using the DBBackend and DBDir
// specified in the ctx.Config.
func DefaultDBProvider(ctx *DBContext) (dbm.DB, error) {
	dbType := dbm.BackendType(ctx.Config.DBBackend)
	return dbm.NewDB(ctx.ID, dbType, ctx.Config.DBDir()), nil
}

// Provider takes a config and a logger and returns a ready to go Node.
type Provider func(*cfg.Config, log.Logger) (*Node, error)

// DefaultNewNode returns a Tendermint node with default settings for the DBProvider
// It implements NodeProvider.
func DefaultNewNode(config *cfg.Config, logger log.Logger) (*Node, error) {
	return NewNode(config,
		DefaultDBProvider,
		logger,
	)
}

// Option sets a parameter for the node.
type Option func(*Node)

// Node is the highest level interface to a full Tendermint node.
// It includes all configuration information and running services.
type Node struct {
	service.BaseService

	// config
	config *cfg.Config

	// services
	rpcListeners []net.Listener // rpc servers
	stateMachine *state.Machine // state machine service for each testnet
}

func initDBs(config *cfg.Config, dbProvider DBProvider) (mystore *store.TestnetDB, err error) {
	var storeDB dbm.DB
	storeDB, err = dbProvider(&DBContext{"testnetDB", config})
	if err != nil {
		return
	}
	mystore = store.NewStore(storeDB, *config.Testnets)

	return
}

// NewNode returns a new, ready to go, Director.
func NewNode(config *cfg.Config,
	dbProvider DBProvider,
	logger log.Logger,
	options ...Option) (*Node, error) {

	testnetStore, err := initDBs(config, dbProvider)
	if err != nil {
		return nil, err
	}

	// Create state machine
	stateMachineLogger := logger.With("module", "state")
	stateMachine := createStateMachine(testnetStore, stateMachineLogger, *config.StateMachineHeartbeat)

	// Log the version info.
	logger.Info("Version info",
		"software", version.Version,
	)

	node := &Node{
		config:       config,
		stateMachine: stateMachine,
	}
	node.BaseService = *service.NewBaseService(logger, "Node", node)

	for _, option := range options {
		option(node)
	}

	return node, nil
}

// OnStart starts the Node. It implements service.Service.
func (n *Node) OnStart() error {

	// Start StateMachine
	err := n.stateMachine.Start()
	if err != nil {
		panic(err)
	}

	// Start the RPC server
	if n.config.RPC.ListenAddress != "" {
		listeners, err := n.startRPC()
		if err != nil {
			return err
		}
		n.rpcListeners = listeners
	}

	return nil
}

// OnStop stops the Node. It implements service.Service.
func (n *Node) OnStop() {
	n.BaseService.OnStop()

	//Stop State Machine
	_ = n.stateMachine.Stop()

	n.Logger.Info("Stopping Node")

	// finally stop the listeners / external services
	for _, l := range n.rpcListeners {
		n.Logger.Info("Closing rpc listener", "listener", l)
		if err := l.Close(); err != nil {
			n.Logger.Error("Error closing listener", "listener", l, "err", err)
		}
	}

	// All writes to the struct save the db too, so the below is unnecessary.
	//n.Logger.Info("Saving database")
	//err := n.testnetDB.saveStore()
	//if err != nil {
	//	panic(err)
	//}
}

// ConfigureRPC sets all variables in rpccore so they will serve
// rpc calls from this node
func (n *Node) ConfigureRPC() {
	rpccore.SetStateMachine(n.stateMachine)
	rpccore.SetLogger(n.Logger.With("module", "rpc"))
	rpccore.SetConfig(*n.config.RPC)
}

func (n *Node) startRPC() ([]net.Listener, error) {
	n.ConfigureRPC()
	listenAddrs := splitAndTrimEmpty(n.config.RPC.ListenAddress, ",", " ")
	coreCodec := amino.NewCodec()
	rpccoretypes.RegisterAmino(coreCodec)
	pex.RegisterMessages(coreCodec)

	config := rpcserver.DefaultConfig()
	config.MaxBodyBytes = n.config.RPC.MaxBodyBytes
	config.MaxHeaderBytes = n.config.RPC.MaxHeaderBytes
	config.MaxOpenConnections = n.config.RPC.MaxOpenConnections
	// If necessary adjust global WriteTimeout to ensure it's greater than
	// TimeoutBroadcastTxCommit.
	// See https://github.com/tendermint/tendermint/issues/3435
	if config.WriteTimeout <= n.config.RPC.TimeoutBroadcastTxCommit {
		config.WriteTimeout = n.config.RPC.TimeoutBroadcastTxCommit + 1*time.Second
	}

	// we may expose the rpc over both a unix and tcp socket
	listeners := make([]net.Listener, len(listenAddrs))
	for i, listenAddr := range listenAddrs {
		mux := http.NewServeMux()
		rpcLogger := n.Logger.With("module", "rpc-server")
		wmLogger := rpcLogger.With("protocol", "websocket")
		wm := rpcserver.NewWebsocketManager(core.Routes, coreCodec,
			rpcserver.OnDisconnect(func(remoteAddr string) {}),
			rpcserver.ReadLimit(config.MaxBodyBytes),
		)
		wm.SetLogger(wmLogger)
		mux.HandleFunc("/websocket", wm.WebsocketHandler)
		rpcserver.RegisterRPCFuncs(mux, core.Routes, coreCodec, rpcLogger)
		listener, err := rpcserver.Listen(
			listenAddr,
			config,
		)
		if err != nil {
			return nil, err
		}

		var rootHandler http.Handler = mux
		if n.config.RPC.IsCorsEnabled() {
			corsMiddleware := cors.New(cors.Options{
				AllowedOrigins: n.config.RPC.CORSAllowedOrigins,
				AllowedMethods: n.config.RPC.CORSAllowedMethods,
				AllowedHeaders: n.config.RPC.CORSAllowedHeaders,
			})
			rootHandler = corsMiddleware.Handler(mux)
		}
		if n.config.RPC.IsTLSEnabled() {
			go rpcserver.StartHTTPAndTLSServer(
				listener,
				rootHandler,
				n.config.RPC.CertFile(),
				n.config.RPC.KeyFile(),
				rpcLogger,
				config,
			)
		} else {
			go rpcserver.StartHTTPServer(
				listener,
				rootHandler,
				rpcLogger,
				config,
			)
		}

		listeners[i] = listener
	}

	// we expose a simplified api over grpc for convenience to app devs
	grpcListenAddr := n.config.RPC.GRPCListenAddress
	if grpcListenAddr != "" {
		config := rpcserver.DefaultConfig()
		config.MaxOpenConnections = n.config.RPC.MaxOpenConnections
		listener, err := rpcserver.Listen(grpcListenAddr, config)
		if err != nil {
			return nil, err
		}
		go grpccore.StartGRPCServer(listener)
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// Create State Machine
func createStateMachine(testnetDB *store.TestnetDB, stateLogger log.Logger, timeoutInterval time.Duration) *state.Machine {
	return state.NewMachine(testnetDB, stateLogger, timeoutInterval)
}

//------------------------------------------------------------------------------

// splitAndTrimEmpty slices s into all subslices separated by sep and returns a
// slice of the string s with all leading and trailing Unicode code points
// contained in cutset removed. If sep is empty, SplitAndTrim splits after each
// UTF-8 sequence. First part is equivalent to strings.SplitN with a count of
// -1.  also filter out empty strings, only return non-empty strings.
func splitAndTrimEmpty(s, sep, cutset string) []string {
	if s == "" {
		return []string{}
	}

	spl := strings.Split(s, sep)
	nonEmptyStrings := make([]string, 0, len(spl))
	for i := 0; i < len(spl); i++ {
		element := strings.Trim(spl[i], cutset)
		if element != "" {
			nonEmptyStrings = append(nonEmptyStrings, element)
		}
	}
	return nonEmptyStrings
}
