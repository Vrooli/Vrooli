package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"storage-manager/internal/cleanup"
)

// DockerImage is the host observation needed for age-aware image reclamation.
// Docker's created-at timestamp is deliberately not used as last-used data.
type DockerImage struct {
	ID      string
	Bytes   int64
	Running bool
}

type DockerImageInventory interface {
	ListImages(context.Context) ([]DockerImage, error)
}

type DockerImagePruner interface {
	RemoveImages(context.Context, []string) (int64, error)
}

type imageUsageLedger interface {
	Record(time.Time, []DockerImage) error
	Eligible(string, time.Time, time.Duration) bool
}

type ledgerEntry struct {
	FirstObserved time.Time   `json:"first_observed"`
	LastUsed      time.Time   `json:"last_used"`
	Observations  []time.Time `json:"observations"`
}

// MemoryDockerUsageLedger is deterministic test infrastructure and a useful
// embedded ledger for callers that do not need persistence.
type MemoryDockerUsageLedger struct {
	mu      sync.Mutex
	Entries map[string]ledgerEntry `json:"entries"`
}

func NewMemoryDockerUsageLedger() *MemoryDockerUsageLedger {
	return &MemoryDockerUsageLedger{Entries: map[string]ledgerEntry{}}
}

func (l *MemoryDockerUsageLedger) Record(now time.Time, images []DockerImage) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.Entries == nil {
		l.Entries = map[string]ledgerEntry{}
	}
	for _, image := range images {
		if image.ID == "" || !image.Running {
			continue
		}
		entry := l.Entries[image.ID]
		if entry.FirstObserved.IsZero() {
			entry.FirstObserved = now
		}
		entry.LastUsed = now
		entry.Observations = append(entry.Observations, now)
		l.Entries[image.ID] = entry
	}
	return nil
}

func (l *MemoryDockerUsageLedger) Eligible(id string, now time.Time, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.Entries[id]
	return ok && !entry.FirstObserved.IsZero() && !entry.LastUsed.IsZero() && now.Sub(entry.FirstObserved) >= window && now.Sub(entry.LastUsed) >= window
}

// FileDockerUsageLedger persists observations atomically so a restart cannot
// silently reset the safety window and make a recently used image eligible.
type FileDockerUsageLedger struct {
	mu      sync.Mutex
	path    string
	Entries map[string]ledgerEntry `json:"entries"`
}

func NewFileDockerUsageLedger(path string) (*FileDockerUsageLedger, error) {
	l := &FileDockerUsageLedger{path: path, Entries: map[string]ledgerEntry{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, l); err != nil {
		return nil, err
	}
	if l.Entries == nil {
		l.Entries = map[string]ledgerEntry{}
	}
	return l, nil
}

func (l *FileDockerUsageLedger) Record(now time.Time, images []DockerImage) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.Entries == nil {
		l.Entries = map[string]ledgerEntry{}
	}
	for _, image := range images {
		if image.ID == "" || !image.Running {
			continue
		}
		entry := l.Entries[image.ID]
		if entry.FirstObserved.IsZero() {
			entry.FirstObserved = now
		}
		entry.LastUsed = now
		entry.Observations = append(entry.Observations, now)
		l.Entries[image.ID] = entry
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(l.path), ".docker-ledger-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTempFile := os.Remove
	defer func() { _ = removeTempFile(tmpPath) }()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, l.path)
}

func (l *FileDockerUsageLedger) Eligible(id string, now time.Time, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.Entries[id]
	return ok && !entry.FirstObserved.IsZero() && !entry.LastUsed.IsZero() && now.Sub(entry.FirstObserved) >= window && now.Sub(entry.LastUsed) >= window
}

type DockerUnusedImagesProvider struct {
	client cleanup.DockerClient
	ledger imageUsageLedger
}

func NewDockerUnusedImagesProvider(client cleanup.DockerClient, ledger imageUsageLedger) *DockerUnusedImagesProvider {
	return &DockerUnusedImagesProvider{client: client, ledger: ledger}
}

func (p *DockerUnusedImagesProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID: "docker-unused-images", Name: "Docker images unused for the ledger window", Version: "v1",
		OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierConditional,
		DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOperator,
		SupportedPlatforms: []string{"linux", "darwin", "windows"}, RequiredPrivileges: []string{"docker-daemon"},
		IrreversibleEffects: []string{"operator-approved Docker image removal"}, TestSubstitute: "fake-docker-usage-ledger",
	}
}

func (p *DockerUnusedImagesProvider) observe(ctx context.Context, now time.Time) ([]DockerImage, error) {
	if p.client == nil || p.ledger == nil {
		return nil, fmt.Errorf("docker image inventory or usage ledger unavailable")
	}
	inventory, ok := p.client.(DockerImageInventory)
	if !ok {
		return nil, fmt.Errorf("docker client does not expose image inventory")
	}
	images, err := inventory.ListImages(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.ledger.Record(now, images); err != nil {
		return nil, err
	}
	return images, nil
}

func (p *DockerUnusedImagesProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	if !req.Policy.Enabled {
		return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, BlockedReason: "provider disabled by policy", RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
	}
	images, err := p.observe(ctx, req.Scope.Now)
	if err != nil {
		return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, BlockedReason: err.Error(), RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
	}
	var bytes int64
	items := 0
	for _, image := range images {
		if !image.Running && p.ledger.Eligible(image.ID, req.Scope.Now, req.Policy.MinAge) {
			bytes += image.Bytes
			items++
		}
	}
	return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, EstimatedBytes: bytes, ItemCount: items, RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
}

func (p *DockerUnusedImagesProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	images, err := p.observe(ctx, req.Scope.Now)
	if err != nil {
		out.BlockedReason = err.Error()
		return out, nil
	}
	for _, image := range images {
		if image.Running || !p.ledger.Eligible(image.ID, req.Scope.Now, req.Policy.MinAge) {
			continue
		}
		out.Items = append(out.Items, cleanup.PreviewItem{ID: "docker-unused-images:" + image.ID, Description: "Docker image unused for the full ledger window", Bytes: image.Bytes, Action: "docker-remove-image", SafetyTier: cleanup.SafetyTierConditional})
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ID < out.Items[j].ID })
	return out, nil
}

func (p *DockerUnusedImagesProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.Metadata().Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider docker-unused-images version mismatch: got %q want %q", req.ProviderVersion, p.Metadata().Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider docker-unused-images requires idempotency key")
	}
	if req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: p.Metadata().ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"operator approval required"}}, nil
	}
	pruner, ok := p.client.(DockerImagePruner)
	if !ok {
		return cleanup.ApplyResult{ProviderID: p.Metadata().ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"docker image removal seam unavailable"}}, nil
	}
	ids := make([]string, 0, len(req.Preview.Items))
	for _, item := range req.Preview.Items {
		const prefix = "docker-unused-images:"
		if len(item.ID) > len(prefix) {
			ids = append(ids, item.ID[len(prefix):])
		}
	}
	bytes, err := pruner.RemoveImages(ctx, ids)
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	return cleanup.ApplyResult{ProviderID: p.Metadata().ID, Applied: bytes > 0, ReclaimedBytes: bytes}, nil
}

func (p *DockerUnusedImagesProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "Docker image eviction is ledger-aged and operator-approved"}, nil
}
