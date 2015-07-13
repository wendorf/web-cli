package service_test

import (
	testapi "github.com/cloudfoundry/cli/cf/actors/service_builder/fakes"
	. "github.com/cloudfoundry/cli/cf/commands/service"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/models"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/cli/testhelpers/matchers"
)

var _ = Describe("marketplace command", func() {
	var ui *testterm.FakeUI
	var requirementsFactory *testreq.FakeReqFactory
	var config core_config.ReadWriter
	var serviceBuilder *testapi.FakeServiceBuilder
	var fakeServiceOfferings []models.ServiceOffering
	var serviceWithAPaidPlan models.ServiceOffering
	var service2 models.ServiceOffering

	BeforeEach(func() {
		serviceBuilder = &testapi.FakeServiceBuilder{}
		ui = &testterm.FakeUI{}
		requirementsFactory = &testreq.FakeReqFactory{ApiEndpointSuccess: true}

		serviceWithAPaidPlan = models.ServiceOffering{
			Plans: []models.ServicePlanFields{
				models.ServicePlanFields{Name: "service-plan-a", Description: "service-plan-a description", Free: true},
				models.ServicePlanFields{Name: "service-plan-b", Description: "service-plan-b description", Free: false},
			},
			ServiceOfferingFields: models.ServiceOfferingFields{
				Label:       "zzz-my-service-offering",
				Guid:        "service-1-guid",
				Description: "service offering 1 description",
			}}
		service2 = models.ServiceOffering{
			Plans: []models.ServicePlanFields{
				models.ServicePlanFields{Name: "service-plan-c", Free: true},
				models.ServicePlanFields{Name: "service-plan-d", Free: true}},
			ServiceOfferingFields: models.ServiceOfferingFields{
				Label:       "aaa-my-service-offering",
				Description: "service offering 2 description",
			},
		}
		fakeServiceOfferings = []models.ServiceOffering{serviceWithAPaidPlan, service2}
	})

	Describe("Requirements", func() {
		Context("when the an API endpoint is not targeted", func() {
			It("does not meet its requirements", func() {
				config := testconfig.NewRepository()
				cmd := NewMarketplaceServices(ui, config, serviceBuilder)
				requirementsFactory.ApiEndpointSuccess = false

				Expect(testcmd.RunCommand(cmd, []string{}, requirementsFactory)).To(BeFalse())
			})
			It("should fail with usage when provided any arguments", func() {
				config := testconfig.NewRepository()
				cmd := NewMarketplaceServices(ui, config, serviceBuilder)
				requirementsFactory.ApiEndpointSuccess = true
				Expect(testcmd.RunCommand(cmd, []string{"blahblah"}, requirementsFactory)).To(BeFalse())
				Expect(ui.FailedWithUsage).To(BeTrue())
			})
		})
	})

	Context("when the user is logged in", func() {
		BeforeEach(func() {
			config = testconfig.NewRepositoryWithDefaults()
		})

		Context("when the user has a space targeted", func() {
			BeforeEach(func() {
				config.SetSpaceFields(models.SpaceFields{
					Guid: "the-space-guid",
					Name: "the-space-name",
				})
				serviceBuilder.GetServicesForSpaceWithPlansReturns(fakeServiceOfferings, nil)
			})

			It("lists all of the service offerings for the space", func() {
				cmd := NewMarketplaceServices(ui, config, serviceBuilder)
				testcmd.RunCommand(cmd, []string{}, requirementsFactory)

				args := serviceBuilder.GetServicesForSpaceWithPlansArgsForCall(0)
				Expect(args).To(Equal("the-space-guid"))

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Getting services from marketplace in org", "my-org", "the-space-name", "my-user"},
					[]string{"OK"},
					[]string{"service", "plans", "description"},
					[]string{"aaa-my-service-offering", "service offering 2 description", "service-plan-c,", "service-plan-d"},
					[]string{"zzz-my-service-offering", "service offering 1 description", "service-plan-a,", "service-plan-b*"},
					[]string{"* These service plans have an associated cost. Creating a service instance will incur this cost."},
					[]string{"TIP:  Use 'cf marketplace -s SERVICE' to view descriptions of individual plans of a given service."},
				))
			})

			Context("when there are no paid plans", func() {
				BeforeEach(func() {
					serviceBuilder.GetServicesForSpaceWithPlansReturns([]models.ServiceOffering{service2}, nil)
				})

				It("lists the service offerings without displaying the paid message", func() {
					cmd := NewMarketplaceServices(ui, config, serviceBuilder)
					testcmd.RunCommand(cmd, []string{}, requirementsFactory)

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Getting services from marketplace in org", "my-org", "the-space-name", "my-user"},
						[]string{"OK"},
						[]string{"service", "plans", "description"},
						[]string{"aaa-my-service-offering", "service offering 2 description", "service-plan-c", "service-plan-d"},
					))
					Expect(ui.Outputs).NotTo(ContainSubstrings(
						[]string{"* The denoted service plans have specific costs associated with them. If a service instance of this type is created, a cost will be incurred."},
					))
				})

			})

			Context("when the user passes the -s flag", func() {
				It("Displays the list of plans for each service with info", func() {
					serviceBuilder.GetServiceByNameForSpaceWithPlansReturns(serviceWithAPaidPlan, nil)

					cmd := NewMarketplaceServices(ui, config, serviceBuilder)
					testcmd.RunCommand(cmd, []string{"-s", "aaa-my-service-offering"}, requirementsFactory)

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Getting service plan information for service aaa-my-service-offering as my-user..."},
						[]string{"OK"},
						[]string{"service plan", "description", "free or paid"},
						[]string{"service-plan-a", "service-plan-a description", "free"},
						[]string{"service-plan-b", "service-plan-b description", "paid"},
					))
				})

				It("informs the user if the service cannot be found", func() {
					cmd := NewMarketplaceServices(ui, config, serviceBuilder)
					testcmd.RunCommand(cmd, []string{"-s", "aaa-my-service-offering"}, requirementsFactory)

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Service offering not found"},
					))
					Expect(ui.Outputs).ToNot(ContainSubstrings(
						[]string{"service plan", "description", "free or paid"},
					))
				})
			})
		})

		Context("when the user doesn't have a space targeted", func() {
			BeforeEach(func() {
				config.SetSpaceFields(models.SpaceFields{})
			})

			It("tells the user to target a space", func() {
				cmd := NewMarketplaceServices(ui, config, serviceBuilder)
				testcmd.RunCommand(cmd, []string{}, requirementsFactory)
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"without", "space"},
				))
			})
		})
	})

	Context("when user is not logged in", func() {
		BeforeEach(func() {
			config = testconfig.NewRepository()
		})

		It("lists all public service offerings if any are available", func() {
			serviceBuilder := &testapi.FakeServiceBuilder{}
			serviceBuilder.GetAllServicesWithPlansReturns(fakeServiceOfferings, nil)

			cmd := NewMarketplaceServices(ui, config, serviceBuilder)
			testcmd.RunCommand(cmd, []string{}, requirementsFactory)

			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Getting all services from marketplace"},
				[]string{"OK"},
				[]string{"service", "plans", "description"},
				[]string{"aaa-my-service-offering", "service offering 2 description", "service-plan-c", "service-plan-d"},
				[]string{"zzz-my-service-offering", "service offering 1 description", "service-plan-a", "service-plan-b"},
			))
		})

		It("does not display a table if no service offerings exist", func() {
			serviceBuilder := &testapi.FakeServiceBuilder{}
			serviceBuilder.GetAllServicesWithPlansReturns([]models.ServiceOffering{}, nil)

			cmd := NewMarketplaceServices(ui, config, serviceBuilder)
			testcmd.RunCommand(cmd, []string{}, requirementsFactory)

			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"No service offerings found"},
			))
			Expect(ui.Outputs).ToNot(ContainSubstrings(
				[]string{"service", "plans", "description"},
			))
		})

		Context("when the user passes the -s flag", func() {
			It("Displays the list of plans for each service with info", func() {
				serviceBuilder.GetServiceByNameWithPlansReturns(serviceWithAPaidPlan, nil)
				cmd := NewMarketplaceServices(ui, config, serviceBuilder)
				testcmd.RunCommand(cmd, []string{"-s", "aaa-my-service-offering"}, requirementsFactory)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Getting service plan information for service aaa-my-service-offering"},
					[]string{"OK"},
					[]string{"service plan", "description", "free or paid"},
					[]string{"service-plan-a", "service-plan-a description", "free"},
					[]string{"service-plan-b", "service-plan-b description", "paid"},
				))
			})

			It("informs the user if the service cannot be found", func() {
				cmd := NewMarketplaceServices(ui, config, serviceBuilder)
				testcmd.RunCommand(cmd, []string{"-s", "aaa-my-service-offering"}, requirementsFactory)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Service offering not found"},
				))
				Expect(ui.Outputs).ToNot(ContainSubstrings(
					[]string{"service plan", "description", "free or paid"},
				))
			})
		})
	})
})
