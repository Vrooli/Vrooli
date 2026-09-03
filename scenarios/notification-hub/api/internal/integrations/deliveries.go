package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"notification-hub/internal/hub"
)

// DeliveryProjectionHandler serves the durable delivery projection to the
// coverage checks of other scenarios: GET ?prefix=<dedupe prefix>&limit=N.
func DeliveryProjectionHandler(service *hub.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := service.ListDeliveryProjection(r.Context(), strings.TrimSpace(r.URL.Query().Get("prefix")), limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if items == nil {
			items = []hub.DeliveryProjection{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"notifications": items})
	})
}

// operatorStateFile is the control plane's operator state, relative to the
// repository root. The hub is outside the control-plane module and cannot
// import internal/operatorstate, so it reads the one field it owns the
// meaning of; the field is declared in .vrooli/schemas/operator-state.schema.json.
const operatorStateFile = ".vrooli/operator-state.json"

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
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return filepath.Join(root, filepath.FromSlash(operatorStateFile))
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, filepath.FromSlash(operatorStateFile))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
