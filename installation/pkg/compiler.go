package pkg

import (
	"os"
	"path"

	boshblob "github.com/cloudfoundry/bosh-agent/blobstore"
	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshcmd "github.com/cloudfoundry/bosh-agent/platform/commands"
	boshsys "github.com/cloudfoundry/bosh-agent/system"

	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bistatepkg "github.com/cloudfoundry/bosh-init/state/pkg"
)

type compiler struct {
	runner              boshsys.CmdRunner
	packagesDir         string
	fileSystem          boshsys.FileSystem
	compressor          boshcmd.Compressor
	blobstore           boshblob.Blobstore
	compiledPackageRepo bistatepkg.CompiledPackageRepo
	packageInstaller    Installer
	logger              boshlog.Logger
	logTag              string
}

func NewPackageCompiler(
	runner boshsys.CmdRunner,
	packagesDir string,
	fileSystem boshsys.FileSystem,
	compressor boshcmd.Compressor,
	blobstore boshblob.Blobstore,
	compiledPackageRepo bistatepkg.CompiledPackageRepo,
	packageInstaller Installer,
	logger boshlog.Logger,
) bistatepkg.Compiler {
	return &compiler{
		runner:              runner,
		packagesDir:         packagesDir,
		fileSystem:          fileSystem,
		compressor:          compressor,
		blobstore:           blobstore,
		compiledPackageRepo: compiledPackageRepo,
		packageInstaller:    packageInstaller,
		logger:              logger,
		logTag:              "packageCompiler",
	}
}

func (c *compiler) Compile(pkg *birelpkg.Package) (record bistatepkg.CompiledPackageRecord, err error) {
	c.logger.Debug(c.logTag, "Checking for compiled package '%s/%s'", pkg.Name, pkg.Fingerprint)
	record, found, err := c.compiledPackageRepo.Find(*pkg)
	if err != nil {
		return record, bosherr.WrapErrorf(err, "Attempting to find compiled package '%s'", pkg.Name)
	}
	if found {
		return record, nil
	}

	c.logger.Debug(c.logTag, "Installing dependencies of package '%s/%s'", pkg.Name, pkg.Fingerprint)
	err = c.installPackages(pkg.Dependencies)
	if err != nil {
		return record, bosherr.WrapErrorf(err, "Installing dependencies of package '%s'", pkg.Name)
	}
	defer c.fileSystem.RemoveAll(c.packagesDir)

	c.logger.Debug(c.logTag, "Compiling package '%s/%s'", pkg.Name, pkg.Fingerprint)
	installDir := path.Join(c.packagesDir, pkg.Name)
	err = c.fileSystem.MkdirAll(installDir, os.ModePerm)
	if err != nil {
		return record, bosherr.WrapError(err, "Creating package install dir")
	}

	packageSrcDir := pkg.ExtractedPath
	if !c.fileSystem.FileExists(path.Join(packageSrcDir, "packaging")) {
		return record, bosherr.Errorf("Packaging script for package '%s' not found", pkg.Name)
	}

	cmd := boshsys.Command{
		Name: "bash",
		Args: []string{"-x", "packaging"},
		Env: map[string]string{
			"BOSH_COMPILE_TARGET": packageSrcDir,
			"BOSH_INSTALL_TARGET": installDir,
			"BOSH_PACKAGE_NAME":   pkg.Name,
			"BOSH_PACKAGES_DIR":   c.packagesDir,
			"PATH":                "/usr/local/bin:/usr/bin:/bin",
		},
		UseIsolatedEnv: true,
		WorkingDir:     packageSrcDir,
	}

	_, _, _, err = c.runner.RunComplexCommand(cmd)
	if err != nil {
		return record, bosherr.WrapError(err, "Compiling package")
	}

	tarball, err := c.compressor.CompressFilesInDir(installDir)
	if err != nil {
		return record, bosherr.WrapError(err, "Compressing compiled package")
	}
	defer c.compressor.CleanUp(tarball)

	blobID, blobSHA1, err := c.blobstore.Create(tarball)
	if err != nil {
		return record, bosherr.WrapError(err, "Creating blob")
	}

	record = bistatepkg.CompiledPackageRecord{
		BlobID:   blobID,
		BlobSHA1: blobSHA1,
	}
	err = c.compiledPackageRepo.Save(*pkg, record)
	if err != nil {
		return record, bosherr.WrapError(err, "Saving compiled package")
	}

	return record, nil
}

func (c *compiler) installPackages(packages []*birelpkg.Package) error {
	for _, pkg := range packages {
		c.logger.Debug(c.logTag, "Checking for compiled package '%s/%s'", pkg.Name, pkg.Fingerprint)
		record, found, err := c.compiledPackageRepo.Find(*pkg)
		if err != nil {
			return bosherr.WrapErrorf(err, "Attempting to find compiled package '%s'", pkg.Name)
		}
		if !found {
			return bosherr.Errorf("Finding compiled package '%s'", pkg.Name)
		}

		c.logger.Debug(c.logTag, "Installing package '%s/%s'", pkg.Name, pkg.Fingerprint)
		compiledPackageRef := CompiledPackageRef{
			Name:        pkg.Name,
			Version:     pkg.Fingerprint,
			BlobstoreID: record.BlobID,
			SHA1:        record.BlobSHA1,
		}

		err = c.packageInstaller.Install(compiledPackageRef, c.packagesDir)
		if err != nil {
			return bosherr.WrapErrorf(err, "Installing package '%s' into '%s'", pkg.Name, c.packagesDir)
		}
	}

	return nil
}
