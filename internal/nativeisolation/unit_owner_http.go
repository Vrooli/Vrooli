// unit_owner_http.go exposes authenticated control of retained fixed unit selectors only.
package nativeisolation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"io"
	"net/http"
)

const UnitOwnerEndpoint = "/internal/finite-effort/v1/unit/control"

type unitControl struct {
	Operation string `json:"operation"`
	Selector  string `json:"selector"`
}
type FixedUnitOwnerHandler struct{ owner *FixedUnitOwner }

// This adapter is control-only. It does not register plans, accept commands,
// stream stdin/models or expose PID/property/unit selectors. Native start over
// HTTP always refuses; actual native streaming uses the private local driver.
func NewFixedUnitOwnerHandler(owner *FixedUnitOwner) (*FixedUnitOwnerHandler, error) {
	if owner == nil || len(owner.peers) == 0 {
		return nil, ErrRefused
	}
	return &FixedUnitOwnerHandler{owner}, nil
}
func (h *FixedUnitOwnerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	refuse := func() { http.Error(w, "finite unit operation refused", http.StatusForbidden) }
	w.Header().Set("Cache-Control", "no-store")
	if h == nil || h.owner == nil || r.Method != "POST" || r.URL.Path != UnitOwnerEndpoint || r.URL.RawQuery != "" || r.TLS == nil || len(r.TLS.PeerCertificates) == 0 || len(r.TLS.VerifiedChains) == 0 {
		refuse()
		return
	}
	for _, header := range []string{"Authorization", "Cookie", "X-Agent-Identity-Token", effortauthority.Header} {
		if len(r.Header.Values(header)) != 0 {
			refuse()
			return
		}
	}
	pin := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	role, ok := h.owner.peers[hex.EncodeToString(pin[:])]
	if !ok {
		refuse()
		return
	}
	var request unitControl
	raw, e := io.ReadAll(io.LimitReader(r.Body, 4097))
	if e != nil || len(raw) > 4096 {
		refuse()
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(new(any)) != io.EOF || !validHash(request.Selector) {
		refuse()
		return
	}
	h.owner.mu.Lock()
	unit := h.owner.units[request.Selector]
	h.owner.mu.Unlock()
	if unit == nil || (unit.probe == nil && role != NativeUnitPeer) || (unit.probe != nil && ((unit.probe.Purpose == FirstQualification && role != BootstrapUnitPeer) || (unit.probe.Purpose == ContinuingWitness && role != WitnessUnitPeer))) {
		refuse()
		return
	}
	switch request.Operation {
	case "probe-start":
		if unit.probe == nil {
			refuse()
			return
		}
		if _, e := h.owner.start(r.Context(), request.Selector, nil); e != nil {
			refuse()
			return
		}
	case "read":
		state, e := h.owner.read(r.Context(), request.Selector)
		if e != nil {
			refuse()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
		return
	case "term", "kill":
		if h.owner.stop(r.Context(), request.Selector, request.Operation == "kill") != nil {
			refuse()
			return
		}
	default:
		refuse()
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
