package domains

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func TestCommandGroupsRegisterInstrumentAndIntegrationVerbs(t *testing.T) {
	got := CommandGroups(nil)
	if len(got) != 1 || len(got[0].Commands) != 10 {
		t.Fatalf("unexpected flat command registration: %#v", got)
	}
}

func TestComparePromotionRejectsMismatch(t *testing.T) {
	reading := promotionReading{Value: float64(12), Unit: "count", ObservedAt: "2026-09-15T12:00:00Z", Source: promotionSource{Select: "credits", ContractVersion: "digest.v1"}}
	differences := comparePromotion(reading, []byte(`{"credits":11,"unit":"count","contract_version":"digest.v1","observed_at":"2026-09-15T12:00:00Z"}`))
	if len(differences) == 0 {
		t.Fatal("expected promotion mismatch")
	}
}

func TestComparePromotionAcceptsEquivalentProducerMetadata(t *testing.T) {
	reading := promotionReading{Value: float64(12), Unit: "count", ObservedAt: "2026-09-15T12:00:00Z", Source: promotionSource{Select: "credits", ContractVersion: "digest.v1"}}
	differences := comparePromotion(reading, []byte(`{"credits":12,"unit":"count","contract_version":"digest.v1","observed_at":"2026-09-15T12:00:00Z"}`))
	if len(differences) != 0 {
		t.Fatalf("unexpected promotion differences: %v", differences)
	}
}

func TestComparePromotionAggregatesProducerPanelsAndDerivedMargin(t *testing.T) {
	lines := []byte(`{"observed_at":"2026-09-15T12:00:00Z","contract_version":"revenue-summary.v1","units":{"revenue_by_line":"currency","credit_margin_30d":"currency"},"cost":{"usd":7.5},"revenue_by_line":[{"key":"credit_top_up","amount_minor":2500},{"key":"subscription","amount_minor":1000}]}`)
	if differences := comparePromotion(promotionReading{Value: 35, Unit: "currency", ObservedAt: "2026-09-15T12:00:00Z", Source: promotionSource{Select: "revenue_by_line", ContractVersion: "revenue-summary.v1"}}, lines); len(differences) != 0 {
		t.Fatalf("revenue-line promotion differences = %v", differences)
	}
	if differences := comparePromotion(promotionReading{Value: 17.5, Unit: "currency", ObservedAt: "2026-09-15T12:00:00Z", Source: promotionSource{Select: "credit_margin_30d", ContractVersion: "revenue-summary.v1"}}, lines); len(differences) != 0 {
		t.Fatalf("margin promotion differences = %v", differences)
	}

	digest := []byte(`{"observed_at":"2026-09-15T12:00:00Z","contract_version":"business-digest.v1","credits":{"by_app":[{"credits":10},{"credits":5}]},"growth":{"new_paid_subscriptions":3}}`)
	if differences := comparePromotion(promotionReading{Value: 15, Unit: "count", ObservedAt: "2026-09-15T12:00:00Z", Source: promotionSource{Select: "digest_credit_app", ContractVersion: "business-digest.v1"}}, digest); len(differences) != 0 {
		t.Fatalf("credit panel promotion differences = %v", differences)
	}
	if differences := comparePromotion(promotionReading{Value: 3, Unit: "count", ObservedAt: "2026-09-15T12:00:00Z", Source: promotionSource{Select: "digest_paid_subs", ContractVersion: "business-digest.v1"}}, digest); len(differences) != 0 {
		t.Fatalf("digest scalar promotion differences = %v", differences)
	}
}

func TestReadProducerUsesRevenueSummaryProcedureForRevenueBinding(t *testing.T) {
	const observed = "2026-09-16T01:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/landing_page_business_suite.v1.AdminRevenueService/GetRevenueSummary" {
			t.Fatalf("unexpected producer request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"observedAt":"` + observed + `","mrrMinor":"1250","revenueTodayMinor":"300","revenueWindowMinor":"900","activeSubscriptions":"2","subscriptionsChurnedWindow":"0","creditBalanceTotal":"40","creditBurnedWindow":"10","usageRecordsWindow":"3","costMicros":"7500000","currency":"usd","revenueByLine":[{"key":"credit_top_up","amountMinor":"2500"}]}`))
	}))
	defer server.Close()
	body, err := readProducer(server.URL, "/api/v1/admin/dashboard/revenue", "reader", "30d")
	if err != nil {
		t.Fatalf("read revenue producer: %v", err)
	}
	differences := comparePromotion(promotionReading{Value: 12.5, Unit: "currency", ObservedAt: observed, Source: promotionSource{Select: "revenue_mrr", ContractVersion: "revenue-summary.v1"}}, body)
	if len(differences) != 0 {
		t.Fatalf("normalized revenue differences = %v; body=%s", differences, body)
	}
}

func TestReadPromotionRoomCanReadBroadcastWhenLedgerDoesNotContainMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rooms/broadcast" || r.URL.Query().Get("samples") != "hide" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"room":{"id":"broadcast"},"readings":[{"id":"new_paid_subscriptions_30d","value":2}]}`))
	}))
	defer server.Close()
	app := &cliapp.ScenarioApp{APIClient: cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} },
		nil,
	)}
	body, reading, err := readPromotionRoom(app, "new_paid_subscriptions_30d", "broadcast")
	if err != nil {
		t.Fatalf("read broadcast promotion room: %v", err)
	}
	if len(body) == 0 || reading.ID != "new_paid_subscriptions_30d" || reading.Value != float64(2) {
		t.Fatalf("unexpected broadcast reading: body=%s reading=%#v", body, reading)
	}
}

func TestUnavailablePromotionWritesEvidenceBeforeFailing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMMAND_CENTER_PROMOTION_EVIDENCE_DIR", dir)
	err := writeUnavailablePromotionEvidence("new_paid_subscriptions_30d", "30d", []byte(`{"metrics":[]}`), "https://vrooli.com", os.ErrNotExist)
	if err == nil || !strings.Contains(err.Error(), "promotion unavailable") {
		t.Fatalf("expected unavailable promotion error, got %v", err)
	}
	body, readErr := os.ReadFile(filepath.Join(dir, "new_paid_subscriptions_30d.json"))
	if readErr != nil {
		t.Fatalf("read unavailable evidence: %v", readErr)
	}
	text := string(body)
	if !strings.Contains(text, `"verdict": "unavailable"`) || !strings.Contains(text, "file does not exist") {
		t.Fatalf("evidence did not retain unavailable reason: %s", text)
	}
}

func TestRunPromotionWritesMatchAndRejectsProducerMismatch(t *testing.T) {
	const observed = "2026-09-16T01:00:00Z"
	producerValue := 12.5
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rooms/ledger":
			_, _ = w.Write([]byte(`{"room":{"id":"ledger"},"readings":[{"id":"revenue_mrr","value":12.5,"unit":"currency","observedAt":"` + observed + `","trust":"VALID","source":{"read":"/api/v1/admin/dashboard/revenue","select":"revenue_mrr","expectedUnit":"currency","contractVersion":"revenue-summary.v1"}}]}`))
		case "/landing_page_business_suite.v1.AdminRevenueService/GetRevenueSummary":
			_, _ = w.Write([]byte(`{"observed_at":"` + observed + `","contract_version":"revenue-summary.v1","unit":"currency","revenue":{"mrr":` + formatPromotionNumber(producerValue) + `}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	newApp := func() *cliapp.ScenarioApp {
		return &cliapp.ScenarioApp{APIClient: cliutil.NewAPIClient(
			cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
			func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} },
			nil,
		)}
	}

	t.Run("match", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("COMMAND_CENTER_LPBS_READER_TOKEN", "test-reader")
		t.Setenv("COMMAND_CENTER_LPBS_PRODUCTION_URL", server.URL)
		t.Setenv("COMMAND_CENTER_PROMOTION_EVIDENCE_DIR", dir)
		if err := runPromotion(newApp(), []string{"revenue_mrr", "--window", "30d"}); err != nil {
			t.Fatalf("matching promotion: %v", err)
		}
		body, err := os.ReadFile(filepath.Join(dir, "revenue_mrr.json"))
		if err != nil {
			t.Fatalf("read match evidence: %v", err)
		}
		if !strings.Contains(string(body), `"verdict": "match"`) || !strings.Contains(string(body), `"producer"`) {
			t.Fatalf("match evidence = %s", body)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("COMMAND_CENTER_LPBS_READER_TOKEN", "test-reader")
		t.Setenv("COMMAND_CENTER_LPBS_PRODUCTION_URL", server.URL)
		t.Setenv("COMMAND_CENTER_PROMOTION_EVIDENCE_DIR", dir)
		producerValue = 11.5
		err := runPromotion(newApp(), []string{"revenue_mrr", "--window", "30d"})
		if err == nil || !strings.Contains(err.Error(), "promotion mismatch") {
			t.Fatalf("mismatch error = %v", err)
		}
		body, readErr := os.ReadFile(filepath.Join(dir, "revenue_mrr.json"))
		if readErr != nil {
			t.Fatalf("read mismatch evidence: %v", readErr)
		}
		if !strings.Contains(string(body), `"verdict": "mismatch"`) || !strings.Contains(string(body), "value differs") {
			t.Fatalf("mismatch evidence = %s", body)
		}
	})
}

func TestRunPromotionMatchesRevenuePanelAggregate(t *testing.T) {
	const observed = "2026-09-16T01:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rooms/ledger":
			_, _ = w.Write([]byte(`{"room":{"id":"ledger"},"readings":[{"id":"revenue_by_line","kind":"panel","value":35,"unit":"currency","observedAt":"` + observed + `","trust":"VALID","source":{"read":"/api/v1/admin/dashboard/revenue","select":"revenue_by_line","expectedUnit":"currency","contractVersion":"revenue-summary.v1"}}]}`))
		case "/landing_page_business_suite.v1.AdminRevenueService/GetRevenueSummary":
			_, _ = w.Write([]byte(`{"observed_at":"` + observed + `","contract_version":"revenue-summary.v1","units":{"revenue_by_line":"currency"},"revenue_by_line":[{"key":"credit_top_up","amount_minor":2500},{"key":"subscription","amount_minor":1000}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	t.Setenv("COMMAND_CENTER_LPBS_READER_TOKEN", "test-reader")
	t.Setenv("COMMAND_CENTER_LPBS_PRODUCTION_URL", server.URL)
	t.Setenv("COMMAND_CENTER_PROMOTION_EVIDENCE_DIR", dir)
	app := &cliapp.ScenarioApp{APIClient: cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} },
		nil,
	)}
	if err := runPromotion(app, []string{"revenue_by_line", "--window", "30d"}); err != nil {
		t.Fatalf("panel promotion: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "revenue_by_line.json"))
	if err != nil {
		t.Fatalf("read panel evidence: %v", err)
	}
	if !strings.Contains(string(body), `"verdict": "match"`) {
		t.Fatalf("panel evidence = %s", body)
	}
}

func formatPromotionNumber(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
}

func TestWalkReadIsRegistered(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := SubcommandGroups(nil, manifest); len(got) != 1 {
		t.Fatalf("missing walk group: %#v", got)
	}
}
