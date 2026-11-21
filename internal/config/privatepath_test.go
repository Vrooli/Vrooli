package config

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPrivateWriterPublishesAndPreservesPriorOnProducerFailure(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "owned", "record.json")
	if err := WritePrivateFileDurable(path, []byte("original"), 0o600, PrivatePathPolicy{}); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("bounded producer failed")
	err = WritePrivateFileDurableStream(path, 0o600, PrivatePathPolicy{}, func(w io.Writer) error { w.Write([]byte("partial")); return failure })
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "original" {
		t.Fatalf("producer failure changed published intent: %q %v", data, err)
	}
	if err = RemovePrivateFileDurable(path, PrivatePathPolicy{}); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("retired intent remains selected", err)
	}
}

func TestPrivateWriterRejectsAliasedOrPublicTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACL behavior requires native Windows fixtures")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "intent")
	if err := os.WriteFile(path, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateFileDurable(path, []byte("new"), 0o600, PrivatePathPolicy{}); err == nil {
		t.Fatal("public intent overwritten without review")
	}
	target := filepath.Join(root, "link")
	if err := os.Symlink(path, target); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateFileDurable(target, []byte("new"), 0o600, PrivatePathPolicy{}); err == nil {
		t.Fatal("aliased intent overwritten")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "foreign" {
		t.Fatal("foreign target changed")
	}
}

func TestPrivateReadOnlyRuntimeCannotOwnAdmission(t *testing.T) {
	for _, p := range []PrivatePathPolicy{{RuntimeReadOnly: true}, {RuntimeTraverseOnly: true}} {
		if err := verifyPrivateOwner("runtime", "management", "runtime", "system", p); err == nil {
			t.Fatal("runtime ownership defeats its purported read-only admission grant")
		}
		if err := verifyPrivateOwner("management", "management", "runtime", "system", p); err != nil {
			t.Fatal("management-owned admission refused", err)
		}
	}
	if err := verifyPrivateOwner("runtime", "runtime", "management", "system", PrivatePathPolicy{}); err != nil {
		t.Fatal("explicit service-owned mutable runtime data refused", err)
	}
	if err := verifyPrivateOwner("foreign", "management", "runtime", "system", PrivatePathPolicy{}); err == nil {
		t.Fatal("foreign-owned state admitted")
	}
}

func TestPrivateWriterSerializesConcurrentPublishers(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "shared", "intent.json")
	var writers sync.WaitGroup
	errors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		writers.Add(1)
		go func(i int) {
			defer writers.Done()
			errors <- WritePrivateFileDurable(path, []byte(strings.Repeat(string(rune('a'+i)), 16<<10)), 0o600, PrivatePathPolicy{})
		}(i)
	}
	writers.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal("concurrent publisher lost its own durable identity", err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) != 16<<10 {
		t.Fatal("concurrent publication was torn", err)
	}
	for _, b := range data {
		if b != data[0] {
			t.Fatal("mixed publisher bodies became visible")
		}
	}
}

func TestPrivateOperationLockHonorsCancellationAndRemainsReusable(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "private", "operation.lock")
	release, err := AcquirePrivateFileLock(t.Context(), path, PrivatePathPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if second, err := AcquirePrivateFileLock(ctx, path, PrivatePathPolicy{}); !errors.Is(err, context.DeadlineExceeded) {
		if second != nil {
			second()
		}
		t.Fatal("queued operation ignored cancellation", err)
	}
	release()
	third, err := AcquirePrivateFileLock(t.Context(), path, PrivatePathPolicy{})
	if err != nil {
		t.Fatal("cancelled waiter stranded native lock", err)
	}
	third()
}
