package application_test

import (
	testApplication "github.com/cloudfoundry/cli/cf/api/applications/fakes"
	. "github.com/cloudfoundry/cli/cf/commands/application"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/models"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	"github.com/cloudfoundry/cli/testhelpers/maker"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/cli/testhelpers/matchers"
)

var _ = Describe("scale command", func() {
	var (
		requirementsFactory *testreq.FakeReqFactory
		restarter           *testcmd.FakeApplicationRestarter
		appRepo             *testApplication.FakeApplicationRepository
		ui                  *testterm.FakeUI
		config              core_config.ReadWriter
		cmd                 *Scale
		app                 models.Application
	)

	BeforeEach(func() {
		requirementsFactory = &testreq.FakeReqFactory{LoginSuccess: true, TargetedSpaceSuccess: true}
		restarter = &testcmd.FakeApplicationRestarter{}
		appRepo = &testApplication.FakeApplicationRepository{}
		ui = new(testterm.FakeUI)
		config = testconfig.NewRepositoryWithDefaults()
		cmd = NewScale(ui, config, restarter, appRepo)
	})

	Describe("requirements", func() {
		It("requires the user to be logged in with a targed space", func() {
			args := []string{"-m", "1G", "my-app"}

			requirementsFactory.LoginSuccess = false
			requirementsFactory.TargetedSpaceSuccess = true

			Expect(testcmd.RunCommand(cmd, args, requirementsFactory)).To(BeFalse())

			requirementsFactory.LoginSuccess = true
			requirementsFactory.TargetedSpaceSuccess = false

			Expect(testcmd.RunCommand(cmd, args, requirementsFactory)).To(BeFalse())
		})

		It("requires an app to be specified", func() {
			passed := testcmd.RunCommand(cmd, []string{"-m", "1G"}, requirementsFactory)

			Expect(ui.FailedWithUsage).To(BeTrue())
			Expect(passed).To(BeFalse())
		})

		It("does not require any flags", func() {
			requirementsFactory.LoginSuccess = true
			requirementsFactory.TargetedSpaceSuccess = true

			Expect(testcmd.RunCommand(cmd, []string{"my-app"}, requirementsFactory)).To(BeTrue())
		})
	})

	Describe("scaling an app", func() {
		BeforeEach(func() {
			app = maker.NewApp(maker.Overrides{"name": "my-app", "guid": "my-app-guid"})
			app.InstanceCount = 42
			app.DiskQuota = 1024
			app.Memory = 256

			requirementsFactory.Application = app
			appRepo.UpdateAppResult = app
		})

		Context("when no flags are specified", func() {
			It("prints a description of the app's limits", func() {
				testcmd.RunCommand(cmd, []string{"my-app"}, requirementsFactory)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Showing", "my-app", "my-org", "my-space", "my-user"},
					[]string{"OK"},
					[]string{"memory", "256M"},
					[]string{"disk", "1G"},
					[]string{"instances", "42"},
				))

				Expect(ui.Outputs).ToNot(ContainSubstrings([]string{"Scaling", "my-app", "my-org", "my-space", "my-user"}))
			})
		})

		Context("when the user does not confirm 'yes'", func() {
			It("does not restart the app", func() {
				ui.Inputs = []string{"whatever"}
				testcmd.RunCommand(cmd, []string{"-i", "5", "-m", "512M", "-k", "2G", "my-app"}, requirementsFactory)

				Expect(restarter.ApplicationRestartCallCount()).To(Equal(0))
			})
		})

		Context("when the user provides the -f flag", func() {
			It("does not prompt the user", func() {
				testcmd.RunCommand(cmd, []string{"-f", "-i", "5", "-m", "512M", "-k", "2G", "my-app"}, requirementsFactory)

				application, orgName, spaceName := restarter.ApplicationRestartArgsForCall(0)
				Expect(application).To(Equal(app))
				Expect(orgName).To(Equal(config.OrganizationFields().Name))
				Expect(spaceName).To(Equal(config.SpaceFields().Name))
			})
		})

		Context("when the user confirms they want to restart", func() {
			BeforeEach(func() {
				ui.Inputs = []string{"yes"}
			})

			It("can set an app's instance count, memory limit and disk limit", func() {
				testcmd.RunCommand(cmd, []string{"-i", "5", "-m", "512M", "-k", "2G", "my-app"}, requirementsFactory)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Scaling", "my-app", "my-org", "my-space", "my-user"},
					[]string{"OK"},
				))

				Expect(ui.Prompts).To(ContainSubstrings([]string{"This will cause the app to restart", "Are you sure", "my-app"}))

				application, orgName, spaceName := restarter.ApplicationRestartArgsForCall(0)
				Expect(application).To(Equal(app))
				Expect(orgName).To(Equal(config.OrganizationFields().Name))
				Expect(spaceName).To(Equal(config.SpaceFields().Name))

				Expect(appRepo.UpdateAppGuid).To(Equal("my-app-guid"))
				Expect(*appRepo.UpdateParams.Memory).To(Equal(int64(512)))
				Expect(*appRepo.UpdateParams.InstanceCount).To(Equal(5))
				Expect(*appRepo.UpdateParams.DiskQuota).To(Equal(int64(2048)))
			})

			It("does not scale the memory and disk limits if they are not specified", func() {
				testcmd.RunCommand(cmd, []string{"-i", "5", "my-app"}, requirementsFactory)

				Expect(restarter.ApplicationRestartCallCount()).To(Equal(0))

				Expect(appRepo.UpdateAppGuid).To(Equal("my-app-guid"))
				Expect(*appRepo.UpdateParams.InstanceCount).To(Equal(5))
				Expect(appRepo.UpdateParams.DiskQuota).To(BeNil())
				Expect(appRepo.UpdateParams.Memory).To(BeNil())
			})

			It("does not scale the app's instance count if it is not specified", func() {
				testcmd.RunCommand(cmd, []string{"-m", "512M", "my-app"}, requirementsFactory)

				application, orgName, spaceName := restarter.ApplicationRestartArgsForCall(0)
				Expect(application).To(Equal(app))
				Expect(orgName).To(Equal(config.OrganizationFields().Name))
				Expect(spaceName).To(Equal(config.SpaceFields().Name))

				Expect(appRepo.UpdateAppGuid).To(Equal("my-app-guid"))
				Expect(*appRepo.UpdateParams.Memory).To(Equal(int64(512)))
				Expect(appRepo.UpdateParams.DiskQuota).To(BeNil())
				Expect(appRepo.UpdateParams.InstanceCount).To(BeNil())
			})
		})
	})
})
