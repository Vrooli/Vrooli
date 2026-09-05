// Package credentialgrant owns the control-plane policy for which non-source
// nodes may receive which credential address. It deliberately contains no
// credential values and no transport code.
package credentialgrant

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Class string

const (
	ClassInfrastructure      Class = "infrastructure"
	ClassPerInstallGenerated Class = "per_install_generated"
	ClassUserPrompt          Class = "user_prompt"
	ClassRemoteFetch         Class = "remote_fetch"
)

type Retention string

const (
	RetentionDurable   Retention = "durable"
	RetentionEphemeral Retention = "ephemeral"
)

type Grant struct {
	ID              string
	NodeID          string
	LogicalID       string
	Field           string
	Class           Class
	Retention       Retention
	Generation      int64
	AckedGeneration int64
	GrantedAt       time.Time
	RevokedAt       time.Time
	ReceiptAt       time.Time
	ReceiptAccepted bool
	ReceiptReason   string
}

type CreateInput struct {
	NodeID     string
	LogicalID  string
	Field      string
	Class      Class
	Retention  Retention
	Generation int64
}

type Repository interface {
	Create(context.Context, Grant) (Grant, error)
	List(context.Context, string) ([]Grant, error)
	Revoke(context.Context, string) error
	Ack(context.Context, string, int64) error
}

// GenerationRepository is an optional extension implemented by the durable
// SQLite repository. Keeping it an extension preserves small policy fakes while
// making source rotation atomic in production.
type GenerationRepository interface {
	BumpGeneration(context.Context, string, string) (int64, error)
	SetGrantGeneration(context.Context, string, string, int64) error
}

type ReceiptRepository interface {
	RecordReceipt(context.Context, string, int64, bool, string, time.Time) error
}

type NodeKindResolver interface {
	NodeKind(context.Context, string) (string, error)
}

type Service struct {
	repo  Repository
	nodes NodeKindResolver
	now   func() time.Time
}

func NewService(repo Repository, nodes NodeKindResolver, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, nodes: nodes, now: now}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Grant, error) {
	nodeID := strings.TrimSpace(in.NodeID)
	logicalID := strings.TrimSpace(in.LogicalID)
	field := strings.TrimSpace(in.Field)
	if nodeID == "" || logicalID == "" || field == "" {
		return Grant{}, fmt.Errorf("node_id, logical_id, and field are required")
	}
	if !validClass(in.Class) {
		return Grant{}, fmt.Errorf("invalid credential class %q", in.Class)
	}
	if in.Class == ClassPerInstallGenerated {
		return Grant{}, fmt.Errorf("per_install_generated credentials are generated locally and cannot be distributed")
	}
	if in.Retention != RetentionDurable && in.Retention != RetentionEphemeral {
		return Grant{}, fmt.Errorf("invalid retention %q", in.Retention)
	}
	if in.Retention == RetentionDurable && in.Class == ClassInfrastructure {
		return Grant{}, fmt.Errorf("infrastructure credentials cannot receive durable grants")
	}
	if s.nodes == nil {
		return Grant{}, fmt.Errorf("node kind resolver is not configured")
	}
	kind, err := s.nodes.NodeKind(ctx, nodeID)
	if err != nil {
		return Grant{}, err
	}
	if strings.TrimSpace(kind) == "control_plane" {
		return Grant{}, fmt.Errorf("control-plane node cannot receive credential grants")
	}
	generation := in.Generation
	if generation <= 0 {
		generation = 1
	}
	return s.repo.Create(ctx, Grant{NodeID: nodeID, LogicalID: logicalID, Field: field, Class: in.Class, Retention: in.Retention, Generation: generation, GrantedAt: s.now().UTC()})
}

func (s *Service) List(ctx context.Context, nodeID string) ([]Grant, error) {
	return s.repo.List(ctx, strings.TrimSpace(nodeID))
}

// ActiveGrant reports metadata only. Dispatch uses it to authorize an
// ephemeral job injection before creating a run; no credential value crosses
// this seam.
func (s *Service) ActiveGrant(ctx context.Context, nodeID, logicalID, field string) (string, string, bool, error) {
	grants, err := s.repo.List(ctx, strings.TrimSpace(nodeID))
	if err != nil {
		return "", "", false, err
	}
	for _, grant := range grants {
		if grant.LogicalID == strings.TrimSpace(logicalID) && grant.Field == strings.TrimSpace(field) {
			return string(grant.Class), string(grant.Retention), true, nil
		}
	}
	return "", "", false, nil
}

func (s *Service) Get(ctx context.Context, id string) (Grant, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Grant{}, fmt.Errorf("grant id is required")
	}
	grants, err := s.repo.List(ctx, "")
	if err != nil {
		return Grant{}, err
	}
	for _, grant := range grants {
		if grant.ID == id {
			return grant, nil
		}
	}
	return Grant{}, fmt.Errorf("grant %q not found", id)
}

// Rotate advances the source generation and marks every active grant for the
// address as needing the new generation. Delivery is deliberately separate:
// callers can report online pushes and leave offline nodes for SyncNode.
func (s *Service) Rotate(ctx context.Context, logicalID, field string) (int64, []Grant, error) {
	logicalID, field = strings.TrimSpace(logicalID), strings.TrimSpace(field)
	if logicalID == "" || field == "" {
		return 0, nil, fmt.Errorf("logical_id and field are required")
	}
	generations, ok := s.repo.(GenerationRepository)
	if !ok {
		return 0, nil, fmt.Errorf("credential generation repository is unavailable")
	}
	generation, err := generations.BumpGeneration(ctx, logicalID, field)
	if err != nil {
		return 0, nil, err
	}
	grants, err := s.repo.List(ctx, "")
	if err != nil {
		return 0, nil, err
	}
	for _, grant := range grants {
		if grant.LogicalID != logicalID || grant.Field != field {
			continue
		}
		if err := generations.SetGrantGeneration(ctx, grant.ID, grant.NodeID, generation); err != nil {
			return 0, nil, err
		}
		grant.Generation = generation
		grant.AckedGeneration = 0
	}
	updated, err := s.repo.List(ctx, "")
	if err != nil {
		return 0, nil, err
	}
	return generation, updated, nil
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("grant id is required")
	}
	return s.repo.Revoke(ctx, id)
}

func (s *Service) Ack(ctx context.Context, id string, generation int64) error {
	if generation <= 0 {
		return fmt.Errorf("generation must be positive")
	}
	return s.repo.Ack(ctx, id, generation)
}

// RecordCredentialReceipt is the node-facing acknowledgement seam. It binds
// the receipt to the authenticated node's active grant before advancing the
// durable acknowledged generation; an arbitrary node cannot acknowledge a
// different node's grant by guessing its id.
func (s *Service) RecordCredentialReceipt(ctx context.Context, id, nodeID string, generation int64, accepted bool, reason string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(nodeID) == "" {
		return fmt.Errorf("grant id and node id are required")
	}
	if generation <= 0 {
		return fmt.Errorf("generation must be positive")
	}
	grants, err := s.repo.List(ctx, strings.TrimSpace(nodeID))
	if err != nil {
		return err
	}
	var found bool
	for _, grant := range grants {
		if grant.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("grant %q is not active for node %q", id, nodeID)
	}
	if !accepted {
		// Rejection is still a valid, value-free receipt. It must not advance
		// acked_generation, but the caller can retain the reason in its audit
		// stream without treating it as a transport failure.
		if receipts, ok := s.repo.(ReceiptRepository); ok {
			return receipts.RecordReceipt(ctx, id, generation, false, reason, s.now().UTC())
		}
		return nil
	}
	if err := s.Ack(ctx, id, generation); err != nil {
		return err
	}
	if receipts, ok := s.repo.(ReceiptRepository); ok {
		return receipts.RecordReceipt(ctx, id, generation, true, "", s.now().UTC())
	}
	return nil
}

func validClass(class Class) bool {
	switch class {
	case ClassInfrastructure, ClassPerInstallGenerated, ClassUserPrompt, ClassRemoteFetch:
		return true
	default:
		return false
	}
}

func validateGrantMetadata(grant Grant) error {
	if grant.ID == "" || grant.NodeID == "" || grant.LogicalID == "" || grant.Field == "" {
		return fmt.Errorf("grant metadata requires id, node_id, logical_id, and field")
	}
	if !validClass(grant.Class) {
		return fmt.Errorf("invalid credential class %q", grant.Class)
	}
	if grant.Class == ClassPerInstallGenerated {
		return fmt.Errorf("per_install_generated credentials are generated locally and cannot be distributed")
	}
	if grant.Retention != RetentionDurable && grant.Retention != RetentionEphemeral {
		return fmt.Errorf("invalid credential retention %q", grant.Retention)
	}
	if grant.Retention == RetentionDurable && grant.Class == ClassInfrastructure {
		return fmt.Errorf("infrastructure credentials cannot receive durable grants")
	}
	if grant.Generation <= 0 {
		return fmt.Errorf("credential grant generation must be positive")
	}
	return nil
}
