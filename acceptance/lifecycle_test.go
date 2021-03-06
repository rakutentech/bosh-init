package acceptance_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/acceptance"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshsys "github.com/cloudfoundry/bosh-agent/system"

	bitestutils "github.com/cloudfoundry/bosh-init/testutils"
)

const (
	stageTimePattern     = "\\(\\d{2}:\\d{2}:\\d{2}\\)"
	stageFinishedPattern = "\\.\\.\\. Finished " + stageTimePattern + "$"
)

var _ = Describe("bosh-init", func() {
	var (
		logger       boshlog.Logger
		fileSystem   boshsys.FileSystem
		sshCmdRunner CmdRunner
		cmdEnv       map[string]string
		quietCmdEnv  map[string]string
		testEnv      Environment
		config       *Config

		instanceSSH      InstanceSSH
		instanceUsername = "vcap"
		instancePassword = "sshpassword" // encrypted value must be in the manifest: resource_pool.env.bosh.password
		instanceIP       = "10.244.0.42"
	)

	var readLogFile = func(logPath string) (stdout string) {
		stdout, _, exitCode, err := sshCmdRunner.RunCommand(cmdEnv, "cat", logPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
		return stdout
	}

	var deleteLogFile = func(logPath string) {
		_, _, exitCode, err := sshCmdRunner.RunCommand(cmdEnv, "rm", "-f", logPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
	}

	var flushLog = func(logPath string) {
		logString := readLogFile(logPath)
		_, err := GinkgoWriter.Write([]byte(logString))
		Expect(err).ToNot(HaveOccurred())

		// only delete after successfully writing to GinkgoWriter
		deleteLogFile(logPath)
	}

	// updateDeploymentManifest copies a source manifest from assets to <workspace>/manifest
	var updateDeploymentManifest = func(sourceManifestPath string) {
		manifestContents, err := ioutil.ReadFile(sourceManifestPath)
		Expect(err).ToNot(HaveOccurred())
		testEnv.WriteContent("test-manifest.yml", manifestContents)
	}

	var deploy = func() (stdout string) {
		os.Stdout.WriteString("\n---DEPLOY---\n")
		outBuffer := bytes.NewBufferString("")
		multiWriter := NewMultiWriter(outBuffer, os.Stdout)
		_, _, exitCode, err := sshCmdRunner.RunStreamingCommand(multiWriter, cmdEnv, testEnv.Path("bosh-init"), "deploy", testEnv.Path("test-manifest.yml"), testEnv.Path("stemcell.tgz"), testEnv.Path("cpi-release.tgz"), testEnv.Path("dummy-release.tgz"))
		println((string)(outBuffer.Bytes()))
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
		return outBuffer.String()
	}

	var expectDeployToError = func() (stdout string) {
		os.Stdout.WriteString("\n---DEPLOY---\n")
		outBuffer := bytes.NewBufferString("")
		multiWriter := NewMultiWriter(outBuffer, os.Stdout)
		_, _, exitCode, err := sshCmdRunner.RunStreamingCommand(multiWriter, cmdEnv, testEnv.Path("bosh-init"), "deploy", testEnv.Path("test-manifest.yml"), testEnv.Path("stemcell.tgz"), testEnv.Path("cpi-release.tgz"), testEnv.Path("dummy-release.tgz"))
		Expect(err).To(HaveOccurred())
		Expect(exitCode).To(Equal(1))
		return outBuffer.String()
	}

	var deleteDeployment = func() (stdout string) {
		os.Stdout.WriteString("\n---DELETE---\n")
		outBuffer := bytes.NewBufferString("")
		multiWriter := NewMultiWriter(outBuffer, os.Stdout)
		_, _, exitCode, err := sshCmdRunner.RunStreamingCommand(multiWriter, cmdEnv, testEnv.Path("bosh-init"), "delete", testEnv.Path("test-manifest.yml"), testEnv.Path("cpi-release.tgz"))
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
		return outBuffer.String()
	}

	var shutdownAgent = func() {
		_, _, exitCode, err := instanceSSH.RunCommandWithSudo("sv -w 14 force-shutdown agent")
		if exitCode == 1 {
			// If timeout was reached, KILL signal was sent before exiting.
			// Retry to wait another 14s for exit.
			_, _, exitCode, err = instanceSSH.RunCommandWithSudo("sv -w 14 force-shutdown agent")
		}
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
	}

	var findStage = func(outputLines []string, stageName string, zeroIndex int) (steps []string, stopIndex int) {
		startLine := fmt.Sprintf("Started %s", stageName)
		startIndex := -1
		for i, line := range outputLines[zeroIndex:] {
			if line == startLine {
				startIndex = zeroIndex + i
				break
			}
		}
		if startIndex < 0 {
			Fail("Failed to find stage start: " + stageName)
		}

		stopLinePattern := fmt.Sprintf("^Finished %s %s$", stageName, stageTimePattern)
		stopLineRegex, err := regexp.Compile(stopLinePattern)
		Expect(err).ToNot(HaveOccurred())

		stopIndex = -1
		for i, line := range outputLines[startIndex:] {
			if stopLineRegex.MatchString(line) {
				stopIndex = startIndex + i
				break
			}
		}
		if stopIndex < 0 {
			Fail("Failed to find stage stop: " + stageName)
		}

		return outputLines[startIndex+1 : stopIndex], stopIndex
	}

	BeforeSuite(func() {
		// writing to GinkgoWriter prints on test failure or when using verbose mode (-v)
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, GinkgoWriter, GinkgoWriter)
		fileSystem = boshsys.NewOsFileSystem(logger)

		var err error
		config, err = NewConfig(fileSystem)
		Expect(err).NotTo(HaveOccurred())

		err = config.Validate()
		Expect(err).NotTo(HaveOccurred())

		testEnv = NewRemoteTestEnvironment(
			config.VMUsername,
			config.VMIP,
			config.VMPort,
			config.PrivateKeyPath,
			fileSystem,
			logger,
		)

		sshCmdRunner = NewSSHCmdRunner(
			config.VMUsername,
			config.VMIP,
			config.VMPort,
			config.PrivateKeyPath,
			logger,
		)
		cmdEnv = map[string]string{
			"TMPDIR":              testEnv.Home(),
			"BOSH_INIT_LOG_LEVEL": "DEBUG",
			"BOSH_INIT_LOG_PATH":  testEnv.Path("bosh-init.log"),
		}
		quietCmdEnv = map[string]string{
			"TMPDIR":              testEnv.Home(),
			"BOSH_INIT_LOG_LEVEL": "ERROR",
			"BOSH_INIT_LOG_PATH":  testEnv.Path("bosh-init-cleanup.log"),
		}

		// clean up from previous failed tests
		deleteLogFile(cmdEnv["BOSH_INIT_LOG_PATH"])
		deleteLogFile(quietCmdEnv["BOSH_INIT_LOG_PATH"])

		instanceSSH = NewInstanceSSH(
			config.VMUsername,
			config.VMIP,
			config.VMPort,
			config.PrivateKeyPath,
			instanceUsername,
			instanceIP,
			instancePassword,
			fileSystem,
			logger,
		)

		err = bitestutils.BuildExecutableForArch("linux-amd64")
		Expect(err).NotTo(HaveOccurred())

		boshInitPath := "./../out/bosh-init"
		Expect(fileSystem.FileExists(boshInitPath)).To(BeTrue())
		err = testEnv.Copy("bosh-init", boshInitPath)
		Expect(err).NotTo(HaveOccurred())
		err = testEnv.DownloadOrCopy("stemcell.tgz", config.StemcellPath, config.StemcellURL)
		Expect(err).NotTo(HaveOccurred())
		err = testEnv.DownloadOrCopy("cpi-release.tgz", config.CpiReleasePath, config.CpiReleaseURL)
		Expect(err).NotTo(HaveOccurred())
		err = testEnv.Copy("dummy-release.tgz", config.DummyReleasePath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		flushLog(cmdEnv["BOSH_INIT_LOG_PATH"])

		// quietly delete the deployment
		_, _, exitCode, err := sshCmdRunner.RunCommand(quietCmdEnv, testEnv.Path("bosh-init"), "delete", testEnv.Path("test-manifest.yml"), testEnv.Path("cpi-release.tgz"))
		if exitCode != 0 || err != nil {
			// only flush the delete log if the delete failed
			flushLog(quietCmdEnv["BOSH_INIT_LOG_PATH"])
		}
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
	})

	It("is able to deploy given many variances", func() {
		updateDeploymentManifest("./assets/manifest.yml")

		println("#################################################")
		println("it can deploy successfully with expected output")
		println("#################################################")
		stdout := deploy()
		outputLines := strings.Split(stdout, "\n")

		doneIndex := 0

		validatingSteps, doneIndex := findStage(outputLines, "validating", doneIndex)
		Expect(validatingSteps[0]).To(MatchRegexp("^  Validating stemcell" + stageFinishedPattern))
		Expect(validatingSteps[1]).To(MatchRegexp("^  Validating releases" + stageFinishedPattern))
		Expect(validatingSteps[2]).To(MatchRegexp("^  Validating deployment manifest" + stageFinishedPattern))
		Expect(validatingSteps[3]).To(MatchRegexp("^  Validating cpi release" + stageFinishedPattern))
		Expect(validatingSteps).To(HaveLen(4))

		installingSteps, doneIndex := findStage(outputLines, "installing CPI", doneIndex+1)
		numInstallingSteps := len(installingSteps)
		for _, line := range installingSteps[:numInstallingSteps-3] {
			Expect(line).To(MatchRegexp("^  Compiling package '.*/.*'" + stageFinishedPattern))
		}
		Expect(installingSteps[numInstallingSteps-3]).To(MatchRegexp("^  Rendering job templates" + stageFinishedPattern))
		Expect(installingSteps[numInstallingSteps-2]).To(MatchRegexp("^  Installing packages" + stageFinishedPattern))
		Expect(installingSteps[numInstallingSteps-1]).To(MatchRegexp("^  Installing job 'cpi'" + stageFinishedPattern))

		Expect(outputLines[doneIndex+2]).To(MatchRegexp("^Starting registry" + stageFinishedPattern))
		Expect(outputLines[doneIndex+3]).To(MatchRegexp("^Uploading stemcell '.*/.*'" + stageFinishedPattern))

		deployingSteps, doneIndex := findStage(outputLines, "deploying", doneIndex+1)
		numDeployingSteps := len(deployingSteps)
		Expect(deployingSteps[0]).To(MatchRegexp("^  Creating VM for instance 'dummy_job/0' from stemcell '.*'" + stageFinishedPattern))
		Expect(deployingSteps[1]).To(MatchRegexp("^  Waiting for the agent on VM '.*' to be ready" + stageFinishedPattern))
		Expect(deployingSteps[2]).To(MatchRegexp("^  Creating disk" + stageFinishedPattern))
		Expect(deployingSteps[3]).To(MatchRegexp("^  Attaching disk '.*' to VM '.*'" + stageFinishedPattern))
		Expect(deployingSteps[4]).To(MatchRegexp("^  Rendering job templates" + stageFinishedPattern))

		for _, line := range deployingSteps[5 : numDeployingSteps-2] {
			Expect(line).To(MatchRegexp("^  Compiling package '.*/.*'" + stageFinishedPattern))
		}

		Expect(deployingSteps[numDeployingSteps-2]).To(MatchRegexp("^  Updating instance 'dummy_job/0'" + stageFinishedPattern))
		Expect(deployingSteps[numDeployingSteps-1]).To(MatchRegexp("^  Waiting for instance 'dummy_job/0' to be running" + stageFinishedPattern))

		println("#################################################")
		println("it sets the ssh password")
		println("#################################################")
		stdout, _, exitCode, err := instanceSSH.RunCommand("echo ssh-succeeded")
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
		Expect(stdout).To(ContainSubstring("ssh-succeeded"))

		println("#################################################")
		println("when there are no changes, it skips deploy")
		println("#################################################")
		stdout = deploy()

		Expect(stdout).To(ContainSubstring("No deployment, stemcell or release changes. Skipping deploy."))
		Expect(stdout).ToNot(ContainSubstring("Started installing CPI jobs"))
		Expect(stdout).ToNot(ContainSubstring("Started deploying"))

		println("#################################################")
		println("when updating with property changes, it deletes the old VM")
		println("#################################################")
		updateDeploymentManifest("./assets/modified_manifest.yml")

		stdout = deploy()

		Expect(stdout).To(ContainSubstring("Deleting VM"))
		Expect(stdout).To(ContainSubstring("Stopping jobs on instance 'unknown/0'"))
		Expect(stdout).To(ContainSubstring("Unmounting disk"))

		Expect(stdout).ToNot(ContainSubstring("Creating disk"))

		println("#################################################")
		println("when updating with disk size changed, it migrates the disk")
		println("#################################################")
		updateDeploymentManifest("./assets/modified_disk_manifest.yml")

		stdout = deploy()

		Expect(stdout).To(ContainSubstring("Deleting VM"))
		Expect(stdout).To(ContainSubstring("Stopping jobs on instance 'unknown/0'"))
		Expect(stdout).To(ContainSubstring("Unmounting disk"))

		Expect(stdout).To(ContainSubstring("Creating disk"))
		Expect(stdout).To(ContainSubstring("Migrating disk"))
		Expect(stdout).To(ContainSubstring("Deleting disk"))

		println("#################################################")
		println("when re-deploying without a working agent, it deletes the vm")
		println("#################################################")
		shutdownAgent()

		updateDeploymentManifest("./assets/modified_manifest.yml")

		stdout = deploy()

		Expect(stdout).To(MatchRegexp("Waiting for the agent on VM '.*'\\.\\.\\. Failed " + stageTimePattern))
		Expect(stdout).To(ContainSubstring("Deleting VM"))
		Expect(stdout).To(ContainSubstring("Creating VM for instance 'dummy_job/0' from stemcell"))
		Expect(stdout).To(ContainSubstring("Finished deploying"))

		println("#################################################")
		println("it can delete all vms, disk, and stemcells")
		println("#################################################")
		stdout = deleteDeployment()

		Expect(stdout).To(ContainSubstring("Stopping jobs on instance"))
		Expect(stdout).To(ContainSubstring("Deleting VM"))
		Expect(stdout).To(ContainSubstring("Deleting disk"))
		Expect(stdout).To(ContainSubstring("Deleting stemcell"))
		Expect(stdout).To(ContainSubstring("Finished deleting deployment"))
	})

	It("delete the vm even without a working agent", func() {
		updateDeploymentManifest("./assets/manifest.yml")

		deploy()
		shutdownAgent()

		stdout := deleteDeployment()

		Expect(stdout).To(MatchRegexp("Waiting for the agent on VM '.*'\\.\\.\\. Failed " + stageTimePattern))
		Expect(stdout).To(ContainSubstring("Deleting VM"))
		Expect(stdout).To(ContainSubstring("Deleting disk"))
		Expect(stdout).To(ContainSubstring("Deleting stemcell"))
		Expect(stdout).To(ContainSubstring("Finished deleting deployment"))
	})

	It("deploys & deletes without registry and ssh tunnel", func() {
		updateDeploymentManifest("./assets/manifest_without_registry.yml")

		stdout := deploy()
		Expect(stdout).To(ContainSubstring("Finished deploying"))

		stdout = deleteDeployment()
		Expect(stdout).To(ContainSubstring("Finished deleting deployment"))
	})

	It("prints multiple validation errors at the same time", func() {
		updateDeploymentManifest("./assets/invalid_manifest.yml")

		stdout := expectDeployToError()

		Expect(stdout).To(ContainSubstring("Validating deployment manifest... Failed"))
		Expect(stdout).To(ContainSubstring("Failed validating"))

		Expect(stdout).To(ContainSubstring(`
Command 'deploy' failed:
  Validating deployment manifest:
    jobs[0].templates[0].release must refer to an available release:
      Release 'unknown-release' is not available`))
	})
})
