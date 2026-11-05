package manifest

import (
	"errors"
	"fmt"

	"github.com/vrooli/binaryfetch"
)

// ValidateAcquisitionAuthReference binds a credential reference to the resource
// that declares it. No credential store is read during manifest validation.
func ValidateAcquisitionAuthReference(manifest ResourceManifest, auth binaryfetch.AcquisitionAuth) error {
	if manifest.Driver != "managed-service" {
		return errors.New("acquisition auth is supported only for managed-service resources")
	}
	for _, descriptor := range manifest.Credentials.All() {
		if descriptor.LogicalID == auth.LogicalID && descriptor.ResolvedField() == auth.Field {
			return nil
		}
	}
	return fmt.Errorf("acquisition auth %s:%s must match a declared resource credential", auth.LogicalID, auth.Field)
}

func validateAcquisitionAuthReferences(manifest ResourceManifest) error {
	acquisitions := []*binaryfetch.Acquisition{manifest.Acquisition}
	if manifest.ManagedService != nil {
		acquisitions = append(acquisitions, manifest.ManagedService.Acquisition)
		for _, data := range manifest.ManagedService.DataArtifacts {
			acquisitions = append(acquisitions, data.Acquisition)
		}
	}
	for _, acquisition := range acquisitions {
		if acquisition == nil {
			continue
		}
		for _, target := range acquisition.Targets {
			if target.Auth != nil {
				if err := ValidateAcquisitionAuthReference(manifest, *target.Auth); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// LoadForDataInspection reads a draft's declared data URLs/credentials without
// admitting any execution. Unknown data/runtime digests are legitimate during
// pin discovery. Normal Load retains every artifact/install validation gate.
func LoadForDataInspection(path string) (ResourceManifest, error) {
	manifest, err := readManifest(path)
	if err != nil {
		return ResourceManifest{}, err
	}
	if manifest.Name == "" || manifest.Driver != "managed-service" || manifest.ManagedService == nil {
		return ResourceManifest{}, errors.New("data inspection requires a named managed-service resource")
	}
	if err := validateAcquisitionAuthReferences(manifest); err != nil {
		return ResourceManifest{}, err
	}
	return manifest, nil
}
