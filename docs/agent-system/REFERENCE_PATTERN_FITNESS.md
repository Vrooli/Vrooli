# Audit Lens: Reference-Pattern Fitness

**Status:** v1 (paired with new `reference-pattern-fitness` skill, 2026-05-04). Established by a `meta-self-improvement` decision filed against [`path:scenarios/prompt-manager/store/teams/meta-optimization/shared/decisions.jsonl`](../../scenarios/prompt-manager/store/teams/meta-optimization/shared/decisions.jsonl). Owned by the `toolchain-validator` member of the `meta-optimization` team.

## Definition

Audit whether an artifact that **exists to be copied** is fit to be a copy source. The lens applies only to a specific class of artifacts:

- **Templates** — `path:templates/scenarios/<name>/`, fed into `vrooli scenario generate`. Every choice multiplies across N future scenarios.
- **Reference scenarios** — entries in [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md). The reference's quality is bounded above by the template it was generated from.
- **Canonical examples** inside scenarios — patterns documented as "copy this when adding X" (e.g., the `notes` CRUD reference inside `path:templates/scenarios/react-vite/`, accompanied by a `REPLACING-NOTES.md`-style guide).

The lens does **not** ask *"is this code good?"* — that is what the seven single-instance audit lenses in [`path:docs/scenario-qa/methods/audit/`](../scenario-qa/methods/audit/) are for. This lens asks the multiplier-aware question: *"is this code good **as a copy source**?"* A 50-line CLI helper duplicated per domain isn't 50 lines of debt — it's 50 × D × S lines across the portfolio. A contract encoded in a doc comment survives a careful read but not a copy/paste/modify cycle. A "delete this with sed" guide proves the architecture's removal promise wasn't real.

The full procedure (applicability gate → run single-instance lenses first → four sub-lenses → tier findings → categorize substrate-vs-template → notebook entry → optional decision proposal) lives in the paired skill. This document is the strategic-canon side: when the lens applies, when it backfires, what the meta-contrarian challenges, and how it composes with the existing audit lenses.

## When it applies

✅ **The artifact is registered as a template.** Anything under `path:templates/scenarios/<name>/`. Every defect multiplies across every scenario generated from it.

✅ **The artifact is registered in [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md).** Gold-star or secondary references are the substrate the `toolchain-validator` validates tools and skills against; their quality directly shapes the validator's signal.

✅ **The artifact is a documented canonical example.** Patterns inside a scenario explicitly marked as "copy this when adding your first X" — typically accompanied by a `REPLACING-X.md` guide. The `notes` reference inside the react-vite template is the exemplar.

✅ **You are evaluating whether to promote an artifact to template/reference status.** Run this lens before nominating, not after — the multiplier framing is what determines fitness for promotion.

✅ **A reference scored dirty against `scenario-auditor` / `test-genie` / `tidiness-manager` and the violations trace back to the template it was generated from.** Per [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md)'s rot-cause taxonomy, the template-rotted-the-reference case is the natural trigger for this lens.

## When it backfires

⚠️ **Run on regular feature code in production scenarios.** The multiplier framing doesn't apply. Findings will read as premature substrate extraction or speculative refactoring. Use the relevant single-instance lens (`refactor`, `screaming-architecture-audit`, `decision-boundary-extraction`, etc.) instead.

⚠️ **Run before single-instance lenses.** The multiplier framing assumes the artifact is otherwise structurally sound. If the underlying code has god modules or scattered responsibilities, fix those first; this lens applied to broken structure will surface noise (every duplicated bad pattern looks like a substrate finding when the real problem is the pattern itself).

⚠️ **Used to justify substrate work without the third-repetition trigger.** Vrooli's "don't extract until you see the pattern" rule still applies. This lens flags candidates and lets you decide whether the proposed substrate home (cli-core, api-core, shared lib) actually exists today; it does not authorize extraction work — that's a separate decision.

⚠️ **Treated as a single-pass audit.** Each substrate addition re-prices what's left in the artifact. The lens needs re-running after substrate changes, not banked as "we already audited this."

⚠️ **Used as a substitute for `scenario-auditor` / `test-genie` / `tidiness-manager`.** The standard tooling gates remain primary; this lens is the longer-cadence companion. A reference that's clean against the tools but failing on multiplier framing is a different conversation than a reference that fails the tools.

## The four sub-lenses

These are the substantive content the lens applies to a qualifying artifact. They produce evidence; the procedure (in the paired skill) tiers and categorizes that evidence.

**Per-replica cost.** Walk the artifact and count duplicated infrastructure that scales per copy. For each candidate, record (a) line count per replica, (b) replication factor (templates: N future scenarios; references: 1; canonical examples: N future domains in this scenario), (c) proposed substrate home, (d) does that home exist today? Anything > ~20 lines of pure infrastructure repeated per copy is a substrate-extraction candidate.

**Drift surface map.** Enumerate every place where N future copies must agree but only convention enforces it. For each: type-system, CI check, or hope? Examples surfaced on the react-vite template (2026-05-04): `lib/api.ts`'s plain `Error` vs `lib/notes.ts`'s typed `ApiError` (already drifted inside the template); `cli_commands_seed.json` as a third source of truth for CLI commands beyond `register.go` and `cli_mapping.command`.

Hand-rolled fakes of shared-package interfaces are also drift surfaces. When a
shared package owns a public surface that consumers need to fake or harness in
tests, the canonical mitigation is a top-level `<pkg>test` sibling package
documented in [`SHARED_PACKAGE_TESTING.md`](SHARED_PACKAGE_TESTING.md). Scenario
and template code should consume those companions instead of copying local
fakes for `api-core` or `cli-core` interfaces.

**Contract location audit.** For every non-trivial contract (precondition, invariant, "callers must / must-not"), where does it live — type signature, code comment, or docs? Comment-only contracts are debt at scale because they don't survive copy-paste-and-modify. Example surfaced on the react-vite template: `Repository.Create` takes a `Note` with caller-zero ID/timestamps; the contract *"callers must leave these zero-valued"* is encoded in a doc comment, not in the type system. The fix is a `RepositoryCreateInput { Title, Body }` DTO that makes the contract type-level.

**Coordinated-edit count for add/delete.** Perform the canonical add-domain and delete-domain walkthroughs (or the artifact's analogues — add-feature, replace-example). Report the count of central files touched. Anything > 5 is a substrate finding. The react-vite template's earlier "9 coordinated edits to delete the notes reference" was the canonical instance of this finding; a Pass-3 module-pattern refactor reduced add-domain to 5 central edits and delete-domain to mostly `rm -rf`.

## What the meta-contrarian challenges

The `contrarian` member of the `meta-optimization` team is the mandatory skeptic for proposals from other members. For `reference-pattern-fitness` outputs specifically, the contrarian challenges:

- **"Could this be wrong on a single instance too?"** If the finding would be valid on regular feature code in a production scenario, it doesn't need the multiplier framing — file it under the relevant single-instance lens, not here. The lens loses signal when used for findings that already had a valid home.
- **"Is the multiplier hypothetical or measured?"** Demand a concrete count: N templates × M future domains × S scenarios. Speculative multipliers ("imagine this gets copied a lot") are noise. A finding without a concrete replication factor is not a Tier 1 candidate.
- **"Does the substrate extraction proposed here actually exist anywhere yet?"** If the proposed home (cli-core, api-core, shared lib) doesn't exist today or doesn't accept the helper today, the finding is a wishlist item, not actionable. Either the substrate work belongs to a separate decision, or the finding stays in the in-artifact-fix category until the substrate exists.
- **"Is this prose paint?"** Description style, naming consistency, prose drift can be surfaced but should be ranked Tier 3, never Tier 1. Tier 1 is reserved for findings that change runtime shape, contract location, or coordinated-edit count.
- **"Did the auditor run the single-instance lenses first?"** If the notebook entry doesn't cite single-instance findings as a prerequisite, the multiplier findings rest on unverified ground. Block the proposal until the prerequisite is satisfied.
- **"Is the artifact actually copied at the rate the auditor assumes?"** A canonical example documented as "copy this for X" but never actually copied (because the X never came up) is a paper template. Real replication count, not documented replication intent, is the load-bearing input.

## Composition with single-instance lenses

This lens **runs after** the relevant single-instance lenses on the same artifact, not instead of them. Multiplier-framed findings are only correct *given that the artifact is otherwise structurally sound*. Asking *"is this fit to be copied?"* before *"is this code structurally sound?"* produces noise.

The auditor selects from [`path:docs/scenario-qa/methods/audit/`](../scenario-qa/methods/audit/) based on the artifact's shape. For a CRUD-template audit (the react-vite case), `screaming-architecture-audit`, `decision-boundary-extraction`, and `utils-unification` are typical prerequisites; for a reference scenario being scrutinized for testing patterns, `seam-discovery-and-enforcement` and `invariant-discovery-and-enforcement` are typical. The paired skill's required-reading section codifies this — the auditor names the prerequisites in their notebook entry.

This lens does **not** belong in the [`path:docs/scenario-qa/methods/audit/`](../scenario-qa/methods/audit/) registry. That registry is scoped to the `quality-auditor` member of the `scenario-qa` team, which audits real scenarios where multiplier framing would mislead. Promotion to its own registry — `path:docs/agent-system/audit-techniques/` — is deferred until a second similar lens lands.

## Worked example: react-vite template (2026-05-04)

The first concrete application of this lens. A long-form session audited `path:templates/scenarios/react-vite/` after the seven single-instance lenses had each been applied to the same artifact at various points without surfacing multiplier-aware findings. The lens produced six tiered findings:

**Tier 1 — per-replica cost / coordinated-edit count > threshold:**
1. CLI per-domain client boilerplate (`apiError`, `decodeEnvelope`, `formatX`, protojson decode ribbon) — ~50 lines duplicated per domain. Substrate home: cli-core (`cliapp.DecodeEnvelope`, `cliapp.WrapAPIError`). Substrate exists.
2. UI per-domain lib boilerplate (`decodeApiError`, `if (!res.ok)` ribbon, `fromJson + guard`) — ~30 lines per method × 3 methods × N domains. Substrate home: in-template `path:ui/src/lib/api.ts` (a `protoFetch<Req,Resp>` helper). Substrate exists; in-template fix.
3. Wire-path duplicated between `handler.go` and `endpoints.go` — drift surface; route descriptors and registered routes can disagree silently. CI check missing. Fix: a `module_test.go`-level walk-the-router parity assertion. Substrate exists.

**Tier 2 — drift surfaces, contract leakage:**
4. `cli_commands_seed.json` as a third source of truth for CLI commands. Cross-check is partial (endpoints → seed only). Fix: build-time check that seed names ⊇ registered subcommand names.
5. `Repository.Create` accepts a `Note` with caller-zero fields; contract is in a doc comment, not the type system. Fix: `RepositoryCreateInput { Title, Body }` DTO; Service translates.

**Tier 3 — prose / style:**
6. `endpoints.go` Description fields mix user-facing copy with implementation notes; future domains will copy the inconsistent style. Fix: convention written into ARCHITECTURE.md ("descriptions are < 100 chars, audience = API-list scanner; implementation notes belong in code comments").

**Substrate-vs-template breakdown:** 1 (CLI), 4 (CLI seed check) → cli-core. 2 (UI helper), 3 (route parity test), 5 (DTO), 6 (style convention) → in-template. The breakdown answered the user's strategic question — *which fixes need cross-scenario substrate work, and which fix in-place?*

The findings became a Pass-4 plan against the template (separate file, separate decision); this lens is the methodology that produced them.

This worked example is frozen as of 2026-05-04. Future audits create new notebook entries under `meta-optimization/notebook/template-fitness/<artifact-slug>/<YYYY-MM-DD>` per the paired skill's procedure; the canon doc cites those by date.

## Paired skill

`path:scenarios/prompt-manager/store/skills/packs/core/reference-pattern-fitness/SKILL.md` — the executable spec. Tagged `template-fitness` (deliberately not `audit-technique` — keeps the skill out of scenario-qa's registry coherence test). Required reading: this PoR doc, [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md), `prompt-manager skills read knowledge-observatory-tools`, plus the single-instance lens(es) appropriate to the artifact.

## Cross-references

- [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md) — registry of templates and the references generated from them. Confirms applicability before running this lens.
- [`README.md`](README.md) — agent-system canon-doc index.
- [`../scenario-qa/methods/audit/README.md`](../scenario-qa/methods/audit/README.md) — sibling registry for the seven single-instance lenses. This lens is **not** in it, by design (see "Composition with single-instance lenses" above).
- [`../../scenarios/prompt-manager/store/teams/meta-optimization/members/toolchain-validator/RESPONSIBILITIES.md`](../../scenarios/prompt-manager/store/teams/meta-optimization/members/toolchain-validator/RESPONSIBILITIES.md) — the consumer.
- [`PROMOTION_LADDER.md`](PROMOTION_LADDER.md) — when this lens accumulates a sibling, graduate to a registry per the ladder's promotion criteria.
