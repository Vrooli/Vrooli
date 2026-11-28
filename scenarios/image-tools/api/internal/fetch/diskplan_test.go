package fetch

import (
	"context"
	"errors"
	"fmt"
	"github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/diskpolicy"
	"image-tools/internal/modelenv"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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
	if err := guard.Observe(context.Background(), "artifact", 2<<30, -1); err == nil || !strings.Contains(err.Error(), "extension denied") {
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

// useManagedRoot gives a test an isolated managed model root; library caches
// are pinned beneath it, never to an inherited HF_HOME or the home directory.
func useManagedRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(modelenv.SwapRoot(root))
	return root
}

func TestInstallPlanChargesTheManagedHubCacheNotAnInheritedOne(t *testing.T) {
	managed := useManagedRoot(t)
	// An operator's shell exporting its own cache must not move the charge.
	t.Setenv("HF_HOME", filepath.Join(t.TempDir(), "operator-cache"))
	t.Setenv("HF_HUB_CACHE", filepath.Join(t.TempDir(), "operator-hub"))
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-1", t.TempDir(), 8, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantHub := filepath.Join(managed, "cache", "huggingface", "hub")
	if cache := HuggingFaceCachePath(request); cache != wantHub {
		t.Fatalf("hub cache=%q, want the managed %q", cache, wantHub)
	}
	for _, role := range []string{"huggingface-home-cache", "huggingface-hub-cache"} {
		found := false
		for _, path := range request.Paths {
			if diskJobPathHasRole(path.Role, role) {
				found = true
				if !strings.HasPrefix(path.Path, managed) {
					t.Fatalf("%s charged outside the managed root: %s", role, path.Path)
				}
			}
		}
		if !found {
			t.Fatalf("disk plan omitted %s", role)
		}
	}
	fetcher, ok := PinHuggingFaceCache(&HFSnapshotFetcher{Python: "python3"}, wantHub).(*HFSnapshotFetcher)
	if !ok {
		t.Fatal("expected pinned production snapshot fetcher")
	}
	joined := strings.Join(fetcher.Env, "\n")
	if !strings.Contains(joined, "HF_HUB_CACHE="+wantHub) || !strings.Contains(joined, "HUGGINGFACE_HUB_CACHE="+wantHub) {
		t.Fatalf("snapshot fetcher did not use admitted cache path %q", wantHub)
	}
}

func TestInstallPlanGrowthExtendsTheManagedHubCacheBound(t *testing.T) {
	managed := useManagedRoot(t)
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-growth", managed, 8, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	hub := filepath.Join(managed, "cache", "huggingface", "hub")
	lease := &growthTestLease{}
	guard, err := NewDiskGrowthGuard(lease, request, 1, InstallOwnerCeilingBytes)
	if err != nil {
		t.Fatal(err)
	}
	// Writing past the planned root peak (8 MiB + 64 MiB slack) extends before the next chunk.
	if err := guard.Observe(context.Background(), "artifact", 80<<20, -1); err != nil {
		t.Fatal(err)
	}
	foundExtension, rootGrown := false, false
	for _, path := range lease.extended {
		if path.Path == hub && diskJobPathHasRole(path.Role, "huggingface-hub-cache") && path.PeakBytes > 8<<20 {
			foundExtension = true
		}
		if diskJobPathHasRole(path.Role, "model-adapter-staging-and-final") && path.PeakBytes >= 80<<20 {
			rootGrown = true
		}
	}
	if !foundExtension || !rootGrown {
		t.Fatalf("growth extension must track observed bytes at the root and keep the Hub cache bound: %+v", lease.extended)
	}
}

// The extension follows what was actually written (a quarter beyond, at least
// 1 GiB) rather than jumping to a multiple of it.
func TestGrowthExtensionTracksObservedBytes(t *testing.T) {
	managed := useManagedRoot(t)
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-track", managed, 100, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	lease := &growthTestLease{}
	guard, err := NewDiskGrowthGuard(lease, request, 1, InstallOwnerCeilingBytes)
	if err != nil {
		t.Fatal(err)
	}
	observed := int64(10 << 30)
	if err := guard.Observe(context.Background(), "artifact", observed, -1); err != nil {
		t.Fatal(err)
	}
	for _, path := range lease.extended {
		if diskJobPathHasRole(path.Role, "model-adapter-staging-and-final") {
			if path.PeakBytes < observed || path.PeakBytes > observed+observed/4+(1<<30) {
				t.Fatalf("root extension %d for %d observed bytes is not tracking real bytes", path.PeakBytes, observed)
			}
		}
	}
}

func TestInstallPlanFloorIsTheSharedPolicyAndSmallWritesUseTheSmallClass(t *testing.T) {
	managed := useManagedRoot(t)
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-floor", managed, 8, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if request.Class != "" || request.FloorBytes != diskpolicy.HostMinimumFloorBytes {
		t.Fatalf("an 8 MiB install writes more than a small write and is held to the host floor, got class=%q floor=%d", request.Class, request.FloorBytes)
	}
	t.Setenv(diskpolicy.EnvFloor, "200GiB")
	request, err = PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-floor", managed, 8, Source{DownloadURL: "https://models.example/weights.bin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if request.FloorBytes != 200<<30 {
		t.Fatalf("operator floor not applied: %d", request.FloorBytes)
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

// A catalog-declared size above the probe ceiling is planned (and left to the
// shared host disk admission), while an undeclared probed estimate stays capped.
func TestInstallPlanBoundsDeclaredSizesByHostAdmissionNotTheProbeCeiling(t *testing.T) {
	useManagedRoot(t)
	const declaredMB = 23000 // a large diffusers tree
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-large", t.TempDir(), declaredMB,
		Source{Repo: RepoSpec{RepoID: "org/large", Revision: strings.Repeat("a", 40)}}, 1)
	if err != nil {
		t.Fatalf("declared 16 GB model must plan: %v", err)
	}
	var total int64
	for _, path := range request.Paths {
		total += path.PeakBytes
	}
	if total <= InstallOwnerCeilingBytes {
		t.Fatalf("expected a planned peak above the probe ceiling, got %d", total)
	}
	if request.FloorBytes != diskpolicy.HostMinimumFloorBytes {
		t.Fatalf("host admission floor = %d, want %d", request.FloorBytes, diskpolicy.HostMinimumFloorBytes)
	}

	oversized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.FormatInt(InstallOwnerCeilingBytes+1, 10))
	}))
	defer oversized.Close()
	if _, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-probed", t.TempDir(), 0,
		Source{DownloadURL: oversized.URL + "/weights.bin"}, 1); err == nil || !strings.Contains(err.Error(), "owner planning ceiling") {
		t.Fatalf("undeclared probed estimate above the ceiling must be refused, got %v", err)
	}
}

func TestInstallPlanChargesOneStagedCopyAndBoundedHubCache(t *testing.T) {
	useManagedRoot(t)
	const declaredMB = 7350 // instruct-pix2pix diffusers tree (~7.7 GB)
	size := int64(declaredMB) << 20
	request, err := PlanInstallDiskJob(context.Background(), "image-tools:model-install", "job-ip2p", t.TempDir(), declaredMB,
		Source{Repo: RepoSpec{RepoID: "org/ip2p", Revision: strings.Repeat("b", 40)}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, path := range request.Paths {
		total += path.PeakBytes
		if diskJobPathHasRole(path.Role, "huggingface-hub-cache") && path.PeakBytes > HuggingFaceCacheAllowanceBytes {
			t.Fatalf("hub cache charged %d bytes; local_dir snapshots write under staging", path.PeakBytes)
		}
		if diskJobPathHasRole(path.Role, "model-adapter-staging-and-final") && path.PeakBytes != size+size/10 {
			t.Fatalf("root peak %d, want one staged copy with 10%% slack %d", path.PeakBytes, size+size/10)
		}
	}
	// One copy, 10% slack, bounded scratch and the bounded cache: about 1.2×,
	// not the former 3.25× (25,025,000,001 bytes for this model).
	if total > size*125/100 {
		t.Fatalf("planned peak %d for a %d-byte model is inflated", total, size)
	}
	if ceiling := GrowthCeilingBytes(request); ceiling != max(InstallOwnerCeilingBytes, 2*total) {
		t.Fatalf("growth ceiling %d, want twice the %d-byte plan or the probe ceiling", ceiling, total)
	}
	small := capacity.DiskJobRequest{Paths: []capacity.DiskJobPath{{PeakBytes: 1 << 30}}}
	if GrowthCeilingBytes(small) != InstallOwnerCeilingBytes {
		t.Fatalf("a small plan keeps the probe ceiling as its growth floor")
	}
	large := capacity.DiskJobRequest{Paths: []capacity.DiskJobPath{{PeakBytes: 30_000_000_000}}}
	if GrowthCeilingBytes(large) != 60_000_000_000 {
		t.Fatalf("growth ceiling must scale with a large declared plan")
	}
}
