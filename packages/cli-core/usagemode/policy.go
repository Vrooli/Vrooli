package usagemode

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// A usage mode stores intent (kind, scope, expiry, extra exempt roles, an
// optional override model, effort, context limit). The models it lowers and
// what they lower to are read from the runner's resource model-policy.json
// every time a mode is applied, so a policy edit reaches active modes at once.
// This file finds and caches that policy. Every failure means "no live policy"
// and the caller keeps the values the mode stored.

// resourceFor is the resource directory that owns a runner's model policy.
func resourceFor(runner string) string {
	if runner = NormalizeRunner(runner); runner == "claude" {
		return "claude-code"
	}
	return runner
}

// searchFromWorkingDir lets the lookup walk up from the working directory, which
// is how a scenario or CLI started inside the repository finds it. Tests turn it
// off so the developer's own checkout never leaks into a fixture.
var searchFromWorkingDir = true

var (
	policyRootsMu sync.Mutex
	policyRoots   []func() string
)

// AddPolicyRoot registers a source of the repository root for callers that
// know it better than the environment does (the launcher's repo contract).
func AddPolicyRoot(root func() string) {
	policyRootsMu.Lock()
	defer policyRootsMu.Unlock()
	policyRoots = append(policyRoots, root)
}

// policyCandidates yields where a resource's policy may live, cheapest first:
// registered roots (which can be slow to discover) are asked only when the
// environment's roots did not have it.
func policyCandidates(resource string) iter.Seq[string] {
	return func(yield func(string) bool) {
		try := func(root string) bool {
			return root == "" || (yield(filepath.Join(root, "resources", resource, "model-policy.json")) && yield(filepath.Join(root, resource, "model-policy.json")))
		}
		for _, key := range []string{"VROOLI_RESOURCE_ROOT", "VROOLI_ROOT", "PROJECT_ROOT", "VROOLI_SOURCE_ROOT"} {
			if !try(strings.TrimSpace(os.Getenv(key))) {
				return
			}
		}
		policyRootsMu.Lock()
		registered := slices.Clone(policyRoots)
		policyRootsMu.Unlock()
		for _, root := range registered {
			if !try(root()) {
				return
			}
		}
		if wd, err := os.Getwd(); err == nil && searchFromWorkingDir {
			for dir, i := wd, 0; i < 8; dir, i = filepath.Dir(dir), i+1 {
				if !try(dir) {
					return
				}
			}
		}
	}
}

// PolicyPath is the runner's resource model-policy.json, if one can be found.
func PolicyPath(runner string) (string, bool) {
	for path := range policyCandidates(resourceFor(runner)) {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, true
		}
	}
	return "", false
}

// ReadPolicy returns the runner's resource model-policy.json.
func ReadPolicy(runner string) ([]byte, error) {
	path, ok := PolicyPath(runner)
	if !ok {
		return nil, os.ErrNotExist
	}
	return readRegular(path, MaxFileBytes)
}

type policyStamp struct {
	path  string
	mod   time.Time
	size  int64
	tiers Tiers
}

var (
	policyCacheMu sync.Mutex
	policyCache   = map[string]policyStamp{}
)

// policyTiers parses the runner's usage tiers, re-reading only when the file
// changed, so applying a mode costs a stat rather than a parse. Any failure
// reads as "no live policy".
func policyTiers(runner string) (Tiers, bool) {
	tiers, err := loadPolicyTiers(runner)
	return tiers, err == nil
}

// PolicyProblem says why a runner's model policy cannot be used, or "" when it
// can. A launch that applies a mode proceeds either way; the launcher prints
// this one line so the operator learns of a broken policy without a failure.
func PolicyProblem(runner string) string {
	if _, err := loadPolicyTiers(runner); err != nil {
		return err.Error()
	}
	return ""
}

func loadPolicyTiers(runner string) (Tiers, error) {
	path, ok := PolicyPath(runner)
	if !ok {
		return Tiers{}, fmt.Errorf("no %s model policy found", resourceFor(runner))
	}
	info, err := os.Stat(path)
	if err != nil {
		return Tiers{}, err
	}
	resource := resourceFor(runner)
	policyCacheMu.Lock()
	hit, cached := policyCache[resource]
	policyCacheMu.Unlock()
	if cached && hit.path == path && hit.mod.Equal(info.ModTime()) && hit.size == info.Size() {
		return hit.tiers, nil
	}
	raw, err := readRegular(path, MaxFileBytes)
	if err != nil {
		return Tiers{}, fmt.Errorf("model policy %s unreadable: %w", path, err)
	}
	tiers, err := LoadTiers(raw)
	if err != nil {
		return Tiers{}, fmt.Errorf("model policy %s unusable: %w", path, err)
	}
	policyCacheMu.Lock()
	policyCache[resource] = policyStamp{path: path, mod: info.ModTime(), size: info.Size(), tiers: tiers}
	policyCacheMu.Unlock()
	return tiers, nil
}

// ModelExcluded reports whether the runner's resource policy lists model in
// excluded_models, by name or through its alias. err is non-nil when that
// cannot be decided (no policy, unreadable, invalid); callers that enforce must
// treat it as "not excluded".
func ModelExcluded(runner, model string) (bool, error) {
	tiers, err := loadPolicyTiers(runner)
	if err != nil {
		return false, err
	}
	model = strings.TrimSpace(model)
	return model != "" && (tiers.excluded[model] || tiers.excluded[tiers.aliases[model]]), nil
}
