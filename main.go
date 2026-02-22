package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)


type VersionCmd struct {
}

func (cmd *VersionCmd) Run() error {
	fmt.Printf("anyk version %s\n", version)
	return nil
}

var CLI struct {
	Version VersionCmd `cmd:"" help:"Print version and exit."`

	Run RunCmd `cmd:"" help:"Run anyk with a configuration file"`
}

const version = "0.1.0"

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("anyk"),
		kong.Description("Anyk is trying to know if a service within the network is alive and route traffic to it while announcing that prefix using frr"),
		kong.UsageOnError(),
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
