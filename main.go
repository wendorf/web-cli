package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/cf"
	"github.com/cloudfoundry/cli/cf/app"
	"github.com/cloudfoundry/cli/cf/command_factory"
	"github.com/cloudfoundry/cli/cf/command_registry"
	"github.com/cloudfoundry/cli/cf/command_runner"
	"github.com/cloudfoundry/cli/cf/net"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/plugin/rpc"
	"github.com/codegangsta/cli"
)

var deps = command_registry.NewDependency()

var cmdRegistry = command_registry.Commands

func main() {
	fmt.Println("hello world")

	fmt.Println(cf.Version + "-" + cf.BuiltOnDate)

	/*fmt.Println("flags")
	//fmt.Println(flags)
	fmt.Println("cmdRegistry")
	fmt.Println(cmdRegistry)
	command := cmdRegistry.FindCommand("space")
	metadata := command.MetaData()
	commandFlags := metadata.Flags
	fc := flags.NewFlagContext(commandFlags)
	command.Execute(fc)*/

	rpcService := newCliRpcServer(deps.TeePrinter, deps.TeePrinter)

	cmdFactory := command_factory.NewFactory(deps.Ui, deps.Config, deps.ManifestRepo, deps.RepoLocator, deps.PluginConfig, rpcService)
	requirementsFactory := requirements.NewFactory(deps.Ui, deps.Config, deps.RepoLocator)
	cmdRunner := command_runner.NewRunner(cmdFactory, requirementsFactory, deps.Ui)

	metaDatas := cmdFactory.CommandMetadatas()

	fmt.Println(os.Args[0:])
	theApp := app.NewApp(cmdRunner, metaDatas...)
	callCoreCommand(os.Args[0:], theApp)
	os.Exit(0)
}

func newCliRpcServer(outputCapture terminal.OutputCapture, terminalOutputSwitch terminal.TerminalOutputSwitch) *rpc.CliRpcService {
	cliServer, err := rpc.NewRpcService(nil, outputCapture, terminalOutputSwitch, deps.Config, deps.RepoLocator, rpc.NewNonCodegangstaRunner())
	if err != nil {
		fmt.Println("Error initializing RPC service: ", err)
		os.Exit(1)
	}

	return cliServer
}

func callCoreCommand(args []string, theApp *cli.App) {
	err := theApp.Run(args)
	if err != nil {
		os.Exit(1)
	}
	gateways := gatewaySliceFromMap(deps.Gateways)

	warningsCollector := net.NewWarningsCollector(deps.Ui, gateways...)
	warningsCollector.PrintWarnings()
}

func gatewaySliceFromMap(gateway_map map[string]net.Gateway) []net.WarningProducer {
	gateways := []net.WarningProducer{}
	for _, gateway := range gateway_map {
		gateways = append(gateways, gateway)
	}
	return gateways
}
