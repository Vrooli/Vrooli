package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const AcceptanceContractEndpoint = "/internal/finite-effort/v1/acceptance/check"

// CommissionSubject is the authored work, independent of a finite credential,
// policy lifetime or completion disposition. Every field is owner-approved.
type CommissionSubject struct {
	Owner         string            `json:"owner"`
	Effort        string            `json:"effort"`
	Revision      string            `json:"revision"`
	ContentDigest string            `json:"contentDigest"`
	Team          string            `json:"team"`
	Members       []string          `json:"members"`
	Repository    string            `json:"repository"`
	TeamDigest    string            `json:"teamDigest"`
	BindingDigest string            `json:"bindingDigest"`
	Profiles      map[string]string `json:"profiles"`
}

func CommissionSubjectFor(p Policy) CommissionSubject {
	return CommissionSubject{p.Owner, p.Effort, p.Revision, p.ContentDigest, p.Team, append([]string(nil), p.Members...), p.Repository, p.TeamDigest, p.BindingDigest, cloneProfiles(p.Profiles)}
}
func cloneProfiles(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func (s CommissionSubject) Validate() error {
	if s.Owner == "" || s.Effort == "" || s.Revision == "" || s.Team == "" || (!filepath.IsAbs(s.Repository) || filepath.Clean(s.Repository) != s.Repository || strings.ContainsAny(s.Repository, "\x00\r\n")) || !validDigest(s.ContentDigest) || !validDigest(s.TeamDigest) || !validDigest(s.BindingDigest) || len(s.Members) == 0 || len(s.Profiles) == 0 {
		return ErrRefused
	}
	seen := map[string]bool{}
	for _, v := range s.Members {
		if v == "" || seen[v] {
			return ErrRefused
		}
		seen[v] = true
	}
	for k, v := range s.Profiles {
		if k == "" || !validDigest(v) {
			return ErrRefused
		}
	}
	return nil
}

type AcceptanceContract struct {
	Subject    CommissionSubject `json:"subject"`
	Actor      string            `json:"actor"`
	AcceptedAt time.Time         `json:"acceptedAt"`
	Generation uint64            `json:"generation"`
	Revoked    bool              `json:"revoked"`
}
type AcceptanceContractOwner interface {
	ReadFiniteAcceptance(context.Context, Policy) (AcceptanceContract, error)
}
type AcceptanceContractGrant struct {
	Policy Policy `json:"policy"`
}
type acceptanceCheckRequest struct {
	Binding Binding `json:"binding"`
}

// AcceptanceContractHandler is unmounted until exact private owner startup.
// The verified peer may only read the full immutable installed commission.
type AcceptanceContractHandler struct {
	owner AcceptanceContractOwner
	peers map[string]map[string]AcceptanceContractGrant
}

func NewAcceptanceContractHandler(owner AcceptanceContractOwner, peers map[string]map[string]AcceptanceContractGrant) (*AcceptanceContractHandler, error) {
	if owner == nil || len(peers) == 0 {
		return nil, ErrRefused
	}
	raw, e := json.Marshal(peers)
	if e != nil {
		return nil, ErrRefused
	}
	copy := map[string]map[string]AcceptanceContractGrant{}
	if json.Unmarshal(raw, &copy) != nil {
		return nil, ErrRefused
	}
	for pin, grants := range copy {
		if !validDigest(pin) || len(grants) == 0 {
			return nil, ErrRefused
		}
		for id, g := range grants {
			if g.Policy.ID != id || g.Policy.Validate() != nil || CommissionSubjectFor(g.Policy).Validate() != nil {
				return nil, ErrRefused
			}
		}
	}
	return &AcceptanceContractHandler{owner, copy}, nil
}
func (h *AcceptanceContractHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	refuse := func() { http.Error(w, "finite acceptance refused", http.StatusForbidden) }
	if h == nil || r.Method != "POST" || r.URL.Path != AcceptanceContractEndpoint || r.URL.RawQuery != "" || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		refuse()
		return
	}
	for _, k := range []string{"Authorization", "Cookie", "X-Agent-Identity-Token", Header} {
		if len(r.Header.Values(k)) != 0 {
			refuse()
			return
		}
	}
	raw, e := io.ReadAll(io.LimitReader(r.Body, 8193))
	if e != nil || len(raw) > 8192 {
		refuse()
		return
	}
	var in acceptanceCheckRequest
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(new(any)) != io.EOF {
		refuse()
		return
	}
	pin := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	g, ok := h.peers[hex.EncodeToString(pin[:])][in.Binding.PolicyID]
	if !ok || !sameProfileBinding(binding(g.Policy), in.Binding) || time.Now().Before(g.Policy.NotBefore) || !time.Now().Before(g.Policy.Deadline) {
		refuse()
		return
	}
	out, e := h.owner.ReadFiniteAcceptance(r.Context(), g.Policy)
	if e != nil || out.Revoked || out.Actor != g.Policy.Owner || out.AcceptedAt.IsZero() || out.AcceptedAt.After(time.Now()) || out.Generation == 0 || Digest(out.Subject) != Digest(CommissionSubjectFor(g.Policy)) {
		refuse()
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}
