package settings

import (
	"strings"
	"testing"
)

func TestSchema_ContainsProviderRoutingTables(t *testing.T) {
	schema := Schema()
	if strings.TrimSpace(schema) == "" {
		t.Fatal("settings schema must be embedded")
	}
	for _, table := range []string{"provider", "voice"} {
		if !strings.Contains(strings.ToLower(schema), table) {
			t.Fatalf("schema missing %q contract", table)
		}
	}
}
