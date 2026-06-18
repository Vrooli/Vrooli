package channel

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"vrooli-bridge/internal/httpx"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"
)

// errMissingNodeID is returned when a node opens the channel or heartbeats
// without identifying itself.
var errMissingNodeID = errors.New("a node id is required")

// channelKeepalive is how often the held SSE stream emits a comment ping so
// intermediaries keep the connection warm and a dead connection is noticed.
const channelKeepalive = 25 * time.Second

// sseDeps wires the SSE dial-out handler.
type sseDeps struct {
	Hub *presence.Hub
	// Verifier, when set, requires the dial-out to present a valid ?token=
	// signed by the node's credential before the stream is held. Nil disables
	// enforcement (the ?node= stub, Phase-1).
	Verifier *nodeauth.Verifier
	Logger   *log.Logger
}

type sseHandler struct {
	deps sseDeps
}

func newSSEHandler(d sseDeps) *sseHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &sseHandler{deps: d}
}

// nodeIDFrom extracts the node's identity from the request. EventSource cannot
// set headers, so the dial-out channel carries it as a ?node= query param (the
// X-Bridge-Node header is accepted for non-browser clients). This is the Phase
// 1 STUB credential — Phase 2 replaces it with a per-node Ed25519-signed token
// the control plane verifies (SECURITY.md boundary 2).
func nodeIDFrom(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Bridge-Node")); v != "" {
		return v
	}
	return strings.TrimSpace(r.URL.Query().Get("node"))
}

// handleEvents is the dial-out channel: the node opens this stream and holds it
// open. The node is online for as long as the stream is held; when it
// disconnects (or the control plane shuts down) the node flips offline. This is
// the NAT/firewall-proof half — the node always initiates, no inbound port.
func (h *sseHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	var nodeID string
	if h.deps.Verifier != nil {
		// Mutual auth: the dial-out must carry a valid signed ?token=. The
		// verified node id (not the client-asserted one) drives presence.
		proof, err := nodeauth.ParseToken(r.URL.Query().Get("token"))
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "a valid dial-out token is required (?token=)")
			return
		}
		if err := h.deps.Verifier.VerifyProof(r.Context(), proof); err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "dial-out token rejected")
			return
		}
		nodeID = proof.NodeID
	} else {
		nodeID = nodeIDFrom(r)
	}
	if nodeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "a node id is required (?node= or X-Bridge-Node)")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	conn := h.deps.Hub.Connect(nodeID)
	defer conn.Close()

	// Greet the node so a just-connected client renders state without waiting,
	// and prove the stream is live.
	if _, err := w.Write([]byte(": connected\n\n")); err != nil {
		return
	}
	flusher.Flush()

	keepalive := time.NewTicker(channelKeepalive)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-conn.Done():
			// The control plane severed this channel (e.g. atomic revocation):
			// stop holding the stream immediately, don't wait for keepalive.
			return
		case payload := <-conn.Out():
			// A control-plane → node push (JobPush in Phase 3, ProvisionCommand
			// in Phase 4): one already-serialised channel.ServerFrame, written
			// as a single SSE `data:` event the agent decodes with
			// DiscardUnknown. Newlines in the JSON would break SSE framing, but
			// protojson emits compact single-line JSON, so one data: line is
			// correct; we still guard by writing the payload as-is.
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		case <-keepalive.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
