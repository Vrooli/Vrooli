package vector

import (
	"context"
	"errors"
	"strings"
)

var ErrGenerationDiskAdmissionUnavailable = errors.New("Source Ledger shared capacity disk admission is unavailable")

// UnavailableGenerationDiskAdmission is a fail-closed composition fallback.
// It keeps read/status operations mounted when the governed shared-capacity
// module cannot be adopted, while every generation mutation refuses before it
// writes source, generation, or operation state.
type UnavailableGenerationDiskAdmission struct{ Reason string }

func (a UnavailableGenerationDiskAdmission) Admit(context.Context, string, string, string, int64) (string, error) {
	if strings.TrimSpace(a.Reason) == "" {
		return "", ErrGenerationDiskAdmissionUnavailable
	}
	return "", errors.Join(ErrGenerationDiskAdmissionUnavailable, errors.New(a.Reason))
}

func (a UnavailableGenerationDiskAdmission) Complete(context.Context, string, string, bool) error {
	return ErrGenerationDiskAdmissionUnavailable
}
