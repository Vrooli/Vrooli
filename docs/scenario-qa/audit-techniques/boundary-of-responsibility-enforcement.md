# Audit Technique: Boundary-of-Responsibility Enforcement

**Status:** v1 (paired with existing `boundary-of-responsibility-enforcement` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Audit whether **each part of the scenario is only doing the work it is responsible for**. Presentation, coordination, domain rules, integrations, and cross-cutting concerns each have a proper home; logic that has bled across those zones gets moved back where it belongs.

The full procedure (identify responsibility zones → detect leaks → move logic to proper home → clarify interfaces → consolidate competing patterns → tests and safety nets) lives in the paired skill. This document is the strategic-canon side: when the lens applies, when it backfires, and what the qa-contrarian watches for.

## When it applies

✅ **Domain decisions inside UI/entrypoint code.** Components/handlers/CLI commands performing complex validation, business rules, or data reshaping that should live deeper in the system.

✅ **Domain logic depending on transport details.** Core logic that imports request/response objects, component state, or styling concerns — the domain is leaking presentation knowledge.

✅ **Integration code scattered.** Database calls, API clients, filesystem access spread across many modules instead of flowing through clear adapter boundaries.

✅ **Cross-cutting concerns intertwined with core logic.** Logging, metrics, tracing, feature flags woven through domain code so that the actual rule is hard to read.

✅ **"Helpers" mixing unrelated responsibilities.** A `utils.go` or `helpers.ts` doing five unrelated things; the file name reveals nothing about the responsibilities inside.

✅ **Mature scenarios where new contributors ask "where should this go?"** When the answer isn't obvious, responsibilities aren't clearly assigned.

## When it backfires

⚠️ **Premature layering.** Forcing a small scenario through a five-layer architecture (presentation → coordinator → domain service → repository → adapter) when two layers would do. The skill explicitly warns against introducing new abstraction layers without clear payoff.

⚠️ **Adapter proliferation.** Wrapping every integration in a thin pass-through adapter "for testability" without actually decoupling anything. Pair with `seam-discovery-and-enforcement` to validate the adapter actually serves a seam.

⚠️ **Domain-purity dogma.** Demanding zero coupling between domain and integrations in a scenario that's intrinsically integration-heavy (e.g., a CLI wrapper around an external API). The "domain" may be near-trivial; forcing a domain layer adds ceremony without clarity.

⚠️ **As a substitute for naming clarity.** A move that puts code in the right zone but leaves bad names behind doesn't actually improve ownership. Pair with `screaming-architecture-audit`.

⚠️ **Behavior changes disguised as boundary moves.** The skill explicitly forbids weakening tests or changing observable behavior; when the lens is misapplied as "make this work differently," it violates its own guardrails.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges audit outcomes; for `boundary-of-responsibility-enforcement` specifically, watch for:

- **New layers added without payoff.** The audit introduced a coordinator layer between two layers that talked fine before. Challenge: what specific responsibility leak was this layer added to fix, and is the new boundary actually respected?
- **Adapters that don't isolate.** A new adapter wraps the old direct call, but the old call's types and errors leak through unchanged. Challenge: would substituting a different implementation actually be possible without rewriting the adapter?
- **Test thinning.** Tests that exercised end-to-end behavior were replaced with thinner unit tests at the new boundaries; the broader behavior is no longer covered. Challenge: did the audit improve test design or just move where coverage used to be?
- **Domain logic relocated, not extracted.** Logic moved from a UI file to a "domain" file but the logic itself is unchanged — still depends on UI types, still couples to transport. Challenge: was responsibility actually transferred, or just the file path?
- **Singleton / global state retained.** Boundary moves don't address shared mutable state; the audit declared boundaries clearer while leaving the global that violates them. Challenge: is the new boundary real or wishful?
- **Conflicts with `screaming-architecture-audit`.** The audit moved code into a "domain" zone whose name doesn't match domain vocabulary, or the responsibility split contradicts the structural one. Challenge: do the two lenses agree on where the code should live?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/boundary-of-responsibility-enforcement/SKILL.md` — the executable spec. Required reading: `prompt-manager skills read knowledge-observatory-tools` and this PoR doc.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`screaming-architecture-audit.md`](screaming-architecture-audit.md) — companion lens for *names and grouping*; this lens governs *who owns what*.
- [`seam-discovery-and-enforcement.md`](seam-discovery-and-enforcement.md) — companion lens; clear boundaries are a precondition for testable seams.
- [`../README.md`](../README.md) — scenario-qa team plan-of-record overview.
