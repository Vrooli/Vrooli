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
- `resource-opencode install-direct` downloads the pinned upstream release and lands the real
  `opencode` binary at `~/.local/bin/opencode`, then writes the default
  config and syncs provider auth.

### Who invokes what

- **agent-manager** runs the raw binary: `opencode run <prompt> --format json
  --print-logs [-m <model>] [--session <id>]`. The agent-manager codec
  (`scenarios/agent-manager/api/internal/adapters/runner/codecs/opencode.go`)
  owns the arg/stream contract.
- **operators / governance** use `resource-opencode` for lifecycle,
  `permissions`, and `upstream-check`.

## OpenCode 1 and OpenCode 2

The resource supports both API generations at runtime. Web Console detects the
installed binary's major version (`opencode --version`) and uses the matching
HTTP contract: `/session` + `/global/event` for 1.x, and authenticated
`/api/session` + `/api/event` for 2.x. The version choice lives in
`scenarios/web-console/api/backends/opencode`; callers keep one interface, so
session listing, message projection, permissions, activity, and rewind do not
need version branches of their own.

Configuration stays in the V1 `opencode.json` shape. OpenCode 2 normalizes those
fields in memory, so both versions read the same generated file and provider
blocks; do not maintain a second V2-only config. Native `permission` rules still
apply after V2 normalization, and the adapter projects a V2 plugin through
`ctx.permission.hook("evaluate")` so ask and deny remain distinct. The posture
stays `hook_unverified` until a live V2 canary proves the plugin fires.

The manifest declares both release lines. The default channel is V1
(`opencode-ai`); the `v2` channel is `@opencode/cli`. Each channel has its own
pinned version, checksum, install prefix, and rollback history. Use:

```bash
vrooli resource update opencode --check --channel v2   # read-only preflight
vrooli resource update opencode --channel v2           # safe, governed upgrade
vrooli resource update opencode --channel v2 --action rollback
```

A plain `vrooli resource update opencode` follows the installed major, so a V2
host is never silently downgraded to V1. The one-command V2 upgrade uses the
same ownership checks, Dependency Analyzer gate, offline version qualification,
and retained rollback prefix as V1. Agent Manager detects the installed major
and adapts its run contract: V1 pins `--dir` and `--variant`; V2 uses the
process working directory (plus `--standalone`) and `model#variant`.

## Configuration (single source of truth)

Raw `opencode` reads the default XDG locations, and every Vrooli surface
writes there — there is exactly one config/auth location:

- Config: `~/.config/opencode/opencode.json` (model + `permission.bash`)
- Policy hook: `~/.config/opencode/plugins/vrooli-policy.js` (managed when deny
  rules exist through OpenCode's `tool.execute.before` plugin seam)
- Auth:   `~/.local/share/opencode/auth.json` (provider keys)

`vrooli resource install opencode` writes a default model into
`opencode.json` (preserving any `permission.bash` map already present) and
syncs the OpenRouter key into `auth.json`.

### Providers

- **OpenRouter** is the wired cloud default
  (`openrouter/x-ai/grok-code-fast-1`); the key is injected by the credential
  authority and written to `auth.json`.
- **OrcaRouter** and **Abliteration** are written as OpenAI-compatible provider
  blocks when their optional credentials (`ORCAROUTER_API_KEY`, `ABLIT_KEY`) are
  present. Each block also registers the provider resource's chat-capable
  models, read from `resource-<provider> policy models --json`. OpenCode only
  discovers catalogs for its built-in providers, so a custom provider without
  those model entries rejects every model ID as invalid. Set
  `OPENCODE_PROVIDER=orcarouter` or `OPENCODE_PROVIDER=abliteration` to make one
  the active provider; otherwise they remain available in the picker without
  changing the default tier order.
- **Ollama** is configured as a keyless local provider at the loopback managed
  activity edge (`http://127.0.0.1:11434/api`). `resource-opencode ensure`
  projects installed models that serve an allowed `chat` or `generate` role in
  `resources/ollama/model-policy.json` into the provider block, so OpenCode's
  model picker follows current Ollama policy and local-inference rights. Each
  ensure also removes previously projected policy models that are now denied.
  Operator-added entries outside the policy and their display names are
  preserved. Projected
  entries follow the policy model's `context_window_tokens` (capped at 32768);
  the coding default still uses `OPENCODE_OLLAMA_NUM_CTX` (default 16384).
  Role `max_tokens` becomes the output limit. Target a model outside the
  policy explicitly with `-m ollama/<model>`; the Ollama activity edge still
  checks current rights before forwarding inference.

### Prepare and select a local model

Use the role-based command to make a policy model ready and select it in one
operation:

```bash
resource-opencode local-model select --role code.local
```

The command resolves the role and checks local inference rights through Ollama
policy before changing runtime state. It creates a short-lived reservation
for the target model plus any models already resident in Ollama. Ollama's
reservation owner obtains strict no-reclaim admission before startup; the
managed resource then starts with that same exact reservation, so its ordinary
lifecycle admission cannot substitute a broader manifest rung. A denied
request does not begin startup or acquisition. The
reservation remains active through acquisition, tool and chat readiness probes,
then hands the loaded footprint back to Ollama's companion. A missing model is
acquired through `resource-ollama ensure`, which owns disk admission and the
managed daemon lock. The command requires a passing model doctor result, a
structured tool-call smoke, and a successful short chat before updating
OpenCode's active model. If capacity, acquisition, or readiness fails, it
stops before changing the active OpenCode model and releases its reservation
when possible. Routine config ensure preserves the active local model while
current rights allow it. Set an explicit non-Ollama `OPENCODE_PROVIDER` to
choose another provider.
If Ollama already has the target resident but its aggregate claim is too small,
the broker corrects accounting only after a fresh, complete GPU process snapshot
attributes enough VRAM to Ollama. That correction does not authorize another
model load. If host attribution is unavailable or ambiguous, selection fails
closed and leaves the active OpenCode model unchanged.
The OpenCode picker includes only roles whose current trusted-context
local-inference rights are allowed. Ollama's activity edge checks those rights
again against the exact installed model digest on every native or compatible
inference request, so a stale OpenCode config cannot bypass a changed decision.
The `chat.uncensored` role is currently refused because its artifact license
evidence is unreviewed; use an eligible role such as `code.local` until that
review is complete. OpenCode is pinned to the loopback activity-edge endpoint,
so an inherited `OLLAMA_HOST` setting cannot route around the rights check.
Use `--json` for a machine-readable completion receipt and
`--timeout-seconds` to bound the complete operation.

## Coding-role policy

OpenCode owns its concrete coding-role inventory in `model-policy.json`. Use
`resource-opencode policy validate`, `policy roles --json`, and `policy resolve
--role code.default --json` to inspect it. The response records the concrete
model, fallbacks, policy provenance, and native permission posture; Agent
Manager consumes that response at run creation but does not duplicate the
inventory or write OpenCode configuration.

For a role's explicit choices, `resource-opencode policy choices --role
code.default --json` returns the resource-declared model/source pairs,
provenance, and policy digest. `resource-opencode policy select --role
code.default --model <model> --source <source> --json` resolves one exact
choice. Unknown choices fail; when the declared source is ambiguous, the caller
must choose it explicitly. These commands report policy evidence only: they do
not claim that a credential is present, that a provider account is bound to a
launch, or that a paid route is qualified. Managed run creation continues to
use `policy resolve` and its immutable default-role evidence.

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

The plugin sends normalized bash tool events to the local policy runner and
applies the runner's decision. V1 writes `plugins/vrooli-policy.js` and uses
`tool.execute.before`, which can only allow or block, so an `ask` result is
blocked. V2 writes `plugins/vrooli-policy-v2.js` and uses
`ctx.permission.hook("evaluate")`, preserving `allow`, `ask`, and `deny`. Only
the plugin for the installed major is present; switching majors removes the
incompatible module so a rollback cannot load the wrong format.

The CLI reports `hook_unverified` until an installed-version live canary proves
that the plugin loads and the decision path fires; a plugin file by itself is
not evidence.

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
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
