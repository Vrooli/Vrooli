package setup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/operatorinput"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/testenv"
)

func requiredBlockedReport(reason hostreqkit.BlockingReason) runtime.Report {
	return runtime.Report{
		MissingRequired: []string{"buf"},
		Tools: []runtime.ItemStatus{{
			Name:           "buf",
			Required:       true,
			BlockingReason: reason,
		}},
	}
}

func TestSetupTerminalResultCategories(t *testing.T) {
	testenv.SetIdentityEnv(t, map[string]string{"HOME": t.TempDir()})
	tests := []struct {
		name      string
		stage     SetupPhase
		report    runtime.Report
		err       error
		category  string
		retryable bool
	}{
		{name: "ordinary success", stage: PhaseFinalize, category: SetupCategorySuccess},
		{name: "original mac requirement failure", stage: PhaseRequirements, report: requiredBlockedReport(hostreqkit.BlockingNeedsSudo), err: errors.New("buf requires privilege"), category: SetupCategoryRequiredRequirementBlocked, retryable: true},
		{name: "real unsupported host", stage: PhaseValidation, err: runtime.ErrUnsupportedPlatform, category: SetupCategoryUnsupportedPlatform},
		{name: "network or checksum failure", stage: PhaseRequirements, report: requiredBlockedReport(hostreqkit.BlockingNone), err: errors.New("checksum mismatch"), category: SetupCategoryRequiredRequirementBlocked, retryable: true},
		{name: "invalid configuration", stage: PhaseResolution, err: errors.New("invalid selector"), category: SetupCategoryInvalidConfiguration},
		{name: "partial state", stage: PhaseResources, err: errors.New("resource install failed"), category: SetupCategoryPartialState, retryable: true},
		{name: "partial bootstrap", stage: PhaseBootstrap, err: errors.New("network unavailable"), category: SetupCategoryPartialState, retryable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setupTerminalResult(tt.stage, tt.report, tt.err)
			if result.Version != SetupResultVersion || result.Category != tt.category || result.Retryable != tt.retryable {
				t.Fatalf("result = %#v", result)
			}
			if tt.category == SetupCategoryRequiredRequirementBlocked && len(result.BlockedRequirements) != 1 {
				t.Fatalf("blocked requirements = %#v", result.BlockedRequirements)
			}
		})
	}
}

func TestEveryDeclaredPhaseHasAResultCategory(t *testing.T) {
	allowedDefault := map[SetupPhase]bool{
		PhaseValidation: true, // validation failures have their host/report classifier.
		PhaseCompletion: true, // completion failures are transient after durable work.
	}
	for _, phase := range setupPhases {
		if category := phaseResultCategory(phase.ID); category == SetupCategoryTransientFailure && !allowedDefault[phase.ID] {
			t.Errorf("phase %q has no explicit result category", phase.ID)
		}
	}
}

func TestSetupTerminalResultPreservesTypedStageValue(t *testing.T) {
	result := setupTerminalResult(PhaseResources, runtime.Report{}, errors.New("resource failure"))
	if result.Stage != string(PhaseResources) {
		t.Fatalf("stage = %q, want %q", result.Stage, PhaseResources)
	}
}

func TestFinalSetupResultConfigurationReadsMarker(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	base := setupTerminalResult(PhaseFinalize, runtime.Report{}, nil)
	ready := &SetupReadiness{Status: ReadinessStatusReady, Source: ReadinessSourceInProcess}
	pending := finalizeSetupResultConfiguration(base, home, root, nil, ready)
	if !pending.ConfigurationPending || pending.Category != SetupCategoryConfigurationPending {
		t.Fatalf("pending result = %#v", pending)
	}
	if err := writeSetupCompleteMarker(t, home, root); err != nil {
		t.Fatalf("write bootstrap marker: %v", err)
	}
	if err := projectstate.MarkConfigurationComplete(home, root, "result-selection"); err != nil {
		t.Fatalf("mark configuration complete: %v", err)
	}
	complete := finalizeSetupResultConfiguration(base, home, root, nil, ready)
	if complete.ConfigurationPending || complete.Category != SetupCategorySuccess {
		t.Fatalf("complete result = %#v", complete)
	}
}

func TestSetupTerminalResultReportsConfigurationPendingWhenInputIsQueued(t *testing.T) {
	testenv.SetIdentityEnv(t, map[string]string{"HOME": t.TempDir()})
	if err := operatorinput.Replace([]operatorinput.Request{{
		ID:       "credential-store-passphrase",
		Kind:     operatorinput.KindSecret,
		Title:    "Credential-store passphrase",
		Required: true,
	}}); err != nil {
		t.Fatalf("queue operator input: %v", err)
	}

	result := setupTerminalResult(PhaseFinalize, runtime.Report{}, nil)
	if result.Status != SetupStatusSuccess || result.Category != SetupCategoryConfigurationPending {
		t.Fatalf("result = %#v, want successful configuration-pending result", result)
	}
	if !result.ConfigurationPending || result.Stage != "complete" {
		t.Fatalf("result = %#v, want configuration_pending=true and complete stage", result)
	}
}

func TestSetupTerminalResultReportsDegradedOptionalResources(t *testing.T) {
	result := setupTerminalResult(PhaseFinalize, runtime.Report{}, nil, []string{"reranker"})
	if result.Status != SetupStatusDegraded || result.Category != SetupCategoryDegraded {
		t.Fatalf("result = %#v, want degraded status/category", result)
	}
	if !reflect.DeepEqual(result.DegradedResources, []string{"reranker"}) {
		t.Fatalf("degraded resources = %#v", result.DegradedResources)
	}
	if !result.Retryable || result.Stage != "complete" {
		t.Fatalf("result = %#v, want retryable completed result", result)
	}
}

func TestWriteSetupResultProducesPrivateSeparateJSONTransport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "setup-result.json")
	want := SetupResult{Version: SetupResultVersion, Status: SetupStatusFailed, Category: SetupCategoryRequiredRequirementBlocked, Stage: "requirements", Retryable: true, BlockedRequirements: []string{"buf"}, Remediation: "retry"}
	if err := writeSetupResult(path, want); err != nil {
		t.Fatalf("writeSetupResult: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var got SetupResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %#v, want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("result permissions = %v, %v; want 0600", info.Mode(), err)
	}
}

func TestSetupResultOnboardingFieldIsAdditive(t *testing.T) {
	payload := SetupResult{
		Version: SetupResultVersion, Status: SetupStatusSuccess, Category: SetupCategoryConfigurationPending,
		Stage: "complete", Retryable: false, Remediation: "continue",
		Onboarding: &OnboardingResult{Decision: "url", PresentationKind: "remote-shell", URL: "http://127.0.0.1:1234"},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var old struct {
		Version string `json:"version"`
		Status  string `json:"status"`
		Stage   string `json:"stage"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		t.Fatalf("old decoder rejected additive payload: %v", err)
	}
	if old.Version != SetupResultVersion || old.Status != SetupStatusSuccess || old.Stage != "complete" {
		t.Fatalf("old fields = %+v", old)
	}
}
