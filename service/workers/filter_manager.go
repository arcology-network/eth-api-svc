package workers

import (
	"math/big"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	ethTypes "github.com/arcology-network/3rd-party/eth/types"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	internal "github.com/arcology-network/eth-api-svc/backend"
)

type FilterManager struct {
	actor.WorkerThread
	filters *internal.Filters
}

//return a Subscriber struct
func NewFilterManager(concurrency int, groupid string, filters *internal.Filters) *FilterManager {
	fm := FilterManager{}
	fm.Set(concurrency, groupid)
	fm.filters = filters
	return &fm
}

func (*FilterManager) OnStart() {}
func (*FilterManager) Stop()    {}

func (fm *FilterManager) OnMessageArrived(msgs []*actor.Message) error {
	result := ""
	var receipts *[]*ethTypes.Receipt
	var block *types.MonacoBlock

	for _, v := range msgs {
		switch v.Name {
		case actor.MsgBlockCompleted:
			result = v.Data.(string)
		case actor.MsgSelectedReceipts:
			receipts = v.Data.(*[]*ethTypes.Receipt)
		case actor.MsgPendingBlock:
			block = v.Data.(*types.MonacoBlock)
		}
	}

	if actor.MsgBlockCompleted_Success == result {

		if receipts != nil {

			blockHash := block.Hash()

			worker := func(start, end int, idx int, args ...interface{}) {
				receipts := args[0].([]interface{})[0].(*[]*ethTypes.Receipt)
				for i := start; i < end; i++ {

					(*receipts)[i].BlockHash = ethCommon.BytesToHash(blockHash)
					(*receipts)[i].BlockNumber = big.NewInt(int64(block.Height))
					(*receipts)[i].TransactionIndex = uint(i)

					for k := range (*receipts)[i].Logs {
						(*receipts)[i].Logs[k].BlockHash = (*receipts)[i].BlockHash
						(*receipts)[i].Logs[k].TxHash = (*receipts)[i].TxHash
						(*receipts)[i].Logs[k].TxIndex = (*receipts)[i].TransactionIndex
					}
					//storageTypes.SaveReceipt(s.datastore, block.Height, txhash, (*receipts)[i])
				}
			}

			common.ParallelWorker(len(*receipts), fm.Concurrency, worker, receipts)
			fm.filters.OnResultsArrived(block.Height, *receipts, ethCommon.BytesToHash(blockHash))
		}

		//s.MsgBroker.Send(actor.MsgLatestHeight, height)
	}

	return nil
}
