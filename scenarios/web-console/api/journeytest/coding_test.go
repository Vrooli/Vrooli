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
	"os"
	"os/exec"
	"path/filepath"
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
			delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "call_read", "type": "function", "function": map[string]string{"name": "read", "arguments": fmt.Sprintf(`{"filePath":%q}`, fixturePath)}}}}
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
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "opencode", WorkingDir: home, Environment: environment, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: []string{"run", "--pure", "--auto", "--dir", home, "--format", "json", "Read fixture.txt and repeat its contents."}, Stdout: &output, Stderr: &output,
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
	err := cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "opencode", WorkingDir: home, Environment: env, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: []string{"run", "--pure", "--auto", "--dir", home, "--format", "json", "Say source-verified."}, Stdout: &output, Stderr: &output,
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
