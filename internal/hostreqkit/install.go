package hostreqkit

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// RunningAsRootFn reports whether the current process has effective UID 0.
// When true, WithSudo skips the sudo wrap entirely — there's nothing to
// escalate. Exposed as a var for tests that want to simulate non-root or
// root execution without actually changing UID.
var RunningAsRootFn = func() bool {
	return os.Geteuid() == 0
}

// ErrSudoSkipped marks privileged commands that were refused because the
// operator passed --sudo-mode=skip. Renderers / runtime use errors.Is to
// classify these without string-matching the human message.
var ErrSudoSkipped = errors.New("sudo skipped")

// ErrSudoUnavailable marks privileged commands refused because
// --sudo-mode=error was set explicitly while sudo is on PATH (the operator
// asked us to fail loudly rather than fall back).
var ErrSudoUnavailable = errors.New("sudo unavailable")

// IsSudoSkipped reports whether err (or any error it wraps) is ErrSudoSkipped
// or ErrSudoUnavailable. Callers that just want to know "was this blocked by
// sudo policy?" should use this.
func IsSudoSkipped(err error) bool {
	return errors.Is(err, ErrSudoSkipped) || errors.Is(err, ErrSudoUnavailable)
}

func InstallCommand(host Host, packageName, sudoMode string) (string, []string, error) {
	switch host.OS {
	case "linux":
		return LinuxInstallCommand(host, packageName, sudoMode)
	case "darwin":
		if strings.TrimSpace(host.PackageManager) != "brew" {
			return "", nil, fmt.Errorf("automatic install is unavailable without Homebrew")
		}
		return "brew", []string{"install", packageName}, nil
	case "windows":
		if _, err := LookPathFn("winget"); err != nil {
			return "", nil, fmt.Errorf("automatic install is unavailable without winget")
		}
		return "winget", []string{"install", "--id", packageName, "--accept-package-agreements", "--accept-source-agreements"}, nil
	default:
		return "", nil, fmt.Errorf("automatic install is unsupported on %s", defaultOS(host.OS))
	}
}

func LinuxInstallCommand(host Host, packageName, sudoMode string) (string, []string, error) {
	manager := strings.TrimSpace(host.PackageManager)
	switch manager {
	case "apt", "apt-get":
		return WithSudo(sudoMode, "apt-get", []string{"install", "-y", packageName})
	case "dnf":
		return WithSudo(sudoMode, "dnf", []string{"install", "-y", packageName})
	case "yum":
		return WithSudo(sudoMode, "yum", []string{"install", "-y", packageName})
	case "pacman":
		return WithSudo(sudoMode, "pacman", []string{"-S", "--noconfirm", packageName})
	case "apk":
		return WithSudo(sudoMode, "apk", []string{"add", packageName})
	case "brew":
		return "brew", []string{"install", packageName}, nil
	default:
		return "", nil, fmt.Errorf("automatic install is unavailable without a supported package manager")
	}
}

func WithSudo(mode, command string, args []string) (string, []string, error) {
	// If we're already root, skip the wrap regardless of mode — `sudo
	// vrooli setup` shouldn't recursively re-prompt for sudo. This must come
	// first: with mode=skip, we'd otherwise return ErrSudoSkipped even though
	// the privileged command would succeed natively.
	if RunningAsRootFn() {
		return command, args, nil
	}
	if SudoAvailable() {
		// Empty mode now defaults to "skip": `vrooli setup` is non-interactive
		// by default and should not produce a sudo password prompt out of the
		// box. Operators opt in via `sudo vrooli setup` (becomes root, which
		// the RunningAsRootFn branch above handles) or `--sudo-mode=ask`.
		normalized := strings.ToLower(strings.TrimSpace(mode))
		if normalized == "" {
			normalized = "skip"
		}
		switch normalized {
		case "ask":
			return "sudo", append([]string{command}, args...), nil
		case "skip":
			return "", nil, fmt.Errorf("%w: re-run as `sudo vrooli setup` (or pass --sudo-mode=ask)", ErrSudoSkipped)
		case "error":
			return "", nil, fmt.Errorf("%w: automatic install requires sudo but --sudo-mode=error", ErrSudoUnavailable)
		default:
			return "", nil, fmt.Errorf("invalid sudo mode: %s", mode)
		}
	}
	return command, args, nil
}

func SudoAvailable() bool {
	_, err := LookPathFn("sudo")
	return err == nil
}

func defaultOS(value string) string {
	if strings.TrimSpace(value) == "" {
		return "this platform"
	}
	return value
}
