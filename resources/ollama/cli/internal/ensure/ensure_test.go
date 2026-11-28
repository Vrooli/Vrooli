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
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/runtimeprofile"
)

type fakeOllama struct {
	installed    map[string]bool
	pullScripts  map[string][]string
	failPull     map[string]string
	pullCalls    int32
	tagCalls     int32
	pullInsecure bool
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func (f *fakeOllama) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.tagCalls, 1)
		if r.Method != http.MethodGet {
			t.Errorf("tags: unexpected method %s", r.Method)
		}
		type model struct {
			Name   string `json:"name"`
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
		}
		out := struct {
			Models []model `json:"models"`
		}{}
		for name, ok := range f.installed {
			if ok {
				out.Models = append(out.Models, model{Name: name, Size: 64, Digest: "sha256:fixture"})
			}
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.pullCalls, 1)
		if r.Method != http.MethodPost {
			t.Errorf("pull: unexpected method %s", r.Method)
		}
		var body struct {
			Name     string `json:"name"`
			Stream   bool   `json:"stream"`
			Insecure bool   `json:"insecure"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("pull: decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.pullInsecure = body.Insecure
		if msg, shouldFail := f.failPull[body.Name]; shouldFail {
			_, _ = w.Write([]byte(`{"status":"pulling","error":"` + msg + `"}` + "\n"))
			return
		}
		script := f.pullScripts[body.Name]
		if len(script) == 0 {
			script = []string{`{"status":"pulling"}`, `{"status":"success"}`}
		}
		flusher, _ := w.(http.Flusher)
		successful := false
		for _, line := range script {
			_, _ = w.Write([]byte(line + "\n"))
			var progress PullProgress
			if json.Unmarshal([]byte(line), &progress) == nil && strings.EqualFold(progress.Status, "success") {
				successful = true
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if successful {
			if f.installed == nil {
				f.installed = map[string]bool{}
			}
			f.installed[body.Name] = true
		}
	})
	return mux
}

func newTestClient(t *testing.T, fake *fakeOllama) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(fake.handler(t))
	return &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}, srv
}

type testPullLease struct {
	expectedRef   string
	progressErr   error
	verifyErr     error
	cleanupErr    error
	provenanceErr error
	remoteErr     error
	progress      int
	verified      bool
	cleanup       bool
	provenance    bool
	completed     bool
	abandoned     bool
}

func (l *testPullLease) PullReference() string                 { return l.expectedRef }
func (l *testPullLease) LeaseID() string                       { return "fixture-lease" }
func (l *testPullLease) Install(context.Context, string) error { return nil }
func (l *testPullLease) WorkContext() context.Context          { return context.Background() }
func (l *testPullLease) Abandon()                              { l.abandoned = true }
func (l *testPullLease) VerifyRemote(context.Context) error    { return l.remoteErr }

func (l *testPullLease) ObserveProgress(context.Context, PullProgress) error {
	l.progress++
	return l.progressErr
}
func (l *testPullLease) VerifyInstalled(_ context.Context, model ModelInfo) error {
	if l.verifyErr != nil {
		return l.verifyErr
	}
	if model.Name != l.expectedRef && model.Name != withLatestTag(l.expectedRef) {
		return fmt.Errorf("unexpected installed model %q", model.Name)
	}
	l.verified = true
	return nil
}
func (l *testPullLease) VerifyCleanup(context.Context) error {
	if l.cleanupErr != nil {
		return l.cleanupErr
	}
	l.cleanup = true
	return nil
}
func (l *testPullLease) RecordInstallProvenance(_ context.Context, model ModelInfo) error {
	if l.provenanceErr != nil {
		return l.provenanceErr
	}
	if !l.cleanup || model.Digest == "" || model.Size <= 0 {
		return errors.New("provenance requires a verified cleaned-up install")
	}
	l.provenance = true
	return nil
}
func (l *testPullLease) Complete(context.Context) error { l.completed = true; return nil }

// testPullAdmission returns a fake lease; Run still owns pull, verification,
// cleanup ordering, and the decision to complete.
func testPullAdmission(_ context.Context, ref string) (PullLease, error) {
	return &testPullLease{expectedRef: ref}, nil
}

func runWithPullAdmission(ctx context.Context, cfg Config, client *Client, stdout io.Writer, validator AdmissionValidator, pullAdmission PullAdmission) error {
	prepare := func(context.Context, *Client) (func(), error) { return func() {}, nil }
	return runWithPullAdmissionAndEndpoint(ctx, cfg, client, stdout, validator, prepare, pullAdmission)
}

func TestRun_NoModelsIsNoop(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	var buf bytes.Buffer
	if err := Run(context.Background(), Config{}, client, &buf, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(buf.String(), "nothing to do") {
		t.Errorf("expected no-op log, got %q", buf.String())
	}
	if fake.tagCalls != 0 {
		t.Errorf("tags should not be queried when no models requested, got %d", fake.tagCalls)
	}
}

func TestRun_AllModelsAlreadyInstalled(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{"qwen3:4b": true, "nomic-embed-text:latest": true}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}, {Name: "nomic-embed-text"}}}
	var buf bytes.Buffer
	if err := Run(context.Background(), cfg, client, &buf, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.pullCalls != 0 {
		t.Errorf("should not have pulled anything, got %d pulls", fake.pullCalls)
	}
	if !strings.Contains(buf.String(), "already installed") {
		t.Errorf("expected already-installed log, got %q", buf.String())
	}
}

func TestRun_MissingModelWithoutAdmissionRefusesBeforePull(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	err := Run(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "missing:latest"}}}, client, &bytes.Buffer{}, nil)
	if err == nil || !strings.Contains(err.Error(), "shared disk-job admission is unavailable") {
		t.Fatalf("Run error = %v, want missing-admission refusal", err)
	}
	if got := atomic.LoadInt32(&fake.pullCalls); got != 0 {
		t.Fatalf("pull requests = %d, want none before admission", got)
	}
}

func TestRun_MissingModelRequiresLoopbackDaemonBeforeAdmission(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	local, srv := newTestClient(t, fake)
	defer srv.Close()
	localURL := local.BaseURL
	client := &Client{BaseURL: "http://192.0.2.10:11434", HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		proxyReq := r.Clone(r.Context())
		proxyReq.URL.Host = strings.TrimPrefix(localURL, "http://")
		proxyReq.URL.Scheme = "http"
		proxyReq.Host = proxyReq.URL.Host
		return http.DefaultTransport.RoundTrip(proxyReq)
	})}}
	admitted := false
	admission := func(context.Context, string) (PullLease, error) {
		admitted = true
		return &testPullLease{expectedRef: "model:latest"}, nil
	}
	err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, &bytes.Buffer{}, nil, admission)
	if err == nil || !strings.Contains(err.Error(), "requires the managed local Ollama endpoint") {
		t.Fatalf("Run error=%v, want remote endpoint refusal", err)
	}
	if admitted || fake.pullCalls != 0 {
		t.Fatalf("remote endpoint proceeded: admitted=%t pulls=%d", admitted, fake.pullCalls)
	}
}

func TestRun_MissingModelWithoutLifecyclePreparationRefusesBeforeAdmission(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()
	admitted := false
	admission := func(context.Context, string) (PullLease, error) {
		admitted = true
		return &testPullLease{expectedRef: "model:latest"}, nil
	}
	err := runWithPullAdmissionAndEndpoint(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, &bytes.Buffer{}, nil, nil, admission)
	if err == nil || !strings.Contains(err.Error(), "lifecycle lock and endpoint preparation") {
		t.Fatalf("Run error = %v, want lifecycle preparation refusal", err)
	}
	if admitted || fake.pullCalls != 0 {
		t.Fatalf("admitted=%v pull requests=%d; expected refusal before admission and pull", admitted, fake.pullCalls)
	}
}

func TestRun_MissingModelRequiresLifecycleLockReleaseHandle(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()
	admitted := false
	prepare := func(context.Context, *Client) (func(), error) { return nil, nil }
	admission := func(context.Context, string) (PullLease, error) {
		admitted = true
		return &testPullLease{expectedRef: "model:latest"}, nil
	}
	err := runWithPullAdmissionAndEndpoint(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, &bytes.Buffer{}, nil, prepare, admission)
	if err == nil || !strings.Contains(err.Error(), "no lock release handle") {
		t.Fatalf("Run error = %v, want missing lock release refusal", err)
	}
	if admitted || fake.pullCalls != 0 {
		t.Fatalf("admitted=%v pull requests=%d; expected refusal before admission and pull", admitted, fake.pullCalls)
	}
}

func TestRun_PullsOnlyMissing(t *testing.T) {
	fake := &fakeOllama{
		installed: map[string]bool{"qwen3:4b": true},
		pullScripts: map[string][]string{
			"nomic-embed-text:latest": {`{"status":"pulling manifest"}`, `{"status":"success"}`},
		},
	}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}, {Name: "nomic-embed-text:latest"}}}
	var lease *testPullLease
	admission := func(_ context.Context, ref string) (PullLease, error) {
		lease = &testPullLease{expectedRef: ref}
		return lease, nil
	}
	var buf bytes.Buffer
	if err := runWithPullAdmission(context.Background(), cfg, client, &buf, nil, admission); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if lease == nil || !lease.verified || !lease.cleanup || !lease.provenance || !lease.completed {
		t.Fatalf("install evidence lifecycle = %+v; want verified, cleaned, recorded, and complete", lease)
	}
	if fake.pullCalls != 1 {
		t.Errorf("expected exactly one pull, got %d", fake.pullCalls)
	}
	if !fake.pullInsecure {
		t.Error("pinned pull did not enable the loopback registry transport")
	}
	out := buf.String()
	if !strings.Contains(out, "pull nomic-embed-text:latest OK") {
		t.Errorf("expected success log for missing model, got %q", out)
	}
	if strings.Contains(out, "qwen3:4b: ") && strings.Contains(out, "pulling") {
		t.Errorf("should not have streamed progress for already-installed qwen3:4b: %q", out)
	}
}

func TestRun_ProvenanceReceiptFailureDoesNotCompleteDiskLease(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}, pullScripts: map[string][]string{
		"model:latest": {`{"status":"success"}`},
	}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()
	lease := &testPullLease{expectedRef: "model:latest", provenanceErr: errors.New("provenance store unavailable")}
	admission := func(context.Context, string) (PullLease, error) { return lease, nil }
	err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, io.Discard, nil, admission)
	if err == nil || !strings.Contains(err.Error(), "provenance store unavailable") {
		t.Fatalf("ensure error = %v, want provenance write failure", err)
	}
	if !lease.verified || !lease.cleanup || lease.provenance || lease.completed || !lease.abandoned {
		t.Fatalf("provenance failure lifecycle = %+v; lease must remain active for owner recovery", lease)
	}
}

func TestRun_ManagedDaemonIdentityFailurePreventsPullAndKeepsLease(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()
	lease := &testPullLease{expectedRef: "model:latest", remoteErr: errors.New("managed daemon identity changed")}
	admission := func(context.Context, string) (PullLease, error) { return lease, nil }
	err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, &bytes.Buffer{}, nil, admission)
	if err == nil || !strings.Contains(err.Error(), "managed daemon identity changed") {
		t.Fatalf("Run error = %v, want identity refusal", err)
	}
	if fake.pullCalls != 0 {
		t.Fatalf("pull requests = %d, want zero", fake.pullCalls)
	}
	if lease.completed || !lease.abandoned {
		t.Fatalf("lease completed=%v abandoned=%v; uncertain admission must remain active", lease.completed, lease.abandoned)
	}
}

func TestRun_HoldsManagedLifecycleLockForMissingModelWork(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()
	lockDir := t.TempDir()
	lockPath := func(string) (string, error) { return filepath.Join(lockDir, "ollama.lock"), nil }
	prepare := func(ctx context.Context, _ *Client) (func(), error) {
		return acquireManagedOllamaLifecycleLockWithPath(ctx, lockPath)
	}
	admission := func(ctx context.Context, _ string) (PullLease, error) {
		probeCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
		defer cancel()
		if release, err := artifactlock.AcquireWithPathContext(probeCtx, ollamaManagedLifecycleLockKey, lockPath); err == nil {
			release()
			t.Fatal("managed lifecycle lock was not held while admission ran")
		} else if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("competing lifecycle lock error = %v, want context deadline", err)
		}
		return nil, errors.New("fixture admission refusal")
	}
	if err := runWithPullAdmissionAndEndpoint(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:latest"}}}, client, &bytes.Buffer{}, nil, prepare, admission); err == nil {
		t.Fatal("Run succeeded after fixture admission refusal")
	}
	probeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	release, err := artifactlock.AcquireWithPathContext(probeCtx, ollamaManagedLifecycleLockKey, lockPath)
	if err != nil {
		t.Fatalf("managed lifecycle lock was not released when ensure returned: %v", err)
	}
	release()
	if fake.pullCalls != 0 {
		t.Fatalf("pull requests = %d, want zero", fake.pullCalls)
	}
}

func TestRun_PullFailureIsReported(t *testing.T) {
	fake := &fakeOllama{
		installed: map[string]bool{},
		failPull:  map[string]string{"broken:1.0": "manifest not found"},
	}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "broken:1.0"}, {Name: "later:1.0"}}}
	var buf bytes.Buffer
	lease := &testPullLease{expectedRef: "broken:1.0"}
	admission := func(context.Context, string) (PullLease, error) { return lease, nil }
	err := runWithPullAdmission(context.Background(), cfg, client, &buf, nil, admission)
	if err == nil {
		t.Fatal("expected error from failed pull")
	}
	if !strings.Contains(err.Error(), "1 model pull(s) failed") {
		t.Errorf("error missing aggregate count: %v", err)
	}
	if !strings.Contains(buf.String(), "FAILED") {
		t.Errorf("expected FAILED log, got %q", buf.String())
	}
	if fake.pullCalls != 1 {
		t.Errorf("pull requests = %d, want stop after uncertain first pull", fake.pullCalls)
	}
	if lease.verified || lease.cleanup || lease.completed {
		t.Fatalf("failed pull advanced lease lifecycle: verified=%t cleanup=%t complete=%t", lease.verified, lease.cleanup, lease.completed)
	}
}

func TestRun_ProgressAdmissionFailureRetainsLeaseAndStopsSequence(t *testing.T) {
	fake := &fakeOllama{
		installed: map[string]bool{},
		pullScripts: map[string][]string{
			"first:1": {`{"status":"downloading","digest":"sha256:first","total":10,"completed":1}`, `{"status":"success"}`},
		},
	}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	lease := &testPullLease{expectedRef: "first:1", progressErr: fmt.Errorf("sample denied")}
	admission := func(context.Context, string) (PullLease, error) { return lease, nil }
	cfg := Config{Models: []policy.DirectModelRequest{{Name: "first:1"}, {Name: "second:1"}}}
	err := runWithPullAdmission(context.Background(), cfg, client, &bytes.Buffer{}, nil, admission)
	if err == nil || !strings.Contains(err.Error(), "sample admitted pull progress") {
		t.Fatalf("Run error = %v, want progress admission failure", err)
	}
	if got := atomic.LoadInt32(&fake.pullCalls); got != 1 {
		t.Fatalf("pull requests = %d, want stop after sample failure", got)
	}
	if lease.verified || lease.cleanup || lease.completed {
		t.Fatalf("sample failure advanced lease lifecycle: verified=%t cleanup=%t complete=%t", lease.verified, lease.cleanup, lease.completed)
	}
}

func TestRun_AdmissionDenialRefusesBeforePull(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()
	admission := func(context.Context, string) (PullLease, error) {
		return nil, errors.New("disk floor denied")
	}
	err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "large:1"}}}, client, &bytes.Buffer{}, nil, admission)
	if err == nil || !strings.Contains(err.Error(), "disk floor denied") {
		t.Fatalf("Run error = %v, want admission denial", err)
	}
	if got := atomic.LoadInt32(&fake.pullCalls); got != 0 {
		t.Fatalf("pull requests = %d, want none after admission denial", got)
	}
}

func TestRun_IncompletePullRetainsLease(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
	}{
		{name: "malformed", line: `{"status":`},
		{name: "missing terminal success", line: `{"status":"downloading","digest":"sha256:partial","total":100,"completed":20}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeOllama{installed: map[string]bool{}, pullScripts: map[string][]string{"model:1": {tc.line}}}
			client, srv := newTestClient(t, fake)
			defer srv.Close()
			lease := &testPullLease{expectedRef: "model:1"}
			admission := func(context.Context, string) (PullLease, error) { return lease, nil }
			err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:1"}}}, client, &bytes.Buffer{}, nil, admission)
			if err == nil {
				t.Fatal("expected incomplete pull to fail")
			}
			if lease.verified || lease.cleanup || lease.completed {
				t.Fatalf("incomplete pull advanced lease: verified=%t cleanup=%t complete=%t", lease.verified, lease.cleanup, lease.completed)
			}
		})
	}
}

func TestRun_CleanupFailureRetainsLease(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}, pullScripts: map[string][]string{"model:1": {`{"status":"success"}`}}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()
	lease := &testPullLease{expectedRef: "model:1", cleanupErr: errors.New("partial files remain")}
	admission := func(context.Context, string) (PullLease, error) { return lease, nil }
	err := runWithPullAdmission(context.Background(), Config{Models: []policy.DirectModelRequest{{Name: "model:1"}}}, client, &bytes.Buffer{}, nil, admission)
	if err == nil || !strings.Contains(err.Error(), "partial files remain") {
		t.Fatalf("Run error = %v, want cleanup failure", err)
	}
	if !lease.verified || lease.completed {
		t.Fatalf("cleanup failure lifecycle: verified=%t completed=%t", lease.verified, lease.completed)
	}
}

func TestRun_ContextCancelAbortsPull(t *testing.T) {
	// Pull handler writes slowly so ctx cancel triggers mid-stream.
	slowMux := http.NewServeMux()
	slowMux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}{})
	})
	slowMux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 10; i++ {
			_, err := fmt.Fprintf(w, `{"status":"pulling"}`+"\n")
			if err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			select {
			case <-r.Context().Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	})
	srv := httptest.NewServer(slowMux)
	defer srv.Close()
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	lease := &testPullLease{expectedRef: "slow"}
	admission := func(context.Context, string) (PullLease, error) { return lease, nil }
	err := runWithPullAdmission(ctx, Config{Models: []policy.DirectModelRequest{{Name: "slow"}}}, client, &bytes.Buffer{}, nil, admission)
	if err == nil {
		t.Fatal("expected error when ctx is cancelled")
	}
	if lease.verified || lease.cleanup || lease.completed || !lease.abandoned {
		t.Fatalf("cancelled uncertain pull advanced lease: verified=%t cleanup=%t complete=%t", lease.verified, lease.cleanup, lease.completed)
	}
}

func TestParseConfig_AcceptsStringsAndObjects(t *testing.T) {
	raw := []byte(`{"model_roles": ["embedding.default", {"role":"chat.default","reason":"answer synthesis","required":false}], "models": ["qwen3:4b", {"name":"nomic-embed-text","tag":"latest","reason":"fixture","owner":"test"}]}`)
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.ModelRoles) != 2 {
		t.Fatalf("expected 2 model roles, got %d", len(cfg.ModelRoles))
	}
	if cfg.ModelRoles[0].Role != "embedding.default" {
		t.Errorf("model_roles[0] = %#v", cfg.ModelRoles[0])
	}
	if cfg.ModelRoles[1].IsRequired() {
		t.Errorf("model_roles[1] required = true, want false")
	}
	if len(cfg.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(cfg.Models))
	}
	if got := cfg.Models[0].Ref(); got != "qwen3:4b" {
		t.Errorf("model[0] ref = %q, want qwen3:4b", got)
	}
	if got := cfg.Models[1].Ref(); got != "nomic-embed-text:latest" {
		t.Errorf("model[1] ref = %q, want nomic-embed-text:latest", got)
	}
}

func TestRun_ResolvesRolesBeforePulling(t *testing.T) {
	fake := &fakeOllama{
		installed: map[string]bool{},
		pullScripts: map[string][]string{
			"nomic-embed-text:latest": {`{"status":"pulling manifest"}`, `{"status":"success"}`},
		},
	}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "embedding.default"}}}
	var buf bytes.Buffer
	if err := runWithPullAdmission(context.Background(), cfg, client, &buf, nil, testPullAdmission); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.pullCalls != 1 {
		t.Fatalf("expected exactly one pull, got %d", fake.pullCalls)
	}
	out := buf.String()
	if !strings.Contains(out, "resolved role embedding.default -> nomic-embed-text:latest") {
		t.Fatalf("missing role resolution log: %q", out)
	}
	if !strings.Contains(out, "pull nomic-embed-text:latest OK") {
		t.Fatalf("missing pull success log: %q", out)
	}
}

func TestResolveBaseURL_Defaults(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "")
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("OLLAMA_PORT", "")
	if got := resolveBaseURL(); got != "http://127.0.0.1:11434" {
		t.Errorf("default base URL = %q", got)
	}
	t.Setenv("OLLAMA_HOST", "box")
	t.Setenv("OLLAMA_PORT", "9000")
	if got := resolveBaseURL(); got != "http://box:9000" {
		t.Errorf("host+port base URL = %q", got)
	}
	t.Setenv("OLLAMA_HOST", "box:8080")
	if got := resolveBaseURL(); got != "http://box:8080" {
		t.Errorf("host-with-port base URL = %q", got)
	}
	t.Setenv("OLLAMA_BASE_URL", "http://override:11/")
	if got := resolveBaseURL(); got != "http://override:11" {
		t.Errorf("override base URL = %q", got)
	}
}

func TestValidateLoopbackBaseURL(t *testing.T) {
	for _, tc := range []struct {
		name string
		url  string
		ok   bool
	}{
		{name: "IPv4 loopback", url: "http://127.0.0.9:11434", ok: true},
		{name: "IPv6 loopback", url: "https://[::1]:11434/", ok: true},
		{name: "localhost", url: "http://localhost:11434", ok: true},
		{name: "other DNS hostname", url: "http://ollama.example:11434"},
		{name: "remote IP", url: "http://192.0.2.10:11434"},
		{name: "credentials", url: "http://user:secret@127.0.0.1:11434"},
		{name: "path escape", url: "http://127.0.0.1:11434/api"},
		{name: "query", url: "http://127.0.0.1:11434/?target=remote"},
		{name: "malformed", url: "://127.0.0.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateLoopbackBaseURL(tc.url)
			if tc.ok {
				if err != nil || got == "" {
					t.Fatalf("validateLoopbackBaseURL(%q) = %q, %v", tc.url, got, err)
				}
				return
			}
			if err == nil || got != "" {
				t.Fatalf("validateLoopbackBaseURL(%q) = %q, %v, want refusal", tc.url, got, err)
			}
		})
	}
}

func TestNewLoopbackClientDisablesProxyAndRedirectFollowing(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:11434")
	client, err := NewLoopbackClient()
	if err != nil {
		t.Fatalf("NewLoopbackClient() error = %v", err)
	}
	transport, ok := client.HTTP.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatalf("loopback transport = %#v, want direct transport with no proxy", client.HTTP.Transport)
	}
	if err := client.HTTP.CheckRedirect(&http.Request{}, nil); err != http.ErrUseLastResponse {
		t.Fatalf("redirect policy = %v, want ErrUseLastResponse", err)
	}
}

func TestGeneratePassesSamplingAndProfileVariantOnly(t *testing.T) {
	var got struct {
		Model   string         `json:"model"`
		Prompt  string         `json:"prompt"`
		Images  []string       `json:"images"`
		Stream  bool           `json:"stream"`
		Think   *bool          `json:"think"`
		Options map[string]any `json:"options"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("path = %q, want /api/generate", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		done := true
		_ = json.NewEncoder(w).Encode(GenerateResponse{Response: "ok", Done: &done, EvalCount: 42, TotalDuration: 100, LoadDuration: 20, PromptEvalCount: 3, PromptEvalDuration: 30, EvalDuration: 40})
	}))
	defer srv.Close()

	maxTokens := 123
	temperature := 0.25
	think := false
	profile := runtimeprofile.Current()
	cpuOnly, err := profile.VariantOptions(runtimeprofile.VariantCPUOnly)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}
	resp, err := client.Generate(context.Background(), GenerateRequest{
		Model:       "llama3.2:1b",
		Prompt:      "hello",
		Think:       &think,
		Images:         []string{"aGVsbG8="},
		NumPredict:     &maxTokens,
		Temperature:    &temperature,
		RuntimeOptions: cpuOnly,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Response != "ok" || resp.EvalCount != 42 {
		t.Fatalf("response = %+v, want response ok with eval_count 42", resp)
	}
	if resp.TotalDuration != 100 || resp.LoadDuration != 20 || resp.PromptEvalCount != 3 || resp.PromptEvalDuration != 30 || resp.EvalDuration != 40 {
		t.Fatalf("timing response = %+v", resp)
	}
	if got.Model != "llama3.2:1b" || got.Prompt != "hello" || got.Stream {
		t.Fatalf("request body = %+v", got)
	}
	if len(got.Images) != 1 || got.Images[0] != "aGVsbG8=" {
		t.Fatalf("images = %#v, want one base64 image", got.Images)
	}
	if got.Think == nil || *got.Think != false {
		t.Fatalf("think = %v, want false", got.Think)
	}
	if got.Options["num_predict"] != float64(123) {
		t.Fatalf("num_predict = %#v, want 123", got.Options["num_predict"])
	}
	if got.Options["temperature"] != 0.25 {
		t.Fatalf("temperature = %#v, want 0.25", got.Options["temperature"])
	}
	for key, value := range cpuOnly {
		if got.Options[key] != float64(value.(int)) {
			t.Fatalf("variant option %s = %#v, want %v", key, got.Options[key], value)
		}
	}
	for key := range got.Options {
		if profile.IsLoadOption(key) {
			if _, ok := cpuOnly[key]; !ok {
				t.Fatalf("generate sent load option %s outside the runtime profile variant: %#v", key, got.Options)
			}
		}
	}
}

func TestChatSendsMessagesSamplingAndNoLoadOptions(t *testing.T) {
	var got struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream  bool           `json:"stream"`
		Think   *bool          `json:"think"`
		Options map[string]any `json:"options"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q, want /api/chat", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":              map[string]string{"content": "summary"},
			"done_reason":          "stop",
			"eval_count":           17,
			"total_duration":       100,
			"load_duration":        20,
			"prompt_eval_count":    3,
			"prompt_eval_duration": 30,
			"eval_duration":        40,
		})
	}))
	defer srv.Close()

	maxTokens := 123
	temperature := 0.25
	think := false
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}
	resp, err := client.Chat(context.Background(), ChatRequest{
		Model: "chat-model",
		Messages: []ChatMessage{
			{Role: "system", Content: "be concise"},
			{Role: "user", Content: "hello"},
		},
		NumPredict:  &maxTokens,
		Temperature: &temperature,
		Think:       &think,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Message.Content != "summary" || resp.DoneReason != "stop" || resp.EvalCount != 17 {
		t.Fatalf("response = %+v", resp)
	}
	if resp.TotalDuration != 100 || resp.LoadDuration != 20 || resp.PromptEvalCount != 3 || resp.PromptEvalDuration != 30 || resp.EvalDuration != 40 {
		t.Fatalf("timing response = %+v", resp)
	}
	if got.Model != "chat-model" || got.Stream {
		t.Fatalf("request model/stream = %q/%v", got.Model, got.Stream)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Content != "hello" {
		t.Fatalf("messages = %+v", got.Messages)
	}
	if got.Think == nil || *got.Think != false {
		t.Fatalf("think = %v, want false", got.Think)
	}
	if got.Options["num_predict"] != float64(123) {
		t.Fatalf("num_predict = %#v, want 123", got.Options["num_predict"])
	}
	if got.Options["temperature"] != 0.25 {
		t.Fatalf("temperature = %#v, want 0.25", got.Options["temperature"])
	}
	for key := range got.Options {
		if runtimeprofile.Current().IsLoadOption(key) {
			t.Fatalf("chat sent runner load option %s; the daemon profile owns it: %#v", key, got.Options)
		}
	}
}

// TestRun_AdmissionGateBlocksOnValidatorFailure proves the fail-closed gate:
// when the injected validator rejects a resolved tool role, ensure returns an
// error (the model is not silently seated).
func TestRun_AdmissionGateBlocksOnValidatorFailure(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{"qwen3:4b": true}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}}}
	called := false
	validator := func(ctx context.Context, roles []string) error {
		called = true
		return fmt.Errorf("model cannot tool-call")
	}
	err := Run(context.Background(), cfg, client, &bytes.Buffer{}, validator)
	if !called {
		t.Fatal("validator should run after models are confirmed installed")
	}
	if err == nil || !strings.Contains(err.Error(), "admission gate") {
		t.Fatalf("expected admission-gate error, got %v", err)
	}
}

// TestRun_AdmissionGatePassesWhenValidatorOK confirms a passing validator does
// not block ensure.
func TestRun_AdmissionGatePassesWhenValidatorOK(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{"qwen3:4b": true}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}}}
	err := Run(context.Background(), cfg, client, &bytes.Buffer{}, func(context.Context, []string) error { return nil })
	if err != nil {
		t.Fatalf("passing validator must not block ensure: %v", err)
	}
}

// Metadata must survive the real HTTP decoder without manufacturing completion
// when the provider omits it. Model is only the provider's raw name echo.
func TestGeneratePreservesOptionalCompletionMetadata(t *testing.T) {
	for _, tc := range []struct {
		name     string
		metadata string
		want     map[string]string
		wantErr  bool
	}{
		{name: "stop", metadata: `,"done":true,"done_reason":"stop","model":"provider:echo"`, want: map[string]string{"done": "true", "done_reason": `"stop"`, "model": `"provider:echo"`}},
		{name: "length", metadata: `,"done":true,"done_reason":"length","model":"provider:echo"`, want: map[string]string{"done": "true", "done_reason": `"length"`, "model": `"provider:echo"`}},
		{name: "reported_false", metadata: `,"done":false,"done_reason":"provider_pause","model":" raw echo "`, want: map[string]string{"done": "false", "done_reason": `"provider_pause"`, "model": `" raw echo "`}},
		{name: "missing", want: map[string]string{}},
		{name: "null_unknown", metadata: `,"done":null,"done_reason":null,"model":null`, want: map[string]string{}},
		{name: "reported_empty", metadata: `,"done":false,"done_reason":"","model":""`, want: map[string]string{"done": "false", "done_reason": `""`, "model": `""`}},
		{name: "malformed_done", metadata: `,"done":"yes"`, wantErr: true},
		{name: "malformed_reason", metadata: `,"done_reason":123`, wantErr: true},
		{name: "malformed_model", metadata: `,"model":{"name":"untrusted"}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"response":"visible unchanged","thinking":"synthetic private field must not be exported","eval_count":7,"total_duration":100,"load_duration":20,"prompt_eval_count":3,"prompt_eval_duration":30,"eval_duration":40` + tc.metadata + `}`
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/generate" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write([]byte(body))
			}))
			t.Cleanup(srv.Close)
			client := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
			got, err := client.Generate(context.Background(), GenerateRequest{Model: "requested:model", Prompt: "literal input"})
			if tc.wantErr {
				if err == nil {
					t.Fatal("malformed reported metadata must fail decoding")
				}
				return
			}
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got.Response != "visible unchanged" || got.EvalCount != 7 || got.TotalDuration != 100 || got.LoadDuration != 20 || got.PromptEvalCount != 3 || got.PromptEvalDuration != 30 || got.EvalDuration != 40 {
				t.Fatalf("visible text/counts/durations changed: %+v", got)
			}
			raw, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			if _, exists := fields["thinking"]; exists {
				t.Fatal("private thinking field was exported")
			}
			for _, key := range []string{"done", "done_reason", "model"} {
				want, present := tc.want[key]
				value, exists := fields[key]
				if exists != present || (present && string(value) != want) {
					t.Errorf("%s = %s (present=%t), want %s (present=%t)", key, value, exists, want, present)
				}
			}
		})
	}
}
