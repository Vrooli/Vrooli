// Sandbox + identity environment-variable construction for runner processes.
//
// Two concerns live here:
//
//   - Sandbox env vars enable sandbox-aware scenario lifecycle commands
//     (start, stop, restart) inside the agent's process. When an agent
//     runs in an overlayfs sandbox, its file changes live in the overlay's
//     upper/ layer; the Vrooli CLI lifecycle reads from the real repo by
//     default. These vars tell the CLI to redirect path resolution to the
//     overlay's merged/ directory so agent edits are visible to scenario
//     restart inside the sandbox.
//
//   - Identity tokens give agent processes a way to authenticate back to
//     agent-manager (or other Vrooli services) on behalf of the run. The
//     token is HMAC-signed with a server-side secret and revoked on
//     run completion.
//
// Sandbox + identity vars take precedence over caller-supplied env on key
// conflicts: they're system-managed and operators must not be able to
// shadow them.
//
// Note on VROOLI_SANDBOX_MERGED: workDir holds the *host* merged path
// (returned by Provider.GetWorkspacePath). For sandbox-routed launches the
// SandboxLauncher rewrites this value onto the server-reported agent-visible
// workspace path before posting to workspace-sandbox /processes — see
// translateEnvHostPaths in the sandbox launcher. Host launches keep the
// host path because the agent runs on the host filesystem.

package phases

import (
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const (
	envAgentIdentityToken = "VROOLI_AGENT_IDENTITY_TOKEN"
)

// SandboxEnvInput is the explicit input for SandboxEnvVars.
type SandboxEnvInput struct {
	RunMode   domain.RunMode
	SandboxID *uuid.UUID
	WorkDir   string
	ScopePath string
}

// SandboxEnvVars returns the VROOLI_SANDBOX_* env vars for a sandboxed run,
// or nil for in-place / unprepared runs (callers can unconditionally merge).
func SandboxEnvVars(in SandboxEnvInput) map[string]string {
	if in.RunMode != domain.RunModeSandboxed {
		return nil
	}
	if in.SandboxID == nil || in.WorkDir == "" {
		return nil
	}
	vars := map[string]string{
		"VROOLI_SANDBOX_ID":     in.SandboxID.String(),
		"VROOLI_SANDBOX_MERGED": in.WorkDir,
		"VROOLI_SANDBOX_SCOPE":  ".",
	}
	if in.ScopePath != "" {
		vars["VROOLI_SANDBOX_SCOPE"] = in.ScopePath
	}
	return vars
}

// IdentityEnvVars returns the identity env vars needed by agent processes.
// The token lets consumers prove which Agent Manager run is acting; service
// location is resolved through lifecycle discovery by the consumer.
func IdentityEnvVars(token string) map[string]string {
	if token == "" {
		return nil
	}
	return map[string]string{envAgentIdentityToken: token}
}

// MergeEnvInput bundles the inputs to MergeEnvVars — the three sources of
// environment variables that flow into a runner process.
type MergeEnvInput struct {
	Custom   map[string]string
	Sandbox  map[string]string
	Identity map[string]string
}

// AssembleRunEnvInput is the explicit input to AssembleRunEnv — the single
// place the three env sources (custom, sandbox, identity) are combined for a
// runner process. Both the fresh-run path (RunExecutor.MergedEnvVars) and the
// continue/wake path build their process env through this helper so they can
// never diverge: a continued turn must carry the same custom env, sandbox
// routing, and identity token as the original Execute call.
type AssembleRunEnvInput struct {
	// Custom is the caller-supplied (VROOLI_*-validated) env, persisted on
	// the run record as Run.CustomEnv.
	Custom map[string]string

	// Sandbox-routing inputs (ignored unless RunMode is sandboxed and a
	// SandboxID + WorkDir are present).
	RunMode   domain.RunMode
	SandboxID *uuid.UUID
	WorkDir   string
	ScopePath string

	// IdentityToken is the plaintext identity token for this turn. Empty when
	// identity is disabled; the continue/wake path must regenerate it first.
	IdentityToken string
}

// AssembleRunEnv merges custom, sandbox, and identity env with system vars
// (sandbox + identity) taking precedence over caller-supplied custom env.
// Returns nil when nothing was supplied so callers can short-circuit.
func AssembleRunEnv(in AssembleRunEnvInput) map[string]string {
	return MergeEnvVars(MergeEnvInput{
		Custom: in.Custom,
		Sandbox: SandboxEnvVars(SandboxEnvInput{
			RunMode:   in.RunMode,
			SandboxID: in.SandboxID,
			WorkDir:   in.WorkDir,
			ScopePath: in.ScopePath,
		}),
		Identity: IdentityEnvVars(in.IdentityToken),
	})
}

// MergeEnvVars combines the three env-var sources with sandbox + identity
// taking precedence on key conflicts. Returns nil when nothing was supplied
// so callers can short-circuit.
func MergeEnvVars(in MergeEnvInput) map[string]string {
	if len(in.Custom) == 0 && len(in.Sandbox) == 0 && len(in.Identity) == 0 {
		return nil
	}
	merged := make(map[string]string, len(in.Custom)+len(in.Sandbox)+len(in.Identity))
	for k, v := range in.Custom {
		merged[k] = v
	}
	// System vars override custom vars for security.
	for k, v := range in.Sandbox {
		merged[k] = v
	}
	for k, v := range in.Identity {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}
