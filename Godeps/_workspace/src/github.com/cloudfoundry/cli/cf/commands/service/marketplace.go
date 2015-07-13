package service

import (
	"sort"
	"strings"

	"github.com/cloudfoundry/cli/cf/flag_helpers"
	. "github.com/cloudfoundry/cli/cf/i18n"

	"github.com/cloudfoundry/cli/cf/actors/service_builder"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/codegangsta/cli"
)

type MarketplaceServices struct {
	ui             terminal.UI
	config         core_config.Reader
	serviceBuilder service_builder.ServiceBuilder
}

func NewMarketplaceServices(ui terminal.UI, config core_config.Reader, serviceBuilder service_builder.ServiceBuilder) MarketplaceServices {
	return MarketplaceServices{
		ui:             ui,
		config:         config,
		serviceBuilder: serviceBuilder,
	}
}

func (cmd MarketplaceServices) Metadata() command_metadata.CommandMetadata {
	return command_metadata.CommandMetadata{
		Name:        "marketplace",
		ShortName:   "m",
		Description: T("List available offerings in the marketplace"),
		Usage:       "CF_NAME marketplace",
		Flags: []cli.Flag{
			flag_helpers.NewStringFlag("s", T("Show plan details for a particular service offering")),
		},
	}
}

func (cmd MarketplaceServices) GetRequirements(requirementsFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 0 {
		cmd.ui.FailWithUsage(c)
	}
	reqs = append(reqs, requirementsFactory.NewApiEndpointRequirement())
	return
}

func (cmd MarketplaceServices) Run(c *cli.Context) {
	serviceName := c.String("s")

	if serviceName != "" {
		cmd.marketplaceByService(serviceName)
	} else {
		cmd.marketplace()
	}
}

func (cmd MarketplaceServices) marketplaceByService(serviceName string) {
	var (
		serviceOffering models.ServiceOffering
		apiErr          error
	)

	if cmd.config.HasSpace() {
		cmd.ui.Say(T("Getting service plan information for service {{.ServiceName}} as {{.CurrentUser}}...",
			map[string]interface{}{
				"ServiceName": terminal.EntityNameColor(serviceName),
				"CurrentUser": terminal.EntityNameColor(cmd.config.Username()),
			}))
		serviceOffering, apiErr = cmd.serviceBuilder.GetServiceByNameForSpaceWithPlans(serviceName, cmd.config.SpaceFields().Guid)
	} else if !cmd.config.IsLoggedIn() {
		cmd.ui.Say(T("Getting service plan information for service {{.ServiceName}}...", map[string]interface{}{"ServiceName": terminal.EntityNameColor(serviceName)}))
		serviceOffering, apiErr = cmd.serviceBuilder.GetServiceByNameWithPlans(serviceName)
	} else {
		cmd.ui.Failed(T("Cannot list plan information for {{.ServiceName}} without a targeted space",
			map[string]interface{}{"ServiceName": terminal.EntityNameColor(serviceName)}))
	}

	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
		return
	}

	cmd.ui.Ok()
	cmd.ui.Say("")

	if serviceOffering.Guid == "" {
		cmd.ui.Say(T("Service offering not found"))
		return
	}

	table := terminal.NewTable(cmd.ui, []string{T("service plan"), T("description"), T("free or paid")})
	for _, plan := range serviceOffering.Plans {
		var freeOrPaid string
		if plan.Free {
			freeOrPaid = "free"
		} else {
			freeOrPaid = "paid"
		}
		table.Add(plan.Name, plan.Description, freeOrPaid)
	}

	table.Print()
}

func (cmd MarketplaceServices) marketplace() {
	var (
		serviceOfferings models.ServiceOfferings
		apiErr           error
	)

	if cmd.config.HasSpace() {
		cmd.ui.Say(T("Getting services from marketplace in org {{.OrgName}} / space {{.SpaceName}} as {{.CurrentUser}}...",
			map[string]interface{}{
				"OrgName":     terminal.EntityNameColor(cmd.config.OrganizationFields().Name),
				"SpaceName":   terminal.EntityNameColor(cmd.config.SpaceFields().Name),
				"CurrentUser": terminal.EntityNameColor(cmd.config.Username()),
			}))
		serviceOfferings, apiErr = cmd.serviceBuilder.GetServicesForSpaceWithPlans(cmd.config.SpaceFields().Guid)
	} else if !cmd.config.IsLoggedIn() {
		cmd.ui.Say(T("Getting all services from marketplace..."))
		serviceOfferings, apiErr = cmd.serviceBuilder.GetAllServicesWithPlans()
	} else {
		cmd.ui.Failed(T("Cannot list marketplace services without a targeted space"))
	}

	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
		return
	}

	cmd.ui.Ok()
	cmd.ui.Say("")

	if len(serviceOfferings) == 0 {
		cmd.ui.Say(T("No service offerings found"))
		return
	}

	table := terminal.NewTable(cmd.ui, []string{T("service"), T("plans"), T("description")})

	sort.Sort(serviceOfferings)
	var paidPlanExists bool
	for _, offering := range serviceOfferings {
		planNames := ""

		for _, plan := range offering.Plans {
			if plan.Name == "" {
				continue
			}
			if plan.Free {
				planNames += ", " + plan.Name
			} else {
				paidPlanExists = true
				planNames += ", " + plan.Name + "*"
			}
		}

		planNames = strings.TrimPrefix(planNames, ", ")

		table.Add(offering.Label, planNames, offering.Description)
	}

	table.Print()
	if paidPlanExists {
		cmd.ui.Say(T("\n* These service plans have an associated cost. Creating a service instance will incur this cost."))
	}
	cmd.ui.Say(T("\nTIP:  Use 'cf marketplace -s SERVICE' to view descriptions of individual plans of a given service."))
}
