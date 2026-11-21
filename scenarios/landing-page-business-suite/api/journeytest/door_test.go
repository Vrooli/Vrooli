package journeytest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/aidoor"
)

func (s *spine) request(t *testing.T, token, operation, content string) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"model": "fixture-model", "stream": true, "messages": []map[string]string{{"role": "user", "content": content}}, "outside_funded": true, "business_account_id": "attacker-account"})
	req, err := http.NewRequest("POST", s.server.URL+"/api/v1/ai/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", operation)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}
func assertCharged(t *testing.T, s *spine, want int64) {
	t.Helper()
	var got int64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("charged=%d, want %d", got, want)
	}
}
func TestJ2_StreamOutlastsTwoMinutes(t *testing.T) {
	s := newSpine(t)
	start := time.Now()
	resp := s.request(t, s.key.Token, "slow", "slow")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || time.Since(start) < 2*time.Minute || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("stream ended early: status %d elapsed %s body %s", resp.StatusCode, time.Since(start), body)
	}
	assertCharged(t, s, 200)
}

func TestJ2_StreamOutlastsTenMinutes(t *testing.T) {
	s := newSpine(t)
	start := time.Now()
	resp := s.request(t, s.key.Token, "ten-minute", "ten-minute")
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != 200 || time.Since(start) <= 10*time.Minute || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("long stream: status=%d elapsed=%s err=%v body=%s", resp.StatusCode, time.Since(start), err, body)
	}
	assertCharged(t, s, 200)
	var events int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_usage_events`).Scan(&events); err != nil || events != 1 {
		t.Fatalf("settled %d times: %v", events, err)
	}
}

func TestJ2_QuotedEnvelopeRejectsUnboundedAndRoutingOverrides(t *testing.T) {
	s := newSpine(t)
	for i, extra := range []string{`"max_tokens":99999999`, `"max_completion_tokens":99999999`, `"n":2`, `"provider":{"data_collection":"allow","only":["attacker"]}`, `"reasoning":{"effort":"high"}`, `"tools":[{"type":"web_search"}]`} {
		body := `{"model":"fixture-model","messages":[{"role":"user","content":"hello"}],` + extra + `}`
		req, _ := http.NewRequest("POST", s.server.URL+"/api/v1/ai/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+s.key.Token)
		req.Header.Set("Idempotency-Key", fmt.Sprint("hostile-", i))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode == 200 {
			t.Fatalf("unquoted dimension accepted: %s", extra)
		}
	}
	assertCharged(t, s, 0)
	// Omitted limits are bounded by the supplier fixture's payload assertions.
	resp := s.request(t, s.key.Token, "bounded-default", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("default quote status %d", resp.StatusCode)
	}
	assertCharged(t, s, 200)
}

func TestJ2_ReleasedKeyClaimCannotRedispatchAfterTopup(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(`UPDATE ai_access_keys SET max_credits=1000;UPDATE credit_funding_lots SET remaining=0`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "released", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 402 {
		t.Fatalf("empty status %d", resp.StatusCode)
	}
	if _, err := s.db.Exec(`UPDATE credit_funding_lots SET remaining=10000`); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for _, operation := range []string{"released", "fresh"} {
		group.Add(1)
		go func(operation string) {
			defer group.Done()
			resp := s.request(t, s.key.Token, operation, "crash")
			io.Copy(io.Discard, resp.Body)
			want := 200
			if operation == "released" {
				want = 409
			}
			if resp.StatusCode != want {
				t.Errorf("%s status=%d want=%d", operation, resp.StatusCode, want)
			}
		}(operation)
	}
	group.Wait()
	assertCharged(t, s, 200)
}

func TestJ1_AccountTierAndExactAppCapsGovernDoor(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(`INSERT INTO subscriptions(subscription_id,business_account_id,status,plan_tier) VALUES('paid','account','active','pro');INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value,cost_multiplier) VALUES('pro','cost_based','ai_credits',10000,2000000);INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value,cost_multiplier,app_bundle_key) VALUES('pro','app_specific','ai_credits',1000,2000000,'journey')`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "paid-first", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("paid status %d", resp.StatusCode)
	}
	assertCharged(t, s, 400)
	resp = s.request(t, s.key.Token, "paid-second", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 402 {
		t.Fatalf("exact app cap bypassed: %d", resp.StatusCode)
	}
	assertCharged(t, s, 400)
}

func TestJ1_UnverifiedGlobalKeyCannotEvadeAppCaps(t *testing.T) {
	s := newSpine(t)
	key, err := (aidoor.KeyStore{DB: s.db}).IssueDevice(context.Background(), "account", "actor", "", 10000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value,cost_multiplier,app_bundle_key) VALUES('free','app_specific','ai_credits',1000,1000000,'protected-app')`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, key.Token, "unverified-app", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 503 {
		t.Fatalf("unverified app evaded configured cap: %d", resp.StatusCode)
	}
	assertCharged(t, s, 0)
}

func TestJ1_ExactAppCapIncludesConcurrentHolds(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(`INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value,cost_multiplier,app_bundle_key) VALUES('free','app_specific','ai_credits',1000,1000000,'journey')`); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	var successful atomic.Int32
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			resp := s.request(t, s.key.Token, fmt.Sprint("app-parallel-", i), "crash")
			io.Copy(io.Discard, resp.Body)
			if resp.StatusCode == 200 {
				successful.Add(1)
			} else if resp.StatusCode != 402 {
				t.Errorf("app cap status %d", resp.StatusCode)
			}
		}(i)
	}
	group.Wait()
	if successful.Load() != 1 {
		t.Fatalf("app allowed %d concurrent holds; want1", successful.Load())
	}
	assertCharged(t, s, 200)
}

func TestJ2_ConnectionRefusedAfterPlanReleasesUndispatchedHolds(t *testing.T) {
	s := newSpine(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	unreachable := "http://" + listener.Addr().String()
	listener.Close()
	door := s.server.Config.Handler.(*aidoor.Door)
	var lookups atomic.Int32
	door.ResolveGateway = func(context.Context) (string, error) {
		if lookups.Add(1) == 1 {
			return s.gateway, nil
		}
		return unreachable, nil
	}
	resp := s.request(t, s.key.Token, "undispatched", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 502 {
		t.Fatalf("dial status %d", resp.StatusCode)
	}
	var pending, keyHeld int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE status='pending'`).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ai_key_charges WHERE actual_credits IS NULL`).Scan(&keyHeld); err != nil {
		t.Fatal(err)
	}
	if pending != 0 || keyHeld != 0 {
		t.Fatalf("undispatched holds: wallet=%d key=%d", pending, keyHeld)
	}
	assertCharged(t, s, 0)
	door.ResolveGateway = nil
	retry := s.request(t, s.key.Token, "undispatched", "hello")
	io.Copy(io.Discard, retry.Body)
	if retry.StatusCode != 409 {
		t.Fatalf("terminal retry status %d", retry.StatusCode)
	}
}

func TestJ3_CanonicalHarnessRecordControlsPickerAndExecution(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_HARNESS_DENIED", "opencode")
	s := newSpine(t)
	key, err := (aidoor.KeyStore{DB: s.db}).IssueSession(context.Background(), s.key, "opencode", "fixture-model", 5000, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("GET", s.server.URL+"/api/v1/ai/models", nil)
	req.Header.Set("Authorization", "Bearer "+key.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var models struct {
		Data []any `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&models) != nil || len(models.Data) != 0 {
		t.Fatalf("pending runner record offered models: %+v", models)
	}
	execute := s.request(t, key.Token, "denied-harness", "hello")
	io.Copy(io.Discard, execute.Body)
	if execute.StatusCode != 403 {
		t.Fatalf("pending runner record executed: %d", execute.StatusCode)
	}
	assertCharged(t, s, 0)
}

func TestJ3_ModelLicenceRestrictionSurvivesProductionLoader(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_DENY_MODEL_LICENCE", "1")
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "licence-denied", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 403 {
		t.Fatalf("commercially prohibited model executed: %d", resp.StatusCode)
	}
	assertCharged(t, s, 0)
}
func TestJ2_ReceiptSettlesLedgerExactly(t *testing.T) {
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "receipt", "hello")
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		t.Fatal(err)
	}
	operation := resp.Header.Get("X-Operation-ID")
	if operation == "" {
		t.Fatal("missing receipt operation")
	}
	receipt, err := http.Get(s.gateway + "/api/v1/private/provider-routes/receipt/" + operation)
	if err != nil {
		t.Fatal(err)
	}
	defer receipt.Body.Close()
	var data struct {
		Credits    int64  `json:"credits"`
		CostMicros int64  `json:"cost_micros"`
		State      string `json:"state"`
	}
	if json.NewDecoder(receipt.Body).Decode(&data) != nil || data.Credits != 200 || data.CostMicros != 2 || data.State != "completed" {
		t.Fatalf("receipt %+v", data)
	}
	assertCharged(t, s, data.Credits)
}
func TestJ2_EmptyWalletGets402(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(`UPDATE credit_funding_lots SET remaining=0`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "empty", "hello")
	if resp.StatusCode != 402 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	assertCharged(t, s, 0)
}
func TestJ2_OnlyIssuedKeysAccepted(t *testing.T) {
	s := newSpine(t)
	for _, token := range []string{"", "unissued", s.key.Token + "x"} {
		resp := s.request(t, token, "bad", "hello")
		if resp.StatusCode != 401 {
			t.Fatalf("unissued key accepted: %d", resp.StatusCode)
		}
	}
	if err := (aidoor.KeyStore{DB: s.db}).Revoke(context.Background(), s.key, s.key.ID); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "revoked", "hello")
	if resp.StatusCode != 401 {
		t.Fatal("revoked key accepted")
	}
	assertCharged(t, s, 0)
}
func TestJ2_OutsideFundedMarkerIsStripped(t *testing.T) {
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "marker", "hello")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("response %d %s", resp.StatusCode, body)
	}
	assertCharged(t, s, 200)
	var account string
	if err := s.db.QueryRow(`SELECT business_account_id FROM business_account_usage_events`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if account != "account" {
		t.Fatalf("caller selected billing account %q", account)
	}
}
func TestJ2_UpstreamFailureRetryChargesOnce(t *testing.T) {
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "failure", "fail")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 502 {
		t.Fatalf("failure status %d", resp.StatusCode)
	}
	assertCharged(t, s, 0)
	retry := s.request(t, s.key.Token, "failure", "fail")
	io.Copy(io.Discard, retry.Body)
	if retry.StatusCode != 409 {
		t.Fatalf("retry status %d", retry.StatusCode)
	}
	assertCharged(t, s, 0)
	success := s.request(t, s.key.Token, "new-operation", "hello")
	io.Copy(io.Discard, success.Body)
	again := s.request(t, s.key.Token, "new-operation", "hello")
	io.Copy(io.Discard, again.Body)
	assertCharged(t, s, 200)
}
func TestJ2_SecondPublicDoorDoesNotExist(t *testing.T) {
	s := newSpine(t)
	for _, path := range []string{"/api/v1/ai/chat/completions", "/api/v1/ai/responses", "/v1/chat/completions"} {
		resp, err := http.Post(s.gateway+path, "application/json", strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Fatalf("Gateway exposes public door %s: %d", path, resp.StatusCode)
		}
	}
}
func TestJ3_ZeroApprovedRecordsDenyEverything(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_RULE_STATUS", "proposed")
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "denied", "hello")
	if resp.StatusCode != 403 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unapproved route status=%d body=%s", resp.StatusCode, body)
	}
	assertCharged(t, s, 0)
}
func TestJ3_ApprovedFixtureRecordAllows(t *testing.T) {
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "approved", "hello")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("approved fixture status %d: %s", resp.StatusCode, body)
	}
	assertCharged(t, s, 200)
}
func TestJ3_PickerAndExecuteAgree(t *testing.T) {
	s := newSpine(t)
	request, _ := http.NewRequest("GET", s.server.URL+"/api/v1/ai/models", nil)
	request.Header.Set("Authorization", "Bearer "+s.key.Token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&models) != nil || len(models.Data) != 1 || models.Data[0].ID != "fixture-model" {
		t.Fatalf("models %+v", models)
	}
	execute := s.request(t, s.key.Token, "picker", "hello")
	io.Copy(io.Discard, execute.Body)
	if execute.StatusCode != 200 {
		t.Fatal("picker route cannot execute")
	}
	assertCharged(t, s, 200)
}
func TestJ3_ClaudeCodeAbsentFromCreditsByRecord(t *testing.T) {
	s := newSpine(t)
	for _, harness := range []string{"opencode", "codex", "claude-code"} {
		req, _ := http.NewRequest("GET", s.server.URL+"/api/v1/ai/models?harness="+harness, nil)
		req.Header.Set("Authorization", "Bearer "+s.key.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var data struct {
			Data []any `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if harness == "claude-code" && len(data.Data) != 0 {
			t.Fatal("Claude Code offered credits")
		}
		if harness != "claude-code" && len(data.Data) != 1 {
			t.Fatalf("supported harness %s has no route", harness)
		}
	}
}

func TestJ2_CancelChargesOnce(t *testing.T) {
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "cancel", "hello")
	chunk := make([]byte, 8)
	if _, err := resp.Body.Read(chunk); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var count int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_usage_events`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cancel lost measured usage")
		}
		time.Sleep(20 * time.Millisecond)
	}
	again := s.request(t, s.key.Token, "cancel", "hello")
	io.Copy(io.Discard, again.Body)
	if again.StatusCode != 409 {
		t.Fatalf("canceled operation redispatched: %d", again.StatusCode)
	}
	assertCharged(t, s, 200)
}

func TestJ3_NewSupplierAddedByDataAlone(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_SUPPLIER", "new-data-only-supplier")
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "new-supplier", "hello")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("new data-only supplier status %d: %s", resp.StatusCode, body)
	}
	var supplier string
	if err := s.db.QueryRow(`SELECT provider FROM business_account_usage_events`).Scan(&supplier); err != nil {
		t.Fatal(err)
	}
	if supplier != "new-data-only-supplier" {
		t.Fatalf("wrong supplier %s", supplier)
	}
}
func TestJ3_AgentCannotReachApproved(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_DECISION_ACTOR", "agent")
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "agent-approved", "hello")
	if resp.StatusCode != 403 {
		t.Fatalf("agent approval authorized spend: %d", resp.StatusCode)
	}
	assertCharged(t, s, 0)
}
func TestJ3_ChangedClauseReopensRecord(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_CHANGED_CLAUSE", "1")
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "changed-clause", "hello")
	if resp.StatusCode != 403 {
		t.Fatalf("changed clause retained old approval: %d", resp.StatusCode)
	}
	assertCharged(t, s, 0)
}

// A configuration change during dispatch must not reprice an admitted request.
func TestDoorPinsCommerceConversionUntilSettlement(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(`UPDATE subscription_tier_limits SET cost_multiplier=500000 WHERE tier_id='free' AND limit_key='ai_credits'`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "pinned-conversion", "crash")
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	// Headers arrive with the first upstream delta, before its delayed usage.
	if _, err := s.db.Exec(`UPDATE subscription_tier_limits SET cost_multiplier=2000000 WHERE tier_id='free' AND limit_key='ai_credits'`); err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("stream: %s, %v", body, err)
	}
	assertCharged(t, s, 100)
}
