package recoverypoint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RestoreRequest restores a recovery point into clean targets.
type RestoreRequest struct {
	Dir string
	// Into maps binding id to its target locator on the replacement host (a
	// directory for the object store, a database name for SQL providers).
	// Every binding in the manifest must have a target unless Bindings
	// narrows the set.
	Into map[string]string
	// Bindings optionally restricts the restore to these ids.
	Bindings  []string
	Keys      KeyResolver
	Sealer    Sealer
	Providers Registry
	Now       func() time.Time
}

// BindingRestore is the per-binding result.
type BindingRestore struct {
	ID        string    `json:"id"`
	Target    string    `json:"target"`
	Captured  Inventory `json:"captured"`
	Restored  Inventory `json:"restored"`
	Matched   bool      `json:"matched"`
	Written   bool      `json:"written"`
	Discarded bool      `json:"discarded"`
}

// RestoreReport is what a restore returns; Outcome is succeeded only when
// every binding restored and its inventory matched.
type RestoreReport struct {
	RecoveryPointID string           `json:"recovery_point_id"`
	DeploymentID    string           `json:"deployment_id"`
	Refs            Refs             `json:"refs"`
	CapturedAt      time.Time        `json:"captured_at"`
	StartedAt       time.Time        `json:"started_at"`
	CompletedAt     time.Time        `json:"completed_at"`
	Bindings        []BindingRestore `json:"bindings"`
	Outcome         string           `json:"outcome"`
	Error           *Error           `json:"error,omitempty"`
}

// Outcomes.
const (
	OutcomeSucceeded = "succeeded"
	OutcomeFailed    = "failed"
)

// Restore verifies the recovery point, resolves the key, opens every
// artifact, proves every target clean, and only then writes. A failure after
// writing discards what providers can undo and reports failed with the
// residue named; it never reports success for a subset.
func Restore(ctx context.Context, req RestoreRequest) (RestoreReport, error) {
	now := time.Now
	if req.Now != nil {
		now = req.Now
	}
	report := RestoreReport{StartedAt: now().UTC(), Outcome: OutcomeFailed}
	fail := func(err error) (RestoreReport, error) {
		report.CompletedAt = now().UTC()
		report.Error = AsError(err, CodeRestoreFailed)
		return report, err
	}
	if req.Keys == nil || req.Providers == nil {
		return fail(newError(CodeInvalidArgument, "key resolver and provider registry are required"))
	}
	manifest, err := ReadManifest(req.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fail(newError(CodeRecoveryPointCorrupt, "%s holds no recovery point manifest", req.Dir))
		}
		return fail(err)
	}
	report.RecoveryPointID, report.DeploymentID, report.Refs, report.CapturedAt = manifest.ID, manifest.DeploymentID, manifest.Refs, manifest.CapturedAt
	selected, err := selectBindings(manifest, req.Bindings, req.Into)
	if err != nil {
		return fail(err)
	}
	// 1. Integrity of every sealed artifact before any key or target is touched.
	if err := verifySealedArtifacts(req.Dir, manifest); err != nil {
		return fail(err)
	}
	// 2. Key availability is a blocker, reported before any write.
	material, err := req.Keys.ResolveKey(ctx, manifest.KeyRef)
	if err != nil {
		return fail(keyUnavailable(manifest.KeyRef, err))
	}
	// 3. Open every artifact into a private stage; authentication failure is corruption.
	stage, err := os.MkdirTemp("", "recoverypoint-restore-")
	if err != nil {
		return fail(newError(CodeRestoreFailed, "create restore stage: %v", err))
	}
	defer os.RemoveAll(stage)
	plainPaths := map[string]string{}
	for _, b := range selected {
		artifact := findArtifact(manifest, b.ID)
		sealed, err := os.ReadFile(filepath.Join(req.Dir, artifact.File)) //nolint:gosec // recovery point directory
		if err != nil {
			return fail(newError(CodeRecoveryPointCorrupt, "binding %s: read sealed artifact: %v", b.ID, err))
		}
		plain, err := req.Sealer.Open(material, sealed, manifest.ID, b.ID)
		if err != nil {
			return fail(err)
		}
		if BytesSHA256(plain) != artifact.PlainSHA256 {
			return fail(newError(CodeRecoveryPointCorrupt, "binding %s: decrypted artifact does not match its recorded checksum", b.ID))
		}
		path := filepath.Join(stage, b.ID+".artifact")
		if err := os.WriteFile(path, plain, 0o600); err != nil {
			return fail(newError(CodeRestoreFailed, "binding %s: stage artifact: %v", b.ID, err))
		}
		plainPaths[b.ID] = path
	}
	// 4. Every target must be clean before the first write.
	providers := map[string]Provider{}
	for _, b := range selected {
		provider, err := req.Providers.Lookup(b)
		if err != nil {
			return fail(err)
		}
		if err := provider.EnsureClean(ctx, b, req.Into[b.ID]); err != nil {
			return fail(err)
		}
		providers[b.ID] = provider
	}
	// 5. Write, then measure. Any failure discards what can be discarded.
	for _, b := range selected {
		entry := BindingRestore{ID: b.ID, Target: req.Into[b.ID], Captured: manifest.Checksums[b.ID]}
		provider := providers[b.ID]
		if err := provider.Restore(ctx, b, plainPaths[b.ID], entry.Target); err != nil {
			report.Bindings = append(report.Bindings, entry)
			return fail(discardWritten(ctx, providers, selected, &report, err))
		}
		entry.Written = true
		restored, err := provider.Inventory(ctx, b, entry.Target)
		if err != nil {
			report.Bindings = append(report.Bindings, entry)
			return fail(discardWritten(ctx, providers, selected, &report, err))
		}
		entry.Restored = restored
		entry.Matched = inventoriesMatch(entry.Captured, restored)
		report.Bindings = append(report.Bindings, entry)
		if !entry.Matched {
			return fail(discardWritten(ctx, providers, selected, &report, newError(CodeRecoveryPointCorrupt, "binding %s: restored inventory (count %d, checksum %s) does not match the recovery point (count %d, checksum %s)", b.ID, restored.Count, restored.Checksum, entry.Captured.Count, entry.Captured.Checksum)))
		}
	}
	report.Outcome = OutcomeSucceeded
	report.CompletedAt = now().UTC()
	return report, nil
}

func inventoriesMatch(captured, restored Inventory) bool {
	if captured.Count != restored.Count {
		return false
	}
	if captured.Comparable && restored.Comparable {
		return captured.Checksum == restored.Checksum
	}
	return true
}

func discardWritten(ctx context.Context, providers map[string]Provider, bindings []Binding, report *RestoreReport, cause error) error {
	typed := AsError(cause, CodeRestoreFailed)
	residue := []string{}
	for i := range report.Bindings {
		entry := &report.Bindings[i]
		if !entry.Written {
			continue
		}
		var b Binding
		for _, candidate := range bindings {
			if candidate.ID == entry.ID {
				b = candidate
			}
		}
		if err := providers[entry.ID].Discard(ctx, b, entry.Target); err != nil {
			residue = append(residue, err.Error())
			continue
		}
		entry.Discarded = true
	}
	if len(residue) > 0 {
		typed = typed.withDetail("residue", residue)
	}
	return typed
}

func selectBindings(manifest Manifest, only []string, into map[string]string) ([]Binding, error) {
	want := map[string]bool{}
	for _, id := range only {
		want[id] = true
	}
	var selected []Binding
	for _, b := range manifest.Bindings {
		if len(want) > 0 && !want[b.ID] {
			continue
		}
		target, ok := into[b.ID]
		if !ok || target == "" {
			return nil, newError(CodeInvalidArgument, "binding %s has no restore target", b.ID)
		}
		if findArtifact(manifest, b.ID) == nil {
			return nil, newError(CodeRecoveryPointCorrupt, "binding %s has no artifact in the manifest", b.ID)
		}
		selected = append(selected, b)
	}
	for id := range want {
		found := false
		for _, b := range selected {
			if b.ID == id {
				found = true
			}
		}
		if !found {
			return nil, newError(CodeInvalidArgument, "binding %s is not part of recovery point %s", id, manifest.ID)
		}
	}
	if len(selected) == 0 {
		return nil, newError(CodeInvalidArgument, "recovery point %s selects no bindings", manifest.ID)
	}
	return selected, nil
}

func findArtifact(manifest Manifest, binding string) *Artifact {
	for i := range manifest.Artifacts {
		if manifest.Artifacts[i].Binding == binding {
			return &manifest.Artifacts[i]
		}
	}
	return nil
}

func verifySealedArtifacts(dir string, manifest Manifest) error {
	for _, artifact := range manifest.Artifacts {
		if filepath.Base(artifact.File) != artifact.File {
			return newError(CodeRecoveryPointCorrupt, "artifact file %q escapes the recovery point directory", artifact.File)
		}
		path := filepath.Join(dir, artifact.File)
		info, err := os.Stat(path)
		if err != nil {
			return newError(CodeRecoveryPointCorrupt, "artifact %s for binding %s is missing", artifact.File, artifact.Binding)
		}
		if info.Size() != artifact.Bytes {
			return newError(CodeRecoveryPointCorrupt, "artifact %s is %d bytes, manifest records %d", artifact.File, info.Size(), artifact.Bytes)
		}
		sum, err := FileSHA256(path)
		if err != nil {
			return newError(CodeRecoveryPointCorrupt, "artifact %s: %v", artifact.File, err)
		}
		if sum != artifact.SHA256 {
			return newError(CodeRecoveryPointCorrupt, "artifact %s checksum mismatch", artifact.File).withDetail("binding", artifact.Binding).withDetail("recorded_sha256", artifact.SHA256).withDetail("computed_sha256", sum)
		}
	}
	return nil
}

// String renders a report outcome for logs.
func (r RestoreReport) String() string {
	return fmt.Sprintf("recovery point %s: %s (%d bindings)", r.RecoveryPointID, r.Outcome, len(r.Bindings))
}
