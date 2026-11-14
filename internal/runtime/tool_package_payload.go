package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	platform "github.com/vrooli/platform-go"
)

var packagePayloadCommand = func(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env = dir, env
	cmd.WaitDelay = 5 * time.Second
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return nil, err
	}
	cmd.Cancel = func() error { return platform.SignalProcessGroup(cmd.Process.Pid, true) }
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil && cmd.Process != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if reapErr := platform.WaitProcessGroupContext(cleanup, cmd.Process.Pid, 0); reapErr != nil {
			return out, fmt.Errorf("package process cleanup: %w", reapErr)
		}
	}
	return out, err
}

var debianPackageName = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]+$`)

func (h toolHandler) userLocalAPT(host hostreqkit.Host) (hostreqkit.UserLocalAPT, bool) {
	payload, ok := h.manifest.UserLocalAPT[runtimeArch()]
	return payload, ok && host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get")
}

func validatePackagePayload(payload hostreqkit.UserLocalAPT) error {
	if len(payload.Packages) == 0 || len(payload.Packages) > 32 {
		return fmt.Errorf("user-local APT payload requires 1–32 declared packages")
	}
	for _, name := range payload.Packages {
		if !debianPackageName.MatchString(name) {
			return fmt.Errorf("invalid declared Debian package %q", name)
		}
	}
	for _, path := range []string{payload.Executable, payload.LibraryDir} {
		if _, err := managedToolEntryPath("/payload", path); err != nil {
			return err
		}
	}
	_, err := resolveLauncherRuntimeEnv("/payload", payload.RuntimeEnv)
	return err
}

// applyUserLocalAPT uses the same managed artifact/launcher owner as release
// tools. A verified, working payload is published under a content identity;
// replacing the launcher does not invalidate a process using the old payload.
func (h toolHandler) applyUserLocalAPT(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions, payload hostreqkit.UserLocalAPT) (hostreqkit.ItemStatus, error) {
	fail := func(err error) (hostreqkit.ItemStatus, error) {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if err := validatePackagePayload(payload); err != nil {
		return fail(err)
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		status.Notes = append(status.Notes, "extract authenticated APT package data into the managed user tool directory; no sudo, package scripts, or system package database changes")
		return status, nil
	}
	release, err := acquireToolInstallLock(h.manifest.Name)
	if err != nil {
		return fail(err)
	}
	defer release()
	latest := h.Inspect(host, hostreqspec.ResolvedRequirement{Name: status.Name, Kind: status.Kind, Required: status.Required})
	if latest.Installed {
		return latest, nil
	}
	optDir, err := userLocalOptDir(h.manifest.Name)
	if err != nil {
		return fail(err)
	}
	if err = os.MkdirAll(optDir, 0755); err != nil {
		return fail(err)
	}
	stage, err := os.MkdirTemp(optDir, ".package-")
	if err != nil {
		return fail(err)
	}
	defer os.RemoveAll(stage)
	parent := opts.Context
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	args := append([]string{"-o", "APT::Get::AllowUnauthenticated=false", "-o", "Acquire::AllowInsecureRepositories=false", "download", "--"}, payload.Packages...)
	if out, err := packagePayloadCommand(ctx, stage, nil, "apt-get", args...); err != nil {
		return fail(fmt.Errorf("download declared package payloads: %w: %s", err, strings.TrimSpace(string(out))))
	}
	archives, err := filepath.Glob(filepath.Join(stage, "*.deb"))
	if err != nil || len(archives) != len(payload.Packages) {
		return fail(fmt.Errorf("expected %d package archives, received %d: %v", len(payload.Packages), len(archives), err))
	}
	tree := filepath.Join(stage, "tree")
	if err := os.Mkdir(tree, 0755); err != nil {
		return fail(err)
	}
	seen := map[string]bool{}
	receipts := []map[string]string{}
	identity := sha256.New()
	for _, archive := range archives {
		out, err := packagePayloadCommand(ctx, stage, nil, "dpkg-deb", "--show", "--showformat=${Package} ${Version} ${Architecture}", archive)
		fields := strings.Fields(string(out))
		if err != nil || len(fields) != 3 || !containsPackage(payload.Packages, fields[0]) || seen[fields[0]] || fields[2] != runtimeArch() && fields[2] != "all" {
			return fail(fmt.Errorf("unexpected package identity %q: %v", out, err))
		}
		seen[fields[0]] = true
		raw, err := os.ReadFile(archive)
		if err != nil {
			return fail(err)
		}
		digest := sha256.Sum256(raw)
		identity.Write(digest[:])
		receipts = append(receipts, map[string]string{"package": fields[0], "version": fields[1], "architecture": fields[2], "sha256": hex.EncodeToString(digest[:])})
		if out, err := packagePayloadCommand(ctx, stage, nil, "dpkg-deb", "--extract", archive, tree); err != nil {
			return fail(fmt.Errorf("extract package payload: %w: %s", err, out))
		}
	}
	runtimeEnv := make(map[string]string, len(payload.RuntimeEnv)+1)
	for key, value := range payload.RuntimeEnv {
		runtimeEnv[key] = value
	}
	runtimeEnv["LD_LIBRARY_PATH"] = payload.LibraryDir
	resolved, err := resolveLauncherRuntimeEnv(tree, runtimeEnv)
	if err != nil {
		return fail(err)
	}
	env := os.Environ()
	for key, value := range resolved {
		env = append(env, key+"="+value)
	}
	entry := filepath.Join(tree, payload.Executable)
	out, err := packagePayloadCommand(ctx, tree, env, entry, h.manifest.VersionArgs...)
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return fail(fmt.Errorf("package runtime qualification failed: %w: %s", err, out))
	}
	version := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
	status.Version = version
	if !h.versionSatisfied(&status, status.MinVersion) {
		return status, nil
	}
	receipt, _ := json.MarshalIndent(map[string]any{"packages": receipts, "version": version, "installed_at": time.Now().UTC()}, "", "  ")
	if err := os.WriteFile(filepath.Join(tree, "package-receipt.json"), receipt, 0644); err != nil {
		return fail(err)
	}
	final := filepath.Join(optDir, hex.EncodeToString(identity.Sum(nil)))
	if _, err := os.Stat(final); os.IsNotExist(err) {
		if err := os.Rename(tree, final); err != nil {
			return fail(err)
		}
	} else if err != nil {
		return fail(err)
	}
	binDir, err := userLocalBinDir()
	if err != nil {
		return fail(err)
	}
	command := h.manifest.Commands[0]
	if err := writeLauncher(binDir, command, final, filepath.Join(final, payload.Executable), runtimeEnv); err != nil {
		return fail(err)
	}
	if err := h.recordFetchedArtifacts(binDir, command, true); err != nil {
		return fail(err)
	}
	status.Command, status.Installed, status.ExecutionState = command, true, hostreqkit.ExecutionInstalled
	status.Notes = append(status.Notes, "user-local package receipt: "+filepath.Join(final, "package-receipt.json"))
	return status, nil
}

func containsPackage(packages []string, name string) bool {
	for _, candidate := range packages {
		if candidate == name {
			return true
		}
	}
	return false
}
