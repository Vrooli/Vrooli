package graphicalsession

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var collectFacts = func() hostinventory.Snapshot {
	return hostinventory.CollectPlatformFacts(context.Background())
}

var runPrivileged = hostreqkit.RunPrivilegedCommand
var runPrivilegedWithStdin = hostreqkit.RunPrivilegedCommandWithStdin

const macOSAutoLoginPasswordEnv = "VROOLI_GRAPHICAL_SESSION_PASSWORD"

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	platform := hostreqspec.PlatformFromGOOS(host.OS)
	if platform != hostreqspec.PlatformLinux && platform != hostreqspec.PlatformMacOS && platform != hostreqspec.PlatformWindows {
		return hostreqkit.NotApplicableRequirementStatus(requirement, "persistent graphical login is not applicable on this platform")
	}
	facts := collectFacts()
	status.Evidence = map[string]any{
		"session_type": facts.SessionType, "display_attached": facts.DisplayAttached,
		"active_session_user": facts.ActiveSessionUser, "auto_login_user": facts.AutoLoginUser,
		"macos_auto_login_status": facts.ProbeStatuses["macos_auto_login"],
	}
	if strings.TrimSpace(facts.SessionType) != "" && facts.DisplayAttached {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("graphical session %q is present for %s", facts.SessionType, facts.ActiveSessionUser))
		return status
	}
	if autoLoginUser := strings.TrimSpace(facts.AutoLoginUser); autoLoginUser != "" {
		status.Notes = append(status.Notes, fmt.Sprintf("automatic login is configured for %s, but no active graphical session was observed", autoLoginUser))
	}
	status.Notes = append(status.Notes, "no persistent graphical session was observed")
	status.Notes = append(status.Notes, operatorAction(platform, requirement))
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.ExecutionState != hostreqkit.ExecutionPending || status.Applied {
		return status, nil
	}
	allowed, _ := status.Config["allow_enable_automatic_login"].(bool)
	if !allowed {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingManual
		status.Command = "vrooli setup --include-optional"
		status.Notes = append(status.Notes, "automatic login is disabled by default; set allow_enable_automatic_login=true only after accepting the physical-security tradeoff")
		return status, nil
	}
	user, _ := status.Config["desktop_user"].(string)
	if strings.TrimSpace(user) == "" {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingManual
		status.Notes = append(status.Notes, "desktop_user is required before automatic login can be configured")
		return status, nil
	}
	platform := hostreqspec.PlatformFromGOOS(host.OS)
	installAgent, _ := status.Config["allow_install_gui_launch_agent"].(bool)
	if installAgent && platform == hostreqspec.PlatformMacOS {
		program, _ := status.Config["gui_launch_agent_program"].(string)
		if strings.TrimSpace(program) == "" {
			status.ExecutionState = hostreqkit.ExecutionManualActionRequired
			status.BlockingReason = hostreqkit.BlockingManual
			status.Notes = append(status.Notes, "gui_launch_agent_program is required before the macOS GUI-domain agent can be installed; provide an absolute target-owned program path")
			return status, nil
		}
	}
	command, args, ok := automaticLoginCommand(platform, user)
	if !ok {
		return hostreqkit.NotApplicableRequirementStatus(hostreqspec.ResolvedRequirement{Name: status.Name, Kind: status.Kind, Required: status.Required, Config: status.Config}, "automatic login is not applicable on this platform"), nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would configure automatic login for %s", user))
		return status, nil
	}
	if platform == hostreqspec.PlatformMacOS {
		_, passwordInjected := os.LookupEnv(macOSAutoLoginPasswordEnv)
		effective, _ := status.Evidence["macos_auto_login_status"].(string)
		if effective != "enabled" && !passwordInjected {
			status.ExecutionState = hostreqkit.ExecutionManualActionRequired
			status.BlockingReason = hostreqkit.BlockingManual
			status.Command = "vrooli-bridge dispatch job <node> --credential-injection <logical-id>:<field>:VROOLI_GRAPHICAL_SESSION_PASSWORD"
			status.Notes = append(status.Notes, "macOS automatic-login status is not confirmed enabled; provide the target user's password through one ephemeral credential injection and rerun the safeguard")
			return status, nil
		}
	}
	var err error
	if platform == hostreqspec.PlatformMacOS {
		if password, present := os.LookupEnv(macOSAutoLoginPasswordEnv); present && password != "" {
			// sysadminctl accepts '-' as a password prompt. Feed the transient
			// credential through stdin so it never appears in argv, a plist, or a
			// repository file. Bridge credential injection supplies and zeroes the
			// environment value for the bounded typed job.
			_, err = runPrivilegedWithStdin(opts.SudoMode, "/usr/sbin/sysadminctl", password+"\n", []string{"-autologin", "set", "-userName", user, "-password", "-"})
		} else {
			err = runPrivileged(opts.SudoMode, command, args, opts)
		}
	} else {
		err = runPrivileged(opts.SudoMode, command, args, opts)
	}
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "configure automatic login failed: "+err.Error())
		return status, nil
	}
	if platform == hostreqspec.PlatformMacOS {
		// sysadminctl can exit successfully while macOS still declines to
		// persist automatic login (for example because the account/security
		// posture is not eligible). Do not claim that the safeguard applied
		// until the platform probe confirms the postcondition.
		facts := collectFacts()
		if status.Evidence == nil {
			status.Evidence = map[string]any{}
		}
		status.Evidence["macos_auto_login_status"] = facts.ProbeStatuses["macos_auto_login"]
		if facts.ProbeStatuses["macos_auto_login"] != "enabled" {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("automatic-login command completed, but macOS reports automatic login %q; no reboot should be used for desktop validation", facts.ProbeStatuses["macos_auto_login"]))
			return status, nil
		}
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "automatic login configured; reboot and re-probe before desktop validation")
	if platform == hostreqspec.PlatformMacOS {
		if _, present := os.LookupEnv(macOSAutoLoginPasswordEnv); !present {
			status.Notes = append(status.Notes, "no transient macOS login password was injected; passworded accounts may require sysadminctl credential injection before reboot")
		}
	}
	if installAgent && platform == hostreqspec.PlatformMacOS {
		status.Notes = append(status.Notes, "GUI launch-agent installation is separately authorized but requires a target-specific agent install operation")
	}
	return status, nil
}

func operatorAction(platform hostreqspec.Platform, requirement hostreqspec.ResolvedRequirement) string {
	if enabled, _ := requirement.Config["allow_enable_automatic_login"].(bool); enabled {
		return "automatic-login opt-in is enabled; run vrooli setup with elevated privileges, then reboot and re-probe"
	}
	switch platform {
	case hostreqspec.PlatformLinux:
		return "log in a graphical session or explicitly opt in to managed display-manager automatic login"
	case hostreqspec.PlatformMacOS:
		return "enable an auto-logged-in Aqua user or explicitly opt in to managed macOS automatic login"
	case hostreqspec.PlatformWindows:
		return "log in an interactive Windows session or explicitly opt in to managed automatic login"
	default:
		return "provide a supported graphical session"
	}
}

func automaticLoginCommand(platform hostreqspec.Platform, user string) (string, []string, bool) {
	switch platform {
	case hostreqspec.PlatformLinux:
		return "sh", []string{"-c", "install -d -m 0755 /etc/gdm3; printf '[daemon]\\nAutomaticLoginEnable=true\\nAutomaticLogin=%s\\n' " + shellQuote(user) + " > /etc/gdm3/custom.conf"}, true
	case hostreqspec.PlatformMacOS:
		return "defaults", []string{"write", "/Library/Preferences/com.apple.loginwindow", "autoLoginUser", "-string", user}, true
	case hostreqspec.PlatformWindows:
		return "reg.exe", []string{"add", `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon`, "/v", "AutoAdminLogon", "/t", "REG_SZ", "/d", "1", "/f"}, true
	default:
		return "", nil, false
	}
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
