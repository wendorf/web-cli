package route

import (
	"github.com/cloudfoundry/cli/cf/api"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/flag_helpers"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/codegangsta/cli"
)

type DeleteRoute struct {
	ui        terminal.UI
	config    core_config.Reader
	routeRepo api.RouteRepository
	domainReq requirements.DomainRequirement
}

func NewDeleteRoute(ui terminal.UI, config core_config.Reader, routeRepo api.RouteRepository) (cmd *DeleteRoute) {
	cmd = new(DeleteRoute)
	cmd.ui = ui
	cmd.config = config
	cmd.routeRepo = routeRepo
	return
}

func (cmd *DeleteRoute) Metadata() command_metadata.CommandMetadata {
	return command_metadata.CommandMetadata{
		Name:        "delete-route",
		Description: T("Delete a route"),
		Usage:       T("CF_NAME delete-route DOMAIN [-n HOSTNAME] [-f]"),
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "f", Usage: T("Force deletion without confirmation")},
			flag_helpers.NewStringFlag("n", T("Hostname")),
		},
	}
}

func (cmd *DeleteRoute) GetRequirements(requirementsFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 1 {
		cmd.ui.FailWithUsage(c)
	}

	cmd.domainReq = requirementsFactory.NewDomainRequirement(c.Args()[0])

	reqs = []requirements.Requirement{
		requirementsFactory.NewLoginRequirement(),
		cmd.domainReq,
	}
	return
}

func (cmd *DeleteRoute) Run(c *cli.Context) {
	host := c.String("n")
	domainName := c.Args()[0]

	url := domainName
	if host != "" {
		url = host + "." + domainName
	}
	if !c.Bool("f") {
		if !cmd.ui.ConfirmDelete("route", url) {
			return
		}
	}

	cmd.ui.Say(T("Deleting route {{.URL}}...", map[string]interface{}{"URL": terminal.EntityNameColor(url)}))

	domain := cmd.domainReq.GetDomain()
	route, apiErr := cmd.routeRepo.FindByHostAndDomain(host, domain)

	switch apiErr.(type) {
	case nil:
	case *errors.ModelNotFoundError:
		cmd.ui.Warn(T("Unable to delete, route '{{.URL}}' does not exist.",
			map[string]interface{}{"URL": url}))
		return
	default:
		cmd.ui.Failed(apiErr.Error())
		return
	}

	apiErr = cmd.routeRepo.Delete(route.Guid)
	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
		return
	}

	cmd.ui.Ok()
}
