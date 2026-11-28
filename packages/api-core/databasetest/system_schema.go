package databasetest

import (
	"context"
	"strings"
	"testing"

	apidb "github.com/vrooli/api-core/database"
)

// AssertEmptySystemSchema verifies a scenario's system schema remains empty
// and that the canonical schema bootstrap safely accepts it more than once.
// Domain-owned schema behavior still belongs in the domain package's tests.
func AssertEmptySystemSchema(t *testing.T, systemSchema func() string) {
	t.Helper()
	if body := strings.TrimSpace(stripSQLComments(systemSchema())); body != "" {
		t.Fatalf("system schema must remain empty unless a cross-cutting owner is approved; got:\n%s", body)
	}

	database := NewSQLite(t)
	provider := apidb.SchemaProviderFunc(systemSchema)
	for attempt := 1; attempt <= 2; attempt++ {
		if err := apidb.EnsureSchemas(context.Background(), database, provider); err != nil {
			t.Fatalf("ensure system schema (attempt %d): %v", attempt, err)
		}
	}
}

func stripSQLComments(sql string) string {
	var body strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	return body.String()
}
