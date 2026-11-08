package localworkbridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/vrooli/pyenv-go"
)

var ErrProcess = errors.New("local work bridge process unavailable")

// ApprovedBinding comes only from the owner's approved configuration provider,
// never a native event or environment variable. Digests pin reviewed config and
// entry code. Runtime authority is owned by the provider; pyenv checks freshness.
type ApprovedBinding struct {
	Runtime                                          pyenv.Spec
	LockSHA256, EntryPath, EntrySHA256, ConfigSHA256 string
	ModuleSHA256                                     map[string]string
	Config                                           json.RawMessage
	NotBefore, ExpiresAt                             time.Time
}

// BindingProvider must honor cancellation and return only authority approved for
// this bridge. Resolve must not retrieve credentials. Credential is lazy and may
// return only the exact requested purpose; no ambient fallback is permitted.
// No production implementation is installed by this disabled integration.
type BindingProvider interface {
	Resolve(context.Context) (ApprovedBinding, error)
	Credential(context.Context, string) ([]byte, error)
}

type ProcessRunner struct {
	enabled  bool
	provider BindingProvider
}

func NewProcessRunner(enabled bool, provider BindingProvider) *ProcessRunner {
	return &ProcessRunner{enabled: enabled, provider: provider}
}
