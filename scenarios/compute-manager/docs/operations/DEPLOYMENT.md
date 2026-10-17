# Deployment: Compute Manager

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

> **Status: nothing is deployable.** Compute Manager was generated from
> the `react-vite` template on 2026-09-03 and contains template code
> only. No tier below is supported today, including the local stack: what
> starts is the template worked example, not the product. Everything here
> is the intended shape and the conditions each tier must meet.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

One property makes this scenario unusual to deploy, and it is worth
stating before the tables. Compute Manager runs no local service and
requires no resource. Its state is one SQLite database and its reach is
outbound HTTPS to a cloud provider API. That makes it trivially portable
in the mechanical sense, and it means the real deployment gates are not
technical: they are the credential path, the metering path, and the
provider's reselling terms. A tier is not blocked here by a missing
runtime. It is blocked by whether money can be reserved and settled
correctly on that tier.

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Tier 1 local stack | not supported yet | Vrooli lifecycle, Go, Node/pnpm, SQLite path | Nothing is implemented. The template `notes` worked example has been removed; the P0 spine must land before claiming this tier. |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Not a candidate. The scenario holds live billing state and runs two unattended loops; an application the operator closes is the wrong host for both. |
| Managed cloud/SaaS | deferred | Hosted runtime, tenant isolation, credential authority reachable from the host, observability | Requires `COMPUTEM-P1-002` (per-tenant ceiling) and a reviewed threat model. Selling provisioned compute also requires that the provider's terms permit it, checked per service and not per provider. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | A self-hoster supplies their own provider credentials and is not metered at all, so this tier needs the free path proven before the paid one. |

Two tier notes that will not change with implementation progress:

- **Connecting a machine the operator already owns stays free forever**
  (`COMPUTEM-P1-001`). Only capacity Vrooli provisions and pays for is
  metered. The monetization contract forbids gating what a self-hoster
  could run with their own keys, so no tier may make adoption of an
  existing host a paid path.
- **Hosted compute is a cost-bearing (Class A) capability.** Enforcement
  is server-side, before the machine boots. No tier may move that check
  to the client, because a client-side check on a cost-bearing path is
  not a check.

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT` (band `15000-19999`).
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: embedded SQLite, resolved from the scenario id by
  `api-core/storage`. One scenario-owned database holds instance intents,
  instances, provider receipts and reconciliation findings.
- Resources: **none**. This scenario runs no local service. There is no
  Postgres, Redis, Qdrant or Vault dependency to provision.
- Network: local API/UI communication, plus outbound HTTPS to one cloud
  provider API per configured adapter, plus reachability to
  `landing-page-business-suite` and `vrooli-bridge`.
- Clock: the expiry sweeper and the metering heartbeat both depend on
  host time being correct. A host with a badly skewed clock will either
  destroy instances early or fail to re-reserve credit in time.
- Credentials: provider API credentials resolve through the Vrooli
  credential authority at call time. Environment variables and command
  arguments are not a credential source. See
  [`../internal/SECURITY.md`](../internal/SECURITY.md).

Dependency behaviour at runtime, which a deployment plan must respect:

| Dependency | Required | Behaviour when unavailable |
|---|---|---|
| `landing-page-business-suite` | yes | **Fail closed.** Provisioning refuses. This is the one dependency that refuses rather than degrades, because a machine that boots unmetered is cost that grows hourly and cannot be recovered afterwards. |
| `vrooli-bridge` | yes | **Degrade.** The instance is still created, metered and expiring. Enrollment queues and retries, and an un-enrolled instance is visible and flagged. |
| `treasury` | conditional | Only the agent-initiated purchase path. Absent, agent-initiated provisioning refuses and operator-initiated provisioning continues. |
| `offer-desk` | catalog only | Holds the sellable definition. No runtime path, so its absence blocks no deployment. |

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. Hosts the Connect-RPC services and both unattended loops (the bidirectional reconciler and the expiry sweeper). |
| UI | Vite production bundle served by `ui/server.js`. Inventory-first operator dashboard; not a marketed surface. |
| CLI | Go CLI installed through scenario manifest install hooks, command name `compute-manager`. Full headless parity is a requirement, not a convenience, because the unattended paths must be drivable with no browser. |
| Proto | Schemas live under `packages/proto/schemas/compute-manager/`; generated clients are shared artifacts. |
| Provider adapters | Compiled into the API binary. Each adapter declares its provider's billing facts as data: rounding behaviour, minimum billable unit, whether a stopped instance bills, and whether inbound traffic counts against the transfer allowance. |

The provider interface is deliberately four methods: `Create`,
`Describe`, `List`, `Destroy`. There is no `Stop`. That is a product
decision rather than an omission: a stopped instance still bills at the
full rate on five of the seven providers surveyed, so a pause button that
maps to power-off costs full price for zero value. `OT-P0-007` makes this
a must-ship target and a structural test asserts no such method exists.
Do not add one to make a packaging problem easier.

## Release Checklist

None of these is satisfied today.

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [x] Template `notes` worked example has been removed by `template-manager
      detemplate compute-manager`. It has not been replaced by a real domain.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.
- [ ] The P0 spine passes against a fake provider: reserve-provision-settle,
      intent-before-action, bidirectional reconciliation, and
      double-enforced expiry. None of these needs a real API key, and all
      four are the failure modes.
- [ ] A structural test asserts the provider interface declares no `Stop`
      method (`OT-P0-007`).
- [ ] Provider API credentials resolve through the credential authority
      in a live run, and appear in no log line, request field, argv entry
      or database column.
- [ ] The reservation window is long enough for an hour of compute, or
      heartbeat re-reservation is proven. The upstream default is ten
      minutes, which is shorter than the smallest unit this scenario
      sells.
- [ ] An out-of-credit result is distinguishable from a server error at
      the API boundary and in the UI.
- [ ] The three manual procedures in [`RUNBOOK.md`](RUNBOOK.md) have been
      performed at least once against a real provider account.
- [ ] The chosen provider's reselling terms have been read per service,
      not per provider, and the finding is recorded in
      [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
- [ ] A threat model review has been completed. See
      [`../internal/SECURITY.md`](../internal/SECURITY.md).

## Rollback

Local development rollback is source-control based.

Rollback in this scenario carries a hazard that most scenarios do not
have, and it must be planned for before the first real provider
credential is configured. Rolling the binary back is safe. Rolling the
**database** back is not: instances created after the backup point become
invisible to the scenario while continuing to bill, and settled usage may
be settled a second time.

The intended rollback procedure, once there is anything to roll back:

1. Stop the scenario with `make stop`.
2. Deploy the previous binary.
3. Leave the database at its current state. Do not restore an older
   database copy unless the current one is unreadable.
4. Start the scenario with `make start`.
5. Run `compute-manager reconcile run`.
6. Review every finding before resuming provisioning.

If an older database copy must be restored, treat every instance at the
provider as unaccounted and work through
[Quarantine An Unaccounted Instance](RUNBOOK.md#quarantine-an-unaccounted-instance)
for each one. That is slow on purpose. The alternative is an automated
cleanup that can destroy a paying customer's node.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) - operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) - health and telemetry
- [`../internal/SECURITY.md`](../internal/SECURITY.md) - credentials and open gaps
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) - dependencies
- [`../reference/configuration.md`](../reference/configuration.md) - env vars and lifecycle config
- [`../../PRD.md`](../../PRD.md) - operational targets and launch sequencing
