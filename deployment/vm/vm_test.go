package vm_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/deployment/vm"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"

	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"

	bicloud "github.com/cloudfoundry/bosh-init/cloud"
	biproperty "github.com/cloudfoundry/bosh-init/common/property"
	biconfig "github.com/cloudfoundry/bosh-init/config"
	biagentclient "github.com/cloudfoundry/bosh-init/deployment/agentclient"
	bias "github.com/cloudfoundry/bosh-init/deployment/applyspec"
	bidisk "github.com/cloudfoundry/bosh-init/deployment/disk"
	bideplmanifest "github.com/cloudfoundry/bosh-init/deployment/manifest"

	fakebicloud "github.com/cloudfoundry/bosh-init/cloud/fakes"
	fakebiconfig "github.com/cloudfoundry/bosh-init/config/fakes"
	fakebiagentclient "github.com/cloudfoundry/bosh-init/deployment/agentclient/fakes"
	fakebidisk "github.com/cloudfoundry/bosh-init/deployment/disk/fakes"
	fakebivm "github.com/cloudfoundry/bosh-init/deployment/vm/fakes"
	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("VM", func() {
	var (
		vm               VM
		fakeVMRepo       *fakebiconfig.FakeVMRepo
		fakeStemcellRepo *fakebiconfig.FakeStemcellRepo
		fakeDiskDeployer *fakebivm.FakeDiskDeployer
		fakeAgentClient  *fakebiagentclient.FakeAgentClient
		fakeCloud        *fakebicloud.FakeCloud
		applySpec        bias.ApplySpec
		diskPool         bideplmanifest.DiskPool
		fs               *fakesys.FakeFileSystem
		logger           boshlog.Logger
	)

	BeforeEach(func() {
		fakeAgentClient = fakebiagentclient.NewFakeAgentClient()

		// apply spec is only being passed to the agent client, so it doesn't need much content for testing
		applySpec = bias.ApplySpec{
			Deployment: "fake-deployment-name",
		}

		diskPool = bideplmanifest.DiskPool{
			Name:     "fake-persistent-disk-pool-name",
			DiskSize: 1024,
			CloudProperties: biproperty.Map{
				"fake-disk-pool-cloud-property-key": "fake-disk-pool-cloud-property-value",
			},
		}

		logger = boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		fakeCloud = fakebicloud.NewFakeCloud()
		fakeVMRepo = fakebiconfig.NewFakeVMRepo()
		fakeStemcellRepo = fakebiconfig.NewFakeStemcellRepo()
		fakeDiskDeployer = fakebivm.NewFakeDiskDeployer()
		vm = NewVM(
			"fake-vm-cid",
			fakeVMRepo,
			fakeStemcellRepo,
			fakeDiskDeployer,
			fakeAgentClient,
			fakeCloud,
			fs,
			logger,
		)
	})

	Describe("Exists", func() {
		It("returns true when the vm exists", func() {
			fakeCloud.HasVMFound = true

			exists, err := vm.Exists()
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("returns false when the vm does not exist", func() {
			fakeCloud.HasVMFound = false

			exists, err := vm.Exists()
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("returns error when checking fails", func() {
			fakeCloud.HasVMErr = errors.New("fake-has-vm-error")

			_, err := vm.Exists()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-has-vm-error"))
		})
	})

	Describe("UpdateDisks", func() {
		var expectedDisks []bidisk.Disk

		BeforeEach(func() {
			fakeDisk := fakebidisk.NewFakeDisk("fake-disk-cid")
			expectedDisks = []bidisk.Disk{fakeDisk}
			fakeDiskDeployer.SetDeployBehavior(expectedDisks, nil)
		})

		It("delegates to DiskDeployer.Deploy", func() {
			fakeStage := fakebiui.NewFakeStage()

			disks, err := vm.UpdateDisks(diskPool, fakeStage)
			Expect(err).NotTo(HaveOccurred())
			Expect(disks).To(Equal(expectedDisks))

			Expect(fakeDiskDeployer.DeployInputs).To(Equal([]fakebivm.DeployInput{
				{
					DiskPool:         diskPool,
					Cloud:            fakeCloud,
					VM:               vm,
					EventLoggerStage: fakeStage,
				},
			}))
		})
	})

	Describe("Stop", func() {
		It("stops agent services", func() {
			err := vm.Stop()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.StopCalled).To(BeTrue())
		})

		Context("when stopping an agent fails", func() {
			BeforeEach(func() {
				fakeAgentClient.SetStopBehavior(errors.New("fake-stop-error"))
			})

			It("returns an error", func() {
				err := vm.Stop()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-stop-error"))
			})
		})
	})

	Describe("Apply", func() {
		It("sends apply spec to the agent", func() {
			err := vm.Apply(applySpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.ApplyApplySpec).To(Equal(applySpec))
		})

		Context("when sending apply spec to the agent fails", func() {
			BeforeEach(func() {
				fakeAgentClient.ApplyErr = errors.New("fake-agent-apply-err")
			})

			It("returns an error", func() {
				err := vm.Apply(applySpec)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-agent-apply-err"))
			})
		})
	})

	Describe("Start", func() {
		It("starts agent services", func() {
			err := vm.Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.StartCalled).To(BeTrue())
		})

		Context("when starting an agent fails", func() {
			BeforeEach(func() {
				fakeAgentClient.SetStartBehavior(errors.New("fake-start-error"))
			})

			It("returns an error", func() {
				err := vm.Start()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-start-error"))
			})
		})
	})

	Describe("WaitToBeRunning", func() {
		BeforeEach(func() {
			fakeAgentClient.SetGetStateBehavior(biagentclient.AgentState{JobState: "pending"}, nil)
			fakeAgentClient.SetGetStateBehavior(biagentclient.AgentState{JobState: "pending"}, nil)
			fakeAgentClient.SetGetStateBehavior(biagentclient.AgentState{JobState: "running"}, nil)
		})

		It("waits until agent reports state as running", func() {
			err := vm.WaitToBeRunning(5, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.GetStateCalledTimes).To(Equal(3))
		})
	})

	Describe("AttachDisk", func() {
		var disk *fakebidisk.FakeDisk

		BeforeEach(func() {
			disk = fakebidisk.NewFakeDisk("fake-disk-cid")
		})

		It("attaches disk to vm in the cloud", func() {
			err := vm.AttachDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.AttachDiskInput).To(Equal(fakebicloud.AttachDiskInput{
				VMCID:   "fake-vm-cid",
				DiskCID: "fake-disk-cid",
			}))
		})

		It("sends mount disk to the agent", func() {
			err := vm.AttachDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.MountDiskCID).To(Equal("fake-disk-cid"))
		})

		Context("when attaching disk to cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.AttachDiskErr = errors.New("fake-attach-error")
			})

			It("returns an error", func() {
				err := vm.AttachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-attach-error"))
			})
		})

		Context("when mounting disk fails", func() {
			BeforeEach(func() {
				fakeAgentClient.SetMountDiskBehavior(errors.New("fake-mount-error"))
			})

			It("returns an error", func() {
				err := vm.AttachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mount-error"))
			})
		})
	})

	Describe("DetachDisk", func() {
		var disk *fakebidisk.FakeDisk

		BeforeEach(func() {
			disk = fakebidisk.NewFakeDisk("fake-disk-cid")
		})

		It("detaches disk from vm in the cloud", func() {
			err := vm.DetachDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.DetachDiskInput).To(Equal(fakebicloud.DetachDiskInput{
				VMCID:   "fake-vm-cid",
				DiskCID: "fake-disk-cid",
			}))
		})

		Context("when detaching disk to cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.DetachDiskErr = errors.New("fake-detach-error")
			})

			It("returns an error", func() {
				err := vm.DetachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-detach-error"))
			})
		})
	})

	Describe("UnmountDisk", func() {
		var disk *fakebidisk.FakeDisk

		BeforeEach(func() {
			disk = fakebidisk.NewFakeDisk("fake-disk-cid")
		})

		It("sends unmount disk to the agent", func() {
			err := vm.UnmountDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.UnmountDiskCID).To(Equal("fake-disk-cid"))
		})

		Context("when unmounting disk fails", func() {
			BeforeEach(func() {
				fakeAgentClient.SetUnmountDiskBehavior(errors.New("fake-unmount-error"))
			})

			It("returns an error", func() {
				err := vm.UnmountDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-unmount-error"))
			})
		})
	})

	Describe("Disks", func() {
		BeforeEach(func() {
			fakeAgentClient.SetListDiskBehavior([]string{"fake-disk-cid-1", "fake-disk-cid-2"}, nil)
		})

		It("returns disks that are reported by the agent", func() {
			disks, err := vm.Disks()
			Expect(err).ToNot(HaveOccurred())
			expectedFirstDisk := bidisk.NewDisk(biconfig.DiskRecord{CID: "fake-disk-cid-1"}, nil, nil)
			expectedSecondDisk := bidisk.NewDisk(biconfig.DiskRecord{CID: "fake-disk-cid-2"}, nil, nil)
			Expect(disks).To(Equal([]bidisk.Disk{expectedFirstDisk, expectedSecondDisk}))
		})

		Context("when listing disks fails", func() {
			BeforeEach(func() {
				fakeAgentClient.SetListDiskBehavior([]string{}, errors.New("fake-list-disk-error"))
			})

			It("returns an error", func() {
				_, err := vm.Disks()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-list-disk-error"))
			})
		})
	})

	Describe("Delete", func() {
		It("deletes vm in the cloud", func() {
			err := vm.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.DeleteVMInput).To(Equal(fakebicloud.DeleteVMInput{
				VMCID: "fake-vm-cid",
			}))
		})

		It("deletes VM in the vm repo", func() {
			err := vm.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeVMRepo.ClearCurrentCalled).To(BeTrue())
		})

		It("clears current stemcell in the stemcell repo", func() {
			err := vm.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeStemcellRepo.ClearCurrentCalled).To(BeTrue())
		})

		Context("when deleting vm in the cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.DeleteVMErr = errors.New("fake-delete-vm-error")
			})

			It("returns an error", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-vm-error"))
			})
		})

		Context("when deleting vm in the cloud fails with VMNotFoundError", func() {
			var deleteErr = bicloud.NewCPIError("delete_vm", bicloud.CmdError{
				Type:    bicloud.VMNotFoundError,
				Message: "fake-vm-not-found-message",
			})

			BeforeEach(func() {
				fakeCloud.DeleteVMErr = deleteErr
			})

			It("deletes vm in the cloud", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))
				Expect(fakeCloud.DeleteVMInput).To(Equal(fakebicloud.DeleteVMInput{
					VMCID: "fake-vm-cid",
				}))
			})

			It("deletes VM in the vm repo", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))
				Expect(fakeVMRepo.ClearCurrentCalled).To(BeTrue())
			})

			It("clears current stemcell in the stemcell repo", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))
				Expect(fakeStemcellRepo.ClearCurrentCalled).To(BeTrue())
			})
		})
	})

	Describe("MigrateDisk", func() {
		It("sends migrate_disk to the agent", func() {
			err := vm.MigrateDisk()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.MigrateDiskCalledTimes).To(Equal(1))
		})

		Context("when migrating disk fails", func() {
			BeforeEach(func() {
				fakeAgentClient.SetMigrateDiskBehavior(errors.New("fake-migrate-error"))
			})

			It("returns an error", func() {
				err := vm.MigrateDisk()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-migrate-error"))
			})
		})
	})
})
