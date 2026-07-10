# Grok Build Resource

xAI's Grok Build coding CLI (`grok`) as an **optional** `external-cli` resource
inside Vrooli.

## Intent

- Resource ID: `grok`
- Category: `developer-tooling`
- Driver: `external-cli`
- Portability tier: `partial`
- Optionality: `enabled: true, required: false` — a missing Grok subscription
  must never fail `make setup`.

## Architecture

This resource follows the `external-cli` template: the upstream `grok` binary
is installed on `PATH` and invoked **directly** (no wrapper shim). A thin Go CLI
(`resource-grok`) adds only the operator surface the raw binary lacks — lifecycle
and an upstream update-check.

- `resource.json` is the declarative authority for install, binary probing,
  version checks, environment exports, freshness, lifecycle metadata, and the
  `durable_data` block that `data-backup-manager` discovers.
- `cli/` is the single binary entrypoint (`resource-grok`). Its only specialised
  internal package is `cli/internal/upstream`, which resolves Grok's
  release-channel version pointer for `upstream-check` (Grok is not on npm or
  GitHub releases). The `install`/`version`/`discovery`/`env` packages are
  reserved stubs for future runner-integration work.
- `lib/install.sh` downloads the **pinned** upstream single-file binary directly
  (the opencode pattern) and lands the real `grok` binary at `~/.local/bin/grok`
  — sudo-free, honoring `resource.json upstream_cli.version_pinned`, and without
  mutating the operator's shell rc.

> **Wired as an agent-manager runner.** Beyond install + update + backup +
> command-permission parity, Grok is registered as an `agent-manager` runner
> (`RUNNER_TYPE_GROK`); agent-manager invokes the raw `grok` binary headlessly
> via its codec. See `docs/OPERATIONS.md` and
> `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`.

## Install / update / uninstall

```bash
# Install the upstream binary + Go CLI through the shared control plane
vrooli resource install grok

# Check binary availability and health
resource-grok status

# Update to the pinned version (idempotent reinstall; never automatic)
bash resources/grok/lib/install.sh update

# Remove only the user-owned binary (leaves ~/.grok durable state intact)
bash resources/grok/lib/install.sh uninstall
```

The install lands `~/.local/bin/grok` with **no sudo** and is fully re-runnable.
A root-owned `grok` already on `PATH` is refused with an actionable message
(this resource never installs `grok` with `sudo`); a pre-existing `grok` that
shadows the user install on `PATH` is warned about after install.

## Authentication

Grok is authenticated by the operator — this resource never stores or forwards
credentials beyond documenting how to provide them.

- **Interactive sign-in:** run `grok login` (browser sign-in for SuperGrok /
  X Premium+). Credentials land in `~/.grok/auth.json` (auto-managed).
- **Headless / deployment:** export `GROK_DEPLOYMENT_KEY=<key>` before running
  `grok` (takes precedence over `auth.json`).

Inspect what Grok discovers for a directory (read-only): `grok inspect`.

## Permissions (command allow/deny)

`resource-grok permissions` manages which shell commands Grok may run — the same
uniform allow/ask/deny surface the `claude-code` and `codex` resources expose.
Unlike Codex (whose rules are intent-only), **Grok enforces these natively**:

- Rules are written into Grok's native `[permission]` table in
  `~/.grok/config.toml` (user scope) or `~/.grok/requirements.toml` (`--scope
  admin`, higher precedence). Grok evaluates them `deny > ask > allow`.
- Every `Bash(...)` deny rule is also paired with a **PreToolUse backstop hook**
  under `~/.grok/hooks/` that hard-denies the matching command *before* any other
  check and applies even under `grok --always-approve`. (Grok runs PreToolUse
  hooks earliest and honours an explicit `{"decision":"deny"}`.)

```bash
# Block a dangerous pattern (note: flags come BEFORE the pattern).
resource-grok permissions deny 'Bash(rm -rf *)'

# Keep common dev commands allowed.
resource-grok permissions allow 'Bash(git *)'

# Inspect / audit.
resource-grok permissions list
resource-grok permissions doctor        # confirms version + enforcement wiring
resource-grok permissions drift-check   # detects hand-edits since the last write
```

For declarative automation, use `permissions plan --scope user|admin --document desired.json --json` and `permissions reconcile --scope user|admin --document desired.json --json`. The strict v1 document contains `schema_version`, matching `scope`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged TOML, and reports desired/live fingerprints, native paths, changes, and the `hook_backed` enforcement posture.

Mutating verbs (`deny/allow/ask/remove/reset`) are gated by the shared
`agentpolicy` substrate: a **detected coding-agent caller is refused** so an agent
cannot disarm its own gate. A human re-runs with `--i-was-explicitly-authorized`
(placed before the pattern). Rule patterns use the Claude/Grok vocabulary
(`Bash(git *)`, `Read(src/**)`, `Grep`, …). See
`~/.grok/docs/user-guide/22-permissions-and-safety.md` for Grok's native model.

The detector recognises **standalone grok** via the `GROK_AGENT=1` sentinel grok
injects into its tool subprocesses (confirmed by `/proc` env-diff 2026-06-28), in
addition to the Vrooli-spawned signals (sandbox / agent-manager / swarm-manager)
and the `VROOLI_CALLER=agent` override. So a grok session — under Vrooli or run
directly — that tries to weaken its own permission rules is refused. See
`packages/cli-core/docs/reference/agent-detection-signals.md` for the evidence.

## Data locations (declared for backup)

Grok keeps its durable state under `~/.grok`. The `durable_data` block in
`resource.json` declares the irreplaceable entries so `data-backup-manager`
discovery surfaces them as one-click backup targets:

| Entry | Path | Notes |
|-------|------|-------|
| `sessions` | `~/.grok/sessions/` | Persisted conversation transcripts (by working directory) |
| `memory` | `~/.grok/memory/` | Cross-session memory files + index |
| `config` | `~/.grok/config.toml` | Main configuration |
| `skills` | `~/.grok/skills/` | User-scoped skill definitions |
| `agents` | `~/.grok/agents/` | User-scoped agent definitions |
| `auth` | `~/.grok/auth.json` | **Sensitive** credentials — flagged, never auto-accepted |

Reconstructable state (`~/.grok/logs/`, `~/.grok/docs/`, `active_sessions.*`,
`managed_config.toml`, completions) is intentionally **not** declared.

## Update check

```bash
# Compare the installed grok against the latest upstream release.
resource-grok upstream-check
resource-grok upstream-check --json
```

Read-only and agent-safe: it reports `up-to-date | behind | ahead | unknown`
against the xAI release-channel pointer (`https://x.ai/cli/<channel>`, GCS
mirror fallback) and never hard-fails on a network error. The same verb exists
on the `codex` and `opencode` resources.

## Notes

- This resource wraps an external CLI. It should stay thin by default.
- Do not grow `cli/main.go` into a second operator framework.
- Do not add a `resource-grok run` passthrough — when a runner is wired later,
  the contract is raw-binary invocation (mirroring opencode/codex).

### web-console conversation capture

When `grok` runs inside a [web-console](../../scenarios/web-console) terminal pane,
web-console captures its conversation into the semantic messages feed (search,
TTS, replay). It does this by injecting a per-pane `GROK_HOME` and tailing the
session's `updates.jsonl` ACP stream — it never scrapes terminal output. The
operator's real `~/.grok` auth/config is symlinked into each per-pane home, so
login keeps working; only the `sessions/` subtree is isolated. Capture is
read-only: web-console stores natural-language user/assistant text only (not
thought chunks or tool arguments) and reads no auth material. See
`scenarios/web-console/docs/guides/CONVERSATION_TRACKING.md`.

## References

- [Grok Build CLI](https://x.ai/cli)
