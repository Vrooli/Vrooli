# The Phase Capability Contract

This document is the design SSOT for the **Phase Capability Contract**: the
uniform shape every Test Genie phase declares so that, at the moment a run
finishes, an agent knows *where each capability stands* and *the single next move
to advance it*. It generalizes the pattern proven by cli-health's
[`command_architecture`](../../../cli-health/docs/reference/cli-architecture-maturity.md)
capability to every phase in the catalog, and it documents the now-gating
provider-conformance checks that keep adoption from drifting.

The problem it closes: the maturity infrastructure already *promises* directed
capability development but does not yet *deliver* it end to end. Providers already
declare gated ladders and compute a per-phase standing
(`commonv1.MaturityAssessment`: current rung, next level, blocking codes, priority
focus with reason), but that signal is dropped before it reaches the agent, the
remediation docs it should point to are structurally inconsistent across the
phase catalog, and the contract is only advisory. A Test Genie run therefore tells
an agent *whether* a phase passed, but not *how mature* the capability is or *what
to do next* — so capability development is guessed rather than directed.

## The four parts of the contract

Every phase declares a contract with four parts. The provider owns parts 1–3 in
its descriptor and computes part 3 at run time; Test Genie only aggregates and
renders. **No phase-specific knowledge lives in Test Genie** — this is a
guard-tested invariant.

1. **A first-class North Star.** The top level of the capability's ladder carries
   an explicit aspiration statement — what *maximum* maturity looks like for this
   capability, stated as a target an agent can aim at, not merely as the absence
   of findings. This is surfaced as **North Star** in both the remediation doc and
   the run output. (See [North Star requirement](#the-north-star-requirement).)

2. **A gated L0–L4 ladder.** The capability declares monotone rungs (each implies
   the one below). Every rung states a `capability_summary` (what the rung means —
   the North Star at the top) and every non-top rung a `next_unlock` (the single
   highest-unlock move to the next rung); these two are the load-bearing,
   enforced fields the standing surfaces. A provider may add `entry_criteria` /
   `exit_criteria` prose but they are optional. The ladder lives in the
   descriptor's `maturity` block — the single source of truth for the north star
   + ladder. A transition into a non-top rung must also have at least one
   declared finding that can actually gate it (`local_level_impact` on the
   destination rung plus required/error semantics); otherwise the scorecard may
   be unable to stop at that rung. For a multi-capability phase the
   per-capability ladders are the North Star SSOT (the standing surfaces the
   focus capability's ladder).

3. **A provider-returned standing.** At run time the provider computes, for each
   of its capabilities, a `commonv1.MaturityAssessment`: the current rung, the next
   level, the ceiling, the blocking finding codes, and the priority focus (the
   single next move plus the reason it is the highest unlock). Test Genie carries
   this standing from the server into the run contract and renders it — it never
   recomputes a standing.

4. **Structured remediation docs.** The finding codes a capability can emit point
   at a remediation document whose structure is fixed (see
   [the remediation-doc skeleton](#the-remediation-doc-skeleton)). Because the
   headings are uniform across every phase, an agent (or search-hub) can resolve a
   doc-search topic emitted in the run output to the exact structured section that
   explains a finding and its canonical fix — with no per-phase glue.

Parts 1, 2, and 4 are *declared* in the descriptor and its docs; part 3 is
*computed* by the provider and *rendered* by Test Genie. A phase is
**contract-conformant** when all four parts are present and well-formed, as
validated by the provider-conformance self-phase.

## Portable phase presentation

`common.v1.MaturityAssessment.presentation` is the portable, provider-owned
rendering contract. Its current `contract_version` is `v1`. The assessment and
its ungrouped `findings` remain the semantic and evidence sources; presentation
is a deterministic projection, never an alternative assessment.

The v1 rules are deliberately narrow:

- Capability groups are keyed by `capability_id`, ordered by positive
  `priority_rank` and then capability id. Each group preserves the computed
  current/next rung, priority reason, and blocking codes.
- Findings are grouped only by capability and code. A group is ordered by code,
  reports a count and sorted locations, and retains representative title,
  message, remediation, severity, and fix affordance. Raw findings remain
  available for complete evidence.
- The phase focus, next action, North Star, ceiling, and documentation topics
  come directly from the computed assessment. Consumers must not regroup,
  reorder, or invent a next action or documentation route.
- `PREVIEW_AVAILABLE` means the provider has reported deterministic-fixer
  availability; it is an invitation to call `PreviewFix`, not permission to
  claim an apply result. Manual and detection-only states remain explicit.

Test Genie stores and forwards the exact presentation object in run events,
terminal records, live status, findings artifacts, and CLI JSON. It may add
run-level status, timing, position, history, and a clearly-labelled cross-phase
top priority, but it does not add phase semantics. A missing, unsupported, or
non-canonical presentation is a maturity-contract failure for a delegated
provider. Native or degraded phases remain nil/degraded evidence rather than a
synthetic passing presentation. Historical runs retain whatever was persisted;
they are not backfilled from the current provider catalog.

The retired `PhaseMaturityStanding` wire fields remain decode-only at their
original protobuf numbers for historical run, event, and `findings.json` bytes.
New writers never set them. A reader exposes such data as
`legacy_maturity_standing` (and `test-genie runs findings` labels it a
historical, non-canonical standing), with no v1 presentation attached; this
prevents a field-number reinterpretation from inventing a malformed canonical
story for an older run.

The Phase Presentation v1 audit rejected a capability-prerequisite schema
extension. Existing ladder gates, blocking codes, current/next levels, and
unknown evidence already express the only verified dependency states. A generic
prerequisite graph would add unsupported semantics and risk hiding findings, so
v1 retains all evidence and introduces no suppression behavior.

## The North Star requirement

The ladder's **top level must carry an aspiration statement** — the North Star.
It answers "what does this capability look like when it is as good as it can be?"
in terms an agent can steer toward, independent of the current finding set. It is
declared as the top level's `capability_summary` (with the top level's `name`
serving as the short label), and it is surfaced verbatim as **North Star** in the
remediation doc's first section and in the run scorecard for a phase at or
climbing toward maximum maturity.

The convention: **the top ladder level is the North Star.** If experience shows
the top-level-summary convention is insufficient to carry a strong aspiration
(for example, a capability whose top rung is purely mechanical), a dedicated
`Spec`-level north-star field is added rather than overloading the summary — but
the convention is tried first and is the default. A capability whose top level
has no aspiration statement is a north-star gap, surfaced by provider-conformance.

## The remediation-doc skeleton

A capability's `docs.path` target is enforced not only for existence (the file
resolves) but for **structure**: it must contain a fixed set of H2 headings,
shaped as the *remediation question-space* an agent walks when a finding fires.
The five required H2 headings, in order:

| H2 heading | Answers | Maps to |
|---|---|---|
| `## North Star` | What maximum maturity looks like for this capability. | Ladder top-level aspiration (part 1). |
| `## The rungs and their gates` | The L0–L4 ladder: each rung's entry/exit criteria and the single next unlock. | Ladder (part 2). |
| `## What each finding means` | The finding-code inventory: each code, the rung it caps the capability at, its severity, and whether it fails the phase. | Standing + finding codes (part 3). |
| `## The canonical fix` | For each class of finding, the specific remediation an agent should apply. | Migration guidance. |
| `## How to verify` | The exact command(s) that confirm a rung was climbed. | Verification. |

Rules:

- The five H2 headings must be present and spelled exactly as above. Additional
  H2 sections (a `## Cross-references` footer, provider-specific detail) are
  permitted *after* the required five; the required five must all appear.
- The doc is the *depth*; the run output emits only runnable doc-search **topics**
  that resolve to these sections, never the depth itself.
- **Auto-fix scaffolds structure only, never content.** `test-genie phases scaffold`
  emits the five headings (and a maturity-spec stub) as empty scaffolding; a human
  or agent fills the prose. A scaffold that fabricated remediation content would
  produce confidently-wrong guidance, which is worse than an honest gap.

The skeleton is referenced from the [phase-docs index](../phases/README.md) so
every phase author sees the required shape.

## Enforcement policy

Provider-conformance now treats the Phase Capability Contract as a gating
descriptor contract:

1. **Descriptor and maturity validity gate.** A provider descriptor must parse,
   match its scenario identity, carry an embedded maturity spec, avoid retired
   `.vrooli/maturity.json`, and use a safe policy combination.
2. **North Star and ladder completeness gate.** Every enforced ladder must put
   the North Star in the top rung's `capability_summary`, and every non-top rung
   must provide `next_unlock`. Missing values are `ERROR` findings because the
   run scorecard cannot truthfully render a ceiling or single next move without
   them.
3. **Remediation-doc skeleton gate.** `docs.path` itself remains an advisory
   presence check when the target is absent, but once the file resolves it must
   contain the five required H2 headings. Missing headings are `ERROR` findings
   because emitted doc-search topics depend on that skeleton.
4. **Rung-gate coverage warning.** During fleet remediation,
   provider-conformance reports `PROVIDER_RUNG_UNGATED` when a ladder transition
   has no required/error finding mapped to its destination rung. This is
   advisory until the existing catalog descriptors are patched, then it becomes
   a gating contract check.
5. **Catalog coverage guard.** The default catalog is descriptor-backed: every
   phase is provider-delegated, every descriptor declares a maturity ladder, and
   any future native exception must be explicit, documented, and tested.

## Auto-fix boundary

Auto-fix for this contract is deliberately limited to scaffold seeding and
classification warnings. Test Genie may seed `.vrooli/test-genie.json` and the
five-heading remediation-doc skeleton, and provider-conformance warns when
finding mappings do not declare `fix_class`. It does not synthesize target
remediation, invent finding-to-rung mappings, or rewrite provider behavior. Those
changes require provider-specific judgment because an inaccurate maturity ladder
is worse than an honest incomplete one.

`contracts` (cli-health) remains the reference lighthouse adopter proven end to
end: run output renders its scorecard, and doc-search topics resolve through the
structured remediation doc.

Severity still owns phase pass/fail per the shared health contract: only ERROR
and BLOCKER findings fail a phase, so advisory findings never fail a run while the
fleet migrates.

## Run-scoped descriptor and evidence projection

The provider contract is live planning input; the run snapshot is historical
truth. At execution start, Test Genie captures the effective descriptor entries
and applicability decisions for the planned run. Catalog changes after that
point cannot rename, reorder, reattribute, or retroactively change the policy of
historical phase results.

Schema v1 is stored atomically at
`coverage/runs/<run-id>/descriptor-snapshot.json` before the durable run index
is promoted to `in_progress`. Its `ds:sha256:` digest and schema version are
stamped into the compact run record and `RunInfo`; a write failure prevents
phase execution. The later terminal snapshot embeds that same compact reference
instead of copying or re-reading the live provider catalog.

The run-owned projection is versioned and contains:

- immutable phase machine key plus optional explicit alias/supersedes lineage;
- display name, description, provider, ordering hint, phase/runtime class,
  dimensions, finding source, policy, maturity/docs references, and declared
  evidence kinds;
- planned, applicable, not-applicable, skipped, and unavailable decisions with
  typed reasons;
- the schema version and digest of the captured descriptor catalog.

Each terminal run also owns a typed artifact catalog. An artifact reference has
an opaque run-scoped id, stable kind, media type, label, producing phase key,
size/time metadata, safe access capability, relationships, and extensible
metadata. Known kinds may receive specialized consumers, but unknown kinds must
remain listable and retrievable through a safe generic path. Provider filesystem
paths are storage details and never cross the browser/API boundary.

Schema v1 is atomically stored at
`coverage/runs/<run-id>/artifact-catalog.json` before terminal publication. It
indexes the existing run tree and run-scoped logs rather than copying bytes into
a second store. IDs include the run identity in their digest, so an ID from one
run cannot address a same-named file in another. `ListRunArtifacts` and
`GetRunArtifact` expose path-free metadata; bytes stream through the opaque
`/api/v1/scenarios/<scenario>/runs/<run-id>/artifacts/<artifact-id>` route with
regular-file, symlink, containment, content-type, and active-content guards.
Pre-catalog runs use read-only discovery with explicit `legacy_discovery`
provenance and degraded metadata. The initial kind vocabulary covers command
output/logs, findings and coverage reports, phase results, screenshots, visual
diffs, workflow video, trace, HAR, console, network, DOM, and generic files;
kind remains an open string so future provider evidence survives unchanged.

Terminal `WaitRun`, show, history, comparison, and downstream adapters all
project one canonical persisted run snapshot. A pre-versioned run may be read
through an explicit legacy/degraded projection; unknowable fields are never
backfilled from the current catalog or represented as empty passing evidence.

Comparison joins immutable phase keys and preserves both sides. New, retired,
inapplicable, skipped, provider-unavailable, missing-artifact,
incompatible-schema, and legacy-metadata-unavailable outcomes use typed reason
codes rather than consumer-specific inference.
Both descriptors returned with a phase diff come from their respective run
snapshots. Finalization and comparison never consult `DefaultCatalog` to fill
historical labels, provider attribution, ordering, policy, or applicability.

## What this is not

- **Not a rewrite.** The substrate exists and is reused: the descriptor `maturity`
  block already deserializes to `assessment.Spec`; providers already return
  `commonv1.MaturityAssessment` via `assessment.BuildProtoAssessment`;
  `maturity-go/report` already renders a human standing; provider-conformance
  already validates provider well-formedness. This contract *finishes and enforces*
  that substrate.
- **Not phase knowledge in Test Genie.** Adding or removing a phase requires zero
  Test Genie code change; the standing and scorecard derive entirely from the
  provider-returned assessment and the descriptor.
- **Not verbose output.** The scorecard is a tight per-phase block; depth lives in
  the enforced, AI-searchable docs. Output emits runnable search topics, not the
  depth.

## Cross-references

- [`cli-architecture-maturity.md`](../../../cli-health/docs/reference/cli-architecture-maturity.md) — the reference adopter this contract generalizes.
- [`../phases/README.md`](../phases/README.md) — the phase catalog and where the skeleton convention is indexed.
- [`../../../../docs/reference/health-maturity-assessments.md`](../../../../docs/reference/health-maturity-assessments.md) — the shared provider maturity contract this plugs into.
- `packages/maturity-go/assessment/` — the `Spec`/ladder model and `BuildProtoAssessment` that computes the standing.
- `scenarios/test-genie/api/internal/providerconformance/` — the self-phase that enforces the contract.
