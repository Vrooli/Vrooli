# Progress — Vrooli Bridge

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-06-18 | Claude (agi) | done | **Phase 0 — Foundations.** Ran `vrooli scenario detemplate vrooli-bridge` (removed the `notes` example domain from api/cli/proto/ui; `example-domain-removed` orientation gate passes). Resolved the two HIGH security decisions in [`DECISIONS.md`](DECISIONS.md) + [`SECURITY.md`](SECURITY.md): (1) mutual auth = per-node Ed25519 keypair pinned both directions at pairing (rejected full PKI/mTLS-CA); (2) runner sandboxing = dedicated non-privileged service user + a structurally separate privileged provisioning helper (different OS principals, not a flag). Authored the versioned dial-out wire contract `packages/proto/schemas/vrooli-bridge/v1/channel/channel.proto` (Handshake + CompatibilityStatus negotiation, Heartbeat + HealthSnapshot, JobPush/ProvisionCommand/ControlPing server frames, RunEvent node frames; DiscardUnknown back-compat policy, CHANNEL_PROTOCOL_VERSION=1). Established the cross-compiled **node-agent** module `agent/` (separate Go module; config/buildinfo/platform/channel packages; stub dial using the channel proto types) — cross-compiles `CGO_ENABLED=0` for linux/darwin/windows × amd64/arm64 via `agent/Makefile` `make matrix`, with build-fingerprint ldflags. api + cli + agent all build/vet/gofumpt/golangci-lint/test green. |
| 2026-06-18 | Claude (agi) | done | Greenfield regeneration from `react-vite` (the prior doc-injection bridge was removed — see [`DECISIONS.md`](DECISIONS.md) superseded log). Authored documentation-first foundation: `PRD.md` (8 P0 / 6 P1 / 4 P2 OTs, validates healthy), `requirements/` (18 modules, one per OT, validates healthy), and bridge-specific concept docs (ARCHITECTURE, DOMAINS, DATA, FLOWS, INTEGRATIONS), internal DECISIONS + SECURITY, and business/operations/performance docs. Captured keystone decisions: dial-out connection direction, two trust tiers, allowlisted typed-verb execution, node = versioned build/test env, compose-don't-reinvent. docs-health 94% / L5. Orientation 5/8 — remaining 3 (scaffold-health `make test`, dependency-decisions service.json resources, example-domain-removed) are implementation-phase. No product code yet. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
