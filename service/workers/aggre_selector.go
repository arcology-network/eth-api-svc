package workers

import (
	ethTypes "github.com/arcology-network/3rd-party/eth/types"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/aggregator/aggregator"
	"github.com/arcology-network/component-lib/log"
	"go.uber.org/zap"
)

const (
	aggrStateInit = iota
	aggrStateCollecting
	aggrStateDone
)

type AggreSelector struct {
	actor.WorkerThread
	aggregator   *aggregator.Aggregator
	savedMessage *actor.Message
	state        int
}

//return a Subscriber struct
func NewAggreSelector(concurrency int, groupid string) *AggreSelector {
	agg := AggreSelector{}
	agg.Set(concurrency, groupid)
	agg.state = aggrStateDone
	agg.aggregator = aggregator.NewAggregator()
	return &agg
}

func (a *AggreSelector) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		aggrStateInit:       {actor.MsgReceipts, actor.MsgInclusive},
		aggrStateCollecting: {actor.MsgReceipts},
		aggrStateDone:       {actor.MsgBlockCompleted},
	}
}

func (a *AggreSelector) GetCurrentState() int {
	return a.state
}

func (*AggreSelector) OnStart() {}
func (*AggreSelector) Stop()    {}

func (a *AggreSelector) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch a.state {
	case aggrStateInit:
		if msg.Name == actor.MsgReceipts {
			a.onDataReceived(msg)
		} else if msg.Name == actor.MsgInclusive {
			if a.onListReceived(msg) {
				a.state = aggrStateDone
			} else {
				a.state = aggrStateCollecting
			}
		}
	case aggrStateCollecting:
		if msg.Name == actor.MsgReceipts {
			if a.onDataReceived(msg) {
				a.state = aggrStateDone
			}
		}
	case aggrStateDone:
		if msg.Name == actor.MsgBlockCompleted {
			a.aggregator.OnClearInfoReceived()
			a.state = aggrStateInit
		}

	}

	return nil
}
func (a *AggreSelector) onListReceived(msg *actor.Message) bool {
	a.AddLog(log.LogLevel_Info, " >>>>>>> execMsgInclusive", zap.Uint64("msgHeight", msg.Height))
	a.savedMessage = msg.CopyHeader()
	inclusive := msg.Data.(*types.InclusiveList)
	inclusive.Mode = types.InclusiveMode_Message
	copyInclusive := inclusive.CopyListAddHeight(msg.Height, msg.Round)
	result, _ := a.aggregator.OnListReceived(copyInclusive)
	return a.sendIfDone(result)
}
func (a *AggreSelector) onDataReceived(msg *actor.Message) bool {
	receipts := msg.Data.(*[]*ethTypes.Receipt)
	for _, receipt := range *receipts {
		newhash := common.ToNewHash(receipt.TxHash, msg.Height, msg.Round)
		result := a.aggregator.OnDataReceived(newhash, receipt)
		if a.sendIfDone(result) {
			return true
		}
	}
	return false
}

func (a *AggreSelector) sendIfDone(SelectedData *[]*interface{}) bool {
	if SelectedData != nil {
		receipts := make([]*ethTypes.Receipt, len(*SelectedData))
		for i, v := range *SelectedData {
			receipt := (*v).(*ethTypes.Receipt)
			receipts[i] = receipt
		}

		a.LatestMessage = a.savedMessage
		a.MsgBroker.LatestMessage = a.savedMessage

		a.AddLog(log.LogLevel_Info, "AggreSelector send selected receipts")
		a.MsgBroker.Send(actor.MsgSelectedReceipts, &receipts)
		return true
	}
	return false
}
