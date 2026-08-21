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
}

type SpawnPreferenceSkip struct{ ExecutionMode, SandboxMode, Reason string }

// ResolveSpawnPolicy intersects declared profile preferences with facts
// published by the selected runner. It ranks combinations lexicographically
// by the profile's axisOrder, so no substrate or sandbox is chosen by a
// runner-specific branch.
func ResolveSpawnPolicy(policy *domain.SpawnPolicy, capabilities runner.Capabilities) (SpawnResolution, error) {
	if policy == nil || len(policy.AxisOrder) == 0 {
		return SpawnResolution{}, fmt.Errorf("spawn policy is required")
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
			score := make([]int, len(policy.AxisOrder))
			valid := true
			for i, axis := range policy.AxisOrder {
				var values []string
				var value string
				switch axis {
				case "executionMode":
					values, value = policy.ExecutionMode.Prefer, capability.ExecutionMode
				case "sandboxMode":
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
		return SpawnResolution{}, fmt.Errorf("spawn policy has no feasible executionMode/sandboxMode combination")
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
