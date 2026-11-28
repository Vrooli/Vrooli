# Grok Build Resource — PRD

## Purpose

Model xAI's Grok Build coding CLI (`grok`) as an optional Vrooli `external-cli`
resource so it is installed reproducibly, health-probed, version-tracked, and its
durable on-disk state (conversations, memory, config, credentials) is discoverable
for backup — feature parity with the other coding agents (claude-code, codex,
opencode) for **install + update + backup**.

## Problem

Vrooli already models coding-agent CLIs as `external-cli` resources. Grok Build
(`grok`; underlying model `grok-build`, large context) had no such resource, so it
could not be installed/updated through the control plane and its state was not
discoverable for backup. This resource closes that gap **without** committing to
runner integration.

## Scope

**In scope**

- `resources/grok/` resource: manifest + `lib/` install scripts + thin
  `resource-grok` Go CLI + docs.
- Sudo-free install/update/uninstall into `~/.local/bin/grok`.
- Registration in `.vrooli/service.json` as `enabled: true, required: false`.
- `durable_data` block (credential entry flagged `sensitive`) so
  `data-backup-manager` discovers Grok's state.
- Validation: resource status/health, backup discovery, build/test/lint.

**Out of scope**

- Registering Grok as an `agent-manager` runner (explicit non-goal; deferred).
- Authenticating on the operator's behalf (operator runs `grok login` or sets
  `GROK_DEPLOYMENT_KEY`).
- Any change to `data-backup-manager` code (discovery is declarative).

## Key decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Driver/template | `external-cli` | Probe-only lifecycle; no managed daemon, no ports. |
| Optionality | `required: false` | A missing Grok subscription must never fail `make setup`. |
| Install strategy | Direct pinned-artifact download (opencode pattern), not the official `curl\|bash` installer | Honors the version pin; no shell-rc mutation; fully re-runnable; clean uninstall. |
| Install prefix | `~/.local/bin/grok` | User-writable; updates never need root. |
| Root-owned copy | Refuse + actionable message | Never clobber; `sudo` only in privileged setup. |
| Version source | xAI release-channel pointer `https://x.ai/cli/<channel>` (GCS mirror fallback) | Grok is not on npm/GitHub releases; custom `upstream-check` fetcher reports drift agent-safely. |
| Backup | Declarative `durable_data`; `auth.json` `sensitive: true` | No `data-backup-manager` code changes; credentials excluded from casual backup. |
| Auth | Browser `grok login` → `~/.grok/auth.json`, or headless `GROK_DEPLOYMENT_KEY` | Resource documents, never stores/forwards the key. |
| CLI surface | Lifecycle + `upstream-check` only | No agent-run wrapper; mirrors opencode/codex. |

## Durable state (declared for backup)

Under `~/.grok`: `sessions/` (transcripts), `memory/`, `config.toml`, `skills/`,
`agents/` (non-regenerable, backed up) and `auth.json` (sensitive credentials,
flagged). Reconstructable state (`logs/`, `docs/`, `active_sessions.*`,
`managed_config.toml`, completions) is not declared.

## Definition of Done

- `vrooli resource status grok` green; sudo-free install + idempotent update.
- `data-backup-manager` discovery lists `grok` with the credential entry flagged
  sensitive.
- `resource-grok` build/test/vet/lint/gofumpt clean.
- Optional resource: `make setup` succeeds whether or not Grok is installed.

## Non-goals / prohibited

- No `agent-manager` runner registration for Grok in this resource (deferred).
- No `sudo` in install/update.
- No edits to `data-backup-manager` code.
- No storing/forwarding the operator's `xai-` API key beyond documenting the env
  var.
