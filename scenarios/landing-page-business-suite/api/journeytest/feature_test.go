package journeytest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"landing-page-business-suite-api/internal/aidoor"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestJ4_AudioToolsKeepsItsPromptAndIsBilled(t *testing.T) {
	var mu sync.Mutex
	var calls atomic.Int64
	var messages []struct{ Role, Content string }
	s := newSpine(t, func(raw []byte) {
		calls.Add(1)
		var request struct {
			Messages []struct{ Role, Content string }
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			t.Error(err)
		}
		mu.Lock()
		messages = request.Messages
		mu.Unlock()
	})
	// Mint the actual device credential through production consumer auth and
	// route registration; the Audio client then crosses that mounted feature route.
	base, refresh := startConsumerDoor(t, s)
	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	resp, err := http.Post(base+"/api/v1/auth/refresh", "application/json", strings.NewReader(string(refreshBody)))
	if err != nil {
		t.Fatal(err)
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || tokens.AccessToken == "" {
		t.Fatalf("consumer refresh status=%d", resp.StatusCode)
	}
	keyReq, _ := http.NewRequest("POST", base+"/api/v1/ai/keys/device", strings.NewReader(`{"business_account_id":"account","app_bundle_key":"audio-tools","max_credits":10000}`))
	keyReq.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	keyReq.Header.Set("Content-Type", "application/json")
	keyResp, err := http.DefaultClient.Do(keyReq)
	if err != nil {
		t.Fatal(err)
	}
	var device aidoor.Key
	if err := json.NewDecoder(keyResp.Body).Decode(&device); err != nil {
		t.Fatal(err)
	}
	keyResp.Body.Close()
	if keyResp.StatusCode != 200 || device.Token == "" {
		t.Fatalf("device issuance status=%d", keyResp.StatusCode)
	}
	s.key = device
	cmd := exec.Command("go", "test", "./integrations/lpbs/clients", "-run", "^TestJourneyAudioSummary$", "-v", "-count=1")
	cmd.Dir = "../../../audio-tools/api"
	cmd.Env = append(os.Environ(), "GOWORK=off", "J4_DOOR_URL="+base, "J4_DEVICE_KEY="+s.key.Token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real Audio client: %v\n%s", err, out)
	}
	t.Log(string(out))
	status, body := postFeature(t, s, "audio-tools.summarize", "j4-audio-summary", audioFeatureInput(t))
	if status != 409 || calls.Load() != 1 {
		t.Fatalf("replay status=%d calls=%d %s", status, calls.Load(), body)
	}
	var operation string
	if err := s.db.QueryRow(`SELECT operation_id FROM business_account_usage_events`).Scan(&operation); err != nil {
		t.Fatal(err)
	}
	receipt, err := http.Get(s.gateway + "/api/v1/private/provider-routes/receipt/" + operation)
	if err != nil {
		t.Fatal(err)
	}
	defer receipt.Body.Close()
	var measured struct {
		FeatureID string `json:"feature_id"`
		State     string
		Credits   int64
		CostKnown bool `json:"cost_known"`
	}
	if err := json.NewDecoder(receipt.Body).Decode(&measured); err != nil {
		t.Fatal(err)
	}
	if measured.FeatureID != "audio-tools.summarize" || measured.State != "completed" || measured.Credits != 200 || !measured.CostKnown {
		t.Fatalf("receipt=%+v", measured)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(messages) != 2 || messages[0].Role != "system" || messages[1].Role != "user" {
		t.Fatalf("messages: %+v", messages)
	}
	var payload struct{ Text, Level string }
	if err := json.Unmarshal([]byte(messages[1].Content), &payload); err != nil || payload.Level != "heavy" || payload.Text != "Our release is ready. Deploy it tomorrow." {
		t.Fatalf("data: %+v %v", payload, err)
	}
	var feature string
	var charged, allocated int64
	if err := s.db.QueryRow(`SELECT feature_id, credits FROM business_account_usage_events`).Scan(&feature, &charged); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations`).Scan(&allocated); err != nil {
		t.Fatal(err)
	}
	if feature != "audio-tools.summarize" || charged != 200 || allocated != 200 {
		t.Fatalf("feature=%s charged=%d allocated=%d", feature, charged, allocated)
	}
}

// Read the scenario-owned constants rather than duplicating its instruction in LPBS.
func audioFeatureInput(t *testing.T) map[string]any {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "../../../audio-tools/api/integrations/lpbs/clients/summarize_client.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		if v, ok := n.(*ast.ValueSpec); ok && len(v.Names) == 1 && len(v.Values) == 1 {
			if lit, ok := v.Values[0].(*ast.BasicLit); ok {
				values[v.Names[0].Name], _ = strconv.Unquote(lit.Value)
			}
		}
		return true
	})
	if values["SummarizeInstruction"] == "" || values["SummarizeSchema"] == "" {
		t.Fatal("Audio contract constants absent")
	}
	return map[string]any{"instruction": values["SummarizeInstruction"], "schema": json.RawMessage(values["SummarizeSchema"]), "turn": map[string]any{"role": "user", "content": `{"text":"Deploy tomorrow","level":"heavy"}`}}
}

func postFeature(t *testing.T, s *spine, feature, operation string, input any) (int, []byte) {
	t.Helper()
	body, _ := json.Marshal(input)
	req, err := http.NewRequest("POST", s.server.URL+"/api/v1/features/"+feature, strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+s.key.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", operation)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}

func assertUncharged(t *testing.T, s *spine) {
	t.Helper()
	var charged, held int64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events`).Scan(&charged); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE status IN ('pending','uncertain')`).Scan(&held); err != nil {
		t.Fatal(err)
	}
	if charged != 0 || held != 0 {
		t.Fatalf("refusal charged=%d holds=%d", charged, held)
	}
}

func TestJ4_UnknownDigestRefused(t *testing.T) {
	var calls atomic.Int64
	s := newSpine(t, func([]byte) { calls.Add(1) })
	input := audioFeatureInput(t)
	input["instruction"] = "attacker-selected instruction"
	status, body := postFeature(t, s, "audio-tools.summarize", "bad-digest", input)
	if status < 400 {
		t.Fatalf("accepted unknown instruction: %d %s", status, body)
	}
	status, body = postFeature(t, s, "unknown.feature", "unknown", audioFeatureInput(t))
	if status < 400 {
		t.Fatalf("accepted unknown feature: %d %s", status, body)
	}
	if calls.Load() != 0 {
		t.Fatal("refused request reached supplier")
	}
	assertUncharged(t, s)
}

func TestJ4_ToolsAndSecondTurnRefused(t *testing.T) {
	var calls atomic.Int64
	s := newSpine(t, func([]byte) { calls.Add(1) })
	for _, field := range []string{"tools", "turns", "messages", "use_class", "outside_funded"} {
		t.Run(field, func(t *testing.T) {
			input := audioFeatureInput(t)
			input[field] = []any{map[string]string{"role": "user", "content": "second turn"}}
			status, body := postFeature(t, s, "audio-tools.summarize", field, input)
			if status != 400 {
				t.Fatalf("unbounded field status=%d %s", status, body)
			}
		})
	}
	input := audioFeatureInput(t)
	input["schema"] = map[string]string{"type": "object"}
	status, body := postFeature(t, s, "audio-tools.summarize", "schema", input)
	if status < 400 {
		t.Fatalf("changed schema accepted: %d %s", status, body)
	}
	if calls.Load() != 0 {
		t.Fatal("unbounded request reached supplier")
	}
	assertUncharged(t, s)
}

func TestJ4_OverCapRefused(t *testing.T) {
	var calls atomic.Int64
	s := newSpine(t, func([]byte) { calls.Add(1) })
	input := audioFeatureInput(t)
	input["turn"] = map[string]string{"role": "user", "content": strings.Repeat("x", 65537)}
	status, body := postFeature(t, s, "audio-tools.summarize", "input-cap", input)
	if status < 400 {
		t.Fatalf("over input cap accepted: %d %s", status, body)
	}
	if _, err := s.db.Exec(`UPDATE ai_access_keys SET max_credits=1 WHERE id=$1`, s.key.ID); err != nil {
		t.Fatal(err)
	}
	status, body = postFeature(t, s, "audio-tools.summarize", "money-cap", audioFeatureInput(t))
	if status != 403 {
		t.Fatalf("over key cap status=%d %s", status, body)
	}
	if calls.Load() != 0 {
		t.Fatal("over cap reached supplier")
	}
	assertUncharged(t, s)
	t.Run("output-schema-cap", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			event, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": strings.Repeat("x", 8193)}}}})
			fmt.Fprintf(w, "data: %s\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\ndata: [DONE]\n\n", event)
		}))
		defer upstream.Close()
		t.Setenv("VROOLI_JOURNEY_ROUTER_URL", upstream.URL)
		outputSpine := newSpine(t)
		status, body := postFeature(t, outputSpine, "audio-tools.summarize", "output-cap", audioFeatureInput(t))
		if status < 400 || strings.Contains(string(body), strings.Repeat("x", 100)) {
			t.Fatalf("over-cap output leaked: %d %s", status, body)
		}
		var charged int64
		if err := outputSpine.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events`).Scan(&charged); err != nil {
			t.Fatal(err)
		}
		if charged != 200 {
			t.Fatalf("measured supplier usage disappeared on output refusal: %d", charged)
		}
	})
}

func TestJ4_LPBSHoldsNoScenarioPrompt(t *testing.T) {
	input := audioFeatureInput(t)
	instruction := input["instruction"].(string)
	root := "../../../.."
	var registry struct {
		Features []struct {
			Scenario string
			Feature  struct {
				ID                      string
				PinnedInstructionDigest string `json:"pinned_instruction_digest"`
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "packages/monetization-go/ai-features.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), instruction) {
		t.Fatal("prompt copied into hosted registry")
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range registry.Features {
		if entry.Feature.ID == "audio-tools.summarize" {
			found = entry.Scenario == "audio-tools" && entry.Feature.PinnedInstructionDigest == fmt.Sprintf("%x", sha256.Sum256([]byte(instruction)))
		}
	}
	if !found {
		t.Fatal("published Audio instruction digest differs")
	}
	for _, owner := range []string{"landing-page-business-suite", "ai-gateway"} {
		err = filepath.WalkDir(filepath.Join(root, "scenarios", owner, "api"), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(source), instruction) {
				t.Errorf("prompt copied into %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", "./packages/monetization-go/cmd/ai-feature-registry", "--check")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("registry drift: %v %s", err, out)
	}
}

func TestJ4_EveryChatCallerClassified(t *testing.T) {
	root := "../../../.."
	var inventory struct {
		Scanner struct {
			Extensions      []string `json:"extensions"`
			Patterns        []string `json:"patterns"`
			ExcludeSegments []string `json:"exclude_path_segments"`
			ExcludeFiles    []string `json:"exclude_file_patterns"`
		} `json:"scanner"`
		Callers []struct{ Path, Class, Rationale string } `json:"callers"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "scenarios/ai-gateway/config/chat-callers.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	classes := map[string]bool{"bounded_feature": true, "in_app": true, "harness_access": true, "own_account": true, "local_only": true, "infrastructure": true}
	for _, caller := range inventory.Callers {
		if listed[caller.Path] || !classes[caller.Class] || strings.TrimSpace(caller.Rationale) == "" {
			t.Fatalf("invalid classification: %+v", caller)
		}
		listed[caller.Path] = true
		if caller.Class == "bounded_feature" && caller.Path != "scenarios/audio-tools/api/integrations/lpbs/clients/summarize_client.go" {
			source, err := os.ReadFile(filepath.Join(root, caller.Path))
			if err == nil && strings.Contains(string(source), "SubmitMedia") {
				continue
			}
			t.Fatalf("bounded classification needs a published invocation proof: %s", caller.Path)
		}
	}
	contains := func(list []string, value string) bool {
		for _, item := range list {
			if item == value {
				return true
			}
		}
		return false
	}
	for _, required := range []string{"chat.default", "/api/v1/ai/inference", "/api/v1/features/", "chat/completions", "NewInferenceServiceClient", "ExecuteRoute(", "ai-gateway/v1", "ai_gateway.v1", "NewGatewayServiceClient", "api/chat", "api/generate", "resource-ollama", "resource-openrouter"} {
		if !contains(inventory.Scanner.Patterns, required) {
			t.Errorf("missing caller signature %s", required)
		}
	}
	matched := map[string]bool{}
	scenarios, err := os.ReadDir(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range scenarios {
		if !scenario.IsDir() {
			continue
		}
		for _, surface := range []string{"api", "cli"} {
			base := filepath.Join(root, "scenarios", scenario.Name(), surface)
			if _, err := os.Stat(base); os.IsNotExist(err) {
				continue
			}
			err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					if contains(inventory.Scanner.ExcludeSegments, entry.Name()) {
						return filepath.SkipDir
					}
					return nil
				}
				if !contains(inventory.Scanner.Extensions, filepath.Ext(path)) {
					return nil
				}
				for _, pattern := range inventory.Scanner.ExcludeFiles {
					if ok, _ := filepath.Match(pattern, entry.Name()); ok {
						return nil
					}
				}
				source, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				text := strings.ToLower(string(source))
				for _, pattern := range inventory.Scanner.Patterns {
					if strings.Contains(text, strings.ToLower(pattern)) {
						rel, err := filepath.Rel(root, path)
						if err != nil {
							return err
						}
						matched[rel] = true
						if !listed[rel] {
							t.Errorf("unclassified chat/inference source: %s", rel)
						}
						break
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for path := range listed {
		if !matched[path] {
			t.Errorf("classification no longer matches source: %s", path)
		}
	}
	t.Logf("%d production caller/infrastructure sources explicitly classified; labels confer no provider permission", len(matched))
}

func TestJ4_ReceiptWriteFailureRetainsFunding(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_RECEIPT_WRITE_FAILURE", "1")
	var calls atomic.Int64
	s := newSpine(t, func([]byte) { calls.Add(1) })
	status, body := postFeature(t, s, "audio-tools.summarize", "receipt-write-failure", audioFeatureInput(t))
	if status != 502 || calls.Load() != 1 {
		t.Fatalf("failed receipt status=%d calls=%d %s", status, calls.Load(), body)
	}
	assertHeld := func() {
		t.Helper()
		var held, pendingKey int64
		if err := s.db.QueryRow(`SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE status='pending'`).Scan(&held); err != nil {
			t.Fatal(err)
		}
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM ai_key_charges WHERE actual_credits IS NULL`).Scan(&pendingKey); err != nil {
			t.Fatal(err)
		}
		if held != 1000 || pendingKey != 1 {
			t.Fatalf("spent operation released funding: held=%d pending-key=%d", held, pendingKey)
		}
		assertCharged(t, s, 0)
	}
	assertHeld()
	// A fresh real Door performs recovery from Gateway's durable dispatching
	// receipt. Missing terminal cost must retain both holds, never guess zero.
	_, _ = startRecoveryDoor(t, s)
	assertHeld()
	status, body = postFeature(t, s, "audio-tools.summarize", "receipt-write-failure", audioFeatureInput(t))
	if status != 409 || calls.Load() != 1 {
		t.Fatalf("retry redispatched: status=%d calls=%d %s", status, calls.Load(), body)
	}
	assertHeld()
}

func TestJ4_LegacyReceiptRetainsFeatureAttribution(t *testing.T) {
	s := newSpine(t)
	base, crash := startRecoveryDoor(t, s)
	req, _ := http.NewRequest("POST", base+"/api/v1/ai/chat/completions", strings.NewReader(`{"model":"fixture-model","stream":true,"messages":[{"role":"user","content":"crash"}]}`))
	req.Header.Set("Authorization", "Bearer "+s.key.Token)
	req.Header.Set("Idempotency-Key", "legacy-coding-receipt")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("coding response %d", resp.StatusCode)
	}
	operation := resp.Header.Get("X-Operation-ID")
	if operation == "" {
		t.Fatal("missing operation")
	}
	if _, err := resp.Body.Read(make([]byte, 10)); err != nil {
		t.Fatal(err)
	}
	crash()
	resp.Body.Close()
	assertCharged(t, s, 0)
	time.Sleep(2300 * time.Millisecond)
	// Emulate a pre-J4 durable receipt by removing only the newly added field;
	// the accepted plan and real measured cost remain in Gateway's SQLite store.
	legacy, err := http.Post(s.gateway+"/__journey/legacy-receipt/"+operation, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	legacy.Body.Close()
	if legacy.StatusCode != 204 {
		t.Fatalf("legacy fixture status=%d", legacy.StatusCode)
	}
	_, _ = startRecoveryDoor(t, s)
	assertCharged(t, s, 200)
	var feature string
	if err := s.db.QueryRow(`SELECT feature_id FROM business_account_usage_events WHERE operation_id=$1`, operation).Scan(&feature); err != nil {
		t.Fatal(err)
	}
	if feature != "agent.session" {
		t.Fatalf("legacy attribution=%q", feature)
	}
	_, _ = startRecoveryDoor(t, s)
	assertCharged(t, s, 200)
}
