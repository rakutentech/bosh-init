package templatescompiler_test

import (
	. "github.com/cloudfoundry/bosh-init/templatescompiler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"os"
	"path/filepath"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshcmd "github.com/cloudfoundry/bosh-agent/platform/commands"
	boshsys "github.com/cloudfoundry/bosh-agent/system"

	fakeboshcmd "github.com/cloudfoundry/bosh-agent/platform/commands/fakes"
	fakeboshsys "github.com/cloudfoundry/bosh-agent/system/fakes"

	bireljob "github.com/cloudfoundry/bosh-init/release/job"

	fakebicrypto "github.com/cloudfoundry/bosh-init/crypto/fakes"
)

var _ = Describe("RenderedJobListCompressor", func() {
	var (
		outBuffer *bytes.Buffer
		errBuffer *bytes.Buffer
		logger    boshlog.Logger

		fakeSHA1Calculator *fakebicrypto.FakeSha1Calculator

		renderedJobList RenderedJobList

		renderedJobListCompressor RenderedJobListCompressor
	)

	BeforeEach(func() {
		outBuffer = bytes.NewBufferString("")
		errBuffer = bytes.NewBufferString("")
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, outBuffer, errBuffer)

		fakeSHA1Calculator = fakebicrypto.NewFakeSha1Calculator()

		renderedJobList = NewRenderedJobList()
	})

	Describe("Compress", func() {

		Context("with a real fs & compressor", func() {
			var (
				fs         boshsys.FileSystem
				cmdRunner  boshsys.CmdRunner
				compressor boshcmd.Compressor
			)

			BeforeEach(func() {
				fs = boshsys.NewOsFileSystem(logger)

				cmdRunner = boshsys.NewExecCmdRunner(logger)
				compressor = boshcmd.NewTarballCompressor(cmdRunner, fs)

				renderedJobListCompressor = NewRenderedJobListCompressor(fs, compressor, fakeSHA1Calculator, logger)
			})

			It("copies rendered jobs into a new temp dir, compresses the temp dir, and wraps it in a RenderedJobListArchive", func() {
				// create rendered job with 2 rendered scripts
				renderedJobDir0, err := fs.TempDir("RenderedJobListCompressorTest")
				Expect(err).ToNot(HaveOccurred())
				renderedJob0 := NewRenderedJob(bireljob.Job{Name: "fake-job-name-0"}, renderedJobDir0, fs, logger)
				defer func() { err := renderedJob0.Delete(); Expect(err).ToNot(HaveOccurred()) }()
				err = fs.WriteFileString(filepath.Join(renderedJobDir0, "script-0"), "fake-rendered-job-0-script-0-content")
				Expect(err).ToNot(HaveOccurred())
				err = fs.WriteFileString(filepath.Join(renderedJobDir0, "script-1"), "fake-rendered-job-0-script-1-content")
				Expect(err).ToNot(HaveOccurred())
				renderedJobList.Add(renderedJob0)

				// create another rendered job with 1 rendered script
				renderedJobDir1, err := fs.TempDir("RenderedJobListCompressorTest")
				Expect(err).ToNot(HaveOccurred())
				renderedJob1 := NewRenderedJob(bireljob.Job{Name: "fake-job-name-1"}, renderedJobDir1, fs, logger)
				defer func() { err := renderedJob1.Delete(); Expect(err).ToNot(HaveOccurred()) }()
				err = fs.WriteFileString(filepath.Join(renderedJobDir1, "script-0"), "fake-rendered-job-1-script-0-content")
				Expect(err).ToNot(HaveOccurred())
				renderedJobList.Add(renderedJob1)

				// compress
				archive, err := renderedJobListCompressor.Compress(renderedJobList)
				Expect(err).ToNot(HaveOccurred())
				defer func() { err := archive.Delete(); Expect(err).ToNot(HaveOccurred()) }()

				// decompress
				renderedJobListDir, err := fs.TempDir("RenderedJobListCompressorTest")
				Expect(err).ToNot(HaveOccurred())
				defer func() { err := fs.RemoveAll(renderedJobListDir); Expect(err).ToNot(HaveOccurred()) }()
				err = compressor.DecompressFileToDir(archive.Path(), renderedJobListDir, boshcmd.CompressorOptions{})
				Expect(err).ToNot(HaveOccurred())

				// verify that archive contained rendered scripts from job 0
				content, err := fs.ReadFileString(filepath.Join(renderedJobListDir, "fake-job-name-0", "script-0"))
				Expect(err).ToNot(HaveOccurred())
				Expect(content).To(Equal("fake-rendered-job-0-script-0-content"))
				content, err = fs.ReadFileString(filepath.Join(renderedJobListDir, "fake-job-name-0", "script-1"))
				Expect(err).ToNot(HaveOccurred())
				Expect(content).To(Equal("fake-rendered-job-0-script-1-content"))

				// verify that archive contained rendered scripts from job 1
				content, err = fs.ReadFileString(filepath.Join(renderedJobListDir, "fake-job-name-1", "script-0"))
				Expect(err).ToNot(HaveOccurred())
				Expect(content).To(Equal("fake-rendered-job-1-script-0-content"))
			})
		})

		Context("with a fake fs & compressor", func() {
			var (
				fakeFS         *fakeboshsys.FakeFileSystem
				fakeCompressor *fakeboshcmd.FakeCompressor
			)

			BeforeEach(func() {
				fakeFS = fakeboshsys.NewFakeFileSystem()

				fakeCompressor = fakeboshcmd.NewFakeCompressor()

				renderedJobListCompressor = NewRenderedJobListCompressor(fakeFS, fakeCompressor, fakeSHA1Calculator, logger)
			})

			It("calculates the fingerprint of the rendered", func() {
				fakeFS.TempDirDir = "fake-rendered-job-list-path"

				fakeSHA1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
					"fake-rendered-job-list-path": fakebicrypto.CalculateInput{Sha1: "fake-sha1"},
				})

				archive, err := renderedJobListCompressor.Compress(renderedJobList)
				Expect(err).ToNot(HaveOccurred())

				Expect(archive.Fingerprint()).To(Equal("fake-sha1"))
			})

			It("calculates the SHA1 of the archive", func() {
				fakeCompressor.CompressFilesInDirTarballPath = "fake-archive-path"

				fakeSHA1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
					"fake-archive-path": fakebicrypto.CalculateInput{Sha1: "fake-sha1"},
				})

				archive, err := renderedJobListCompressor.Compress(renderedJobList)
				Expect(err).ToNot(HaveOccurred())

				Expect(archive.SHA1()).To(Equal("fake-sha1"))
			})

			It("deletes the temp dir compressed into the archive", func() {
				fakeFS.TempDirDir = "fake-rendered-job-list-path"
				err := fakeFS.MkdirAll("fake-rendered-job-list-path", os.ModePerm)
				Expect(err).ToNot(HaveOccurred())

				_, err = renderedJobListCompressor.Compress(renderedJobList)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFS.FileExists("fake-rendered-job-list-path")).To(BeFalse())
			})
		})
	})
})
