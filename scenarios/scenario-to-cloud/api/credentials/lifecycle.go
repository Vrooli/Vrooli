package credentials

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"

	"github.com/google/uuid"
)

// Restarter restarts consumers on the target after a non-dual-accept
// rotation so they re-read the new version. Nil means consumers re-read on
// their own schedule and the operation records that no restart was issued.
type Restarter interface {
	Restart(ctx context.Context, target Target, consumers []string) error
}

// RecoveryClient re-provisions a replacement host from an encrypted recovery
// bundle. Verify proves the bundle opens and names what it covers; Restore
// writes it into the replacement host's authority. The passphrase is passed
// in memory and travels on standard input only.
type RecoveryClient interface {
	Verify(ctx context.Context, bundleRef, passphrase string) ([]domain.CredentialDescriptor, error)
	Restore(ctx context.Context, bundleRef, passphrase string) error
}

// Service owns the credential lifecycle for cloud deployments. It composes
// the ledger, one transport distributor and the per-class providers; it is
// the only writer of bindings and operations.
type Service struct {
	Store       Store
	Distributor Distributor
	Providers   Providers
	Restarter   Restarter
	Recovery    RecoveryClient
	// Generate mints a value for a generated class. Defaults to 32 random
	// bytes, base64url.
	Generate func(binding domain.CredentialBinding) (string, error)
	Now      func() time.Time
	NewID    func() string
	Logger   *log.Logger
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) newID() string {
	if s.NewID != nil {
		return s.NewID()
	}
	return uuid.NewString()
}

func (s *Service) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}

func (s *Service) generate(binding domain.CredentialBinding) (string, error) {
	if s.Generate != nil {
		return s.Generate(binding)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *Service) provider(class domain.CredentialClass) (Provider, error) {
	if p, ok := s.Providers[class]; ok && p != nil {
		return p, nil
	}
	return nil, newError(CodeRotationRefused, "no provider is declared for credential class "+string(class)).
		WithDetail("class", string(class)).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "doc", Reference: "docs/reference/credential-lifecycle.md", Label: "Declare the provider hook for this class"})
}

func newContentRef() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return uuid.NewString()
	}
	return "cref_" + base64.RawURLEncoding.EncodeToString(buf)
}

// MaterializeRequest provisions the first version of planned bindings and
// preserves existing generated values on ordinary redeploy.
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
		existing, err := s.Store.GetBinding(ctx, planned.DeploymentID, planned.ID)
		if err != nil {
			return nil, err
		}
		binding := planned
		if existing != nil {
			binding = *existing
			binding.ConsumerRefs = planned.ConsumerRefs
			binding.Target = planned.Target
			binding.Class = planned.Class
			binding.SourceClass = planned.SourceClass
		}
		probe, err := s.Distributor.Probe(ctx, req.Target, binding)
		if err != nil {
			return nil, err
		}
		operatorValue := strings.TrimSpace(req.OperatorValues[binding.ID])
		now := s.now()
		if operatorValue == "" && binding.Version.Number > 0 && probe.Configured && binding.State != domain.CredentialBindingRevoked && !binding.Version.Expired(now) {
			binding.UpdatedAt = now
			if err := s.Store.UpsertBinding(ctx, &binding); err != nil {
				return nil, err
			}
			result.Preserved = append(result.Preserved, binding.ID)
			result.Receipts = append(result.Receipts, domain.CredentialReceipt{Step: "materialize", State: string(binding.State), Outcome: "preserved", At: s.now(), Details: map[string]any{"binding_id": binding.ID, "version": binding.Version.Number}})
			continue
		}
		value := operatorValue
		if value == "" {
			value = req.GeneratedValues[binding.ID]
		}
		if value == "" && binding.SourceClass == domain.SecretClassPerInstallGenerated {
			value, err = s.generate(binding)
			if err != nil {
				return nil, err
			}
		}
		if value == "" {
			binding.UpdatedAt = s.now()
			if err := s.Store.UpsertBinding(ctx, &binding); err != nil {
				return nil, err
			}
			result.Skipped = append(result.Skipped, binding.ID)
			continue
		}
		next, err := s.nextVersion(ctx, binding)
		if err != nil {
			return nil, err
		}
		version := domain.CredentialVersion{Number: next, ContentRef: newContentRef(), CreatedAt: now}
		if expires, ok := req.ExpiresAt[binding.ID]; ok {
			expires = expires.UTC()
			version.ExpiresAt = &expires
		}
		step := "materialize-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(version.Number)
		receipt, err := s.Distributor.Deliver(ctx, DeliverRequest{Target: req.Target, Binding: binding, Version: version, Value: value, OperationID: req.OperationID, Step: step})
		if err != nil {
			return nil, err
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
			return nil, err
		}
		s.logf("credential materialized binding=%s version=%d transport=%s", binding.ID, version.Number, receipt.Transport)
		result.Materialized = append(result.Materialized, binding.ID)
		result.Receipts = append(result.Receipts, domain.CredentialReceipt{Step: step, State: string(binding.State), Outcome: "materialized", At: s.now(), TargetReceiptRef: receipt.Ref, Details: map[string]any{"binding_id": binding.ID, "version": version.Number, "transport": receipt.Transport}})
	}
	return result, nil
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

// RotateRequest starts a versioned rotation of one binding.
type RotateRequest struct {
	Target       Target
	DeploymentID string
	BindingID    string
	// Value is the operator-supplied replacement for external classes; it is
	// kept in memory and never persisted. Empty means mint (generated classes).
	Value      string
	RequestKey string
}

// ResumeInput continues an operation that stopped on an operator step,
// an overlap window or missing acknowledgements.
type ResumeInput struct {
	OperatorConfirmed bool
}

// Rotate runs the state machine as far as it can in one call and returns the
// durable operation. A non-terminal return is truthful: the state names what
// is still outstanding.
func (s *Service) Rotate(ctx context.Context, req RotateRequest) (*domain.CredentialRotation, error) {
	binding, err := s.Store.GetBinding(ctx, req.DeploymentID, req.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "no such credential binding on this deployment").WithDetail("binding_id", req.BindingID)
	}
	if binding.State == domain.CredentialBindingRevoked {
		return nil, newError(CodeRotationRefused, "the binding is revoked; materialise a new version through deploy instead").WithDetail("binding_id", binding.ID)
	}
	if binding.Class == domain.CredentialClassMachineEnrollment {
		return nil, newError(CodeRotationRefused, "machine enrollment credentials are owned by vrooli-bridge; rotate or revoke the node identity there").
			WithDetail("binding_id", binding.ID).
			WithNextAction(apierrors.NextAction{Owner: "vrooli-bridge", Kind: "command", Reference: "vrooli-bridge nodes", Label: "Rotate the node enrollment through Bridge"})
	}
	if _, err := s.provider(binding.Class); err != nil {
		return nil, err
	}
	if active, err := s.activeOperation(ctx, binding); err != nil {
		return nil, err
	} else if active != nil {
		if req.RequestKey != "" && active.ID == req.RequestKey {
			return active, nil
		}
		return nil, newError(CodeRotationConflict, "another lifecycle operation is still open on this binding").WithDetail("rotation_id", active.ID).WithDetail("state", string(active.State))
	}
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	now := s.now()
	rotation := &domain.CredentialRotation{ID: id, DeploymentID: binding.DeploymentID, BindingID: binding.ID, Kind: domain.CredentialOperationRotate, FromVersion: binding.Version.Number, State: domain.RotationPlanned, Consumers: initialConsumers(binding, now), Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now}
	if err := s.Store.SaveRotation(ctx, rotation); err != nil {
		return nil, err
	}
	return s.advance(ctx, rotation, binding, req.Target, req.Value, ResumeInput{})
}

// Resume continues a non-terminal operation.
func (s *Service) Resume(ctx context.Context, target Target, rotationID string, input ResumeInput) (*domain.CredentialRotation, error) {
	rotation, err := s.Store.GetRotation(ctx, rotationID)
	if err != nil {
		return nil, err
	}
	if rotation == nil {
		return nil, newError(CodeRotationNotFound, "no such credential operation").WithDetail("rotation_id", rotationID)
	}
	if rotation.State.Terminal() {
		return rotation, nil
	}
	binding, err := s.Store.GetBinding(ctx, rotation.DeploymentID, rotation.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "the operation's binding no longer exists").WithDetail("binding_id", rotation.BindingID)
	}
	switch rotation.Kind {
	case domain.CredentialOperationRevoke:
		return s.revokeAdvance(ctx, rotation, binding, target)
	case domain.CredentialOperationRecover:
		return rotation, nil
	}
	return s.advance(ctx, rotation, binding, target, "", input)
}

func (s *Service) activeOperation(ctx context.Context, binding *domain.CredentialBinding) (*domain.CredentialRotation, error) {
	rotations, err := s.Store.ListRotations(ctx, binding.DeploymentID)
	if err != nil {
		return nil, err
	}
	for i := range rotations {
		r := rotations[i]
		if r.BindingID == binding.ID && !r.State.Terminal() && r.State != domain.RotationRevocationIncomplete {
			return &r, nil
		}
	}
	return nil, nil
}

func initialConsumers(binding *domain.CredentialBinding, now time.Time) []domain.CredentialConsumerProgress {
	out := make([]domain.CredentialConsumerProgress, 0, len(binding.ConsumerRefs))
	for _, consumer := range binding.ConsumerRefs {
		out = append(out, domain.CredentialConsumerProgress{Consumer: consumer, State: domain.ConsumerPending, UpdatedAt: now})
	}
	return out
}

func (s *Service) record(ctx context.Context, rotation *domain.CredentialRotation, step string, state domain.CredentialRotationState, outcome string, ref string, details map[string]any, limitations ...string) error {
	now := s.now()
	rotation.State = state
	rotation.UpdatedAt = now
	rotation.Receipts = append(rotation.Receipts, domain.CredentialReceipt{Step: step, State: string(state), Outcome: outcome, At: now, TargetReceiptRef: ref, Details: details, Limitations: limitations})
	if state.Terminal() {
		rotation.CompletedAt = &now
	}
	s.logf("credential operation=%s binding=%s step=%s state=%s outcome=%s", rotation.ID, rotation.BindingID, step, state, outcome)
	return s.Store.SaveRotation(ctx, rotation)
}

func (s *Service) fail(ctx context.Context, rotation *domain.CredentialRotation, code, message string, details map[string]any) (*domain.CredentialRotation, error) {
	rotation.Error = &domain.CredentialOperationError{Code: code, Message: message}
	if err := s.record(ctx, rotation, "fail", domain.RotationFailed, "failed", "", details); err != nil {
		return rotation, err
	}
	return rotation, newError(code, message).WithDetail("rotation_id", rotation.ID)
}

// advance drives the rotation state machine. Each transition is persisted
// before the next side effect so a crash leaves an accurate ledger.
func (s *Service) advance(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, value string, input ResumeInput) (*domain.CredentialRotation, error) {
	provider, err := s.provider(binding.Class)
	if err != nil {
		return rotation, err
	}
	var version domain.CredentialVersion
	if rotation.ToVersion > 0 {
		version = domain.CredentialVersion{Number: rotation.ToVersion}
		if binding.Version.Number == rotation.ToVersion {
			version = binding.Version
		} else if detail := versionFromReceipts(rotation); detail != nil {
			version = *detail
		}
	}
	for {
		switch rotation.State {
		case domain.RotationPlanned:
			if value == "" {
				if binding.SourceClass != domain.SecretClassPerInstallGenerated && binding.Class == domain.CredentialClassExternalAPICredential {
					return s.fail(ctx, rotation, CodeRotationRefused, "an external credential rotation needs the replacement value supplied in memory; none was given", nil)
				}
				value, err = s.generate(*binding)
				if err != nil {
					return s.fail(ctx, rotation, CodeRotationRefused, "mint replacement value: "+err.Error(), nil)
				}
			}
			next, err := s.nextVersion(ctx, *binding)
			if err != nil {
				return rotation, err
			}
			version = domain.CredentialVersion{Number: next, ContentRef: newContentRef(), CreatedAt: s.now()}
			rotation.ToVersion = version.Number
			if err := s.record(ctx, rotation, "new-version", domain.RotationNewVersionCreated, "created", "", map[string]any{"version": version.Number, "content_ref": version.ContentRef, "created_at": version.CreatedAt}); err != nil {
				return rotation, err
			}
		case domain.RotationNewVersionCreated:
			if value == "" {
				return s.fail(ctx, rotation, CodeRotationRefused, "the replacement value is no longer in memory; start a new rotation", map[string]any{"version": rotation.ToVersion})
			}
			prepared, err := provider.Prepare(ctx, PrepareRequest{Target: target, Binding: *binding, Version: version, Value: value, OperationID: rotation.ID})
			if err != nil {
				var refusal *RotationRefusal
				if errors.As(err, &refusal) {
					return s.fail(ctx, rotation, CodeRotationRefused, refusal.Reason, map[string]any{"owner": refusal.Owner})
				}
				// Provider rejected: nothing was distributed and the
				// predecessor is untouched (SECRET-03).
				return s.fail(ctx, rotation, CodeDistributionFailed, "provider rejected the new version: "+err.Error(), map[string]any{"predecessor_retained": true, "version": rotation.FromVersion})
			}
			details := prepared.Details
			if details == nil {
				details = map[string]any{}
			}
			details["dual_accept"] = prepared.DualAccept
			if prepared.ExpiresAt != nil {
				expires := prepared.ExpiresAt.UTC()
				version.ExpiresAt = &expires
				details["expires_at"] = expires
			}
			if err := s.record(ctx, rotation, "provider-prepare", domain.RotationProviderPrepared, "prepared", "", details); err != nil {
				return rotation, err
			}
		case domain.RotationProviderPrepared:
			if value == "" {
				return s.recover(ctx, rotation, binding, target, version, "the replacement value is no longer in memory after provider preparation")
			}
			step := "rotate-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(version.Number)
			receipt, err := s.Distributor.Deliver(ctx, DeliverRequest{Target: target, Binding: *binding, Version: version, Value: value, OperationID: rotation.ID, Step: step})
			if err != nil {
				if apierrors.Is(err, apierrors.CodeForbiddenRevoked) {
					rotation.Error = &domain.CredentialOperationError{Code: apierrors.CodeForbiddenRevoked, Message: "distribution denied: the grant is revoked"}
					_ = s.record(ctx, rotation, "consumer-update", domain.RotationFailed, "denied", "", map[string]any{"predecessor_retained": true})
					return rotation, err
				}
				return s.recover(ctx, rotation, binding, target, version, "distribution failed: "+err.Error())
			}
			binding.GrantRef = grantRefFrom(receipt, binding.GrantRef)
			now := s.now()
			for i := range rotation.Consumers {
				rotation.Consumers[i].State = domain.ConsumerUpdated
				rotation.Consumers[i].Version = version.Number
				rotation.Consumers[i].UpdatedAt = now
			}
			restart := "not_requested"
			if s.Restarter != nil && len(binding.ConsumerRefs) > 0 {
				if err := s.Restarter.Restart(ctx, target, binding.ConsumerRefs); err != nil {
					return s.recover(ctx, rotation, binding, target, version, "consumer restart failed: "+err.Error())
				}
				restart = "restarted"
			}
			if err := s.record(ctx, rotation, "consumer-update", domain.RotationConsumersUpdated, "updated", receipt.Ref, map[string]any{"version": version.Number, "transport": receipt.Transport, "consumers": binding.ConsumerRefs, "restart": restart}); err != nil {
				return rotation, err
			}
		case domain.RotationConsumersUpdated:
			if err := s.collectAcks(ctx, rotation, binding, target, version.Number); err != nil {
				return rotation, err
			}
			acks, err := s.Store.ListAcks(ctx, binding.ID)
			if err != nil {
				return rotation, err
			}
			verified, err := provider.Verify(ctx, VerifyRequest{Target: target, Binding: *binding, Version: version, Consumers: binding.ConsumerRefs, Acks: acks})
			if err != nil {
				return s.recover(ctx, rotation, binding, target, version, "verification failed: "+err.Error())
			}
			now := s.now()
			for i := range rotation.Consumers {
				c := &rotation.Consumers[i]
				if reason, missing := verified.Unverified[c.Consumer]; missing {
					c.State = domain.ConsumerUnreachable
					c.Reason = reason
				} else {
					c.State = domain.ConsumerAcknowledged
					c.Reason = ""
				}
				c.UpdatedAt = now
			}
			if len(verified.Unverified) > 0 {
				rotation.Unreached = sortedKeys(verified.Unverified)
				details := map[string]any{"version": version.Number, "unreached": rotation.Unreached, "verified": verified.Verified}
				if err := s.record(ctx, rotation, "verify", domain.RotationConsumersUpdated, "incomplete", "", details, "rotation is incomplete until every listed consumer acknowledges version "+fmt.Sprint(version.Number)); err != nil {
					return rotation, err
				}
				return rotation, nil
			}
			rotation.Unreached = nil
			prev := binding.Version
			binding.PreviousVersion = &prev
			binding.Version = version
			binding.State = domain.CredentialBindingMaterialized
			binding.UpdatedAt = now
			if err := s.Store.UpsertBinding(ctx, binding); err != nil {
				return rotation, err
			}
			details := verified.Details
			if details == nil {
				details = map[string]any{}
			}
			details["version"] = version.Number
			details["verified"] = verified.Verified
			if err := s.record(ctx, rotation, "verify", domain.RotationVerified, "verified", "", details); err != nil {
				return rotation, err
			}
		case domain.RotationVerified, domain.RotationPendingOperatorInput:
			if rotation.State == domain.RotationPendingOperatorInput && !input.OperatorConfirmed {
				return rotation, newError(CodePendingOperatorInput, "the rotation is waiting for the operator to revoke the predecessor at the provider").
					WithDetail("rotation_id", rotation.ID).WithDetail("handoff", rotation.PendingOperatorInput)
			}
			if rotation.ResumeAfter != nil && s.now().Before(*rotation.ResumeAfter) && !input.OperatorConfirmed {
				return rotation, nil
			}
			predecessor := domain.CredentialVersion{Number: rotation.FromVersion}
			if binding.PreviousVersion != nil {
				predecessor = *binding.PreviousVersion
			}
			var result RevokePredecessorResult
			if rotation.State == domain.RotationPendingOperatorInput {
				result = RevokePredecessorResult{Done: true, Details: map[string]any{"operator_confirmed": true, "handoff": rotation.PendingOperatorInput.Reference}}
			} else {
				result, err = provider.RevokePredecessor(ctx, RevokePredecessorRequest{Target: target, Binding: *binding, Predecessor: predecessor, Successor: binding.Version, OperationID: rotation.ID})
				if err != nil {
					return s.fail(ctx, rotation, CodeDistributionFailed, "predecessor revocation failed at the provider: "+err.Error(), map[string]any{"new_version_active": binding.Version.Number})
				}
			}
			if result.Handoff != nil {
				rotation.PendingOperatorInput = result.Handoff
				rotation.PendingOperatorInput.RequestedAt = s.now()
				if err := s.record(ctx, rotation, "revoke-predecessor", domain.RotationPendingOperatorInput, "pending_operator_input", "", map[string]any{"handoff": result.Handoff.Reference, "provider": result.Handoff.Provider}); err != nil {
					return rotation, err
				}
				return rotation, nil
			}
			if result.ResumeAfter != nil && !result.Done {
				rotation.ResumeAfter = result.ResumeAfter
				if err := s.record(ctx, rotation, "revoke-predecessor", domain.RotationVerified, "deferred", "", map[string]any{"resume_after": result.ResumeAfter, "details": result.Details}); err != nil {
					return rotation, err
				}
				return rotation, nil
			}
			rotation.PendingOperatorInput = nil
			rotation.ResumeAfter = nil
			targetReceipt := Receipt{}
			if predecessor.Number > 0 {
				step := "revoke-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(predecessor.Number)
				targetReceipt, err = s.Distributor.Revoke(ctx, RevokeRequest{Target: target, Binding: *binding, Version: predecessor.Number, OperationID: rotation.ID, Step: step})
				if err != nil {
					if IsUnreachable(err) {
						rotation.Unreached = []string{target.NodeLabel()}
						if err := s.record(ctx, rotation, "revoke-predecessor", domain.RotationVerified, "incomplete", "", map[string]any{"predecessor": predecessor.Number, "unreached": rotation.Unreached}, RevocationLimitation); err != nil {
							return rotation, err
						}
						return rotation, newError(CodeRevocationIncomplete, "the new version is active but the predecessor could not be purged from an unreachable target").WithDetail("rotation_id", rotation.ID).WithDetail("unreached", rotation.Unreached)
					}
					return s.fail(ctx, rotation, CodeDistributionFailed, "predecessor purge failed on the target: "+err.Error(), map[string]any{"new_version_active": binding.Version.Number})
				}
			}
			binding.PreviousVersion = nil
			binding.UpdatedAt = s.now()
			if err := s.Store.UpsertBinding(ctx, binding); err != nil {
				return rotation, err
			}
			details := result.Details
			if details == nil {
				details = map[string]any{}
			}
			details["predecessor"] = predecessor.Number
			if err := s.record(ctx, rotation, "revoke-predecessor", domain.RotationOldVersionRevoked, "revoked", targetReceipt.Ref, details, RevocationLimitation); err != nil {
				return rotation, err
			}
		case domain.RotationOldVersionRevoked:
			if err := s.record(ctx, rotation, "complete", domain.RotationComplete, "complete", "", map[string]any{"active_version": binding.Version.Number}); err != nil {
				return rotation, err
			}
			return rotation, nil
		case domain.RotationRecovering:
			return s.recover(ctx, rotation, binding, target, version, "resumed while recovering")
		default:
			return rotation, nil
		}
	}
}

// recover retires whatever the failed attempt distributed, keeps the
// predecessor as the active version and terminates in Failed.
func (s *Service) recover(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, version domain.CredentialVersion, reason string) (*domain.CredentialRotation, error) {
	if err := s.record(ctx, rotation, "recover", domain.RotationRecovering, "recovering", "", map[string]any{"reason": reason, "predecessor_retained": true, "active_version": binding.Version.Number}); err != nil {
		return rotation, err
	}
	details := map[string]any{"predecessor_retained": true, "active_version": binding.Version.Number}
	if version.Number > 0 {
		step := "recover-purge-v" + fmt.Sprint(version.Number)
		if receipt, err := s.Distributor.Revoke(ctx, RevokeRequest{Target: target, Binding: *binding, Version: version.Number, OperationID: rotation.ID, Step: step}); err != nil {
			details["purge_new_version"] = "unconfirmed: " + err.Error()
			if IsUnreachable(err) {
				rotation.Unreached = []string{target.NodeLabel()}
			}
		} else {
			details["purge_new_version"] = "purged"
			details["purge_receipt"] = receipt.Ref
		}
	}
	rotation.Error = &domain.CredentialOperationError{Code: CodeDistributionFailed, Message: reason}
	if err := s.record(ctx, rotation, "fail", domain.RotationFailed, "failed", "", details); err != nil {
		return rotation, err
	}
	return rotation, newError(CodeDistributionFailed, reason).WithDetail("rotation_id", rotation.ID).WithDetail("predecessor_retained", true)
}

// collectAcks asks the target to prove each consumer can use the version and
// records the acknowledgements it obtains. A consumer the target cannot
// confirm (unreachable transport, failed verb) is left without an ack, so
// verification names it and the rotation stays incomplete (P13-A05).
func (s *Service) collectAcks(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, version int64) error {
	acks, err := s.Store.ListAcks(ctx, binding.ID)
	if err != nil {
		return err
	}
	acked := map[string]bool{}
	for _, ack := range acks {
		if ack.Version == version {
			acked[ack.Consumer] = true
		}
	}
	for _, consumer := range binding.ConsumerRefs {
		if acked[consumer] {
			continue
		}
		step := "ack-" + sanitizeStep(consumer) + "-v" + fmt.Sprint(version)
		receipt, err := s.Distributor.Acknowledge(ctx, AckRequest{Target: target, Binding: *binding, Version: version, Consumer: consumer, OperationID: rotation.ID, Step: step})
		if err != nil {
			s.logf("credential ack unconfirmed operation=%s consumer=%s version=%d unreachable=%t", rotation.ID, consumer, version, IsUnreachable(err))
			continue
		}
		if err := s.Store.RecordAck(ctx, domain.CredentialAck{BindingID: binding.ID, Consumer: consumer, Version: version, VerifiedAt: s.now()}); err != nil {
			return err
		}
		s.logf("credential ack binding=%s consumer=%s version=%d ref=%s", binding.ID, consumer, version, receipt.Ref)
	}
	return nil
}

func versionFromReceipts(rotation *domain.CredentialRotation) *domain.CredentialVersion {
	var version *domain.CredentialVersion
	for _, receipt := range rotation.Receipts {
		if receipt.Step != "new-version" && receipt.Step != "provider-prepare" {
			continue
		}
		if version == nil {
			version = &domain.CredentialVersion{Number: rotation.ToVersion}
		}
		if ref, ok := receipt.Details["content_ref"].(string); ok {
			version.ContentRef = ref
		}
		switch created := receipt.Details["created_at"].(type) {
		case time.Time:
			version.CreatedAt = created
		case string:
			if parsed, err := time.Parse(time.RFC3339Nano, created); err == nil {
				version.CreatedAt = parsed
			}
		}
		switch expires := receipt.Details["expires_at"].(type) {
		case time.Time:
			value := expires.UTC()
			version.ExpiresAt = &value
		case string:
			if parsed, err := time.Parse(time.RFC3339Nano, expires); err == nil {
				parsed = parsed.UTC()
				version.ExpiresAt = &parsed
			}
		}
	}
	return version
}

func expiryFromDetails(details map[string]any) (time.Time, bool) {
	value, ok := details["expires_at"]
	if !ok {
		return time.Time{}, false
	}
	switch expires := value.(type) {
	case time.Time:
		return expires.UTC(), true
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, expires)
		if err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Acknowledge records that a consumer verified use of a version. Only a
// consumer the closure authorises may acknowledge.
func (s *Service) Acknowledge(ctx context.Context, deploymentID, bindingID, consumer string, version int64) (*domain.CredentialAck, error) {
	binding, err := s.Store.GetBinding(ctx, deploymentID, bindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "no such credential binding").WithDetail("binding_id", bindingID)
	}
	authorized := false
	for _, ref := range binding.ConsumerRefs {
		if ref == consumer {
			authorized = true
		}
	}
	if !authorized {
		return nil, apierrors.New(apierrors.CodeForbiddenScope, "the consumer is not an authorized reader of this credential").WithDetail("consumer", consumer).WithDetail("binding_id", bindingID)
	}
	ack := domain.CredentialAck{BindingID: bindingID, Consumer: consumer, Version: version, VerifiedAt: s.now()}
	if err := s.Store.RecordAck(ctx, ack); err != nil {
		return nil, err
	}
	s.logf("credential ack binding=%s consumer=%s version=%d", bindingID, consumer, version)
	return &ack, nil
}

// RevokeBindingRequest revokes the current version of a binding.
type RevokeBindingRequest struct {
	Target       Target
	DeploymentID string
	BindingID    string
	RequestKey   string
}

// Revoke purges the binding's current version from the target and denies
// further distribution. An unreachable target leaves the operation in
// revocation_incomplete with the node retained until confirmed (P13-A05).
func (s *Service) Revoke(ctx context.Context, req RevokeBindingRequest) (*domain.CredentialRotation, error) {
	binding, err := s.Store.GetBinding(ctx, req.DeploymentID, req.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "no such credential binding").WithDetail("binding_id", req.BindingID)
	}
	if binding.Class == domain.CredentialClassMachineEnrollment {
		return nil, newError(CodeRotationRefused, "machine enrollment credentials are revoked through vrooli-bridge").WithDetail("binding_id", binding.ID)
	}
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	if existing, err := s.Store.GetRotation(ctx, id); err != nil {
		return nil, err
	} else if existing != nil {
		return s.revokeAdvance(ctx, existing, binding, req.Target)
	}
	now := s.now()
	rotation := &domain.CredentialRotation{ID: id, DeploymentID: binding.DeploymentID, BindingID: binding.ID, Kind: domain.CredentialOperationRevoke, FromVersion: binding.Version.Number, ToVersion: 0, State: domain.RotationPlanned, Consumers: initialConsumers(binding, now), Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now}
	if err := s.Store.SaveRotation(ctx, rotation); err != nil {
		return nil, err
	}
	return s.revokeAdvance(ctx, rotation, binding, req.Target)
}

func (s *Service) revokeAdvance(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target) (*domain.CredentialRotation, error) {
	if rotation.State.Terminal() {
		return rotation, nil
	}
	step := "revoke-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(binding.Version.Number)
	receipt, err := s.Distributor.Revoke(ctx, RevokeRequest{Target: target, Binding: *binding, Version: binding.Version.Number, OperationID: rotation.ID, Step: step})
	if err != nil {
		if IsUnreachable(err) {
			rotation.Unreached = []string{target.NodeLabel()}
			if err := s.record(ctx, rotation, "revoke", domain.RotationRevocationIncomplete, "incomplete", "", map[string]any{"version": binding.Version.Number, "unreached": rotation.Unreached}, RevocationLimitation, "the unreached node is retained until its purge receipt is confirmed"); err != nil {
				return rotation, err
			}
			return rotation, newError(CodeRevocationIncomplete, "the target could not be reached; revocation is incomplete and the node is retained until confirmed").
				WithDetail("rotation_id", rotation.ID).WithDetail("unreached", rotation.Unreached).
				WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "endpoint", Reference: "/api/v1/deployments/{id}/credentials/rotations/" + rotation.ID + "/resume", Label: "Retry once the target is reachable"})
		}
		return s.fail(ctx, rotation, CodeDistributionFailed, "revocation failed on the target: "+err.Error(), map[string]any{"version": binding.Version.Number})
	}
	rotation.Unreached = nil
	now := s.now()
	for i := range rotation.Consumers {
		rotation.Consumers[i].State = domain.ConsumerUpdated
		rotation.Consumers[i].UpdatedAt = now
	}
	binding.State = domain.CredentialBindingRevoked
	binding.UpdatedAt = now
	if err := s.Store.UpsertBinding(ctx, binding); err != nil {
		return rotation, err
	}
	details := map[string]any{"version": binding.Version.Number, "transport": receipt.Transport}
	for k, v := range receipt.Details {
		details[k] = v
	}
	if err := s.record(ctx, rotation, "revoke", domain.RotationOldVersionRevoked, "revoked", receipt.Ref, details, receipt.Limitations...); err != nil {
		return rotation, err
	}
	if err := s.record(ctx, rotation, "complete", domain.RotationComplete, "complete", "", map[string]any{"binding_state": string(binding.State)}); err != nil {
		return rotation, err
	}
	return rotation, nil
}

// RecoverRequest re-provisions a deployment's bindings onto a replacement
// host from an encrypted recovery bundle already present on that host.
type RecoverRequest struct {
	DeploymentID string
	NewTarget    Target
	BundleRef    string
	// Passphrase stays in memory and travels only on standard input.
	Passphrase string
	RequestKey string
}

// Recover is the lost-host path. It verifies the bundle covers every
// materialised binding, restores it into the replacement host's authority
// and confirms each descriptor is configured there. A locked store on the
// replacement host fails closed with credential_store_locked.
func (s *Service) Recover(ctx context.Context, req RecoverRequest) (*domain.CredentialRotation, error) {
	if s.Recovery == nil {
		return nil, newError(CodeRecoveryFailed, "no recovery client is configured for the replacement host")
	}
	if strings.TrimSpace(req.Passphrase) == "" {
		return nil, apierrors.New(apierrors.CodeNeedsInput, "the recovery passphrase is required; supply it in the request body, never on a command line")
	}
	bindings, err := s.Store.ListBindings(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	now := s.now()
	rotation := &domain.CredentialRotation{ID: id, DeploymentID: req.DeploymentID, Kind: domain.CredentialOperationRecover, State: domain.RotationPlanned, Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now}
	if err := s.Store.SaveRotation(ctx, rotation); err != nil {
		return nil, err
	}
	covered, err := s.Recovery.Verify(ctx, req.BundleRef, req.Passphrase)
	if err != nil {
		if apierrors.Is(err, CodeStoreLocked) {
			rotation.Error = &domain.CredentialOperationError{Code: CodeStoreLocked, Message: "the replacement host's credential store is locked"}
			_ = s.record(ctx, rotation, "verify-bundle", domain.RotationFailed, "store_locked", "", map[string]any{"target": req.NewTarget.NodeLabel()})
			return rotation, StoreLocked(req.NewTarget.NodeLabel())
		}
		return s.fail(ctx, rotation, CodeRecoveryFailed, "the recovery bundle could not be verified: "+err.Error(), map[string]any{"bundle_ref": req.BundleRef})
	}
	coveredSet := map[string]bool{}
	for _, d := range covered {
		coveredSet[d.Address()] = true
	}
	var missing []string
	for _, b := range bindings {
		if b.State == domain.CredentialBindingMaterialized && !coveredSet[b.Descriptor.Address()] {
			missing = append(missing, b.Descriptor.Address())
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return s.fail(ctx, rotation, CodeRecoveryFailed, "the recovery bundle does not cover every materialised binding", map[string]any{"missing": missing, "bundle_ref": req.BundleRef})
	}
	if err := s.record(ctx, rotation, "verify-bundle", domain.RotationProviderPrepared, "verified", "", map[string]any{"bundle_ref": req.BundleRef, "entries": len(covered)}); err != nil {
		return rotation, err
	}
	if err := s.Recovery.Restore(ctx, req.BundleRef, req.Passphrase); err != nil {
		if apierrors.Is(err, CodeStoreLocked) {
			rotation.Error = &domain.CredentialOperationError{Code: CodeStoreLocked, Message: "the replacement host's credential store is locked"}
			_ = s.record(ctx, rotation, "restore", domain.RotationFailed, "store_locked", "", map[string]any{"target": req.NewTarget.NodeLabel()})
			return rotation, StoreLocked(req.NewTarget.NodeLabel())
		}
		return s.fail(ctx, rotation, CodeRecoveryFailed, "restore into the replacement host failed: "+err.Error(), map[string]any{"target": req.NewTarget.NodeLabel()})
	}
	if err := s.record(ctx, rotation, "restore", domain.RotationConsumersUpdated, "restored", "", map[string]any{"target": req.NewTarget.NodeLabel()}); err != nil {
		return rotation, err
	}
	var unconfirmed []string
	for i := range bindings {
		b := &bindings[i]
		if b.State != domain.CredentialBindingMaterialized {
			continue
		}
		probe, err := s.Distributor.Probe(ctx, req.NewTarget, *b)
		if err != nil {
			if apierrors.Is(err, CodeStoreLocked) {
				return rotation, err
			}
			unconfirmed = append(unconfirmed, b.Descriptor.Address())
			continue
		}
		if !probe.Configured {
			unconfirmed = append(unconfirmed, b.Descriptor.Address())
			continue
		}
		b.GrantRef = ""
		b.UpdatedAt = s.now()
		if err := s.Store.UpsertBinding(ctx, b); err != nil {
			return rotation, err
		}
		rotation.Consumers = append(rotation.Consumers, domain.CredentialConsumerProgress{Consumer: b.ID, State: domain.ConsumerAcknowledged, Version: b.Version.Number, UpdatedAt: s.now()})
	}
	if len(unconfirmed) > 0 {
		sort.Strings(unconfirmed)
		rotation.Unreached = unconfirmed
		if err := s.record(ctx, rotation, "confirm", domain.RotationConsumersUpdated, "incomplete", "", map[string]any{"unconfirmed": unconfirmed}); err != nil {
			return rotation, err
		}
		return rotation, newError(CodeRecoveryFailed, "restore completed but some bindings are not confirmed on the replacement host").WithDetail("unconfirmed", unconfirmed).WithDetail("rotation_id", rotation.ID)
	}
	if err := s.record(ctx, rotation, "confirm", domain.RotationVerified, "verified", "", map[string]any{"target": req.NewTarget.NodeLabel()}); err != nil {
		return rotation, err
	}
	if err := s.record(ctx, rotation, "complete", domain.RotationComplete, "complete", "", map[string]any{"target": req.NewTarget.NodeLabel(), "bindings": len(rotation.Consumers)}); err != nil {
		return rotation, err
	}
	return rotation, nil
}

// BreakGlassRequest is the emergency access path: scoped, time-bounded,
// audited, confirmed with an explicit string.
type BreakGlassRequest struct {
	Target       Target
	DeploymentID string
	BindingID    string
	Scope        string
	Window       time.Duration
	Confirmation string
	Operator     string
	RequestKey   string
}

// MaxBreakGlassWindow bounds emergency access.
const MaxBreakGlassWindow = 4 * time.Hour

// ExpectedBreakGlassConfirmation is the exact string an operator must send.
func ExpectedBreakGlassConfirmation(bindingID string) string {
	return "BREAK-GLASS " + bindingID
}

// BreakGlass issues an emergency version through the normal preparation and
// distribution stages, records the window, and leaves predecessor revocation
// to SweepBreakGlass once the window closes (SECRET-08).
func (s *Service) BreakGlass(ctx context.Context, req BreakGlassRequest) (*domain.CredentialRotation, error) {
	if req.Confirmation != ExpectedBreakGlassConfirmation(req.BindingID) {
		return nil, newError(CodeConfirmationRequired, "break-glass requires the exact confirmation string").
			WithDetail("expected_form", "BREAK-GLASS <binding-id>")
	}
	if strings.TrimSpace(req.Scope) == "" || strings.TrimSpace(req.Operator) == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "break-glass needs a scope and an operator identity")
	}
	if req.Window <= 0 || req.Window > MaxBreakGlassWindow {
		return nil, apierrors.Newf(apierrors.CodeInvalidRequest, "break-glass window must be between 1s and %s", MaxBreakGlassWindow)
	}
	binding, err := s.Store.GetBinding(ctx, req.DeploymentID, req.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "no such credential binding").WithDetail("binding_id", req.BindingID)
	}
	if binding.Class == domain.CredentialClassMachineEnrollment {
		return nil, newError(CodeRotationRefused, "machine enrollment credentials have no break-glass path outside vrooli-bridge")
	}
	if active, err := s.activeOperation(ctx, binding); err != nil {
		return nil, err
	} else if active != nil {
		return nil, newError(CodeRotationConflict, "another lifecycle operation is still open on this binding").WithDetail("rotation_id", active.ID)
	}
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	now := s.now()
	predecessor := binding.Version
	rotation := &domain.CredentialRotation{
		ID: id, DeploymentID: binding.DeploymentID, BindingID: binding.ID, Kind: domain.CredentialOperationBreakGlass, FromVersion: binding.Version.Number, State: domain.RotationPlanned, Consumers: initialConsumers(binding, now), Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now,
		BreakGlass: &domain.BreakGlassWindow{Scope: req.Scope, Operator: req.Operator, IssuedAt: now, ExpiresAt: now.Add(req.Window), PredecessorRef: fmt.Sprintf("%s@%d", binding.ID, predecessor.Number), ConfirmationRef: "confirmed:" + ExpectedBreakGlassConfirmation(req.BindingID)},
	}
	if err := s.Store.SaveRotation(ctx, rotation); err != nil {
		return nil, err
	}
	if err := s.record(ctx, rotation, "break-glass-audit", domain.RotationPlanned, "confirmed", "", map[string]any{"scope": req.Scope, "operator": req.Operator, "expires_at": rotation.BreakGlass.ExpiresAt, "predecessor": predecessor.Number}); err != nil {
		return rotation, err
	}
	// Run the ordinary stages; the emergency version becomes active only once
	// consumers verified it. Predecessor revocation waits for the window.
	rotation, err = s.advance(ctx, rotation, binding, req.Target, "", ResumeInput{})
	if err != nil && !apierrors.Is(err, CodePendingOperatorInput) {
		return rotation, err
	}
	return rotation, nil
}

// SweepBreakGlass closes expired windows: every emergency version past its
// window is rotated away and revoked, so revealed material stops working
// without operator action. Returns the follow-on operations.
func (s *Service) SweepBreakGlass(ctx context.Context, deploymentID string, target Target) ([]*domain.CredentialRotation, error) {
	rotations, err := s.Store.ListRotations(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	var followOns []*domain.CredentialRotation
	for i := range rotations {
		r := rotations[i]
		if r.Kind != domain.CredentialOperationBreakGlass || r.BreakGlass == nil || r.BreakGlass.AutoRevokedAt != nil || now.Before(r.BreakGlass.ExpiresAt) {
			continue
		}
		if !r.State.Terminal() {
			if _, err := s.Resume(ctx, target, r.ID, ResumeInput{OperatorConfirmed: true}); err != nil && !apierrors.Is(err, CodeRevocationIncomplete) {
				return followOns, err
			}
		}
		follow, err := s.Rotate(ctx, RotateRequest{Target: target, DeploymentID: deploymentID, BindingID: r.BindingID, RequestKey: "break-glass-expiry-" + r.ID})
		if err != nil {
			return followOns, err
		}
		followOns = append(followOns, follow)
		stored, err := s.Store.GetRotation(ctx, r.ID)
		if err != nil || stored == nil {
			return followOns, err
		}
		stored.BreakGlass.AutoRevokedAt = &now
		stored.Receipts = append(stored.Receipts, domain.CredentialReceipt{Step: "break-glass-expiry", State: string(stored.State), Outcome: "auto_revoked", At: now, Details: map[string]any{"follow_on_rotation": follow.ID, "follow_on_state": string(follow.State)}})
		stored.UpdatedAt = now
		if err := s.Store.SaveRotation(ctx, stored); err != nil {
			return followOns, err
		}
	}
	return followOns, nil
}

const (
	LifecyclePlanned    = "planned"
	LifecycleActive     = "active"
	LifecycleRenewalDue = "renewal_due"
	LifecycleExpired    = "expired"
	LifecycleRevoked    = "revoked"
)

const DefaultRenewalWindow = 30 * 24 * time.Hour

// BindingView is the metadata-only read surface: bindings, versions, acks and
// a derived lifecycle standing. The standing is deliberately not persisted;
// it is evaluated against the read clock so expiry cannot become stale state.
type BindingView struct {
	Binding         domain.CredentialBinding `json:"binding"`
	Acks            []domain.CredentialAck   `json:"acks"`
	LifecycleState  string                   `json:"lifecycle_state"`
	LifecycleDetail string                   `json:"lifecycle_detail,omitempty"`
	NextAction      string                   `json:"next_action,omitempty"`
}

func lifecycleStanding(binding domain.CredentialBinding, now time.Time) (string, string, string) {
	switch binding.State {
	case domain.CredentialBindingRevoked:
		return LifecycleRevoked, "This credential was revoked and cannot be used for new distribution.", "materialize a replacement credential"
	case domain.CredentialBindingPlanned:
		return LifecyclePlanned, "This credential has not been materialized on the target.", "materialize the credential"
	}
	if binding.Version.Expired(now) {
		return LifecycleExpired, "The active credential version has expired and is not release-ready.", "rotate the credential before retrying release"
	}
	if binding.Version.RenewalDue(now, DefaultRenewalWindow) {
		return LifecycleRenewalDue, "The active credential version is inside its renewal window.", "rotate the credential before it expires"
	}
	return LifecycleActive, "The active credential version is within its declared validity period.", ""
}

// ListBindings returns every binding with its acknowledgements.
func (s *Service) ListBindings(ctx context.Context, deploymentID string) ([]BindingView, error) {
	bindings, err := s.Store.ListBindings(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	out := make([]BindingView, 0, len(bindings))
	now := s.now()
	for _, b := range bindings {
		acks, err := s.Store.ListAcks(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		if acks == nil {
			acks = []domain.CredentialAck{}
		}
		state, detail, next := lifecycleStanding(b, now)
		out = append(out, BindingView{Binding: b, Acks: acks, LifecycleState: state, LifecycleDetail: detail, NextAction: next})
	}
	return out, nil
}

// GetRotation reads one operation.
func (s *Service) GetRotation(ctx context.Context, id string) (*domain.CredentialRotation, error) {
	rotation, err := s.Store.GetRotation(ctx, id)
	if err != nil {
		return nil, err
	}
	if rotation == nil {
		return nil, newError(CodeRotationNotFound, "no such credential operation").WithDetail("rotation_id", id)
	}
	return rotation, nil
}
