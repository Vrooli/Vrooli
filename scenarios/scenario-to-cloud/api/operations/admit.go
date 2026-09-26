package operations

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/persistence"
)

// Admit persists a reviewed plan as an admitted operation under the request
// key. The same key with the same digest returns the existing operation
// (created=false); a different digest is a typed request_key_conflict from
// the store. Admission is the only way a cloud_operations row comes into
// existence: handlers review the digest, this package owns the record.
func Admit(ctx context.Context, repo *persistence.Repository, deploymentID, requestKey string, plan *execplan.Plan) (*domain.CloudOperation, bool, *apierrors.Error) {
	requestKey = strings.TrimSpace(requestKey)
	if requestKey == "" {
		return nil, false, apierrors.New(apierrors.CodeInvalidRequest, "request_key is required")
	}
	if repo == nil {
		return nil, false, apierrors.New(apierrors.CodeInternal, "Operation store is not configured")
	}
	if plan == nil {
		return nil, false, apierrors.New(apierrors.CodeInvalidRequest, "a compiled plan is required")
	}
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return nil, false, apierrors.Internal("Failed to encode plan", err)
	}
	id := uuid.New().String()
	op, err := repo.CreateOperation(ctx, &domain.CloudOperation{
		ID:           id,
		DeploymentID: deploymentID,
		RequestKey:   requestKey,
		PlanDigest:   plan.MustDigest(),
		Plan:         planJSON,
		State:        domain.OperationAdmitted,
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		if typed := apierrors.As(err); typed != nil {
			return nil, false, typed
		}
		return nil, false, apierrors.Internal("Failed to admit operation", err)
	}
	return op, op.ID == id, nil
}
