package domain

import (
	"context"
	"github.com/google/uuid"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	"time"
)

// AuthorizationGrant preserves consent separately from original run admission.
// ClaimsJSON and TokenHash never contain a plaintext bearer credential.
type AuthorizationGrant struct {
	ID               uuid.UUID                       `json:"id"`
	RunID            uuid.UUID                       `json:"run_id"`
	State            string                          `json:"state"`
	Policy           string                          `json:"policy"`
	Targets          []string                        `json:"targets"`
	DurationSeconds  int64                           `json:"duration_seconds"`
	Binding          cliutil.ProcessBinding          `json:"binding"`
	HarnessKind      string                          `json:"harness_kind"`
	HarnessSession   string                          `json:"harness_session"`
	CreatedAt        time.Time                       `json:"created_at"`
	RequestExpiresAt time.Time                       `json:"request_expires_at"`
	Grant            *requestidentity.OperationGrant `json:"grant,omitempty"`
	RevokedAt        *time.Time                      `json:"revoked_at,omitempty"`
	RevokedBy        string                          `json:"revoked_by,omitempty"`
	ClaimsJSON       string                          `json:"-"`
	TokenHash        string                          `json:"-"`
}

type AuthorizationRepository interface {
	Create(context.Context, *AuthorizationGrant) error
	Get(context.Context, uuid.UUID) (*AuthorizationGrant, error)
	List(context.Context, int) ([]*AuthorizationGrant, error)
	Replace(context.Context, *AuthorizationGrant, string) (bool, error)
}
