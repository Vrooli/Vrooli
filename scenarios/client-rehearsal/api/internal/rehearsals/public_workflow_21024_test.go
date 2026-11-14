package rehearsals

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestPublic21024FailsClosedBeforeRunCredit(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/21024.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "00c63ca77cbe0631d55b19e8633a4ee704b0a09d48ec28dd151316e92db34cfa" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	if strings.Contains(workflowJSON, `"credentials"`) {
		t.Fatal("sanitized graph retained credential objects")
	}
	for _, match := range regexp.MustCompile(`https?://([^/\s"'<>]+)`).FindAllStringSubmatch(workflowJSON, -1) {
		if len(match) < 2 || !strings.HasSuffix(match[1], ".synthetic.invalid") {
			t.Fatalf("sanitized graph retained a non-synthetic URL host: %q", match)
		}
	}
	_, _, _, err := normalizeWorkflow(workflowJSON)
	if err == nil {
		t.Fatal("source workflow unexpectedly imports; add synthetic profiles and prove a complete simulator run before claiming credit")
	}
	if strings.Contains(err.Error(), "Code node is outside the bounded safe subset") {
		t.Fatalf("generic Code interpreter blockers should be fixed or accurately isolated before retaining this candidate: %v", err)
	}
	t.Logf("zero public run credit: full import remains blocked before profile-backed execution: %v", err)
}

func TestPublic21024CodeSubsetProgressIsCountedWithoutRunCredit(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/21024.json")
	var workflow struct {
		Nodes []struct {
			Name       string `json:"name"`
			Type       string `json:"type"`
			Parameters struct {
				JS string `json:"jsCode"`
			} `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(workflowJSON), &workflow); err != nil {
		t.Fatal(err)
	}
	blocked := map[string]string{"Sum up the week": "arrow functions support one named parameter and an expression body"}
	codeCount, parsedCount := 0, 0
	for _, node := range workflow.Nodes {
		if !strings.HasSuffix(node.Type, ".code") {
			continue
		}
		codeCount++
		_, err := parseSafeCode(node.Parameters.JS)
		if err == nil {
			parsedCount++
			if _, expected := blocked[node.Name]; expected {
				t.Errorf("known blocked node %q now parses; update the recorded subset coverage", node.Name)
			}
			continue
		}
		expected, ok := blocked[node.Name]
		if !ok || !strings.Contains(err.Error(), expected) {
			t.Errorf("unexpected Code blocker for %q: %v", node.Name, err)
		}
	}
	if codeCount != 9 || parsedCount != 8 {
		t.Fatalf("safe subset coverage = %d/%d Code nodes; want 8/9", parsedCount, codeCount)
	}
}
