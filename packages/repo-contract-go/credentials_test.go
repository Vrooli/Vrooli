package repocontract

import (
	"strings"
	"testing"
)

func TestFindCredentialDescriptorDuplicates(t *testing.T) {
	data := []byte(`{"credentials":{"descriptors":[{"logical_id":"vrooli/demo","field":"api-key"},{"logical_id":"vrooli/demo","field":"api-key"},{"logical_id":"vrooli/demo"}]},"nested":{"logical_id":"vrooli/demo"}}`)
	duplicates, err := FindCredentialDescriptorDuplicates(data)
	if err != nil {
		t.Fatalf("FindCredentialDescriptorDuplicates() error = %v", err)
	}
	if len(duplicates) != 2 {
		t.Fatalf("duplicate count = %d, want 2: %+v", len(duplicates), duplicates)
	}
	if duplicates[0].LogicalID != "vrooli/demo" || duplicates[0].Field != "api-key" || duplicates[0].FirstPath != "/credentials/descriptors/0" || duplicates[0].DuplicatePath != "/credentials/descriptors/1" {
		t.Fatalf("first duplicate = %+v", duplicates[0])
	}
	if duplicates[1].Field != DefaultCredentialField || !strings.HasSuffix(duplicates[1].DuplicatePath, "/nested") {
		t.Fatalf("default-field duplicate = %+v", duplicates[1])
	}
}

func TestFindCredentialDescriptorDuplicatesIgnoresDistinctPairs(t *testing.T) {
	duplicates, err := FindCredentialDescriptorDuplicates([]byte(`{"a":{"logical_id":"vrooli/demo","field":"one"},"b":{"logical_id":"vrooli/demo","field":"two"}}`))
	if err != nil {
		t.Fatalf("FindCredentialDescriptorDuplicates() error = %v", err)
	}
	if len(duplicates) != 0 {
		t.Fatalf("distinct pairs reported as duplicates: %+v", duplicates)
	}
}

func TestFindCredentialDescriptorDuplicatesRejectsInvalidJSON(t *testing.T) {
	if _, err := FindCredentialDescriptorDuplicates([]byte("{")); err == nil {
		t.Fatal("invalid JSON did not return an error")
	}
}
