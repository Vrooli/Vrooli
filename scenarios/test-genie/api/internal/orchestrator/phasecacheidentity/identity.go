// Package phasecacheidentity owns the phase cache's identity and its read and
// write sides.
//
// It was extracted from the suite orchestrator, where it sat beside batching,
// admission, and orchestration in one 2,900-line file. Keeping it separate
// matters because the identity IS the safety argument for the cache: a result
// may be reused exactly when the scoped input digest, the provider build
// identity, the descriptor snapshot, and the execution configuration are all
// unchanged. Any weakening of that rule is a correctness bug, and it is easier
// to see — and to review — when the rule is not interleaved with scheduling.
package phasecacheidentity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/captureprofile"
	"test-genie/internal/orchestrator/phasecache"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerreadiness"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

// Identity derives the four-part cache identity for a phase, or reports that
// the phase has none.
//
// A phase without an identity is simply not cacheable, which is the correct
// outcome for an observational provider or for a run whose descriptor and
// execution-configuration digests were not resolved.
func Identity(env workspacepkg.Environment, phase phases.Definition, readiness map[string]providerreadiness.Outcome) (phasecache.Identity, bool) {
	if strings.ToLower(strings.TrimSpace(phase.Determinism.Default)) != "file-determined" || len(phase.Determinism.Inputs) == 0 {
		return phasecache.Identity{}, false
	}
	digest, err := phasecache.ScopedDigest(env.ScenarioDir, phase.Determinism.Inputs)
	if err != nil {
		return phasecache.Identity{}, false
	}
	providerIdentity := "native:" + phase.Name.Key()
	if phase.ProviderScenario != "" {
		outcome := readiness[phase.Name.Key()]
		provider := strings.TrimSpace(outcome.ProviderScenario)
		if provider == "" {
			provider = strings.TrimSpace(phase.ProviderScenario)
		}
		if provider == "" {
			return phasecache.Identity{}, false
		}
		buildRevision := strings.TrimSpace(outcome.BuildRevision)
		binaryModifiedAt := strings.TrimSpace(outcome.BinaryModifiedAt)
		freshnessDigest := strings.TrimSpace(outcome.FreshnessDigest)
		// A descriptor contract describes what a provider promises, not which
		// provider build produced a verdict. If readiness did not supply any live
		// build evidence, the result is not safe to reuse across runs.
		if buildRevision == "" && binaryModifiedAt == "" && freshnessDigest == "" {
			return phasecache.Identity{}, false
		}
		providerIdentity = strings.Join([]string{
			provider,
			strings.TrimSpace(outcome.SpecVersion),
			buildRevision,
			binaryModifiedAt,
			freshnessDigest,
		}, "|")
	}
	if env.DescriptorSnapshotDigest == "" || env.ExecutionConfigurationDigest == "" {
		return phasecache.Identity{}, false
	}
	return phasecache.Identity{
		ScopedInputDigest:      digest,
		ProviderBuildIdentity:  providerIdentity,
		DescriptorSnapshotHash: env.DescriptorSnapshotDigest,
		ExecutionConfiguration: env.ExecutionConfigurationDigest,
	}, true
}

// Lookup carries the cache identity captured before a phase runs together
// with the cached result found under that exact identity, if any.
type Lookup struct {
	Identity       phasecache.Identity
	IdentityValid  bool
	Result         phases.ExecutionResult
	Audit          bool
	Found          bool
	CachedDuration int64
}

// Load captures the phase's current identity and returns a reusable result for
// that identity, if available. Callers retain the lookup through execution so
// completion and audit use the same key.
//
// projectRoot and logPath are passed explicitly rather than derived, so this
// package holds no orchestrator state and its contract is visible in its
// signature.
func Load(projectRoot string, env workspacepkg.Environment, runID, logPath string, phase phases.Definition, readiness map[string]providerreadiness.Outcome) Lookup {
	identity, identityOK := Identity(env, phase, readiness)
	lookup := Lookup{Identity: identity, IdentityValid: identityOK}
	// Baseline captures are measurement runs. Reusing a cached phase would
	// erase its execution interval and resource sample, leaving the envelope
	// estimator with an apparently reliable but unmeasurable row.
	if strings.EqualFold(strings.TrimSpace(env.CaptureProfile), captureprofile.NameBaseline) {
		return lookup
	}
	if !identityOK {
		return lookup
	}
	key := phasecache.Key(identity)
	store := phasecache.New(env.EffectivePhaseCacheRoot())
	entry, found, err := store.Load(key)
	if err != nil || !found {
		return lookup
	}
	result := entry.Phase
	lookup.CachedDuration = result.DurationMilliseconds
	result.Name = phase.Name.String()
	result.DurationMilliseconds = 0
	result.DurationSeconds = 0
	result.CacheHit = true
	result.CacheSourceRunID = entry.RunID
	result.LogPath = logPath
	// State plainly that nothing changed. A cached FAILURE served without this
	// marker reads as a fresh failure, and an agent would re-investigate a
	// verdict that has already been established — the opposite of the saving.
	marker := fmt.Sprintf(
		"[INFO] cache hit: %s — unchanged since run %s; no input, provider build, descriptor, or execution configuration differs\n",
		result.Status, entry.RunID)
	if err := os.WriteFile(result.LogPath, []byte(marker), 0o644); err != nil {
		return lookup
	}
	if projectRoot != "" {
		if rel, err := filepath.Rel(projectRoot, result.LogPath); err == nil {
			result.LogPath = rel
		}
	}
	lookup.Result = result
	lookup.Audit = store.ShouldAudit(key, runID)
	lookup.Found = true
	return lookup
}

// InputsMatch reports whether the declared input digest still matches the
// identity captured before the phase ran.
func InputsMatch(env workspacepkg.Environment, phase phases.Definition, lookup Lookup) bool {
	if !lookup.IdentityValid {
		return false
	}
	after, err := phasecache.ScopedDigest(env.ScenarioDir, phase.Determinism.Inputs)
	return err == nil && after == lookup.Identity.ScopedInputDigest
}

// Save stores a reusable result under the exact identity captured before the
// phase ran.
//
// It is a no-op for a phase with no identity, for a status the cache may not
// serve, and — critically — when the scoped input digest moved while the phase
// ran. It never relabels the result with a post-execution identity.
func Save(env workspacepkg.Environment, runID string, phase phases.Definition, lookup Lookup, result phases.ExecutionResult) {
	if !lookup.IdentityValid || !phasecache.Reusable(result) || !InputsMatch(env, phase, lookup) {
		return
	}
	_ = phasecache.New(env.EffectivePhaseCacheRoot()).Save(phasecache.Key(lookup.Identity), runID, result)
}
