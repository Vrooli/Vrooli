// This file reconciles declared agent state with persisted orchestration state.
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
	"path/filepath"
	"sort"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/workflowcatalog"

	repocontract "github.com/vrooli/repo-contract-go"
)

// The unified scenario-owned declaration layer. Profiles and workflows are
// declared through one service.json block
// (dependencies.scenarios.agent-manager.config.declarations) and live in one
// directory (.vrooli/agent-manager/). Each source file is discriminated by its
// schemaVersion; reconcile fans out per file to the profile (mutable, drift
// tracked) or workflow (digest-pinned) path while preserving each kind's
// lifecycle semantics. The old locations and config blocks are rejected with
// actionable diagnostics — there is no dual-read fallback.
const (
	declarationSourceDir    = ".vrooli/agent-manager/"
	legacyProfileSourceDir  = ".vrooli/agent-profiles/"
	legacyWorkflowSourceDir = ".vrooli/agent-workflows/"

	// agentManagerSelfScenario is the provider scenario that registers its own
	// declarations. It cannot declare a dependency on itself, so its sources are
	// discovered directly from .vrooli/agent-manager/ rather than a service.json
	// declarations block; the shared validators and ownership rules still apply.
	agentManagerSelfScenario = "agent-manager"
)

type declarationsConfig struct {
	Declarations struct {
		Reconcile   *bool    `json:"reconcile"`
		ProfileMode string   `json:"profileMode"`
		Sources     []string `json:"sources"`
	} `json:"declarations"`
}

// readScenarioDeclarationConfig reads and strictly validates the unified
// declaration block. It rejects the legacy config.profiles/config.workflows
// blocks and old-directory source paths with actionable hints so the
// no-fallback cutover is enforced at the config contract.
func readScenarioDeclarationConfig(servicePath string) (*declarationsConfig, error) {
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
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager", "dependency is required",
			"Declare agent-manager under dependencies.scenarios with config.declarations.sources")
	}
	if dep.Enabled != nil && !*dep.Enabled {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager.enabled", "dependency must be enabled",
			"Enable agent-manager before reconciling its scenario-owned declarations")
	}
	if len(dep.Config) == 0 {
		return &declarationsConfig{}, nil
	}
	var configObject map[string]json.RawMessage
	if err := json.Unmarshal(dep.Config, &configObject); err != nil || configObject == nil {
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "must be a JSON object containing declarations", err)
	}
	if _, legacy := configObject["profiles"]; legacy {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager.config.profiles", "legacy declaration block is no longer supported",
			"Move sources into config.declarations.sources and place files under .vrooli/agent-manager/")
	}
	if _, legacy := configObject["workflows"]; legacy {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager.config.workflows", "legacy declaration block is no longer supported",
			"Move sources into config.declarations.sources and place files under .vrooli/agent-manager/")
	}
	declRaw, hasDeclarations := configObject["declarations"]
	if !hasDeclarations || string(declRaw) == "null" {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager.config.declarations", "field is required",
			"Declare declarations.reconcile, declarations.profileMode, and declarations.sources or omit config when no scenario-owned declaration is needed")
	}
	var cfg declarationsConfig
	decoder := json.NewDecoder(bytes.NewReader(dep.Config))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "failed to parse declaration config", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "failed to parse declaration config", err)
	}
	if cfg.Declarations.Reconcile == nil {
		return nil, domain.NewValidationErrorWithHint("config.declarations.reconcile", "field is required",
			"Set whether the declared scenario-owned declarations reconcile")
	}
	if mode := strings.TrimSpace(cfg.Declarations.ProfileMode); mode != "" && !validProfileReconcileMode(mode) {
		return nil, domain.NewValidationErrorWithHint("config.declarations.profileMode", "invalid profile reconcile mode",
			"valid values: create_only, update_if_unmodified, force")
	}
	if len(cfg.Declarations.Sources) == 0 {
		return nil, domain.NewValidationErrorWithHint("config.declarations.sources", "must declare at least one source",
			"Omit config entirely when the scenario uses only direct portable role requests")
	}
	seen := make(map[string]struct{}, len(cfg.Declarations.Sources))
	for _, source := range cfg.Declarations.Sources {
		key := strings.TrimSpace(source)
		if key == "" {
			return nil, domain.NewValidationErrorWithHint("config.declarations.sources", "declaration source must not be empty", "Declare a target-relative declaration JSON file")
		}
		if _, exists := seen[key]; exists {
			return nil, domain.NewValidationErrorWithHint("config.declarations.sources", "duplicate declaration source", "Declare each scenario-owned source once")
		}
		seen[key] = struct{}{}
		if err := validateDeclarationSourceLocation(key); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

// validateDeclarationSourceLocation rejects sources that still point at the
// retired profile/workflow directories and requires every source to live under
// .vrooli/agent-manager/.
func validateDeclarationSourceLocation(source string) error {
	// filepath.Clean strips a trailing slash and normalizes separators; compare
	// against the directory prefixes with a trailing slash re-applied so a bare
	// directory name never matches a longer sibling.
	normalized := filepath.ToSlash(filepath.Clean(source)) + "/"
	if strings.HasPrefix(normalized, legacyProfileSourceDir) || strings.HasPrefix(normalized, legacyWorkflowSourceDir) {
		return domain.NewValidationErrorWithHint("config.declarations.sources", "declaration sources must live under .vrooli/agent-manager/",
			"Move "+source+" into .vrooli/agent-manager/ and update the source")
	}
	if !strings.HasPrefix(normalized, declarationSourceDir) {
		return domain.NewValidationErrorWithHint("config.declarations.sources", "declaration sources must live under .vrooli/agent-manager/",
			"Move "+source+" into .vrooli/agent-manager/ and update the source")
	}
	return nil
}

// ValidateScenarioDeclarationConfig validates the unified declaration block
// without touching the profile repository, workflow catalog, or target files.
// It is shared by read-only conformance and mutating reconciliation so the two
// surfaces cannot accept different manifests. It returns nil when the scenario
// does not declare the unified block (no dependency, no config, or no
// declarations key) so conformance stays silent for scenarios that use only
// direct portable role requests.
func ValidateScenarioDeclarationConfig(servicePath string) error {
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return err
	}
	var manifest scenarioServiceProfileConfig
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	dep, ok := manifest.Dependencies.Scenarios["agent-manager"]
	if !ok || len(dep.Config) == 0 {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(dep.Config, &object); err != nil {
		return err
	}
	if _, hasDeclarations := object["declarations"]; !hasDeclarations {
		return nil
	}
	_, err = readScenarioDeclarationConfig(servicePath)
	return err
}

// ParseScenarioProfileSource parses a declared profile source file into the
// runtime profile entity using the same peek-and-strip schemaVersion handling
// and strict proto unmarshal that reconcile uses. Read-only conformance calls
// it so it validates a profile exactly as reconcile would accept it, with no
// repository or filesystem access. It does not enforce ownership; the caller
// applies scenario-scoped policy.
func ParseScenarioProfileSource(data []byte) (*domain.AgentProfile, error) {
	return parseSourceProfile(data)
}

// peekSchemaVersion decodes only the schemaVersion discriminator so the reader
// can route a declaration source to the profile or workflow path before any
// kind-specific strict parse.
func peekSchemaVersion(data []byte) (string, error) {
	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("parse declaration schemaVersion: %w", err)
	}
	return strings.TrimSpace(envelope.SchemaVersion), nil
}

// ReconcileScenarioDeclarations resolves the scenario's unified declaration
// block and reconciles every source into agent-manager's runtime state,
// fanning out per source by schemaVersion.
func (o *Orchestrator) ReconcileScenarioDeclarations(ctx context.Context, req ReconcileScenarioDeclarationsRequest) (*ReconcileScenarioDeclarationsResult, error) {
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
	// Agent Manager registers its own declarations through the self-registration
	// seam: it cannot declare a dependency on itself, so its sources come from the
	// .vrooli/agent-manager/ directory rather than a service.json block.
	if scenario == agentManagerSelfScenario {
		return o.reconcileSelfDeclarationsAt(ctx, scenarioRoot, req.DryRun, req.ValidateOnly)
	}
	servicePath, err := contract.ScenarioFile(repoRoot, scenario, "service")
	if err != nil {
		return nil, domain.NewConfigInvalidError("serviceManifest", "failed to resolve scenario service manifest", err)
	}
	return o.reconcileScenarioDeclarationsAt(ctx, scenario, scenarioRoot, servicePath, req.DryRun, req.ValidateOnly)
}

// ReconcileSelfDeclarations registers agent-manager's own declaration files at
// startup. It bypasses only the dependency-declaration gate — every source runs
// through the same shared validators, ownership checks, and activation as any
// other scenario. It tolerates zero declaration files so the seam is inert until
// agent-manager adds its own definitions.
func (o *Orchestrator) ReconcileSelfDeclarations(ctx context.Context, repoRoot string) (*ReconcileScenarioDeclarationsResult, error) {
	scenarioRoot := filepath.Join(repoRoot, "scenarios", agentManagerSelfScenario)
	return o.reconcileSelfDeclarationsAt(ctx, scenarioRoot, false, false)
}

// reconcileSelfDeclarationsAt discovers agent-manager's declaration sources from
// its .vrooli/agent-manager/ directory and reconciles them with owner
// agent-manager. Sources are the JSON files in that directory; a missing
// directory yields an empty (successful) result.
func (o *Orchestrator) reconcileSelfDeclarationsAt(ctx context.Context, scenarioRoot string, dryRun, validateOnly bool) (*ReconcileScenarioDeclarationsResult, error) {
	sources, err := listSelfDeclarationSources(scenarioRoot)
	if err != nil {
		return nil, err
	}
	return o.reconcileDeclarationSources(ctx, agentManagerSelfScenario, scenarioRoot, sources, profileReconcileModeUpdateIfUnmodified, true, dryRun, validateOnly)
}

// listSelfDeclarationSources returns the target-relative paths of every JSON
// declaration file under .vrooli/agent-manager/, sorted for deterministic
// ordering. A missing directory is not an error.
func listSelfDeclarationSources(scenarioRoot string) ([]string, error) {
	dir := filepath.Join(scenarioRoot, filepath.FromSlash(declarationSourceDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, domain.NewConfigInvalidError("selfDeclarations", "failed to read agent-manager declaration directory", err)
	}
	var sources []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		sources = append(sources, declarationSourceDir+entry.Name())
	}
	sort.Strings(sources)
	return sources, nil
}

// reconcileScenarioDeclarationsAt is the path-explicit core so fixtures can
// exercise the full fan-out without a repository-contract resolution.
func (o *Orchestrator) reconcileScenarioDeclarationsAt(ctx context.Context, scenario, scenarioRoot, servicePath string, dryRun, validateOnly bool) (*ReconcileScenarioDeclarationsResult, error) {
	cfg, err := readScenarioDeclarationConfig(servicePath)
	if err != nil {
		return nil, err
	}
	reconcile := true
	if cfg.Declarations.Reconcile != nil {
		reconcile = *cfg.Declarations.Reconcile
	}
	profileMode := strings.TrimSpace(cfg.Declarations.ProfileMode)
	if profileMode == "" {
		profileMode = profileReconcileModeUpdateIfUnmodified
	}
	return o.reconcileDeclarationSources(ctx, scenario, scenarioRoot, cfg.Declarations.Sources, profileMode, reconcile, dryRun, validateOnly)
}

// reconcileDeclarationSources classifies a source list by schemaVersion and
// reconciles each kind with its own lifecycle semantics. It is the shared core
// behind both the config-driven scenario path and the directory-driven
// self-registration seam, so the two surfaces run identical validators and
// activation.
func (o *Orchestrator) reconcileDeclarationSources(ctx context.Context, scenario, scenarioRoot string, sources []string, profileMode string, reconcile, dryRun, validateOnly bool) (*ReconcileScenarioDeclarationsResult, error) {
	result := &ReconcileScenarioDeclarationsResult{Scenario: scenario, DryRun: dryRun, ValidateOnly: validateOnly}

	var profileSources, workflowSources []string
	for _, raw := range sources {
		source := strings.TrimSpace(raw)
		kind, failed, ok := o.classifyDeclarationSource(scenarioRoot, source)
		if !ok {
			result.addProfile(failed)
			continue
		}
		switch kind {
		case domain.ProfileSchemaVersionV1:
			profileSources = append(profileSources, source)
		case domain.WorkflowSchemaVersionV1:
			workflowSources = append(workflowSources, source)
		}
	}

	if !reconcile {
		for _, source := range profileSources {
			result.addProfile(ProfileReconcileResult{SourcePath: source, Status: ProfileReconcileStatusSkipped, Message: "declaration reconciliation disabled by manifest config"})
		}
		for _, source := range workflowSources {
			result.addWorkflow(WorkflowReconcileResult{SourcePath: source, Status: WorkflowReconcileSkipped, Message: "declaration reconciliation disabled by manifest config"})
		}
		return result, nil
	}

	for _, source := range profileSources {
		result.addProfile(o.reconcileProfileSource(ctx, scenario, scenarioRoot, source, profileMode, dryRun))
	}

	workflowResults, err := o.reconcileDeclaredWorkflows(ctx, scenario, scenarioRoot, workflowSources, dryRun, validateOnly)
	if err != nil {
		return nil, err
	}
	for _, item := range workflowResults {
		result.addWorkflow(item)
	}
	return result, nil
}

// classifyDeclarationSource peeks a source's schemaVersion to route it. A source
// that cannot be resolved, read, or that carries an unknown discriminator yields
// a failed-validation diagnostic instead of a routing decision.
func (o *Orchestrator) classifyDeclarationSource(scenarioRoot, source string) (string, ProfileReconcileResult, bool) {
	item := ProfileReconcileResult{SourcePath: source}
	path, err := resolveProfileSourcePath(scenarioRoot, source)
	if err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return "", item, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = fmt.Errorf("read declaration source: %w", err).Error()
		return "", item, false
	}
	version, err := peekSchemaVersion(data)
	if err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return "", item, false
	}
	switch version {
	case domain.ProfileSchemaVersionV1, domain.WorkflowSchemaVersionV1:
		return version, item, true
	default:
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = fmt.Sprintf("declaration source has unknown schemaVersion %q; expected %q or %q", version, domain.ProfileSchemaVersionV1, domain.WorkflowSchemaVersionV1)
		return "", item, false
	}
}

// reconcileDeclaredWorkflows runs the digest-pinned, atomic workflow activation
// over the workflow-kind sources. All-or-nothing: if any source fails
// validation every other source is withheld, preserving the catalog's atomic
// reload semantics.
func (o *Orchestrator) reconcileDeclaredWorkflows(ctx context.Context, scenario, scenarioRoot string, sources []string, dryRun, validateOnly bool) ([]WorkflowReconcileResult, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	if o.workflows == nil {
		return nil, domain.NewConfigInvalidError("workflowRepository", "workflow catalog persistence is not configured", nil)
	}
	var results []WorkflowReconcileResult
	type candidate struct {
		revision *domain.WorkflowRevision
		item     int
		active   *domain.WorkflowRevision
	}
	var candidates []candidate
	definitions := map[string]bool{}
	for _, source := range sources {
		item := WorkflowReconcileResult{SourcePath: source}
		path, pathErr := resolveProfileSourcePath(scenarioRoot, source)
		if pathErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = pathErr.Error()
			results = append(results, item)
			continue
		}
		data, info, readErr := readWorkflowSource(path)
		if readErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = readErr.Error()
			results = append(results, item)
			continue
		}
		// promptRef nodes resolve against prompt-manager here, before the digest,
		// so the pinned revision carries the resolved content. A resolution
		// failure (missing skill, prompt-manager unreachable) fails this source,
		// which withholds the whole atomic batch — never a partial revision.
		resolved, resolveErr := o.resolveWorkflowPromptRefs(ctx, data)
		if resolveErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = resolveErr.Error()
			results = append(results, item)
			continue
		}
		parsed, parseErr := workflowcatalog.Parse(resolved, nil)
		if parseErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = parseErr.Error()
			results = append(results, item)
			continue
		}
		item.WorkflowKey, item.Version, item.Digest, item.Diagnostics = parsed.Definition.Key, parsed.Definition.Version, parsed.Digest, parsed.Diagnostics
		if parsed.Definition.Owner != scenario {
			item.Diagnostics = append(item.Diagnostics, domain.WorkflowDiagnostic{Code: "ownership", Path: "owner", Message: "owner must match the declaring scenario", Severity: domain.DiagnosticSeverityError})
		}
		if domain.HasBlockingDiagnostic(item.Diagnostics) {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = "workflow definition is invalid"
			results = append(results, item)
			continue
		}
		identity := parsed.Definition.Key + "@" + parsed.Definition.Version
		if definitions[identity] {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = "duplicate workflow key and version in source set"
			results = append(results, item)
			continue
		}
		definitions[identity] = true
		active, getErr := o.workflows.GetActive(ctx, scenario, parsed.Definition.Key)
		if getErr != nil {
			item.Status = WorkflowReconcileFailedValidation
			item.Message = getErr.Error()
			results = append(results, item)
			continue
		}
		sum := sha256.Sum256(data)
		revision := &domain.WorkflowRevision{Owner: scenario, Key: parsed.Definition.Key, SemanticVersion: parsed.Definition.Version, Digest: parsed.Digest, Definition: parsed.Definition, SourcePath: source, SourceHash: hex.EncodeToString(sum[:]), SourceUpdatedAt: info.ModTime().UTC(), CreatedAt: o.now().UTC()}
		if active != nil && active.Digest == revision.Digest {
			item.Status = WorkflowReconcileUnchanged
			item.Message = "active revision already matches source"
			results = append(results, item)
			continue
		}
		if active == nil {
			item.Status = WorkflowReconcileCreated
		} else {
			item.Status = WorkflowReconcileActivated
		}
		results = append(results, item)
		candidates = append(candidates, candidate{revision: revision, item: len(results) - 1, active: active})
	}

	// Resolve references only after every source has parsed so sibling child
	// workflows can refer to each other without source-order dependence.
	known := map[string]bool{}
	for _, c := range candidates {
		known[c.revision.Key] = true
	}
	for _, c := range candidates {
		if diagnostics := o.validateWorkflowTargets(ctx, &c.revision.Definition, known); len(diagnostics) != 0 {
			results[c.item].Status = WorkflowReconcileFailedValidation
			results[c.item].Message = "workflow references are invalid"
			results[c.item].Diagnostics = diagnostics
		}
	}
	if countWorkflowFailures(results) != 0 {
		for _, c := range candidates {
			if results[c.item].Status != WorkflowReconcileFailedValidation {
				results[c.item].Status = WorkflowReconcileSkipped
				results[c.item].Message = "atomic reload withheld because another source failed validation"
			}
		}
		return results, nil
	}
	if dryRun || validateOnly {
		for _, c := range candidates {
			results[c.item].Message = "validated; catalog not modified"
		}
		return results, nil
	}
	revisions := make([]*domain.WorkflowRevision, 0, len(candidates))
	for _, c := range candidates {
		revisions = append(revisions, c.revision)
	}
	if err := o.workflows.ActivateBatch(ctx, revisions); err != nil {
		return nil, err
	}
	created, activated, unchanged := 0, 0, 0
	for _, item := range results {
		switch item.Status {
		case WorkflowReconcileCreated:
			created++
		case WorkflowReconcileActivated:
			activated++
		case WorkflowReconcileUnchanged:
			unchanged++
		}
	}
	obs.Component("workflow-catalog").Info("scenario workflow catalog reconciled", "scenario", scenario, "created", created, "activated", activated, "unchanged", unchanged, "digest_count", len(revisions))
	return results, nil
}

func countWorkflowFailures(results []WorkflowReconcileResult) int {
	failed := 0
	for _, item := range results {
		if item.Status == WorkflowReconcileFailedValidation {
			failed++
		}
	}
	return failed
}

func (r *ReconcileScenarioDeclarationsResult) addProfile(item ProfileReconcileResult) {
	r.ProfileResults = append(r.ProfileResults, item)
	switch item.Status {
	case ProfileReconcileStatusCreated:
		r.ProfilesCreated++
	case ProfileReconcileStatusUpdated:
		r.ProfilesUpdated++
	case ProfileReconcileStatusUnchanged:
		r.ProfilesUnchanged++
	case ProfileReconcileStatusSkipped:
		r.ProfilesSkipped++
	case ProfileReconcileStatusConflictedLocalOverride:
		r.ProfilesConflicted++
	case ProfileReconcileStatusFailedValidation:
		r.ProfilesFailed++
	}
}

func (r *ReconcileScenarioDeclarationsResult) addWorkflow(item WorkflowReconcileResult) {
	r.WorkflowResults = append(r.WorkflowResults, item)
	switch item.Status {
	case WorkflowReconcileCreated:
		r.WorkflowsCreated++
	case WorkflowReconcileActivated:
		r.WorkflowsActivated++
	case WorkflowReconcileUnchanged:
		r.WorkflowsUnchanged++
	case WorkflowReconcileSkipped:
		r.WorkflowsSkipped++
	case WorkflowReconcileFailedValidation:
		r.WorkflowsFailed++
	}
}

// ReconcileDeclaringScenarios sweeps every scenario under repoRoot declaring the
// unified block and reconciles it, isolating per-scenario failures so a single
// broken manifest never blocks agent-manager readiness. It resolves scenario
// roots from the swept repoRoot so the glob and the reconcile agree on the same
// tree. Per-source failures are already isolated inside reconcile; this wrapper
// adds per-scenario isolation.
func (o *Orchestrator) ReconcileDeclaringScenarios(ctx context.Context, repoRoot string) SweepSummary {
	summary := SweepSummary{}
	matches, _ := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "service.json"))
	for _, servicePath := range matches {
		summary.Scanned++
		if !scenarioDeclaresBlock(servicePath) {
			continue
		}
		summary.Declaring++
		scenarioRoot := filepath.Dir(filepath.Dir(servicePath))
		scenario := filepath.Base(scenarioRoot)
		res := o.reconcileScenarioGuarded(ctx, scenario, scenarioRoot, servicePath)
		summary.Scenarios = append(summary.Scenarios, res)
		if res.Err != "" || res.Failed != 0 {
			summary.Failed++
			obs.Logger().Warn("scenario declaration reconciliation failed", "scenario", scenario, "failedItems", res.Failed, obs.KeyError, res.Err)
		} else {
			summary.Reconciled++
		}
	}
	return summary
}

func (o *Orchestrator) reconcileScenarioGuarded(ctx context.Context, scenario, scenarioRoot, servicePath string) (out ScenarioSweepResult) {
	out.Scenario = scenario
	defer func() {
		if r := recover(); r != nil {
			out.Err = fmt.Sprintf("panic reconciling declarations: %v", r)
		}
	}()
	res, err := o.reconcileScenarioDeclarationsAt(ctx, scenario, scenarioRoot, servicePath, false, false)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	out.ProfilesCreated = res.ProfilesCreated
	out.ProfilesUpdated = res.ProfilesUpdated
	out.WorkflowsCreated = res.WorkflowsCreated
	out.WorkflowsActivated = res.WorkflowsActivated
	out.Failed = res.ProfilesFailed + res.WorkflowsFailed
	return out
}

func scenarioDeclaresBlock(servicePath string) bool {
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return false
	}
	var manifest scenarioServiceProfileConfig
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	dep, ok := manifest.Dependencies.Scenarios["agent-manager"]
	if !ok {
		return false
	}
	if dep.Enabled != nil && !*dep.Enabled {
		return false
	}
	if len(dep.Config) == 0 {
		return false
	}
	var configObject map[string]json.RawMessage
	if json.Unmarshal(dep.Config, &configObject) != nil {
		return false
	}
	raw, has := configObject["declarations"]
	return has && string(raw) != "null"
}
