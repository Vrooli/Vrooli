package capabilities

import (
	"context"
	"errors"
	"io"
)

// ResourceController is the lifecycle seam (SEAMS.md row
// "capabilities.ResourceController"). Production wires CLIController,
// which shells out to `vrooli resource …`; tests wire a recording fake.
// All packages that need to mutate a local resource's process state
// MUST call through this interface — never invoke `vrooli`, docker, or
// systemctl directly elsewhere.
type ResourceController interface {
	// Start brings the resource up. Idempotent: starting an already-up
	// resource must succeed.
	Start(ctx context.Context, slug string) error
	// Stop brings the resource down. Idempotent.
	Stop(ctx context.Context, slug string) error
	// Restart bounces the resource. Equivalent to Stop+Start with
	// controller-defined wait semantics.
	Restart(ctx context.Context, slug string) error
	// Logs returns a streaming reader of stdout/stderr lines (newline
	// delimited). Caller closes the reader; the controller is
	// responsible for terminating the underlying child process when
	// the reader is closed.
	Logs(ctx context.Context, slug string, follow bool, tailLines int) (io.ReadCloser, error)
	// PullModel triggers a model download on a model-server resource
	// (ollama). Returns an error for resources that don't support it;
	// the handler converts that into CodeFailedPrecondition.
	PullModel(ctx context.Context, model string) error
}

// ErrControllerUnavailable is the sentinel error CLIController returns
// when the `vrooli` binary is not on PATH. Handlers translate it to
// connect.CodeUnavailable.
var ErrControllerUnavailable = errors.New("vrooli resource controller unavailable: vrooli binary not on PATH")

// ResourceSlugForProviderID maps a stable provider_id (capabilities.Known[].ID)
// to the resource slug accepted by `vrooli resource <verb> <slug>`. The
// second return is false for provider_ids that are not local-tier
// (e.g. "openrouter") or unknown.
func ResourceSlugForProviderID(id string) (string, bool) {
	switch id {
	case "whisper-stt":
		return "whisper", true
	case "kokoro-tts":
		return "kokoro", true
	case "speaker-verification":
		return "speaker-verification", true
	case "ollama":
		return "ollama", true
	default:
		return "", false
	}
}

// IsLocalProvider reports whether a provider_id corresponds to a
// local-tier resource we own.
func IsLocalProvider(id string) bool {
	_, ok := ResourceSlugForProviderID(id)
	return ok
}

// SupportsPullModel reports whether a provider_id accepts PullModel.
// Currently only `ollama`.
func SupportsPullModel(id string) bool {
	return id == "ollama"
}
