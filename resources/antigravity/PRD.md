# Antigravity CLI Resource — PRD

## Purpose

Model Google's Antigravity coding CLI (`agy`) as an optional Vrooli
`external-cli` resource so it is installed reproducibly, health-probed,
version-tracked, its durable on-disk state (conversations, memory, config,
permission grants) is discoverable for backup, its command allow/deny lists are
governed through the shared `agentpolicy` surface, and Vrooli can detect when a
call originates from it — feature parity with the other coding agents
(claude-code, codex, opencode, grok) for **install + update + backup + shared
permissions + caller detection**.

## Problem

Vrooli already models coding-agent CLIs as `external-cli` resources. Antigravity
— Google's go-forward terminal coding agent that **replaces Gemini CLI for
individual users** (Gemini CLI individual serving ended 2026-06-18) — had no such
resource, so it could not be installed/updated through the control plane, its
state was not discoverable for backup, its permissions were ungoverned, and a
request coming from it was invisible to Vrooli's caller gate. This resource
closes that gap **without** committing to runner integration.

## Scope

**In scope**

- `resources/antigravity/` resource: manifest + `lib/` install scripts + thin
  `resource-antigravity` Go CLI (lifecycle + `upstream-check` + `permissions`) +
  docs.
- Sudo-free install/update/uninstall into `~/.local/bin/agy`.
- Registration in `.vrooli/service.json` as `enabled: true, required: false`.
- `durable_data` block (single `~/.gemini` base) so `data-backup-manager`
  discovers Antigravity's state.
- `resource-antigravity permissions` governance via the shared `agentpolicy`
  decision matrix, enforced through Antigravity's native settings `permissions`.
- A caller-detection signal (or an honest "deferred" outcome if no reliable
  self-set env var is found).
- Validation: resource status/health, backup discovery, permissions round-trip,
  build/test/lint.

**Out of scope (deliberate follow-up plan)**

- Registering Antigravity as an `agent-manager` runner (`RUNNER_TYPE_ANTIGRAVITY`,
  codec, model registry) — the equivalent of the Grok runner work, done later.
- Authenticating on the operator's behalf (operator completes Google OAuth).
- Any change to `data-backup-manager`, `agentpolicy`, or `internal/resources`
  code (consumption is declarative / additive-row).
- The hosted **Gemini API** resource (`resources/gemini`) — different thing.

## Key decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Driver/template | `external-cli` | Probe-only lifecycle; no managed daemon, no ports. |
| Optionality | `required: false` | A missing Antigravity install/subscription must never fail `make setup`. |
| Install strategy | Manifest-driven direct artifact download (opencode/grok pattern), not the official `curl\|bash` installer | No shell-rc mutation; sha512-verified; fully re-runnable; clean uninstall. |
| Install prefix | `~/.local/bin/agy` | User-writable; updates never need root. |
| Root-owned copy | Refuse + actionable message | Never clobber; `sudo` only in privileged setup. |
| Version source | Per-platform release manifest (`<auto-updater>/manifests/<platform>.json`) | Antigravity is not on npm/GitHub; custom `upstream-check` fetcher reports drift agent-safely. `version_pinned` is the known-good baseline (not force-installed — the artifact URL embeds an opaque build id; `agy` self-updates anyway). |
| Backup | Declarative `durable_data`, single `~/.gemini` base | No `data-backup-manager` code changes. |
| Permissions | Native `permissions` object in `~/.gemini/antigravity-cli/settings.json`, gated by shared `agentpolicy` | Antigravity reads grants natively; no user-writable hook backstop exists. Exact JSON schema confirmed post-sign-in. |
| Auth | Google OAuth (`agy` sign-in); no `ANTIGRAVITY_API_KEY` in 1.0.13 | Resource documents, never stores/forwards credentials. |
| CLI surface | Lifecycle + `upstream-check` + `permissions` only | No agent-run wrapper; mirrors opencode/codex/grok. |

## Durable state (declared for backup)

Under `~/.gemini`: `antigravity-cli/settings.json` (config + native permission
grants) and `jetski/` (conversation trajectories + cross-session memory) —
non-regenerable, backed up. The credential location is finalized post-sign-in
(declared `sensitive` if a token-cache file exists; otherwise documented as
keyring/OAuth re-auth on restore).

## Definition of Done

- `vrooli resource status antigravity` green; sudo-free install + idempotent
  update.
- `data-backup-manager` discovery lists `antigravity` with sensitive entries
  flagged (gaps documented).
- `resource-antigravity permissions` governs via shared `agentpolicy`; live block
  proven (or advisory honestly documented).
- Caller-detection signal confirmed + wired, or deferred-row updated honestly.
- `resource-antigravity` build/test/vet/lint/gofumpt + `cli-health validate`
  clean.
- Optional resource: `make setup` succeeds whether or not Antigravity is
  installed.

## Non-goals / prohibited

- No `agent-manager` runner registration for Antigravity in this resource
  (deferred).
- No `sudo` in install/update.
- No edits to `data-backup-manager`, `agentpolicy`, or `internal/resources` code.
- No edits to the hosted **Gemini API** resource (`resources/gemini`).
- No faked permission enforcement; no guessed detection signal.
