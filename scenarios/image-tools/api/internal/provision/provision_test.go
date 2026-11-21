package provision

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"image-tools/internal/capabilities"
	"image-tools/internal/jobs"
	"image-tools/internal/models"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	reg, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	return &Service{Installer: &models.Installer{Root: t.TempDir(), Reg: reg}}
}

func requestJob(t *testing.T, ids ...string) jobs.Job {
	t.Helper()
	raw, err := json.Marshal(Request{ModelIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	return jobs.Job{ID: "provision-test", Payload: raw}
}

func evidence(t *testing.T, result jobs.Result) Report {
	t.Helper()
	var report Report
	if err := json.Unmarshal([]byte(result.Meta[EvidenceKey]), &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != 1 || report.StartedAt.IsZero() || report.FinishedAt.Before(report.StartedAt) {
		t.Fatalf("invalid report: %+v", report)
	}
	return report
}

func TestProvisionPreparesRuntimeOnceAndRetainsEveryModel(t *testing.T) {
	s := fixture(t)
	runtimes, backends := 0, 0
	s.RuntimeNeeded = func(models.Model) bool { return true }
	s.EnsureRuntime = func(context.Context, func(int, string)) error { runtimes++; return nil }
	s.EnsureBackend = func(context.Context, models.Model) error { backends++; return nil }
	result, err := s.Run(context.Background(), requestJob(t, "naturalize-detail-v1", "normals-from-depth", "mask-fill"), nil)
	if err != nil {
		t.Fatal(err)
	}
	report := evidence(t, result)
	if !report.Ready || len(report.Models) != 3 || runtimes != 1 || backends != 3 {
		t.Fatalf("report=%+v runtimes=%d backends=%d", report, runtimes, backends)
	}
	for _, model := range report.Models {
		if !model.Lifecycle.Ready() {
			t.Fatalf("not prepared: %+v", model)
		}
	}
}

func TestProvisionHostFitReportsReadyOnlyWithCompleteCoverage(t *testing.T) {
	s := fixture(t)
	plan := Plan{Mode: PlanHostFit, SelectedModelIDs: []string{"naturalize-detail-v1"}, CoveredOperations: []string{"naturalize"}, CoverageComplete: true}
	result, err := s.Run(context.Background(), requestJobWithPlan(t, plan, "naturalize-detail-v1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	report := evidence(t, result)
	if !report.Ready || !report.CoverageComplete || report.Mode != PlanHostFit || len(report.Models) != 1 {
		t.Fatalf("complete host-fit coverage was not ready: %+v", report)
	}
}

func TestHostFitPlanMinimizesModelSetAndExplainsUnsupportedCandidates(t *testing.T) {
	const gib = uint64(1 << 30)
	modelsForPlan := []models.Model{
		{ID: "shared-small", Operations: []string{"edit", "remove"}, Tier: models.TierDefault, SizeMBApprox: 100, Enabled: true, CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes}, Hardware: models.Hardware{CPUCapable: true, MinRAMGB: 4}},
		{ID: "edit-alone", Operations: []string{"edit"}, Tier: models.TierQuality, SizeMBApprox: 50, Enabled: true, CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes}, Hardware: models.Hardware{CPUCapable: true, MinRAMGB: 4}},
		{ID: "impossible-large", Operations: []string{"edit"}, Enabled: true, CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes}, Hardware: models.Hardware{GPURequired: true, MinVRAMGB: 24, MinRAMGB: 32}},
		{ID: "unverified", Operations: []string{"remove"}, Enabled: true, CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseConditional}},
		{ID: "disabled", Operations: []string{"edit", "remove"}, Enabled: false, CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes}, Hardware: models.Hardware{CPUCapable: true}},
	}
	host := capabilities.Host{OS: "linux", Arch: "amd64", TotalMemoryBytes: 20 * gib, AvailableMemoryBytes: 18 * gib, GPUs: []capabilities.GPU{{VRAMBytes: 16 * gib}}}
	plan := HostFitPlan(modelsForPlan, []string{"edit", "remove", "inpaint"}, host, nil, nil)
	if plan.Mode != PlanHostFit || plan.CoverageComplete {
		t.Fatalf("unexpected plan status: %+v", plan)
	}
	if len(plan.SelectedModelIDs) != 1 || plan.SelectedModelIDs[0] != "shared-small" {
		t.Fatalf("selected %v, want minimal shared model", plan.SelectedModelIDs)
	}
	if len(plan.CoveredOperations) != 2 || plan.CoveredOperations[0] != "edit" || plan.CoveredOperations[1] != "remove" {
		t.Fatalf("covered operations: %v", plan.CoveredOperations)
	}
	if len(plan.UncoveredOperations) != 1 || plan.UncoveredOperations[0].Operation != "inpaint" {
		t.Fatalf("uncovered operations: %+v", plan.UncoveredOperations)
	}
	excluded := map[string]string{}
	for _, candidate := range plan.ExcludedModels {
		excluded[candidate.ModelID] = candidate.Reason
	}
	if excluded["impossible-large"] != "requires_32_gib_ram_host_has_18_gib_available" {
		t.Fatalf("large model exclusion not explained: %v", excluded)
	}
	if excluded["unverified"] != "commercial_use_not_verified" {
		t.Fatalf("rights gate not preserved: %v", excluded)
	}
	if _, ok := excluded["edit-alone"]; !ok {
		t.Fatalf("candidate omitted by minimal set not reported: %v", excluded)
	}
	if _, ok := excluded["disabled"]; ok {
		t.Fatalf("disabled model should not be considered a candidate: %v", excluded)
	}
}

func TestHostFitPlanRejectsImpossibleVRAMEvenWhenRAMFits(t *testing.T) {
	const gib = uint64(1 << 30)
	candidate := models.Model{ID: "gpu-only", Operations: []string{"generate"}, Enabled: true,
		CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes},
		Hardware:         models.Hardware{GPURequired: true, MinVRAMGB: 24, MinRAMGB: 8}}
	plan := HostFitPlan([]models.Model{candidate}, []string{"generate"}, capabilities.Host{
		OS: "linux", Arch: "amd64", AvailableMemoryBytes: 18 * gib,
		GPUs: []capabilities.GPU{{VRAMBytes: 16 * gib}},
	}, nil, nil)
	if len(plan.SelectedModelIDs) != 0 || len(plan.UncoveredOperations) != 1 {
		t.Fatalf("impossible GPU model was admitted: %+v", plan)
	}
	if len(plan.ExcludedModels) != 1 || plan.ExcludedModels[0].Reason != "requires_24_gib_vram" {
		t.Fatalf("VRAM exclusion missing: %+v", plan.ExcludedModels)
	}
}

func TestEnabledOperationsRetainsUnsupportedEnabledCapabilities(t *testing.T) {
	candidates := []models.Model{
		{ID: "host-fit", Operations: []string{"remove"}, Enabled: true},
		{ID: "rights-review", Operations: []string{"generate"}, Enabled: true, CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseConditional}},
		{ID: "derived-unproven", Backend: models.BackendBuiltin, Architecture: models.ArchSDXL, Operations: []string{"text_to_image"}, Enabled: true, CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes}, Hardware: models.Hardware{CPUCapable: true}},
		{ID: "disabled", Operations: []string{"caption"}, Enabled: false},
	}
	got := EnabledOperations(candidates, []string{"remove", "generate", "caption", "image_to_image", "inpaint", "outpaint", "unknown"}, nil)
	want := []string{"generate", "image_to_image", "inpaint", "outpaint", "remove"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("enabled operation scope = %v, want %v", got, want)
	}
	plan := HostFitPlan(candidates[2:3], []string{"outpaint"}, capabilities.Host{OS: "linux", Arch: "amd64", AvailableMemoryBytes: 16 << 30}, nil, nil)
	if plan.CoverageComplete || len(plan.UncoveredOperations) != 1 || len(plan.CandidateGaps) != 1 || plan.CandidateGaps[0].Reason != "derived_pipeline_unproven:diffusers-outpaint" {
		t.Fatalf("unproven derived capability was hidden from host-fit evidence: %+v", plan)
	}
}

func TestHostFitPlanRequiresRegisteredOperationBackendAndSmokeContract(t *testing.T) {
	const gib = uint64(1 << 30)
	modelsForPlan := []models.Model{
		{ID: "missing-backend", Operations: []string{"generate"}, Backend: "onnxruntime", Enabled: true,
			CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes},
			Source:           models.Source{Assets: []models.Asset{{Filename: "model.onnx", Kind: models.ArtifactONNX}}},
			Hardware:         models.Hardware{CPUCapable: true, MinRAMGB: 4}},
		{ID: "unsupported-smoke", Operations: []string{"edit"}, Backend: models.BackendDiffusers, Enabled: true,
			CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes}, Hardware: models.Hardware{CPUCapable: true, MinRAMGB: 4}},
	}
	plan := HostFitPlan(modelsForPlan, []string{"generate", "edit"}, capabilities.Host{OS: "linux", Arch: "amd64", AvailableMemoryBytes: 12 * gib}, nil,
		func(_ models.Model, _ string) string { return "no_registered_local_backend_for_operation" })
	if len(plan.SelectedModelIDs) != 0 || len(plan.UncoveredOperations) != 2 {
		t.Fatalf("candidate without serving route was selected: %+v", plan)
	}
	if len(plan.CandidateGaps) != 1 || plan.CandidateGaps[0].ModelID != "missing-backend" {
		t.Fatalf("backend gap was not retained: %+v", plan.CandidateGaps)
	}
	excluded := map[string]string{}
	for _, candidate := range plan.ExcludedModels {
		excluded[candidate.ModelID] = candidate.Reason
	}
	if excluded["unsupported-smoke"] != "unsupported_load_smoke_contract" {
		t.Fatalf("unsupported smoke contract was not rejected: %+v", plan.ExcludedModels)
	}
}

func TestProvisionKeepsIndependentModelsMovingAfterRuntimeFailure(t *testing.T) {
	s := fixture(t)
	s.RuntimeNeeded = func(m models.Model) bool { return m.ID == "naturalize-detail-v1" }
	s.EnsureRuntime = func(context.Context, func(int, string)) error { return errors.New("runtime unavailable") }
	result, err := s.Run(context.Background(), requestJob(t, "naturalize-detail-v1", "mask-fill"), nil)
	if err == nil {
		t.Fatal("runtime failure was hidden")
	}
	report := evidence(t, result)
	if report.Ready || len(report.Models) != 2 || report.RuntimeError == "" || report.Models[0].Error == "" || !report.Models[1].Lifecycle.Ready() {
		t.Fatalf("partial outcome: %+v", report)
	}
}

func TestProvisionCancellationStopsFurtherModelsAndRetainsPartialReport(t *testing.T) {
	s := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	s.EnsureBackend = func(ctx context.Context, _ models.Model) error { cancel(); return ctx.Err() }
	result, err := s.Run(ctx, requestJob(t, "naturalize-detail-v1", "mask-fill"), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error: %v", err)
	}
	report := evidence(t, result)
	if report.Ready || len(report.Models) != 1 || report.Models[0].Error == "" {
		t.Fatalf("cancellation report: %+v", report)
	}
}

func TestProvisionReportRetainsHostFitScopeAndIncompleteCoverage(t *testing.T) {
	s := fixture(t)
	plan := Plan{Mode: PlanHostFit, SelectedModelIDs: []string{"mask-fill"}, CoveredOperations: []string{"object_removal"},
		UncoveredOperations: []UncoveredOperation{{Operation: "text_to_image", Reason: "no_enabled_commercial_clean_local_candidate_fits_host"}},
		ExcludedModels:      []ExcludedModel{{ModelID: "qwen-image-edit", Operations: []string{"edit_instruct"}, Reason: "requires_32_gib_ram_host_has_18_gib_available"}}}
	plan.CoverageComplete = false
	result, err := s.Run(context.Background(), requestJobWithPlan(t, plan, "mask-fill"), nil)
	if err == nil || !strings.Contains(err.Error(), "enabled capability coverage is incomplete") {
		t.Fatalf("incomplete host-fit plan should fail readiness, got %v", err)
	}
	report := evidence(t, result)
	if report.Ready || report.CoverageComplete || report.Mode != PlanHostFit || len(report.Models) != 1 {
		t.Fatalf("incomplete capability coverage was reported ready: %+v", report)
	}
	if len(report.UncoveredOperations) != 1 || len(report.ExcludedModels) != 1 || report.ExcludedModels[0].ModelID != "qwen-image-edit" {
		t.Fatalf("plan omissions were not retained: %+v", report)
	}
}

func TestProvisionNoHostFitCandidatesIsRetainedAsNotReady(t *testing.T) {
	s := fixture(t)
	plan := Plan{Mode: PlanHostFit, UncoveredOperations: []UncoveredOperation{{Operation: "text_to_image", Reason: "no_enabled_commercial_clean_local_candidate_fits_host"}}}
	result, err := s.Run(context.Background(), requestJobWithPlan(t, plan), nil)
	if err == nil {
		t.Fatal("empty host-fit plan with uncovered capabilities should fail readiness")
	}
	report := evidence(t, result)
	if report.Ready || report.CoverageComplete || len(report.Models) != 0 || len(report.UncoveredOperations) != 1 {
		t.Fatalf("empty plan was reported ready: %+v", report)
	}
}

func requestJobWithPlan(t *testing.T, plan Plan, ids ...string) jobs.Job {
	t.Helper()
	raw, err := json.Marshal(Request{ModelIDs: ids, Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	return jobs.Job{ID: "provision-test", Payload: raw}
}
