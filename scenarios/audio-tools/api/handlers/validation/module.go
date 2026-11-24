//go:build audio_lab

// Package validation exposes the audio-tools product qualification as a
// Test Genie validation-provider phase. The provider owns the browser soak;
// Test Genie owns only phase orchestration and verdict transport.
package validation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	soakapi "audio-tools/handlers/soak"
	"audio-tools/internal/conformance"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/soak"
	"audio-tools/internal/stt/session"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gorilla/mux"
)

const (
	providerScenario = "audio-tools"
	providerPhase    = "soak"
	defaultReference = "the quick brown fox jumps."
)

// Deps keeps the browser and evidence seams injectable for focused provider
// tests. Production uses the real driver and persistence functions.
type Deps struct {
	ScenarioDir     string
	StreamLedgers   *session.Registry
	RunSoak         func(context.Context, soak.Options, *session.Registry) (soak.Result, error)
	PersistEvidence func(conformance.Run) (string, error)
	// VerifyDemandLease keeps the Test Genie demand lease as a separately
	// owner-issued prerequisite to the AM browser grant. Production uses the
	// Test Genie receipt endpoint; tests inject only this verifier boundary.
	VerifyDemandLease func(context.Context, string, string) error
}

func Module(d Deps) modulekit.Module {
	if strings.TrimSpace(d.ScenarioDir) == "" {
		d.ScenarioDir = discoverScenarioDir()
	}
	if d.RunSoak == nil {
		d.RunSoak = soak.RunWithEvidence
	}
	if d.PersistEvidence == nil {
		d.PersistEvidence = soak.PersistEvidence
	}
	if d.VerifyDemandLease == nil {
		d.VerifyDemandLease = func(ctx context.Context, baseURL, receipt string) error {
			return soakapi.VerifyAudioToolsDemandLease(ctx, baseURL, receipt, &http.Client{Timeout: 5 * time.Second})
		}
	}

	describer, _ := assessment.LoadDescriber(d.ScenarioDir)
	provider := &handler{deps: d, spec: loadSpec(d.ScenarioDir)}
	path, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(
		assessment.Serve(provider, describer),
	)
	return modulekit.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	deps Deps
	spec *assessment.Spec
}

func (h *handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h == nil || h.spec == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("audio-tools soak provider descriptor is unavailable"))
	}
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		scenario = providerScenario
	}
	if !req.Msg.GetIncludeExecution() {
		assessmentResult, err := h.buildAssessment(scenario, nil)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		response, err := assessment.BuildValidationResponse(
			scenario, assessmentResult, nil, nil,
			assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED),
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(response), nil
	}

	collector := metrics.Start()
	ownerCtx, err := h.managedExecutionContext(ctx, req.Header())
	if err != nil {
		return h.failedResponse(scenario, collector, "SOAK_CONFIGURATION_MISSING", err)
	}
	options, err := h.options(ownerCtx)
	if err != nil {
		return h.failedResponse(scenario, collector, "SOAK_CONFIGURATION_MISSING", err)
	}
	result, runErr := h.deps.RunSoak(ownerCtx, options, h.deps.StreamLedgers)
	artifactRef := ""
	if h.deps.PersistEvidence != nil {
		artifactRef, err = h.deps.PersistEvidence(result.Run)
		if err != nil && runErr == nil {
			runErr = fmt.Errorf("persist soak evidence: %w", err)
		}
	}
	verdict := result.Run.Evaluate()
	if runErr != nil || !verdict.Qualified {
		detail := strings.TrimSpace(firstNonEmpty(runErrString(runErr), strings.Join(verdict.Reasons, "; ")))
		if artifactRef != "" {
			detail += "; artifact=" + artifactRef
		}
		return h.failedResponse(scenario, collector, "SOAK_QUALIFICATION_FAILED", errors.New(detail))
	}

	assessmentResult, err := h.buildAssessment(scenario, nil)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	nativeFields := map[string]any{
		"run_id":                 result.Run.RunID,
		"lane":                   string(result.Run.Lane),
		"simulated_minutes":      result.Run.SimulatedMinutes,
		"assertion_count":        len(result.Run.Assertions),
		"artifact_ref":           artifactRef,
		"qualified":              verdict.Qualified,
		"provider_engine_id":     result.Run.Cell.EngineID,
		"provider_model_id":      result.Run.Cell.ModelID,
		"managed_broker_session": true,
		"mobile_recovery_passed": result.MobileRecoveryPassed,
	}
	if result.AudioDeviceEvidence != nil {
		nativeFields["audio_device_evidence"] = map[string]any{
			"enumerated":     result.AudioDeviceEvidence.Enumerated,
			"selected_label": result.AudioDeviceEvidence.SelectedLabel,
			"sample_rate":    result.AudioDeviceEvidence.SampleRate,
			"channel_count":  result.AudioDeviceEvidence.ChannelCount,
		}
	}
	native, _ := structpb.NewStruct(nativeFields)
	response, err := assessment.BuildValidationResponse(scenario, assessmentResult, native, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build soak validation response: %w", err))
	}
	return connect.NewResponse(response), nil
}

func (h *handler) failedResponse(scenario string, collector *metrics.Collector, code string, cause error) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	assessmentResult, err := h.buildAssessment(scenario, &assessment.Finding{
		Code:        code,
		Severity:    "SEVERITY_ERROR",
		Title:       "Long-form dictation soak did not qualify",
		Message:     cause.Error(),
		Location:    "audio-tools validation provider",
		Remediation: "Run the registered soak phase with a verified Test Genie lease and BAS session grant, then inspect the persisted conformance artifact before promoting the provider cell.",
		FixClass:    "manual",
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response, err := assessment.BuildValidationResponse(
		scenario, assessmentResult, nil, collector.Stop(),
		assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build failed soak response: %w", err))
	}
	return connect.NewResponse(response), nil
}

func (h *handler) buildAssessment(scenario string, finding *assessment.Finding) (*commonv1.MaturityAssessment, error) {
	findings := []assessment.Finding{}
	if finding != nil {
		findings = append(findings, *finding)
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{Scenario: scenario, Spec: *h.spec, Findings: findings})
}

func (h *handler) managedExecutionContext(ctx context.Context, headers http.Header) (context.Context, error) {
	identity, hasIdentity := exactlyOneHeader(headers, "X-Agent-Identity-Token")
	grant, hasGrant := exactlyOneHeader(headers, "X-Vrooli-Browser-Grant-Receipt")
	lease, hasLease := exactlyOneHeader(headers, "X-Test-Genie-Lease-Receipt")
	if !hasIdentity || !hasGrant || !hasLease {
		missing := make([]string, 0, 3)
		if !hasIdentity {
			missing = append(missing, "exactly one Agent Manager identity token")
		}
		if !hasGrant {
			missing = append(missing, "exactly one BAS browser grant receipt")
		}
		if !hasLease {
			missing = append(missing, "exactly one Test Genie lease receipt")
		}
		return ctx, fmt.Errorf("soak is missing %s", strings.Join(missing, ", "))
	}
	testGenieURL := firstNonEmpty(os.Getenv("TEST_GENIE_API_URL"), os.Getenv("TEST_GENIE_URL"))
	if testGenieURL == "" {
		return ctx, errors.New("Test Genie lease verifier is not configured")
	}
	if err := h.deps.VerifyDemandLease(ctx, testGenieURL, lease); err != nil {
		return ctx, errors.New("Test Genie did not verify an active Audio Tools lease")
	}
	brokerURL := firstNonEmpty(os.Getenv("BROWSER_AUTOMATION_STUDIO_API_URL"), os.Getenv("BAS_API_URL"))
	if brokerURL == "" {
		return ctx, errors.New("BAS Agent Access endpoint is not configured")
	}
	if !strings.HasSuffix(strings.TrimRight(brokerURL, "/"), "/api/v1") {
		brokerURL = strings.TrimRight(brokerURL, "/") + "/api/v1"
	}
	fixture, err := soak.FixedDictationFixture(os.Getenv("VROOLI_ROOT"))
	if err != nil {
		return ctx, errors.New("fixed Audio Tools qualification fixture is unavailable")
	}
	ctx = soak.WithBrowserOwnerCredentials(ctx, identity, grant, lease)
	ctx = soak.WithManagedBrowserOwner(ctx, brokerURL)
	ctx = context.WithValue(ctx, managedFixtureKey{}, fixture)
	return ctx, nil
}

type managedFixtureKey struct{}

func exactlyOneHeader(headers http.Header, name string) (string, bool) {
	values := headers.Values(name)
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", false
	}
	return strings.TrimSpace(values[0]), true
}

func (h *handler) options(ctx context.Context) (soak.Options, error) {
	fixture, ok := ctx.Value(managedFixtureKey{}).(string)
	if !ok || fixture == "" {
		return soak.Options{}, errors.New("fixed Audio Tools qualification fixture is unavailable")
	}
	uiURL := firstNonEmpty(os.Getenv("SOAK_UI_URL"), os.Getenv("AUDIO_TOOLS_UI_URL"), os.Getenv("UI_BASE_URL"))
	if uiURL == "" {
		return soak.Options{}, errors.New("SOAK_UI_URL/AUDIO_TOOLS_UI_URL is required for the BAS browser task")
	}
	if _, err := os.Stat(fixture); err != nil {
		return soak.Options{}, fmt.Errorf("soak fixture %q: %w", fixture, err)
	}
	engineID := strings.TrimSpace(os.Getenv("SOAK_ENGINE_ID"))
	modelID := strings.TrimSpace(os.Getenv("SOAK_MODEL_ID"))
	if engineID == "" || modelID == "" {
		return soak.Options{}, errors.New("SOAK_ENGINE_ID and SOAK_MODEL_ID are required for a managed realtime qualification cell")
	}
	return soak.Options{
		UIURL:            strings.TrimRight(uiURL, "/"),
		Fixture:          fixture,
		Lane:             conformance.LaneAccelerated,
		Profile:          "continuous",
		Turns:            3,
		FeedMS:           2000,
		Reference:        firstNonEmpty(os.Getenv("SOAK_REFERENCE_TEXT"), defaultReference),
		EngineID:         "virtual-replay",
		ModelID:          "virtual-corpus",
		Strategy:         "passthrough",
		Policy:           "default",
		Shape:            "burst",
		SimulatedMinutes: 60,
	}, nil
}

func loadSpec(dir string) *assessment.Spec {
	spec, err := assessment.LoadSpecFromScenario(dir)
	if err != nil {
		return nil
	}
	return spec
}

func discoverScenarioDir() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return filepath.Join(root, "scenarios", providerScenario)
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "scenarios", providerScenario)
			if _, err := os.Stat(filepath.Join(candidate, ".vrooli", "test-genie.json")); err == nil {
				return candidate
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func runErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "validation.soak.validate_scenario", Path: scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure, Method: "POST", Summary: "Run the accelerated long-form dictation qualification", Category: "validation"},
	{ID: "validation.soak.describe_provider", Path: scenariovalidationconnect.ScenarioValidationServiceDescribeProviderProcedure, Method: "POST", Summary: "Describe the long-form dictation qualification provider", Category: "validation"},
}
