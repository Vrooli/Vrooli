package validation

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"storage-manager/internal/module"
	"storage-manager/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	corestorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// ProtoFile exposes the shared validation proto FileDescriptor so the global
// parity test (api/internal/modules/registry_test.go) can walk it.
var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

// Module returns the validation domain's contribution to the API: the shared
// ScenarioValidationService handler mounted at its generated procedure path.
// The validator is constructed with the real detector/analyzer seams rooted at
// repoRoot.
func Module(logger *log.Logger, repoRoot string) module.Module {
	validator := validation.New(validation.Deps{RepoRoot: repoRoot, Logger: logger})
	// DescribeProvider answers readiness from this provider's own descriptor,
	// so a readiness probe no longer costs a full target analysis. A load
	// failure yields the zero Describer, which reports Unimplemented and makes
	// consumers fall back to the legacy probe.
	describer, _ := assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", "storage-manager"))
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "storage-manager"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	// Capture host facts once; they do not change during the process lifetime.
	// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib.
	environment, envErr := vroolicli.New().HostCaptureEnvironment(context.Background())
	if envErr != nil {
		if logger != nil {
			logger.Printf("validation: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		}
		environment = nil
	}
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(NewConnectHandler(Deps{
		Logger:       logger,
		Validator:    validator,
		MaturitySpec: spec,
		RepoRoot:     repoRoot,
		Environment:  environment,
	}), describer))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/validation/validate/{kind}/{id}", func(w http.ResponseWriter, req *http.Request) {
				kind, ok := parseOwnerKind(mux.Vars(req)["kind"])
				if !ok {
					http.Error(w, "kind must be scenario, resource, tool, or safeguard", http.StatusBadRequest)
					return
				}
				platform := corestorage.NormalizePlatform(req.URL.Query().Get("platform"))
				if platform == "" {
					platform = corestorage.HostPlatform()
				}
				report, validateErr := validator.ValidateOwner(req.Context(), kind, mux.Vars(req)["id"], platform)
				if validateErr != nil {
					http.Error(w, validateErr.Error(), http.StatusBadRequest)
					return
				}
				writeJSON(w, report)
			}).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/validation/validate/fleet", func(w http.ResponseWriter, req *http.Request) {
				platform := corestorage.NormalizePlatform(req.URL.Query().Get("platform"))
				if platform == "" {
					platform = corestorage.HostPlatform()
				}
				inventory, inventoryErr := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: repoRoot, Platform: platform})
				if inventoryErr != nil {
					http.Error(w, inventoryErr.Error(), http.StatusInternalServerError)
					return
				}
				allowed := make(map[corestorage.OwnerKind]bool)
				for _, rawKind := range req.URL.Query()["kind"] {
					if kind, ok := parseOwnerKind(rawKind); ok {
						allowed[kind] = true
					}
				}
				reports := make([]validation.Report, 0, len(inventory.Owners))
				counts := make(map[string]int)
				errorsFound, blockersFound := 0, 0
				for _, owner := range inventory.Owners {
					if len(allowed) > 0 && !allowed[owner.Kind] {
						continue
					}
					report, validateErr := validator.ValidateOwnerFromInventoryFast(req.Context(), owner.Kind, owner.ID, platform, inventory)
					if validateErr != nil {
						http.Error(w, validateErr.Error(), http.StatusInternalServerError)
						return
					}
					reports = append(reports, report)
					for _, finding := range report.Findings {
						counts[finding.Code]++
						if finding.Severity >= validation.SeverityError {
							errorsFound++
						}
					}
				}
				writeJSON(w, fleetReport{Platform: platform, Reports: reports, CountsByCode: counts, ErrorCount: errorsFound, BlockerCount: blockersFound, MigrationAllowlistSize: validator.MigrationAllowlistSize()})
			}).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

type fleetReport struct {
	Platform               corestorage.Platform `json:"platform"`
	Reports                []validation.Report  `json:"reports"`
	CountsByCode           map[string]int       `json:"counts_by_code"`
	ErrorCount             int                  `json:"error_count"`
	BlockerCount           int                  `json:"blocker_count"`
	MigrationAllowlistSize int                  `json:"migration_allowlist_size"`
}

func parseOwnerKind(value string) (corestorage.OwnerKind, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(corestorage.OwnerScenario):
		return corestorage.OwnerScenario, true
	case string(corestorage.OwnerResource):
		return corestorage.OwnerResource, true
	case string(corestorage.OwnerTool):
		return corestorage.OwnerTool, true
	case string(corestorage.OwnerSafeguard):
		return corestorage.OwnerSafeguard, true
	default:
		return "", false
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

// Schema returns "" — validation is stateless (no tables). The registry
// re-exports it anyway so the per-domain shape stays uniform.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the validation module's
// public surface. References the generated *Procedure constant so renaming the
// RPC in validation.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{ID: "validation_validate_target", Path: scenariovalidationconnect.ScenarioValidationServiceValidateTargetProcedure, Method: "POST", Summary: "Validate a first-class repository target", Category: "validation"},
	{
		ID:          "validation_validate_owner",
		Path:        "/api/v1/validation/validate/{kind}/{id}",
		Method:      "GET",
		Summary:     "Validate one storage owner",
		Description: "Runs the shared storage validation report for a scenario, resource, tool, or safeguard on the requested platform.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"kind": "scenario|resource|tool|safeguard", "id": "string", "platform": "linux|macos|windows"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"owner_kind": "string", "owner_id": "string", "platform": "string", "status": "string", "findings": "[]storage finding"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Unknown owner kind, invalid platform, or unresolved owner"}},
	},
	{
		ID:          "validation_validate_fleet",
		Path:        "/api/v1/validation/validate/fleet",
		Method:      "GET",
		Summary:     "Validate every storage owner",
		Description: "Runs the shared validation report for every selected scenario, resource, tool, and safeguard and returns per-code roll-up counts.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"kind": "repeatable owner-kind filter", "platform": "linux|macos|windows"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"platform": "string", "reports": "[]storage validation report", "counts_by_code": "map[string]int", "error_count": "int", "blocker_count": "int", "migration_allowlist_size": "int"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Owner inventory cannot be loaded or validated"}},
	},
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's storage judgment",
		Description: "Detects the target scenario's storage surface (engines + API language) and runs the applicable static storage analyzers, returning the shared scenario-validation response with findings in the maturity assessment.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or scenario not found under scenarios/"},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"storage-manager\"}'"},
		},
	},
	{
		ID:          "validation_describe_provider",
		Path:        scenariovalidationconnect.ScenarioValidationServiceDescribeProviderProcedure,
		Method:      "POST",
		Summary:     "Describe this provider's identity and contract",
		Description: "Reports provider identity, backed phase, maturity spec version, contract, build provenance, and capabilities. Inspects no target, so readiness consumers can confirm this provider is live and current without paying for a full validation run.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"provider":     "string",
			"phase":        "string",
			"spec_version": "string",
			"contract":     "string",
			"build":        "scenario_validation.v1.ProviderBuild",
			"capabilities": "scenario_validation.v1.ProviderCapabilities",
		}},
		Examples: []module.Example{{Name: "Describe provider", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/DescribeProvider -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview the deterministic storage autofixes for a scenario",
		Description: "Shared ScenarioValidationService Fix RPC. Returns the candidate edits the storage-manager autofix registry would apply (DB_ROWS_NOT_CLOSED, ENSURE_SCHEMAS_NOT_WIRED) without writing anything. Optional rule_ids restricts the preview to specific finding codes.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "[]string"}},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "[]scenario_validation.v1.FixCandidate", "messages": "[]string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id/path or scenario directory not resolvable"},
		},
		Examples: []module.Example{
			{Name: "Preview storage fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/PreviewFix -H 'Content-Type: application/json' -d '{\"scenario\":\"storage-manager\"}'"},
		},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply the deterministic storage autofixes for a scenario",
		Description: "Shared ScenarioValidationService Fix RPC. Applies the storage-manager autofix registry's deterministic edits (DB_ROWS_NOT_CLOSED, ENSURE_SCHEMAS_NOT_WIRED) and reports what changed. Idempotent: a second apply over an already-fixed tree is a no-op.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "[]string"}},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "[]scenario_validation.v1.FixCandidate", "messages": "[]string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id/path or scenario directory not resolvable"},
		},
		Examples: []module.Example{
			{Name: "Apply storage fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ApplyFix -H 'Content-Type: application/json' -d '{\"scenario\":\"storage-manager\"}'"},
		},
	},
}
