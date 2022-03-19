package service

import (
	"net/http"
	"time"

	tmCommon "github.com/arcology-network/3rd-party/tm/common"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/evm/params"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"context"
	"fmt"
	"math/big"

	internal "github.com/arcology-network/eth-api-svc/backend"
	wal "github.com/arcology-network/eth-api-svc/wallet"
	jsonrpc "github.com/deliveroo/jsonrpc-go"

	mainCfg "github.com/arcology-network/component-lib/config"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start eth-api service Daemon",
	RunE:  startCmd,
}

func init() {
	flags := StartCmd.Flags()

	//common
	flags.String("msgexch", "msgexch", "topic for receive msg exchange")
	flags.String("log", "log", "topic for send log")
	flags.Int("concurrency", 4, "num of threads")

	flags.String("inclusive-txs", "inclusive-txs", "topic of send txlist")

	flags.String("mqaddr", "localhost:9092", "host:port of kafka for update apc")

	flags.String("receipts", "receipts", "topic for send  receipts ")

	flags.String("logcfg", "./log.toml", "log conf path")

	flags.Int("nidx", 0, "node index in cluster")
	flags.String("nname", "node1", "node name in cluster")

	flags.String("local-block", "local-block", "topic of received proposer block ")

	flags.Int("filtertimeout", 5, "filter timeout minutes")

	flags.String("ethapicfg", "./eth-api.tom", "eth config file path")
	flags.String("monacocfg", "./monaco.toml", "main config file path")
}

func startCmd(cmd *cobra.Command, args []string) error {
	//mainConfig.InitCfg(viper.GetString("maincfg"))

	filters := internal.NewFilters(time.Minute * viper.GetDuration("filtertimeout"))
	rpcStart(filters)
	log.InitLog("ethapi.log", viper.GetString("logcfg"), "ethapi", viper.GetString("nname"), viper.GetInt("nidx"))
	en := NewConfig(filters)
	en.Start()

	// Wait forever
	tmCommon.TrapSignal(func() {
		// Cleanup
		en.Stop()
	})

	return nil
}

func rpcStart(filters *internal.Filters) {

	LoadCfg(viper.GetString("ethapicfg"), &options)
	mainCfg.InitCfg(viper.GetString("monacocfg"))
	options.ChainID = mainCfg.MainConfig.ChainId.Uint64()
	fmt.Println(options)

	if options.ChainID == 0 {
		options.ChainID = params.MainnetChainConfig.ChainID.Uint64()
	}
	privateKeys := LoadKeys(options.KeyFile)

	server := jsonrpc.New()
	server.Use(func(next jsonrpc.Next) jsonrpc.Next {
		return func(ctx context.Context, params interface{}) (interface{}, error) {
			//method := jsonrpc.MethodFromContext(ctx)
			//fmt.Printf("method: %v \t params:%v \n", method, params)
			//fmt.Println("logger: ", params)
			return next(ctx, params)
		}
	})
	server.Register(jsonrpc.Methods{
		"net_version":               version,
		"eth_chainId":               chainId,
		"eth_blockNumber":           blockNumber,
		"eth_getBlockByNumber":      getBlockByNumber,
		"eth_getBlockByHash":        getBlockByHash,
		"eth_getTransactionCount":   getTransactionCount,
		"eth_getCode":               getCode,
		"eth_getBalance":            getBalance,
		"eth_getStorageAt":          getStorageAt,
		"eth_accounts":              accounts,
		"eth_estimateGas":           estimateGas,
		"eth_gasPrice":              gasPrice,
		"eth_sendTransaction":       sendTransaction,
		"eth_getTransactionReceipt": getTransactionReceipt,
		"eth_getTransactionByHash":  getTransactionByHash,
		"eth_sendRawTransaction":    sendRawTransaction,
		"eth_call":                  call,
		"eth_getLogs":               getLogs,

		"eth_getBlockTransactionCountByHash":      getBlockTransactionCountByHash,
		"eth_getBlockTransactionCountByNumber":    getBlockTransactionCountByNumber,
		"eth_getTransactionByBlockHashAndIndex":   getTransactionByBlockHashAndIndex,
		"eth_getTransactionByBlockNumberAndIndex": getTransactionByBlockNumberAndIndex,
		"eth_getUncleCountByBlockHash":            getUncleCountByBlockHash,
		"eth_getUncleCountByBlockNumber":          getUncleCountByBlockNumber,
		"eth_submitWork":                          submitWork,
		"eth_submitHashrate":                      submitHashrate,
		"eth_hashrate":                            hashrate,
		"eth_getWork":                             getWork,
		"eth_protocolVersion":                     protocolVersion,
		"eth_coinbase":                            coinbase,

		"eth_sign":            sign,
		"eth_signTransaction": signTransaction,
		"eth_feeHistory":      feeHistory,
		"eth_syncing":         syncing,
		"eth_mining":          mining,

		"eth_newFilter":                   newFilter,
		"eth_newBlockFilter":              newBlockFilter,
		"eth_newPendingTransactionFilter": newPendingTransactionFilter,
		"eth_uninstallFilter":             uninstallFilter,
		"eth_getFilterChanges":            getFilterChanges,
		"eth_getFilterLogs":               getFilterLogs,
	})

	if options.Debug {
		backend = internal.NewEthereumAPIMock(new(big.Int).SetUint64(options.ChainID))
	} else {
		backend = internal.NewMonaco(options.Zookeeper, filters)
	}

	wallet = wal.NewWallet(new(big.Int).SetUint64(options.ChainID), privateKeys)

	c := cors.AllowAll()

	go http.ListenAndServe(fmt.Sprintf(":%d", options.Port), c.Handler(server))

}
