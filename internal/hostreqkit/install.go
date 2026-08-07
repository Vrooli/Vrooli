package hostreqkit

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// RunningAsRootFn reports whether the current process has effective UID 0.
// When true, WithSudo skips the sudo wrap entirely — there's nothing to
// escalate. Exposed as a var for tests that want to simulate non-root or
// root execution without actually changing UID.
var RunningAsRootFn = func() bool {
	return os.Geteuid() == 0
}

// ElevationFacts is the small typed seam consumed by WithSudo. The default
// implementation is platform-neutral; runtime and tests can replace it with
// a richer host-inventory-backed observation.
type ElevationFacts struct {
	Platform   string
	Elevated   bool
	CanElevate bool
	Mechanism  string
}

var ElevationFactsFn = func() ElevationFacts {
	platform := hostreqspec.CurrentPlatform()
	if RunningAsRootFn() {
		return ElevationFacts{Platform: platform, Elevated: true, CanElevate: true, Mechanism: "root"}
	}
	if SudoAvailable() {
		return ElevationFacts{Platform: platform, CanElevate: true, Mechanism: "sudo"}
	}
	return ElevationFacts{Platform: platform, Mechanism: "none"}
}

// ErrSudoSkipped marks privileged commands that were refused because the
// operator passed --sudo-mode=skip. Renderers / runtime use errors.Is to
// classify these without string-matching the human message.
var ErrSudoSkipped = errors.New("sudo skipped")

// ErrSudoUnavailable marks privileged commands refused because
// --sudo-mode=error was set explicitly while sudo is on PATH (the operator
// asked us to fail loudly rather than fall back).
var ErrSudoUnavailable = errors.New("sudo unavailable")

// ErrElevationRequired is returned on Windows when an operation needs an
// elevated process. UAC is interactive and cannot be safely spawned by the
// non-interactive setup runner, so the caller must show the exact command.
var ErrElevationRequired = errors.New("elevation required")

// ErrElevationUnavailable marks a fail-closed Unix elevation decision.
var ErrElevationUnavailable = errors.New("elevation unavailable")

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
		switch strings.TrimSpace(host.PackageManager) {
		case "brew":
			return "brew", []string{"install", packageName}, nil
		default:
			return "", nil, fmt.Errorf("automatic install is unavailable without Homebrew")
		}
	case "windows":
		switch strings.TrimSpace(host.PackageManager) {
		case "winget":
			return "winget", []string{"install", "--id", packageName, "--accept-package-agreements", "--accept-source-agreements"}, nil
		case "choco":
			return "choco", []string{"install", packageName, "-y"}, nil
		case "scoop":
			return "scoop", []string{"install", packageName}, nil
		default:
			return "", nil, fmt.Errorf("automatic install is unavailable without a supported Windows package manager")
		}
	default:
		return "", nil, fmt.Errorf("automatic install is unsupported on %s", defaultOS(host.OS))
	}
}

// InstallCommandPreview returns the command shape without evaluating
// elevation. Dry-run is an inspection operation and must not require sudo;
// real execution always goes through InstallCommand and WithSudo.
func InstallCommandPreview(host Host, packageName string) (string, []string, error) {
	switch host.OS {
	case "linux":
		switch strings.TrimSpace(host.PackageManager) {
		case "apt", "apt-get":
			return "apt-get", []string{"install", "-y", packageName}, nil
		case "dnf":
			return "dnf", []string{"install", "-y", packageName}, nil
		case "yum":
			return "yum", []string{"install", "-y", packageName}, nil
		case "pacman":
			return "pacman", []string{"-S", "--noconfirm", packageName}, nil
		case "apk":
			return "apk", []string{"add", packageName}, nil
		case "brew":
			return "brew", []string{"install", packageName}, nil
		}
	case "darwin":
		if strings.TrimSpace(host.PackageManager) == "brew" {
			return "brew", []string{"install", packageName}, nil
		}
	case "windows":
		switch strings.TrimSpace(host.PackageManager) {
		case "winget":
			return "winget", []string{"install", "--id", packageName, "--accept-package-agreements", "--accept-source-agreements"}, nil
		case "choco":
			return "choco", []string{"install", packageName, "-y"}, nil
		case "scoop":
			return "scoop", []string{"install", packageName}, nil
		}
	}
	return "", nil, fmt.Errorf("automatic install is unavailable without a supported package manager")
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
	facts := ElevationFactsFn()
	// If we're already elevated, skip the wrap regardless of mode — `sudo
	// vrooli setup` shouldn't recursively re-prompt for sudo. This must come
	// first: with mode=skip, we'd otherwise return ErrSudoSkipped even though
	// the privileged command would succeed natively.
	if facts.Elevated {
		return command, args, nil
	}
	if strings.EqualFold(strings.TrimSpace(facts.Platform), "windows") {
		return "", nil, fmt.Errorf("%w: run `%s %s` from an elevated Windows prompt", ErrElevationRequired, command, strings.Join(args, " "))
	}
	if facts.CanElevate {
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
	return "", nil, fmt.Errorf("%w: sudo unavailable for `%s %s`", ErrElevationUnavailable, command, strings.Join(args, " "))
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
