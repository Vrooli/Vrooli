package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/testutil"
)

// mockInitiativeAssigner implements InitiativeAssigner for testing.
type mockInitiativeAssigner struct {
	snapshots   map[string]InitiativeSnapshot
	addedItems  map[string][]string
	createOrder []string
	updateOrder []string
	replaceLog  []InitiativeSnapshot
	getErr      error
	createErr   error
	updateErr   error
	replaceErr  error
	deleteErr   error
	addErr      error
}

func newMockInitiativeAssigner() *mockInitiativeAssigner {
	return &mockInitiativeAssigner{
		snapshots:  make(map[string]InitiativeSnapshot),
		addedItems: make(map[string][]string),
	}
}

func (m *mockInitiativeAssigner) Get(name string) (*InitiativeSnapshot, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	snapshot, ok := m.snapshots[name]
	if !ok {
		return nil, fmt.Errorf("initiative %q not found", name)
	}
	copied := snapshot
	copied.Items = append([]string(nil), snapshot.Items...)
	copied.DependsOn = append([]string(nil), snapshot.DependsOn...)
	return &copied, nil
}

func (m *mockInitiativeAssigner) Create(spec InitiativeSpec) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createOrder = append(m.createOrder, spec.Name)
	m.snapshots[spec.Name] = InitiativeSnapshot{
		Name:        spec.Name,
		Title:       spec.Title,
		Description: spec.Description,
		Status:      spec.Status,
		Priority:    spec.Priority,
		DependsOn:   append([]string(nil), spec.DependsOn...),
		Items:       nil,
		CreatedBy:   spec.CreatedBy,
	}
	return nil
}

func (m *mockInitiativeAssigner) Update(spec InitiativeSpec) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	snapshot, ok := m.snapshots[spec.Name]
	if !ok {
		return fmt.Errorf("initiative %q not found", spec.Name)
	}
	m.updateOrder = append(m.updateOrder, spec.Name)
	snapshot.Title = spec.Title
	snapshot.Description = spec.Description
	snapshot.Status = spec.Status
	snapshot.Priority = spec.Priority
	snapshot.DependsOn = append([]string(nil), spec.DependsOn...)
	m.snapshots[spec.Name] = snapshot
	return nil
}

func (m *mockInitiativeAssigner) Replace(snapshot InitiativeSnapshot) error {
	if m.replaceErr != nil {
		return m.replaceErr
	}
	copied := snapshot
	copied.Items = append([]string(nil), snapshot.Items...)
	copied.DependsOn = append([]string(nil), snapshot.DependsOn...)
	m.replaceLog = append(m.replaceLog, copied)
	m.snapshots[snapshot.Name] = copied
	return nil
}

func (m *mockInitiativeAssigner) Delete(name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.snapshots, name)
	return nil
}

func (m *mockInitiativeAssigner) AddItems(name string, items []string) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.addedItems[name] = append(m.addedItems[name], items...)
	snapshot, ok := m.snapshots[name]
	if !ok {
		return fmt.Errorf("initiative %q not found", name)
	}
	snapshot.Items = append(snapshot.Items, items...)
	m.snapshots[name] = snapshot
	return nil
}

func (m *mockInitiativeAssigner) RememberItem(name, ref string) error {
	if m.addErr != nil {
		return m.addErr
	}
	snapshot, ok := m.snapshots[name]
	if !ok {
		return fmt.Errorf("initiative %q not found", name)
	}
	for _, existing := range snapshot.Items {
		if existing == ref {
			return nil
		}
	}
	snapshot.Items = append(snapshot.Items, ref)
	m.snapshots[name] = snapshot
	m.addedItems[name] = append(m.addedItems[name], ref)
	return nil
}

func (m *mockInitiativeAssigner) ForgetItem(name, ref string) error {
	snapshot, ok := m.snapshots[name]
	if !ok {
		return nil
	}
	filtered := make([]string, 0, len(snapshot.Items))
	for _, existing := range snapshot.Items {
		if existing == ref {
			continue
		}
		filtered = append(filtered, existing)
	}
	snapshot.Items = filtered
	m.snapshots[name] = snapshot
	return nil
}

func setupBatchTestHandler(t *testing.T) (*Handler, string, *mockInitiativeAssigner) {
	t.Helper()
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "batch-run", TaskID: "batch-task"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	ia := newMockInitiativeAssigner()
	h.SetInitiativeAssigner(ia)
	return h, rootDir, ia
}

func doBatchCreate(t *testing.T, h *Handler, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/v1/backlog/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BatchCreate(w, req)
	return w
}

func TestBatchCreate_Success(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "auth-service", Title: "Auth Service", Kind: "idea"},
			{Name: "user-mgmt", Title: "User Management", Kind: "idea"},
			{Name: "fix-login", Title: "Fix Login Bug", Kind: "fix"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	var resp batchCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Count != 3 {
		t.Errorf("expected count 3, got %d", resp.Count)
	}
	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}

	// Verify items exist on disk.
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "auth-service", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "user-mgmt", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "fix", "fix-login", "spec.json"))

	// No initiative was requested.
	if len(ia.snapshots) != 0 {
		t.Errorf("expected no initiatives created, got %d", len(ia.snapshots))
	}
}

func TestBatchCreate_StampsCreatedByFromRequestProvenance(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "agent-batch-a", Title: "Agent Batch A", Kind: "idea", Initiative: "agent-batch-init"},
			{Name: "agent-batch-b", Title: "Agent Batch B", Kind: "execute"},
		},
		Initiatives: []batchCreateInitiative{
			{Name: "agent-batch-init", Title: "Agent Batch Initiative"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	prov := identity.Provenance{
		Type:       identity.TypeAgent,
		RunID:      "run-batch-1",
		TaskID:     "task-batch-1",
		ProfileKey: "swarm-manager/default",
	}
	req := httptest.NewRequest("POST", "/api/v1/backlog/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(identity.NewContext(req.Context(), prov))
	w := httptest.NewRecorder()

	h.BatchCreate(w, req)

	testutil.AssertStatusCreated(t, w)
	for _, tc := range []struct {
		kind BacklogKind
		dir  string
		name string
	}{
		{kind: KindIdea, dir: "ideas", name: "agent-batch-a"},
		{kind: KindExecute, dir: "execute", name: "agent-batch-b"},
	} {
		saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, tc.dir, tc.name, "spec.json"))
		if saved.CreatedBy == nil {
			t.Fatalf("%s/%s missing created_by", tc.kind, tc.name)
		}
		if *saved.CreatedBy != prov {
			t.Fatalf("%s/%s created_by = %+v, want %+v", tc.kind, tc.name, saved.CreatedBy, prov)
		}
	}
	snapshot := h.initiativeAssigner.(*mockInitiativeAssigner).snapshots["agent-batch-init"]
	if snapshot.CreatedBy == nil {
		t.Fatal("batch-created initiative missing created_by")
	}
	if *snapshot.CreatedBy != prov {
		t.Fatalf("batch-created initiative created_by = %+v, want %+v", snapshot.CreatedBy, prov)
	}
}

func TestBatchCreate_SessionProvenanceRecordsItemAndInitiativeArtifacts(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)
	recorder := &fakeSessionArtifacts{}
	h.SetAgentSessionArtifactRecorder(recorder)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "session-batch-a", Title: "Session Batch A", Kind: "idea", Initiative: "session-batch-init"},
			{Name: "session-batch-b", Title: "Session Batch B", Kind: "execute", Initiative: "session-batch-init"},
		},
		Initiatives: []batchCreateInitiative{
			{Name: "session-batch-init", Title: "Session Batch Initiative"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	prov := identity.Provenance{
		Type:        identity.TypeAgent,
		RunID:       "run-batch-session",
		TaskID:      "task-batch-session",
		ProfileKey:  "swarm-manager/default",
		SessionID:   "sess_batch",
		SessionKind: "meta_orchestration",
		Source:      "session/sess_batch",
	}
	req := httptest.NewRequest("POST", "/api/v1/backlog/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(identity.NewContext(req.Context(), prov))
	w := httptest.NewRecorder()

	h.BatchCreate(w, req)

	testutil.AssertStatusCreated(t, w)
	if len(recorder.artifacts) != 3 {
		t.Fatalf("artifacts = %d, want 3: %+v", len(recorder.artifacts), recorder.artifacts)
	}
	refs := map[string]agentsessions.ArtifactType{}
	for _, artifact := range recorder.artifacts {
		refs[artifact.EntityRef] = artifact.ArtifactType
		if artifact.SessionID != "sess_batch" || artifact.RunID != "run-batch-session" {
			t.Fatalf("unexpected artifact provenance: %+v", artifact)
		}
	}
	if refs["idea/session-batch-a"] != agentsessions.ArtifactBacklogItem {
		t.Fatalf("missing backlog artifact for session-batch-a: %+v", refs)
	}
	if refs["execute/session-batch-b"] != agentsessions.ArtifactBacklogItem {
		t.Fatalf("missing backlog artifact for session-batch-b: %+v", refs)
	}
	if refs["session-batch-init"] != agentsessions.ArtifactInitiative {
		t.Fatalf("missing initiative artifact: %+v", refs)
	}
}

func TestApplyAgentSessionBacklogBatchImportCreatesItemsAndArtifacts(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)
	recorder := &fakeSessionArtifacts{}
	h.SetAgentSessionArtifactRecorder(recorder)

	payload := `{
		"items": [
			{"name": "session-apply-a", "title": "Session Apply A", "kind": "idea", "initiative": "session-apply-init"},
			{"name": "session-apply-b", "title": "Session Apply B", "kind": "execute", "initiative": "session-apply-init"}
		],
		"initiatives": [
			{"name": "session-apply-init", "title": "Session Apply Initiative"}
		]
	}`
	prov := identity.Provenance{
		Type:        identity.TypeAgent,
		RunID:       "run-session-apply",
		TaskID:      "task-session-apply",
		ProfileKey:  "swarm-manager/default",
		SessionID:   "sess_apply",
		SessionKind: "meta_orchestration",
		Source:      "session/sess_apply",
	}

	artifacts, err := h.ApplyAgentSessionBacklogBatchImport(identity.NewContext(context.Background(), prov), payload, prov)
	if err != nil {
		t.Fatalf("ApplyAgentSessionBacklogBatchImport() error = %v", err)
	}
	if len(artifacts) != 3 {
		t.Fatalf("artifacts = %d, want 3: %+v", len(artifacts), artifacts)
	}
	for _, tc := range []struct {
		kind BacklogKind
		dir  string
		name string
	}{
		{kind: KindIdea, dir: "ideas", name: "session-apply-a"},
		{kind: KindExecute, dir: "execute", name: "session-apply-b"},
	} {
		saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, tc.dir, tc.name, "spec.json"))
		if saved.CreatedBy == nil || saved.CreatedBy.SessionID != "sess_apply" || saved.CreatedBy.RunID != "run-session-apply" {
			t.Fatalf("%s/%s created_by = %+v", tc.kind, tc.name, saved.CreatedBy)
		}
	}
	if got := ia.snapshots["session-apply-init"].CreatedBy; got == nil || got.SessionID != "sess_apply" {
		t.Fatalf("initiative created_by = %+v", got)
	}
	for _, artifact := range recorder.artifacts {
		if artifact.MutationSource != "agent_sessions.apply.backlog_batch_import" {
			t.Fatalf("artifact mutation source = %q", artifact.MutationSource)
		}
	}
}

func TestApplyAgentSessionBacklogBatchImportRollsBackWhenArtifactRecordingFails(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)
	recorder := &fakeSessionArtifacts{err: fmt.Errorf("artifact store unavailable")}
	h.SetAgentSessionArtifactRecorder(recorder)

	payload := `{
		"items": [
			{"name": "session-rollback-a", "title": "Session Rollback A", "kind": "idea", "initiative": "session-rollback-init"},
			{"name": "session-rollback-b", "title": "Session Rollback B", "kind": "execute", "initiative": "session-rollback-init"}
		],
		"initiatives": [
			{"name": "session-rollback-init", "title": "Session Rollback Initiative"}
		]
	}`
	prov := identity.Provenance{
		Type:        identity.TypeAgent,
		RunID:       "run-session-rollback",
		TaskID:      "task-session-rollback",
		ProfileKey:  "swarm-manager/default",
		SessionID:   "sess_rollback",
		SessionKind: "meta_orchestration",
		Source:      "session/sess_rollback",
	}

	_, err := h.ApplyAgentSessionBacklogBatchImport(identity.NewContext(context.Background(), prov), payload, prov)
	if err == nil {
		t.Fatal("ApplyAgentSessionBacklogBatchImport() error = nil, want artifact recording failure")
	}
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "session-rollback-a", "spec.json"))
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "execute", "session-rollback-b", "spec.json"))
	if _, ok := ia.snapshots["session-rollback-init"]; ok {
		t.Fatalf("initiative was not rolled back: %+v", ia.snapshots["session-rollback-init"])
	}
}

func TestBatchCreate_WithInitiative(t *testing.T) {
	h, _, ia := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "dashboard", Title: "Dashboard", Kind: "idea", Initiative: "q1-sprint"},
			{Name: "api-refactor", Title: "API Refactor", Kind: "execute", Initiative: "q1-sprint"},
		},
		Initiatives: []batchCreateInitiative{
			{Name: "q1-sprint", Title: "Q1 Sprint"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	var resp batchCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Initiatives) != 1 || resp.Initiatives[0].Name != "q1-sprint" {
		t.Fatalf("expected initiative summary for q1-sprint, got %+v", resp.Initiatives)
	}
	if resp.Initiatives[0].Action != "create" {
		t.Errorf("expected create action, got %q", resp.Initiatives[0].Action)
	}

	snapshot, ok := ia.snapshots["q1-sprint"]
	if !ok {
		t.Fatal("expected initiative 'q1-sprint' to be created")
	}
	if len(ia.addedItems["q1-sprint"]) != 2 {
		t.Errorf("expected 2 items added to initiative, got %d", len(ia.addedItems["q1-sprint"]))
	}
	if len(snapshot.Items) != 2 {
		t.Errorf("expected 2 persisted initiative items, got %d", len(snapshot.Items))
	}
}

func TestBatchCreate_EmptyBatch(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

func TestBatchCreate_RejectsUnknownField(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/backlog/batch", strings.NewReader(`{
		"items": [
			{
				"name": "same-item",
				"title": "Same Item",
				"kind": "idea",
				"scope": "scenarios/swarm-manager"
			}
		]
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.BatchCreate(w, req)

	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), `unknown field "scope"`) {
		t.Fatalf("expected unknown scope field error, got: %s", w.Body.String())
	}
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "same-item", "spec.json"))
	if len(ia.snapshots) != 0 {
		t.Fatalf("expected no initiative mutations, got %d", len(ia.snapshots))
	}
}

func TestBatchCreate_DuplicateNameInBatch(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "same-item", Title: "First", Kind: "idea"},
			{Name: "same-item", Title: "Second", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	// Verify no items were created (all-or-nothing).
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "same-item", "spec.json"))
}

func TestBatchCreate_InvalidItem_RollsBackAll(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "valid-item", Title: "Valid", Kind: "idea"},
			{Name: "bad-item", Title: "", Kind: "idea"}, // missing title
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	// Verify no items were created.
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "valid-item", "spec.json"))
}

func TestBatchCreate_CycleDetection(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// Create an existing item that item-b will depend on.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:   "existing",
		Title:  "Existing",
		Status: StatusBacklog,
		Tags:   []string{},
	})

	// Create a cycle: a depends on b, b depends on a.
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-a", Title: "Item A", Kind: "idea", DependsOn: []string{"idea/item-b"}},
			{Name: "item-b", Title: "Item B", Kind: "idea", DependsOn: []string{"idea/item-a"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	body := w.Body.String()
	if !contains(body, "cycle") {
		t.Errorf("expected cycle error, got: %s", body)
	}

	// Verify no items were created.
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "item-a", "spec.json"))
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "item-b", "spec.json"))
}

func TestBatchCreate_WithIntraBatchDependencies(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// item-b depends on item-a (both in the batch) — valid, no cycle.
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-a", Title: "Item A", Kind: "idea"},
			{Name: "item-b", Title: "Item B", Kind: "idea", DependsOn: []string{"idea/item-a"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-a", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-b", "spec.json"))
}

func TestBatchCreate_DependencyOnNonexistent(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "orphan", Title: "Orphan", Kind: "idea", DependsOn: []string{"idea/ghost"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	body := w.Body.String()
	if !contains(body, "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %s", body)
	}
}

func TestBatchCreate_ConflictWithExisting(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:   "existing",
		Title:  "Existing",
		Status: StatusBacklog,
		Tags:   []string{},
	})

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "existing", Title: "Duplicate", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusConflict)
}

func TestBatchCreate_InvalidKind(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "invalid-kind"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

func TestBatchCreate_PriorityValidation(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	p := int32(15)
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "idea", Priority: &p},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

// failingSaveStore wraps a FileStore but returns an error on the Nth SaveItem call.
// This allows testing the rollback path in batch-create.
type failingSaveStore struct {
	*FileStore
	failOnCall int // 1-based: fail on the Nth SaveItem call
	callCount  int
}

func (f *failingSaveStore) SaveItem(item BacklogItem) error {
	f.callCount++
	if f.callCount >= f.failOnCall {
		return fmt.Errorf("simulated disk write failure")
	}
	return f.FileStore.SaveItem(item)
}

func TestBatchCreate_SaveFailure_RollsBack(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// Inject a store that fails on the 2nd SaveItem call.
	// The first item creates successfully, the second triggers rollback.
	h.store = &failingSaveStore{
		FileStore:  NewFileStore(rootDir),
		failOnCall: 2,
	}

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-ok", Title: "Item OK", Kind: "idea"},
			{Name: "item-fail", Title: "Item Fail", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusInternalServerError)

	// Verify rollback: both item directories should be cleaned up.
	okDir := filepath.Join(rootDir, "ideas", "item-ok")
	failDir := filepath.Join(rootDir, "ideas", "item-fail")

	if _, err := os.Stat(okDir); !os.IsNotExist(err) {
		t.Errorf("expected %q to be removed during rollback, but it still exists", okDir)
	}
	if _, err := os.Stat(failDir); !os.IsNotExist(err) {
		t.Errorf("expected %q to be removed during rollback, but it still exists", failDir)
	}
}

func TestBatchCreate_ExceedsMaxBatchSize(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	items := make([]batchCreateItem, 101)
	for i := range items {
		items[i] = batchCreateItem{
			Name:  fmt.Sprintf("item-%03d", i),
			Title: fmt.Sprintf("Item %d", i),
			Kind:  "idea",
		}
	}

	payload := batchCreateRequest{Items: items}
	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBatchCreate_InitiativeAddItemsFails_RollsBackEverything(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)
	ia.addErr = fmt.Errorf("disk full")

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "widget-a", Title: "Widget A", Kind: "idea", Initiative: "my-init"},
			{Name: "widget-b", Title: "Widget B", Kind: "fix", Initiative: "my-init"},
		},
		Initiatives: []batchCreateInitiative{
			{Name: "my-init", Title: "My Initiative"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusInternalServerError)
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "widget-a", "spec.json"))
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "fix", "widget-b", "spec.json"))
	if _, ok := ia.snapshots["my-init"]; ok {
		t.Fatal("expected initiative rollback to remove my-init")
	}
}

func TestBatchCreate_InitiativeCreateFails_Returns500(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)
	ia.createErr = fmt.Errorf("store corrupt")

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "orphan-item", Title: "Orphan", Kind: "idea", Initiative: "broken-init"},
		},
		Initiatives: []batchCreateInitiative{
			{Name: "broken-init", Title: "Broken Init"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusInternalServerError)

	// Item should NOT be on disk — initiative creation fails before item creation.
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "orphan-item", "spec.json"))
}

func TestBatchCreate_PreviewDoesNotMutateDiskOrInitiatives(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Preview: true,
		Items: []batchCreateItem{
			{Name: "preview-item", Title: "Preview Item", Kind: "idea", Initiative: "preview-init"},
		},
		Initiatives: []batchCreateInitiative{
			{Name: "preview-init", Title: "Preview Init"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if !resp.Preview {
		t.Fatal("expected preview response")
	}
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "preview-item", "spec.json"))
	if len(ia.snapshots) != 0 {
		t.Fatalf("expected preview to avoid initiative mutations, got %d snapshots", len(ia.snapshots))
	}
}

func TestBatchCreate_DependencyValidation_ExistingAndBatch(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// Create item-a on disk so it can be referenced.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:   "item-a",
		Title:  "Item A",
		Status: StatusBacklog,
		Tags:   []string{},
	})

	// Batch-create: item-b depends on existing item-a, item-c depends on batch item-b.
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-b", Title: "Item B", Kind: "idea", DependsOn: []string{"idea/item-a"}},
			{Name: "item-c", Title: "Item C", Kind: "idea", DependsOn: []string{"idea/item-b"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	// Verify both items exist with correct dependencies.
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-b", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-c", "spec.json"))

	// Now try to batch-create an item with a nonexistent dependency.
	payload2 := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-d", Title: "Item D", Kind: "idea", DependsOn: []string{"idea/nonexistent"}},
		},
	}

	w2 := doBatchCreate(t, h, payload2)
	testutil.AssertStatusBadRequest(t, w2)

	body := w2.Body.String()
	if !contains(body, "does not exist") {
		t.Errorf("expected 'does not exist' error for nonexistent dep, got: %s", body)
	}
}

func TestBatchCreate_WithEffort(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	effortS := "S"
	effortXL := "xl" // lowercase to test normalization
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-with-effort", Title: "Item With Effort", Kind: "idea", Effort: &effortS},
			{Name: "item-with-xl", Title: "Item XL", Kind: "fix", Effort: &effortXL},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Effort != "S" {
		t.Errorf("expected effort 'S', got %q", resp.Items[0].Effort)
	}
	if resp.Items[1].Effort != "XL" {
		t.Errorf("expected effort 'XL', got %q", resp.Items[1].Effort)
	}
}

func TestBatchCreate_InvalidEffort(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	badEffort := "HUGE"
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "bad-effort-item", Title: "Bad Effort", Kind: "idea", Effort: &badEffort},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	body := w.Body.String()
	if !contains(body, "effort must be") {
		t.Errorf("expected effort validation error, got: %s", body)
	}
}

func TestBatchCreate_EffortOptional(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "no-effort-item", Title: "No Effort", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if resp.Items[0].Effort != "" {
		t.Errorf("expected empty effort, got %q", resp.Items[0].Effort)
	}
}
