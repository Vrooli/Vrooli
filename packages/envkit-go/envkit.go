// Package envkit provides the repository-wide environment boundary primitive.
//
// An environment is an exec boundary, not an ownership boundary. Relative
// variables therefore cannot be inherited into a process that belongs to a
// different scenario or resource. This package keeps that policy in one place
// and folds names according to the target platform.
package envkit

import (
	"runtime"
	"sort"
	"strings"
)

// Env is an os.Environ-shaped environment: one KEY=value entry per variable.
type Env []string

// Relationship describes the ownership boundary of a child process.
type Relationship uint8

const (
	SameScenario Relationship = iota
	ForeignScenario
	Resource
	DelegatedAgent
)

// Platform contains the platform-specific environment rules. Tests can use
// Platform{CaseInsensitive: true, Floor: ...} on any host to exercise the
// Windows branch without requiring a Windows runner.
type Platform struct {
	CaseInsensitive bool
	Floor           Env
}

// DefaultPlatform is the host platform policy used by ForChild.
func DefaultPlatform() Platform {
	p := Platform{CaseInsensitive: runtime.GOOS == "windows"}
	if p.CaseInsensitive {
		p.Floor = Env{"SYSTEMROOT=", "COMSPEC=", "PATHEXT=", "TEMP="}
	}
	return p
}

// ForChild derives a child environment with no inherited relative identity.
// SameScenario retains the parent's relative variables. All other boundaries
// drop them; callers may add the child's own identity through WithOverlay.
func ForChild(parent Env, relationship Relationship) Env {
	return ForChildWithPlatform(parent, relationship, DefaultPlatform())
}

// WithOverlay derives a child environment and applies explicit child-owned
// values after the relationship filter. It is the only supported way to add a
// replacement identity or other boundary-specific values.
func WithOverlay(parent Env, relationship Relationship, overlay Env) Env {
	return WithOverlayWithPlatform(parent, relationship, overlay, DefaultPlatform())
}

// ForChildWithPlatform is ForChild with an injected platform policy.
func ForChildWithPlatform(parent Env, relationship Relationship, platform Platform) Env {
	return WithOverlayWithPlatform(parent, relationship, nil, platform)
}

// WithOverlayWithPlatform is the testable implementation of boundary
// derivation. Overlay entries win regardless of their position in the parent.
func WithOverlayWithPlatform(parent Env, relationship Relationship, overlay Env, platform Platform) Env {
	values := make(map[string]string, len(parent)+len(overlay)+len(platform.Floor))
	for _, entry := range parent {
		key, value, ok := split(entry)
		if !ok || !keepInherited(key, relationship, platform) {
			continue
		}
		values[fold(key, platform)] = key + "=" + value
	}
	for _, entry := range overlay {
		key, value, ok := split(entry)
		if !ok {
			continue
		}
		values[fold(key, platform)] = key + "=" + value
	}
	for _, entry := range platform.Floor {
		key, value, ok := split(entry)
		if !ok {
			continue
		}
		if _, exists := values[fold(key, platform)]; !exists {
			values[fold(key, platform)] = key + "=" + value
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(Env, 0, len(keys))
	for _, key := range keys {
		out = append(out, values[key])
	}
	return out
}

func keepInherited(key string, relationship Relationship, platform Platform) bool {
	normalized := key
	if platform.CaseInsensitive {
		normalized = strings.ToUpper(normalized)
	}
	if isRelative(normalized) {
		return relationship == SameScenario
	}
	if relationship == DelegatedAgent && normalized == "VROOLI_AGENT_IDENTITY_TOKEN" {
		return true
	}
	// Inbound observations are never emitted as delegated identity. They may
	// remain ambient for a same-process child, but are dropped at boundaries.
	if relationship != SameScenario && (normalized == "CLAUDE_CODE_SESSION_ID" || normalized == "CODEX_THREAD_ID") {
		return false
	}
	return true
}

func isRelative(key string) bool {
	return key == "API_PORT" || key == "UI_PORT" || strings.HasPrefix(key, "SCENARIO_") || strings.HasPrefix(key, "VROOLI_SCENARIO")
}

func split(entry string) (string, string, bool) {
	parts := strings.SplitN(entry, "=", 2)
	return parts[0], func() string {
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}(), len(parts) == 2 && parts[0] != ""
}

func fold(key string, platform Platform) string {
	if platform.CaseInsensitive {
		return strings.ToUpper(key)
	}
	return key
}
