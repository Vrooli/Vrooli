package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockScenarioCLI struct {
	scenarios []scenarioSummary
}

func (m *mockScenarioCLI) ListScenarios(ctx context.Context) ([]scenarioSummary, error) {
	return m.scenarios, nil
}

type mockManifestBuilder struct {
	calls []DeploymentManifestRequest
}

func (m *mockManifestBuilder) Build(ctx context.Context, req DeploymentManifestRequest) (*DeploymentManifest, error) {
	m.calls = append(m.calls, req)
	now := time.Now()
	return &DeploymentManifest{
		Scenario:    req.Scenario,
		Tier:        req.Tier,
		GeneratedAt: now,
		Summary: DeploymentSummary{
			TotalSecrets:       4,
			StrategizedSecrets: 2,
			RequiresAction:     2,
			BlockingSecrets:    []string{"alpha:SECRET"},
		},
	}, nil
}

type mockCampaignStore struct {
	list    []CampaignSummary
	upserts []CampaignSummary
}

func (m *mockCampaignStore) List(ctx context.Context, scenarioFilter string) ([]CampaignSummary, error) {
	if scenarioFilter == "" {
		return m.list, nil
	}
	var filtered []CampaignSummary
	for _, c := range m.list {
		if c.Scenario == scenarioFilter {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

func (m *mockCampaignStore) Upsert(ctx context.Context, campaign CampaignSummary) error {
	m.upserts = append(m.upserts, campaign)
	found := false
	for i, c := range m.list {
		if c.ID == campaign.ID {
			m.list[i] = campaign
			found = true
			break
		}
	}
	if !found {
		m.list = append(m.list, campaign)
	}
	return nil
}

func TestListCampaignsFiltersAndReadiness(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VROOLI_ROOT", tmp)

	cli := &mockScenarioCLI{scenarios: []scenarioSummary{{Name: "alpha"}, {Name: "beta"}}}
	builder := &mockManifestBuilder{}
	handlers := NewCampaignHandlersWithCLI(cli, builder, nil)

	req := httptest.NewRequest("GET", "/campaigns?include_readiness=true&scenario=alpha", nil)
	rr := httptest.NewRecorder()
	handlers.ListCampaigns(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var payload struct {
		Campaigns []CampaignSummary `json:"campaigns"`
		Count     int               `json:"count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Count != 1 || len(payload.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", payload.Count)
	}
	if payload.Campaigns[0].Scenario != "alpha" {
		t.Fatalf("filtered scenario mismatch: %s", payload.Campaigns[0].Scenario)
	}
	if payload.Campaigns[0].Summary == nil {
		t.Fatalf("expected readiness summary attached")
	}
	if len(builder.calls) != 1 || builder.calls[0].Scenario != "alpha" {
		t.Fatalf("manifest builder called incorrectly: %+v", builder.calls)
	}
}

func TestUpsertCampaignPersistsToFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VROOLI_ROOT", tmp)

	handlers := NewCampaignHandlersWithCLI(&mockScenarioCLI{}, &mockManifestBuilder{}, nil)

	body := bytes.NewBufferString(`{"scenario":"alpha","tier":"tier-2-desktop","status":"active","progress":10}`)
	req := httptest.NewRequest("POST", "/campaigns", body)
	rr := httptest.NewRecorder()
	handlers.UpsertCampaign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	path := filepath.Join(tmp, "scenarios", "secrets-manager", "data", "campaigns.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("campaign file not written: %v", err)
	}

	var payload campaignFilePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal file: %v", err)
	}
	if len(payload.Campaigns) != 1 || payload.Campaigns[0].Scenario != "alpha" {
		t.Fatalf("unexpected campaigns: %+v", payload.Campaigns)
	}
}

func TestListCampaignsUsesStoreAndSeed(t *testing.T) {
	store := &mockCampaignStore{
		list: []CampaignSummary{{
			ID:       "alpha::tier-2-desktop",
			Scenario: "alpha",
			Tier:     "tier-2-desktop",
			Status:   "active",
		}},
	}
	cli := &mockScenarioCLI{scenarios: []scenarioSummary{{Name: "alpha"}, {Name: "beta"}}}
	builder := &mockManifestBuilder{}
	handlers := NewCampaignHandlersWithCLI(cli, builder, store)

	req := httptest.NewRequest("GET", "/campaigns", nil)
	rr := httptest.NewRecorder()
	handlers.ListCampaigns(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var payload struct {
		Campaigns []CampaignSummary `json:"campaigns"`
		Count     int               `json:"count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Count != 2 {
		t.Fatalf("expected 2 campaigns (store + seed), got %d", payload.Count)
	}
}

func TestUpsertCampaignUsesStore(t *testing.T) {
	store := &mockCampaignStore{}
	handlers := NewCampaignHandlersWithCLI(&mockScenarioCLI{}, &mockManifestBuilder{}, store)

	body := bytes.NewBufferString(`{"scenario":"alpha","tier":"tier-2-desktop","status":"active","progress":5}`)
	req := httptest.NewRequest("POST", "/campaigns", body)
	rr := httptest.NewRecorder()
	handlers.UpsertCampaign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if len(store.upserts) != 1 {
		t.Fatalf("expected 1 upsert, got %d", len(store.upserts))
	}
	if store.upserts[0].Scenario != "alpha" {
		t.Fatalf("unexpected upsert payload: %+v", store.upserts[0])
	}
}
