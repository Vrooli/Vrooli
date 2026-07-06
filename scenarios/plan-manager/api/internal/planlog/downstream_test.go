package planlog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	planmodel "plan-manager/internal/planmodel"
)

type staticURLResolver map[string]string

func (r staticURLResolver) ResolveScenarioURLDefault(_ context.Context, scenario string) (string, error) {
	if u := r[scenario]; u != "" {
		return u, nil
	}
	return "", errors.New("missing scenario URL")
}

type scenarioQAFixture struct {
	t                 *testing.T
	existingID        string
	expectedTopic     string
	posted            knowledgeAddRequest
	postedAttribution map[string]any
	gets              int
	posts             int
}

func newScenarioQAServer(t *testing.T, existingID, expectedTopicPrefix string) (*httptest.Server, *scenarioQAFixture) {
	t.Helper()
	f := &scenarioQAFixture{t: t, existingID: existingID, expectedTopic: expectedTopicPrefix}
	return httptest.NewServer(http.HandlerFunc(f.serve)), f
}

func (f *scenarioQAFixture) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/teams/scenario-qa/knowledge" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		f.handleLookup(w, r)
		return
	}
	if r.Method == http.MethodPost {
		f.handleCreate(w, r)
		return
	}
	http.NotFound(w, r)
}

func (f *scenarioQAFixture) handleLookup(w http.ResponseWriter, r *http.Request) {
	f.gets++
	topic := r.URL.Query().Get("topic")
	if f.expectedTopic != "" && !strings.HasPrefix(topic, f.expectedTopic) {
		f.t.Fatalf("lookup topic = %q", topic)
	}
	entries := []knowledgeEntryResponse{}
	if f.existingID != "" {
		entries = append(entries, knowledgeEntryResponse{ID: f.existingID, Topic: topic})
	}
	_ = json.NewEncoder(w).Encode(knowledgeListResponse{Entries: entries})
}

func (f *scenarioQAFixture) handleCreate(w http.ResponseWriter, r *http.Request) {
	f.posts++
	rawHeader := r.Header.Get("X-Vrooli-Attribution")
	decoded, err := base64.StdEncoding.DecodeString(rawHeader)
	if err != nil {
		f.t.Fatalf("decode attribution header: %v", err)
	}
	if err := json.Unmarshal(decoded, &f.postedAttribution); err != nil {
		f.t.Fatalf("unmarshal attribution header: %v", err)
	}
	if err := json.NewDecoder(r.Body).Decode(&f.posted); err != nil {
		f.t.Fatalf("decode post: %v", err)
	}
	_ = json.NewEncoder(w).Encode(knowledgeEntryResponse{ID: "knw-123", Topic: f.posted.Topic})
}

type swarmFixture struct {
	t         *testing.T
	existing  recordResponse
	posted    recordCreateRequest
	gets      int
	posts     int
	createID  string
	lookupHit bool
}

func newSwarmServer(t *testing.T, existing recordResponse) (*httptest.Server, *swarmFixture) {
	t.Helper()
	f := &swarmFixture{t: t, existing: existing, createID: "rec-123"}
	return httptest.NewServer(http.HandlerFunc(f.serve)), f
}

func (f *swarmFixture) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != swarmRecordsAPI {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		f.handleLookup(w, r)
		return
	}
	if r.Method == http.MethodPost {
		f.handleCreate(w, r)
		return
	}
	http.NotFound(w, r)
}

func (f *swarmFixture) handleLookup(w http.ResponseWriter, r *http.Request) {
	f.gets++
	if r.URL.Query().Get("scenario") != recordScenario || r.URL.Query().Get("kind") != recordKind {
		f.t.Fatalf("lookup query = %s", r.URL.RawQuery)
	}
	records := []recordResponse{}
	if f.existing.ID != "" {
		records = append(records, f.existing)
	}
	_ = json.NewEncoder(w).Encode(recordListResponse{Records: records})
}

func (f *swarmFixture) handleCreate(w http.ResponseWriter, r *http.Request) {
	f.posts++
	if err := json.NewDecoder(r.Body).Decode(&f.posted); err != nil {
		f.t.Fatalf("decode post: %v", err)
	}
	_ = json.NewEncoder(w).Encode(recordEnvelope{Record: recordResponse{ID: f.createID, Trigger: f.posted.Trigger}})
}

func TestScenarioQABugReporterCreatesKnowledgeEntry(t *testing.T) {
	server, fixture := newScenarioQAServer(t, "", "bug-inbox/code-defect/cache-drift-breaks-plan-sync-")
	defer server.Close()

	reporter := NewScenarioQABugReporter(server.Client(), staticURLResolver{promptManagerSystem: server.URL})
	ref, err := reporter.FileBug(context.Background(), Entry{
		ID:               "entry-abcdef123456",
		Type:             planmodel.LogEntryBugReport,
		PlanID:           "plan-1",
		ExecutionID:      "exec-1",
		PhaseID:          "phase-1",
		Title:            "Cache drift breaks plan sync",
		Detail:           "Observed stale downstream status after retry.",
		Severity:         planmodel.LogSeverityHigh,
		SourceCommand:    "plan-manager log bug-add exec-1 --title ...",
		AttributionRunID: "run-1",
		CreatedAt:        "2026-07-06T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("FileBug error: %v", err)
	}
	if ref.System != scenarioQASystem || ref.Kind != "bug_report" || ref.Reference != "knw-123" {
		t.Fatalf("ref = %+v", ref)
	}
	if fixture.gets != 1 || fixture.posts != 1 {
		t.Fatalf("GET/POST counts = %d/%d", fixture.gets, fixture.posts)
	}
	if fixture.posted.Topic == "" || !strings.Contains(fixture.posted.Content, "severity: major") || !strings.Contains(fixture.posted.Content, "entry_id: entry-abcdef123456") {
		t.Fatalf("posted payload missing contract fields: topic=%q content=%s", fixture.posted.Topic, fixture.posted.Content)
	}
	if fixture.postedAttribution["kind"] != "writer-skill" || fixture.postedAttribution["team_id"] != scenarioQASystem || fixture.postedAttribution["source_skill_id"] != bugWriterSkillID {
		t.Fatalf("bad attribution: %#v", fixture.postedAttribution)
	}
}

func TestScenarioQABugReporterReturnsExistingKnowledgeEntry(t *testing.T) {
	server, fixture := newScenarioQAServer(t, "knw-existing", "")
	defer server.Close()

	reporter := NewScenarioQABugReporter(server.Client(), staticURLResolver{promptManagerSystem: server.URL})
	ref, err := reporter.FileBug(context.Background(), Entry{ID: "entry-1", Title: "duplicate", CreatedAt: "2026-07-06T10:00:00Z"})
	if err != nil {
		t.Fatalf("FileBug error: %v", err)
	}
	if ref.Reference != "knw-existing" {
		t.Fatalf("reference = %q", ref.Reference)
	}
	if fixture.posts != 0 {
		t.Fatalf("expected no create when lookup finds an entry, got %d posts", fixture.posts)
	}
}

func TestSwarmRecordWriterCreatesRecord(t *testing.T) {
	server, fixture := newSwarmServer(t, recordResponse{})
	defer server.Close()

	writer := NewSwarmRecordWriter(server.Client(), staticURLResolver{swarmManagerSystem: server.URL})
	ref, err := writer.WriteRecord(context.Background(), Entry{
		ID:          "entry-rec-1",
		Type:        planmodel.LogEntryRecord,
		PlanID:      "plan-1",
		ExecutionID: "exec-1",
		Title:       "Reusable downstream contract",
		Detail:      "Use lookup-before-create.",
		Evidence:    []string{"go test ./api/internal/planlog"},
	})
	if err != nil {
		t.Fatalf("WriteRecord error: %v", err)
	}
	if ref.System != swarmManagerSystem || ref.Kind != "record" || ref.Reference != "rec-123" {
		t.Fatalf("ref = %+v", ref)
	}
	if fixture.gets != 1 || fixture.posts != 1 {
		t.Fatalf("GET/POST counts = %d/%d", fixture.gets, fixture.posts)
	}
	if fixture.posted.Kind != recordKind || fixture.posted.Scenario != recordScenario || fixture.posted.Outcome != recordOutcome || fixture.posted.CreatedBy != "plan-manager" {
		t.Fatalf("posted metadata = %+v", fixture.posted)
	}
	if !strings.Contains(fixture.posted.Trigger, "[planlog-entry:entry-rec-1]") || !strings.Contains(fixture.posted.Approach, "entry_id: entry-rec-1") {
		t.Fatalf("posted payload missing provenance: %+v", fixture.posted)
	}
}

func TestSwarmRecordWriterReturnsExistingRecord(t *testing.T) {
	server, fixture := newSwarmServer(t, recordResponse{
		ID:      "rec-existing",
		Trigger: "[planlog-entry:entry-rec-1] Reusable downstream contract",
	})
	defer server.Close()

	writer := NewSwarmRecordWriter(server.Client(), staticURLResolver{swarmManagerSystem: server.URL})
	ref, err := writer.WriteRecord(context.Background(), Entry{ID: "entry-rec-1", Title: "duplicate"})
	if err != nil {
		t.Fatalf("WriteRecord error: %v", err)
	}
	if ref.Reference != "rec-existing" {
		t.Fatalf("reference = %q", ref.Reference)
	}
	if fixture.posts != 0 {
		t.Fatalf("expected no create when lookup finds a record, got %d posts", fixture.posts)
	}
}

func TestDownstreamAdaptersMapResolutionFailureToUnavailable(t *testing.T) {
	reporter := NewScenarioQABugReporter(http.DefaultClient, staticURLResolver{})
	_, err := reporter.FileBug(context.Background(), Entry{ID: "entry-1", Title: "bug"})
	var unavailable ErrDownstreamUnavailable
	if !errors.As(err, &unavailable) {
		t.Fatalf("FileBug error = %T %[1]v, want ErrDownstreamUnavailable", err)
	}

	writer := NewSwarmRecordWriter(http.DefaultClient, staticURLResolver{})
	_, err = writer.WriteRecord(context.Background(), Entry{ID: "entry-1", Title: "record"})
	if !errors.As(err, &unavailable) {
		t.Fatalf("WriteRecord error = %T %[1]v, want ErrDownstreamUnavailable", err)
	}
}
