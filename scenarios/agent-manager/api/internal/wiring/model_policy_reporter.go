package wiring

import (
	"context"

	"agent-manager/internal/modelpolicydrift"
	"agent-manager/internal/promptmanager"
)

type modelPolicyReporter struct{ client *promptmanager.HTTPClient }

func (r modelPolicyReporter) Report(ctx context.Context, report modelpolicydrift.Report) error {
	finding := report.Finding
	severity := "minor"
	if finding.Severity == "error" {
		severity = "major"
	}
	return r.client.PublishScenarioQABug(ctx, promptmanager.ScenarioQABug{
		Title: "model-policy drift: " + finding.Fingerprint, SignalType: "code-defect", Severity: severity,
		Repro:    []string{"vrooli scenario start agent-manager", "GET /api/v1/health/model-policy-drift"},
		Expected: "resource-owned policy models remain present in the live runner catalog", Actual: finding.Message,
		Description:  "Scheduled model-policy drift detection measured runner=" + finding.Runner + ", role=" + finding.Role + ", model=" + finding.Model + ".",
		Context:      map[string]string{"runner": finding.Runner, "role": finding.Role, "model": finding.Model, "fingerprint": finding.Fingerprint},
		HonestyFlags: []string{"ai-generated-summary"}, IdempotencyKey: finding.Fingerprint,
	})
}
