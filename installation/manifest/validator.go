package manifest

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"

	birelset "github.com/cloudfoundry/bosh-init/release/set"
)

type Validator interface {
	Validate(Manifest) error
}

type validator struct {
	logger          boshlog.Logger
	releaseResolver birelset.Resolver
}

func NewValidator(logger boshlog.Logger, releaseResolver birelset.Resolver) Validator {
	return &validator{
		logger:          logger,
		releaseResolver: releaseResolver,
	}
}

func (v *validator) Validate(manifest Manifest) error {
	errs := []error{}

	cpiJobName := manifest.Template.Name
	if v.isBlank(cpiJobName) {
		errs = append(errs, bosherr.Error("cloud_provider.template.name must be provided"))
	}

	cpiReleaseName := manifest.Template.Release
	if v.isBlank(cpiReleaseName) {
		errs = append(errs, bosherr.Error("cloud_provider.template.release must be provided"))
	}

	_, err := v.releaseResolver.Find(cpiReleaseName)
	if err != nil {
		errs = append(errs, bosherr.WrapErrorf(err, "cloud_provider.template.release '%s' must refer to a provided release", cpiReleaseName))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil
}

func (v *validator) isBlank(str string) bool {
	return str == "" || strings.TrimSpace(str) == ""
}
