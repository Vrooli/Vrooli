package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git-control-tower/handlers/repo"
	repodomain "git-control-tower/internal/repo"
	"git-control-tower/internal/worktree"
)

// This opt-in proving ground runs the production UI aggregator in Chrome
// against the real generated RepoService Connect handler. Its only Git tree
// is created below t.TempDir and removed by the Go test harness.
func TestRepoStatusBrowserConsumerTraversesRetriesAndShowsError(t *testing.T) {
	if os.Getenv("GCT_STATUS_BROWSER_E2E") != "1" {
		t.Skip("set GCT_STATUS_BROWSER_E2E=1 to run the opt-in Chrome + generated Connect proof")
	}
	repoPath := makeStatusBrowserFixture(t)
	control := &statusBrowserControl{repoPath: repoPath}
	control.mutationsRemaining.Store(1)
	apiServer := newStatusBrowserAPIServer(t, repoPath, control)
	viteURL, stopVite := startStatusBrowserVite(t)
	t.Cleanup(stopVite)
	proxy := httptestServer(statusBrowserProxy(t, apiServer.URL, viteURL, control))
	t.Cleanup(proxy.Close)

	uiDir, err := filepath.Abs(filepath.Join("..", "ui"))
	if err != nil {
		t.Fatal(err)
	}
	pythonScript := filepath.Join(uiDir, "e2e", "status-pagination.browser.py")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", pythonScript, proxy.URL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Chrome consumer proof failed: %v (initial=%d pages=%d failed=%d mutations=%d; first response %s)\n%s", err, control.initialRequests.Load(), control.successfulPages.Load(), control.failedContinuations.Load(), control.mutationSequence.Load(), control.firstResponseDebug(), output)
	}
	t.Logf("Chrome consumer proof: %s", strings.TrimSpace(string(output)))
	if got := control.initialRequests.Load(); got != 5 {
		t.Fatalf("initial status requests=%d, want 5 (one changed-snapshot retry plus three visible error retries)", got)
	}
	if got := control.failedContinuations.Load(); got != 4 {
		t.Fatalf("failed continuations=%d, want 4 across success and exhausted-refresh flows", got)
	}
	if got := control.unexpectedContinuationErrors.Load(); got != 0 {
		t.Fatalf("continuations failed with errors other than FailedPrecondition: %d", got)
	}
	if got := control.oversizedResponses.Load(); got != 0 {
		t.Fatalf("generated API emitted %d pages over the response byte cap", got)
	}
	if got := control.successfulPages.Load(); got < 9 {
		t.Fatalf("only %d successful generated API pages observed; expected all pages across both flows", got)
	}
}

type statusBrowserControl struct {
	repoPath            string
	initialRequests     atomic.Int64
	failedContinuations atomic.Int64
	unexpectedContinuationErrors atomic.Int64
	successfulPages     atomic.Int64
	oversizedResponses  atomic.Int64
	mutationsRemaining  atomic.Int32
	alwaysMutate        atomic.Bool
	mutationSequence    atomic.Int64
	debugMu             sync.Mutex
	debug               string
}

func (c *statusBrowserControl) firstResponseDebug() string {
	c.debugMu.Lock()
	defer c.debugMu.Unlock()
	return c.debug
}

type statusBrowserRepoStore struct{ path string }

func (s statusBrowserRepoStore) List(context.Context) ([]RepoRecord, error) {
	return []RepoRecord{{ID: 1, Path: s.path, Name: "status-browser"}}, nil
}
func (s statusBrowserRepoStore) GetByID(_ context.Context, id int64) (*RepoRecord, error) {
	if id != 1 {
		return nil, sql.ErrNoRows
	}
	return &RepoRecord{ID: 1, Path: s.path, Name: "status-browser"}, nil
}
func (s statusBrowserRepoStore) GetByPath(_ context.Context, path string) (*RepoRecord, error) {
	if filepath.Clean(path) != filepath.Clean(s.path) {
		return nil, sql.ErrNoRows
	}
	return &RepoRecord{ID: 1, Path: s.path, Name: "status-browser"}, nil
}
func (s statusBrowserRepoStore) Upsert(_ context.Context, record RepoRecord) (RepoRecord, error) {
	return record, nil
}
func (statusBrowserRepoStore) Delete(context.Context, int64) error            { return nil }
func (statusBrowserRepoStore) GetActive(context.Context) (*RepoRecord, error) { return nil, nil }
func (statusBrowserRepoStore) SetActive(context.Context, int64) error         { return nil }
func (statusBrowserRepoStore) ClearActive(context.Context) error              { return nil }
func (statusBrowserRepoStore) TouchLastOpened(context.Context, int64) error   { return nil }

func newStatusBrowserAPIServer(t *testing.T, repoPath string, control *statusBrowserControl) *httptest.Server {
	t.Helper()
	git := &ExecGitRunner{GitPath: "git"}
	owner := &Server{git: git, repos: NewRepoService(statusBrowserRepoStore{path: repoPath}, git), configCache: NewGitConfigCache(time.Minute)}
	owner.statusCache = NewRepoStatusCache(5 * time.Second)
	owner.statusPageCache = NewRepoStatusPageCache(repoStatusPageCacheMaxBytes, repoStatusPageCacheMaxEntries)
	owner.statusCache.SetPageSnapshots(owner.statusPageCache)
	inspector := worktree.NewGitInspector(gctGitRunner{gitPath: "git"})
	path, handler := repo.NewHandler(repo.Deps{Service: repodomain.NewService(inspector), GetRepoStatus: owner.getRepoStatusConnect})
	mux := http.NewServeMux()
	mux.Handle(path, statusBrowserCapture{next: handler, control: control})
	return httptest.NewServer(mux)
}

type statusBrowserCapture struct {
	next    http.Handler
	control *statusBrowserControl
}

func (h statusBrowserCapture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/GetRepoStatus") {
		h.next.ServeHTTP(w, r)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	var request struct {
		PageToken string `json:"pageToken"`
	}
	_ = json.Unmarshal(body, &request)
	isInitial := request.PageToken == ""
	if isInitial {
		h.control.initialRequests.Add(1)
	}
	buffered := &bufferedStatusResponse{header: make(http.Header)}
	h.next.ServeHTTP(buffered, r)
	responseBody := buffered.body.Bytes()
	if buffered.header.Get("Content-Encoding") == "gzip" {
		reader, gzipErr := gzip.NewReader(bytes.NewReader(responseBody))
		if gzipErr == nil {
			decompressed, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr == nil {
				responseBody = decompressed
			}
		}
	}
	var response struct {
		Page *struct {
			NextPageToken string `json:"nextPageToken"`
			ResponseBytes uint32 `json:"responseBytes"`
		} `json:"page"`
	}
	_ = json.Unmarshal(responseBody, &response)
	if isInitial {
		sample := buffered.body.String()
		if len(sample) > 400 {
			sample = sample[:400]
		}
		h.control.debugMu.Lock()
		h.control.debug = fmt.Sprintf("status=%d content-type=%q body=%q", buffered.status, buffered.header.Get("Content-Type"), sample)
		h.control.debugMu.Unlock()
	}
	if buffered.status >= 200 && buffered.status < 300 && response.Page != nil {
		h.control.successfulPages.Add(1)
		if response.Page.ResponseBytes > repoStatusResponseBudget || buffered.body.Len() > repoStatusResponseBudget {
			h.control.oversizedResponses.Add(1)
		}
	}
	if !isInitial && (buffered.status < 200 || buffered.status >= 300) {
		var failure struct { Code string `json:"code"` }
		_ = json.Unmarshal(responseBody, &failure)
		if failure.Code == "failed_precondition" {
			h.control.failedContinuations.Add(1)
		} else {
			h.control.unexpectedContinuationErrors.Add(1)
		}
	}
	if isInitial && response.Page != nil && response.Page.NextPageToken != "" && h.shouldMutate() {
		sequence := h.control.mutationSequence.Add(1)
		name := "unstable-refresh-" + strconv.FormatInt(sequence, 10) + ".txt"
		if sequence == 1 {
			name = "refresh-added-after-first-page.txt"
		}
		if err := os.WriteFile(filepath.Join(h.control.repoPath, name), []byte("visible status refresh\n"), 0600); err != nil {
			h.control.oversizedResponses.Add(1)
		}
	}
	buffered.flush(w)
}

func (h statusBrowserCapture) shouldMutate() bool {
	if h.control.alwaysMutate.Load() {
		return true
	}
	for {
		remaining := h.control.mutationsRemaining.Load()
		if remaining <= 0 {
			return false
		}
		if h.control.mutationsRemaining.CompareAndSwap(remaining, remaining-1) {
			return true
		}
	}
}

type bufferedStatusResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *bufferedStatusResponse) Header() http.Header    { return w.header }
func (w *bufferedStatusResponse) WriteHeader(status int) { w.status = status }
func (w *bufferedStatusResponse) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}
func (w *bufferedStatusResponse) flush(dst http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			dst.Header().Add(key, value)
		}
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	dst.WriteHeader(w.status)
	_, _ = dst.Write(w.body.Bytes())
}

func statusBrowserProxy(t *testing.T, apiURL, viteURL string, control *statusBrowserControl) http.Handler {
	t.Helper()
	apiTarget, err := url.Parse(apiURL)
	if err != nil {
		t.Fatal(err)
	}
	viteTarget, err := url.Parse(viteURL)
	if err != nil {
		t.Fatal(err)
	}
	apiProxy := httputil.NewSingleHostReverseProxy(apiTarget)
	viteProxy := httputil.NewSingleHostReverseProxy(viteTarget)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__test/always-mutate" && r.Method == http.MethodPost {
			control.alwaysMutate.Store(true)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/vrooli.git_control_tower.v1.repo.RepoService/") {
			apiProxy.ServeHTTP(w, r)
			return
		}
		viteProxy.ServeHTTP(w, r)
	})
}

func httptestServer(handler http.Handler) *httptest.Server { return httptest.NewServer(handler) }

func startStatusBrowserVite(t *testing.T) (string, func()) {
	t.Helper()
	uiDir, err := filepath.Abs(filepath.Join("..", "ui"))
	if err != nil {
		t.Fatal(err)
	}
	portListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := portListener.Addr().(*net.TCPAddr).Port
	_ = portListener.Close()
	viteBin := filepath.Join(uiDir, "node_modules", ".bin", "vite")
	cmd := exec.Command(viteBin, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--strictPort")
	cmd.Dir = uiDir
	logs := &lockedStatusBuffer{}
	cmd.Stdout, cmd.Stderr = logs, logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start local Vite browser harness: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	urlPattern := regexp.MustCompile(`http://127\.0\.0\.1:[0-9]+/`)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if match := urlPattern.FindString(logs.String()); match != "" {
			return strings.TrimRight(match, "/"), func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				select {
				case <-done:
				case <-time.After(5 * time.Second):
				}
			}
		}
		select {
		case err := <-done:
			t.Fatalf("Vite exited before ready: %v\n%s", err, logs.String())
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	t.Fatalf("Vite did not report its local URL: %s", logs.String())
	return "", func() {}
}

type lockedStatusBuffer struct {
	mu   sync.Mutex
	body bytes.Buffer
}

func (b *lockedStatusBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.body.Write(p)
}
func (b *lockedStatusBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.body.String()
}

func makeStatusBrowserFixture(t *testing.T) string {
	t.Helper()
	repoPath := t.TempDir()
	mustGitPaging(t, repoPath, "init", "-q")
	mustGitPaging(t, repoPath, "config", "user.name", "GCT browser fixture")
	mustGitPaging(t, repoPath, "config", "user.email", "gct-browser@example.invalid")
	for i := 0; i < 4096; i++ {
		writePagingFile(t, repoPath, fmt.Sprintf("tracked-%06d.txt", i), "base line\n")
	}
	mustGitPaging(t, repoPath, "add", ".")
	mustGitPaging(t, repoPath, "commit", "-qm", "browser status baseline")
	for i := 0; i < 1024; i++ {
		writePagingFile(t, repoPath, fmt.Sprintf("tracked-%06d.txt", i), "base line\nchanged line\n")
		writePagingFile(t, repoPath, fmt.Sprintf("untracked-%06d.txt", i), "untracked line\n")
	}
	return repoPath
}
