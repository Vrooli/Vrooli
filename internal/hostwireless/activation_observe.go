package hostwireless

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

var nativeName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

func observeActivationHost(ctx context.Context, plan *ActivationPlan, spec buildProfile, inventory hostinventory.HostInventory) {
	for _, module := range spec.modules {
		plan.Originals = append(plan.Originals, observeProvider(ctx, plan.KernelRelease, module))
	}
	for _, file := range bootFiles(plan.KernelRelease) {
		plan.Preserved = append(plan.Preserved, observeFile(ctx, file))
	}
	for _, name := range []string{"modules.dep", "modules.dep.bin", "modules.alias", "modules.alias.bin"} {
		plan.Preserved = append(plan.Preserved, observeFile(ctx, filepath.Join("/lib/modules", plan.KernelRelease, name)))
	}
	plan.Preserved = append(plan.Preserved, observeFile(ctx, "/etc/depmod.d/ubuntu.conf"))
	plan.Preserved = append(plan.Preserved, observeFile(ctx, "/lib/modules/"+plan.KernelRelease+"/build/Module.symvers"))
	for _, release := range inventory.Kernel.InstalledModuleTrees {
		if release != plan.KernelRelease && nativeName.MatchString(release) &&
			regularFileExists("/boot/vmlinuz-"+release) && regularFileExists("/boot/initrd.img-"+release) {
			plan.RecoveryCandidates = append(plan.RecoveryCandidates, release)
		}
	}
	slices.Sort(plan.RecoveryCandidates)
	providers := map[string]bool{}
	route, routeErr := readSmallRegular(os.DirFS("/"), "proc/net/route", 64<<10)
	for _, iface := range defaultIPv4Interfaces(route) {
		deviceType, typeErr := os.ReadFile(filepath.Join("/sys/class/net", iface, "type"))
		_, wirelessErr := os.Stat(filepath.Join("/sys/class/net", iface, "wireless"))
		_, phyErr := os.Stat(filepath.Join("/sys/class/net", iface, "phy80211"))
		if typeErr != nil || strings.TrimSpace(string(deviceType)) != "1" || wirelessErr == nil || phyErr == nil {
			continue
		}
		path, err := os.Readlink(filepath.Join("/sys/class/net", iface, "device/driver/module"))
		if err != nil {
			continue
		}
		module := filepath.Base(path)
		if nativeName.MatchString(module) {
			plan.WiredInterfaces = append(plan.WiredInterfaces, iface)
			if plan.WiredDrivers == nil {
				plan.WiredDrivers = map[string]string{}
			}
			plan.WiredDrivers[iface] = module
			providers[module] = true
		}
	}
	for _, module := range inventory.Kernel.LoadedModules {
		if module == "nvidia" || strings.HasPrefix(module, "nvidia_") {
			plan.NVIDIALoadedModules = append(plan.NVIDIALoadedModules, module)
			providers[module] = true
		}
	}
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		plan.Preserved = append(plan.Preserved, observeProvider(ctx, plan.KernelRelease, name))
	}
	routeIdentity := FileIdentity{Path: "/proc/net/route"}
	if routeErr == nil {
		routeIdentity.SHA256 = digest(route)
	} else {
		routeIdentity.Issue = routeErr.Error()
	}
	plan.Preserved = append(plan.Preserved, routeIdentity)
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func observeProvider(ctx context.Context, abi, module string) FileIdentity {
	result := FileIdentity{Path: module}
	if !nativeName.MatchString(module) || abi != nobleABI {
		result.Issue = "invalid module lookup"
		return result
	}
	out, err := exec.CommandContext(ctx, "/usr/sbin/modinfo", "-k", abi, "-F", "filename", module).Output()
	if err != nil {
		result.Issue = "module provider unavailable: " + err.Error()
		return result
	}
	path := strings.TrimSpace(string(out))
	if filepath.Clean(path) != path || !strings.HasPrefix(path, "/lib/modules/"+abi+"/") {
		result.Issue = "module provider is outside the expected ABI tree"
		return result
	}
	return observeFile(ctx, path)
}

func observeFile(ctx context.Context, path string) FileIdentity {
	identity := FileIdentity{Path: path}
	info, err := os.Lstat(path)
	if err == nil && (!info.Mode().IsRegular() || info.Size() > maxPreservedBytes) {
		err = fmt.Errorf("file type or size is outside the reviewed bounds")
	}
	if err == nil {
		identity.SHA256, err = binaryfetch.FileDigestContext(ctx, path)
	}
	if err != nil {
		identity.Issue = err.Error()
	}
	return identity
}

func defaultIPv4Interfaces(data []byte) []string {
	var interfaces []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) <= 7 || fields[1] != "00000000" || fields[7] != "00000000" || !nativeName.MatchString(fields[0]) {
			continue
		}
		flags, err := strconv.ParseUint(fields[3], 16, 32)
		if err == nil && flags&1 != 0 {
			interfaces = append(interfaces, fields[0])
		}
	}
	slices.Sort(interfaces)
	return slices.Compact(interfaces)
}
