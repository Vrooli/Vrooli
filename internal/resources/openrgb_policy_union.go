package resources

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func (r OpenRGBDevicePolicyRecord) ID() (string, error) {
	if r.NativePolicy != nil {
		if !reflect.DeepEqual(r.Policy, hostreqkit.OpenRGBLinuxPolicy{}) {
			return "", errors.New("ambiguous OpenRGB Linux/native policy")
		}
		return r.NativePolicy.ID()
	}
	return r.Policy.ID()
}
func (r OpenRGBDevicePolicyRecord) selectedDetectors(catalog map[string]bool) ([]string, error) {
	if r.NativePolicy == nil {
		return selectedOpenRGBDetectors(r.Policy, catalog)
	}
	if _, err := r.ID(); err != nil {
		return nil, err
	}
	if err := hostreqkit.ValidateOpenRGBNativeDetectorScope(*r.NativePolicy); err != nil {
		return nil, err
	}
	var detectors []string
	for _, d := range r.NativePolicy.Devices {
		if _, ok := catalog[d.Detector]; !ok {
			return nil, errors.New("native HID detector absent from pinned catalog")
		}
		if !slices.Contains(detectors, d.Detector) {
			detectors = append(detectors, d.Detector)
		}
	}
	slices.Sort(detectors)
	return detectors, nil
}
func (r OpenRGBDevicePolicyRecord) freezeExecPolicy(artifact string, catalog, settings []byte) (openRGBExecPolicy, error) {
	if _, err := r.ID(); err != nil {
		return openRGBExecPolicy{}, err
	}
	if r.NativePolicy != nil {
		return freezeOpenRGBNativeExecPolicy(*r.NativePolicy, artifact, catalog, settings)
	}
	return freezeOpenRGBExecPolicy(r.Policy, artifact, catalog, settings)
}
func freezeOpenRGBNativeExecPolicy(p hostreqkit.OpenRGBNativePolicy, artifact string, catalog, settings []byte) (openRGBExecPolicy, error) {
	if err := hostreqkit.ValidateOpenRGBNativeDetectorScope(p); err != nil {
		return openRGBExecPolicy{}, err
	}
	id, err := p.ID()
	if err != nil {
		return openRGBExecPolicy{}, err
	}
	catalogSHA := fmt.Sprintf("%x", sha256.Sum256(catalog))
	if p.ArtifactSHA256 != artifact || p.DetectorSHA256 != catalogSHA {
		return openRGBExecPolicy{}, errors.New("native OpenRGB policy differs from the frozen release/catalog")
	}
	decoded, err := decodeOpenRGBDetectorCatalog(catalog)
	if err != nil {
		return openRGBExecPolicy{}, err
	}
	detectors, err := (OpenRGBDevicePolicyRecord{NativePolicy: &p}).selectedDetectors(decoded)
	if err != nil {
		return openRGBExecPolicy{}, err
	}
	if err = verifyOpenRGBExecDetectors(settings, decoded, detectors); err != nil {
		return openRGBExecPolicy{}, err
	}
	if err = verifyNativeOpenRGBSettings(settings); err != nil {
		return openRGBExecPolicy{}, err
	}
	return openRGBExecPolicy{ID: id, ArtifactSHA256: artifact, DetectorCatalogSHA256: catalogSHA, ConfigSHA256: fmt.Sprintf("%x", sha256.Sum256(settings)), Platform: p.Platform, Account: p.Account}, nil
}
func (r openRGBFrozenUpdateRole) record() OpenRGBDevicePolicyRecord {
	if r.NativePolicy != nil {
		return OpenRGBDevicePolicyRecord{SchemaVersion: openRGBNativeDevicePolicySchemaVersion, Policy: r.Policy, NativePolicy: r.NativePolicy}
	}
	return OpenRGBDevicePolicyRecord{SchemaVersion: openRGBDevicePolicySchemaVersion, Policy: r.Policy}
}
func (r openRGBFrozenUpdateRole) policyIdentity() (id, artifact, catalog string, err error) {
	id, err = r.record().ID()
	if err != nil {
		return
	}
	if r.NativePolicy != nil {
		return id, r.NativePolicy.ArtifactSHA256, r.NativePolicy.DetectorSHA256, nil
	}
	return id, r.Policy.ArtifactSHA256, r.Policy.DetectorSHA256, nil
}
func (r openRGBFrozenUpdateRole) samePhysicalGrant(other openRGBFrozenUpdateRole) bool {
	if r.NativePolicy == nil && other.NativePolicy == nil {
		return sameOpenRGBPhysicalGrant(r.Policy, other.Policy)
	}
	if r.NativePolicy == nil || other.NativePolicy == nil {
		return false
	}
	return sameNativeOpenRGBPhysicalGrant(*r.NativePolicy, *other.NativePolicy)
}
func sameNativeOpenRGBPhysicalGrant(a, b hostreqkit.OpenRGBNativePolicy) bool {
	if a.Platform != b.Platform || a.MachineID != b.MachineID || a.Account != b.Account || len(a.Devices) != len(b.Devices) {
		return false
	}
	keys := func(p hostreqkit.OpenRGBNativePolicy) []string {
		var result []string
		for _, d := range p.Devices {
			result = append(result, fmt.Sprintf("%q:%q:%q:%04x:%04x", d.Kind, d.StableID, d.HardwareID, d.UsagePage, d.Usage))
		}
		slices.Sort(result)
		return result
	}
	return slices.Equal(keys(a), keys(b))
}
