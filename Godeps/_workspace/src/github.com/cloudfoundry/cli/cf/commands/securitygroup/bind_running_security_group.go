package securitygroup

import (
	"strings"

	"github.com/cloudfoundry/cli/cf/api/security_groups"
	"github.com/cloudfoundry/cli/cf/api/security_groups/defaults/running"
	"github.com/cloudfoundry/cli/cf/command"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/codegangsta/cli"
)

type bindToRunningGroup struct {
	ui                terminal.UI
	configRepo        core_config.Reader
	securityGroupRepo security_groups.SecurityGroupRepo
	runningGroupRepo  running.RunningSecurityGroupsRepo
}

func NewBindToRunningGroup(ui terminal.UI, configRepo core_config.Reader, securityGroupRepo security_groups.SecurityGroupRepo, runningGroupRepo running.RunningSecurityGroupsRepo) command.Command {
	return &bindToRunningGroup{
		ui:                ui,
		configRepo:        configRepo,
		securityGroupRepo: securityGroupRepo,
		runningGroupRepo:  runningGroupRepo,
	}
}

func (cmd *bindToRunningGroup) Metadata() command_metadata.CommandMetadata {
	primaryUsage := T("CF_NAME bind-running-security-group SECURITY_GROUP")
	tipUsage := T("TIP: Changes will not apply to existing running applications until they are restarted.")
	return command_metadata.CommandMetadata{
		Name:        "bind-running-security-group",
		Description: T("Bind a security group to the list of security groups to be used for running applications"),
		Usage:       strings.Join([]string{primaryUsage, tipUsage}, "\n\n"),
	}
}

func (cmd *bindToRunningGroup) GetRequirements(requirementsFactory requirements.Factory, context *cli.Context) ([]requirements.Requirement, error) {
	if len(context.Args()) != 1 {
		cmd.ui.FailWithUsage(context)
	}

	return []requirements.Requirement{
		requirementsFactory.NewLoginRequirement(),
	}, nil
}

func (cmd *bindToRunningGroup) Run(context *cli.Context) {
	name := context.Args()[0]

	securityGroup, err := cmd.securityGroupRepo.Read(name)
	if err != nil {
		cmd.ui.Failed(err.Error())
	}

	cmd.ui.Say(T("Binding security group {{.security_group}} to defaults for running as {{.username}}",
		map[string]interface{}{
			"security_group": terminal.EntityNameColor(securityGroup.Name),
			"username":       terminal.EntityNameColor(cmd.configRepo.Username()),
		}))

	err = cmd.runningGroupRepo.BindToRunningSet(securityGroup.Guid)
	if err != nil {
		cmd.ui.Failed(err.Error())
	}

	cmd.ui.Ok()
	cmd.ui.Say("\n\n")
	cmd.ui.Say(T("TIP: Changes will not apply to existing running applications until they are restarted."))
}
