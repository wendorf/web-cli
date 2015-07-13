package route

import (
	"github.com/cloudfoundry/cli/cf/api"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/flag_helpers"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/codegangsta/cli"
)

type UnmapRoute struct {
	ui        terminal.UI
	config    core_config.Reader
	routeRepo api.RouteRepository
	appReq    requirements.ApplicationRequirement
	domainReq requirements.DomainRequirement
}

func NewUnmapRoute(ui terminal.UI, config core_config.Reader, routeRepo api.RouteRepository) (cmd *UnmapRoute) {
	cmd = new(UnmapRoute)
	cmd.ui = ui
	cmd.config = config
	cmd.routeRepo = routeRepo
	return
}

func (cmd *UnmapRoute) Metadata() command_metadata.CommandMetadata {
	return command_metadata.CommandMetadata{
		Name:        "unmap-route",
		Description: T("Remove a url route from an app"),
		Usage:       T("CF_NAME unmap-route APP_NAME DOMAIN [-n HOSTNAME]"),
		Flags: []cli.Flag{
			flag_helpers.NewStringFlag("n", T("Hostname")),
		},
	}
}

func (cmd *UnmapRoute) GetRequirements(requirementsFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 2 {
		cmd.ui.FailWithUsage(c)
	}

	domainName := c.Args()[1]

	if cmd.appReq == nil {
		cmd.appReq = requirementsFactory.NewApplicationRequirement(c.Args()[0])
	} else {
		cmd.appReq.SetApplicationName(c.Args()[0])
	}

	cmd.domainReq = requirementsFactory.NewDomainRequirement(domainName)

	reqs = []requirements.Requirement{
		requirementsFactory.NewLoginRequirement(),
		cmd.appReq,
		cmd.domainReq,
	}
	return
}

func (cmd *UnmapRoute) Run(c *cli.Context) {
	hostName := c.String("n")
	domain := cmd.domainReq.GetDomain()
	app := cmd.appReq.GetApplication()

	route, apiErr := cmd.routeRepo.FindByHostAndDomain(hostName, domain)
	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
	}
	cmd.ui.Say(T("Removing route {{.URL}} from app {{.AppName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...",
		map[string]interface{}{
			"URL":       terminal.EntityNameColor(route.URL()),
			"AppName":   terminal.EntityNameColor(app.Name),
			"OrgName":   terminal.EntityNameColor(cmd.config.OrganizationFields().Name),
			"SpaceName": terminal.EntityNameColor(cmd.config.SpaceFields().Name),
			"Username":  terminal.EntityNameColor(cmd.config.Username())}))

	var routeFound bool
	for _, routeApp := range route.Apps {
		if routeApp.Guid == app.Guid {
			routeFound = true
			apiErr = cmd.routeRepo.Unbind(route.Guid, app.Guid)
			if apiErr != nil {
				cmd.ui.Failed(apiErr.Error())
				return
			}
		}
	}
	cmd.ui.Ok()

	if !routeFound {
		cmd.ui.Warn(T("\nRoute to be unmapped is not currently mapped to the application."))
	}

}
