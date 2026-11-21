package credentials

import (
	"context"
	"fmt"
	"sort"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

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
		if created, ok := timeValue(receipt.Details["created_at"]); ok {
			version.CreatedAt = created
		}
		if expires, ok := expiryFromDetails(receipt.Details); ok {
			version.ExpiresAt = &expires
		}
	}
	return version
}

func timeValue(value any) (time.Time, bool) {
	switch parsed := value.(type) {
	case time.Time:
		return parsed, true
	case string:
		created, err := time.Parse(time.RFC3339Nano, parsed)
		if err == nil {
			return created, true
		}
	}
	return time.Time{}, false
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
