package runmodel

import "testing"

func TestSchemaIsAvailable(t *testing.T) {
	if Schema() == "" {
		t.Fatal("run-model schema must be embedded")
	}
}
