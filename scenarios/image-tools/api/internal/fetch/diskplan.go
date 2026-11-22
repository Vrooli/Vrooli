package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/diskpolicy"
	"image-tools/internal/modelenv"
)

const (
	InstallOwnerCeilingBytes = int64(25_000_000_000)
	// HuggingFaceCacheAllowanceBytes bounds the Hub cache charge. Repo
	// snapshots use snapshot_download(local_dir=staging), which writes files
	// and incomplete downloads under the staging directory (charged to the
	// model root); the Hub cache only receives metadata and any small configs a
	// runtime smoke resolves.
	HuggingFaceCacheAllowanceBytes = int64(512 << 20)
	sizeProbeTimeout               = 5 * time.Second
	sizeProbeBodyLimit             = 1 << 20
)

// PlanInstallDiskJob creates a bounded peak sized by the model's real bytes
// for one model or adapter install: the artifact once at the model root (staging
// is renamed into place on the same filesystem) plus 10% slack, bounded tool
// temporaries and a bounded Hub cache allowance (library caches are pinned under
// the managed root, see internal/modelenv). It is admitted by shared host disk
// admission against the floor internal/diskpolicy resolves. A missing catalog
// size invokes bounded public metadata probes before any acquisition begins; an
// unknown or over-ceiling result is rejected before the caller writes. Growth
// beyond the slack extends the lease before the next chunk is written.
func PlanInstallDiskJob(ctx context.Context, ownerID, jobID, root string, sizeMB int, source Source, buildExpansion int) (sharedcapacity.DiskJobRequest, error) {
	if strings.TrimSpace(ownerID) == "" || strings.TrimSpace(jobID) == "" || strings.TrimSpace(root) == "" {
		return sharedcapacity.DiskJobRequest{}, errors.New("image-tools disk plan requires owner, job, and model root")
	}
	if source.LocalPath != "" {
		return sharedcapacity.DiskJobRequest{}, errors.New("local-path artifacts do not write an install copy and do not need a disk acquisition lease")
	}
	size := int64(0)
	if sizeMB > 0 {
		if int64(sizeMB) > math.MaxInt64/(1<<20) {
			return sharedcapacity.DiskJobRequest{}, errors.New("catalog size overflows bytes")
		}
		size = int64(sizeMB) << 20
	}
	for _, a := range source.Assets {
		if a.MinBytes > size {
			size = a.MinBytes
		}
	}
	// The fixed owner ceiling bounds an estimate the catalog does not declare
	// (a bounded metadata probe). A catalog-declared size is a reviewed fact: its
	// derived peak is bounded by that declaration, by the growth guard (which
	// refuses writes past the planned root peak beyond the owner ceiling), and by
	// the shared host disk admission (free space above the diskpolicy floor).
	// Capping declared sizes too made every catalog model above ~7.7 GB
	// uninstallable regardless of free disk.
	declared := size > 0
	if size <= 0 {
		probed, err := ProbeSourceSize(ctx, source)
		if err != nil {
			return sharedcapacity.DiskJobRequest{}, fmt.Errorf("bounded size probe failed before acquisition: %w", err)
		}
		size = probed
	}
	if size <= 0 {
		return sharedcapacity.DiskJobRequest{}, errors.New("size probe returned no positive artifact size")
	}
	if !declared && size > InstallOwnerCeilingBytes {
		return sharedcapacity.DiskJobRequest{}, fmt.Errorf("image-tools install estimate %d bytes exceeds the 25 GB owner planning ceiling", size)
	}
	if size > math.MaxInt64/2 {
		return sharedcapacity.DiskJobRequest{}, errors.New("install peak overflows bytes")
	}
	// Staging lives beside the destination and is renamed into place on the
	// same filesystem, so the root writes one copy; a replaced install already
	// occupies its bytes. Ten percent covers approximate catalog sizes; the
	// growth guard extends the lease before any byte beyond it is written.
	rootPeak := size + max(size/10, int64(64<<20))
	if buildExpansion > 1 {
		if rootPeak > math.MaxInt64/int64(buildExpansion) {
			return sharedcapacity.DiskJobRequest{}, errors.New("model build peak overflows bytes")
		}
		rootPeak *= int64(buildExpansion)
	}
	// Installers hash and verify in place; the temp directory only holds tool
	// scratch (archive listings, partial index files), never a second copy.
	tempPeak := max(int64(128<<20), min(size/16, int64(2<<30)))
	canonicalRoot, err := canonicalDiskJobPath(root)
	if err != nil {
		return sharedcapacity.DiskJobRequest{}, fmt.Errorf("resolve model install root for disk admission: %w", err)
	}
	canonicalTemp, err := canonicalDiskJobPath(os.TempDir())
	if err != nil {
		return sharedcapacity.DiskJobRequest{}, fmt.Errorf("resolve temporary directory for disk admission: %w", err)
	}
	paths := []sharedcapacity.DiskJobPath{{Path: canonicalRoot, Role: "model-adapter-staging-and-final", PeakBytes: rootPeak}, {Path: canonicalTemp, Role: "installer-and-model-tool-temporary", PeakBytes: tempPeak}}
	seenCache := map[string]int{}
	addCache := func(path, role string, peak int64) error {
		if strings.TrimSpace(path) == "" {
			return nil
		}
		canonical, err := canonicalDiskJobPath(path)
		if err != nil {
			return err
		}
		if index, found := seenCache[canonical]; found {
			if peak > paths[index].PeakBytes {
				paths[index].PeakBytes = peak
			}
			if !diskJobPathHasRole(paths[index].Role, role) {
				paths[index].Role = strings.Trim(paths[index].Role+","+role, ",")
			}
		} else {
			seenCache[canonical] = len(paths)
			paths = append(paths, sharedcapacity.DiskJobPath{Path: canonical, Role: role, PeakBytes: peak})
		}
		return nil
	}
	// Library caches are pinned under the managed root (internal/modelenv), so
	// the admitted cache charge is that directory, never an inherited HF_HOME.
	pins := modelenv.MustLoad().Pins(modelenv.Root())
	for _, item := range []struct{ path, role string }{
		{pins["HF_HOME"], "huggingface-home-cache"}, {pins["HF_HUB_CACHE"], "huggingface-hub-cache"},
	} {
		peak := min(size, HuggingFaceCacheAllowanceBytes)
		if item.role == "huggingface-home-cache" {
			peak = 1
		}
		if err := addCache(item.path, item.role, peak); err != nil {
			return sharedcapacity.DiskJobRequest{}, err
		}
	}
	paths, err = sumDuplicateDiskJobPaths(paths)
	if err != nil {
		return sharedcapacity.DiskJobRequest{}, err
	}

	total := int64(0)
	for _, p := range paths {
		if total > math.MaxInt64-p.PeakBytes {
			return sharedcapacity.DiskJobRequest{}, errors.New("install peak overflows bytes")
		}
		total += p.PeakBytes
	}
	if !declared && total > InstallOwnerCeilingBytes {
		return sharedcapacity.DiskJobRequest{}, fmt.Errorf("complete model/adapter install peak %d bytes exceeds the 25 GB owner planning ceiling", total)
	}
	return diskpolicy.Resolve(os.Getenv).Request(ownerID, jobID, paths), nil
}

func canonicalDiskJobPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(abs)
	var suffix []string
	for {
		info, statErr := os.Stat(candidate)
		if statErr == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("disk admission path %s is not a directory", path)
			}
			resolved, resolveErr := filepath.EvalSymlinks(candidate)
			if resolveErr != nil {
				return "", resolveErr
			}
			return filepath.Clean(filepath.Join(append([]string{resolved}, suffix...)...)), nil
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", fmt.Errorf("disk admission path %s has no existing parent", path)
		}
		suffix = append([]string{filepath.Base(candidate)}, suffix...)
		candidate = parent
	}
}

func sumDuplicateDiskJobPaths(paths []sharedcapacity.DiskJobPath) ([]sharedcapacity.DiskJobPath, error) {
	byPath := make(map[string]sharedcapacity.DiskJobPath, len(paths))
	for _, path := range paths {
		old, found := byPath[path.Path]
		if found {
			if old.PeakBytes > math.MaxInt64-path.PeakBytes {
				return nil, fmt.Errorf("disk admission peak overflows for canonical path %s", path.Path)
			}
			old.PeakBytes += path.PeakBytes
			if !diskJobPathHasRole(old.Role, path.Role) {
				old.Role = strings.Trim(old.Role+","+path.Role, ",")
			}
			byPath[path.Path] = old
		} else {
			byPath[path.Path] = path
		}
	}
	out := make([]sharedcapacity.DiskJobPath, 0, len(byPath))
	for _, path := range byPath {
		out = append(out, path)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// PinHuggingFaceCache returns a copy of the production repository fetcher with
// all Hugging Face cache paths redirected to the admitted filesystem.
func PinHuggingFaceCache(repo RepoFetcher, cachePath string) RepoFetcher {
	fetcher, ok := repo.(*HFSnapshotFetcher)
	if !ok || strings.TrimSpace(cachePath) == "" {
		return repo
	}
	clone := *fetcher
	values := make(map[string]string)
	baseEnv := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	for _, item := range append(baseEnv, fetcher.Env...) {
		key, value, found := strings.Cut(item, "=")
		if found {
			values[key] = value
		}
	}
	values["HF_HOME"] = filepath.Dir(cachePath)
	values["HF_HUB_CACHE"] = cachePath
	values["HUGGINGFACE_HUB_CACHE"] = cachePath
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	clone.Env = make([]string, 0, len(keys))
	for _, key := range keys {
		clone.Env = append(clone.Env, key+"="+values[key])
	}
	return &clone
}

// GrowthCeilingBytes bounds lease extensions for a planned request: twice the
// plan, and never less than the fixed probe ceiling. Shared admission still
// refuses any extension the host cannot hold.
func GrowthCeilingBytes(request sharedcapacity.DiskJobRequest) int64 {
	var total int64
	for _, p := range request.Paths {
		if total > math.MaxInt64/2-p.PeakBytes {
			return math.MaxInt64 / 2
		}
		total += p.PeakBytes
	}
	return max(InstallOwnerCeilingBytes, 2*total)
}

// HuggingFaceCachePath selects the cache declared by the admitted request.
func HuggingFaceCachePath(request sharedcapacity.DiskJobRequest) string {
	for _, path := range request.Paths {
		if diskJobPathHasRole(path.Role, "huggingface-hub-cache") {
			return path.Path
		}
	}
	return ""
}

// ProbeSourceSize performs a bounded metadata-only probe. It reads no artifact
// bodies beyond one byte when a server ignores Range, and caps repository
// manifests to one MiB and five seconds.
func ProbeSourceSize(ctx context.Context, source Source) (int64, error) {
	probeCtx, cancel := context.WithTimeout(ctx, sizeProbeTimeout)
	defer cancel()
	client := &http.Client{Timeout: sizeProbeTimeout}
	var total int64
	if source.HasRepo() {
		repoBytes, err := probeRepoSize(probeCtx, client, source.Repo)
		if err != nil {
			return 0, err
		}
		total = repoBytes
	}
	assets, err := source.ResolvedAssets()
	if err != nil {
		return 0, err
	}
	if len(assets) == 0 && !source.HasRepo() {
		return 0, ErrNoDownloadSource
	}
	for _, asset := range assets {
		size, err := probeURLSize(probeCtx, client, asset.URL)
		if err != nil {
			return 0, fmt.Errorf("probe %s: %w", asset.Filename, err)
		}
		if total > math.MaxInt64-size {
			return 0, errors.New("artifact size sum overflows bytes")
		}
		total += size
	}
	return total, nil
}

func probeURLSize(ctx context.Context, client *http.Client, rawURL string) (int64, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" && u.Scheme != "http" {
		return 0, fmt.Errorf("artifact URL must use HTTP(S)")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return 0, err
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 && response.ContentLength > 0 {
		return response.ContentLength, nil
	}
	request, err = http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Range", "bytes=0-0")
	response, err = client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1))
	if value := response.Header.Get("Content-Range"); value != "" {
		if i := strings.LastIndex(value, "/"); i >= 0 {
			n, parseErr := strconv.ParseInt(value[i+1:], 10, 64)
			if parseErr == nil && n > 0 {
				return n, nil
			}
		}
	}
	if response.StatusCode == http.StatusPartialContent && response.ContentLength > 0 {
		return response.ContentLength, nil
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 && response.ContentLength == 1 {
		return 1, nil
	}
	return 0, fmt.Errorf("server did not provide a bounded content length (status %d)", response.StatusCode)
}

func probeRepoSize(ctx context.Context, client *http.Client, repo RepoSpec) (int64, error) {
	if strings.TrimSpace(repo.RepoID) == "" || strings.TrimSpace(repo.Revision) == "" {
		return 0, ErrRepoRevisionUnpinned
	}
	u := "https://huggingface.co/api/models/" + url.PathEscape(repo.RepoID) + "/tree/" + url.PathEscape(repo.Revision) + "?recursive=true&expand=true"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, fmt.Errorf("metadata endpoint returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, sizeProbeBodyLimit+1))
	if err != nil {
		return 0, err
	}
	if len(body) > sizeProbeBodyLimit {
		return 0, fmt.Errorf("repository size metadata exceeds %d-byte probe limit", sizeProbeBodyLimit)
	}
	var files []struct {
		Type string `json:"type"`
		Size int64  `json:"size"`
	}
	if err := json.Unmarshal(body, &files); err != nil {
		return 0, err
	}
	var total int64
	for _, file := range files {
		if file.Type == "directory" || file.Size <= 0 {
			continue
		}
		if total > math.MaxInt64-file.Size {
			return 0, errors.New("repository size sum overflows bytes")
		}
		total += file.Size
	}
	if total <= 0 {
		return 0, errors.New("repository metadata contains no file sizes")
	}
	return total, nil
}
