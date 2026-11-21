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
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/commerce"
)

type purchasedBrowser struct {
	*spine
	Account string `json:"account"`
	Device  string `json:"device"`
	Base    string `json:"-"`
	Result  string `json:"-"`
}

func browserPurchase(t *testing.T) *purchasedBrowser {
	t.Helper()
	s := newSpine(t)
	if _, err := s.db.Exec(`DELETE FROM credit_funding_lots`); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", ".", "-run", "^TestJourneyBrowserServer$", "-v", "-count=1", "-timeout", "10m")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "GOWORK=off", "LPBS_SECURE_COOKIES=false", "VROOLI_J15_DB="+s.dbURL, "VROOLI_J15_GATEWAY="+s.gateway)
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var base string
	var runtimeOutput bytes.Buffer
	var drained chan struct{}
	t.Cleanup(func() {
		if base != "" {
			resp, err := http.Post(base+"/__j15/stop", "text/plain", nil)
			if err == nil {
				resp.Body.Close()
			}
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("browser API helper: %v: %s", err, stderr.String())
		}
		if drained != nil {
			<-drained
		}
		if t.Failed() {
			t.Logf("browser API diagnostics: %s\n%s", runtimeOutput.String(), stderr.String())
		}
	})
	scanner := bufio.NewScanner(output)
	var startup strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "BROWSER_URL=") {
			base = strings.TrimPrefix(line, "BROWSER_URL=")
			drained = make(chan struct{})
			go func() { io.Copy(&runtimeOutput, output); close(drained) }()
			break
		}
		startup.WriteString(line + "\n")
	}
	if base == "" {
		t.Fatalf("browser API did not start: %s\n%s", startup.String(), stderr.String())
	}
	result := filepath.Join(t.TempDir(), "browser-result.json")
	evidence := os.Getenv("VROOLI_J15_EVIDENCE")
	if evidence == "" {
		evidence = "/home/matthalloran8/.vrooli/evidence/agent-connect/j15-browser"
	}
	browser := exec.CommandContext(t.Context(), "python3", "../../test/credits_journey.py", base, t.Name(), result, evidence)
	data, err := browser.CombinedOutput()
	t.Log(string(data))
	if err != nil {
		t.Fatalf("real browser purchase: %v", err)
	}
	selected := purchasedBrowser{spine: s, Base: base, Result: result}
	payload, err := os.ReadFile(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &selected); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots WHERE business_account_id=$1 AND livemode=FALSE AND provider='stripe' AND remaining>0 AND provider_ref='pi_j15_mock'`, selected.Account).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("browser payment did not create exactly one account-bound test lot: %d", count)
	}
	return &selected
}

func TestJ15_SignInAndAddCreditsInBrowser(t *testing.T) { browserPurchase(t) }

func buyThenRunAgent(t *testing.T) *purchasedBrowser {
	t.Helper()
	var calls atomic.Int64
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"j15-router\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"J15 real agent completed\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
	}))
	defer router.Close()
	t.Setenv("VROOLI_JOURNEY_ROUTER_URL", router.URL)
	purchased := browserPurchase(t)
	if purchased.Device == "" {
		t.Fatal("browser did not issue a real account device key")
	}
	req, _ := http.NewRequest("POST", purchased.Base+"/api/v1/ai/keys/session", strings.NewReader(`{"harness":"opencode","model":"fixture-model","max_credits":5000,"ttl_seconds":300}`))
	req.Header.Set("Authorization", "Bearer "+purchased.Device)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var session struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(response.Body).Decode(&session); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 || session.Key == "" {
		t.Fatalf("session issuance status=%d", response.StatusCode)
	}
	home := t.TempDir()
	source := cliutil.ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: purchased.Base + "/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	environment := []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "XDG_CONFIG_HOME=" + filepath.Join(home, "config"), "XDG_DATA_HOME=" + filepath.Join(home, "data"), "XDG_STATE_HOME=" + filepath.Join(home, "state"), "VROOLI_CODING_SESSION_KEY=" + session.Key}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	var output bytes.Buffer
	err = cliutil.LaunchCodingAgent(ctx, cliutil.AgentLaunchRequest{Agent: "opencode", WorkingDir: home, Environment: environment, ModelSource: &source, APIBase: "http://127.0.0.1:1", Args: []string{"run", "--pure", "--auto", "--dir", home, "--format", "json", "Say hello."}, Stdout: &output, Stderr: &output, RunChild: func(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
		child := exec.CommandContext(ctx, path, args...)
		child.Dir = home
		child.Env = env
		child.Stdin = stdin
		child.Stdout = stdout
		child.Stderr = stderr
		return child.Run()
	}})
	if err != nil || !strings.Contains(output.String(), "J15 real agent completed") || calls.Load() == 0 {
		t.Fatalf("real OpenCode: %v calls=%d output=%s", err, calls.Load(), strings.ReplaceAll(output.String(), session.Key, "<session-key>"))
	}
	var charged, allocated, granted, remaining int64
	for _, item := range []struct {
		query       string
		destination *int64
	}{
		{`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events WHERE business_account_id=$1`, &charged},
		{`SELECT COALESCE(SUM(a.credits),0) FROM credit_lot_allocations a JOIN credit_funding_lots l ON l.id=a.lot_id WHERE l.business_account_id=$1 AND l.provider_ref='pi_j15_mock'`, &allocated},
		{`SELECT COALESCE(SUM(granted),0) FROM credit_funding_lots WHERE business_account_id=$1 AND provider_ref='pi_j15_mock'`, &granted},
		{`SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots WHERE business_account_id=$1 AND provider_ref='pi_j15_mock'`, &remaining},
	} {
		if err := purchased.db.QueryRow(item.query, purchased.Account).Scan(item.destination); err != nil {
			t.Fatal(err)
		}
	}
	if charged != calls.Load()*200 || allocated != charged || remaining != granted-charged {
		t.Fatalf("browser-purchased funding does not match real OpenCode: calls=%d charged=%d allocated=%d granted=%d remaining=%d", calls.Load(), charged, allocated, granted, remaining)
	}
	t.Logf("Real OpenCode completed through authenticated LPBS/Gateway; %d calls, %d credits charged to the browser-purchased Stripe test lot", calls.Load(), charged)
	return purchased
}

func TestJ15_BuyThenRunAgentWholeJourney(t *testing.T) { buyThenRunAgent(t) }

func verifyWalletBrowser(t *testing.T, p *purchasedBrowser, expected map[string]any) {
	t.Helper()
	expected["account"] = p.Account
	encoded, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Result, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	evidence := "/home/matthalloran8/.vrooli/evidence/agent-connect/j15-browser"
	cmd := exec.CommandContext(t.Context(), "python3", "../../test/credits_journey.py", p.Base, t.Name(), p.Result, evidence, "verify")
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("real wallet browser: %v", err)
	}
}

func purchasedWallet(t *testing.T, p *purchasedBrowser) (*commerce.BusinessAccountUsageService, string, int64) {
	t.Helper()
	key, err := (aidoor.KeyStore{DB: p.db}).Authenticate(t.Context(), "Bearer "+p.Device)
	if err != nil {
		t.Fatal(err)
	}
	var granted int64
	if err := p.db.QueryRow(`SELECT SUM(granted) FROM credit_funding_lots WHERE business_account_id=$1 AND livemode=FALSE`, p.Account).Scan(&granted); err != nil {
		t.Fatal(err)
	}
	live := false
	return commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: p.db, Limits: commerce.NewLimitsService(p.db, "postgres", nil), Dialect: "postgres", LiveMode: &live}), key.Actor, granted
}

func TestJ15_BalanceShownAvailableAndOnHold(t *testing.T) {
	p := browserPurchase(t)
	wallet, actor, granted := purchasedWallet(t, p)
	// An isolated policy proposal creates a real age hold; no production trust
	// values or approvals are written by this fixture.
	if _, err := p.db.Exec(`INSERT INTO funding_source_policies(id,provider,source_kind,young_account_seconds,young_hold_seconds,young_cap_credits,mature_cap_credits) VALUES('j15-isolated','stripe','j15_test_hold',86400,3600,10000,10000)`); err != nil {
		t.Fatal(err)
	}
	funding := commerce.NewCreditWalletService(p.db)
	for _, live := range []bool{false, true} {
		if err := funding.AddCreditsForBusinessAccount(p.Account, "buyer@example.invalid", 1000, "j15_test_hold", "stripe", fmt.Sprintf("evt_j15_hold_%t", live), map[string]any{"livemode": live, "payment_intent_id": fmt.Sprintf("pi_j15_hold_%t", live)}); err != nil {
			t.Fatal(err)
		}
	}
	reservation, err := wallet.ReserveCredits(t.Context(), p.Account, actor, "free", "ai_credits", 500)
	if err != nil {
		t.Fatal(err)
	}
	// These are isolated mode-labelled funding facts, never a real payment or
	// live Stripe request. The live view must not subtract the test hold.
	if err := funding.AddCreditsForBusinessAccount(p.Account, "buyer@example.invalid", 7000, "j15_live_fixture", "stripe", "evt_j15_live_fixture", map[string]any{"livemode": true, "payment_intent_id": "pi_j15_live_fixture"}); err != nil {
		t.Fatal(err)
	}
	liveMode := true
	liveWallet := commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: p.db, Limits: commerce.NewLimitsService(p.db, "postgres", nil), Dialect: "postgres", LiveMode: &liveMode})
	assertView := func(service *commerce.BusinessAccountUsageService, available, held int64) {
		t.Helper()
		view, err := service.WalletView(t.Context(), p.Account)
		if err != nil || view.Available != available || view.OnHold != held {
			t.Fatalf("mode-specific wallet: %+v error=%v want available=%d held=%d", view, err, available, held)
		}
	}
	assertView(wallet, granted-500, 1500)
	assertView(liveWallet, 7000, 1000)
	// Admission uses the same mode-bound hold reader: all 7000 live credits
	// remain reservable while the independent 500-credit test hold is pending.
	liveReservation, err := liveWallet.ReserveCredits(t.Context(), p.Account, actor, "free", "ai_credits", 7000)
	if err != nil {
		t.Fatal(err)
	}
	assertView(wallet, granted-500, 1500)
	assertView(liveWallet, 0, 8000)
	if _, _, err := liveWallet.ReserveCreditsWithID(t.Context(), p.Account, actor, "free", "ai_credits", 500, reservation); err == nil {
		t.Fatal("reservation replay changed its funding mode")
	}
	if err := liveWallet.FinalizeReservationWithCost(t.Context(), p.Account, reservation, 500, "", "fixture-model", 5, 1, 1, "fixture"); err == nil {
		t.Fatal("test reservation settled against live funding")
	}
	if err := liveWallet.ReleaseReservation(t.Context(), p.Account, reservation); err == nil {
		t.Fatal("live service released test reservation")
	}
	if err := liveWallet.RetainUncertainReservation(t.Context(), p.Account, reservation); err == nil {
		t.Fatal("live service changed test reservation lease")
	}
	if err := wallet.ReleaseReservation(t.Context(), p.Account, liveReservation); err == nil {
		t.Fatal("test service released live reservation")
	}
	assertView(wallet, granted-500, 1500)
	assertView(liveWallet, 0, 8000)
	verifyWalletBrowser(t, p, map[string]any{"available": granted - 500, "on_hold": 1500, "reasons": []string{"Safety hold", "Requests in progress"}})
	if err := liveWallet.ReleaseReservation(t.Context(), p.Account, liveReservation); err != nil {
		t.Fatal(err)
	}
	assertView(liveWallet, 7000, 1000)
	if err := wallet.ReleaseReservation(t.Context(), p.Account, reservation); err != nil {
		t.Fatal(err)
	}
	verifyWalletBrowser(t, p, map[string]any{"available": granted, "on_hold": 1000, "reasons": []string{"Safety hold"}})
	unknown, err := wallet.ReserveCredits(t.Context(), p.Account, actor, "free", "ai_credits", 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.db.Exec(`UPDATE business_account_credit_reservations SET livemode=NULL WHERE id=$1`, unknown); err != nil {
		t.Fatal(err)
	}
	for _, service := range []*commerce.BusinessAccountUsageService{wallet, liveWallet} {
		if _, err := service.WalletView(t.Context(), p.Account); err == nil {
			t.Fatal("wallet guessed mode of an untagged active hold")
		}
		if _, err := service.ReserveCredits(t.Context(), p.Account, actor, "free", "ai_credits", 100); err == nil {
			t.Fatal("admission ignored an untagged active hold")
		}
	}
	if _, err := p.db.Exec(`UPDATE business_account_credit_reservations SET expires_at=NOW()-interval '1 second' WHERE id=$1`, unknown); err != nil {
		t.Fatal(err)
	}
	assertView(wallet, granted, 1000)
	assertView(liveWallet, 7000, 1000)
}

func TestJ15_HistoryByFeature(t *testing.T) {
	p := buyThenRunAgent(t)
	wallet, actor, granted := purchasedWallet(t, p)
	reservation, err := wallet.ReserveCredits(t.Context(), p.Account, actor, "free", "ai_credits", 100)
	if err != nil {
		t.Fatal(err)
	}
	// Seed a second feature through the real transactional settlement owner.
	// J4 owns the generic feature HTTP caller; this check owns its history view.
	for range 2 {
		if err := wallet.FinalizeFeatureReservationWithCost(t.Context(), p.Account, reservation, 100, "journey-app", "fixture-model", 1, 2, 1, "fixture-router", "fixture.summary"); err != nil {
			t.Fatal(err)
		}
	}
	if err := wallet.FinalizeFeatureReservationWithCost(t.Context(), p.Account, reservation, 100, "journey-app", "fixture-model", 1, 2, 1, "fixture-router", "changed.feature"); err == nil {
		t.Fatal("settlement replay changed feature attribution")
	}
	var charged, calls int64
	if err := p.db.QueryRow(`SELECT SUM(credits),COUNT(*) FROM business_account_usage_events WHERE business_account_id=$1 AND feature_id='agent.session'`, p.Account).Scan(&charged, &calls); err != nil {
		t.Fatal(err)
	}
	if charged <= 0 || calls < 1 {
		t.Fatal("real agent usage has no server-selected feature")
	}
	verifyWalletBrowser(t, p, map[string]any{"available": granted - charged - 100, "on_hold": 0, "history": map[string]any{"agent.session": map[string]int64{"credits": charged, "calls": calls}, "fixture.summary": map[string]int64{"credits": 100, "calls": 1}}})
}
