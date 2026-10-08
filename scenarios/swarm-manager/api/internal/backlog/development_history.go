package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"swarm-manager/internal/identity"
)

var errDevelopmentHistoryMissing = errors.New("development retained history is unavailable")

const developmentMaxHistory = 64
const developmentMaxHistoryBytes = 4 * 1024 * 1024

func developmentHistoryDigest(record identity.DevelopmentHistoryRecord) string {
	record.SnapshotDigest = ""
	return developmentDigest(record)
}
func developmentCaptureHistory(ref *api.DevelopmentReference, contract DevelopmentContract) (identity.DevelopmentHistoryRecord, error) {
	record := identity.DevelopmentHistoryRecord{ArtifactBytes: map[string][]byte{}}
	var err error
	record.Reference, err = protojson.Marshal(ref)
	if err != nil {
		return record, err
	}
	record.Contract, err = json.Marshal(contract)
	if err != nil {
		return record, err
	}
	for _, a := range contract.Artifacts {
		raw, err := developmentReadFile(contract.RetainedRoot, a.RelativePath)
		if err != nil || int64(len(raw)) != a.SizeBytes || strings.TrimPrefix(developmentBytesDigest(raw), "sha256:") != strings.TrimPrefix(a.SHA256, "sha256:") {
			return record, ErrDevelopmentRefused
		}
		record.ArtifactBytes[a.ID] = raw
	}
	raw, err := json.Marshal(record)
	if err != nil || len(raw) > developmentMaxHistoryBytes {
		return record, ErrDevelopmentRefused
	}
	record.SnapshotDigest = developmentHistoryDigest(record)
	return record, nil
}
func developmentDecodeHistory(ref *api.DevelopmentReference, record identity.DevelopmentHistoryRecord) (DevelopmentContract, error) {
	if ref == nil || record.SnapshotDigest != developmentHistoryDigest(record) {
		return DevelopmentContract{}, ErrDevelopmentRefused
	}
	retained := new(api.DevelopmentReference)
	if protojson.Unmarshal(record.Reference, retained) != nil || !proto.Equal(ref, retained) {
		return DevelopmentContract{}, ErrDevelopmentRefused
	}
	var contract DevelopmentContract
	if json.Unmarshal(record.Contract, &contract) != nil || developmentDigest(contract) != ref.ContractDigest || developmentDigest(contract.Subject) != ref.CommissionSubjectDigest || contract.Subject.Effort != ref.EffortId || contract.Subject.Validate() != nil || !developmentValidBinding(contract.Owner) || len(contract.Artifacts) != len(record.ArtifactBytes) {
		return contract, ErrDevelopmentRefused
	}
	for _, a := range contract.Artifacts {
		raw, ok := record.ArtifactBytes[a.ID]
		if !ok || int64(len(raw)) != a.SizeBytes || strings.TrimPrefix(developmentBytesDigest(raw), "sha256:") != strings.TrimPrefix(a.SHA256, "sha256:") {
			return contract, ErrDevelopmentRefused
		}
	}
	return contract, nil
}

// Historical visibility is decided by CURRENT verified setup. Retaining an
// original ACL never reinstalls an old credential, reader or commissioning grant.
func (o *DevelopmentOwner) historical(ctx context.Context, ref *api.DevelopmentReference, write bool) (identity.EffortControl, identity.DevelopmentHistoryRecord, DevelopmentContract, error) {
	if ref == nil {
		return identity.EffortControl{}, identity.DevelopmentHistoryRecord{}, DevelopmentContract{}, ErrDevelopmentRefused
	}
	if _, _, err := o.authorize(ctx, ref.EffortId, write); err != nil {
		return identity.EffortControl{}, identity.DevelopmentHistoryRecord{}, DevelopmentContract{}, err
	}
	current, err := o.finite.store.Load(ref.EffortId)
	if err != nil {
		return current, identity.DevelopmentHistoryRecord{}, DevelopmentContract{}, err
	}
	if current.Development == nil {
		return current, identity.DevelopmentHistoryRecord{}, DevelopmentContract{}, errDevelopmentHistoryMissing
	}
	record, ok := current.Development.History[developmentDigest(ref)]
	if !ok {
		return current, record, DevelopmentContract{}, errDevelopmentHistoryMissing
	}
	contract, err := developmentDecodeHistory(ref, record)
	return current, record, contract, err
}
func (o *DevelopmentOwner) replayHistorical(ctx context.Context, ref *api.DevelopmentReference, key, action, requestDigest string) (*api.DevelopmentDecisionResponse, bool, error) {
	if ref == nil {
		return nil, false, nil
	}
	if _, _, err := o.authorize(ctx, ref.EffortId, true); err != nil {
		return nil, false, err
	}
	var receipt *api.DevelopmentDecisionResponse
	found := false
	err := o.finite.store.withFiniteLock(ref.EffortId, func() error {
		current, err := o.finite.store.Load(ref.EffortId)
		if err != nil {
			return err
		}
		if current.Development == nil {
			return nil
		}
		record, ok := current.Development.Decisions[key]
		if !ok {
			return nil
		}
		found = true
		if record.RequestDigest != requestDigest {
			return ErrDevelopmentRefused
		}
		receipt, err = developmentDecodeReceipt(record)
		if err != nil || receipt.RequestId != key || receipt.Action != action || !proto.Equal(receipt.Reference, ref) {
			return ErrDevelopmentRefused
		}
		if _, _, _, err := o.historical(ctx, ref, true); err != nil {
			return err
		}
		return nil
	})
	return receipt, found, err
}
