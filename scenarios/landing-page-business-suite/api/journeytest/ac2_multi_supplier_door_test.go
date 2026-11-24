package journeytest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// This crosses the funded LPBS door into a production Gateway process. Its
// recorded-shape fixture offers one common chooser model on every candidate
// and one additional model for each candidate identity. Gateway's common-model
// choice must match LPBS credit attribution; candidate-specific models also
// exercise the LPBS chooser and availability filter across a broader catalog.
func TestAC2MultiSupplierCreditsDoorUsesPickedSupplier(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_MULTI_SUPPLIER", "1")
	fixtureRoot := filepath.Join(t.TempDir(), "gateway-candidates")
	if err := os.MkdirAll(fixtureRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_JOURNEY_MULTI_SUPPLIER_ROOT", fixtureRoot)
	s := newSpine(t)

	candidateIDs := ac2DoorCandidateIDs(t)
	allModels := []string{"fixture-model"}
	for _, candidateID := range candidateIDs {
		allModels = append(allModels, "fixture-model-"+candidateID)
	}
	assertAC2DoorModelChooser(t, s, allModels)

	execute := s.request(t, s.key.Token, "ac2-multi-supplier-door", "hello")
	body, err := io.ReadAll(execute.Body)
	if err != nil || execute.StatusCode != http.StatusOK {
		t.Fatalf("credits execution status=%d body=%s err=%v", execute.StatusCode, body, err)
	}
	var supplier string
	var fundingEvents int
	var initialCredits int64
	if err := s.db.QueryRow(`SELECT count(*), min(provider), min(credits) FROM business_account_usage_events WHERE operation_id=$1`, execute.Header.Get("X-Operation-ID")).Scan(&fundingEvents, &supplier, &initialCredits); err != nil {
		t.Fatal(err)
	}
	if supplier != "fixture-router-router-one" || fundingEvents != 1 || initialCredits != 200 {
		t.Fatalf("LPBS receipt supplier=%q events=%d credits=%d; want selected Router One fixture, one event, and 200 credits", supplier, fundingEvents, initialCredits)
	}
	assertCharged(t, s, 200)

	seen := map[string]bool{"router-one": true}
	totalCredits := int64(200)
	for _, candidateID := range candidateIDs {
		modelID := "fixture-model-" + candidateID
		assertAC2DoorModelChooser(t, s, allModels)
		operation := "ac2-multi-supplier-model-" + candidateID
		modelExecution := requestAC2CreditsModel(t, s, s.key.Token, operation, modelID)
		body, err := io.ReadAll(modelExecution.Body)
		if err != nil || modelExecution.StatusCode != http.StatusOK {
			t.Fatalf("candidate model %q credits execution status=%d body=%s err=%v", modelID, modelExecution.StatusCode, body, err)
		}
		var attributedSupplier string
		var fundingEvents int
		var credits int64
		if err := s.db.QueryRow(`SELECT count(*), min(provider), min(credits) FROM business_account_usage_events WHERE operation_id=$1`, modelExecution.Header.Get("X-Operation-ID")).Scan(&fundingEvents, &attributedSupplier, &credits); err != nil {
			t.Fatalf("candidate model %q usage attribution: %v", modelID, err)
		}
		wantSupplier := "fixture-router-" + candidateID
		if attributedSupplier != wantSupplier || fundingEvents != 1 || credits != 200 {
			t.Fatalf("candidate model %q settled supplier=%q events=%d credits=%d; want %q, one event, and 200", modelID, attributedSupplier, fundingEvents, credits, wantSupplier)
		}
		totalCredits += credits
		assertCharged(t, s, totalCredits)
	}
	for _, candidateID := range candidateIDs {
		if candidateID == "router-one" {
			continue
		}
		setAC2DoorCheapestSupplier(t, fixtureRoot, candidateID)
		assertAC2DoorModelChooser(t, s, allModels)
		operation := "ac2-multi-supplier-door-" + candidateID
		execute = s.request(t, s.key.Token, operation, "hello")
		body, err = io.ReadAll(execute.Body)
		if err != nil || execute.StatusCode != http.StatusOK {
			t.Fatalf("candidate %q credits execution status=%d body=%s err=%v", candidateID, execute.StatusCode, body, err)
		}
		var attributedSupplier string
		var fundingEvents int
		var credits int64
		if err := s.db.QueryRow(`SELECT count(*), min(provider), min(credits) FROM business_account_usage_events WHERE operation_id=$1`, execute.Header.Get("X-Operation-ID")).Scan(&fundingEvents, &attributedSupplier, &credits); err != nil {
			t.Fatalf("candidate %q usage attribution: %v", candidateID, err)
		}
		wantSupplier := "fixture-router-" + candidateID
		if attributedSupplier != wantSupplier || fundingEvents != 1 || credits != 200 {
			t.Fatalf("candidate %q LPBS settled supplier=%q events=%d credits=%d; want %q, one event, and 200", candidateID, attributedSupplier, fundingEvents, credits, wantSupplier)
		}
		seen[candidateID] = true
		totalCredits += credits
		assertCharged(t, s, totalCredits)
	}
	if len(seen) != 19 {
		t.Fatalf("LPBS credits door exercised %d candidates, want 19", len(seen))
	}

	// Removing the cheapest candidate from the recorded fixture models an
	// unavailable route at the hosted boundary. The funded request must then
	// settle to the next cheapest supplier and charge that supplier's receipt.
	fallbackID := ""
	for _, candidateID := range candidateIDs {
		if candidateID != "router-one" {
			fallbackID = candidateID
			break
		}
	}
	if fallbackID == "" {
		t.Fatal("no fallback candidate available after removing router-one")
	}
	setAC2DoorCheapestSupplier(t, fixtureRoot, fallbackID)
	setAC2DoorSupplierUnavailable(t, fixtureRoot, "router-one")
	modelsAfterUnavailable := make([]string, 0, len(allModels)-1)
	for _, modelID := range allModels {
		if modelID != "fixture-model-router-one" {
			modelsAfterUnavailable = append(modelsAfterUnavailable, modelID)
		}
	}
	assertAC2DoorModelChooser(t, s, modelsAfterUnavailable)
	operation := "ac2-multi-supplier-door-router-one-unavailable"
	execute = s.request(t, s.key.Token, operation, "hello")
	body, err = io.ReadAll(execute.Body)
	if err != nil || execute.StatusCode != http.StatusOK {
		t.Fatalf("unavailable-cheapest credits execution status=%d body=%s err=%v", execute.StatusCode, body, err)
	}
	var fallbackSupplier string
	var fallbackEvents int
	var fallbackCredits int64
	if err := s.db.QueryRow(`SELECT count(*), min(provider), min(credits) FROM business_account_usage_events WHERE operation_id=$1`, execute.Header.Get("X-Operation-ID")).Scan(&fallbackEvents, &fallbackSupplier, &fallbackCredits); err != nil {
		t.Fatalf("unavailable-cheapest usage attribution: %v", err)
	}
	if fallbackSupplier != "fixture-router-"+fallbackID || fallbackEvents != 1 || fallbackCredits != 200 {
		t.Fatalf("unavailable-cheapest fallback settled supplier=%q events=%d credits=%d; want %q, one event, and 200", fallbackSupplier, fallbackEvents, fallbackCredits, "fixture-router-"+fallbackID)
	}
	totalCredits += fallbackCredits
	assertCharged(t, s, totalCredits)
}

// Each chooser read crosses LPBS's authenticated Door proxy after the
// recorded route snapshot changes. The no-store assertion and subsequent
// funded execution connect current Gateway availability to the supplier LPBS
// credits are charged for.
func assertAC2DoorModelChooser(t *testing.T, s *spine, wantModels []string) {
	t.Helper()
	request, err := http.NewRequest("GET", s.server.URL+"/api/v1/ai/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+s.key.Token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		t.Fatal(err)
	}
	gotModels := make([]string, 0, len(models.Data))
	for _, model := range models.Data {
		gotModels = append(gotModels, model.ID)
	}
	sort.Strings(gotModels)
	wantModels = append([]string(nil), wantModels...)
	sort.Strings(wantModels)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Cache-Control") != "no-store" || len(gotModels) != len(wantModels) {
		t.Fatalf("model chooser status=%d cache=%q models=%v; want 200/no-store/%v", resp.StatusCode, resp.Header.Get("Cache-Control"), gotModels, wantModels)
	}
	for index := range gotModels {
		if gotModels[index] != wantModels[index] {
			t.Fatalf("model chooser status=%d cache=%q models=%v; want 200/no-store/%v", resp.StatusCode, resp.Header.Get("Cache-Control"), gotModels, wantModels)
		}
	}
}

func requestAC2CreditsModel(t *testing.T, s *spine, token, operation, model string) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"model": model, "stream": true, "messages": []map[string]string{{"role": "user", "content": "hello"}}, "outside_funded": true, "business_account_id": "attacker-account"})
	request, err := http.NewRequest("POST", s.server.URL+"/api/v1/ai/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", operation)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { response.Body.Close() })
	return response
}

func ac2DoorCandidateIDs(t *testing.T) []string {
	t.Helper()
	directory := filepath.Join("..", "..", "..", "ai-gateway", "config", "suppliers")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read AC2 supplier declarations: %v", err)
	}
	ids := []string{"openrouter"}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		ids = append(ids, entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))])
	}
	sort.Strings(ids)
	if len(ids) != 19 {
		t.Fatalf("LPBS credits fixture discovered %d candidates, want 19", len(ids))
	}
	return ids
}

func setAC2DoorCheapestSupplier(t *testing.T, root, candidateID string) {
	t.Helper()
	foundTarget := false
	for candidateIndex, id := range ac2DoorCandidateIDs(t) {
		supplierID := "fixture-router-" + id
		path := filepath.Join(root, "resources", supplierID, "model-policy.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read candidate %q fixture policy: %v", id, err)
		}
		var document map[string]json.RawMessage
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("decode candidate %q fixture policy: %v", id, err)
		}
		var provider map[string]json.RawMessage
		if err := json.Unmarshal(document["provider"], &provider); err != nil {
			t.Fatalf("decode candidate %q provider policy: %v", id, err)
		}
		var routes []map[string]json.RawMessage
		if err := json.Unmarshal(provider["hosted_routes"], &routes); err != nil {
			t.Fatalf("decode candidate %q routes: %v", id, err)
		}
		for routeIndex, route := range routes {
			var supplier string
			if err := json.Unmarshal(route["supplier"], &supplier); err != nil {
				t.Fatalf("decode candidate %q route supplier: %v", id, err)
			}
			if supplier == "fixture-router-"+candidateID {
				foundTarget = true
			}
			quote := int64(2000 + candidateIndex)
			if id == candidateID {
				quote = 1000
			}
			route["quoted_credits"], _ = json.Marshal(quote)
			route["quote_ref"], _ = json.Marshal(fmt.Sprintf("lpbs-sweep-%s-%d", id, routeIndex))
		}
		provider["hosted_routes"], _ = json.Marshal(routes)
		document["provider"], _ = json.Marshal(provider)
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("encode candidate %q fixture policy: %v", id, err)
		}
		temporary := path + ".ac2-sweep"
		if err := os.WriteFile(temporary, encoded, 0o600); err != nil {
			t.Fatalf("write candidate %q fixture policy: %v", id, err)
		}
		if err := os.Rename(temporary, path); err != nil {
			t.Fatalf("replace candidate %q fixture policy: %v", id, err)
		}
	}
	if !foundTarget {
		t.Fatalf("candidate %q has no fixture route", candidateID)
	}
}

func setAC2DoorSupplierUnavailable(t *testing.T, root, candidateID string) {
	t.Helper()
	path := filepath.Join(root, "resources", "fixture-router-"+candidateID, "model-policy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read unavailable candidate %q fixture policy: %v", candidateID, err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode unavailable candidate %q fixture policy: %v", candidateID, err)
	}
	var provider map[string]json.RawMessage
	if err := json.Unmarshal(document["provider"], &provider); err != nil {
		t.Fatalf("decode unavailable candidate %q provider policy: %v", candidateID, err)
	}
	provider["hosted_routes"], _ = json.Marshal([]any{})
	document["provider"], _ = json.Marshal(provider)
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode unavailable candidate %q fixture policy: %v", candidateID, err)
	}
	temporary := path + ".ac2-unavailable"
	if err := os.WriteFile(temporary, encoded, 0o600); err != nil {
		t.Fatalf("write unavailable candidate %q fixture policy: %v", candidateID, err)
	}
	if err := os.Rename(temporary, path); err != nil {
		t.Fatalf("replace unavailable candidate %q fixture policy: %v", candidateID, err)
	}
}
