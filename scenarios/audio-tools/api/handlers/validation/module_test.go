package validation

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"audio-tools/internal/conformance"
	"audio-tools/internal/soak"
	"audio-tools/internal/stt/session"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateScenarioDoesNotExecuteWhenExecutionIsNotRequested(t *testing.T) {
	runs := 0
	h := &handler{
		deps: Deps{RunSoak: func(context.Context, soak.Options, *session.Registry) (soak.Result, error) {
			runs++
			return soak.Result{}, nil
		}},
		spec: testSpec(),
	}

	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "audio-tools",
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED, response.Msg.GetStatus())
	require.Zero(t, runs)
}

func TestManagedProviderRequiresOwnerReceiptsAndUsesBrokerSession(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../../")
	require.NoError(t, err)
	t.Setenv("VROOLI_ROOT", repoRoot)
	t.Setenv("BAS_API_URL", "http://bas.test")
	t.Setenv("SOAK_UI_URL", "http://audio-tools.test")
	t.Setenv("SOAK_ENGINE_ID", "engine-qualified")
	t.Setenv("SOAK_MODEL_ID", "model-qualified")
	t.Setenv("VROOLI_AUDIO_SOAK_REPLAY", "")

	verified := 0
	runs := 0
	h := &handler{
		deps: Deps{
			ScenarioDir: filepath.Join(repoRoot, "scenarios", "audio-tools"),
			VerifyDemandLease: func(_ context.Context, baseURL, receipt string) error {
				verified++
				require.Equal(t, "http://test-genie.test", baseURL)
				require.Equal(t, "opaque-tg-receipt", receipt)
				return nil
			},
			RunSoak: func(_ context.Context, opt soak.Options, _ *session.Registry) (soak.Result, error) {
				runs++
				require.Equal(t, conformance.LaneAccelerated, opt.Lane)
				require.Empty(t, opt.Surface)
				require.Equal(t, "virtual-replay", opt.EngineID)
				require.FileExists(t, opt.Fixture)
				return soak.Result{}, errors.New("controlled test stop")
			},
			PersistEvidence: func(conformance.Run) (string, error) { return "", nil },
		},
		spec: testSpec(),
	}
	t.Setenv("TEST_GENIE_API_URL", "http://test-genie.test")
	req := connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "audio-tools", IncludeExecution: true})
	req.Header().Add("X-Agent-Identity-Token", "run-identity")
	req.Header().Add("X-Vrooli-Browser-Grant-Receipt", "run-browser-grant")
	req.Header().Add("X-Test-Genie-Lease-Receipt", "opaque-tg-receipt")
	response, err := h.ValidateScenario(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, response.Msg.GetStatus())
	require.Equal(t, 1, verified)
	require.Equal(t, 1, runs)
}

func TestManagedProviderPublishesPipeWireMetadataWithoutBearerOrTranscript(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../../")
	require.NoError(t, err)
	t.Setenv("VROOLI_ROOT", repoRoot)
	t.Setenv("BAS_API_URL", "http://bas.test")
	t.Setenv("SOAK_UI_URL", "http://audio-tools.test")
	t.Setenv("SOAK_ENGINE_ID", "engine-qualified")
	t.Setenv("SOAK_MODEL_ID", "model-qualified")
	t.Setenv("TEST_GENIE_API_URL", "http://test-genie.test")
	t.Setenv("VROOLI_AUDIO_SOAK_REPLAY", "")

	run := conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "managed-private-evidence-check",
		Lane:          conformance.LaneRealtime,
		Profile:       "continuous",
		Cell:          conformance.Cell{EngineID: "engine-qualified", ModelID: "model-qualified", Strategy: "product", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "capture-sha", Server: "server-sha"},
		DurationMs:    1000,
	}
	for _, name := range conformance.RequiredAssertions(conformance.LaneRealtime) {
		run.Assertions = append(run.Assertions, conformance.Assertion{Name: name, Outcome: conformance.OutcomePassed, Detail: "observed"})
	}
	var persisted string
	verifyCount := 0
	h := &handler{
		deps: Deps{
			ScenarioDir: filepath.Join(repoRoot, "scenarios", "audio-tools"),
			VerifyDemandLease: func(_ context.Context, baseURL, receipt string) error {
				verifyCount++
				require.Equal(t, "http://test-genie.test", baseURL)
				require.Equal(t, "private-tg-lease", receipt)
				return nil
			},
			RunSoak: func(_ context.Context, opt soak.Options, _ *session.Registry) (soak.Result, error) {
				require.Equal(t, conformance.LaneAccelerated, opt.Lane)
				require.FileExists(t, opt.Fixture)
				return soak.Result{
					Run: run,
					AudioDeviceEvidence: &soak.AudioDeviceEvidence{
						Enumerated: true, SelectedLabel: "PipeWire", SampleRate: 16000, ChannelCount: 1,
					},
				}, nil
			},
			PersistEvidence: func(got conformance.Run) (string, error) {
				var b strings.Builder
				require.NoError(t, got.WriteJSON(&b))
				persisted = b.String()
				return "audio-tools://evidence/managed-private-evidence-check", nil
			},
		},
		spec: testSpec(),
	}
	req := connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "audio-tools", IncludeExecution: true})
	req.Header().Add("X-Agent-Identity-Token", "private-agent-token")
	req.Header().Add("X-Vrooli-Browser-Grant-Receipt", "private-browser-grant")
	req.Header().Add("X-Test-Genie-Lease-Receipt", "private-tg-lease")
	response, err := h.ValidateScenario(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, response.Msg.GetStatus())
	require.Equal(t, 1, verifyCount)
	require.NotEmpty(t, persisted)

	var native structpb.Struct
	require.NoError(t, response.Msg.GetNativeDetail().UnmarshalTo(&native))
	device, ok := native.AsMap()["audio_device_evidence"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "PipeWire", device["selected_label"])
	require.Equal(t, float64(16000), device["sample_rate"])

	responseJSON, err := json.Marshal(response.Msg)
	require.NoError(t, err)
	for _, secret := range []string{"private-agent-token", "private-browser-grant", "private-tg-lease", "private transcript"} {
		require.NotContains(t, persisted, secret)
		require.NotContains(t, string(responseJSON), secret)
	}
}

func TestManagedProviderRejectsPartialReceiptsBeforeRunning(t *testing.T) {
	runs := 0
	h := &handler{
		deps: Deps{
			ScenarioDir: t.TempDir(),
			RunSoak: func(context.Context, soak.Options, *session.Registry) (soak.Result, error) {
				runs++
				return soak.Result{}, nil
			},
			VerifyDemandLease: func(context.Context, string, string) error { t.Fatal("must not verify partial receipts"); return nil },
		},
		spec: testSpec(),
	}
	req := connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "audio-tools", IncludeExecution: true})
	req.Header().Add("X-Agent-Identity-Token", "run-identity")
	req.Header().Add("X-Test-Genie-Lease-Receipt", "opaque-tg-receipt")
	response, err := h.ValidateScenario(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, response.Msg.GetStatus())
	require.Contains(t, response.Msg.GetAssessment().GetFindings()[0].GetMessage(), "requires one verified run identity")
	require.Zero(t, runs)
}

func TestValidateScenarioFailsClosedWhenQualificationInputsAreMissing(t *testing.T) {
	runs := 0
	h := &handler{
		deps: Deps{
			ScenarioDir: t.TempDir(),
			RunSoak: func(context.Context, soak.Options, *session.Registry) (soak.Result, error) {
				runs++
				return soak.Result{}, nil
			},
		},
		spec: testSpec(),
	}

	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         "audio-tools",
		IncludeExecution: true,
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, response.Msg.GetStatus())
	require.Contains(t, response.Msg.GetAssessment().GetFindings()[0].GetMessage(), "requires one verified run identity")
	require.Zero(t, runs)
}

func testSpec() *assessment.Spec {
	return &assessment.Spec{
		Provider: "audio-tools",
		Phase:    "soak",
		Version:  "1.0.0",
		Capabilities: []assessment.CapabilitySpec{{
			ID: "qualification", Label: "Qualification",
			Levels: []assessment.Level{
				{ID: "L0", Name: "Unavailable", Description: "Unavailable", EntryCriteria: []string{}, ExitCriteria: []string{"artifact"}},
				{ID: "L1", Name: "Qualified", Description: "Qualified", EntryCriteria: []string{"artifact"}, ExitCriteria: []string{}},
			},
		}},
		Findings: map[string]assessment.FindingMapping{
			"SOAK_CONFIGURATION_MISSING": {
				CapabilityID: "qualification", LocalLevelImpact: "L0", GlobalImpact: assessment.ImpactFoundationBlocker,
				Dimension: "tests", SeverityDefault: "SEVERITY_ERROR", CleanRequirement: "required",
				FixClass: assessment.FixClassManual, FixReason: "test input is deployment-specific",
			},
			"SOAK_QUALIFICATION_FAILED": {
				CapabilityID: "qualification", LocalLevelImpact: "L0", GlobalImpact: assessment.ImpactSafetyBlocker,
				Dimension: "performance", SeverityDefault: "SEVERITY_ERROR", CleanRequirement: "required",
				FixClass: assessment.FixClassManual, FixReason: "artifact requires investigation",
			},
		},
		Fallback: assessment.FallbackPolicy{CapabilityID: "qualification", LocalLevelImpact: "L0", GlobalImpact: assessment.ImpactUnknown, Dimension: "tests", SeverityDefault: "SEVERITY_ERROR", CleanRequirement: "required"},
	}
}
