package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const ScopeContractEndpoint = "/internal/finite-effort/v1/scope/check"
const CommissionScopeEndpoint = "/internal/finite-effort/v1/scope/commission/check"

type scopeCheckResult struct {
	PolicyDigest string `json:"policyDigest"`
}

// ScopeContractHandler reads the actual PM team/member/heartbeat owner. It
// accepts only a full frozen installed policy over verified private mTLS.
// Its response is a scope check, never an acceptance or completion record.
type ScopeContractHandler struct {
	scope    ScopeOwner
	endpoint string
	peers    map[string]map[string]AcceptanceContractGrant
}
type unboundAcceptanceOwner struct{}

func (unboundAcceptanceOwner) ReadFiniteAcceptance(context.Context, Policy) (AcceptanceContract, error) {
	return AcceptanceContract{}, ErrRefused
}
func NewScopeContractHandler(owner ScopeOwner, peers map[string]map[string]AcceptanceContractGrant) (*ScopeContractHandler, error) {
	if owner == nil {
		return nil, ErrRefused
	}
	frozen, e := NewAcceptanceContractHandler(unboundAcceptanceOwner{}, peers)
	if e != nil {
		return nil, e
	}
	return &ScopeContractHandler{scope: owner, endpoint: ScopeContractEndpoint, peers: frozen.peers}, nil
}
func (h *ScopeContractHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	refuse := func() { http.Error(w, "finite scope refused", http.StatusForbidden) }
	if h == nil || h.scope == nil || r.Method != "POST" || r.URL.Path != h.endpoint || r.URL.RawQuery != "" || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		refuse()
		return
	}
	for _, key := range []string{"Authorization", "Cookie", "X-Agent-Identity-Token", Header} {
		if len(r.Header.Values(key)) != 0 {
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
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(new(any)) != io.EOF {
		refuse()
		return
	}
	pin := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	grant, ok := h.peers[hex.EncodeToString(pin[:])][in.Binding.PolicyID]
	if !ok || !sameProfileBinding(binding(grant.Policy), in.Binding) || time.Now().Before(grant.Policy.NotBefore) || !time.Now().Before(grant.Policy.Deadline) {
		refuse()
		return
	}
	if h.scope.CheckEffortScope(r.Context(), grant.Policy) != nil {
		refuse()
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(scopeCheckResult{Digest(grant.Policy)})
}

type ScopeContractClient struct {
	config   AcceptanceContractClientConfig
	endpoint string
}

func NewScopeContractClient(cfg AcceptanceContractClientConfig) (*ScopeContractClient, error) {
	original := cfg.URL
	u, e := parseContractURL(original, ScopeContractEndpoint)
	if e != nil {
		return nil, e
	}
	u.Path = AcceptanceContractEndpoint
	cfg.URL = u.String()
	a, e := NewAcceptanceContractClient(cfg)
	if e != nil {
		return nil, e
	}
	a.config.URL = original
	return &ScopeContractClient{config: a.config, endpoint: ScopeContractEndpoint}, nil
}
func (c *ScopeContractClient) CheckEffortScope(ctx context.Context, p Policy) error {
	if c == nil || Digest(p) != Digest(c.config.Policy) || time.Now().Before(p.NotBefore) || !time.Now().Before(p.Deadline) {
		return ErrRefused
	}
	var out scopeCheckResult
	if privateContractRead(ctx, c.config, c.endpoint, acceptanceCheckRequest{c.config.Binding}, &out) != nil || out.PolicyDigest != Digest(p) {
		return ErrRefused
	}
	return nil
}
func (c *AcceptanceContractClient) CheckInstallation(p Policy) error {
	if c == nil || Digest(c.config.Policy) != Digest(p) {
		return ErrRefused
	}
	return nil
}
func (c *ScopeContractClient) CheckInstallation(p Policy) error {
	if c == nil || c.endpoint != ScopeContractEndpoint || Digest(c.config.Policy) != Digest(p) {
		return ErrRefused
	}
	return nil
}

var _ ScopeOwner = (*ScopeContractClient)(nil)

// Commission reads validate the exact authored disabled target before human
// finite consent. They confer no run/queue authority and use a distinct route.
func NewCommissionScopeContractHandler(owner ScopeOwner, peers map[string]map[string]AcceptanceContractGrant) (*ScopeContractHandler, error) {
	h, e := NewScopeContractHandler(owner, peers)
	if e == nil {
		h.endpoint = CommissionScopeEndpoint
	}
	return h, e
}
func NewCommissionScopeContractClient(cfg AcceptanceContractClientConfig) (*ScopeContractClient, error) {
	original := cfg.URL
	u, e := parseContractURL(original, CommissionScopeEndpoint)
	if e != nil {
		return nil, e
	}
	u.Path = ScopeContractEndpoint
	cfg.URL = u.String()
	c, e := NewScopeContractClient(cfg)
	if e != nil {
		return nil, e
	}
	c.endpoint = CommissionScopeEndpoint
	c.config.URL = original
	return c, nil
}
func (c *ScopeContractClient) CheckApprovalInstallation(p Policy) error {
	if c == nil || c.endpoint != CommissionScopeEndpoint || Digest(c.config.Policy) != Digest(p) {
		return ErrRefused
	}
	return nil
}
