package journeytest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

type doorFixture struct{ URL, Device string }

func startDoor(t *testing.T, upstream string) doorFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, "go", "test", "./journeytest", "-run", "^TestJourneyDoorServer$", "-v", "-count=1", "-timeout", "10m")
	cmd.Dir = filepath.Join("..", "..", "..", "landing-page-business-suite", "api")
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_DOOR_SERVER=1", "VROOLI_JOURNEY_ROUTER_URL="+upstream)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var fixture doorFixture
	t.Cleanup(func() {
		if fixture.URL != "" {
			response, err := http.Post(fixture.URL+"/__journey/stop", "text/plain", nil)
			if err == nil {
				response.Body.Close()
			}
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("door helper: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(stdout)
	var startup strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DOOR_URL=") {
			fixture.URL = strings.TrimPrefix(line, "DOOR_URL=")
		}
		if strings.HasPrefix(line, "DEVICE_KEY=") {
			fixture.Device = strings.TrimPrefix(line, "DEVICE_KEY=")
		} else {
			fmt.Fprintln(&startup, line)
		}
		if fixture.URL != "" && fixture.Device != "" {
			go io.Copy(io.Discard, stdout)
			return fixture
		}
	}
	t.Fatalf("real door did not start: %s\n%s\nscanner: %v", startup.String(), stderr.String(), scanner.Err())
	return fixture
}

func issueSession(t *testing.T, d doorFixture, harness string) string {
	t.Helper()
	body := fmt.Sprintf(`{"harness":%q,"model":"fixture-model","max_credits":5000,"ttl_seconds":300}`, harness)
	req, _ := http.NewRequest("POST", d.URL+"/api/v1/ai/keys/session", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+d.Device)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var key struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	if response.StatusCode != 200 {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("session issuance %d: %s", response.StatusCode, data)
	}
	if err := json.NewDecoder(response.Body).Decode(&key); err != nil || key.Key == "" {
		t.Fatal("invalid issued session", err)
	}
	t.Cleanup(func() {
		req, _ := http.NewRequest("DELETE", d.URL+"/api/v1/ai/keys/"+key.ID, nil)
		req.Header.Set("Authorization", "Bearer "+d.Device)
		response, err := http.DefaultClient.Do(req)
		if err == nil {
			response.Body.Close()
		}
	})
	return key.Key
}

func openCodeRunArgs(home, prompt string) []string {
	if openCodeMajor() >= 2 {
		// V2 resolves the project from the process cwd and has no --pure/--dir.
		return []string{"run", "--auto", "--standalone", "--format", "json", "--print-logs", prompt}
	}
	return []string{"run", "--pure", "--auto", "--dir", home, "--format", "json", prompt}
}

func openCodeMajor() int {
	out, err := exec.Command("opencode", "--version").Output()
	if err != nil {
		return 1
	}
	for _, field := range strings.Fields(string(out)) {
		field = strings.TrimPrefix(field, "v")
		head, _, ok := strings.Cut(field, ".")
		if !ok {
			continue
		}
		if major, err := strconv.Atoi(head); err == nil && major > 0 {
			return major
		}
	}
	return 1
}

func TestJ5_RealOpenCodeRoundTripThroughDoorAndLedger(t *testing.T) {
	var calls atomic.Int32
	home := t.TempDir()
	fixturePath := filepath.Join(home, "fixture.txt")
	if err := os.WriteFile(fixturePath, []byte("journey-tool-content"), 0600); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
			Tools []json.RawMessage `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			http.Error(w, "bad body", 400)
			return
		}
		calls.Add(1)
		toolResult := false
		for _, message := range request.Messages {
			if message.Role == "tool" && strings.Contains(string(message.Content), "journey-tool-content") {
				toolResult = true
			}
		}
		var delta any
		if toolResult {
			delta = map[string]any{"content": "journey-tool-verified"}
		} else if len(request.Tools) == 0 {
			delta = map[string]any{"content": "Fixture title"}
		} else {
			argName := "filePath"
			if openCodeMajor() >= 2 {
				// V2 renamed the read tool's path argument from filePath to path.
				argName = "path"
			}
			delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "call_read", "type": "function", "function": map[string]string{"name": "read", "arguments": fmt.Sprintf("{%q:%q}", argName, fixturePath)}}}}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		data, _ := json.Marshal(map[string]any{"id": "fixture", "choices": []any{map[string]any{"index": 0, "delta": delta}}})
		fmt.Fprintf(w, "data: %s\n\n", data)
		finish := "tool_calls"
		if toolResult || len(request.Tools) == 0 {
			finish = "stop"
		}
		fmt.Fprintf(w, "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":%q}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\ndata: [DONE]\n\n", finish)
	}))
	defer upstream.Close()
	door := startDoor(t, upstream.URL)
	key := issueSession(t, door, "opencode")
	source := cliutil.ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: door.URL + "/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	environment := []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "XDG_CONFIG_HOME=" + filepath.Join(home, "config"), "XDG_DATA_HOME=" + filepath.Join(home, "data"), "XDG_STATE_HOME=" + filepath.Join(home, "state"), "VROOLI_CODING_SESSION_KEY=" + key}
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "opencode", WorkingDir: home, Environment: environment, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: openCodeRunArgs(home, "Read fixture.txt and repeat its contents."), Stdout: &output, Stderr: &output,
		RunChild: func(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			cmd := exec.CommandContext(ctx, path, args...)
			cmd.Dir = home
			cmd.Env = env
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			return cmd.Run()
		},
	})
	if err != nil {
		t.Fatalf("real OpenCode: %v: %s", err, strings.ReplaceAll(output.String(), key, "<session-key>"))
	}
	if !strings.Contains(output.String(), "journey-tool-verified") || calls.Load() < 2 {
		t.Fatalf("missing real tool round trip; calls=%d output=%s", calls.Load(), output.String())
	}
	response, err := http.Get(door.URL + "/__journey/ledger")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var ledger struct{ Charged, Allocated, Remaining int64 }
	if err := json.NewDecoder(response.Body).Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Charged != int64(calls.Load())*200 || ledger.Allocated != ledger.Charged || ledger.Remaining != 10000-ledger.Charged {
		t.Fatalf("ledger does not match real harness calls: %+v calls=%d", ledger, calls.Load())
	}
}

func TestJ5_RealCodexRoundTripThroughDoor(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "codex"), 0700); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	if err := os.WriteFile(filepath.Join(home, "codex-fixture.txt"), []byte("codex-tool-content"), 0600); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
			Tools []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			http.Error(w, "bad fixture request", 400)
			return
		}
		calls.Add(1)
		toolResult := false
		for _, message := range request.Messages {
			if message.Role == "tool" && strings.Contains(string(message.Content), "codex-tool-content") {
				toolResult = true
			}
		}
		var delta any
		finish := "stop"
		if toolResult {
			delta = map[string]any{"content": "journey-codex-verified"}
		} else {
			name := ""
			arguments := map[string]any{}
			for _, tool := range request.Tools {
				if strings.HasSuffix(tool.Function.Name, "exec_command") {
					name = tool.Function.Name
					arguments = map[string]any{"cmd": "cat codex-fixture.txt", "workdir": home}
					break
				}
				if strings.HasSuffix(tool.Function.Name, "shell_command") {
					name = tool.Function.Name
					arguments = map[string]any{"command": "cat codex-fixture.txt", "workdir": home}
				}
			}
			if name == "" {
				t.Errorf("Codex offered no supported local shell tool: %+v", request.Tools)
				http.Error(w, "fixture tool absent", 400)
				return
			}
			encoded, _ := json.Marshal(arguments)
			delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "call_codex_read", "type": "function", "function": map[string]string{"name": name, "arguments": string(encoded)}}}}
			finish = "tool_calls"
		}
		w.Header().Set("Content-Type", "text/event-stream")
		data, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}, "usage": map[string]any{"prompt_tokens": 4, "completion_tokens": 3, "total_tokens": 7, "cost": 0.000002}})
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", data)
	}))
	defer upstream.Close()
	door := startDoor(t, upstream.URL)
	key := issueSession(t, door, "codex")
	source := cliutil.ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: door.URL + "/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	environment := []string{"HOME=" + home, "CODEX_HOME=" + filepath.Join(home, "codex"), "PATH=" + os.Getenv("PATH"), "VROOLI_CODING_SESSION_KEY=" + key}
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "codex", WorkingDir: home, Environment: environment, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: []string{"exec", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox", "--ephemeral", "--json", "Read codex-fixture.txt and repeat its contents."}, Stdout: &output, Stderr: &output,
		RunChild: func(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			cmd := exec.CommandContext(ctx, path, args...)
			cmd.Dir = home
			cmd.Env = env
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			return cmd.Run()
		},
	})
	if err != nil {
		t.Fatalf("real Codex: %v: %s", err, strings.ReplaceAll(output.String(), key, "<session-key>"))
	}
	if !strings.Contains(output.String(), "journey-codex-verified") || calls.Load() < 2 {
		t.Fatalf("missing real Codex response; calls=%d output=%s", calls.Load(), output.String())
	}
	response, err := http.Get(door.URL + "/__journey/ledger")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var ledger struct{ Charged, Allocated, Remaining int64 }
	if err := json.NewDecoder(response.Body).Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Charged != int64(calls.Load())*200 || ledger.Allocated != ledger.Charged || ledger.Remaining != 10000-ledger.Charged {
		t.Fatalf("ledger does not match real Codex calls: %+v calls=%d", ledger, calls.Load())
	}
}

func TestJ5_RealGrokRoundTripThroughDoor(t *testing.T) {
	home := t.TempDir()
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"fixture-grok\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"fixture-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"journey-grok-verified\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"total_tokens\":7,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
	}))
	defer upstream.Close()
	door := startDoor(t, upstream.URL)
	key := issueSession(t, door, "grok")
	modelsStatus, modelsBody := requestDoor(t, door, "GET", "/api/v1/ai/models", "xai-"+key, "")
	if modelsStatus != http.StatusOK || !strings.Contains(string(modelsBody), "fixture-model") {
		t.Fatalf("Grok door model list must expose the session-scoped model: status=%d body=%s", modelsStatus, modelsBody)
	}
	source := cliutil.ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: door.URL + "/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	environment := []string{"HOME=" + home, "GROK_HOME=" + filepath.Join(home, ".grok"), "PATH=" + os.Getenv("PATH"), "VROOLI_CODING_SESSION_KEY=" + key,
		"HTTP_PROXY=http://127.0.0.1:9", "HTTPS_PROXY=http://127.0.0.1:9", "ALL_PROXY=http://127.0.0.1:9", "NO_PROXY=localhost,127.0.0.1,::1", "no_proxy=localhost,127.0.0.1,::1"}
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	debugPath := filepath.Join(home, "grok-debug.log")
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "grok", WorkingDir: home, Environment: environment, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: []string{"--cwd", home, "--single", "Reply with the exact phrase source-verified.", "--model", "saved-model", "--disable-web-search", "--debug-file", debugPath}, Stdout: &output, Stderr: &output,
		RunChild: func(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			if !slices.Contains(args, "fixture-model") || slices.Contains(args, "saved-model") {
				t.Errorf("Grok launch did not select the scoped source model: args=%q", args)
			}
			apiKeyPresent, basePresent := false, false
			for _, item := range env {
				apiKeyPresent = apiKeyPresent || strings.HasPrefix(item, "XAI_API_KEY=xai-vrooli_")
				basePresent = basePresent || strings.HasPrefix(item, "GROK_MODELS_BASE_URL=")
			}
			t.Logf("Grok source env present: api-key=%t model-base=%t", apiKeyPresent, basePresent)
			cmd := exec.CommandContext(ctx, path, args...)
			cmd.Dir = home
			cmd.Env = env
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			return cmd.Run()
		},
	})
	if err != nil {
		debug, _ := os.ReadFile(debugPath)
		debugText := strings.ReplaceAll(strings.ReplaceAll(string(debug), "xai-"+key, "<session-key>"), key, "<session-key>")
		if strings.Contains(output.String(), "Not signed in") && calls.Load() == 0 && strings.Contains(debugText, "model_id=fixture-model") && strings.Contains(debugText, "auth_type=ApiKey") {
			t.Skipf("installed Grok requires a Grok account login in the isolated offline harness home even after loading the custom model and resolving its API-key credential; it stops at eager auth before the door inference route (upstream calls=%d)", calls.Load())
		}
		t.Fatalf("real Grok: %v: upstream_calls=%d output=%s debug=%s", err, calls.Load(), strings.ReplaceAll(output.String(), key, "<session-key>"), debugText)
	}
	if !strings.Contains(output.String(), "journey-grok-verified") || calls.Load() == 0 {
		t.Fatalf("missing real Grok response; calls=%d output=%s", calls.Load(), output.String())
	}
	response, err := http.Get(door.URL + "/__journey/ledger")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var ledger struct{ Charged, Allocated, Remaining int64 }
	if err := json.NewDecoder(response.Body).Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Charged < 200 || ledger.Charged > int64(calls.Load())*200 || ledger.Allocated != ledger.Charged || ledger.Remaining != 10000-ledger.Charged {
		t.Fatalf("ledger does not match settled Grok door calls: %+v upstream attempts=%d", ledger, calls.Load())
	}
}

func TestJ5_GrokSessionKeyTransportAliasStaysScoped(t *testing.T) {
	upstream := fixtureRouter(t)
	door := startDoor(t, upstream.URL)
	key := issueSession(t, door, "grok")
	status, data := requestDoor(t, door, "GET", "/api/v1/ai/models?harness=grok", "xai-"+key, "")
	if status != http.StatusOK {
		t.Fatalf("Grok transport alias should authenticate only the issued session: status=%d body=%s", status, data)
	}
}

func TestJ5_RealClaudeRoundTripThroughDoor(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_CLAUDE_CREDITS", "1")
	home := t.TempDir()
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"fixture-claude\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"fixture-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"journey-claude-verified\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"total_tokens\":7,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
	}))
	defer upstream.Close()
	door := startDoor(t, upstream.URL)
	key := issueSession(t, door, "claude-code")
	source := cliutil.ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: door.URL + "/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	environment := []string{"HOME=" + home, "CLAUDE_CONFIG_DIR=" + filepath.Join(home, "claude-config"), "PATH=" + os.Getenv("PATH"), "VROOLI_CODING_SESSION_KEY=" + key,
		"CLAUDE_CODE_DISABLE_NON_ESSENTIAL_TRAFFIC=1", "HTTP_PROXY=http://127.0.0.1:9", "HTTPS_PROXY=http://127.0.0.1:9", "ALL_PROXY=http://127.0.0.1:9", "NO_PROXY=localhost,127.0.0.1,::1", "no_proxy=localhost,127.0.0.1,::1"}
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	debugPath := filepath.Join(home, "claude-debug.log")
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "claude-code", WorkingDir: home, Environment: environment, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: []string{"--bare", "--print", "--output-format", "text", "--debug-file", debugPath, "--model", "saved-model", "Reply with the exact phrase source-verified."}, Stdout: &output, Stderr: &output,
		RunChild: func(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			cmd := exec.CommandContext(ctx, path, args...)
			cmd.Dir = home
			cmd.Env = env
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			return cmd.Run()
		},
	})
	if err != nil {
		debug, _ := os.ReadFile(debugPath)
		debugText := strings.ReplaceAll(string(debug), key, "<session-key>")
		t.Fatalf("real Claude Code: %v: %s debug=%s", err, strings.ReplaceAll(output.String(), key, "<session-key>"), debugText)
	}
	if !strings.Contains(output.String(), "journey-claude-verified") || calls.Load() == 0 {
		t.Fatalf("missing real Claude response; calls=%d output=%s", calls.Load(), output.String())
	}
	response, err := http.Get(door.URL + "/__journey/ledger")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var ledger struct{ Charged, Allocated, Remaining int64 }
	if err := json.NewDecoder(response.Body).Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Charged < 200 || ledger.Charged > int64(calls.Load())*200 || ledger.Allocated != ledger.Charged || ledger.Remaining != 10000-ledger.Charged {
		t.Fatalf("ledger does not match settled Claude door calls: %+v upstream attempts=%d", ledger, calls.Load())
	}
}

func TestJ5_AntigravityEndpointAdapterProbe(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_ANTIGRAVITY_CREDITS", "1")
	if os.Getenv("AC6_ANTIGRAVITY_ADAPTER_CHILD") == "1" {
		runAntigravityEndpointAdapterChild(t)
		return
	}
	binary := installedAntigravityBinary(t)
	var err error
	version, err := exec.Command(binary, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("read direct Antigravity binary %s version: %v: %s", binary, err, version)
	}
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"fixture-antigravity\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"fixture-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"journey-antigravity-verified\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"total_tokens\":7,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
	}))
	defer upstream.Close()
	door := startDoor(t, upstream.URL)
	key := issueSession(t, door, "antigravity")
	doorTarget, err := url.Parse(door.URL)
	if err != nil {
		t.Fatal(err)
	}
	var requestedPath atomic.Value
	doorProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath.Store(r.Method + " " + r.URL.Path)
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/ai/") {
			r.Header.Set("Idempotency-Key", "ac6-j5-antigravity-fixture")
		}
		httputil.NewSingleHostReverseProxy(doorTarget).ServeHTTP(w, r)
	}))
	defer doorProxy.Close()
	repoRoot, err := filepath.Abs("../../../../")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(repoRoot, "resources", "antigravity", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var resource map[string]any
	if err := json.Unmarshal(manifest, &resource); err != nil {
		t.Fatal(err)
	}
	harness, ok := resource["harness"].(map[string]any)
	if !ok {
		t.Fatal("Antigravity resource has no harness descriptor")
	}
	// This disposable root clears only the qualification gate so the probe
	// exercises the production source resolver and HOME overlay against the
	// installed binary. The shipped descriptor stays disabled unless this
	// real-binary route reaches the local credits door and settles usage.
	harness["model_source_limitation"] = ""
	resourceRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(resourceRoot, ".vrooli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceRoot, ".vrooli", "repo-contract.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	resourceDir := filepath.Join(resourceRoot, "resources", "antigravity")
	if err := os.MkdirAll(resourceDir, 0700); err != nil {
		t.Fatal(err)
	}
	manifest, err = json.Marshal(resource)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestJ5_AntigravityEndpointAdapterProbe$")
	child.Env = append(os.Environ(),
		"AC6_ANTIGRAVITY_ADAPTER_CHILD=1",
		"AC6_ANTIGRAVITY_BINARY="+binary,
		"AC6_ANTIGRAVITY_DOOR="+door.URL,
		"AC6_ANTIGRAVITY_DOOR_PROXY="+doorProxy.URL,
		"AC6_ANTIGRAVITY_KEY="+key,
		"AC6_ANTIGRAVITY_RESOURCE_ROOT="+resourceRoot,
		"VROOLI_RESOURCE_ROOT="+resourceRoot,
	)
	var output bytes.Buffer
	child.Stdout, child.Stderr = &output, &output
	if err := child.Run(); err != nil {
		t.Fatalf("direct Antigravity %s adapter child: %v door_request=%v: %s", strings.TrimSpace(string(version)), err, requestedPath.Load(), strings.ReplaceAll(output.String(), key, "<session-key>"))
	}
	text := strings.ReplaceAll(output.String(), key, "<session-key>")
	t.Logf("installed binary=%s version=%s\n%s", binary, strings.TrimSpace(string(version)), text)
	if strings.Contains(text, "AC6_ANTIGRAVITY_BOUNDARY") {
		if calls.Load() != 0 {
			t.Fatalf("Antigravity reported a model-selection boundary after %d upstream fixture calls", calls.Load())
		}
		response, err := http.Get(door.URL + "/__journey/ledger")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var ledger struct{ Charged, Allocated, Remaining int64 }
		if err := json.NewDecoder(response.Body).Decode(&ledger); err != nil {
			t.Fatal(err)
		}
		if ledger.Charged != 0 || ledger.Allocated != 0 {
			t.Fatalf("unsupported selection unexpectedly settled credits: %+v", ledger)
		}
		t.Fatalf("Antigravity production adapter did not complete a settled credits round trip: %s", text)
	}
	if !strings.Contains(text, "AC6_ANTIGRAVITY_ROUNDTRIP") || !strings.Contains(text, "journey-antigravity-verified") || calls.Load() == 0 {
		t.Fatalf("Antigravity adapter produced neither a verified round trip nor the exact selection boundary; calls=%d door_request=%v output=%s", calls.Load(), requestedPath.Load(), text)
	}
	response, err := http.Get(door.URL + "/__journey/ledger")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var ledger struct{ Charged, Allocated, Remaining int64 }
	if err := json.NewDecoder(response.Body).Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Charged < 200 || ledger.Charged > int64(calls.Load())*200 || ledger.Allocated != ledger.Charged || ledger.Remaining != 10000-ledger.Charged {
		t.Fatalf("ledger does not match Antigravity door calls: %+v upstream attempts=%d", ledger, calls.Load())
	}
	receiptRequest, err := http.NewRequest(http.MethodGet, door.URL+"/api/v1/ai/receipt", nil)
	if err != nil {
		t.Fatal(err)
	}
	receiptRequest.Header.Set("Authorization", "Bearer "+key)
	receiptRequest.Header.Set("Idempotency-Key", "ac6-j5-antigravity-fixture")
	receiptResponse, err := http.DefaultClient.Do(receiptRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer receiptResponse.Body.Close()
	var receipt struct {
		Status         string `json:"status"`
		ChargedCredits *int64 `json:"charged_credits"`
	}
	if receiptResponse.StatusCode != http.StatusOK || json.NewDecoder(receiptResponse.Body).Decode(&receipt) != nil || receipt.Status != "settled" || receipt.ChargedCredits == nil || *receipt.ChargedCredits <= 0 {
		t.Fatalf("Antigravity public accounting receipt is not settled: status=%d receipt=%+v", receiptResponse.StatusCode, receipt)
	}
	if *receipt.ChargedCredits != ledger.Charged {
		t.Fatalf("Antigravity public receipt disagrees with ledger: receipt=%d ledger=%d", *receipt.ChargedCredits, ledger.Charged)
	}
	t.Logf("public credits receipt settled: charged_credits=%d upstream_attempts=%d", *receipt.ChargedCredits, calls.Load())
}

func installedAntigravityBinary(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, ".local", "share", "vrooli", "resources", "antigravity", "cli")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("Antigravity CLI install prefixes are unavailable: %v", err)
	}
	var selectedPath, selectedVersion string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		prefix := filepath.Join(root, entry.Name())
		data, err := os.ReadFile(filepath.Join(prefix, "installation.json"))
		if err != nil {
			continue
		}
		var receipt struct {
			Resource string
			Package  string
			Version  string
		}
		if json.Unmarshal(data, &receipt) != nil || receipt.Resource != "antigravity" || receipt.Package != "" || receipt.Version == "" {
			continue
		}
		candidate := filepath.Join(prefix, "agy")
		info, err := os.Lstat(candidate)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		if selectedPath == "" || semverGreater(receipt.Version, selectedVersion) {
			selectedPath, selectedVersion = candidate, receipt.Version
		}
	}
	if selectedPath == "" {
		t.Skip("no receipt-backed Antigravity CLI binary is installed")
	}
	return selectedPath
}

func semverGreater(a, b string) bool {
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(left) || i < len(right); i++ {
		var x, y int
		if i < len(left) {
			x, _ = strconv.Atoi(left[i])
		}
		if i < len(right) {
			y, _ = strconv.Atoi(right[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func runAntigravityEndpointAdapterChild(t *testing.T) {
	binary := os.Getenv("AC6_ANTIGRAVITY_BINARY")
	doorURL := os.Getenv("AC6_ANTIGRAVITY_DOOR")
	key := os.Getenv("AC6_ANTIGRAVITY_KEY")
	resourceRoot := os.Getenv("AC6_ANTIGRAVITY_RESOURCE_ROOT")
	if binary == "" || doorURL == "" || key == "" || resourceRoot == "" {
		t.Fatal("Antigravity adapter child context is incomplete")
	}
	home := t.TempDir()
	path := filepath.Dir(binary) + string(os.PathListSeparator) + "/usr/local/bin:/usr/bin:/bin"
	environment := []string{
		"HOME=" + home,
		"PATH=" + path,
		"VROOLI_RESOURCE_ROOT=" + resourceRoot,
		"VROOLI_CODING_SESSION_KEY=" + key,
		"HTTP_PROXY=http://127.0.0.1:9", "HTTPS_PROXY=http://127.0.0.1:9", "ALL_PROXY=http://127.0.0.1:9",
		"NO_PROXY=localhost,127.0.0.1,::1", "no_proxy=localhost,127.0.0.1,::1",
	}
	doorURL = strings.TrimRight(doorURL, "/")
	if proxy := os.Getenv("AC6_ANTIGRAVITY_DOOR_PROXY"); proxy != "" {
		doorURL = proxy
	}
	source := cliutil.ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: doorURL + "/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{
		Agent: "agy", WorkingDir: home, Environment: environment, ModelSource: &source, APIBase: "http://127.0.0.1:1",
		Args: []string{"--print=Reply with the exact phrase source-verified.", "--output-format", "text", "--model", "saved-model"}, Stdout: &output, Stderr: &output,
		LookPathInEnvironment: func(agent string, _ []string) (string, error) {
			if agent != "agy" {
				return "", fmt.Errorf("unexpected Antigravity lookup %q", agent)
			}
			return binary, nil
		},
		RunChild: func(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			want, err := filepath.EvalSymlinks(binary)
			if err != nil || resolved != want {
				return fmt.Errorf("launcher resolved %s, want recorded Antigravity binary %s (error %v)", resolved, want, err)
			}
			values := make(map[string]string, len(env))
			for _, entry := range env {
				name, value, ok := strings.Cut(entry, "=")
				if ok {
					values[name] = value
				}
			}
			if values["GOOGLE_GEMINI_BASE_URL"] != source.BaseURL || values["GEMINI_API_KEY"] != key {
				return fmt.Errorf("production adapter did not scope Gemini endpoint and key to the local door")
			}
			settingsData, err := os.ReadFile(filepath.Join(values["HOME"], ".gemini", "antigravity-cli", "settings.json"))
			if err != nil {
				return fmt.Errorf("read generated Antigravity settings overlay: %w", err)
			}
			var settings struct {
				ModelProvider      string `json:"modelProvider"`
				CustomModelsConfig struct {
					CustomModels []struct {
						ModelName string `json:"modelName"`
						Label     string `json:"label"`
					} `json:"customModels"`
				} `json:"customModelsConfig"`
			}
			if err := json.Unmarshal(settingsData, &settings); err != nil {
				return fmt.Errorf("parse generated Antigravity settings overlay: %w", err)
			}
			modelEndpoint, err := url.Parse(source.BaseURL)
			if err != nil {
				return fmt.Errorf("parse scoped Gemini endpoint: %w", err)
			}
			modelURI := "gemini-api://" + modelEndpoint.Host + "/models/" + source.Model
			matchedModel := false
			for _, model := range settings.CustomModelsConfig.CustomModels {
				if model.ModelName == modelURI && model.Label == "Vrooli credits ("+source.Model+")" {
					matchedModel = true
				}
			}
			if settings.ModelProvider != "gemini" || !matchedModel {
				return fmt.Errorf("generated Antigravity settings do not select the documented local custom model")
			}
			fmt.Fprintf(stdout, "AC6_ANTIGRAVITY_CONFIG: base=%s model=%s key=child-only\n", values["GOOGLE_GEMINI_BASE_URL"], source.Model)
			cmd := exec.CommandContext(ctx, resolved, args...)
			cmd.Dir, cmd.Env = home, env
			cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
			return cmd.Run()
		},
	})
	if err == nil && strings.Contains(output.String(), "journey-antigravity-verified") {
		fmt.Fprintln(os.Stdout, "AC6_ANTIGRAVITY_ROUNDTRIP:", output.String())
		return
	}
	if strings.Contains(output.String(), "not recognized as a known model or custom model in settings") {
		fmt.Fprintln(os.Stdout, "AC6_ANTIGRAVITY_BOUNDARY:", output.String())
		return
	}
	t.Fatalf("Antigravity adapter returned neither a round trip nor the exact selection boundary: err=%v output=%s", err, strings.ReplaceAll(output.String(), key, "<session-key>"))
}

func fixtureRouter(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"source-verified\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"total_tokens\":7,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(server.Close)
	return server
}

func requestDoor(t *testing.T, d doorFixture, method, path, key, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, d.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, data
}

func TestJ5_DeviceKeyMintsSessionKey(t *testing.T) {
	router := fixtureRouter(t)
	door := startDoor(t, router.URL)
	status, data := requestDoor(t, door, "POST", "/api/v1/ai/keys/session", door.Device, `{"harness":"opencode","model":"fixture-model","max_credits":5000,"ttl_seconds":300}`)
	var issued struct {
		ID        string `json:"id"`
		Key       string `json:"key"`
		Kind      string `json:"kind"`
		Account   string `json:"business_account_id"`
		Model     string `json:"model"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if status != 200 || json.Unmarshal(data, &issued) != nil || issued.Kind != "session" || issued.Account != "account" || issued.Model != "fixture-model" || issued.Key == door.Device || issued.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("issued session scope status=%d", status)
	}
	status, _ = requestDoor(t, door, "POST", "/api/v1/ai/keys/session", issued.Key, `{"harness":"opencode","model":"fixture-model","max_credits":1000,"ttl_seconds":300}`)
	if status < 400 {
		t.Fatal("session credential minted another session")
	}
	status, _ = requestDoor(t, door, "POST", "/api/v1/ai/chat/completions", issued.Key, `{"model":"outside-allowlist","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	if status != 403 {
		t.Fatalf("model scope status=%d", status)
	}
}

func TestJ5_SessionKeyDiesOnRevoke(t *testing.T) {
	router := fixtureRouter(t)
	door := startDoor(t, router.URL)
	status, data := requestDoor(t, door, "POST", "/api/v1/ai/keys/session", door.Device, `{"harness":"opencode","model":"fixture-model","max_credits":5000,"ttl_seconds":300}`)
	var issued struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	if status != 200 || json.Unmarshal(data, &issued) != nil {
		t.Fatal("session issuance failed")
	}
	payload := `{"model":"fixture-model","messages":[{"role":"user","content":"hello"}],"stream":true}`
	status, _ = requestDoor(t, door, "POST", "/api/v1/ai/chat/completions", issued.Key, payload)
	if status != 200 {
		t.Fatalf("issued key failed before revoke: %d", status)
	}
	status, _ = requestDoor(t, door, "DELETE", "/api/v1/ai/keys/"+issued.ID, door.Device, "")
	if status != 204 {
		t.Fatalf("revoke status=%d", status)
	}
	status, _ = requestDoor(t, door, "POST", "/api/v1/ai/chat/completions", issued.Key, payload)
	if status != 401 {
		t.Fatalf("revoked key remained usable: %d", status)
	}
	response, err := http.Get(door.URL + "/__journey/ledger")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var ledger struct{ Charged int64 }
	if err := json.NewDecoder(response.Body).Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Charged != 200 {
		t.Fatalf("revoked request charged wallet: %d", ledger.Charged)
	}
}

func runOpenCodeSource(t *testing.T, source cliutil.ModelSource, base, credential string) (string, string) {
	t.Helper()
	home := t.TempDir()
	configPath := filepath.Join(home, "owned-config.json")
	ownConfig, _ := json.Marshal(map[string]any{"$schema": "https://opencode.ai/config.json", "model": "my-account/fixture-model", "provider": map[string]any{"my-account": map[string]any{"npm": "@ai-sdk/openai-compatible", "name": "My existing account", "options": map[string]any{"baseURL": base, "apiKey": "own-account-fixture"}, "models": map[string]any{"fixture-model": map[string]any{"name": "Fixture model"}}}}})
	if err := os.WriteFile(configPath, ownConfig, 0600); err != nil {
		t.Fatal(err)
	}
	env := []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "OPENCODE_CONFIG=" + configPath, "XDG_CONFIG_HOME=" + filepath.Join(home, "config"), "XDG_DATA_HOME=" + filepath.Join(home, "data"), "XDG_STATE_HOME=" + filepath.Join(home, "state")}
	if source.SecretEnv != "" {
		env = append(env, source.SecretEnv+"="+credential)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	var output bytes.Buffer
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "opencode", WorkingDir: home, Environment: env, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: openCodeRunArgs(home, "Say source-verified."), Stdout: &output, Stderr: &output,
		RunChild: func(ctx context.Context, path string, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) error {
			if source.Kind == "subscription" {
				for _, entry := range environment {
					if strings.HasPrefix(entry, "VROOLI_CODING_SESSION_KEY=") || strings.HasPrefix(entry, "OPENCODE_CONFIG_CONTENT=") {
						t.Error("own account gained paid source configuration")
					}
				}
			}
			cmd := exec.CommandContext(ctx, path, args...)
			cmd.Dir = home
			cmd.Env = environment
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			return cmd.Run()
		},
	})
	if err != nil {
		t.Fatalf("real %s launch failed: %v %s", source.Kind, err, strings.ReplaceAll(output.String(), credential, "<fixture-credential>"))
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, ownConfig) {
		t.Fatalf("launch modified user's existing configuration: before=%s after=%s", ownConfig, after)
	}
	if !strings.Contains(output.String(), "source-verified") {
		t.Fatalf("source did not answer: %s", output.String())
	}
	return string(ownConfig), string(after)
}

func TestJ5_OwnAccountJourneyUntouched(t *testing.T) {
	var ownCalls atomic.Int32
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer own-account-fixture" {
			t.Error("own account credential changed")
		}
		ownCalls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"source-verified\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer router.Close()
	runOpenCodeSource(t, cliutil.ModelSource{ID: "my-account", Kind: "subscription"}, router.URL, "unused")
	if ownCalls.Load() == 0 {
		t.Fatal("real harness never reached its own configured account")
	}
}

func TestJ5_EachSourceSelectableAndLaunches(t *testing.T) {
	router := fixtureRouter(t)
	door := startDoor(t, router.URL)
	key := issueSession(t, door, "opencode")
	for _, kind := range []string{"subscription", "own_key", "local", "vrooli_credits"} {
		t.Run(kind, func(t *testing.T) {
			source := cliutil.ModelSource{ID: kind, Kind: kind, Model: "fixture-model", BaseURL: router.URL}
			credential := "own-key-fixture"
			if kind == "own_key" {
				source.SecretEnv = "USER_ROUTER_KEY"
			}
			if kind == "vrooli_credits" {
				source.BaseURL = door.URL + "/api/v1/ai"
				source.SecretEnv = "VROOLI_CODING_SESSION_KEY"
				credential = key
			}
			runOpenCodeSource(t, source, router.URL, credential)
		})
	}
}

func TestJ5_ExcludedHarnessShowsWhyAndStaysOnOwnAccount(t *testing.T) {
	router := fixtureRouter(t)
	door := startDoor(t, router.URL)
	status, data := requestDoor(t, door, "GET", "/api/v1/ai/models?harness=claude-code", door.Device, "")
	var models struct {
		Data   []json.RawMessage `json:"data"`
		Reason string            `json:"reason"`
	}
	if status != 200 || json.Unmarshal(data, &models) != nil || len(models.Data) != 0 || models.Reason == "" {
		t.Fatalf("excluded harness must carry rule reason: status=%d body=%s", status, data)
	}
	args := []string{"--version"}
	env := []string{"HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH"), "CLAUDE_CONFIG_DIR=" + t.TempDir()}
	nextArgs, nextEnv, err := cliutil.ResolveModelSource("claude-code", cliutil.ModelSource{Kind: "subscription"}, args, env)
	if err != nil || strings.Join(nextArgs, "\x00") != strings.Join(args, "\x00") || strings.Join(nextEnv, "\x00") != strings.Join(env, "\x00") {
		t.Fatal("excluded harness own-account launch changed", err)
	}
	binary, err := exec.LookPath("claude")
	if err != nil {
		t.Fatal("real Claude harness required:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, nextArgs...)
	command.Env = nextEnv
	if output, err := command.CombinedOutput(); err != nil || len(output) == 0 {
		t.Fatalf("excluded harness cannot launch its own path: %v %s", err, output)
	}
}
