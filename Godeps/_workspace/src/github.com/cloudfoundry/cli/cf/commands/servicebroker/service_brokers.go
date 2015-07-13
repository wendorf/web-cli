package servicebroker

import (
	"sort"

	"github.com/cloudfoundry/cli/cf/api"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/codegangsta/cli"
)

type ListServiceBrokers struct {
	ui     terminal.UI
	config core_config.Reader
	repo   api.ServiceBrokerRepository
}

type serviceBrokerRow struct {
	name string
	url  string
}

type serviceBrokerTable []serviceBrokerRow

func NewListServiceBrokers(ui terminal.UI, config core_config.Reader, repo api.ServiceBrokerRepository) (cmd ListServiceBrokers) {
	cmd.ui = ui
	cmd.config = config
	cmd.repo = repo
	return
}

func (cmd ListServiceBrokers) Metadata() command_metadata.CommandMetadata {
	return command_metadata.CommandMetadata{
		Name:        "service-brokers",
		Description: T("List service brokers"),
		Usage:       "CF_NAME service-brokers",
	}
}

func (cmd ListServiceBrokers) GetRequirements(requirementsFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 0 {
		cmd.ui.FailWithUsage(c)
	}
	reqs = append(reqs, requirementsFactory.NewLoginRequirement())
	return
}

func (cmd ListServiceBrokers) Run(c *cli.Context) {
	sbTable := serviceBrokerTable{}

	cmd.ui.Say(T("Getting service brokers as {{.Username}}...\n",
		map[string]interface{}{
			"Username": terminal.EntityNameColor(cmd.config.Username()),
		}))

	table := cmd.ui.Table([]string{T("name"), T("url")})
	foundBrokers := false
	apiErr := cmd.repo.ListServiceBrokers(func(serviceBroker models.ServiceBroker) bool {
		sbTable = append(sbTable, serviceBrokerRow{
			name: serviceBroker.Name,
			url:  serviceBroker.Url,
		})
		foundBrokers = true
		return true
	})

	sort.Sort(sbTable)

	for _, sb := range sbTable {
		table.Add(sb.name, sb.url)
	}

	table.Print()

	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
		return
	}

	if !foundBrokers {
		cmd.ui.Say(T("No service brokers found"))
	}
}

func (a serviceBrokerTable) Len() int           { return len(a) }
func (a serviceBrokerTable) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a serviceBrokerTable) Less(i, j int) bool { return a[i].name < a[j].name }
