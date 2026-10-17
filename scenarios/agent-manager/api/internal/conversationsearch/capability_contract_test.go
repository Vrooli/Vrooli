package conversationsearch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversationRecallCapabilityLayers(t *testing.T) {
	t.Parallel()
	scenarioRoot := filepath.Join("..", "..", "..")
	usage := readCapabilityFile(t, filepath.Join(scenarioRoot, "skills", "agent-manager", "SKILL.md"))
	for _, expected := range []string{
		"unknown run id", "agent-manager conversation search", "agent-manager conversation context",
		"copied text", "provenance", "agent-manager.conversation-recall", "do not paste snippets",
	} {
		require.Contains(t, strings.ToLower(usage), strings.ToLower(expected))
	}
	improve := readCapabilityFile(t, filepath.Join(scenarioRoot, "skills", "agent-manager-improve", "SKILL.md"))
	for _, expected := range []string{
		"agent-manager conversation index status", "search-hub evals runs", "agent-manager.conversation-recall",
		"usage-judgment defect", "governed orchestration", "meta optimization manager",
	} {
		require.Contains(t, strings.ToLower(improve), strings.ToLower(expected))
	}

	var contract struct {
		Name   string `json:"name"`
		Inputs map[string]struct {
			Default any      `json:"default"`
			Enum    []string `json:"enum"`
		} `json:"inputs"`
		Invariants []string `json:"invariants"`
		Bindings   []struct {
			ID     string `json:"id"`
			Effect string `json:"effect"`
		} `json:"bindings"`
		Budget struct {
			InferenceCalls int `json:"inference_calls"`
			DelegatedRuns  int `json:"delegated_runs"`
			Materialize    int `json:"materialize_limit"`
			OutputBytes    int `json:"output_bytes"`
		} `json:"budget"`
		Fixtures []struct {
			ID string `json:"id"`
		} `json:"fixtures"`
	}
	content := readCapabilityFile(t, filepath.Join(scenarioRoot, ".vrooli", "program-runtime", "conversation-recall.json"))
	require.NoError(t, json.Unmarshal([]byte(content), &contract))
	require.Equal(t, "agent-manager.conversation-recall", contract.Name)
	require.Equal(t, []string{"metadata", "snippets", "context"}, contract.Inputs["output_mode"].Enum)
	require.Zero(t, contract.Budget.InferenceCalls)
	require.Zero(t, contract.Budget.DelegatedRuns)
	require.LessOrEqual(t, contract.Budget.Materialize, 20)
	require.LessOrEqual(t, contract.Budget.OutputBytes, 65536)
	require.GreaterOrEqual(t, len(contract.Fixtures), 4)
	for _, binding := range contract.Bindings {
		require.Equal(t, "read", binding.Effect)
	}
	require.Contains(t, strings.Join(contract.Invariants, "\n"), "raw clues are never echoed")
	require.Contains(t, strings.Join(contract.Invariants, "\n"), "three context calls")
}

func readCapabilityFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
