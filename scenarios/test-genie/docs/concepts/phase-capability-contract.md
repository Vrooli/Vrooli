# The Phase Capability Contract

This document is the design SSOT for the **Phase Capability Contract**: the
uniform shape every Test Genie phase declares so that, at the moment a run
finishes, an agent knows *where each capability stands* and *the single next move
to advance it*. It generalizes the pattern proven by cli-health's
[`command_architecture`](../../../cli-health/docs/reference/cli-architecture-maturity.md)
capability to every phase in the catalog, and it defines the advisory-then-gating,
lighthouse-first rollout policy that governs adoption.

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
   + ladder. For a multi-capability phase the per-capability ladders are the
   North Star SSOT (the standing surfaces the focus capability's ladder).

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

## Rollout policy (advisory → gating, lighthouse-first)

Adoption follows the same shape proven by the requirements-traceability and
cli-architecture-maturity rollouts: **advisory first, one lighthouse phase proven
end to end, then gating.**

1. **Advisory.** Provider-conformance emits the new north-star and doc-skeleton
   checks as *advisory* findings (WARNING, non-failing) for every phase that has
   gaps. Nothing is destabilized; the fleet's honest debt becomes visible.
2. **Lighthouse.** One phase — **cli-health's `contracts` phase** — is brought to
   full conformance first and proven end to end: a real run renders its scorecard,
   and a doc-search topic from the output resolves through search-hub /
   knowledge-observatory to the intended structured remediation section, with no
   manual glue. The lighthouse proves the docs are executable before the fleet
   migrates.
3. **Migrate.** Every provider-delegated phase is brought to advisory-clean
   (north star + complete ladder + skeleton docs). Test-Genie-*native* phases,
   which do not fit a provider ladder, carry an **explicit, documented exemption**
   rather than a forced ill-fitting ladder — they are marked, never silently
   skipped.
4. **Gate.** The north-star and doc-skeleton checks graduate from advisory to
   **gating** for compliant phases, guarded against drift. Test Genie validates
   its *own* descriptor against the contract (recursion), and anti-drift guard
   tests assert that every catalog phase resolves a conformant contract and that
   the scorecard renders for every phase.

Severity still owns phase pass/fail per the shared health contract: only ERROR
and BLOCKER findings fail a phase, so advisory findings never fail a run while the
fleet migrates.

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
