package discovery

import cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"

// TargetSuggestionIDForTest exposes the unexported stable-id derivation to the
// external _test package so tests can pre-seed a dismissal by the exact id the
// service will compute.
func TargetSuggestionIDForTest(locator string) string { return targetSuggestionID(locator) }

// FilterResourcesForTest exposes the typed `vrooli resource list` filter so the
// external _test package can table-test which resources are kept.
func FilterResourcesForTest(resources []*cliv1.Resource) []ResourceRef {
	return filterEnabledExternalCLI(resources)
}

// ResolveBaseForTest exposes durable_data base resolution to tests.
func ResolveBaseForTest(base, home string) (string, bool) { return resolveBase(base, home) }

// LoadDurableDataForTest exposes manifest durable_data decoding to tests.
func LoadDurableDataForTest(manifestPath string) *DurableData { return loadDurableData(manifestPath) }
