// Resource daemons must not inherit the cgroup of whoever started them: a
// resource restarted from a coding-agent session would otherwise live in that
// agent's scope and die with it (observed 2026-09-21: kokoro and whisper
// riding in a dead opencode session scope). Adoption into the services slice
// gives each resource process a scope of its own, sibling to scenario
// services, so its lifetime is tied to the operator's host, not the caller.
package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

// resourceServiceScopeName returns the scope unit stem for one resource
// process. Placement and ownership inspection share this identity: the same
// resource/component tuple always maps to the same unit name. A digest
// preserves identity across sanitization and truncation, mirroring
// scenarioruntime.ServiceScopeName.
func resourceServiceScopeName(resource, component string) string {
	clean := func(value string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(value) {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				b.WriteRune(r)
			default:
				b.WriteRune('-')
			}
		}
		return strings.Trim(b.String(), "-")
	}
	name := "vrooli-service-resource-" + clean(resource) + "-" + clean(component)
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s%d:%s", len(resource), resource, len(component), component)))
	suffix := fmt.Sprintf("-%x", digest[:8])
	const limit = 200
	if len(name) > limit-len(suffix) {
		name = name[:limit-len(suffix)]
	}
	return strings.TrimRight(name, "-") + suffix
}

// placeResourceProcess adopts a resource-owned daemon into its own scope
// under the services slice. Failure is reported, not fatal: the process is
// already recorded and supervised, it just keeps running where it was born.
// Platforms without an adoption primitive are not an error.
func placeResourceProcess(resource, component string, pid int) error {
	_, _, err := platform.AdoptIntoScope(platform.AdoptSpec{
		PID:         pid,
		Scope:       resourceServiceScopeName(resource, component),
		Slice:       platform.ServicesSlice,
		Description: fmt.Sprintf("Vrooli resource %s (%s)", resource, component),
	})
	if err != nil && !errors.Is(err, platform.ErrUnsupported) {
		return err
	}
	return nil
}

// Recover only a unique process in the canonical companion scope next to the
// current native service. Missing PID files must not turn an idempotent start
// into another proxy or capacity reporter. Ports are not ownership evidence.
func existingResourceCompanion(resource string, c ResourceCompanion, bin string, parentPID int) (int, error) {
	if parentPID <= 1 {
		return 0, nil
	}
	parentScope, err := platform.ProcessScope(parentPID)
	if err != nil {
		return 0, fmt.Errorf("inspect companion parent scope: %w", err)
	}
	if parentScope == "" { // Platforms without persisted resource scopes.
		return 0, nil
	}
	if filepath.Base(parentScope) != resourceServiceScopeName(resource, "service")+".scope" || filepath.Base(filepath.Dir(parentScope)) != platform.ServicesSlice {
		return 0, fmt.Errorf("companion parent is outside the resource's service scope")
	}
	name := resourceServiceScopeName(resource, "companion-"+c.Name) + ".scope"
	scope := filepath.Join(filepath.Dir(parentScope), name)
	pids, err := companionScopeProcesses(platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: name, Path: scope})
	if err != nil {
		return 0, fmt.Errorf("inspect existing companion scope: %w", err)
	}
	pid, uniqueErr := uniqueCompanionScopePID(pids)
	if uniqueErr != nil {
		return 0, uniqueErr
	}
	if pid == 0 {
		return 0, nil
	}
	actualScope, scopeErr := platform.ProcessScope(pid)
	executable, exeErr := platform.ProcessExecutablePath(pid)
	command, commandErr := platform.ProcessCommandLine(pid)
	dir, dirErr := companionDir(resource)
	if scopeErr != nil || exeErr != nil || commandErr != nil || dirErr != nil {
		return 0, fmt.Errorf("existing companion scope process identity does not match the current service")
	}
	if err := validateExistingCompanionIdentity(resource, c, dir, parentPID, scope, actualScope, bin, executable, command, activityEdgeControlSocketReady); err != nil {
		return 0, err
	}
	return pid, nil
}

func uniqueCompanionScopePID(pids []int) (int, error) {
	if len(pids) > 1 {
		return 0, fmt.Errorf("companion scope has %d processes; refusing ambiguous recovery or duplicate launch", len(pids))
	}
	if len(pids) == 0 {
		return 0, nil
	}
	if pids[0] <= 1 {
		return 0, fmt.Errorf("companion scope contains an invalid process identity")
	}
	return pids[0], nil
}

func validateExistingCompanionIdentity(resource string, c ResourceCompanion, dir string, parentPID int, expectedScope, actualScope, bin, executable, command string, ready func(context.Context, string) error) error {
	args := companionLaunchArgs(resource, c, dir, parentPID)
	if !companionProcessIdentityMatches(expectedScope, actualScope, bin, args, executable, command) {
		return fmt.Errorf("existing companion scope process identity does not match the current service")
	}
	if hasStableActivityEdge(resource) && c.Name == "activity-edge" {
		if ready == nil {
			return fmt.Errorf("%s activity edge control socket readiness check is unavailable", resource)
		}
		readyCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := ready(readyCtx, filepath.Join(dir, "activity-edge.control.sock"))
		cancel()
		if err != nil {
			return fmt.Errorf("existing %s activity edge control socket is not ready: %w", resource, err)
		}
	}
	return nil
}

// A stopped companion's scope may already have been removed by the service
// manager. Keep absence distinct from an unreadable ownership inventory.
func companionScopeProcesses(ref platform.ScopeRef) ([]int, error) {
	pids, err := platform.ScopeProcesses(ref)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return pids, err
}

func companionCommandMatches(bin string, args []string, parentPID int, executable, command string) bool {
	expected := append(append([]string{}, args...), "--parent-pid", strconv.Itoa(parentPID))
	return companionProcessIdentityMatches("", "", bin, expected, executable, command)
}

// verifyCompanionPIDAbsentFromScope closes the gap between observing a dead
// PID and allowing the backend owner to be replaced. Cgroup inventory can
// retain a zombie/ghost entry after /proc no longer reports a running process;
// that state is not proof that a companion scope is clear.
func verifyCompanionPIDAbsentFromScope(resource, component string, parentPID, pid int) error {
	parentScope, err := platform.ProcessScope(parentPID)
	if err != nil {
		return fmt.Errorf("inspect companion parent scope: %w", err)
	}
	if parentScope == "" || filepath.Base(parentScope) != resourceServiceScopeName(resource, "service")+".scope" || filepath.Base(filepath.Dir(parentScope)) != platform.ServicesSlice {
		return fmt.Errorf("cannot verify %s companion scope from the current service scope", resource)
	}
	name := resourceServiceScopeName(resource, "companion-"+component) + ".scope"
	ref := platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: name, Path: filepath.Join(filepath.Dir(parentScope), name)}
	pids, err := companionScopeProcesses(ref)
	if err != nil {
		return fmt.Errorf("inspect retired companion scope: %w", err)
	}
	if !companionPIDAbsentFromScope(pids, pid) {
		return fmt.Errorf("retired companion PID %d remains listed in its canonical scope", pid)
	}
	return nil
}

func companionPIDAbsentFromScope(pids []int, pid int) bool {
	for _, scopedPID := range pids {
		if scopedPID == pid {
			return false
		}
	}
	return true
}

func companionProcessIdentityMatches(expectedScope, actualScope, bin string, expected []string, executable, command string) bool {
	// Atomic CLI replacement leaves the old live executable marked deleted.
	// Its exact owned location remains valid; a same-name binary elsewhere does not.
	if (expectedScope != "" && actualScope != expectedScope) || filepath.Clean(strings.TrimSuffix(executable, " (deleted)")) != filepath.Clean(bin) {
		return false
	}
	// ProcessCommandLine projects argv as space-separated text. Refuse recovery
	// when that projection cannot distinguish argument boundaries.
	for _, arg := range expected {
		if arg == "" || strings.ContainsAny(arg, " \t\r\n") {
			return false
		}
	}
	actual := strings.Fields(command)
	return len(actual) == len(expected)+1 && strings.Join(actual[1:], " ") == strings.Join(expected, " ")
}
