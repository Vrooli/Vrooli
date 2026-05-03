# Audit Technique: Seam Discovery & Enforcement

**Status:** v1 (paired with existing `seam-discovery-and-enforcement` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Audit whether the scenario has **clear, robust seams at points where behavior may need to vary or be substituted**. A seam is a deliberate boundary — a place where a different implementation, a stub, or a side-effect-free version can be plugged in for testing, environment changes, or future evolution.

The full procedure (understand variation points → discover existing seams → identify missing/eroded ones → strengthen and enforce → improve testability at seams → safe-refactoring guidelines) lives in the paired skill. This document is the strategic-canon side.

## When it applies

✅ **Hard-to-test core logic.** Tests require excessive setup, hit real network/database, or can't isolate the unit under test because side effects are entangled with rules.

✅ **Repeated low-level integration calls.** The same external system is invoked from many places, each with its own setup boilerplate — there's no single boundary to mock or substitute.

✅ **Side effects scattered.** Filesystem access, network calls, time/random sources called inline in the middle of domain logic.

✅ **Environment-specific branches.** `if process.env.NODE_ENV === 'production'` (or equivalent) littered through code that should be environment-agnostic.

✅ **Feature flags evaluated everywhere.** Flag checks repeated at many call sites instead of being routed through a single decision point — scenario behavior is hard to enumerate.

✅ **Future variation needs.** A scenario where a new backend, new provider, or new mode is on the roadmap; introducing the seam *before* the variation lands lowers the cost of the variation.

## When it backfires

⚠️ **Speculative seams.** Adding adapters, interfaces, or strategy patterns "in case" a future variation lands. The skill prefers strengthening seams that exist; speculative ones add indirection without payoff. Wait for the second concrete consumer before extracting.

⚠️ **Over-mocking.** Strengthening a seam tempts test authors to mock through it for everything; tests then describe the mock contract rather than the scenario's behavior. Pair with the test-design discipline in `cognitive-load-reduction`.

⚠️ **Renaming that doesn't decouple.** An "adapter" that lets the same concrete type leak through still couples callers to the implementation. The skill warns against this — challenge whether substitution is actually possible after the change.

⚠️ **Conflict with simplicity.** A two-call pass-through wrapper adds friction without benefit. If the integration only has one consumer and one mode, a direct call may be the right choice; introduce the seam when the second consumer or mode arrives.

⚠️ **As a substitute for `boundary-of-responsibility-enforcement`.** Seams are about *variation*, boundaries are about *ownership*. A seam at the wrong layer (e.g., between two domain modules that should be one) creates a new problem instead of solving the old one.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges audit outcomes; for `seam-discovery-and-enforcement` specifically, watch for:

- **Adapters with leaky types.** The new seam wraps the old implementation but exposes the old type system unchanged — substitution is theoretical, not actual. Challenge: name a second implementation that could plug in here without changes elsewhere.
- **Test setup that grew, not shrank.** A new seam was supposed to make tests easier; instead test files now construct elaborate mocks for the seam itself. Challenge: did this change reduce setup or relocate it?
- **Over-mocking.** New tests use the seam to mock everything that crosses it, reducing tests to "the mock returned what we told it to." Challenge: do tests still describe scenario behavior, or only mock interactions?
- **Speculative extraction.** A seam was introduced for a variation that doesn't exist and isn't on the roadmap. Challenge: what's the second consumer or use case? If none, why now?
- **Existing seams left weak.** The audit added a new seam while bypassing or weakening another seam that was already eroded. Challenge: was the right seam strengthened, or just the most convenient one?
- **Performance regression hidden by mock-fast tests.** Real-call latency increased because the new seam adds indirection; mocked tests don't notice. Challenge: did real-environment performance get measured?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/seam-discovery-and-enforcement/SKILL.md` — the executable spec. Required reading: `prompt-manager skills read knowledge-observatory-tools` and this PoR doc.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`boundary-of-responsibility-enforcement.md`](boundary-of-responsibility-enforcement.md) — companion lens; clear ownership is a precondition for testable seams.
- [`invariant-discovery-and-enforcement.md`](invariant-discovery-and-enforcement.md) — companion lens; seams are often where invariants must be enforced (e.g., at integration boundaries).
- [`../README.md`](../README.md) — scenario-qa team plan-of-record overview.
- `docs/internal/SEAMS.md` (per-scenario) — operator/agent-curated seam map produced by audits.
