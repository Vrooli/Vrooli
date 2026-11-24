package repository

import (
	"context"
	"strings"
	"testing"

	"lifestyle-dashboard/domain"
)

func TestDomainReadsRejectMalformedStoredCapabilities(t *testing.T) {
	for _, read := range []struct {
		name string
		call func(*SQLiteDomainRepository) error
	}{
		{
			name: "get by name",
			call: func(repo *SQLiteDomainRepository) error {
				_, err := repo.GetByName(context.Background(), "broken-domain")
				return err
			},
		},
		{
			name: "list",
			call: func(repo *SQLiteDomainRepository) error {
				_, err := repo.List(context.Background())
				return err
			},
		},
	} {
		t.Run(read.name, func(t *testing.T) {
			db := setupTestDB(t)
			t.Cleanup(func() { _ = db.Close() })
			repo := NewSQLiteDomainRepository(db)
			if err := repo.Upsert(context.Background(), &domain.Domain{Name: "broken-domain", DisplayName: "Broken"}); err != nil {
				t.Fatalf("seed domain: %v", err)
			}
			if _, err := db.Exec(`UPDATE domains SET capabilities = ? WHERE name = ?`, `{"events":`, "broken-domain"); err != nil {
				t.Fatalf("corrupt stored capabilities: %v", err)
			}

			err := read.call(repo)
			if err == nil || !strings.Contains(err.Error(), "broken-domain") {
				t.Fatalf("read malformed stored capabilities error = %v, want an error identifying broken-domain", err)
			}
		})
	}
}
