package modules

import "testing"

func TestAllSchemasRegistersEveryOwnedDomain(t *testing.T) {
	schemas := AllSchemas()
	if len(schemas) != 5 {
		t.Fatalf("schema providers = %d, want 5", len(schemas))
	}
	for i, schema := range schemas {
		if schema == nil {
			t.Fatalf("schema provider %d is nil", i)
		}
	}
}
