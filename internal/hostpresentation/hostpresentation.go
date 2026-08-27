// Package hostpresentation classifies the presentation capability of the

// session that invoked Vrooli. It deliberately describes the session, not the
// host: a graphical host reached over SSH is still a remote shell.
package hostpresentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/shell"
)

const (
	hostpresentationYes = "yes"
)

type Kind string

const (
	KindLocalGraphical     Kind = "local-graphical"
	KindWSLGraphical       Kind = "wsl-graphical"
	KindForwardedGraphical Kind = "forwarded-graphical"
	KindRemoteDesktop      Kind = "remote-desktop"
	KindRemoteShell        Kind = "remote-shell"
	KindHeadless           Kind = "headless"
	KindUnknown            Kind = "unknown"
)

type Capability struct {
	Kind      Kind     `json:"kind"`
	Reachable bool     `json:"reachable"`
	Reason    string   `json:"reason"`
	Evidence  []string `json:"evidence,omitempty"`
	Degraded  bool     `json:"degraded,omitempty"`
}

type Probe interface {
	Env(string) string
	shell.Runner
}

type defaultProbe struct{}

func (defaultProbe) Env(name string) string { return os.Getenv(name) }

func (defaultProbe) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (defaultProbe) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	// These two pseudo-commands keep filesystem probes inside the narrow Probe
	// contract without spawning a shell or an unbounded helper process.
	if name == "file-exists" && len(args) == 1 {
		_, err := os.Stat(args[0])
		if err == nil {
			return []byte("true"), nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return []byte("false"), nil
		}
		return nil, err
	}
	if name == "file-read" && len(args) == 1 {
		return os.ReadFile(args[0])
	}
	if name == "windows-session" {
		return nativeWindowsSession(ctx)
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, path, args...).Output()
}

func Detect(ctx context.Context) Capability { return DetectWith(ctx, defaultProbe{}) }

// DetectWith bounds the entire classification. Every subprocess probe is
// separately bounded by probeRun, and a timeout can never produce a reachable
// result.
func DetectWith(ctx context.Context, p Probe) Capability {
	if p == nil {
		p = defaultProbe{}
	}
	ctx, cancel := context.WithTimeout(ctx, tuning.ShortOperationDeadline)
	defer cancel()
	return detectWithOS(ctx, p, runtime.GOOS)
}

// detectWithOS is intentionally kept as a small test seam. It lets one fake
// probe exercise every platform rule on the development host without claiming
// hardware evidence for another operating system.
func detectWithOS(ctx context.Context, p Probe, goos string) Capability {
	if result, ok := universalOverride(ctx, p); ok {
		return result
	}
	switch goos {
	case "linux":
		return detectLinux(ctx, p)
	case "darwin":
		return detectDarwin(ctx, p)
	case "windows":
		return detectWindows(ctx, p)
	default:
		return capability(KindUnknown, false, "unsupported operating system", "GOOS="+goos)
	}
}

func universalOverride(ctx context.Context, p Probe) (Capability, bool) {
	if value := p.Env("CI"); isTruthy(value) {
		return capability(KindHeadless, false, "CI environment", "CI="+value), true
	}
	if value := p.Env("container"); strings.TrimSpace(value) != "" {
		return capability(KindHeadless, false, "container environment", "container="+value), true
	}
	value, err := probeRun(ctx, p, "file-exists", "/.dockerenv")
	if err == nil && strings.EqualFold(strings.TrimSpace(string(value)), "true") {
		return capability(KindHeadless, false, "Docker container", "file /.dockerenv=present"), true
	}
	return Capability{}, false
}

func detectLinux(ctx context.Context, p Probe) Capability {
	return detectLinuxWithUID(ctx, p, effectiveUID())
}

//nolint:gocyclo // Linux presentation detection is an OS, display, session, and user capability matrix.
func detectLinuxWithUID(ctx context.Context, p Probe, uid int) Capability {
	evidence := []string{}
	wslVersion, wslErr := probeRun(ctx, p, "file-read", "/proc/version")
	isWSL := wslErr == nil && strings.Contains(strings.ToLower(string(wslVersion)), "microsoft")
	evidence = append(evidence, "IsWSL="+fmt.Sprintf("%t", isWSL))
	if wslErr != nil && isTimeout(wslErr) {
		return degraded(KindHeadless, "WSL probe timed out", evidence)
	}
	display := firstNonEmpty(p.Env("DISPLAY"), p.Env("WAYLAND_DISPLAY"))
	sshConnection := p.Env("SSH_CONNECTION")
	sshTTY := p.Env("SSH_TTY")
	evidence = append(evidence, "DISPLAY="+p.Env("DISPLAY"), "WAYLAND_DISPLAY="+p.Env("WAYLAND_DISPLAY"), "SSH_CONNECTION="+sshConnection, "SSH_TTY="+sshTTY)

	if uid == 0 && strings.TrimSpace(p.Env("SUDO_USER")) != "" {
		active, err := probeRun(ctx, p, "loginctl", "show-seat", "seat0", "-p", "ActiveSession", "--value")
		evidence = append(evidence, "loginctl ActiveSession="+strings.TrimSpace(string(active)))
		if err != nil && isTimeout(err) {
			return degraded(KindHeadless, "session probe timed out", evidence)
		}
		if err == nil && strings.TrimSpace(string(active)) != "" {
			session := strings.TrimSpace(string(active))
			info, infoErr := probeRun(ctx, p, "loginctl", "show-session", session, "-p", "Name,Type,Remote")
			evidence = append(evidence, "loginctl session="+strings.TrimSpace(string(info)))
			if infoErr != nil && isTimeout(infoErr) {
				return degraded(KindHeadless, "session probe timed out", evidence)
			}
			name, kind, remote := loginctlFields(string(info))
			if infoErr == nil && name == p.Env("SUDO_USER") && (kind == "x11" || kind == "wayland") && remote != hostpresentationYes {
				return capability(KindLocalGraphical, true, "elevated setup uses the invoking user's graphical session", evidence...)
			}
		}
	}

	if isWSL && display != "" {
		return capability(KindWSLGraphical, true, "WSL graphical session", evidence...)
	}
	if display != "" {
		remote, err := probeRun(ctx, p, "loginctl", "show-session", "self", "-p", "Remote", "--value")
		remoteValue := strings.TrimSpace(string(remote))
		evidence = append(evidence, "loginctl Remote="+remoteValue)
		if err != nil && isTimeout(err) {
			return degraded(KindHeadless, "session probe timed out", evidence)
		}
		if remoteValue != hostpresentationYes && sshConnection == "" {
			return capability(KindLocalGraphical, true, "local graphical session", evidence...)
		}
		if sshConnection != "" {
			return capability(KindForwardedGraphical, true, "SSH session with graphical forwarding", evidence...)
		}
	}
	if sshConnection != "" || sshTTY != "" {
		remote, err := probeRun(ctx, p, "loginctl", "show-session", "self", "-p", "Remote", "--value")
		evidence = append(evidence, "loginctl Remote="+strings.TrimSpace(string(remote)))
		if err != nil && isTimeout(err) {
			return degraded(KindRemoteShell, "session probe timed out", evidence)
		}
		return capability(KindRemoteShell, false, "SSH session, no local display", evidence...)
	}
	return capability(KindHeadless, false, "no reachable graphical session", evidence...)
}

func detectDarwin(ctx context.Context, p Probe) Capability {
	manager, err := probeRun(ctx, p, "launchctl", "managername")
	managerValue := strings.TrimSpace(string(manager))
	evidence := []string{"launchctl managername=" + managerValue}
	if err != nil && isTimeout(err) {
		return degraded(KindUnknown, "launchctl probe timed out", evidence)
	}
	if managerValue == "Aqua" {
		return capability(KindLocalGraphical, true, "Aqua graphical session", evidence...)
	}
	if p.Env("SSH_CONNECTION") != "" {
		evidence = append(evidence, "SSH_CONNECTION="+p.Env("SSH_CONNECTION"))
		return capability(KindRemoteShell, false, "SSH session without Aqua", evidence...)
	}
	return capability(KindHeadless, false, "no Aqua graphical session", evidence...)
}

func detectWindows(ctx context.Context, p Probe) Capability {
	output, err := probeRun(ctx, p, "windows-session")
	evidence := []string{"windows-session=" + strings.TrimSpace(string(output))}
	if err != nil {
		if isTimeout(err) {
			return degraded(KindUnknown, "Windows session probe timed out", evidence)
		}
		return capability(KindUnknown, false, "Windows session probe unavailable", evidence...)
	}
	process, console, ok := parseWindowsSession(string(output))
	if !ok {
		return capability(KindUnknown, false, "Windows session probe returned invalid data", evidence...)
	}
	if process == 0 {
		return capability(KindHeadless, false, "process is in session 0", evidence...)
	}
	if process == console {
		return capability(KindLocalGraphical, true, "active Windows console session", evidence...)
	}
	return capability(KindRemoteDesktop, true, "non-console Windows session", evidence...)
}

func capability(kind Kind, reachable bool, reason string, evidence ...string) Capability {
	return Capability{Kind: kind, Reachable: reachable, Reason: reason, Evidence: append([]string(nil), evidence...)}
}

func degraded(kind Kind, reason string, evidence []string) Capability {
	return Capability{Kind: kind, Reason: reason, Evidence: append([]string(nil), evidence...), Degraded: true}
}

func probeRun(ctx context.Context, p Probe, name string, args ...string) ([]byte, error) {
	probeCtx, cancel := context.WithTimeout(ctx, tuning.HostPresentationProbeTimeout)
	defer cancel()
	return p.Run(probeCtx, name, args...)
}

func isTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", hostpresentationYes, "on":
		return true
	default:
		return false
	}
}

func loginctlFields(value string) (name, kind, remote string) {
	for _, line := range strings.Split(value, "\n") {
		key, field, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "Name":
			name = field
		case "Type":
			kind = field
		case "Remote":
			remote = field
		}
	}
	return name, kind, remote
}

func parseWindowsSession(value string) (process, console uint32, ok bool) {
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "process=%d console=%d", &process, &console); err != nil {
		return 0, 0, false
	}
	return process, console, true
}
