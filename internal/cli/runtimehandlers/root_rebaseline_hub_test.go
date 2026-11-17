package runtimehandlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/rootrebaselinedecision"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/rootdecision"
	"testing"
	"time"
)

func TestHubPublicationCapabilityProbeHasNoEffects(t *testing.T) {
	var out, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", "--capability", "--probe-hub-publication"}, nil, &out, &stderr, nil)
	if code != 0 || out.String() != "hub-source-publication-v1\n" || stderr.Len() != 0 {
		t.Fatalf("probe: code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
	}
}

func TestRootDecisionRequiresExactQualifiedTermsAndVerifiedHuman(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	terms := rootDecisionTerms{RequesterRef: "req", Deadline: now.Add(time.Hour).Format(time.RFC3339)}
	good := rootDecisionLookup{AskID: "ask", RequesterRef: "req", Digest: "digest", State: "answered", Answer: "approve", IdentityKind: "verified_human", IdentitySource: "cloudflare_access", Deadline: terms.Deadline, AnsweredAt: now.Add(-time.Minute).Format(time.RFC3339)}
	if ok, err := verifyRootDecision(good, "ask", terms, "digest", now); !ok || err != nil {
		t.Fatalf("valid approval: %v %v", ok, err)
	}
	password := good
	password.IdentitySource = "scenario_authenticator_step_up"
	if ok, err := verifyRootDecision(password, "ask", terms, "digest", now); !ok || err != nil {
		t.Fatalf("verified password approval: %v %v", ok, err)
	}
	cases := map[string]func(*rootDecisionLookup){"wrong ask": func(a *rootDecisionLookup) { a.AskID = "other" }, "wrong digest": func(a *rootDecisionLookup) { a.Digest = "other" }, "wrong request": func(a *rootDecisionLookup) { a.RequesterRef = "other" }, "wrong deadline": func(a *rootDecisionLookup) { a.Deadline = "other" }, "default": func(a *rootDecisionLookup) { a.State = "defaulted" }, "reject": func(a *rootDecisionLookup) { a.Answer = "reject" }, "unverified": func(a *rootDecisionLookup) { a.IdentityKind = "unknown" }, "local loopback": func(a *rootDecisionLookup) { a.IdentitySource = "personal_local" }, "unknown source": func(a *rootDecisionLookup) { a.IdentitySource = "bearer" }, "future answer": func(a *rootDecisionLookup) { a.AnsweredAt = now.Add(time.Minute).Format(time.RFC3339) }, "missing answer time": func(a *rootDecisionLookup) { a.AnsweredAt = "" }}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			bad := good
			mutate(&bad)
			if ok, err := verifyRootDecision(bad, "ask", terms, "digest", now); ok || err == nil {
				t.Fatalf("accepted: %v %v", ok, err)
			}
		})
	}
	pending := good
	pending.State = "pending"
	if ok, err := verifyRootDecision(pending, "ask", terms, "digest", now); ok || err != nil {
		t.Fatalf("pending: %v %v", ok, err)
	}
	if ok, err := verifyRootDecision(good, "ask", terms, "digest", now.Add(time.Hour)); ok || err == nil {
		t.Fatalf("expired: %v %v", ok, err)
	}
}

func TestHubDecisionPublishesQualifiedTermsAndReadsBackAnswer(t *testing.T) {
	var saved rootDecisionTerms
	var digest string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing service token")
		}
		switch r.URL.Path {
		case "/api/v1/internal/root-rebaseline/publish":
			var raw map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
				t.Error(err)
				return
			}
			_ = json.Unmarshal(raw["canonical_terms_sha256"], &digest)
			delete(raw, "canonical_terms_sha256")
			data, _ := json.Marshal(raw)
			if err := json.Unmarshal(data, &saved); err != nil {
				t.Error(err)
			}
			expected, _ := rootrebaselinedecision.Digest(saved)
			if digest != expected {
				t.Error("digest does not cover published terms")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"ask_id": "ask", "requester_ref": saved.RequesterRef})
		case "/api/v1/internal/root-rebaseline/lookup", "/api/v1/internal/root-rebaseline/consume":
			var lookup map[string]string
			_ = json.NewDecoder(r.Body).Decode(&lookup)
			if lookup["requester_ref"] != saved.RequesterRef || lookup["canonical_terms_sha256"] != digest {
				t.Error("lookup changed identity")
			}
			_ = json.NewEncoder(w).Encode(rootDecisionLookup{AskID: "ask", RequesterRef: saved.RequesterRef, Digest: digest, State: "answered", Answer: "approve", IdentityKind: "verified_human", IdentitySource: "cloudflare_access", Deadline: saved.Deadline, AnsweredAt: time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano), ConsumedAt: time.Now().UTC().Format(time.RFC3339Nano)})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := &rootDecisionClient{base: server.URL, http: server.Client(), token: func(context.Context) (string, error) { return "token", nil }}
	plan := runtimeapp.RebaselinePlan{CandidateSHA: "candidate", ServingSHA: "serving", WatchdogSHA: "watchdog", CandidateContract: "new", ServingContract: "old", Changes: []string{"added state"}}
	ok, err := confirmRootViaHub(context.Background(), client, plan, "bundle", filepath.Join(t.TempDir(), "decision.json"), io.Discard)
	if !ok || err != nil {
		t.Fatalf("decision: %v %v", ok, err)
	}
	if saved.CandidateSHA != plan.CandidateSHA || saved.ServingSHA != plan.ServingSHA || saved.WatchdogSHA != plan.WatchdogSHA || saved.CandidateContract != plan.CandidateContract || saved.ServingContract != plan.ServingContract || len(saved.Changes) != 1 || saved.Changes[0] != plan.Changes[0] {
		t.Fatalf("qualified terms not bound: %+v", saved)
	}
	if saved.Version != 2 || saved.Operation != "contract_change" {
		t.Fatalf("wrong decision type: %+v", saved)
	}
	plan.Continuation = true
	ok, err = confirmRootViaHub(context.Background(), client, plan, "bundle", filepath.Join(t.TempDir(), "decision.json"), io.Discard)
	if !ok || err != nil || saved.Version != 2 || saved.Operation != "continuation" {
		t.Fatalf("continuation decision: %v %v %+v", ok, err, saved)
	}
}

func TestHubDecisionResumesAndSpentApprovalCannotReplay(t *testing.T) {
	var saved rootDecisionTerms
	var digest string
	approved, consumed := false, false
	publishes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/internal/root-rebaseline/publish":
			var raw map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&raw)
			var incoming string
			_ = json.Unmarshal(raw["canonical_terms_sha256"], &incoming)
			delete(raw, "canonical_terms_sha256")
			data, _ := json.Marshal(raw)
			var terms rootDecisionTerms
			_ = json.Unmarshal(data, &terms)
			if publishes > 0 && terms.RequesterRef != saved.RequesterRef {
				approved, consumed = false, false
			}
			saved = terms
			digest = incoming
			publishes++
			_ = json.NewEncoder(w).Encode(map[string]string{"ask_id": "ask", "requester_ref": saved.RequesterRef})
		case "/api/v1/internal/root-rebaseline/lookup", "/api/v1/internal/root-rebaseline/consume":
			state, answer, kind, source, at := "pending", "", "", "", ""
			if approved {
				state, answer, kind, source, at = "answered", "approve", "verified_human", "cloudflare_access", time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)
			}
			if r.URL.Path == "/api/v1/internal/root-rebaseline/consume" {
				if consumed {
					http.Error(w, "spent", http.StatusConflict)
					return
				}
				consumed = true
			}
			consumedAt := ""
			if consumed {
				consumedAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
			_ = json.NewEncoder(w).Encode(rootDecisionLookup{AskID: "ask", RequesterRef: saved.RequesterRef, Digest: digest, State: state, Answer: answer, IdentityKind: kind, IdentitySource: source, Deadline: saved.Deadline, AnsweredAt: at, ConsumedAt: consumedAt})
		}
	}))
	defer srv.Close()
	c := &rootDecisionClient{base: srv.URL, http: srv.Client(), token: func(context.Context) (string, error) { return "token", nil }}
	plan := runtimeapp.RebaselinePlan{CandidateSHA: "candidate", ServingSHA: "serving", WatchdogSHA: "watchdog", CandidateContract: "new", ServingContract: "old", Changes: []string{"state"}}
	path := filepath.Join(t.TempDir(), "pending.json")
	if ok, err := confirmRootViaHub(context.Background(), c, plan, "bundle", path, io.Discard); ok || err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("pending: %v %v", ok, err)
	}
	approved = true
	if ok, err := confirmRootViaHub(context.Background(), c, plan, "bundle", path, io.Discard); !ok || err != nil {
		t.Fatalf("resumed: %v %v", ok, err)
	}
	if ok, err := confirmRootViaHub(context.Background(), c, plan, "bundle", path, io.Discard); ok || err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("spent decision did not require fresh approval: %v %v", ok, err)
	}
	approved = true
	if ok, err := confirmRootViaHub(context.Background(), c, plan, "bundle", path, io.Discard); !ok || err != nil {
		t.Fatalf("fresh approval: %v %v", ok, err)
	}
}

func TestConsumedOrExpiredHubDecisionRequiresFreshAnswer(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	terms := rootDecisionTerms{Version: 2, Operation: "contract_change", RequesterRef: "root-rebaseline-fixture", Deadline: now.Add(-time.Minute).Format(time.RFC3339), CandidateSHA: strings.Repeat("a", 64), ServingSHA: strings.Repeat("b", 64), WatchdogSHA: strings.Repeat("c", 64), BundleSHA: strings.Repeat("d", 64), CandidateContract: strings.Repeat("e", 64), ServingContract: strings.Repeat("f", 64), Changes: []string{"state"}}
	digest, err := rootrebaselinedecision.Digest(terms)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pending.json")
	// A restored or forged local progress file is not authority to reuse an
	// already-consumed answer, even when it names the exact same images.
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := rebaselineprogress.Write(filepath.Join(home, ".vrooli", "state", "root-rebaseline", "progress.json"), rebaselineprogress.Progress{Version: 1, Candidate: terms.CandidateSHA, Watchdog: terms.WatchdogSHA, Bundle: terms.BundleSHA}); err != nil {
		t.Fatal(err)
	}
	got := rootDecisionLookup{AskID: "ask", RequesterRef: terms.RequesterRef, Digest: digest, State: "answered", Answer: "approve", IdentityKind: "verified_human", IdentitySource: "cloudflare_access", Deadline: terms.Deadline, AnsweredAt: now.Add(-3 * time.Minute).Format(time.RFC3339Nano), ConsumedAt: now.Add(-2 * time.Minute).Format(time.RFC3339Nano)}
	var fresh rootDecisionTerms
	var freshDigest string
	consumes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/internal/root-rebaseline/publish":
			var raw map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&raw)
			_ = json.Unmarshal(raw["canonical_terms_sha256"], &freshDigest)
			delete(raw, "canonical_terms_sha256")
			data, _ := json.Marshal(raw)
			_ = json.Unmarshal(data, &fresh)
			_ = json.NewEncoder(w).Encode(map[string]string{"ask_id": "new-ask", "requester_ref": fresh.RequesterRef})
		case "/api/v1/internal/root-rebaseline/lookup":
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in["requester_ref"] == terms.RequesterRef {
				_ = json.NewEncoder(w).Encode(got)
			} else {
				_ = json.NewEncoder(w).Encode(rootDecisionLookup{AskID: "new-ask", RequesterRef: fresh.RequesterRef, Digest: freshDigest, State: "pending", Deadline: fresh.Deadline})
			}
		case "/api/v1/internal/root-rebaseline/consume":
			consumes++
			http.Error(w, "unexpected", 500)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := &rootDecisionClient{base: srv.URL, http: srv.Client(), token: func(context.Context) (string, error) { return "token", nil }}
	plan := runtimeapp.RebaselinePlan{CandidateSHA: terms.CandidateSHA, ServingSHA: terms.ServingSHA, WatchdogSHA: terms.WatchdogSHA, CandidateContract: terms.CandidateContract, ServingContract: terms.ServingContract, Changes: terms.Changes}
	if err := rootdecision.Save(path, terms); err != nil {
		t.Fatal(err)
	}
	if ok, err := confirmRootViaHub(context.Background(), c, plan, terms.BundleSHA, path, io.Discard); ok || err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("fresh decision was not required: %v %v", ok, err)
	}
	if fresh.RequesterRef == "" || fresh.RequesterRef == terms.RequesterRef || fresh.BundleSHA != terms.BundleSHA || consumes != 0 {
		t.Fatalf("invalid fresh decision: %+v consumes=%d", fresh, consumes)
	}
	if err := rootdecision.Save(path, terms); err != nil {
		t.Fatal(err)
	}
	got.ConsumedAt = ""
	if ok, err := confirmRootViaHub(context.Background(), c, plan, terms.BundleSHA, path, io.Discard); ok || err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("expired unconsumed decision did not request fresh approval: %v %v", ok, err)
	}
	got.ConsumedAt = now.Add(-2 * time.Minute).Format(time.RFC3339Nano)
	for _, mutate := range []func(*rootDecisionLookup){func(x *rootDecisionLookup) { x.Digest = "other" }, func(x *rootDecisionLookup) { x.IdentitySource = "unverified" }, func(x *rootDecisionLookup) { x.ConsumedAt = now.Format(time.RFC3339Nano) }} {
		if err := rootdecision.Save(path, terms); err != nil {
			t.Fatal(err)
		}
		saved := got
		mutate(&got)
		if ok, err := confirmRootViaHub(context.Background(), c, plan, terms.BundleSHA, path, io.Discard); ok || err == nil {
			t.Fatal("tampered consumed decision accepted")
		}
		got = saved
	}
	if err := rootdecision.Save(path, terms); err != nil {
		t.Fatal(err)
	}
	if ok, err := confirmRootViaHub(context.Background(), c, plan, "other bundle", path, io.Discard); ok || err == nil {
		t.Fatal("other bundle accepted")
	}
}

func TestLegacyFinishingDecisionSurvivesPendingThenApproval(t *testing.T) {
	p := &rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64), Bundle: strings.Repeat("c", 64)}
	plan := runtimeapp.RebaselinePlan{Continuation: true, CandidateSHA: p.Candidate, ServingSHA: p.Candidate, WatchdogSHA: p.Watchdog, CandidateContract: strings.Repeat("d", 64), ServingContract: strings.Repeat("d", 64), Changes: []string{"Complete interrupted transition."}}
	var terms rootDecisionTerms
	var digest string
	approved, consumed := false, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/internal/root-rebaseline/publish":
			var raw map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&raw)
			_ = json.Unmarshal(raw["canonical_terms_sha256"], &digest)
			delete(raw, "canonical_terms_sha256")
			data, _ := json.Marshal(raw)
			_ = json.Unmarshal(data, &terms)
			_ = json.NewEncoder(w).Encode(map[string]string{"ask_id": "ask", "requester_ref": terms.RequesterRef})
		case "/api/v1/internal/root-rebaseline/lookup", "/api/v1/internal/root-rebaseline/consume":
			if r.URL.Path == "/api/v1/internal/root-rebaseline/consume" {
				consumed = true
			}
			got := rootDecisionLookup{AskID: "ask", RequesterRef: terms.RequesterRef, Digest: digest, State: "pending", Deadline: terms.Deadline}
			if approved {
				got.State, got.Answer, got.IdentityKind, got.IdentitySource, got.AnsweredAt = "answered", "approve", "verified_human", "cloudflare_access", time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)
			}
			if consumed {
				got.ConsumedAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
			_ = json.NewEncoder(w).Encode(got)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := &rootDecisionClient{base: srv.URL, http: srv.Client(), token: func(context.Context) (string, error) { return "token", nil }}
	path := filepath.Join(t.TempDir(), "pending.json")
	if !legacyFinishingNeedsDecision(p, p.Bundle, rootDecisionTerms{}) {
		t.Fatal("missing initial legacy decision")
	}
	if ok, err := confirmRootViaHub(context.Background(), c, plan, p.Bundle, path, io.Discard); ok || err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("first invocation: %v %v", ok, err)
	}
	loaded, err := rootdecision.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !legacyFinishingNeedsDecision(p, p.Bundle, loaded) {
		t.Fatal("pending locator suppressed next confirmation")
	}
	approved = true
	if ok, err := confirmRootViaHub(context.Background(), c, plan, p.Bundle, path, io.Discard); !ok || err != nil || !consumed {
		t.Fatalf("approved retry did not consume: %v %v consumed=%v", ok, err, consumed)
	}
	if err := verifyConsumedHubDecision(context.Background(), c, path, p.Bundle, p.Candidate, p.Watchdog); err != nil {
		t.Fatalf("consumed authority did not permit effects: %v", err)
	}
}

func TestRejectedDecisionKeepsItsIdentityUntilRetired(t *testing.T) {
	refs := []string{}
	var terms rootDecisionTerms
	var digest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/internal/root-rebaseline/publish":
			var raw map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&raw)
			_ = json.Unmarshal(raw["canonical_terms_sha256"], &digest)
			delete(raw, "canonical_terms_sha256")
			data, _ := json.Marshal(raw)
			_ = json.Unmarshal(data, &terms)
			refs = append(refs, terms.RequesterRef)
			_ = json.NewEncoder(w).Encode(map[string]string{"ask_id": "ask", "requester_ref": terms.RequesterRef})
		case "/api/v1/internal/root-rebaseline/lookup":
			_ = json.NewEncoder(w).Encode(rootDecisionLookup{AskID: "ask", RequesterRef: terms.RequesterRef, Digest: digest, State: "answered", Answer: "reject", IdentityKind: "verified_human", IdentitySource: "cloudflare_access", Deadline: terms.Deadline, AnsweredAt: time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := &rootDecisionClient{base: srv.URL, http: srv.Client(), token: func(context.Context) (string, error) { return "token", nil }}
	plan := runtimeapp.RebaselinePlan{CandidateSHA: "candidate", ServingSHA: "serving", WatchdogSHA: "watchdog", CandidateContract: "new", ServingContract: "old", Changes: []string{"state"}}
	path := filepath.Join(t.TempDir(), "pending.json")
	for i := 0; i < 2; i++ {
		if ok, err := confirmRootViaHub(context.Background(), c, plan, "bundle", path, io.Discard); ok || err == nil {
			t.Fatalf("rejection authorized: %v %v", ok, err)
		}
	}
	if len(refs) != 2 || refs[0] != refs[1] {
		t.Fatalf("rejected snapshot identity was replaced: %v", refs)
	}
}

func TestMakeHubResumeUsesPreparedImagesWithoutBuild(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"mk", "tools"} {
		if err := os.Symlink(filepath.Join(root, name), filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	build := filepath.Join(dir, "build")
	snapshot := filepath.Join(build, "rebaseline-hub.fixture")
	if err := os.MkdirAll(snapshot, 0700); err != nil {
		t.Fatal(err)
	}
	names := []string{"vrooli", "vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog", "vrooli-atomic-install"}
	var manifest strings.Builder
	for _, name := range names {
		data := []byte("fixture " + name)
		if name == "vrooli" {
			data = []byte("#!/bin/sh\nif [ \"$4\" = --capability ]; then [ -z \"${PUBLISHED:-}\" ] || echo hub-source-publication-v1; exit 0; fi\nprintf '%s\\n' \"$*\" >> \"$TRACE\"\nif [ -n \"${MUTATE:-}\" ]; then printf replaced > \"$MUTATE\"; fi\n")
		}
		if name == "vrooli-atomic-install" {
			data = []byte("#!/bin/sh\nprintf 'legacy installer must not run\\n' >&2\nexit 2\n")
		}
		if err := os.WriteFile(filepath.Join(snapshot, name), data, 0700); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(&manifest, "%x  %s\n", sum, name)
	}
	if err := os.WriteFile(filepath.Join(snapshot, "SHA256SUMS"), []byte(manifest.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapshot, ".ready"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(build, "rebaseline-hub-current"), []byte(snapshot+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(build, "current"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(dir, "trace")
	publisher := filepath.Join(dir, "current-publisher")
	if err := os.WriteFile(publisher, []byte("#!/bin/sh\nif [ \"$1\" = --snapshot-root ]; then printf '%s\\n' \"$SNAPSHOT_ROOT\"; exit 0; fi\nif [ \"$1\" = --snapshot-lease-exec ]; then [ \"$2\" = \"$SNAPSHOT_ROOT\" ] && [ \"$3\" = \"$SNAPSHOT_ROOT/rebaseline-hub.fixture\" ] && [ \"$4\" = -- ] || exit 2; shift 4; exec \"$@\"; fi\n[ \"$1\" != --verify-runtime-paths ] || exit 0\n[ \"$1\" = --publish-source ] || exit 2\nprintf 'publish-source %s %s %s %s %s\\n' \"$2\" \"$3\" \"$4\" \"$5\" \"$6\" >> \"$TRACE\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	run := func() ([]byte, error) {
		makefile := filepath.Join(root, "Makefile")
		cmd := exec.Command("make", "-f", makefile, "rebaseline-hub-resume", "BUILD_DIR=build", "INSTALL_DIR="+filepath.Join(dir, "install"), "SOURCE_ROOT_POINTER="+filepath.Join(dir, "source"), "BUNDLE_PUBLISHER="+publisher, "MAKE=make -f "+makefile)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "SNAPSHOT_ROOT="+build, "TRACE="+trace)
		return cmd.CombinedOutput()
	}
	if out, err := run(); err != nil {
		t.Fatalf("resume failed: %v %s", err, out)
	}
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	manifestSHA := fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.String())))
	expectedPublication := "publish-source " + snapshot + " " + filepath.Join(dir, "install") + " " + manifestSHA + " " + dir + " " + filepath.Join(dir, "source")
	if !strings.Contains(text, "runtime update rebaseline "+filepath.Join(snapshot, "vrooli")+" --hub") || !strings.Contains(text, expectedPublication) || len(strings.Split(strings.TrimSpace(text), "\n")) != 2 {
		t.Fatalf("resume invoked wrong commands: %s", text)
	}
	if err := os.Remove(trace); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PUBLISHED", "1")
	if out, err := run(); err != nil {
		t.Fatalf("completed transaction failed: %v %s", err, out)
	}
	data, err = os.ReadFile(trace)
	if err != nil || strings.Contains(string(data), "publish-source") {
		t.Fatalf("completed transaction repeated publication: %s, %v", data, err)
	}
	if err := os.Remove(trace); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PUBLISHED", "")
	t.Setenv("MUTATE", filepath.Join(snapshot, "vrooli-api"))
	if out, err := run(); err == nil || !strings.Contains(string(out), "source publication could not be confirmed") {
		t.Fatalf("replaced companion did not report unconfirmed legacy publication: %v %s", err, out)
	}
	data, err = os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "install ") {
		t.Fatalf("companions installed after replacement: %s", data)
	}
}

func TestFinishingRequiresHubConsumedApprovalNotLocalLocator(t *testing.T) {
	now := time.Now().UTC()
	terms := rootDecisionTerms{RequesterRef: "req", Deadline: now.Add(time.Hour).Format(time.RFC3339), CandidateSHA: "candidate", WatchdogSHA: "watchdog", BundleSHA: "bundle", Changes: []string{"state"}}
	path := filepath.Join(t.TempDir(), "pending.json")
	if err := rootdecision.Save(path, terms); err != nil {
		t.Fatal(err)
	}
	digest, _ := rootrebaselinedecision.Digest(terms)
	consumed := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rootDecisionLookup{AskID: "ask", RequesterRef: terms.RequesterRef, Digest: digest, State: "answered", Answer: "approve", IdentityKind: "verified_human", IdentitySource: "cloudflare_access", Deadline: terms.Deadline, AnsweredAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ConsumedAt: consumed})
	}))
	defer srv.Close()
	c := &rootDecisionClient{base: srv.URL, http: srv.Client(), token: func(context.Context) (string, error) { return "token", nil }}
	if err := verifyConsumedHubDecision(context.Background(), c, path, "bundle", "candidate", "watchdog"); err == nil {
		t.Fatal("unconsumed approval authorized recovery")
	}
	consumed = now.Add(-time.Second).Format(time.RFC3339Nano)
	if err := verifyConsumedHubDecision(context.Background(), c, path, "bundle", "candidate", "watchdog"); err != nil {
		t.Fatal(err)
	}
	if err := verifyConsumedHubDecision(context.Background(), c, path, "changed", "candidate", "watchdog"); err == nil {
		t.Fatal("changed bundle authorized")
	}
	if err := verifyConsumedHubDecision(context.Background(), c, path, "bundle", "candidate", "rebuilt"); err == nil {
		t.Fatal("changed watchdog authorized")
	}
}
