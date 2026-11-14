package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/artifactlock"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestDataArtifactCopyFailurePreservesInstalledBytes(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "weights.bin")
	if err := os.WriteFile(destination, []byte("verified installed weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A directory opens successfully but fails during the streamed read.
	if err := copyDataArtifact(t.TempDir(), destination); err == nil {
		t.Fatal("expected the failed source read to reject publication")
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "verified installed weights" {
		t.Fatalf("installed bytes changed after failure: %q, %v", got, err)
	}
	staging, err := filepath.Glob(filepath.Join(directory, ".data-artifact-*"))
	if err != nil || len(staging) != 0 {
		t.Fatalf("failed copy left temporary files: %v, %v", staging, err)
	}
}

func TestDataGroupsDoNotDownloadUnusedOrRetiredProfilesOnOrdinarySetup(t *testing.T) {
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	root := t.TempDir()
	controller := NewController(root, t.TempDir())
	var mu sync.Mutex
	calls := map[string]int{}
	body := []byte("pinned-model-fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls[r.URL.Path]++
		mu.Unlock()
		w.Write(body)
	}))
	defer server.Close()
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.Name = "data-groups-fixture"
	manifest.ManagedService.MinimumDataGroups = 1
	manifest.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate}
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	for _, group := range []string{"first", "second"} {
		manifest.ManagedService.DataArtifacts = append(manifest.ManagedService.DataArtifacts, resourcedeployment.ManagedServiceDataArtifact{Name: group, SizeBytes: int64(len(body)), Path: "models/" + group + "/weights.bin", Group: group, AcquisitionMode: "on-demand", Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: server.URL + "/" + group, SHA256: fmt.Sprintf("%x", sha256.Sum256(body)), Layout: "file"}}}})
	}
	writeResourceManifest(t, root, manifest.Name, manifest)
	if err := verifyManagedServiceDataGroupAvailability(t.Context(), manifest); err == nil || !strings.Contains(err.Error(), "resource acquisition data") {
		t.Fatalf("missing group did not fail promptly: %v", err)
	}
	if err := ensureManagedServiceDataArtifacts(t.Context(), controller, manifest); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatal("ordinary setup downloaded optional data", calls)
	}
	count, err := controller.AcquireManagedServiceDataGroup(t.Context(), manifest.Name, "first")
	if err != nil || count != 1 {
		t.Fatalf("acquire first=%d %v", count, err)
	}
	if err := verifyManagedServiceDataGroupAvailability(t.Context(), manifest); err != nil {
		t.Fatal("acquired group unavailable", err)
	}
	paths, err := resourceStoragePaths(manifest.Name)
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(paths.DataDir, "models/first/weights.bin")
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
	if calls["/first"] != 1 || calls["/second"] != 0 {
		t.Fatal("downloaded unselected group", calls)
	}
	if err := os.WriteFile(first, []byte("truncated"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyManagedServiceDataGroupAvailability(t.Context(), manifest); err == nil {
		t.Fatal("truncated data group passed quick preflight")
	}

	// A retention sweep removes an unused bundle. The next normal start/install
	// must leave it absent rather than undoing the disk policy with a download.
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedServiceDataArtifacts(t.Context(), controller, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("ordinary setup restored retired data", err)
	}
	count, err = controller.AcquireManagedServiceDataGroup(t.Context(), manifest.Name, "second")
	if err != nil || count != 1 || calls["/first"] != 1 || calls["/second"] != 1 {
		t.Fatalf("selected acquisition=%d %v calls=%v", count, err, calls)
	}
	if _, err := controller.AcquireManagedServiceDataGroup(t.Context(), manifest.Name, "unknown"); err == nil {
		t.Fatal("unknown group accepted")
	}
	release, err := artifactlock.AcquireContext(t.Context(), "resource-data:"+manifest.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := controller.AcquireManagedServiceDataGroup(ctx, manifest.Name, "first"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("group bypassed shared data fence: %v", err)
	}
	lifecycleCtx, stopLifecycle := context.WithTimeout(t.Context(), time.Second)
	defer stopLifecycle()
	unlock, err := acquireManagedServiceLifecycleLock(lifecycleCtx, manifest.Name)
	if err != nil {
		t.Fatal("failed data acquisition stranded lifecycle ownership", err)
	}
	unlock()

}
