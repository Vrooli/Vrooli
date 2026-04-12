package hostreqcheck

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

type FindingCode string

const (
	FindingUndeclaredReference FindingCode = "undeclared_reference"
	FindingMissingHandler      FindingCode = "missing_handler"
	FindingRootOverreach       FindingCode = "root_overreach"
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
	hostTools  []hostreq.Declaration
	safeguards []hostreq.Declaration
}

var (
	rootCoreToolAllowlist = map[string]struct{}{
		"curl":   {},
		"docker": {},
		"git":    {},
		"go":     {},
		"jq":     {},
		"node":   {},
		"python": {},
	}
	referenceScanCandidates = []string{
		"ffmpeg",
		"tmux",
		"helm",
		"yq",
		"shellcheck",
		"bats",
		"lychee",
		"ast-grep",
		"ajv",
		"js-yaml",
		"Xvfb",
		"xdotool",
		"x11vnc",
		"websockify",
		"openbox",
	}
	scannableExtensions = map[string]struct{}{
		".go": {},
		".sh": {},
	}
)

func Validate(root, home string) (Report, error) {
	owners, err := loadOwners(root, home)
	if err != nil {
		return Report{}, err
	}

	findings := make([]Finding, 0)
	for _, owner := range owners {
		findings = append(findings, validateDeclarations(owner)...)
		findings = append(findings, validateReferences(root, owner)...)
	}

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
	resourceItems, err := controller.Discover()
	if err != nil {
		return nil, fmt.Errorf("discover resources: %w", err)
	}
	for _, item := range resourceItems {
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
		})
	}

	scenarioItems, err := scenario.Discover(root, scenario.SandboxEnv{})
	if err != nil {
		return nil, fmt.Errorf("discover scenarios: %w", err)
	}
	for _, item := range scenarioItems {
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

func validateDeclarations(owner ownerManifest) []Finding {
	findings := make([]Finding, 0)

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
		if !runtime.HasHandler(hostreq.KindTool, name) {
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
		if !runtime.HasHandler(hostreq.KindSafeguard, name) {
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

	return findings
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
	case ".git", ".next", ".tmp", "coverage", "data", "dist", "docs", "node_modules", "resources", "scenarios", "vendor":
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
