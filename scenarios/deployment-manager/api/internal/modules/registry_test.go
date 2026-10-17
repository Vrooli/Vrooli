package modules

import "testing"

func TestAllSchemasRegistersEveryOwnedDomain(t *testing.T) {
	schemas := AllSchemas()
	if len(schemas) != 6 {
		t.Fatalf("schema providers = %d, want 6", len(schemas))
	}
	for i, schema := range schemas {
		if schema == nil {
			t.Fatalf("schema provider %d is nil", i)
		}
	}
}
