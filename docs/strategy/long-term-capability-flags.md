# Long-Term Capability Flags

A lightweight, append-only log for **capability classes that aren't on the roadmap and aren't manifesto-grade** but are worth preserving for when prerequisite substrate matures.

## What goes here

Capability classes that satisfy all of:
- **Not currently actionable** — substrate doesn't exist yet, or capability is too speculative for near-term roadmap inclusion
- **Worth not-forgetting** — could become viable when prerequisite scenarios / hardware / external substrate matures, or could shape long-term direction-setting
- **Lighter than `VISION.md`** — these are speculative direction-flags, not philosophical claims about Vrooli's nature
- **Lighter than `docs/director-swarm/strategy/ROADMAP.md`** — not yet promoted to active initiative-portfolio consideration

If a flag matures (substrate becomes viable, audience interest emerges, or it crystallizes into an actionable scope), promote it to the appropriate active layer:
- A new initiative in `docs/director-swarm/strategy/ROADMAP.md` if it's actionable now
- A new bundle / SKU candidate in `docs/monetization/catalogs/CATALOG.md` if it's monetization-shaped
- A new section in `VISION.md` if it's manifesto-grade direction (rare)
- Retire the flag from this file once promoted

## What does NOT go here

- Active initiatives (those go in `docs/director-swarm/strategy/ROADMAP.md`)
- Backlog items (those go in `swarm-manager`)
- Operator-curated philosophical claims (those go in `VISION.md`)
- Marketing positioning (that goes in `path:docs/marketing/`)
- Specific scenario candidates that just aren't built yet (those go in swarm-manager as `kind=idea` backlog items)

The bar for inclusion is "this is a *class* of capability that requires substrate Vrooli doesn't have yet, but if/when that substrate exists, the class would be worth pursuing." Specific actionable scenarios go to swarm-manager; philosophical commitments go to VISION.

## Posture

- **Append-only.** Add new flags as they surface; don't restructure or rewrite existing ones.
- **Operator-curated.** Agents may propose new entries via decisions (any team that surfaces a long-term capability flag); operator approves wording before append.
- **Retire on promotion or obsolescence.** When a flag becomes an initiative, gets superseded by a different direction, or stops being relevant, retire it with a one-line note about why.

---

## Flags

### 2026-04-27 — Prompt-to-physical-product capability class

**Surfaced by:** operator (vision walk #4 chore-audit), via a bookmark of someone who used a "prompt-to-product" website to design their own Kindle (the website took a prompt and returned hardware specs + parts list for self-assembly).

**The class:** software that takes a natural-language prompt and produces a physical-product design or production-ready specification. Variants include:
- **Prompt → 3D model → 3D-printer output.** Combine LLM reasoning with CAD-generation (e.g., OpenSCAD scripting, parametric model generation) plus a 3D-printer resource (e.g., wrapping OctoPrint per the wrap-not-use principle). User describes a part / fixture / decor item; system generates the model and prints it.
- **Prompt → furniture / decor design.** Layout, dimensions, materials list, optional cut-list for specific tools (CNC, table saw). User describes the piece; system produces buildable plans.
- **Prompt → electronics product spec.** Hardware parts list, schematic, firmware skeleton (the original Kindle-design example). User describes the device; system produces an assembly guide and source list.
- **Prompt → home / household item.** Combinations of the above, scoped to common household needs (organizers, mounts, custom containers, etc.).

**Why flag now, not pursue now:** None of these are tractable today. They require:
- Mature CAD-generation tooling integrated into an LLM-controlled scenario (not yet built)
- Hardware-side resources (3D printer, eventually CNC and other production substrates) integrated into Vrooli's resource layer (not yet planned beyond placeholders)
- A scenario template for prompt→model→production-output flows (not yet exists)

But the *direction* is consistent with Vrooli's ambition (extending agentic automation from the digital into the physical world per VISION.md Phase 3-4) and the *audience signal* (people are visibly excited about prompt-to-product on social media) suggests it would land well when the substrate matures. Worth not-forgetting.

**Prerequisite substrate to watch:**
- 3D-printer resource integration (likely starting with OctoPrint wrapper per wrap-not-use principle)
- CAD-model generation skill / scenario
- Parts-sourcing scenario (could leverage existing scraping / aggregation patterns)
- Robotics integration broadly (touched in `VISION.md` Phase 3+ as "engineering servers")

**Promotion trigger:** when 2+ prerequisite substrate items above ship, re-evaluate this flag for promotion to an actionable initiative.

**Source bookmark:** social-media post about a prompt-to-product website that returned a Kindle design with hardware specs + parts list. URL not captured (pre-BIH-rework workflow).

**Cross-references:**
- `VISION.md` Phase 3 ("Domain Specialization — Engineering Servers") — adjacent direction that this class would fit under once substrate matures.
- Initiative `bookmark-intelligence-hub-rework-and-ideation` — when shipped, the BIH ideation-extraction agent should surface bookmarks classified as `capability-class-flag` for review against this file. New flags may append; promotion candidates may be flagged for operator review.

---

*(More flags will be appended as they surface in future vision walks or operator-flagged signals. The file grows over time; entries may be retired as they're promoted to active layers or obsoleted.)*
