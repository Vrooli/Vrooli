# Operations

`grok` is organized as an `external-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, health, and
  `durable_data` metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Grok-specific Go logic that cannot be expressed through
  the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. The only specialised
Go package today is `cli/internal/upstream` (Grok's release-channel version
fetcher for the shared `upstream-check` verb). Install and binary download live
in `lib/install.sh` / `lib/common.sh`.

## Install model

- Strategy: download the **pinned** single-file artifact directly
  (`https://x.ai/cli/grok-<version>-<os>-<arch>`, direct-GCS mirror fallback),
  the opencode pattern — rather than running the official `curl | bash`
  installer. This honors the version pin, never mutates the operator's shell rc,
  and stays fully re-runnable.
- Target: `~/.local/bin/grok` (user-writable; updates never need root).
- Root-owned copy: refused with an actionable migrate message; never clobbered.
  `sudo` is reserved for privileged setup vacating such a copy.
- `uninstall` removes only the user-owned binary and **leaves `~/.grok`** intact
  (durable config/auth/session state — removing it is an operator's deliberate
  choice, not an uninstall side effect).

## Operator Checklist

- Keep upstream binary/install/version expectations declared in `resource.json`.
- Keep durable host state declared in `durable_data` (host-filesystem only);
  mark credential entries `sensitive: true`.
- Separate auth/config validation from raw binary detection.
- Prefer shared lifecycle and invoke behavior before adding resource-local
  commands.

## agent-manager runner integration (wired)

This resource provides **install + update + backup** parity with the other
coding agents (claude-code, codex, opencode). Grok is now also registered as an
`agent-manager` runner (`RUNNER_TYPE_GROK`).

The integration follows the established contract: agent-manager invokes the
**raw `grok` binary directly** (mirroring how it runs `opencode`/`codex`), via the
codec at `scenarios/agent-manager/api/internal/adapters/runner/codecs/grok.go`,
which owns the headless arg/stream contract. The headless surface used is
`grok -p <prompt> --output-format streaming-json [-m <model>] [--max-turns N]
[--always-approve] [--cwd <dir>]`, with `--resume <session-id>` for continuation.
There is deliberately **no `resource-grok run` passthrough**.

Capability honesty: grok's headless stdout surfaces assistant text + session id
but NOT tool-call/result events or token/cost, so the codec's `Capabilities()`
reports those as false. See
`scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md` (Codec capability
contract). Permission posture is the resource-managed native deny hook
(`SkipPermissionPrompt` → `--always-approve`); per-run rule enforcement is a
filed follow-up.
