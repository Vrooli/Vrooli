# Trust Model

## Purpose Of This Document

This is the **single canonical source** for the **Trust axis**: the model that answers *"is this reading evidence, or is it instrument fault?"*

It is the sibling of [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md), which owns what a target *is* and where the setpoint lives. Read that file first; the target model, deadband semantics, and honesty flags are defined there and are **not** restated here.

This document describes the **ideal** instrumentation, not what exists today. That is deliberate and matches how `meta-optimization-manager`'s [`CONDITION-MODEL.md`](../../../meta-optimization-manager/docs/concepts/CONDITION-MODEL.md) is authored: the model is written at full maturity so the distance between it and reality is *measurable* rather than invisible. Current adoption is recorded in [§ Current State](#current-state) as data, never by trimming the model down to what already works.

## Why Trust Is A Distinct Axis

`meta-optimization-manager` needs no equivalent of this document, and the reason is precise. Its numerators are joins over registries — a provider is declared or it is not, a binding resolves or it does not. Registries do not lie in interesting ways.

This scenario's numerators are **alarms on running processes**, and alarms lie in four named ways. The evidence is not hypothetical:

> The 2026-07-23 alarm-flood decomposition measured **4,624 critical events in 24 hours, of which roughly 92% came from ghost and saturated checks.**

A board that reported that reading at face value would have said *the platform is critically unhealthy*. The platform was fine. The instrument was broken. Reporting instrument fault as plant fault is the single most expensive mistake this scenario could make, because it routes real engineering effort at an imaginary problem while the alarm channel that would have shown the real one stays saturated.

The governing rule is already canon in the team's operating model:

> **Discriminate sensor fault from plant fault before routing plant-side work; a degraded alarm channel is the alarm-flood target's finding, never a reason to silently distrust all sensors.**
> — `docs/infra-health/operating/OPERATING_MODEL.md` rule 5

Both halves of that sentence are load-bearing. The first says an untrusted reading must not become plant work. The second says an untrusted reading must not become *nothing* — it is a finding about the instrument, routed to the instrument's owner.

### Trust is not condition

The obvious objection is that this duplicates the Condition axis. It does not, and the boundary is precise:

> **Condition asks whether the thing being measured still works. Trust asks whether the measurement itself can be believed.**

A leg can be genuinely degraded and honestly measured (condition problem, plant-side). A leg can be perfectly healthy and reported through a saturated check (trust problem, instrument-side). Folding them means a plant fix gets scheduled for an instrument defect, and the fix cannot possibly move the sensor — which the `focus` domain's efficacy join would then correctly report as an unmoved finding, one cycle too late.

## The Trust Vocabulary

**The vocabulary is closed**, and `UNTRUSTED` is its conservative default. Two distinct situations resolve to it: a token this scenario does not recognise, and a reading that no integrity rule could evaluate at all. Both claim nothing, and treating them as one verdict is deliberate — from the board's side they are the same fact, which is that the reading cannot be believed and the reason is the instrument's, not the plant's.

| Verdict | Meaning | Source of the rule |
|---|---|---|
| `VALID` | The reading passes every integrity rule. It is evidence. | — |
| `GHOST` | The check's target no longer exists as a plant element. Its events never count as downtime and are excluded from every aggregate. | ISA-18.2 stale alarm |
| `SATURATED` | The check has been pinned at a single status for a full 24h window. The *transition* is the signal; the repeat event is not. | EEMUA 191 chattering / flood |
| `SHELVED` | The plant element is deliberately stopped (paused team, decommissioning, maintenance) and its check is suppressed with a named reason and a mandatory expiry. | ISA-18.2 shelving |
| `UNIT_MISMATCH` | The reading is real but is being used for a claim its unit cannot support — the event-weighted aggregate cited for a per-element claim. | The team's unit rule |
| `UNAVAILABLE` | The source could not be read. The reason is stated verbatim. | The degradation contract |
| `UNTRUSTED` | No integrity rule could evaluate this reading — the rule is unshipped, the evidence is absent, or the verdict token is unrecognised. **Not a synonym for valid, and not a synonym for broken.** It is declared blindness about the reading itself. | The honesty invariant below |

`VALID` is the only verdict that contributes to an aggregate. Every other verdict is reported, counted, and excluded.

The distinction between `UNAVAILABLE` and `UNTRUSTED` is worth holding: `UNAVAILABLE` means the *source* could not be reached, so there is no reading; `UNTRUSTED` means a reading exists but its trustworthiness could not be established. The first is a gap in the fan-out, the second a gap in the integrity rules — and they route to different fixes.

### The unit rule deserves its own verdict

It is tempting to treat `UNIT_MISMATCH` as a caller error rather than a reading property. It is a reading property here for one reason: the event-weighted `actions uptime` aggregate is a *legitimate* sensor — it is the alarm-flood sensor — and it is simultaneously an *illegitimate* source for any per-element uptime claim. The same reading is trustworthy for one target and untrustworthy for another. Encoding that on the reading, scoped to the target it is serving, is the only shape that catches the misuse at the point it happens.

## The Honesty Invariant

This is the rule the whole axis exists to enforce, and it is the same discipline `CONDITION-MODEL.md` applies one layer up:

> **A reading with no trust verdict is never reported as `VALID`, never counted in a healthy total, and never contributes to a band evaluation as a passing member.**

Without this rule the axis inverts on contact with reality: a fleet with no integrity checking at all would report perfect trust, and the incentive to build the checking would run exactly backwards.

### Every trust number is reported as a triple

Never a bare count:

```
<verdict distribution> of <checked readings>, of <total readings>
```

So a reader sees `3 GHOST, 1 SATURATED, 14 VALID of 18 checked, of 23 readings` — and can never mistake the five remaining readings for five valid ones. Those five carry `UNTRUSTED`; the verdict and the "checked" denominator are two views of one fact, never two independent tallies. This mirrors the recursive honesty of denominator-confidence in `COVERAGE-MODEL.md`: the board reports its own blindness alongside its findings.

**`UNTRUSTED` readings are themselves a rankable finding.** A reading backing an in-band claim that no integrity rule could evaluate is a weaker claim than a checked one, and it appears on `focus next` ranked by the target's weight.

## Where Trust Verdicts Come From

**Trust is computed here; integrity primitives are owned upstream.** The distinction matters for the same reason the setpoint is owned elsewhere.

| Rule | Who can answer it | Status |
|---|---|---|
| Ghost — does the check's target still exist? | `vrooli-autoheal check reconcile` (roadmap Gap 10) | Not shipped. Interim: derived-set membership from `scenario-dependency-analyzer` |
| Saturated — did the check transition in the window? | `vrooli-autoheal actions transitions` | Ships today |
| Shelved — is the element deliberately stopped? | `vrooli-autoheal check shelved` (roadmap Gap 10) | Not shipped. Interim: the team's runtime lessons artifact, read as `estimate` |
| Unit mismatch — is this reading's unit valid for this target? | This scenario, from the target model | Computable today |

Two of the four depend on **Gap 10**, which is already top of the team's own sensor queue and whose priority signal has fired twice. Until it ships, this scenario computes what it can, marks the rest `UNTRUSTED` rather than assuming `VALID`, and reports the resulting instrumentation shortfall as a finding against itself.

**This scenario never mutates a check.** It does not shelve, unshelve, retire, or reconcile — those are autoheal's verbs and they change a running system. It reads their output and forms a verdict. A board that could shelve its own inconvenient alarms would be the purest possible form of an observer confirming itself.

## How Trust Interacts With Banding

Trust and banding must not be folded together, and the reason is symmetrical to why coverage and condition stay separate: folding them destroys both numbers.

1. **An untrusted reading does not produce an in-band verdict at all.** It is neither in band nor out of band — it is not evidence. Reporting it as in-band would manufacture false confidence; reporting it as out-of-band would manufacture false alarm.
2. **An untrusted reading is still a finding** — routed to the instrument, not the plant. Per operating-model rule 5, a degraded alarm channel is the alarm-flood target's finding.
3. **Trust never suppresses a target.** A target whose every reading is untrusted is reported as *unmeasurable*, with its verdict distribution attached. It never silently disappears from the board, because a target that vanishes when its sensor breaks is the exact failure this axis exists to catch.

### Cascade discipline

The team's existing cascade rule applies to trust before anything else, and this scenario's ranking must honour it:

> Layer order (inner → outer): **sensor-channel integrity**, host/process substrate, capability availability, efficiency and performance trends, measurement improvement.

Sensor-channel integrity is the innermost layer. A performance finding raised while the alarm channel is saturated is premature by construction. The `focus` domain therefore ranks trust findings above plant findings whenever both are present, and states that it is doing so.

## Load-Bearing Constants

Following the discipline in `RELIABILITY_TARGETS.md` — judgment constants are named, documented, and auditable rather than buried in code:

- **`saturationWindow = 24h`** — inherited directly from the team's sensor-integrity rules. A check pinned at one status for a full window carries no information.
- **`shelfExpiryRequired = true`** — permanent suppression is prohibited by the team's shelving rule. A shelf with no expiry is itself a finding, and an expired shelf reverts to live alarming.
- **`readDeadline = 3s` per source** — a slower source is an honest `UNAVAILABLE`, not a hang. Mirrors `meta-optimization-manager`'s `numeratorDeadline`.
- **`trustCheckCoverageFloor`** — deliberately unset. Setting it to whatever the current coverage happens to be would report in-band while the defect stands, which is precisely the dead-deadband failure `FRAMEWORK_HEALTH.md` § "Deadband rule" names.

## Current State

Recorded as of 2026-08-17, as data. This section is the measured distance from the model above and is expected to change; the model is not expected to change with it.

| Fact | Value |
|---|---|
| Integrity rules computable today | 2 of 4 (saturated, unit mismatch) |
| Integrity rules blocked on Gap 10 | 2 of 4 (ghost, shelved) |
| Alarm-channel state at last reading | 1,058 critical events / 24h against a ≤500 deadband — out of band |
| Prior decomposition | 4,624 / 24h on 2026-07-23; ~92% attributed to ghost and saturated checks |
| Shelving record today | Hand-maintained rows in the team's runtime lessons artifact |
| Team loop status | `paused-manual` since 2026-07-24 — no reading is a live baseline until it resumes |

## Governing Principles

- **Untrusted is never valid.** The axis reports its own blindness alongside its findings, or it is worse than having no axis.
- **Sensor fault is not plant fault.** An untrusted reading routes to the instrument's owner, never to plant-side work.
- **The vocabulary is closed.** An unrecognized verdict coerces to `UNTRUSTED`, which claims nothing.
- **Read integrity, never mutate it.** Shelving and reconciliation are autoheal's verbs. A board that could silence its own alarms is not an observer.
- **A broken sensor never hides its target.** Unmeasurable is a reported state, not an absence.

## Cross-References

- [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md) — the target model, deadbands, and honesty flags this document builds on.
- [`DOMAINS.md`](DOMAINS.md), [`ARCHITECTURE.md`](ARCHITECTURE.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).
- `docs/infra-health/strategy/RELIABILITY_TARGETS.md` § Sensor integrity — the ISA-18.2 / EEMUA 191 rules this axis mechanizes.
- `docs/infra-health/operating/OPERATING_MODEL.md` rule 5 — the sensor-integrity routing rule.
- `docs/infra-health/evidence/INSTRUMENTATION_ROADMAP.md` Gap 10 — check shelving and registry reconciliation, which unblocks two of the four rules.
- `scenarios/meta-optimization-manager/docs/concepts/CONDITION-MODEL.md` — the sibling instrument's condition axis; the precedent for derived populations and the uninstrumented-is-not-healthy rule.
