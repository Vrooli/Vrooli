package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"notification-hub/internal/hub"

	"github.com/vrooli/api-core/storage"
)

// DeliveryProjectionHandler serves the durable delivery projection to the
// coverage checks of other scenarios: GET ?prefix=<dedupe prefix>&incident_id=ID&limit=N.
func DeliveryProjectionHandler(service *hub.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		incidentIDs, err := boundedIncidentIDs(r.URL.Query()["incident_id"])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		items, truncated, err := service.ListDeliveryProjection(r.Context(), strings.TrimSpace(r.URL.Query().Get("prefix")), incidentIDs, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if items == nil {
			items = []hub.DeliveryProjection{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"notifications": items, "truncated": truncated})
	})
}

func boundedIncidentIDs(values []string) ([]string, error) {
	if len(values) > 200 {
		return nil, fmt.Errorf("at most 200 incident_id values are allowed")
	}
	ids := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" || len(id) > 200 {
			return nil, fmt.Errorf("incident_id must be non-empty and at most 200 characters")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

// The hub is outside the control-plane module and cannot import
// internal/operatorstate, so it reads the one field it owns the meaning of
// through the same contract-routed state namespace.

// OperatorStateRecipient reads notifications.recipient from operator state
// at call time, so a recipient set after the hub started is honored on the
// next event. The root is VROOLI_ROOT, else the nearest ancestor of the
// working directory that holds the state file.
func OperatorStateRecipient() func(context.Context) string {
	return func(context.Context) string {
		path := operatorStatePath()
		if path == "" {
			return ""
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		var doc struct {
			Notifications struct {
				Recipient string `json:"recipient"`
			} `json:"notifications"`
		}
		if json.Unmarshal(data, &doc) != nil {
			return ""
		}
		return strings.TrimSpace(doc.Notifications.Recipient)
	}
}

func operatorStatePath() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return ""
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		return ""
	}
	return filepath.Join(paths.StateDir, "operator-state.json")
}
