package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/projectstate"
)

const (
	filesystemParameterA = 100_000
)

const (
	filesystemParameterB = 4
)

func ensureProjectFilesystem(root, home string) error {
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		return err
	}
	// Repo-project paths (under the repo root, covered by layout.project_config_dir).
	for _, path := range []string{
		filepath.Join(root, "data"),
		filepath.Join(config.RepoConfigDir(root), "build"),
	} {
		if err := os.MkdirAll(path, tuning.PermDir); err != nil {
			return err
		}
	}
	// Operator-home paths: resolve names from the runtime_home authority and
	// route every create through the owned-write seam so a sudo'd setup never
	// leaves root-owned dirs in the operator's home.
	homeDirs := make([]string, 0, filesystemParameterB)
	for _, key := range []string{repocontract.HomeKeyBin, repocontract.HomeKeyLogs, repocontract.HomeKeyProcesses} {
		dir, err := repocontract.RuntimeHomeEntryPath(home, key)
		if err != nil {
			return err
		}
		homeDirs = append(homeDirs, dir)
	}
	homeDirs = append(homeDirs, locator.SetupStateDir())
	for _, dir := range homeDirs {
		if _, err := config.EnsureOwnedDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func ensureProjectFilesystemWithRecovery(root, home string) error {
	if err := ensureProjectFilesystem(root, home); err == nil {
		return nil
	} else {
		path, ok := permissionErrorPath(err)
		if !ok {
			return err
		}
		class, managedRoot, target, ok := managedRepairTarget(home, path)
		if !ok {
			return err
		}
		_, _, elevated := hostreqkit.InvokingUserIDs()
		if !elevated {
			return err
		}
		uid, gid := config.RepairIdentity()
		service := config.RepairService{ResolveRoot: func(string) (string, error) { return managedRoot, nil }}
		result, repairErr := service.Repair(context.Background(), config.RepairRequest{
			Scope:       config.RepairScope{RootClass: class, RootPath: target},
			ExpectedUID: uid,
			ExpectedGID: gid,
			Apply:       true,
			MaxEntries:  filesystemParameterA,
			Deadline:    time.Now().Add(tuning.StandardOperationTimeout),
		})
		if repairErr != nil || result.Failed > 0 || result.Status == config.RepairPartial {
			if repairErr == nil {
				repairErr = fmt.Errorf("repair status=%s failed=%d", result.Status, result.Failed)
			}
			return fmt.Errorf("filesystem permission failure at %s; targeted repair failed: %v; original: %w", target, repairErr, err)
		}
		if retryErr := ensureProjectFilesystem(root, home); retryErr != nil {
			return fmt.Errorf("filesystem permission failure at %s; targeted repair succeeded but retry failed: %w", target, errors.Join(err, retryErr))
		}
		return nil
	}
}

func permissionErrorPath(err error) (string, bool) {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && (errors.Is(pathErr.Err, os.ErrPermission) || strings.Contains(strings.ToLower(pathErr.Error()), "permission")) {
		return pathErr.Path, true
	}
	if strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		return "", false
	}
	return "", false
}

func managedRepairTarget(home, failedPath string) (class, root, target string, ok bool) {
	failedPath = filepath.Clean(failedPath)
	for _, class = range ownershipMigrationClasses {
		candidate, err := repocontract.RuntimeHomeEntryPath(home, class)
		if err != nil || !withinPath(failedPath, candidate) {
			continue
		}
		target = failedPath
		if _, err := os.Lstat(target); err != nil {
			target = filepath.Dir(target)
		}
		return class, candidate, target, true
	}
	return "", "", "", false
}

func withinPath(path, root string) bool {
	path, root = filepath.Clean(path), filepath.Clean(root)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

func configureGit(root string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return nil
	}
	cmd := exec.Command("git", "config", "core.filemode", "false")
	cmd.Dir = root
	return cmd.Run()
}
