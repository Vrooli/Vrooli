# Antigravity CLI Resource

Google's Antigravity coding CLI (binary `agy`) as an **optional** `external-cli`
resource inside Vrooli.

> Antigravity is Google's go-forward terminal coding agent for individual
> (Pro / Ultra / free-tier) users — it **replaces Gemini CLI**, whose individual
> serving ended 2026-06-18. This resource models the *coding agent*; it is
> unrelated to the hosted **Gemini API** resource (`resources/gemini`).

## Intent

- Resource ID: `antigravity`
- Category: `developer-tooling`
- Driver: `external-cli`
- Portability tier: `partial` (Antigravity is early/preview; no musl build)
- Optionality: `enabled: true, required: false` — a missing Antigravity
  install/subscription must never fail `make setup`.

## Architecture

This resource follows the `external-cli` template: the upstream `agy` binary is
installed on `PATH` and invoked **directly** (no wrapper shim). A thin Go CLI
(`resource-antigravity`) adds only the operator surface the raw binary lacks —
lifecycle, an upstream update-check, and shared command-permission governance.

- `resource.json` is the declarative authority for install, binary probing,
  version checks, environment exports, freshness, lifecycle metadata, and the
  `durable_data` block that `data-backup-manager` discovers.
- `cli/` is the single binary entrypoint (`resource-antigravity`). Its
  specialised internal packages are `cli/internal/upstream` (Antigravity's
  release-manifest version fetcher for `upstream-check`) and
  `cli/internal/permissions` + `cli/internal/permissionscli` (the native
  `permissions` governance). The `install`/`version`/`discovery`/`env` packages
  are reserved stubs for future runner-integration work.
- `lib/install.sh` queries the upstream per-platform manifest and downloads the
  artifact it points at directly (the opencode/grok pattern), landing the real
  `agy` binary at `~/.local/bin/agy` — sudo-free and without mutating the
  operator's shell rc.

> **NOT wired as an agent-manager runner.** This resource delivers **install +
> update + backup + shared permissions + caller detection** parity with the
> other coding agents. Registering Antigravity as an `agent-manager` runner
> (`RUNNER_TYPE_ANTIGRAVITY`, a codec, model registry) is a deliberate
> **follow-up plan** — exactly as the Grok runner work followed the Grok
> resource. See `docs/OPERATIONS.md`.

## Install / update / uninstall

```bash
# Install the upstream binary + Go CLI through the shared control plane
vrooli resource install antigravity

# Check binary availability and health
resource-antigravity status

# Update to the current upstream version (idempotent reinstall; never automatic)
bash resources/antigravity/lib/install.sh update

# Remove only the user-owned binary (leaves ~/.gemini durable state intact)
bash resources/antigravity/lib/install.sh uninstall
```

The install lands `~/.local/bin/agy` with **no sudo** and is fully re-runnable.
A root-owned `agy` already on `PATH` is refused with an actionable message (this
resource never installs `agy` with `sudo`); a pre-existing `agy` that shadows
the user install on `PATH` is warned about after install.

**Version model.** Antigravity is not on npm/GitHub releases: its latest
version, artifact URL, and sha512 are served as a per-platform JSON manifest
from the auto-updater service. The install fetches that manifest, verifies the
sha512, and installs the current version. `resource.json
upstream_cli.version_pinned` is the **known-good baseline** used for drift
reporting (`upstream-check`); it is not force-installed because the artifact URL
embeds an opaque build id that cannot be reconstructed from a version string.
For reproducible/air-gapped installs, set `ANTIGRAVITY_ARTIFACT_URL` (and
optionally `ANTIGRAVITY_ARTIFACT_SHA512`). Note that **`agy` self-updates in the
background** during normal runs, so the installed version is a floor, not a
ceiling.

> The resource deliberately does **not** run `agy install` after placing the
> binary: that upstream step mutates the operator's shell profile (PATH append +
> alias purge). `~/.local/bin` is already on PATH, so the handoff is unnecessary.

## Authentication

Antigravity is authenticated by the operator — this resource never stores or
forwards credentials beyond documenting how to provide them.

- **Interactive sign-in:** run `agy` and complete the Google OAuth flow (a
  browser opens locally; over SSH, `agy` prints an authorization URL + one-time
  code). `/logout` clears credentials.
- As of `agy 1.0.13` there is **no `ANTIGRAVITY_API_KEY` headless key path** in
  the binary — sign-in is OAuth-only. (This will be revisited if a headless key
  ships upstream.)
- Credentials are stored **as files** under `~/.gemini` (`oauth_creds.json`,
  `google_accounts.json`), not in the OS keyring — so they are declarable for
  backup (flagged `sensitive`; see Data locations). Antigravity shares this
  `~/.gemini` home with the Antigravity 2.0 desktop app.

## Permissions (command allow/deny)

`resource-antigravity permissions` manages which shell commands (and files/URLs)
Antigravity may act on — the same uniform allow/ask/deny surface the
`claude-code`, `codex`, and `grok` resources expose. Antigravity **enforces these
natively**: it reads the grants from the `permissions` object in its own settings
file.

- Rules are written into the native `permissions` object in
  `~/.gemini/antigravity-cli/settings.json` (global/user scope). The adapter owns
  only the managed `deny`/`ask`/`allow` arrays and preserves every other settings
  key. There is **no user-writable hook backstop** (Antigravity's hooks are
  compiled-in), so this settings object is the single enforcement seam.
- **Schema (confirmed 2026-06-29** against `antigravity.google/docs/cli-permissions`
  and the settings `agy 1.0.13` writes): the `permissions` object holds `allow`,
  `deny`, and `ask` string arrays, evaluated **Deny > Ask > Allow**. Each rule is
  an `action(target)` string — `command(git)`, `command(rm -rf)`,
  `read_file(/var/log/app)`, `read_file(*)`, `mcp(*)` — where `*` is a namespace
  wildcard. The CLI merges project-level, shared-user, and CLI settings.json
  permissions.

```bash
# Block a dangerous command (note: flags come BEFORE the rule).
resource-antigravity permissions deny 'command(rm -rf)'

# Keep common dev commands allowed.
resource-antigravity permissions allow 'command(git)'

# Prompt before any MCP tool call.
resource-antigravity permissions ask 'mcp(*)'

# Inspect / audit.
resource-antigravity permissions list
resource-antigravity permissions doctor        # version + enforcement wiring
resource-antigravity permissions drift-check   # detects hand-edits since the last write
```

Mutating verbs (`deny/allow/ask/remove/reset`) are gated by the shared
`agentpolicy` substrate: a **detected coding-agent caller is refused** so an
agent cannot disarm its own gate. A human re-runs with
`--i-was-explicitly-authorized` (placed before the rule). Rules use Antigravity's
native `action(target)` vocabulary (`command(...)`, `read_file(...)`, `mcp(...)`,
`web_browsing(...)`). End-to-end enforcement (agy actually blocking a denied
command) is proven with a live canary inside an `agy` session.

> **Caller detection of standalone Antigravity is wired** via `ANTIGRAVITY_AGENT=1`
> — the fixed sentinel `agy` injects into the shells it spawns for tool commands
> (the direct analog of grok's `GROK_AGENT=1`; binary-confirmed in `agy 1.0.13`'s
> command-exec path, live `/proc` confirmation pending). `cliutil.DetectCallerKind`
> matches the **exact value `1`** (not mere presence), so a human who exports
> `ANTIGRAVITY_AGENT=<other>` is not misclassified. A detected standalone-Antigravity
> caller is therefore refused mutating `permissions` verbs without
> `--i-was-explicitly-authorized`, alongside the Vrooli-spawned signals and the
> `VROOLI_CALLER=agent` override. See
> `packages/cli-core/docs/reference/agent-detection-signals.md`.

## Data locations (declared for backup)

Antigravity keeps its durable state under `~/.gemini`. The `durable_data` block
in `resource.json` declares the irreplaceable entries so `data-backup-manager`
discovery surfaces them as one-click backup targets:

| Entry | Path | Notes |
|-------|------|-------|
| `settings` | `~/.gemini/settings.json` | Shared Gemini/Antigravity global settings (auth type, preview toggles) |
| `cli_settings` | `~/.gemini/antigravity-cli/settings.json` | CLI config + native `permissions` grants (created by `agy`; absent until the CLI runs) |
| `agent_state` | `~/.gemini/antigravity/` | Conversation transcripts (`conversations/*.pb`), memory (`brain/`), code tracker, MCP config, `user_settings.pb` |
| `oauth` | `~/.gemini/oauth_creds.json` | **Sensitive** Google OAuth tokens — flagged, never auto-accepted |
| `accounts` | `~/.gemini/google_accounts.json` | **Sensitive** active/old Google account identifiers |

- **Single base.** All Antigravity state lives under `~/.gemini` (no
  `~/.config/antigravity` split), so a single home-relative `durable_data` base
  covers it. The regenerable `~/.gemini/antigravity-browser-profile/` (a full
  Chromium cache for the agent's browser tool), `tmp/`, and `state.json` (UI
  banner counts) are deliberately **not** declared.
- **Credentials are file-based** (`oauth_creds.json`, `google_accounts.json`),
  not keyring — so they are declared `sensitive: true`. Restoring stale tokens
  can break auth; the safe restore path is re-running `agy` sign-in.
- Verified live: `data-backup-manager discovery targets` surfaces `antigravity/`
  {settings, agent_state, oauth ⚠, accounts ⚠} with the credential entries
  flagged sensitive.

## Update check

```bash
# Compare the installed agy against the latest upstream release.
resource-antigravity upstream-check
resource-antigravity upstream-check --json
```

Read-only and agent-safe: it reports `up-to-date | behind | ahead | unknown`
against the per-platform release manifest and never hard-fails on a network
error. The same verb exists on the `codex`, `opencode`, and `grok` resources.

## Notes

- This resource wraps an external CLI. It should stay thin by default.
- Do not grow `cli/main.go` into a second operator framework.
- Do not add a `resource-antigravity run` passthrough — when a runner is wired
  later, the contract is raw-binary invocation (mirroring opencode/codex/grok).

## References

- [Antigravity CLI docs](https://antigravity.google/docs)
