package resources

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

// DefaultArtifactKeepPrevious is the rollback depth kept beside the active
// version: one previous install is enough to switch back without a rebuild.
const DefaultArtifactKeepPrevious = 1

// Retention reasons. A version is removed only when none of these apply.
const (
	ArtifactRetainActive   = "active"
	ArtifactRetainRunning  = "running"
	ArtifactRetainInUse    = "in-use"
	ArtifactRetainRollback = "rollback"
	ArtifactRetainUpdate   = "update-operation"
	ArtifactRetainRemove   = "superseded"
)

// ArtifactRetentionOptions selects how far a prune goes. The zero value is a
// dry run that keeps no rollback versions; callers normally start from
// DefaultArtifactKeepPrevious.
type ArtifactRetentionOptions struct {
	KeepPrevious int
	Apply        bool
}

// ArtifactVersionDecision is one version directory and why it stays or goes.
type ArtifactVersionDecision struct {
	Version  string    `json:"version"`
	Path     string    `json:"path"`
	Bytes    int64     `json:"bytes"` // allocated bytes, counting shared files once
	Modified time.Time `json:"modified"`
	Keep     bool      `json:"keep"`
	Reason   string    `json:"reason"`
	Removed  bool      `json:"removed,omitempty"`
}

// ArtifactRetentionPlan is the full decision for one resource's version store.
type ArtifactRetentionPlan struct {
	Resource         string                    `json:"resource"`
	Root             string                    `json:"root"`
	ActiveVersion    string                    `json:"active_version"`
	KeepPrevious     int                       `json:"keep_previous"`
	Applied          bool                      `json:"applied"`
	Versions         []ArtifactVersionDecision `json:"versions"`
	ReclaimableBytes int64                     `json:"reclaimable_bytes"` // bytes not linked from any kept file
	RemovedBytes     int64                     `json:"removed_bytes"`
}

// artifactVersionsInUse reports which version directories under root hold the
// executable or working directory of a live process. It is a variable so tests
// can model a process the supervisor does not own (an import probe, a peer).
var artifactVersionsInUse = scanArtifactVersionsInUse

// PruneManagedServiceArtifacts plans, and with opts.Apply executes, retention
// for one resource's user-owned artifact store
// (<artifacts>/<resource>/<version>/). The manifest-pinned version, the
// version a supervised instance runs, any version a live process executes
// from, and the opts.KeepPrevious most recent other versions are kept.
func (c *Controller) PruneManagedServiceArtifacts(name string, opts ArtifactRetentionOptions) (ArtifactRetentionPlan, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ArtifactRetentionPlan{}, fmt.Errorf("resource name is required")
	}
	manifest, err := c.LoadManifest(filepath.Join(c.Root, "resources", name, "resource.json"))
	if err != nil {
		return ArtifactRetentionPlan{}, err
	}
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return ArtifactRetentionPlan{Resource: manifest.Name}, nil
	}
	if !opts.Apply {
		return planManagedServiceArtifactRetention(c, manifest, opts)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// The lifecycle lock keeps a start from selecting a version between the
	// running check and the removal.
	release, err := acquireManagedLifecycleLocks(ctx, manifest)
	if err != nil {
		return ArtifactRetentionPlan{}, err
	}
	defer release()
	return applyManagedServiceArtifactRetention(ctx, c, manifest, opts)
}

// applyManagedServiceArtifactRetention removes what the plan rejects. The
// caller holds the resource's lifecycle lock; this takes the artifact lock so
// no acquisition writes into the store while it is pruned.
func applyManagedServiceArtifactRetention(ctx context.Context, c *Controller, manifest ResourceManifest, opts ArtifactRetentionOptions) (ArtifactRetentionPlan, error) {
	releaseArtifact, err := artifactlock.AcquireContext(ctx, "resource:"+manifest.Name)
	if err != nil {
		return ArtifactRetentionPlan{}, err
	}
	defer releaseArtifact()
	opts.Apply = true
	plan, err := planManagedServiceArtifactRetention(c, manifest, opts)
	if err != nil {
		return plan, err
	}
	for i := range plan.Versions {
		decision := &plan.Versions[i]
		if decision.Keep {
			continue
		}
		if err := os.RemoveAll(decision.Path); err != nil {
			return plan, fmt.Errorf("prune %s artifact version %s: %w", manifest.Name, decision.Version, err)
		}
		decision.Removed = true
	}
	plan.RemovedBytes = plan.ReclaimableBytes
	return plan, nil
}

func planManagedServiceArtifactRetention(c *Controller, manifest ResourceManifest, opts ArtifactRetentionOptions) (ArtifactRetentionPlan, error) {
	keep := opts.KeepPrevious
	if keep < 0 {
		keep = 0
	}
	plan := ArtifactRetentionPlan{
		Resource:      manifest.Name,
		ActiveVersion: manifest.ManagedService.Artifact.Version,
		KeepPrevious:  keep,
		Applied:       opts.Apply,
	}
	root, err := managedServiceArtifactVersionsRoot(c, manifest)
	if err != nil {
		return plan, err
	}
	plan.Root = root
	// Caller death can leave neither a candidate nor a restored process alive.
	// The operation's frozen bytes remain recovery inputs nonetheless. Read the
	// strict ledger before planning any removals; an unreadable ledger is not
	// evidence that no operation owns these artifacts.
	updateVersions, err := managedUpdateArtifactVersions(manifest.Name, root)
	if err != nil {
		return plan, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return plan, nil
	}
	if err != nil {
		return plan, err
	}

	running, previous := "", ""
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return plan, err
	}
	state, isRunning, err := supervisor.Status()
	if err != nil {
		return plan, err
	}
	if isRunning {
		running = state.ArtifactVersion
	}
	previous = state.PreviousArtifactVersion
	inUse := artifactVersionsInUse(root)

	candidates := make([]ArtifactVersionDecision, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		decision := ArtifactVersionDecision{Version: entry.Name(), Path: path, Keep: true}
		if info, err := entry.Info(); err == nil {
			decision.Modified = info.ModTime().UTC()
		}
		decision.Bytes = artifactTreeUsage(path, map[artifactInode]*artifactFileUse{})
		switch {
		case decision.Version == plan.ActiveVersion:
			decision.Reason = ArtifactRetainActive
		case decision.Version == running:
			decision.Reason = ArtifactRetainRunning
		case inUse[decision.Version]:
			decision.Reason = ArtifactRetainInUse
		case updateVersions[decision.Version]:
			decision.Reason = ArtifactRetainUpdate
		default:
			candidates = append(candidates, decision)
			continue
		}
		plan.Versions = append(plan.Versions, decision)
	}
	// Rollback order: the version the supervisor last replaced first, since
	// that is what a rollback restores, then the most recently installed.
	sort.SliceStable(candidates, func(i, j int) bool {
		if (candidates[i].Version == previous) != (candidates[j].Version == previous) {
			return candidates[i].Version == previous
		}
		if !candidates[i].Modified.Equal(candidates[j].Modified) {
			return candidates[i].Modified.After(candidates[j].Modified)
		}
		return candidates[i].Version > candidates[j].Version
	})
	removed := map[artifactInode]*artifactFileUse{}
	for i, decision := range candidates {
		if i < keep {
			decision.Reason = ArtifactRetainRollback
		} else {
			decision.Keep, decision.Reason = false, ArtifactRetainRemove
			artifactTreeUsage(decision.Path, removed)
		}
		plan.Versions = append(plan.Versions, decision)
	}
	plan.ReclaimableBytes = reclaimableArtifactBytes(removed)
	sort.SliceStable(plan.Versions, func(i, j int) bool {
		return plan.Versions[i].Modified.After(plan.Versions[j].Modified)
	})
	return plan, nil
}

func managedUpdateArtifactVersions(resource, root string) (map[string]bool, error) {
	store, err := newManagedUpdateStore(resource)
	if err != nil {
		return nil, err
	}
	u, err := store.Active()
	if errors.Is(err, ErrManagedUpdateNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("retain managed update artifacts: %w", err)
	}
	versions := map[string]bool{}
	for _, c := range []ManagedUpdateContract{u.Target, u.Rollback} {
		relative, err := filepath.Rel(root, c.Artifact.Path)
		if err != nil {
			return nil, fmt.Errorf("locate retained update artifact: %w", err)
		}
		// Legacy artifacts outside this store are not prune candidates here.
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		version := strings.SplitN(relative, string(filepath.Separator), 2)[0]
		if version == "." || version != c.Artifact.Version {
			return nil, fmt.Errorf("retained update artifact path disagrees with its version")
		}
		versions[version] = true
	}
	return versions, nil
}

// retainManagedServiceArtifactsAfterActivation applies default retention once
// a start has switched the supervised instance to a different version. A
// restart of the same version leaves the store alone. The caller holds the
// lifecycle lock. Failure never fails the start that already succeeded.
func retainManagedServiceArtifactsAfterActivation(ctx context.Context, c *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) {
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil || supervisor == nil {
		return
	}
	state, running, err := supervisor.Status()
	if err != nil || !running || state.PreviousArtifactVersion == "" || state.PreviousArtifactVersion == state.ArtifactVersion {
		return
	}
	plan, err := applyManagedServiceArtifactRetention(ctx, c, manifest, ArtifactRetentionOptions{KeepPrevious: DefaultArtifactKeepPrevious})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: artifact retention for %q skipped: %s\n", manifest.Name, err)
		return
	}
	if plan.RemovedBytes > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "pruned superseded %s artifact versions (%d bytes)\n", manifest.Name, plan.RemovedBytes)
	}
}

type artifactInode struct{ dev, ino uint64 }

type artifactFileUse struct {
	bytes        int64
	links, found uint64
}

// artifactTreeUsage adds every regular file under root to uses and returns the
// allocated bytes of the distinct files it holds. Versions share files through
// hardlinks, so a file is reclaimable only when all its links are removed.
func artifactTreeUsage(root string, uses map[artifactInode]*artifactFileUse) int64 {
	var total int64
	seen := map[artifactInode]bool{}
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		key, links, bytes := artifactFileIdentity(info)
		if key == (artifactInode{}) {
			total += bytes
			return nil
		}
		if !seen[key] {
			seen[key] = true
			total += bytes
		}
		use := uses[key]
		if use == nil {
			use = &artifactFileUse{bytes: bytes, links: links}
			uses[key] = use
		}
		use.found++
		return nil
	})
	return total
}

// reclaimableArtifactBytes sums the removed files whose every link was found
// in the removed versions; a file also linked from a kept version, or from
// outside the store, frees nothing.
func reclaimableArtifactBytes(uses map[artifactInode]*artifactFileUse) int64 {
	var total int64
	for _, use := range uses {
		if use.found >= use.links {
			total += use.bytes
		}
	}
	return total
}

// scanArtifactVersionsInUse reads /proc for processes whose executable or
// working directory lies beneath root/<version>/. Hosts without /proc report
// nothing; the supervisor state still protects the running instance.
func scanArtifactVersionsInUse(root string) map[string]bool {
	inUse := map[string]bool{}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return inUse
	}
	prefix := filepath.Clean(root) + string(filepath.Separator)
	for _, entry := range entries {
		if !entry.IsDir() || strings.TrimLeft(entry.Name(), "0123456789") != "" {
			continue
		}
		for _, link := range []string{"exe", "cwd"} {
			target, err := os.Readlink(filepath.Join("/proc", entry.Name(), link))
			if err != nil {
				continue
			}
			target = strings.TrimSuffix(target, " (deleted)")
			if !strings.HasPrefix(target, prefix) {
				continue
			}
			version, _, _ := strings.Cut(strings.TrimPrefix(target, prefix), string(filepath.Separator))
			if version != "" {
				inUse[version] = true
			}
		}
	}
	return inUse
}
