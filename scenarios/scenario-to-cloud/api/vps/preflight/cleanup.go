package preflight

import (
	"context"
	"fmt"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// DiskCleanupResult records bounded target-owner repairs attempted for disk
// pressure. It contains no paths or shell commands.
type DiskCleanupResult struct {
	Attempted []string
	Succeeded []string
	Errors    []string
}

// CleanupDiskPressure runs the safe storage repairs appropriate for a failed
// disk-free check. Active Docker images remain protected by the target owner;
// release GC is separate so its active/previous/recovery protections apply.
func CleanupDiskPressure(ctx context.Context, rr reach.Reach, target identity.TargetRef, availableKB, requiredKB int64) DiskCleanupResult {
	result := DiskCleanupResult{}
	if availableKB >= requiredKB {
		return result
	}
	actions := []struct {
		name    string
		subject any
	}{
		{name: "journald.vacuum", subject: map[string]any{"journal": map[string]any{"max_use_bytes": int64(16 * 1024 * 1024)}}},
		{name: "docker.prune.unused-images", subject: map[string]any{"docker": map[string]any{}}},
	}
	for _, action := range actions {
		result.Attempted = append(result.Attempted, action.name)
		res, err := hostRepair(ctx, rr, target, action.name, action.subject)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", action.name, err))
			continue
		}
		if failure := verbFailure(res, nil); failure != "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", action.name, failure))
			continue
		}
		result.Succeeded = append(result.Succeeded, action.name)
	}
	return result
}
