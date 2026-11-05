package modules

import "testing"

func TestAllSchemasRegistersEveryOwnedDomain(t *testing.T) {
	schemas := AllSchemas()
	// 7 owned domains: database, profiles, deployments, releases, evidence,
	// readiness, and the ask-don't-block release-decision pending-ask ledger.
	if len(schemas) != 7 {
		t.Fatalf("schema providers = %d, want 7", len(schemas))
	}
	for i, schema := range schemas {
		if schema == nil {
			t.Fatalf("schema provider %d is nil", i)
		}
	}
}
