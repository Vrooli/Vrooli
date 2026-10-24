# Deployment — Persona

Canonical deployment contract: where this scenario can run, what it
needs, how it is packaged, and how a release is rolled back.

## Purpose Of This Document

Use this document to answer:

- Which deployment tiers are supported?
- What must exist at runtime?
- How is it packaged and released?
- How is a bad release reversed?

## Supported Tiers

| Tier | Supported? | Notes |
|---|---|---|
| Tier 1 — local development | **Yes** | The primary and reference target. `make start` through the Vrooli lifecycle. |
| Tier 2 — self-hosted single operator | **Yes — the intended production shape** | The custody argument only holds when the operator owns the host. This is the tier the scenario is designed for. |
| Tier 3 — packaged desktop | Possible, deferred | Nothing structural blocks it; the handoff queue would benefit from native notification. Revisit after P1. |
| Tier 4 — multi-tenant hosted | **No, and deliberately not** | Hosting other people's persona state contradicts the scenario's entire positioning. A hosted deployment would need a different threat model and should be treated as a different product. |

**The Tier 4 exclusion is a product decision, not a technical gap.**
Record any reversal in [`../internal/DECISIONS.md`](../internal/DECISIONS.md)
with its security rationale before implementing it.

## Runtime Requirements

| Requirement | Detail |
|---|---|
| Storage | Embedded SQLite, resolved by `api-core/storage`. No shared resource. |
| Required scenarios | `agent-manager`, `document-manager`, `secrets-manager` — see [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) |
| Optional scenarios | `prompt-manager`, `device-control`, `notification-hub` |
| Network egress | Only what a configured channel adapter needs (mailbox, SMS provider). No outbound call is made by default. |
| Privilege | **None.** No elevated privilege at runtime, ever. Privilege belongs solely to `vrooli setup`. |
| Clock | Reasonably accurate — handoff expiry and code expiry both depend on it. Significant skew is a correctness problem, not a cosmetic one. |

## Packaging

Standard Vrooli scenario packaging: Go API binary, static UI bundle, Go
CLI installed to the user's Vrooli bin. Lifecycle metadata in
`.vrooli/service.json`. Nothing scenario-specific.

**Never packaged**: document bytes, credentials, or journal contents.
Any artifact carrying those is a packaging defect.

## Release Checklist

- [ ] `vrooli scenario test persona` green.
- [ ] `vrooli scenario requirements validate persona --json` passes.
- [ ] `experience-manager spec validate persona --json` passes.
- [ ] `business-health validate scenario persona` passes.
- [ ] Required dependencies reachable and healthy.
- [ ] **Fail-closed verified deliberately**: with `agent-manager`
      stopped, confirm act-as is refused and journaled, and that reads
      still work. This check is not optional; it is the security
      property most likely to regress silently.
- [ ] **No-relay path verified**: with `notification-hub` absent,
      confirm handoffs still queue and complete.
- [ ] Journal export produces a readable artifact.
- [ ] Schema changes reviewed against the append-only and immutability
      constraints in [`../concepts/DATA.md`](../concepts/DATA.md).

## Rollback

Rolling back the binary is ordinary. Rolling back **data** is not, and
the constraints differ by table:

- **The journal is never rolled back.** Rows written by a bad release
  are still true statements about what happened. Correct forward with
  compensating rows.
- **Release records are never rolled back.** A document that was
  disclosed was disclosed; deleting the record does not undo it.
- **Personas, bindings, and ACL entries** may be restored from backup,
  but any journal row referencing a restored-away persona must remain
  resolvable — restore the persona rather than pruning the reference.

If a release is suspected of having widened access, **revoke and
re-grant** rather than rolling back, and read the journal to enumerate
what was permitted in the window.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — day-to-day operation and incidents
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — signals and alerts
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — the properties a release must preserve
