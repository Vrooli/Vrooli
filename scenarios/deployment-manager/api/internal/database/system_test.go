package database

import (
	"strings"
	"testing"
)

func TestSystemSchemaContainsNoTables(t *testing.T) {
	if strings.Contains(strings.ToLower(Schema()), "create table") {
		t.Fatal("system schema must not own domain tables")
	}
}
