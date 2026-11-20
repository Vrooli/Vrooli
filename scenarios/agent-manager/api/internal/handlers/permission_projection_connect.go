package handlers

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

	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions"
	permissionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions/permissionsv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const permissionProjectionPageLimit = 64
const permissionProjectionMaxRows = 10000

type permissionProjectionHandler struct {
	admin    StandingGrantAdmin
	repoRoot string
}

func NewPermissionProjectionHandler(admin StandingGrantAdmin) (permissionsv1connect.PermissionProjectionServiceHandler, error) {
	root, err := config.RepoRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve canonical scope catalog root: %w", err)
	}
	return &permissionProjectionHandler{admin: admin, repoRoot: root}, nil
}

func (h *permissionProjectionHandler) ListPermissionProjection(ctx context.Context, req *connect.Request[permissionsv1.ListPermissionProjectionRequest]) (*connect.Response[permissionsv1.ListPermissionProjectionResponse], error) {
	actor, owner, err := authorizeStandingGrantProjectionRead(ctx)
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.GetLimit())
	if limit == 0 {
		limit = permissionProjectionPageLimit
	}
	if limit < 1 || limit > permissionProjectionPageLimit {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("permission projection page limit is invalid"))
	}
	offset := 0
	cursorRevision := ""
	if raw := req.Msg.GetCursor(); raw != "" {
		b, e := base64.RawURLEncoding.DecodeString(raw)
		if e != nil || len(b) > 512 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("permission projection cursor is invalid"))
		}
		var c projectionCursor
		if json.Unmarshal(b, &c) != nil || c.Offset < 0 || c.Offset > permissionProjectionMaxRows || c.Revision == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("permission projection cursor is invalid"))
		}
		offset, cursorRevision = c.Offset, c.Revision
	}
	snapshot, err := h.snapshot(ctx, actor, owner)
	if err != nil {
		return nil, err
	}
	if cursorRevision != "" && cursorRevision != snapshot.revision {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("permission projection source changed; restart pagination"))
	}
	if offset > len(snapshot.grants) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("permission projection cursor is outside the current source"))
	}
	end := offset + limit
	if end > len(snapshot.grants) {
		end = len(snapshot.grants)
	}
	out := &permissionsv1.ListPermissionProjectionResponse{Owner: "agent-manager", SourceRevision: snapshot.revision}
	if offset == 0 {
		for _, s := range snapshot.scopes {
			out.ScopeCatalog = append(out.ScopeCatalog, scopeToProto(s))
		}
	}
	for _, g := range snapshot.grants[offset:end] {
		out.Grants = append(out.Grants, grantToProto(g))
	}
	if end < len(snapshot.grants) {
		raw, _ := json.Marshal(projectionCursor{Offset: end, Revision: snapshot.revision})
		out.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return connect.NewResponse(out), nil
}

func (h *permissionProjectionHandler) ReadPermissionProjection(ctx context.Context, req *connect.Request[permissionsv1.ReadPermissionProjectionRequest]) (*connect.Response[permissionsv1.ReadPermissionProjectionResponse], error) {
	actor, owner, err := authorizeStandingGrantProjectionRead(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, err := h.snapshot(ctx, actor, owner)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetExpectedSourceRevision() == "" || req.Msg.GetExpectedSourceRevision() != snapshot.revision {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("permission projection source changed; refresh inventory"))
	}
	out := &permissionsv1.ReadPermissionProjectionResponse{Owner: "agent-manager", SourceRevision: snapshot.revision}
	switch req.Msg.GetKind() {
	case "scope":
		for _, scope := range snapshot.scopes {
			if req.Msg.GetReference() == scope.Value {
				out.Value = &permissionsv1.ReadPermissionProjectionResponse_Scope{Scope: scopeToProto(scope)}
				return connect.NewResponse(out), nil
			}
		}
	case "grant":
		for _, grant := range snapshot.grants {
			if req.Msg.GetReference() == grantRef(grant.ID) {
				out.Value = &permissionsv1.ReadPermissionProjectionResponse_Grant{Grant: grantToProto(grant)}
				return connect.NewResponse(out), nil
			}
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("permission projection kind is unsupported"))
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.New("permission projection reference is unavailable"))
}

type projectionCursor struct {
	Offset   int    `json:"offset"`
	Revision string `json:"revision"`
}
type projectionSnapshot struct {
	revision string
	scopes   []scopecatalog.Scope
	grants   []*domain.StandingGrant
}

func (h *permissionProjectionHandler) snapshot(ctx context.Context, actor identity.Principal, owner string) (projectionSnapshot, error) {
	catalog, err := scopecatalog.Build(h.repoRoot)
	if err != nil {
		return projectionSnapshot{}, connect.NewError(connect.CodeUnavailable, errors.New("canonical scope catalog is unavailable"))
	}
	scopes := make([]scopecatalog.Scope, 0)
	for _, scope := range catalog.Scopes {
		if scope.Scenario == "agent-manager" {
			scopes = append(scopes, scope)
		}
	}
	var grants []*domain.StandingGrant
	if actor.Kind == identity.ActorAgent {
		grants, err = h.admin.ListStandingGrantsForApprover(ctx, owner, permissionProjectionMaxRows+1)
	} else {
		grants, err = h.admin.ListStandingGrants(ctx, permissionProjectionMaxRows+1)
	}
	if err != nil {
		return projectionSnapshot{}, connect.NewError(connect.CodeUnavailable, errors.New("standing grant projection is unavailable"))
	}
	owned := make([]*domain.StandingGrant, 0, len(grants))
	for _, g := range grants {
		if g == nil || g.Grant == nil || (owner != "" && (g.ApprovedBy != owner || g.Grant.Approver != owner)) {
			continue
		}
		owned = append(owned, g)
	}
	if len(owned) > permissionProjectionMaxRows {
		return projectionSnapshot{}, connect.NewError(connect.CodeResourceExhausted, errors.New("standing grant projection exceeds the bounded inventory"))
	}
	source, _ := json.Marshal(struct {
		Scopes []scopecatalog.Scope    `json:"scopes"`
		Grants []*domain.StandingGrant `json:"grants"`
	}{scopes, owned})
	digest := sha256.Sum256(source)
	return projectionSnapshot{revision: "sha256:" + hex.EncodeToString(digest[:]), scopes: scopes, grants: owned}, nil
}

func authorizeStandingGrantProjectionRead(ctx context.Context) (identity.Principal, string, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || !p.IsVerified() {
		return identity.Principal{}, "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified Agent Manager identity required"))
	}
	switch p.Kind {
	case identity.ActorHuman:
		human, err := authn.RequireIndependentHuman(ctx)
		if err != nil {
			return identity.Principal{}, "", connect.NewError(connect.CodePermissionDenied, errors.New("independent human owner required"))
		}
		if _, err := authn.RequireCapability(ctx, "agent-manager:read"); err != nil {
			return identity.Principal{}, "", connect.NewError(connect.CodePermissionDenied, errors.New("Agent Manager read capability required"))
		}
		return human, "", nil
	case identity.ActorAgent:
		owner := strings.TrimSpace(p.OwnerSubject())
		if owner == "" {
			return identity.Principal{}, "", connect.NewError(connect.CodePermissionDenied, errors.New("verified Agent Manager owner is required"))
		}
		return p, owner, nil
	default:
		return identity.Principal{}, "", connect.NewError(connect.CodePermissionDenied, errors.New("verified human or owned agent identity required"))
	}
}

func grantRef(id string) string { sum := sha256.Sum256([]byte(id)); return hex.EncodeToString(sum[:]) }
func scopeToProto(s scopecatalog.Scope) *permissionsv1.PermissionScopeDeclaration {
	return &permissionsv1.PermissionScopeDeclaration{Scenario: s.Scenario, Scope: s.Value, Effect: string(s.Effect), RunEligible: s.RunEligible, Permissions: append([]string(nil), s.Permissions...), Command: s.Command, Service: s.Service, Method: s.Method}
}
func grantToProto(g *domain.StandingGrant) *permissionsv1.StandingGrantMetadata {
	state := g.State
	if state == domain.StandingGrantApproved && !g.Active(time.Now()) {
		state = "expired"
	}
	out := &permissionsv1.StandingGrantMetadata{GrantRef: grantRef(g.ID), Subject: g.Subject, Approver: g.Grant.Approver, Operations: append([]string(nil), g.Grant.Operations...), Targets: append([]string(nil), g.Grant.Targets...), State: state, MaxConcurrentRuns: int32(g.Limits.MaxConcurrentRuns), MaxRunBudgetUsd: g.Limits.MaxRunBudgetUSD, ApprovedAt: timestamppb.New(g.ApprovedAt), ExpiresAt: timestamppb.New(g.ExpiresAt)}
	if g.RevokedAt != nil {
		out.RevokedAt = timestamppb.New(*g.RevokedAt)
	}
	return out
}
