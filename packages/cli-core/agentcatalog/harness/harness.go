// Package harness reads the typed harness block each coding-agent resource
// declares in resources/<name>/resource.json. It imports only the standard
// library so cliutil can use it: agentcatalog itself imports cliutil.
package harness

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
)

// Context limit mechanisms, home isolation kinds and home layouts.
const (
	ContextLimitEnv = "env"
	ContextLimitArg = "arg"

	HomeIsolationNone = "none"
	HomeIsolationEnv  = "env"

	// HomeLayoutAllowlist links a fixed list of shared entries; HomeLayoutMirror
	// links everything in the shared home except session and lock state.
	HomeLayoutAllowlist = "allowlist"
	HomeLayoutMirror    = "mirror"
)

// UsageWindow is a window the provider is expected to report. It reserves card
// space and never proves that the provider currently reports it.
type UsageWindow struct {
	ID      string `json:"id"`
	Minutes int64  `json:"minutes"`
}

// ContextLimit says how a launch asks this harness to compact sooner.
type ContextLimit struct {
	Kind    string `json:"kind"`
	Key     string `json:"key"`
	Minimum int64  `json:"minimum"`
}

// HomeIsolation says how a Web Console pane gets its own harness home.
type HomeIsolation struct {
	Kind        string `json:"kind"`
	Env         string `json:"env,omitempty"`
	SharedDir   string `json:"shared_dir,omitempty"`
	StateDir    string `json:"state_dir,omitempty"`
	SessionsEnv string `json:"sessions_env,omitempty"`
	Layout      string `json:"layout,omitempty"`
}

// ModelSelection is the verified native launch argument for a model.
type ModelSelection struct {
	Flag string `json:"flag"`
	// Sources are the policy billing sources this argument can reach.
	Sources []string `json:"sources"`
	// PrefixSource means the argument takes source/model.
	PrefixSource bool `json:"prefix_source,omitempty"`
}

// UsageMode marks a harness whose launch usage modes can lower.
type UsageMode struct {
	// Provider is the quota provider a window-reset mode names.
	Provider string `json:"provider"`
}

// Features lists what this harness supports. A nil or false feature is absent,
// and a surface must not offer it.
type Features struct {
	ContextLimit        *ContextLimit   `json:"context_limit,omitempty"`
	ContextWindowSource string          `json:"context_window_source,omitempty"`
	UsageMode           *UsageMode      `json:"usage_mode,omitempty"`
	ModelSelection      *ModelSelection `json:"model_selection,omitempty"`
	ProviderSelection   bool            `json:"provider_selection,omitempty"`
	Resume              bool            `json:"resume,omitempty"`
	GoalMode            bool            `json:"goal_mode,omitempty"`
	MCP                 bool            `json:"mcp,omitempty"`
	Hooks               bool            `json:"hooks,omitempty"`
}

// Descriptor is one harness. Resource comes from the manifest's name, which is
// also the runner name; ID is the launcher agent ID.
type Descriptor struct {
	Resource      string        `json:"resource"`
	ID            string        `json:"id"`
	Aliases       []string      `json:"aliases,omitempty"`
	Binary        string        `json:"binary"`
	DisplayName   string        `json:"display_name"`
	Icon          string        `json:"icon"`
	SessionEnv    []string      `json:"session_env,omitempty"`
	HomeIsolation HomeIsolation `json:"home_isolation"`
	UsageWindows  []UsageWindow `json:"usage_windows,omitempty"`
	Features      Features      `json:"features"`
}

// Set is an immutable collection of descriptors sorted by ID.
type Set struct {
	descriptors []Descriptor
}

// All returns the descriptors sorted by ID.
func (s Set) All() []Descriptor {
	return append([]Descriptor(nil), s.descriptors...)
}

// ByID finds a harness by its launcher agent ID only.
func (s Set) ByID(id string) (Descriptor, bool) {
	for _, descriptor := range s.descriptors {
		if descriptor.ID == id {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

// Lookup finds a harness by any name it answers to: ID, resource, binary or
// alias, ignoring case and surrounding space.
func (s Set) Lookup(name string) (Descriptor, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, descriptor := range s.descriptors {
		if name == descriptor.ID || name == descriptor.Resource || name == descriptor.Binary {
			return descriptor, true
		}
		for _, alias := range descriptor.Aliases {
			if name == alias {
				return descriptor, true
			}
		}
	}
	return Descriptor{}, false
}

// names lists every name this harness answers to.
func (d Descriptor) names() []string {
	return append([]string{d.ID, d.Resource, d.Binary}, d.Aliases...)
}

func newSet(descriptors []Descriptor) Set {
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].ID < descriptors[j].ID })
	return Set{descriptors: descriptors}
}

// bareName keeps a descriptor's binary a PATH lookup. The launcher treats a
// value with a path separator as an explicit executable. The other patterns
// keep what a launch composes into a command line free of shell syntax.
var (
	bareName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	envName  = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	argKey   = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)
	flagName = regexp.MustCompile(`^--?[a-z][a-z0-9-]*$`)
)

func validate(d Descriptor) error {
	var errs []error
	for field, value := range map[string]string{"resource": d.Resource, "id": d.ID, "binary": d.Binary} {
		if !bareName.MatchString(value) {
			errs = append(errs, fmt.Errorf("%s %q must be a bare lowercase name", field, value))
		}
	}
	for _, alias := range d.Aliases {
		if !bareName.MatchString(alias) {
			errs = append(errs, fmt.Errorf("alias %q must be a bare lowercase name", alias))
		}
	}
	if strings.TrimSpace(d.DisplayName) == "" || strings.TrimSpace(d.Icon) == "" {
		errs = append(errs, errors.New("display_name and icon are required"))
	}
	for _, window := range d.UsageWindows {
		if window.ID == "" || window.Minutes <= 0 {
			errs = append(errs, fmt.Errorf("usage window %q requires an id and positive minutes", window.ID))
		}
	}
	if limit := d.Features.ContextLimit; limit != nil {
		validKey := (limit.Kind == ContextLimitEnv && envName.MatchString(limit.Key)) || (limit.Kind == ContextLimitArg && argKey.MatchString(limit.Key))
		if !validKey || limit.Minimum < 0 {
			errs = append(errs, fmt.Errorf("context_limit %+v requires kind env or arg, a key valid for that kind and a non-negative minimum", *limit))
		}
	}
	switch d.Features.ContextWindowSource {
	case "", WindowSourceClaudeStatusline, WindowSourceCodexModelsCache:
	default:
		errs = append(errs, fmt.Errorf("context_window_source %q has no reader", d.Features.ContextWindowSource))
	}
	if selection := d.Features.ModelSelection; selection != nil && (!flagName.MatchString(selection.Flag) || len(selection.Sources) == 0) {
		errs = append(errs, fmt.Errorf("model_selection flag %q must be a plain option and needs sources", selection.Flag))
	}
	for _, name := range append([]string{d.HomeIsolation.Env, d.HomeIsolation.SessionsEnv}, d.SessionEnv...) {
		if name != "" && !envName.MatchString(name) {
			errs = append(errs, fmt.Errorf("environment variable %q is not a valid name", name))
		}
	}
	if d.Features.UsageMode != nil && d.Features.UsageMode.Provider == "" {
		errs = append(errs, errors.New("usage_mode requires a provider"))
	}
	switch home := d.HomeIsolation; home.Kind {
	case HomeIsolationNone:
	case HomeIsolationEnv:
		if home.Env == "" || !bareName.MatchString(home.StateDir) || home.SharedDir == "" || strings.ContainsAny(home.SharedDir, `/\`) || (home.Layout != HomeLayoutAllowlist && home.Layout != HomeLayoutMirror) {
			errs = append(errs, fmt.Errorf("home_isolation %+v requires env, state_dir, shared_dir and layout", home))
		}
	default:
		errs = append(errs, fmt.Errorf("home_isolation kind %q must be none or env", home.Kind))
	}
	return errors.Join(errs...)
}

// parseManifest returns the harness block of one resource.json, if any.
func parseManifest(data []byte) (Descriptor, bool, error) {
	var manifest struct {
		Name      string          `json:"name"`
		Harness   json.RawMessage `json:"harness"`
		Execution struct {
			RunnerType string `json:"runner_type"`
		} `json:"execution"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Descriptor{}, false, err
	}
	if len(manifest.Harness) == 0 {
		return Descriptor{}, false, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(manifest.Harness))
	decoder.DisallowUnknownFields()
	var descriptor Descriptor
	if err := decoder.Decode(&descriptor); err != nil {
		return Descriptor{}, false, err
	}
	descriptor.Resource = manifest.Name
	if err := validate(descriptor); err != nil {
		return Descriptor{}, false, err
	}
	if manifest.Execution.RunnerType != "" && manifest.Execution.RunnerType != manifest.Name {
		return Descriptor{}, false, fmt.Errorf("execution.runner_type %q must equal the resource name %q", manifest.Execution.RunnerType, manifest.Name)
	}
	return descriptor, true, nil
}

func scan(root string) ([]Descriptor, []error) {
	paths, err := filepath.Glob(filepath.Join(root, "resources", "*", "resource.json"))
	if err != nil {
		return nil, []error{err}
	}
	var descriptors []Descriptor
	var errs []error
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		descriptor, ok, err := parseManifest(data)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		if ok {
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors, errs
}

// Load strictly reads every harness declared under root/resources.
func Load(root string) (Set, error) {
	descriptors, errs := scan(root)
	seen := map[string]string{}
	for _, descriptor := range descriptors {
		for _, name := range descriptor.names() {
			if owner, taken := seen[name]; taken && owner != descriptor.Resource {
				errs = append(errs, fmt.Errorf("%s and %s both answer to %q", owner, descriptor.Resource, name))
			}
			seen[name] = descriptor.Resource
		}
	}
	if err := errors.Join(errs...); err != nil {
		return Set{}, err
	}
	return newSet(descriptors), nil
}

//go:generate env VROOLI_UPDATE_HARNESS_SNAPSHOT=1 go test . -run TestBuiltinSnapshotMatchesResources

//go:embed builtin.json
var builtinJSON []byte

var builtinOnce = sync.OnceValue(func() Set {
	var descriptors []Descriptor
	if err := json.Unmarshal(builtinJSON, &descriptors); err != nil {
		panic("harness: embedded builtin.json is invalid: " + err.Error())
	}
	return newSet(descriptors)
})

// Builtin is the compiled-in snapshot of the descriptors, for binaries that run
// without the repository: PATH shims, desktop bundles and remote agents.
func Builtin() Set { return builtinOnce() }

// repoMarker identifies a Vrooli checkout.
const repoMarker = ".vrooli/repo-contract.json"

var activeByRoot sync.Map

// Active is the snapshot overlaid with what the repository named by
// VROOLI_RESOURCE_ROOT or VROOLI_ROOT declares, so a new resource directory takes
// effect without a rebuild. A root counts only if it holds the repository marker.
// This stops a stray variable, not a hostile one: a process that controls the
// environment also controls PATH. Active never searches from the working
// directory: a PATH shim runs in arbitrary project directories, and a resources/
// folder there must not choose what the shim executes.
//
// A malformed or colliding manifest keeps the snapshot's value, because reading
// this must never stop a harness starting. The manifests are read once per root,
// so a long-lived server sees a new or edited one after it restarts.
func Active() Set {
	for _, key := range []string{"VROOLI_RESOURCE_ROOT", "VROOLI_ROOT"} {
		root := strings.TrimSpace(os.Getenv(key))
		if root == "" {
			continue
		}
		if cached, ok := activeByRoot.Load(root); ok {
			return cached.(Set)
		}
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(repoMarker))); err != nil || info.IsDir() {
			continue
		}
		descriptors, _ := scan(root)
		set := overlay(Builtin(), descriptors)
		activeByRoot.Store(root, set)
		return set
	}
	return Builtin()
}

// overlay applies descriptors in resource order. One that answers to a name
// another harness already holds is dropped, so a lookup has one answer and the
// result does not depend on the order of the input.
func overlay(base Set, descriptors []Descriptor) Set {
	merged := make(map[string]Descriptor, len(base.descriptors)+len(descriptors))
	for _, descriptor := range base.descriptors {
		merged[descriptor.Resource] = descriptor
	}
	ordered := append([]Descriptor(nil), descriptors...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Resource < ordered[j].Resource })
	for _, descriptor := range ordered {
		if !collides(merged, descriptor) {
			merged[descriptor.Resource] = descriptor
		}
	}
	result := make([]Descriptor, 0, len(merged))
	for _, descriptor := range merged {
		result = append(result, descriptor)
	}
	return newSet(result)
}

func collides(merged map[string]Descriptor, candidate Descriptor) bool {
	for _, existing := range merged {
		if existing.Resource == candidate.Resource {
			continue
		}
		for _, name := range candidate.names() {
			if slices.Contains(existing.names(), name) {
				return true
			}
		}
	}
	return false
}
