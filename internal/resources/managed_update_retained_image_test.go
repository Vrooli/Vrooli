package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedUpdateRetainsVerifiedImageThroughOwnedStream(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
	path := filepath.Join(root, "running-image")
	if err := os.WriteFile(path, []byte("verified-serving-image"), 0o700); err != nil {
		t.Fatal(err)
	}
	image, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	digest := updateTestDigest("verified-serving-image")
	c := NewController(root, root)
	first, err := retainManagedUpdateFile(context.Background(), c, "fixture", "old-release", digest, image)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path == path || first.Version != "old-release" || first.SHA256 != digest {
		t.Fatal("retention selected the desired path instead of retaining the witnessed image")
	}
	if err := binaryfetch.VerifyArtifact(first.Path, "file", digest); err != nil {
		t.Fatal(err)
	}
	second, err := retainManagedUpdateFile(context.Background(), c, "fixture", "old-release", digest, image)
	if err != nil || !reflect.DeepEqual(second, first) {
		t.Fatalf("repeated retention changed identity: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := retainManagedUpdateFile(ctx, c, "fixture", "old-release", digest, image); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled retention acknowledged: %v", err)
	}
	if err := binaryfetch.VerifyArtifact(first.Path, "file", digest); err != nil {
		t.Fatal("canceled retention destroyed the verified old image")
	}
}

func TestManagedUpdateInvalidImageCannotReplaceRetainedBytes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
	digest := updateTestDigest("verified-serving-image")
	destination := filepath.Join(root, "artifacts", "fixture", "old-release", ".rollback", digest)
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("verified-serving-image"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "wrong-image")
	if err := os.WriteFile(source, []byte("different-image"), 0o700); err != nil {
		t.Fatal(err)
	}
	image, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	if _, err := retainManagedUpdateFile(context.Background(), NewController(root, root), "fixture", "old-release", digest, image); err == nil {
		t.Fatal("wrong process image was retained as the rollback release")
	}
	if err := binaryfetch.VerifyArtifact(destination, "file", digest); err != nil {
		t.Fatal("rejected image overwrote existing rollback bytes")
	}
}

func TestManagedUpdateRollbackRetentionKeepsServingIdentityAndSupervisorRecord(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("native running-image retention is currently qualified on Linux")
	}
	if testing.Short() {
		t.Skip("starts a fixed owned fixture process")
	}
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "storage"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
	m, err := manifestpkg.Load(filepath.Join(projectRoot(t), "resources", "cloudflared", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.Name = "rollback-retention-live-fixture"
	dir := filepath.Join(root, "resources", m.Name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	sup, _, err := managedServiceSupervisorFor(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	sup.place = nil // This fixture does not adopt or affect a host service unit.
	digest, err := binaryfetch.FileDigestContext(context.Background(), os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	state, err := sup.Start(os.Args[0], resourcedeployment.ServiceArtifact{Path: "fixture", Version: "old-release", SHA256: digest}, []string{"-test.run=^TestManagedServiceFixtureProcess$", "--"}, []string{"VROOLI_MANAGED_SERVICE_FIXTURE=1"}, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sup.Stop(ctx); err != nil {
			t.Errorf("fixed serving fixture cleanup: %v", err)
		}
	})
	before, err := os.ReadFile(sup.statePath)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewController(root, root).RetainManagedServiceRollbackArtifact(context.Background(), m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Version != "old-release" || artifact.SHA256 != digest || artifact.Version == m.ManagedService.Artifact.Version {
		t.Fatal("rollback retained the desired release instead of the serving release")
	}
	if _, running, err := sup.Status(); err != nil || !running {
		t.Fatalf("backup interrupted serving fixture: %v", err)
	}
	after, err := os.ReadFile(sup.statePath)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatal("retention silently adopted or changed the supervisor's launch identity")
	}
	if state.PID <= 0 {
		t.Fatal("serving fixture identity missing")
	}
}

func TestManagedUpdateCurrentHostRollbackRetentionProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_RETAIN_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_RETAIN_PROBE=1 to retain the exact current cloudflared image without service mutation")
	}
	if runtime.GOOS != "linux" {
		t.Fatal("current-host native retention requires the qualified Linux backend")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	c := NewController(projectRoot(t), home)
	sup, _, err := managedServiceSupervisorFor("cloudflared")
	if err != nil {
		t.Fatal(err)
	}
	before, running, err := sup.Status()
	if err != nil || !running {
		t.Fatalf("current serving connector is not owned: %v", err)
	}
	record, err := os.ReadFile(sup.statePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	artifact, err := c.RetainManagedServiceRollbackArtifact(ctx, "cloudflared")
	if err != nil {
		t.Fatal(err)
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, artifact.Path, "file", before.ArtifactSHA256); err != nil {
		t.Fatal(err)
	}
	after, running, err := sup.Status()
	if err != nil || !running || !reflect.DeepEqual(after, before) {
		t.Fatalf("retention changed serving ownership: %v", err)
	}
	afterRecord, err := os.ReadFile(sup.statePath)
	if err != nil || !reflect.DeepEqual(afterRecord, record) {
		t.Fatal("retention modified the serving supervisor record")
	}
	if report := os.Getenv("VROOLI_MANAGED_UPDATE_RETAIN_REPORT"); report != "" {
		body, err := json.MarshalIndent(map[string]any{"retained": artifact, "servingPid": before.PID, "servingInstance": before.InstanceID, "servingVersion": before.ArtifactVersion, "servingRecordUnchanged": true, "launchContractAdopted": false, "updateAdmitted": false}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(report, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("retained serving cloudflared %s with verified checksum; PID %d and supervisor record unchanged; launch contract not adopted", before.ArtifactVersion, before.PID)
}
