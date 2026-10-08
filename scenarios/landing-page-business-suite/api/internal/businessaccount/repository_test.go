package businessaccount

import (
	"context"
	"testing"
)

func TestMemoryRepositoryResolvesOnlyMemberSelectedAccount(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	first, err := repo.ResolveForUser(ctx, "user-1", "one@example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateForUser(ctx, "user-1", "one@example.com", "Second business")
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := repo.ListForUser(ctx, "user-1", "one@example.com")
	if err != nil || len(accounts) != 2 {
		t.Fatalf("accounts = %#v, err = %v", accounts, err)
	}
	selected, err := repo.ResolveForUser(ctx, "user-1", "one@example.com", second.ID)
	if err != nil || selected.ID != second.ID {
		t.Fatalf("selected = %#v, err = %v", selected, err)
	}
	if _, err := repo.ResolveForUser(ctx, "user-2", "two@example.com", second.ID); err != ErrNotMember {
		t.Fatalf("cross-user selection error = %v, want %v", err, ErrNotMember)
	}
	if first.ID == second.ID {
		t.Fatal("default and created accounts must be distinct")
	}
}
