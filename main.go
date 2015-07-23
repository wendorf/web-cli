package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/cf"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/cloudfoundry/cli/maincli"
	"github.com/codegangsta/cli"
	"github.com/gopherjs/gopherjs/js"
)

func main() {
	maincli.Deps.Ui = NewUI()
	js.Global.Set("cf", callCommand)
}

func callCommand(args []string) {
	go func() {
		maincli.Maincli(args)
	}()
}

type browserUI struct{}

func NewUI() terminal.UI {
	return &browserUI{}
}

func (ui *browserUI) PrintPaginator(rows []string, err error) {
	if err != nil {
		ui.Failed(err.Error())
		return
	}

	for _, row := range rows {
		ui.Say(row)
	}
}

func (ui *browserUI) PrintCapturingNoOutput(message string, args ...interface{}) {
	ui.Say(message, args...)
}

func (ui *browserUI) Say(message string, args ...interface{}) {
	fullMessage := fmt.Sprintf(message+"\n", args...)
	fullMessage = strings.Replace(fullMessage, "\n", "\r\n", -1)
	js.Global.Get("term").Call("write", fullMessage)
}

func (ui browserUI) AskForPassword(prompt string, args ...interface{}) (passwd string) {
	return ui.Ask(prompt, args...)
}

func (ui *browserUI) Warn(message string, args ...interface{}) {
	message = fmt.Sprintf(message, args...)
	ui.Say(terminal.WarningColor(message))
}

func (c *browserUI) ConfirmDeleteWithAssociations(modelType, modelName string) bool {
	return c.confirmDelete(T("Really delete the {{.ModelType}} {{.ModelName}} and everything associated with it?",
		map[string]interface{}{
			"ModelType": modelType,
			"ModelName": terminal.EntityNameColor(modelName),
		}))
}

func (c *browserUI) ConfirmDelete(modelType, modelName string) bool {
	return c.confirmDelete(T("Really delete the {{.ModelType}} {{.ModelName}}?",
		map[string]interface{}{
			"ModelType": modelType,
			"ModelName": terminal.EntityNameColor(modelName),
		}))
}

func (c *browserUI) confirmDelete(message string) bool {
	result := c.Confirm(message)

	if !result {
		c.Warn(T("Delete cancelled"))
	}

	return result
}

func (c *browserUI) Confirm(message string, args ...interface{}) bool {
	response := c.Ask(message, args...)
	switch strings.ToLower(response) {
	case "y", "yes", T("yes"):
		return true
	}
	return false
}

func (c *browserUI) Ask(prompt string, args ...interface{}) (answer string) {
	return js.Global.Call("prompt").String()
}

func (c *browserUI) Ok() {
	c.Say(terminal.SuccessColor(T("OK")))
}

const QuietPanic = "This shouldn't print anything"

func (c *browserUI) Failed(message string, args ...interface{}) {
	message = fmt.Sprintf(message, args...)

	if T == nil {
		c.Say(terminal.FailureColor("FAILED"))
		c.Say(message)

		trace.Logger.Print("FAILED")
		trace.Logger.Print(message)
		c.PanicQuietly()
	} else {
		c.Say(terminal.FailureColor(T("FAILED")))
		c.Say(message)

		trace.Logger.Print(T("FAILED"))
		trace.Logger.Print(message)
		c.PanicQuietly()
	}
}

func (c *browserUI) PanicQuietly() {
	js.Global.Get("console").Call("error", QuietPanic)
}

func (c *browserUI) FailWithUsage(context *cli.Context) {
	c.Say(terminal.FailureColor(T("FAILED")))
	c.Say(T("Incorrect Usage.\n"))
	cli.ShowCommandHelp(context, context.Command.Name)
	c.Say("")
}

func (ui *browserUI) ShowConfiguration(config core_config.Reader) {
	table := terminal.NewTable(ui, []string{"", ""})

	if config.HasAPIEndpoint() {
		table.Add(
			T("API endpoint:"),
			T("{{.ApiEndpoint}} (API version: {{.ApiVersionString}})",
				map[string]interface{}{
					"ApiEndpoint":      terminal.EntityNameColor(config.ApiEndpoint()),
					"ApiVersionString": terminal.EntityNameColor(config.ApiVersion()),
				}),
		)
	}

	if !config.IsLoggedIn() {
		table.Print()
		ui.Say(terminal.NotLoggedInText())
		return
	} else {
		table.Add(
			T("User:"),
			terminal.EntityNameColor(config.UserEmail()),
		)
	}

	if !config.HasOrganization() && !config.HasSpace() {
		table.Print()
		command := fmt.Sprintf("%s target -o ORG -s SPACE", cf.Name())
		ui.Say(T("No org or space targeted, use '{{.CFTargetCommand}}'",
			map[string]interface{}{
				"CFTargetCommand": terminal.CommandColor(command),
			}))
		return
	}

	if config.HasOrganization() {
		table.Add(
			T("Org:"),
			terminal.EntityNameColor(config.OrganizationFields().Name),
		)
	} else {
		command := fmt.Sprintf("%s target -o Org", cf.Name())
		table.Add(
			T("Org:"),
			T("No org targeted, use '{{.CFTargetCommand}}'",
				map[string]interface{}{
					"CFTargetCommand": terminal.CommandColor(command),
				}),
		)
	}

	if config.HasSpace() {
		table.Add(
			T("Space:"),
			terminal.EntityNameColor(config.SpaceFields().Name),
		)
	} else {
		command := fmt.Sprintf("%s target -s SPACE", cf.Name())
		table.Add(
			T("Space:"),
			T("No space targeted, use '{{.CFTargetCommand}}'", map[string]interface{}{"CFTargetCommand": terminal.CommandColor(command)}),
		)
	}

	table.Print()
}

func (c *browserUI) LoadingIndication() {
	c.Say(".")
}

func (c *browserUI) Wait(duration time.Duration) {
	time.Sleep(duration)
}

func (ui *browserUI) Table(headers []string) terminal.Table {
	return terminal.NewTable(ui, headers)
}

func (ui *browserUI) NotifyUpdateIfNeeded(config core_config.Reader) {
	if !config.IsMinCliVersion(cf.Version) {
		ui.Say("")
		ui.Say(T("Cloud Foundry API version {{.ApiVer}} requires CLI version {{.CliMin}}.  You are currently on version {{.CliVer}}. To upgrade your CLI, please visit: https://github.com/cloudfoundry/cli#downloads",
			map[string]interface{}{
				"ApiVer": config.ApiVersion(),
				"CliMin": config.MinCliVersion(),
				"CliVer": cf.Version,
			}))
	}
}
