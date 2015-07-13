package commands_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	testapi "github.com/cloudfoundry/cli/cf/api/fakes"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/trace"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"
	"github.com/cloudfoundry/gofileutils/fileutils"

	. "github.com/cloudfoundry/cli/cf/commands"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("curl command", func() {
	var deps curlDependencies

	BeforeEach(func() {
		deps = newCurlDependencies()
	})

	It("does not pass requirements when not logged in", func() {
		Expect(runCurlWithInputs(deps, []string{"/foo"})).To(BeFalse())
	})

	Context("when logged in", func() {
		BeforeEach(func() {
			deps.requirementsFactory.LoginSuccess = true
		})

		It("fails with usage when not given enough input", func() {
			runCurlWithInputs(deps, []string{})
			Expect(deps.ui.FailedWithUsage).To(BeTrue())
		})

		It("passes requirements", func() {
			Expect(runCurlWithInputs(deps, []string{"/foo"})).To(BeTrue())
		})

		It("makes a get request given an endpoint", func() {
			deps.curlRepo.ResponseHeader = "Content-Size:1024"
			deps.curlRepo.ResponseBody = "response for get"
			runCurlWithInputs(deps, []string{"/foo"})

			Expect(deps.curlRepo.Method).To(Equal("GET"))
			Expect(deps.curlRepo.Path).To(Equal("/foo"))
			Expect(deps.ui.Outputs).To(ContainSubstrings([]string{"response for get"}))
			Expect(deps.ui.Outputs).ToNot(ContainSubstrings(
				[]string{"FAILED"},
				[]string{"Content-Size:1024"},
			))
		})

		Context("when the --output flag is provided", func() {
			It("saves the body of the response to the given filepath if it exists", func() {
				fileutils.TempFile("poor-mans-pipe", func(tempFile *os.File, err error) {
					Expect(err).ToNot(HaveOccurred())
					deps.curlRepo.ResponseBody = "hai"

					runCurlWithInputs(deps, []string{"--output", tempFile.Name(), "/foo"})
					contents, err := ioutil.ReadAll(tempFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(Equal("hai"))
				})
			})

			It("saves the body of the response to the given filepath if it doesn't exists", func() {
				fileutils.TempDir("poor-mans-dir", func(tmpDir string, err error) {
					Expect(err).ToNot(HaveOccurred())
					deps.curlRepo.ResponseBody = "hai"

					filePath := filepath.Join(tmpDir, "subdir1", "banana.txt")
					runCurlWithInputs(deps, []string{"--output", filePath, "/foo"})

					file, err := os.Open(filePath)
					Expect(err).ToNot(HaveOccurred())

					contents, err := ioutil.ReadAll(file)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(Equal("hai"))
				})
			})
		})

		It("makes a post request given -X", func() {
			runCurlWithInputs(deps, []string{"-X", "post", "/foo"})

			Expect(deps.curlRepo.Method).To(Equal("post"))
			Expect(deps.ui.Outputs).ToNot(ContainSubstrings([]string{"FAILED"}))
		})

		It("sends headers given -H", func() {
			runCurlWithInputs(deps, []string{"-H", "Content-Type:cat", "/foo"})

			Expect(deps.curlRepo.Header).To(Equal("Content-Type:cat"))
			Expect(deps.ui.Outputs).ToNot(ContainSubstrings([]string{"FAILED"}))
		})

		It("sends multiple headers given multiple -H flags", func() {
			runCurlWithInputs(deps, []string{"-H", "Content-Type:cat", "-H", "Content-Length:12", "/foo"})

			Expect(deps.curlRepo.Header).To(Equal("Content-Type:cat\nContent-Length:12"))
			Expect(deps.ui.Outputs).ToNot(ContainSubstrings([]string{"FAILED"}))
		})

		It("prints out the response headers given -i", func() {
			deps.curlRepo.ResponseHeader = "Content-Size:1024"
			deps.curlRepo.ResponseBody = "response for get"
			runCurlWithInputs(deps, []string{"-i", "/foo"})

			Expect(deps.ui.Outputs).To(ContainSubstrings(
				[]string{"Content-Size:1024"},
				[]string{"response for get"},
			))
			Expect(deps.ui.Outputs).ToNot(ContainSubstrings([]string{"FAILED"}))
		})

		It("sets the request body given -d", func() {
			runCurlWithInputs(deps, []string{"-d", "body content to upload", "/foo"})

			Expect(deps.curlRepo.Body).To(Equal("body content to upload"))
			Expect(deps.ui.Outputs).ToNot(ContainSubstrings([]string{"FAILED"}))
		})

		It("prints verbose output given the -v flag", func() {
			output := bytes.NewBuffer(make([]byte, 1024))
			trace.SetStdout(output)

			runCurlWithInputs(deps, []string{"-v", "/foo"})
			trace.Logger.Print("logging enabled")

			Expect([]string{output.String()}).To(ContainSubstrings([]string{"logging enabled"}))
		})

		It("prints a failure message when the response is not success", func() {
			deps.curlRepo.Error = errors.New("ooops")
			runCurlWithInputs(deps, []string{"/foo"})

			Expect(deps.ui.Outputs).To(ContainSubstrings(
				[]string{"FAILED"},
				[]string{"ooops"},
			))
		})

		Context("Whent the content type is JSON", func() {
			BeforeEach(func() {
				deps.curlRepo.ResponseHeader = "Content-Type: application/json;charset=utf-8"
				deps.curlRepo.ResponseBody = `{"total_results":0,"total_pages":1,"prev_url":null,"next_url":null,"resources":[]}`
			})

			It("pretty-prints the response body", func() {
				runCurlWithInputs(deps, []string{"/ugly-printed-json-endpoint"})

				Expect(deps.ui.Outputs).To(ContainSubstrings(
					[]string{"{"},
					[]string{"  \"total_results", "0"},
					[]string{"  \"total_pages", "1"},
					[]string{"  \"prev_url", "null"},
					[]string{"  \"next_url", "null"},
					[]string{"  \"resources", "[]"},
					[]string{"}"},
				))
			})

			Context("But the body is not JSON", func() {
				BeforeEach(func() {
					deps.curlRepo.ResponseBody = "FAIL: crumpets need MOAR butterz"
				})

				It("regular-prints the response body", func() {
					runCurlWithInputs(deps, []string{"/whateverz"})

					Expect(deps.ui.Outputs).To(Equal([]string{"FAIL: crumpets need MOAR butterz"}))
				})
			})
		})
	})
})

type curlDependencies struct {
	ui                  *testterm.FakeUI
	config              core_config.Reader
	requirementsFactory *testreq.FakeReqFactory
	curlRepo            *testapi.FakeCurlRepository
}

func newCurlDependencies() (deps curlDependencies) {
	deps.ui = &testterm.FakeUI{}
	deps.config = testconfig.NewRepository()
	deps.requirementsFactory = &testreq.FakeReqFactory{}
	deps.curlRepo = &testapi.FakeCurlRepository{}
	return
}

func runCurlWithInputs(deps curlDependencies, args []string) bool {
	cmd := NewCurl(deps.ui, deps.config, deps.curlRepo)
	return testcmd.RunCommand(cmd, args, deps.requirementsFactory)
}
