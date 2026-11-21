// Responsibility: resolve declared spawn preferences against runner capabilities.
package orchestration

import (
	"fmt"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

type SpawnResolution struct {
	ExecutionMode   string
	SandboxMode     string
	NativeObjective bool
	Skipped         []SpawnPreferenceSkip
	// Fallback names the declared capability chosen when no preferred
	// combination was feasible (empty on a normal preference match). It is
	// recorded on the run as spawn_fallback so a silent fallback is visible.
	Fallback string
}

type SpawnPreferenceSkip struct{ ExecutionMode, SandboxMode, Reason string }

// SpawnPolicyConstraints carries caller-selected axes that a profile's
// declaration-only preferences may rank around but never replace.
type SpawnPolicyConstraints struct {
	ExecutionMode string
	SandboxMode   string
	RunMode       *domain.RunMode
	ForceInPlace  bool
}

// ResolveSpawnPolicy intersects declared profile preferences with facts
// published by the selected runner. It ranks combinations lexicographically
// by the profile's axisOrder, so no substrate or sandbox is chosen by a
// runner-specific branch.
func ResolveSpawnPolicy(policy *domain.SpawnPolicy, capabilities runner.Capabilities) (SpawnResolution, error) {
	return ResolveSpawnPolicyWithConstraints(policy, capabilities, SpawnPolicyConstraints{})
}

// ResolveSpawnPolicyWithConstraints ranks only combinations that satisfy the
// caller's explicit execution, sandbox, and run-mode constraints. Profile
// preferences rank the remaining feasible combinations; they cannot silently
// weaken an explicit caller request.
func ResolveSpawnPolicyWithConstraints(policy *domain.SpawnPolicy, capabilities runner.Capabilities, constraints SpawnPolicyConstraints) (SpawnResolution, error) {
	if policy == nil || len(policy.AxisOrder) == 0 {
		return SpawnResolution{}, fmt.Errorf("spawn policy is required")
	}
	if err := validateSpawnPolicyConstraints(constraints); err != nil {
		return SpawnResolution{}, err
	}
	allowed := make(map[string]bool)
	for _, required := range policy.Require {
		allowed[required.ExecutionMode+"\x00"+required.SandboxMode] = true
	}
	if len(allowed) > 0 {
		for key := range allowed {
			parts := strings.SplitN(key, "\x00", 2)
			if !capabilityExists(capabilities, parts[0], parts[1]) {
				return SpawnResolution{}, fmt.Errorf("required spawn combination unavailable: executionMode=%s sandboxMode=%s", parts[0], parts[1])
			}
		}
	}
	type candidate struct {
		runner.SpawnCapability
		score []int
	}
	candidates := make([]candidate, 0)
	for _, capability := range capabilities.SpawnCapabilities {
		for _, sandbox := range capability.SandboxModes {
			if len(allowed) > 0 && !allowed[capability.ExecutionMode+"\x00"+sandbox] {
				continue
			}
			if !matchesSpawnPolicyConstraints(capability.ExecutionMode, sandbox, constraints) {
				continue
			}
			score := make([]int, len(policy.AxisOrder))
			valid := true
			for i, axis := range policy.AxisOrder {
				var values []string
				var value string
				switch axis {
				case "executionMode":
					if constraints.ExecutionMode != "" {
						continue
					}
					values, value = policy.ExecutionMode.Prefer, capability.ExecutionMode
				case "sandboxMode":
					if constraints.SandboxMode != "" {
						continue
					}
					values, value = policy.SandboxMode.Prefer, sandbox
				default:
					valid = false
				}
				if !valid {
					break
				}
				index := preferenceIndex(values, value)
				if index < 0 {
					valid = false
					break
				}
				score[i] = index
			}
			if valid {
				candidates = append(candidates, candidate{SpawnCapability: runner.SpawnCapability{ExecutionMode: capability.ExecutionMode, SandboxModes: []string{sandbox}, NativeObjective: capability.NativeObjective}, score: score})
			}
		}
	}
	if len(candidates) == 0 {
		if len(allowed) > 0 {
			return SpawnResolution{}, fmt.Errorf("spawn policy has no feasible executionMode/sandboxMode combination")
		}
		if len(capabilities.SpawnCapabilities) == 0 {
			if hasSpawnPolicyConstraints(constraints) {
				return SpawnResolution{}, fmt.Errorf("runner declared no spawn capability satisfying explicit caller constraints")
			}
			// A runner with no declarations retains the launcher's historical
			// codec-pipe default only when the caller supplied no explicit selector.
			return SpawnResolution{ExecutionMode: "codec_pipe", SandboxMode: fallbackSandboxMode(policy)}, nil
		}
		if !hasSpawnPolicyConstraints(constraints) && len(capabilities.SpawnCapabilities[0].SandboxModes) == 0 {
			return SpawnResolution{}, fmt.Errorf("runner declared spawn capability without a sandbox mode: executionMode=%s", capabilities.SpawnCapabilities[0].ExecutionMode)
		}
		var fallback *candidate
		for _, capability := range capabilities.SpawnCapabilities {
			for _, sandbox := range capability.SandboxModes {
				if len(allowed) > 0 && !allowed[capability.ExecutionMode+"\x00"+sandbox] {
					continue
				}
				if !matchesSpawnPolicyConstraints(capability.ExecutionMode, sandbox, constraints) {
					continue
				}
				item := candidate{SpawnCapability: runner.SpawnCapability{ExecutionMode: capability.ExecutionMode, SandboxModes: []string{sandbox}, NativeObjective: capability.NativeObjective}}
				fallback = &item
				break
			}
			if fallback != nil {
				break
			}
		}
		if fallback == nil {
			return SpawnResolution{}, fmt.Errorf("runner declared no spawn capability satisfying explicit caller constraints")
		}
		// No preferred combination was feasible. This is allowed only because
		// require is empty and the runner declared a compatible capability; record
		// it as a visible fallback rather than choosing silently.
		resolution := SpawnResolution{
			ExecutionMode:   fallback.ExecutionMode,
			SandboxMode:     fallback.SandboxModes[0],
			NativeObjective: fallback.NativeObjective,
			Fallback:        fallback.ExecutionMode + "/" + fallback.SandboxModes[0],
		}
		for _, capability := range capabilities.SpawnCapabilities {
			for _, sandbox := range capability.SandboxModes {
				if len(allowed) > 0 && !allowed[capability.ExecutionMode+"\x00"+sandbox] {
					continue
				}
				if !matchesSpawnPolicyConstraints(capability.ExecutionMode, sandbox, constraints) || (capability.ExecutionMode == resolution.ExecutionMode && sandbox == resolution.SandboxMode) {
					continue
				}
				resolution.Skipped = append(resolution.Skipped, SpawnPreferenceSkip{
					ExecutionMode: capability.ExecutionMode,
					SandboxMode:   sandbox,
					Reason:        "no preferred spawn combination was feasible; declared compatible capability used as fallback",
				})
			}
		}
		return resolution, nil
	}
	best := candidates[0]
	for _, item := range candidates[1:] {
		if lessScore(item.score, best.score) {
			best = item
		}
	}
	resolution := SpawnResolution{ExecutionMode: best.ExecutionMode, SandboxMode: best.SandboxModes[0], NativeObjective: best.NativeObjective}
	for _, item := range candidates {
		if item.ExecutionMode == resolution.ExecutionMode && item.SandboxModes[0] == resolution.SandboxMode {
			continue
		}
		resolution.Skipped = append(resolution.Skipped, SpawnPreferenceSkip{ExecutionMode: item.ExecutionMode, SandboxMode: item.SandboxModes[0], Reason: "declared combination ranked below the selected feasible preference"})
	}
	return resolution, nil
}

func validateSpawnPolicyConstraints(constraints SpawnPolicyConstraints) error {
	if constraints.ForceInPlace && constraints.RunMode != nil && *constraints.RunMode != domain.RunModeInPlace {
		return fmt.Errorf("forceInPlace conflicts with explicit runMode=%s", *constraints.RunMode)
	}
	if constraints.RunMode == nil {
		return nil
	}
	switch *constraints.RunMode {
	case domain.RunModeSandboxed:
		if constraints.SandboxMode == string(domain.SandboxModeOff) {
			return fmt.Errorf("explicit sandbox mode off conflicts with runMode=sandboxed")
		}
	case domain.RunModeInPlace:
		if constraints.SandboxMode != "" && constraints.SandboxMode != string(domain.SandboxModeOff) {
			return fmt.Errorf("explicit sandbox mode %s conflicts with runMode=in_place", constraints.SandboxMode)
		}
	default:
		return fmt.Errorf("unsupported explicit runMode=%s", *constraints.RunMode)
	}
	return nil
}

func hasSpawnPolicyConstraints(constraints SpawnPolicyConstraints) bool {
	return constraints.ExecutionMode != "" || constraints.SandboxMode != "" || constraints.RunMode != nil || constraints.ForceInPlace
}

func matchesSpawnPolicyConstraints(executionMode, sandboxMode string, constraints SpawnPolicyConstraints) bool {
	if constraints.ExecutionMode != "" && executionMode != constraints.ExecutionMode {
		return false
	}
	if constraints.SandboxMode != "" && sandboxMode != constraints.SandboxMode {
		return false
	}
	if constraints.ForceInPlace && sandboxMode != string(domain.SandboxModeOff) {
		return false
	}
	if constraints.RunMode != nil {
		switch *constraints.RunMode {
		case domain.RunModeInPlace:
			if sandboxMode != string(domain.SandboxModeOff) {
				return false
			}
		case domain.RunModeSandboxed:
			if sandboxMode == string(domain.SandboxModeOff) {
				return false
			}
		}
	}
	return true
}

func fallbackSandboxMode(policy *domain.SpawnPolicy) string {
	if policy != nil {
		for _, preferred := range policy.SandboxMode.Prefer {
			if strings.TrimSpace(preferred) != "" {
				return strings.TrimSpace(preferred)
			}
		}
	}
	return string(domain.SandboxModeOff)
}

func preferenceIndex(values []string, value string) int {
	for i, item := range values {
		if item == value {
			return i
		}
	}
	return -1
}

func lessScore(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func capabilityExists(c runner.Capabilities, executionMode, sandboxMode string) bool {
	for _, item := range c.SpawnCapabilities {
		if item.ExecutionMode == executionMode {
			for _, mode := range item.SandboxModes {
				if mode == sandboxMode {
					return true
				}
			}
		}
	}
	return false
}
