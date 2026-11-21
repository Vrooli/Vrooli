package journeytest

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
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/businessaccount"
	"landing-page-business-suite-api/internal/commerce"
)

type spine struct {
	trace   *requestTrace
	db      *sql.DB
	server  *httptest.Server
	key     aidoor.Key
	gateway string
	dbURL   string
}

func newSpine(t *testing.T, observe ...func([]byte)) *spine {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("journey"), postgres.WithUsername("journey"), postgres.WithPassword("isolated-fixture-password"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{businessaccount.Schema(), commerce.FinancialSchema(), commerce.OperationsSchema(), aidoor.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	for _, query := range []string{
		`INSERT INTO business_accounts(id,display_name,billing_email) VALUES('account','Journey','actor@example.invalid')`,
		`INSERT INTO business_account_members(business_account_id,user_id,role) VALUES('account','actor','owner')`,
		`INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value) VALUES('free','cost_based','ai_credits',10000)`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES(gen_random_uuid(),'account','paid','stripe','fixture-payment',FALSE,'settled',10000,10000,'open',$1)`, now)
	if err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		payload, _ := io.ReadAll(r.Body)
		for _, f := range observe {
			f(payload)
		}
		var envelope struct {
			MaxTokens int `json:"max_tokens"`
			N         int `json:"n"`
			Provider  struct {
				DataCollection string   `json:"data_collection"`
				Only           []string `json:"only"`
				AllowFallbacks bool     `json:"allow_fallbacks"`
			} `json:"provider"`
		}
		if json.Unmarshal(payload, &envelope) != nil || envelope.MaxTokens <= 0 || envelope.MaxTokens > 65536 || envelope.N != 1 || envelope.Provider.DataCollection != "deny" || len(envelope.Provider.Only) != 1 || envelope.Provider.Only[0] != "fixture-host" || envelope.Provider.AllowFallbacks {
			t.Errorf("supplier received an unbounded request or lost downstream constraints: %s", payload)
			w.WriteHeader(400)
			return
		}
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(payload, &request)
		if len(request.Messages) > 0 && request.Messages[0].Content == "fail" {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"fixture\",\"choices\":[{\"delta\":{\"content\":\"Hello from router\"}}]}\n\n")
		w.(http.Flusher).Flush()
		if len(request.Messages) > 0 && request.Messages[0].Content == "crash" {
			time.Sleep(2 * time.Second)
		}
		if len(request.Messages) > 0 && request.Messages[0].Content == "slow" {
			time.Sleep(125 * time.Second)
		}
		if len(request.Messages) > 0 && request.Messages[0].Content == "ten-minute" {
			time.Sleep(605 * time.Second)
		}
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(upstream.Close)
	routerURL := upstream.URL
	if override := os.Getenv("VROOLI_JOURNEY_ROUTER_URL"); override != "" {
		routerURL = override
	}
	gateway := startGateway(t, routerURL)
	live := false
	wallet := commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: db, Limits: commerce.NewLimitsService(db, "postgres", nil), Dialect: "postgres", LiveMode: &live})
	keys := aidoor.KeyStore{DB: db}
	key, err := keys.IssueDevice(ctx, "account", "actor", "journey", 10000)
	if err != nil {
		t.Fatal(err)
	}
	trace := newRequestTrace(t)
	server := httptest.NewUnstartedServer(&aidoor.Door{Keys: keys, Wallet: wallet, GatewayURL: gateway, ObserveTransition: trace.observe})
	server.Config.WriteTimeout = 30 * time.Second
	server.Start()
	t.Cleanup(server.Close)
	return &spine{trace: trace, db: db, server: server, key: key, gateway: gateway, dbURL: url}
}

func startGateway(t *testing.T, upstream string) string {
	t.Helper()
	dir, err := filepath.Abs("../../../ai-gateway/api")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./handlers/routing", "-run", "^TestJourneyGatewayServer$", "-count=1", "-v", "-timeout", "20m")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_UPSTREAM="+upstream, "VROOLI_JOURNEY_GATEWAY_DB="+filepath.Join(t.TempDir(), "gateway.db"))
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = input.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("Gateway process: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "GATEWAY_URL=") {
			go io.Copy(io.Discard, output)
			url := strings.TrimPrefix(line, "GATEWAY_URL=")
			t.Cleanup(func() {
				resp, err := http.Post(url+"/__journey/stop", "text/plain", nil)
				if err == nil {
					resp.Body.Close()
				}
			})
			return url
		}
	}
	t.Fatalf("Gateway failed to start: %s", stderr.String())
	return ""
}

func TestJ2_RequestThroughRealLPBSAndRealGatewayStreams(t *testing.T) {
	s := newSpine(t)
	req, err := http.NewRequest("POST", s.server.URL+"/api/v1/ai/chat/completions", strings.NewReader(`{"model":"fixture-model","messages":[{"role":"user","content":"Hello"}],"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+s.key.Token)
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || !strings.Contains(string(body), "Hello from router") || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("stream status=%d body=%s", response.StatusCode, body)
	}
	var charged, allocated, remaining int64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events`).Scan(&charged); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations`).Scan(&allocated); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT SUM(remaining) FROM credit_funding_lots`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	s.trace.require(t, "accepted", "plan", "reserve", "dispatch", "receive", "commit")
	if charged != 200 || allocated != 200 || remaining != 9800 {
		t.Fatalf("ledger charged=%d allocated=%d remaining=%d; want 200,200,9800", charged, allocated, remaining)
	}
}
