package hostreqcheck

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

type FindingCode string

const (
	FindingUndeclaredReference   FindingCode = "undeclared_reference"
	FindingMissingHandler        FindingCode = "missing_handler"
	FindingRootOverreach         FindingCode = "root_overreach"
	FindingMissingClassification FindingCode = "missing_classification"
	FindingPrivilegeMismatch     FindingCode = "privilege_mismatch"
	FindingUnvendorable          FindingCode = "unvendorable"
)

type Finding struct {
	Code        FindingCode `json:"code"`
	OwnerKind   string      `json:"owner_kind"`
	OwnerName   string      `json:"owner_name"`
	Requirement string      `json:"requirement"`
	Source      string      `json:"source,omitempty"`
	Message     string      `json:"message"`
}

type Report struct {
	Findings []Finding `json:"findings"`
}

type ownerManifest struct {
	kind       string
	name       string
	basePath   string
	manifest   string
	privilege  string
	bundling   string
	hostTools  []hostreq.Declaration
	safeguards []hostreq.Declaration
}

var (
	// rootCoreToolAllowlist enumerates tools that are intentionally declared
	// at the root manifest level. Four categories qualify:
	//
	//   1. Universal Vrooli prerequisites (curl, docker, git, go, java, jq,
	//      node, python, yq) — needed by setup itself and many scenarios.
	//   2. Cross-scenario codegen toolchain (buf, protoc, protoc-gen-*) —
	//      drives the proto pipeline that all proto-aware scenarios consume.
	//   3. Cross-scenario formal verification (quint) — drives temporal-flow
	//      model checking and generated conformance artifacts for templates.
	//   4. Host-wide observability/forensics (rasdaemon, mcelog,
	//      kdump-tools) — captures crash data Vrooli's autoheal and
	//      system-monitor scenarios both read.
	rootCoreToolAllowlist = map[string]struct{}{
		"buf":                   {},
		"curl":                  {},
		"docker":                {},
		"git":                   {},
		"go":                    {},
		"java":                  {},
		"jq":                    {},
		"kdump-tools":           {},
		"mcelog":                {},
		"node":                  {},
		"pnpm":                  {},
		"protoc":                {},
		"protoc-gen-connect-go": {},
		"protoc-gen-es":         {},
		"protoc-gen-go":         {},
		"python":                {},
		"quint":                 {},
		"rasdaemon":             {},
		"yq":                    {},
	}
	referenceScanCandidates = []string{"stripe", "ffmpeg", "tmux", "helm", "yq", "bats", "ast-grep", "cloudflared", "lychee", "Xvfb", "xdotool", "x11vnc", "websockify", "openbox"}
	scannableExtensions     = map[string]struct{}{".go": {}, ".sh": {}}
)

func Validate(root, home string) (Report, error) {
	owners, err := loadOwners(root, home)
	if err != nil {
		return Report{}, err
	}

	findings := make([]Finding, 0)
	for _, owner := range owners {
		declarationFindings, declErr := validateDeclarations(owner)
		if declErr != nil {
			return Report{}, declErr
		}
		findings = append(findings, declarationFindings...)
	}
	findings = append(findings, validateCatalogDeclarations(root)...)

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code == findings[j].Code {
			if findings[i].OwnerKind == findings[j].OwnerKind {
				if findings[i].OwnerName == findings[j].OwnerName {
					if findings[i].Requirement == findings[j].Requirement {
						return findings[i].Source < findings[j].Source
					}
					return findings[i].Requirement < findings[j].Requirement
				}
				return findings[i].OwnerName < findings[j].OwnerName
			}
			return findings[i].OwnerKind < findings[j].OwnerKind
		}
		return findings[i].Code < findings[j].Code
	})

	return Report{Findings: findings}, nil
}

// validateCatalogDeclarations is deliberately limited to the deployable
// substrate. Scenario source scans are too broad to be a deployment gate: a
// scenario can invoke a command for an optional operational path without that
// command being a desktop requirement. The catalogue manifests and the native
// handler sources are the authoritative pair for this conformance check.
func validateCatalogDeclarations(root string) []Finding {
	findings := make([]Finding, 0)
	findings = append(findings, validateToolCatalog(root)...)
	findings = append(findings, validateSafeguardCatalog(root)...)
	return findings
}

func validateToolCatalog(root string) []Finding {
	entries, err := os.ReadDir(filepath.Join(root, "internal", "tools"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{Code: FindingMissingClassification, OwnerKind: "tool", OwnerName: "catalog", Message: err.Error()}}
	}
	findings := make([]Finding, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "internal", "tools", entry.Name(), "tool.json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var manifest hostreqkit.ToolManifest
		if json.Unmarshal(data, &manifest) != nil {
			continue // schema/registry loaders report malformed manifests separately.
		}
		if manifest.Bundling == "vendorable" && !hasPortableToolSource(manifest) {
			findings = append(findings, Finding{Code: FindingUnvendorable, OwnerKind: "tool", OwnerName: manifest.Name, Source: manifestSource(path), Message: "tool declares bundling vendorable without a checksummed per-platform source target"})
		}
		if sourceRequiresElevation(filepath.Join(root, "internal", "tools", entry.Name())) && effectiveToolPrivilege(manifest) != "elevated" {
			findings = append(findings, privilegeFinding("tool", manifest.Name, path, effectiveToolPrivilege(manifest), "handler invokes sudo or writes a system path"))
		}
	}
	return findings
}

func validateSafeguardCatalog(root string) []Finding {
	entries, err := os.ReadDir(filepath.Join(root, "internal", "safeguards"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{Code: FindingMissingClassification, OwnerKind: "safeguard", OwnerName: "catalog", Message: err.Error()}}
	}
	findings := make([]Finding, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "internal", "safeguards", entry.Name(), "safeguard.json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var manifest hostreqkit.SafeguardManifest
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		evidence := sourceRequiresElevation(filepath.Join(root, "internal", "safeguards", entry.Name()))
		if manifest.VerificationCheck != nil {
			for _, file := range manifest.VerificationCheck.Files {
				if isSystemPath(file) {
					evidence = true
					break
				}
			}
		}
		if evidence && manifest.Privilege != "elevated" {
			findings = append(findings, privilegeFinding("safeguard", manifest.Name, path, manifest.Privilege, "handler or verification check accesses a system path"))
		}
	}
	return findings
}

func effectiveToolPrivilege(manifest hostreqkit.ToolManifest) hostreqspec.Privilege {
	if manifest.Privilege != "" {
		return manifest.Privilege
	}
	if manifest.SourceType() == "package" {
		return hostreqspec.PrivilegeElevated
	}
	return hostreqspec.PrivilegeUser
}

func hasPortableToolSource(manifest hostreqkit.ToolManifest) bool {
	if manifest.Source == nil || len(manifest.Source.Targets) == 0 {
		return false
	}
	for _, target := range manifest.Source.Targets {
		if strings.TrimSpace(target.URL) == "" || len(strings.TrimSpace(target.SHA256)) != 64 {
			return false
		}
	}
	return true
}

func sourceRequiresElevation(dir string) bool {
	requires := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || (filepath.Ext(path) != ".go" && filepath.Ext(path) != ".sh") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(data)
		if strings.Contains(text, "sudo") || strings.Contains(text, "/etc/") || strings.Contains(text, "/usr/") || strings.Contains(text, "/var/") {
			requires = true
		}
		return nil
	})
	return requires
}

func isSystemPath(path string) bool {
	clean := filepath.ToSlash(strings.TrimSpace(path))
	return strings.HasPrefix(clean, "/etc/") || strings.HasPrefix(clean, "/usr/") || strings.HasPrefix(clean, "/var/") || strings.HasPrefix(clean, "/opt/")
}

func privilegeFinding(kind, name, path string, declared hostreqspec.Privilege, evidence string) Finding {
	return Finding{Code: FindingPrivilegeMismatch, OwnerKind: kind, OwnerName: name, Source: manifestSource(path), Message: fmt.Sprintf("declares privilege %q but %s; required value is %q", declared, evidence, hostreqspec.PrivilegeElevated)}
}

func loadOwners(root, home string) ([]ownerManifest, error) {
	rootManifestPath := filepath.Join(root, ".vrooli", "service.json")
	rootManifest, err := scenario.ReadService(rootManifestPath)
	if err != nil {
		return nil, fmt.Errorf("load root manifest: %w", err)
	}

	owners := []ownerManifest{{
		kind:       "root",
		name:       "vrooli",
		basePath:   root,
		manifest:   rootManifestPath,
		hostTools:  rootManifest.HostTools,
		safeguards: rootManifest.HostSafeguards,
	}}

	controller := resources.NewController(root, home)
	resourceReport, err := controller.DiscoverReport()
	if err != nil {
		return nil, fmt.Errorf("discover resources: %w", err)
	}
	for _, item := range resourceReport.Items {
		if strings.TrimSpace(item.ManifestPath) == "" {
			continue
		}
		manifest, err := controller.LoadManifest(item.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("load resource manifest %s: %w", item.Name, err)
		}
		owners = append(owners, ownerManifest{
			kind:       "resource",
			name:       item.Name,
			basePath:   filepath.Join(root, "resources", item.Name),
			manifest:   item.ManifestPath,
			hostTools:  manifest.HostTools,
			safeguards: manifest.HostSafeguards,
			privilege:  string(manifest.Privilege),
			bundling:   string(manifest.Bundling),
		})
	}

	scenarioReport, err := scenario.DiscoverReport(root, scenario.SandboxEnv{})
	if err != nil {
		return nil, fmt.Errorf("discover scenarios: %w", err)
	}
	for _, item := range scenarioReport.Items {
		owners = append(owners, ownerManifest{
			kind:       "scenario",
			name:       item.Slug,
			basePath:   item.Path,
			manifest:   item.ServicePath,
			hostTools:  item.Manifest.HostTools,
			safeguards: item.Manifest.HostSafeguards,
		})
	}

	return owners, nil
}

func validateDeclarations(owner ownerManifest) ([]Finding, error) {
	findings := make([]Finding, 0)
	if owner.kind == "resource" {
		if owner.privilege == "" || owner.bundling == "" {
			findings = append(findings, Finding{
				Code: FindingMissingClassification, OwnerKind: owner.kind, OwnerName: owner.name,
				Source:  manifestSource(owner.manifest),
				Message: "resource must declare both privilege and bundling for deployment eligibility",
			})
		}
	}

	for _, declaration := range owner.hostTools {
		name := strings.TrimSpace(declaration.Name)
		if owner.kind == "root" {
			if _, ok := rootCoreToolAllowlist[name]; !ok {
				findings = append(findings, Finding{
					Code:        FindingRootOverreach,
					OwnerKind:   owner.kind,
					OwnerName:   owner.name,
					Requirement: name,
					Source:      manifestSource(owner.manifest),
					Message:     fmt.Sprintf("root manifest declares specialized tool %q; keep root core intentionally small", name),
				})
			}
		}
		has, err := runtime.HasHandler(hostreq.KindTool, name)
		if err != nil {
			return nil, fmt.Errorf("runtime registry unavailable while validating %s/%s: %w", owner.kind, owner.name, err)
		}
		if !has {
			findings = append(findings, Finding{
				Code:        FindingMissingHandler,
				OwnerKind:   owner.kind,
				OwnerName:   owner.name,
				Requirement: name,
				Source:      manifestSource(owner.manifest),
				Message:     fmt.Sprintf("declared host tool %q has no native runtime handler", name),
			})
		}
	}

	for _, declaration := range owner.safeguards {
		name := strings.TrimSpace(declaration.Name)
		has, err := runtime.HasHandler(hostreq.KindSafeguard, name)
		if err != nil {
			return nil, fmt.Errorf("runtime registry unavailable while validating %s/%s: %w", owner.kind, owner.name, err)
		}
		if !has {
			findings = append(findings, Finding{
				Code:        FindingMissingHandler,
				OwnerKind:   owner.kind,
				OwnerName:   owner.name,
				Requirement: name,
				Source:      manifestSource(owner.manifest),
				Message:     fmt.Sprintf("declared host safeguard %q has no native runtime handler", name),
			})
		}
	}

	return findings, nil
}

func validateReferences(root string, owner ownerManifest) []Finding {
	declared := make(map[string]struct{}, len(owner.hostTools))
	for _, declaration := range owner.hostTools {
		declared[strings.TrimSpace(declaration.Name)] = struct{}{}
	}

	findings := make([]Finding, 0)
	seen := map[string]struct{}{}
	_ = filepath.WalkDir(owner.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(owner, path, d.Name()) {
				return filepath.SkipDir
			}
			switch d.Name() {
			case ".git", ".next", "coverage", "data", "dist", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !shouldScanFile(path) || path == owner.manifest {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(content)
		for _, candidate := range referenceScanCandidates {
			if _, ok := declared[candidate]; ok {
				continue
			}
			if !containsCandidateReference(text, candidate) {
				continue
			}
			key := candidate + "|" + path
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			findings = append(findings, Finding{
				Code:        FindingUndeclaredReference,
				OwnerKind:   owner.kind,
				OwnerName:   owner.name,
				Requirement: candidate,
				Source:      relSource(root, path),
				Message:     fmt.Sprintf("%s %q references %q without declaring it in the owner manifest", owner.kind, owner.name, candidate),
			})
		}
		return nil
	})
	return findings
}

func shouldScanFile(path string) bool {
	if info, err := os.Stat(path); err == nil && info.Size() > 1<<20 {
		return false
	}
	base := filepath.Base(path)
	if base == "package.json" {
		return true
	}
	_, ok := scannableExtensions[filepath.Ext(path)]
	return ok
}

func containsCandidateReference(text, candidate string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isIgnorableCommentLine(trimmed, candidate) {
			continue
		}
		if strings.Contains(line, "command -v "+candidate) ||
			strings.Contains(line, "which "+candidate) ||
			containsCommandCallReference(line, candidate) ||
			containsShellCommand(line, candidate) {
			return true
		}
	}
	return false
}

func shouldSkipDir(owner ownerManifest, path, base string) bool {
	if owner.kind != "root" {
		return false
	}
	if path == owner.basePath {
		return false
	}
	switch base {
	case ".git", ".next", ".tmp", "__test", "coverage", "data", "dist", "docs", "internal", "k8s", "node_modules", "resources", "scenarios", "scripts", "vendor":
		return true
	default:
		return false
	}
}

func isIgnorableCommentLine(line, candidate string) bool {
	switch {
	case strings.HasPrefix(line, "#!/"):
		return !(candidate == "bats" && containsShellCommand(line, candidate))
	case strings.HasPrefix(line, "#"),
		strings.HasPrefix(line, "//"),
		strings.HasPrefix(line, "/*"),
		strings.HasPrefix(line, "*"):
		return true
	default:
		return false
	}
}

func containsCommandCallReference(text, token string) bool {
	quoted := `"` + token + `"`
	patterns := []string{
		"exec.Command(" + quoted,
		"exec.CommandContext(",
		"exec.LookPath(" + quoted + ")",
		".LookPath(" + quoted + ")",
		".shell(",
		"ExecuteWithResult(ctx, " + quoted,
	}
	for _, pattern := range patterns {
		if !strings.Contains(text, pattern) {
			continue
		}
		if strings.Contains(pattern, "CommandContext(") || strings.Contains(pattern, ".shell(") {
			if strings.Contains(text, quoted) {
				return true
			}
			continue
		}
		return true
	}
	return false
}

func containsShellCommand(text, token string) bool {
	if token == "" {
		return false
	}
	index := 0
	for {
		offset := strings.Index(text[index:], token)
		if offset < 0 {
			return false
		}
		start := index + offset
		end := start + len(token)
		if shellCommandBoundary(text, start-1) && shellCommandTailBoundary(text, end) {
			return true
		}
		index = end
	}
}

func shellCommandBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return true
	}
	switch text[index] {
	case ' ', '\t', '(', '|', '&', ';':
		return true
	default:
		return false
	}
}

func shellCommandTailBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return true
	}
	next := index
	for next < len(text) && (text[next] == ' ' || text[next] == '\t') {
		next++
	}
	if next >= len(text) {
		return true
	}
	switch text[next] {
	case '\n', '\r', ')', ';', '|', '&', ',':
		return true
	case ':', '=':
		return false
	default:
		return text[index] == ' ' || text[index] == '\t'
	}
}

func manifestSource(path string) string {
	return filepath.ToSlash(path)
}

func relSource(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}
