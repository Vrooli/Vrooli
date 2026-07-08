package cleanupfakes

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"cleanup-manager/internal/cleanup"
)

type Clock struct {
	Time time.Time
}

func (c Clock) Now() time.Time { return c.Time }

type FileSystem struct {
	Root        string
	Files       map[string]cleanup.FileInfo
	Removed     []string
	AllowRemove bool
}

func (fsys *FileSystem) Stat(_ context.Context, path string) (cleanup.FileInfo, error) {
	info, ok := fsys.Files[path]
	if !ok {
		return cleanup.FileInfo{}, fmt.Errorf("stat %s: not found", path)
	}
	return info, nil
}

func (fsys *FileSystem) Walk(_ context.Context, root string, visit func(cleanup.FileInfo) error) error {
	for path, info := range fsys.Files {
		if path == root || strings.HasPrefix(path, strings.TrimRight(root, string(filepath.Separator))+string(filepath.Separator)) {
			if err := visit(info); err != nil {
				return err
			}
		}
	}
	return nil
}

func (fsys *FileSystem) RemoveAll(_ context.Context, path string) error {
	if !fsys.AllowRemove {
		return fmt.Errorf("remove blocked by fake filesystem: %s", path)
	}
	if fsys.Root == "" || !strings.HasPrefix(filepath.Clean(path), filepath.Clean(fsys.Root)+string(filepath.Separator)) {
		return fmt.Errorf("remove outside fake root blocked: %s", path)
	}
	fsys.Removed = append(fsys.Removed, path)
	delete(fsys.Files, path)
	return nil
}

type ProcessRunner struct {
	Forbidden []string
	Commands  []cleanup.ProcessCommand
	Result    cleanup.ProcessResult
}

func (r *ProcessRunner) Run(_ context.Context, cmd cleanup.ProcessCommand) (cleanup.ProcessResult, error) {
	line := strings.TrimSpace(cmd.Name + " " + strings.Join(cmd.Args, " "))
	for _, forbidden := range r.Forbidden {
		if strings.Contains(line, forbidden) {
			return cleanup.ProcessResult{}, fmt.Errorf("forbidden command invoked in test: %s", line)
		}
	}
	r.Commands = append(r.Commands, cmd)
	return r.Result, nil
}

type DockerClient struct {
	Usage  cleanup.DockerUsage
	Prunes []cleanup.DockerPruneRequest
}

func (c *DockerClient) SystemUsage(context.Context) (cleanup.DockerUsage, error) {
	return c.Usage, nil
}

func (c *DockerClient) Prune(_ context.Context, req cleanup.DockerPruneRequest) (cleanup.DockerPruneResult, error) {
	if req.Volumes {
		return cleanup.DockerPruneResult{}, fmt.Errorf("fake docker blocks volume prune")
	}
	c.Prunes = append(c.Prunes, req)
	return cleanup.DockerPruneResult{ReclaimedBytes: c.Usage.BuildCacheBytes + c.Usage.ImagesBytes}, nil
}
