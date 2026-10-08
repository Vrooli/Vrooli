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
