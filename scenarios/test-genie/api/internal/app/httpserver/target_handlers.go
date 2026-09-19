package httpserver

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// handleListTargets is the target-browser read model. It is deliberately
// contract-backed: the UI and API expose the same enumerable identity that the
// executor resolves, rather than maintaining a second scenario-only catalog.
func (s *Server) handleListTargets(w http.ResponseWriter, _ *http.Request) {
	root := strings.TrimSpace(s.repoRoot)
	if root == "" && s.scenarios != nil {
		root = filepath.Dir(s.scenarios.ScenarioRoot())
	}
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	targets, err := contract.EnumerateTargets(root)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]string, 0, len(targets))
	for _, target := range targets {
		items = append(items, map[string]string{
			"kind":   string(target.Kind),
			"id":     target.ID,
			"root":   target.Root,
			"target": string(target.Kind) + ":" + target.ID,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "count": len(items)})
}
