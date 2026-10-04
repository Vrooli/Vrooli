package heartbeat

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/api-core/effortauthority"
	"github.com/vrooli/api-core/owneridentity"
	"prompt-manager/internal/store"
)

// EffortSigner is an explicitly enrolled custody owner. No keys are generated,
// stored in queues, read from environment or bootstrapped by this adapter.
type EffortSigner interface {
	SignEffort(context.Context, string, effortauthority.Proof) (effortauthority.Proof, error)
}
type effortCallerKey struct{}
type finiteEffortCaller struct {
	authority *FiniteEffortAuthority
	binding   effortauthority.Binding
	member    string
}

func effortCaller(ctx context.Context) (finiteEffortCaller, bool) {
	v, ok := ctx.Value(effortCallerKey{}).(finiteEffortCaller)
	return v, ok
}

// FiniteEffortAuthority is installed only by exact owner-approved setup. The
// protected binding table and custody provider do not follow editable labels.
// An empty/unconfigured authority retains AUTH-01 human admission behavior.
type FiniteEffortAuthority struct {
	Authority effortauthority.Engine
	Signer    EffortSigner
	Bindings  map[string]effortauthority.Binding // exact team/agent
	Tasks     interface {
		GetTask(context.Context, string) (*Task, error)
	}
	Now func() time.Time
}

func (a *FiniteEffortAuthority) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}
func (a *FiniteEffortAuthority) Check(ctx context.Context) error {
	caller, ok := effortCaller(ctx)
	if !ok || caller.authority != a || a.Signer == nil {
		return effortauthority.ErrRefused
	}
	_, e := a.Authority.CheckBinding(ctx, caller.binding)
	return e
}
func (f *FiniteLeaderRuntime) prepareFiniteCaller(ctx context.Context, team, agent string, cfg *store.HeartbeatConfig) (context.Context, error) {
	if f.Efforts != nil {
		b, configured := f.Efforts.Bindings[team+"/"+agent]
		if configured {
			if cfg == nil || cfg.FiniteLeader == nil || !cfg.Enabled || cfg.FiniteLeader.Retired || f.Efforts.Signer == nil {
				return nil, effortauthority.ErrRefused
			}
			// Offered human/run proof cannot be silently ignored by the effort route.
			proof := createRunCaller(ctx)
			if proof.authorization != "" || proof.runIdentity != "" {
				return nil, effortauthority.ErrRefused
			}
			p, e := f.Efforts.Authority.CheckBinding(ctx, b)
			if e != nil || p.Team != team || p.Effort != cfg.FiniteLeader.EffortRef || p.Revision != cfg.FiniteLeader.AcceptedRevision || p.Profiles[cfg.ProfileKey] == "" || !slices.Contains(p.Members, agent) {
				return nil, effortauthority.ErrRefused
			}
			return context.WithValue(ctx, effortCallerKey{}, finiteEffortCaller{f.Efforts, b, agent}), nil
		}
	}
	if c, ok := f.Executor.agentClient.(*AgentManagerClient); ok {
		return c.prepareCreateRunCaller(ctx)
	}
	// Test/other implementations must enforce their own caller contract. This
	// preserves existing typed adapters without creating production fallback.
	return ctx, nil
}
func requireQualifiedCaller(ctx context.Context) error {
	if c, ok := effortCaller(ctx); ok {
		return c.authority.Check(ctx)
	}
	return owneridentity.RequireCreateRunCaller(ctx, time.Now())
}
func carryQualifiedCaller(from, to context.Context) (context.Context, error) {
	if c, ok := effortCaller(from); ok {
		if e := c.authority.Check(from); e != nil {
			return nil, e
		}
		return context.WithValue(to, effortCallerKey{}, c), nil
	}
	return owneridentity.CarryCreateRunCaller(from, to, time.Now())
}

// Proof is request-bound at the actual native transport. Queue/restart metadata
// is not a credential; the same immutable binding is rechecked and explicit
// custody signs each transport nonce without extending its effort deadline.
func (a *FiniteEffortAuthority) Proof(ctx context.Context, b effortauthority.Binding, body []byte) (string, error) {
	if e := a.Check(ctx); e != nil {
		return "", e
	}
	p, e := a.Authority.CheckBinding(ctx, b)
	if e != nil || a.Tasks == nil {
		return "", effortauthority.ErrRefused
	}
	var req CreateRunRequest
	if json.Unmarshal(body, &req) != nil || req.ProfileRef == nil || req.ParentRunID != nil || req.RequestedScopes != nil || req.RunMode != "" {
		return "", effortauthority.ErrRefused
	}
	task, e := a.Tasks.GetTask(ctx, req.TaskID)
	if e != nil || task == nil || task.ProjectRoot != p.Repository || task.ScopePath != p.Repository {
		return "", effortauthority.ErrRefused
	}
	// Attached native task projection is not yet qualified. Retain its presence
	// and refuse before sending a proof rather than signing an incomplete view.
	attachments := strings.TrimSpace(string(task.ContextAttachments))
	if attachments != "" && attachments != "null" && attachments != "[]" {
		return "", effortauthority.ErrRefused
	}
	// PM owns the exact leader inputs; task labels are not external attestation.
	taskDigest := effortauthority.Digest(struct {
		Title, Description, ScopePath, ProjectRoot string
		Attachments                                []any
	}{task.Title, task.Description, task.ScopePath, task.ProjectRoot, nil})
	tag := ""
	if req.Tag != nil {
		tag = *req.Tag
	}
	input := effortauthority.RunInput{TaskID: req.TaskID, TaskDigest: taskDigest, Profile: req.ProfileRef.ProfileKey, IdempotencyKey: req.IdempotencyKey, Tag: tag, Environment: req.Environment}
	caller, ok := effortCaller(ctx)
	if !ok || caller.binding != b {
		return "", effortauthority.ErrRefused
	}
	member := caller.member
	if member == "" {
		return "", effortauthority.ErrRefused
	}
	intent := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: member, Profile: req.ProfileRef.ProfileKey, ProfileDigest: p.Profiles[req.ProfileRef.ProfileKey], Effect: "run.create", IdempotencyKey: req.IdempotencyKey, InputDigest: effortauthority.Digest(input), Turns: p.MaxTurns, ToolCalls: p.MaxToolCalls, RunSeconds: p.MaxRunSeconds}
	nonce := make([]byte, 24)
	if _, e = rand.Read(nonce); e != nil {
		return "", effortauthority.ErrRefused
	}
	proof, e := a.Signer.SignEffort(ctx, b.Client, effortauthority.Proof{Intent: intent, Nonce: hex.EncodeToString(nonce), IssuedAt: a.now()})
	if e != nil {
		return "", effortauthority.ErrRefused
	}
	checked, e := a.Authority.CheckProof(ctx, proof)
	if e != nil || checked != b {
		return "", effortauthority.ErrRefused
	}
	data, e := json.Marshal(proof)
	if e != nil {
		return "", effortauthority.ErrRefused
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
func splitEffortMember(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			return key[:i], key[i+1:]
		}
	}
	return "", ""
}

func qualifiedEffortBinding(ctx context.Context) *effortauthority.Binding {
	if c, ok := effortCaller(ctx); ok {
		b := c.binding
		return &b
	}
	return nil
}
func (a *FiniteEffortAuthority) Restore(ctx context.Context, b effortauthority.Binding, team, agent, profile string) (context.Context, error) {
	configured, ok := a.Bindings[team+"/"+agent]
	if !ok || configured != b || a.Signer == nil {
		return nil, effortauthority.ErrRefused
	}
	p, e := a.Authority.CheckBinding(ctx, b)
	if e != nil || p.Team != team || p.Profiles[profile] == "" || !slices.Contains(p.Members, agent) {
		return nil, effortauthority.ErrRefused
	}
	return context.WithValue(context.WithoutCancel(ctx), effortCallerKey{}, finiteEffortCaller{a, b, agent}), nil
}

type ingressProofKey struct{}

// ReserveLeader is the shared authority admission point before PM bookkeeping.
// It conservatively charges the native run, and native binds its actual task
// exactly once to this same idempotency key. No speculative slot is released.
func (f *FiniteLeaderRuntime) reserveEffortLeader(ctx context.Context, cfg *store.HeartbeatConfig, state *store.FiniteLeaderState) error {
	caller, ok := effortCaller(ctx)
	if !ok {
		return nil
	}
	a := caller.authority
	if e := a.Check(ctx); e != nil {
		return e
	}
	p, e := a.Authority.CheckBinding(ctx, caller.binding)
	if e != nil {
		return e
	}
	if state.ID == "" || cfg == nil || cfg.FiniteLeader == nil {
		return effortauthority.ErrRefused
	}
	intent := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: EffortStartEndpoint(p.Team, caller.member), PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: caller.member, Profile: cfg.ProfileKey, ProfileDigest: p.Profiles[cfg.ProfileKey], Effect: "run.create", IdempotencyKey: "finite-leader-" + state.ID, InputDigest: effortauthority.Digest(struct {
		ID      string
		Binding any
	}{state.ID, cfg.FiniteLeader}), Turns: p.MaxTurns, ToolCalls: p.MaxToolCalls, RunSeconds: p.MaxRunSeconds}
	nonce := make([]byte, 24)
	if _, e = rand.Read(nonce); e != nil {
		return effortauthority.ErrRefused
	}
	proof, e := a.Signer.SignEffort(ctx, p.Client, effortauthority.Proof{Intent: intent, Nonce: hex.EncodeToString(nonce), IssuedAt: a.now()})
	if e != nil {
		return effortauthority.ErrRefused
	}
	ingress, _ := ctx.Value(ingressProofKey{}).(*effortauthority.Proof)
	receipt, replay, e := a.Authority.Prepare(ctx, proof, ingress)
	if e != nil {
		return e
	}
	if replay && receipt.NativeBound && (!state.TaskStarted || state.TaskID == "") {
		return fmt.Errorf("finite native outcome retained; refuse to reconstruct task for %s", intent.IdempotencyKey)
	}
	return nil
}
func reserveQualifiedFiniteLeader(ctx context.Context, state *store.FiniteLeaderState) error {
	if ingress, ok := ctx.Value(ingressProofKey{}).(*effortauthority.Proof); ok && state.ID == "" {
		const prefix = "finite-leader-"
		if !strings.HasPrefix(ingress.Intent.IdempotencyKey, prefix) {
			return effortauthority.ErrRefused
		}
		id := strings.TrimPrefix(ingress.Intent.IdempotencyKey, prefix)
		if _, e := uuid.Parse(id); e != nil {
			return effortauthority.ErrRefused
		}
		state.ID = id
		state.Status = "queued"
		state.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	reserveFiniteLeader(state)
	return nil
}
