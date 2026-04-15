package maintenance

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/vrooli/vrooli/internal/resources"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

type UserStorageAction struct {
	Kind    string `json:"kind"`
	Name    string `json:"name,omitempty"`
	Path    string `json:"path,omitempty"`
	Source  string `json:"source,omitempty"`
	Dest    string `json:"dest,omitempty"`
	Message string `json:"message,omitempty"`
}

type UserStorageReport struct {
	Actions []UserStorageAction `json:"actions"`
}

type legacyMove struct {
	Name                   string
	Src                    string
	Dst                    string
	ReplaceStubDestination bool
}

type moveResult struct {
	Moved          bool
	Blocked        bool
	Conflict       bool
	ConflictReason string
}

type resourceNormalization struct {
	Resource   string
	Moves      []legacyMove
	CleanupDir string
}

var (
	resourceRunningFn = func(root, home, name string) (bool, error) {
		status, err := resources.NewController(root, home).Status(name, true)
		if err != nil {
			return false, err
		}
		return status.Running, nil
	}
	runResourceActionFn = func(root, home, name, action string) error {
		return resources.NewController(root, home).Run(name, []string{action}, io.Discard, io.Discard)
	}
)

func (c *Controller) CleanupUserStorage() (report UserStorageReport, err error) {
	report = UserStorageReport{Actions: make([]UserStorageAction, 0, 128)}
	legacyRoot := filepath.Join(c.Home, ".vrooli")

	normalizations, err := c.resourceNormalizations(legacyRoot)
	if err != nil {
		return report, err
	}

	active := make([]resourceNormalization, 0, len(normalizations))
	for _, item := range normalizations {
		if hasPendingMoves(item.Moves) {
			active = append(active, item)
		}
	}

	restart := make([]string, 0, len(active))
	var restartErr error
	defer func() {
		for _, name := range restart {
			if err := runResourceActionFn(c.Root, c.Home, name, "start"); err != nil {
				if restartErr == nil {
					restartErr = fmt.Errorf("restart resource %s: %w", name, err)
				}
				report.Actions = append(report.Actions, UserStorageAction{
					Kind:    "restart_failed",
					Name:    name,
					Message: err.Error(),
				})
				continue
			}
			report.Actions = append(report.Actions, UserStorageAction{
				Kind:    "started_resource",
				Name:    name,
				Message: "restarted resource after storage migration",
			})
		}
		sort.SliceStable(report.Actions, func(i, j int) bool {
			left := report.Actions[i].Kind + "|" + report.Actions[i].Name + "|" + report.Actions[i].Path + "|" + report.Actions[i].Source
			right := report.Actions[j].Kind + "|" + report.Actions[j].Name + "|" + report.Actions[j].Path + "|" + report.Actions[j].Source
			return left < right
		})
		if err == nil && restartErr != nil {
			err = restartErr
		}
	}()

	for _, item := range active {
		running, err := resourceRunningFn(c.Root, c.Home, item.Resource)
		if err != nil {
			return report, fmt.Errorf("inspect resource %s: %w", item.Resource, err)
		}
		if !running {
			continue
		}
		if err := runResourceActionFn(c.Root, c.Home, item.Resource, "stop"); err != nil {
			return report, fmt.Errorf("stop resource %s: %w", item.Resource, err)
		}
		restart = append(restart, item.Resource)
		report.Actions = append(report.Actions, UserStorageAction{
			Kind:    "stopped_resource",
			Name:    item.Resource,
			Message: "stopped running resource before storage migration",
		})
	}

	for _, item := range active {
		for _, move := range item.Moves {
			if blocked, err := applyLegacyMove(&report, move); err != nil {
				return report, err
			} else if blocked {
				continue
			}
		}
		if removed, blocked, err := removeDirIfEmpty(item.CleanupDir); err != nil {
			return report, err
		} else if blocked {
			report.Actions = append(report.Actions, UserStorageAction{
				Kind:    "blocked",
				Path:    item.CleanupDir,
				Message: "legacy resource root could not be removed because it is not writable by the current user",
			})
		} else if removed {
			report.Actions = append(report.Actions, UserStorageAction{
				Kind:    "removed",
				Path:    item.CleanupDir,
				Message: "removed empty legacy resource root",
			})
		}
	}

	for _, path := range legacyCleanupPaths(legacyRoot) {
		removed, blocked, err := removePathIfExists(path)
		if err != nil {
			return report, err
		}
		if blocked {
			report.Actions = append(report.Actions, UserStorageAction{
				Kind:    "blocked",
				Path:    path,
				Message: "legacy path could not be removed because it is not writable by the current user",
			})
			continue
		}
		if !removed {
			continue
		}
		report.Actions = append(report.Actions, UserStorageAction{
			Kind:    "removed",
			Path:    path,
			Message: "removed known-safe legacy path",
		})
	}

	return report, nil
}

func (c *Controller) resourceNormalizations(legacyRoot string) ([]resourceNormalization, error) {
	resolverCfg := runtimestorage.ResolverConfig{AppID: "vrooli"}
	if c.Home != "" && c.Home != "." {
		home := c.Home
		resolverCfg.UserHomeDir = func() (string, error) { return home, nil }
		resolverCfg.UserConfigDir = func() (string, error) { return filepath.Join(home, ".config"), nil }
		resolverCfg.UserCacheDir = func() (string, error) { return filepath.Join(home, ".cache"), nil }
	}
	resolver, err := runtimestorage.NewResolver(resolverCfg)
	if err != nil {
		return nil, err
	}

	resolve := func(name string) (runtimestorage.Paths, error) {
		return resolver.Resolve(runtimestorage.Options{ResourceID: name})
	}

	browserlessPaths, err := resolve("browserless")
	if err != nil {
		return nil, err
	}
	minioPaths, err := resolve("minio")
	if err != nil {
		return nil, err
	}
	postgisPaths, err := resolve("postgis")
	if err != nil {
		return nil, err
	}
	postgresPaths, err := resolve("postgres")
	if err != nil {
		return nil, err
	}
	redisPaths, err := resolve("redis")
	if err != nil {
		return nil, err
	}
	twilioPaths, err := resolve("twilio")
	if err != nil {
		return nil, err
	}

	return []resourceNormalization{
		{
			Resource: "browserless",
			Moves: []legacyMove{
				{
					Name:                   "browserless-home-root",
					Src:                    filepath.Join(legacyRoot, "browserless"),
					Dst:                    browserlessPaths.DataDir,
					ReplaceStubDestination: true,
				},
			},
			CleanupDir: filepath.Join(legacyRoot, "browserless"),
		},
		{
			Resource: "minio",
			Moves: []legacyMove{
				{
					Name: "minio-data",
					Src:  filepath.Join(legacyRoot, "minio", "data"),
					Dst:  minioPaths.DataDir,
				},
			},
			CleanupDir: filepath.Join(legacyRoot, "minio"),
		},
		{
			Resource: "postgis",
			Moves: []legacyMove{
				{Name: "postgis-data", Src: filepath.Join(legacyRoot, "postgis", "data"), Dst: filepath.Join(postgisPaths.DataDir, "data")},
				{Name: "postgis-import", Src: filepath.Join(legacyRoot, "postgis", "import"), Dst: filepath.Join(postgisPaths.DataDir, "import")},
				{Name: "postgis-export", Src: filepath.Join(legacyRoot, "postgis", "export"), Dst: filepath.Join(postgisPaths.DataDir, "export")},
				{Name: "postgis-sql", Src: filepath.Join(legacyRoot, "postgis", "sql"), Dst: filepath.Join(postgisPaths.DataDir, "sql")},
				{Name: "postgis-health-pid", Src: filepath.Join(legacyRoot, "postgis", "health_server.pid"), Dst: filepath.Join(postgisPaths.StateDir, "health_server.pid")},
				{Name: "postgis-test-results", Src: filepath.Join(legacyRoot, "postgis", "test_results.json"), Dst: filepath.Join(postgisPaths.StateDir, "test_results.json")},
			},
			CleanupDir: filepath.Join(legacyRoot, "postgis"),
		},
		{
			Resource: "postgres",
			Moves: []legacyMove{
				{Name: "postgres-main", Src: filepath.Join(legacyRoot, "postgres", "main"), Dst: filepath.Join(postgresPaths.DataDir, "instances", "main")},
				{Name: "postgres-instances", Src: filepath.Join(legacyRoot, "postgres", "instances"), Dst: filepath.Join(postgresPaths.DataDir, "instances")},
				{Name: "postgres-backups", Src: filepath.Join(legacyRoot, "backups", "postgres"), Dst: filepath.Join(postgresPaths.StateDir, "backups")},
			},
			CleanupDir: filepath.Join(legacyRoot, "postgres"),
		},
		{
			Resource: "redis",
			Moves: []legacyMove{
				{Name: "redis-config", Src: filepath.Join(legacyRoot, "redis", "config"), Dst: redisPaths.ConfigDir},
				{Name: "redis-config-logrotate", Src: filepath.Join(legacyRoot, "redis", "logrotate.conf"), Dst: filepath.Join(redisPaths.ConfigDir, "logrotate.conf")},
				{Name: "redis-cli-wrapper", Src: filepath.Join(legacyRoot, "redis", "redis-cli"), Dst: filepath.Join(redisPaths.ConfigDir, "redis-cli")},
				{Name: "redis-data", Src: filepath.Join(legacyRoot, "redis", "data"), Dst: redisPaths.DataDir},
				{Name: "redis-logs", Src: filepath.Join(legacyRoot, "redis", "logs"), Dst: redisPaths.LogsDir},
				{Name: "redis-backups", Src: filepath.Join(legacyRoot, "redis", "backups"), Dst: filepath.Join(redisPaths.StateDir, "backups")},
			},
			CleanupDir: filepath.Join(legacyRoot, "redis"),
		},
		{
			Resource: "twilio",
			Moves: []legacyMove{
				{Name: "twilio-credentials", Src: filepath.Join(legacyRoot, "twilio-credentials.json"), Dst: filepath.Join(twilioPaths.ConfigDir, "credentials.json"), ReplaceStubDestination: true},
				{Name: "twilio-phone-numbers", Src: filepath.Join(legacyRoot, "twilio", "phone-numbers.json"), Dst: filepath.Join(twilioPaths.ConfigDir, "phone-numbers.json"), ReplaceStubDestination: true},
				{Name: "twilio-templates", Src: filepath.Join(legacyRoot, "twilio", "templates"), Dst: filepath.Join(twilioPaths.ConfigDir, "templates")},
				{Name: "twilio-twiml", Src: filepath.Join(legacyRoot, "twilio", "twiml"), Dst: filepath.Join(twilioPaths.ConfigDir, "twiml")},
				{Name: "twilio-workflows", Src: filepath.Join(legacyRoot, "twilio", "workflows"), Dst: filepath.Join(twilioPaths.ConfigDir, "workflows")},
				{Name: "twilio-cli-wrapper", Src: filepath.Join(legacyRoot, "twilio", "twilio-cli"), Dst: filepath.Join(twilioPaths.ConfigDir, "twilio-cli")},
				{Name: "twilio-voice-history", Src: filepath.Join(legacyRoot, "twilio", "voice_history.json"), Dst: filepath.Join(twilioPaths.DataDir, "voice_history.json")},
				{Name: "twilio-whatsapp-history", Src: filepath.Join(legacyRoot, "twilio", "whatsapp_history.json"), Dst: filepath.Join(twilioPaths.DataDir, "whatsapp_history.json")},
			},
			CleanupDir: filepath.Join(legacyRoot, "twilio"),
		},
	}, nil
}

func legacyCleanupPaths(legacyRoot string) []string {
	paths := []string{
		filepath.Join(legacyRoot, "comment-system"),
		filepath.Join(legacyRoot, "contact-book"),
		filepath.Join(legacyRoot, "crypto-tools"),
		filepath.Join(legacyRoot, "email-triage"),
		filepath.Join(legacyRoot, "quiz-generator"),
		filepath.Join(legacyRoot, "recommendation-engine"),
		filepath.Join(legacyRoot, "blender"),
		filepath.Join(legacyRoot, "esphome"),
		filepath.Join(legacyRoot, "freecad"),
		filepath.Join(legacyRoot, "geth"),
		filepath.Join(legacyRoot, "godot"),
		filepath.Join(legacyRoot, "gridlabd"),
		filepath.Join(legacyRoot, "haystack"),
		filepath.Join(legacyRoot, "keycloak"),
		filepath.Join(legacyRoot, "llamaindex"),
		filepath.Join(legacyRoot, "mifos"),
		filepath.Join(legacyRoot, "obs-studio"),
		filepath.Join(legacyRoot, "openfoam"),
		filepath.Join(legacyRoot, "pihole"),
		filepath.Join(legacyRoot, "ros2"),
		filepath.Join(legacyRoot, "segment-anything"),
		filepath.Join(legacyRoot, "su2"),
		filepath.Join(legacyRoot, "vnc"),
		filepath.Join(legacyRoot, "scenario-to-desktop"),
		filepath.Join(legacyRoot, "processes-backup-20250901-165155.tar.gz"),
	}

	for _, pattern := range []string{
		filepath.Join(legacyRoot, "resources.local.json.backup*"),
		filepath.Join(legacyRoot, "resources.local.json.invalid.*"),
		filepath.Join(legacyRoot, "resources.local.json.test-backup"),
		filepath.Join(legacyRoot, "secrets.json.pre-migration-*"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		paths = append(paths, matches...)
	}

	sort.Strings(paths)
	return paths
}

func hasPendingMoves(moves []legacyMove) bool {
	for _, move := range moves {
		if pathExists(move.Src) {
			return true
		}
	}
	return false
}

func applyLegacyMove(report *UserStorageReport, move legacyMove) (bool, error) {
	result, err := movePath(move.Src, move.Dst, move.ReplaceStubDestination)
	if err != nil {
		return false, fmt.Errorf("migrate %s from %s to %s: %w", move.Name, move.Src, move.Dst, err)
	}
	if result.Blocked {
		report.Actions = append(report.Actions, UserStorageAction{
			Kind:    "blocked",
			Name:    move.Name,
			Source:  move.Src,
			Dest:    move.Dst,
			Message: "legacy path could not be migrated because it is not writable by the current user",
		})
	}
	if result.Conflict {
		report.Actions = append(report.Actions, UserStorageAction{
			Kind:    "conflict",
			Name:    move.Name,
			Source:  move.Src,
			Dest:    move.Dst,
			Message: result.ConflictReason,
		})
	}
	if result.Moved {
		report.Actions = append(report.Actions, UserStorageAction{
			Kind:    "migrated",
			Name:    move.Name,
			Source:  move.Src,
			Dest:    move.Dst,
			Message: "migrated legacy path into canonical resource storage",
		})
	}
	return result.Blocked || result.Conflict, nil
}

func movePath(src, dst string, replaceStubDestination bool) (moveResult, error) {
	info, err := os.Lstat(src)
	if os.IsNotExist(err) {
		return moveResult{}, nil
	}
	if err != nil {
		return moveResult{}, err
	}

	if info.IsDir() {
		return moveDir(src, dst, replaceStubDestination)
	}
	return moveFile(src, dst, replaceStubDestination)
}

func moveDir(src, dst string, replaceStubDestination bool) (moveResult, error) {
	if src == dst {
		return moveResult{}, nil
	}
	if !pathExists(src) {
		return moveResult{}, nil
	}
	if !pathExists(dst) {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return moveResult{}, err
		}
		if err := os.Rename(src, dst); err == nil {
			return moveResult{Moved: true}, nil
		} else if isPermissionDenied(err) {
			return moveResult{Blocked: true}, nil
		}
		// Fall through to recursive merge when rename crosses devices or collides.
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		if isPermissionDenied(err) {
			return moveResult{Blocked: true}, nil
		}
		return moveResult{}, err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return moveResult{}, err
	}

	result := moveResult{}
	for _, entry := range entries {
		childSrc := filepath.Join(src, entry.Name())
		childDst := filepath.Join(dst, entry.Name())
		childResult, err := movePath(childSrc, childDst, replaceStubDestination)
		if err != nil {
			return result, err
		}
		result.Moved = result.Moved || childResult.Moved
		result.Blocked = result.Blocked || childResult.Blocked
		if childResult.Conflict {
			result.Conflict = true
			if result.ConflictReason == "" {
				result.ConflictReason = childResult.ConflictReason
			}
		}
	}
	if removed, blocked, err := removeDirIfEmpty(src); err != nil {
		return result, err
	} else {
		result.Moved = result.Moved || removed
		result.Blocked = result.Blocked || blocked
	}
	return result, nil
}

func moveFile(src, dst string, replaceStubDestination bool) (moveResult, error) {
	if src == dst {
		return moveResult{}, nil
	}
	if !pathExists(src) {
		return moveResult{}, nil
	}
	if pathExists(dst) {
		equal, err := filesEqual(src, dst)
		if err != nil {
			return moveResult{}, err
		}
		if equal {
			removed, blocked, err := removePathIfExists(src)
			if err != nil {
				return moveResult{}, err
			}
			return moveResult{Moved: removed, Blocked: blocked}, nil
		}
		if replaceStubDestination {
			stub, err := isStubFile(dst)
			if err != nil {
				return moveResult{}, err
			}
			if stub {
				removed, blocked, err := removePathIfExists(dst)
				if err != nil {
					return moveResult{}, err
				}
				if blocked {
					return moveResult{Blocked: true}, nil
				}
				if removed {
					return moveFile(src, dst, false)
				}
			}
		}
		return moveResult{
			Conflict:       true,
			ConflictReason: "legacy path differs from the canonical destination; leaving both in place for manual review",
		}, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return moveResult{}, err
	}
	if err := os.Rename(src, dst); err == nil {
		return moveResult{Moved: true}, nil
	} else if isPermissionDenied(err) {
		return moveResult{Blocked: true}, nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		if isPermissionDenied(err) {
			return moveResult{Blocked: true}, nil
		}
		return moveResult{}, err
	}
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(src); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return moveResult{}, err
	}
	if err := os.Remove(src); err != nil {
		if isPermissionDenied(err) {
			return moveResult{Moved: true, Blocked: true}, nil
		}
		return moveResult{}, err
	}
	return moveResult{Moved: true}, nil
}

func removePathIfExists(path string) (bool, bool, error) {
	if !pathExists(path) {
		return false, false, nil
	}
	if err := os.RemoveAll(path); err != nil {
		if isPermissionDenied(err) {
			return false, true, nil
		}
		return false, false, err
	}
	return true, false, nil
}

func removeDirIfEmpty(path string) (bool, bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		if isPermissionDenied(err) {
			return false, true, nil
		}
		return false, false, err
	}
	if !info.IsDir() {
		return false, false, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		if isPermissionDenied(err) {
			return false, true, nil
		}
		return false, false, err
	}
	if len(entries) != 0 {
		return false, false, nil
	}
	if err := os.Remove(path); err != nil {
		if isPermissionDenied(err) {
			return false, true, nil
		}
		return false, false, err
	}
	return true, false, nil
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func isPermissionDenied(err error) bool {
	return errors.Is(err, fs.ErrPermission) || os.IsPermission(err)
}

func isStubFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if isPermissionDenied(err) {
			return false, nil
		}
		return false, err
	}
	trimmed := bytes.TrimSpace(data)
	return bytes.Equal(trimmed, []byte("{}")) || bytes.Equal(trimmed, []byte("[]")) || len(trimmed) == 0, nil
}

func filesEqual(left, right string) (bool, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return false, err
	}
	if leftInfo.IsDir() || rightInfo.IsDir() {
		return false, nil
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}
	leftBytes, err := os.ReadFile(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := os.ReadFile(right)
	if err != nil {
		return false, err
	}
	if len(leftBytes) != len(rightBytes) {
		return false, nil
	}
	for i := range leftBytes {
		if leftBytes[i] != rightBytes[i] {
			return false, nil
		}
	}
	return true, nil
}
