package renderjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"device-studio/internal/jobs"
)

func TestExplicitCacheReuseReturnsOriginalVerifiedRenderAndRerenderStaysFresh(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	original := submitIntegrationJob(t, c, ctx)
	if _, err := c.Advance(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := c.Reconcile(ctx, original.ID)
	if err != nil || completed.State != jobs.Succeeded {
		t.Fatalf("source render did not complete: state=%s err=%v", completed.State, err)
	}
	output := completed.Attempts[0].Completion.Outputs[0]

	cached, created, receipt, err := c.SubmitWithReceipt(ctx, SubmitRequest{
		ProjectID: "project", ShotID: "shot", Revision: 1, IdempotencyKey: "cache-replay",
		OutputProfile: OutputProfilePNG, ReuseCachedSuccess: true,
	})
	if err != nil || created || cached.ID != original.ID || receipt == nil {
		t.Fatalf("unchanged opt-in request should reuse the verified succeeded render: job=%s created=%v receipt=%+v err=%v", cached.ID, created, receipt, err)
	}
	if receipt.SourceRenderID != original.ID || receipt.SourceAttemptID != completed.Attempts[0].ID || receipt.SHA256 != output.SHA256 || receipt.OutputKind != output.Kind {
		t.Fatalf("cache receipt did not identify the original output: %+v output=%+v", receipt, output)
	}
	if _, _, _, err := c.SubmitWithReceipt(ctx, SubmitRequest{
		ProjectID: "project", ShotID: "shot", Revision: 2, IdempotencyKey: "cache-replay",
		OutputProfile: OutputProfilePNG, ReuseCachedSuccess: true,
	}); !errors.Is(err, jobs.ErrIdempotencyConflict) {
		t.Fatalf("same cache idempotency key with a different immutable shot revision must conflict, got %v", err)
	}
	ops.mu.Lock()
	starts := ops.starts
	ops.mu.Unlock()
	if starts != 1 {
		t.Fatalf("cache hit started additional native work: starts=%d", starts)
	}

	rerender, rerenderCreated, err := c.Submit(ctx, SubmitRequest{
		ProjectID: "project", ShotID: "shot", Revision: 1, IdempotencyKey: "explicit-rerender",
		OutputProfile: OutputProfilePNG,
	})
	if err != nil || !rerenderCreated || rerender.ID == original.ID || rerender.State != jobs.Queued {
		t.Fatalf("ordinary explicit rerender must remain a fresh job: job=%s created=%v state=%s err=%v", rerender.ID, rerenderCreated, rerender.State, err)
	}

	changedShot := integrationResolved(c.media.(*integrationMedia))
	changedShot.Revision.Revision = 2
	changedShot.Revision.RecipeSHA256 = digestBytes([]byte("changed immutable shot revision"))
	c.shots = integrationShot{resolved: changedShot}
	changed, changedCreated, changedReceipt, err := c.SubmitWithReceipt(ctx, SubmitRequest{
		ProjectID: "project", ShotID: "shot", Revision: 2, IdempotencyKey: "cache-dependency-change",
		OutputProfile: OutputProfilePNG, ReuseCachedSuccess: true,
	})
	if err != nil || !changedCreated || changedReceipt != nil || changed.ID == original.ID {
		t.Fatalf("changed immutable shot revision must miss cache and admit a new job: job=%s created=%v receipt=%+v err=%v", changed.ID, changedCreated, changedReceipt, err)
	}
}

func TestCacheReuseMissesWhenSucceededOutputIsTamperedOrMissing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(context.Context, *Coordinator, string) error
	}{
		{name: "tampered", mutate: func(ctx context.Context, c *Coordinator, key string) error {
			return c.blobs.Put(ctx, key, bytes.NewReader([]byte("tampered output")), "image/png")
		}},
		{name: "missing", mutate: func(ctx context.Context, c *Coordinator, key string) error {
			return c.blobs.Delete(ctx, key)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ops := &integrationOps{}
			c, ctx, _ := integrationCoordinator(t, ops, false)
			original := submitIntegrationJob(t, c, ctx)
			if _, err := c.Advance(ctx, original.ID); err != nil {
				t.Fatal(err)
			}
			completed, err := c.Reconcile(ctx, original.ID)
			if err != nil || completed.State != jobs.Succeeded {
				t.Fatalf("source render did not complete: state=%s err=%v", completed.State, err)
			}
			key := completed.Attempts[0].Completion.Outputs[0].ArtifactID
			if err := tc.mutate(ctx, c, key); err != nil {
				t.Fatal(err)
			}
			fresh, created, receipt, err := c.SubmitWithReceipt(ctx, SubmitRequest{
				ProjectID: "project", ShotID: "shot", Revision: 1, IdempotencyKey: "cache-invalid-" + tc.name,
				OutputProfile: OutputProfilePNG, ReuseCachedSuccess: true,
			})
			if err != nil || !created || receipt != nil || fresh.ID == original.ID || fresh.State != jobs.Queued {
				t.Fatalf("invalid old output must miss and admit fresh work: job=%s created=%v receipt=%+v state=%s err=%v", fresh.ID, created, receipt, fresh.State, err)
			}
		})
	}
}

func TestCanonicalCacheFingerprintChangesWithEachPinnedDependency(t *testing.T) {
	digest := func(s string) string { return digestBytes([]byte(s)) }
	closureFields := map[string]string{
		"assets": "asset-set-1", "display_uv": "uv-1", "rig": "rig-1", "scene": "scene-1",
		"lighting": "light-1", "camera": "camera-1", "masks": "masks-1", "strategy": "strategy-1",
		"color": "color-1", "profile": "profile-1",
	}
	closureDigest := func(fields map[string]string) string {
		data, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		return digestBytes(data)
	}
	base := jobs.Input{
		ProjectID: "project", ShotRevisionID: "shot:watch:revision:1", ShotRevision: 1,
		ToolProfileDigest: digest("tool-profile"), RendererVersion: "blender-worker-v1",
		OutputProfile: OutputProfilePNG, SubjectKind: jobs.SubjectPhysicalRender, SubjectID: "watch",
		Bindings: []jobs.ArtifactBinding{
			{Role: "shot_snapshot", ArtifactID: "shot:watch:revision:1", RevisionID: "revision:1", SHA256: digest("snapshot")},
			{Role: "worker_executable", ArtifactID: "worker", RevisionID: "build:1", SHA256: digest("worker")},
			{Role: "physical_closure", ArtifactID: "closure", RevisionID: "sha256:" + closureDigest(closureFields), SHA256: closureDigest(closureFields)},
			{Role: "physical_adapter", ArtifactID: "adapter", RevisionID: "sha256:" + digest("adapter"), SHA256: digest("adapter")},
			{Role: "physical_asset:display", ArtifactID: "display", RevisionID: "revision:1", SHA256: digest("display")},
			{Role: "physical_asset:scene", ArtifactID: "scene", RevisionID: "revision:1", SHA256: digest("scene")},
			{Role: "test_route", ArtifactID: "route", RevisionID: "route:lease", SHA256: digest("route")},
		},
	}
	_, baseline, err := jobs.CanonicalInput(base)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name string
		edit func(*jobs.Input)
	}{
		{"shot revision", func(in *jobs.Input) { in.ShotRevision++; in.ShotRevisionID += ":changed" }},
		{"tool profile", func(in *jobs.Input) { in.ToolProfileDigest = digest("tool-profile-v2") }},
		{"renderer and strategy version", func(in *jobs.Input) { in.RendererVersion = "blender-worker-v2" }},
		{"output color/profile", func(in *jobs.Input) { in.OutputProfile = OutputProfilePhysicalPreviewPNG }},
		{"worker build", func(in *jobs.Input) { in.Bindings[1].SHA256 = digest("worker-v2") }},
		{"display asset bytes", func(in *jobs.Input) { in.Bindings[4].SHA256 = digest("display-v2") }},
		{"scene asset bytes", func(in *jobs.Input) { in.Bindings[5].SHA256 = digest("scene-v2") }},
		{"project namespace", func(in *jobs.Input) { in.ProjectID = "other-project" }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			changed := base
			changed.Bindings = append([]jobs.ArtifactBinding(nil), base.Bindings...)
			tc.edit(&changed)
			_, fingerprint, err := jobs.CanonicalInput(changed)
			if err != nil {
				t.Fatal(err)
			}
			if fingerprint == baseline {
				t.Fatalf("one-at-a-time dependency change did not invalidate cache fingerprint: %s", tc.name)
			}
		})
	}
	for _, field := range []string{"assets", "display_uv", "rig", "scene", "lighting", "camera", "masks", "strategy", "color", "profile"} {
		t.Run("closure/"+field, func(t *testing.T) {
			changed := base
			changed.Bindings = append([]jobs.ArtifactBinding(nil), base.Bindings...)
			updated := make(map[string]string, len(closureFields))
			for key, value := range closureFields {
				updated[key] = value
			}
			updated[field] += "-changed"
			changed.Bindings[2].SHA256 = closureDigest(updated)
			_, fingerprint, err := jobs.CanonicalInput(changed)
			if err != nil || fingerprint == baseline {
				t.Fatalf("one-at-a-time %s closure change did not invalidate cache fingerprint: equal=%v err=%v", field, fingerprint == baseline, err)
			}
		})
	}

	still := base
	still.SubjectKind, still.SubjectID, still.OutputProfile = jobs.SubjectPhysicalRender, "watch", OutputProfilePNG
	motion := still
	motion.OutputProfile = OutputProfilePhysicalMotionMP4V2
	_, stillFingerprint, err := jobs.CanonicalInput(still)
	if err != nil {
		t.Fatal(err)
	}
	_, motionFingerprint, err := jobs.CanonicalInput(motion)
	if err != nil {
		t.Fatal(err)
	}
	if stillFingerprint == motionFingerprint {
		t.Fatal("still and motion profiles must never share a cache identity")
	}

	clocked := jobs.Input{ProjectID: "project", ShotRevisionID: "shot:phone:revision:1", ShotRevision: 1, SourceClockDigest: digest("clock-1"), ToolProfileDigest: digest("tool"), RendererVersion: "still-v1", OutputProfile: OutputProfilePNG, Bindings: []jobs.ArtifactBinding{{Role: "worker", ArtifactID: "worker", RevisionID: "build:1", SHA256: digest("worker")}}}
	_, clock1, err := jobs.CanonicalInput(clocked)
	if err != nil {
		t.Fatal(err)
	}
	clocked.SourceClockDigest = digest("clock-2")
	_, clock2, err := jobs.CanonicalInput(clocked)
	if err != nil || clock1 == clock2 {
		t.Fatalf("source/time change must invalidate cache identity: equal=%v err=%v", clock1 == clock2, err)
	}
}
