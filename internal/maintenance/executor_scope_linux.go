//go:build linux

package maintenance

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/vrooli/vrooli/internal/process"
)

func readScopeIdentity(entry processTableEntry) (scopeIdentity, error) {
	var identity scopeIdentity
	handle, err := process.OpenHandle(entry.PID)
	if isMissingProcessError(err) {
		return identity, nil
	}
	if err != nil {
		return identity, err
	}
	defer handle.Close()
	info, err := os.Stat("/proc/" + strconv.Itoa(entry.PID))
	if os.IsNotExist(err) {
		return identity, nil
	}
	if err != nil {
		return identity, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return identity, errors.New("process uid unavailable")
	}
	// Runs are local-user executions. Foreign users cannot inherit this user's
	// managed run labels. PID/PGID positives still block regardless of UID.
	if stat.Uid != uint32(os.Getuid()) {
		return identity, nil
	}
	env, readErr := process.ReadEnvironment(entry.PID)
	alive, err := handle.Alive()
	if err != nil {
		return identity, err
	}
	if !alive {
		return identity, nil
	}
	if readErr != nil {
		if errors.Is(readErr, os.ErrPermission) && outsideManagedExecutionScope(entry) {
			// Native service ownership is positive scope evidence. In
			// particular a non-dumpable desktop credential service must not
			// make every historical managed run permanently unknown. This
			// is never a name/argv exemption and never covers Vrooli slices,
			// inherited terminal scopes, or an unreadable cgroup identity.
			return identity, nil
		}
		return identity, readErr
	}
	identity.RunID = env["VROOLI_RUN_ID"]
	identity.Scenario = env[runtimeScenarioEnv]
	identity.Variant = env[runtimeVariantEnv]
	identity.InstanceID = env[runtimeInstanceEnv]
	identity.HomePaths = executorScopeHomePathsFromEnvironment(env)
	identity.TagLabels = map[string]string{}
	for key, value := range env {
		if strings.HasSuffix(key, "AGENT_TAG") && value != "" {
			identity.Tags = append(identity.Tags, value)
			identity.TagLabels[key] = value
		}
	}
	return identity, nil
}

func outsideManagedExecutionScope(entry processTableEntry) bool {
	path := entry.Cgroup
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "vrooli") {
		return false
	}
	leaf := filepath.Base(path)
	// The user manager's native init.scope is the control-plane parent, not
	// an execution scope. Its unreadable credentials are not run identity.
	if leaf == "init.scope" && strings.HasPrefix(filepath.Base(filepath.Dir(path)), "user@") {
		return true
	}
	return strings.HasSuffix(leaf, ".service") && !strings.HasPrefix(leaf, "user@")
}

// executorHomeIdentityInScope is intentionally conservative. A denied read
// can be ignored only when the process table positively places it in a native
// service/init scope, which is outside Agent Manager's run-scoped harness
// homes. Missing cgroup identity stays in scope and cannot prove absence.
func executorHomeIdentityInScope(entry processTableEntry) bool {
	if isDockerWorkloadScope(entry.Cgroup) {
		return false
	}
	return !outsideManagedExecutionScope(entry)
}

// isDockerWorkloadScope accepts only the kernel's exact systemd cgroup form
// for a Docker workload. These are non-native processes and are outside the
// requested native Codex-home inventory. A missing, malformed, or nested
// cgroup is not enough to exclude a process from that inventory.
func isDockerWorkloadScope(cgroup string) bool {
	const prefix = "/workload.slice/docker-"
	if !strings.HasPrefix(cgroup, prefix) || !strings.HasSuffix(cgroup, ".scope") {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(cgroup, prefix), ".scope")
	if len(id) != 64 {
		return false
	}
	for _, r := range id {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
