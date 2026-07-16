package orchestration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
	"agent-manager/internal/workflowcatalog"

	repocontract "github.com/vrooli/repo-contract-go"
)

type workflowSourcesConfig struct {
	Profiles  json.RawMessage `json:"profiles,omitempty"`
	Workflows struct {
		Reconcile *bool    `json:"reconcile"`
		Sources   []string `json:"sources"`
	} `json:"workflows"`
}

func (o *Orchestrator) ValidateWorkflow(ctx context.Context, data []byte) (*WorkflowValidationResult, error) {
	parsed, err := workflowcatalog.Parse(data, nil)
	if err != nil {
		return &WorkflowValidationResult{Diagnostics: []domain.WorkflowDiagnostic{{Code: "decode", Message: err.Error()}}}, nil
	}
	result := &WorkflowValidationResult{Valid: len(parsed.Diagnostics) == 0, Digest: parsed.Digest, Diagnostics: parsed.Diagnostics}
	if result.Valid {
		if diagnostics := o.validateWorkflowTargets(ctx, &parsed.Definition, nil); len(diagnostics) != 0 {
			result.Valid = false
			result.Diagnostics = diagnostics
			result.Digest = ""
		} else {
			result.Definition = &parsed.Definition
		}
	}
	return result, nil
}

func (o *Orchestrator) ReconcileScenarioWorkflows(ctx context.Context, req ReconcileScenarioWorkflowsRequest) (*ReconcileScenarioWorkflowsResult, error) {
	if o.workflows == nil {
		return nil, domain.NewConfigInvalidError("workflowRepository", "workflow catalog persistence is not configured", nil)
	}
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		return nil, domain.NewValidationErrorWithHint("scenario", "field is required", "Provide the owning scenario slug")
	}
	contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return nil, domain.NewConfigInvalidError("repoContract", "failed to resolve repository contract", err)
	}
	scenarioRoot, err := contract.ScenarioRoot(repoRoot, scenario)
	if err != nil {
		return nil, domain.NewValidationErrorWithHint("scenario", "invalid scenario slug", err.Error())
	}
	servicePath, err := contract.ScenarioFile(repoRoot, scenario, "service")
	if err != nil {
		return nil, domain.NewConfigInvalidError("serviceManifest", "failed to resolve scenario service manifest", err)
	}
	cfg, err := readScenarioWorkflowConfig(servicePath)
	if err != nil {
		return nil, err
	}
	result := &ReconcileScenarioWorkflowsResult{Scenario: scenario, DryRun: req.DryRun, ValidateOnly: req.ValidateOnly}
	if cfg.Workflows.Reconcile != nil && !*cfg.Workflows.Reconcile {
		for _, source := range cfg.Workflows.Sources {
			result.addWorkflow(WorkflowReconcileResult{SourcePath: source, Status: WorkflowReconcileSkipped, Message: "workflow reconciliation disabled"})
		}
		return result, nil
	}
	if len(cfg.Workflows.Sources) == 0 {
		result.addWorkflow(WorkflowReconcileResult{Status: WorkflowReconcileSkipped, Message: "no workflow sources declared"})
		return result, nil
	}

	type candidate struct {
		revision *domain.WorkflowRevision
		item     int
		active   *domain.WorkflowRevision
	}
	var candidates []candidate
	definitions := map[string]bool{}
	for _, source := range cfg.Workflows.Sources {
		item := WorkflowReconcileResult{SourcePath: source}
		path, pathErr := resolveProfileSourcePath(scenarioRoot, source)
		if pathErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = pathErr.Error()
			result.addWorkflow(item)
			continue
		}
		data, info, readErr := readWorkflowSource(path)
		if readErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = readErr.Error()
			result.addWorkflow(item)
			continue
		}
		parsed, parseErr := workflowcatalog.Parse(data, nil)
		if parseErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = parseErr.Error()
			result.addWorkflow(item)
			continue
		}
		item.WorkflowKey, item.Version, item.Digest, item.Diagnostics = parsed.Definition.Key, parsed.Definition.Version, parsed.Digest, parsed.Diagnostics
		if parsed.Definition.Owner != scenario {
			item.Diagnostics = append(item.Diagnostics, domain.WorkflowDiagnostic{Code: "ownership", Path: "owner", Message: "owner must match the declaring scenario"})
		}
		if len(item.Diagnostics) != 0 {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = "workflow definition is invalid"
			result.addWorkflow(item)
			continue
		}
		identity := parsed.Definition.Key + "@" + parsed.Definition.Version
		if definitions[identity] {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = "duplicate workflow key and version in source set"
			result.addWorkflow(item)
			continue
		}
		definitions[identity] = true
		active, getErr := o.workflows.GetActive(ctx, scenario, parsed.Definition.Key)
		if getErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = getErr.Error()
			result.addWorkflow(item)
			continue
		}
		sum := sha256.Sum256(data)
		revision := &domain.WorkflowRevision{Owner: scenario, Key: parsed.Definition.Key, SemanticVersion: parsed.Definition.Version, Digest: parsed.Digest, Definition: parsed.Definition, SourcePath: source, SourceHash: hex.EncodeToString(sum[:]), SourceUpdatedAt: info.ModTime().UTC(), CreatedAt: time.Now().UTC()}
		if active != nil && active.Digest == revision.Digest {
			item.Status = WorkflowReconcileUnchanged
			item.Message = "active revision already matches source"
			result.addWorkflow(item)
			continue
		}
		if active == nil {
			item.Status = WorkflowReconcileCreated
		} else {
			item.Status = WorkflowReconcileActivated
		}
		result.Results = append(result.Results, item)
		candidates = append(candidates, candidate{revision: revision, item: len(result.Results) - 1, active: active})
	}

	// Resolve references only after every source has parsed so sibling child
	// workflows can refer to each other without source-order dependence.
	known := map[string]bool{}
	for _, c := range candidates {
		known[c.revision.Key] = true
	}
	for _, c := range candidates {
		if diagnostics := o.validateWorkflowTargets(ctx, &c.revision.Definition, known); len(diagnostics) != 0 {
			result.Results[c.item].Status = WorkflowReconcileFailedValidation
			result.Results[c.item].Message = "workflow references are invalid"
			result.Results[c.item].Diagnostics = diagnostics
		}
	}
	result.recountWorkflows()
	if result.Failed != 0 {
		for _, c := range candidates {
			if result.Results[c.item].Status != WorkflowReconcileFailedValidation {
				result.Results[c.item].Status = WorkflowReconcileSkipped
				result.Results[c.item].Message = "atomic reload withheld because another source failed validation"
			}
		}
		result.recountWorkflows()
		return result, nil
	}
	if req.DryRun || req.ValidateOnly {
		for _, c := range candidates {
			result.Results[c.item].Message = "validated; catalog not modified"
		}
		result.recountWorkflows()
		return result, nil
	}
	revisions := make([]*domain.WorkflowRevision, 0, len(candidates))
	for _, c := range candidates {
		revisions = append(revisions, c.revision)
	}
	if err := o.workflows.ActivateBatch(ctx, revisions); err != nil {
		return nil, err
	}
	result.recountWorkflows()
	obs.Component("workflow-catalog").Info("scenario workflow catalog reconciled", "scenario", scenario, "created", result.Created, "activated", result.Activated, "unchanged", result.Unchanged, "digest_count", len(revisions))
	return result, nil
}

func (o *Orchestrator) validateWorkflowTargets(ctx context.Context, d *domain.WorkflowDefinition, siblingKeys map[string]bool) []domain.WorkflowDiagnostic {
	var diagnostics []domain.WorkflowDiagnostic
	for i, node := range d.Nodes {
		path := fmt.Sprintf("nodes[%d]", i)
		if node.Run != nil && node.Run.ProfileKey != "" {
			profile, err := o.profiles.GetByKey(ctx, node.Run.ProfileKey)
			if err != nil || profile == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "profile_missing", Path: path + ".run.profileKey", Message: "profile does not resolve"})
			}
		}
		if node.Run != nil && node.Run.RoleRef != "" {
			if o.rolePolicy == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "role_catalog_unavailable", Path: path + ".run.roleRef", Message: "portable role catalog is unavailable"})
			} else if active := o.rolePolicy.Active(); active == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "role_catalog_unavailable", Path: path + ".run.roleRef", Message: "portable role catalog is unavailable"})
			} else if _, ok := active.Catalog().Roles[node.Run.RoleRef]; !ok {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "role_missing", Path: path + ".run.roleRef", Message: "portable role does not resolve"})
			}
		}
		if node.Child != nil && !siblingKeys[node.Child.WorkflowKey] {
			if o.workflows == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "child_catalog_unavailable", Path: path + ".childWorkflow.workflowKey", Message: "workflow catalog is unavailable"})
				continue
			}
			active, err := o.workflows.GetActive(ctx, d.Owner, node.Child.WorkflowKey)
			if err != nil || active == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "child_missing", Path: path + ".childWorkflow.workflowKey", Message: "child workflow does not resolve"})
			}
		}
	}
	return diagnostics
}

func (o *Orchestrator) ListWorkflowRevisions(ctx context.Context, owner, key string, opts ListOptions) ([]*domain.WorkflowRevision, error) {
	if o.workflows == nil {
		return nil, domain.NewConfigInvalidError("workflowRepository", "workflow catalog persistence is not configured", nil)
	}
	return o.workflows.List(ctx, strings.TrimSpace(owner), strings.TrimSpace(key), repository.ListFilter{Limit: opts.Limit, Offset: opts.Offset})
}

func (o *Orchestrator) GetWorkflowRevision(ctx context.Context, owner, key, digest string) (*domain.WorkflowRevision, error) {
	if o.workflows == nil {
		return nil, domain.NewConfigInvalidError("workflowRepository", "workflow catalog persistence is not configured", nil)
	}
	if strings.TrimSpace(digest) != "" {
		return o.workflows.GetByDigest(ctx, strings.TrimSpace(digest))
	}
	return o.workflows.GetActive(ctx, strings.TrimSpace(owner), strings.TrimSpace(key))
}

func readScenarioWorkflowConfig(servicePath string) (*workflowSourcesConfig, error) {
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return nil, domain.NewConfigInvalidError("serviceManifest", "failed to read scenario service manifest", err)
	}
	var manifest scenarioServiceProfileConfig
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, domain.NewConfigInvalidError("serviceManifest", "failed to parse scenario service manifest", err)
	}
	dep, ok := manifest.Dependencies.Scenarios["agent-manager"]
	if !ok {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager", "dependency is required", "Declare agent-manager before workflows")
	}
	if dep.Enabled != nil && !*dep.Enabled {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager.enabled", "dependency must be enabled", "Enable agent-manager before workflows")
	}
	if len(dep.Config) == 0 {
		return &workflowSourcesConfig{}, nil
	}
	var configObject map[string]json.RawMessage
	if err := json.Unmarshal(dep.Config, &configObject); err != nil {
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "must be a JSON object", err)
	}
	if _, exists := configObject["workflows"]; !exists {
		return &workflowSourcesConfig{}, nil
	}
	var cfg workflowSourcesConfig
	decoder := json.NewDecoder(bytes.NewReader(dep.Config))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "failed to parse workflow config", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "multiple JSON values", err)
	}
	if cfg.Workflows.Reconcile == nil {
		return nil, domain.NewValidationErrorWithHint("config.workflows.reconcile", "field is required", "Set whether workflow sources reconcile")
	}
	if len(cfg.Workflows.Sources) == 0 {
		return nil, domain.NewValidationErrorWithHint("config.workflows.sources", "must declare at least one source", "Omit workflows when unused")
	}
	seen := map[string]bool{}
	for _, source := range cfg.Workflows.Sources {
		source = strings.TrimSpace(source)
		if source == "" || seen[source] {
			return nil, domain.NewValidationErrorWithHint("config.workflows.sources", "sources must be non-empty and unique", "Declare each target-relative source once")
		}
		seen[source] = true
	}
	return &cfg, nil
}

// ValidateScenarioWorkflowConfig is the read-only conformance seam.
func ValidateScenarioWorkflowConfig(servicePath string) error {
	_, err := readScenarioWorkflowConfig(servicePath)
	return err
}

func readWorkflowSource(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("stat workflow source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("workflow source must be a regular file")
	}
	if info.Size() <= 0 || info.Size() > workflowcatalog.MaxDefinitionBytes {
		return nil, nil, fmt.Errorf("workflow source exceeds bounded size")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read workflow source: %w", err)
	}
	return data, info, nil
}

func (r *ReconcileScenarioWorkflowsResult) addWorkflow(item WorkflowReconcileResult) {
	r.Results = append(r.Results, item)
	r.recountWorkflows()
}

func (r *ReconcileScenarioWorkflowsResult) recountWorkflows() {
	r.Created, r.Activated, r.Unchanged, r.Skipped, r.Failed = 0, 0, 0, 0, 0
	for _, item := range r.Results {
		switch item.Status {
		case WorkflowReconcileCreated:
			r.Created++
		case WorkflowReconcileActivated:
			r.Activated++
		case WorkflowReconcileUnchanged:
			r.Unchanged++
		case WorkflowReconcileSkipped:
			r.Skipped++
		case WorkflowReconcileFailedValidation:
			r.Failed++
		}
	}
}
