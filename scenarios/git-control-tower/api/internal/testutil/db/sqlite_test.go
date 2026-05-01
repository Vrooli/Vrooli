package db

import "testing"

func TestOpenSQLiteMemory(t *testing.T) {
	handle := OpenSQLiteMemory(t)
	if err := handle.Ping(); err != nil {
		t.Fatalf("ping memory sqlite: %v", err)
	}
}

func TestOpenSQLiteFile(t *testing.T) {
	handle := OpenSQLiteFile(t, "fixture.db")
	if err := handle.Ping(); err != nil {
		t.Fatalf("ping file sqlite: %v", err)
	}
}
