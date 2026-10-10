// Package effortauthority owns session-independent, finite work authorization.
// It does not issue human credentials or grant native profile permissions.
package effortauthority

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/api-core/scopecatalog"
)

var ErrRefused = errors.New("finite effort authority refused")

const Header = "X-Vrooli-Effort-Proof"
const Audience = "agent-manager:finite-effort:v1"

// Policy is an immutable, protected approval. Amendment is a new ID; neither
// login refresh nor replay can change its deadline, limits or permissions.
// Approval is made by the configured identity provider, not by client labels.
type Policy struct {
	ID            string   `json:"id"`
	Owner         string   `json:"owner"`
	Client        string   `json:"client"`
	ClientKey     []byte   `json:"clientKey"`
	Epoch         uint64   `json:"epoch"`
	Repository    string   `json:"repository"`
	Effort        string   `json:"effort"`
	Revision      string   `json:"revision"`
	ContentDigest string   `json:"contentDigest"`
	Team          string   `json:"team"`
	TeamDigest    string   `json:"teamDigest"`
	BindingDigest string   `json:"bindingDigest"`
	Members       []string `json:"members"`
	// Profiles binds the complete owner-resolved execution contract for each
	// allowed existing profile. Profile/team changes require a new approval.
	Profiles        map[string]string `json:"profiles"`
	Scopes          []string          `json:"scopes"`
	Effects         []string          `json:"effects"`
	NotBefore       time.Time         `json:"notBefore"`
	Deadline        time.Time         `json:"deadline"`
	MaxStarts       int               `json:"maxStarts"`
	MaxConcurrent   int               `json:"maxConcurrent"`
	MaxTurns        int               `json:"maxTurns"`
	MaxToolCalls    int               `json:"maxToolCalls"`
	MaxRunSeconds   int64             `json:"maxRunSeconds"`
	TotalTurns      int64             `json:"totalTurns"`
	TotalToolCalls  int64             `json:"totalToolCalls"`
	TotalRunSeconds int64             `json:"totalRunSeconds"`
	SerialEdges     []SerialEdge      `json:"serialEdges,omitempty"`
	AllowChildren   bool              `json:"allowChildren"`
	AllowRecovery   bool              `json:"allowRecovery"`
	// Required explicit choice: logout ends a session, not this finite grant.
	SurviveLogout bool `json:"surviveLogout"`
}

// Intent names actual inputs resolved by the owning adapter/native service.
// External client task names are data; no platform attestation is inferred.
type Intent struct {
	Audience            string `json:"audience"`
	Method              string `json:"method"`
	Endpoint            string `json:"endpoint"`
	PolicyID            string `json:"policyId"`
	Epoch               uint64 `json:"epoch"`
	Client              string `json:"client"`
	Repository          string `json:"repository"`
	Effort              string `json:"effort"`
	Revision            string `json:"revision"`
	ContentDigest       string `json:"contentDigest"`
	Team                string `json:"team"`
	Member              string `json:"member"`
	Profile             string `json:"profile"`
	ProfileDigest       string `json:"profileDigest"`
	Effect              string `json:"effect"`
	IdempotencyKey      string `json:"idempotencyKey"`
	InputDigest         string `json:"inputDigest"`
	SerialPayloadDigest string `json:"serialPayloadDigest,omitempty"`
	ParentRunID         string `json:"parentRunId,omitempty"`
	SourceRunID         string `json:"sourceRunId,omitempty"`
	Turns               int    `json:"turns"`
	ToolCalls           int    `json:"toolCalls"`
	RunSeconds          int64  `json:"runSeconds"`
}

type Proof struct {
	Intent    Intent    `json:"intent"`
	Nonce     string    `json:"nonce"`
	IssuedAt  time.Time `json:"issuedAt"`
	Signature []byte    `json:"signature"`
}

// Binding contains no bearer or key. It remains identical across queue,
// dispatch, descendants and recovery, and cannot itself authenticate a client.
type Binding struct {
	PolicyID     string    `json:"policyId"`
	Epoch        uint64    `json:"epoch"`
	PolicyDigest string    `json:"policyDigest"`
	Owner        string    `json:"owner"`
	Client       string    `json:"client"`
	Deadline     time.Time `json:"deadline"`
}

type OwnerState struct {
	Subject string
	Enabled bool
	Scopes  []string
	Ceiling []string
}

// Owners is a live identity-provider read. No cached JWT or expired human
// bearer can implement it. Errors/mismatched principals fail closed.
type Owners interface {
	CurrentEffortOwner(context.Context, string) (OwnerState, error)
}

// Approver verifies live human session and current self grants at approval.
type Approver interface {
	ApproveEffortOwner(context.Context, string) (OwnerState, error)
}

// ScopeOwner reads trusted current effort/revision/content/team/profile state.
// Repository files, submitted WorkReferences and approval quotes are not it.
type ScopeOwner interface {
	CheckEffortScope(context.Context, Policy) error
}

type Record struct {
	Policy         Policy                   `json:"policy"`
	Revoked        bool                     `json:"revoked"`
	Reservations   map[string]Reservation   `json:"reservations"`
	Nonces         map[string]string        `json:"nonces"`
	Ingress        map[string]string        `json:"ingress"`
	SerialHandoffs map[string]SerialHandoff `json:"serialHandoffs,omitempty"`
}
type Reservation struct {
	Preparation  *Intent `json:"preparation,omitempty"`
	NativeBound  bool    `json:"nativeBound"`
	IntentDigest string  `json:"intentDigest"`
	Intent       Intent  `json:"intent"`
	RunID        string  `json:"runId,omitempty"`
	Terminal     bool    `json:"terminal"`
}

// Store must serialize Update durably; a failed callback has zero writes.
type Store interface {
	Get(context.Context, string) (Record, error)
	Create(context.Context, Record) error
	Update(context.Context, string, func(*Record) error) error
}
type Authority struct {
	Store  Store
	Owners Owners
	Scope  ScopeOwner
	// ApprovalScope is an optional read-only commissioning check; live admission
	// always uses Scope and therefore still requires current enabled target state.
	ApprovalScope ScopeOwner
	Approver      Approver
	Now           func() time.Time
}

func (a *Authority) now() time.Time {
	if a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}
func Digest(value any) string {
	b, e := json.Marshal(value)
	if e != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func validDigest(s string) bool { b, e := hex.DecodeString(s); return e == nil && len(b) == 32 }
func (p Policy) Validate() error {
	if p.ID == "" || p.Owner == "" || p.Client == "" || len(p.ClientKey) != ed25519.PublicKeySize || p.Epoch == 0 || p.Effort == "" || p.Revision == "" || p.Team == "" || len(p.Members) == 0 || len(p.Profiles) == 0 || (!validDigest(p.ContentDigest) || !validDigest(p.TeamDigest) || !validDigest(p.BindingDigest)) || !filepath.IsAbs(p.Repository) || filepath.Clean(p.Repository) != p.Repository || !p.Deadline.After(p.NotBefore) || p.Deadline.Sub(p.NotBefore) > 24*time.Hour || p.MaxStarts < 1 || p.MaxStarts > 1000 || p.MaxConcurrent < 1 || p.MaxConcurrent > p.MaxStarts || p.MaxTurns < 1 || p.MaxTurns > 100000 || p.MaxToolCalls < 1 || p.MaxToolCalls > 100000 || p.MaxRunSeconds < 1 || p.MaxRunSeconds > 86400 || p.TotalTurns > 100000000 || p.TotalToolCalls > 100000000 || p.TotalRunSeconds > 86400000 || p.TotalTurns < int64(p.MaxTurns) || p.TotalToolCalls < int64(p.MaxToolCalls) || p.TotalRunSeconds < p.MaxRunSeconds || !p.SurviveLogout || len(p.Effects) == 0 || len(p.Scopes) == 0 {
		return ErrRefused
	}
	for _, effect := range p.Effects {
		if effect != "run.create" && effect != "run.child" && effect != "run.recover" {
			return ErrRefused
		}
	}
	for k, v := range p.Profiles {
		if k == "" || !validDigest(v) {
			return ErrRefused
		}
	}
	for _, s := range p.Scopes {
		if strings.TrimSpace(s) != s || s == "" || scopecatalog.IsWildcard(s) {
			return ErrRefused
		}
	}
	if !scopecatalog.MatchCapability(p.Scopes, "agent-manager:write") {
		return ErrRefused
	}
	if validateSerialEdges(p) != nil {
		return ErrRefused
	}
	return nil
}
func scopesCurrent(p Policy, s OwnerState) bool {
	if !s.Enabled || s.Subject != p.Owner {
		return false
	}
	for _, v := range p.Scopes {
		if !scopecatalog.MatchCapability(s.Scopes, v) || (s.Ceiling != nil && !scopecatalog.MatchCapability(s.Ceiling, v)) {
			return false
		}
	}
	return true
}
func (a *Authority) Approve(ctx context.Context, humanToken string, p Policy) error {
	if a == nil || a.Store == nil || a.Approver == nil || a.Owners == nil || a.Scope == nil || p.Validate() != nil || a.now().Before(p.NotBefore) || !p.Deadline.After(a.now()) {
		return ErrRefused
	}
	owner, e := a.Approver.ApproveEffortOwner(ctx, humanToken)
	if e != nil || !scopesCurrent(p, owner) {
		return ErrRefused
	}
	current, e := a.Owners.CurrentEffortOwner(ctx, p.Owner)
	approvalScope := a.Scope
	if a.ApprovalScope != nil {
		approvalScope = a.ApprovalScope
	}
	if e != nil || !scopesCurrent(p, current) || approvalScope.CheckEffortScope(ctx, p) != nil {
		return ErrRefused
	}
	return a.Store.Create(ctx, Record{Policy: p, Reservations: map[string]Reservation{}, Nonces: map[string]string{}, Ingress: map[string]string{}})
}
func (a *Authority) Revoke(ctx context.Context, humanToken, id string) error {
	if a == nil || a.Approver == nil || a.Store == nil {
		return ErrRefused
	}
	owner, e := a.Approver.ApproveEffortOwner(ctx, humanToken)
	if e != nil || !owner.Enabled {
		return ErrRefused
	}
	return a.Store.Update(ctx, id, func(r *Record) error {
		if r.Policy.Owner != owner.Subject {
			return ErrRefused
		}
		r.Revoked = true
		return nil
	})
}
func (a *Authority) live(ctx context.Context, id string) (Record, error) {
	if a == nil || a.Store == nil || a.Owners == nil || a.Scope == nil || ctx.Err() != nil {
		return Record{}, ErrRefused
	}
	r, e := a.Store.Get(ctx, id)
	if e != nil || r.Revoked || r.Policy.Validate() != nil || a.now().Before(r.Policy.NotBefore) || !r.Policy.Deadline.After(a.now()) {
		return Record{}, ErrRefused
	}
	s, e := a.currentOwner(ctx, r.Policy)
	if e != nil || !scopesCurrent(r.Policy, s) || a.Scope.CheckEffortScope(ctx, r.Policy) != nil {
		return Record{}, ErrRefused
	}
	return r, nil
}
func binding(p Policy) Binding {
	return Binding{p.ID, p.Epoch, Digest(p), p.Owner, p.Client, p.Deadline}
}
func (a *Authority) CheckBinding(ctx context.Context, b Binding) (Policy, error) {
	r, e := a.live(ctx, b.PolicyID)
	if e != nil || binding(r.Policy) != b {
		return Policy{}, ErrRefused
	}
	return r.Policy, nil
}
func unsigned(p Proof) []byte { p.Signature = nil; b, _ := json.Marshal(p); return b }

// Sign is for an explicitly enrolled client's custody owner. Production must
// never synthesize a key, exchange account credentials or infer enrollment.
func Sign(p Proof, key ed25519.PrivateKey) (Proof, error) {
	if len(key) != ed25519.PrivateKeySize {
		return Proof{}, ErrRefused
	}
	p.Signature = ed25519.Sign(key, unsigned(p))
	return p, nil
}
func (a *Authority) CheckProof(ctx context.Context, p Proof) (Binding, error) {
	r, e := a.live(ctx, p.Intent.PolicyID)
	if e != nil {
		return Binding{}, ErrRefused
	}
	age := a.now().Sub(p.IssuedAt)
	if age < 0 || age > 2*time.Minute || len(p.Nonce) < 16 || len(p.Nonce) > 128 || !ed25519.Verify(r.Policy.ClientKey, unsigned(p), p.Signature) || checkIntent(r.Policy, p.Intent) != nil {
		return Binding{}, ErrRefused
	}
	return binding(r.Policy), nil
}
func checkIntent(p Policy, i Intent) error {
	if i.Audience != Audience || i.Method != "POST" || ((i.Effect != "run.recover" && i.Endpoint != "/api/v1/runs" && !(i.Effect == "run.create" && i.Endpoint == "/api/v1/teams/"+i.Team+"/members/"+i.Member+"/finite/start")) || (i.Effect == "run.recover" && i.Endpoint != "/api/v1/runs/resume-from-failed")) || i.PolicyID != p.ID || i.Epoch != p.Epoch || i.Client != p.Client || i.Repository != p.Repository || i.Effort != p.Effort || i.Revision != p.Revision || i.ContentDigest != p.ContentDigest || i.Team != p.Team || !slices.Contains(p.Members, i.Member) || p.Profiles[i.Profile] != i.ProfileDigest || i.ProfileDigest == "" || !slices.Contains(p.Effects, i.Effect) || i.IdempotencyKey == "" || len(i.IdempotencyKey) > 256 || !validDigest(i.InputDigest) || i.Turns < 1 || i.Turns > p.MaxTurns || i.ToolCalls < 1 || i.ToolCalls > p.MaxToolCalls || i.RunSeconds < 1 || i.RunSeconds > p.MaxRunSeconds {
		return ErrRefused
	}
	if i.SerialPayloadDigest != "" && (!validDigest(i.SerialPayloadDigest) || len(p.SerialEdges) == 0 || i.Effect != "run.child") {
		return ErrRefused
	}
	switch i.Effect {
	case "run.create":
		if i.ParentRunID != "" || i.SourceRunID != "" {
			return ErrRefused
		}
	case "run.child":
		if !p.AllowChildren || i.ParentRunID == "" || i.SourceRunID != "" {
			return ErrRefused
		}
	case "run.recover":
		if !p.AllowRecovery || i.SourceRunID == "" {
			return ErrRefused
		}
	default:
		return ErrRefused
	}
	return nil
}

// Reserve is the admission linearization point, immediately before the first
// native effect. Remote revocation after this check does not undo admission.
// Consuming conservative full ceilings preserves unknown usage as occupied.
// A pending/lost outcome never releases budget or authorizes redispatch.
func (a *Authority) Reserve(ctx context.Context, b Binding, i Intent, p *Proof) (Reservation, bool, error) {
	return a.reserve(ctx, b, i, p, "")
}
func (a *Authority) reserve(ctx context.Context, b Binding, i Intent, p *Proof, handoffKey string) (Reservation, bool, error) {
	policy, e := a.CheckBinding(ctx, b)
	if e != nil || checkIntent(policy, i) != nil || (i.Endpoint != "/api/v1/runs" && i.Endpoint != "/api/v1/runs/resume-from-failed") {
		return Reservation{}, false, ErrRefused
	}
	if len(policy.SerialEdges) > 0 && ((handoffKey == "" && i.Effect != "run.create") || (handoffKey != "" && (p != nil || i.Effect != "run.child"))) {
		return Reservation{}, false, ErrRefused
	}
	if handoffKey != "" && len(policy.SerialEdges) == 0 {
		return Reservation{}, false, ErrRefused
	}
	if p != nil {
		v, e := a.CheckProof(ctx, *p)
		if e != nil || v != b || Digest(p.Intent) != Digest(i) {
			return Reservation{}, false, ErrRefused
		}
	}
	var result Reservation
	var replay bool
	e = a.Store.Update(ctx, b.PolicyID, func(r *Record) error {
		if r.Revoked || binding(r.Policy) != b || !r.Policy.Deadline.After(a.now()) {
			return ErrRefused
		}
		if handoffKey != "" && (ctx.Err() != nil || checkSerialHandoff(*r, handoffKey, i, true) != nil) {
			return ErrRefused
		}
		digest := Digest(i)
		if p != nil {
			if old, ok := r.Nonces[p.Nonce]; ok && old != digest {
				return ErrRefused
			}
		}
		if old, ok := r.Reservations[i.IdempotencyKey]; ok {
			if old.Preparation != nil && !old.NativeBound {
				// The prepared slot is bound once to owner-resolved native inputs. Only
				// the enrolled client's request-bound root proof can finalize it.
				prepared := *old.Preparation
				candidate := i
				prepared.Endpoint = candidate.Endpoint
				prepared.InputDigest = candidate.InputDigest
				if p == nil || i.Effect != "run.create" || Digest(prepared) != digest || old.RunID != "" || old.Terminal {
					return ErrRefused
				}
				old.Intent = i
				old.IntentDigest = digest
				old.NativeBound = true
				if _, ok := r.Nonces[p.Nonce]; ok {
					return ErrRefused
				}
				r.Nonces[p.Nonce] = digest
				r.Reservations[i.IdempotencyKey] = old
				result = old
				return nil
			}
			if old.IntentDigest != digest {
				return ErrRefused
			}
			result, replay = old, true
			return nil
		}
		if len(policy.SerialEdges) > 0 && len(r.Reservations) > 0 && i.Effect == "run.create" {
			return ErrRefused
		}
		var active int
		var turns, tools, seconds int64
		for _, v := range r.Reservations {
			if v.Intent.Turns < 1 || v.Intent.ToolCalls < 1 || v.Intent.RunSeconds < 1 || int64(v.Intent.Turns) > policy.TotalTurns-turns || int64(v.Intent.ToolCalls) > policy.TotalToolCalls-tools || v.Intent.RunSeconds > policy.TotalRunSeconds-seconds {
				return ErrRefused
			}
			if !v.Terminal {
				active++
			}
			turns += int64(v.Intent.Turns)
			tools += int64(v.Intent.ToolCalls)
			seconds += v.Intent.RunSeconds
		}
		if len(r.Reservations) >= policy.MaxStarts || active >= policy.MaxConcurrent || turns > policy.TotalTurns-int64(i.Turns) || tools > policy.TotalToolCalls-int64(i.ToolCalls) || seconds > policy.TotalRunSeconds-i.RunSeconds {
			return ErrRefused
		}
		if p != nil {
			if _, ok := r.Nonces[p.Nonce]; ok {
				return ErrRefused
			}
			r.Nonces[p.Nonce] = digest
		}
		result = Reservation{IntentDigest: digest, Intent: i, NativeBound: true}
		r.Reservations[i.IdempotencyKey] = result
		return nil
	})
	if e != nil {
		return Reservation{}, false, ErrRefused
	}
	return result, replay, nil
}

// Accept/Settle are native owner callbacks, never external request operations.
// Settle needs the exact accepted run identity, so queued proof cannot free slots.
func (a *Authority) Accept(ctx context.Context, b Binding, key, runID string) error {
	return a.Store.Update(ctx, b.PolicyID, func(r *Record) error {
		v, ok := r.Reservations[key]
		if !ok || !v.NativeBound || runID == "" || (v.RunID != "" && v.RunID != runID) || binding(r.Policy) != b {
			return ErrRefused
		}
		// Serial native episodes always have fresh identities, including after
		// owner restart. An exact accepted replay is permitted, reuse is not.
		if len(r.Policy.SerialEdges) > 0 {
			for otherKey, other := range r.Reservations {
				if otherKey != key && other.RunID == runID {
					return ErrRefused
				}
			}
		}
		v.RunID = runID
		r.Reservations[key] = v
		return nil
	})
}
func (a *Authority) Settle(ctx context.Context, b Binding, key, runID string) error {
	return a.Store.Update(ctx, b.PolicyID, func(r *Record) error {
		v, ok := r.Reservations[key]
		if !ok || v.RunID == "" || v.RunID != runID || binding(r.Policy) != b {
			return ErrRefused
		}
		v.Terminal = true
		r.Reservations[key] = v
		return nil
	})
}

// Prepare reserves all conservative native ceilings before any Planner save,
// history, queue or task effect. Optional external ingress is committed in the
// same transaction. Crash/lost outcomes retain the slot; replay cannot renew it.
func (a *Authority) Prepare(ctx context.Context, proof Proof, ingress *Proof) (Reservation, bool, error) {
	b, e := a.CheckProof(ctx, proof)
	if e != nil || proof.Intent.Endpoint != "/api/v1/teams/"+proof.Intent.Team+"/members/"+proof.Intent.Member+"/finite/start" || proof.Intent.Effect != "run.create" {
		return Reservation{}, false, ErrRefused
	}
	policy, e := a.CheckBinding(ctx, b)
	if e != nil {
		return Reservation{}, false, ErrRefused
	}
	if ingress != nil {
		ib, err := a.CheckProof(ctx, *ingress)
		if err != nil || ib != b {
			return Reservation{}, false, ErrRefused
		}
		original := ingress.Intent
		actual := proof.Intent
		original.InputDigest = actual.InputDigest
		if Digest(original) != Digest(actual) {
			return Reservation{}, false, ErrRefused
		}
	}
	var result Reservation
	var replay bool
	e = a.Store.Update(ctx, b.PolicyID, func(r *Record) error {
		if r.Revoked || binding(r.Policy) != b || !r.Policy.Deadline.After(a.now()) {
			return ErrRefused
		}
		digest := Digest(proof.Intent)
		if old, ok := r.Nonces[proof.Nonce]; ok && old != digest {
			return ErrRefused
		}
		if ingress != nil {
			if old, ok := r.Nonces[ingress.Nonce]; ok && old != Digest(ingress.Intent) {
				return ErrRefused
			}
		}
		if old, ok := r.Reservations[proof.Intent.IdempotencyKey]; ok {
			if old.Preparation == nil || Digest(*old.Preparation) != digest {
				return ErrRefused
			}
			if ingress != nil && r.Ingress[ingress.Intent.IdempotencyKey] != Digest(ingress.Intent) {
				return ErrRefused
			}
			result = old
			replay = true
			return nil
		}
		if len(policy.SerialEdges) > 0 && len(r.Reservations) > 0 {
			return ErrRefused
		}
		var active int
		var turns, tools, seconds int64
		for _, v := range r.Reservations {
			if v.Intent.Turns < 1 || v.Intent.ToolCalls < 1 || v.Intent.RunSeconds < 1 || int64(v.Intent.Turns) > policy.TotalTurns-turns || int64(v.Intent.ToolCalls) > policy.TotalToolCalls-tools || v.Intent.RunSeconds > policy.TotalRunSeconds-seconds {
				return ErrRefused
			}
			if !v.Terminal {
				active++
			}
			turns += int64(v.Intent.Turns)
			tools += int64(v.Intent.ToolCalls)
			seconds += v.Intent.RunSeconds
		}
		i := proof.Intent
		if i.Turns != policy.MaxTurns || i.ToolCalls != policy.MaxToolCalls || i.RunSeconds != policy.MaxRunSeconds || len(r.Reservations) >= policy.MaxStarts || active >= policy.MaxConcurrent || turns > policy.TotalTurns-int64(i.Turns) || tools > policy.TotalToolCalls-int64(i.ToolCalls) || seconds > policy.TotalRunSeconds-i.RunSeconds {
			return ErrRefused
		}
		if _, ok := r.Nonces[proof.Nonce]; ok {
			return ErrRefused
		}
		if ingress != nil {
			if _, ok := r.Nonces[ingress.Nonce]; ok {
				return ErrRefused
			}
			if _, ok := r.Ingress[ingress.Intent.IdempotencyKey]; ok {
				return ErrRefused
			}
			if r.Ingress == nil {
				r.Ingress = map[string]string{}
			}
			r.Ingress[ingress.Intent.IdempotencyKey] = Digest(ingress.Intent)
			r.Nonces[ingress.Nonce] = Digest(ingress.Intent)
		}
		r.Nonces[proof.Nonce] = digest
		prepared := i
		result = Reservation{Preparation: &prepared, Intent: i, IntentDigest: digest}
		r.Reservations[i.IdempotencyKey] = result
		return nil
	})
	if e != nil {
		return Reservation{}, false, ErrRefused
	}
	return result, replay, nil
}

func (a *Authority) currentOwner(ctx context.Context, p Policy) (OwnerState, error) {
	if remote, ok := a.Owners.(PolicyOwners); ok {
		return remote.CurrentEffortPolicyOwner(ctx, p)
	}
	return a.Owners.CurrentEffortOwner(ctx, p.Owner)
}
