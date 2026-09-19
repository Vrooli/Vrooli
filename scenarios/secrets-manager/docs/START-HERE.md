# Start Here — Secrets Manager

## Initialization Protocol

1. Read `PRD.md`, `docs/concepts/ARCHITECTURE.md`, and `docs/concepts/DOMAINS.md`.
2. Start the scenario with `make start`.
3. Check the lifecycle-owned ports with `make status`.
4. Run `vrooli scenario test secrets-manager` for server-owned validation evidence.

## Architecture Rules

- Keep secret values outside responses, logs, and scenario metadata storage.
- Use the canonical `resource.json` credential descriptor and `vrooli credentials` authority; do not add a plaintext fallback or direct Vault credential path.
- Keep the Go API as the business-logic owner. UI and CLI call its contracts.
- Use lifecycle-managed startup. Do not run API or UI binaries directly.

## Replacing The Example Domain

This scenario has no template example domain. New capability work belongs in an existing product domain—vault coverage, security intelligence, resource intelligence, deployment readiness, or scenario overrides—or must first be added to `docs/concepts/DOMAINS.md`.
