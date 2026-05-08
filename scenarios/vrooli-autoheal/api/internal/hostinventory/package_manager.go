package hostinventory

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
)

var kernelReleaseRE = regexp.MustCompile(`\d+\.\d+\.\d+[-._A-Za-z0-9]*`)

func collectPackageState(ctx context.Context, c *DefaultCollector, runningKernel string) PackageState {
	for _, manager := range []string{"dpkg", "rpm", "pacman"} {
		if _, err := exec.LookPath(manager); err == nil {
			switch manager {
			case "dpkg":
				return collectDPKG(ctx, c, runningKernel)
			case "rpm":
				return collectRPM(ctx, c, runningKernel)
			case "pacman":
				return collectPacman(ctx, c, runningKernel)
			}
		}
	}
	return PackageState{}
}

func collectDPKG(ctx context.Context, c *DefaultCollector, runningKernel string) PackageState {
	state := PackageState{Manager: "dpkg"}
	out, err := c.exec.CombinedOutput(ctx, "dpkg-query", "-W", "-f=${db:Status-Abbrev} ${Package} ${Version}\n", "linux-image*", "linux-modules*", "nvidia-*", "amdgpu*", "mesa-*")
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			pkg := fields[1]
			if strings.HasPrefix(fields[0], "ii") {
				state.Installed = append(state.Installed, pkg)
			} else if strings.TrimSpace(fields[0]) != "" {
				state.BrokenOrHeld = append(state.BrokenOrHeld, strings.TrimSpace(line))
			}
		}
	}
	if out, err := c.exec.CombinedOutput(ctx, "apt-mark", "showhold"); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				state.BrokenOrHeld = append(state.BrokenOrHeld, "held: "+line)
			}
		}
	}
	state.KernelModuleDrift = detectKernelPackageDrift(state.Installed, runningKernel)
	return state
}

func collectRPM(ctx context.Context, c *DefaultCollector, runningKernel string) PackageState {
	state := PackageState{Manager: "rpm"}
	out, err := c.exec.CombinedOutput(ctx, "rpm", "-qa", "kernel*", "kmod*", "akmod*", "nvidia*", "mesa*")
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				state.Installed = append(state.Installed, line)
			}
		}
	}
	state.KernelModuleDrift = detectKernelPackageDrift(state.Installed, runningKernel)
	return state
}

func collectPacman(ctx context.Context, c *DefaultCollector, runningKernel string) PackageState {
	state := PackageState{Manager: "pacman"}
	out, err := c.exec.CombinedOutput(ctx, "pacman", "-Q")
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "linux") || strings.Contains(line, "nvidia") || strings.Contains(line, "mesa") {
				state.Installed = append(state.Installed, strings.TrimSpace(line))
			}
		}
	}
	state.KernelModuleDrift = detectKernelPackageDrift(state.Installed, runningKernel)
	return state
}

func detectKernelPackageDrift(packages []string, runningKernel string) []string {
	if runningKernel == "" {
		return nil
	}
	var drift []string
	for _, pkg := range packages {
		if !strings.Contains(pkg, "modules") && !strings.Contains(pkg, "kmod") && !strings.Contains(pkg, "nvidia") {
			continue
		}
		matches := kernelReleaseRE.FindAllString(pkg, -1)
		for _, match := range matches {
			if match != runningKernel && strings.Count(match, ".") >= 2 {
				drift = append(drift, pkg)
				break
			}
		}
	}
	return drift
}
