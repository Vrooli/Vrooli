// Package hosttoolpermissions projects owner supplied host tool declarations
// into onboarding's shared permission registry. It is intentionally read-only:
// installation is an effect owned by the host control plane and this package
// has no authority to invoke it.
package hosttoolpermissions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID       = "onboarding.host-tools"
	Owner            = "vrooli-onboarding"
	PermissionPrefix = "onboarding/host-tool/"
	SubjectKind      = "manifest-host-tool"
	Source           = "scenario host requirement manifests via host control-plane inventory"
	SchemaVersion    = "operator-state/host-tools-opt-in/v1"
	PermissionScope  = "vrooli-onboarding:write"
	LocalTarget      = "local"
)

// Tool is the safe, serializable declaration projection supplied by the host
// requirement owner. Runtime paths and command arguments are deliberately not
// part of this permission surface.
type Tool struct {
	Name         string   `json:"name"`
	Scenario     string   `json:"scenario"`
	Required     bool     `json:"required"`
	Reason       string   `json:"reason"`
	MinVersion   string   `json:"min_version,omitempty"`
	Platforms    []string `json:"platforms,omitempty"`
	Privilege    string   `json:"privilege"`
	SchemaDigest string   `json:"schema_digest"`
	OptedIn      bool     `json:"opted_in"`
}

type Inventory func(context.Context) (revision string, tools []Tool, err error)
type StateReader interface {
	Get(context.Context) (operatorstate.Document, error)
}
type StateMutation struct {
	ActorID, Scenario, Name, ExpectedRevision, RequestID, RequestDigest, DecisionRef string
	OptedIn                                                                          bool
}
type StateReceipt struct {
	RequestID, ActorID, Target, Subject, ExpectedRevision, NewRevision, RequestDigest, Outcome string
	At                                                                                         time.Time
}
type StateWriter interface {
	CompareAndSetHostTool(context.Context, StateMutation) (StateReceipt, error)
}
type Provider struct {
	inventory         Inventory
	reader            StateReader
	writer            StateWriter
	now               func() time.Time
	authorizeMutation func(context.Context) error
}

func New(inventory Inventory, reader StateReader, writer StateWriter) *Provider {
	return &Provider{inventory: inventory, reader: reader, writer: writer, now: func() time.Time { return time.Now().UTC() }, authorizeMutation: authz.RequireMutation}
}

var _ permissions.Provider = (*Provider)(nil)

func (*Provider) ProviderID() string                 { return ProviderID }
func (*Provider) Owner() string                      { return Owner }
func (*Provider) MatchesPermissionID(id string) bool { return strings.HasPrefix(id, PermissionPrefix) }

func verifiedActor(ctx context.Context) (identity.Principal, error) {
	a, ok := identity.PrincipalFromContext(ctx)
	if !ok || !a.Verified || a.Subject == "" {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding operator required"))
	}
	return a, nil
}

func (p *Provider) snapshot(ctx context.Context) (string, []Tool, error) {
	if p == nil || p.inventory == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errors.New("host tool owner inventory is unavailable"))
	}
	revision, tools, err := p.inventory(ctx)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(revision) == "" {
		return "", nil, connect.NewError(connect.CodeDataLoss, errors.New("host tool owner returned no revision"))
	}
	seen := map[string]bool{}
	for i := range tools {
		t := &tools[i]
		t.Name, t.Scenario = strings.TrimSpace(t.Name), strings.TrimSpace(t.Scenario)
		if t.Name == "" || t.Scenario == "" || strings.TrimSpace(t.SchemaDigest) == "" || seen[t.Scenario+"/"+t.Name] {
			return "", nil, connect.NewError(connect.CodeDataLoss, errors.New("host tool inventory is incomplete or duplicated"))
		}
		if t.Privilege != "none" && t.Privilege != "user" && t.Privilege != "elevated" {
			return "", nil, connect.NewError(connect.CodeDataLoss, fmt.Errorf("host tool %q has unsupported privilege classification", t.Name))
		}
		seen[t.Scenario+"/"+t.Name] = true
	}
	return revision, tools, nil
}

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	a, err := verifiedActor(ctx)
	if err != nil {
		return nil, err
	}
	revision, tools, err := p.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]permissions.Descriptor, 0, len(tools))
	for _, t := range tools {
		subject := t.Scenario + "/" + t.Name
		out = append(out, permissions.Descriptor{Version: permissions.ContractVersion, ID: PermissionPrefix + subject, Owner: Owner, PrincipalKind: string(a.Kind), PrincipalID: a.Subject, Scope: PermissionScope, SubjectKind: SubjectKind, SubjectID: subject, Source: Source, SchemaVersion: SchemaVersion + "/" + t.SchemaDigest + "/" + revision, CanRead: true, CanWrite: p.writer != nil, Sensitivity: "operator"})
	}
	return out, nil
}

func (p *Provider) bind(ctx context.Context, d permissions.Descriptor, actorID, owner string, target permissions.TargetBinding) (Tool, string, error) {
	a, err := verifiedActor(ctx)
	if err != nil {
		return Tool{}, "", err
	}
	if target != (permissions.TargetBinding{}) || owner != Owner || actorID != a.Subject || d.Owner != Owner || d.PrincipalID != a.Subject || d.PrincipalKind != string(a.Kind) || d.Scope != PermissionScope || d.SubjectKind != SubjectKind || d.Source != Source || d.TargetBound || d.Target != (permissions.TargetBinding{}) || d.ID != PermissionPrefix+d.SubjectID || !d.CanRead || d.CanWrite != (p.writer != nil) {
		return Tool{}, "", connect.NewError(connect.CodePermissionDenied, errors.New("host tool permission binding mismatch"))
	}
	revision, tools, err := p.snapshot(ctx)
	if err != nil {
		return Tool{}, "", err
	}
	for _, t := range tools {
		if d.SubjectID == t.Scenario+"/"+t.Name && d.SchemaVersion == SchemaVersion+"/"+t.SchemaDigest+"/"+revision {
			if p.reader == nil {
				return Tool{}, "", connect.NewError(connect.CodeUnavailable, errors.New("OperatorState reader is unavailable"))
			}
			doc, err := p.reader.Get(ctx)
			if err != nil {
				return Tool{}, "", err
			}
			if strings.TrimSpace(operatorstate.Revision(doc)) == "" {
				return Tool{}, "", connect.NewError(connect.CodeDataLoss, errors.New("OperatorState has no revision"))
			}
			if choice, ok := doc.HostTools[t.Name]; ok && choice.OptedIn != nil {
				t.OptedIn = *choice.OptedIn
			}
			return t, operatorstate.Revision(doc), nil
		}
	}
	return Tool{}, "", connect.NewError(connect.CodeFailedPrecondition, errors.New("host tool descriptor is stale or unsupported"))
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	if _, _, err := p.bind(ctx, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: "host-tool:" + req.ActorID + ":" + d.SubjectID}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("host tool read permit mismatch"))
	}
	t, revision, err := p.bind(ctx, d, req.ActorID, req.Owner, req.Target)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	b, err := json.Marshal(t)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	sum := sha256.Sum256(b)
	return permissions.ReadResult{PermissionID: d.ID, Owner: Owner, Target: req.Target, Revision: revision, Value: b, ValueDigest: hex.EncodeToString(sum[:]), ObservedAt: p.now().UTC()}, nil
}

func (p *Provider) Authorize(ctx context.Context, d permissions.Descriptor, req permissions.WriteRequest) (permissions.Decision, error) {
	if !d.CanWrite || p.writer == nil {
		return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, errors.New("host tool opt-in requires an owner CAS writer"))
	}
	if _, _, err := p.bind(ctx, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	if strings.TrimSpace(req.ExpectedRevision) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return permissions.Decision{}, connect.NewError(connect.CodeInvalidArgument, errors.New("expected OperatorState revision and stable request id are required"))
	}
	if _, err := decodeOptIn(req.Value); err != nil {
		return permissions.Decision{}, err
	}
	if p.authorizeMutation == nil {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("host tool opt-in mutation authorization is unavailable"))
	}
	if err := p.authorizeMutation(ctx); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: "host-tool-opt-in:" + req.ActorID + ":" + d.SubjectID}, nil
}
func (p *Provider) Write(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.WriteRequest) (permissions.Receipt, error) {
	if !permit.ValidForWrite(d.ID, req) {
		return permissions.Receipt{}, connect.NewError(connect.CodePermissionDenied, errors.New("host tool write permit mismatch"))
	}
	t, _, err := p.bind(ctx, d, req.ActorID, req.Owner, req.Target)
	if err != nil {
		return permissions.Receipt{}, err
	}
	optedIn, err := decodeOptIn(req.Value)
	if err != nil {
		return permissions.Receipt{}, err
	}
	digest := sha256.Sum256(req.Value)
	requestDigest := hex.EncodeToString(digest[:])
	stored, err := p.writer.CompareAndSetHostTool(ctx, StateMutation{ActorID: req.ActorID, Scenario: t.Scenario, Name: t.Name, ExpectedRevision: req.ExpectedRevision, RequestID: req.IdempotencyKey, RequestDigest: requestDigest, DecisionRef: permit.DecisionRef(), OptedIn: optedIn})
	if err != nil {
		return permissions.Receipt{}, err
	}
	subject := SubjectKind + ":" + t.Scenario + "/" + t.Name
	if stored.RequestID != req.IdempotencyKey || stored.ActorID != req.ActorID || stored.Target != LocalTarget || stored.Subject != subject || stored.ExpectedRevision != req.ExpectedRevision || stored.NewRevision == "" || stored.NewRevision == req.ExpectedRevision || stored.RequestDigest != requestDigest || stored.Outcome == "" || stored.At.IsZero() {
		return permissions.Receipt{}, connect.NewError(connect.CodeDataLoss, errors.New("OperatorState returned an unbound host tool receipt"))
	}
	return permissions.Receipt{ID: "operator-state-host-tool:" + stored.RequestID, PermissionID: d.ID, Owner: d.Owner, ActorID: stored.ActorID, ExpectedRevision: stored.ExpectedRevision, ResultRevision: stored.NewRevision, IdempotencyKey: stored.RequestID, RequestDigest: requestDigest, DecisionRef: permit.DecisionRef(), Outcome: stored.Outcome, At: stored.At.UTC()}, nil
}

func decodeOptIn(raw []byte) (bool, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || len(fields) != 1 || fields["opted_in"] == nil {
		return false, connect.NewError(connect.CodeInvalidArgument, errors.New("host tool permission accepts only an opted_in boolean"))
	}
	var value bool
	if err := json.Unmarshal(fields["opted_in"], &value); err != nil {
		return false, connect.NewError(connect.CodeInvalidArgument, errors.New("host tool opted_in must be boolean"))
	}
	return value, nil
}
