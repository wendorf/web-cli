package organization

import (
	"github.com/cloudfoundry/cli/cf/api/quotas"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/codegangsta/cli"
)

type SetQuota struct {
	ui        terminal.UI
	config    core_config.Reader
	quotaRepo quotas.QuotaRepository
	orgReq    requirements.OrganizationRequirement
}

func NewSetQuota(ui terminal.UI, config core_config.Reader, quotaRepo quotas.QuotaRepository) (cmd *SetQuota) {
	cmd = new(SetQuota)
	cmd.ui = ui
	cmd.config = config
	cmd.quotaRepo = quotaRepo
	return
}

func (cmd *SetQuota) Metadata() command_metadata.CommandMetadata {
	return command_metadata.CommandMetadata{
		Name:        "set-quota",
		Description: T("Assign a quota to an org"),
		Usage:       T("CF_NAME set-quota ORG QUOTA\n\n") + T("TIP:\n") + T("   View allowable quotas with 'CF_NAME quotas'"),
	}
}

func (cmd *SetQuota) GetRequirements(requirementsFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 2 {
		cmd.ui.FailWithUsage(c)
	}

	if cmd.orgReq == nil {
		cmd.orgReq = requirementsFactory.NewOrganizationRequirement(c.Args()[0])
	} else {
		cmd.orgReq.SetOrganizationName(c.Args()[0])
	}

	reqs = []requirements.Requirement{
		requirementsFactory.NewLoginRequirement(),
		cmd.orgReq,
	}
	return
}

func (cmd *SetQuota) Run(c *cli.Context) {
	org := cmd.orgReq.GetOrganization()
	quotaName := c.Args()[1]
	quota, apiErr := cmd.quotaRepo.FindByName(quotaName)

	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
		return
	}

	cmd.ui.Say(T("Setting quota {{.QuotaName}} to org {{.OrgName}} as {{.Username}}...",
		map[string]interface{}{
			"QuotaName": terminal.EntityNameColor(quota.Name),
			"OrgName":   terminal.EntityNameColor(org.Name),
			"Username":  terminal.EntityNameColor(cmd.config.Username())}))

	apiErr = cmd.quotaRepo.AssignQuotaToOrg(org.Guid, quota.Guid)
	if apiErr != nil {
		cmd.ui.Failed(apiErr.Error())
		return
	}

	cmd.ui.Ok()
}
