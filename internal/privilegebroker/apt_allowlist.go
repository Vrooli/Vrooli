package privilegebroker

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// AptAllowedPackages is the closed set of Debian packages a cloud target may
// ask the broker to ensure. It is compiled into the binary on purpose: the
// list is policy, not configuration, and a target cannot widen it.
var AptAllowedPackages = []string{"curl", "git", "unzip", "tar", "jq", "ca-certificates", "gnupg", "lsb-release", "caddy"}

func aptAllowed(name string) bool {
	for _, allowed := range AptAllowedPackages {
		if name == allowed {
			return true
		}
	}
	return false
}

func validateApt(req Request) error {
	if req.Subject != (Subject{}) || req.Volume != nil || req.RuntimeHome != nil || req.Log != nil || req.Journal != nil || req.Docker != nil || req.Edge != nil || req.Process != nil || req.Caddy != nil {
		return fmt.Errorf("subject_not_allowed")
	}
	if req.Apt == nil || len(req.Apt.Packages) == 0 {
		return fmt.Errorf("apt_package_list_required")
	}
	for _, name := range req.Apt.Packages {
		if name != strings.TrimSpace(name) || !aptAllowed(name) {
			return fmt.Errorf("apt_package_not_allowed")
		}
	}
	return nil
}

// AptArgs returns the fixed argv pair (update, install) for an accepted
// request. Package names are sorted and de-duplicated so equal subjects build
// equal argv. Non-interactive front-end selection travels as argv through
// env(1) because the executor seam carries no environment.
func AptArgs(req Request) (update, install []string, err error) {
	if err := Validate(req); err != nil {
		return nil, nil, err
	}
	if req.Action != ActionAptPackagesEnsure {
		return nil, nil, fmt.Errorf("action_not_allowed")
	}
	seen := map[string]struct{}{}
	names := make([]string, 0, len(req.Apt.Packages))
	for _, name := range req.Apt.Packages {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	prefix := []string{"DEBIAN_FRONTEND=noninteractive", "NEEDRESTART_MODE=a", "apt-get"}
	update = append(append([]string{}, prefix...), "update", "-qq")
	install = append(append(append([]string{}, prefix...), "install", "-y", "-qq", "--no-install-recommends"), names...)
	return update, install, nil
}

func executeApt(ctx context.Context, executor Executor, req Request) Result {
	update, install, err := AptArgs(req)
	if err != nil {
		return NewFailure(req.RequestID, req.Action, "action_not_allowed")
	}
	if _, err := executor.Run(ctx, "env", update...); err != nil {
		return NewFailure(req.RequestID, req.Action, "apt_update_failed")
	}
	if _, err := executor.Run(ctx, "env", install...); err != nil {
		return NewFailure(req.RequestID, req.Action, "apt_install_failed")
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "completed", Changed: true}
}
