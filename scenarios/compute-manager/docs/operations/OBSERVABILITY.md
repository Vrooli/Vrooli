# Observability: Compute Manager

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

> **Status: none of the scenario-specific signals below exist.** Compute
> Manager was generated from the `react-vite` template on 2026-09-03 and
> contains template code only. The only signals that currently emit are
> the template lifecycle ones: API `/health`, the UI health endpoint, and
> the test-genie result. In the tables below, a Status of `intended`
> describes a signal that must be built, not one that is dark.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

There is a distinction here that decides most of the design below. In an
ordinary scenario, observability tells you whether the software is
working. In this one, it also tells you what you are being charged, and
those two questions have opposite failure modes. Software health fails
loudly: a process dies and something notices. Cost fails silently: an
instance nobody is watching keeps billing perfectly well, and every health
signal stays green while it does. So the signals that matter most here are
not the ones that go red when something breaks. They are the ones that go
red when something is *missing* from a picture that otherwise looks fine.

That is also why usage is metered from the state transitions this
scenario caused, and never from an observer loop that watches what is
running. An observer loop is an appealing design and a trap: when the
observer dies, billing stops while the provider keeps charging. A
transition-sourced meter fails the other way, which is the direction you
can recover from.

## Signals

Purpose says what question the signal answers. Threshold says the value
at which someone should act. Status says whether the signal emits today.

| Signal | Type | Source | Status | Purpose | Threshold |
|---|---|---|---|---|---|
| `/health` status | health | API | emitting | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | emitting | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | emitting | scenario correctness evidence | all required phases pass |
| Business suite reachability | health | `meter` domain | intended | Whether credit can be reserved, which decides whether any provisioning may be attempted at all | Any unhealthy result refuses provisioning. This dependency fails closed, so unhealthy is a refusal and not a warning. |
| Bridge reachability | health | `enroll` domain | intended | Whether a new instance can become a trusted node, and whether the onboarding public key is retrievable | Unhealthy is a degradation, not an outage. Alarm on sustained unavailability rather than on a single failure; instances are still created, metered and expiring while enrollment queues. |
| Provider adapter reachability | health | `provider` domain | intended | Whether new capacity can be bought from each configured provider | Per configured provider. Unreachable blocks new capacity and does not affect existing instances, so it alarms on duration rather than on the first failure. |
| Unaccounted instance count | cost | `reconcile` domain | intended | Whether anything is running at a provider that this scenario has no record of, which is the only spend no other signal can see | Target is zero. Any value above zero alarms once the finding survives one full sweep. |
| Unreconciled local record count | cost | `reconcile` domain | intended | Whether the meter is running against local records that have no provider counterpart | Target is zero. Above zero means usage may be accruing against something that no longer exists; close the usage window rather than continuing to meter. |
| Reconciler last-success age | health | `reconcile` domain | intended | Whether the unaccounted count above can be trusted at all, since a stale reconciler is indistinguishable from a clean fleet | Alarm at a small multiple of the sweep interval, and well before a day. |
| Expiry sweeper last-success age | health | `expiry` domain | intended | Whether expiry is still being enforced by the scenario, as opposed to only by the instance-side timer | Alarm at a small multiple of the sweep interval. A backstop firing is itself an incident, so this must alarm before the first-boot timer has to act. |
| Instances past expiry and still running | cost | `expiry` domain | intended | Whether both enforcement paths have failed at once | Target is zero. Alarm when any instance exceeds its expiry by the grace period; a sustained non-zero value means the instance-side timer failed too. |
| Reservation heartbeat failures | cost | `meter` domain | intended | Whether every running instance is still covered by reserved credit | Alarm on repeated failure for the same instance, not on a single failure. One miss is a retry; a run of them means the instance is running outside its reserved credit. |
| Elapsed metered cost per tenant | cost | `meter` domain | intended | What each tenant has spent so far, measured from the transitions this scenario caused | Feeds the per-tenant ceiling (`COMPUTEM-P1-002`). Alarm at a fraction of the ceiling so the refusal at the boundary is not a surprise. |
| Metered-versus-billed divergence | cost | `reconcile` domain | intended (`COMPUTEM-P1-003`) | Whether this scenario's meter still agrees with what the provider actually charges | Compared daily. Provider billing lags by hours to more than a day, so alarm only on a divergence sustained beyond that lag and beyond a set percentage. |
| Un-enrolled instance count and age | product | `enroll` domain | intended | Whether provisioned capacity is actually reaching the fleet as trusted nodes | Age matters more than count. Expected to be non-zero while the bridge onboarding key endpoint is unpublished, so alarm on age past the enrollment retry window. |
| Time from request to running | product | `instance` domain | intended | How long an operator waits between asking for capacity and having it, which is the number they actually feel | No budget set. Establish a baseline from the first real provider before setting one; a target invented before measurement would be fiction. |

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Currently template request logging only. Intended to carry every state transition, every provider call outcome, every reservation event, and every reconciliation finding. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Provider call log | `provider` domain | intended, via `make logs` | One line per provider call with the idempotency key, the outcome and the latency. Never the credential, and never a full request body, because provider request bodies carry the rendered first-boot configuration. |
| Reconciliation findings | `reconcile` domain, SQLite | intended, via `compute-manager reconcile findings` | Findings are durable records rather than log lines, because an operator acts on them hours later. Logs are for the sweep; the table is for the finding. |

Two logging rules that are security-relevant and are stated in full in
[`../internal/SECURITY.md`](../internal/SECURITY.md):

- No provider API credential value appears in any log line, at any level,
  including on error paths.
- The rendered first-boot configuration is never logged whole. It carries
  the bridge onboarding public key, which is public and harmless, but
  logging the document whole is the habit that later leaks whatever else
  is added to it.

## Metrics

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Product activation | deferred | Define once instances can actually be requested. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |
| Provider call latency and error rate | intended | Per provider, per method. A provider that is slow rather than failing is the case that produces lost responses. |
| Lost-response rate | intended | Provider calls that timed out with an unknown outcome. This is the metric that justifies intent-before-action, so it should be visible rather than inferred. |
| Reservation refusal rate, split by cause | intended | Out-of-credit and server error must be counted separately. The upstream reference client discards the response body on a non-2xx, which makes the two indistinguishable; this scenario must not inherit that. |
| Cost per instance-hour, metered | intended | Sourced from transitions this scenario caused. |
| Cost per instance-hour, billed | intended | Sourced from provider billing data, lagging. Reconciliation signal only, never a control. |
| Hour-rounding waste | intended | Billable time minus used time. Hetzner rounds a partial hour up to a full hour, so short-lived instances are a margin problem. This metric is what would justify warm pooling (`COMPUTEM-P2-003`). |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI, and
nothing else.

The intended alert set is small on purpose, because every alert here
competes for the same operator attention that the reconciliation queue
needs:

| Alert | Intended trigger | Why it earns an alert |
|---|---|---|
| Unaccounted instance detected | Any `provider-only` finding survives one full sweep | It is spend that is invisible to every other signal. |
| Reconciler stale | Last successful sweep older than the sweep interval by a wide margin | A silent reconciler makes the fleet look clean. |
| Instance past expiry and still running | Any instance exceeds its expiry by a grace period | Both enforcement paths have failed. |
| Reservation heartbeat failing | Re-reservation fails repeatedly for a running instance | The instance is running outside its reserved credit. |
| Tenant approaching its ceiling | Metered cost crosses a fraction of the ceiling | The ceiling refuses at the boundary; the alert exists so the refusal is not a surprise. |

Provider-side spend alerts are deliberately not part of this set. They
email you and they stop nothing. The real ceiling is the one computed
from our own meter, which is why `COMPUTEM-P1-002` exists.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Every scenario-specific signal above | No cost visibility of any kind exists. The scenario cannot yet tell an operator what is running or what it costs. | Implement alongside the domain that owns each signal. Reconciliation and expiry signals ship with the P0 spine. |
| Provider billing ingestion | Cannot compare metered against billed, so hour-rounding waste and meter drift are both unmeasurable. | `COMPUTEM-P1-003`. Needs a provider billing API per adapter. |
| Out-of-credit distinguishability | An out-of-credit refusal and a server error would read identically, which makes the refusal rate uninterpretable and would trip a circuit breaker on a business outcome. | Fix at this scenario's client boundary; do not wait for the upstream reference client to be corrected. |
| Per-tenant attribution | Costs cannot be split per tenant, so a ceiling cannot be enforced and a hosted tier cannot be priced. | `COMPUTEM-P1-002` and any managed cloud tier. |
| Product usage telemetry | Cannot validate monetization or adoption. | Add before public launch or monetization review. |
| Long-term retention of findings and cost history | Cost questions are asked about last month, not last hour. No retention window is defined. | Define with the storage schema, before the first real provider credential is configured. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) - operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) - readiness gates
- [`../internal/SECURITY.md`](../internal/SECURITY.md) - what must never be logged
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) - business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) - performance measurements
