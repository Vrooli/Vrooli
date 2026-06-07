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
// trusting the human/agent to pass --writes-shared-store, we *query the
// scenario-auditor `storage-namespace-v1` standard for the target* (the P5
// detection producer, parts 23) and derive the signal. The declared flag stays
// as an OR override (force live even when detection is unavailable or clean).
//
// Detection is best-effort and degrades safely: if the auditor is unreachable
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

// namespaceScanTimeout bounds the auditor scan so a slow or hung auditor degrades
// to "unavailable" (safe fallback) instead of wedging `baseline start`. A
// targeted single-rule scan of one scenario is a seconds-scale operation.
const namespaceScanTimeout = 120 * time.Second

const (
	// storageNamespaceStandard is the auditor standard the P5 detection rule
	// emits (rules/api/storage_namespace_helpers.go `Standard:` field). A
	// violation means the scenario hardcodes a Redis/Qdrant namespace instead of
	// routing it through the variant-aware api-core helpers — i.e. it cannot be
	// safely shadowed.
	storageNamespaceStandard = "storage-namespace-v1"
	// storageNamespaceRuleID is the auditor rule ID used to filter the targeted
	// scan. The auditor derives a rule's ID from its filename when no explicit ID
	// is declared (internal/ruleengine/loader.go), so this tracks the rule file
	// scenarios/scenario-auditor/api/rules/api/storage_namespace_helpers.go.
	storageNamespaceRuleID = "storage_namespace_helpers"
)

// sharedStoreVerdict is the resolved namespaceability signal plus a human note
// explaining how it was derived (auto-detected, declared, or detection-failed),
// which the decision tree folds into its reasons so the mode choice is auditable.
type sharedStoreVerdict struct {
	writesSharedStore bool
	note              string
}

// detectSharedStore queries the scenario-auditor `storage-namespace-v1` standard
// for the scenario and reports whether it has any violation (i.e. it writes a
// hardcoded, un-adopted Redis/Qdrant namespace). It is a seam (package var) so
// the decision-tree tests inject a deterministic verdict without a live auditor.
//
// detail carries either a short evidence string (the violating file) on a hit or
// the reason detection could not run; err is non-nil only when the auditor could
// not be consulted at all (so the caller can distinguish "clean" from "unknown").
var detectSharedStore = func(ctx context.Context, scenario string) (found bool, detail string, err error) {
	// Targeted scan over just the namespace rule keeps the query cheap (one rule,
	// one scenario) and the result unambiguous: any violation that comes back is a
	// storage-namespace-v1 hit. --wait blocks until the scan finishes; --json
	// emits the final job status (StandardsScanStatus) verbatim.
	scanCtx, cancel := context.WithTimeout(ctx, namespaceScanTimeout)
	defer cancel()
	out, scanErr := runCommand(scanCtx, "scenario-auditor", "standards", "scan", scenario,
		"--type", "targeted", "--rules", storageNamespaceRuleID,
		"--wait", "--timeout", namespaceScanTimeout.String(), "--json")
	if scanErr != nil {
		return false, "scenario-auditor unavailable", scanErr
	}

	var status struct {
		Status string `json:"status"`
		Result *struct {
			Violations []struct {
				Standard string `json:"standard"`
				FilePath string `json:"file_path"`
			} `json:"violations"`
		} `json:"result"`
	}
	if jsonErr := json.Unmarshal(out, &status); jsonErr != nil {
		return false, "unparseable auditor response", fmt.Errorf("parse storage-namespace scan for %s: %w", scenario, jsonErr)
	}

	// A scan that errored or never completed is "unknown", not "clean" — surface
	// it as an error so the caller falls back to the declared flag.
	if st := strings.ToLower(strings.TrimSpace(status.Status)); st != "" && st != "completed" && st != "complete" && st != "success" && st != "succeeded" {
		return false, "scan did not complete (status: " + status.Status + ")", fmt.Errorf("storage-namespace scan for %s did not complete: status=%s", scenario, status.Status)
	}

	if status.Result == nil {
		return false, "no storage-namespace violations", nil
	}
	for _, v := range status.Result.Violations {
		// The scan was filtered to the one rule, but match the standard defensively
		// so a future multi-rule reuse of this parser stays correct.
		if strings.TrimSpace(v.Standard) == "" || v.Standard == storageNamespaceStandard {
			detail := strings.TrimSpace(v.FilePath)
			if detail == "" {
				detail = "hardcoded Redis/Qdrant namespace"
			}
			return true, detail, nil
		}
	}
	return false, "no storage-namespace violations", nil
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
			note:              "auto-detected via scenario-auditor " + storageNamespaceStandard + " (" + detail + ")",
		}
	default:
		return sharedStoreVerdict{
			writesSharedStore: false,
			note:              "auto-detected namespaceable via scenario-auditor " + storageNamespaceStandard,
		}
	}
}
