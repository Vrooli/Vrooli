package resources

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
)

func TestHealthObservationDoesNotAuditRuntimeTreeOrAuthorizeLaunch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	t.Setenv("VROOLI_MANAGED_SERVICE_FIXTURE", "1")
	m := managedServiceTestManifest(strings.Repeat("0", 64))
	m.ManagedService.Artifact.Layout, m.ManagedService.Artifact.EntryPath = "dir", "service"
	artifact := filepath.Join(root, "resources", m.Name, filepath.FromSlash(m.ManagedService.Artifact.Path))
	if err := os.MkdirAll(artifact, 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "service"), body, 0o700); err != nil {
		t.Fatal(err)
	}
	aux := filepath.Join(artifact, "large-runtime-library")
	if err := os.WriteFile(aux, []byte("original runtime bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.ManagedService.Artifact.SHA256, err = binaryfetch.TreeDigest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	c := NewController(root, filepath.Join(root, "home"))
	health := c.ForHealthObservation()
	driver := managedServiceDriver{}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := driver.Run(ctx, c, Resource{Name: m.Name}, m, "start", nil, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = driver.Run(context.Background(), c, Resource{Name: m.Name}, m, "stop", nil, io.Discard, io.Discard)
	})
	supervisor, _, err := managedServiceSupervisorFor(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	before, running, err := supervisor.Status()
	if err != nil || !running {
		t.Fatalf("fixture ownership: %+v %v", before, err)
	}
	if err := os.WriteFile(aux, []byte("changed runtime bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, fast := range []bool{false, true} {
		observed, err := driver.Status(ctx, health, Resource{Name: m.Name}, m, fast)
		if err != nil || !observed.Running || observed.Healthy == nil || !*observed.Healthy || observed.ArtifactVerification != "not_evaluated" || observed.StagedArtifactVersion != "" {
			t.Fatalf("health observation made an integrity claim or lost serving state: %+v %v", observed, err)
		}
		audited, err := driver.Status(ctx, c, Resource{Name: m.Name}, m, fast)
		if err != nil || audited.Healthy == nil || *audited.Healthy || audited.ArtifactVerification != "failed" {
			t.Fatalf("default audit failed to detect changed bytes: %+v %v", audited, err)
		}
		if want := "restart it with `vrooli resource restart " + m.Name + "`"; !strings.Contains(audited.Message, want) {
			t.Fatalf("audit message %q does not name the remedy %q", audited.Message, want)
		}
	}
	after, running, err := supervisor.Status()
	if err != nil || !running || after.PID != before.PID || after.InstanceID != before.InstanceID {
		t.Fatalf("observation changed process ownership: %+v %v", after, err)
	}
	if err := driver.Run(ctx, health, Resource{Name: m.Name}, m, "stop", nil, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := driver.Run(ctx, health, Resource{Name: m.Name}, m, "start", nil, io.Discard, io.Discard); err == nil {
		t.Fatal("health-observation configuration bypassed launch checksum verification")
	}
}
