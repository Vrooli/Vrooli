//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func nativePrivateIndexes(ctx context.Context, copied *CopiedActivation, plan ActivationPlan) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if copied == nil || copied.root == nil || plan.KernelRelease != nobleABI {
		return errors.New("native indexing requires an owned private root and reviewed ABI")
	}
	root := copied.root.Name()
	if err := verifyFile(ctx, copied.root.FS(), "etc/depmod.d/ubuntu.conf", preservedSHA(plan, "/etc/depmod.d/ubuntu.conf")); err != nil {
		return err
	}
	symbols := abiTree + "/build/Module.symvers"
	if err := verifyFile(ctx, copied.root.FS(), symbols, preservedSHA(plan, "/"+symbols)); err != nil {
		return err
	}
	cmd := privateKmod(ctx, "/usr/sbin/depmod", "-b", root, "-C", filepath.Join(root, "etc/depmod.d/ubuntu.conf"),
		"-a", "-e", "-E", filepath.Join(root, symbols), nobleABI)
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) != 0 {
		return fmt.Errorf("private depmod did not produce clean indexes: %w", errors.Join(err, errors.New(strings.TrimSpace(string(out)))))
	}
	for _, file := range plan.Candidates {
		module := strings.TrimSuffix(file.Path, ".ko")
		if err := requirePrivateProvider(ctx, root, module, filepath.Join(root, plan.OverlayDirectory, file.Path)); err != nil {
			return err
		}
	}
	providers := map[string]bool{}
	for _, module := range plan.NVIDIALoadedModules {
		providers[module] = true
	}
	for _, module := range plan.WiredDrivers {
		providers[module] = true
	}
	for module := range providers {
		var expected string
		for _, file := range plan.Preserved {
			base := strings.ReplaceAll(filepath.Base(file.Path), "_", "-")
			name := strings.ReplaceAll(module, "_", "-") + ".ko"
			if base == name || base == name+".zst" || base == name+".xz" || base == name+".gz" {
				expected = filepath.Join(root, strings.TrimPrefix(file.Path, "/"))
				break
			}
		}
		if expected == "" {
			return fmt.Errorf("preserved provider missing for %s", module)
		}
		if err := requirePrivateProvider(ctx, root, module, expected); err != nil {
			return err
		}
	}
	selected := "mt7925e"
	if plan.Profile == NobleBluetoothProfile {
		selected = "btusb"
	}
	for _, alias := range plan.Modaliases {
		if len(alias) > 256 || strings.ContainsAny(alias, "\x00\r\n") || (!strings.HasPrefix(alias, "pci:") && !strings.HasPrefix(alias, "usb:")) {
			return errors.New("invalid observed alias for private indexing")
		}
		out, err := privateKmod(ctx, "/usr/sbin/modprobe", "-C", "/dev/null", "-d", root, "-S", nobleABI, "--resolve-alias", alias).Output()
		if err != nil || strings.TrimSpace(string(out)) != selected {
			return errors.New("private module index does not resolve the observed device to the candidate")
		}
		out, err = privateKmod(ctx, "/usr/sbin/modprobe", "-C", "/dev/null", "-d", root, "-S", nobleABI, "--show-depends", alias).Output()
		if err != nil {
			return err
		}
		for _, module := range plan.Candidates {
			if !strings.Contains(string(out), "insmod "+filepath.Join(root, plan.OverlayDirectory, module.Path)) {
				return fmt.Errorf("private dependency selection omits signed %s", module.Path)
			}
		}
	}
	return ctx.Err()
}

func requirePrivateProvider(ctx context.Context, root, module, expected string) error {
	if !nativeName.MatchString(module) {
		return errors.New("invalid private module lookup")
	}
	out, err := privateKmod(ctx, "/usr/sbin/modinfo", "-b", root, "-k", nobleABI, "-F", "filename", module).Output()
	if err != nil || strings.TrimSpace(string(out)) != expected {
		return fmt.Errorf("private provider selection differs for %s", module)
	}
	return nil
}

func privateKmod(ctx context.Context, executable string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LC_ALL=C"}
	cmd.WaitDelay = time.Second
	return cmd
}
