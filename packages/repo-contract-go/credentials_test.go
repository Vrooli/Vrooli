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

func TestValidateCredentialDescriptorUniqueness(t *testing.T) {
	err := ValidateCredentialDescriptorUniqueness([]byte(`{"credentials":{"descriptors":[{"logical_id":"vrooli/demo"},{"logical_id":"vrooli/demo"}]}}`), "resource.json")
	if err == nil || !strings.Contains(err.Error(), "resource.json declares credential vrooli/demo:value more than once") {
		t.Fatalf("ValidateCredentialDescriptorUniqueness() error = %v", err)
	}
}

func TestFindCredentialDescriptorDuplicatesIgnoresConsumerRegistrations(t *testing.T) {
	data := []byte(`{"credentials":{"descriptors":[{"logical_id":"vrooli/demo","field":"token"}],"consumers":[{"logical_id":"vrooli/demo","field":"token","kind":"dynamic","consumer":"session","source_ref":"api/session.go:1"},{"address_pattern":"vrooli/demo:token","kind":"environment","consumer":"compatibility","source_ref":"api/compat.go:1"}]}}`)
	duplicates, err := FindCredentialDescriptorDuplicates(data)
	if err != nil {
		t.Fatalf("FindCredentialDescriptorDuplicates() error = %v", err)
	}
	if len(duplicates) != 0 {
		t.Fatalf("consumer registration was treated as a duplicate declaration: %+v", duplicates)
	}
}

func TestFindCredentialDescriptorDuplicatesMatchesBridgeManifestShape(t *testing.T) {
	data := []byte(`{"credentials":{"descriptors":[{"logical_id":"vrooli/device-sync-hub","field":"bridge-origin-device-token"},{"logical_id":"vrooli/device-sync-hub","field":"bridge-target-device-token"}],"consumers":[{"logical_id":"vrooli/device-sync-hub","field":"bridge-origin-device-token"},{"address_pattern":"vrooli/device-sync-hub:bridge-origin-device-token"},{"logical_id":"vrooli/device-sync-hub","field":"bridge-target-device-token"}]}}`)
	duplicates, err := FindCredentialDescriptorDuplicates(data)
	if err != nil {
		t.Fatalf("FindCredentialDescriptorDuplicates() error = %v", err)
	}
	if len(duplicates) != 0 {
		t.Fatalf("Bridge-shaped consumer registrations were treated as declarations: %+v", duplicates)
	}
}

func TestFindCredentialDescriptorDuplicatesMatchesTunnelManagerShape(t *testing.T) {
	data := []byte(`{"credentials":{"descriptors":[{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-account-id"},{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-tunnel-id"},{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-api-token"},{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-connector-token"}],"consumers":[{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-api-token"},{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-account-id"},{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-tunnel-id"},{"logical_id":"vrooli/tunnel-manager","field":"cloudflare-connector-token"}]}}`)
	duplicates, err := FindCredentialDescriptorDuplicates(data)
	if err != nil {
		t.Fatalf("FindCredentialDescriptorDuplicates() error = %v", err)
	}
	if len(duplicates) != 0 {
		t.Fatalf("tunnel-manager consumer registrations were treated as declarations: %+v", duplicates)
	}
}
