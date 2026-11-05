// Package policybridge translates Security Health's provider evidence into the
// standalone agent-policy snapshot wire shape. It does not perform policy
// evaluation and never changes finding evidence while producing a snapshot.
package policybridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"security-health/internal/validation"
)

const contractVersion = "agent-policy/v1"

type SnapshotSink interface {
	PublishProviderSnapshot(context.Context, []byte) error
}

type snapshot struct {
	ProviderID   string              `json:"provider_id"`
	Version      string              `json:"version"`
	Capabilities []capability        `json:"capabilities"`
	Scope        map[string][]string `json:"scope"`
	Health       health              `json:"health"`
	Readiness    readiness           `json:"readiness"`
	Evidence     string              `json:"evidence_state"`
	CapturedAt   time.Time           `json:"captured_at"`
	ExpiresAt    time.Time           `json:"expires_at"`
	Rules        []rule              `json:"rules"`
	Provenance   map[string]string   `json:"provenance"`
}

type capability struct {
	ID               string `json:"id"`
	IdealPosture     string `json:"ideal_posture"`
	DeclaredMaturity string `json:"declared_maturity"`
	SupportsAnalysis bool   `json:"supports_analysis"`
	SupportsEnforce  bool   `json:"supports_enforcement"`
	SupportsRepair   bool   `json:"supports_repair"`
}

type health struct {
	State     string    `json:"state"`
	CheckedAt time.Time `json:"checked_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message,omitempty"`
}

type readiness struct {
	State        string `json:"state"`
	RollbackPlan string `json:"rollback_plan"`
}

type rule struct {
	Risk     string `json:"risk"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Evidence []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Source  string `json:"source"`
	} `json:"evidence,omitempty"`
}

func BuildSnapshot(report validation.Report, now time.Time) ([]byte, error) {
	if now.IsZero() {
		return nil, errors.New("snapshot time is required")
	}
	evidence := "clean"
	if len(report.Findings) > 0 {
		evidence = "finding"
	}
	if len(report.SkippedScanners) > 0 {
		evidence = "unknown"
	}
	message := "Security Health evidence is current"
	if evidence != "clean" {
		message = "Security Health findings or coverage gaps remain; findings are immutable evidence"
	}
	out := snapshot{
		ProviderID: "security-health",
		Version:    contractVersion,
		Capabilities: []capability{{
			ID: "security-findings", IdealPosture: "clean, fresh, evidence-backed dependency policy", DeclaredMaturity: "advisory",
			SupportsAnalysis: true, SupportsEnforce: true, SupportsRepair: true,
		}},
		Health:     health{State: "healthy", CheckedAt: now.UTC(), ExpiresAt: now.UTC().Add(time.Hour), Message: message},
		Readiness:  readiness{State: "ready", RollbackPlan: "withdraw security-health snapshot and retain prior bundle"},
		Evidence:   evidence,
		CapturedAt: now.UTC(),
		ExpiresAt:  now.UTC().Add(time.Hour),
		Rules:      []rule{{Risk: "dependency_addition", Action: "allow", Reason: "Security Health provider evidence is evaluated centrally by the selected rollout profile"}, {Risk: "dependency_upgrade", Action: "allow", Reason: "Security Health provider evidence is evaluated centrally by the selected rollout profile"}, {Risk: "lifecycle_execution", Action: "allow", Reason: "Security Health provider evidence is evaluated centrally by the selected rollout profile"}, {Risk: "unknown", Action: "allow", Reason: "Unknown risk remains visible to the central evaluator"}},
		Provenance: map[string]string{"scenario": report.Scenario, "policy_mode": string(report.PolicyMode), "finding_count": fmt.Sprint(len(report.Findings))},
	}
	return json.MarshalIndent(out, "", "  ")
}

func Publish(ctx context.Context, sink SnapshotSink, report validation.Report, now time.Time) error {
	if sink == nil {
		return errors.New("snapshot sink is required")
	}
	data, err := BuildSnapshot(report, now)
	if err != nil {
		return err
	}
	return sink.PublishProviderSnapshot(ctx, data)
}
