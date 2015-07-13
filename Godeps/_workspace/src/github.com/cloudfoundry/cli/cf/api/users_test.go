package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	testapi "github.com/cloudfoundry/cli/cf/api/fakes"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/net"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testnet "github.com/cloudfoundry/cli/testhelpers/net"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/cf/api"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("User Repository", func() {
	var (
		ccServer   *httptest.Server
		ccHandler  *testnet.TestHandler
		uaaServer  *httptest.Server
		uaaHandler *testnet.TestHandler
		repo       UserRepository
		config     core_config.ReadWriter
	)

	BeforeEach(func() {
		config = testconfig.NewRepositoryWithDefaults()
		ccGateway := net.NewCloudControllerGateway((config), time.Now, &testterm.FakeUI{})
		uaaGateway := net.NewUAAGateway(config, &testterm.FakeUI{})
		repo = NewCloudControllerUserRepository(config, uaaGateway, ccGateway)
	})

	AfterEach(func() {
		if uaaServer != nil {
			uaaServer.Close()
		}
		if ccServer != nil {
			ccServer.Close()
		}
	})

	setupCCServer := func(requests ...testnet.TestRequest) {
		ccServer, ccHandler = testnet.NewServer(requests)
		config.SetApiEndpoint(ccServer.URL)
	}

	setupUAAServer := func(requests ...testnet.TestRequest) {
		uaaServer, uaaHandler = testnet.NewServer(requests)
		config.SetUaaEndpoint(uaaServer.URL)
	}

	Describe("listing the users with a given role using ListUsersInOrgForRole()", func() {
		Context("when there are no users in the given org", func() {
			It("lists the users in a org with a given role", func() {
				ccReqs := []testnet.TestRequest{
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method: "GET",
						Path:   "/v2/organizations/my-org-guid/managers",
						Response: testnet.TestResponse{
							Status: http.StatusOK,
							Body:   `{"resources": []}`,
						}}),
				}

				setupCCServer(ccReqs...)

				users, apiErr := repo.ListUsersInOrgForRole("my-org-guid", models.ORG_MANAGER)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())
				Expect(len(users)).To(Equal(0))
			})
		})

		Context("when there are users in the given org", func() {
			It("lists the users in an organization with a given role", func() {
				ccReqs, uaaReqs := createUsersByRoleEndpoints("/v2/organizations/my-org-guid/managers")

				setupCCServer(ccReqs...)
				setupUAAServer(uaaReqs...)

				users, apiErr := repo.ListUsersInOrgForRole("my-org-guid", models.ORG_MANAGER)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(uaaHandler).To(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())

				Expect(len(users)).To(Equal(3))
				Expect(users[0].Guid).To(Equal("user-1-guid"))
				Expect(users[0].Username).To(Equal("Super user 1"))
				Expect(users[1].Guid).To(Equal("user-2-guid"))
				Expect(users[1].Username).To(Equal("Super user 2"))
			})
		})

		Context("when there are no users in the space", func() {
			It("lists the users in a space with a given role", func() {
				ccReqs := []testnet.TestRequest{
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method: "GET",
						Path:   "/v2/spaces/my-space-guid/managers",
						Response: testnet.TestResponse{
							Status: http.StatusOK,
							Body:   `{"resources": []}`,
						}}),
				}

				setupCCServer(ccReqs...)

				users, apiErr := repo.ListUsersInSpaceForRole("my-space-guid", models.SPACE_MANAGER)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())
				Expect(len(users)).To(Equal(0))
			})
		})

		Context("when there are users in the space", func() {
			It("lists the users in a space with a given role", func() {
				ccReqs, uaaReqs := createUsersByRoleEndpoints("/v2/spaces/my-space-guid/managers")

				setupCCServer(ccReqs...)
				setupUAAServer(uaaReqs...)

				users, apiErr := repo.ListUsersInSpaceForRole("my-space-guid", models.SPACE_MANAGER)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(uaaHandler).To(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())

				Expect(len(users)).To(Equal(3))
				Expect(users[0].Guid).To(Equal("user-1-guid"))
				Expect(users[0].Username).To(Equal("Super user 1"))
				Expect(users[1].Guid).To(Equal("user-2-guid"))
				Expect(users[1].Username).To(Equal("Super user 2"))
			})
		})

		It("does not make a request to the UAA when the cloud controller returns an error", func() {
			ccReqs := []testnet.TestRequest{
				testapi.NewCloudControllerTestRequest(testnet.TestRequest{
					Method: "GET",
					Path:   "/v2/organizations/my-org-guid/managers",
					Response: testnet.TestResponse{
						Status: http.StatusGatewayTimeout,
					},
				}),
			}

			setupCCServer(ccReqs...)

			_, apiErr := repo.ListUsersInOrgForRole("my-org-guid", models.ORG_MANAGER)

			Expect(ccHandler).To(HaveAllRequestsCalled())
			httpErr, ok := apiErr.(errors.HttpError)
			Expect(ok).To(BeTrue())
			Expect(httpErr.StatusCode()).To(Equal(http.StatusGatewayTimeout))
		})

		It("returns an error when the UAA endpoint cannot be determined", func() {
			ccReqs, _ := createUsersByRoleEndpoints("/v2/organizations/my-org-guid/managers")

			setupCCServer(ccReqs...)

			config.SetAuthenticationEndpoint("")

			_, apiErr := repo.ListUsersInOrgForRole("my-org-guid", models.ORG_MANAGER)
			Expect(apiErr).To(HaveOccurred())
		})
	})

	Describe("listing the users with a given role using ListUsersInOrgForRoleWithNoUAA()", func() {
		Context("when there are users in the given org", func() {
			It("lists the users in an organization with a given role without hitting UAA endpoint", func() {
				ccReqs, uaaReqs := createUsersByRoleEndpoints("/v2/organizations/my-org-guid/managers")

				setupCCServer(ccReqs...)
				setupUAAServer(uaaReqs...)

				users, apiErr := repo.ListUsersInOrgForRoleWithNoUAA("my-org-guid", models.ORG_MANAGER)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(uaaHandler).ToNot(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())

				Expect(len(users)).To(Equal(3))
				Expect(users[0].Guid).To(Equal("user-1-guid"))
				Expect(users[0].Username).To(Equal(""))
				Expect(users[0].Username).ToNot(Equal("Super user 1"))

				Expect(users[1].Username).To(Equal("user 2 from cc"))
				Expect(users[1].Guid).To(Equal("user-2-guid"))
				Expect(users[1].Username).ToNot(Equal("Super user 2"))

				Expect(users[2].Username).To(Equal("user 3 from cc"))
				Expect(users[2].Guid).To(Equal("user-3-guid"))
				Expect(users[2].Username).ToNot(Equal("Super user 3"))
			})
		})

	})

	Describe("listing the users with a given role using ListUsersInSpaceForRoleWithNoUAA()", func() {
		Context("when there are users in the given space", func() {
			It("lists the users in a space with a given role without hitting UAA endpoint", func() {
				ccReqs, uaaReqs := createUsersByRoleEndpoints("/v2/spaces/my-space-guid/managers")

				setupCCServer(ccReqs...)
				setupUAAServer(uaaReqs...)

				users, apiErr := repo.ListUsersInSpaceForRoleWithNoUAA("my-space-guid", models.SPACE_MANAGER)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(uaaHandler).ToNot(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())

				Expect(len(users)).To(Equal(3))
				Expect(users[0].Guid).To(Equal("user-1-guid"))
				Expect(users[0].Username).To(Equal(""))
				Expect(users[0].Username).ToNot(Equal("Super user 1"))

				Expect(users[1].Username).To(Equal("user 2 from cc"))
				Expect(users[1].Guid).To(Equal("user-2-guid"))
				Expect(users[1].Username).ToNot(Equal("Super user 2"))

				Expect(users[2].Username).To(Equal("user 3 from cc"))
				Expect(users[2].Guid).To(Equal("user-3-guid"))
				Expect(users[2].Username).ToNot(Equal("Super user 3"))
			})
		})

	})

	Describe("FindByUsername", func() {
		Context("when the user exists", func() {
			It("finds the user", func() {
				setupUAAServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method: "GET",
						Path:   "/Users?attributes=id,userName&filter=userName+Eq+%22damien%2Buser1%40pivotallabs.com%22",
						Response: testnet.TestResponse{
							Status: http.StatusOK,
							Body: `
							{
								"resources": [{ "id": "my-guid", "userName": "my-full-username" }]
							}`,
						}}))

				user, err := repo.FindByUsername("damien+user1@pivotallabs.com")
				Expect(uaaHandler).To(HaveAllRequestsCalled())
				Expect(err).NotTo(HaveOccurred())

				Expect(user).To(Equal(models.UserFields{
					Username: "my-full-username",
					Guid:     "my-guid",
				}))
			})
		})

		Context("when the user does not exist", func() {
			It("returns a ModelNotFoundError", func() {
				setupUAAServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "GET",
						Path:     "/Users?attributes=id,userName&filter=userName+Eq+%22my-user%22",
						Response: testnet.TestResponse{Status: http.StatusOK, Body: `{"resources": []}`},
					}))

				_, err := repo.FindByUsername("my-user")
				Expect(uaaHandler).To(HaveAllRequestsCalled())
				Expect(err).To(BeAssignableToTypeOf(&errors.ModelNotFoundError{}))
			})
		})

		Context("when the user does not have permission", func() {
			It("returns a AccessDeniedError", func() {
				setupUAAServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "GET",
						Path:     "/Users?attributes=id,userName&filter=userName+Eq+%22my-user%22",
						Response: testnet.TestResponse{Status: http.StatusForbidden, Body: `{"error":"access_denied","error_description":"Access is denied"}`},
					}))

				_, err := repo.FindByUsername("my-user")
				Expect(uaaHandler).To(HaveAllRequestsCalled())
				Expect(err).To(BeAssignableToTypeOf(&errors.AccessDeniedError{}))

			})
		})

		Context("when the uaa endpoint request returns a non-403 error", func() {
			It("returns the error", func() {
				setupUAAServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "GET",
						Path:     "/Users?attributes=id,userName&filter=userName+Eq+%22my-user%22",
						Response: testnet.TestResponse{Status: 500, Body: `server down!`},
					}))

				_, err := repo.FindByUsername("my-user")
				Expect(uaaHandler).To(HaveAllRequestsCalled())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("creating users", func() {
		It("it creates users using the UAA /Users endpoint", func() {
			setupCCServer(
				testapi.NewCloudControllerTestRequest(testnet.TestRequest{
					Method:   "POST",
					Path:     "/v2/users",
					Matcher:  testnet.RequestBodyMatcher(`{"guid":"my-user-guid"}`),
					Response: testnet.TestResponse{Status: http.StatusCreated},
				}))

			setupUAAServer(
				testapi.NewCloudControllerTestRequest(testnet.TestRequest{
					Method: "POST",
					Path:   "/Users",
					Matcher: testnet.RequestBodyMatcher(`{
					"userName":"my-user",
					"emails":[{"value":"my-user"}],
					"password":"my-password",
					"name":{
						"givenName":"my-user",
						"familyName":"my-user"}
					}`),
					Response: testnet.TestResponse{
						Status: http.StatusCreated,
						Body:   `{"id":"my-user-guid"}`,
					},
				}))

			apiErr := repo.Create("my-user", "my-password")
			Expect(ccHandler).To(HaveAllRequestsCalled())
			Expect(uaaHandler).To(HaveAllRequestsCalled())
			Expect(apiErr).NotTo(HaveOccurred())
		})

		It("warns the user if the requested new user already exists", func() {
			setupUAAServer(
				testapi.NewCloudControllerTestRequest(testnet.TestRequest{
					Method: "POST",
					Path:   "/Users",
					Response: testnet.TestResponse{
						Status: http.StatusConflict,
						Body: `
						{
							"message":"Username already in use: my-user",
							"error":"scim_resource_already_exists"
						}`,
					},
				}))

			err := repo.Create("my-user", "my-password")
			Expect(err).To(BeAssignableToTypeOf(&errors.ModelAlreadyExistsError{}))
		})
		It("Returns any http error", func() {
			setupUAAServer(
				testapi.NewCloudControllerTestRequest(testnet.TestRequest{
					Method: "POST",
					Path:   "/Users",
					Response: testnet.TestResponse{
						Status: http.StatusForbidden,
						Body: `
						{
							"message":"Access Denied",
							"error":"Forbidden"
						}`,
					},
				}))

			err := repo.Create("my-user", "my-password")
			Expect(err.Error()).To(ContainSubstring("Forbidden"))
		})
	})

	Describe("deleting users", func() {
		It("deletes the user", func() {
			setupCCServer(
				testapi.NewCloudControllerTestRequest(testnet.TestRequest{
					Method:   "DELETE",
					Path:     "/v2/users/my-user-guid",
					Response: testnet.TestResponse{Status: http.StatusOK},
				}))

			setupUAAServer(
				testapi.NewCloudControllerTestRequest(testnet.TestRequest{
					Method:   "DELETE",
					Path:     "/Users/my-user-guid",
					Response: testnet.TestResponse{Status: http.StatusOK},
				}))

			apiErr := repo.Delete("my-user-guid")
			Expect(ccHandler).To(HaveAllRequestsCalled())
			Expect(uaaHandler).To(HaveAllRequestsCalled())
			Expect(apiErr).NotTo(HaveOccurred())
		})

		Context("when the user is not found on the cloud controller", func() {
			It("when the user is not found on the cloud controller", func() {
				setupCCServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method: "DELETE",
						Path:   "/v2/users/my-user-guid",
						Response: testnet.TestResponse{Status: http.StatusNotFound, Body: `
						{
							"code": 20003,
							"description": "The user could not be found"
						}`},
					}))

				setupUAAServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "DELETE",
						Path:     "/Users/my-user-guid",
						Response: testnet.TestResponse{Status: http.StatusOK},
					}))

				err := repo.Delete("my-user-guid")
				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(uaaHandler).To(HaveAllRequestsCalled())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("assigning users organization roles", func() {
		orgRoleURLS := map[string]string{
			"OrgManager":     "/v2/organizations/my-org-guid/managers/my-user-guid",
			"BillingManager": "/v2/organizations/my-org-guid/billing_managers/my-user-guid",
			"OrgAuditor":     "/v2/organizations/my-org-guid/auditors/my-user-guid",
		}

		for role, roleURL := range orgRoleURLS {
			It("gives users the "+role+" role", func() {
				setupCCServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "PUT",
						Path:     roleURL,
						Response: testnet.TestResponse{Status: http.StatusOK},
					}),

					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "PUT",
						Path:     "/v2/organizations/my-org-guid/users/my-user-guid",
						Response: testnet.TestResponse{Status: http.StatusOK},
					}))

				err := repo.SetOrgRole("my-user-guid", "my-org-guid", role)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(err).NotTo(HaveOccurred())
			})

			It("unsets the org role from user", func() {
				setupCCServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "DELETE",
						Path:     roleURL,
						Response: testnet.TestResponse{Status: http.StatusOK},
					}))

				apiErr := repo.UnsetOrgRole("my-user-guid", "my-org-guid", role)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())
			})
		}

		It("returns an error when given an invalid role to set", func() {
			err := repo.SetOrgRole("user-guid", "org-guid", "foo")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid Role"))
		})

		It("returns an error when given an invalid role to unset", func() {
			err := repo.UnsetOrgRole("user-guid", "org-guid", "foo")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid Role"))
		})
	})

	Describe("assigning space roles", func() {
		spaceRoleURLS := map[string]string{
			"SpaceManager":   "/v2/spaces/my-space-guid/managers/my-user-guid",
			"SpaceDeveloper": "/v2/spaces/my-space-guid/developers/my-user-guid",
			"SpaceAuditor":   "/v2/spaces/my-space-guid/auditors/my-user-guid",
		}

		for role, roleURL := range spaceRoleURLS {
			It("gives the user the "+role+" role", func() {
				setupCCServer(
					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "PUT",
						Path:     "/v2/organizations/my-org-guid/users/my-user-guid",
						Response: testnet.TestResponse{Status: http.StatusOK},
					}),

					testapi.NewCloudControllerTestRequest(testnet.TestRequest{
						Method:   "PUT",
						Path:     roleURL,
						Response: testnet.TestResponse{Status: http.StatusOK},
					}))

				err := repo.SetSpaceRole("my-user-guid", "my-space-guid", "my-org-guid", role)

				Expect(ccHandler).To(HaveAllRequestsCalled())
				Expect(err).NotTo(HaveOccurred())
			})
		}

		It("returns an error when given an invalid role to set", func() {
			err := repo.SetSpaceRole("user-guid", "space-guid", "org-guid", "foo")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid Role"))
		})
	})

	It("lists all users in the org", func() {
		ccReqs, uaaReqs := createUsersByRoleEndpoints("/v2/organizations/my-org-guid/users")

		setupCCServer(ccReqs...)
		setupUAAServer(uaaReqs...)

		users, err := repo.ListUsersInOrgForRole("my-org-guid", models.ORG_USER)

		Expect(ccHandler).To(HaveAllRequestsCalled())
		Expect(uaaHandler).To(HaveAllRequestsCalled())
		Expect(err).NotTo(HaveOccurred())

		Expect(len(users)).To(Equal(3))
		Expect(users[0].Guid).To(Equal("user-1-guid"))
		Expect(users[0].Username).To(Equal("Super user 1"))
		Expect(users[1].Guid).To(Equal("user-2-guid"))
		Expect(users[1].Username).To(Equal("Super user 2"))
		Expect(users[2].Guid).To(Equal("user-3-guid"))
		Expect(users[2].Username).To(Equal("Super user 3"))
	})
})

func createUsersByRoleEndpoints(rolePath string) (ccReqs []testnet.TestRequest, uaaReqs []testnet.TestRequest) {
	nextUrl := rolePath + "?page=2"

	ccReqs = []testnet.TestRequest{
		testapi.NewCloudControllerTestRequest(testnet.TestRequest{
			Method: "GET",
			Path:   rolePath,
			Response: testnet.TestResponse{
				Status: http.StatusOK,
				Body: fmt.Sprintf(`
				{
					"next_url": "%s",
					"resources": [
						{"metadata": {"guid": "user-1-guid"}, "entity": {}}
					]
				}`, nextUrl)}}),

		testapi.NewCloudControllerTestRequest(testnet.TestRequest{
			Method: "GET",
			Path:   nextUrl,
			Response: testnet.TestResponse{
				Status: http.StatusOK,
				Body: `
				{
					"resources": [
					{"metadata": {"guid": "user-2-guid"}, "entity": {"username":"user 2 from cc"}},
					{"metadata": {"guid": "user-3-guid"}, "entity": {"username":"user 3 from cc"}}
					]
				}`}}),
	}

	uaaReqs = []testnet.TestRequest{
		testapi.NewCloudControllerTestRequest(testnet.TestRequest{
			Method: "GET",
			Path: fmt.Sprintf(
				"/Users?attributes=id,userName&filter=%s",
				url.QueryEscape(`Id eq "user-1-guid" or Id eq "user-2-guid" or Id eq "user-3-guid"`)),
			Response: testnet.TestResponse{
				Status: http.StatusOK,
				Body: `
				{
					"resources": [
						{ "id": "user-1-guid", "userName": "Super user 1" },
						{ "id": "user-2-guid", "userName": "Super user 2" },
  						{ "id": "user-3-guid", "userName": "Super user 3" }
					]
				}`}})}

	return
}
