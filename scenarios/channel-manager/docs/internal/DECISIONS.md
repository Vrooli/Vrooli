# Decisions — Channel Manager

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

D-000 through D-011 come from the design pass completed 2026-07-28, before any
product code existed. The evidence behind each is recorded with it, because the
evidence is the durable part. If you are about to relitigate one, check whether the
evidence still holds first.

| ID | Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|---|
| D-000 | 2026-07-28 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold generated from the template, matching `content-desk` and `asset-studio`. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Adoption of a different template or doc contract. |
| D-001 | 2026-07-28 | **Replace `social-media-scheduler` outright rather than migrating it; adopt a new name.** | The predecessor's `api/build_errors.log` is dated 2025-09-08 and reports unused imports plus "too many errors" — it had not compiled in roughly eleven months. No consumers referenced it, and its four "platform adapters" were `Post` methods in a 561-line file that never built. This is the opposite of the `landing-page-business-suite` case, where ~29.5k lines of working product justified in-place migration. | Nothing was migrated. The old scenario was moved to `/tmp` on 2026-07-28; git history preserves it. The name changed because the scenario is mostly identity and account health, of which scheduling is roughly a third — `channel-manager` pairs with `docs/marketing/strategy/CHANNELS.md` the way canon pairs with a scenario elsewhere. | Never for the replacement decision. The name is settled; changing it again would churn the proto package path. |
| D-002 | 2026-07-28 | **Warming programs are JSON descriptors carrying their own provenance, and the ones shipped at P0 are marked speculative.** | Warming cadences are operator folklore synthesised from practitioner posts, not platform documentation. Shipping invented constants without marking them invites a future reader to treat them as established practice — the same failure `vrooli-memory` avoids by labelling every summary with its span and depth. | Every program carries `source_kind`, `confidence`, `captured_at`, `sources`, a note, and a `revisit_trigger`, and that block is surfaced wherever the program is displayed (`CHANMGR-P0-018`). An append-only `observations[]` log accumulates real outcomes, which is the path from folklore to measurement (`CHANMGR-P1-006`). | After enough real runs that observations, rather than the source note, are the better evidence. The shipped programs name five runs as their own trigger. |
| D-003 | 2026-07-28 | **Manual execution is a first-class permanent executor, not a stepping stone.** | Three executors are possible — manual, browser automation, platform API. Manual works on every platform for every action on day one, and it is the only one with no terms-of-service exposure. Making it a temporary scaffold would mean rebuilding the action model when automation arrives. | The action record is identical regardless of executor, so moving to automation is a swap rather than a rewrite (`CHANMGR-P0-015`). Manual is never removed, and a browser dispatch failure degrades *to* manual rather than failing the action. | Never — the terms-of-service exposure does not go away. |
| D-004 | 2026-07-28 | **Signals report measurements and flags, never verdicts, and never react automatically.** | Platform penalties are unobservable; only reach, impressions, engagement, follower delta, and audience geography are real. Every plausible automatic response to suspected decay — posting more, posting less, changing niche — either worsens the situation or destroys the evidence needed to understand it. | A flag states "reach below 40% of 14-day baseline for 3 consecutive posts" and carries the observations that raised it. Raising a flag pauses the identity's queue and does nothing else (`CHANMGR-P0-017`). No message in the scenario asserts that an account has been penalized. | Never for the no-verdict rule. The margin and run-length thresholds are unvalidated and expected to change. |
| D-005 | 2026-07-28 | **One action queue per identity, spanning warming, maintenance, and publishing.** | Platforms count actions per account, not per workflow. Separate queues for warming and publishing would let an identity exceed its daily budget by doing both, and the breach would be invisible to each queue individually. | The unified queue is the scenario's spine (`CHANMGR-P0-006`). Cadence ceilings come from the platform descriptor rather than the warming program, so a program asking for more than the ceiling allows is clamped at plan generation rather than failing at the platform (`CHANMGR-P0-007`). | Never expected — this is a correctness property, not a preference. |
| D-006 | 2026-07-28 | **Record and check the identity's environment; do not provision it.** | Device fingerprint, residential proxy, and region consistency are reported as the difference between an account that reaches its audience and one throttled permanently. But provisioning them means third-party services with their own cost, contracts, and terms — a different product from account operations. | Preconditions are manual attestations that block program start until satisfied (`CHANMGR-P0-005`). The scenario cannot verify them, and an unverifiable gate is still strictly better than no gate. A leak the scenario cannot detect remains a real and undetectable failure mode, recorded in the PRD's risk list. | If an environment provider exposes a programmatic check, or if environment drift becomes a recurring quarantine cause. |
| D-007 | 2026-07-28 | **Warming can fail terminally. Quarantine cancels everything for that identity.** | The trust-check in the source material has a "under a few hundred views → start fresh" branch, meaning the correct response to a failed warm is to abandon the identity, not to continue. An earlier draft of this design had only graduated and not-yet-graduated, which would have let an identity silently continue a program it had already failed. | Quarantine is a terminal state that cancels every pending queue entry and records the failing measurement (`CHANMGR-P0-011`). A quarantined identity is rebuilt, not resumed — `quarantined → running` is an illegal transition. | If measurement shows accounts recovering from a failed trust-check, which would make quarantine too harsh. |
| D-008 | 2026-07-28 | **The boundary with `content-desk` is exactly two questions, and eligibility fails closed.** | `content-desk` specified this seam from its own side before this scenario existed: *release this draft* and *is this identity eligible for this lane*, with "no account state, warming detail, or credential" crossing, and eligibility resolving to unknown rather than eligible on failure. | This scenario answers rather than negotiates. Eligibility returns a three-valued result and never defaults to eligible (`CHANMGR-P0-013`); release is idempotent by key with a unique index that *is* the retry guarantee (`CHANMGR-P0-014`). A wider surface would let editorial logic depend on account internals. | Never for the fail-closed rule. Widening the surface requires a decision on both sides. |
| D-009 | 2026-07-28 | **Browser automation is an operator decision recorded per platform and per action kind, not a default.** | Automating account actions through a browser is against the terms of service of most major platforms, and enforcement is typically account suspension. This is a normal marketing-operations tradeoff to accept deliberately, but discovering it after losing a warmed persona account is expensive and avoidable. | The browser executor is sequenced last among P1 and is disabled for a platform until a decision recording that acceptance exists in this file (`CHANMGR-P1-001`). Everything above it must work manually first. Comment-text generation is deliberately excluded from the descriptor: generating "genuine, specific comments" at scale produces exactly the low-effort pattern warming exists to avoid. | Per platform, whenever the operator decides to accept the exposure. Each acceptance is its own row below. |
| D-010 | 2026-07-28 | **Three capabilities offered by comparable tools are refused permanently, and the refusal is recorded rather than left to judgement.** | A competitive scan across mainstream schedulers (Buffer, Hootsuite, Later, Metricool, Publer, Sprout Social, Postiz, Mixpost) and the multi-account operator category produced four features worth adopting — `CHANMGR-P1-008`, `-009`, `-010`, and `CHANMGR-P2-005`. It also produced three that sit on the wrong side of the line `../business/MONETIZATION.md` draws. Someone reading a competitor feature list in a year will encounter them again, and the reason they were refused is not self-evident from the feature name. | See the table below. Each is refused on its own grounds, not by association. Adding any of them requires reversing this decision explicitly. | **None expected.** These are positioning and integrity boundaries, not capability gaps. |
| D-011 | 2026-07-28 | **When the cadence ceiling is reached, a release outranks maintenance engagement; maintenance defers rather than drops, and a deferred action that misses its phase window is recorded rather than silently skipped.** | `DOMAINS.md` states that maintenance engagement "competes for the same cadence budget as posting" and nothing said who wins. Left unspecified, the arbitration becomes an arbitrary line inside the scheduler that nobody documented and no test covers — and the failure is silent, because both candidate behaviours look like a working queue. The ranking follows from what each action is for: a release carries approved, time-bounded work from `content-desk`, while maintenance is a standing background obligation with no deadline of its own. | Scheduling is a strict ordering rather than first-come. A deferred maintenance action stays queued with its deferral recorded; if its phase window closes before it runs, the miss is written to the program's observation log (`CHANMGR-P1-006`) rather than dropped, so a program whose maintenance load exceeds its ceiling is visible as evidence instead of quietly under-executing. Constrains `CHANMGR-P0-007`. | If observations show maintenance being starved often enough to affect distribution, which would mean the ceiling is set below what the program actually needs — a program or ceiling problem, not an arbitration one. |

### D-009 — per-platform automation acceptances

Browser execution stays disabled for a platform until it has a row here.

| Platform | Action kinds accepted | Date | Accepted by | Note |
|---|---|---|---|---|
| _(none yet)_ | — | — | — | No platform has an accepted automation decision. All execution is manual. |

### D-010 — refused capabilities

| Capability | Offered by | Why refused |
|---|---|---|
| **Content spinning for uniqueness** | Multi-account operator tools, to defeat duplicate-media hashing | It is deception by construction: the point is making identical material appear distinct to a platform that is deliberately trying to detect it. `CHANMGR-P1-003` solves the legitimate half of the same problem — do not publish the same asset from two identities — by *not reusing the asset*, which is the honest answer to the same constraint. |
| **Bulk account creation** | Multi-account operator tools and account marketplaces | The spam pattern itself, and it inverts the scenario's model. An identity here is a durable thing with an environment, a warming history, and a permanent action record. Creating them in bulk means creating them disposably, which is the opposite posture. |
| **Engagement pods / reciprocal engagement** | Growth tools and operator communities | The behaviour platforms enforce against most directly, and it manufactures the exact signal `signals` exists to measure honestly. It would also corrupt the baseline: reciprocal engagement inflates the numbers a decay flag is computed against, so the monitoring would stop working at the same time as the risk went up. |

Two further notes from the same scan, recorded so they are not rediscovered as gaps:

- **Competitor and benchmark tracking** (Metricool's strength) is a research
  capability, not an account-operations one. It belongs to `content-desk`'s
  researcher surface, and building it here would put audience research inside the
  scenario that holds credentials.
- **UTM and link tagging** is a genuine gap across the whole marketing trio, not
  this scenario alone. Tags should be authored at draft time in `content-desk` and
  preserved through release here; `CHANMGR-P1-007` returns platform-reported
  performance, which stops at reach and never reaches conversion without them.

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
