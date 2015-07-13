package quota

import (
	"github.com/cloudfoundry/cli/cf/api/quotas"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/errors"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/codegangsta/cli"
)

type DeleteQuota struct {
	ui        terminal.UI
	config    core_config.Reader
	quotaRepo quotas.QuotaRepository
	orgReq    requirements.OrganizationRequirement
}

func NewDeleteQuota(ui terminal.UI, config core_config.Reader, quotaRepo quotas.QuotaRepository) (cmd *DeleteQuota) {
	return &DeleteQuota{
		ui:        ui,
		config:    config,
		quotaRepo: quotaRepo,
	}
}

func (cmd *DeleteQuota) Metadata() command_metadata.CommandMetadata {
	return command_metadata.CommandMetadata{
		Name:        "delete-quota",
		Description: T("Delete a quota"),
		Usage:       T("CF_NAME delete-quota QUOTA [-f]"),
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "f", Usage: T("Force deletion without confirmation")},
		},
	}
}

func (cmd *DeleteQuota) GetRequirements(requirementsFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 1 {
		err = errors.New(T("Incorrect Usage"))
		cmd.ui.FailWithUsage(c)
		return
	}

	reqs = []requirements.Requirement{
		requirementsFactory.NewLoginRequirement(),
	}
	return
}

func (cmd *DeleteQuota) Run(c *cli.Context) {
	quotaName := c.Args()[0]

	if !c.Bool("f") {
		response := cmd.ui.ConfirmDelete("quota", quotaName)
		if !response {
			return
		}
	}

	cmd.ui.Say(T("Deleting quota {{.QuotaName}} as {{.Username}}...", map[string]interface{}{
		"QuotaName": terminal.EntityNameColor(quotaName),
		"Username":  terminal.EntityNameColor(cmd.config.Username()),
	}))

	quota, apiErr := cmd.quotaRepo.FindByName(quotaName)

	switch (apiErr).(type) {
	case nil: // no error
	case *errors.ModelNotFoundError:
		cmd.ui.Ok()
		cmd.ui.Warn(T("Quota {{.QuotaName}} does not exist", map[string]interface{}{"QuotaName": quotaName}))
		return
	default:
		cmd.ui.Failed(apiErr.Error())
	}

	apiErr = cmd.quotaRepo.Delete(quota.Guid)
	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
	}

	cmd.ui.Ok()
}
