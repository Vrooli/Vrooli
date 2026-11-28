package cloudtarget

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// DataRestoreRequest restores one recovery point into clean bindings on this
// target. Into maps binding id to its target locator; every manifest binding
// (or every id in Bindings when set) needs one.
type DataRestoreRequest struct {
	Effect           EffectRequest
	RecoveryPointID  string
	RecoveryPointDir string
	Into             map[string]string
	Bindings         []string
	Deps             DataDeps
}

// ParseRestoreTargets decodes --into values (`<binding>=<locator>` or a JSON
// object) into the restore target map.
func ParseRestoreTargets(values []string) (map[string]string, error) {
	into := map[string]string{}
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "{") {
			var many map[string]string
			if err := json.Unmarshal([]byte(raw), &many); err != nil {
				return nil, refuse(CodeInvalidArgument, "parse restore targets: %v", err)
			}
			for k, v := range many {
				into[k] = v
			}
			continue
		}
		idx := strings.Index(raw, "=")
		if idx <= 0 || idx == len(raw)-1 {
			return nil, refuse(CodeInvalidArgument, "restore target %q is not <binding>=<locator>", raw)
		}
		into[raw[:idx]] = raw[idx+1:]
	}
	if len(into) == 0 {
		return nil, refuse(CodeInvalidArgument, "at least one --into <binding>=<locator> is required")
	}
	return into, nil
}

func (s *Store) resolveRecoveryPointDir(deploymentID, recoveryPointID, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return s.RecoveryPointDir(deploymentID, recoveryPointID)
}

// DataRestore is fenced and receipted like every other effectful verb. The
// engine proves the manifest and every sealed artifact intact, resolves the
// key, proves every target clean and only then writes; a corrupt archive is
// recovery_point_corrupt, a missing key recovery_key_unavailable and an
// occupied target restore_target_not_clean, each refused before any write.
func (s *Store) DataRestore(ctx context.Context, req DataRestoreRequest) (recoverypoint.RestoreReport, EffectResult, error) {
	if err := validIdentifier("recovery point id", req.RecoveryPointID); err != nil {
		return recoverypoint.RestoreReport{}, EffectResult{}, err
	}
	dir, err := s.resolveRecoveryPointDir(req.Effect.DeploymentID, req.RecoveryPointID, req.RecoveryPointDir)
	if err != nil {
		return recoverypoint.RestoreReport{}, EffectResult{}, err
	}
	req.Effect.Verb = "data restore"
	req.Effect.Input = map[string]any{"recovery_point_id": req.RecoveryPointID, "recovery_point_dir": dir, "into": req.Into, "bindings": req.Bindings}
	var report recoverypoint.RestoreReport
	result, err := s.RunEffect(ctx, req.Effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		restored, restoreErr := recoverypoint.Restore(ctx, recoverypoint.RestoreRequest{
			Dir: dir, Into: req.Into, Bindings: req.Bindings,
			Keys: req.Deps.keys(), Sealer: req.Deps.Sealer, Providers: req.Deps.providers(), Now: s.now,
		})
		report = restored
		details := map[string]any{
			"recovery_point_id": restored.RecoveryPointID, "recovery_point_dir": dir, "refs": restored.Refs,
			"captured_at": restored.CapturedAt, "started_at": restored.StartedAt, "completed_at": restored.CompletedAt,
			"bindings": restored.Bindings, "outcome": restored.Outcome,
			"measured_rto_ms":        restored.CompletedAt.Sub(restored.StartedAt).Milliseconds(),
			"recovery_point_age_ms":  restored.StartedAt.Sub(restored.CapturedAt).Milliseconds(),
			"recovery_point_age_key": "started_at - captured_at",
		}
		if restoreErr != nil {
			return details, OutcomeFailed, dataError(restoreErr)
		}
		return details, OutcomeSucceeded, nil
	})
	return report, result, err
}

// DataVerifyRequest checks a recovery point without restoring it. Expect
// optionally carries the inventory the caller expects per binding (for
// example the fixture-oracle/v1 expected state); each entry becomes an
// invariant result on the report.
type DataVerifyRequest struct {
	DeploymentID     string
	RecoveryPointID  string
	RecoveryPointDir string
	// OpenArtifacts also resolves the key and decrypts every artifact,
	// proving the key reference still resolves from this host.
	OpenArtifacts bool
	Expect        map[string]recoverypoint.Inventory
	Deps          DataDeps
}

// InvariantResult is one expected-vs-recorded comparison.
type InvariantResult struct {
	Binding  string `json:"binding"`
	Check    string `json:"check"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	Passed   bool   `json:"passed"`
}

// DataVerifyReport is the verify verb output.
type DataVerifyReport struct {
	recoverypoint.VerifyReport
	Invariants []InvariantResult `json:"invariants"`
	// InvariantsPassed is false when any invariant failed; a report whose
	// artifacts verify but whose invariants fail is still verify_failed.
	InvariantsPassed bool `json:"invariants_passed"`
}

// ParseExpectedInventory decodes --expect JSON: {binding: {count, checksum}}.
func ParseExpectedInventory(raw string) (map[string]recoverypoint.Inventory, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var expect map[string]recoverypoint.Inventory
	if err := json.Unmarshal([]byte(raw), &expect); err != nil {
		return nil, refuse(CodeInvalidArgument, "parse expected inventory: %v", err)
	}
	return expect, nil
}

// DataVerify reads and never writes; it needs no fence.
func (s *Store) DataVerify(ctx context.Context, req DataVerifyRequest) (DataVerifyReport, error) {
	if err := validIdentifier("recovery point id", req.RecoveryPointID); err != nil {
		return DataVerifyReport{}, err
	}
	dir, err := s.resolveRecoveryPointDir(req.DeploymentID, req.RecoveryPointID, req.RecoveryPointDir)
	if err != nil {
		return DataVerifyReport{}, err
	}
	verifyReq := recoverypoint.VerifyRequest{Dir: dir, Sealer: req.Deps.Sealer, Now: s.now}
	if req.OpenArtifacts {
		verifyReq.Keys = req.Deps.keys()
	}
	verified, verifyErr := recoverypoint.Verify(ctx, verifyReq)
	report := DataVerifyReport{VerifyReport: verified, Invariants: []InvariantResult{}, InvariantsPassed: true}
	if verifyErr != nil {
		report.InvariantsPassed = false
		return report, dataError(verifyErr)
	}
	report.Invariants, report.InvariantsPassed = CompareInventories(req.Expect, verified.Checksums)
	if !report.InvariantsPassed {
		report.Outcome = recoverypoint.OutcomeFailed
		return report, fail(recoverypoint.CodeVerifyFailed, "recovery point %s does not satisfy the expected inventory", req.RecoveryPointID)
	}
	return report, nil
}

// CompareInventories renders expected-vs-recorded inventories as invariant
// results. A checksum is compared only when the recorded inventory is
// reproducible (Comparable); counts are always compared. An expected binding
// missing from the recovery point fails.
func CompareInventories(expect, recorded map[string]recoverypoint.Inventory) ([]InvariantResult, bool) {
	results := []InvariantResult{}
	passed := true
	for _, binding := range sortedKeys(expect) {
		want := expect[binding]
		got, ok := recorded[binding]
		if !ok {
			results = append(results, InvariantResult{Binding: binding, Check: "present", Expected: "captured", Observed: "missing"})
			passed = false
			continue
		}
		count := InvariantResult{Binding: binding, Check: "count", Expected: itoa(want.Count), Observed: itoa(got.Count), Passed: want.Count == got.Count}
		results = append(results, count)
		passed = passed && count.Passed
		if want.Checksum != "" && got.Comparable {
			sum := InvariantResult{Binding: binding, Check: "checksum", Expected: want.Checksum, Observed: got.Checksum, Passed: want.Checksum == got.Checksum}
			results = append(results, sum)
			passed = passed && sum.Passed
		}
	}
	return results, passed
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }
