package programsvalidation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	maturityassessment "github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
	"program-runtime/internal/module"
	programsinternal "program-runtime/internal/programs"
)

var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

type fixtureRunner interface {
	RunDeclaredProgram(context.Context, *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error)
}

func Module(repoRoot string, registry *bindings.Registry, runners ...fixtureRunner) module.Module {
	h := &handler{repoRoot: repoRoot, registry: registry}
	if len(runners) > 0 {
		h.runner = runners[0]
	}
	path, endpoint := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
	return module.Module{Name: "programs-validation", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: endpoint})
	}, Endpoints: Endpoints}
}

func (*handler) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{Provider: "program-runtime", Phase: "programs", SpecVersion: "2.0.0", Contract: "scenario-validation/v1", Capabilities: &scenariovalidationv1.ProviderCapabilities{SupportsExecution: true, DeliveryMode: "inline", TargetKinds: []commonv1.ValidationTargetKind{commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO}}}), nil
}

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	repoRoot string
	registry *bindings.Registry
	runner   fixtureRunner
}

func (h *handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario is required"))
	}
	findings := validateScenario(h.repoRoot, scenario, h.registry, req.Msg.GetIncludeExecution())
	if len(findings) == 0 && req.Msg.GetIncludeExecution() {
		findings = append(findings, h.executeFixtures(ctx, scenario)...)
	}
	clean := len(findings) == 0
	level := "L1"
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if clean {
		level = "L2"
	}
	if !clean {
		level = "L0"
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	levels := []*commonv1.LocalMaturityLevel{{Id: "L0", Name: "Program contract unavailable", StatusLabel: "Unavailable", NextUnlock: "Readable program declarations."}, {Id: "L1", Name: "Program contract inspectable", StatusLabel: "Foundation", NextUnlock: "Clear program validation findings."}, {Id: "L2", Name: "Program contract clean", StatusLabel: "Complete", CapabilitySummary: "Declared program contracts pass the requested static or execution checks."}}
	local := &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: "L2", Clean: clean, BlockingFindingCodes: findings, Levels: levels}
	assessmentValue := &commonv1.MaturityAssessment{Scenario: scenario, Provider: "program-runtime", Phase: "programs", Version: "1.0.0", Local: local}
	if clean {
		assessmentValue.Local.NextLevel = ""
	} else {
		assessmentValue.Local.NextLevel = "L1"
	}
	assessmentValue.Presentation = maturityassessment.BuildPhasePresentation(assessmentValue)
	detail, _ := structpb.NewStruct(map[string]any{"findings": findings, "include_execution": req.Msg.GetIncludeExecution()})
	native, _ := anypb.New(detail)
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: status, Assessment: assessmentValue, NativeDetail: native, Metrics: executionMetrics()}), nil
}

// validateScenario is deliberately filesystem-first: a validation phase must
// grade the declarations that will be shipped, not a separate in-memory test
// fixture. It also uses the same binding registry and preflight analyzer as
// submitted programs, keeping the phase's definition of "callable" aligned
// with production execution.
func validateScenario(repoRoot, scenario string, registry *bindings.Registry, includeExecution bool) []string {
	root := filepath.Join(repoRoot, "scenarios", scenario)
	if _, err := os.Stat(root); err != nil {
		return []string{"programs.scenario_missing"}
	}
	programRoot := filepath.Join(root, ".vrooli", "program-runtime")
	entries, err := os.ReadDir(programRoot)
	if err != nil {
		if os.IsNotExist(err) {
			// A scenario without declared programs is valid. Programs are earned by
			// repeated multi-operation workflows; absence is not a contract defect.
			return nil
		}
		return []string{"programs.programs_unreadable"}
	}
	if len(entries) == 0 {
		return nil
	}

	jsonFiles := map[string]string{}
	pyFiles := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		switch filepath.Ext(entry.Name()) {
		case ".json":
			jsonFiles[base] = filepath.Join(programRoot, entry.Name())
		case ".py":
			pyFiles[base] = filepath.Join(programRoot, entry.Name())
		}
	}
	findings := make([]string, 0)
	for base := range pyFiles {
		if _, ok := jsonFiles[base]; !ok {
			findings = append(findings, "programs.contract_missing")
		}
	}
	for base := range jsonFiles {
		if _, ok := pyFiles[base]; !ok {
			findings = append(findings, "programs.source_missing")
		}
	}

	index := contracts.NewIndex()
	if err := index.Load(repoRoot); err != nil {
		findings = append(findings, "programs.contract_index_unavailable")
	} else {
		for _, contract := range index.List() {
			if contract.Scenario != scenario || contract.ValidationError == "" {
				continue
			}
			findings = append(findings, "programs.contract_invalid")
		}
	}

	known := knownBindingNames(registry)
	bindingIDs := knownBindingIDs(registry)
	for base, contractPath := range jsonFiles {
		data, readErr := os.ReadFile(contractPath)
		if readErr != nil {
			continue
		}
		var contract struct {
			Bindings []struct {
				ID     string `json:"id"`
				Effect string `json:"effect"`
			} `json:"bindings"`
		}
		if json.Unmarshal(data, &contract) != nil {
			continue
		}
		for _, declared := range contract.Bindings {
			if _, ok := bindingIDs[declared.ID]; !ok && registry != nil {
				findings = append(findings, "programs.binding_missing")
			}
		}
		sourcePath, ok := pyFiles[base]
		if !ok {
			continue
		}
		source, readErr := os.ReadFile(sourcePath)
		if readErr != nil {
			continue
		}
		for _, diagnostic := range programsinternal.ResolveSource(string(source), known, filepath.Join(repoRoot, "scenarios", "program-runtime", "kernel", "host", "analyze.py")) {
			if diagnostic.GetSeverity() == "error" {
				findings = append(findings, "programs.preflight_diagnostic")
				break
			}
		}
		if !hasEnvelopePrint(string(source)) {
			findings = append(findings, "programs.envelope_missing")
		}
		if registry != nil {
			declared := make(map[string]struct{}, len(contract.Bindings))
			for _, binding := range contract.Bindings {
				declared[binding.ID] = struct{}{}
			}
			for _, binding := range registry.List("", "") {
				if binding.GetEffect() == "read" || binding.GetId() == "" {
					continue
				}
				name := strings.ReplaceAll(binding.GetScenario(), "-", "_") + "." + strings.ReplaceAll(binding.GetGroup(), "-", "_") + "." + strings.ReplaceAll(strings.ReplaceAll(binding.GetCommand(), "-", "_"), "/", ".")
				if strings.Contains(string(source), name+"(") {
					if _, ok := declared[binding.GetId()]; !ok {
						findings = append(findings, "programs.write_undeclared")
						break
					}
				}
			}
		}
		if includeExecution && !hasFixtures(data) {
			findings = append(findings, "programs.fixture_missing")
		} else if includeExecution && !fixturesAreWellFormed(data) {
			findings = append(findings, "programs.fixture_malformed")
		}
	}
	return uniqueStrings(findings)
}

func knownBindingIDs(registry *bindings.Registry) map[string]struct{} {
	result := map[string]struct{}{}
	if registry == nil {
		return result
	}
	for _, binding := range registry.List("", "") {
		result[binding.GetId()] = struct{}{}
	}
	return result
}

func knownBindingNames(registry *bindings.Registry) []string {
	names := []string{"discover", "recall", "guide", "validate", "capture", "ai", "agent", "gather", "describe", "reachable", "lib", "vrooli", "__vrooli__", "Handle"}
	if registry == nil {
		return names
	}
	for _, binding := range registry.List("", "") {
		scenario := strings.ReplaceAll(binding.GetScenario(), "-", "_")
		group := strings.ReplaceAll(binding.GetGroup(), "-", "_")
		command := strings.ReplaceAll(strings.ReplaceAll(binding.GetCommand(), "-", "_"), "/", ".")
		if scenario != "" {
			names = append(names, scenario, scenario+"."+group+"."+command)
		}
	}
	return names
}

func hasEnvelopePrint(source string) bool {
	return strings.Contains(source, "print(") && strings.Contains(source, "status")
}

func hasFixtures(data []byte) bool {
	var contract struct {
		Fixtures []json.RawMessage `json:"fixtures"`
	}
	return json.Unmarshal(data, &contract) == nil && len(contract.Fixtures) > 0
}

func fixturesAreWellFormed(data []byte) bool {
	var contract struct {
		Fixtures []struct {
			ID     string          `json:"id"`
			Inputs json.RawMessage `json:"inputs"`
			Expect json.RawMessage `json:"expect"`
		} `json:"fixtures"`
	}
	if json.Unmarshal(data, &contract) != nil || len(contract.Fixtures) == 0 {
		return false
	}
	for _, fixture := range contract.Fixtures {
		if strings.TrimSpace(fixture.ID) == "" || len(fixture.Expect) == 0 || len(fixture.Inputs) == 0 {
			return false
		}
	}
	return true
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func executionMetrics() *commonv1.ExecutionMetrics {
	now := time.Now()
	return &commonv1.ExecutionMetrics{StartedAt: timestamppb.New(now), CompletedAt: timestamppb.New(now)}
}

func (*handler) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("programs validation has no deterministic fixer"))
}

func (*handler) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("programs validation has no deterministic fixer"))
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "programs_validation_validate_scenario", Path: scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure, Method: "POST", Summary: "Validate program declarations and fixtures", Category: "validation"},
	{ID: "programs_validation_validate_target", Path: scenariovalidationconnect.ScenarioValidationServiceValidateTargetProcedure, Method: "POST", Summary: "Validate a program target", Category: "validation"},
	{ID: "programs_validation_describe_provider", Path: scenariovalidationconnect.ScenarioValidationServiceDescribeProviderProcedure, Method: "POST", Summary: "Describe the programs validation provider", Category: "validation"},
	{ID: "programs_validation_preview_fix", Path: scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure, Method: "POST", Summary: "Preview program validation fixes", Category: "validation"},
	{ID: "programs_validation_apply_fix", Path: scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure, Method: "POST", Summary: "Apply program validation fixes", Category: "validation"},
}

// Expectations are partial object assertions; status accepts a closed list of
// alternatives. Nested signals must be checked, not merely their presence.
func matchesExpectation(actual, expected any) bool {
	switch want := expected.(type) {
	case map[string]any:
		got, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for key, value := range want {
			if !matchesExpectation(got[key], value) {
				return false
			}
		}
		return true
	case []any:
		if _, ok := actual.([]any); ok {
			return reflect.DeepEqual(actual, expected)
		}
		for _, value := range want {
			if reflect.DeepEqual(actual, value) {
				return true
			}
		}
		return false
	default:
		return reflect.DeepEqual(actual, expected)
	}
}
func (h *handler) executeFixtures(ctx context.Context, scenario string) []string {
	paths, err := filepath.Glob(filepath.Join(h.repoRoot, "scenarios", scenario, ".vrooli", "program-runtime", "*.json"))
	if err != nil {
		return []string{"programs.fixtures_unreadable"}
	}
	index := contracts.NewIndex()
	if err := index.Load(h.repoRoot); err != nil {
		return []string{"programs.contract_index_unavailable"}
	}
	findings := []string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, "programs.fixtures_unreadable")
			continue
		}
		var declaration struct {
			Name     string `json:"name"`
			Fixtures []struct {
				ID     string         `json:"id"`
				Inputs map[string]any `json:"inputs"`
				Expect map[string]any `json:"expect"`
			} `json:"fixtures"`
		}
		if json.Unmarshal(data, &declaration) != nil {
			findings = append(findings, "programs.fixture_malformed")
			continue
		}
		parts := strings.SplitN(declaration.Name, ".", 2)
		if len(parts) != 2 {
			continue
		}
		contract, ok := index.Get(parts[0], parts[1])
		if !ok {
			findings = append(findings, "programs.contract_missing")
			continue
		}
		for _, fixture := range declaration.Fixtures {
			prefix := "programs.fixture:" + declaration.Name + ":" + fixture.ID + ":"
			if h.runner == nil {
				findings = append(findings, prefix+"runner_unavailable")
				continue
			}
			input, err := structpb.NewStruct(fixture.Inputs)
			if err != nil {
				findings = append(findings, prefix+"invalid_input")
				continue
			}
			result, err := h.runner.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: declaration.Name, ExpectedDigest: contract.Digest, Inputs: input, Provenance: programsv1.Provenance_PROVENANCE_TEST}))
			if err != nil || result == nil || !result.Msg.GetTerminal() || result.Msg.GetProgram().GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
				findings = append(findings, prefix+"execution_unavailable")
				continue
			}
			stdout := result.Msg.GetProgram().GetStdout()
			var envelope map[string]any
			if int64(len(stdout)) > contract.OutputBytes || json.Unmarshal([]byte(stdout), &envelope) != nil {
				findings = append(findings, prefix+"invalid_envelope")
				continue
			}
			if contract.OutputSchema != nil {
				if err := contract.OutputSchema.Validate(envelope); err != nil {
					findings = append(findings, prefix+"output_schema_mismatch")
					continue
				}
			}
			if !matchesExpectation(envelope, fixture.Expect) {
				findings = append(findings, prefix+"expectation_mismatch")
			}
		}
	}
	return findings
}
