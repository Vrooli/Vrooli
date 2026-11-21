package journeytest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	"landing-page-business-suite-api/handlers/serviceorders"
	"landing-page-business-suite-api/internal/commerce"
)

type phoneDeliveryArtifact struct {
	OrderID      string          `json:"order_id"`
	JobID        string          `json:"job_id"`
	OfferID      string          `json:"offer_id"`
	OfferVersion int             `json:"offer_version"`
	JobInput     json.RawMessage `json:"job_input"`
	Report       string          `json:"report_markdown"`
}

type phoneOrderDelivery struct {
	root       string
	report     []byte
	sampleHash string
}

func (d *phoneOrderDelivery) Verify(_ context.Context, order commerce.ServiceOrder) (commerce.ServiceOrderVerification, error) {
	if d == nil || d.root == "" {
		return commerce.ServiceOrderVerification{}, fmt.Errorf("phone test delivery fixture is not configured")
	}
	passed := d != nil && order.ID != "" && order.JobID != "" && order.OfferID == "company-market-scan" && order.OfferVersion == 1 && compactPhoneJSON(order.JobInput, json.RawMessage(`{"sector":"Apple FY2026 public filings"}`)) && sha256Hex(d.report) == d.sampleHash
	artifact := phoneDeliveryArtifact{OrderID: order.ID, JobID: order.JobID, OfferID: order.OfferID, OfferVersion: order.OfferVersion, JobInput: append(json.RawMessage(nil), order.JobInput...), Report: string(d.report)}
	raw, err := json.Marshal(artifact)
	if err != nil {
		return commerce.ServiceOrderVerification{}, err
	}
	if err := os.WriteFile(filepath.Join(d.root, order.JobID+".json"), raw, 0o600); err != nil {
		return commerce.ServiceOrderVerification{}, err
	}
	digest := sha256.Sum256(raw)
	return commerce.ServiceOrderVerification{ReceiptID: "phone-delivery:" + hex.EncodeToString(digest[:]), Passed: passed}, nil
}

func (d *phoneOrderDelivery) ReadResult(_ context.Context, order commerce.ServiceOrder) (serviceorders.DeliveryResult, error) {
	if d == nil || order.State != commerce.ServiceOrderReconciled {
		return serviceorders.DeliveryResult{}, fmt.Errorf("order result is unavailable before reconciliation")
	}
	raw, err := os.ReadFile(filepath.Join(d.root, order.JobID+".json"))
	if err != nil {
		return serviceorders.DeliveryResult{}, fmt.Errorf("read order-bound delivery artifact: %w", err)
	}
	var artifact phoneDeliveryArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return serviceorders.DeliveryResult{}, err
	}
	digest := sha256.Sum256(raw)
	if artifact.OrderID != order.ID || artifact.JobID != order.JobID || artifact.OfferID != order.OfferID || artifact.OfferVersion != order.OfferVersion || !compactPhoneJSON(artifact.JobInput, order.JobInput) || sha256Hex([]byte(artifact.Report)) != d.sampleHash || order.VerificationID != "phone-delivery:"+hex.EncodeToString(digest[:]) {
		return serviceorders.DeliveryResult{}, fmt.Errorf("delivery artifact does not match the verified order and job")
	}
	body := fmt.Sprintf("# Test result for order %s / job %s\n\n> Deterministic test fixture from the prepared public sample; this is not custom research.\n\n%s", order.ID, order.JobID, artifact.Report)
	return serviceorders.DeliveryResult{Format: "markdown", Body: []byte(body)}, nil
}

func compactPhoneJSON(left, right json.RawMessage) bool {
	var a, b bytes.Buffer
	if json.Compact(&a, left) != nil || json.Compact(&b, right) != nil {
		return false
	}
	return bytes.Equal(a.Bytes(), b.Bytes())
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// This browser journey owns a complete local test composition: the maintained
// route handler and canonical Offer Desk source, disposable Postgres, the real
// Money Ledger test service/fixture, and LPBS's shipped UI server around the
// compiled React application. No live payment or scenario production API is used.
func TestServiceOrderReactPhoneCheckoutBrowserJourney(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()

	spine := newSpine(t)
	ledgerConfig := startFundingLedger(t)
	ledgerAuth := &serviceOrderLedgerAuthTransport{base: http.DefaultTransport}
	ledgerWriter, err := commerce.NewMoneyLedgerTestServiceWriter(ledgerConfig["url"], serviceOrderJourneyToken("lpbs-phone-viewport-run-only-money-ledger-write"), &http.Client{Transport: ledgerAuth})
	if err != nil {
		t.Fatal(err)
	}
	offer := loadCompanyMarketScan(t)
	samplePath := filepath.Join("..", "..", "..", "..", "scenarios", "offer-desk", "api", "internal", "serviceoffer", "testdata", "company-market-scan.md")
	sample, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	delivery := &phoneOrderDelivery{root: t.TempDir(), report: sample, sampleHash: offer.Sample.SHA256}
	if sha256Hex(sample) != offer.Sample.SHA256 {
		t.Fatal("Offer Desk sample bytes do not match their recorded SHA-256")
	}
	offerDeskURL := strings.TrimSpace(os.Getenv("LPBS_TEST_OFFER_DESK_URL"))
	var offerDesk *httptest.Server
	if offerDeskURL == "" {
		offerDesk = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"offer": offer})
		}))
		defer offerDesk.Close()
		offerDeskURL = offerDesk.URL
	}
	offers, err := commerce.NewOfferDeskServiceOrderSource(offerDeskURL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := commerce.NewServiceOrderSandbox(commerce.ServiceOrderSandboxOptions{
		DB: spine.db, Provider: commerce.NewLocalTestServiceOrderProvider(),
		Verifier: delivery, Ledger: ledgerWriter,
		LedgerTarget: commerce.ServiceOrderLedgerTarget{
			FixtureID: ledgerConfig["fixture_id"], AdapterID: ledgerConfig["adapter"],
			BookID: ledgerConfig["book"], RevenueAccountID: ledgerConfig["service_order_revenue"],
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := serviceorders.NewHandler(serviceorders.Dependencies{Runtime: runtime, Offers: offers, Results: delivery, TestMode: true})
	if err != nil {
		t.Fatal(err)
	}
	apiRouter := mux.NewRouter()
	handler.RegisterRoutes(apiRouter)
	apiServer := httptest.NewServer(apiRouter)
	defer apiServer.Close()
	apiPort, err := loopbackServerPort(apiServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	uiPort, err := findFreeTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	uiRoot, err := filepath.Abs(filepath.Join("..", "..", "ui"))
	if err != nil {
		t.Fatal(err)
	}
	uiBuild := exec.CommandContext(ctx, "pnpm", "exec", "vite", "build")
	uiBuild.Dir = uiRoot
	if buildOutput, err := uiBuild.CombinedOutput(); err != nil {
		t.Fatalf("build maintained LPBS React UI: %v\n%s", err, buildOutput)
	}
	distRoot := filepath.Join(uiRoot, "dist")
	if info, err := os.Stat(filepath.Join(distRoot, "index.html")); err != nil || info.IsDir() {
		t.Fatalf("LPBS UI build artifact is unavailable; run the supported UI build before this journey: %v", err)
	}
	if _, err := os.Stat(filepath.Join(uiRoot, "node_modules", "@vrooli", "api-base")); err != nil {
		t.Fatalf("LPBS UI runtime dependencies are unavailable: %v", err)
	}
	uiServer, err := startPhoneJourneyUIServer(t, uiRoot, uiPort, apiPort)
	if err != nil {
		t.Fatal(err)
	}
	defer uiServer.stop(t)
	if err := waitForPhoneJourneyUI(ctx, fmt.Sprintf("http://127.0.0.1:%d/services/%s", uiPort, offer.ID)); err != nil {
		t.Fatalf("LPBS UI server did not become ready: %v; server output: %s", err, uiServer.output.String())
	}

	resultPath := strings.TrimSpace(os.Getenv("LPBS_PS2_PHONE_RESULT"))
	if resultPath == "" {
		resultPath = filepath.Join("..", "..", ".vrooli", "test-artifacts", "commerce", "service-checkout-phone-result.json")
		resultPath, err = filepath.Abs(resultPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(resultPath), 0o700); err != nil {
			t.Fatal(err)
		}
	} else {
		if !filepath.IsAbs(resultPath) || filepath.Clean(resultPath) != resultPath {
			t.Fatalf("phone result path must be absolute and clean: %q", resultPath)
		}
		if err := os.MkdirAll(filepath.Dir(resultPath), 0o700); err != nil {
			t.Fatalf("create phone result directory: %v", err)
		}
	}
	screenshotPath := strings.TrimSpace(os.Getenv("LPBS_PS2_PHONE_SCREENSHOT"))
	if screenshotPath == "" {
		screenshotPath = filepath.Join("..", "..", ".vrooli", "test-artifacts", "commerce", "service-checkout-phone-390x844.png")
		screenshotPath, err = filepath.Abs(screenshotPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(screenshotPath), 0o700); err != nil {
			t.Fatal(err)
		}
	} else if err := os.MkdirAll(filepath.Dir(screenshotPath), 0o700); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join("..", "test", "service_checkout_phone_journey.py")
	browserCmd := exec.CommandContext(ctx, "python3", scriptPath,
		fmt.Sprintf("http://127.0.0.1:%d/services/%s", uiPort, offer.ID),
		resultPath, screenshotPath,
	)
	browserCmd.Dir = uiRoot
	pythonHome, xdgHome := os.Getenv("HOME"), t.TempDir()
	browserCmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + pythonHome,
		"XDG_CONFIG_HOME=" + filepath.Join(xdgHome, "config"),
		"XDG_CACHE_HOME=" + filepath.Join(xdgHome, "cache"),
		"XDG_DATA_HOME=" + filepath.Join(xdgHome, "data"),
	}
	browserOutput, err := browserCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real phone viewport checkout failed: %v\n%s\nUI server output: %s", err, browserOutput, uiServer.output.String())
	}
	t.Log(strings.TrimSpace(string(browserOutput)))

	var evidence struct {
		Viewport struct {
			Width    int `json:"width"`
			Document int `json:"document"`
		} `json:"viewport"`
		OfferID         string `json:"offer_id"`
		OfferVersion    int    `json:"offer_version"`
		AmountMinor     int64  `json:"amount_minor"`
		Currency        string `json:"currency"`
		Door            string `json:"door"`
		OrderID         string `json:"order_id"`
		JobID           string `json:"job_id"`
		LedgerReceiptID string `json:"ledger_receipt_id"`
		LedgerPostingID string `json:"ledger_posting_id"`
		DeliveredResult struct {
			Format       string `json:"format"`
			Fixture      bool   `json:"fixture"`
			Path         string `json:"path"`
			SHA256       string `json:"sha256"`
			SHA256Header string `json:"sha256_header"`
			HTTPStatus   int    `json:"http_status"`
			OrderID      string `json:"order_id"`
			JobID        string `json:"job_id"`
		} `json:"delivered_result"`
		Requests []struct {
			Method string `json:"method"`
			URL    string `json:"url"`
			Status int    `json:"status"`
		} `json:"requests"`
	}
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Viewport.Width != 390 || evidence.Viewport.Document > 390 || evidence.OfferID != offer.ID || evidence.OfferVersion != offer.Version || evidence.AmountMinor != offer.ProposedPrice.AmountMinor || evidence.Currency != offer.ProposedPrice.Currency || evidence.Door != commerce.ServiceOrderDoorBrowser || evidence.OrderID == "" || evidence.JobID == "" || evidence.LedgerReceiptID == "" || evidence.LedgerPostingID == "" || evidence.DeliveredResult.Format != "markdown" || !evidence.DeliveredResult.Fixture || evidence.DeliveredResult.HTTPStatus != http.StatusOK || evidence.DeliveredResult.OrderID != evidence.OrderID || evidence.DeliveredResult.JobID != evidence.JobID || evidence.DeliveredResult.SHA256 != evidence.DeliveredResult.SHA256Header {
		t.Fatalf("phone journey result is not bound to the canonical offer and accepted test receipt: %+v", evidence)
	}
	deliveredBytes, err := os.ReadFile(evidence.DeliveredResult.Path)
	if err != nil || sha256Hex(deliveredBytes) != evidence.DeliveredResult.SHA256 || !strings.Contains(string(deliveredBytes), "# Test result for order "+evidence.OrderID+" / job "+evidence.JobID) || !strings.Contains(string(deliveredBytes), "https://www.sec.gov/Archives/edgar/data/320193/000032019326000026/aapl-20260926.htm") {
		t.Fatalf("downloaded Markdown result is not the verified order-bound fixture: err=%v", err)
	}
	var durableOrder, durableJob, durablePosting, durableReceipt string
	if err := spine.db.QueryRowContext(ctx, `SELECT id::text,job_id::text,ledger_posting_id,ledger_receipt_id FROM service_orders WHERE id=$1`, evidence.OrderID).Scan(&durableOrder, &durableJob, &durablePosting, &durableReceipt); err != nil {
		t.Fatal(err)
	}
	if durableJob != evidence.JobID || durablePosting != evidence.LedgerPostingID || durableReceipt != evidence.LedgerReceiptID {
		t.Fatalf("browser evidence does not match durable LPBS order: browser=%+v db=(%s,%s,%s,%s)", evidence, durableOrder, durableJob, durablePosting, durableReceipt)
	}
	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, ledgerConfig["url"]).ListPostings(ctx, connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: ledgerConfig["book"], Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.Postings) != 1 || postings.Msg.Postings[0].Id != evidence.LedgerPostingID || postings.Msg.Postings[0].Event.Category != "test.service_order.sandbox" || postings.Msg.Postings[0].Event.AmountMinor != offer.ProposedPrice.AmountMinor {
		t.Fatalf("phone checkout test Ledger posting is not the accepted $49 service capture: %+v", postings.Msg.Postings)
	}
	ledgerAuth.mu.Lock()
	ledgerTokens := append([]string(nil), ledgerAuth.authorization...)
	ledgerAuth.mu.Unlock()
	if len(ledgerTokens) != 1 || ledgerTokens[0] != "Bearer lpbs-phone-viewport-run-only-money-ledger-write" {
		t.Fatalf("phone journey did not use exactly one run-only Money Ledger writer token: %v", ledgerTokens)
	}
	t.Logf("Phone viewport %dx%d: order=%s job=%s accepted test receipt=%s posting=%s screenshot=%s", evidence.Viewport.Width, 844, evidence.OrderID, evidence.JobID, evidence.LedgerReceiptID, evidence.LedgerPostingID, screenshotPath)

	// Use Browser Automation Studio's managed Playwright driver as an actual
	// browser client for the second checkout door. The page and API remain the
	// same disposable composition used above; this does not use a live account
	// or payment rail.
	basArtifactRoot := strings.TrimSpace(os.Getenv("LPBS_PS2_BROWSING_AGENT_ARTIFACTS"))
	if basArtifactRoot == "" {
		basArtifactRoot = filepath.Join(t.TempDir(), "bas-browsing-agent-checkout")
	} else if !filepath.IsAbs(basArtifactRoot) || filepath.Clean(basArtifactRoot) != basArtifactRoot {
		t.Fatalf("BAS artifact root must be absolute and clean: %q", basArtifactRoot)
	}
	if err := os.MkdirAll(basArtifactRoot, 0o700); err != nil {
		t.Fatalf("create BAS artifact root: %v", err)
	}
	flowPath := filepath.Join(t.TempDir(), "browsing-agent-checkout-flow.json")
	flow := `{
  "metadata":{"name":"PS2 browsing-agent test checkout","version":"1"},
  "settings":{"viewport_width":390,"viewport_height":844},
  "edges":[
    {"id":"e1","source":"wait-offer","target":"enter-request","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e2","source":"enter-request","target":"accept-test-terms","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e3","source":"accept-test-terms","target":"begin-order","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e4","source":"begin-order","target":"wait-order","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e5","source":"wait-order","target":"verify-delivery","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e6","source":"verify-delivery","target":"wait-reconcile","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e7","source":"wait-reconcile","target":"reconcile-ledger","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e8","source":"reconcile-ledger","target":"assert-complete","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"},
    {"id":"e9","source":"assert-complete","target":"capture-result","type":"WORKFLOW_EDGE_TYPE_SMOOTHSTEP"}
  ],
  "nodes":[
    {"id":"wait-offer","action":{"type":"ACTION_TYPE_WAIT","wait":{"selector":".site-service-checkout-summary","state":"WAIT_STATE_VISIBLE","timeout_ms":10000}}},
    {"id":"enter-request","action":{"type":"ACTION_TYPE_INPUT","input":{"selector":"textarea[name=\"sector\"]","value":"Independent test request: local company-market scan"}}},
    {"id":"accept-test-terms","action":{"type":"ACTION_TYPE_CLICK","click":{"selector":"input[type=\"checkbox\"]"}}},
    {"id":"begin-order","action":{"type":"ACTION_TYPE_CLICK","click":{"selector":"button[type=\"submit\"]"}}},
    {"id":"wait-order","action":{"type":"ACTION_TYPE_WAIT","wait":{"selector":"button:has-text(\"Verify delivery and capture test authorization\")","state":"WAIT_STATE_VISIBLE","timeout_ms":15000}}},
    {"id":"verify-delivery","action":{"type":"ACTION_TYPE_CLICK","click":{"selector":"button:has-text(\"Verify delivery and capture test authorization\")"}}},
    {"id":"wait-reconcile","action":{"type":"ACTION_TYPE_WAIT","wait":{"selector":"button:has-text(\"Reconcile accepted Money Ledger receipt\")","state":"WAIT_STATE_VISIBLE","timeout_ms":15000}}},
    {"id":"reconcile-ledger","action":{"type":"ACTION_TYPE_CLICK","click":{"selector":"button:has-text(\"Reconcile accepted Money Ledger receipt\")"}}},
    {"id":"assert-complete","action":{"type":"ACTION_TYPE_ASSERT","assert":{"mode":"ASSERTION_MODE_EXISTS","selector":"h2:has-text(\"Test order complete\")","timeout_ms":15000,"failure_message":"Browsing-agent checkout did not reach its completed test-order state"}}},
    {"id":"capture-result","action":{"type":"ACTION_TYPE_SCREENSHOT","screenshot":{"full_page":true}}}
  ]
}`
	if err := os.WriteFile(flowPath, []byte(flow), 0o600); err != nil {
		t.Fatalf("write BAS browsing-agent flow: %v", err)
	}
	basURL := fmt.Sprintf("http://127.0.0.1:%d/service-checkout?offer_id=%s&offer_version=%d&door=browsing_agent", uiPort, offer.ID, offer.Version)
	basCmd := exec.CommandContext(ctx, "browser-automation-studio", "capture", "--url", basURL,
		"--capture", "screenshot,network", "--dimensions", "mobile",
		"--wait-for", ".site-service-checkout-summary", "--interaction-flow-file", flowPath,
		"--out", basArtifactRoot, "--label", "lpbs-ps2-browsing-agent-test-checkout", "--json")
	basOutput, err := basCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("BAS browsing-agent checkout failed: %v\n%s", err, basOutput)
	}
	if len(strings.TrimSpace(string(basOutput))) == 0 {
		t.Fatal("BAS browsing-agent checkout returned no execution evidence")
	}
	t.Logf("BAS browsing-agent checkout evidence: %s", strings.TrimSpace(string(basOutput)))
	var agentOrderID, agentJobID, agentReceiptID, agentPostingID string
	if err := spine.db.QueryRowContext(ctx, `SELECT id::text,job_id::text,ledger_receipt_id,ledger_posting_id FROM service_orders WHERE door=$1 AND state='reconciled'`, commerce.ServiceOrderDoorBrowserAgent).Scan(&agentOrderID, &agentJobID, &agentReceiptID, &agentPostingID); err != nil {
		t.Fatalf("BAS browser did not reconcile a browsing-agent order: %v", err)
	}
	if agentOrderID == evidence.OrderID || agentJobID == evidence.JobID || agentReceiptID == "" || agentPostingID == "" {
		t.Fatalf("BAS browsing-agent receipt is not a distinct reconciled durable order: order=%s job=%s receipt=%s posting=%s", agentOrderID, agentJobID, agentReceiptID, agentPostingID)
	}
	postingsAfterBAS, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, ledgerConfig["url"]).ListPostings(ctx, connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: ledgerConfig["book"], Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postingsAfterBAS.Msg.Postings) != 2 {
		t.Fatalf("phone plus BAS journey should produce exactly two isolated test postings, got %d", len(postingsAfterBAS.Msg.Postings))
	}
	for _, posting := range postingsAfterBAS.Msg.Postings {
		if posting.Id == agentPostingID && (posting.Event.Category != "test.service_order.sandbox" || posting.Event.AmountMinor != offer.ProposedPrice.AmountMinor) {
			t.Fatalf("BAS browsing-agent posting does not match the canonical $49 test offer: %+v", posting)
		}
	}
	t.Logf("BAS browsing-agent reconciled order=%s job=%s accepted test receipt=%s posting=%s artifacts=%s", agentOrderID, agentJobID, agentReceiptID, agentPostingID, basArtifactRoot)
}

type phoneJourneyUIServer struct {
	cmd    *exec.Cmd
	output *strings.Builder
}

func startPhoneJourneyUIServer(t *testing.T, uiRoot string, uiPort, apiPort int) (*phoneJourneyUIServer, error) {
	t.Helper()
	program := `import { createLandingServer } from './server.js';
const app = createLandingServer({
  uiPort: process.env.UI_PORT, apiPort: process.env.API_PORT, verbose: true,
  allowedOrigins: ['http://127.0.0.1:' + process.env.UI_PORT],
});
const server = app.listen(Number(process.env.UI_PORT), '127.0.0.1');
server.on('error', error => { console.error(error); process.exitCode = 1; });
process.on('SIGTERM', () => server.close(() => process.exit(0)));`
	cmd := exec.Command("node", "--input-type=module", "-e", program)
	cmd.Dir = uiRoot
	tempHome := t.TempDir()
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + tempHome, "NODE_ENV=test",
		"UI_PORT=" + strconv.Itoa(uiPort), "API_PORT=" + strconv.Itoa(apiPort),
	}
	output := &strings.Builder{}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start LPBS-owned UI server entrypoint: %w", err)
	}
	return &phoneJourneyUIServer{cmd: cmd, output: output}, nil
}

func (s *phoneJourneyUIServer) stop(t *testing.T) {
	t.Helper()
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	_ = s.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- s.cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "signal: terminated") {
			t.Errorf("stop LPBS-owned UI server: %v; output: %s", err, s.output.String())
		}
	case <-time.After(5 * time.Second):
		_ = s.cmd.Process.Kill()
		<-done
		t.Errorf("LPBS-owned UI server did not stop after SIGTERM")
	}
}

func waitForPhoneJourneyUI(ctx context.Context, healthURL string) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err == nil {
			response, requestErr := client.Do(request)
			if requestErr == nil {
				response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("timed out waiting for local UI health at %s", healthURL)
		case <-ticker.C:
		}
	}
}

func loopbackServerPort(serverURL string) (int, error) {
	address := strings.TrimPrefix(serverURL, "http://")
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return 0, fmt.Errorf("parse server port from %q: %w", serverURL, err)
	}
	return strconv.Atoi(portText)
}

func findFreeTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
