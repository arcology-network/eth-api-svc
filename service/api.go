package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/arcology-network/component-lib/ethrpc"
	internal "github.com/arcology-network/eth-api-svc/backend"
	wal "github.com/arcology-network/eth-api-svc/wallet"
	ethtyp "github.com/arcology-network/evm/core/types"
	jsonrpc "github.com/deliveroo/jsonrpc-go"
)

func LoadCfg(tomfile string, conf interface{}) {
	_, err := toml.DecodeFile(tomfile, conf)
	if err != nil {
		fmt.Printf("err=%v\n", err)
		panic("load conf err :" + err.Error())
	}
}

type Options struct {
	KeyFile         string `short:"k" long:"keyfile" description:"Private keys file path" default:"./accounts.txt"`
	ChainID         uint64 `short:"c" long:"chainid" description:"Network chain ID"`
	Port            uint64 `short:"p" long:"port" description:"Service port" default:"7545"`
	Zookeeper       string `short:"z" long:"zookeeper" description:"Zookeeper service address" default:"127.0.0.1:2181"`
	Debug           bool   `short:"d" long:"debug" description:"Enable debug mode"`
	Waits           int    `short:"w" long:"waits" description:"wait seconds when query receipt" default:"60"`
	Coinbase        string `short:"cb" long:"coinbase" description:"coinbase address of node`
	ProtocolVersion int    `short:"pv" long:"protocolVersion" description:"Protocol Version`
	Hashrate        int    `short:"hr" long:hashrate" description:"hash rate`
}

var options Options
var backend internal.EthereumAPI
var wallet *wal.Wallet

func version(ctx context.Context) (interface{}, error) {
	return options.ChainID, nil
	//return NumberToHex(options.ChainID), nil
}

func chainId(ctx context.Context) (interface{}, error) {
	return NumberToHex(options.ChainID), nil
}

func blockNumber(ctx context.Context) (interface{}, error) {
	number, err := backend.BlockNumber()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}

	return NumberToHex(number), nil
}

func getBlockByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[0])
	}

	block, err := backend.GetBlockByNumber(number, false)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	// for i := range block.Transactions {
	// 	fmt.Printf("====hash:%x\n", block.Transactions[i])
	// }
	return parseBlock(block), nil
}

func getBlockByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	block, err := backend.GetBlockByHash(hash, false)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return parseBlock(block), nil
}

func parseBlock(block *ethrpc.RPCBlock) interface{} {
	transactions := make([]string, len(block.Transactions))
	for i := range block.Transactions {
		transactions[i] = fmt.Sprintf("%x", block.Transactions[i])
	}
	header := block.Header
	return map[string]interface{}{
		"number":           NumberToHex(header.Number),
		"hash":             header.Hash(),
		"parentHash":       header.ParentHash,
		"nonce":            NumberToHex(header.Nonce),
		"mixHash":          header.MixDigest,
		"sha3Uncles":       header.UncleHash,
		"logsBloom":        header.Bloom,
		"stateRoot":        header.Root,
		"miner":            header.Coinbase,
		"difficulty":       NumberToHex(header.Difficulty),
		"extraData":        header.Extra,
		"size":             header.Size(),
		"gasLimit":         NumberToHex(header.GasLimit),
		"gasUsed":          NumberToHex(header.GasUsed),
		"timestamp":        NumberToHex(header.Time),
		"transactionsRoot": header.TxHash,
		"receiptsRoot":     header.ReceiptHash,
		"transactions":     transactions,
	}
}

func getTransactionCount(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[1])
	}

	nonce, err := backend.GetTransactionCount(address, number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(nonce), nil
}

func getCode(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[1])
	}

	code, err := backend.GetCode(address, number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return code, nil
}

func getBalance(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[1])
	}

	balance, err := backend.GetBalance(address, number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(balance), nil
}

func getStorageAt(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	key, err := ToHash(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid key given %v", params[1])
	}

	number, err := ToBlockNumber(params[2])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[2])
	}

	value, err := backend.GetStorageAt(address, key.Hex(), number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return "0x" + hex.EncodeToString(value), nil
}

func accounts(ctx context.Context) (interface{}, error) {
	return wallet.Accounts(), nil
}

func estimateGas(ctx context.Context, params []interface{}) (interface{}, error) {
	msg, err := ToCallMsg(params[0], true)
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid call msg given %v", params[0])
	}

	gas, err := backend.EstimateGas(msg)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(gas), nil
}

func gasPrice(ctx context.Context) (interface{}, error) {
	gp, err := backend.GasPrice()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(gp), nil
}

func sendTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	tx, err := ToSendTxArgs(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid transaction given %v", params[0])
	}

	rawTx, err := wallet.SignTx(0, tx.Nonce, tx.To, tx.Value, tx.Gas, tx.GasPrice, tx.Data)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}

	hash, err := backend.SendRawTransaction(rawTx)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return hash.Hex(), nil
}

func getTransactionReceipt(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}
	var receipt *ethtyp.Receipt

	queryCounter := options.Waits
	for queryIdx := 0; queryIdx < queryCounter; queryIdx++ {
		receipt, err = backend.GetTransactionReceipt(hash)
		if err != nil {
			if queryIdx == queryCounter-1 {
				return nil, jsonrpc.InternalError(err)
			}
			time.Sleep(time.Duration(time.Second * 1))
		} else {
			break
		}
	}
	return receipt, nil
}

func getTransactionByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	tx, err := backend.GetTransactionByHash(hash)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return ToTransactionResponse(tx), nil
}

func sendRawTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	bytes, err := ToBytes(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid raw transaction given %v", params[0])
	}

	hash, err := backend.SendRawTransaction(bytes)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return hash.Hex(), nil
}

func call(ctx context.Context, params []interface{}) (interface{}, error) {
	msg, err := ToCallMsg(params[0], true)
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid call msg given %v", params[0])
	}
	if msg.Gas == 0 {
		msg.Gas = math.MaxUint32
	}
	if msg.GasPrice == nil {
		msg.GasPrice = big.NewInt(0xff)
	}
	ret, err := backend.Call(msg)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return "0x" + hex.EncodeToString(ret), nil
}

func getLogs(ctx context.Context, params []interface{}) (interface{}, error) {
	filter, err := ToFilter(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}

	logs, err := backend.GetLogs(filter)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return logs, nil
}

func getBlockTransactionCountByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	txsNum, err := backend.GetBlockTransactionCountByHash(hash)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(txsNum), nil
}

func getBlockTransactionCountByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid number given %v", params[0])
	}

	txsNum, err := backend.GetBlockTransactionCountByNumber(number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(txsNum), nil
}

func getTransactionByBlockHashAndIndex(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	index, err := ToBlockIndex(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid index given %v", params[1])
	}

	tx, err := backend.GetTransactionByBlockHashAndIndex(hash, index)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return ToTransactionResponse(tx), nil
}
func getTransactionByBlockNumberAndIndex(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid number given %v", params[0])
	}
	index, err := ToBlockIndex(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid index given %v", params[1])
	}

	tx, err := backend.GetTransactionByBlockNumberAndIndex(number, index)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return ToTransactionResponse(tx), nil
}
func getUncleCountByBlockHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	txsNum, err := backend.GetUncleCountByBlockHash(hash)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(txsNum), nil
}
func getUncleCountByBlockNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid number given %v", params[0])
	}

	txsNum, err := backend.GetUncleCountByBlockNumber(number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(txsNum), nil
}

func submitWork(ctx context.Context, params []interface{}) (interface{}, error) {

	// ok, err := backend.SubmitWork()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return fmt.Sprintf("%v", true), nil
}
func submitHashrate(ctx context.Context, params []interface{}) (interface{}, error) {

	// ok, err := backend.SubmitHashrate()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return fmt.Sprintf("%v", true), nil
}
func hashrate(ctx context.Context, params []interface{}) (interface{}, error) {

	// hashrate, err := backend.Hashrate()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return NumberToHex(options.Hashrate), nil
}
func getWork(ctx context.Context, params []interface{}) (interface{}, error) {

	// works, err := backend.GetWork()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return []string{}, nil
}
func protocolVersion(ctx context.Context, params []interface{}) (interface{}, error) {

	// version, err := backend.ProtocolVersion()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return fmt.Sprintf("%v", options.ProtocolVersion), nil
}
func coinbase(ctx context.Context, params []interface{}) (interface{}, error) {
	return options.Coinbase, nil
}

func sign(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}
	txdata, err := ToBytes(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid bytes given %v", params[0])
	}
	retData, err := wallet.Sign(address, txdata)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%x", retData), nil
}

func signTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	tx, err := ToSendTxArgs(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid transaction given %v", params[0])
	}
	rawTx, err := wallet.SignTx(0, tx.Nonce, tx.To, tx.Value, tx.Gas, tx.GasPrice, tx.Data)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%x", rawTx), nil
}
func feeHistory(ctx context.Context, params []interface{}) (interface{}, error) {
	return ethrpc.FeeHistoryResult{}, nil
}
func syncing(ctx context.Context, params []interface{}) (interface{}, error) {
	ok, err := backend.Syncing()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%v", ok), nil
}
func mining(ctx context.Context, params []interface{}) (interface{}, error) {
	ok, err := backend.Proposer()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%v", ok), nil
}

func newFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	filter, err := ToFilter(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}

	id, err := backend.NewFilter(filter)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return id, nil
}

func newBlockFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := backend.NewBlockFilter()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return id, nil
}
func newPendingTransactionFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := backend.NewPendingTransactionFilter()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return id, nil
}
func uninstallFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ToID(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}
	ok, err := backend.UninstallFilter(id)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return ok, nil
}

func getFilterChanges(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ToID(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}
	results, err := backend.GetFilterChanges(id)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return results, nil
}
func getFilterLogs(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ToID(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}

	logs, err := backend.GetFilterLogs(id)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return logs, nil
}
