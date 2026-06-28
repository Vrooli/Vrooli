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

> **Not yet wired as an agent-manager runner.** This resource deliberately stops
> at install + update + backup parity with the other coding agents. Registering
> Grok as an `agent-manager` runner is a separate, later phase — see
> `docs/OPERATIONS.md`.

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

## References

- [Grok Build CLI](https://x.ai/cli)
