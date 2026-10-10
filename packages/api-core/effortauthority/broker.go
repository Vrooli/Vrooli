package effortauthority

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const BrokerEndpoint = "/internal/finite-effort/v1/transaction/"

type BrokerGrant struct {
	PolicyID   string
	Operations map[string]bool
}

// BrokerHandler is mounted only on a separately protected issuer listener.
// RequireAndVerifyClientCert and exact enrolled peer/policy/operation pins are
// mandatory. Native process UID access to custody/ledger must be denied by host
// isolation; localhost and file mode alone are not this boundary.
type BrokerHandler struct {
	Authority *Authority
	Custody   *CustodySigner
	Peers     map[string]map[string]BrokerGrant
}
type transaction struct {
	Payload json.RawMessage `json:"payload,omitempty"`
	Binding Binding         `json:"binding"`
	Intent  Intent          `json:"intent"`
	Proof   Proof           `json:"proof"`
	Ingress *Proof          `json:"ingress,omitempty"`
	Key     string          `json:"key"`
	RunID   string          `json:"runId"`
}
type transactionResult struct {
	Policy      Policy         `json:"policy"`
	Binding     Binding        `json:"binding"`
	Proof       Proof          `json:"proof"`
	Reservation Reservation    `json:"reservation"`
	Replay      bool           `json:"replay"`
	Handoff     *SerialHandoff `json:"handoff,omitempty"`
}

func (h *BrokerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	refuse := func() { http.Error(w, "finite transaction refused", http.StatusForbidden) }
	if h == nil || h.Authority == nil || r.Method != "POST" || r.URL.RawQuery != "" || !strings.HasPrefix(r.URL.Path, BrokerEndpoint) || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		refuse()
		return
	}
	for _, header := range []string{"Authorization", "Cookie", "X-Agent-Identity-Token", Header} {
		if len(r.Header.Values(header)) != 0 {
			refuse()
			return
		}
	}
	op := strings.TrimPrefix(r.URL.Path, BrokerEndpoint)
	var in transaction
	body, readErr := io.ReadAll(io.LimitReader(r.Body, 32769))
	if readErr != nil || len(body) > 32768 {
		refuse()
		return
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(new(any)) != io.EOF {
		refuse()
		return
	}
	id := in.Binding.PolicyID
	if op == "check-proof" || op == "prepare" || op == "observe-ingress" {
		id = in.Proof.Intent.PolicyID
	}
	if op == "sign" {
		id = in.Intent.PolicyID
	}
	pin := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	grant, exists := h.Peers[hex.EncodeToString(pin[:])][id]
	if !exists || grant.PolicyID != id || !grant.Operations[op] {
		refuse()
		return
	}
	// Current purpose revocation closes new admission even for a still-fresh
	// previously signed proof. Exact already-owned read/settlement operations
	// retain their own owner checks and do not require a usable signing key.
	if h.preflightCustody(r.Context(), op, in) != nil {
		refuse()
		return
	}
	var out transactionResult
	var err error
	switch op {
	case "check-binding":
		out.Policy, err = h.Authority.CheckBinding(r.Context(), in.Binding)
	case "check-proof":
		out.Binding, err = h.Authority.CheckProof(r.Context(), in.Proof)
	case "prepare":
		out.Reservation, out.Replay, err = h.Authority.Prepare(r.Context(), in.Proof, in.Ingress)
	case "reserve":
		out.Reservation, out.Replay, err = h.Authority.Reserve(r.Context(), in.Binding, in.Intent, in.Ingress)
	case "prepare-serial-handoff":
		var handoff SerialHandoff
		if len(in.Payload) > 0 {
			handoff, out.Replay, err = h.Authority.PrepareSerialHandoffWithPayload(r.Context(), in.Binding, in.Key, in.Intent, in.Payload)
		} else {
			handoff, out.Replay, err = h.Authority.PrepareSerialHandoff(r.Context(), in.Binding, in.Key, in.Intent)
		}
		if err == nil {
			out.Handoff = &handoff
		}
	case "reserve-serial-episode":
		out.Reservation, out.Replay, err = h.Authority.ReserveSerialEpisode(r.Context(), in.Binding, in.Key)
	case "read-serial-handoff":
		var handoff SerialHandoff
		handoff, err = h.Authority.ReadSerialHandoff(r.Context(), in.Binding, in.Key)
		if err == nil {
			out.Handoff = &handoff
		}
	case "accept":
		err = h.Authority.Accept(r.Context(), in.Binding, in.Key, in.RunID)
	case "settle":
		err = h.Authority.Settle(r.Context(), in.Binding, in.Key, in.RunID)
	case "observe-ingress":
		out.Reservation, out.Replay, err = h.Authority.ObserveIngress(r.Context(), in.Proof)
	case "read-reservation":
		out.Reservation, err = h.Authority.ReadReservation(r.Context(), in.Binding, in.Key)
	case "sign":
		var p Policy
		p, err = h.Authority.CheckBinding(r.Context(), in.Binding)
		if err == nil {
			err = checkIntent(p, in.Intent)
		}
		if err == nil && (h.Custody == nil || in.Intent.Effect != "run.create" || in.Intent.PolicyID != in.Binding.PolicyID) {
			err = ErrRefused
		}
		if err == nil {
			err = h.Custody.CheckPolicy(r.Context(), p)
		}
		if err == nil {
			// Proof lifetime and nonce are issuer-owned; arbitrary bytes are never signed.
			nonce := make([]byte, 32)
			if _, err = rand.Read(nonce); err == nil {
				out.Proof, err = h.Custody.SignPolicyProof(r.Context(), p, Proof{Intent: in.Intent, Nonce: hex.EncodeToString(nonce), IssuedAt: h.Authority.now()})
			}
		}
	default:
		err = ErrRefused
	}
	if err != nil {
		refuse()
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *BrokerHandler) preflightCustody(ctx context.Context, op string, in transaction) error {
	var p Policy
	var e error
	switch op {
	case "check-binding":
		p, e = h.Authority.CheckBinding(ctx, in.Binding)
	case "check-proof", "prepare", "observe-ingress":
		var b Binding
		b, e = h.Authority.CheckProof(ctx, in.Proof)
		if e == nil {
			p, e = h.Authority.CheckBinding(ctx, b)
		}
	case "prepare-serial-handoff", "reserve-serial-episode":
		p, e = h.Authority.CheckBinding(ctx, in.Binding)
	case "reserve":
		p, e = h.Authority.CheckBinding(ctx, in.Binding)
		if e == nil {
			e = checkIntent(p, in.Intent)
		}
		if e == nil && in.Ingress != nil {
			var b Binding
			b, e = h.Authority.CheckProof(ctx, *in.Ingress)
			if e == nil && !sameProfileBinding(b, in.Binding) {
				e = ErrRefused
			}
		}
	default:
		return nil
	}
	if e != nil || h.Custody == nil {
		return ErrRefused
	}
	return h.Custody.CheckPolicy(ctx, p)
}
