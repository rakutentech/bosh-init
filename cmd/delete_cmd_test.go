package cmd_test

import (
	. "github.com/cloudfoundry/bosh-init/cmd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"

	"code.google.com/p/gomock/gomock"
	mock_blobstore "github.com/cloudfoundry/bosh-init/blobstore/mocks"
	mock_cloud "github.com/cloudfoundry/bosh-init/cloud/mocks"
	mock_httpagent "github.com/cloudfoundry/bosh-init/deployment/agentclient/http/mocks"
	mock_agentclient "github.com/cloudfoundry/bosh-init/deployment/agentclient/mocks"
	mock_deployment "github.com/cloudfoundry/bosh-init/deployment/mocks"
	mock_install "github.com/cloudfoundry/bosh-init/installation/mocks"
	mock_release "github.com/cloudfoundry/bosh-init/release/mocks"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshsys "github.com/cloudfoundry/bosh-agent/system"

	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-agent/uuid/fakes"

	biproperty "github.com/cloudfoundry/bosh-init/common/property"
	biconfig "github.com/cloudfoundry/bosh-init/config"
	biinstallmanifest "github.com/cloudfoundry/bosh-init/installation/manifest"
	birel "github.com/cloudfoundry/bosh-init/release"
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	birelset "github.com/cloudfoundry/bosh-init/release/set"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	biui "github.com/cloudfoundry/bosh-init/ui"

	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
	fakeui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("DeleteCmd", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {
		var (
			fs                           boshsys.FileSystem
			logger                       boshlog.Logger
			releaseManager               birel.Manager
			mockInstaller                *mock_install.MockInstaller
			mockInstallerFactory         *mock_install.MockInstallerFactory
			mockInstallation             *mock_install.MockInstallation
			mockCloudFactory             *mock_cloud.MockFactory
			mockReleaseExtractor         *mock_release.MockExtractor
			fakeUUIDGenerator            *fakeuuid.FakeGenerator
			setupDeploymentConfigService biconfig.DeploymentConfigService
			userConfig                   biconfig.UserConfig

			fakeUI *fakeui.FakeUI

			mockBlobstoreFactory *mock_blobstore.MockFactory
			mockBlobstore        *mock_blobstore.MockBlobstore

			mockDeploymentManagerFactory *mock_deployment.MockManagerFactory
			mockDeploymentManager        *mock_deployment.MockManager
			mockDeployment               *mock_deployment.MockDeployment

			mockAgentClient        *mock_agentclient.MockAgentClient
			mockAgentClientFactory *mock_httpagent.MockAgentClientFactory
			mockCloud              *mock_cloud.MockCloud

			fakeStage *fakebiui.FakeStage

			directorID string

			deploymentManifestPath = "/deployment-dir/fake-deployment-manifest.yml"
			deploymentConfigPath   string

			expectCPIExtractRelease *gomock.Call
			expectCPIInstall        *gomock.Call
			expectNewCloud          *gomock.Call
			expectStartRegistry     *gomock.Call
			expectStopRegistry      *gomock.Call

			mbusURL = "http://fake-mbus-user:fake-mbus-password@fake-mbus-endpoint"
		)

		var writeDeploymentManifest = func() {
			fs.WriteFileString(deploymentManifestPath, `---
name: test-release

cloud_provider:
  template:
    name: fake-cpi-release-job-name
    release: fake-cpi-release-name
  mbus: http://fake-mbus-user:fake-mbus-password@fake-mbus-endpoint
`)
		}

		var writeCPIReleaseTarball = func() {
			fs.WriteFileString("/fake-cpi-release.tgz", "fake-tgz-content")
		}

		var allowCPIToBeExtracted = func() {
			cpiRelease := birel.NewRelease(
				"fake-cpi-release-name",
				"fake-cpi-release-version",
				[]bireljob.Job{
					{
						Name: "fake-cpi-release-job-name",
						Templates: map[string]string{
							"templates/cpi.erb": "bin/cpi",
						},
					},
				},
				[]*birelpkg.Package{},
				"fake-cpi-extracted-dir",
				fs,
			)

			expectCPIExtractRelease = mockReleaseExtractor.EXPECT().Extract("/fake-cpi-release.tgz").Do(func(_ string) {
				err := fs.MkdirAll("fake-cpi-extracted-dir", os.ModePerm)
				Expect(err).ToNot(HaveOccurred())
			}).Return(cpiRelease, nil).AnyTimes()
		}

		var allowCPIToBeInstalled = func() {
			installationManifest := biinstallmanifest.Manifest{
				Name: "test-release",
				Template: biinstallmanifest.ReleaseJobRef{
					Name:    "fake-cpi-release-job-name",
					Release: "fake-cpi-release-name",
				},
				Mbus:       mbusURL,
				Properties: biproperty.Map{},
			}

			mockInstallerFactory.EXPECT().NewInstaller().Return(mockInstaller, nil).AnyTimes()

			expectCPIInstall = mockInstaller.EXPECT().Install(installationManifest, gomock.Any()).Do(func(_ biinstallmanifest.Manifest, stage biui.Stage) {
				Expect(fakeStage.SubStages).To(ContainElement(stage))
			}).Return(mockInstallation, nil).AnyTimes()

			expectNewCloud = mockCloudFactory.EXPECT().NewCloud(mockInstallation, directorID).Return(mockCloud, nil).AnyTimes()

			expectStartRegistry = mockInstallation.EXPECT().StartRegistry().AnyTimes()
			expectStopRegistry = mockInstallation.EXPECT().StopRegistry().AnyTimes()
		}

		var newDeleteCmd = func() Cmd {
			releaseSetParser := birelsetmanifest.NewParser(fs, logger)
			releaseSetResolver := birelset.NewResolver(releaseManager, logger)
			releaseSetValidator := birelsetmanifest.NewValidator(logger, releaseSetResolver)
			installationValidator := biinstallmanifest.NewValidator(logger, releaseSetResolver)
			installationParser := biinstallmanifest.NewParser(fs, logger)

			return NewDeleteCmd(
				fakeUI,
				userConfig,
				fs,
				releaseSetParser,
				installationParser,
				biconfig.NewFileSystemDeploymentConfigService(fs, fakeUUIDGenerator, logger),
				releaseSetValidator,
				installationValidator,
				mockInstallerFactory,
				mockReleaseExtractor,
				releaseManager,
				releaseSetResolver,
				mockCloudFactory,
				mockAgentClientFactory,
				mockBlobstoreFactory,
				mockDeploymentManagerFactory,
				logger,
			)
		}

		var expectDeleteAndCleanup = func() {
			mockDeploymentManagerFactory.EXPECT().NewManager(mockCloud, mockAgentClient, mockBlobstore).Return(mockDeploymentManager)
			mockDeploymentManager.EXPECT().FindCurrent().Return(mockDeployment, true, nil)

			gomock.InOrder(
				mockDeployment.EXPECT().Delete(gomock.Any()).Do(func(stage biui.Stage) {
					Expect(fakeStage.SubStages).To(ContainElement(stage))
				}),
				mockDeploymentManager.EXPECT().Cleanup(fakeStage),
			)
		}

		var expectCleanup = func() {
			mockDeploymentManagerFactory.EXPECT().NewManager(mockCloud, mockAgentClient, mockBlobstore).Return(mockDeploymentManager).AnyTimes()
			mockDeploymentManager.EXPECT().FindCurrent().Return(nil, false, nil).AnyTimes()

			mockDeploymentManager.EXPECT().Cleanup(fakeStage)
		}

		var expectValidationInstallationDeletionEvents = func() {
			Expect(fakeUI.Said).To(Equal([]string{
				"Deployment manifest: '/deployment-dir/fake-deployment-manifest.yml'",
				"Deployment state: '/deployment-dir/deployment.json'",
			}))

			Expect(fakeStage.PerformCalls).To(Equal([]fakebiui.PerformCall{
				{
					Name: "validating",
					Stage: &fakebiui.FakeStage{
						PerformCalls: []fakebiui.PerformCall{
							{Name: "Validating releases"},
							{Name: "Validating deployment manifest"},
							{Name: "Validating cpi release"},
						},
					},
				},
				{
					Name:  "installing CPI",
					Stage: &fakebiui.FakeStage{},
				},
				{Name: "Starting registry"},
				{
					Name:  "deleting deployment",
					Stage: &fakebiui.FakeStage{},
				},
				// mock deployment manager cleanup doesn't add sub-stages
			}))

			// installing steps handled by installer.Install()
			// deleting steps handled by deployment.Delete()
		}

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()
			logger = boshlog.NewLogger(boshlog.LevelNone)
			fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
			setupDeploymentConfigService = biconfig.NewFileSystemDeploymentConfigService(fs, fakeUUIDGenerator, logger)
			deploymentConfigPath = biconfig.UserConfig{DeploymentManifestPath: deploymentManifestPath}.DeploymentConfigPath()
			setupDeploymentConfigService.SetConfigPath(deploymentConfigPath)
			setupDeploymentConfigService.Load()

			fakeUI = &fakeui.FakeUI{}

			fakeStage = fakebiui.NewFakeStage()

			mockCloud = mock_cloud.NewMockCloud(mockCtrl)
			mockCloudFactory = mock_cloud.NewMockFactory(mockCtrl)

			mockInstaller = mock_install.NewMockInstaller(mockCtrl)
			mockInstallerFactory = mock_install.NewMockInstallerFactory(mockCtrl)
			mockInstallation = mock_install.NewMockInstallation(mockCtrl)

			mockBlobstoreFactory = mock_blobstore.NewMockFactory(mockCtrl)
			mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)
			mockBlobstoreFactory.EXPECT().Create(mbusURL).Return(mockBlobstore, nil).AnyTimes()

			mockDeploymentManagerFactory = mock_deployment.NewMockManagerFactory(mockCtrl)
			mockDeploymentManager = mock_deployment.NewMockManager(mockCtrl)
			mockDeployment = mock_deployment.NewMockDeployment(mockCtrl)

			mockReleaseExtractor = mock_release.NewMockExtractor(mockCtrl)
			releaseManager = birel.NewManager(logger)

			mockAgentClientFactory = mock_httpagent.NewMockAgentClientFactory(mockCtrl)
			mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)

			userConfig = biconfig.UserConfig{}

			mockAgentClientFactory.EXPECT().NewAgentClient(gomock.Any(), gomock.Any()).Return(mockAgentClient).AnyTimes()

			directorID = "fake-uuid-0"

			writeDeploymentManifest()
			writeCPIReleaseTarball()
		})

		JustBeforeEach(func() {
			allowCPIToBeExtracted()
			allowCPIToBeInstalled()
		})

		Context("when the deployment manifest does not exist", func() {
			It("returns an error", func() {
				err := newDeleteCmd().Run(fakeStage, []string{"/garbage", "/fake-cpi-release.tgz"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Deployment manifest does not exist at '/garbage'"))
				Expect(fakeUI.Errors).To(ContainElement("Deployment '/garbage' does not exist"))
			})
		})

		Context("when the deployment config file does not exist", func() {
			BeforeEach(func() {
				err := fs.RemoveAll(deploymentConfigPath)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not delete anything", func() {
				err := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath, "/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeUI.Said).To(Equal([]string{
					"Deployment manifest: '/deployment-dir/fake-deployment-manifest.yml'",
					"Deployment state: '/deployment-dir/deployment.json'",
					"No deployment config file found.",
				}))
			})
		})

		Context("when the deployment has been deployed", func() {
			BeforeEach(func() {
				directorID = "fake-director-id"

				// create deployment manifest yaml file
				setupDeploymentConfigService.Save(biconfig.DeploymentFile{
					DirectorID: directorID,
				})
			})

			It("extracts & install CPI release tarball", func() {
				expectDeleteAndCleanup()

				gomock.InOrder(
					expectCPIExtractRelease.Times(1),
					expectCPIInstall.Times(1),
					expectNewCloud.Times(1),
				)

				err := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath, "/fake-cpi-release.tgz"})
				Expect(err).NotTo(HaveOccurred())
			})

			It("starts & stops the registry", func() {
				expectDeleteAndCleanup()

				gomock.InOrder(
					expectStartRegistry.Times(1),
					expectStopRegistry.Times(1),
				)

				err := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath, "/fake-cpi-release.tgz"})
				Expect(err).NotTo(HaveOccurred())
			})

			It("deletes the extracted CPI release", func() {
				expectDeleteAndCleanup()

				err := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath, "/fake-cpi-release.tgz"})
				Expect(err).NotTo(HaveOccurred())
				Expect(fs.FileExists("fake-cpi-extracted-dir")).To(BeFalse())
			})

			It("deletes the deployment & cleans up orphans", func() {
				expectDeleteAndCleanup()

				err := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath, "/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeUI.Errors).To(BeEmpty())
			})

			It("logs validating & deleting stages", func() {
				expectDeleteAndCleanup()

				err := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath, "/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())

				expectValidationInstallationDeletionEvents()
			})
		})

		Context("when nothing has been deployed", func() {
			BeforeEach(func() {
				setupDeploymentConfigService.Save(biconfig.DeploymentFile{DirectorID: "fake-uuid-0"})
			})

			It("cleans up orphans, but does not delete any deployment", func() {
				expectCleanup()

				err := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath, "/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeUI.Errors).To(BeEmpty())
			})
		})
	})
})
