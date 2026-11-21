package credentials

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/domain"

	"github.com/google/uuid"
)

type MaterializeRequest struct {
	Target      Target
	Bindings    []domain.CredentialBinding
	OperationID string
	// GeneratedValues are fresh candidates per binding id; they are used only
	// when the target holds nothing for the binding yet.
	GeneratedValues map[string]string
	// OperatorValues are explicit instructions for this deploy and always
	// produce a new version.
	OperatorValues map[string]string
	// ExpiresAt carries owner-observed expiry for an initial materialization.
	// It is metadata only and is never inferred from the secret value.
	ExpiresAt map[string]time.Time
}

// MaterializeResult is metadata only.
type MaterializeResult struct {
	Preserved    []string                   `json:"preserved"`
	Materialized []string                   `json:"materialized"`
	Skipped      []string                   `json:"skipped"`
	Receipts     []domain.CredentialReceipt `json:"receipts"`
}

// Materialize provisions each binding: an existing materialised binding
// whose target still holds the value is preserved untouched (P13-A01);
// otherwise the operator value or the generated candidate becomes the next
// version. A locked or unavailable target store fails closed before any write.
func (s *Service) Materialize(ctx context.Context, req MaterializeRequest) (*MaterializeResult, error) {
	if strings.TrimSpace(req.OperationID) == "" {
		req.OperationID = "materialize-" + s.newID()
	}
	result := &MaterializeResult{Preserved: []string{}, Materialized: []string{}, Skipped: []string{}}
	for _, planned := range req.Bindings {
		if err := s.materializeOne(ctx, req, planned, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *Service) materializeOne(ctx context.Context, req MaterializeRequest, planned domain.CredentialBinding, result *MaterializeResult) error {
	binding, err := s.materializationBinding(ctx, planned)
	if err != nil {
		return err
	}
	probe, err := s.Distributor.Probe(ctx, req.Target, binding)
	if err != nil {
		return err
	}
	now := s.now()
	if canPreserveBinding(req, binding, probe, now) {
		return s.preserveBinding(ctx, binding, result, now)
	}
	value, err := s.materializationValue(binding, req)
	if err != nil {
		return err
	}
	if value == "" {
		binding.UpdatedAt = s.now()
		if err := s.Store.UpsertBinding(ctx, &binding); err != nil {
			return err
		}
		result.Skipped = append(result.Skipped, binding.ID)
		return nil
	}
	return s.writeMaterializedBinding(ctx, req, binding, value, now, result)
}

func (s *Service) materializationBinding(ctx context.Context, planned domain.CredentialBinding) (domain.CredentialBinding, error) {
	existing, err := s.Store.GetBinding(ctx, planned.DeploymentID, planned.ID)
	if err != nil || existing == nil {
		return planned, err
	}
	binding := *existing
	binding.ConsumerRefs = planned.ConsumerRefs
	binding.Target = planned.Target
	binding.Class = planned.Class
	binding.SourceClass = planned.SourceClass
	return binding, nil
}

func canPreserveBinding(req MaterializeRequest, binding domain.CredentialBinding, probe ProbeResult, now time.Time) bool {
	return strings.TrimSpace(req.OperatorValues[binding.ID]) == "" && binding.Version.Number > 0 && probe.Configured && binding.State != domain.CredentialBindingRevoked && !binding.Version.Expired(now)
}

func (s *Service) preserveBinding(ctx context.Context, binding domain.CredentialBinding, result *MaterializeResult, now time.Time) error {
	binding.UpdatedAt = now
	if err := s.Store.UpsertBinding(ctx, &binding); err != nil {
		return err
	}
	result.Preserved = append(result.Preserved, binding.ID)
	result.Receipts = append(result.Receipts, domain.CredentialReceipt{Step: "materialize", State: string(binding.State), Outcome: "preserved", At: s.now(), Details: map[string]any{"binding_id": binding.ID, "version": binding.Version.Number}})
	return nil
}

func (s *Service) materializationValue(binding domain.CredentialBinding, req MaterializeRequest) (string, error) {
	value := strings.TrimSpace(req.OperatorValues[binding.ID])
	if value == "" {
		value = req.GeneratedValues[binding.ID]
	}
	if value != "" || binding.SourceClass != domain.SecretClassPerInstallGenerated {
		return value, nil
	}
	return s.generate(binding)
}

func (s *Service) writeMaterializedBinding(ctx context.Context, req MaterializeRequest, binding domain.CredentialBinding, value string, now time.Time, result *MaterializeResult) error {
	next, err := s.nextVersion(ctx, binding)
	if err != nil {
		return err
	}
	version := domain.CredentialVersion{Number: next, ContentRef: newContentRef(), CreatedAt: now}
	if expires, ok := req.ExpiresAt[binding.ID]; ok {
		expires = expires.UTC()
		version.ExpiresAt = &expires
	}
	step := "materialize-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(version.Number)
	receipt, err := s.Distributor.Deliver(ctx, DeliverRequest{Target: req.Target, Binding: binding, Version: version, Value: value, OperationID: req.OperationID, Step: step})
	if err != nil {
		return err
	}
	if expires, ok := expiryFromDetails(receipt.Details); ok {
		version.ExpiresAt = &expires
	}
	if binding.Version.Number > 0 {
		prev := binding.Version
		binding.PreviousVersion = &prev
	}
	binding.Version = version
	binding.State = domain.CredentialBindingMaterialized
	binding.GrantRef = grantRefFrom(receipt, binding.GrantRef)
	binding.UpdatedAt = s.now()
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = binding.UpdatedAt
	}
	if err := s.Store.UpsertBinding(ctx, &binding); err != nil {
		return err
	}
	s.logf("credential materialized binding=%s version=%d transport=%s", binding.ID, version.Number, receipt.Transport)
	result.Materialized = append(result.Materialized, binding.ID)
	result.Receipts = append(result.Receipts, domain.CredentialReceipt{Step: step, State: string(binding.State), Outcome: "materialized", At: s.now(), TargetReceiptRef: receipt.Ref, Details: map[string]any{"binding_id": binding.ID, "version": version.Number, "transport": receipt.Transport}})
	return nil
}

// nextVersion is strictly greater than every version the binding ever
// minted, including versions of failed or recovering operations, so a number
// is never reused for different content.
func (s *Service) nextVersion(ctx context.Context, binding domain.CredentialBinding) (int64, error) {
	high := binding.Version.Number
	rotations, err := s.Store.ListRotations(ctx, binding.DeploymentID)
	if err != nil {
		return 0, err
	}
	for _, r := range rotations {
		if r.BindingID == binding.ID && r.ToVersion > high {
			high = r.ToVersion
		}
	}
	return high + 1, nil
}

func grantRefFrom(receipt Receipt, fallback string) string {
	if id, ok := receipt.Details["grant_id"].(string); ok && id != "" {
		return id
	}
	return fallback
}

func sanitizeStep(field string) string {
	var b strings.Builder
	for _, r := range field {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

func newContentRef() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return uuid.NewString()
	}
	return "cref_" + base64.RawURLEncoding.EncodeToString(buf)
}
