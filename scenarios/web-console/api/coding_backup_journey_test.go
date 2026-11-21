//go:build journey

package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
	"web-console/session"
)

type backupJourney struct {
	server            *Server
	api               *httptest.Server
	owner             context.Context
	home, door, token string
	calls             atomic.Int64
	paid              chan struct{}
}

func newBackupJourney(t *testing.T) *backupJourney {
	t.Helper()
	j := &backupJourney{paid: make(chan struct{}, 32)}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if len(data) == 0 {
			t.Error("native harness sent no request")
		}
		j.calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"backup-fixture\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Backup journey completed.\"}}]}\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"total_tokens\":7,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
		select {
		case j.paid <- struct{}{}:
		default:
		}
	}))
	t.Cleanup(upstream.Close)
	caches, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	cache := strings.Fields(string(caches))
	cmd := exec.Command("go", "test", "./journeytest", "-run", "^TestJourneyDoorServer$", "-count=1", "-v", "-timeout", "12m")
	cmd.Dir = "../../landing-page-business-suite/api"
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_DOOR_SERVER=1", "VROOLI_JOURNEY_CONSUMER=1", "VROOLI_JOURNEY_ROUTER_URL="+upstream.URL)
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if j.door != "" {
			resp, _ := http.Post(j.door+"/__journey/stop", "text/plain", nil)
			if resp != nil {
				resp.Body.Close()
			}
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("real door process: %v: %s", err, logs.String())
		}
	})
	scanner := bufio.NewScanner(output)
	var refresh string
	var startup strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DOOR_URL=") {
			j.door = strings.TrimPrefix(line, "DOOR_URL=")
		}
		if strings.HasPrefix(line, "CONSUMER_REFRESH=") {
			refresh = strings.TrimPrefix(line, "CONSUMER_REFRESH=")
		} else if !strings.HasPrefix(line, "DEVICE_KEY=") {
			fmt.Fprintln(&startup, line)
		}
		if j.door != "" && refresh != "" {
			go io.Copy(io.Discard, output)
			break
		}
	}
	if j.door == "" || refresh == "" {
		t.Fatalf("consumer door not ready: %s\n%s\nscanner: %v", startup.String(), logs.String(), scanner.Err())
	}
	j.home = t.TempDir()
	for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "OPENROUTER_API_KEY", "VROOLI_CODING_SESSION_KEY", "VROOLI_MODEL_SOURCE"} {
		t.Setenv(key, "")
	}
	t.Setenv("HOME", j.home)
	t.Setenv("VROOLI_HOME", filepath.Join(j.home, ".vrooli"))
	t.Setenv("WC_SESSION_STATE_ROOT", filepath.Join(j.home, "sessions"))
	t.Setenv("WC_TMUX_SOCKET", filepath.Join(j.home, "tmux.sock"))
	t.Setenv("CODEX_HOME", filepath.Join(j.home, ".codex"))
	t.Setenv("LPBS_URL", j.door)
	t.Setenv("GOCACHE", cache[0])
	t.Setenv("GOMODCACHE", cache[1])
	for _, class := range []string{"CONFIG", "DATA", "CACHE", "LOGS", "STATE"} {
		t.Setenv("VROOLI_"+class+"_ROOT", filepath.Join(j.home, strings.ToLower(class)))
		if class != "LOGS" {
			t.Setenv("XDG_"+class+"_HOME", filepath.Join(j.home, "xdg-"+strings.ToLower(class)))
		}
	}
	os.MkdirAll(filepath.Join(j.home, "bin"), 0700)
	launcher := filepath.Join(j.home, "bin", "vrooli-agent-launcher")
	build := exec.Command("go", "build", "-o", launcher, "./cmd/vrooli-agent-launcher")
	build.Dir = "../../.."
	if data, err := build.CombinedOutput(); err != nil {
		t.Fatalf("source launcher build: %v: %s", err, data)
	}
	t.Setenv("PATH", filepath.Dir(launcher)+":"+os.Getenv("PATH"))
	os.MkdirAll(os.Getenv("CODEX_HOME"), 0700)
	os.WriteFile(filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), []byte(fmt.Sprintf("[projects.%q]\ntrust_level = \"trusted\"\n", j.home)), 0600)
	db, err := sql.Open("sqlite", resolveSQLiteDSN())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	j.server = NewServer(database.NewFromPrimary(db))
	j.server.agentUsageURL = "http://127.0.0.1:1"
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(j.home, "fixture-credentials.enc"), "synthetic-fixture-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	j.server.credentialClient = client
	j.server.subscriptionResolver = &credentialclient.ConsumerSessionResolver{Credentials: client, LPBSBaseURL: j.door}
	_, err = j.server.credentialClient.Provision(t.Context(), credentialclient.ProvisionRequest{Identity: credentialclient.SubscriptionIdentity, Field: credentialclient.SubscriptionField, Value: refresh})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		j.server.clearCodingBackups()
		j.server.sweeper.Stop()
		j.server.sessions.StopReattachWatchdog()
		j.server.sessions.Shutdown()
		if j.server.codexTailer != nil {
			j.server.codexTailer.Stop()
		}
		if j.server.claudeTailer != nil {
			j.server.claudeTailer.Stop()
		}
		if j.server.grokTailer != nil {
			j.server.grokTailer.Stop()
		}
		if j.server.opencodeWatcher != nil {
			j.server.opencodeWatcher.Stop()
		}
		db.Close()
	})
	if _, err := j.server.subscriptionResolver.Resolve(t.Context()); err != nil {
		t.Fatalf("real consumer refresh: %v", err)
	}
	j.token = "synthetic-local-session-token-for-backup-journey"
	tokenFile := filepath.Join(j.home, "local-token")
	os.WriteFile(tokenFile, []byte(j.token), 0600)
	provider := authn.NewPersonalLocalProviderWithTokenFile(tokenFile)
	proof := httptest.NewRequest("GET", "http://localhost/", nil)
	proof.RemoteAddr = "127.0.0.1:1"
	proof.Header.Set("Authorization", "Bearer "+j.token)
	principal, err := provider.VerifyRequest(t.Context(), proof)
	if err != nil {
		t.Fatal(err)
	}
	j.owner = identity.WithPrincipal(t.Context(), principal)
	j.api = httptest.NewServer(authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(j.server.Handler()))
	t.Cleanup(j.api.Close)
	return j
}
func (j *backupJourney) request(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()
	var input io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		input = bytes.NewReader(data)
	}
	req, _ := http.NewRequest(method, j.api.URL+path, input)
	req.Header.Set("Authorization", "Bearer "+j.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}
func (j *backupJourney) consent(t *testing.T, cap int64, automatic bool) {
	t.Helper()
	status, data := j.request(t, "POST", "/api/v1/coding-backup", cliutil.CodingBackupConsent{Harness: "codex", Account: "account", Model: "fixture-model", MaxCredits: cap, ExpiresAt: time.Now().Add(time.Hour), Automatic: automatic})
	if status != 200 {
		t.Fatalf("real consent: %d %s", status, data)
	}
}
func (j *backupJourney) seedLimit(t *testing.T) string {
	t.Helper()
	created, err := j.server.desktopCreateSession(j.owner, sessionsH.CreateInput{Shell: "/bin/bash", Cols: 100, Rows: 30, Backend: "standard", LaunchCommand: "codex --version", ExecuteLaunchCommand: true, AgentType: "codex", Origin: "ui", Owner: codingBackupOwner(j.owner), WorkingDir: j.home})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(j.home, "synthetic-limit.jsonl")
	data := fmt.Sprintf("{\"timestamp\":%q,\"type\":\"event_msg\",\"payload\":{\"type\":\"error\",\"code\":\"usage_limit_reached\"}}\n", time.Now().UTC().Format(time.RFC3339Nano))
	os.WriteFile(path, []byte(data), 0600)
	row, err := j.server.sessionStore.Get(j.owner, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	row.LastRolloutPath = path
	row.AgentType = sessionstore.AgentCodex
	row.Owner = codingBackupOwner(j.owner)
	if err = j.server.sessionStore.Save(j.owner, row); err != nil {
		t.Fatal(err)
	}
	j.server.conversations.AppendUserEvent(j.owner, created.ID, "journey", "Finish the backup journey. Do not use tools.")
	t.Log("SYNTHETIC limit event in real native transcript storage")
	return created.ID
}
func (j *backupJourney) waitPaid(t *testing.T, id string) {
	t.Helper()
	// Native project trust prompts remain native. The fixture marks only its
	// empty directory trusted above; input here acknowledges a possible TUI
	// update announcement, never an API/permission approval.
	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()
	select {
	case <-j.paid:
		return
	case <-timer.C:
		if sess, ok := j.server.sessions.Get(id); ok {
			snap := sess.Subscribe()
			defer sess.Unsubscribe(snap.OutputCh)
			t.Logf("terminal: %s", snap.Snapshot)
		}
		t.Fatal("real harness did not reach the real inference door")
	}
}

func (j *backupJourney) assertLedger(t *testing.T) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		response, err := http.Get(j.door + "/__journey/ledger")
		if err != nil {
			t.Fatal(err)
		}
		var ledger struct{ Charged, Allocated, Remaining int64 }
		err = json.NewDecoder(response.Body).Decode(&ledger)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if ledger.Charged == j.calls.Load()*200 && ledger.Allocated == ledger.Charged && ledger.Remaining == 10000-ledger.Charged {
			t.Logf("real wallet charged=%d allocated=%d remaining=%d", ledger.Charged, ledger.Allocated, ledger.Remaining)
			return
		}
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatalf("wallet did not settle measured native calls: %+v calls=%d", ledger, j.calls.Load())
		}
	}
}

func TestCodingBackupProductionJourney(t *testing.T) {
	check := os.Getenv("CODING_BACKUP_JOURNEY")
	if check == "" {
		t.Skip("called by named J6 journey")
	}
	j := newBackupJourney(t)
	old := j.seedLimit(t)
	switch check {
	case "TestJ6_OneTapRestartOnCredits":
		j.consent(t, 5000, false)
		createdID := j.browserRestart(t, old)
		j.waitPaid(t, createdID)
		if j.server.conversations.CountSessionEvents(j.owner, old) == 0 {
			t.Fatal("original conversation lost")
		}
	case "TestJ6_NoCreditsSpentWithoutConsent":
		status, _ := j.request(t, "POST", "/api/v1/coding-backup/restart", map[string]string{"session_id": old, "source": "credits"})
		if status != 409 {
			t.Fatalf("unconsented restart returned %d", status)
		}
		if j.calls.Load() != 0 {
			t.Fatal("unconsented credits spent")
		}
	case "TestJ6_NewSessionsSwitchWithConsentAndCap":
		j.consent(t, 1000, true)
		in := sessionsH.CreateInput{AgentLaunch: &sessionsH.AgentLaunch{AgentID: "codex"}, Shell: "/bin/bash", Cols: 100, Rows: 30, Backend: "standard", LaunchCommand: "codex exec --skip-git-repo-check --json 'Say done without tools'", ExecuteLaunchCommand: true, AgentType: "codex", Origin: "ui", Owner: codingBackupOwner(j.owner), WorkingDir: j.home}
		created, err := j.server.desktopCreateSession(j.owner, in)
		if err != nil {
			t.Fatal(err)
		}
		j.waitPaid(t, created.ID)
		// A second child shares the exhausted reservation capacity: its first
		// quote cannot fit the remaining 800 credits under this 1000-credit parent.
		second, err := j.server.desktopCreateSession(j.owner, in)
		if err != nil {
			t.Fatal(err)
		}
		env := activePaidCodingSessions.Environment(second.ID)
		if env == nil {
			t.Fatal("second session missing scoped child")
		}
		req, _ := http.NewRequest("POST", j.door+"/api/v1/ai/responses", strings.NewReader(`{"model":"fixture-model","stream":true,"input":"cap probe"}`))
		req.Header.Set("Authorization", "Bearer "+env["VROOLI_CODING_SESSION_KEY"])
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 402 {
			t.Fatalf("shared cap bypassed: %d", resp.StatusCode)
		}
	case "TestJ6_UnattendedRunSwitchesAndNotifies":
		j.consent(t, 5000, true)
		cmd := exec.Command("go", "test", "-tags=journey", "./internal/orchestration", "-run", "^TestCodingBackupUnattendedJourney$", "-v", "-count=1", "-timeout", "10m")
		cmd.Dir = "../../agent-manager/api"
		cmd.Env = append(os.Environ(), "GOWORK=off", "AM_BACKUP_JOURNEY=1", "AM_BACKUP_AUTHORITY_PATH="+filepath.Join(j.home, "fixture-credentials.enc"), "AM_BACKUP_OWNER="+codingBackupOwner(j.owner))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("real AM backup: %v\n%s", err, output)
		}
		t.Log(string(output))
		if j.calls.Load() == 0 {
			t.Fatal("AM did not reach billed inference")
		}
	case "TestJ6_SwitchBackWorks":
		j.consent(t, 5000, true)
		status, data := j.request(t, "POST", "/api/v1/coding-backup/restart", map[string]string{"session_id": old, "source": "credits"})
		if status != 200 {
			t.Fatalf("initial paid restart: %d %s", status, data)
		}
		var paid struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &paid); err != nil {
			t.Fatal(err)
		}
		j.waitPaid(t, paid.ID)
		if activePaidCodingSessions.Environment(paid.ID) == nil {
			t.Fatal("paid session never became active")
		}
		t.Log("SM8 trace: subscription -> structured synthetic limit -> consent -> paid replacement -> real inference")
		status, data = j.request(t, "POST", "/api/v1/coding-backup/restart", map[string]string{"session_id": paid.ID, "source": "subscription"})
		if status != 200 {
			t.Fatalf("switch back: %d %s", status, data)
		}
		var created struct {
			ID string `json:"id"`
		}
		json.Unmarshal(data, &created)
		if activePaidCodingSessions.Environment(created.ID) != nil {
			t.Fatal("own-account replacement received paid credential")
		}
		row, err := j.server.sessionStore.Get(j.owner, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if row.BusinessAccountID != "" || strings.Contains(row.LaunchCommand, "vrooli-source") {
			t.Fatal("own-account path retained billed source")
		}
		if created.ID == paid.ID || created.ID == old {
			t.Fatal("switch-back reused the prior native session")
		}
		if _, err := j.server.sessionStore.Get(j.owner, paid.ID); err != nil {
			t.Fatal("paid session history lost", err)
		}
		if j.server.conversations.CountSessionEvents(j.owner, old) == 0 {
			t.Fatal("original context lost")
		}
		t.Log("SM8 trace: paid active -> explicit switch-back -> distinct subscription replacement without paid environment; prior sessions retained")
		if sess, ok := j.server.sessions.Get(created.ID); ok {
			_ = sess.SendInput(session.InputText("\x03"))
		}
	default:
		t.Fatalf("unknown journey %s", check)
	}
	j.assertLedger(t)
}

func (j *backupJourney) browserRestart(t *testing.T, old string) string {
	t.Helper()
	evidence := os.Getenv("CODING_BACKUP_EVIDENCE")
	if evidence == "" {
		evidence = "/home/matthalloran8/.vrooli/evidence/agent-connect/t2-backup-browser"
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	dist := filepath.Join(j.home, "ui-dist")
	build := exec.Command("node_modules/.bin/vite", "build", "--outDir", dist)
	build.Dir = "../ui"
	if data, err := build.CombinedOutput(); err != nil {
		t.Fatalf("served UI build: %v %s", err, data)
	}
	files := http.FileServer(http.Dir(dist))
	api := j.api.Config.Handler
	served := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/vrooli.") || r.URL.Path == "/health" {
			api.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
	defer served.Close()
	// The Python package is installed for the operator, while Chrome's profile
	// and every application credential stay inside this fixture's isolated home.
	script, err := filepath.Abs("../test/coding_backup_journey.py")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", script, served.URL, old, evidence)
	cmd.Env = append(os.Environ(), "BACKUP_LOCAL_TOKEN="+j.token, "PYTHONPATH=/home/matthalloran8/.local/lib/python3.12/site-packages")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("operated backup UI: %v\n%s", err, output)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "CREATED_SESSION=") {
			return strings.TrimPrefix(line, "CREATED_SESSION=")
		}
	}
	t.Fatalf("browser did not return created session: %s", output)
	return ""
}
