package app_files_test

import (
	. "github.com/cloudfoundry/cli/cf/app_files"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/cli/cf/models"
	cffileutils "github.com/cloudfoundry/cli/fileutils"
	"github.com/cloudfoundry/gofileutils/fileutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppFiles", func() {
	var appFiles = ApplicationFiles{}
	fixturePath := filepath.Join("..", "..", "fixtures", "applications")

	Describe("AppFilesInDir", func() {
		It("all files have '/' path separators", func() {
			files, err := appFiles.AppFilesInDir(fixturePath)
			Expect(err).ShouldNot(HaveOccurred())

			for _, afile := range files {
				Expect(afile.Path).Should(Equal(filepath.ToSlash(afile.Path)))
			}
		})

		It("excludes files based on the .cfignore file", func() {
			appPath := filepath.Join(fixturePath, "app-with-cfignore")
			files, err := appFiles.AppFilesInDir(appPath)
			Expect(err).ShouldNot(HaveOccurred())

			paths := []string{}
			for _, file := range files {
				paths = append(paths, file.Path)
			}

			Expect(paths).To(Equal([]string{
				"dir1",
				"dir1/child-dir",
				"dir1/child-dir/file3.txt",
				"dir1/file1.txt",
				"dir2",

				// TODO: this should be excluded.
				// .cfignore doesn't handle ** patterns right now
				"dir2/child-dir2",
			}))
		})

		// NB: on windows, you can never rely on the size of a directory being zero
		// see: http://msdn.microsoft.com/en-us/library/windows/desktop/aa364946(v=vs.85).aspx
		// and: https://www.pivotaltracker.com/story/show/70470232
		It("always sets the size of directories to zero bytes", func() {
			fileutils.TempDir("something", func(tempdir string, err error) {
				Expect(err).ToNot(HaveOccurred())

				err = os.Mkdir(filepath.Join(tempdir, "nothing"), 0600)
				Expect(err).ToNot(HaveOccurred())

				files, err := appFiles.AppFilesInDir(tempdir)
				Expect(err).ToNot(HaveOccurred())

				sizes := []int64{}
				for _, file := range files {
					sizes = append(sizes, file.Size)
				}

				Expect(sizes).To(Equal([]int64{0}))
			})
		})
	})

	Describe("CopyFiles", func() {
		It("copies only the files specified", func() {
			copyDir := filepath.Join(fixturePath, "app-copy-test")

			filesToCopy := []models.AppFileFields{
				{Path: filepath.Join("dir1")},
				{Path: filepath.Join("dir1", "child-dir", "file2.txt")},
			}

			files := []string{}

			cffileutils.TempDir("copyToDir", func(tmpDir string, err error) {
				copyErr := appFiles.CopyFiles(filesToCopy, copyDir, tmpDir)
				Expect(copyErr).ToNot(HaveOccurred())

				filepath.Walk(tmpDir, func(path string, fileInfo os.FileInfo, err error) error {
					Expect(err).ToNot(HaveOccurred())

					if !fileInfo.IsDir() {
						files = append(files, fileInfo.Name())
					}
					return nil
				})
			})

			// file2.txt is in lowest subtree, thus is walked first.
			Expect(files).To(Equal([]string{
				"file2.txt",
			}))
		})
	})
})
