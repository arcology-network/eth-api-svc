package service

import (
	"github.com/arcology-network/component-lib/actor"
	aggrv3 "github.com/arcology-network/component-lib/aggregator/v3"
	"github.com/arcology-network/component-lib/kafka"
	"github.com/arcology-network/component-lib/streamer"
	"github.com/arcology-network/eth-api-svc/service/workers"

	//"github.com/sirupsen/logrus"

	internal "github.com/arcology-network/eth-api-svc/backend"
	"github.com/spf13/viper"
)

type Config struct {
	concurrency int
	groupid     string
	filters     *internal.Filters
}

//return a Subscriber struct
func NewConfig(filters *internal.Filters) *Config {
	return &Config{
		concurrency: viper.GetInt("concurrency"),
		groupid:     "ethapi",
		filters:     filters,
	}
}

func (cfg *Config) Start() {

	broker := streamer.NewStatefulStreamer()
	//00 initializer
	initializer := actor.NewActor(
		"initializer",
		broker,
		[]string{actor.MsgStarting},
		[]string{actor.MsgStartSub},
		[]int{1},
		workers.NewInitializer(cfg.concurrency, cfg.groupid),
	)
	initializer.Connect(streamer.NewDisjunctions(initializer, 1))

	receiveMseeages := []string{
		actor.MsgReceipts,
		actor.MsgInclusive,
		actor.MsgPendingBlock,
		actor.MsgBlockCompleted,
	}

	receiveTopics := []string{
		viper.GetString("msgexch"),
		viper.GetString("receipts"),
		viper.GetString("inclusive-txs"),
		viper.GetString("local-block"),
	}

	//01 apc module kafkaDownloader
	apcKafkaDownloader := actor.NewActor(
		"apcKafkaDownloader",
		broker,
		[]string{actor.MsgStartSub},
		receiveMseeages,
		[]int{1, 1, 1, 1},
		kafka.NewKafkaDownloader(cfg.concurrency, cfg.groupid, receiveTopics, receiveMseeages, viper.GetString("mqaddr")),
	)
	apcKafkaDownloader.Connect(streamer.NewDisjunctions(apcKafkaDownloader, 10))

	aggrSelectorBase := &actor.FSMController{}
	aggrSelectorBase.EndWith(aggrv3.NewReceiptAggreSelector(cfg.concurrency, cfg.groupid))
	//04 aggre-receipt
	aggreReceipt := actor.NewActor(
		"aggreReceipt",
		broker,
		[]string{
			actor.MsgReceipts,
			actor.MsgInclusive,
			actor.MsgBlockCompleted,
		},
		[]string{
			actor.MsgSelectedReceipts,
		},
		[]int{1},
		aggrSelectorBase, //workers.NewAggreSelector(cfg.concurrency, cfg.groupid),
	)
	aggreReceipt.Connect(streamer.NewDisjunctions(aggreReceipt, 1))

	//09 filterManager
	filterManager := actor.NewActor(
		"filterManager",
		broker,
		[]string{
			actor.MsgSelectedReceipts,
			actor.MsgBlockCompleted,
			actor.MsgPendingBlock,
		},
		[]string{},
		[]int{},
		workers.NewFilterManager(cfg.concurrency, cfg.groupid, cfg.filters),
	)
	filterManager.Connect(streamer.NewConjunctions(filterManager))

	//starter
	selfStarter := streamer.NewDefaultProducer("selfStarter", []string{actor.MsgStarting}, []int{1})
	broker.RegisterProducer(selfStarter)
	broker.Serve()

	//start signel
	streamerStarting := actor.Message{
		Name:   actor.MsgStarting,
		Height: 0,
		Round:  0,
		Data:   "start",
	}
	broker.Send(actor.MsgStarting, &streamerStarting)

}

func (*Config) Stop() {}
