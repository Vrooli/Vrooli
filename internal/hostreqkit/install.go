package hostreqkit

import (
	"fmt"
	"strings"
)

func InstallCommand(host Host, packageName, sudoMode string) (string, []string, error) {
	switch host.OS {
	case "linux":
		return LinuxInstallCommand(host, packageName, sudoMode)
	case "darwin":
		if host.PackageManager != "brew" {
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
	if SudoAvailable() {
		switch strings.ToLower(strings.TrimSpace(mode)) {
		case "", "ask":
			return "sudo", append([]string{command}, args...), nil
		case "skip":
			return "", nil, fmt.Errorf("automatic install skipped because --sudo-mode=skip")
		case "error":
			return "", nil, fmt.Errorf("automatic install requires sudo but --sudo-mode=error")
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
