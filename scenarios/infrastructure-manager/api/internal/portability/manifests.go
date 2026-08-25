// Package portability aggregates the platform declarations that are authored
// beside every tool, safeguard, resource and scenario manifest, and resolves
// them through the control plane's pure resolver.
//
// The aggregation lives here, in the instrument, because aggregating is
// instrument work. The declarations themselves stay with the thing they
// describe: a tool's platform block belongs in that tool's manifest. Pulling
// the declarations in here would turn this domain into a second roster of the
// fleet, and a roster drifts from the thing it lists the moment either side
// changes. This package therefore only ever reads.
//
// The resolver stays in the control plane's `deployability` package for the
// same reason in the other direction: `vrooli setup` must be able to resolve a
// capability with no scenario running.
package portability

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/deployability"
)

// operatingSystems is the host OS axis of the grid. Every capability is
// resolved against all three, so an OS with no declaration is reported as
// peerless rather than being absent from the row.
var operatingSystems = []deployability.HostOS{
	deployability.HostOSLinux,
	deployability.HostOSMacOS,
	deployability.HostOSWindows,
}

// UnresolvedRootError is returned when the manifest tree cannot be located.
// It names the path that was tried and the file whose absence was decisive, so
// an operator can tell "wrong root" from "right root, missing vocabulary". A
// missing root is never degraded into an empty grid: an empty grid reads as
// "this repository declares no capabilities", which is a claim, not a failure.
type UnresolvedRootError struct {
	Root   string
	Wanted string
	Err    error
}

func (e UnresolvedRootError) Error() string {
	if strings.TrimSpace(e.Root) == "" {
		return "capability manifest root is not configured; no repository root was resolved, so the capability grid cannot be computed"
	}
	return fmt.Sprintf("capability manifest root %q is not usable: %s is unreadable (%v)", e.Root, e.Wanted, e.Err)
}

func (e UnresolvedRootError) Unwrap() error { return e.Err }

// Reader is the typed manifest reader. It is constructed against one resolved
// repository root and reports that root on every readout it feeds, because a
// grid computed against the wrong tree is a complete-looking answer about a
// repository nobody asked about.
type Reader struct {
	root string
}

// NewReader validates the root eagerly. Construction fails when the root is
// empty or does not carry the capability vocabulary, so the failure surfaces
// at wiring time rather than as a silently short grid at read time.
func NewReader(root string) (*Reader, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, UnresolvedRootError{}
	}
	path := vocabularyPath(root)
	if _, err := os.Stat(path); err != nil {
		return nil, UnresolvedRootError{Root: root, Wanted: relativeTo(root, path), Err: err}
	}
	return &Reader{root: root}, nil
}

// Root returns the resolved repository root this reader reads from.
func (r *Reader) Root() string { return r.root }

// Vocabulary is the operator-owned list of capability names, plus the per-OS
// policy that says when an absent implementation is a deliberate decision
// rather than a gap. The reserved "hardware-persistence" policy key carries
// dated qualification metadata; it is not a capability row and is ignored by
// resolution.
type Vocabulary struct {
	Capabilities         []string                                `json:"capabilities"`
	PlatformPolicies     map[string]map[string]string            `json:"platform_policies,omitempty"`
	ControlPolicies      map[string]map[string]map[string]string `json:"control_policies,omitempty"`
	ControlPolicyReasons map[string]map[string]map[string]string `json:"control_policy_reasons,omitempty"`
}

// Manifest is one capability declaration as authored. Status stays a raw
// string here: turning a token into vocabulary is the resolver's job, and a
// token outside the vocabulary must reach the resolver intact so it can be
// reported rather than silently downgraded.
type Manifest struct {
	Path                 string                         `json:"-"`
	Name                 string                         `json:"name"`
	Capability           string                         `json:"capability"`
	Role                 string                         `json:"capability_role"`
	Platforms            []string                       `json:"platforms"`
	PlatformStatus       map[string]PlatformDeclaration `json:"platform_status,omitempty"`
	PlatformDeclarations map[string]PlatformDeclaration `json:"platform_declarations,omitempty"`
	Packages             map[string]interface{}         `json:"packages"`
	Source               json.RawMessage                `json:"source"`
	Handler              json.RawMessage                `json:"handler"`
	Manual               bool                           `json:"manual"`
}

type PlatformDeclaration struct {
	Status    string          `json:"status"`
	Mechanism string          `json:"mechanism"`
	Evidence  json.RawMessage `json:"evidence"`
}

func evidenceValue(raw json.RawMessage) *deployability.Evidence {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var evidence deployability.Evidence
	if json.Unmarshal(raw, &evidence) != nil || !evidence.Complete() {
		return nil
	}
	return &evidence
}

// ResourceInput is the slice of a resource manifest the fleet view reads.
type ResourceInput struct {
	Name         string                              `json:"name"`
	Bundling     deployability.Bundling              `json:"bundling"`
	Platforms    map[string]string                   `json:"platforms"`
	Requirements *deployability.ResourceRequirements `json:"requirements"`
	Deployment   ResourceDeploymentInput             `json:"deployment"`
}

type ResourceProfileInput struct {
	Requires      []string `json:"requires"`
	Architectures []string `json:"architectures"`
}

type ResourceDeploymentInput struct {
	Profiles map[string]map[string]ResourceProfileInput `json:"profiles"`
}

// ScenarioInput is the slice of a scenario service manifest the fleet view
// reads: which capabilities it claims and which resources it depends on.
type ScenarioInput struct {
	Name         string
	Capabilities []string
	Resources    map[string]json.RawMessage
	Swaps        []deployability.SwapSource
}

// vocabularyPath is the single place the capability vocabulary file lives.
func vocabularyPath(root string) string {
	return filepath.Join(root, ".vrooli", "capability-vocabulary.json")
}

func relativeTo(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return path
}

// Vocabulary reads the operator-owned capability list. The names are sorted so
// the grid's row order is a property of the vocabulary rather than of map
// iteration.
func (r *Reader) Vocabulary() (Vocabulary, error) {
	path := vocabularyPath(r.root)
	data, err := os.ReadFile(path)
	if err != nil {
		return Vocabulary{}, UnresolvedRootError{Root: r.root, Wanted: relativeTo(r.root, path), Err: err}
	}
	var vocabulary Vocabulary
	if err := json.Unmarshal(data, &vocabulary); err != nil {
		return Vocabulary{}, fmt.Errorf("decode capability vocabulary %s: %w", path, err)
	}
	sort.Strings(vocabulary.Capabilities)
	return vocabulary, nil
}

// CapabilityManifests reads every tool, safeguard and scenario declaration
// that names a capability. A scenario's platform_capabilities block yields one
// synthetic manifest per capability so a scenario claiming three capabilities
// competes on three rows rather than one.
func (r *Reader) CapabilityManifests() ([]Manifest, error) {
	paths := make([]string, 0)
	for _, pattern := range []string{"internal/tools/*/tool.json", "internal/safeguards/*/safeguard.json"} {
		matches, err := filepath.Glob(filepath.Join(r.root, pattern))
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	manifests := make([]Manifest, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read capability manifest %s: %w", path, err)
		}
		var item Manifest
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, fmt.Errorf("decode capability manifest %s: %w", path, err)
		}
		if strings.TrimSpace(item.Name) == "" {
			item.Name = filepath.Base(filepath.Dir(path))
		}
		item.Path = path
		manifests = append(manifests, item)
	}
	servicePaths, err := filepath.Glob(filepath.Join(r.root, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(servicePaths)
	for _, path := range servicePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read service capability manifest %s: %w", path, err)
		}
		var service struct {
			Service struct {
				Name                 string                                    `json:"name"`
				Capabilities         []string                                  `json:"capabilities"`
				PlatformCapabilities map[string]map[string]PlatformDeclaration `json:"platform_capabilities"`
			} `json:"service"`
		}
		if err := json.Unmarshal(data, &service); err != nil {
			return nil, fmt.Errorf("decode service capability manifest %s: %w", path, err)
		}
		for capability, declarations := range service.Service.PlatformCapabilities {
			if strings.TrimSpace(capability) == "" {
				continue
			}
			name := strings.TrimSpace(service.Service.Name)
			if name == "" {
				name = filepath.Base(filepath.Dir(filepath.Dir(path)))
			}
			manifests = append(manifests, Manifest{
				Path: path, Name: name + "/" + capability, Capability: capability, Role: "primary",
				PlatformDeclarations: declarations,
			})
		}
	}
	sort.Slice(manifests, func(i, j int) bool {
		if manifests[i].Capability != manifests[j].Capability {
			return manifests[i].Capability < manifests[j].Capability
		}
		return manifests[i].Name < manifests[j].Name
	})
	return manifests, nil
}

// Resources reads every resource manifest, keyed by declared name. A duplicate
// name is an error rather than a last-writer-wins overwrite: two manifests
// claiming one name means every scenario depending on it resolves against
// whichever one the filesystem happened to hand over last.
func (r *Reader) Resources() (map[string]ResourceInput, error) {
	paths, err := filepath.Glob(filepath.Join(r.root, "resources", "*", "resource.json"))
	if err != nil {
		return nil, err
	}
	result := make(map[string]ResourceInput, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read resource manifest %s: %w", path, err)
		}
		var item ResourceInput
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, fmt.Errorf("decode resource manifest %s: %w", path, err)
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = filepath.Base(filepath.Dir(path))
			item.Name = name
		}
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("resource manifests declare duplicate name %q", name)
		}
		result[name] = item
	}
	return result, nil
}

// Scenarios reads every scenario service manifest's dependency closure.
func (r *Reader) Scenarios() ([]ScenarioInput, error) {
	paths, err := filepath.Glob(filepath.Join(r.root, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	result := make([]ScenarioInput, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read scenario manifest %s: %w", path, err)
		}
		var raw struct {
			Service struct {
				Name         string   `json:"name"`
				Capabilities []string `json:"capabilities"`
			} `json:"service"`
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
			Deployment struct {
				Dependencies struct {
					Resources map[string]json.RawMessage `json:"resources"`
				} `json:"dependencies"`
			} `json:"deployment"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode scenario manifest %s: %w", path, err)
		}
		resources := map[string]json.RawMessage{}
		swaps := make([]deployability.SwapSource, 0)
		for name, dep := range raw.Dependencies.Resources {
			resources[name] = dep
			swaps = append(swaps, deployability.SwapSource{Original: name, Alternatives: deployability.ExtractDeclaredAlternatives(dep)})
		}
		for name, dep := range raw.Deployment.Dependencies.Resources {
			resources[name] = dep
			swaps = append(swaps, deployability.SwapSource{Original: name, Alternatives: deployability.ExtractDeclaredAlternatives(dep)})
		}
		name := raw.Service.Name
		if name == "" {
			name = filepath.Base(filepath.Dir(filepath.Dir(path)))
		}
		result = append(result, ScenarioInput{Name: name, Capabilities: raw.Service.Capabilities, Resources: resources, Swaps: swaps})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// IsUnresolvedRoot reports whether an error is the missing-manifest-root
// failure, so a caller can distinguish "the tree is wrong" from "the tree is
// right and one manifest is malformed".
func IsUnresolvedRoot(err error) bool {
	var target UnresolvedRootError
	return errors.As(err, &target)
}

func normalizeHostOS(value string) (deployability.HostOS, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "linux":
		return deployability.HostOSLinux, true
	case "macos", "darwin":
		return deployability.HostOSMacOS, true
	case "windows", "win32":
		return deployability.HostOSWindows, true
	default:
		return "", false
	}
}
