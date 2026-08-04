# OpenCode Resource

OpenCode CLI resource for terminal-first coding workflows inside Vrooli.

## Intent

- Resource ID: `opencode`
- Category: `development`
- Driver: `external-cli`
- Portability tier: `partial`

## Architecture

This resource follows the `external-cli` template: the upstream `opencode`
binary is installed on `PATH` and invoked **directly** (no wrapper shim).
A thin Go CLI (`resource-opencode`) adds only the operator surface the raw
binary lacks — governed permissions and an upstream update-check.

- `resource.json` is the declarative authority for install, binary probing,
  version checks, environment exports, freshness, and lifecycle metadata.
- `cli/` is the single binary entrypoint (`resource-opencode`) and command
  wiring surface. Its only specialised internal package is
  `cli/internal/permissions` (+ `permissionscli`), which manages the
  `permission.bash` map in `opencode.json`.
- `lib/install.sh` downloads the pinned upstream release and lands the real
  `opencode` binary at `~/.local/bin/opencode`, then writes the default
  config and syncs provider auth.

### Who invokes what

- **agent-manager** runs the raw binary: `opencode run <prompt> --format json
  --print-logs [-m <model>] [--session <id>]`. The agent-manager codec
  (`scenarios/agent-manager/api/internal/adapters/runner/codecs/opencode.go`)
  owns the arg/stream contract.
- **operators / governance** use `resource-opencode` for lifecycle,
  `permissions`, and `upstream-check`.

## Configuration (single source of truth)

Raw `opencode` reads the default XDG locations, and every Vrooli surface
writes there — there is exactly one config/auth location:

- Config: `~/.config/opencode/opencode.json` (model + `permission.bash`)
- Auth:   `~/.local/share/opencode/auth.json` (provider keys)

`vrooli resource install opencode` writes a default model into
`opencode.json` (preserving any `permission.bash` map already present) and
syncs the OpenRouter key into `auth.json`.

### Providers

- **OpenRouter** is the wired cloud default
  (`openrouter/x-ai/grok-code-fast-1`); the key is resolved from Vault and
  written to `auth.json`.
- **Ollama** is auto-configured as a keyless local provider
  (OpenAI-compatible, `http://localhost:11434/v1`) when a local daemon is
  reachable and no OpenRouter key is present. Target a local model
  explicitly with `-m ollama/<model>`.

## Coding-role policy

OpenCode owns its concrete coding-role inventory in `model-policy.json`. Use
`resource-opencode policy validate`, `policy roles --json`, and `policy resolve
--role code.default --json` to inspect it. The response records the concrete
model, fallbacks, policy provenance, and native permission posture; Agent
Manager consumes that response at run creation but does not duplicate the
inventory or write OpenCode configuration.

## Permissions

Manage the bash patterns OpenCode is allowed (or denied) to run via the
`permissions` subgroup. The adapter owns `permission.bash` entries in
`~/.config/opencode/opencode.json`; hand-written entries and unrelated
config keys (including `model`) round-trip untouched.

```bash
# Block git stash for OpenCode (motivating example)
resource-opencode permissions deny 'git stash *'

# View managed patterns
resource-opencode permissions list
resource-opencode permissions show --raw

# Detect drift since the last Vrooli write
resource-opencode permissions drift-check

# Check installed OpenCode version against the pinned upstream
resource-opencode permissions doctor
```

For declarative automation, use `permissions plan --document desired.json --json` and `permissions reconcile --document desired.json --json`. The strict v1 document contains `schema_version`, optional `scope: "user"`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged config, and reports desired/live fingerprints, native paths, changes, and `native` enforcement.

Mutating verbs (`deny`, `allow`, `ask`, `remove`, `reset`) refuse agent
callers (detected via `cliutil.DetectCallerKind` — `VROOLI_CALLER=agent`,
`CLAUDECODE=1`, opencode PID-match, etc.) unless
`--i-was-explicitly-authorized` is passed. Read verbs (`list`, `show`,
`drift-check`, `doctor`) are always allowed.

OpenCode evaluates `permission.bash` as last-match-wins. The adapter writes
alphabetically-sorted keys, so `*` (a likely default wildcard) sorts before
letters and specific patterns end up later in iteration order, winning the
match. If you need different ordering, hand-edit and `drift-check` will
surface it.

Upstream docs: <https://opencode.ai/docs/permissions/>.

## Update check

```bash
# Compare the installed opencode against the latest upstream release.
resource-opencode upstream-check
resource-opencode upstream-check --json
```

Read-only and agent-safe: it reports `up-to-date | behind | ahead | unknown`
against the GitHub releases API and never hard-fails on a network error.
The same verb exists on the `codex` and `claude-code` resources.

## Usage

```bash
# Install the upstream binary + Go CLI through the shared control plane
vrooli resource install opencode

# Check binary availability and health
resource-opencode status
```

## Notes

- This resource wraps an external CLI. It should stay thin by default.
- Do not grow `cli/main.go` into a second operator framework.
- Do not reintroduce a `resource-opencode run` passthrough — the contract is
  raw-binary invocation.

### web-console conversation capture

When `opencode` runs inside a [web-console](../../scenarios/web-console) terminal
pane, web-console captures its conversation into the semantic messages feed
(search, TTS, replay). It does this over the HTTP API — web-console owns a
single loopback-only `opencode serve` instance, subscribes to its `/event` SSE
stream, and reconciles transcripts via `GET /session/{id}/message`. It does not
parse the OpenCode SQLite store and reads no provider credentials. Because
OpenCode shares one global storage dir across processes, a pane's session is
attributed to it only when the match is unambiguous (directory + creation time +
uniqueness); concurrent `opencode` panes in the same directory are skipped rather
than misrouted. See `scenarios/web-console/docs/guides/CONVERSATION_TRACKING.md`.

## Model catalog operations

`resource-opencode models list --json` reads the runner's current model listing without a completion call. `resource-opencode models resolve --model <id> --json` returns the resource-owned canonical pricing identity. Run `resource-opencode policy validate --against-live --json` after a retarget. Record the provider slug and observation date, keep fallbacks no more expensive than their primary where evidence permits, and review catalog changes explicitly within the 14-day staleness budget.

## References

- [OpenCode](https://opencode.ai)
