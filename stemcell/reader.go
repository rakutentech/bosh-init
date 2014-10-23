package stemcell

import (
	"github.com/cloudfoundry-incubator/candiedyaml"

	"path/filepath"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshcmd "github.com/cloudfoundry/bosh-agent/platform/commands"
	boshsys "github.com/cloudfoundry/bosh-agent/system"
)

// Reader reads a stemcell tarball and returns a stemcell object containing
// parsed information (e.g. version, name)
//
type Reader interface {
	Read(stemcellTarballPath string, extractedPath string) (Stemcell, error)
}

type reader struct {
	compressor boshcmd.Compressor
	fs         boshsys.FileSystem
}

func NewReader(compressor boshcmd.Compressor, fs boshsys.FileSystem) Reader {
	return reader{compressor: compressor, fs: fs}
}

func (s reader) Read(stemcellTarballPath string, extractedPath string) (Stemcell, error) {
	var stemcell Stemcell

	err := s.compressor.DecompressFileToDir(stemcellTarballPath, extractedPath, boshcmd.CompressorOptions{})
	if err != nil {
		return Stemcell{}, bosherr.WrapError(err, "Extracting stemcell from %s to %s", stemcellTarballPath, extractedPath)
	}

	var stemcellManifest Manifest
	stemcellManifestPath := filepath.Join(extractedPath, "stemcell.MF")

	stemcellManifestContents, err := s.fs.ReadFile(stemcellManifestPath)
	if err != nil {
		return Stemcell{}, bosherr.WrapError(err, "Reading stemcell manifest %s", stemcellManifestPath)
	}

	err = candiedyaml.Unmarshal(stemcellManifestContents, &stemcellManifest)
	if err != nil {
		return Stemcell{}, bosherr.WrapError(err, "Parsing stemcell manifest %s", stemcellManifestContents)
	}

	stemcellManifest.ImagePath = filepath.Join(extractedPath, "image")
	stemcell.Manifest = stemcellManifest

	return stemcell, nil
}
