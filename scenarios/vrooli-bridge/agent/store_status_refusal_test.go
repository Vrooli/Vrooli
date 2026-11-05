package main

import (
	"errors"
	"testing"
)

func TestStoreStatusRefusalReadsTheStatusDocument(t *testing.T) {
	open := []byte(`{"active": true, "initialized": true, "unlocked": true, "entries": 0}`)
	if refusal, refused := storeStatusRefusal(open, nil); refused {
		t.Fatalf("an unlocked store was refused as %q", refusal.State)
	}
	locked := []byte("note: checking\n{\"initialized\": true, \"unlocked\": false}")
	if refusal, refused := storeStatusRefusal(locked, nil); !refused || refusal.State != "locked" {
		t.Fatalf("locked store: refused=%v state=%q", refused, refusal.State)
	}
	absent := []byte(`{"initialized": false, "unlocked": false}`)
	if refusal, refused := storeStatusRefusal(absent, nil); !refused || refusal.State != "uninitialized" {
		t.Fatalf("absent store: refused=%v state=%q", refused, refusal.State)
	}
	if refusal, refused := storeStatusRefusal([]byte("Runtime error: store locked"), errors.New("exit 1")); !refused || refusal.State != "locked" {
		t.Fatalf("text-only locked: refused=%v state=%q", refused, refusal.State)
	}
}
