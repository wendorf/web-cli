package securitygroup_test

import (
	"github.com/cloudfoundry/cli/cf/api/fakes"
	test_org "github.com/cloudfoundry/cli/cf/api/organizations/fakes"
	testapi "github.com/cloudfoundry/cli/cf/api/security_groups/fakes"
	zoidberg "github.com/cloudfoundry/cli/cf/api/security_groups/spaces/fakes"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/models"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/cf/commands/securitygroup"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("bind-security-group command", func() {
	var (
		ui                    *testterm.FakeUI
		cmd                   BindSecurityGroup
		configRepo            core_config.ReadWriter
		fakeSecurityGroupRepo *testapi.FakeSecurityGroupRepo
		requirementsFactory   *testreq.FakeReqFactory
		fakeSpaceRepo         *fakes.FakeSpaceRepository
		fakeOrgRepo           *test_org.FakeOrganizationRepository
		fakeSpaceBinder       *zoidberg.FakeSecurityGroupSpaceBinder
	)

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		fakeOrgRepo = &test_org.FakeOrganizationRepository{}
		fakeSpaceRepo = &fakes.FakeSpaceRepository{}
		requirementsFactory = &testreq.FakeReqFactory{}
		fakeSecurityGroupRepo = &testapi.FakeSecurityGroupRepo{}
		configRepo = testconfig.NewRepositoryWithDefaults()
		fakeSpaceBinder = &zoidberg.FakeSecurityGroupSpaceBinder{}
		cmd = NewBindSecurityGroup(ui, configRepo, fakeSecurityGroupRepo, fakeSpaceRepo, fakeOrgRepo, fakeSpaceBinder)
	})

	runCommand := func(args ...string) bool {
		return testcmd.RunCommand(cmd, args, requirementsFactory)
	}

	Describe("requirements", func() {
		It("fails when the user is not logged in", func() {
			Expect(runCommand("my-craaaaaazy-security-group", "my-org", "my-space")).To(BeFalse())
		})

		It("succeeds when the user is logged in", func() {
			requirementsFactory.LoginSuccess = true

			Expect(runCommand("my-craaaaaazy-security-group", "my-org", "my-space")).To(BeTrue())
		})

		It("fails with usage when not provided the name of a security group, org, and space", func() {
			requirementsFactory.LoginSuccess = true
			runCommand("one fish", "two fish", "three fish", "purple fish")

			Expect(ui.FailedWithUsage).To(BeTrue())
		})
	})

	Context("when the user is logged in and provides the name of a security group", func() {
		BeforeEach(func() {
			requirementsFactory.LoginSuccess = true
		})

		Context("when a security group with that name does not exist", func() {
			BeforeEach(func() {
				fakeSecurityGroupRepo.ReadReturns(models.SecurityGroup{}, errors.NewModelNotFoundError("security group", "my-nonexistent-security-group"))
			})

			It("fails and tells the user", func() {
				runCommand("my-nonexistent-security-group", "my-org", "my-space")

				Expect(fakeSecurityGroupRepo.ReadArgsForCall(0)).To(Equal("my-nonexistent-security-group"))
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"FAILED"},
					[]string{"security group", "my-nonexistent-security-group", "not found"},
				))
			})
		})

		Context("when the org does not exist", func() {
			BeforeEach(func() {
				fakeOrgRepo.FindByNameReturns(models.Organization{}, errors.New("Org org not found"))
			})

			It("fails and tells the user", func() {
				runCommand("sec group", "org", "space")

				Expect(fakeOrgRepo.FindByNameArgsForCall(0)).To(Equal("org"))
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"FAILED"},
					[]string{"Org", "org", "not found"},
				))
			})
		})

		Context("when the space does not exist", func() {
			BeforeEach(func() {
				org := models.Organization{}
				org.Name = "org-name"
				org.Guid = "org-guid"
				fakeOrgRepo.ListOrgsReturns([]models.Organization{org}, nil)
				fakeOrgRepo.FindByNameReturns(org, nil)
				fakeSpaceRepo.FindByNameInOrgError = errors.NewModelNotFoundError("Space", "space-name")
			})

			It("fails and tells the user", func() {
				runCommand("sec group", "org-name", "space-name")

				Expect(fakeSpaceRepo.FindByNameInOrgName).To(Equal("space-name"))
				Expect(fakeSpaceRepo.FindByNameInOrgOrgGuid).To(Equal("org-guid"))
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"FAILED"},
					[]string{"Space", "space-name", "not found"},
				))
			})
		})

		Context("everything is hunky dory", func() {
			BeforeEach(func() {
				org := models.Organization{}
				org.Name = "org-name"
				org.Guid = "org-guid"
				fakeOrgRepo.ListOrgsReturns([]models.Organization{org}, nil)

				space := models.Space{}
				space.Name = "space-name"
				space.Guid = "space-guid"
				fakeSpaceRepo.FindByNameInOrgSpace = space

				securityGroup := models.SecurityGroup{}
				securityGroup.Name = "security-group"
				securityGroup.Guid = "security-group-guid"
				fakeSecurityGroupRepo.ReadReturns(securityGroup, nil)
			})

			JustBeforeEach(func() {
				runCommand("security-group", "org-name", "space-name")
			})

			It("assigns the security group to the space", func() {
				secGroupGuid, spaceGuid := fakeSpaceBinder.BindSpaceArgsForCall(0)
				Expect(secGroupGuid).To(Equal("security-group-guid"))
				Expect(spaceGuid).To(Equal("space-guid"))
			})

			It("describes what it is doing for the user's benefit", func() {
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Assigning security group security-group to space space-name in org org-name as my-user"},
					[]string{"OK"},
					[]string{"TIP: Changes will not apply to existing running applications until they are restarted."},
				))
			})
		})
	})
})
