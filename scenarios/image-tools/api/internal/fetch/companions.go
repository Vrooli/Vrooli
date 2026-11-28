package fetch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/diskpolicy"
	"image-tools/internal/modelenv"
)

var commitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// CompanionFetcher fetches a pinned Hub repo into the managed Hub cache (not a
// local_dir tree), so a runtime that loads it by repo id finds it while the Hub
// is offline. HFSnapshotFetcher is the production implementation.
type CompanionFetcher interface {
	Companion(ctx context.Context, repo RepoSpec, hubCache string) error
}

// ValidateHubCompanions checks a source's companion declarations: each names a
// repo pinned to a full commit SHA and restricts the files it takes, so the
// bounded cache charge can never turn into a hidden full-model download.
func ValidateHubCompanions(source Source) error {
	seen := map[string]bool{}
	for i, c := range source.HubCompanions {
		id := strings.TrimSpace(c.RepoID)
		if parts := strings.Split(id, "/"); len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("hub_companions[%d]: repo_id %q must look like org/name", i, c.RepoID)
		}
		if !commitSHA.MatchString(c.Revision) {
			return fmt.Errorf("hub_companions[%d] %s: revision must be a full 40-character commit SHA", i, id)
		}
		if len(c.AllowPatterns) == 0 {
			return fmt.Errorf("hub_companions[%d] %s: allow_patterns is required (companions are small config files, never a whole model)", i, id)
		}
		if seen[id] {
			return fmt.Errorf("hub_companions[%d]: %s is declared twice", i, id)
		}
		seen[id] = true
	}
	return nil
}

// HubCacheDir is the managed Hub cache the runtime reads offline.
func HubCacheDir() string { return modelenv.MustLoad().Pins(modelenv.Root())["HF_HUB_CACHE"] }

func companionSnapshotDir(hubCache string, repo RepoSpec) string {
	return filepath.Join(hubCache, "models--"+strings.ReplaceAll(repo.RepoID, "/", "--"), "snapshots", repo.Revision)
}

// MissingCompanions lists the declared companions absent from the managed cache.
func MissingCompanions(source Source) []RepoSpec {
	hub := HubCacheDir()
	var missing []RepoSpec
	for _, c := range source.HubCompanions {
		if info, err := os.Stat(companionSnapshotDir(hub, c)); err != nil || !info.IsDir() {
			missing = append(missing, c)
		}
	}
	return missing
}

// EnsureCompanions fetches every declared companion that is not already in the
// managed Hub cache. It is the only place a companion is fetched.
func (a Acquirer) EnsureCompanions(ctx context.Context, source Source, emit func(int, string)) error {
	if emit == nil {
		emit = func(int, string) {}
	}
	if err := ValidateHubCompanions(source); err != nil {
		return err
	}
	missing := MissingCompanions(source)
	if len(missing) == 0 {
		return nil
	}
	fetcher, ok := a.RepoFetch.(CompanionFetcher)
	if !ok {
		return fmt.Errorf("fetch: the configured repository fetcher cannot place hub companions in the managed cache")
	}
	hub := HubCacheDir()
	if err := os.MkdirAll(hub, 0o755); err != nil {
		return err
	}
	for _, c := range missing {
		emit(86, "fetching hub companion "+c.RepoID+"@"+ShortRev(c.Revision))
		if err := fetcher.Companion(ctx, c, hub); err != nil {
			return fmt.Errorf("fetch: hub companion %s: %w", c.RepoID, err)
		}
		dir := companionSnapshotDir(hub, c)
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("fetch: hub companion %s finished but %s is absent: %w", c.RepoID, dir, err)
		}
		// A pinned companion is "main" in the managed cache: libraries resolve a
		// repo id offline through refs/main, which a by-SHA download never writes.
		repoDir := filepath.Dir(filepath.Dir(dir))
		if err := os.MkdirAll(filepath.Join(repoDir, "refs"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(repoDir, "refs", "main"), []byte(c.Revision), 0o644); err != nil {
			return err
		}
		size, err := directoryBytes(repoDir)
		if err != nil {
			return err
		}
		if size > HuggingFaceCacheAllowanceBytes {
			return errors.Join(fmt.Errorf("fetch: hub companion %s took %d bytes, above the %d-byte companion allowance: narrow its allow_patterns", c.RepoID, size, HuggingFaceCacheAllowanceBytes), os.RemoveAll(filepath.Dir(filepath.Dir(dir))))
		}
	}
	return nil
}

// RepairCompanions places missing companions for an install whose weights are
// already valid (it would otherwise plan no disk job at all). The write is
// bounded by the companion allowance and admitted like every other write.
func RepairCompanions(ctx context.Context, admit sharedcapacity.DiskJobAdmitter, owner, job string, acquirer Acquirer, source Source, emit func(int, string)) error {
	if len(MissingCompanions(source)) == 0 {
		return nil
	}
	if admit == nil {
		return errors.New("fetch: shared host disk admission is unavailable; refusing to fetch hub companions")
	}
	hub := HubCacheDir()
	if err := os.MkdirAll(hub, 0o755); err != nil {
		return err
	}
	canonical, err := canonicalDiskJobPath(hub)
	if err != nil {
		return err
	}
	lease, err := ReserveAttempt(ctx, admit, diskpolicy.Resolve(os.Getenv).Request(owner, job, []sharedcapacity.DiskJobPath{{Path: canonical, Role: "huggingface-hub-cache", PeakBytes: HuggingFaceCacheAllowanceBytes}}))
	if err != nil {
		return fmt.Errorf("fetch: reserve disk for hub companions: %w", err)
	}
	workCtx, guard := MaintainDiskLease(ctx, lease, 45*time.Second)
	fetchErr := acquirer.EnsureCompanions(workCtx, source, emit)
	if stopErr := guard.Stop(); stopErr != nil {
		return errors.Join(fetchErr, stopErr)
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if _, completeErr := lease.Complete(cleanup); completeErr != nil {
		fetchErr = errors.Join(fetchErr, completeErr)
	}
	return fetchErr
}
