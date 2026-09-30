# Audit Technique: Invariant Discovery & Enforcement

**Status:** v1 (paired with existing `invariant-discovery-and-enforcement` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Audit whether the scenario's **critical invariants** — conditions that must always be true for the system to behave correctly — are explicit, named, and safely enforced. Examples: "this list is always sortable by date," "this route requires an authenticated user," "this state always has an ID." Implicit invariants are landmines; explicit ones are guardrails.

The full procedure (understand intended behavior → discover existing implicit invariants → name them → encode safely via types/assertions/tests → handle violations gracefully → avoid over-constraining) lives in the paired skill. This document is the strategic-canon side.

## When it applies

✅ **Repeated guards or null checks.** The same defensive check appears at many sites — that's an unnamed invariant ("X is never null here") begging to be promoted.

✅ **Crash classes the team keeps fixing.** Same shape of bug recurs (e.g., "ID was empty," "user wasn't authenticated," "config field missing"). The bugs are invariant violations the system isn't enforcing.

✅ **Data corruption or security risk.** Conditions whose violation would corrupt data or breach access — these *must* be invariants, not norms.

✅ **State machines with implicit transitions.** A scenario whose states transition correctly today but only because every caller happens to follow the right sequence. The right sequence should be the only representable sequence.

✅ **Onboarding friction.** New contributors break things by not knowing the implicit rules. Each "you have to do X first or else Y" piece of tribal knowledge is a candidate invariant.

✅ **Critical entrypoints.** Public APIs, scheduled jobs, webhook handlers — places where invalid input must be rejected before contaminating downstream state.

## When it backfires

⚠️ **Promoting bugs to invariants.** The skill explicitly warns: do not turn current bugs, temporary workarounds, or incomplete behavior into invariants. Freezing accidental behavior makes future correct fixes harder.

⚠️ **Over-strict types blocking legitimate edge cases.** A "stricter shape" that excludes a real-world case the scenario is supposed to handle. Pair with the PRD: if the spec allows the case, the type must too.

⚠️ **User-hostile failure modes.** Hard assertions in user-facing flows that crash instead of rendering a clear error state. The skill specifies "fail fast in trusted internal flows; fail gracefully in user-facing flows" — getting that distinction wrong creates outages.

⚠️ **Invariant inflation.** Over-naming every condition as an invariant when most are mere preconditions of a single function. The audit value comes from identifying the *critical* ones; cataloguing the trivial ones is busywork.

⚠️ **Conflict with future variation.** An invariant encoded today blocks a planned-but-unreached extension. The skill warns: do not block legitimate future extensions. If a roadmap item will require relaxing the invariant, encoding it now is premature.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges audit outcomes; for `invariant-discovery-and-enforcement` specifically, watch for:

- **Frozen bugs.** The audit named a current behavior as an invariant when the behavior is actually a bug (e.g., "this list always has duplicates" when it should not). Challenge: does this invariant align with the PRD, or is it just describing what the broken code does today?
- **Crash-on-violation in user flows.** A new assertion fires in a render path or API handler, taking down the user instead of rendering an error. Challenge: was the failure mode chosen for the right context?
- **Tests that lock in the wrong rule.** New tests assert the invariant; if the invariant is wrong, the tests now defend the wrong behavior against future correct fixes. Challenge: did the invariant's correctness get verified independent of the test it ships with?
- **Performance regression from runtime checks.** Assertions added in hot paths that fire on every request; the invariant is enforced but the scenario got slower. Challenge: was static encoding (types) preferred over runtime enforcement?
- **Documentation gap.** The invariant is enforced in code but not described in `docs/internal/INVARIANTS.md`; future readers won't know it exists. Challenge: is the rule visible to a reader who doesn't grep the codebase?
- **Roadmap conflict.** The invariant is incompatible with a planned feature, but the audit didn't surface this. Challenge: was the planned roadmap consulted before locking the rule?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/invariant-discovery-and-enforcement/SKILL.md` — the executable spec. Required reading: this PoR doc. The skill writes findings to `docs/internal/INVARIANTS.md` per scenario.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`seam-discovery-and-enforcement.md`](seam-discovery-and-enforcement.md) — companion lens; seams are often where invariants get enforced (validation at the boundary).
- [`decision-boundary-extraction.md`](decision-boundary-extraction.md) — companion lens; named decision points often imply invariants on their inputs.
- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
