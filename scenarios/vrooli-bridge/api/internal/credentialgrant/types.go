// Package credentialgrant owns the control-plane policy for which non-source
// nodes may receive which credential address. It deliberately contains no
// credential values and no transport code.
package credentialgrant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	Revision        int64
	AckedGeneration int64
	GrantedAt       time.Time
	RevokedAt       time.Time
	ReceiptAt       time.Time
	ReceiptAccepted bool
	ReceiptReason   string
	PurgeState      string
	PurgeReceiptAt  time.Time
	PurgeAccepted   bool
	PurgeReason     string
}

type CreateInput struct {
	NodeID     string
	LogicalID  string
	Field      string
	Class      Class
	Retention  Retention
	Generation int64
}

type MutationBinding struct {
	RequestID            string
	ActorID              string
	NodeID               string
	EnrollmentGeneration int64
	Subject              string
	ExpectedRevision     int64
	PayloadDigest        string
}

type MutationReceipt struct {
	RequestID            string
	ActorID              string
	NodeID               string
	EnrollmentGeneration int64
	Subject              string
	ExpectedRevision     int64
	ResultRevision       int64
	PayloadDigest        string
	GrantID              string
	Outcome              string
	DeliveryState        string
	CreatedAt            time.Time
	Replayed             bool
}

const (
	CreateGrantSubject = "credential-grant.create"
	RevokeGrantSubject = "credential-grant.revoke"
)

type MutationRepository interface {
	CreateWithReceipt(context.Context, Grant, MutationBinding) (Grant, MutationReceipt, error)
	RevokeWithReceipt(context.Context, string, MutationBinding) (Grant, MutationReceipt, error)
}

type MutationDeliveryRepository interface {
	SetMutationDeliveryState(context.Context, string, string) error
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
	ReactivateGrant(context.Context, string, string, int64) error
}

type ReceiptRepository interface {
	RecordReceipt(context.Context, string, int64, bool, string, time.Time) error
}

// RevokedRepository exposes revoked metadata for node reconnect. It is
// separate from Repository.List so ordinary owner inventory remains an active
// grant view.
type RevokedRepository interface {
	ListRevoked(context.Context, string) ([]Grant, error)
}

// PurgeReceiptRepository records the node's acknowledgement after a revoke.
// It never accepts a credential value and keeps remote purge evidence separate
// from delivery receipts for a still-active grant.
type PurgeReceiptRepository interface {
	RecordPurgeReceipt(context.Context, string, string, int64, bool, string, time.Time) error
}

type NodeKindResolver interface {
	NodeKind(context.Context, string) (string, error)
}

// NodeGenerationResolver binds metadata projections to Bridge's current node
// enrollment. It is optional for older fakes but mandatory for metadata RPCs.
type NodeGenerationResolver interface {
	NodeGeneration(context.Context, string) (generation int64, revoked bool, err error)
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

func (s *Service) CreateWithReceipt(ctx context.Context, in CreateInput, binding MutationBinding) (Grant, MutationReceipt, error) {
	nodeID, logicalID, field := strings.TrimSpace(in.NodeID), strings.TrimSpace(in.LogicalID), strings.TrimSpace(in.Field)
	if nodeID == "" || logicalID == "" || field == "" || !validClass(in.Class) || in.Class == ClassPerInstallGenerated || (in.Retention != RetentionDurable && in.Retention != RetentionEphemeral) || (in.Retention == RetentionDurable && in.Class == ClassInfrastructure) {
		return Grant{}, MutationReceipt{}, fmt.Errorf("invalid credential grant request")
	}
	if binding.NodeID != nodeID || binding.ActorID == "" || binding.RequestID == "" || binding.EnrollmentGeneration <= 0 || binding.Subject != CreateGrantSubject || binding.ExpectedRevision != 0 || !validPayloadDigest(binding.PayloadDigest) {
		return Grant{}, MutationReceipt{}, fmt.Errorf("credential grant mutation binding is incomplete or invalid")
	}
	if s.nodes == nil {
		return Grant{}, MutationReceipt{}, fmt.Errorf("node kind resolver is not configured")
	}
	kind, err := s.nodes.NodeKind(ctx, nodeID)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if strings.TrimSpace(kind) == "control_plane" {
		return Grant{}, MutationReceipt{}, fmt.Errorf("control-plane node cannot receive credential grants")
	}
	resolver, ok := s.nodes.(NodeGenerationResolver)
	if !ok {
		return Grant{}, MutationReceipt{}, fmt.Errorf("node enrollment-generation resolver is unavailable")
	}
	generation, revoked, err := resolver.NodeGeneration(ctx, nodeID)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if revoked || generation != binding.EnrollmentGeneration {
		return Grant{}, MutationReceipt{}, ErrStaleNodeGeneration{NodeID: nodeID}
	}
	repository, ok := s.repo.(MutationRepository)
	if !ok {
		return Grant{}, MutationReceipt{}, fmt.Errorf("durable credential mutation receipts are unavailable")
	}
	grant := Grant{NodeID: nodeID, LogicalID: logicalID, Field: field, Class: in.Class, Retention: in.Retention, Generation: in.Generation, GrantedAt: s.now().UTC(), Revision: 1}
	if grant.Generation <= 0 {
		grant.Generation = 1
	}
	return repository.CreateWithReceipt(ctx, grant, binding)
}

func (s *Service) List(ctx context.Context, nodeID string) ([]Grant, error) {
	return s.repo.List(ctx, strings.TrimSpace(nodeID))
}

// ListMetadata returns only Bridge-owned grant status after checking the
// caller's expected current enrollment generation. It performs no delivery,
// presence, secret-resolution, or agent transport work.
func (s *Service) ListMetadata(ctx context.Context, nodeID string, expectedGeneration int64) ([]Grant, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" || expectedGeneration <= 0 {
		return nil, fmt.Errorf("node_id and positive enrollment_generation are required")
	}
	resolver, ok := s.nodes.(NodeGenerationResolver)
	if !ok {
		return nil, fmt.Errorf("node enrollment-generation resolver is unavailable")
	}
	generation, revoked, err := resolver.NodeGeneration(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if revoked || generation <= 0 || generation != expectedGeneration {
		return nil, ErrStaleNodeGeneration{NodeID: nodeID}
	}
	active, err := s.repo.List(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	var revokedGrants []Grant
	if repository, ok := s.repo.(RevokedRepository); ok {
		revokedGrants, err = repository.ListRevoked(ctx, nodeID)
		if err != nil {
			return nil, err
		}
	}
	grants := append(active, revokedGrants...)
	for _, grant := range grants {
		if grant.NodeID != nodeID {
			return nil, fmt.Errorf("credential grant belongs to another node")
		}
	}
	return grants, nil
}

type ErrStaleNodeGeneration struct{ NodeID string }

func (e ErrStaleNodeGeneration) Error() string { return "credential grant target enrollment changed" }

// MetadataVersion is a deterministic version of the allowlisted status fields.
// It does not incorporate or expose any receipt reason or credential value.
func MetadataVersion(g Grant) string {
	if g.Revision > 0 {
		return fmt.Sprintf("%d", g.Revision)
	}
	material := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%d\x00%d\x00%s\x00%t\x00%s\x00%s\x00%t", g.ID, g.NodeID, g.LogicalID, g.Field, g.Class, g.Generation, g.AckedGeneration, g.RevokedAt.UTC().Format(time.RFC3339Nano), g.ReceiptAccepted, g.PurgeState, g.PurgeReceiptAt.UTC().Format(time.RFC3339Nano), g.PurgeAccepted)
	sum := sha256.Sum256([]byte(material))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validPayloadDigest(value string) bool {
	return len(value) == 71 && strings.HasPrefix(value, "sha256:")
}

// Active returns the existing grant for an address on a node, when present.
// Answering an existing grant is an owner-authorized re-delivery operation;
// it must not create a second row because the node/address identity is unique.
func (s *Service) Active(ctx context.Context, nodeID, logicalID, field string) (Grant, bool, error) {
	grants, err := s.repo.List(ctx, strings.TrimSpace(nodeID))
	if err != nil {
		return Grant{}, false, err
	}
	logicalID, field = strings.TrimSpace(logicalID), strings.TrimSpace(field)
	for _, grant := range grants {
		if grant.LogicalID == logicalID && grant.Field == field && grant.RevokedAt.IsZero() {
			return grant, true, nil
		}
	}
	return Grant{}, false, nil
}

// RefreshEphemeral advances an existing one-shot grant in place. The grant
// address is unique per node, so replacing it with a second row is invalid;
// generation is the value-version boundary the node agent already uses to
// replace a consumed in-memory secret.
func (s *Service) RefreshEphemeral(ctx context.Context, grant Grant) (Grant, error) {
	if grant.Retention != RetentionEphemeral {
		return Grant{}, fmt.Errorf("credential grant %q is not ephemeral", grant.ID)
	}
	generations, ok := s.repo.(GenerationRepository)
	if !ok {
		return Grant{}, fmt.Errorf("credential generation repository is unavailable")
	}
	next := grant.Generation + 1
	if next <= 0 {
		next = 1
	}
	if err := generations.SetGrantGeneration(ctx, grant.ID, grant.NodeID, next); err != nil {
		return Grant{}, err
	}
	grant.Generation = next
	grant.AckedGeneration = 0
	grant.ReceiptAt = time.Time{}
	grant.ReceiptAccepted = false
	grant.ReceiptReason = ""
	return grant, nil
}

// ReactivateEphemeral reuses the unique durable row after a prior one-shot
// grant was purged. The row identity is retained for audit continuity while
// its generation and active status are reset for the new answered value.
func (s *Service) ReactivateEphemeral(ctx context.Context, nodeID, logicalID, field string, class Class) (Grant, bool, error) {
	revokedRepo, ok := s.repo.(RevokedRepository)
	if !ok {
		return Grant{}, false, nil
	}
	grants, err := revokedRepo.ListRevoked(ctx, strings.TrimSpace(nodeID))
	if err != nil {
		return Grant{}, false, err
	}
	for _, grant := range grants {
		if grant.LogicalID != strings.TrimSpace(logicalID) || grant.Field != strings.TrimSpace(field) || grant.Retention != RetentionEphemeral {
			continue
		}
		if class != "" && grant.Class != class {
			continue
		}
		generations, ok := s.repo.(GenerationRepository)
		if !ok {
			return Grant{}, false, fmt.Errorf("credential generation repository is unavailable")
		}
		next := grant.Generation + 1
		if err := generations.ReactivateGrant(ctx, grant.ID, grant.NodeID, next); err != nil {
			return Grant{}, false, err
		}
		grant.Generation = next
		grant.AckedGeneration = 0
		grant.RevokedAt = time.Time{}
		grant.PurgeState = ""
		grant.PurgeAccepted = false
		grant.PurgeReason = ""
		grant.PurgeReceiptAt = time.Time{}
		return grant, true, nil
	}
	return Grant{}, false, nil
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

func (s *Service) RevokeWithReceipt(ctx context.Context, id string, binding MutationBinding) (Grant, MutationReceipt, error) {
	id = strings.TrimSpace(id)
	if id == "" || binding.ActorID == "" || binding.RequestID == "" || binding.NodeID == "" || binding.EnrollmentGeneration <= 0 || binding.Subject != RevokeGrantSubject+":"+id || binding.ExpectedRevision <= 0 || !validPayloadDigest(binding.PayloadDigest) {
		return Grant{}, MutationReceipt{}, fmt.Errorf("credential revoke mutation binding is incomplete or invalid")
	}
	resolver, ok := s.nodes.(NodeGenerationResolver)
	if !ok {
		return Grant{}, MutationReceipt{}, fmt.Errorf("node enrollment-generation resolver is unavailable")
	}
	generation, revoked, err := resolver.NodeGeneration(ctx, binding.NodeID)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if revoked || generation != binding.EnrollmentGeneration {
		return Grant{}, MutationReceipt{}, ErrStaleNodeGeneration{NodeID: binding.NodeID}
	}
	repository, ok := s.repo.(MutationRepository)
	if !ok {
		return Grant{}, MutationReceipt{}, fmt.Errorf("durable credential mutation receipts are unavailable")
	}
	return repository.RevokeWithReceipt(ctx, id, binding)
}

func (s *Service) Ack(ctx context.Context, id string, generation int64) error {
	if generation <= 0 {
		return fmt.Errorf("generation must be positive")
	}
	return s.repo.Ack(ctx, id, generation)
}

func (s *Service) Revoked(ctx context.Context, nodeID string) ([]Grant, error) {
	repository, ok := s.repo.(RevokedRepository)
	if !ok {
		// Small policy fakes and older repositories have no durable revoked
		// history. They can still serve active grants; reconnect cleanup is
		// simply unavailable until the durable repository is upgraded.
		return nil, nil
	}
	return repository.ListRevoked(ctx, strings.TrimSpace(nodeID))
}

func (s *Service) RecordPurgeReceipt(ctx context.Context, id, nodeID string, generation int64, accepted bool, reason string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(nodeID) == "" {
		return fmt.Errorf("purge receipt requires grant id and node id")
	}
	if generation <= 0 {
		return fmt.Errorf("purge receipt generation must be positive")
	}
	receipts, ok := s.repo.(PurgeReceiptRepository)
	if !ok {
		return fmt.Errorf("purge receipt repository is unavailable")
	}
	return receipts.RecordPurgeReceipt(ctx, id, strings.TrimSpace(nodeID), generation, accepted, reason, s.now().UTC())
}

// SetMutationDeliveryState records queueing/retry progress on the same durable
// owner mutation receipt. Target completion is recorded only by an
// authenticated grant or purge receipt.
func (s *Service) SetMutationDeliveryState(ctx context.Context, requestID, state string) error {
	repo, ok := s.repo.(MutationDeliveryRepository)
	if !ok {
		return fmt.Errorf("durable credential mutation delivery state is unavailable")
	}
	return repo.SetMutationDeliveryState(ctx, strings.TrimSpace(requestID), state)
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
