package cleanup

import (
	"context"
	"io/fs"
	"time"
)

type FileInfo struct {
	Path    string
	Size    int64
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
}

// FileSystem is the only seam allowed to mutate filesystem cleanup targets.
type FileSystem interface {
	Stat(ctx context.Context, path string) (FileInfo, error)
	Walk(ctx context.Context, root string, visit func(FileInfo) error) error
	RemoveAll(ctx context.Context, path string) error
}

type ProcessCommand struct {
	Name string
	Args []string
}

type ProcessResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Redacted bool
}

// ProcessRunner is the only seam allowed to execute host cleanup commands.
type ProcessRunner interface {
	Run(ctx context.Context, cmd ProcessCommand) (ProcessResult, error)
}

type DockerUsage struct {
	ImagesBytes     int64
	ContainersBytes int64
	BuildCacheBytes int64
	VolumesBytes    int64
}

type DockerPruneRequest struct {
	DanglingImages bool
	StoppedOnly    bool
	BuildCache     bool
	Volumes        bool
}

type DockerPruneResult struct {
	ReclaimedBytes int64
	Warnings       []string
}

// DockerClient is the only seam allowed to inspect or prune Docker state.
type DockerClient interface {
	SystemUsage(ctx context.Context) (DockerUsage, error)
	Prune(ctx context.Context, req DockerPruneRequest) (DockerPruneResult, error)
}

type JournalVacuumRequest struct {
	MaxAge time.Duration
}

type JournalVacuumResult struct {
	ReclaimedBytes int64
}

// JournalClient is the only seam allowed to inspect or vacuum systemd journals.
type JournalClient interface {
	DiskUsage(ctx context.Context) (int64, error)
	Vacuum(ctx context.Context, req JournalVacuumRequest) (JournalVacuumResult, error)
}

type ScenarioCleanupRequest struct {
	ScenarioID string
	ProviderID string
	Preview    Preview
}

// ScenarioProviderClient delegates scenario-private cleanup to the owner.
type ScenarioProviderClient interface {
	Estimate(ctx context.Context, scenarioID string, policy ProviderPolicy) (Estimate, error)
	Preview(ctx context.Context, scenarioID string, estimate Estimate) (Preview, error)
	Apply(ctx context.Context, req ScenarioCleanupRequest) (ApplyResult, error)
}

type Clock interface {
	Now() time.Time
}
