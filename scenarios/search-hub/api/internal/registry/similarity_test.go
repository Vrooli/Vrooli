package registry

import (
	"testing"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestSimilarDescriptorWarningsAreAdvisoryAndDeterministic(t *testing.T) {
	candidate := &registryv1.ProviderDescriptor{ProviderId: "new.records", ProviderGroup: "new", Type: "record", Description: "Search project records and decisions."}
	existing := []*registryv1.ProviderDescriptor{
		{ProviderId: "old.records", ProviderGroup: "old", Type: "record", Description: "Search project records and decisions."},
		{ProviderId: "different.code", ProviderGroup: "different", Type: "code", Description: "Compile unrelated source files."},
	}
	warnings := SimilarDescriptorWarnings(candidate, existing)
	if len(warnings) != 1 || warnings[0].ExistingProviderID != "old.records" || warnings[0].Similarity < 0.99 {
		t.Fatalf("warnings = %+v", warnings)
	}
}

func TestSimilarDescriptorWarningsIgnoreSelfAndDoNotBlock(t *testing.T) {
	candidate := &registryv1.ProviderDescriptor{ProviderId: "same.records", ProviderGroup: "same", Type: "record", Description: "Search records."}
	if got := SimilarDescriptorWarnings(candidate, []*registryv1.ProviderDescriptor{candidate}); len(got) != 0 {
		t.Fatalf("self warning = %+v", got)
	}
}
