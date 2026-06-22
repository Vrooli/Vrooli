// Namespaceability gate auto-detection (Baseline Modes, plan phase P2 ↔ P5).
//
// The decision tree (decideMode in engagement.go) routes any scenario that
// *writes* an un-adopted Redis/Qdrant store to live mode — a shadow instance of
// such a scenario would read and write the LIVE keyspace/collection, corrupting
// the very state the engagement exists to protect (plan §1a, Contract §8).
//
// Plan §1a's load-bearing promise is that the mode is "automatically chosen —
// the planning/dispatching agent should not have to reason about self-conflict
// manually." This file delivers that for the namespaceability gate: instead of
// trusting the human/agent to pass --writes-shared-store, we *query storage-health's
// ScenarioValidationService for the target* (its isolation_namespace analyzer,
// which emits STORAGE_NAMESPACE_HARDCODED) and derive the signal. The declared
// flag stays as an OR override (force live even when detection is unavailable or
// clean).
//
// storage-health owns all storage judgment now; this supersedes the retired
// scenario-auditor `storage_namespace_helpers` rule (storage-health plan,
// Phase 12 — the rule was deleted once this consumer was re-pointed).
//
// Detection is best-effort and degrades safely: if storage-health is unreachable
// the gate falls back to the declared flag and surfaces a note, never silently
// routing a non-namespaceable writer to shadow on a missing signal — the operator
// can always pass --writes-shared-store to force live.
package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// namespaceScanTimeout bounds the storage-health validation so a slow or hung
// provider degrades to "unavailable" (safe fallback) instead of wedging
// `baseline start`. storage-health's per-scenario validate runs a full static
// storage analysis (AST + code-facts language/domain resolution), so it is a
// few-tens-of-seconds operation, not instant; the operator can skip it entirely
// by passing --writes-shared-store.
const namespaceScanTimeout = 120 * time.Second

// storageNamespaceFindingCode is the storage-health finding emitted when a
// scenario hardcodes a Redis/Qdrant namespace instead of routing it through the
// variant-aware api-core helpers — i.e. it cannot be safely shadowed. storage-health's
// isolation_namespace analyzer owns this judgment; it supersedes the retired
// scenario-auditor storage_namespace_helpers rule.
const storageNamespaceFindingCode = "STORAGE_NAMESPACE_HARDCODED"

// sharedStoreVerdict is the resolved namespaceability signal plus a human note
// explaining how it was derived (auto-detected, declared, or detection-failed),
// which the decision tree folds into its reasons so the mode choice is auditable.
type sharedStoreVerdict struct {
	writesSharedStore bool
	note              string
}

// detectSharedStore queries storage-health's ScenarioValidationService for the
// scenario and reports whether its assessment carries a STORAGE_NAMESPACE_HARDCODED
// finding (i.e. it writes a hardcoded, un-adopted Redis/Qdrant namespace). It is
// a seam (package var) so the decision-tree tests inject a deterministic verdict
// without a live storage-health.
//
// detail carries either a short evidence string (the violating file) on a hit or
// the reason detection could not run; err is non-nil only when storage-health
// could not be consulted at all (so the caller can distinguish "clean" from
// "unknown").
//
// `storage-health validate scenario --json` exits non-zero when the scenario has
// ERROR-severity findings (e.g. unwired routed seams), but it still emits the
// full assessment JSON to stdout first. We therefore decode the captured stdout
// regardless of the exit code and only treat a non-parseable response — meaning
// storage-health produced no assessment at all — as "unknown".
var detectSharedStore = func(ctx context.Context, scenario string) (found bool, detail string, err error) {
	scanCtx, cancel := context.WithTimeout(ctx, namespaceScanTimeout)
	defer cancel()
	out, scanErr := runCommand(scanCtx, "storage-health", "validate", "scenario", scenario, "--json")

	var resp struct {
		Assessment *struct {
			Findings []struct {
				Code     string `json:"code"`
				Location string `json:"location"`
			} `json:"findings"`
		} `json:"assessment"`
	}
	if jsonErr := json.Unmarshal(out, &resp); jsonErr != nil {
		// No parseable assessment — storage-health is genuinely unavailable or
		// errored before producing output. Surface "unknown" so the caller falls
		// back to the declared flag rather than treating it as clean.
		if scanErr != nil {
			return false, "storage-health unavailable", scanErr
		}
		return false, "unparseable storage-health response", fmt.Errorf("parse storage validation for %s: %w", scenario, jsonErr)
	}

	if resp.Assessment == nil {
		return false, "no storage-namespace findings", nil
	}
	for _, fnd := range resp.Assessment.Findings {
		if strings.TrimSpace(fnd.Code) == storageNamespaceFindingCode {
			detail := strings.TrimSpace(fnd.Location)
			if detail == "" {
				detail = "hardcoded Redis/Qdrant namespace"
			}
			return true, detail, nil
		}
	}
	return false, "no storage-namespace findings", nil
}

// resolveSharedStoreSignal derives the namespaceability gate input for a
// scenario. The declared flag (--writes-shared-store) is authoritative when set
// (an explicit operator override forcing live). Otherwise it auto-detects via the
// auditor; an unreachable auditor leaves the gate open (shadow-eligible) but says
// so, since over-routing to live on a flaky signal would defeat shadow mode for
// every scenario whenever the auditor is down — the operator can force live with
// the flag.
func resolveSharedStoreSignal(ctx context.Context, scenario string, declared bool) sharedStoreVerdict {
	if declared {
		return sharedStoreVerdict{writesSharedStore: true, note: "declared via --writes-shared-store (forces live)"}
	}
	found, detail, err := detectSharedStore(ctx, scenario)
	switch {
	case err != nil:
		return sharedStoreVerdict{
			writesSharedStore: false,
			note:              "auto-detection unavailable (" + detail + ") — treating as namespaceable; pass --writes-shared-store to force live",
		}
	case found:
		return sharedStoreVerdict{
			writesSharedStore: true,
			note:              "auto-detected via storage-health " + storageNamespaceFindingCode + " (" + detail + ")",
		}
	default:
		return sharedStoreVerdict{
			writesSharedStore: false,
			note:              "auto-detected namespaceable via storage-health " + storageNamespaceFindingCode,
		}
	}
}
