package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	types2 "github.com/chainpoint/tendermint/types"

	"github.com/chainpoint/tendermint/node"
	"github.com/chainpoint/tendermint/proxy"

	"github.com/chainpoint/tendermint/p2p"
	"github.com/chainpoint/tendermint/privval"

	"github.com/knq/pemutil"
	"github.com/spf13/viper"

	"github.com/chainpoint/tendermint/libs/log"

	cfg "github.com/chainpoint/tendermint/config"
	tmflags "github.com/chainpoint/tendermint/libs/cli/flags"
	cmn "github.com/chainpoint/tendermint/libs/common"
	tmtime "github.com/chainpoint/tendermint/types/time"
	"github.com/chp-project/chainpoint-core/go-abci-service/abci"
	"github.com/chp-project/chainpoint-core/go-abci-service/types"
	"github.com/chp-project/chainpoint-core/go-abci-service/util"
)

func main() {

	//Instantiate Tendermint Node Config
	tmConfig, err := initTendermintConfig()
	if util.LogError(err) != nil {
		return
	}
	logger := tmConfig.Logger

	//Instantiate ABCI application
	config := initABCIConfig(tmConfig.FilePV)
	app := abci.NewAnchorApplication(config)

	//declare connection to abci app
	appProxy := proxy.NewLocalClientCreator(app)

	/* Instantiate Tendermint Node with given config and abci app */
	n, err := node.NewNode(tmConfig.Config,
		&tmConfig.FilePV,
		tmConfig.NodeKey,
		appProxy,
		node.DefaultGenesisDocProviderFunc(tmConfig.Config),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(tmConfig.Config.Instrumentation),
		logger,
	)
	if err != nil {
		util.LogError(err)
		return
	}

	// Wait forever, shutdown gracefully upon
	cmn.TrapSignal(*config.Logger, func() {
		if n.IsRunning() {
			logger.Info("Shutting down Core...")
			n.Stop()
		}
	})

	// Start Tendermint Node
	if err := n.Start(); err != nil {
		util.LogError(err)
		return
	}
	logger.Info("Started node", "nodeInfo", n.Switch().NodeInfo())

	// Wait forever
	select {}
	return
}

// initABCIConfig: receives ENV variables and initializes app config struct
func initABCIConfig(pv privval.FilePV) types.AnchorConfig {
	// Perform env type conversions
	doPrivateNetwork, _ := strconv.ParseBool(util.GetEnv("PRIVATE_NETWORK", "false"))
	nodeIPs := strings.Split(util.GetEnv("PRIVATE_NODE_IPS", ""), ",")
	coreIPs := strings.Split(util.GetEnv("PRIVATE_CORE_IPS", ""), ",")
	doNodeManagement, _ := strconv.ParseBool(util.GetEnv("NODE_MANAGEMENT", "true"))
	doAuditLoop, _ := strconv.ParseBool(util.GetEnv("AUDIT", "true"))
	doNodeManagement = doNodeManagement && !doPrivateNetwork           //only allow node management if private networking is disabled
	doAuditLoop = doNodeManagement && doAuditLoop && !doPrivateNetwork //only allow auditing if node management enabled and private networking disabled
	doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))
	anchorInterval, _ := strconv.Atoi(util.GetEnv("ANCHOR_INTERVAL", "60"))
	ethInfuraApiKey := util.GetEnv("ETH_INFURA_API_KEY", "")
	ethereumURL := util.GetEnv("ETH_URI", fmt.Sprintf("https://ropsten.infura.io/v3/%s", ethInfuraApiKey))
	testMode := util.GetEnv("NETWORK", "testnet")
	useTestNets := (testMode == "testnet")
	ethTokenContract := ""
	if doAuditLoop {
		ethTokenContract = util.ReadContractJSON("/go/src/github.com/chp-project/chainpoint-core/go-abci-service/ethcontracts/TierionNetworkToken.json", useTestNets)
		if ethTokenContract == "" {
			fmt.Println("Token Contract: Cannot read from JSON ABI file, defaulting to hardcoded contract address")
			ethTokenContract = util.GetEnv("TokenContractAddr", "0x84294776884A92E6E06989DE0c675db81f8C9bD3")
		}
	}
	ethRegistryContract := ""
	if doNodeManagement {
		ethRegistryContract = util.ReadContractJSON("/go/src/github.com/chp-project/chainpoint-core/go-abci-service/ethcontracts/ChainpointRegistry.json", useTestNets)
		if ethRegistryContract == "" && doNodeManagement {
			fmt.Println("Registry Contract: Cannot read from JSON ABI file, defaulting to hardcoded contract address")
			ethRegistryContract = util.GetEnv("RegistryContractAddr", "0xE05da394fAE477De2eE6F64d5C64cf1D8F67a803")
		}
	}
	ethPrivateKey := util.GetEnv("ETH_PRIVATE_KEY", "")
	if len(ethPrivateKey) > 0 && strings.Contains(ethPrivateKey, "0x") {
		ethPrivateKey = ethPrivateKey[2:]
	}
	tendermintRPC := types.TendermintConfig{
		TMServer: util.GetEnv("TENDERMINT_HOST", "127.0.0.1"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
	}
	postgresUser := util.GetEnv(" POSTGRES_CONNECT_USER", "chainpoint")
	postgresPw := util.GetEnv("POSTGRES_CONNECT_PW", "chainpoint")
	postgresHost := util.GetEnv("POSTGRES_CONNECT_HOST", "postgres")
	postgresPort := util.GetEnv("POSTGRES_CONNECT_PORT", "5432")
	postgresDb := util.GetEnv("POSTGRES_CONNECT_DB", "chainpoint")
	redisURI := util.GetEnv("REDIS", "redis://redis:6379")
	apiURI := util.GetEnv("API_URI", "http://api:8080")

	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "info")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	ethConfig := types.EthConfig{
		EthereumURL:          ethereumURL,
		EthPrivateKey:        ethPrivateKey,
		TokenContractAddr:    ethTokenContract,
		RegistryContractAddr: ethRegistryContract,
	}

	store, err := pemutil.LoadFile("/run/secrets/ECDSA_PKPEM")
	if err != nil {
		util.LogError(err)
	}
	ecPrivKey, ok := store.ECPrivateKey()
	if !ok {
		util.LogError(errors.New("ecdsa key load failed"))
	}

	// Create config object
	return types.AnchorConfig{
		DBType:           "goleveldb",
		RabbitmqURI:      util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintConfig: tendermintRPC,
		PostgresURI:      fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPw, postgresHost, postgresPort, postgresDb),
		RedisURI:         redisURI,
		APIURI:           apiURI,
		EthConfig:        ethConfig,
		ECPrivateKey:     *ecPrivKey,
		DoNodeAudit:      doAuditLoop,
		DoNodeManagement: doNodeManagement,
		DoPrivateNetwork: doPrivateNetwork,
		PrivateNodeIPs:   nodeIPs,
		PrivateCoreIPs:   coreIPs,
		DoCal:            doCalLoop,
		DoAnchor:         doAnchorLoop,
		AnchorInterval:   anchorInterval,
		Logger:           &tmLogger,
		FilePV:           pv,
	}
}

// initTendermintConfig : imports tendermint config.toml and initializes config variables
func initTendermintConfig() (types.TendermintConfig, error) {
	var TMConfig types.TendermintConfig
	initEnv("TM")
	homeFlag := os.ExpandEnv(filepath.Join("$HOME", cfg.DefaultTendermintDir))
	homeDir := "/tendermint"
	viper.Set(homeFlag, homeDir)
	viper.SetConfigName("config")                         // name of config file (without extension)
	viper.AddConfigPath(homeDir)                          // search root directory
	viper.AddConfigPath(filepath.Join(homeDir, "config")) // search root directory /config

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// stderr, so if we redirect output to json file, this doesn't appear
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// ignore not found error, return other errors
		return TMConfig, err
	}
	defaultConfig := cfg.DefaultConfig()
	err := viper.Unmarshal(defaultConfig)
	if err != nil {
		return TMConfig, err
	}
	defaultConfig.SetRoot(homeDir)
	defaultConfig.Consensus.TimeoutCommit = time.Duration(60 * time.Second)
	defaultConfig.RPC.TimeoutBroadcastTxCommit = time.Duration(65 * time.Second) // allows us to wait for tx to commit + 5 sec latency margin
	defaultConfig.RPC.ListenAddress = "tcp://0.0.0.0:26657"
	defaultConfig.P2P.ListenAddress = "tcp://0.0.0.0:26656"
	defaultConfig.P2P.MaxNumInboundPeers = 300
	defaultConfig.P2P.MaxNumOutboundPeers = 75
	defaultConfig.TxIndex.IndexAllTags = true
	if tendermintPeers := util.GetEnv("PEERS", ""); tendermintPeers != "" {
		defaultConfig.P2P.PersistentPeers = tendermintPeers
	}
	if tendermintSeeds := util.GetEnv("SEEDS", ""); tendermintSeeds != "" {
		defaultConfig.P2P.Seeds = tendermintSeeds
	}
	fmt.Printf("Config : %#v\n", defaultConfig)
	cfg.EnsureRoot(defaultConfig.RootDir)

	//initialize logger
	tmlogger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	if defaultConfig.LogFormat == cfg.LogFormatJSON {
		tmlogger = log.NewTMJSONLogger(log.NewSyncWriter(os.Stdout))
	}
	logger, err := tmflags.ParseLogLevel(util.GetEnv("LOG_FILTER", "main:debug,state:info,*:error"), tmlogger, cfg.DefaultLogLevel())
	if err != nil {
		panic(err)
	}
	logger = logger.With("module", "main")
	TMConfig.Logger = logger

	// initialize private validator key
	// Convert old PrivValidator if it exists.
	oldPrivVal := defaultConfig.OldPrivValidatorFile()
	newPrivValKey := defaultConfig.PrivValidatorKeyFile()
	newPrivValState := defaultConfig.PrivValidatorStateFile()
	if _, err := os.Stat(oldPrivVal); !os.IsNotExist(err) {
		oldPV, err := privval.LoadOldFilePV(oldPrivVal)
		if err != nil {
			panic(err)
		}
		logger.Info("Upgrading PrivValidator file",
			"old", oldPrivVal,
			"newKey", newPrivValKey,
			"newState", newPrivValState,
		)
		oldPV.Upgrade(newPrivValKey, newPrivValState)
	}
	TMConfig.FilePV = *privval.LoadOrGenFilePV(newPrivValKey, newPrivValState)

	//initialize this node's keys
	nodeKey, err := p2p.LoadOrGenNodeKey(defaultConfig.NodeKeyFile())
	TMConfig.NodeKey = nodeKey

	// initialize genesis file
	genFile := defaultConfig.GenesisFile()
	if cmn.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		genDoc := types2.GenesisDoc{
			ChainID:         fmt.Sprintf("test-chain-%v", cmn.RandStr(6)),
			GenesisTime:     tmtime.Now(),
			ConsensusParams: types2.DefaultConsensusParams(),
		}
		key := TMConfig.FilePV.GetPubKey()
		genDoc.Validators = []types2.GenesisValidator{{
			Address: key.Address(),
			PubKey:  key,
			Power:   10,
		}}
		if err := genDoc.SaveAs(genFile); err != nil {
			panic(err)
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	TMConfig.Config = defaultConfig

	return TMConfig, nil
}

// initEnv sets to use ENV variables if set.
func initEnv(prefix string) {
	copyEnvVars(prefix)

	// env variables with TM prefix (eg. TM_ROOT)
	viper.SetEnvPrefix(prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

// This copies all variables like TMROOT to TM_ROOT,
// so we can support both formats for the user
func copyEnvVars(prefix string) {
	prefix = strings.ToUpper(prefix)
	ps := prefix + "_"
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) == 2 {
			k, v := kv[0], kv[1]
			if strings.HasPrefix(k, prefix) && !strings.HasPrefix(k, ps) {
				k2 := strings.Replace(k, prefix, ps, 1)
				os.Setenv(k2, v)
			}
		}
	}
}
