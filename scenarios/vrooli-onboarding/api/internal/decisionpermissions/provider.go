// Package decisionpermissions exposes Notification Hub's safe pending-decision
// owner projection through onboarding's typed permission registry. Human
// answer, denial and resolution remain in Notification Hub's existing UX/API.
package decisionpermissions

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
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID       = "onboarding.operator-decisions"
	Owner            = "notification-hub"
	PermissionPrefix = "onboarding/operator-decision/"
	Source           = "notification-hub/owner-pending-decision-projection/v1"
	SchemaVersion    = "notification-hub/pending-decision-reference/v1"
	SubjectKind      = "pending-operator-decision"
	Scope            = "notification-hub:pending-decisions:read"
	maxInventory     = 500
	pageSize         = 100
)

// Reference is the complete allowlist accepted from the decision owner.
type Reference struct {
	DecisionRef     string `json:"decision_ref"`
	OwnerID         string `json:"owner_id"`
	Source          string `json:"source"`
	SourceEventType string `json:"source_event_type"`
	Revision        string `json:"revision"`
	State           string `json:"state"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	Deadline        string `json:"deadline,omitempty"`
	Urgency         string `json:"urgency,omitempty"`
}

type Page struct {
	Decisions  []Reference `json:"decisions"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type OwnerProjection interface {
	ListPending(context.Context, int, string) (Page, error)
	ReadPending(context.Context, string, string) (Reference, error)
}

type Provider struct {
	owner OwnerProjection
	now   func() time.Time
}

func New(owner OwnerProjection) *Provider {
	return &Provider{owner: owner, now: func() time.Time { return time.Now().UTC() }}
}

var _ permissions.Provider = (*Provider)(nil)

func (*Provider) ProviderID() string                 { return ProviderID }
func (*Provider) Owner() string                      { return Owner }
func (*Provider) MatchesPermissionID(id string) bool { return strings.HasPrefix(id, PermissionPrefix) }

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	actor, err := verifiedHuman(ctx)
	if err != nil {
		return nil, err
	}
	if p.owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("decision owner projection is unavailable"))
	}
	out := make([]permissions.Descriptor, 0)
	cursor := ""
	seen := map[string]bool{}
	seenCursors := map[string]bool{"": true}
	for pageNumber := 0; pageNumber < maxInventory/pageSize; pageNumber++ {
		page, err := p.owner.ListPending(ctx, pageSize, cursor)
		if err != nil {
			return nil, ownerError(err)
		}
		for _, ref := range page.Decisions {
			if err := validateReference(ref, actor.Subject); err != nil {
				return nil, connect.NewError(connect.CodeDataLoss, errors.New("decision owner returned an invalid pending reference"))
			}
			if seen[ref.DecisionRef] {
				return nil, connect.NewError(connect.CodeDataLoss, errors.New("decision owner returned a duplicate reference"))
			}
			seen[ref.DecisionRef] = true
			id := permissionID(ref.DecisionRef, ref.Revision)
			out = append(out, permissions.Descriptor{
				Version: permissions.ContractVersion, ID: id, Owner: Owner,
				PrincipalKind: "human", PrincipalID: actor.Subject, Scope: Scope,
				SubjectKind: SubjectKind, SubjectID: ref.DecisionRef, Source: Source,
				SchemaVersion: SchemaVersion + "#" + ref.Revision,
				CanRead:       true, CanWrite: false, Sensitivity: "operator",
			})
		}
		if page.NextCursor == "" {
			return out, nil
		}
		if len(page.Decisions) == 0 || seenCursors[page.NextCursor] {
			return nil, connect.NewError(connect.CodeDataLoss, errors.New("decision owner returned an invalid page cursor"))
		}
		cursor = page.NextCursor
		seenCursors[cursor] = true
	}
	return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("pending decision inventory exceeds the bounded onboarding projection; use the Notification Hub owner inbox"))
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	actor, err := verifiedHuman(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	ref, revision, ok := decodePermissionID(d.ID)
	if !ok || req.Owner != Owner || req.ActorID != actor.Subject || d.PrincipalID != actor.Subject || d.SubjectID != ref || !strings.HasSuffix(d.SchemaVersion, "#"+revision) {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("decision reference is outside the verified owner's scope"))
	}
	return permissions.Decision{Allowed: true, DecisionRef: "notification-hub-owner:" + actor.Subject}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("decision read permit is invalid"))
	}
	ref, revision, ok := decodePermissionID(d.ID)
	if !ok || p.owner == nil {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("decision reference is unavailable"))
	}
	value, err := p.owner.ReadPending(ctx, ref, revision)
	if err != nil {
		return permissions.ReadResult{}, ownerError(err)
	}
	if err := validateReference(value, d.PrincipalID); err != nil || value.DecisionRef != ref || value.Revision != revision {
		return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("pending decision changed; refresh the permission inventory"))
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return permissions.ReadResult{}, connect.NewError(connect.CodeInternal, errors.New("encode pending decision reference"))
	}
	digest := sha256.Sum256(encoded)
	return permissions.ReadResult{
		PermissionID: d.ID, Owner: Owner, Revision: revision,
		Value: encoded, ValueDigest: hex.EncodeToString(digest[:]), ObservedAt: p.now().UTC(),
	}, nil
}

func (*Provider) Authorize(context.Context, permissions.Descriptor, permissions.WriteRequest) (permissions.Decision, error) {
	return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, errors.New("operator decisions remain human-only and cannot be written through onboarding permissions"))
}
func (*Provider) Write(context.Context, permissions.Descriptor, permissions.Permit, permissions.WriteRequest) (permissions.Receipt, error) {
	return permissions.Receipt{}, connect.NewError(connect.CodeUnimplemented, errors.New("operator decisions remain human-only and cannot be written through onboarding permissions"))
}

func permissionID(ref, revision string) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(ref + "\x00" + revision))
	return PermissionPrefix + payload
}

func decodePermissionID(id string) (string, string, bool) {
	encoded, ok := strings.CutPrefix(id, PermissionPrefix)
	if !ok || encoded == "" || len(encoded) > 4096 {
		return "", "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", false
	}
	ref, revision, ok := strings.Cut(string(payload), "\x00")
	return ref, revision, ok && strings.TrimSpace(ref) != "" && strings.TrimSpace(revision) != "" && !strings.Contains(revision, "\x00")
}

func validateReference(ref Reference, expectedOwner string) error {
	if strings.TrimSpace(ref.DecisionRef) == "" || ref.OwnerID != expectedOwner ||
		strings.TrimSpace(ref.Source) == "" || !strings.HasSuffix(ref.SourceEventType, ".decision_requested.v1") ||
		strings.TrimSpace(ref.Revision) == "" || ref.State != "pending" ||
		strings.TrimSpace(ref.CreatedAt) == "" || strings.TrimSpace(ref.UpdatedAt) == "" {
		return errors.New("invalid owner decision reference")
	}
	if ref.Urgency != "" && ref.Urgency != "low" && ref.Urgency != "normal" && ref.Urgency != "high" && ref.Urgency != "critical" {
		return errors.New("invalid owner decision urgency")
	}
	return nil
}

func verifiedHuman(ctx context.Context) (identity.Principal, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsHuman() {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified human decision owner required"))
	}
	return principal, nil
}

func ownerError(err error) error {
	if err == nil {
		return nil
	}
	if code := connect.CodeOf(err); code != connect.CodeUnknown {
		return err
	}
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("notification-hub decision owner is unavailable: %w", err))
}
