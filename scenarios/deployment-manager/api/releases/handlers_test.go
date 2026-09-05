package releases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"deployment-manager/profiles"

	"github.com/gorilla/mux"
)

// --- mocks ---

type mockRepo struct {
	mu        sync.Mutex
	store     map[string]*Release
	locked    map[string]bool
	insertErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{store: map[string]*Release{}, locked: map[string]bool{}}
}

func (m *mockRepo) Insert(_ context.Context, rel *Release) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertErr != nil {
		return m.insertErr
	}
	m.store[rel.ID] = rel
	return nil
}

func (m *mockRepo) Get(_ context.Context, id string) (*Release, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rel, ok := m.store[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return rel, nil
}

func (m *mockRepo) ListByProfile(_ context.Context, profileID string, _ int) ([]*Release, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Release
	for _, r := range m.store {
		if r.ProfileID == profileID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rel, ok := m.store[id]; ok {
		rel.Status = status
	}
	return nil
}

func (m *mockRepo) SetVerificationEvidence(_ context.Context, id string, items []VerificationItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rel, ok := m.store[id]; ok {
		rel.VerificationEvidence = items
	}
	return nil
}

func (m *mockRepo) MarkPlatformPublished(_ context.Context, id, platform string, artifactID int64) error {
	return nil
}

func (m *mockRepo) MarkPlatformStatus(_ context.Context, id, platform, status, errMsg string) error {
	return nil
}

func (m *mockRepo) MarkSuperseded(_ context.Context, profileID, channel, exceptID string) error {
	return nil
}

func (m *mockRepo) AcquireProfileLock(_ context.Context, profileID string) (bool, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.locked[profileID] {
		return false, func() {}, nil
	}
	m.locked[profileID] = true
	return true, func() {
		m.mu.Lock()
		delete(m.locked, profileID)
		m.mu.Unlock()
	}, nil
}

func (m *mockRepo) RecordReadinessWaiver(_ context.Context, profileID, commit, reason, actor string) error {
	if profileID == "" || commit == "" || reason == "" || actor == "" {
		return errors.New("waiver fields required")
	}
	return nil
}

func (m *mockRepo) GetLatestReadiness(_ context.Context, _ string) (*ReadinessRecord, error) {
	return &ReadinessRecord{}, nil
}

type mockLPBSConfig struct {
	mu      sync.Mutex
	configs map[string]*profiles.LPBSReleaseConfig
}

func newMockLPBSConfig() *mockLPBSConfig {
	return &mockLPBSConfig{configs: map[string]*profiles.LPBSReleaseConfig{}}
}

func (m *mockLPBSConfig) Get(_ context.Context, profileID string) (*profiles.LPBSReleaseConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configs[profileID], nil
}

func (m *mockLPBSConfig) Upsert(_ context.Context, cfg *profiles.LPBSReleaseConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[cfg.ProfileID] = cfg
	return nil
}

func (m *mockLPBSConfig) Delete(_ context.Context, profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.configs, profileID)
	return nil
}

type mockOrch struct {
	called  int
	lastReq DeployRequest
	result  *DeployResult
	err     error
}

func (m *mockOrch) RunDeploy(_ context.Context, req DeployRequest) (*DeployResult, error) {
	m.called++
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &DeployResult{Status: "complete"}, nil
}

type mockVerifier struct {
	outcomes map[string]*VerifyOutcome
	err      error
}

func (m *mockVerifier) Verify(_ context.Context, req *VerifyCall) (*VerifyOutcome, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := req.Platform + ":" + req.ExpectedVersion
	if v, ok := m.outcomes[key]; ok {
		return v, nil
	}
	return &VerifyOutcome{
		Match: true, SHA512Match: true, Platform: req.Platform, Channel: req.Channel,
		ExpectedVersion: req.ExpectedVersion, ObservedVersion: req.ExpectedVersion,
	}, nil
}

// --- helpers ---

func newServer(repo Repository, cfg profiles.LPBSReleaseConfigRepository, ver LPBSVerifier, orch Orchestrator) http.Handler {
	h := NewHandler(repo, cfg, ver, orch, nil)
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/profiles/{id}/releases", h.ListByProfile).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}/releases/start", h.Start).Methods("POST")
	r.HandleFunc("/api/v1/releases/{release_id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/releases/{release_id}/verify", h.Verify).Methods("POST")
	return r
}

func TestStartRequiresExactApprovedReadinessIdentity(t *testing.T) {
	tests := []struct {
		name       string
		approval   *ReadinessApproval
		wantStatus int
		wantCalls  int
	}{
		{name: "artifact mismatch", approval: &ReadinessApproval{Key: "rr-1", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:other", Targets: []string{"linux-x64"}, Channel: "stable", Status: "approved", ApprovedAt: timePointer(time.Now())}, wantStatus: http.StatusPreconditionFailed},
		{name: "exact approved identity", approval: &ReadinessApproval{Key: "rr-1", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux-x64"}, Channel: "stable", Status: "approved", ApprovedAt: timePointer(time.Now())}, wantStatus: http.StatusOK, wantCalls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
			handler := NewHandler(repo, cfg, nil, orch, nil).WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) { return tc.approval, nil })
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
			body := bytes.NewBufferString(`{"git_commit_hash":"abc","artifact_digest":"sha256:candidate","readiness_review_key":"rr-1","release_version":"1.0.0","channel":"stable","platforms":["linux-x64"]}`)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", body)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.wantStatus || orch.called != tc.wantCalls {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, orch.called, response.Body.String())
			}
		})
	}
}

func timePointer(value time.Time) *time.Time { return &value }

// --- tests ---

func TestStartRequiresCommitAndVersion(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	srv := httptest.NewServer(newServer(repo, cfg, nil, &mockOrch{}))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestStartHappyPath(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp", DefaultChannel: "stable"})
	orch := &mockOrch{}
	srv := httptest.NewServer(newServer(repo, cfg, nil, orch))
	defer srv.Close()

	body := `{"git_commit_hash":"deadbeef","release_version":"1.0.0","platforms":["linux-x64"]}`
	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if orch.called != 1 {
		t.Errorf("expected orch called once, got %d", orch.called)
	}
	if orch.lastReq.Channel != "stable" {
		t.Errorf("expected channel=stable from default, got %q", orch.lastReq.Channel)
	}
	if orch.lastReq.GitCommitHash != "deadbeef" {
		t.Errorf("expected commit=deadbeef, got %q", orch.lastReq.GitCommitHash)
	}
	// Release inserted
	if len(repo.store) != 1 {
		t.Errorf("expected 1 release in store, got %d", len(repo.store))
	}
}

func TestStartLockContentionReturns409(t *testing.T) {
	repo := newMockRepo()
	repo.locked["p1"] = true // pre-acquired by something else
	cfg := newMockLPBSConfig()
	srv := httptest.NewServer(newServer(repo, cfg, nil, &mockOrch{}))
	defer srv.Close()

	body := `{"git_commit_hash":"d","release_version":"1.0.0"}`
	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	var raw map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	if raw["error"] != "release_in_flight" {
		t.Errorf("expected release_in_flight error, got %v", raw)
	}
}

func TestStartUsesExplicitChannel(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k", DefaultChannel: "stable"})
	orch := &mockOrch{}
	srv := httptest.NewServer(newServer(repo, cfg, nil, orch))
	defer srv.Close()

	body := `{"git_commit_hash":"d","release_version":"1.0.0","channel":"beta"}`
	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if orch.lastReq.Channel != "beta" {
		t.Errorf("expected channel=beta override, got %q", orch.lastReq.Channel)
	}
}

func TestVerifyMarksFailedOnMismatch(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp"})
	rel := &Release{
		ID: "r1", ProfileID: "p1", Channel: "stable", ReleaseVersion: "1.0.0", Status: StatusPublished,
		Platforms: []ReleasePlatform{{Platform: "linux-x64"}, {Platform: "darwin-arm64"}},
	}
	_ = repo.Insert(context.Background(), rel)

	ver := &mockVerifier{outcomes: map[string]*VerifyOutcome{
		"linux-x64:1.0.0":    {Match: true, SHA512Match: true, ObservedVersion: "1.0.0"},
		"darwin-arm64:1.0.0": {Match: false, ObservedVersion: "0.9.9"},
	}}
	srv := httptest.NewServer(newServer(repo, cfg, ver, nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/releases/r1/verify", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (verify succeeds even when outcome failed), got %d", resp.StatusCode)
	}
	got, _ := repo.Get(context.Background(), "r1")
	if got.Status != StatusVerifyFailed {
		t.Errorf("expected status=verify_failed, got %q", got.Status)
	}
	if len(got.VerificationEvidence) != 2 {
		t.Errorf("expected 2 evidence items, got %d", len(got.VerificationEvidence))
	}
}

func TestListByProfileEmpty(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	srv := httptest.NewServer(newServer(repo, cfg, nil, nil))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/profiles/p1/releases")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var raw map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	if releases, ok := raw["releases"]; ok && releases != nil {
		// An empty repo lists nil slice; ok if the field is missing or null.
		if list, isList := releases.([]interface{}); isList && len(list) > 0 {
			t.Errorf("expected empty list, got %v", releases)
		}
	}
}

func TestVerifyReturns412WithoutAppKey(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1"}) // missing app key
	rel := &Release{ID: "r1", ProfileID: "p1", Channel: "stable", ReleaseVersion: "1"}
	_ = repo.Insert(context.Background(), rel)

	ver := &mockVerifier{}
	srv := httptest.NewServer(newServer(repo, cfg, ver, nil))
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/v1/releases/r1/verify", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d", resp.StatusCode)
	}
}

func TestNewReleaseIDIsHex32(t *testing.T) {
	id := newReleaseID()
	if len(id) != 32 {
		t.Errorf("expected 32-char hex id, got %q (len=%d)", id, len(id))
	}
	for _, ch := range id {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("non-hex char in id: %c", ch)
		}
	}
}

// Compile-time check that mockRepo satisfies Repository.
var _ Repository = (*mockRepo)(nil)

// Sentinel to prevent unused-import errors in case fmt drops out.
var _ = fmt.Sprintf
