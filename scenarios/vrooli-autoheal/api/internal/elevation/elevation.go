// Package elevation is autoheal's only runtime privilege boundary. It
// accepts a closed set of service actions and delegates the actual consent
// and elevation decision to the control-plane hostreqkit seam.
package elevation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

type State string

const (
	Granted     State = "granted"
	NeedsSetup  State = "needs_setup"
	Unsupported State = "unsupported"
	Refused     State = "refused"
)

type Outcome struct {
	State     State  `json:"state"`
	Command   string `json:"command,omitempty"`
	Reason    string `json:"reason,omitempty"`
	GrantName string `json:"grantName,omitempty"`
}

type Action string

const (
	ServiceStart   Action = "service.start"
	ServiceRestart Action = "service.restart"
	ServiceStop    Action = "service.stop"
)

type Executor interface {
	CombinedOutput(context.Context, string, ...string) ([]byte, error)
}

var grantPath = "/etc/sudoers.d/vrooli-autoheal"

var grantPresent = func() bool {
	info, err := os.Stat(grantPath)
	return err == nil && info.Mode().Perm() == 0o440
}

var osNameFn = func() string { return runtime.GOOS }

// allowedUnits is intentionally closed. The setup safeguard grants only
// these literal service subjects; accepting an arbitrary unit here would turn
// the NOPASSWD policy into a general root command.
var allowedUnits = map[string]struct{}{
	"docker":                       {},
	"systemd-resolved":             {},
	"cloudflared":                  {},
	"NetworkManager":               {},
	"systemd-networkd":             {},
	"systemd-timesyncd":            {},
	"gnome-remote-desktop":         {},
	"gnome-remote-desktop.service": {},
	"xrdp":                         {},
	"gdm":                          {},
	"gdm3":                         {},
	"lightdm":                      {},
	"sddm":                         {},
}

func SetGrantPathForTest(path string, present func() bool) func() {
	oldPath, oldPresent := grantPath, grantPresent
	grantPath, grantPresent = path, present
	return func() { grantPath, grantPresent = oldPath, oldPresent }
}

func Run(ctx context.Context, executor Executor, action Action, unit string) (Outcome, []byte, error) {
	if executor == nil {
		return Outcome{State: Refused, Reason: "no command executor configured"}, nil, errors.New("elevation: nil executor")
	}
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return Outcome{State: Refused, Reason: "service unit is empty"}, nil, errors.New("elevation: empty service unit")
	}
	if osNameFn() != "linux" {
		return Outcome{State: Unsupported, Command: fmt.Sprintf("service action %s %s", action, unit), Reason: "service recovery grant is Linux/systemd-only; use the declared native service mechanism", GrantName: "autoheal_recovery_privileges"}, nil, nil
	}
	if _, ok := allowedUnits[unit]; !ok {
		return Outcome{State: Refused, Reason: "service unit is not in the setup-provisioned grant registry", GrantName: "autoheal_recovery_privileges"}, nil, fmt.Errorf("elevation: unsupported service unit %q", unit)
	}
	if !grantPresent() && !hostreqkit.RunningAsRootFn() {
		return Outcome{State: NeedsSetup, Command: "sudo vrooli setup", Reason: "the autoheal recovery sudoers grant is missing", GrantName: "autoheal_recovery_privileges"}, nil, nil
	}
	verb := map[Action]string{ServiceStart: "start", ServiceRestart: "restart", ServiceStop: "stop"}[action]
	if verb == "" {
		return Outcome{State: Refused, Reason: "action is not in the privilege broker registry"}, nil, fmt.Errorf("elevation: unsupported action %q", action)
	}
	command := platform.ServiceManagerCommandPath()
	if command == "" {
		return Outcome{State: Unsupported, Reason: "no native service-manager command is available for this platform", GrantName: "autoheal_recovery_privileges"}, nil, nil
	}
	args := []string{verb, unit}
	out, err := hostreqkit.RunPrivilegedCommandWithOutput("ask", command, args, func(name string, args ...string) ([]byte, error) {
		return executor.CombinedOutput(ctx, name, args...)
	})
	if err != nil {
		state := Refused
		if errors.Is(err, hostreqkit.ErrSudoSkipped) || errors.Is(err, hostreqkit.ErrSudoUnavailable) || errors.Is(err, hostreqkit.ErrElevationRequired) {
			state = NeedsSetup
		}
		return Outcome{State: state, Command: "sudo -n " + command + " " + strings.Join(args, " "), Reason: err.Error(), GrantName: "autoheal_recovery_privileges"}, out, err
	}
	return Outcome{State: Granted, Command: command + " " + strings.Join(args, " "), GrantName: "autoheal_recovery_privileges"}, out, nil
}
