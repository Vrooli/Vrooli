package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"agent-manager/internal/supervision"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

const ExactSetupPath = "/api/v1/keyring/authority/setup"
const ExactChallengePath = "/api/v1/keyring/exact-challenge"
const ExactReadChallengePath = "/api/v1/keyring/exact-read-challenge"
const ExactReceiptPath = "/api/v1/keyring/exact-receipt"

// Normal wiring selects an installed-state verifier with no authority records.
// Neither registration method installs keys/grants or attaches a run.
func (h *AgentManagerConnectHandler) SetExactVerifier(v *supervision.ExactVerifier) {
	h.exactVerifier = v
}
func (h *AgentManagerConnectHandler) RegisterExactRoutes(router *mux.Router) {
	if h.exactVerifier == nil {
		return
	}
	router.HandleFunc(ExactSetupPath, h.exactSetup).Methods(http.MethodPost)
	router.HandleFunc(ExactSetupPath, h.exactSetupStatus).Methods(http.MethodGet)
	router.HandleFunc(ExactChallengePath, h.exactChallenge).Methods(http.MethodPost)
	router.HandleFunc(ExactReceiptPath, h.exactReceipt).Methods(http.MethodGet)
	router.HandleFunc(ExactReadChallengePath, h.exactReadChallenge).Methods(http.MethodPost)
	router.HandleFunc(ExactReceiptPath, h.exactAgentReceipt).Methods(http.MethodPost)
}
func (h *AgentManagerConnectHandler) exactChallenge(w http.ResponseWriter, r *http.Request) {
	if h.exactVerifier == nil {
		http.Error(w, "exact-action authority unavailable", 503)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, supervision.ExactMaxBytes+1))
	req := &pb.ReconcileEffortMetadataRequest{}
	if err != nil || len(raw) > supervision.ExactMaxBytes || protojson.Unmarshal(raw, req) != nil {
		http.Error(w, "bounded typed request required", 400)
		return
	}
	challenge, err := h.exactVerifier.Challenge(r.Context(), r.URL.Query().Get("grant"), req)
	if err != nil {
		http.Error(w, supervision.ErrExactAuthority.Error(), 403)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(challenge)
}
func (h *AgentManagerConnectHandler) exactReceipt(w http.ResponseWriter, r *http.Request) {
	// Site v1 signs mutations only. This existing human authority path does
	// not reinterpret a reconcile proof as read authority for another actor.
	if r.Header.Get(supervision.ExactProofHeader) != "" || r.Header.Get(supervision.ExactSignatureHeader) != "" {
		http.Error(w, supervision.ErrExactAuthority.Error(), 403)
		return
	}
	p, err := authn.RequireIndependentHuman(r.Context())
	if err != nil {
		http.Error(w, "verified independent operator required", 403)
		return
	}
	if _, err = authn.RequireCapability(r.Context(), "agent-manager:read"); err != nil {
		http.Error(w, "receipt read capability required", 403)
		return
	}
	s, err := h.effortService()
	if err != nil {
		http.Error(w, "effort service unavailable", 503)
		return
	}
	out, err := s.ReadExactReceipt(r.Context(), r.URL.Query().Get("key"), r.URL.Query().Get("effort"), p.Subject)
	if err != nil {
		http.Error(w, supervision.ErrExactAuthority.Error(), 403)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *AgentManagerConnectHandler) exactReadChallenge(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, supervision.ExactMaxBytes+1))
	req, decodeErr := supervision.DecodeExactReceiptRequest(raw)
	if err != nil || decodeErr != nil {
		http.Error(w, "bounded typed receipt request required", 400)
		return
	}
	challenge, err := h.exactVerifier.ReceiptChallenge(r.Context(), r.URL.Query().Get("grant"), req)
	if err != nil {
		http.Error(w, supervision.ErrExactAuthority.Error(), 403)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(challenge)
}
func (h *AgentManagerConnectHandler) exactAgentReceipt(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, supervision.ExactMaxBytes+1))
	req, decodeErr := supervision.DecodeExactReceiptRequest(raw)
	proof, sig := r.Header.Values(supervision.ExactProofHeader), r.Header.Values(supervision.ExactSignatureHeader)
	if err != nil || decodeErr != nil || len(proof) != 1 || len(sig) != 1 || h.exactVerifier == nil {
		http.Error(w, supervision.ErrExactAuthority.Error(), 403)
		return
	}
	ctx, err := h.exactVerifier.VerifyReceipt(r.Context(), req, proof[0], sig[0])
	if err != nil {
		http.Error(w, supervision.ErrExactAuthority.Error(), 403)
		return
	}
	service, err := h.effortService()
	if err != nil {
		http.Error(w, "effort service unavailable", 503)
		return
	}
	receipt, err := service.ReadExactReceiptProof(ctx, req)
	if err != nil {
		status := 403
		if err == supervision.ErrExactRetryable {
			status = 503
		}
		http.Error(w, supervision.ErrExactAuthority.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(receipt)
}

func (h *AgentManagerConnectHandler) exactSetup(w http.ResponseWriter, r *http.Request) {
	// Setup/approval must not borrow the human accompanying agent attribution.
	if len(r.Header.Values("X-Agent-Identity-Token")) != 0 {
		http.Error(w, "independent owner setup required", 403)
		return
	}
	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !authorizationOriginAllowed(u, r) {
			http.Error(w, "same-origin owner setup required", 403)
			return
		}
	} else if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		http.Error(w, "independent owner bearer required", 403)
		return
	}
	var req supervision.ExactSetupRequest
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, supervision.ExactMaxBytes))
	d.DisallowUnknownFields()
	if d.Decode(&req) != nil || d.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "bounded typed setup request required", 400)
		return
	}
	out, err := h.exactVerifier.Setup(r.Context(), req)
	if err != nil {
		status := 403
		if err == supervision.ErrExactRetryable {
			status = 503
		}
		http.Error(w, "exact setup refused; reconcile original policy identity", status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *AgentManagerConnectHandler) exactSetupStatus(w http.ResponseWriter, r *http.Request) {
	out, err := h.exactVerifier.Setup(r.Context(), supervision.ExactSetupRequest{Action: "status", Grant: r.URL.Query().Get("grant")})
	if err != nil {
		status := 403
		if err == supervision.ErrExactRetryable {
			status = 503
		}
		http.Error(w, "independent owner status required", status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}
