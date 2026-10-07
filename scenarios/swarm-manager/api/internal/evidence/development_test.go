package evidence

import (
	"context"
	"database/sql"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	runs "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
	"swarm-manager/internal/eventlog"
)

type fixtureDevelopmentProducer struct {
	runsconnect.UnimplementedRunsServiceHandler
	response *runs.GetRunResponse
	calls    int
}

func (p *fixtureDevelopmentProducer) GetRun(ctx context.Context, req *connect.Request[runs.GetRunRequest]) (*connect.Response[runs.GetRunResponse], error) {
	p.calls++
	if req.Msg.Target != p.response.Run.Target || req.Msg.RunId != p.response.Run.RunId {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("fixture target"))
	}
	return connect.NewResponse(proto.Clone(p.response).(*runs.GetRunResponse)), nil
}
func developmentLedgerFixture(t *testing.T) (*DevelopmentEvidenceReader, *fixtureDevelopmentProducer, *database.RoutedDB, *api.DevelopmentReference, DevelopmentReceiptBinding) {
	t.Helper()
	routed, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "evidence.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { routed.Close() })
	if err := eventlog.NewSQLiteRepository(routed).InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := evidenceValueDigest("fixture binding")
	response := &runs.GetRunResponse{TerminalSnapshotSchemaVersion: 1, Run: &runs.RunInfo{RunId: "fixture-run", Target: "fixture-scenario", StartedAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano), CompletedAt: time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano), Status: "passed", SourceStable: true, TreeDigest: "td:" + strings.Repeat("a", 64), SourceScope: "target:scenario:fixture-scenario", ExecutionConfigurationFingerprint: h, DescriptorSnapshotSchemaVersion: 1, DescriptorSnapshotDigest: h, DescriptorSnapshot: &runs.RunDescriptorSnapshot{SchemaVersion: 1, Digest: h}, PhaseSetDigest: h, PlannedPhases: []string{"unit"}, Phases: []*runs.PhaseInfo{{Name: "unit", Status: "passed", Comparable: true, ArtifactBacked: true}}}}
	raw, err := (proto.MarshalOptions{Deterministic: true}).Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	ref := &api.DevelopmentReference{EffortId: "fixture-effort", Revision: 1, AuthorityDigest: h, ContractDigest: h, CommissionSubjectDigest: h}
	binding := DevelopmentReceiptBinding{Reference: ref, CriterionID: "criterion-unit", ReceiptID: "fixture-unit-receipt", Target: response.Run.Target, RunID: response.Run.RunId, ReceiptDigest: evidenceDigest(raw), TreeDigest: response.Run.TreeDigest, SourceScope: response.Run.SourceScope, ConfigurationFingerprint: h, DescriptorSnapshotDigest: h, PhaseSetDigest: h, RequiredPhases: []string{"unit"}}
	producer := &fixtureDevelopmentProducer{response: response}
	path, handler := runsconnect.NewRunsServiceHandler(producer)
	mux := httptest.NewServer(handler)
	t.Cleanup(mux.Close)
	_ = path
	client := runsconnect.NewRunsServiceClient(mux.Client(), mux.URL)
	owner, err := NewDevelopmentEvidenceReader(NewLedger(routed), client, []DevelopmentReceiptBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	return owner, producer, routed, ref, binding
}
func TestDevelopmentEvidenceActualProducerReadImmutableImportAndExplicitFenceHold(t *testing.T) {
	owner, producer, db, ref, _ := developmentLedgerFixture(t)
	for range 2 {
		if err := owner.CaptureProducerReceipts(context.Background(), ref); err != nil {
			t.Fatal(err)
		}
	}
	if producer.calls != 2 {
		t.Fatal("actual read-only generated client not used")
	}
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT count(*) FROM development_producer_receipts").Scan(&count); err != nil || count != 1 {
		t.Fatalf("import not idempotent: %d %v", count, err)
	}
	snapshot, err := owner.ReadDevelopmentEvidence(context.Background(), ref, []string{"criterion-unit"})
	if err != nil || len(snapshot.Criteria) != 1 || len(snapshot.Receipts) != 1 || snapshot.Criteria[0].TestedProductDigest != snapshot.ProductDigest {
		t.Fatalf("snapshot: %+v %v", snapshot, err)
	}
	called := false
	if err := owner.CommitDevelopmentEvidence(context.Background(), ref, snapshot, func(func() error) error { called = true; return nil }); !errors.Is(err, ErrDevelopmentProductFenceUnavailable) || called {
		t.Fatalf("ordinary SourceStable became product fence: %v callback=%t", err, called)
	}
	if _, err := db.ExecContext(context.Background(), "UPDATE development_producer_receipts SET producer='other'"); err == nil {
		t.Fatal("receipt allowed update")
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM development_producer_receipts"); err == nil {
		t.Fatal("receipt allowed deletion")
	}
	replay, err := owner.ReadDevelopmentEvidence(context.Background(), ref, []string{"criterion-unit"})
	if err != nil || !reflect.DeepEqual(snapshot, replay) {
		t.Fatal("immutable receipt projection changed", err)
	}
}
func TestDevelopmentEvidenceProducerRefusalsCreateNoLedgerRows(t *testing.T) {
	cases := map[string]func(*runs.GetRunResponse){
		"failed":                  func(r *runs.GetRunResponse) { r.Run.Status = "failed" },
		"failed-preflight-stable": func(r *runs.GetRunResponse) { r.Run.Status = "failed"; r.Run.Phases = nil },
		"unstable":                func(r *runs.GetRunResponse) { r.Run.SourceStable = false },
		"legacy":                  func(r *runs.GetRunResponse) { r.TerminalSnapshotSchemaVersion = 0; r.Run.TreeDigest = "" },
		"partial-denominator":     func(r *runs.GetRunResponse) { r.Run.PlannedPhases = append(r.Run.PlannedPhases, "typecheck") },
		"skipped":                 func(r *runs.GetRunResponse) { r.Run.Phases[0].Status = "skipped" },
		"non-comparable":          func(r *runs.GetRunResponse) { r.Run.Phases[0].Comparable = false },
		"advisory":                func(r *runs.GetRunResponse) { r.Run.Phases[0].Advisory = true },
		"no-artifact":             func(r *runs.GetRunResponse) { r.Run.Phases[0].ArtifactBacked = false },
		"degraded":                func(r *runs.GetRunResponse) { r.DegradedReasons = []string{"missing terminal provenance"} },
		"wrong-source-scope":      func(r *runs.GetRunResponse) { r.Run.SourceScope = "target:scenario:other" },
		"wrong-configuration":     func(r *runs.GetRunResponse) { r.Run.ExecutionConfigurationFingerprint = "other" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			owner, producer, db, ref, binding := developmentLedgerFixture(t)
			mutate(producer.response)
			// Bind exact mutated producer bytes so these negatives exercise semantic
			// eligibility, not merely the byte-digest mismatch guard.
			raw, _ := (proto.MarshalOptions{Deterministic: true}).Marshal(producer.response)
			binding.ReceiptDigest = evidenceDigest(raw)
			next, err := NewDevelopmentEvidenceReader(owner.ledger, owner.producer, []DevelopmentReceiptBinding{binding})
			if err != nil {
				t.Fatal(err)
			}
			if err := next.CaptureProducerReceipts(context.Background(), ref); err == nil {
				t.Fatal("ineligible producer evidence imported")
			}
			var count int
			if err := db.QueryRowContext(context.Background(), "SELECT count(*) FROM development_producer_receipts").Scan(&count); err != nil || count != 0 {
				t.Fatalf("refusal wrote receipts: %d %v", count, err)
			}
		})
	}
}
func TestDevelopmentEvidenceMissingCriteriaAndRoutedPoolNeverFallBack(t *testing.T) {
	owner, producer, db, ref, _ := developmentLedgerFixture(t)
	if err := owner.CaptureProducerReceipts(database.WithTestMode(context.Background()), ref); err == nil || producer.calls != 0 {
		t.Fatal("missing routed test pool invoked producer or primary")
	}
	if err := owner.CaptureProducerReceipts(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.ReadDevelopmentEvidence(context.Background(), ref, []string{"other"}); err == nil {
		t.Fatal("caller denominator replaced installed criterion")
	}
	if _, err := owner.ReadDevelopmentEvidence(database.WithTestMode(context.Background()), ref, []string{"criterion-unit"}); err == nil {
		t.Fatal("missing test lease read primary receipt")
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return eventlog.NewSQLiteRepository(database.NewFromPrimary(pool)).InitSchema(ctx)
	})
	if err := db.InstallTestPool(context.Background(), filepath.Join(t.TempDir(), "test-evidence.db"), "fixture-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	testContext := database.WithTestMode(context.Background())
	if _, err := owner.ReadDevelopmentEvidence(testContext, ref, []string{"criterion-unit"}); err == nil {
		t.Fatal("empty isolated pool leaked primary evidence")
	}
	if err := owner.CaptureProducerReceipts(testContext, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.ReadDevelopmentEvidence(testContext, ref, []string{"criterion-unit"}); err != nil {
		t.Fatal("isolated receipt missing", err)
	}
	if err := db.ClearTestPool("fixture-lease"); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.ReadDevelopmentEvidence(testContext, ref, []string{"criterion-unit"}); err == nil {
		t.Fatal("ended lease fell back to primary")
	}
}
