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

	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/workflowcatalog"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

var scenarioNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const (
	CodeDependencyMissing  = "agent_conformance.dependency_missing"
	CodeDependencyDisabled = "agent_conformance.dependency_disabled"
	CodeProfileInvalid     = "agent_conformance.profile_invalid"
	CodeProfileOrphan      = "agent_conformance.profile_orphan"
	CodeProfileOwnership   = "agent_conformance.profile_ownership_mismatch"
	CodeProfileLegacy      = "agent_conformance.profile_legacy_field"
	CodeRoleUnresolved     = "agent_conformance.role_unresolved"
	CodeDirectSpawnBypass  = "agent_conformance.direct_spawn_bypass"
	CodePermissionPosture  = "agent_conformance.permission_posture"
	CodeWorkflowInvalid    = "agent_conformance.workflow_invalid"
	CodeWorkflowOrphan     = "agent_conformance.workflow_orphan"
	CodeWorkflowOwnership  = "agent_conformance.workflow_ownership_mismatch"
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
	if !present {
		report.add(CodeDependencyMissing, "Agent Manager dependency missing", manifestPath, "Declare dependencies.scenarios.agent-manager with declared profile sources.")
		reportOrphanProfiles(&report, abs)
		reportDirectSpawnBypasses(&report, abs)
		sortFindings(report.Findings)
		return report, nil
	}
	if dep.Enabled != nil && !*dep.Enabled {
		report.add(CodeDependencyDisabled, "Agent Manager dependency disabled", manifestPath, "Enable the Agent Manager dependency before requesting coding-agent profiles.")
		reportOrphanProfiles(&report, abs)
		reportDirectSpawnBypasses(&report, abs)
		sortFindings(report.Findings)
		return report, nil
	}
	var config struct {
		Profiles struct {
			Sources []string `json:"sources"`
		} `json:"profiles"`
		Workflows struct {
			Sources []string `json:"sources"`
		} `json:"workflows"`
	}
	// A consumer may request a portable role directly at runtime and therefore
	// legitimately have no scenario-owned profile sources. Once it supplies
	// dependency configuration, however, that configuration is the strict
	// profile-source contract it must satisfy.
	if len(dep.Config) > 0 {
		if err := orchestration.ValidateScenarioProfileConfig(manifestPath); err != nil {
			report.add(CodeProfileInvalid, "Agent profile configuration invalid", manifestPath, err.Error())
			return report, nil
		}
		if err := orchestration.ValidateScenarioWorkflowConfig(manifestPath); err != nil {
			report.add(CodeWorkflowInvalid, "Agent workflow configuration invalid", manifestPath, err.Error())
			return report, nil
		}
		if json.Unmarshal(dep.Config, &config) != nil {
			report.add(CodeProfileInvalid, "Agent profile configuration invalid", manifestPath, "Declare a valid config.profiles.sources list for Agent Manager.")
			return report, nil
		}
	}
	declaredSources := make(map[string]bool, len(config.Profiles.Sources))
	for _, source := range config.Profiles.Sources {
		path, err := resolveSource(abs, source)
		if err != nil {
			report.add(CodeProfileInvalid, "Agent profile source invalid", manifestPath, err.Error())
			continue
		}
		rel, err := filepath.Rel(abs, path)
		if err == nil {
			declaredSources[filepath.ToSlash(rel)] = true
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.add(CodeProfileInvalid, "Agent profile unreadable", path, err.Error())
			continue
		}
		if legacyProfileField(data) {
			report.add(CodeProfileLegacy, "Legacy runner or model profile input", path, "Replace runner/model/policy inputs with roleRef.")
			continue
		}
		var proto domainpb.AgentProfile
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, &proto); err != nil {
			report.add(CodeProfileInvalid, "Agent profile does not satisfy the role-only contract", path, err.Error())
			continue
		}
		profile := protoconv.AgentProfileFromProto(&proto)
		if strings.TrimSpace(profile.RoleRef) == "" {
			report.add(CodeProfileInvalid, "Agent profile roleRef missing", path, "Set a portable roleRef such as code.default.")
		} else if !roles[profile.RoleRef] {
			report.add(CodeRoleUnresolved, "Agent profile roleRef is unresolved", path, "Choose a role declared by Agent Manager's role-policy catalog.")
		}
		if !strings.HasPrefix(profile.ProfileKey, scenario+"/") {
			report.add(CodeProfileOwnership, "Agent profile key is owned by another scenario", path, "Use a profileKey prefixed by "+scenario+"/.")
		}
	}
	reportOrphanProfiles(&report, abs, declaredSources)
	declaredWorkflows := make(map[string]bool, len(config.Workflows.Sources))
	for _, source := range config.Workflows.Sources {
		path, err := resolveSource(abs, source)
		if err != nil {
			report.add(CodeWorkflowInvalid, "Agent workflow source invalid", manifestPath, err.Error())
			continue
		}
		if rel, err := filepath.Rel(abs, path); err == nil {
			declaredWorkflows[filepath.ToSlash(rel)] = true
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.add(CodeWorkflowInvalid, "Agent workflow unreadable", path, err.Error())
			continue
		}
		parsed, err := workflowcatalog.Parse(data, nil)
		if err != nil {
			report.add(CodeWorkflowInvalid, "Agent workflow is malformed", path, err.Error())
			continue
		}
		if len(parsed.Diagnostics) != 0 {
			report.add(CodeWorkflowInvalid, "Agent workflow contract is invalid", path, parsed.Diagnostics[0].Message)
			continue
		}
		if parsed.Definition.Owner != scenario || !strings.HasPrefix(parsed.Definition.Key, scenario+"/") {
			report.add(CodeWorkflowOwnership, "Agent workflow is owned by another scenario", path, "Use owner "+scenario+" and a key prefixed by "+scenario+"/.")
		}
		for _, node := range parsed.Definition.Nodes {
			if node.Run != nil && node.Run.RoleRef != "" && !roles[node.Run.RoleRef] {
				report.add(CodeRoleUnresolved, "Agent workflow roleRef is unresolved", path, "Choose a role declared by Agent Manager's role-policy catalog.")
			}
		}
	}
	reportOrphanWorkflows(&report, abs, declaredWorkflows)
	reportDirectSpawnBypasses(&report, abs)
	reportPermissionPosture(&report, s.PermissionPosture)
	sortFindings(report.Findings)
	return report, nil
}

func reportOrphanWorkflows(report *Report, root string, declared map[string]bool) {
	for _, source := range jsonFiles(filepath.Join(root, ".vrooli", "agent-workflows"), root) {
		if !declared[source] {
			report.add(CodeWorkflowOrphan, "Agent workflow source is undeclared", filepath.Join(root, filepath.FromSlash(source)), "Register this workflow under dependencies.scenarios.agent-manager.config.workflows.sources.")
		}
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

func reportOrphanProfiles(report *Report, root string, declared ...map[string]bool) {
	declaredSources := map[string]bool{}
	if len(declared) > 0 {
		declaredSources = declared[0]
	}
	for _, source := range profileFiles(root) {
		if !declaredSources[source] {
			report.add(CodeProfileOrphan, "Agent profile source is undeclared", filepath.Join(root, filepath.FromSlash(source)), "Register this scenario-owned profile under dependencies.scenarios.agent-manager.config.profiles.sources.")
		}
	}
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

func profileFiles(root string) []string {
	profilesRoot := filepath.Join(root, ".vrooli", "agent-profiles")
	var files []string
	_ = filepath.WalkDir(profilesRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".json") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	return files
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
