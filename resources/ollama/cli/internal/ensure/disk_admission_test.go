package ensure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

type fakeDiskAdmitter struct {
	request capacity.DiskJobRequest
	lease   *fakeDiskJobLease
	err     error
}

func (f *fakeDiskAdmitter) ReserveDiskJob(_ context.Context, req capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	f.request = req
	if f.err != nil {
		return nil, f.err
	}
	if f.lease.status == "" {
		f.lease.status = "active"
	}
	return f.lease, nil
}

type fakeDiskJobLease struct {
	heartbeatErr error
	sampleErr    error
	heartbeats   int
	samples      []int64
	extensions   []int64
	completed    bool
	released     bool
	status       string
}

func (f *fakeDiskJobLease) Heartbeat(context.Context) error {
	f.heartbeats++
	return f.heartbeatErr
}
func (f *fakeDiskJobLease) Observe(context.Context) (capacity.DiskJobHeadroom, error) {
	return capacity.DiskJobHeadroom{}, nil
}
func (f *fakeDiskJobLease) Extend(context.Context, []capacity.DiskJobPath) (capacity.DiskJobReceipt, error) {
	return capacity.DiskJobReceipt{}, nil
}
func (f *fakeDiskJobLease) SamplePath(_ context.Context, _ string, bytes, largerPeak int64) (capacity.DiskJobReceipt, error) {
	f.samples = append(f.samples, bytes)
	f.extensions = append(f.extensions, largerPeak)
	return capacity.DiskJobReceipt{}, f.sampleErr
}
func (f *fakeDiskJobLease) Complete(context.Context) (capacity.DiskJobReceipt, error) {
	f.completed = true
	f.status = "completed"
	return capacity.DiskJobReceipt{}, nil
}
func (f *fakeDiskJobLease) Release(context.Context) error {
	f.released = true
	f.status = "released"
	return nil
}
func (f *fakeDiskJobLease) Reservation() capacity.DiskJobReceipt {
	return capacity.DiskJobReceipt{}
}

func validTestManifest() []byte {
	return testManifest(testDigest("config"), testDigest("layer"), 2, 3)
}

func TestConfiguredModelRootUsesResourceStorageAndRejectsCallerDataDir(t *testing.T) {
	home := t.TempDir()
	managedData := filepath.Join(home, "custom-data")
	callerData := filepath.Join(home, "scenario-data")
	values := map[string]string{
		"XDG_DATA_HOME":     managedData,
		"RESOURCE_DATA_DIR": callerData,
	}
	getenv := func(key string) string { return values[key] }
	want := filepath.Join(managedData, "vrooli", "resources", "ollama", "models")
	got, err := configuredModelRoot(getenv, home, "linux", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("configured model root=%q, want resource-owned path %q", got, want)
	}

	values["OLLAMA_MODELS"] = want
	if got, err := configuredModelRoot(getenv, home, "linux", false); err != nil || got != want {
		t.Fatalf("matching OLLAMA_MODELS override: root=%q err=%v", got, err)
	}
	values["OLLAMA_MODELS"] = filepath.Join(callerData, "models")
	if _, err := configuredModelRoot(getenv, home, "linux", false); err == nil {
		t.Fatal("accepted caller scenario's OLLAMA_MODELS path")
	}
}

func newDiskAdmissionFixture(t *testing.T) (*DiskAdmission, *fakeDiskAdmitter, *fakeDiskJobLease, string) {
	t.Helper()
	root := t.TempDir()
	lease := &fakeDiskJobLease{}
	broker := &fakeDiskAdmitter{lease: lease}
	a := &DiskAdmission{
		Broker: broker,
		Root:   root,
		Client: &Client{},
		Fetch: preflightFunc(func(context.Context, string, string) ([]byte, error) {
			return validTestManifest(), nil
		}),
	}
	return a, broker, lease, root
}

func TestDiskAdmissionReservesBeforePullWithBoundedPeakAndBaseline(t *testing.T) {
	a, broker, lease, root := newDiskAdmissionFixture(t)
	pull, err := a.Admit(context.Background(), "model:latest")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pull.Abandon)
	if broker.request.OwnerID != ollamaDiskOwner || len(broker.request.Paths) != 1 || broker.request.Paths[0].Path != root {
		t.Fatalf("reservation request=%+v", broker.request)
	}
	wantPeak := int64(5) + manifestWriteAllowance + 2*metadataPerBlobAllowance
	if broker.request.Paths[0].PeakBytes != wantPeak || broker.request.FloorBytes != 0 {
		t.Fatalf("reserved peak/floor=%d/%d, want %d/0", broker.request.Paths[0].PeakBytes, broker.request.FloorBytes, wantPeak)
	}
	if len(lease.samples) != 1 || lease.samples[0] != 0 {
		t.Fatalf("baseline samples=%v, want one zero-byte sample", lease.samples)
	}
	if pull.PullReference() == "model:latest" {
		t.Fatal("pull was not redirected through immutable local manifest pin")
	}
}

func TestDiskAdmissionRefusesUnknownRootAndManifestBeforeReservation(t *testing.T) {
	a, broker, _, _ := newDiskAdmissionFixture(t)
	a.Root = filepath.Join(a.Root, "missing")
	if _, err := a.Admit(context.Background(), "model:latest"); err == nil {
		t.Fatal("unknown model root was admitted")
	}
	if len(broker.request.Paths) != 0 {
		t.Fatal("reserved capacity for unknown model root")
	}
	a, broker, _, _ = newDiskAdmissionFixture(t)
	a.Fetch = preflightFunc(func(context.Context, string, string) ([]byte, error) {
		return testManifest(testDigest("config"), testDigest("layer"), 0, 3), nil
	})
	if _, err := a.Admit(context.Background(), "model:latest"); err == nil {
		t.Fatal("unknown descriptor size was admitted")
	}
	if len(broker.request.Paths) != 0 {
		t.Fatal("reserved capacity for unknown model size")
	}
}

func TestDiskAdmissionRefusesMissingSharedBroker(t *testing.T) {
	a, _, _, _ := newDiskAdmissionFixture(t)
	a.Broker = nil
	if _, err := a.Admit(context.Background(), "model:latest"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("Admit error = %v, want missing shared broker refusal", err)
	}
}

func TestDiskAdmissionDenialAndSampleFailureLeaveLeaseIncomplete(t *testing.T) {
	a, broker, _, _ := newDiskAdmissionFixture(t)
	broker.err = errors.New("floor denial")
	if _, err := a.Admit(context.Background(), "model:latest"); err == nil {
		t.Fatal("broker denial was ignored")
	}
	a, _, lease, root := newDiskAdmissionFixture(t)
	lease.sampleErr = errors.New("sample denied")
	pull, err := a.Admit(context.Background(), "model:latest")
	if err == nil || pull == nil {
		t.Fatalf("baseline sample failure returned lease=%v err=%v", pull, err)
	}
	pull.Abandon()
	if lease.completed {
		t.Fatal("sample failure completed the owner lease")
	}

	a, _, lease, root = newDiskAdmissionFixture(t)
	pull, err = a.Admit(context.Background(), "model:latest")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pull.Abandon)
	file, err := os.Create(filepath.Join(root, "growth"))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(brokerPeakForTest()); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	lease.sampleErr = errors.New("extension denied")
	if err := pull.ObserveProgress(context.Background(), PullProgress{}); err == nil {
		t.Fatal("sample/extension failure was ignored")
	}
	if len(lease.extensions) < 2 || lease.extensions[len(lease.extensions)-1] <= 0 || lease.completed {
		t.Fatalf("growth extension samples=%v completed=%v", lease.extensions, lease.completed)
	}
}

func TestDiskAdmissionRefusesManagedDaemonIdentityChangeAfterReservation(t *testing.T) {
	a, _, lease, _ := newDiskAdmissionFixture(t)
	want := errors.New("managed daemon identity changed")
	checks := 0
	a.VerifyDaemon = func(context.Context) error {
		checks++
		if checks == 2 {
			return want
		}
		return nil
	}
	pull, err := a.Admit(context.Background(), "model:latest")
	if pull == nil || !errors.Is(err, want) {
		t.Fatalf("Admit pull=%v err=%v; want active receipt and identity error", pull, err)
	}
	pull.Abandon()
	if lease.status != "active" || lease.completed || lease.released {
		t.Fatalf("changed daemon released its reserved lease: status=%q completed=%t released=%t", lease.status, lease.completed, lease.released)
	}
}

func TestRunWithManagedEndpointRechecksInstalledInventoryBeforeAdmission(t *testing.T) {
	caller := &fakeOllama{installed: map[string]bool{}}
	callerClient, callerServer := newTestClient(t, caller)
	defer callerServer.Close()
	managed := &fakeOllama{installed: map[string]bool{"model:1": true}}
	managedClient, managedServer := newTestClient(t, managed)
	defer managedServer.Close()
	prepared := false
	admitted := false
	prepare := func(_ context.Context, client *Client) (func(), error) {
		prepared = true
		*client = *managedClient
		return func() {}, nil
	}
	admit := func(context.Context, string) (PullLease, error) {
		admitted = true
		return nil, errors.New("admission should not run for a model already on managed Ollama")
	}
	err := runWithPullAdmissionAndEndpoint(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:1"}}}, callerClient, &bytes.Buffer{}, nil, prepare, admit)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared || admitted {
		t.Fatalf("managed endpoint prepared=%t, admission called=%t", prepared, admitted)
	}
}

func TestRunWithManagedEndpointLeavesAlreadyInstalledInventoryRouteAlone(t *testing.T) {
	installed := &fakeOllama{installed: map[string]bool{"model:1": true}}
	client, server := newTestClient(t, installed)
	defer server.Close()
	prepared := false
	prepare := func(context.Context, *Client) (func(), error) { prepared = true; return nil, nil }
	err := runWithPullAdmissionAndEndpoint(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:1"}}}, client, &bytes.Buffer{}, nil, prepare, nil)
	if err != nil {
		t.Fatal(err)
	}
	if prepared {
		t.Fatal("managed endpoint preparation changed the already-installed inventory path")
	}
}

func TestBaseURLFromOllamaHostRequiresLoopbackAndValidPort(t *testing.T) {
	for _, raw := range []string{"0.0.0.0:11434", "192.0.2.5:11434", "127.0.0.1:0", "127.0.0.1:nope", "invalid"} {
		if _, err := baseURLFromOllamaHost(raw); err == nil {
			t.Errorf("accepted managed OLLAMA_HOST %q", raw)
		}
	}
	got, err := baseURLFromOllamaHost("127.0.0.1:21434")
	if err != nil || got != "http://127.0.0.1:21434" {
		t.Fatalf("baseURLFromOllamaHost()=%q err=%v", got, err)
	}
}

func TestDiskAdmissionSamplesMonotonicPeakAcrossPartialCleanup(t *testing.T) {
	a, _, diskLease, root := newDiskAdmissionFixture(t)
	pull, err := a.Admit(context.Background(), "model:latest")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pull.Abandon)
	file := filepath.Join(root, "temporary-partial")
	if err := os.WriteFile(file, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := pull.ObserveProgress(context.Background(), PullProgress{}); err != nil {
		t.Fatal(err)
	}
	observedPeak := diskLease.samples[len(diskLease.samples)-1]
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := pull.ObserveProgress(context.Background(), PullProgress{}); err != nil {
		t.Fatalf("partial cleanup invalidated the monotonic receipt: %v", err)
	}
	if got := diskLease.samples[len(diskLease.samples)-1]; got != observedPeak {
		t.Fatalf("last sample=%d, want retained peak sample %d", got, observedPeak)
	}
}

func brokerPeakForTest() int64 {
	return int64(16<<20) + 1
}

func TestDiskAdmissionHeartbeatFailureCancelsWorkAndRetainsReceipt(t *testing.T) {
	a, _, lease, _ := newDiskAdmissionFixture(t)
	original := diskHeartbeatInterval
	diskHeartbeatInterval = time.Millisecond
	defer func() { diskHeartbeatInterval = original }()
	lease.heartbeatErr = errors.New("heartbeat denied")
	pull, err := a.Admit(context.Background(), "model:latest")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pull.Abandon)
	select {
	case <-pull.WorkContext().Done():
	case <-time.After(time.Second):
		t.Fatal("heartbeat failure did not cancel active pull")
	}
	if lease.heartbeats == 0 || lease.completed {
		t.Fatalf("heartbeats=%d completed=%v", lease.heartbeats, lease.completed)
	}
}

func TestDiskAdmissionAbandonRetainsActiveLeaseDuringStaleWorkerWrites(t *testing.T) {
	a, _, lease, root := newDiskAdmissionFixture(t)
	pull, err := a.Admit(context.Background(), "model:latest")
	if err != nil {
		t.Fatal(err)
	}
	pull.Abandon()
	// Model Ollama work that outlives the caller's canceled stream. Abandon
	// cannot prove this writer stopped, so it must not release the capacity.
	if err := os.WriteFile(filepath.Join(root, "stale-worker.partial"), []byte("still writing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if lease.status != "active" || lease.completed || lease.released {
		t.Fatalf("uncertain stale worker lease status=%q completed=%t released=%t", lease.status, lease.completed, lease.released)
	}
}

func TestDiskAdmissionRunsPinnedPullAndCompletesOnlyAfterAliasRemoval(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "blobs"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestBytes := validTestManifest()
	wantDigest := "sha256:" + sha256Hex(manifestBytes)
	models := map[string]bool{}
	var alias string
	var pullInsecure, deleted bool
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		rows := make([]ModelInfo, 0, len(models))
		for name, present := range models {
			if present {
				rows = append(rows, ModelInfo{Name: name, Size: 3, Digest: wantDigest})
			}
		}
		_ = json.NewEncoder(w).Encode(tagsResponse{Models: rows})
	})
	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Insecure bool   `json:"insecure"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		alias, pullInsecure = req.Name, req.Insecure
		models[alias] = true
		_, _ = fmt.Fprintln(w, `{"status":"success"}`)
	})
	mux.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		delete(models, req.Name)
		models["model:latest"] = true // Ollama now sees the atomically linked target manifest.
		deleted = true
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}
	job := &fakeDiskJobLease{}
	broker := &fakeDiskAdmitter{lease: job}
	a := &DiskAdmission{Broker: broker, Root: root, Client: client, Fetch: preflightFunc(func(context.Context, string, string) ([]byte, error) { return manifestBytes, nil })}
	err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, io.Discard, nil, a.Admit)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifests", RegistryHost, "library", "model", "latest")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("atomically published target manifest: %v", err)
	}
	if !pullInsecure || !deleted || alias == "" || models[alias] || !models["model:latest"] {
		t.Fatalf("pull/install lifecycle insecure=%t deleted=%t alias=%q models=%v", pullInsecure, deleted, alias, models)
	}
	if !job.completed || len(job.samples) < 2 {
		t.Fatalf("disk lease completion=%t samples=%v", job.completed, job.samples)
	}
}

func TestDiskAdmissionPreservesTargetInstalledDuringPull(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "blobs"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestBytes := validTestManifest()
	wantDigest := "sha256:" + sha256Hex(manifestBytes)
	models := map[string]bool{}
	manifestPath := filepath.Join(root, "manifests", RegistryHost, "library", "model", "latest")
	tagCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		tagCalls++
		rows := make([]ModelInfo, 0, len(models))
		for name, present := range models {
			if present {
				rows = append(rows, ModelInfo{Name: name, Size: 3, Digest: wantDigest})
			}
		}
		if tagCalls == 3 {
			// Install after the ensure pre-copy inventory but before its no-clobber
			// hard-link commit. The API response intentionally remains stale.
			if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
				t.Errorf("create concurrent manifest directory: %v", err)
			} else if err := os.WriteFile(manifestPath, []byte("concurrent model"), 0o644); err != nil {
				t.Errorf("write concurrent model manifest: %v", err)
			}
			models["model:latest"] = true
		}
		_ = json.NewEncoder(w).Encode(tagsResponse{Models: rows})
	})
	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		models[req.Name] = true
		_, _ = fmt.Fprintln(w, `{"status":"success"}`)
	})
	mux.HandleFunc("/api/delete", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}
	job := &fakeDiskJobLease{}
	a := &DiskAdmission{Broker: &fakeDiskAdmitter{lease: job}, Root: root, Client: client, Fetch: preflightFunc(func(context.Context, string, string) ([]byte, error) { return manifestBytes, nil })}
	err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, io.Discard, nil, a.Admit)
	if err == nil || !strings.Contains(err.Error(), "publish pinned Ollama manifest") {
		t.Fatalf("ensure error=%v, want no-clobber commit refusal", err)
	}
	got, readErr := os.ReadFile(manifestPath)
	if readErr != nil || string(got) != "concurrent model" || !models["model:latest"] || job.completed {
		t.Fatalf("target preservation manifest=%q err=%v target=%t lease-complete=%t", got, readErr, models["model:latest"], job.completed)
	}
}
