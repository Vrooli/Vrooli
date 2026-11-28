package approveddependencies

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
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	PermissionProviderID = "onboarding.approved-dependencies"
	PermissionOwner      = "scenario-dependency-analyzer"
	PermissionID         = "onboarding/approved-dependencies"
	PermissionScope      = "scenario-dependency-analyzer:read"
	PermissionSubject    = "approved-dependency-registry"
	PermissionSchema     = "approved-dependencies/v1"
)

// PermissionProvider exposes the repository-owned dependency registry as a
// read-only onboarding permission. Approval and edits remain with the review
// workflow; this adapter never infers approval or offers a mutation path.
type PermissionProvider struct{ source *Provider }

func NewPermissionProvider(source *Provider) *PermissionProvider {
	return &PermissionProvider{source: source}
}

var _ permissions.Provider = (*PermissionProvider)(nil)

func (*PermissionProvider) ProviderID() string                 { return PermissionProviderID }
func (*PermissionProvider) Owner() string                      { return PermissionOwner }
func (*PermissionProvider) MatchesPermissionID(id string) bool { return id == PermissionID }

func (p *PermissionProvider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	principal, err := verifiedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, err := p.read(ctx, principal.Subject)
	if err != nil {
		return nil, err
	}
	return []permissions.Descriptor{{
		Version: permissions.ContractVersion, ID: PermissionID, Owner: PermissionOwner,
		PrincipalKind: string(principal.Kind), PrincipalID: principal.Subject, Scope: PermissionScope,
		SubjectKind: PermissionSubject, SubjectID: SourcePath, Source: SourcePath,
		SchemaVersion: permissionRevision(snapshot), CanRead: true, CanWrite: false, Sensitivity: "operator",
	}}, nil
}

func (p *PermissionProvider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	principal, err := verifiedPrincipal(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if req.Target != (permissions.TargetBinding{}) || req.Owner != PermissionOwner || req.ActorID != principal.Subject || d.ID != PermissionID || d.Owner != PermissionOwner || d.PrincipalID != principal.Subject || d.PrincipalKind != string(principal.Kind) || d.Scope != PermissionScope || d.SubjectKind != PermissionSubject || d.SubjectID != SourcePath || d.Source != SourcePath || !d.CanRead || d.CanWrite {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("approved dependency permission binding mismatch"))
	}
	snapshot, err := p.read(ctx, principal.Subject)
	if err != nil {
		return permissions.Decision{}, err
	}
	if d.SchemaVersion != permissionRevision(snapshot) {
		return permissions.Decision{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("approved dependency registry changed; refresh permission inventory"))
	}
	return permissions.Decision{Allowed: true, DecisionRef: "approved-dependencies:" + principal.Subject + ":" + snapshot.SourceDigest}, nil
}

func (p *PermissionProvider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("approved dependency read permit mismatch"))
	}
	principal, err := verifiedPrincipal(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	snapshot, err := p.read(ctx, principal.Subject)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	revision := permissionRevision(snapshot)
	if d.SchemaVersion != revision {
		return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("approved dependency registry changed; refresh permission inventory"))
	}
	value, err := json.Marshal(snapshot)
	if err != nil {
		return permissions.ReadResult{}, connect.NewError(connect.CodeInternal, errors.New("encode approved dependency snapshot"))
	}
	digest := sha256.Sum256(value)
	return permissions.ReadResult{PermissionID: d.ID, Owner: PermissionOwner, Revision: revision, Value: value, ValueDigest: hex.EncodeToString(digest[:]), ObservedAt: time.Now().UTC()}, nil
}

func (*PermissionProvider) Authorize(context.Context, permissions.Descriptor, permissions.WriteRequest) (permissions.Decision, error) {
	return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, ErrReadOnly)
}
func (*PermissionProvider) Write(context.Context, permissions.Descriptor, permissions.Permit, permissions.WriteRequest) (permissions.Receipt, error) {
	return permissions.Receipt{}, connect.NewError(connect.CodeUnimplemented, ErrReadOnly)
}

func (p *PermissionProvider) read(ctx context.Context, principal string) (Snapshot, error) {
	if p == nil || p.source == nil {
		return Snapshot{}, connect.NewError(connect.CodeUnavailable, errors.New("approved dependency source is unavailable"))
	}
	snapshot, err := p.source.Read(ctx, principal)
	if err != nil {
		return Snapshot{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("approved dependency source unavailable: %w", err))
	}
	return snapshot, nil
}

func permissionRevision(snapshot Snapshot) string {
	return PermissionSchema + "#" + snapshot.SourceDigest + "#" + snapshot.SourceRevision
}

func verifiedPrincipal(ctx context.Context) (identity.Principal, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || strings.TrimSpace(principal.Subject) == "" {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding principal required"))
	}
	return principal, nil
}
