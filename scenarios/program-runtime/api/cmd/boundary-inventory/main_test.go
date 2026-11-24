package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicWriteReplacesCompleteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	want := []byte("complete-new-document\n")
	if err := atomicWrite(path, want, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("output=%q want %q", got, want)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "inventory.json" {
		t.Fatalf("temporary files remain after replacement: %+v", entries)
	}
}

func TestAtomicWriteConcurrentReadersSeeCompleteDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	old := []byte("complete-old-document\n")
	if err := os.WriteFile(path, old, 0644); err != nil {
		t.Fatal(err)
	}
	want := []byte("complete-new-document\n")
	stop := make(chan struct{})
	readErrors := make(chan error, 1)
	var readers sync.WaitGroup
	readers.Add(1)
	go func() {
		defer readers.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			got, err := os.ReadFile(path)
			if err != nil {
				select {
				case readErrors <- err:
				default:
				}
				return
			}
			if string(got) != string(old) && string(got) != string(want) {
				select {
				case readErrors <- fmt.Errorf("reader observed partial content %q", got):
				default:
				}
				return
			}
		}
	}()
	for i := 0; i < 50; i++ {
		if err := atomicWrite(path, want, nil); err != nil {
			close(stop)
			readers.Wait()
			t.Fatal(err)
		}
		if err := atomicWrite(path, old, nil); err != nil {
			close(stop)
			readers.Wait()
			t.Fatal(err)
		}
	}
	close(stop)
	readers.Wait()
	select {
	case err := <-readErrors:
		t.Fatal(err)
	default:
	}
}

func TestAtomicWriteDoesNotReplaceWhenVerificationFindsDrift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	err := atomicWrite(path, []byte("new"), func() error { return os.ErrInvalid })
	if err == nil {
		t.Fatal("expected verification failure")
	}
	got, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if string(got) != "old" {
		t.Fatalf("old output was replaced despite drift: %q", got)
	}
}

func TestCheckExportBytesDetectsCurrentAndStaleExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	expected := []byte("current\n")
	if err := os.WriteFile(path, expected, 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkExportBytes(path, expected); err != nil {
		t.Fatalf("current export rejected: %v", err)
	}
	if err := checkExportBytes(path, []byte("stale\n")); err == nil {
		t.Fatal("stale export accepted")
	}
}

func TestCheckExportVerifiesLiveInputsEvenWhenBytesMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	want := []byte("current document\n")
	if err := os.WriteFile(path, want, 0644); err != nil {
		t.Fatal(err)
	}
	called := false
	err := checkExport(path, want, func() error {
		called = true
		return os.ErrInvalid
	})
	if !called {
		t.Fatal("live input verifier was not called")
	}
	if err == nil {
		t.Fatal("matching bytes bypassed live source verification")
	}
}
