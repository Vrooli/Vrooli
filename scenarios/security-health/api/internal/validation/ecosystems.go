package validation

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Ecosystem is intentionally a string contract. Adding an adapter must not
// require changing the security policy or the Code Facts broker.
type Ecosystem string

const (
	EcosystemGo     Ecosystem = "go"
	EcosystemPnpm   Ecosystem = "pnpm"
	EcosystemNPM    Ecosystem = "npm"
	EcosystemYarn   Ecosystem = "yarn"
	EcosystemBun    Ecosystem = "bun"
	EcosystemPython Ecosystem = "python"
	EcosystemRust   Ecosystem = "rust"
	EcosystemC      Ecosystem = "c"
	EcosystemCPP    Ecosystem = "cpp"
)

type CoverageState string

const (
	CoverageSupported   CoverageState = "supported"
	CoverageEvidence    CoverageState = "evidence_only"
	CoverageUnsupported CoverageState = "unsupported"
	CoverageUnknown     CoverageState = "unknown"
)

type EcosystemTarget struct {
	Ecosystem Ecosystem     `json:"ecosystem"`
	Root      string        `json:"root"`
	Manifests []string      `json:"manifests"`
	Coverage  CoverageState `json:"coverage"`
	Reason    string        `json:"reason,omitempty"`
}

// EcosystemAdapter is the discovery seam. Adapters consume Code Facts-style
// target roots/parse units when supplied and may use the same bounded
// filesystem fallback for direct CLI validation. They never own policy.
type EcosystemAdapter interface {
	ID() Ecosystem
	Capability() CoverageState
	Detect(root string) ([]EcosystemTarget, error)
}

type AdapterRegistry struct {
	byID map[Ecosystem]EcosystemAdapter
}

func NewAdapterRegistry(adapters ...EcosystemAdapter) *AdapterRegistry {
	r := &AdapterRegistry{byID: make(map[Ecosystem]EcosystemAdapter)}
	for _, adapter := range adapters {
		if adapter != nil {
			r.byID[adapter.ID()] = adapter
		}
	}
	return r
}

func DefaultAdapterRegistry() *AdapterRegistry {
	return NewAdapterRegistry(
		manifestAdapter{id: EcosystemGo, files: []string{"go.mod"}, capability: CoverageSupported},
		manifestAdapter{id: EcosystemPnpm, files: []string{"pnpm-lock.yaml"}, capability: CoverageSupported},
		manifestAdapter{id: EcosystemNPM, files: []string{"package-lock.json", "npm-shrinkwrap.json"}, capability: CoverageSupported},
		manifestAdapter{id: EcosystemYarn, files: []string{"yarn.lock"}, capability: CoverageSupported},
		manifestAdapter{id: EcosystemBun, files: []string{"bun.lock", "bun.lockb"}, capability: CoverageSupported},
		manifestAdapter{id: EcosystemPython, files: []string{"requirements.txt", "pyproject.toml", "Pipfile", "Pipfile.lock"}, capability: CoverageSupported},
		manifestAdapter{id: EcosystemRust, files: []string{"Cargo.toml", "Cargo.lock"}, capability: CoverageSupported},
		manifestAdapter{id: EcosystemC, files: []string{"CMakeLists.txt", "meson.build", "conanfile.txt", "vcpkg.json"}, capability: CoverageEvidence},
		manifestAdapter{id: EcosystemCPP, files: []string{"CMakeLists.txt", "meson.build", "conanfile.py", "conanfile.txt", "vcpkg.json"}, capability: CoverageEvidence},
	)
}

func (r *AdapterRegistry) Discover(root string) ([]EcosystemTarget, error) {
	if r == nil {
		return nil, fmt.Errorf("ecosystem adapter registry is nil")
	}
	var out []EcosystemTarget
	for _, adapter := range r.byID {
		targets, err := adapter.Detect(root)
		if err != nil {
			return nil, fmt.Errorf("discover %s: %w", adapter.ID(), err)
		}
		out = append(out, targets...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ecosystem != out[j].Ecosystem {
			return out[i].Ecosystem < out[j].Ecosystem
		}
		return out[i].Root < out[j].Root
	})
	return out, nil
}

// DiscoverFromFacts maps the target/surface/parse-unit facts emitted by Code
// Facts into the same adapter result. This deliberately accepts a small
// neutral projection so Security Health does not import a provider's generated
// transport types or duplicate Code Facts policy.
type FactTarget struct {
	Root      string
	Language  string
	Manager   string
	Files     []string
	ParseUnit string
}

func (r *AdapterRegistry) DiscoverFromFacts(facts []FactTarget) []EcosystemTarget {
	var out []EcosystemTarget
	for _, fact := range facts {
		matched := 0
		for id, adapter := range r.byID {
			if !factMatchesAdapter(fact, id) {
				continue
			}
			matched++
			out = append(out, EcosystemTarget{Ecosystem: id, Root: filepath.Clean(fact.Root), Manifests: sortedStrings(fact.Files), Coverage: adapter.Capability()})
		}
		if matched == 0 && isLanguageOnlyFact(fact) {
			out = append(out, EcosystemTarget{Ecosystem: Ecosystem(strings.ToLower(strings.TrimSpace(fact.Language))), Root: filepath.Clean(fact.Root), Manifests: sortedStrings(fact.Files), Coverage: CoverageUnknown, Reason: "language evidence exists but no package-manager manifest or lockfile identifies a scanner"})
		}
	}
	return dedupeTargets(out)
}

func isLanguageOnlyFact(f FactTarget) bool {
	if strings.TrimSpace(f.Manager) != "" || len(f.Files) == 0 {
		return false
	}
	language := strings.ToLower(strings.TrimSpace(f.Language))
	return strings.Contains(language, "javascript") || strings.Contains(language, "typescript") || strings.Contains(language, "python") || strings.Contains(language, "rust")
}

func factMatchesAdapter(f FactTarget, id Ecosystem) bool {
	language := strings.ToLower(strings.TrimSpace(f.Language + " " + f.ParseUnit))
	manager := strings.ToLower(strings.TrimSpace(f.Manager))
	switch id {
	case EcosystemGo:
		return manager == "go" || hasFile(f.Files, "go.mod")
	case EcosystemPnpm, EcosystemNPM, EcosystemYarn, EcosystemBun:
		// JavaScript/TypeScript is language evidence only. A manager is selected
		// only by its own lockfile or an explicit Code Facts manager field.
		if manager != "" {
			return manager == string(id)
		}
		switch id {
		case EcosystemPnpm:
			return hasFile(f.Files, "pnpm-lock.yaml")
		case EcosystemNPM:
			return hasAnyFile(f.Files, "package-lock.json", "npm-shrinkwrap.json")
		case EcosystemYarn:
			return hasFile(f.Files, "yarn.lock")
		case EcosystemBun:
			return hasAnyFile(f.Files, "bun.lock", "bun.lockb")
		}
		return false
	case EcosystemPython:
		return manager == "python" || manager == "pip" || strings.Contains(language, "python") && hasAnyFile(f.Files, "requirements.txt", "pyproject.toml", "Pipfile", "Pipfile.lock")
	case EcosystemRust:
		return manager == "rust" || manager == "cargo" || hasAnyFile(f.Files, "Cargo.toml", "Cargo.lock")
	case EcosystemC, EcosystemCPP:
		if manager != "" {
			return manager == string(id) || id == EcosystemC && manager == "cmake" || id == EcosystemCPP && manager == "cmake"
		}
		if id == EcosystemC {
			return strings.TrimSpace(strings.ToLower(f.Language)) == "c"
		}
		return strings.TrimSpace(strings.ToLower(f.Language)) == "c++" || strings.TrimSpace(strings.ToLower(f.Language)) == "cpp"
	default:
		return false
	}
}

type manifestAdapter struct {
	id         Ecosystem
	files      []string
	capability CoverageState
}

func (a manifestAdapter) ID() Ecosystem             { return a.id }
func (a manifestAdapter) Capability() CoverageState { return a.capability }

func (a manifestAdapter) Detect(root string) ([]EcosystemTarget, error) {
	found := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if _, skip := skipDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !hasFile(a.files, entry.Name()) {
			return nil
		}
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		found[dir] = append(found[dir], filepath.ToSlash(filepath.Join(dir, entry.Name())))
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []EcosystemTarget
	for dir, files := range found {
		if dir == "." {
			dir = ""
		}
		out = append(out, EcosystemTarget{Ecosystem: a.id, Root: filepath.Join(root, dir), Manifests: sortedStrings(files), Coverage: a.capability, Reason: coverageReason(a.capability)})
	}
	return out, nil
}

func coverageReason(state CoverageState) string {
	if state == CoverageEvidence {
		return "manifest evidence is indexed; vulnerability scanning is not claimed for this ecosystem"
	}
	return "adapter and scanner capability are available for this ecosystem"
}

func hasFile(files []string, want string) bool {
	for _, file := range files {
		if filepath.Base(file) == want || file == want {
			return true
		}
	}
	return false
}

func hasAnyFile(files []string, wants ...string) bool {
	for _, want := range wants {
		if hasFile(files, want) {
			return true
		}
	}
	return false
}

func sortedStrings(values []string) []string {
	copy := append([]string(nil), values...)
	sort.Strings(copy)
	return copy
}

func dedupeTargets(targets []EcosystemTarget) []EcosystemTarget {
	seen := map[string]struct{}{}
	out := make([]EcosystemTarget, 0, len(targets))
	for _, target := range targets {
		key := string(target.Ecosystem) + "\x00" + target.Root
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool {
		return string(out[i].Ecosystem)+out[i].Root < string(out[j].Ecosystem)+out[j].Root
	})
	return out
}
