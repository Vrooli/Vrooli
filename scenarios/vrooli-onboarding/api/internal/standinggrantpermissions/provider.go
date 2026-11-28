// Package standinggrantpermissions projects Agent Manager's owner-defined
// standing-grant read model into onboarding. Grant lifecycle and authority
// remain exclusively in Agent Manager.
package standinggrantpermissions

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID       = "onboarding.agent-manager-standing-grants"
	Owner            = "agent-manager"
	PermissionPrefix = "agent-manager/standing-grant/"
	Source           = "agent-manager/owner-standing-grant-projection/v1"
	SchemaVersion    = "agent-manager/standing-grant-metadata/v1"
	maxInventory     = 10000
	pageSize         = 64
)

type Grant struct {
	ID                string     `json:"id"`
	Subject           string     `json:"subject"`
	Approver          string     `json:"approver"`
	Operations        []string   `json:"operations"`
	Targets           []string   `json:"targets"`
	State             string     `json:"state"`
	MaxConcurrentRuns int        `json:"max_concurrent_runs"`
	MaxRunBudgetUSD   float64    `json:"max_run_budget_usd"`
	ApprovedAt        time.Time  `json:"approved_at"`
	ExpiresAt         time.Time  `json:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
}
type Page struct {
	Owner          string               `json:"owner"`
	SourceRevision string               `json:"source_revision"`
	Scopes         []scopecatalog.Scope `json:"scopes"`
	Grants         []Grant              `json:"grants"`
	NextCursor     string               `json:"next_cursor,omitempty"`
}
type OwnerValue struct {
	Owner          string
	SourceRevision string
	Scope          *scopecatalog.Scope
	Grant          *Grant
}
type OwnerReader interface {
	List(context.Context, int, string) (Page, error)
	Read(context.Context, string, string, string) (OwnerValue, error)
}
type Provider struct {
	owner OwnerReader
	now   func() time.Time
}

func New(owner OwnerReader) *Provider {
	return &Provider{owner: owner, now: func() time.Time { return time.Now().UTC() }}
}

var _ permissions.Provider = (*Provider)(nil)

func (*Provider) ProviderID() string                 { return ProviderID }
func (*Provider) Owner() string                      { return Owner }
func (*Provider) MatchesPermissionID(id string) bool { return strings.HasPrefix(id, PermissionPrefix) }

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	actor, err := verifiedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if p.owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("Agent Manager owner projection is unavailable"))
	}
	out := make([]permissions.Descriptor, 0)
	cursor := ""
	seen := map[string]bool{}
	sourceRevision := ""
	for count := 0; count < maxInventory/pageSize; count++ {
		page, err := p.owner.List(ctx, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		if page.Owner != Owner || strings.TrimSpace(page.SourceRevision) == "" {
			return nil, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned an invalid permission page"))
		}
		if sourceRevision == "" {
			sourceRevision = page.SourceRevision
		} else if sourceRevision != page.SourceRevision {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Agent Manager permission source changed during enumeration"))
		}
		if cursor == "" {
			for _, scope := range page.Scopes {
				if scope.Scenario != Owner || strings.TrimSpace(scope.Value) == "" {
					return nil, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned an invalid scope catalog entry"))
				}
				id := permissionID("scope", scope.Value, page.SourceRevision)
				if seen[id] {
					return nil, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned duplicate scope catalog metadata"))
				}
				seen[id] = true
				out = append(out, permissions.Descriptor{Version: permissions.ContractVersion, ID: id, Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "agent-manager:read", SubjectKind: "scope-declaration", SubjectID: scope.Value, Source: Source, SchemaVersion: "agent-manager/scope-catalog/v1#" + page.SourceRevision, CanRead: true, CanWrite: false, Sensitivity: "operator"})
			}
		}
		for _, g := range page.Grants {
			if err := validateGrant(g); err != nil {
				return nil, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned invalid grant metadata"))
			}
			if actor.Kind == identity.ActorAgent && (actor.OwnerSubject() != g.Approver) {
				continue
			}
			id := permissionID("grant", g.ID, page.SourceRevision)
			if seen[id] {
				return nil, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned duplicate grant metadata"))
			}
			seen[id] = true
			out = append(out, permissions.Descriptor{Version: permissions.ContractVersion, ID: id, Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "agent-manager:read", SubjectKind: "standing-grant", SubjectID: g.ID, Source: Source, SchemaVersion: SchemaVersion + "#" + page.SourceRevision, CanRead: true, CanWrite: false, Sensitivity: "operator"})
		}
		if page.NextCursor == "" {
			return out, nil
		}
		if page.NextCursor == cursor || len(page.Grants) == 0 {
			return nil, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned invalid pagination cursor"))
		}
		cursor = page.NextCursor
	}
	return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("Agent Manager grant inventory exceeds onboarding projection limit"))
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	a, err := verifiedPrincipal(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	_, id, revision, ok := decodePermissionID(d.ID)
	if !ok || req.Owner != Owner || req.ActorID != a.Subject || d.PrincipalID != a.Subject || d.SubjectID != id || !strings.HasSuffix(d.SchemaVersion, "#"+revision) {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("standing grant is outside the verified Agent Manager owner scope"))
	}
	if a.Kind == identity.ActorAgent && a.OwnerSubject() == "" {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("verified Agent Manager owner is required"))
	}
	return permissions.Decision{Allowed: true, DecisionRef: "agent-manager-owner:" + a.Subject}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("standing grant read permit mismatch"))
	}
	kind, subjectID, revision, ok := decodePermissionID(d.ID)
	if !ok || p.owner == nil {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("standing grant reference unavailable"))
	}
	if kind != "scope" && kind != "grant" {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("unsupported Agent Manager permission kind"))
	}
	value, err := p.owner.Read(ctx, kind, subjectID, revision)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	if value.Owner != Owner || value.SourceRevision != revision {
		return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("Agent Manager permission source changed; refresh inventory"))
	}
	if kind == "scope" && value.Scope != nil && value.Scope.Value == subjectID {
		return ownerReadResult(d, req, revision, value.Scope, p.now().UTC())
	}
	if kind == "grant" && value.Grant != nil && value.Grant.ID == subjectID {
		if err := validateGrant(*value.Grant); err != nil {
			return permissions.ReadResult{}, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned invalid grant metadata"))
		}
		return ownerReadResult(d, req, revision, value.Grant, p.now().UTC())
	}
	return permissions.ReadResult{}, connect.NewError(connect.CodeNotFound, errors.New("permission projection reference is unavailable in the current Agent Manager owner scope"))
}
func (*Provider) Authorize(context.Context, permissions.Descriptor, permissions.WriteRequest) (permissions.Decision, error) {
	return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, errors.New("standing grant issuance and revocation remain Agent Manager owner-only operations"))
}
func (*Provider) Write(context.Context, permissions.Descriptor, permissions.Permit, permissions.WriteRequest) (permissions.Receipt, error) {
	return permissions.Receipt{}, connect.NewError(connect.CodeUnimplemented, errors.New("standing grant issuance and revocation remain Agent Manager owner-only operations"))
}

func verifiedPrincipal(ctx context.Context) (identity.Principal, error) {
	a, ok := identity.PrincipalFromContext(ctx)
	if !ok || !a.IsVerified() || a.Subject == "" || (a.Kind != identity.ActorHuman && a.Kind != identity.ActorAgent) {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified human or owned Agent Manager identity required"))
	}
	if a.Kind == identity.ActorAgent && a.OwnerSubject() == "" {
		return identity.Principal{}, connect.NewError(connect.CodePermissionDenied, errors.New("verified Agent Manager owner is required"))
	}
	return a, nil
}
func validateGrant(g Grant) error {
	if strings.TrimSpace(g.ID) == "" || strings.TrimSpace(g.Subject) == "" || strings.TrimSpace(g.Approver) == "" || strings.TrimSpace(g.State) == "" || len(g.Operations) == 0 || len(g.Targets) == 0 || g.ExpiresAt.IsZero() {
		return errors.New("incomplete standing grant")
	}
	switch g.State {
	case "approved", "revoked", "expired":
	default:
		return fmt.Errorf("unknown standing grant state %q", g.State)
	}
	return nil
}
func ownerReadResult(d permissions.Descriptor, req permissions.ReadRequest, revision string, value any, at time.Time) (permissions.ReadResult, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return permissions.ReadResult{}, connect.NewError(connect.CodeInternal, errors.New("encode Agent Manager permission metadata"))
	}
	sum := sha256.Sum256(encoded)
	return permissions.ReadResult{PermissionID: d.ID, Owner: Owner, Target: req.Target, Revision: revision, Value: encoded, ValueDigest: hex.EncodeToString(sum[:]), ObservedAt: at}, nil
}

func permissionID(kind, id, revision string) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(kind + "\x00" + id + "\x00" + revision))
	return PermissionPrefix + payload
}
func decodePermissionID(value string) (string, string, string, bool) {
	raw, ok := strings.CutPrefix(value, PermissionPrefix)
	if !ok || raw == "" || len(raw) > 4096 {
		return "", "", "", false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", "", "", false
	}
	kind, rest, ok := strings.Cut(string(decoded), "\x00")
	if !ok {
		return "", "", "", false
	}
	id, revision, ok := strings.Cut(rest, "\x00")
	return kind, id, revision, ok && (kind == "grant" || kind == "scope") && strings.TrimSpace(id) != "" && strings.TrimSpace(revision) != "" && !strings.Contains(revision, "\x00")
}
