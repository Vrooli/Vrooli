package cleanupfakes

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"storage-manager/internal/cleanup"
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

// ReadDir lists the immediate children of path from the in-memory file map.
//
// Children are derived rather than stored so a test only has to declare the
// files it cares about; intermediate directories that were never declared are
// still reported, which mirrors a real filesystem where a nested file implies
// the directories above it.
func (fsys *FileSystem) ReadDir(_ context.Context, path string) ([]cleanup.FileInfo, error) {
	prefix := strings.TrimRight(filepath.Clean(path), string(filepath.Separator)) + string(filepath.Separator)

	seen := make(map[string]cleanup.FileInfo)
	for candidate, info := range fsys.Files {
		if !strings.HasPrefix(candidate, prefix) {
			continue
		}
		rest := strings.TrimPrefix(candidate, prefix)
		name, _, nested := strings.Cut(rest, string(filepath.Separator))
		if name == "" {
			continue
		}
		child := filepath.Join(path, name)
		if nested {
			// An implied intermediate directory.
			if _, ok := seen[child]; !ok {
				if declared, found := fsys.Files[child]; found {
					seen[child] = declared
				} else {
					seen[child] = cleanup.FileInfo{Path: child, IsDir: true}
				}
			}
			continue
		}
		seen[child] = info
	}

	out := make([]cleanup.FileInfo, 0, len(seen))
	for _, info := range seen {
		out = append(out, info)
	}
	// Deterministic order; a real ReadDir on the platforms we target is sorted.
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
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

type JournalClient struct {
	Usage   int64
	Vacuums []cleanup.JournalVacuumRequest
}

func (c *JournalClient) DiskUsage(context.Context) (int64, error) {
	return c.Usage, nil
}

func (c *JournalClient) Vacuum(_ context.Context, req cleanup.JournalVacuumRequest) (cleanup.JournalVacuumResult, error) {
	c.Vacuums = append(c.Vacuums, req)
	return cleanup.JournalVacuumResult{ReclaimedBytes: c.Usage}, nil
}

type ScenarioProviderClient struct {
	EstimateResult cleanup.Estimate
	PreviewResult  cleanup.Preview
	ApplyResult    cleanup.ApplyResult
	Applies        []cleanup.ScenarioCleanupRequest
}

func (c *ScenarioProviderClient) Estimate(_ context.Context, _ string, _ cleanup.ProviderPolicy) (cleanup.Estimate, error) {
	return c.EstimateResult, nil
}

func (c *ScenarioProviderClient) Preview(_ context.Context, _ string, _ cleanup.Estimate) (cleanup.Preview, error) {
	return c.PreviewResult, nil
}

func (c *ScenarioProviderClient) Apply(_ context.Context, req cleanup.ScenarioCleanupRequest) (cleanup.ApplyResult, error) {
	c.Applies = append(c.Applies, req)
	return c.ApplyResult, nil
}
