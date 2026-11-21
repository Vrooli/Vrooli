//go:build linux

package hostwireless

// This owner observes cold, full request_firmware filesystem lookup, not a
// previously cached/in-flight request or device-resident image. The device owner
// must establish a fresh request boundary separately. Algorithm provenance:
// https://github.com/torvalds/linux/blob/v7.0/drivers/base/firmware_loader/main.c
// (_request_firmware_prepare, fw_get_filesystem_firmware, _request_firmware).
// Initramfs format: https://www.kernel.org/doc/html/latest/driver-api/early-userspace/buffer-format.html
// Ubuntu's lsinitramfs(8) documents incomplete multi-segment support; we parse
// bounded supported formats directly instead of treating a partial listing as
// absence: https://manpages.ubuntu.com/manpages/noble/man8/lsinitramfs.8.html

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"syscall"
)

const firmwarePolicyImageLimit = 64 << 20
const firmwarePolicyInitramfsLimit = 256 << 20

// InspectFirmwarePolicy admits only staging whose next cold firmware request
// selects the exact bytes through the existing raw lib/firmware destination.
// Foreign higher-precedence raw files and built-ins are measured refusals; this
// owner never overwrites override paths or infers built-in contents from config.
// An affected existing initramfs requires a separate exact backup/publication/
// rollback owner; no regeneration is attempted speculatively.
func InspectFirmwarePolicy(ctx context.Context, req FirmwarePolicyRequest) (FirmwarePolicyEvidence, error) {
	e := FirmwarePolicyEvidence{KernelRelease: req.KernelRelease}
	if req.Root == nil || !firmwarePolicyComponent(req.KernelRelease) || len(req.Firmware) == 0 || len(req.Firmware) > 16 || len(req.Modules) == 0 || len(req.Modules) > 32 {
		return e, errors.New("invalid firmware policy request")
	}
	seen := map[string]bool{}
	for _, t := range req.Firmware {
		if !firmwarePolicyPath(t.Name) || !validSHA(t.SHA256) || seen[t.Name] {
			return e, errors.New("invalid or duplicate firmware target")
		}
		seen[t.Name] = true
	}
	for _, m := range req.Modules {
		if !firmwarePolicyComponent(m) || strings.Contains(m, ".") {
			return e, errors.New("invalid module name")
		}
	}
	config, err := firmwarePolicyRead(ctx, req.Root, "boot/config-"+req.KernelRelease, 2<<20)
	if err != nil {
		return e, fmt.Errorf("kernel config prerequisite: %w", err)
	}
	cfg, err := firmwarePolicyConfig(config)
	if err != nil {
		return e, err
	}
	e.ConfigSHA256 = digest(config)
	if cfg["CONFIG_FW_LOADER"] != "y" && cfg["CONFIG_FW_LOADER"] != "m" {
		return e, errors.New("firmware loader configuration is not established")
	}
	extra, ok := cfg["CONFIG_EXTRA_FIRMWARE"]
	if !ok || len(extra) < 2 || extra[0] != '"' || extra[len(extra)-1] != '"' {
		return e, errors.New("built-in firmware configuration is not established")
	}
	builtins := strings.Fields(extra[1 : len(extra)-1])
	for _, n := range builtins {
		if !firmwarePolicyPath(n) {
			return e, errors.New("unsupported built-in firmware name")
		}
	}
	custom, err := firmwarePolicyReadParameter(ctx, req.Root)
	if err != nil {
		return e, fmt.Errorf("live firmware_class.path prerequisite: %w", err)
	}
	// Kernel truncates this parameter at its first newline, not arbitrary spaces.
	e.ParameterSHA256 = digest(custom)
	e.CustomPath = strings.SplitN(string(custom), "\n", 2)[0]
	dirs := []string{"lib/firmware/updates/" + req.KernelRelease, "lib/firmware/updates", "lib/firmware/" + req.KernelRelease, "lib/firmware"}
	if e.CustomPath != "" {
		if !strings.HasPrefix(e.CustomPath, "/") || !firmwarePolicyPath(strings.TrimPrefix(e.CustomPath, "/")) {
			return e, errors.New("unsupported custom firmware path")
		}
		dirs = append([]string{strings.TrimPrefix(e.CustomPath, "/")}, dirs...)
	}
	suffixes := []string{""}
	if cfg["CONFIG_FW_LOADER_COMPRESS_ZSTD"] == "y" {
		suffixes = append(suffixes, ".zst")
	}
	if cfg["CONFIG_FW_LOADER_COMPRESS_XZ"] == "y" {
		suffixes = append(suffixes, ".xz")
	}
	for _, target := range req.Firmware {
		s := FirmwarePolicySelection{Name: target.Name, StagePath: path.Join("lib/firmware", target.Name)}
		if cfg["CONFIG_FW_LOADER"] == "y" && slices.Contains(builtins, target.Name) {
			s.Builtin = true
			e.Reasons = append(e.Reasons, "built-in firmware preempts filesystem: "+target.Name)
		} else {
			// Compression fallback is a second traversal of ALL directories, not a
			// per-directory choice. Any uncertainty/error fails closed rather than
			// pretending a potentially shadowing source is absent.
			selected := false
			for _, suffix := range suffixes {
				for _, dir := range dirs {
					name := path.Join(dir, target.Name) + suffix
					data, readErr := firmwarePolicyRead(ctx, req.Root, name, firmwarePolicyImageLimit)
					if errors.Is(readErr, os.ErrNotExist) {
						continue
					}
					if readErr != nil {
						return e, fmt.Errorf("firmware lookup %s: %w", name, readErr)
					}
					decoded, decodeErr := firmwarePolicyDecode(ctx, data, suffix, firmwarePolicyImageLimit)
					if decodeErr != nil {
						return e, fmt.Errorf("firmware lookup %s: %w", name, decodeErr)
					}
					if len(decoded) == 0 {
						return e, fmt.Errorf("empty firmware at %s", name)
					}
					s.Path = name
					s.Encoding = map[string]string{"": "raw", ".zst": "zstd", ".xz": "xz"}[suffix]
					s.SHA256 = digest(decoded)
					s.StoredSHA256 = digest(data)
					selected = true
					break
				}
				if selected {
					break
				}
			}
			s.StageRequired = s.SHA256 != target.SHA256
			if s.StageRequired && s.Encoding == "raw" && s.Path != s.StagePath {
				e.Reasons = append(e.Reasons, "higher-precedence raw firmware requires separate transaction: "+s.Path)
			}
		}
		e.Selections = append(e.Selections, s)
	}
	builtinModules, err := firmwarePolicyRead(ctx, req.Root, "lib/modules/"+req.KernelRelease+"/modules.builtin", 8<<20)
	if err != nil {
		return e, fmt.Errorf("built-in module inventory prerequisite: %w", err)
	}
	e.BuiltinModulesSHA256 = digest(builtinModules)
	for _, line := range strings.Split(string(builtinModules), "\n") {
		if firmwarePolicyModuleMatch(line, req.Modules) {
			e.BuiltinModules = append(e.BuiltinModules, line)
		}
	}
	if len(e.BuiltinModules) > 0 {
		e.Reasons = append(e.Reasons, "affected module is built into kernel")
	}
	initrd, err := firmwarePolicyRead(ctx, req.Root, "boot/initrd.img-"+req.KernelRelease, firmwarePolicyInitramfsLimit)
	if err != nil {
		return e, fmt.Errorf("initramfs prerequisite: %w", err)
	}
	e.InitramfsSHA256 = digest(initrd)
	entries, err := firmwarePolicyArchive(ctx, initrd)
	if err != nil {
		return e, fmt.Errorf("initramfs inventory prerequisite: %w", err)
	}
	for _, name := range entries {
		affected := firmwarePolicyModuleMatch(name, req.Modules)
		for _, target := range req.Firmware {
			// Conservatively include relocated copies and symlink targets:
			// archive directory links may change every component but basename.
			for _, suffix := range []string{"", ".xz", ".zst"} {
				if path.Base(name) == path.Base(target.Name)+suffix {
					affected = true
				}
			}
		}
		if affected {
			e.InitramfsAffected = append(e.InitramfsAffected, name)
		}
	}
	slices.Sort(e.InitramfsAffected)
	if len(e.InitramfsAffected) > 0 {
		e.Reasons = append(e.Reasons, "affected initramfs entries require a separately qualified exact initramfs transaction")
	}
	e.StageAllowed = len(e.Reasons) == 0
	return e, ctx.Err()
}

func firmwarePolicyComponent(s string) bool {
	return s != "" && len(s) <= 255 && !strings.ContainsAny(s, "/\\\x00\n\r\t ") && s != "." && s != ".." && !strings.HasPrefix(s, "-")
}
func firmwarePolicyPath(s string) bool {
	if len(s) > 2048 || !firmwarePolicyComponent(path.Base(s)) || strings.HasPrefix(s, "/") || path.Clean(s) != s {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if !firmwarePolicyComponent(p) {
			return false
		}
	}
	return true
}
func firmwarePolicyModuleMatch(name string, modules []string) bool {
	base := path.Base(name)
	for _, s := range []string{".zst", ".xz", ".gz"} {
		base = strings.TrimSuffix(base, s)
	}
	if !strings.HasSuffix(base, ".ko") {
		return false
	}
	base = strings.ReplaceAll(strings.TrimSuffix(base, ".ko"), "-", "_")
	for _, m := range modules {
		if base == strings.ReplaceAll(m, "-", "_") {
			return true
		}
	}
	return false
}
func firmwarePolicyRead(ctx context.Context, root *os.Root, name string, limit int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() < 0 || st.Size() > limit {
		return nil, errors.New("not a bounded regular file")
	}
	b, err := io.ReadAll(io.LimitReader(firmwarePolicyContextReader{ctx, f}, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) != st.Size() {
		return nil, errors.New("file size changed or exceeded limit")
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if after.Size() != st.Size() || !after.ModTime().Equal(st.ModTime()) {
		return nil, errors.New("file changed during read")
	}
	return b, ctx.Err()
}
func firmwarePolicyConfig(data []byte) (map[string]string, error) {
	cfg := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "CONFIG_") {
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				return nil, errors.New("malformed kernel configuration")
			}
			if _, exists := cfg[k]; exists {
				return nil, errors.New("duplicate kernel configuration")
			}
			cfg[k] = v
		}
	}
	return cfg, nil
}

// sysfs regular attributes advertise PAGE_SIZE rather than their textual size.
// Read the bounded exact parameter twice rather than applying disk-size rules.
func firmwarePolicyReadParameter(ctx context.Context, root *os.Root) ([]byte, error) {
	var previous []byte
	for i := 0; i < 2; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		f, err := root.OpenFile("sys/module/firmware_class/parameters/path", os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return nil, err
		}
		st, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, err
		}
		if !st.Mode().IsRegular() {
			f.Close()
			return nil, errors.New("firmware parameter is not a regular attribute")
		}
		b, err := io.ReadAll(io.LimitReader(firmwarePolicyContextReader{ctx, f}, 257))
		f.Close()
		if err != nil {
			return nil, err
		}
		if len(b) > 256 {
			return nil, errors.New("firmware parameter exceeds bound")
		}
		if i == 1 && string(b) != string(previous) {
			return nil, errors.New("firmware parameter changed during observation")
		}
		previous = b
	}
	return previous, nil
}
