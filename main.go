package main

import (
	"os"

	_ "net/http/pprof"

	tmCli "github.com/arcology-network/3rd-party/tm/cli"
	"github.com/arcology-network/eth-api-svc/service"
)

func main() {

	st := service.StartCmd

	cmd := tmCli.PrepareMainCmd(st, "BC", os.ExpandEnv("$HOME/monacos/ethapi"))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
