# Scenario QA Readiness Checks — Plan of Record

This folder is the **strategic canon** for readiness-dimension checks. One paired doc + skill per check, once entries exist.

> **Pre-emptive readiness ordering lives in swarm-manager now.** A former QA member ran these reviews on idle scenarios and filed fix items ahead of feature work. That ordering is now a deterministic swarm-manager gate (`fix_before_feature`) plus optional policy-governed maintenance intake through the backlog auto-filer (`auto_filer`). This registry remains the home for *individual readiness-check techniques* should they graduate into Vrooli skills; it no longer belongs to a specific QA member.

These docs would answer: *what is this readiness check, when does it apply, when does it backfire, what's the failure mode the qa-contrarian watches for?*

They would **not** answer: *step-by-step procedure for applying it.* That belongs in the paired skill.

## Status: stub

This registry currently has **no entries**. It exists to reserve the canonical home for individual readiness checks once GCT readiness dimensions stabilize and individual checks become candidate skills.

The registry's shape — paired doc + skill per check — mirrors the `methods/investigation/` and `methods/audit/` siblings. Adoption rules below apply when the first concrete entry lands; until then, this stub is the only file.

## Why this is intentional asymmetry

The scenario-qa team's three technique registries are at deliberately different maturity levels:

| Registry | Maturity | Reason |
|---|---|---|
| `methods/audit/` | 7 entries (full) | Seven `quality-auditor` audit lenses already had paired skills before this PoR was authored. Closing the `skillless canon` smell on them was a single mechanical pass. |
| `methods/investigation/` | 1 entry (`scientific-debugging`) | Default investigation method; future entries (bisect-debugging, minimal-reproduction, differential-trace, etc.) graduate via `meta-self-improvement` decisions when the bug-investigator's audit log surfaces graduation candidates. |
| `methods/readiness/` (this folder) | 0 entries (stub) | GCT readiness dimensions are externally driven — by the GCT scenario itself, not by Vrooli skills. The right registry shape isn't yet known until those dimensions stabilize or are replaced by an internal Vrooli equivalent. Stubbing now reserves the home. |

Cross-reference: scenario-qa README `## Future PoR work` enumerates this and other gaps.

## Doc + paired skill discipline

When this registry receives its first entry, the same mandatory rule applies (mirrored from [`docs/marketing/catalogs/post-types/README.md`](../../../marketing/post-types/README.md)):

> Every entry ships as `doc + paired skill`. This is a hard rule, not a recommendation. Neither half is optional, and neither half replaces the other. The doc holds *reasoning*; the skill holds *procedure*. A doc with no skill is a stale shrine. A skill with no doc is brittle.

The canon coherence test at `scenarios/prompt-manager/test/agent_system_canon_test.sh` enforces pairing across all three registries via a shared `{registryDir, skillTag}` pair table; this readiness registry plugs in by appending one entry to that table once the first paired entry exists.

## Lifecycle (applies when entries exist)

Each check has a status:

- **v0** — Strategic canon documented, not yet active. PoR doc exists; paired skill missing or incomplete; consumers must not include it.
- **v1** — Active. Three activation requirements:
  1. Skill is authored at `path:scenarios/prompt-manager/store/skills/packs/core/<slug>/`.
  2. Skill cites this check's PoR doc as required reading.
  3. PoR doc Status line bumped to `v1`.

## Files in this folder

None yet — populated as GCT dimensions stabilize.

## Adding a check

When the substrate calls for a concrete entry, follow the same flow as the sibling registries: file a `meta-self-improvement` decision proposing the addition, including (1) why now, (2) draft PoR doc, (3) draft paired skill with `tags: ["readiness-check"]`. Operator approval activates.

## Cross-references

- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview; full set of registries and `## Future PoR work` items.
- [`../investigation/README.md`](../investigation/README.md) — sister registry; same lifecycle, currently 1 entry.
- [`../audit/README.md`](../audit/README.md) — sister registry; same lifecycle, currently 7 entries.
- [`docs/marketing/methods/post-techniques/README.md`](../../../marketing/post-techniques/README.md) — gold-standard reference all three registries replicate.
