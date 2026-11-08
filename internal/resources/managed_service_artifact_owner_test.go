package resources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
)

func TestManagedServiceArtifactOwnerStagesVerifiesAndNeverStarts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; _, _ = w.Write(body) }))
	defer server.Close()
	m := managedServiceTestManifest(fmt.Sprintf("%x", sha256.Sum256(body)))
	m.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: server.URL, SHA256: m.ManagedService.Artifact.SHA256, Layout: "file", Archive: "none"}}}
	m.ManagedService.ProviderPolicy.AllowedModes = m.ManagedService.ProviderPolicy.AllowedModes[:1]
	m.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	writeResourceManifest(t, root, m.Name, m)
	c := NewController(root, filepath.Join(root, "home"))
	ctx := context.Background()
	missing, err := c.InspectManagedServiceArtifact(ctx, m.Name)
	if err == nil || missing.Version != "1.0.0" {
		t.Fatalf("missing artifact must retain target intent: %+v %v", missing, err)
	}
	if _, err := c.StageManagedServiceArtifact(ctx, m.Name, "0.9.0"); err == nil || calls != 0 {
		t.Fatalf("changed admitted release had effects: calls=%d err=%v", calls, err)
	}
	staged, err := c.StageManagedServiceArtifact(ctx, m.Name, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if staged.Version != "1.0.0" || staged.SHA256 != m.ManagedService.Artifact.SHA256 || calls != 1 {
		t.Fatalf("staging: %+v calls=%d", staged, calls)
	}
	if _, err := c.StageManagedServiceArtifact(ctx, m.Name, "1.0.0"); err != nil || calls != 1 {
		t.Fatalf("valid reuse: calls=%d err=%v", calls, err)
	}
	sup, _, err := managedServiceSupervisorFor(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if state, running, err := sup.Status(); err != nil || running || state.PID != 0 {
		t.Fatalf("staging changed process lifetime: %+v running=%v err=%v", state, running, err)
	}
	if err := os.WriteFile(staged.Path, []byte("corrupt"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := c.InspectManagedServiceArtifact(ctx, m.Name); err == nil {
		t.Fatal("tampered staged artifact accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.StageManagedServiceArtifact(canceled, m.Name, "1.0.0"); err != context.Canceled || calls != 1 {
		t.Fatalf("canceled staging had effects: calls=%d err=%v", calls, err)
	}
	for _, name := range []string{"../escape", "/escape", "..", "a\\b"} {
		if _, err := c.StageManagedServiceArtifact(ctx, name, "1.0.0"); err == nil {
			t.Fatalf("unsafe resource name accepted: %q", name)
		}
	}
}

func TestManagedServiceStatusPreservesVerifiedRunningReleaseWhenDesiredChanges(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
	t.Setenv("VROOLI_MANAGED_SERVICE_FIXTURE", "1")
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	m := managedServiceTestManifest(sum)
	binary := filepath.Join(root, "resources", m.Name, "bin", "service")
	if err := os.MkdirAll(filepath.Dir(binary), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, body, 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c := NewController(root, filepath.Join(root, "home"))
	d := managedServiceDriver{}
	item := Resource{Name: m.Name}
	if err := d.Run(ctx, c, item, m, "start", nil, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Run(context.Background(), c, item, m, "stop", nil, io.Discard, io.Discard) })
	sup, _, err := managedServiceSupervisorFor(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	before, running, err := sup.Status()
	if err != nil || !running {
		t.Fatalf("fixture not running: %v", err)
	}
	// A new declared version with no acquired bytes is update intent, not downtime.
	m.ManagedService.Artifact.Version = "2.0.0"
	m.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/new-release", SHA256: sum, Layout: "file", Archive: "none"}}}
	for _, fast := range []bool{true, false} {
		got, err := d.Status(ctx, c, item, m, fast)
		if err != nil || !got.Running || got.Healthy == nil || !*got.Healthy || got.Serving == nil || !*got.Serving || got.StatusCode != StatusCodeOK {
			t.Fatalf("working old release hidden: %+v %v", got, err)
		}
		if got.DeclaredArtifactVersion != "2.0.0" || got.RunningArtifactVersion != "1.0.0" || got.StagedArtifactVersion != "" || !got.ArtifactUpdateAvailable || got.ArtifactStagingError == "" {
			t.Fatalf("update evidence lost: %+v", got)
		}
	}
	after, running, err := sup.Status()
	if err != nil || !running || after.PID != before.PID || after.InstanceID != before.InstanceID {
		t.Fatalf("status changed serving identity: %+v %v", after, err)
	}
	// A failed integrity proof of the retained image must still be visible.
	changed := before
	changed.ArtifactSHA256 = fmt.Sprintf("%064x", 1)
	if err := sup.writeState(changed); err != nil {
		t.Fatal(err)
	}
	got, err := d.Status(ctx, c, item, m, true)
	if err != nil || got.Healthy == nil || *got.Healthy || got.ProbeError == "" {
		t.Fatalf("unverified running image accepted: %+v %v", got, err)
	}
	if err := sup.writeState(before); err != nil {
		t.Fatal(err)
	}
}
