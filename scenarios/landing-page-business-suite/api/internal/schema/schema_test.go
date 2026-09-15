package schema

import (
	"strings"
	"testing"
)

func TestSystemSchemaIsDeclarative(t *testing.T) {
	t.Parallel()

	sql := strings.ToLower(System())
	if strings.Contains(sql, "alter table") || strings.Contains(sql, "drop table") {
		t.Fatal("system schema must be declarative; data/schema migrations belong in one-shot operator scripts")
	}
}
