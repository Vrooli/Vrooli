package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	hrepo "git-control-tower/handlers/repo"
	"git-control-tower/internal/config"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/pushsafety"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/cli-core/cliutil"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
)

// Only remote inspection is substituted. Preparation, Git processes, bundles,
// storage routing, HTTP encoding and request-bound single-use intents are real.
type recoveryHTTPRunner struct {
	*ExecGitRunner
	mu     sync.Mutex
	report pushsafety.Report
}

func (r *recoveryHTTPRunner) InspectPushSafety(context.Context, string, string, string, *StoredCredential) pushsafety.Report {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.report
}

func (r *recoveryHTTPRunner) setReport(report pushsafety.Report) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report = report
}

// Given a disposable dirty repository, when a human confirms an exact preview,
// then only isolated artifacts are created and status survives remote changes.
// Called by the real-Git four-commit fixture; never points at the Vrooli checkout.
func exerciseRecoveryHTTP(t *testing.T, source string, report pushsafety.Report) {
	t.Helper()
	runner := &recoveryHTTPRunner{ExecGitRunner: &ExecGitRunner{}, report: report}
	store := newTestRepoStore(t)
	record, e := store.Upsert(context.Background(), RepoRecord{Path: source, Name: "disposable-recovery"})
	if e != nil {
		t.Fatal(e)
	}
	if e = store.SetActive(context.Background(), record.ID); e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(root, "forbidden-production")})
	leased := filepath.Join(root, "leased")
	if e = roots.InstallTestRoots(storage.Paths{DataDir: leased}, "recovery-http-test", time.Minute); e != nil {
		t.Fatal(e)
	}
	server := &Server{git: runner, repos: NewRepoService(store, runner), fileRoots: roots, intentService: policygate.NewIntentService(policygate.NewMemoryIntentStore())}
	path, handler := hrepo.NewHandler(hrepo.Deps{InspectPushSafety: server.inspectPushSafetyConnect, PreparePushRecovery: server.preparePushRecoveryConnect, GetPushRecovery: server.getPushRecoveryConnect}, connect.WithInterceptors(policygate.NewInterceptor(config.PolicyConfig{}, nil)))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.HandleFunc("/preview", server.handleMutationPreview)
	mux.HandleFunc("/intent", server.handleMutationIntent)
	var browserPassed atomic.Bool
	fixtureDir := os.Getenv("GCT_RECOVERY_BROWSER_FIXTURE")
	if fixtureDir != "" {
		if !strings.HasPrefix(fixtureDir, "/tmp/") {
			t.Fatal("browser fixture must be isolated under /tmp")
		}
		mux.Handle("/", http.FileServer(http.Dir(fixtureDir)))
		mux.HandleFunc("/test/browser-passed", func(w http.ResponseWriter, r *http.Request) {
			browserPassed.Store(true)
			w.WriteHeader(http.StatusNoContent)
		})
		mux.HandleFunc("/test/remote-moved", func(w http.ResponseWriter, r *http.Request) {
			moved := report
			moved.Fingerprint = strings.Repeat("b", 64)
			runner.setReport(moved)
			w.WriteHeader(http.StatusNoContent)
		})
	}
	// Authentication is injected only inside this test server. Production does
	// not trust this test-only header. No consumed intent is injected here.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := database.WithTestMode(r.Context())
		if kind := r.Header.Get("X-Test-Principal"); kind != "" {
			principal := policygate.Principal{Subject: "test-human", Kind: cliutil.CallerKindHuman, Verified: true}
			if kind == "agent" {
				principal.Kind = cliutil.CallerKindVrooliAgent
			}
			ctx = policygate.WithPrincipal(ctx, principal)
		}
		mux.ServeHTTP(w, r.WithContext(ctx))
	}))
	defer srv.Close()
	client := repoconnect.NewRepoServiceClient(srv.Client(), srv.URL)
	ctx := context.Background()
	for _, kind := range []string{"", "agent", "human"} {
		req := connect.NewRequest(&repov1.PreparePushRecoveryRequest{Fingerprint: report.Fingerprint})
		req.Header().Set("X-Test-Principal", kind)
		if _, e = client.PreparePushRecovery(ctx, req); e == nil {
			t.Fatalf("%q prepared without an intent", kind)
		}
	}
	if _, e = os.Stat(filepath.Join(leased, "push-recovery")); !os.IsNotExist(e) {
		t.Fatal("refused request created artifacts")
	}
	post := func(path string, in, out any) {
		t.Helper()
		data, e := json.Marshal(in)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Principal", "human")
		resp, e := srv.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("%s status %d", path, resp.StatusCode)
		}
		if e = json.NewDecoder(resp.Body).Decode(out); e != nil {
			t.Fatal(e)
		}
	}
	if fixtureDir != "" {
		exerciseRecoveryBrowser(t, srv.URL)
		if !browserPassed.Load() {
			t.Fatal("browser did not finish consent, preparation and reattachment assertions")
		}
		runner.setReport(report)
	}
	var preview MutationPreviewResponse
	post("/preview", MutationPreviewRequest{Operation: "repo.recovery.prepare", SubjectContext: report.Fingerprint}, &preview)
	var intent MutationIntentResponse
	post("/intent", MutationIntentRequest{RepositoryID: preview.RepositoryID, Operation: preview.Operation, ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest, SubjectContext: report.Fingerprint}, &intent)
	req := connect.NewRequest(&repov1.PreparePushRecoveryRequest{RepositoryId: preview.RepositoryID, IntentId: intent.IntentID, Fingerprint: report.Fingerprint, Remote: report.Remote, Branch: report.Branch})
	req.Header().Set("X-Test-Principal", "human")
	prepared, e := client.PreparePushRecovery(ctx, req)
	if e != nil || prepared.Msg.State != "prepared" {
		t.Fatalf("prepare: %v %v", prepared, e)
	}
	if !strings.HasPrefix(prepared.Msg.OriginalBundle, leased+string(filepath.Separator)) {
		t.Fatal("artifact escaped leased root")
	}
	if _, e = client.PreparePushRecovery(ctx, req); e == nil {
		t.Fatal("consumed intent replay accepted")
	}
	get := func(want string) {
		t.Helper()
		resp, e := client.GetPushRecovery(ctx, connect.NewRequest(&repov1.GetPushRecoveryRequest{}))
		if e != nil || resp.Msg.State != want || resp.Msg.Fingerprint != report.Fingerprint {
			t.Fatalf("discovered status want %s: %v %v", want, resp, e)
		}
	}
	get("prepared")
	moved := report
	moved.Fingerprint = strings.Repeat("b", 64)
	runner.setReport(moved)
	get("stale")
	runner.setReport(pushsafety.Report{})
	get("unverified")
	runner.setReport(report)
	// Tamper only with this test's copied artifact. Original source is unchanged.
	if e = os.WriteFile(prepared.Msg.RepairedBundle, []byte("damaged test bundle"), 0o600); e != nil {
		t.Fatal(e)
	}
	get("damaged")
	if _, e = os.Stat(filepath.Join(root, "forbidden-production")); !os.IsNotExist(e) {
		t.Fatal("production root was touched")
	}
}

// Opt-in browser evidence uses only this httptest server and leased artifacts.
// Ordinary unit runs have no browser/service dependency.
func exerciseRecoveryBrowser(t *testing.T, url string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "browser-automation-studio", "capture", "--url", url, "--capture", "screenshot,console-logs,network", "--wait-for", "body[data-recovery-check=passed]", "--label", "gct-isolated-recovery-browser", "--json")
	cmd.WaitDelay = 5 * time.Second
	output, e := cmd.CombinedOutput()
	t.Logf("Browser evidence: %s", output)
	if e != nil {
		t.Fatalf("isolated browser capture: %v", e)
	}
	var result map[string]any
	if e = json.Unmarshal(output, &result); e != nil {
		t.Fatalf("browser result is not JSON: %v", e)
	}
	if len(result) == 0 {
		t.Fatal("browser returned no evidence")
	}
}
