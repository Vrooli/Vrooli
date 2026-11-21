package fetch

import (
	"context"
	"errors"
	"fmt"
	"github.com/vrooli/vrooli/packages/capacity"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProbeSourceSizeUsesBoundedMetadataWithoutDownloadingArtifact(t *testing.T) {
	bodyBytes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "7340032")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			if got := r.Header.Get("Range"); got != "bytes=0-0" {
				t.Errorf("GET Range=%q", got)
			}
			w.Header().Set("Content-Range", "bytes 0-0/7340032")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("x"))
			bodyBytes++
			return
		}
		t.Errorf("unexpected probe method %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	got, err := ProbeSourceSize(context.Background(), Source{DownloadURL: server.URL + "/weights.bin"})
	if err != nil {
		t.Fatal(err)
	}
	if got != 7340032 {
		t.Fatalf("size=%d, want 7340032", got)
	}
	if bodyBytes != 0 {
		t.Fatalf("probe read artifact bytes after successful HEAD: %d", bodyBytes)
	}
}

type growthTestLease struct {
	extended []capacity.DiskJobPath
	reject   bool
}

func (l *growthTestLease) Heartbeat(context.Context) error { return nil }
func (l *growthTestLease) Observe(context.Context) (capacity.DiskJobHeadroom, error) {
	return capacity.DiskJobHeadroom{}, nil
}
func (l *growthTestLease) Extend(_ context.Context, paths []capacity.DiskJobPath) (capacity.DiskJobReceipt, error) {
	if l.reject {
		return capacity.DiskJobReceipt{}, errors.New("floor would be violated")
	}
	l.extended = append([]capacity.DiskJobPath(nil), paths...)
	return capacity.DiskJobReceipt{}, nil
}
func (l *growthTestLease) SamplePath(context.Context, string, int64, int64) (capacity.DiskJobReceipt, error) {
	return capacity.DiskJobReceipt{}, nil
}
func (l *growthTestLease) Complete(context.Context) (capacity.DiskJobReceipt, error) {
	return capacity.DiskJobReceipt{}, nil
}
func (l *growthTestLease) Release(ctx context.Context) error { _, err := l.Complete(ctx); return err }
func (l *growthTestLease) Reservation() capacity.DiskJobReceipt {
	return capacity.DiskJobReceipt{Status: "active"}
}

func TestDiskGrowthObserverExtendsOrRefusesBeforeAdditionalWrites(t *testing.T) {
	request := capacity.DiskJobRequest{Paths: []capacity.DiskJobPath{{Path: t.TempDir(), Role: "model-adapter-staging-and-final", PeakBytes: 4096}, {Path: t.TempDir(), Role: "installer-and-model-tool-temporary", PeakBytes: 128 << 20}}}
	lease := &growthTestLease{}
	guard, err := NewDiskGrowthGuard(lease, request, 1, InstallOwnerCeilingBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.Observe(context.Background(), "artifact", 4097, -1); err != nil {
		t.Fatal(err)
	}
	if len(lease.extended) != 2 || lease.extended[0].PeakBytes <= 4096 {
		t.Fatalf("expected an extension before continued writes, got %+v", lease.extended)
	}
	lease.reject = true
	if err := guard.Observe(context.Background(), "artifact", 9000, -1); err == nil || !strings.Contains(err.Error(), "extension denied") {
		t.Fatalf("denied extension error=%v", err)
	}
}

func TestObservedHTTPDownloaderRefusesChunkBeforePersistingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(strings.Repeat("x", 64<<10))) }))
	defer server.Close()
	dest := t.TempDir() + "/blocked.bin"
	denied := errors.New("disk lease extension denied")
	err := (HTTPDownloader{}).DownloadObserved(context.Background(), server.URL, dest, func(done, total int64) error {
		if done > 0 {
			return denied
		}
		return nil
	})
	if !errors.Is(err, denied) {
		t.Fatalf("download error=%v", err)
	}
	info, statErr := os.Stat(dest)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Size() != 0 {
		t.Fatalf("refused chunk was persisted: %d bytes", info.Size())
	}
}

func TestInstallPlanIncludesAndPinsDefaultHuggingFaceCache(t *testing.T) {
	t.Setenv("HF_HOME", "")
	t.Setenv("HF_HUB_CACHE", "")
	t.Setenv("HUGGINGFACE_HUB_CACHE", "")
	t.Setenv("TRANSFORMERS_CACHE", "")
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-1", t.TempDir(), 8, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	cache := HuggingFaceCachePath(request)
	if !strings.HasSuffix(cache, "/.cache/huggingface/hub") {
		t.Fatalf("default hub cache=%q", cache)
	}
	for _, role := range []string{"huggingface-home-cache", "huggingface-hub-cache"} {
		found := false
		for _, path := range request.Paths {
			if path.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("disk plan omitted %s", role)
		}
	}
	fetcher, ok := PinHuggingFaceCache(&HFSnapshotFetcher{Python: "python3"}, cache).(*HFSnapshotFetcher)
	if !ok {
		t.Fatal("expected pinned production snapshot fetcher")
	}
	joined := strings.Join(fetcher.Env, "\n")
	if !strings.Contains(joined, "HF_HUB_CACHE="+cache) || !strings.Contains(joined, "HUGGINGFACE_HUB_CACHE="+cache) {
		t.Fatalf("snapshot fetcher did not use admitted cache path %q", cache)
	}
}

func TestInstallPlanMergesSameHomeAndHubCacheWithLargestPeak(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "shared-cache")
	alias := filepath.Join(root, "shared-cache-alias")
	if err := os.Mkdir(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(cache, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HF_HOME", alias)
	t.Setenv("HF_HUB_CACHE", cache)
	t.Setenv("HUGGINGFACE_HUB_CACHE", "")
	t.Setenv("TRANSFORMERS_CACHE", "")
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-shared-cache", t.TempDir(), 8, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	var cachePath *capacity.DiskJobPath
	for i := range request.Paths {
		if request.Paths[i].Path == cache {
			cachePath = &request.Paths[i]
			break
		}
	}
	if cachePath == nil || cachePath.PeakBytes < 8<<20 || !diskJobPathHasRole(cachePath.Role, "huggingface-home-cache") || !diskJobPathHasRole(cachePath.Role, "huggingface-hub-cache") {
		t.Fatalf("shared home/hub cache plan lost its full bound or role: %+v", request.Paths)
	}

	lease := &growthTestLease{}
	guard, err := NewDiskGrowthGuard(lease, request, 1, InstallOwnerCeilingBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.Observe(context.Background(), "artifact", 16<<20+1, -1); err != nil {
		t.Fatal(err)
	}
	foundExtension := false
	for _, path := range lease.extended {
		if path.Path == cache && path.PeakBytes >= 32<<20 && diskJobPathHasRole(path.Role, "huggingface-hub-cache") {
			foundExtension = true
		}
	}
	if !foundExtension {
		t.Fatalf("growth extension omitted merged Hub cache bound: %+v", lease.extended)
	}
}

func TestInstallPlanSumsDistinctWriteRolesWhenModelRootIsCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HF_HOME", root)
	t.Setenv("HF_HUB_CACHE", root)
	t.Setenv("HUGGINGFACE_HUB_CACHE", "")
	t.Setenv("TRANSFORMERS_CACHE", "")
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-colocated-cache", root, 8, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Paths) != 2 {
		t.Fatalf("co-located path plan=%+v, want one combined root plus temp path", request.Paths)
	}
	var rootPath *capacity.DiskJobPath
	for i := range request.Paths {
		if request.Paths[i].Path == root {
			rootPath = &request.Paths[i]
		}
	}
	if rootPath == nil || rootPath.PeakBytes < 24<<20 || !diskJobPathHasRole(rootPath.Role, "model-adapter-staging-and-final") || !diskJobPathHasRole(rootPath.Role, "huggingface-hub-cache") {
		t.Fatalf("co-located writes did not sum their role peaks: %+v", request.Paths)
	}
	if cache := HuggingFaceCachePath(request); cache != root {
		t.Fatalf("merged Hub cache path=%q, want %q", cache, root)
	}

	lease := &growthTestLease{}
	guard, err := NewDiskGrowthGuard(lease, request, 1, InstallOwnerCeilingBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.Observe(context.Background(), "artifact", 24<<20+1, -1); err != nil {
		t.Fatal(err)
	}
	var rootExtension *capacity.DiskJobPath
	for i := range lease.extended {
		if lease.extended[i].Path == root {
			rootExtension = &lease.extended[i]
		}
	}
	if len(lease.extended) != 2 || rootExtension == nil || rootExtension.PeakBytes < 96<<20 {
		t.Fatalf("co-located root+cache growth extension=%+v", lease.extended)
	}
}

func TestProbeSourceSizeRejectsUnboundedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(3<<20))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 2<<20))
	}))
	defer server.Close()
	if _, err := ProbeSourceSize(context.Background(), Source{DownloadURL: server.URL + "/weights.bin"}); err == nil {
		t.Fatal("probe without a bounded range/content length was accepted")
	}
}
