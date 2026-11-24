package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	_ "modernc.org/sqlite"
)

type stripeEventPages struct {
	events []string
	calls  []string
}

type durableStripeEventPages struct {
	events    []string
	listCalls []string
	gets      int
	mu        sync.Mutex
}

func (p *durableStripeEventPages) Request(_ context.Context, method, path string, _ io.Reader, _ string) ([]byte, error) {
	if method == http.MethodGet && strings.HasPrefix(path, "/v1/events?") {
		p.mu.Lock()
		p.listCalls = append(p.listCalls, path)
		p.mu.Unlock()
		query, _ := url.ParseQuery(strings.TrimPrefix(path, "/v1/events?"))
		limit := 100
		if query.Get("limit") != "" {
			fmt.Sscanf(query.Get("limit"), "%d", &limit)
		}
		start := 0
		if after := query.Get("starting_after"); after != "" {
			for i, raw := range p.events {
				if strings.Contains(raw, `"id":"`+after+`"`) {
					start = i + 1
					break
				}
			}
		}
		end := start + limit
		if end > len(p.events) {
			end = len(p.events)
		}
		return []byte(fmt.Sprintf(`{"data":[%s],"has_more":%t}`, strings.Join(p.events[start:end], ","), end < len(p.events))), nil
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/v1/events/") {
		id, _ := url.PathUnescape(strings.TrimPrefix(path, "/v1/events/"))
		for _, raw := range p.events {
			if strings.Contains(raw, `"id":"`+id+`"`) {
				p.mu.Lock()
				p.gets++
				p.mu.Unlock()
				return []byte(raw), nil
			}
		}
	}
	return nil, fmt.Errorf("unexpected Stripe request: %s %s", method, path)
}

func (p *stripeEventPages) Request(_ context.Context, method, path string, _ io.Reader, _ string) ([]byte, error) {
	if method != http.MethodGet || !strings.HasPrefix(path, "/v1/events?") {
		return nil, fmt.Errorf("unexpected Stripe request: %s %s", method, path)
	}
	p.calls = append(p.calls, path)
	query, err := url.ParseQuery(strings.TrimPrefix(path, "/v1/events?"))
	if err != nil {
		return nil, err
	}
	limit := 100
	if query.Get("limit") != "" {
		fmt.Sscanf(query.Get("limit"), "%d", &limit)
	}
	start := 0
	if after := query.Get("starting_after"); after != "" {
		for i, raw := range p.events {
			if strings.Contains(raw, `"id":"`+after+`"`) {
				start = i + 1
				break
			}
		}
	}
	end := start + limit
	if end > len(p.events) {
		end = len(p.events)
	}
	page := p.events[start:end]
	return []byte(fmt.Sprintf(`{"data":[%s],"has_more":%t}`, strings.Join(page, ","), end < len(p.events))), nil
}

func TestStripeEventSourceSortsAcrossPagesAndIncludesCursorTimestampTies(t *testing.T) {
	page := func(id string, created int64, mode bool) string {
		return fmt.Sprintf(`{"id":%q,"type":"invoice.paid","created":%d,"livemode":%t,"data":{"object":{"id":"in_%s"}}}`, id, created, mode, id)
	}
	// Provider order is newest-first. The page boundary splits equal-time events.
	requester := &stripeEventPages{events: []string{page("evt_z", 200, false), page("evt_b", 100, false), page("evt_a", 100, false), page("evt_old", 99, false)}}
	source := &StripeEventReconciliationSource{requester: requester, pageSize: 2, maxPages: 10}
	events, err := source.ListAfter(context.Background(), PaymentReconciliationCursor{CreatedAt: time.Unix(100, 0), EventID: "evt_a"}, true, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Cursor.EventID != "evt_b" || events[1].Cursor.EventID != "evt_z" {
		t.Fatalf("events = %+v; want evt_b then evt_z", events)
	}
	if len(requester.calls) != 2 {
		t.Fatalf("provider pages = %d; want 2 to inspect timestamp ties and stop at older event", len(requester.calls))
	}
	if got := events[0].IdempotencyKey(); got != "stripe:test:evt_b" {
		t.Fatalf("idempotency key = %q", got)
	}
}

func TestStripeEventSourceRejectsWrongModeAndUnboundedBackfill(t *testing.T) {
	wrongMode := &stripeEventPages{events: []string{`{"id":"evt_live","type":"invoice.paid","created":100,"livemode":true,"data":{"object":{"id":"in_1"}}}`}}
	source := &StripeEventReconciliationSource{requester: wrongMode, pageSize: 2, maxPages: 1}
	if _, err := source.ListAfter(context.Background(), PaymentReconciliationCursor{}, false, 10); err == nil || !strings.Contains(err.Error(), "mode does not match") {
		t.Fatalf("wrong mode error = %v", err)
	}
	backlog := &stripeEventPages{events: []string{
		`{"id":"evt_3","type":"invoice.paid","created":3,"livemode":false,"data":{"object":{"id":"in_3"}}}`,
		`{"id":"evt_2","type":"invoice.paid","created":2,"livemode":false,"data":{"object":{"id":"in_2"}}}`,
		`{"id":"evt_1","type":"invoice.paid","created":1,"livemode":false,"data":{"object":{"id":"in_1"}}}`,
	}}
	source = &StripeEventReconciliationSource{requester: backlog, pageSize: 1, maxPages: 1}
	if _, err := source.ListAfter(context.Background(), PaymentReconciliationCursor{}, false, 1); err == nil || !strings.Contains(err.Error(), "exceeded 1 provider pages") {
		t.Fatalf("bounded backfill error = %v", err)
	}
}

func TestStripeReconciliationProcessorSharesDurableWebhookReceipt(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(stripeWebhookReceiptsSchema); err != nil {
		t.Fatal(err)
	}
	webhook := &StripeWebhookService{db: db}
	processor := StripeWebhookReconciliationProcessor{Webhook: webhook}
	raw := []byte(`{"id":"evt_reconcile","type":"future.event","created":123,"livemode":false,"data":{"object":{"id":"obj_123"}}}`)
	event, err := decodeStripeReconciliationEvent(raw)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := processor.Apply(context.Background(), event)
	if err != nil || outcome != PaymentReconciliationApplied {
		t.Fatalf("first reconciliation apply = %q, %v", outcome, err)
	}
	outcome, err = processor.Apply(context.Background(), event)
	if err != nil || outcome != PaymentReconciliationReplay {
		t.Fatalf("replayed reconciliation apply = %q, %v", outcome, err)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM stripe_webhook_events WHERE stripe_event_id='evt_reconcile'`).Scan(&status); err != nil || status != "processed" {
		t.Fatalf("shared webhook receipt status = %q, err=%v", status, err)
	}
	changed := event
	changed.Payload = []byte(`{"id":"evt_reconcile","type":"future.event","created":123,"livemode":false,"data":{"object":{"id":"obj_123","metadata":{"changed":true}}}}`)
	if _, err := processor.Apply(context.Background(), changed); err == nil || !strings.Contains(err.Error(), "different payload") {
		t.Fatalf("changed event payload error = %v", err)
	}
}

func TestStripeDurableBackfillResumesScanAndReplaysReferencesAfterRestart(t *testing.T) {
	event := func(id string, created int64) string {
		return fmt.Sprintf(`{"id":%q,"type":"invoice.paid","created":%d,"livemode":false,"data":{"object":{"id":"in_%s"}}}`, id, created, id)
	}
	provider := &durableStripeEventPages{events: []string{event("evt_5", 5), event("evt_4", 4), event("evt_3", 3), event("evt_2", 2), event("evt_1", 1)}}
	dbPath := filepath.Join(t.TempDir(), "backfill.db")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS payment_reconciliation_backfill_state(scope_id TEXT PRIMARY KEY, scan_active BOOLEAN NOT NULL DEFAULT FALSE, scan_complete BOOLEAN NOT NULL DEFAULT FALSE, starting_after TEXT NOT NULL DEFAULT '', updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP); CREATE TABLE IF NOT EXISTS payment_reconciliation_backfill_events(scope_id TEXT NOT NULL, event_id TEXT NOT NULL, created_at TIMESTAMP NOT NULL, event_type TEXT NOT NULL, object_id TEXT NOT NULL, PRIMARY KEY(scope_id,event_id));`); err != nil {
			t.Fatal(err)
		}
		return db
	}
	db := open()
	source, err := NewDurableStripeEventReconciliationSource(provider, db, false)
	if err != nil {
		t.Fatal(err)
	}
	source.pageSize = 2
	cursor := PaymentReconciliationCursor{}
	if events, err := source.ListAfter(context.Background(), cursor, false, 2); err != nil || len(events) != 0 {
		t.Fatalf("first scan page = %d events, %v", len(events), err)
	}
	var state stripeEventBackfillState
	if err := db.QueryRow(`SELECT scan_active,scan_complete,starting_after FROM payment_reconciliation_backfill_state WHERE scope_id=?`, stripeEventBackfillScope).Scan(&state.Active, &state.Complete, &state.StartingAfter); err != nil || !state.Active || state.StartingAfter != "evt_4" {
		t.Fatalf("saved page state = %+v, %v", state, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopening the store must resume from the persisted provider page cursor.
	db = open()
	defer db.Close()
	source, err = NewDurableStripeEventReconciliationSource(provider, db, false)
	if err != nil {
		t.Fatal(err)
	}
	source.pageSize = 2
	if events, err := source.ListAfter(context.Background(), cursor, false, 2); err != nil || len(events) != 0 {
		t.Fatalf("resumed scan page = %d events, %v", len(events), err)
	}
	if !strings.Contains(provider.listCalls[len(provider.listCalls)-1], "starting_after=evt_4") {
		t.Fatalf("resumed provider request = %q", provider.listCalls[len(provider.listCalls)-1])
	}
	if events, err := source.ListAfter(context.Background(), cursor, false, 2); err != nil || len(events) != 0 {
		t.Fatalf("final scan page = %d events, %v", len(events), err)
	}
	if events, err := source.ListAfter(context.Background(), cursor, false, 2); err != nil || len(events) != 2 || events[0].Cursor.EventID != "evt_1" || events[1].Cursor.EventID != "evt_2" {
		t.Fatalf("first replay batch = %+v, %v", events, err)
	}
	// Simulate process loss before the runner advances its durable cursor: the
	// same refs are fetched again, not deleted during the source read.
	if events, err := source.ListAfter(context.Background(), cursor, false, 2); err != nil || len(events) != 2 || events[0].Cursor.EventID != "evt_1" {
		t.Fatalf("restart retry batch = %+v, %v", events, err)
	}
	if _, err := db.Exec(`SELECT COUNT(*) FROM payment_reconciliation_backfill_events`); err != nil {
		t.Fatal(err)
	}
	var staged int
	if err := db.QueryRow(`SELECT COUNT(*) FROM payment_reconciliation_backfill_events`).Scan(&staged); err != nil || staged != 5 {
		t.Fatalf("staged refs = %d, err=%v; want five references, retained across restart", staged, err)
	}
	var columns []string
	rows, err := db.Query(`PRAGMA table_info(payment_reconciliation_backfill_events)`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, strings.ToLower(name))
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(columns, ","), "payload") {
		t.Fatalf("backfill schema stores raw payload: %v", columns)
	}
	if events, err := source.ListAfter(context.Background(), PaymentReconciliationCursor{CreatedAt: time.Unix(2, 0), EventID: "evt_2"}, true, 2); err != nil || len(events) != 2 || events[0].Cursor.EventID != "evt_3" || events[1].Cursor.EventID != "evt_4" {
		t.Fatalf("cursor-advanced batch = %+v, %v", events, err)
	}
}

func TestStripeDurableBackfillPostgresSchemaAndRestart(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("lpbs_backfill_test"), postgres.WithUsername("lpbs"), postgres.WithPassword("test-only"))
	if err != nil {
		t.Fatalf("start isolated PostgreSQL fixture: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	connection, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	open := func() *sql.DB {
		db, err := sql.Open("postgres", connection)
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			if err := db.PingContext(ctx); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("isolated PostgreSQL fixture did not accept connections")
			}
			time.Sleep(100 * time.Millisecond)
		}
		return db
	}
	db := open()
	if _, err := db.Exec(FinancialSchema()); err != nil {
		t.Fatalf("apply declarative backfill schema: %v", err)
	}
	provider := &durableStripeEventPages{events: []string{
		`{"id":"evt_3","type":"invoice.paid","created":3,"livemode":false,"data":{"object":{"id":"in_3"}}}`,
		`{"id":"evt_2","type":"invoice.paid","created":2,"livemode":false,"data":{"object":{"id":"in_2"}}}`,
		`{"id":"evt_1","type":"invoice.paid","created":1,"livemode":false,"data":{"object":{"id":"in_1"}}}`,
	}}
	source, err := NewDurableStripeEventReconciliationSource(provider, db, false)
	if err != nil {
		t.Fatal(err)
	}
	source.pageSize = 1
	if events, err := source.ListAfter(ctx, PaymentReconciliationCursor{}, false, 1); err != nil || len(events) != 0 {
		t.Fatalf("initial page = %d events, %v", len(events), err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = open()
	defer db.Close()
	source, err = NewDurableStripeEventReconciliationSource(provider, db, false)
	if err != nil {
		t.Fatal(err)
	}
	source.pageSize = 1
	if events, err := source.ListAfter(ctx, PaymentReconciliationCursor{}, false, 1); err != nil || len(events) != 0 {
		t.Fatalf("resumed page = %d events, %v", len(events), err)
	}
	if events, err := source.ListAfter(ctx, PaymentReconciliationCursor{}, false, 1); err != nil || len(events) != 0 {
		t.Fatalf("final page = %d events, %v", len(events), err)
	}
	if events, err := source.ListAfter(ctx, PaymentReconciliationCursor{}, false, 1); err != nil || len(events) != 1 || events[0].Cursor.EventID != "evt_1" {
		t.Fatalf("first ordered event = %+v, %v", events, err)
	}
	var active, complete bool
	var startingAfter string
	if err := db.QueryRow(`SELECT scan_active,scan_complete,starting_after FROM payment_reconciliation_backfill_state WHERE scope_id=$1`, stripeEventBackfillScope).Scan(&active, &complete, &startingAfter); err != nil || !complete || active || startingAfter != "evt_1" {
		t.Fatalf("backfill completion state: active=%t complete=%t after=%q err=%v", active, complete, startingAfter, err)
	}
}
