# Scenario QA Readiness Checks — Plan of Record

This folder is the **strategic canon** for readiness-dimension checks. Each active check has one method document; the single `readiness-signal` skill holds the return contract they all share.

> **Pre-emptive readiness ordering lives in swarm-manager now.** A former QA member ran these reviews on idle scenarios and filed fix items ahead of feature work. That ordering is now a deterministic swarm-manager gate (`fix_before_feature`) plus optional policy-governed maintenance intake through the backlog auto-filer (`auto_filer`). This registry remains the home for *individual readiness-check techniques* used by deployment readiness; it no longer belongs to a specific QA member.

These docs answer: *what is this readiness check, when does it apply, when does it backfire, what's the failure mode the qa-contrarian watches for?*

They do **not** answer: *step-by-step procedure for applying it.* That belongs in the evidence producer and the `readiness-signal` skill.

## Status: v1

This registry contains the reasoning canon used by Deployment Manager's
deployment-time aggregator. Deployment Manager owns the declarative checklist
and JSON Schema in `scenarios/deployment-manager/api/readiness/`; its Go loader
embeds and validates that policy. Evidence producers retain ownership of their
measurements, and the `readiness-signal` skill owns the return contract. Adding a checklist
item that fits the existing schema does not itself add a Test Genie phase; a new
producer route still needs owner wiring and validation.

The v1 entries are the canonical reasoning layer for the deployment-manager checklist; the return contract lives once in the `readiness-signal` skill.

## Why this is intentional asymmetry

The scenario-qa team's three technique registries are at deliberately different maturity levels:

| Registry | Maturity | Reason |
|---|---|---|
| `methods/audit/` | 7 entries (full) | Seven `quality-auditor` audit lenses already had paired skills before this PoR was authored. Closing the `skillless canon` smell on them was a single mechanical pass. |
| `methods/investigation/` | 1 entry (`scientific-debugging`) | Default investigation method; future entries (bisect-debugging, minimal-reproduction, differential-trace, etc.) graduate via `meta-self-improvement` decisions when the bug-investigator's audit log surfaces graduation candidates. |
| `methods/readiness/` (this folder) | 21 entries (v1) | Deployment-manager owns the declarative checklist and consumes producer signals; method docs and the `readiness-signal` skill provide consistent review reasoning and procedure. |

Cross-reference: scenario-qa README `## Future PoR work` enumerates this and other gaps.

## Method doc and signal skill

Each check ships one method doc (reasoning). The `readiness-signal` skill carries the one return contract (status, source, run id, time, commit, evidence); the checklist item id picks the method doc.

## Shared evidence and failure requirements

Apply each check to a scenario and commit being prepared for sale. Use declared
evidence anchored to an attributable run or artifact. Do not infer a pass from a
declaration alone, reuse evidence for another commit, or treat an unavailable
producer as a pass. For a failure, record the mismatch, source, run identifier,
timestamp and exact checklist item; unanchored observations remain unknown until
directly evidenced. The authoritative policy is
[`readiness-policy.json`](../../../../scenarios/deployment-manager/api/readiness/readiness-policy.json).

## Lifecycle (applies when entries exist)

Each check has a status:

## Files in this folder

Each of the 21 checklist identifiers has one markdown method document here. The `readiness-signal` skill (`scenarios/prompt-manager/store/skills/packs/core/readiness-signal/`) is the one procedure skill for all of them.

## Adding a check

When the substrate calls for a concrete entry, follow the same flow as the sibling registries: file a `meta-self-improvement` decision proposing the addition, including (1) why now, (2) draft method doc. Operator approval activates.

## Cross-references

- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview; full set of registries and `## Future PoR work` items.
- [`../investigation/README.md`](../investigation/README.md) — sister registry; same lifecycle, currently 1 entry.
- [`../audit/README.md`](../audit/README.md) — sister registry; same lifecycle, currently 7 entries.
- [`docs/marketing/methods/post-techniques/README.md`](../../../marketing/methods/post-techniques/README.md) — gold-standard reference all three registries replicate.
