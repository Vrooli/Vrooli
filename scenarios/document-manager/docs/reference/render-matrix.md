# Render Matrix — Document Manager

What this scenario can produce, which renderer chain produces it, what
that chain can faithfully represent, and what happens when a target
cannot express something the spec requires.

This is the write-side twin of
[`format-matrix.md`](format-matrix.md). Read them together: the format
matrix answers "what can we understand," this one answers "what can we
produce," and they share one normalized document model.

> **Status: P2, unscaffolded.** The generation spine is designed here and
> in [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) so the boundary
> is not improvised later. No renderer is selected, no registry file
> exists, and no code is generated until the read spine ships — the same
> treatment `handoff` received. See the write-spine rows in
> [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Purpose Of This Document

Use this document to answer:

- Which output targets does this scenario produce, and at what priority?
- Which renderer chain produces a given target, and what can it honor?
- What does a template have to declare before it can render to a target?
- What happens when a target cannot represent part of a spec?
- What has to happen to add a target or a renderer?

Narrative rationale belongs in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md). Spec and
template storage belongs in [`../concepts/DATA.md`](../concepts/DATA.md).
The runtime sequence belongs in [`../concepts/FLOWS.md`](../concepts/FLOWS.md).

## The Registry Is The Source Of Truth

Target support is **declared data, not inferred behavior**. The renderer
registry at `api/internal/render/registry.json` is the single source of
truth; this document is its human-readable projection, exactly as
`format-matrix.md` projects the handler registry.

A test asserts the two agree. A target present in one and absent from the
other fails the build. Nothing renders by accident, and nothing silently
stops rendering.

The registry exists so that **no renderer is load-bearing by default**.
Every renderer is a declared, replaceable participant. Adopting a good
library means routing to it, not building the scenario inside it — the
reasoning error the `anydoc` rows already record on the read side, and
the one most likely to recur here because a render toolchain is a large,
opinionated dependency.

## Renderers Declare Fidelity, Not Formats

The read side's routing unit is not "format X uses parser Y" but a chain
of handlers selected by **capability**. The write side mirrors this: the
routing unit is not "target X uses renderer Y" but a chain selected by
**fidelity** — what a target can actually represent.

| Fidelity | Meaning |
|---|---|
| `styled-text` | Typed, styled prose with heading hierarchy and lists |
| `paged-geometry` | Fixed pages with placeable regions — the property that makes a slide a slide |
| `cell-structure` | Addressable cells rather than a table rendered as prose |
| `speaker-notes` | A second content stream per unit, addressable separately from body content |
| `vector-embed` | Embedded vector graphics that survive as vectors rather than as rasterized images |

A **spec** declares which fidelities it *requires* and which are
*desirable*. A **template** declares, per target, which it can honor. The
router selects the cheapest chain satisfying the required set; anything
in the desirable set that no chain provides is recorded as a partial
render rather than dropped.

This is why "can this deck also be a PDF?" is a declared property rather
than a hopeful attempt: a deck template that declares `speaker-notes` for
`.pptx` and not for PDF is telling the truth about both.

## The Fidelity/Target Matrix

Targets are listed with the fidelity a well-chosen renderer should be
able to honor. **The renderer column is deliberately unfilled** — no
toolchain has been selected, and pre-committing one here would repeat the
mistake of treating a library as the architecture.

| Family | Targets | Fidelity honored | Renderer | Priority |
|---|---|---|---|---|
| Presentations | `.pptx` | `styled-text`, `paged-geometry`, `speaker-notes`, `vector-embed` | unselected | P2 |
| Paged documents | `.pdf` | `styled-text`, `paged-geometry`, `vector-embed` | unselected | P2 |
| Word processing | `.docx` | `styled-text`, `vector-embed` | unselected | P2 |
| Spreadsheets | `.xlsx` | `cell-structure`, `styled-text` | unselected | P2 |
| Markup | `.md`, `.html` | `styled-text` | unselected | P2 |
| Presentations, open | `.odp` | `styled-text`, `paged-geometry`, `speaker-notes` | unselected | future |
| Word processing, open | `.odt` | `styled-text` | unselected | future |

**Symmetry with intake is a target, not a promise.** The read side
accepts far more than this list, and it should: accepting `.eml`, `.epub`
and scanned TIFFs is cheap, while *producing* them is a different kind of
work with no evidenced demand. Where a format appears in
[`format-matrix.md`](format-matrix.md) and not here, that is a decision
rather than a gap, and adding it needs the same evidence any other target
needs.

## Templates Declare Fidelity Per Target

A template pins a target **family**, not a single format, and must
declare its fidelity for each target it claims (`DOC-P2-013`). Three
rules follow:

1. A target the template does not declare is `no_renderer_for_target` —
   never a best-effort attempt.
2. A spec requiring a fidelity the template does not declare for the
   chosen target is `unrepresentable_element`, reported **per element**.
3. Presentation **references `brand-manager` tokens and never redefines
   them.** A template carrying a literal color or font fails validation.
   This is what makes a rebrand a corpus-wide re-render rather than a
   corpus-wide template edit.

## Degradation And Partial Renders

**What a render honored is a property of the chain that actually ran, not
of the target.** The matrix above states the *best available* fidelity.
What a render version records is what its chain could prove — precisely
the read side's rule about anchor kind.

| Situation | Result |
|---|---|
| Full chain runs | Best available fidelity; the render version records the chain and its versions |
| A `desirable` fidelity is unavailable | **Partial render.** Bytes are produced, the missing fidelity is recorded, and the receipt names what was skipped |
| A `required` fidelity is unavailable | No render version. Terminal state, named per the table below |
| An element cannot be represented | `unrepresentable_element` per element, surfaced **before** a switch commits — never a silent drop |

Because the render version carries its chain, re-rendering when a better
renderer arrives is a query rather than a guess: find every render
version produced by chain A and re-run under chain B. That is the write
side of the same property that makes `DOC-P1-003` mechanical.

## When Nothing Can Render It

Six conditions, six remedies, therefore six states. Collapsing them into
one "render failed" is the failure mode this table exists to prevent —
`renderer_unavailable` is a one-command fix and `no_renderer_for_target`
is a roadmap item, and a user cannot tell them apart from a shared
message. This mirrors the read side's five terminal states exactly.

| State | Meaning | Recoverable | What the user is told |
|---|---|---|---|
| `no_renderer_for_target` | No registry entry produces this target | No, without a new renderer | The target, and that support does not exist yet |
| `renderer_unavailable` | A required renderer is declared but not running | **Yes** — start the resource | Which renderer, and how to start it |
| `render_failed` | The renderer ran and could not produce output | Sometimes — retry, or a different chain | That the spec or an asset may be malformed |
| `missing_required_slot` | The template requires a slot the spec does not fill, or a referenced asset does not resolve | **Yes** — fill the slot or fix the reference | Which slot, and which block is missing |
| `unrepresentable_element` | The target cannot express something the spec requires | **Yes** — change target, template, or element | Which element, and which fidelity was needed |
| `blocked_by_policy` | The only viable chain would exceed the document's privacy class | Yes — reclassify, or install a local renderer | Which step was needed and why it was refused |

Every terminal state writes a custody record. A document that failed to
render still has provenance: what was attempted, by which chain, when,
and where it ran. **The spec is never lost to a failed render** — it is
the authority, and the render is the disposable half.

## Anchors Come Free On This Side

The read side's deepest technical risk is that `logical` anchors need
alignment maps no handler chain produces for free, because a parser must
*recover* the mapping from bytes it did not create.

A renderer has the opposite problem, which is to say none: it **placed**
every element, so it already knows the spec-block → output-region
mapping and emits it as a byproduct (`DOC-P2-017`).

| | Parsed document | Generated document |
|---|---|---|
| `logical` anchor durability | Only through a computed alignment | **Unconditional** — the renderer emitted the alignment |
| Cost of that alignment | An algorithm no tier provides | Zero; it is a byproduct |

This is **not a fourth anchor kind.** It is `logical` with an
author-minted stable identity, so it needs no migration and no new
resolver branch. A renderer that cannot emit alignment for some target
produces ordinary `logical` anchors for it and says so per unit — the
same honesty rule as a PDF parsed without its geometry handler.

## The Round-Trip Gate

`render(spec) → ingest → derive → assert the derived model matches the
spec` (`DOC-P2-018`).

This exists because it is a fidelity oracle **available only to a system
that owns both directions**, and it is the difference between "we emit
PPTX" and "we emit PPTX that survives being read back." It should be the
first test written on the write spine, not the last — a renderer whose
output this scenario cannot itself parse is a renderer producing
documents nobody can cite.

## Adding A Target

1. Add the row to `api/internal/render/registry.json`, then mirror it
   here. The registry is the source of truth; the test comparing them is
   what keeps this document honest.
2. Name the fidelities it honors, not a library. If no renderer provides
   a required fidelity, the work is a new renderer, not a new target row.
3. State whether the renderer can emit a block→region alignment. If it
   cannot, say so — generated anchors for that target degrade, and that
   must be visible rather than discovered.
4. Add or extend an operational target and a test carrying its
   `[REQ:DOC-...]` tag, including a round-trip case. A target with no
   round-trip test is a claim, not a capability.

## Adding A Renderer

1. Declare the fidelities it provides. A renderer providing nothing no
   existing renderer provides is not worth adding.
2. Add it to the candidate chains of every target it improves. Existing
   chains keep working; a new renderer is additive.
3. Renderers are replaceable by construction. If adding one requires
   changing routing logic rather than registry data, the router has
   drifted and that is the defect to fix first.
4. Govern it through `scenario-dependency-analyzer` before use, and
   measure its real per-document cost including process spawn — the read
   side already learned that a published in-process figure is not a
   budget.

## Gaps

| Gap | Detail | Owner |
|---|---|---|
| No renderer selected | Nothing in the repo renders `.pptx`, `.docx`, `.xlsx` or PDF. Selection criteria are fidelity coverage and whether the renderer can emit alignment. | Resolve at launch-sequencing step 8 |
| No reusable agent-chat surface | The Composer's chat has no adoptable in-repo component and carries streaming, tool-access, approval and session concerns of its own. | Unassigned; see `../concepts/INTEGRATIONS.md` |
| Spawn cost unmeasured | Inherited from the read side's tier-1 lesson: a published in-process figure excludes spawn and serialization. Renders are larger and less frequent than parses, so the budget shape differs and needs its own number. | Whoever packages the renderer |
| ~~Anchor URI scheme unspecified~~ **Closed 2026-08-07** | Specified in [`anchor-uri.md`](anchor-uri.md) under `DOC-P0-028`. Relevant here because a renderer-emitted alignment lets a generated document's `logical` anchor declare `align=authored` — a hint the resolver still verifies against the derivation record, never trusts from the URI. | Closed |

## Deliberately Out Of Scope

| Capability | Reason |
|---|---|
| Authoring the content | This scenario renders; it does not author. Generation prompts live in skills, per the write-spine decision and `content-desk`'s precedent. |
| Charts, diagrams, images | Owned by `chart-generator`, `graph-studio` and `asset-studio`. They arrive as references in a spec; this scenario embeds them. |
| Brand values | Owned by `brand-manager`. Templates reference tokens and never redefine them. |
| Multi-user concurrent editing | A distinct product with its own concerns, deferred alongside `DOC-P2-009`. Chat editing is single-author. |
| Byte-level editing of rendered output | Structurally forbidden. Editing the artifact diverges it from its authority and voids reproducibility, undo and diff at once. |

## Cross-References

- [`format-matrix.md`](format-matrix.md) — the read-side twin
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the write-spine decision rows
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — `templates`, `composition`, `render`
- [`../concepts/DATA.md`](../concepts/DATA.md) — the authority mirror and spec/render storage
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — generation, switch, refresh and chat-turn flows
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — unresolved write-spine risk
