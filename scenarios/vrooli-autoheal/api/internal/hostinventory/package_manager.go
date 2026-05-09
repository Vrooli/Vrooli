package hostinventory

import (
	"context"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

var (
	kernelReleaseRE       = regexp.MustCompile(`\d+\.\d+\.\d+[-._A-Za-z0-9]*`)
	nvidiaDriverPackageRE = regexp.MustCompile(`^nvidia-driver-([0-9]+)(-open)?$`)
)

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
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		pkg := fields[1]
		version := ""
		if len(fields) >= 3 {
			version = fields[2]
		}
		if strings.HasPrefix(status, "ii") {
			state.Installed = append(state.Installed, pkg)
			state.InstalledPackages = append(state.InstalledPackages, PackageInfo{Name: pkg, Version: version, Status: status})
		} else if strings.TrimSpace(status) != "" && !strings.HasPrefix(status, "rc") {
			state.BrokenOrHeld = append(state.BrokenOrHeld, strings.TrimSpace(line))
		}
	}
	if err != nil && strings.TrimSpace(string(out)) == "" {
		state.BrokenOrHeld = append(state.BrokenOrHeld, "dpkg-query failed: "+err.Error())
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
	state.Kernel = buildKernelPackageState(state.Installed, state.BrokenOrHeld, runningKernel)
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
	state.Kernel = buildKernelPackageState(state.Installed, state.BrokenOrHeld, runningKernel)
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
	state.Kernel = buildKernelPackageState(state.Installed, state.BrokenOrHeld, runningKernel)
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

func enrichDriverPackageState(ctx context.Context, c *DefaultCollector, inv *HostInventory) {
	if inv == nil || inv.Packages.Manager == "" || inv.Kernel.Release == "" {
		return
	}
	if !hasNVIDIAEvidence(*inv) {
		return
	}
	driver := buildNVIDIADriverState(ctx, c, *inv)
	if driver.Applicability != "" {
		inv.Packages.Drivers = append(inv.Packages.Drivers, driver)
	}
}

func buildNVIDIADriverState(ctx context.Context, c *DefaultCollector, inv HostInventory) DriverPackageState {
	driver := DriverPackageState{
		Vendor:        "nvidia",
		Applicability: "unsupported",
		LoadedModules: filterModuleNames(inv.Kernel.LoadedModules, []string{"nvidia", "nouveau"}),
	}
	if inv.Packages.Manager != "dpkg" {
		return driver
	}
	driver.Applicability = "needs_corroboration"
	for _, pkg := range inv.Packages.InstalledPackages {
		matches := nvidiaDriverPackageRE.FindStringSubmatch(pkg.Name)
		if len(matches) == 0 {
			continue
		}
		driver.Series = matches[1]
		if matches[2] == "-open" {
			driver.Flavor = "open"
		}
		driver.InstalledPackages = append(driver.InstalledPackages, pkg)
	}
	if driver.Series == "" {
		for _, pkg := range inv.Packages.InstalledPackages {
			if strings.HasPrefix(pkg.Name, "linux-modules-nvidia-") {
				parts := strings.Split(pkg.Name, "-")
				for _, part := range parts {
					if allDigits(part) {
						driver.Series = part
						break
					}
				}
			}
			if driver.Series != "" {
				break
			}
		}
	}
	if driver.Series == "" {
		return driver
	}
	parts := []string{"linux-modules-nvidia", driver.Series}
	if driver.Flavor != "" {
		parts = append(parts, driver.Flavor)
	}
	parts = append(parts, inv.Kernel.Release)
	driver.ExpectedModulePackage = strings.Join(parts, "-")
	for _, pkg := range inv.Packages.InstalledPackages {
		if pkg.Name == driver.ExpectedModulePackage {
			driver.ExpectedPackageInstalled = true
			driver.InstalledPackages = append(driver.InstalledPackages, pkg)
			break
		}
	}
	if driver.ExpectedPackageInstalled {
		driver.Applicability = "applicable"
		return driver
	}
	driver.MissingModulePackage = driver.ExpectedModulePackage
	driver.Applicability = "applicable"
	candidate := collectAPTCandidate(ctx, c, driver.ExpectedModulePackage)
	driver.Candidate = &candidate
	return driver
}

func collectAPTCandidate(ctx context.Context, c *DefaultCollector, packageName string) PackageCandidate {
	candidate := PackageCandidate{Name: packageName, Source: "apt-cache policy"}
	if _, err := exec.LookPath("apt-cache"); err != nil {
		candidate.Error = err.Error()
		return candidate
	}
	out, err := c.exec.CombinedOutput(ctx, "apt-cache", "policy", packageName)
	if err != nil {
		candidate.Error = truncateEvidence(string(out)+" "+err.Error(), 300)
		return candidate
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Candidate:") {
			continue
		}
		version := strings.TrimSpace(strings.TrimPrefix(line, "Candidate:"))
		if version != "" && version != "(none)" {
			candidate.Version = version
			candidate.Available = true
		}
		break
	}
	return candidate
}

func buildKernelPackageState(installed, held []string, runningKernel string) KernelPackageState {
	state := KernelPackageState{RunningKernel: runningKernel, HeldOrBlocked: held}
	if runningKernel == "" {
		return state
	}
	for _, pkg := range installed {
		if strings.Contains(pkg, runningKernel) {
			state.InstalledMatching = append(state.InstalledMatching, pkg)
			continue
		}
		if kernelReleaseRE.MatchString(pkg) {
			state.InstalledOtherKernel = append(state.InstalledOtherKernel, pkg)
		}
	}
	for _, expected := range []string{"linux-image-" + runningKernel, "linux-modules-" + runningKernel} {
		if !containsString(installed, expected) {
			state.MissingMatching = append(state.MissingMatching, expected)
		}
	}
	sort.Strings(state.InstalledMatching)
	sort.Strings(state.InstalledOtherKernel)
	sort.Strings(state.MissingMatching)
	return state
}

func hasNVIDIAEvidence(inv HostInventory) bool {
	for _, device := range inv.Devices {
		if strings.Contains(strings.ToLower(device.VendorName+" "+device.DeviceName), "nvidia") || strings.EqualFold(device.VendorID, "10de") {
			return true
		}
	}
	for _, runtime := range inv.Runtimes {
		if runtime.Name == "nvidia-smi" && runtime.Path != "" {
			return true
		}
	}
	for _, module := range inv.Kernel.LoadedModules {
		if strings.HasPrefix(module, "nvidia") || module == "nouveau" {
			return true
		}
	}
	return false
}

func filterModuleNames(modules, prefixes []string) []string {
	var out []string
	for _, module := range modules {
		for _, prefix := range prefixes {
			if strings.HasPrefix(module, prefix) {
				out = append(out, module)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
