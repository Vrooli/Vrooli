package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"time"
)

const ProfileContractEndpoint = "/internal/finite-effort/v1/profile/check"

type ProfileContract struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Digest string `json:"digest"`
}
type ProfileContractOwner interface {
	ReadFiniteProfileContract(context.Context, string) (ProfileContract, error)
}
type ProfileContractGrant struct {
	Binding  Binding           `json:"binding"`
	Profiles map[string]string `json:"profiles"`
}
type profileCheckRequest struct {
	Binding Binding `json:"binding"`
	Key     string  `json:"key"`
}

// ProfileContractHandler is a separately mounted owner-only read. It changes
// neither ordinary profile authentication nor profile state. Existing enrolled
// verified peers receive only their exact installed policy/profile contracts.
type ProfileContractHandler struct {
	owner ProfileContractOwner
	peers map[string]map[string]ProfileContractGrant
}

func NewProfileContractHandler(owner ProfileContractOwner, peers map[string]map[string]ProfileContractGrant) (*ProfileContractHandler, error) {
	if owner == nil || len(peers) == 0 {
		return nil, ErrRefused
	}
	raw, e := json.Marshal(peers)
	if e != nil {
		return nil, ErrRefused
	}
	copy := map[string]map[string]ProfileContractGrant{}
	if json.Unmarshal(raw, &copy) != nil {
		return nil, ErrRefused
	}
	for pin, grants := range copy {
		if !validDigest(pin) || len(grants) == 0 {
			return nil, ErrRefused
		}
		for id, g := range grants {
			if g.Binding.PolicyID != id || !validDigest(g.Binding.PolicyDigest) || len(g.Profiles) == 0 || g.Binding.Deadline.IsZero() || g.Binding.Epoch == 0 || g.Binding.Owner == "" || g.Binding.Client == "" {
				return nil, ErrRefused
			}
			for key, uuid := range g.Profiles {
				if key == "" || uuid == "" {
					return nil, ErrRefused
				}
			}
		}
	}
	return &ProfileContractHandler{owner, copy}, nil
}
func (h *ProfileContractHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	refuse := func() { http.Error(w, "finite profile contract refused", http.StatusForbidden) }
	if h == nil || r.Method != "POST" || r.URL.Path != ProfileContractEndpoint || r.URL.RawQuery != "" || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		refuse()
		return
	}
	for _, k := range []string{"Authorization", "Cookie", "X-Agent-Identity-Token", Header} {
		if len(r.Header.Values(k)) != 0 {
			refuse()
			return
		}
	}
	var in profileCheckRequest
	body, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(body) > 8192 {
		refuse()
		return
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(new(any)) != io.EOF {
		refuse()
		return
	}
	pin := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	g, ok := h.peers[hex.EncodeToString(pin[:])][in.Binding.PolicyID]
	if !ok || !sameProfileBinding(g.Binding, in.Binding) || !time.Now().Before(g.Binding.Deadline) || g.Profiles[in.Key] == "" {
		refuse()
		return
	}
	out, e := h.owner.ReadFiniteProfileContract(r.Context(), in.Key)
	if e != nil || out.Key != in.Key || out.ID != g.Profiles[in.Key] || !validDigest(out.Digest) {
		refuse()
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}

// CheckInstallation performs no owner read. Trusted startup checks that every
// private grant belongs to its exact installed policy and native profile IDs.
func (h *ProfileContractHandler) CheckInstallation(p Policy) error {
	if h == nil || p.Validate() != nil {
		return ErrRefused
	}
	for _, grants := range h.peers {
		for _, g := range grants {
			b := g.Binding
			if b.PolicyID != p.ID || b.PolicyDigest != Digest(p) || b.Epoch != p.Epoch || b.Owner != p.Owner || b.Client != p.Client || !b.Deadline.Equal(p.Deadline) {
				return ErrRefused
			}
			for key := range g.Profiles {
				if p.Profiles[key] == "" {
					return ErrRefused
				}
			}
		}
	}
	return nil
}

// QualifyOwnerContracts binds this private read to the actual native owner and
// complete persisted profiles before trusted startup publishes any installation.
func (h *ProfileContractHandler) QualifyOwnerContracts(ctx context.Context, p Policy, owner ProfileContractOwner) error {
	if h == nil || owner == nil || reflect.TypeOf(owner) != reflect.TypeOf(h.owner) || !reflect.TypeOf(owner).Comparable() || h.owner != owner || h.CheckInstallation(p) != nil {
		return ErrRefused
	}
	for _, grants := range h.peers {
		for _, g := range grants {
			for key, id := range g.Profiles {
				current, e := owner.ReadFiniteProfileContract(ctx, key)
				if e != nil || current.Key != key || current.ID != id || current.Digest != p.Profiles[key] {
					return ErrRefused
				}
			}
		}
	}
	return nil
}

// ProfileContractPlan freezes grants before the actual native owner is built.
// Startup binds the read handler to that newly composed owner, never another
// orchestrator instance or an editable callback from an HTTP request.
type ProfileContractPlan struct {
	peers  map[string]map[string]ProfileContractGrant
	policy Policy
}
type unreadProfilePlan struct{}

func (unreadProfilePlan) ReadFiniteProfileContract(context.Context, string) (ProfileContract, error) {
	return ProfileContract{}, ErrRefused
}
func PrepareProfileContract(peers map[string]map[string]ProfileContractGrant, p Policy) (*ProfileContractPlan, error) {
	h, e := NewProfileContractHandler(unreadProfilePlan{}, peers)
	if e != nil || h.CheckInstallation(p) != nil {
		return nil, ErrRefused
	}
	raw, e := json.Marshal(p)
	if e != nil {
		return nil, ErrRefused
	}
	var frozen Policy
	if json.Unmarshal(raw, &frozen) != nil {
		return nil, ErrRefused
	}
	return &ProfileContractPlan{peers: h.peers, policy: frozen}, nil
}
func (p *ProfileContractPlan) Handler(ctx context.Context, owner ProfileContractOwner) (*ProfileContractHandler, error) {
	if p == nil || owner == nil {
		return nil, ErrRefused
	}
	h, e := NewProfileContractHandler(owner, p.peers)
	if e != nil || h.QualifyOwnerContracts(ctx, p.policy, owner) != nil {
		return nil, ErrRefused
	}
	return h, nil
}

func sameProfileBinding(a, b Binding) bool {
	return a.PolicyID == b.PolicyID && a.Epoch == b.Epoch && a.PolicyDigest == b.PolicyDigest && a.Owner == b.Owner && a.Client == b.Client && a.Deadline.Equal(b.Deadline)
}
