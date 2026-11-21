package privilegebroker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// This registry preserves autoheal's existing service recovery targets and adds
// the host logging/pressure services needed for infrastructure maintenance.
var recoveryUnits = map[string]bool{
	"docker": true, "systemd-resolved": true, "cloudflared": true,
	"NetworkManager": true, "systemd-networkd": true, "systemd-timesyncd": true,
	"gnome-remote-desktop": true, "xrdp": true, "gdm": true, "gdm3": true,
	"lightdm": true, "sddm": true, "rsyslog": true, "systemd-journald": true,
	"systemd-oomd": true,
}

func ServiceRecoveryAllowed(operation, unit string) bool {
	return (operation == "start" || operation == "restart") && recoveryUnits[strings.TrimSuffix(unit, ".service")]
}

func validateServiceRecovery(req Request) error {
	if req.Service == nil || req.Subject != (Subject{}) || req.Volume != nil || req.SMART != nil || req.RuntimeHome != nil || req.Log != nil || req.Journal != nil || req.Docker != nil || hasCloudSubject(req) {
		return fmt.Errorf("service_subject_required")
	}
	if !ServiceRecoveryAllowed(req.Service.Operation, req.Service.Unit) {
		return fmt.Errorf("service_recovery_not_allowed")
	}
	return nil
}

func executeServiceRecovery(ctx context.Context, executor Executor, req Request) Result {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	unit := strings.TrimSuffix(req.Service.Unit, ".service") + ".service"
	if _, err := executor.Run(ctx, "/usr/bin/systemctl", req.Service.Operation, unit); err != nil {
		return NewFailure(req.RequestID, req.Action, "service_recovery_failed")
	}
	if _, err := executor.Run(ctx, "/usr/bin/systemctl", "is-active", "--quiet", unit); err != nil {
		return NewFailure(req.RequestID, req.Action, "service_not_active")
	}
	return Result{Status: "verified", Changed: true, Evidence: Evidence{Available: true, Active: true, Detail: unit}}
}
