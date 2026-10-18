package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestModelCacheCLIRejectsIncompleteOrUnboundedRequests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(path, bytes.Repeat([]byte(" "), 65537), 0600); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		mode string
		args []string
	}{
		{"apply", nil}, {"pin", nil}, {"preview", []string{"extra"}},
		{"pin", []string{"--profile", "heartmula-3b", "--pinned=maybe"}},
		{"apply", []string{"--preview", path}},
	} {
		if err := cacheCommand(item.mode, item.args); err == nil {
			t.Fatal("invalid request accepted", item)
		}
	}
}
