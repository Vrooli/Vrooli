package investigation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/investigation"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	_ "modernc.org/sqlite"

	internalpolicy "plan-manager/internal/investigationpolicy"
)

type fakeDeclaredRunner struct {
	mu       sync.Mutex
	calls    int
	inputs   map[string]any
	stdout   string
	outputs  []string
	readback *programsv1.Program
	err      error
}

func (f *fakeDeclaredRunner) RunDeclaredProgram(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	f.inputs = req.Msg.GetInputs().AsMap()
	stdout := f.stdout
	if len(f.outputs) > 0 {
		index := f.calls - 1
		if index >= len(f.outputs) {
			index = len(f.outputs) - 1
		}
		stdout = f.outputs[index]
	}
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{
		Terminal: true,
		Program:  &programsv1.Program{Id: "prog-trigger-1", Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: stdout},
	}), nil
}

func (f *fakeDeclaredRunner) GetProgram(context.Context, *connect.Request[programsv1.GetProgramRequest]) (*connect.Response[programsv1.GetProgramResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readback == nil {
		return connect.NewResponse(&programsv1.GetProgramResponse{}), nil
	}
	return connect.NewResponse(&programsv1.GetProgramResponse{Program: f.readback}), nil
}

func testHandler(t *testing.T, runner declaredProgramRunner) (*ConnectHandler, *internalpolicy.Repository) {
	h, repo, _ := testHandlerAt(t, "file:"+t.Name()+"?mode=memory&cache=shared", runner)
	return h, repo
}

func testHandlerAt(t *testing.T, dsn string, runner declaredProgramRunner) (*ConnectHandler, *internalpolicy.Repository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(internalpolicy.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := internalpolicy.NewRepository(db, schedule.System())
	return NewConnectHandlerWithRunner(repo, nil, runner), repo, db
}

func automaticPolicy() internalpolicy.Policy {
	return internalpolicy.Policy{
		SchemaVersion:   internalpolicy.SchemaVersion,
		Version:         "automatic-test.v1",
		Mode:            internalpolicy.ModeAutomatic,
		Rules:           []internalpolicy.Rule{{Kind: "terminal_mismatch"}},
		Limits:          internalpolicy.Limits{MaxInvestigationsPerExecution: 3, MaxConcurrentPerExecution: 1, MaxConcurrentGlobal: 4, MaxInvestigatorDepth: 1},
		Recommendations: internalpolicy.RecommendationPolicy{AllowedKinds: []string{"observe"}},
	}
}

func automaticObservation() internalpolicy.Observation {
	return internalpolicy.Observation{
		ExecutionID:      "exec-trigger-1",
		PhaseID:          "phase-1",
		PhaseGeneration:  "2",
		RuleVersion:      "automatic-test.v1",
		Now:              time.Date(2026, 9, 6, 13, 0, 0, 0, time.UTC),
		TerminalMismatch: true,
		CallerKey:        "trigger-key-1",
		Question:         "What evidence explains the terminal mismatch?",
		BriefRef:         "plan-brief://exec-trigger-1/phase-1",
		RunIDs:           []string{"run-a", "run-b"},
	}
}

func TestAutomaticTriggerDispatchesOnceAndPersistsProgramIdentity(t *testing.T) {
	runner := &fakeDeclaredRunner{}
	h, repo := testHandler(t, runner)
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(automaticObservation())
	first, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || first.Msg.GetReused() {
		t.Fatalf("first response=%+v calls=%d", first.Msg, runner.calls)
	}
	incident, found, err := repo.GetIncident(context.Background(), first.Msg.GetDecision().GetIncidentFingerprint())
	if err != nil || !found {
		t.Fatalf("incident=%+v found=%v err=%v", incident, found, err)
	}
	if incident.ProgramID != "prog-trigger-1" || incident.State != "dispatched" {
		t.Fatalf("incident=%+v", incident)
	}
	listed, err := h.ListIncidents(context.Background(), connect.NewRequest(&policyv1.ListIncidentsRequest{ExecutionId: "exec-trigger-1", Limit: 10}))
	if err != nil || len(listed.Msg.GetIncidents()) != 1 || listed.Msg.GetIncidents()[0].GetOccurrenceCount() != 1 || len(listed.Msg.GetIncidents()[0].GetOccurrences()) != 1 {
		t.Fatalf("listed incidents=%v err=%v", listed.Msg.GetIncidents(), err)
	}
	loaded, err := h.GetIncident(context.Background(), connect.NewRequest(&policyv1.GetIncidentRequest{IncidentFingerprint: first.Msg.GetDecision().GetIncidentFingerprint()}))
	if err != nil || loaded.Msg.GetOccurrenceCount() != 1 || len(loaded.Msg.GetOccurrences()) != 1 {
		t.Fatalf("loaded incident=%v err=%v", loaded.Msg, err)
	}
	if runner.inputs["caller_key"] != "trigger-key-1" || runner.inputs["execution_id"] != "exec-trigger-1" {
		t.Fatalf("inputs=%v", runner.inputs)
	}

	second, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || !second.Msg.GetReused() {
		t.Fatalf("second response=%+v calls=%d", second.Msg, runner.calls)
	}
}

func TestAutomaticTriggerConcurrentSameObservationDispatchesOnce(t *testing.T) {
	runner := &fakeDeclaredRunner{}
	h, repo := testHandler(t, runner)
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(automaticObservation())
	const callers = 8
	responses := make(chan *connect.Response[policyv1.TriggerRecord], callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
			responses <- response
			errs <- err
		}()
	}
	wg.Wait()
	close(responses)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent record error: %v", err)
		}
	}
	if runner.calls != 1 {
		t.Fatalf("declared program calls=%d, want one stable dispatch", runner.calls)
	}
	decision, _ := policy.Evaluate(automaticObservation())
	incident, found, err := repo.GetIncident(context.Background(), decision.IncidentFingerprint)
	if err != nil || !found {
		t.Fatalf("incident=%+v found=%v err=%v", incident, found, err)
	}
	if incident.OccurrenceCount != 1 {
		t.Fatalf("occurrence count=%d, want one exact-observation occurrence", incident.OccurrenceCount)
	}
}

func TestAutomaticTriggerReconcilesTerminalNestedInvestigation(t *testing.T) {
	runner := &fakeDeclaredRunner{}
	h, repo := testHandler(t, runner)
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	// The outer program may succeed for either a pending or terminal nested
	// result; only the explicit wrapper envelope is allowed to settle the
	// incident.
	runner.stdout = `{"status":"ok","signals":{"execution_id":"workflow-terminal-1","pending":false}}`
	raw, _ := json.Marshal(automaticObservation())
	response, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	incident, found, err := repo.GetIncident(context.Background(), response.Msg.GetDecision().GetIncidentFingerprint())
	if err != nil || !found {
		t.Fatalf("incident=%+v found=%v err=%v", incident, found, err)
	}
	if incident.State != "completed" || incident.InvestigationID != "workflow-terminal-1" {
		t.Fatalf("incident=%+v, want terminal nested identity", incident)
	}
	if err := repo.LinkInvestigation(context.Background(), incident.IncidentFingerprint, "workflow-terminal-1"); err != nil {
		t.Fatal(err)
	}
	incident, _, err = repo.GetIncident(context.Background(), incident.IncidentFingerprint)
	if err != nil || incident.State != "completed" {
		t.Fatalf("manual relink changed terminal incident: %+v err=%v", incident, err)
	}
}

func TestAutomaticTriggerReadsTerminalProgramStdoutOnceBeforeLeavingDispatched(t *testing.T) {
	runner := &fakeDeclaredRunner{readback: &programsv1.Program{
		Id:     "prog-trigger-readback",
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED,
		Stdout: `{"status":"ok","signals":{"execution_id":"workflow-readback-1","pending":false}}`,
	}}
	h, repo := testHandler(t, runner)
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(automaticObservation())
	response, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	incident, found, err := repo.GetIncident(context.Background(), response.Msg.GetDecision().GetIncidentFingerprint())
	if err != nil || !found || incident.State != "completed" || incident.InvestigationID != "workflow-readback-1" {
		t.Fatalf("incident=%+v found=%v err=%v, want completed readback identity", incident, found, err)
	}
}

func TestAutomaticTriggerReconcilesPreviouslyDispatchedProgramWithoutRedispatch(t *testing.T) {
	runner := &fakeDeclaredRunner{readback: &programsv1.Program{
		Id:     "prog-trigger-existing",
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED,
		Stdout: `{"status":"ok","signals":{"nested_signals":{"execution_id":"workflow-existing-1","pending":false}}}`,
	}}
	h, repo := testHandler(t, runner)
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	decision, _ := policy.Evaluate(automaticObservation())
	incident, _, err := repo.RecordIncident(context.Background(), automaticObservation(), decision)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.LinkProgram(context.Background(), incident.IncidentFingerprint, "prog-trigger-existing", "PROGRAM_STATUS_SUCCEEDED"); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(automaticObservation())
	response, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	loaded, found, err := repo.GetIncident(context.Background(), response.Msg.GetDecision().GetIncidentFingerprint())
	if err != nil || !found || loaded.State != "completed" || loaded.InvestigationID != "workflow-existing-1" || runner.calls != 0 {
		t.Fatalf("incident=%+v found=%v calls=%d err=%v, want one readback and no redispatch", loaded, found, runner.calls, err)
	}
}

func TestAutomaticTriggerRetriesUnavailableCompositionWithoutDuplicateLogicalInvestigation(t *testing.T) {
	runner := &fakeDeclaredRunner{outputs: []string{
		`{"status":"unavailable","signals":{"pending":false}}`,
		`{"status":"ok","signals":{"execution_id":"investigation-recovered-1","pending":false}}`,
	}}
	h, repo := testHandler(t, runner)
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(automaticObservation())
	first, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || !strings.Contains(first.Msg.GetIncidentJson(), `"state":"dispatch_failed"`) {
		t.Fatalf("first response=%v calls=%d, want retryable dispatch failure", first.Msg, runner.calls)
	}
	second, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	incident, found, err := repo.GetIncident(context.Background(), second.Msg.GetDecision().GetIncidentFingerprint())
	if err != nil || !found {
		t.Fatalf("incident=%+v found=%v err=%v", incident, found, err)
	}
	if runner.calls != 2 || !second.Msg.GetReused() || incident.State != "completed" || incident.InvestigationID != "investigation-recovered-1" || incident.OccurrenceCount != 1 {
		t.Fatalf("incident=%+v calls=%d reused=%v, want one occurrence and recovered terminal diagnosis", incident, runner.calls, second.Msg.GetReused())
	}
}

func TestAutomaticTriggerRecoversClaimAfterOwnerRestart(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "investigation.db") + "?mode=rwc"
	_, firstRepo, firstDB := testHandlerAt(t, dsn, &fakeDeclaredRunner{})
	policy := automaticPolicy()
	if err := firstRepo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	observation := automaticObservation()
	decision, err := policy.Evaluate(observation)
	if err != nil {
		t.Fatal(err)
	}
	incident, reused, err := firstRepo.RecordIncident(context.Background(), observation, decision)
	if err != nil || reused || incident == nil {
		t.Fatalf("incident=%+v reused=%v err=%v", incident, reused, err)
	}
	claimed, err := firstRepo.ClaimDispatch(context.Background(), incident.IncidentFingerprint, "crashed-owner-claim")
	if err != nil || !claimed {
		t.Fatalf("claim=%v err=%v", claimed, err)
	}
	if _, err := firstDB.Exec(`UPDATE plan_investigation_incidents SET dispatch_started_at=? WHERE incident_fingerprint=?`, time.Now().UTC().Add(-11*time.Minute).Format(time.RFC3339Nano), incident.IncidentFingerprint); err != nil {
		t.Fatal(err)
	}
	if err := firstDB.Close(); err != nil {
		t.Fatal(err)
	}

	secondRunner := &fakeDeclaredRunner{stdout: `{"status":"ok","signals":{"execution_id":"recovered-investigation-1","pending":false}}`}
	secondHandler, secondRepo, secondDB := testHandlerAt(t, dsn, secondRunner)
	defer secondDB.Close()
	raw, _ := json.Marshal(observation)
	response, err := secondHandler.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		t.Fatal(err)
	}
	loaded, found, err := secondRepo.GetIncident(context.Background(), incident.IncidentFingerprint)
	if err != nil || !found || loaded.State != "completed" || loaded.InvestigationID != "recovered-investigation-1" || loaded.OccurrenceCount != 1 || secondRunner.calls != 1 {
		t.Fatalf("loaded=%+v found=%v calls=%d err=%v response=%v, want one recovered terminal dispatch", loaded, found, secondRunner.calls, err, response.Msg)
	}
}

func TestAutomaticTriggerReplaysDuplicatedReorderedFamilyObservationsAcrossRestart(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "reorder.db") + "?mode=rwc"
	firstRunner := &fakeDeclaredRunner{stdout: `{"status":"unavailable","signals":{"pending":false}}`}
	_, firstRepo, firstDB := testHandlerAt(t, dsn, firstRunner)
	policy := automaticPolicy()
	if err := firstRepo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	first := automaticObservation()
	first.ExecutionID = "child-a"
	first.FamilyID = "family-reorder"
	first.SharedFailureRef = "receipt://producer/shared-1"
	first.CallerKey = "reorder-child-a"
	second := first
	second.ExecutionID = "child-b"
	second.CallerKey = "reorder-child-b"
	second.Now = first.Now.Add(time.Minute)

	firstRaw, _ := json.Marshal(first)
	firstResponse, err := firstHandlerRecord(t, firstRunner, firstRepo, firstRaw)
	if err != nil || !strings.Contains(firstResponse.Msg.GetIncidentJson(), `"state":"dispatch_failed"`) {
		t.Fatalf("first response=%v err=%v, want retryable first observation", firstResponse.Msg, err)
	}
	if err := firstDB.Close(); err != nil {
		t.Fatal(err)
	}

	secondRunner := &fakeDeclaredRunner{stdout: `{"status":"ok","signals":{"execution_id":"reordered-investigation-1","pending":false}}`}
	secondHandler, secondRepo, secondDB := testHandlerAt(t, dsn, secondRunner)
	defer secondDB.Close()
	secondRaw, _ := json.Marshal(second)
	secondResponse, err := secondHandler.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(secondRaw)}))
	if err != nil {
		t.Fatal(err)
	}
	// Replay the earlier event after the later event. The exact occurrence
	// key makes this a no-op, while the family relation remains durable.
	_, err = secondHandler.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(firstRaw)}))
	if err != nil {
		t.Fatal(err)
	}
	loaded, found, err := secondRepo.GetIncident(context.Background(), secondResponse.Msg.GetDecision().GetIncidentFingerprint())
	if err != nil || !found || loaded.State != "completed" || loaded.OccurrenceCount != 2 || loaded.InvestigationID != "reordered-investigation-1" || len(loaded.SubjectExecutionIDs) != 2 || secondRunner.calls != 1 {
		t.Fatalf("loaded=%+v found=%v calls=%d err=%v, want two ordered-independent subjects and one dispatch", loaded, found, secondRunner.calls, err)
	}
}

func firstHandlerRecord(t *testing.T, runner declaredProgramRunner, repo *internalpolicy.Repository, raw []byte) (*connect.Response[policyv1.TriggerRecord], error) {
	t.Helper()
	h := NewConnectHandlerWithRunner(repo, nil, runner)
	return h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
}

func TestAutomaticTriggerRejectsMissingDispatchContext(t *testing.T) {
	h, repo := testHandler(t, &fakeDeclaredRunner{})
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	observation := automaticObservation()
	observation.Question = ""
	raw, _ := json.Marshal(observation)
	_, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err=%v code=%s", err, connect.CodeOf(err))
	}
	if _, found, getErr := repo.GetIncident(context.Background(), ""); getErr != nil || found {
		t.Fatalf("missing observation unexpectedly persisted: found=%v err=%v", found, getErr)
	}
}

func TestAutomaticTriggerRecordsDispatchFailureForRetry(t *testing.T) {
	h, repo := testHandler(t, &fakeDeclaredRunner{err: errors.New("program runtime unavailable")})
	policy := automaticPolicy()
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(automaticObservation())
	_, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("err=%v code=%s", err, connect.CodeOf(err))
	}
	decision, _ := policy.Evaluate(automaticObservation())
	incident, found, getErr := repo.GetIncident(context.Background(), decision.IncidentFingerprint)
	if getErr != nil || !found || incident.State != "dispatch_failed" || incident.DispatchError == "" {
		t.Fatalf("incident=%+v found=%v err=%v", incident, found, getErr)
	}
}

func TestTriggerPersistsSuppressedKnownWaitOccurrence(t *testing.T) {
	h, repo := testHandler(t, &fakeDeclaredRunner{})
	policy := internalpolicy.Policy{
		SchemaVersion: internalpolicy.SchemaVersion,
		Version:       "shadow-wait.v1",
		Mode:          internalpolicy.ModeShadow,
		Rules:         []internalpolicy.Rule{{Kind: "no_material_progress", AfterSeconds: 900, IgnoreKnownOwnerWaits: true}},
		Limits:        internalpolicy.Limits{MaxInvestigationsPerExecution: 3, MaxConcurrentPerExecution: 1, MaxConcurrentGlobal: 4, MaxInvestigatorDepth: 1},
	}
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	observation := internalpolicy.Observation{
		ExecutionID:          "exec-wait-1",
		PhaseID:              "phase-1",
		PhaseGeneration:      "1",
		RuleVersion:          policy.Version,
		Now:                  time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC),
		LastMaterialProgress: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC),
		KnownOwnerWait:       true,
	}
	raw, _ := json.Marshal(observation)
	response, err := h.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil || response.Msg.GetOccurrenceId() == "" || response.Msg.GetIncidentJson() != "" {
		t.Fatalf("response=%v err=%v", response.Msg, err)
	}
	if !containsString(response.Msg.GetDecision().GetReasons(), "no_material_progress suppressed: known owner wait") {
		t.Fatalf("decision reasons=%v, want explicit known-wait suppression", response.Msg.GetDecision().GetReasons())
	}
	occurrences, err := h.ListOccurrences(context.Background(), connect.NewRequest(&policyv1.ListOccurrencesRequest{ExecutionId: observation.ExecutionID, Limit: 10}))
	if err != nil || len(occurrences.Msg.GetOccurrences()) != 1 || occurrences.Msg.GetOccurrences()[0].GetDecision().GetEligible() {
		t.Fatalf("occurrences=%v err=%v", occurrences.Msg.GetOccurrences(), err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
