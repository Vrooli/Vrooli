package hostreqcheck

import (
	"encoding/json"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
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

// rootCoreToolAllowlist enumerates tools that are intentionally declared
// at the root manifest level. Four categories qualify:
//
//  1. Universal Vrooli prerequisites (curl, docker, git, go, java, jq,
//     node, python, yq) — needed by setup itself and many scenarios.
//  2. Cross-scenario codegen toolchain (buf, protoc, protoc-gen-*) —
//     drives the proto pipeline that all proto-aware scenarios consume.
//  3. Cross-scenario formal verification (quint) — drives temporal-flow
//     model checking and generated conformance artifacts for templates.
//  4. Host-wide observability/forensics (rasdaemon, mcelog,
//     kdump-tools) — captures crash data Vrooli's autoheal and
//     system-monitor scenarios both read.
var rootCoreToolAllowlist = map[string]struct{}{
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
	if manifest.Acquisition == nil || len(manifest.Acquisition.Targets) == 0 {
		return false
	}
	for _, target := range manifest.Acquisition.Targets {
		if strings.TrimSpace(target.Unsupported) != "" {
			continue
		}
		if strings.TrimSpace(target.URL) != "" && len(strings.TrimSpace(target.SHA256)) == 64 {
			return true
		}
		if image := strings.TrimSpace(target.Image); image != "" && strings.Contains(image, "@sha256:") {
			return true
		}
	}
	return false
}

// sourceRequiresElevation reports whether a tool or safeguard's implementation
// shows evidence of needing elevated privilege.
//
// TEST FILES ARE EXCLUDED. A test names system paths as FIXTURES — path-hygiene
// builds `"/usr/bin:/bin"` to exercise PATH rewriting, and matching on that
// concluded the safeguard mutates `/usr`, when it only edits shell rc files
// under $HOME. The consequence of that false positive is not a noisy warning:
// the only way to satisfy it is to declare `privilege: elevated`, which would
// make `vrooli setup` request elevation a safeguard does not need. Privilege is
// a consent boundary, so over-declaring it is a real defect, not a safe default.
func sourceRequiresElevation(dir string) bool {
	requires := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || (filepath.Ext(path) != ".go" && filepath.Ext(path) != ".sh") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if sourceTextRequiresElevation(data, filepath.Ext(path)) {
			requires = true
		}
		return nil
	})
	return requires
}

// sourceTextRequiresElevation examines executable source tokens rather than
// raw file text. Comments describe behavior and must not be treated as proof
// that a handler performs privileged work.
func sourceTextRequiresElevation(data []byte, extension string) bool {
	if extension != ".go" {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, "sudo") || strings.Contains(line, "/etc/") || strings.Contains(line, "/usr/") || strings.Contains(line, "/var/") {
				return true
			}
		}
		return false
	}

	var fileSet token.FileSet
	var tokens scanner.Scanner
	tokens.Init(fileSet.AddFile("source.go", -1, len(data)), data, nil, 0)
	for {
		_, kind, literal := tokens.Scan()
		if kind == token.EOF {
			return false
		}
		if kind == token.IDENT && literal == "sudo" {
			return true
		}
		if kind == token.STRING && (strings.Contains(literal, "sudo") || strings.Contains(literal, "/etc/") || strings.Contains(literal, "/usr/") || strings.Contains(literal, "/var/")) {
			return true
		}
	}
}

func isSystemPath(path string) bool {
	clean := filepath.ToSlash(strings.TrimSpace(path))
	return strings.HasPrefix(clean, "/etc/") || strings.HasPrefix(clean, "/usr/") || strings.HasPrefix(clean, "/var/") || strings.HasPrefix(clean, "/opt/")
}

func privilegeFinding(kind, name, path string, declared hostreqspec.Privilege, evidence string) Finding {
	return Finding{Code: FindingPrivilegeMismatch, OwnerKind: kind, OwnerName: name, Source: manifestSource(path), Message: fmt.Sprintf("declares privilege %q but %s; required value is %q", declared, evidence, hostreqspec.PrivilegeElevated)}
}

func loadOwners(root, home string) ([]ownerManifest, error) {
	rootManifestPath := filepath.Join(root, repocontractmeta.ProjectConfigDir, "service.json")
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

func manifestSource(path string) string {
	return filepath.ToSlash(path)
}
