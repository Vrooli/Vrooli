package rehearsals

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestSelectedPublicWorkflowCatalogRetainsImportOutcomesAndTriggerFamilies(t *testing.T) {
	ids := []string{
		"1110", "1804", "2385", "2713", "5333", "5852", "5889", "6338", "6920", "7032",
		"7950", "8258", "15238", "15273", "16079", "16081", "16364", "16366", "16667", "16904",
		"17164", "17831", "20047", "20743", "20819", "21013", "21024", "21098", "22436", "22567",
	}
	if len(ids) != 30 {
		t.Fatalf("selected public catalog contains %d IDs, want 30", len(ids))
	}
	seenIDs, triggerKinds := map[string]bool{}, map[TriggerKind]bool{}
	importable, typedRejected := 0, 0
	for _, id := range ids {
		if seenIDs[id] {
			t.Fatalf("selected public workflow ID %q is duplicated", id)
		}
		seenIDs[id] = true
		workflowJSON := readFixture(t, "public-workflows/"+id+".json")
		var raw struct {
			Nodes []struct {
				Type string `json:"type"`
			} `json:"nodes"`
			Workflows []struct {
				Nodes []struct {
					Type string `json:"type"`
				} `json:"nodes"`
			} `json:"workflows"`
		}
		if err := json.Unmarshal([]byte(workflowJSON), &raw); err != nil {
			t.Errorf("selected public workflow %s has no readable node list: %v", id, err)
			continue
		}
		nodeSets := [][]struct {
			Type string `json:"type"`
		}{raw.Nodes}
		for _, child := range raw.Workflows {
			nodeSets = append(nodeSets, child.Nodes)
		}
		nodeCount := 0
		for _, nodes := range nodeSets {
			for _, node := range nodes {
				nodeCount++
				if kind, ok := classifyTrigger(node.Type); ok {
					triggerKinds[kind] = true
				}
			}
		}
		if nodeCount == 0 {
			t.Errorf("selected public workflow %s has no readable node list", id)
			continue
		}
		if _, _, _, err := normalizeWorkflow(workflowJSON); err != nil {
			var invalid ErrInvalid
			if !errors.As(err, &invalid) || invalid.Field == "" || invalid.Reason == "" {
				t.Errorf("public workflow %s failure is not actionable: %v", id, err)
				continue
			}
			typedRejected++
		} else {
			importable++
		}
	}
	for _, kind := range []TriggerKind{TriggerManual, TriggerWebhook, TriggerPolling, TriggerScheduled, TriggerError, TriggerEvent, TriggerSubworkflow} {
		if !triggerKinds[kind] {
			t.Errorf("selected public corpus lost trigger family %q: observed %v", kind, triggerKinds)
		}
	}
	if importable+typedRejected != len(ids) {
		t.Fatalf("catalog outcome denominator=%d, want 30 (importable=%d typed rejections=%d)", importable+typedRejected, importable, typedRejected)
	}
	t.Logf("selected catalog: %d attributed IDs, %d source-normalized, %d typed source rejections, trigger families=%v", len(ids), importable, typedRejected, triggerKinds)
}
