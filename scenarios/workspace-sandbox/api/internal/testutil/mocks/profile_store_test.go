package mocks

import (
	"errors"
	"testing"

	"workspace-sandbox/internal/config"
)

func TestFakeProfileStore_RoundTrip(t *testing.T) {
	store := NewFakeProfileStore(config.IsolationProfile{ID: "full"})
	got, err := store.Get("full")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "full" {
		t.Errorf("got %q, want full", got.ID)
	}
	if _, err := store.Get("nope"); err == nil {
		t.Error("Get(missing) should error")
	}
}

func TestFakeProfileStore_ListErr(t *testing.T) {
	store := NewFakeProfileStore()
	store.ListErr = errors.New("nope")
	if _, err := store.List(); err == nil {
		t.Error("ListErr should surface")
	}
}
