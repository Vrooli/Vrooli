package teams_test

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	teamhandlers "prompt-manager/handlers/teams"
	"prompt-manager/internal/heartbeat"
	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
)

func TestFrictionCaptureIsolatedMeaningDiscoveryRepeatAndFailures(t *testing.T) {
	roots := paths.RootsForTest(t)
	relation := store.NewFileRelationStore(roots.Config)
	corpus := store.NewFileTeamStore(roots.Config, roots.RuntimeData, relation)
	handlers := heartbeat.NewHandlers(heartbeat.HandlersDeps{TeamStore: corpus})
	mount, handler := teamhandlers.NewConnectMount(nil, handlers)
	denied, unavailable, lost := false, false, false
	readUnavailable := false
	writes := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","readiness":true}`))
	})
	mux.Handle(mount, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if readUnavailable && strings.HasSuffix(r.URL.Path, "/ListKnowledge") {
			http.Error(w, "fixture read unavailable", http.StatusServiceUnavailable)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/AddKnowledge") {
			if denied {
				http.Error(w, "denied fixture write", http.StatusForbidden)
				return
			}
			if unavailable {
				http.Error(w, "unavailable fixture storage", http.StatusServiceUnavailable)
				return
			}
			writes++
			if lost {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, r)
				if rec.Code != 200 {
					t.Fatalf("lost-response committed handler status %d: %s", rec.Code, rec.Body.String())
				}
				http.Error(w, "fixture response lost after commit", http.StatusGatewayTimeout)
				return
			}
		}
		handler.ServeHTTP(w, r)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	cliDir, err := filepath.Abs("../../../cli")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "prompt-manager-i10")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = cliDir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("isolated CLI build: %v: %s", err, output)
	}
	binaryBytes, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Qualified isolated CLI build sha256=%x", sha256.Sum256(binaryBytes))
	run := func(args ...string) (string, error) {
		t.Helper()
		command := exec.Command(binary, append([]string{"--api-base", server.URL}, args...)...)
		returnBytes, err := command.CombinedOutput()
		return string(returnBytes), err
	}

	report := map[string]any{
		"scope": "toolchain", "severity": "recurring", "slug": "synthetic-contract-test-no-action",
		"reporter": "synthetic-agent", "reporter_team": "synthetic-team", "observed_at": "2026-10-02",
		"context":  map[string]any{"scenario": "prompt-manager", "skill": "report-friction", "member": "synthetic-member", "command": "printf 'a: # [b]'\nnext command", "doc": "synthetic contract (no implementation authority)", "task": "I10 fixture"},
		"expected": "SYNTHETIC CONTRACT TEST: no real incident or remedy authorized.", "actual": "Fictional fixture observation only.", "description": "Run the fixture.\nRead exact output: a: # [b] — café.", "honesty_flags": []string{"auto-generated", "speculative-cause"}, "recurrence_count": 2,
		"attempt":     "SYNTHETIC CONTRACT TEST. Attempt to transfer a fictional observation; do not implement it.",
		"observation": "Observe fictional a: # [b] output twice.\nUncertainty: this proves fixture behavior only.",
		"explanation": "This is fictional friction for contract testing, not a real bug or immediate fix. Expected contract: I10 A meaning preservation.",
	}
	path := filepath.Join(t.TempDir(), "report.json")
	write := func(v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	capture := func() (string, error) {
		return run("team", "friction-capture", "meta-optimization", "--report-file="+path, "--json")
	}
	write(report)
	out, err := capture()
	if err != nil {
		t.Fatal(err)
	}
	var receipt store.KnowledgeEntry
	if err := json.Unmarshal([]byte(out), &receipt); err != nil {
		t.Fatal(err)
	}
	// A receiver discovers by its documented prefix, without a record-ID lookup.
	receiver, err := run("team", "knowledge-list", "meta-optimization", "--topic-prefix=friction-inbox/", "--last=100", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var listed struct {
		Entries []store.KnowledgeEntry `json:"entries"`
	}
	if err := json.Unmarshal([]byte(receiver), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Entries) != 1 {
		t.Fatalf("receiver found %d cases", len(listed.Entries))
	}
	entry := listed.Entries[0]
	if entry.ID != receipt.ID || entry.Content != receipt.Content {
		t.Fatal("receiver disagrees with receipt")
	}
	pieces := strings.SplitN(entry.Content, "\n---\n", 2)
	if len(pieces) != 2 {
		t.Fatal("missing front matter boundary")
	}
	// JSON-quoted YAML scalar values round-trip exactly through JSON decoding.
	for _, key := range []string{"reporter", "reporter_team", "observed_at", "expected", "actual", "description"} {
		marker := key + ": "
		start := strings.Index(pieces[0], "\n"+marker)
		if start < 0 {
			t.Fatalf("missing %s", key)
		}
		value := strings.SplitN(pieces[0][start+1+len(marker):], "\n", 2)[0]
		var got any
		if err := json.Unmarshal([]byte(value), &got); err != nil {
			t.Fatal(err)
		}
		if got != report[key] {
			t.Fatalf("%s changed: %v", key, got)
		}
	}
	for key, value := range report["context"].(map[string]any) {
		want, _ := json.Marshal(value)
		if !strings.Contains(pieces[0], "  "+key+": "+string(want)+"\n") {
			t.Fatalf("anchor %s changed", key)
		}
	}
	if !strings.Contains(pieces[0], `honesty_flags: ["auto-generated","speculative-cause"]`) {
		t.Fatal("flags changed")
	}
	if pieces[1] != "\n"+report["attempt"].(string)+"\n\n"+report["observation"].(string)+"\n\n"+report["explanation"].(string)+"\n" {
		t.Fatal("explanatory body changed")
	}
	if entry.Attribution.Kind != store.KnowledgeKindWriterSkill || entry.Caller != "skill:report-friction" || entry.Attribution.TeamID == nil || *entry.Attribution.TeamID != "meta-optimization" {
		t.Fatalf("trusted writer attribution changed: %+v", entry.Attribution)
	}
	repeat, err := capture()
	if err != nil {
		t.Fatal(err)
	}
	if repeat != out || writes != 1 {
		t.Fatalf("repeat creates another case writes=%d", writes)
	}
	// Same topic plus changed meaning must fail without another write.
	original := report["description"]
	report["description"] = "Changed fictional meaning"
	write(report)
	if _, err := capture(); err == nil {
		t.Fatal("stable topic conflict accepted")
	}
	if writes != 1 {
		t.Fatal("conflict created another case")
	}
	report["description"] = original
	write(report)
	readUnavailable = true
	if _, err := capture(); err == nil {
		t.Fatal("unavailable reconciliation read accepted")
	}
	if writes != 1 {
		t.Fatal("unavailable reconciliation read caused write")
	}
	readUnavailable = false
	t.Logf("Receiver prefix found synthetic topic %s as %s; exact content matches; repeated receipt identical; one case", entry.Topic, entry.ID)
	for _, key := range []string{"reporter", "reporter_team", "observed_at", "expected", "actual", "description", "attempt", "observation", "explanation", "scope", "severity", "slug"} {
		t.Run("missing-"+key, func(t *testing.T) {
			copy := map[string]any{}
			for k, v := range report {
				if k != key {
					copy[k] = v
				}
			}
			write(copy)
			before := writes
			if _, err := capture(); err == nil {
				t.Fatalf("missing %s accepted", key)
			}
			if writes != before {
				t.Fatal("missing field caused write")
			}
		})
	}
	for _, tc := range []struct {
		name, key string
		value     any
	}{{"required", "expected", ""}, {"scope", "scope", "bad"}, {"severity", "severity", "bad"}, {"slug", "slug", "../escape"}, {"route", "slug", "a/b"}, {"recurrence", "recurrence_count", 1}, {"unknown-field", "security_role", "admin"}, {"future-date", "observed_at", "2999-01-01"}, {"flags", "honesty_flags", []string{"bad"}}, {"blocking", "severity", "blocking"}} {
		t.Run(tc.name, func(t *testing.T) {
			copy := map[string]any{}
			for k, v := range report {
				copy[k] = v
			}
			copy[tc.key] = tc.value
			write(copy)
			before := writes
			if _, err := capture(); err == nil {
				t.Fatal("invalid report accepted")
			}
			if writes != before {
				t.Fatal("invalid report wrote storage")
			}
		})
	}
	for _, raw := range []string{"{broken", `{} {}`, `null`} {
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := capture(); err == nil {
			t.Fatalf("malformed content accepted %q", raw)
		}
	}
	write(report)
	if _, err := run("team", "friction-capture", "meta-optimization", "--topic=unsupported"); err == nil {
		t.Fatal("unsupported flag accepted")
	}
	if _, err := run("team", "friction-capture", "meta-optimization", "--report-file="+path, "--scope=toolchain"); err == nil {
		t.Fatal("mixed input accepted")
	}
	report["slug"] = "synthetic-unavailable-initial-read"
	write(report)
	readUnavailable = true
	beforeRead := writes
	if _, err := capture(); err == nil {
		t.Fatal("unavailable initial read accepted")
	}
	if writes != beforeRead {
		t.Fatal("unavailable initial read caused write")
	}
	readUnavailable = false
	for _, kind := range []string{"denied", "unavailable", "lost"} {
		report["slug"] = "synthetic-contract-test-" + kind
		write(report)
		denied = kind == "denied"
		unavailable = kind == "unavailable"
		lost = kind == "lost"
		before := writes
		if _, err := capture(); err == nil {
			t.Fatalf("%s accepted as known success", kind)
		}
		denied = false
		unavailable = false
		lost = false
		if kind == "lost" {
			if _, err := capture(); err != nil {
				t.Fatal(err)
			}
			if writes != before+1 {
				t.Fatal("uncertain write duplicated")
			}
		} else if writes != before {
			t.Fatal("denied/unavailable mutated fixture")
		}
	}
	beforeLegacy := writes
	if _, err := run("team", "friction-capture", "meta-optimization", "--scope=toolchain", "--severity=one-off", "--expected=e", "--actual=a", "--description=d", "--slug=../invalid"); err == nil {
		t.Fatal("unsafe typed slug accepted")
	}
	if writes != beforeLegacy {
		t.Fatal("unsafe typed slug caused write")
	}
	legacy, err := run("team", "friction-capture", "meta-optimization", "--scope=unknown", "--severity=one-off", "--expected=legacy expected", "--actual=legacy actual", "--description=legacy detail", "--slug=synthetic-legacy-test", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(legacy, "reporter: operator") {
		t.Fatal("legacy operator contract changed")
	}
	t.Log("Malformed/missing/taxonomy/routing/unsupported input produces no records; denied/unavailable writes fail; lost response reconciles one committed case; legacy typed capture works")
}
