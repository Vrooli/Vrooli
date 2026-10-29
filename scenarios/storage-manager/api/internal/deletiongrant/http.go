package deletiongrant

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/provenance"
)

type requestBody struct {
	Paths     []string `json:"paths"`
	Reason    string   `json:"reason"`
	ExpiresIn string   `json:"expires_in"` // Go duration, e.g. "24h"; blank uses the default
	// WebConsoleSessionID, when supplied by a web-console-hosted agent (its own
	// WC_WEB_CONSOLE_SESSION_ID), is the stable delivery handle for the answer
	// tie-back — used when the ephemeral identity token can't identify a run.
	WebConsoleSessionID string `json:"web_console_session_id"`
}

// webConsoleSessionIDHeader mirrors cliutil.WebConsoleSessionIDHeader; a header
// is accepted as an alternative to the body field.
const webConsoleSessionIDHeader = "X-Wc-Session-Id"

// RequestHandler is the endpoint an agent calls to ask an operator for a
// time-boxed deletion grant. It raises an operatordecision ask (surfaced in the
// web-console inbox) and answers "pending"; it never grants authority — only the
// operator's approval, delivered to the resolution webhook, does. 202 means the
// ask is outstanding; 429 means too many are already pending for the caller.
func RequestHandler(raiser DecisionRaiser, protect Protection) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body requestBody
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		ttl := defaultGrantTTL
		if body.ExpiresIn != "" {
			if parsed, err := time.ParseDuration(body.ExpiresIn); err == nil && parsed > 0 {
				ttl = parsed
			}
		}
		// Prefer the shared agent-provenance identity: for a coding agent it
		// yields a verified agent-manager run id (attributed to ANY coding agent,
		// even ones agent-manager did not create, by verifying the agent's token
		// against agent-manager). That run id is what lets agent-manager deliver
		// the answer back automatically. Fall back to the browser principal for a
		// human caller (no tie-back, but still attributed for the flood cap).
		requestedBy := ""
		raisedByRunID := ""
		if prov := provenance.FromContext(r.Context()); prov.IsVerifiedAgent() {
			_, _, _, _, runID, _ := prov.WriteFields()
			raisedByRunID = runID
			requestedBy = runID
		} else if principal, ok := identity.PrincipalFromContext(r.Context()); ok {
			requestedBy = principal.Subject
		}
		// Stable delivery handle for a web-console-hosted agent: body field first,
		// then the X-Wc-Session-Id header.
		wcSessionID := strings.TrimSpace(body.WebConsoleSessionID)
		if wcSessionID == "" {
			wcSessionID = strings.TrimSpace(r.Header.Get(webConsoleSessionIDHeader))
		}
		id, raised, err := RequestProtected(r.Context(), raiser, protect, body.Paths, requestedBy, raisedByRunID, wcSessionID, body.Reason, ttl)
		switch {
		case err == operatordecision.ErrFlood:
			http.Error(w, "too many outstanding deletion-grant requests; wait for the operator to decide", http.StatusTooManyRequests)
			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case !raised:
			http.Error(w, "deletion-grant approvals are not configured on this host", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "pending",
			"decision_id": id,
			"paths":       normalizePaths(body.Paths),
			"message":     "operator approval requested; the deletion is permitted once approved in the decision inbox",
			// An agent-manager-managed agent can park on this decision to burn zero
			// tokens while waiting and be woken automatically with the answer. The
			// await key is the decision id; the producer is "operator-decision".
			"await": map[string]any{
				"producer":      "operator-decision",
				"key":           id,
				"park_command":  "agent-manager run park <run-id> --producer operator-decision --key " + id + " --timeout 12h",
				"park_optional": true,
				"note":          "Optional: only agent-manager-managed runs can park/be woken. Without parking, re-check the grant before deleting.",
			},
		})
	})
}
