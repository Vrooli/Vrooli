## Steer focus: Domain Clarity

Prioritize making the **domain model of `scenarios/{{TARGET}}/` small, consistently named, and explicit about intent** — so any agent can answer "what are the core concepts, what is each called, and why does each exist?" from the code and its `docs/concepts/` alone. This skill governs the conceptual layer: which concepts exist, what they are named, and whether their purpose is visible. It does not move files, own lifecycle, or polish local function shape — those hand off (see §1).

Do not change observable behavior, regress tests, or add product features. Every change must maintain or improve completeness and test health.

### Required reading

- `prompt-manager skill read knowledge-observatory-tools` — read and update `docs/concepts/` through the canonical docs CLI.
- `prompt-manager skill read screaming-architecture-audit` — structural/folder shape; the audit this skill hands structural findings to.
- `prompt-manager skill read cognitive-load-reduction` — local naming and function shape; the audit this skill hands local-readability findings to.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — current domain model (entities, actions, states).
- `scenarios/{{TARGET}}/docs/concepts/GLOSSARY.md` — current canonical vocabulary.
- The scenario's PRD, operational targets, and technical requirements — the authoritative domain language.

---

### 1. Scope Boundaries

This skill owns the **conceptual model**, not its physical layout or its local code-craft. Three lenses, one boundary.

**In scope:**
- **Concept compression** — discover the smallest, clearest set of domain concepts; collapse conceptual duplicates (multiple names for one idea, near-duplicate shapes, parallel flows for the same journey) toward one canonical concept per idea.
- **Vocabulary unification** — one primary term per concept across code, tests, and UI; split overloaded names so each concept gets a distinct term; align terms with the PRD's domain language.
- **Intent documentation** — make purpose visible: names that answer "what is this for / when does it change", and `why`-comments where a non-obvious decision, invariant, or constraint would otherwise be lost.

**Out of scope (hand off):**

| Finding | Hand off to |
|---|---|
| Folders organized by tech bucket instead of domain; files in the wrong domain folder | `screaming-architecture-audit` |
| Lifecycle, retry, cancellation, state-machine ownership complexity | `temporal-flow-audit` |
| Local naming of variables/params, function length/nesting, comment-restates-code | `cognitive-load-reduction` |
| Whether an abstraction earns its keep across domains | `seam-discovery-and-enforcement` |
| Public API surface, transport, contracts | `api-steer` |
| Dead code, stale config, abandoned files | `code-cleanup` |

The distinction from `cognitive-load-reduction`: that skill asks "is this *name* clear and this *function* shaped well?"; this skill asks "is this the *right concept*, is it *named consistently everywhere*, and is its *purpose* visible?" Rename a local loop variable → cog-load. Reconcile that `job`, `run`, and `task` are three names for one concept → here.

---

### 2. Domain Clarity Maturity Model

Assess the scenario against this ladder; each level is gated by a verifiable artifact, not an adjective. Use it to pick the next concrete move, not as a score to inflate.

| Level | Name | What exists | When to stop here |
|---|---|---|---|
| 0 | Untracked | No `docs/concepts/ARCHITECTURE.md`; domain model lives only in code and agent memory. | Never — write the minimal domain model first. |
| 1 | Documented | `ARCHITECTURE.md` names the core entities, primary actions, and key states, verified against code. | The domain is small and the doc is accurate; no duplication or naming drift found. |
| 2 | Compressed | Conceptual duplicates are collapsed: one canonical shape per core concept, one primary flow per journey. `ARCHITECTURE.md` records where duplicates were merged. | `grep` finds no parallel near-duplicate types/flows for the same concept; distinctions that remain are PRD-justified. |
| 3 | Unified vocabulary | One primary term per concept across code, tests, and UI; overloaded names split. `GLOSSARY.md` records canonical terms and old→new mappings. | `grep` for each retired synonym returns only historical/adapter mentions; boundary layers use domain terms, not implementation jargon. |
| 4 | Intent explicit | Names express purpose; non-obvious logic carries `why`-comments referencing the invariant or decision. Names, comments, and tests agree on the same intent. | A new agent can state why each core concept exists without reading call sites. |

A scenario may sit at different levels for different surfaces; advance the weakest core concept first. Do not claim a level whose artifact (`ARCHITECTURE.md` / `GLOSSARY.md` entry, or a clean `grep`) does not exist.

---

### 3. Lens-Routing Table

Classify each finding into one lens and act deterministically. Two agents reading the same scenario must land in the same row.

| Signal | Lens | Action | Durable anchor |
|---|---|---|---|
| Two+ names for the same idea; near-duplicate types/shapes; parallel flows for one journey | Concept compression | Choose one canonical concept; align code where clearly safe and PRD-consistent | `ARCHITECTURE.md` (what was merged, core vs edge) |
| Same concept, different terms across code/tests/UI; one name used for two concepts | Vocabulary unification | Choose one primary term; split overloaded names; update uses cohesively | `GLOSSARY.md` (canonical terms, old→new) |
| Generic name (`data`, `manager`, `doStuff`) where a domain term exists; purpose not visible | Intent documentation | Rename to purpose; add a `why`-comment only where intent is non-obvious | Code + `ARCHITECTURE.md` |
| Structural / lifecycle / local-craft (see §1 table) | — | Stop and hand off | The owning audit's doc |

Rules: only remove code when it is provably unused or superseded and tests still pass meaningfully. Prefer localized, safe changes over repo-wide renames in one step. Avoid over-compression — do not merge genuinely distinct concepts (materially different behavior, or the PRD keeps them separate) just to cut concept count. If a larger merge is beneficial but risky, record it in `ARCHITECTURE.md` rather than partially applying it.

---

### 4. Anti-Gaming

Real domain clarity changes which concepts exist and what they are called throughout, not surface symbols. Avoid: renaming purely stylistic local symbols, reformatting, shuffling files, or adding comments that restate the signature. A change counts only when it removes a conceptual duplicate, retires a synonym across the codebase, or makes a previously hidden purpose visible.

---

### 5. Output Expectations

You may update naming of domain types/entities/operations, canonical data shapes (where safe), core-journey flow structure, UI labels, tests that misrepresent intent, and `docs/concepts/` docs. You must preserve observable behavior and user workflows, keep public contracts stable where consumed externally (update all in-scenario call sites when you must change them), and maintain or improve test coverage.

Update `docs/concepts/ARCHITECTURE.md` and `docs/concepts/GLOSSARY.md` (user-facing) via `knowledge-observatory-tools`. The code is the source of truth: verify existing claims against code before extending, correct inaccuracies, and record new discoveries. Do not create standalone `*_AUDIT.md` files — findings route to these existing docs.

By the end of a loop the scenario should have advanced at least one core concept on the §2 ladder: a duplicate collapsed, a synonym retired across the codebase, or a hidden purpose made explicit — with the matching `docs/concepts/` entry updated.
