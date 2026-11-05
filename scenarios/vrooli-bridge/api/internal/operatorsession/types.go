// Package operatorsession stores Bridge's enrollment authority records. It
// stores only a public key and a scope ceiling; the client private key and all
// locally minted bearer material remain on the enrolled operator machine.
package operatorsession

import (
	"context"
	"fmt"
	"strings"
	"time"

	shared "github.com/vrooli/api-core/operatorsession"
)

type Record struct {
	Reference  string
	OperatorID string
	Mode       shared.Mode
	PublicKey  []byte
	Scopes     []string
	EnrolledAt time.Time
	Revoked    bool
}

type Store interface {
	Enroll(context.Context, Record) (Record, error)
	Lookup(context.Context, string) (Record, error)
}

type ErrNotFound struct{ Reference string }

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("operator enrollment %q not found", e.Reference)
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string {
	return fmt.Sprintf("operator enrollment %s: %s", e.Field, e.Reason)
}

func Validate(r Record) error {
	if strings.TrimSpace(r.Reference) == "" {
		return ErrInvalid{"reference", "required"}
	}
	if strings.TrimSpace(r.OperatorID) == "" {
		return ErrInvalid{"operator_id", "required"}
	}
	if r.Mode != shared.ModePersonal && r.Mode != shared.ModeShared && r.Mode != shared.ModeHosted {
		return ErrInvalid{"mode", "unsupported"}
	}
	if len(r.PublicKey) != 32 {
		return ErrInvalid{"public_key", "must be an Ed25519 public key"}
	}
	if r.EnrolledAt.IsZero() {
		return ErrInvalid{"enrolled_at", "required"}
	}
	return nil
}
