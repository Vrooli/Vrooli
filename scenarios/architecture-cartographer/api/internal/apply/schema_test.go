package apply

import (
	"strings"
	"testing"
)

func TestSchema_NotEmpty(t *testing.T) {
	if Schema() == "" {
		t.Fatal("apply.Schema() returned empty")
	}
}

func TestSchema_ContainsPlansTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS apply_plans") {
		t.Fatal("apply.Schema() missing apply_plans table")
	}
}
