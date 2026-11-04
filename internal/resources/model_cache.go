package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/vrooli/internal/artifactlock"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

const modelCacheOwnerCommand = "vrooli resource acquisition model-cache"

type ModelCacheUsage struct {
	LastActivity time.Time `json:"last_activity"`
	Event        string    `json:"event"`
	Content      string    `json:"content_fingerprint"`
	Pinned       bool      `json:"pinned"`
}

type ModelCachePreview struct {
	Resource       string    `json:"resource"`
	Entry          string    `json:"entry"`
	Eligible       bool      `json:"eligible"`
	Reason         string    `json:"reason"`
	Bytes          int64     `json:"bytes"`
	MinIdleSeconds int64     `json:"min_idle_seconds"`
	LastActivity   time.Time `json:"last_activity"`
	Pinned         bool      `json:"pinned"`
	Snapshot       string    `json:"snapshot_sha256"`
}

type ModelCacheResult struct {
	Preview ModelCachePreview `json:"preview"`
	Removed bool              `json:"removed"`
}

type modelCacheStore struct {
	manifest               ResourceManifest
	entryName, root, state string
	age                    time.Duration
	now                    func() time.Time
	running                func() (bool, error)
}

func modelCacheEntry(manifest ResourceManifest, name string) (ResourceStorageEntry, time.Duration, error) {
	if name == "" || strings.ContainsAny(name, `/\\:`) || name == "." || name == ".." {
		return ResourceStorageEntry{}, 0, fmt.Errorf("invalid model-cache entry identifier")
	}
	if manifest.Storage == nil || manifest.ManagedService == nil {
		return ResourceStorageEntry{}, 0, fmt.Errorf("resource has no owned model cache")
	}
	entry, ok := manifest.Storage.Entries[name]
	if !ok || entry.Reclaim == nil || entry.Reclaim.Command != modelCacheOwnerCommand+" "+manifest.Name || entry.Reclaim.Pruner != "custom" || entry.Budget == nil || entry.Class != "data" || entry.Kind != "dir" {
		return entry, 0, fmt.Errorf("storage entry does not delegate model-cache retention to this owner")
	}
	if entry.Subpath == "" || entry.Subpath == "." || entry.Subpath == ".." || filepath.Base(entry.Subpath) != entry.Subpath || strings.ContainsAny(entry.Subpath, `/\\`) {
		return entry, 0, fmt.Errorf("model cache requires one named child of the owned data directory")
	}
	for _, path := range entry.Reclaim.GeneratedFiles {
		if !fs.ValidPath(path) || path == "." || strings.ContainsAny(path, `\:*?[]`) {
			return entry, 0, fmt.Errorf("model-cache generated_files requires exact relative file paths")
		}
	}
	age, err := retention.ParseAge(entry.Budget.MaxAge)
	if err != nil || age <= 0 {
		return entry, 0, fmt.Errorf("model cache requires a positive declared max_age")
	}
	return entry, age, nil
}

func newModelCacheStore(manifest ResourceManifest, entryName string) (*modelCacheStore, error) {
	entry, age, err := modelCacheEntry(manifest, entryName)
	if err != nil {
		return nil, err
	}
	paths, err := resourceStoragePaths(manifest.Name)
	if err != nil {
		return nil, err
	}
	supervisor := newManagedServiceSupervisor(filepath.Join(paths.StateDir, managedServiceStateFile), filepath.Join(paths.LogsDir, "service.log"))
	return &modelCacheStore{manifest: manifest, entryName: entryName, root: filepath.Join(paths.DataDir, entry.Subpath),
		state: filepath.Join(paths.StateDir, "model-cache-"+entryName+".json"), age: age, now: time.Now,
		running: func() (bool, error) { _, running, err := supervisor.Status(); return running, err }}, nil
}

func (s *modelCacheStore) content() string {
	entry := s.manifest.Storage.Entries[s.entryName]
	data, _ := json.Marshal([]any{s.manifest.ManagedService.DataArtifacts, entry.Reclaim.GeneratedFiles})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (s *modelCacheStore) usage() (ModelCacheUsage, error) {
	var usage ModelCacheUsage
	data, err := os.ReadFile(s.state)
	if os.IsNotExist(err) {
		return usage, nil
	}
	if err != nil {
		return usage, err
	}
	if len(data) > 65536 {
		return usage, fmt.Errorf("model-cache state exceeds 64KiB")
	}
	if err := json.Unmarshal(data, &usage); err != nil {
		return usage, fmt.Errorf("corrupt model-cache usage: %w", err)
	}
	return usage, nil
}

func (s *modelCacheStore) save(usage ModelCacheUsage) error {
	data, err := json.Marshal(usage)
	if err != nil {
		return err
	}
	return runtimestorage.WriteFileAtomic(s.state, data, 0600)
}

func (s *modelCacheStore) note(event string) error {
	usage, err := s.usage()
	if err != nil {
		return err
	}
	running, err := s.running()
	if err != nil {
		return err
	}
	if running {
		event = "started"
	}
	usage.LastActivity, usage.Content, usage.Event = s.now().UTC(), s.content(), event
	return s.save(usage)
}

func (s *modelCacheStore) preview() (ModelCachePreview, error) {
	usage, err := s.usage()
	if err != nil {
		return ModelCachePreview{}, err
	}
	p := ModelCachePreview{Resource: s.manifest.Name, Entry: s.entryName, Pinned: usage.Pinned, LastActivity: usage.LastActivity, MinIdleSeconds: int64(s.age / time.Second)}
	running, err := s.running()
	if err != nil {
		return p, err
	}
	switch {
	case running:
		p.Reason = "resource is running"
	case usage.Pinned:
		p.Reason = "model cache is pinned"
	case usage.LastActivity.IsZero() || usage.Content != s.content():
		p.Reason = "usage or model identity is unknown"
	case usage.Event != "stopped" && usage.Event != "installed":
		p.Reason = "last active lifetime has no confirmed stop; use normal resource stop first"
	case s.now().Sub(usage.LastActivity) < s.age:
		p.Reason = "declared idle retention has not elapsed"
	default:
		p.Eligible = true
	}
	info, err := os.Lstat(s.root)
	if os.IsNotExist(err) {
		p.Eligible, p.Reason = false, "model cache is absent"
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return p, fmt.Errorf("model cache root is not an owned directory")
	}
	expected := map[string]bool{}
	dataRoot := filepath.Dir(s.root)
	for _, artifact := range s.manifest.ManagedService.DataArtifacts {
		path := filepath.Join(dataRoot, filepath.FromSlash(artifact.Path))
		if strings.HasPrefix(path, s.root+string(filepath.Separator)) {
			expected[path], expected[path+".sha256"] = true, true
		}
	}
	for _, file := range s.manifest.Storage.Entries[s.entryName].Reclaim.GeneratedFiles {
		path := filepath.Join(s.root, filepath.FromSlash(file))
		expected[path], expected[path+".sha256"] = true, true
	}

	if len(expected) == 0 {
		return p, fmt.Errorf("model cache has no declared model files")
	}
	type fileState struct {
		Path     string
		Size     int64
		Modified int64
		Mode     fs.FileMode
	}
	files := []fileState{}
	err = filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("model cache contains a symlink")
		}
		if entry.IsDir() {
			return nil
		}
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		if !stat.Mode().IsRegular() || !expected[path] {
			p.Eligible, p.Reason = false, "cache contains undeclared or special files"
		}
		p.Bytes += stat.Size()
		files = append(files, fileState{Path: path, Size: stat.Size(), Modified: stat.ModTime().UnixNano(), Mode: stat.Mode()})
		if len(files) > 100000 {
			return fmt.Errorf("model-cache inventory exceeds 100000 files")
		}
		return nil
	})
	if err != nil {
		return p, err
	}
	data, err := json.Marshal([]any{usage, s.content(), files})
	if err != nil {
		return p, err
	}
	p.Snapshot = fmt.Sprintf("%x", sha256.Sum256(data))
	return p, nil
}

func (s *modelCacheStore) apply(reviewed ModelCachePreview) (ModelCacheResult, error) {
	current, err := s.preview()
	result := ModelCacheResult{Preview: current}
	if err != nil {
		return result, err
	}
	if reviewed != current || !current.Eligible {
		return result, fmt.Errorf("model-cache preview changed or cache is protected: %s", current.Reason)
	}
	if err := os.RemoveAll(s.root); err != nil {
		return result, err
	}
	result.Removed = true
	return result, nil
}

// ModelCache retains lifecycle and acquisition fences until the exact preview
// has been revalidated and applied. It never stops a resource to reclaim disk.
func (c *Controller) ModelCache(ctx context.Context, name, entry string, pin *bool, reviewed *ModelCachePreview) (ModelCacheResult, error) {
	manifest, err := c.ResourceManifest(name)
	if err != nil {
		return ModelCacheResult{}, err
	}
	if _, _, err := modelCacheEntry(manifest, entry); err != nil {
		return ModelCacheResult{}, err
	}
	release, err := acquireManagedLifecycleLocks(ctx, manifest)
	if err != nil {
		return ModelCacheResult{}, err
	}
	defer release()
	releaseData, err := artifactlock.AcquireContext(ctx, "resource-data:"+manifest.Name)
	if err != nil {
		return ModelCacheResult{}, err
	}
	defer releaseData()
	store, err := newModelCacheStore(manifest, entry)
	if err != nil {
		return ModelCacheResult{}, err
	}
	if pin != nil {
		if reviewed != nil {
			return ModelCacheResult{}, fmt.Errorf("pin and apply must be separate operations")
		}
		usage, err := store.usage()
		if err != nil {
			return ModelCacheResult{}, err
		}
		usage.Pinned = *pin
		if err := store.save(usage); err != nil {
			return ModelCacheResult{}, err
		}
	}
	if reviewed != nil {
		return store.apply(*reviewed)
	}
	p, err := store.preview()
	return ModelCacheResult{Preview: p}, err
}

// Caller already holds resource-data. A successful acquisition starts a fresh
// window even when weights were reacquired after an earlier retention sweep.
func noteModelCachesLocked(manifest ResourceManifest, event string) error {
	if manifest.Storage == nil {
		return nil
	}
	for name, entry := range manifest.Storage.Entries {
		if entry.Reclaim == nil || entry.Reclaim.Command != modelCacheOwnerCommand+" "+manifest.Name {
			continue
		}
		store, err := newModelCacheStore(manifest, name)
		if err != nil {
			return err
		}
		if err := store.note(event); err != nil {
			return err
		}
	}
	return nil
}

func noteModelCaches(ctx context.Context, manifest ResourceManifest, event string) error {
	release, err := artifactlock.AcquireContext(ctx, "resource-data:"+manifest.Name)
	if err != nil {
		return err
	}
	defer release()
	return noteModelCachesLocked(manifest, event)
}

// Called while the selected member's group lock is held. Retention is an
// opportunistic owner action at model activation, never a private scenario
// deleter and never a reason to stop a running peer. Unknown history is kept.
func pruneIdleExclusiveModelCaches(ctx context.Context, controller *Controller, selected ResourceManifest) error {
	if selected.Lifecycle.ExclusiveGroup == "" {
		return nil
	}
	items, err := controller.Discover()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name == selected.Name {
			continue
		}
		peer, err := controller.ResourceManifest(item.Name)
		if err != nil || peer.Lifecycle.ExclusiveGroup != selected.Lifecycle.ExclusiveGroup || peer.Storage == nil {
			continue
		}
		for name, entry := range peer.Storage.Entries {
			if entry.Reclaim == nil || entry.Reclaim.Command != modelCacheOwnerCommand+" "+peer.Name {
				continue
			}
			if err := pruneIdlePeerModelCache(ctx, peer, name); err != nil {
				return err
			}
		}
	}
	return nil
}

func pruneIdlePeerModelCache(ctx context.Context, peer ResourceManifest, entry string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	release, err := acquireManagedServiceLifecycleLock(ctx, peer.Name)
	if err != nil {
		return err
	}
	defer release()
	releaseData, err := artifactlock.AcquireContext(ctx, "resource-data:"+peer.Name)
	if err != nil {
		return err
	}
	defer releaseData()
	store, err := newModelCacheStore(peer, entry)
	if err != nil {
		return err
	}
	p, err := store.preview()
	if err != nil || !p.Eligible {
		return err
	}
	result, err := store.apply(p)
	if err == nil && result.Removed {
		_, _ = fmt.Fprintf(os.Stderr, "model-cache retention: reclaimed %d bytes from inactive %s/%s\n", p.Bytes, peer.Name, entry)
	}
	return err
}
