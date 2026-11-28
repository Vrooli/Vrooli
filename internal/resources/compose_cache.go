package resources

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/artifactlock"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// Cache recipe identity changes with the compiled acquisition recipe. A local
// wrapper edit leaves unchanged runtime/wheel parts reusable; changing the
// owner recipe invalidates them without a second hand-maintained version pin.
//
//go:embed managed_service_acquisition.go
var composeRecipe []byte

type (
	composeStepCache struct {
		root, resource string
		lockRoot       string
		offline        bool
	}
	offlineComposeKey struct{}
)

// WithOfflineCompose requires every external compose part to be in the
// verified owner cache. It never treats an installed expanded tree as a cache.
func WithOfflineCompose(ctx context.Context) context.Context {
	return context.WithValue(ctx, offlineComposeKey{}, true)
}

func composeManagedServiceArtifactWithCache(ctx context.Context, name string, target binaryfetch.AcquisitionTarget, resourceRoot, artifactRoot string) error {
	paths, err := resourceStoragePaths(name)
	if err != nil {
		return err
	}
	offline, _ := ctx.Value(offlineComposeKey{}).(bool)
	return composeManagedServiceArtifactCached(ctx, target, resourceRoot, artifactRoot, &composeStepCache{root: filepath.Join(paths.CacheDir, "compose"), resource: name, offline: offline, lockRoot: filepath.Join(paths.StateDir, "compose-locks")})
}

func (cache composeStepCache) key(step binaryfetch.ComposeStep, resourceRoot string) (resourcedeployment.ArtifactCacheKey, error) {
	h := sha256.New()
	h.Write(composeRecipe)
	b, err := json.Marshal(step)
	if err != nil {
		return resourcedeployment.ArtifactCacheKey{}, err
	}
	h.Write(b)
	if step.Kind == "python-wheels" {
		// The entire hash lock, rather than a filename or package list, identifies
		// the closure. Only already-validated, resource-owned inputs reach here.
		b, err := os.ReadFile(filepath.Join(resourceRoot, filepath.FromSlash(step.Lockfile)))
		if err != nil {
			return resourcedeployment.ArtifactCacheKey{}, err
		}
		h.Write(b)
	}
	return resourcedeployment.ArtifactCacheKey{Resource: cache.resource, Version: "compose-part", Platform: resourcedeployment.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}, SourceIdentity: hex.EncodeToString(h.Sum(nil))}, nil
}

func (cache composeStepCache) stage(ctx context.Context, step binaryfetch.ComposeStep, resourceRoot, dest string, build func(string) error) error {
	key, err := cache.key(step, resourceRoot)
	if err != nil {
		return err
	}
	// The cache package already owns verification, staging and atomic publish.
	// Add the existing cross-process owner lock for independent CLI callers.
	unlock, err := artifactlock.AcquireWithPathContext(ctx, "compose-"+key.SourceIdentity, func(string) (string, error) { return filepath.Join(cache.lockRoot, key.SourceIdentity+".lock"), nil })
	if err != nil {
		return err
	}
	defer unlock()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(filepath.Dir(dest), ".compose-part-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	_, err = resourcedeployment.StageCachedArtifacts(ctx, cache.root, staging, key, func(ctx context.Context, root string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if cache.offline {
			return errors.New("offline compose requires a verified cached part; network acquisition refused")
		}
		if err := build(filepath.Join(root, "payload")); err != nil {
			return err
		}
		current, err := cache.key(step, resourceRoot)
		if err != nil {
			return err
		}
		if current != key {
			return errors.New("compose inputs changed during cache materialization")
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		return errors.New("compose cache destination must be absent")
	}
	return os.Rename(filepath.Join(staging, "payload"), dest)
}

type ComposePartPreview struct {
	Role                  string `json:"role"`
	Kind                  string `json:"kind"`
	CacheState            string `json:"cache_state"`
	SourceIdentity        string `json:"source_identity"`
	NetworkBytes          *int64 `json:"network_bytes"`
	NetworkCapEnforceable bool   `json:"network_cap_enforceable"`
	// The network bound is conditional on offline execution. An online caller
	// may reacquire a part if the verified cache disappears after this preview.
	NetworkBoundMode string `json:"network_bound_mode"`
}

// PreviewComposeCache does not download, build, copy installed trees or invoke
// uv. Unknown byte/peak forecasts stay unknown; offline hits need no network.
func PreviewComposeCache(ctx context.Context, name string, target binaryfetch.AcquisitionTarget, resourceRoot string) ([]ComposePartPreview, error) {
	paths, err := resourceStoragePaths(name)
	if err != nil {
		return nil, err
	}
	cache := composeStepCache{root: filepath.Join(paths.CacheDir, "compose"), resource: name}
	var previews []ComposePartPreview
	for _, step := range target.Compose {
		if step.Kind != "url" && step.Kind != "python-wheels" {
			continue
		}
		key, err := cache.key(step, resourceRoot)
		if err != nil {
			return nil, err
		}
		_, hit, err := resourcedeployment.InspectCachedArtifacts(ctx, cache.root, key)
		if err != nil {
			return nil, err
		}
		preview := ComposePartPreview{Role: step.Role, Kind: step.Kind, CacheState: "miss", SourceIdentity: key.SourceIdentity, NetworkBoundMode: "offline"}
		if hit {
			zero := int64(0)
			preview.CacheState = "hit"
			preview.NetworkBytes = &zero
			preview.NetworkCapEnforceable = true
		}
		previews = append(previews, preview)
	}
	return previews, nil
}
