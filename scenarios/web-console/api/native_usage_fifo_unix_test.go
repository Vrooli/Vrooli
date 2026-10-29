//go:build unix

package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestNativeUsageRejectsFIFO(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "sessions")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "pipe.jsonl")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		f, err := openNativeRollout(dir, fifo)
		if f != nil {
			f.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO admitted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO blocked open")
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE threads(id TEXT PRIMARY KEY, rollout_path TEXT); INSERT INTO threads VALUES ('one', ?)", fifo); err != nil {
		t.Fatal(err)
	}
	db.Close()
	result, err := readNativeUsage(context.Background(), home, nativeUsageRequest{ThreadIDs: []string{"one"}, Start: time.Now().Add(-time.Hour), End: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "partial" || result.Threads[0].Status != "unknown" || result.Total.Usage != nil || result.Total.Responses != 0 {
		t.Fatalf("FIFO reported usage: %+v", result)
	}
}
