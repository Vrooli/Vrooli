package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

const OwnerReadEndpoint = "/internal/finite-effort/v1/current-owner"

// PolicyOwners avoids turning the identity service's subject read into a
// public impersonation oracle. Remote readers present a protected policy;
// the serving owner derives the subject from its own immutable ledger.
type PolicyOwners interface {
	CurrentEffortPolicyOwner(context.Context, Policy) (OwnerState, error)
}

// OwnerReadHandler is opt-in; main does not register it. The server MUST use
// RequireAndVerifyClientCert. Peer certificate pins authorize only explicit
// policy IDs. Loopback, forwarded headers and account strings confer nothing.
// Store and Owners belong to the approval issuer; this handler never grants.
type OwnerReadHandler struct {
	Store  Store
	Owners Owners
	Peers  map[string]map[string]bool // leaf DER SHA256 -> exact policy IDs
	Now    func() time.Time
}

func (h *OwnerReadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	refuse := func() { http.Error(w, "finite owner read refused", http.StatusForbidden) }
	if h == nil || h.Store == nil || h.Owners == nil || r.Method != http.MethodPost || r.URL.Path != OwnerReadEndpoint || r.URL.RawQuery != "" || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		refuse()
		return
	}
	for _, header := range []string{"Authorization", "X-Agent-Identity-Token", Header, "Cookie"} {
		if len(r.Header.Values(header)) != 0 {
			refuse()
			return
		}
	}
	sum := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	allowed := h.Peers[hex.EncodeToString(sum[:])]
	if len(allowed) == 0 {
		refuse()
		return
	}
	var b Binding
	dec := json.NewDecoder(io.LimitReader(r.Body, 4097))
	dec.DisallowUnknownFields()
	if dec.Decode(&b) != nil || dec.Decode(new(any)) != io.EOF || !allowed[b.PolicyID] {
		refuse()
		return
	}
	rec, err := h.Store.Get(r.Context(), b.PolicyID)
	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}
	if err != nil || rec.Revoked || rec.Policy.Validate() != nil || binding(rec.Policy) != b || now.Before(rec.Policy.NotBefore) || !rec.Policy.Deadline.After(now) {
		refuse()
		return
	}
	owner, err := h.Owners.CurrentEffortOwner(r.Context(), rec.Policy.Owner)
	if err != nil || !scopesCurrent(rec.Policy, owner) {
		refuse()
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(owner)
}

// OwnerReadClient uses a preconfigured mutually authenticated HTTPS client.
// It additionally pins the server leaf and always refuses redirects. TLS
// verification must remain enabled; raw subject reads are deliberately refused.
// No sessions are exchanged, credentials minted, or retry/refresh performed.
type OwnerReadClient struct {
	URL       string
	HTTP      *http.Client
	ServerPin string
}

func (*OwnerReadClient) CurrentEffortOwner(context.Context, string) (OwnerState, error) {
	return OwnerState{}, ErrRefused
}
func (c *OwnerReadClient) CurrentEffortPolicyOwner(ctx context.Context, p Policy) (OwnerState, error) {
	if c == nil || c.HTTP == nil || c.HTTP.Timeout <= 0 || c.ServerPin == "" {
		return OwnerState{}, ErrRefused
	}
	u, err := url.Parse(c.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != OwnerReadEndpoint {
		return OwnerState{}, ErrRefused
	}
	transport, ok := c.HTTP.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil || transport.TLSClientConfig.InsecureSkipVerify || len(transport.TLSClientConfig.Certificates) == 0 {
		return OwnerState{}, ErrRefused
	}
	raw, _ := json.Marshal(binding(p))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(raw))
	if err != nil {
		return OwnerState{}, ErrRefused
	}
	req.Header.Set("Content-Type", "application/json")
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return OwnerState{}, ErrRefused
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 || len(resp.TLS.PeerCertificates) == 0 {
		return OwnerState{}, ErrRefused
	}
	sum := sha256.Sum256(resp.TLS.PeerCertificates[0].Raw)
	if hex.EncodeToString(sum[:]) != c.ServerPin {
		return OwnerState{}, ErrRefused
	}
	var owner OwnerState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 16385))
	dec.DisallowUnknownFields()
	if dec.Decode(&owner) != nil || dec.Decode(new(any)) != io.EOF || !scopesCurrent(p, owner) {
		return OwnerState{}, ErrRefused
	}
	return owner, nil
}
