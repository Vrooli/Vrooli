# Go To Market — Tunnel Manager

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

> **This is an internal-adoption GTM.** Tunnel Manager ships as **core
> Vrooli infrastructure** ([`MONETIZATION.md`](MONETIZATION.md):
> foundational interface enabler). External / commercial GTM is
> **deferred** — there is no external buyer for the tunnel control plane
> itself.

## Audience And Positioning

- Audience (internal "customers"):
  - **Operators** — replace the manual Cloudflare dashboard workflow.
  - **Other scenarios** — request their own reachability via the
    exposure-request API ("expose me, I'll be used soon").
  - **Infrastructure agents** — consume a reliable "is remote access
    working?" signal to drive recovery decisions.
- Positioning: the external-access control plane — programmatic, tiered,
  self-healing exposure that makes published scenarios stay reachable
  without manual tunnel babysitting.
- Main claim: "Exposure becomes a one-command, policy-driven capability;
  the tunnel self-heals."
- Proof needed: core-tier scenarios always reachable; leasing works
  end-to-end; auto-recovery restores connectivity without operator
  intervention.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Ships as core Vrooli infra | Distributed with the local stack; no marketing channel needed. | Working CLI/API/UI, lifecycle integration. | Installed and running in the Tier 1 local stack. |
| Operator runbook + CLI | Operators adopt `tunnel-manager expose/lease/...` instead of the Cloudflare dashboard. | RUNBOOK, proto-typed `--json` CLI. | Manual dashboard edits drop to zero. |
| Exposure-request API for scenarios | Other scenarios call the broker to become reachable. | Stable Connect-RPC contracts under `packages/proto/schemas/tunnel-manager`. | Scenarios request exposure programmatically. |
| app-monitor "open in new tab" integration | app-monitor uses `IsExposed`/`ExposeAndGetURL` to open scenarios remotely. | Exposure-query API (OT-P1-007); app-monitor-side change is a separate task. | New-tab flow returns a working tunnel URL. |
| External / commercial | deferred | n/a | Only if a SaaS/enterprise remote-access tier is pursued. |

## Launch Motion

Mirrors the PRD launch sequencing (internal rollout):

1. Ship CLI + API first, then the 5-surface dashboard.
2. Seed **core-tier** exposure from `api-core/coreset` (always-on).
3. Enable **leasing** (request/extend/revoke/reap) for operators and
   other scenarios.
4. Confirm the real Cloudflare hostname cap against the live plan.
5. Flip `vrooli-autoheal`'s cloudflared check to **alert-only** (single
   restart owner — see RUNBOOK / DECISIONS).
6. Integrate app-monitor's "open in new tab" (separate task).

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Expose a scenario with one command — no Cloudflare dashboard." | Operators | RUNBOOK CLI procedures | implemented |
| "Request your own reachability via the API." | Other scenarios | Exposure-request API (OT-P0-005) | implemented |
| "The tunnel self-heals; core scenarios stay reachable." | Operators, infra agents | Auto-recovery + core-tier guarantee | implemented; live Cloudflare validation remains operator-attended |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Replace manual dashboard exposure with `tunnel-manager expose` | Operator CLI | Operators stop hand-editing Cloudflare hostnames | Adopt as default exposure path. |
| Core-tier reconciliation | Internal infra | All `coreset` scenarios always exposed | Confirms always-on guarantee. |
| app-monitor new-tab via exposure-query | app-monitor integration | New-tab opens a working tunnel URL | Greenlight the app-monitor-side change. |
| External / commercial validation | deferred | n/a | Only if a remote-access tier is pursued. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
