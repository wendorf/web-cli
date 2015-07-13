package application_test

import (
	testApplication "github.com/cloudfoundry/cli/cf/api/applications/fakes"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/models"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/cf/commands/application"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rename command", func() {
	var (
		ui                  *testterm.FakeUI
		requirementsFactory *testreq.FakeReqFactory
		configRepo          core_config.ReadWriter
		appRepo             *testApplication.FakeApplicationRepository
	)

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		configRepo = testconfig.NewRepositoryWithDefaults()
		requirementsFactory = &testreq.FakeReqFactory{}
		appRepo = &testApplication.FakeApplicationRepository{}
	})

	runCommand := func(args ...string) bool {
		return testcmd.RunCommand(NewRenameApp(ui, configRepo, appRepo), args, requirementsFactory)
	}

	Describe("requirements", func() {
		It("fails with usage when not invoked with an old name and a new name", func() {
			requirementsFactory.LoginSuccess = true
			runCommand("foo")
			Expect(ui.FailedWithUsage).To(BeTrue())
		})

		It("fails when not logged in", func() {
			Expect(runCommand("my-app", "my-new-app")).To(BeFalse())
		})
		It("fails if a space is not targeted", func() {
			requirementsFactory.LoginSuccess = true
			requirementsFactory.TargetedSpaceSuccess = false
			Expect(runCommand("my-app", "my-new-app")).To(BeFalse())
		})
	})

	It("renames an application", func() {
		app := models.Application{}
		app.Name = "my-app"
		app.Guid = "my-app-guid"
		requirementsFactory.LoginSuccess = true
		requirementsFactory.TargetedSpaceSuccess = true
		requirementsFactory.Application = app
		runCommand("my-app", "my-new-app")

		Expect(appRepo.UpdateAppGuid).To(Equal(app.Guid))
		Expect(*appRepo.UpdateParams.Name).To(Equal("my-new-app"))
		Expect(ui.Outputs).To(ContainSubstrings(
			[]string{"Renaming app", "my-app", "my-new-app", "my-org", "my-space", "my-user"},
			[]string{"OK"},
		))
	})
})
