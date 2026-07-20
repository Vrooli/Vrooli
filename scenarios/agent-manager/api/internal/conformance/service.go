// Package conformance validates the portable coding-agent contract without
// reconciling profiles, modifying native policies, or starting target services.
package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/workflowcatalog"
)

var scenarioNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// agentManagerSelfScenario is the provider scenario that registers its own
// declarations through the self-registration seam rather than a service.json
// dependency block.
const agentManagerSelfScenario = "agent-manager"

const (
	CodeDependencyMissing       = "agent_conformance.dependency_missing"
	CodeDependencyDisabled      = "agent_conformance.dependency_disabled"
	CodeProfileInvalid          = "agent_conformance.profile_invalid"
	CodeProfileOrphan           = "agent_conformance.profile_orphan"
	CodeProfileOwnership        = "agent_conformance.profile_ownership_mismatch"
	CodeProfileLegacy           = "agent_conformance.profile_legacy_field"
	CodeRoleUnresolved          = "agent_conformance.role_unresolved"
	CodeDirectSpawnBypass       = "agent_conformance.direct_spawn_bypass"
	CodePermissionPosture       = "agent_conformance.permission_posture"
	CodeWorkflowInvalid         = "agent_conformance.workflow_invalid"
	CodeWorkflowOrphan          = "agent_conformance.workflow_orphan"
	CodeWorkflowOwnership       = "agent_conformance.workflow_ownership_mismatch"
	CodeWorkflowInlinePrompt    = "agent_conformance.workflow_inline_prompt"
	CodeDeclarationLegacyLayout = "agent_conformance.declaration_legacy_layout"
	CodeDeclarationLegacyBlock  = "agent_conformance.declaration_legacy_block"
)

type Finding struct {
	Code, Title, Message, Location, Remediation string
	Severity                                    string
}

type Report struct {
	Scenario string
	Root     string
	Findings []Finding
}

// PermissionPostureReader exposes only the read-only readiness assertion that
// conformance needs. Native permission configuration remains resource-owned.
type PermissionPostureReader interface {
	ReadinessError(context.Context) error
}

type Service struct {
	RepoRoot          string
	PermissionPosture PermissionPostureReader
}

func (s Service) Validate(scenario, explicitPath string) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	explicitPath = strings.TrimSpace(explicitPath)
	if scenario == "" && explicitPath == "" {
		return Report{}, fmt.Errorf("scenario or path is required")
	}
	if scenario != "" && !scenarioNamePattern.MatchString(scenario) {
		return Report{}, fmt.Errorf("scenario must be a canonical scenario slug")
	}
	root := explicitPath
	if root == "" {
		root = filepath.Join(s.RepoRoot, "scenarios", scenario)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("resolve scenario path: %w", err)
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return Report{}, fmt.Errorf("target scenario path is not a readable directory: %s", abs)
	}
	rootBase, err := filepath.Abs(filepath.Join(s.RepoRoot, "scenarios"))
	if err != nil {
		return Report{}, fmt.Errorf("resolve scenarios root: %w", err)
	}
	rel, err := filepath.Rel(rootBase, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.Contains(rel, string(filepath.Separator)) {
		return Report{}, fmt.Errorf("target path must identify one scenario beneath the repository scenarios root")
	}
	if scenario == "" {
		scenario = filepath.Base(abs)
	}
	report := Report{Scenario: scenario, Root: abs}
	roles, err := loadRoles(s.RepoRoot)
	if err != nil {
		return Report{}, fmt.Errorf("load Agent Manager role catalog: %w", err)
	}
	// Agent Manager is a first-class declaring scenario for its own definitions,
	// but it cannot declare a dependency on itself. Its declaration sources are
	// every file under .vrooli/agent-manager/; validate them with the shared
	// validators and owner agent-manager, with no dependency requirement.
	if scenario == agentManagerSelfScenario {
		return s.validateSelfDeclarations(report, abs, scenario, roles)
	}
	manifestPath := filepath.Join(abs, ".vrooli", "service.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("read service manifest: %w", err)
	}
	var manifest struct {
		Dependencies struct {
			Scenarios map[string]struct {
				Enabled *bool           `json:"enabled"`
				Config  json.RawMessage `json:"config"`
			} `json:"scenarios"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Report{}, fmt.Errorf("parse service manifest: %w", err)
	}
	dep, present := manifest.Dependencies.Scenarios["agent-manager"]
	// Files left under the retired .vrooli/agent-profiles/ or
	// .vrooli/agent-workflows/ directories are rejected regardless of the
	// dependency or config state: the unified layout is the only readable
	// location.
	reportLegacyLayoutFiles(&report, abs)
	if !present {
		report.add(CodeDependencyMissing, "Agent Manager dependency missing", manifestPath, "Declare dependencies.scenarios.agent-manager with a config.declarations block.")
		reportOrphanDeclarations(&report, abs, nil)
		reportDirectSpawnBypasses(&report, abs)
		sortFindings(report.Findings)
		return report, nil
	}
	if dep.Enabled != nil && !*dep.Enabled {
		report.add(CodeDependencyDisabled, "Agent Manager dependency disabled", manifestPath, "Enable the Agent Manager dependency before declaring scenario-owned agent assets.")
		reportOrphanDeclarations(&report, abs, nil)
		reportDirectSpawnBypasses(&report, abs)
		sortFindings(report.Findings)
		return report, nil
	}
	declared := map[string]bool{}
	// A consumer may request a portable role directly at runtime and therefore
	// legitimately declare no scenario-owned assets. Once it supplies dependency
	// configuration, however, that configuration is the strict unified
	// declaration contract it must satisfy.
	if len(dep.Config) > 0 {
		var configObject map[string]json.RawMessage
		if err := json.Unmarshal(dep.Config, &configObject); err != nil || configObject == nil {
			report.add(CodeProfileInvalid, "Agent declaration configuration invalid", manifestPath, "config must be a JSON object containing config.declarations.")
		} else {
			_, hasLegacyProfiles := configObject["profiles"]
			_, hasLegacyWorkflows := configObject["workflows"]
			if hasLegacyProfiles {
				report.add(CodeDeclarationLegacyBlock, "Legacy config.profiles block is no longer supported", manifestPath, "Move sources into config.declarations.sources and place files under .vrooli/agent-manager/.")
			}
			if hasLegacyWorkflows {
				report.add(CodeDeclarationLegacyBlock, "Legacy config.workflows block is no longer supported", manifestPath, "Move sources into config.declarations.sources and place files under .vrooli/agent-manager/.")
			}
			if _, hasDeclarations := configObject["declarations"]; hasDeclarations && !hasLegacyProfiles && !hasLegacyWorkflows {
				if err := orchestration.ValidateScenarioDeclarationConfig(manifestPath); err != nil {
					report.add(CodeProfileInvalid, "Agent declaration configuration invalid", manifestPath, err.Error())
				} else {
					declared = s.validateDeclaredSources(&report, abs, manifestPath, scenario, roles, configObject["declarations"])
				}
			}
		}
	}
	reportOrphanDeclarations(&report, abs, declared)
	reportDirectSpawnBypasses(&report, abs)
	reportPermissionPosture(&report, s.PermissionPosture)
	sortFindings(report.Findings)
	return report, nil
}

// validateSelfDeclarations validates agent-manager's own declaration files. It
// mirrors the scenario path's validators and ownership rules but discovers its
// sources from the .vrooli/agent-manager/ directory (every file is declared, so
// no orphan findings) and never requires an agent-manager dependency. A missing
// or empty directory is conformant.
func (s Service) validateSelfDeclarations(report Report, root, scenario string, roles map[string]bool) (Report, error) {
	// Files left under the retired directories are rejected for the provider too.
	reportLegacyLayoutFiles(&report, root)
	for _, source := range jsonFiles(filepath.Join(root, ".vrooli", "agent-manager"), root) {
		path := filepath.Join(root, filepath.FromSlash(source))
		data, err := os.ReadFile(path)
		if err != nil {
			report.add(CodeProfileInvalid, "Agent declaration source unreadable", path, err.Error())
			continue
		}
		version, err := peekDeclarationVersion(data)
		if err != nil {
			report.add(CodeProfileInvalid, "Agent declaration source malformed", path, err.Error())
			continue
		}
		switch version {
		case domain.ProfileSchemaVersionV1:
			validateDeclaredProfile(&report, scenario, roles, path, data)
		case domain.WorkflowSchemaVersionV1:
			validateDeclaredWorkflow(&report, scenario, roles, path, data)
		default:
			report.add(CodeProfileInvalid, "Agent declaration source has an unknown schemaVersion", path, fmt.Sprintf("schemaVersion must be %q or %q", domain.ProfileSchemaVersionV1, domain.WorkflowSchemaVersionV1))
		}
	}
	reportDirectSpawnBypasses(&report, root)
	reportPermissionPosture(&report, s.PermissionPosture)
	sortFindings(report.Findings)
	return report, nil
}

// validateDeclaredSources validates each unified declaration source with the
// shared validators, fanning out by schemaVersion, and returns the set of
// declared source paths (scenario-relative, slash form) so orphan detection can
// exclude them.
func (s Service) validateDeclaredSources(report *Report, root, manifestPath, scenario string, roles map[string]bool, declarationsRaw json.RawMessage) map[string]bool {
	declared := map[string]bool{}
	var block struct {
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal(declarationsRaw, &block); err != nil {
		report.add(CodeProfileInvalid, "Agent declaration configuration invalid", manifestPath, err.Error())
		return declared
	}
	for _, source := range block.Sources {
		path, err := resolveSource(root, source)
		if err != nil {
			report.add(CodeProfileInvalid, "Agent declaration source invalid", manifestPath, err.Error())
			continue
		}
		if rel, e := filepath.Rel(root, path); e == nil {
			declared[filepath.ToSlash(rel)] = true
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.add(CodeProfileInvalid, "Agent declaration source unreadable", path, err.Error())
			continue
		}
		version, err := peekDeclarationVersion(data)
		if err != nil {
			report.add(CodeProfileInvalid, "Agent declaration source malformed", path, err.Error())
			continue
		}
		switch version {
		case domain.ProfileSchemaVersionV1:
			validateDeclaredProfile(report, scenario, roles, path, data)
		case domain.WorkflowSchemaVersionV1:
			validateDeclaredWorkflow(report, scenario, roles, path, data)
		default:
			report.add(CodeProfileInvalid, "Agent declaration source has an unknown schemaVersion", path, fmt.Sprintf("schemaVersion must be %q or %q", domain.ProfileSchemaVersionV1, domain.WorkflowSchemaVersionV1))
		}
	}
	return declared
}

func validateDeclaredProfile(report *Report, scenario string, roles map[string]bool, path string, data []byte) {
	if legacyProfileField(data) {
		report.add(CodeProfileLegacy, "Legacy runner or model profile input", path, "Replace runner/model/policy inputs with roleRef.")
		return
	}
	profile, err := orchestration.ParseScenarioProfileSource(data)
	if err != nil {
		report.add(CodeProfileInvalid, "Agent profile does not satisfy the role-only contract", path, err.Error())
		return
	}
	if strings.TrimSpace(profile.RoleRef) == "" {
		report.add(CodeProfileInvalid, "Agent profile roleRef missing", path, "Set a portable roleRef such as code.default.")
	} else if !roles[profile.RoleRef] {
		report.add(CodeRoleUnresolved, "Agent profile roleRef is unresolved", path, "Choose a role declared by Agent Manager's role-policy catalog.")
	}
	if !strings.HasPrefix(profile.ProfileKey, scenario+"/") {
		report.add(CodeProfileOwnership, "Agent profile key is owned by another scenario", path, "Use a profileKey prefixed by "+scenario+"/.")
	}
}

func validateDeclaredWorkflow(report *Report, scenario string, roles map[string]bool, path string, data []byte) {
	parsed, err := workflowcatalog.Parse(data, nil)
	if err != nil {
		report.add(CodeWorkflowInvalid, "Agent workflow is malformed", path, err.Error())
		return
	}
	// Report every blocking diagnostic (CEL compile errors, unbound prompt
	// placeholders, structural defects) so the ladder surfaces the full defect
	// set, not just the first. Warnings are non-blocking and are surfaced at
	// reconcile/validate time rather than as conformance findings.
	if domain.HasBlockingDiagnostic(parsed.Diagnostics) {
		for _, d := range parsed.Diagnostics {
			if d.IsError() {
				report.add(CodeWorkflowInvalid, "Agent workflow contract is invalid", path, diagnosticMessage(d))
			}
		}
		return
	}
	if parsed.Definition.Owner != scenario || !strings.HasPrefix(parsed.Definition.Key, scenario+"/") {
		report.add(CodeWorkflowOwnership, "Agent workflow is owned by another scenario", path, "Use owner "+scenario+" and a key prefixed by "+scenario+"/.")
	}
	for _, node := range parsed.Definition.Nodes {
		if node.Run != nil && strings.TrimSpace(node.Run.PromptTemplate) != "" && node.Run.PromptRef == nil && node.Run.PromptProvenance == nil {
			report.add(CodeWorkflowInlinePrompt, "Workflow prompt must use promptRef at the mature rung", path, "Move the run prompt into a prompt-manager skill and declare promptRef.")
		}
		if node.Continue != nil && strings.TrimSpace(node.Continue.PromptTemplate) != "" && node.Continue.PromptRef == nil && node.Continue.PromptProvenance == nil {
			report.add(CodeWorkflowInlinePrompt, "Workflow prompt must use promptRef at the mature rung", path, "Move the continuation prompt into a prompt-manager skill and declare promptRef.")
		}
		if node.Run != nil && node.Run.RoleRef != "" && !roles[node.Run.RoleRef] {
			report.add(CodeRoleUnresolved, "Agent workflow roleRef is unresolved", path, "Choose a role declared by Agent Manager's role-policy catalog.")
		}
	}
}

func diagnosticMessage(d domain.WorkflowDiagnostic) string {
	if strings.TrimSpace(d.Path) != "" {
		return d.Path + ": " + d.Message
	}
	return d.Message
}

func peekDeclarationVersion(data []byte) (string, error) {
	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("parse declaration schemaVersion: %w", err)
	}
	return strings.TrimSpace(envelope.SchemaVersion), nil
}

// reportLegacyLayoutFiles rejects any declaration file that remains under the
// retired .vrooli/agent-profiles/ or .vrooli/agent-workflows/ directories.
func reportLegacyLayoutFiles(report *Report, root string) {
	for _, dir := range [][]string{{".vrooli", "agent-profiles"}, {".vrooli", "agent-workflows"}} {
		for _, source := range jsonFiles(filepath.Join(append([]string{root}, dir...)...), root) {
			report.add(CodeDeclarationLegacyLayout, "Declaration file remains under a retired directory", filepath.Join(root, filepath.FromSlash(source)), "Move this file into .vrooli/agent-manager/ and declare it under config.declarations.sources.")
		}
	}
}

// reportOrphanDeclarations flags files under the unified .vrooli/agent-manager/
// directory that no declared source references, routing each to the profile or
// workflow orphan code by its schemaVersion.
func reportOrphanDeclarations(report *Report, root string, declared map[string]bool) {
	for _, source := range jsonFiles(filepath.Join(root, ".vrooli", "agent-manager"), root) {
		if declared[source] {
			continue
		}
		code, title := CodeProfileOrphan, "Agent profile source is undeclared"
		if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source))); err == nil {
			if version, _ := peekDeclarationVersion(data); version == domain.WorkflowSchemaVersionV1 {
				code, title = CodeWorkflowOrphan, "Agent workflow source is undeclared"
			}
		}
		report.add(code, title, filepath.Join(root, filepath.FromSlash(source)), "Register this source under dependencies.scenarios.agent-manager.config.declarations.sources.")
	}
}

func jsonFiles(searchRoot, relativeRoot string) []string {
	var files []string
	_ = filepath.WalkDir(searchRoot, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".json") {
			if rel, e := filepath.Rel(relativeRoot, path); e == nil {
				files = append(files, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func loadRoles(repoRoot string) (map[string]bool, error) {
	revision, err := rolepolicy.Load(filepath.Join(repoRoot, "scenarios", "agent-manager", "config", "role-policy-catalog.json"))
	if err != nil {
		return nil, err
	}
	roles := make(map[string]bool, len(revision.Catalog().Roles))
	for role := range revision.Catalog().Roles {
		roles[role] = true
	}
	return roles, nil
}

func (r *Report) add(code, title, location, remediation string) {
	r.Findings = append(r.Findings, Finding{Code: code, Title: title, Location: location, Message: title, Remediation: remediation, Severity: "SEVERITY_ERROR"})
}

func reportDirectSpawnBypasses(report *Report, root string) {
	for _, path := range directSpawnBypasses(root) {
		report.add(CodeDirectSpawnBypass, "Direct coding-agent spawn bypass", path, "Request a profile key or portable role through Agent Manager instead of invoking a coding-agent executable directly.")
	}
}

func reportPermissionPosture(report *Report, reader PermissionPostureReader) {
	if reader == nil {
		return
	}
	if err := reader.ReadinessError(context.Background()); err != nil {
		report.add(CodePermissionPosture, "Global permission posture is not ready", "permission-policy", "Reconcile the active desired-permission catalog with explicit authorization and restore every required hard-enforcement rule: "+err.Error())
	}
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Code+findings[i].Location < findings[j].Code+findings[j].Location
	})
}

// directSpawnBypasses deliberately looks only for executable construction next
// to a known coding-agent command. It is advisory because static analysis is
// necessarily incomplete; resource-owned implementations are outside a target
// scenario and are never scanned here.
func directSpawnBypasses(root string) []string {
	var matches []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			if entry != nil && entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".sh":
		default:
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := strings.ToLower(string(data))
		if !(strings.Contains(text, "exec.command") || strings.Contains(text, "commandcontext") || strings.Contains(text, "spawn(") || strings.Contains(text, "execfile(") || strings.Contains(text, "subprocess.")) {
			return nil
		}
		for _, command := range []string{"\"claude\"", "\"codex\"", "\"opencode\"", "\"grok\"", "resource-claude-code", "resource-codex", "resource-opencode", "resource-grok"} {
			if strings.Contains(text, command) {
				matches = append(matches, path)
				break
			}
		}
		return nil
	})
	sort.Strings(matches)
	return matches
}

func resolveSource(root, source string) (string, error) {
	if strings.TrimSpace(source) == "" || filepath.IsAbs(source) {
		return "", fmt.Errorf("profile source must be a non-empty relative path")
	}
	clean := filepath.Clean(source)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("profile source must not escape the scenario root")
	}
	path := filepath.Join(root, clean)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve profile source: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("profile source escapes the scenario root")
	}
	return resolved, nil
}

func legacyProfileField(data []byte) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	for _, key := range []string{"runner", "runnerType", "model", "modelPreset", "policyRef"} {
		if _, found := raw[key]; found {
			return true
		}
	}
	return false
}
