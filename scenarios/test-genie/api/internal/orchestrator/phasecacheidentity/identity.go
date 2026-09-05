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
		providerIdentity = strings.Join([]string{
			provider,
			strings.TrimSpace(outcome.SpecVersion),
			strings.TrimSpace(outcome.BuildRevision),
			strings.TrimSpace(outcome.FreshnessDigest),
		}, "|")
		// A provider with no readiness policy has no live build identity to
		// report. The descriptor snapshot is still part of the cache key, so
		// bind this safe file-determined fallback to the provider contract rather
		// than silently treating an empty readiness outcome as an identity.
		if strings.Trim(providerIdentity, "|") == provider {
			providerIdentity = "descriptor-contract:" + env.DescriptorSnapshotDigest + "|" + provider
		}
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

// Load returns a reusable result for a phase whose identity is unchanged.
//
// The four return values are the result, whether this hit should be audited,
// whether a hit was found at all, and the cached duration.
//
// projectRoot and logPath are passed explicitly rather than derived, so this
// package holds no orchestrator state and its contract is visible in its
// signature.
func Load(projectRoot string, env workspacepkg.Environment, runID, logPath string, phase phases.Definition, readiness map[string]providerreadiness.Outcome) (phases.ExecutionResult, bool, bool, int64) {
	// Baseline captures are measurement runs. Reusing a cached phase would
	// erase its execution interval and resource sample, leaving the envelope
	// estimator with an apparently reliable but unmeasurable row.
	if strings.EqualFold(strings.TrimSpace(env.CaptureProfile), captureprofile.NameBaseline) {
		return phases.ExecutionResult{}, false, false, 0
	}
	identity, ok := Identity(env, phase, readiness)
	if !ok {
		return phases.ExecutionResult{}, false, false, 0
	}
	key := phasecache.Key(identity)
	store := phasecache.New(env.EffectivePhaseCacheRoot())
	entry, found, err := store.Load(key)
	if err != nil || !found {
		return phases.ExecutionResult{}, false, false, 0
	}
	result := entry.Phase
	cachedDuration := result.DurationMilliseconds
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
		return phases.ExecutionResult{}, false, false, 0
	}
	if projectRoot != "" {
		if rel, err := filepath.Rel(projectRoot, result.LogPath); err == nil {
			result.LogPath = rel
		}
	}
	return result, store.ShouldAudit(key, runID), true, cachedDuration
}

// Save stores a reusable result under the phase's current identity.
//
// It is a no-op for a phase with no identity, for a status the cache may not
// serve, and — critically — when the scoped input digest moved while the phase
// ran. That last check is what stops a source edit mid-run from publishing a
// result under the digest from before the edit.
func Save(env workspacepkg.Environment, runID string, phase phases.Definition, readiness map[string]providerreadiness.Outcome, result phases.ExecutionResult) {
	identity, ok := Identity(env, phase, readiness)
	if !ok || !phasecache.Cacheable(result.Status) {
		return
	}
	// Recompute immediately after the phase. A source edit during execution is
	// never allowed to publish a result under the digest from before the edit.
	after, err := phasecache.ScopedDigest(env.ScenarioDir, phase.Determinism.Inputs)
	if err != nil || after != identity.ScopedInputDigest {
		return
	}
	_ = phasecache.New(env.EffectivePhaseCacheRoot()).Save(phasecache.Key(identity), runID, result)
}
