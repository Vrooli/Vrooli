package repository

import (
	"database/sql"
	"testing"
)

func TestNewPostgresRepository(t *testing.T) {
	repo := NewPostgresRepository(nil)
	if repo == nil {
		t.Fatal("expected repository instance")
	}
	if repo.db != nil {
		t.Fatalf("expected nil db, got %#v", repo.db)
	}
}

func TestNewPostgresRepositoryWithDB(t *testing.T) {
	db := &sql.DB{}
	repo := NewPostgresRepository(db)
	if repo == nil {
		t.Fatal("expected repository instance")
	}
	if repo.db != db {
		t.Fatal("expected repository to keep provided db reference")
	}
}
