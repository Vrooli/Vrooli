package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"connectrpc.com/connect"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	operatorstateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate/operatorstatev1connect"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/profiles"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/profiles/profilesv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type semanticJourneyCorpus struct {
	SchemaVersion  int                    `json:"schema_version"`
	Artifact       string                 `json:"artifact"`
	Target         string                 `json:"target"`
	Normalization  []string               `json:"normalization"`
	Journeys       []semanticJourney      `json:"journeys"`
	NegativeChecks []semanticNegativeCase `json:"negative_controls"`
}

type semanticJourney struct {
	ID              string                               `json:"id"`
	Requirements    []string                             `json:"requirements"`
	Surfaces        []string                             `json:"surfaces"`
	Choices         map[string]map[string]map[string]any `json:"choices"`
	ExpectedState   map[string]bool                      `json:"expected_state"`
	ExpectedEffects []string                             `json:"expected_effects"`
}

type semanticNegativeCase struct {
	ID              string `json:"id"`
	Surface         string `json:"surface"`
	Input           string `json:"input"`
	ExpectedFailure string `json:"expected_failure"`
}

func loadSemanticCorpus(t *testing.T) semanticJourneyCorpus {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "semantic-journeys.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus semanticJourneyCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	return corpus
}

func TestSemanticJourneyCorpusIsRequirementAndArtifactBound(t *testing.T) {
	corpus := loadSemanticCorpus(t)
	if corpus.SchemaVersion != 1 || corpus.Artifact != "vrooli-onboarding" || corpus.Target == "" {
		t.Fatalf("corpus identity is incomplete: %+v", corpus)
	}
	if len(corpus.Normalization) == 0 || len(corpus.Journeys) == 0 || len(corpus.NegativeChecks) < 2 {
		t.Fatalf("corpus lacks normalization, journeys, or negative controls: %+v", corpus)
	}
	for _, journey := range corpus.Journeys {
		if journey.ID == "" || len(journey.Requirements) == 0 || len(journey.ExpectedState) == 0 || len(journey.ExpectedEffects) == 0 {
			t.Fatalf("journey %q is not requirement- and outcome-bound: %+v", journey.ID, journey)
		}
		for _, surface := range []string{"ui", "cli-pty", "cli-unattended", "api"} {
			if !containsString(journey.Surfaces, surface) {
				t.Fatalf("journey %q is missing real surface %q", journey.ID, surface)
			}
		}
	}
	for _, negative := range corpus.NegativeChecks {
		if negative.ID == "" || negative.Surface == "" || negative.Input == "" || negative.ExpectedFailure == "" {
			t.Fatalf("negative control is incomplete: %+v", negative)
		}
	}
}

func TestSemanticJourneyMatrixCoversAllPlanDeliverables(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "experience", "semantic-journey-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix struct {
		Deliverables []struct {
			ID       string   `json:"id"`
			Status   string   `json:"status"`
			Journeys []string `json:"journeys"`
			Evidence []string `json:"evidence"`
		} `json:"deliverables"`
	}
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, deliverable := range matrix.Deliverables {
		if seen[deliverable.ID] || len(deliverable.Journeys) == 0 {
			t.Fatalf("deliverable is duplicated or has no journey: %+v", deliverable)
		}
		seen[deliverable.ID] = true
	}
	for index := 1; index <= 25; index++ {
		id := "D" + twoDigits(index)
		if !seen[id] {
			t.Fatalf("semantic journey matrix omits %s", id)
		}
	}
	for _, deliverable := range matrix.Deliverables {
		if (deliverable.ID == "D15" || deliverable.ID == "D16") && (deliverable.Status != "implemented" || len(deliverable.Evidence) == 0) {
			t.Fatalf("phase-20 deliverable %s lacks implementation evidence: %+v", deliverable.ID, deliverable)
		}
	}
}

func TestGeneratedAPIAndBuiltCLIProduceFixtureOutcome(t *testing.T) {
	corpus := loadSemanticCorpus(t)
	journey := corpus.Journeys[0]

	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	srv := NewServer()
	httpServer := httptest.NewServer(srv.Handler())
	t.Cleanup(httpServer.Close)
	// The built CLI uses the supported token environment contract; keep this
	// fixture explicit so it cannot regress to OS-user or loopback admission.
	t.Setenv("VROOLI_AUTH_LOCAL_TOKEN", "test-personal-local-session")

	apiState := applyViaGeneratedAPI(t, httpServer.URL, journey.Choices)
	apiOutcome := normalizeSemanticState(apiState)
	if err := compareSemanticState(journey.ExpectedState, apiOutcome); err != nil {
		t.Fatal(err)
	}
	wrongExpected := cloneSemanticState(journey.ExpectedState)
	wrongExpected["resources.ollama.enabled"] = true
	if err := compareSemanticState(wrongExpected, apiOutcome); err == nil {
		t.Fatal("deliberately wrong expected state was accepted by the parity assertion")
	}

	statePath := operatorStateFixturePath(t, root)
	if err := os.Remove(statePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	cliPath := buildOnboardingCLI(t)
	patchPath := filepath.Join(t.TempDir(), "semantic-patch.json")
	patch, err := json.Marshal(journey.Choices)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(patchPath, patch, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(cliPath, "--api-base", httpServer.URL, "operator", "patch", "--body-file", patchPath, "--json")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("built CLI patch failed: %v\n%s", err, output)
	}
	cliState := getViaGeneratedAPI(t, httpServer.URL)
	if err := compareSemanticState(journey.ExpectedState, normalizeSemanticState(cliState)); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedCLIInputIsARealNegativeControl(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	srv := NewServer()
	httpServer := httptest.NewServer(srv.Handler())
	t.Cleanup(httpServer.Close)
	patchPath := filepath.Join(t.TempDir(), "malformed.json")
	if err := os.WriteFile(patchPath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(buildOnboardingCLI(t), "--api-base", httpServer.URL, "operator", "patch", "--body-file", patchPath, "--json")
	if output, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("malformed CLI input unexpectedly succeeded: %s", output)
	}
	if _, err := os.Stat(operatorStateFixturePath(t, root)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("malformed input mutated operator state: %v", err)
	}
}

func TestPurposeProfileJourneyUsesGeneratedAPIAndCLI(t *testing.T) {
	repoRoot := repositoryRootFromWorkingTree(t)
	t.Setenv("VROOLI_ROOT", repoRoot)
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())

	srv := NewServer()
	httpServer := httptest.NewServer(srv.Handler())
	t.Cleanup(httpServer.Close)

	profilesClient := profilesconnect.NewProfileServiceClient(http.DefaultClient, httpServer.URL)
	listed, err := profilesClient.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{Target: "local"}))
	if err != nil {
		t.Fatal(err)
	}
	if !containsProfileID(listed.Msg.GetProfiles(), "general-purpose") {
		t.Fatalf("generated API profile list omitted general-purpose: %+v", listed.Msg.GetProfiles())
	}

	answers, err := structpb.NewStruct(map[string]any{
		"purposes":           []any{"publish-apps"},
		"desktopPlatforms":   []any{"linux"},
		"mobilePlatforms":    []any{"android", "ios"},
		"hosting":            "managed-vps",
		"distributionMethod": "direct-download",
		"accountOwnership":   "operator-owned",
		"payments":           "stripe",
		"mail":               "provider",
		"downloadStorage":    "s3",
		"remoteTargets":      "connected-host",
	})
	if err != nil {
		t.Fatal(err)
	}
	presetAnswers, err := structpb.NewStruct(map[string]any{"purposes": []any{"develop-apps"}})
	if err != nil {
		t.Fatal(err)
	}
	targetContext, err := structpb.NewStruct(map[string]any{"catalogRevision": "catalog-r17"})
	if err != nil {
		t.Fatal(err)
	}
	evaluated, err := profilesClient.EvaluateProfile(context.Background(), connect.NewRequest(&profilesv1.EvaluateProfileRequest{
		Target: "local", ProfileId: "general-purpose", Answers: answers.Fields, TargetContext: targetContext,
		Preset: &profilesv1.ProfilePreset{Id: "customer-fixture", Version: "1.0.0", Source: "installation", Answers: presetAnswers.Fields},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !evaluated.Msg.GetValid() || evaluated.Msg.GetDigest() == "" || evaluated.Msg.GetCatalogRevision() != "catalog-r17" || evaluated.Msg.GetPreset().GetSource() != "installation" || len(evaluated.Msg.GetExplanations()) == 0 || !containsString(evaluated.Msg.GetScenarios(), "scenario-to-ios") || !containsString(evaluated.Msg.GetScenarios(), "scenario-to-cloud") {
		t.Fatalf("generated API profile evaluation did not produce publishing closure: %+v", evaluated.Msg)
	}
	customerAnswers, err := structpb.NewStruct(map[string]any{
		"optionalCapabilities": []any{"development", "managed-hosting"},
	})
	if err != nil {
		t.Fatal(err)
	}
	customer, err := profilesClient.EvaluateProfile(context.Background(), connect.NewRequest(&profilesv1.EvaluateProfileRequest{
		Target: "local", ProfileId: "customer-preinstalled", Answers: customerAnswers.Fields,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !customer.Msg.GetValid() || customer.Msg.GetProfile().GetProvenanceSource() != "customer-preset" || !containsString(customer.Msg.GetScenarios(), "landing-page-business-suite") {
		t.Fatalf("generated API customer profile evaluation lost provenance or closure: %+v", customer.Msg)
	}
	composedAnswers, err := structpb.NewStruct(map[string]any{"purposes": []any{"use-local-apps"}})
	if err != nil {
		t.Fatal(err)
	}
	composed, err := profilesClient.EvaluateProfile(context.Background(), connect.NewRequest(&profilesv1.EvaluateProfileRequest{
		Target: "local", ProfileIds: []string{"local-use", "customer-preinstalled"}, Answers: composedAnswers.Fields,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !composed.Msg.GetValid() || composed.Msg.GetProfile().GetId() != "composed" || len(composed.Msg.GetProfiles()) != 2 || !containsProfileID(composed.Msg.GetProfiles(), "local-use") || !containsProfileID(composed.Msg.GetProfiles(), "customer-preinstalled") {
		t.Fatalf("generated API did not preserve composed profile provenance: %+v", composed.Msg)
	}

	cliPath := buildOnboardingCLI(t)
	listOutput := runProfileCLI(t, cliPath, httpServer.URL, "profiles", "list", "--json")
	var listedJSON struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(listOutput, &listedJSON); err != nil {
		t.Fatalf("decode CLI profile list: %v\n%s", err, listOutput)
	}
	found := false
	for _, profile := range listedJSON.Profiles {
		if profile.ID == "general-purpose" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("CLI profile list omitted general-purpose: %s", listOutput)
	}

	evaluationOutput := runProfileCLI(t, cliPath, httpServer.URL, "profiles", "evaluate", "--profile-id", "general-purpose", "--answers", `{"purposes":["publish-apps"],"desktopPlatforms":["linux"],"mobilePlatforms":["android","ios"],"hosting":"managed-vps","distributionMethod":"direct-download","accountOwnership":"operator-owned","payments":"stripe","mail":"provider","downloadStorage":"s3","remoteTargets":"connected-host"}`, "--json")
	var evaluationJSON struct {
		Valid     bool     `json:"valid"`
		Scenarios []string `json:"scenarios"`
	}
	if err := json.Unmarshal(evaluationOutput, &evaluationJSON); err != nil {
		t.Fatalf("decode CLI profile evaluation: %v\n%s", err, evaluationOutput)
	}
	if !evaluationJSON.Valid || !containsString(evaluationJSON.Scenarios, "scenario-to-ios") || !containsString(evaluationJSON.Scenarios, "scenario-to-cloud") {
		t.Fatalf("CLI profile evaluation did not match generated API outcome: %s", evaluationOutput)
	}

	customerOutput := runProfileCLI(t, cliPath, httpServer.URL, "profiles", "evaluate", "--profile-id", "customer-preinstalled", "--answers", `{"optionalCapabilities":["development","managed-hosting"]}`, "--json")
	var customerJSON struct {
		Valid     bool     `json:"valid"`
		Scenarios []string `json:"scenarios"`
		Profile   struct {
			ProvenanceSource string `json:"provenance_source"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(customerOutput, &customerJSON); err != nil {
		t.Fatalf("decode CLI customer profile evaluation: %v\n%s", err, customerOutput)
	}
	if !customerJSON.Valid || customerJSON.Profile.ProvenanceSource != "customer-preset" || !containsString(customerJSON.Scenarios, "landing-page-business-suite") {
		t.Fatalf("CLI customer profile evaluation lost provenance or closure: %s", customerOutput)
	}

	composedOutput := runProfileCLI(t, cliPath, httpServer.URL, "profiles", "evaluate", "--profile-ids", "local-use", "--profile-ids", "customer-preinstalled", "--answers", `{"purposes":["use-local-apps"]}`, "--json")
	var composedJSON struct {
		Profile struct {
			ID string `json:"id"`
		} `json:"profile"`
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(composedOutput, &composedJSON); err != nil {
		t.Fatalf("decode CLI composed profile evaluation: %v\n%s", err, composedOutput)
	}
	if !composedJSON.Valid || composedJSON.Profile.ID != "composed" || len(composedJSON.Profiles) != 2 {
		t.Fatalf("CLI profile composition lost source provenance: %s", composedOutput)
	}
}

func repositoryRootFromWorkingTree(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for current := filepath.Clean(cwd); ; current = filepath.Dir(current) {
		marker := filepath.Join(current, "scenarios", "vrooli-onboarding", "profiles")
		if info, statErr := os.Stat(marker); statErr == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	t.Fatalf("could not locate Vrooli repository root from %s", cwd)
	return ""
}

func containsProfileID(profiles []*profilesv1.Profile, want string) bool {
	for _, profile := range profiles {
		if profile.GetId() == want {
			return true
		}
	}
	return false
}

func runProfileCLI(t *testing.T, cliPath, apiBase string, args ...string) []byte {
	t.Helper()
	commandArgs := append([]string{"--api-base", apiBase}, args...)
	output, err := exec.Command(cliPath, commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("profile CLI command failed: %v\n%s", err, output)
	}
	return output
}

func applyViaGeneratedAPI(t *testing.T, baseURL string, choices map[string]map[string]map[string]any) *operatorstatev1.OperatorState {
	t.Helper()
	data, err := json.Marshal(choices)
	if err != nil {
		t.Fatal(err)
	}
	state := new(operatorstatev1.OperatorState)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, state); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(choices))
	for key := range choices {
		paths = append(paths, camelCase(key))
	}
	sort.Strings(paths)
	client := operatorstateconnect.NewOperatorStateServiceClient(http.DefaultClient, baseURL)
	request := connect.NewRequest(&operatorstatev1.PatchOperatorStateRequest{
		Target: "local", State: state, UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
	})
	request.Header().Set("Authorization", "LocalSession test-personal-local-session")
	response, err := client.PatchOperatorState(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return response.Msg.GetState()
}

func getViaGeneratedAPI(t *testing.T, baseURL string) *operatorstatev1.OperatorState {
	t.Helper()
	client := operatorstateconnect.NewOperatorStateServiceClient(http.DefaultClient, baseURL)
	request := connect.NewRequest(&operatorstatev1.GetOperatorStateRequest{Target: "local"})
	request.Header().Set("Authorization", "LocalSession test-personal-local-session")
	response, err := client.GetOperatorState(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return response.Msg.GetState()
}

func normalizeSemanticState(state *operatorstatev1.OperatorState) map[string]bool {
	result := map[string]bool{}
	for name, choice := range state.GetScenarios() {
		result["scenarios."+name+".enabled"] = choice.GetEnabled()
		result["scenarios."+name+".auto_restart"] = choice.GetAutoRestart()
	}
	for name, choice := range state.GetResources() {
		result["resources."+name+".enabled"] = choice.GetEnabled()
	}
	for name, choice := range state.GetHostTools() {
		result["host_tools."+name+".opted_in"] = choice.GetOptedIn()
	}
	for name, choice := range state.GetHostSafeguards() {
		result["host_safeguards."+name+".opted_in"] = choice.GetOptedIn()
	}
	return result
}

func compareSemanticState(expected, actual map[string]bool) error {
	if reflect.DeepEqual(expected, actual) {
		return nil
	}
	return fmt.Errorf("semantic outcome differs from independent fixture: want %#v got %#v", expected, actual)
}

func cloneSemanticState(state map[string]bool) map[string]bool {
	clone := make(map[string]bool, len(state))
	for key, value := range state {
		clone[key] = value
	}
	return clone
}

func buildOnboardingCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("semantic CLI driver uses the Unix build toolchain in this lane")
	}
	path := filepath.Join(t.TempDir(), "vrooli-onboarding")
	cmd := exec.Command("go", "build", "-o", path, ".")
	cmd.Dir = filepath.Join("..", "cli")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build onboarding CLI: %v\n%s", err, output)
	}
	return path
}

func camelCase(value string) string {
	parts := strings.Split(value, "_")
	for index := 1; index < len(parts); index++ {
		if parts[index] != "" {
			parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
		}
	}
	return strings.Join(parts, "")
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10))
}
