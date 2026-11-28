# Start Here — Scenario Authenticator

Scenario Authenticator is Vrooli's local identity provider. This document is
the current entry point for operators and contributors; the old generated
`react-vite` orientation checklist and fenced `notes` example are retired.

## Getting Started

From the scenario directory, use the lifecycle rather than starting binaries
directly:

```bash
make setup
make start
make status
vrooli scenario test scenario-authenticator
```

The API exposes health and JWKS endpoints, and the auth CLI provides account,
refresh, session, password-change, and local machine-exchange operations.
Passwords are read from a masked TTY or stdin; they are never supplied as
argv values.

## Current Architecture

The API owns account credentials, Argon2id verification, RS256 tokens,
rotating refresh families, MFA primitives, machine exchange, sessions,
JWKS publication, coarse capability grants, and audit events. The default
realm is currently the only realm. Relying parties verify tokens locally and
must treat scope claims explicitly; an absent scope claim never grants
authority.

The UI currently provides the health dashboard, shared shell, settings, and
hosted login work. The full admin console, account self-service, connected
account management, recovery, and true multi-realm screens remain staged
capabilities and are labelled as such in the reference documents.

## Architecture Rules

- Manage lifecycle through `make` or `vrooli scenario`; do not run binaries by
  hand.
- Keep wire contracts in proto and CLI command metadata in `cli/manifest.json`.
- Keep passwords and tokens out of argv and logs.
- Keep the default realm boundary explicit; multi-realm support is not implied.
- Keep person identity, business accounts, commercial entitlements, desktop
  installation identity, and OS credential storage as separate concerns.
- Use `personal_local` as the default Tier 2 bundled mode. Require an explicit
  setup flow before enabling `local_multi_user` or a remote provider.
- Update the requirements registry and the owning document when behavior
  changes.

## Replacing The Example Domain

This scenario has already been detemplated. There is no `notes` domain to
remove. If a future template migration is performed, verify the generated
files against the actual handlers before accepting any documentation claim.

## Documentation Map

- [`README.md`](../README.md) — scenario summary and runtime surfaces.
- [`../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md`](../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md) — project-wide identity and deployment contract.
- [`../../../docs/concepts/AUTHORIZATION.md`](../../../docs/concepts/AUTHORIZATION.md) — capability and domain authorization boundary.
- [`concepts/DOMAINS.md`](concepts/DOMAINS.md) — ownership and deferred domains.
- [`concepts/AUTHORIZATION.md`](concepts/AUTHORIZATION.md) — scope grammar and
  assignment rules.
- [`operations/RUNBOOK.md`](operations/RUNBOOK.md) — operator procedures.
- [`reference/api-endpoints.md`](reference/api-endpoints.md) — current and
  explicitly planned API surface.
- [`reference/cli-commands.md`](reference/cli-commands.md) — current CLI
  commands and deferred parity work.
- [`internal/SECURITY.md`](internal/SECURITY.md) — security invariants and
  open gaps.
