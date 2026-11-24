package monetization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const cl3HealthPhase = "cloud-production-health"

type CL3HealthSuiteResult struct {
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	Command      []string `json:"command"`
	WorkingDir   string   `json:"working_directory"`
	DurationMS   int64    `json:"duration_ms"`
	ExitCode     int      `json:"exit_code"`
	Output       string   `json:"output,omitempty"`
	EvidenceRefs []string `json:"evidence_refs"`
}

type CL3HealthReport struct {
	Status             string                 `json:"status"`
	LocalContracts     string                 `json:"local_contracts"`
	StagingShowcase    string                 `json:"staging_showcase"`
	ProductionShowcase string                 `json:"production_showcase"`
	Suites             []CL3HealthSuiteResult `json:"suites"`
	Limitations        []string               `json:"limitations"`
}

type CL3HealthRunner func(context.Context, string) (*CL3HealthReport, error)

// RunCL3HealthContracts runs the fixed, owner-authored CL3 contract suites.
// A passing local suite never implies staging or production readiness; those
// remain independent required receipts and keep this phase failed closed.
func RunCL3HealthContracts(ctx context.Context, scenarioPath string) (*CL3HealthReport, error) {
	root, err := cl3RepositoryRoot(scenarioPath)
	if err != nil {
		return nil, err
	}
	suites := []struct {
		name, dir string
		args, env []string
		refs      []string
	}{
		{"health-tls-alerts-and-restart-persistence", "scenarios/scenario-to-cloud/api", []string{"go", "test", "-count=1", "./health", "./persistence"}, nil, []string{"api/health/alerts_test.go::TestAlerterCertificateExpiringAndRenewalFailed", "api/health/alerts_test.go::TestAlerterRestartPublishesRecoveryFromPersistedOpenState", "api/persistence/health_alert_state_test.go::TestHealthAlertStateRoundTripsAcrossStoreInstances", "api/health/scheduler_test.go::TestSchedulerRunOnceObservesEachTargetAndContinuesAfterFailure"}},
		{"dbm-schedule-compile-and-target-readback", "scenarios/scenario-to-cloud/api", []string{"go", "test", "-count=1", "./execplan", "./vps"}, nil, []string{"api/execplan/execplan_test.go::TestBackupPolicyCompilesTargetLocalDBMScheduleAfterReadiness", "api/vps/data_schedule_test.go::TestDataScheduleCompilesEffectfulTargetVerb"}},
		{"dbm-target-reconciliation-and-idempotent-readback", ".", []string{"go", "test", "-count=1", "./internal/cloudtarget", "./internal/cli/cloudtargethandlers", "./cli"}, nil, []string{"internal/cloudtarget/data_schedule_test.go::TestDataScheduleReconcilesExplicitBindingsToS3OnTarget"}},
		{"notification-hub-alert-lifecycle-dedupe-contract", "scenarios/notification-hub/api/internal/integrations", []string{"go", "test", "-count=1", "event_dedupe.go", "event_dedupe_test.go"}, nil, []string{"scenarios/notification-hub/api/internal/integrations/event_dedupe_test.go::TestEventDedupeKeyKeepsAlertOpenAndRecoveryDistinct"}},
		{"cli-plan-apply-wait-parity-under-test-genie-base", "scenarios/scenario-to-cloud/cli", []string{"go", "test", "-count=1", "-run", "TestExecuteIsPlanReviewApplyWait|TestFailedAndTimedOutOperationsNeverImplySuccess|TestAPIStartedOperationIsResumedByTheCLI", "."}, []string{"SCENARIO_TO_CLOUD_API_BASE=http://127.0.0.1:18888"}, []string{"cli/parity_test.go::TestExecuteIsPlanReviewApplyWait", "cli/parity_test.go::TestFailedAndTimedOutOperationsNeverImplySuccess", "cli/parity_test.go::TestAPIStartedOperationIsResumedByTheCLI"}},
		{"lpbs-signin-delivery-health-provider-feedback", "scenarios/landing-page-business-suite/api", []string{"go", "test", "-count=1", "-run", "^(TestSignInDeliveryHealthTreatsProviderBounceAsFailure|TestHealthEndpointDegradesAfterSignInProviderBounce)$", "."}, nil, []string{"scenarios/landing-page-business-suite/api/sign_in_flow_test.go::TestSignInDeliveryHealthTreatsProviderBounceAsFailure", "scenarios/landing-page-business-suite/api/health_signin_integration_test.go::TestHealthEndpointDegradesAfterSignInProviderBounce"}},
	}
	results := make([]CL3HealthSuiteResult, 0, len(suites))
	localOK := true
	for _, suite := range suites {
		result := runCL3HealthSuite(ctx, root, suite.name, suite.dir, suite.args, suite.env, suite.refs)
		results = append(results, result)
		if result.Status != "passed" {
			localOK = false
		}
	}
	local := "failed"
	if localOK {
		local = "passed"
	}
	return &CL3HealthReport{
		Status: "failed", LocalContracts: local, StagingShowcase: "unverified", ProductionShowcase: "unverified", Suites: results,
		Limitations: []string{
			"Local owner tests use test doubles and prove code contracts only; they do not prove Bridge reach, Notification Hub or email delivery, off-host objects, backup verification, or staging restore.",
			"The staging showcase and production observation/A1 redeploy receipt are missing and therefore fail this phase closed, even when every local contract passes.",
		},
	}, nil
}

func runCL3HealthSuite(ctx context.Context, root, name, relDir string, args, extraEnv, refs []string) CL3HealthSuiteResult {
	started := time.Now()
	result := CL3HealthSuiteResult{Name: name, Status: "failed", Command: append([]string(nil), args...), WorkingDir: relDir, EvidenceRefs: append([]string(nil), refs...)}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = filepath.Join(root, relDir)
	cmd.Env = cl3SafeTestEnv(extraEnv)
	var output cl3BoundedBuffer
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	result.DurationMS, result.Output = time.Since(started).Milliseconds(), output.String()
	if err == nil {
		result.Status, result.ExitCode = "passed", 0
	} else if exit, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exit.ExitCode()
	} else {
		result.ExitCode = -1
		result.Output += "\ncommand error: " + err.Error()
	}
	return result
}

func cl3SafeTestEnv(extra []string) []string {
	allowed := []string{"PATH", "HOME", "GOCACHE", "GOPATH", "GOMODCACHE", "GOTOOLCHAIN", "CGO_ENABLED"}
	env := make([]string, 0, len(allowed)+len(extra))
	for _, key := range allowed {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return append(env, extra...)
}

func cl3RepositoryRoot(scenarioPath string) (string, error) {
	if strings.TrimSpace(scenarioPath) == "" {
		return "", fmt.Errorf("scenario-to-cloud path is required")
	}
	path, err := filepath.Abs(scenarioPath)
	if err != nil {
		return "", fmt.Errorf("resolve scenario-to-cloud path: %w", err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve scenario-to-cloud path: %w", err)
	}
	if filepath.Base(path) != "scenario-to-cloud" || filepath.Base(filepath.Dir(path)) != "scenarios" {
		return "", fmt.Errorf("path must resolve to scenarios/scenario-to-cloud")
	}
	root := filepath.Dir(filepath.Dir(path))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	return root, nil
}

type cl3BoundedBuffer struct {
	bytes.Buffer
	truncated bool
}

func (b *cl3BoundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	const max = 32 << 10
	if b.Len() < max {
		remaining := max - b.Len()
		if len(p) > remaining {
			b.truncated = true
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}

func (b *cl3BoundedBuffer) String() string {
	output := b.Buffer.String()
	if b.truncated {
		output += "\n[output truncated at 32768 bytes]"
	}
	return output
}

func cl3HealthAssessment(scenario string, report *CL3HealthReport) *commonv1.MaturityAssessment {
	level, next := "L0", "L1"
	if report != nil && report.LocalContracts == "passed" {
		level, next = "L1", "L2"
	}
	findings := make([]*commonv1.AssessmentFinding, 0, 3)
	if report == nil || report.LocalContracts != "passed" {
		findings = append(findings, cl3Finding("cl3.local_contracts_failed", "CL3 local owner contracts did not pass", "Run the phase and inspect its owner test outputs.", "L0"))
	}
	if report == nil || report.StagingShowcase != "verified" {
		findings = append(findings, cl3Finding("cl3.staging_showcase_missing", "CL3 staging showcase receipts are missing", "Provide CL2 staging evidence for current-build rehearsal, mail and alert delivery, scheduled off-host backup, verification, and restore.", "L1"))
	}
	if report == nil || report.ProductionShowcase != "verified" {
		findings = append(findings, cl3Finding("cl3.production_showcase_missing", "CL3 production observation and A1 redeploy receipt are missing", "After staging passes, obtain A1 approval and retain the reviewed deploy receipt, current build identity, healthy status, and TLS observation.", "L2"))
	}
	blocking := make([]string, 0, len(findings))
	for _, finding := range findings {
		blocking = append(blocking, finding.Code)
	}
	severity := map[string]int32{}
	if len(findings) > 0 {
		severity["FINDING_SEVERITY_ERROR"] = int32(len(findings))
	}
	return &commonv1.MaturityAssessment{Scenario: scenario, Provider: "landing-page-business-suite", Phase: cl3HealthPhase, Version: "0.1.0", Findings: findings, FindingsBySeverity: severity,
		Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: next, BlockingFindingCodes: blocking, Clean: len(findings) == 0}}
}

func cl3Finding(code, title, remediation, localLevel string) *commonv1.AssessmentFinding {
	return &commonv1.AssessmentFinding{Code: code, Severity: "SEVERITY_ERROR", Title: title, Message: remediation, Remediation: remediation, FixClass: "manual",
		Maturity: &commonv1.FindingMaturity{CapabilityId: "production_health", LocalLevel: localLevel, GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP, Dimension: "operational-targets", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED}}
}

func cl3HealthResponse(scenario string, report *CL3HealthReport) *scenariovalidationv1.ValidateScenarioResponse {
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	if report != nil && report.Status == "passed" {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	}
	if report != nil && report.Status == "not_evaluated" {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED
	}
	raw, _ := json.Marshal(report)
	var reportDetail map[string]any
	_ = json.Unmarshal(raw, &reportDetail)
	detail, _ := structpb.NewStruct(map[string]any{"cl3_production_health": reportDetail})
	native, _ := anypb.New(detail)
	return &scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: status, NativeDetail: native, Assessment: cl3HealthAssessment(scenario, report), FailureClassification: "showcase_incomplete"}
}
