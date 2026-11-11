package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"web-console/backends/codex"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
)

type archiveTestWriter struct{ err error }

func (archiveTestWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w archiveTestWriter) Close() error              { return w.err }

func TestManagedCodexArchiveRetiresRuntimeAndKeepsMetadata(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "close failure"}[failure], func(t *testing.T) {
			store := sessionstore.NewInMemory()
			id := "native-artifact"
			meta := sessionstore.Metadata{ID: id, LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeOwner: "generation", NativeThreadID: "thread", Created: time.Now().UTC()}
			if err := store.Save(t.Context(), meta); err != nil {
				t.Fatal(err)
			}
			var closeErr error
			if failure {
				closeErr = errors.New("transport close failed")
			}
			owner, err := codex.NewManagedOwner(codex.NewClient(archiveTestWriter{closeErr}, strings.NewReader("")), "generation", "thread")
			if err != nil {
				t.Fatal(err)
			}
			registry := newManagedCodexRegistry(store)
			registry.entries[id] = managedCodexEntry{session: sessionsH.Session{ID: id, NativeOwner: "generation", NativeThreadID: "thread"}, owner: owner}
			err = registry.Archive(t.Context(), id)
			if (err != nil) != failure {
				t.Fatalf("Archive error=%v", err)
			}
			_, retained := registry.entries[id]
			if retained != failure {
				t.Fatalf("runtime retry owner retained=%v want%v", retained, failure)
			}
			if _, err := store.Get(t.Context(), id); err != nil {
				t.Fatalf("archive deleted metadata: %v", err)
			}
			if !failure {
				if err := registry.Archive(t.Context(), id); err != nil {
					t.Fatalf("confirmed retirement retry failed: %v", err)
				}
			}
		})
	}
	if err := newManagedCodexRegistry(sessionstore.NewInMemory()).Archive(t.Context(), "unknown-after-restart"); err == nil {
		t.Fatal("missing registry owner claimed retired")
	}
}

func TestManagedCodexArchiveCancellationRetainsSameRetirement(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	owner, err := codex.NewManagedOwner(codex.NewClient(archiveTestWriter{}, reader), "generation", "thread")
	if err != nil {
		t.Fatal(err)
	}
	registry := newManagedCodexRegistry(sessionstore.NewInMemory())
	registry.entries["native"] = managedCodexEntry{session: sessionsH.Session{ID: "native"}, owner: owner}
	registry.eventSink = func(string, codex.NativeEvent) {}
	registry.watch("native", owner)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := registry.Archive(ctx, "native"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Archive=%v", err)
	}
	registry.markUnavailable("native", owner)
	registry.mu.RLock()
	entry, ok := registry.entries["native"]
	registry.mu.RUnlock()
	if !ok || entry.owner != owner || !entry.retiring {
		t.Fatal("cancellation discarded exact retiring runtime")
	}
	if _, _, err := registry.DeliverPrompt(t.Context(), "native", "generation", "thread", "", "must not submit"); err == nil {
		t.Fatal("retiring native owner accepted input")
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	retryCtx, retryCancel := context.WithTimeout(t.Context(), time.Second)
	defer retryCancel()
	if err := registry.Archive(retryCtx, "native"); err != nil {
		t.Fatal(err)
	}
	registry.mu.RLock()
	_, stillPresent := registry.entries["native"]
	registry.mu.RUnlock()
	if stillPresent {
		t.Fatal("completed retry retained live owner")
	}
}

func TestManagedCodexDeleteFailureRetainsRuntimeAndArtifacts(t *testing.T) {
	store := sessionstore.NewInMemory()
	id := "delete-failed-native"
	if err := store.Save(t.Context(), sessionstore.Metadata{ID: id, Created: time.Now().UTC(), LaunchMode: sessionstore.LaunchModeCodexAppServer}); err != nil {
		t.Fatal(err)
	}
	owner, err := codex.NewManagedOwner(codex.NewClient(archiveTestWriter{errors.New("close refused")}, strings.NewReader("")), "generation", "thread")
	if err != nil {
		t.Fatal(err)
	}
	registry := newManagedCodexRegistry(store)
	registry.entries[id] = managedCodexEntry{session: sessionsH.Session{ID: id}, owner: owner}
	if err := registry.Delete(t.Context(), id); err == nil {
		t.Fatal("Delete claimed success before retirement")
	}
	if _, ok := registry.entries[id]; !ok {
		t.Fatal("failed Delete discarded retry owner")
	}
	if _, err := store.Get(t.Context(), id); err != nil {
		t.Fatalf("failed Delete lost artifacts: %v", err)
	}
}
