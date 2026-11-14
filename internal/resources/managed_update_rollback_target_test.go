package resources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagedUpdateRetainedRollbackRequiresQualifiedTerminalReceipt(t *testing.T) {
	ctx := context.Background()
	u := updateTestInitial(t)
	path := filepath.Join(t.TempDir(), "retained-image")
	content := []byte("retained launch bytes for checksum boundary")
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(content))
	u.Rollback.Artifact.Path, u.Rollback.Artifact.SHA256 = path, digest
	u.Rollback.Manifest.ManagedService.Artifact.SHA256 = digest
	u.Rollback.Digest = u.Rollback.digest()
	u.Original.ArtifactPath, u.Original.ArtifactSHA256 = path, digest
	u.RollbackQualificationRequired = true
	s := updateTestStore(t, u)
	if _, _, err := s.Create(ctx, u, false); err != nil {
		t.Fatal(err)
	}
	opts := ManagedServiceUpdateAdmission{RollbackOf: u.RequestID, ExpectedVersion: u.Rollback.Artifact.Version, ExpectedSHA256: digest}
	c := NewController(t.TempDir(), t.TempDir())
	if _, err := c.managedUpdateAdmissionTarget(ctx, u.Target.Manifest, s, opts); err == nil {
		t.Fatal("active receipt admitted a rollback target")
	}
	u.RollbackProbe.Intent = updateTestIntent("qualification", u.Rollback)
	s.now = func() time.Time { return u.UpdatedAt.Add(time.Second) }
	var err error
	u, err = s.Commit(ctx, u, UpdateLaunchRollbackProbe)
	if err != nil {
		t.Fatal(err)
	}
	updateTestObserveStarted(&u.RollbackProbe, u.Rollback, u.UpdatedAt)
	u, err = s.Commit(ctx, u, UpdateRollbackProbeStarted)
	if err != nil {
		t.Fatal(err)
	}
	u.RollbackProbe.Proof = updateTestProof(*u.RollbackProbe.State, u.UpdatedAt)
	u, err = s.Commit(ctx, u, UpdateRollbackProbeVerified)
	if err != nil {
		t.Fatal(err)
	}
	u.Error = "fixture failure before target launch"
	u, err = s.Commit(ctx, u, UpdateFailure)
	if err != nil {
		t.Fatal(err)
	}
	u.OriginalProof = updateTestProof(u.Original, u.UpdatedAt)
	u, err = s.Commit(ctx, u, UpdateOriginalVerified)
	if err != nil {
		t.Fatal(err)
	}
	u.OriginalProof = updateTestProof(u.Original, u.UpdatedAt)
	u.CandidateQuiescent, u.RollbackProbeQuiescent = true, true
	u, err = s.Commit(ctx, u, UpdateFinishPreserved)
	if err != nil {
		t.Fatal(err)
	}
	target, err := c.managedUpdateAdmissionTarget(ctx, u.Target.Manifest, s, opts)
	if err != nil || target.Digest != u.Rollback.Digest {
		t.Fatalf("qualified retained target: %+v %v", target.Artifact, err)
	}
	wrong := opts
	wrong.ExpectedSHA256 = updateTestDigest("different")
	if _, err := c.managedUpdateAdmissionTarget(ctx, u.Target.Manifest, s, wrong); err == nil {
		t.Fatal("rollback ignored explicitly selected image")
	}
	if err := os.WriteFile(path, []byte("corrupted"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := c.managedUpdateAdmissionTarget(ctx, u.Target.Manifest, s, opts); err == nil {
		t.Fatal("retained rollback accepted changed bytes")
	}
}
