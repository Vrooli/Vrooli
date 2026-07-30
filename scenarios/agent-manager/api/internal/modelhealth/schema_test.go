package modelhealth

import "testing"

func TestSchemaIsAvailable(t *testing.T) {
	if Schema() == "" {
		t.Fatal("model-health schema must be embedded")
	}
}
