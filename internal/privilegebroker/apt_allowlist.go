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
var AptAllowedPackages = []string{"curl", "git", "unzip", "tar", "jq", "ca-certificates", "gnupg", "lsb-release", "iproute2", "postgresql-client", "caddy"}

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
	// Small VPS hosts frequently have IPv6 DNS but no usable IPv6 route. Apt's
	// default address selection can then fail before it tries IPv4, making a
	// healthy target look unrepairable. Force IPv4 and retry transient fetches
	// through the broker's fixed policy surface.
	update = append(append([]string{}, prefix...), "-o", "Acquire::ForceIPv4=true", "-o", "Acquire::Retries=3", "update", "-qq")
	install = append(append(append([]string{}, prefix...), "-o", "Dpkg::Options::=--force-confold", "install", "-y", "-qq", "--no-install-recommends"), names...)
	return update, install, nil
}

func executeApt(ctx context.Context, executor Executor, req Request) Result {
	update, install, err := AptArgs(req)
	if err != nil {
		return NewFailure(req.RequestID, req.Action, "action_not_allowed")
	}
	if aptPackagesInstalled(ctx, executor, req.Apt.Packages) {
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "completed", Changed: false}
	}
	if _, err := executor.Run(ctx, "env", update...); err != nil {
		return NewFailure(req.RequestID, req.Action, "apt_update_failed")
	}
	if _, err := executor.Run(ctx, "env", install...); err != nil {
		return NewFailure(req.RequestID, req.Action, "apt_install_failed")
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "completed", Changed: true}
}

// aptPackagesInstalled keeps recovery independent of package mirror health
// when the target already satisfies the requested host baseline. The broker
// uses a fixed dpkg-query argv and only accepts Debian's exact installed
// status; any probe error falls through to the normal update/install path.
func aptPackagesInstalled(ctx context.Context, executor Executor, packages []string) bool {
	for _, name := range packages {
		out, err := executor.Run(ctx, "dpkg-query", "-W", "-f=${Status}", name)
		if err != nil || strings.TrimSpace(string(out)) != "install ok installed" {
			return false
		}
	}
	return len(packages) > 0
}
