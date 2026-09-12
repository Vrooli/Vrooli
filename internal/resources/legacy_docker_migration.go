package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	legacyDockerMigrationParameterA = 3
)

var runDockerLifecycleCommand = runDockerLifecycleCommandReal

func runDockerLifecycleCommandReal(ctx context.Context, controller *Controller, stdout, stderr io.Writer, args ...string) error {
	cmd := shell.Command(shell.Spec{
		Name: "docker", Args: args, Dir: controller.Root,
		Env: resourceEnv(controller.Root, controller.Home), Stdout: stdout, Stderr: stderr,
	})
	runCtx, cancel := context.WithTimeout(ctx, tuning.DockerRuntimeOperationTimeout())
	defer cancel()
	if err := cmd.Start(); err != nil {
		return err
	}
	waitErr := cmd.Wait()
	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	return waitErr
}

func dockerCommand(ctx context.Context, controller *Controller, stdout, stderr io.Writer, args ...string) error {
	return runDockerLifecycleCommand(ctx, controller, stdout, stderr, args...)
}

func dockerOutput(ctx context.Context, controller *Controller, args ...string) ([]byte, error) {
	cmd := shell.Command(shell.Spec{
		Name: "docker", Args: args, Dir: controller.Root,
		Env: resourceEnv(controller.Root, controller.Home),
	})
	result := runCommandResource(ctx, cmd)
	if result.err != nil {
		return nil, fmt.Errorf("%w: %s", result.err, strings.TrimSpace(string(result.output)))
	}
	return result.output, nil
}

func isMissingDockerArtifact(err error) bool {
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	for _, marker := range []string{"no such object", "no such image", "no such volume", "no such network", "no such container", "not found", "does not exist"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

// legacyDockerContainer is intentionally smaller than Docker's full inspect
// document. The migration is allowed to act only when the container identity
// and its canonical data mount both match the resource being started.
type legacyDockerContainer struct {
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	Mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
	} `json:"Mounts"`
}

type legacyStorageMount struct {
	path        string
	regenerable bool
}

// migrateLegacyDockerStorage removes the old Vrooli Docker instance before a
// managed-service resource starts. It is deliberately fail-closed: an
// arbitrary container, an unexpected mount, or a path outside the resolved
// resource data directory is never touched.
//
// Foreign-owned data is not chowned or deleted. It is moved to a timestamped
// sibling backup and replaced with a fresh directory owned by the invoking
// user. This is safe for resources whose manifest declares the data as
// regenerable cache, while preserving the old bytes for recovery or a later
// resource-specific importer.
func migrateLegacyDockerStorage(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if controller == nil || strings.TrimSpace(manifest.Name) == "" {
		return nil
	}
	paths, err := resourceStoragePaths(manifest.Name)
	if err != nil {
		return err
	}
	dataDir := filepath.Clean(paths.DataDir)
	mounts := legacyDockerStorageMounts(manifest, dataDir)

	containerName, container, mountPath, found, err := inspectLegacyDockerContainer(ctx, controller, manifest, mounts)
	if err != nil || !found {
		return err
	}

	foreign, err := storagePathForeignOwned(mountPath)
	if err != nil {
		return err
	}
	mount := legacyStorageMount{path: mountPath}
	for _, candidate := range mounts {
		if filepath.Clean(candidate.path) == mountPath {
			mount = candidate
			break
		}
	}
	if foreign && !mount.regenerable {
		return fmt.Errorf("refuse legacy %s migration for foreign-owned non-regenerable storage %s; perform a controlled maintenance-window copy and ownership transition", manifest.Name, mountPath)
	}
	backupDir := ""
	if foreign {
		backupDir = legacyStorageBackupPath(mountPath)
		if err := os.Rename(mountPath, backupDir); err != nil { //nolint:forbidigo // intentional directory migration
			return fmt.Errorf("preserve legacy %s storage at %s: %w", manifest.Name, backupDir, err)
		}
		if err := os.Mkdir(mountPath, tuning.PermPrivateDir); err != nil {
			_ = os.Rename(backupDir, mountPath) //nolint:forbidigo // intentional directory rollback
			return fmt.Errorf("create current-user %s storage: %w", manifest.Name, err)
		}
	}

	rollbackStorage := func() {
		if backupDir == "" {
			return
		}
		_ = os.Remove(mountPath)
		_ = os.Rename(backupDir, mountPath) //nolint:forbidigo // intentional directory rollback
	}

	if container.State.Running {
		if err := dockerCommand(ctx, controller, io.Discard, io.Discard, "stop", containerName); err != nil {
			rollbackStorage()
			return fmt.Errorf("stop legacy %s container %s: %w", manifest.Name, containerName, err)
		}
	}
	if err := dockerCommand(ctx, controller, io.Discard, io.Discard, "rm", containerName); err != nil {
		rollbackStorage()
		return fmt.Errorf("remove legacy %s container %s: %w", manifest.Name, containerName, err)
	}
	return nil
}

func inspectLegacyDockerContainer(ctx context.Context, controller *Controller, manifest ResourceManifest, mounts []legacyStorageMount) (string, legacyDockerContainer, string, bool, error) {
	for _, name := range legacyDockerContainerNames(manifest) {
		output, err := dockerOutput(ctx, controller, "container", "inspect", name)
		if err != nil {
			if isMissingDockerArtifact(err) {
				continue
			}
			return "", legacyDockerContainer{}, "", false, err
		}
		// Docker emits an array for an unformatted inspect. Decode exactly one
		// element so malformed or ambiguous output fails closed.
		var containers []legacyDockerContainer
		if err := json.Unmarshal(output, &containers); err != nil || len(containers) != 1 {
			if err == nil {
				err = fmt.Errorf("expected one inspected container, got %d", len(containers))
			}
			return "", legacyDockerContainer{}, "", false, fmt.Errorf("parse legacy docker container %s: %w", name, err)
		}
		container := containers[0]
		mountPath, ok := legacyDockerMountPath(container, mounts)
		if !ok {
			continue
		}
		return name, container, mountPath, true, nil
	}
	return "", legacyDockerContainer{}, "", false, nil
}

func legacyDockerStorageMounts(manifest ResourceManifest, dataDir string) []legacyStorageMount {
	mounts := []legacyStorageMount{{path: dataDir}}
	entries := manifest.StorageEntries()
	for _, entry := range entries {
		if entry.Relocation == nil || !strings.EqualFold(strings.TrimSpace(entry.Relocation.Key), "RESOURCE_DATA_DIR") || strings.TrimSpace(entry.Subpath) == "" {
			continue
		}
		path := filepath.Clean(filepath.Join(dataDir, filepath.FromSlash(entry.Subpath)))
		relative, err := filepath.Rel(dataDir, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		mounts = append(mounts, legacyStorageMount{path: path, regenerable: entry.Regenerable})
	}
	for index := range mounts {
		if mounts[index].path != dataDir {
			continue
		}
		for _, entry := range entries {
			if strings.EqualFold(strings.TrimSpace(entry.Class), "cache") && entry.Regenerable && strings.TrimSpace(entry.Subpath) == "" {
				mounts[index].regenerable = true
			}
		}
	}
	return mounts
}

func legacyDockerContainerNames(manifest ResourceManifest) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, legacyDockerMigrationParameterA)
	for _, name := range []string{
		strings.TrimSpace(manifest.Runtime.ContainerName),
		"vrooli-" + strings.TrimSpace(manifest.Name) + "-resource",
		"vrooli-" + strings.TrimSpace(manifest.Name),
	} {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func legacyDockerMountMatches(container legacyDockerContainer, dataDir string) bool {
	for _, mount := range container.Mounts {
		if filepath.Clean(filepath.FromSlash(strings.TrimSpace(mount.Destination))) != "/data" {
			continue
		}
		if filepath.Clean(mount.Source) == filepath.Clean(dataDir) {
			return true
		}
	}
	return false
}

func legacyDockerMountPath(container legacyDockerContainer, mounts []legacyStorageMount) (string, bool) {
	for _, mount := range container.Mounts {
		if !filepath.IsAbs(filepath.FromSlash(strings.TrimSpace(mount.Destination))) {
			continue
		}
		for _, expected := range mounts {
			if filepath.Clean(mount.Source) == filepath.Clean(expected.path) {
				return filepath.Clean(expected.path), true
			}
		}
	}
	return "", false
}

func storagePathForeignOwned(path string) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect legacy storage %s: %w", path, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("refuse legacy storage migration for non-directory %s", path)
	}
	uid, ok := legacyStorageOwnerUID(info)
	if !ok {
		return false, fmt.Errorf("inspect legacy storage %s: missing POSIX ownership metadata", path)
	}
	return uid != legacyStorageExpectedUID(), nil
}

var (
	legacyStorageNow         = time.Now
	legacyStorageExpectedUID = func() uint32 {
		expected := uint32(os.Geteuid())
		if invokingUID, _, ok := hostreqkit.InvokingUserIDs(); ok {
			expected = uint32(invokingUID)
		}
		return expected
	}
)

func legacyStorageBackupPath(dataDir string) string {
	stamp := legacyStorageNow().UTC().Format("20060102T150405.000000000Z")
	return dataDir + ".legacy-docker-" + stamp
}
