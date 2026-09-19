// Package hosttool is image-tools' seam onto the platform host-tool installer.
// It ensures a backend's host binary (e.g. realesrgan-ncnn-vulkan) is present by
// shelling the root `vrooli host install <tool> --json` command through the
// typed vrooli-cli-go client — the same compose-don't-reinvent pattern
// capabilities uses for `host inventory`. The on-demand EnsureBackend RPC runs
// it as a durable job; the doctor uses it for readiness.
package hosttool

import (
	"context"
	"strings"

	"image-tools/internal/ai"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// EnsureJobOperation is the durable-job operation name a backend host-tool
// install runs under (mirrors models.InstallJobOperation).
const EnsureJobOperation = "backend_ensure"

// EnsurePayload is the durable-job payload for a host-tool ensure.
type EnsurePayload struct {
	Tool string `json:"tool"`
}

// Status is the host-tool ensure verdict: a thin alias over the CLI wire
// contract so callers consume one shape end-to-end.
type Status = cliv1.CliHostInstallStatus

// installer is the slice of vrooli-cli-go the ensurer needs; an interface so
// tests inject a fake without shelling a real CLI.
type installer interface {
	HostInstall(ctx context.Context, tool string, dryRun bool) (*cliv1.CliHostInstallStatus, error)
}

// Ensurer ensures host tools through the root vrooli CLI.
type Ensurer struct {
	client installer
}

// NewEnsurer returns an Ensurer backed by the default vrooli CLI client.
func NewEnsurer(opts ...vroolicli.Option) *Ensurer {
	return &Ensurer{client: vroolicli.New(opts...)}
}

// NewEnsurerWithClient injects a custom installer (tests / alternate clients).
func NewEnsurerWithClient(c installer) *Ensurer { return &Ensurer{client: c} }

// Inspect reports a host tool's readiness without downloading (dry-run).
func (e *Ensurer) Inspect(ctx context.Context, tool string) (*Status, error) {
	return e.client.HostInstall(ctx, tool, true)
}

// Ensure installs a host tool (fetch + verify), blocking until terminal. The
// caller's context owns the deadline (a durable server-side job).
func (e *Ensurer) Ensure(ctx context.Context, tool string) (*Status, error) {
	return e.client.HostInstall(ctx, tool, false)
}

// ToolForOperation resolves the host tool that serves an operation via the
// provider bindings (the single source of truth). Reports false when no
// host-tool-backed backend serves the op.
func ToolForOperation(operation string) (string, bool) {
	return ai.HostToolForOperation(strings.TrimSpace(operation))
}

// KnownTool reports whether tool is referenced by any backend provider binding —
// the guard that keeps EnsureBackend from shelling arbitrary tool names.
func KnownTool(tool string) bool {
	tool = strings.TrimSpace(tool)
	for _, b := range ai.HostToolBindings() {
		if b.HostTool == tool {
			return true
		}
	}
	return false
}
