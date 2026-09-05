package accounts

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrMachineBindingNotFound  = errors.New("machine binding not found")
	ErrMachineBindingAmbiguous = errors.New("machine binding is ambiguous")
	ErrMachineBindingInvalid   = errors.New("invalid machine binding")
	ErrMachineExchangeRefused  = errors.New("machine principal exchange refused")
)

// MachineBinding maps one local operating-system principal to an account. A
// principal may have many historical/alternate rows, but resolution accepts
// exactly one default row and never guesses.
type MachineBinding struct {
	ID             string
	MachineID      string
	LocalPrincipal string
	AccountID      string
	RealmID        string
	IsDefault      bool
	LinkedAt       time.Time
}

// MachineBindingStore is the persistence seam used by linking and the local
// exchange service.
type MachineBindingStore interface {
	LinkMachineBinding(context.Context, MachineBinding) (MachineBinding, error)
	ResolveDefaultMachineBinding(context.Context, string, string) (MachineBinding, error)
}

// BreakGlassProvisioner creates the owner-only offline capability during the
// same authenticated link operation. The accounts domain does not know where
// key material lives; that policy belongs to the trust-posture seam.
type BreakGlassProvisioner interface {
	Provision(context.Context, string, string, []string, time.Time) error
}

// BreakGlassIssuer signs a short-lived credential from the material created by
// the provisioner. It is intentionally a separate seam so linking never
// returns private key material.
type BreakGlassIssuer interface {
	Issue(context.Context, string, string, []string, time.Time) (string, time.Time, error)
}

func validateMachineBinding(b MachineBinding) error {
	if strings.TrimSpace(b.MachineID) == "" || strings.TrimSpace(b.LocalPrincipal) == "" || strings.TrimSpace(b.AccountID) == "" || strings.TrimSpace(b.RealmID) == "" {
		return ErrMachineBindingInvalid
	}
	return nil
}
