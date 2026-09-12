package runnerhealth

import "testing"

func TestSchemaIsAvailable(t *testing.T) {
	if Schema() == "" {
		t.Fatal("runner-health schema must be embedded")
	}
}
