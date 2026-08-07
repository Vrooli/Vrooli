# Problems — Document Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear, newest at the bottom.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-06 — BAS experience-spec cases were removed, not replaced

**Symptom:** `bas/registry.json` lists one playbook (the routed-database
proof). The three generated experience-spec smokes are gone.

**Root cause:** They were generated from the old experience index
(`dashboard`, `notes`, `settings`). That index now declares the three
real surfaces — `corpus`, `reader`, `receipt` — so every case carried a
`spec_entry_id` pointing at a spec entry that no longer exists. Writing
replacements now is not possible either: the cases assert on
`@selector/pages.*` entries in `ui/src/consts/selectors.manifest.json`,
and no Corpus, Reader or Receipt page exists in the UI yet.

**Workaround:** None needed. BAS coverage of scaffold pages that are
about to be replaced has no value, and dangling spec references would
have read as coverage that does not exist.

**Real fix:** When each page lands in the UI, add its testid to the
selector manifest, author the experience-spec case against it, and run
`test-genie registry build --scenario document-manager` from
`scenarios/`. The claims in `experience/pages/*.json` name exactly what
each case has to assert — the `machine`-tier ones are the automatable
set.

**Owner:** whoever builds the first UI surface.

**Refs:** `experience/index.json`, `experience/pages/{corpus,reader,receipt}.json`,
`bas/registry.json`.

### 2026-08-06 — `template-manager detemplate` has not been run

**Symptom:** The `notes` example domain is still live in
`api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`,
`ui/src/features/notes/` and
`packages/proto/schemas/document-manager/v1/notes/`. Docs carry it inside
`<!-- EXAMPLE-DOMAIN:notes -->` fences.

**Root cause:** Not a defect — the fenced form is the template's designed
removable state, and detemplate is a deliberate step that has not been
taken yet.

**Workaround:** The fences make it unambiguous which content is example
scaffolding, so it does not confuse the domain map.

**Real fix:** Run `template-manager detemplate document-manager` — but
**not yet, and the ordering matters**. `DOMAINS.md` states the example is
removed "once the real domains are green," and the README lists `notes`
as the worked vertical slice to copy. It is the reference implementation
for the proto → API → CLI → UI shape, and no real domain has been built
from it yet: nine product domains are documented and none has a proto
package. Removing it now would delete the only working example and leave
the scenario with zero domains.

Correct sequence: build the first real domain (`intake` is the natural
first, since everything downstream needs it) by copying the `notes`
shape, get it green, then detemplate. The `example-domain-removed`
orientation gate is the last one for a reason.

**Owner:** whoever builds the first domain.

**Refs:** `docs/concepts/DOMAINS.md` (fenced example section), `README.md`
("Remove" list), `.vrooli/orientation.json` (`example-domain-removed`).

### 2026-08-06 — Three P0 requirements hide multi-phase complexity behind one line

**Symptom:** All 57 requirements are single-line stubs with one manual
validation entry each. `business-health validate` passes, which makes the
set look uniformly sized. It is not.

**Root cause:** The requirement modules mirror the PRD one-to-one rather
than elaborating it. Three P0s carry substantially more work than their
neighbours:

- `DOC-P0-009` / `DOC-P0-019` — anchor durability across re-derivation.
  Two anchor kinds, an alignment-map algorithm no parse tier produces for
  free, and a resolver that must never read a pruned parse output.
- `DOC-P0-006` — one normalized model across three tiers whose native
  outputs differ structurally, not just in quality.
- `DOC-P0-013` / `DOC-P0-026` — fail-closed routing, whose guarantee
  lives in a class→profile mapping enforced by an AST check rather than
  by anything upstream.

**Workaround:** They are flagged here so phase planning does not treat
them as peers of, say, `DOC-P0-003`.

**Real fix:** Decompose these into sub-requirements during
implementation-plan authoring, before they are assigned to phases.

**Owner:** whoever authors the implementation plan.

**Refs:** `requirements/01-must-ship/module.json`,
`docs/internal/DECISIONS.md` (anchor-kinds decision).

### 2026-08-06 — The custody journal can be pruned by a storage kind it must not carry

**Symptom:** Not yet observable — no code exists. Recorded before it can
bite.

**Root cause:** `storage-manager`'s enforcer prunes any `kind=dir` entry
that carries a budget, regardless of append-only intent.
`vrooli-memory`'s append-only journal declared `max_age 365d` and became
prune-eligible exactly this way. `DOC-P0-015` requires the custody trail
to outlive the document, and `DOC-P0-016` requires every artifact-store
kind to be declared with retention — those two pull in opposite
directions if the custody kind is declared naively.

**Workaround:** Declare the custody kind without a prunable budget, and
verify enforcement behavior against a real prune run rather than
trusting the flag.

**Real fix:** Either `storage-manager` grows a retention mode that
respects append-only intent, or this scenario keeps custody records out
of any budgeted kind permanently. The long-horizon retention policy for
the journal is separately undecided and is a policy question, not an
engineering one.

**Owner:** unassigned; overlaps `storage-manager`.

**Refs:** `docs/concepts/DATA.md` (Retention And Deletion),
`docs/internal/DECISIONS.md` (custody journal decision).

### 2026-08-06 — Every requirement validation is a `manual` stub, so the registry gates nothing

**Symptom:** All 57 requirements carry exactly one validation entry of
`"type": "manual"`, `"status": "planned"`. `business-health validate`
passes. Read quickly, the registry looks like a working quality gate.

**Root cause:** Scaffolding, not a defect in itself — the stubs were
generated with the modules. But a manual validation is not executable,
so nothing in the requirements registry can currently fail. This is the
maturity-ladder-starts-at-max shape: a capability that measures nothing
reports healthy, and the reported number goes *down* once real
validations land and start failing honestly. Expect that dip and do not
treat it as a regression.

**Workaround:** Do not read requirement status as evidence of anything
until validations are test-typed. `PROGRESS.md` and actual test runs are
the real signal.

**Real fix:** As each requirement is implemented, replace its stub with a
test-typed validation carrying a `[REQ:DOC-P0-NNN]` tag and a ref to the
covering test. The implementation plan should treat "validation
converted" as part of a requirement's definition of done, not as
follow-up work.

**Owner:** whoever authors the implementation plan.

**Refs:** `requirements/01-must-ship/module.json` and siblings.

### 2026-08-06 — P0 as written is a program, not a first milestone

**Symptom:** 26 P0 targets spanning nine product domains, with zero
domains implemented and no proto package for any of them. There is no
intermediate point at which the scenario is demonstrably working.

**Root cause:** The P0 set was derived from "what does viability mean for
the finished product," which is the right question for a PRD and the
wrong granularity for a first phase. Two of the earliest targets in the
pipeline are also externally blocked: `DOC-P0-004` needs a pdf-inspector
resource that is not packaged, and `DOC-P0-005` needs `unstructured-io`
verified. Nine domains each need proto, schema, repository, service,
handler, mocks, tests, CLI and UI.

**Workaround:** None yet — this bites at plan-authoring time, not now.

**Real fix:** Split P0 into a first vertical slice and a completion set
before phases are assigned. The natural slice is the spine that proves
the product claim end to end on one file type:
`intake → sensitivity → derivation (tier 2 only) → anchors → custody`,
with `DOC-P0-026`'s choke point and `DOC-P0-013`'s fail-closed test in
it. That reaches a demonstrable "a document was processed, provably
locally, and a passage is citable" without waiting on the unpackaged
tier-1 resource, without retrieval, and without the Reader. Everything
else in P0 then has something real to attach to. Note this is a
*sequencing* split for planning, not a re-prioritisation of the PRD —
the viability bar itself does not move.

**Owner:** whoever authors the implementation plan.

**Refs:** `PRD.md` (Operational Targets, Launch sequencing),
`docs/concepts/INTEGRATIONS.md` (Vrooli Resources).

### 2026-08-06 — PRD format wording predates `anydoc` and is read-only

**Symptom:** Three statements in `PRD.md` are now wrong, and `PRD.md` is
read-only after the orientation gate (`docs/START-HERE.md`), so they
cannot simply be edited here.

- `OT-P0-005` reads "DOCX, HTML, EPUB, Markdown and plain text parsed
  through the structural resource." The format list is too narrow — no
  presentations, spreadsheets, OpenDocument, RTF, CSV, XML or email — and
  "the structural resource" is singular where there are now several
  declared handlers selected by capability. HTML is correct as a P0
  format and is handled by `unstructured-io`.
- `OT-P0-008` reads "Every unit carries document hash, page and character
  range." This presumes every source has pages. Spreadsheets, CSV, HTML
  and reflowable EPUB do not, and `tabular` anchors carry a cell range
  rather than a character range.
- The Tech Direction snapshot and Dependencies section name "a new
  pdf-inspector resource for tier 1" and `unstructured-io` for tier 2.
  Both are superseded by the `anydoc` decisions.

**Root cause:** `anydoc` was open-sourced 2026-08-04, two days after the
PRD was written. Not an authoring defect.

**Workaround:** [`../reference/format-matrix.md`](../reference/format-matrix.md)
is authoritative for formats and anchor kinds; the PRD is not. Where the
two disagree, the matrix wins and this entry records why.

**Real fix:** Amend the three PRD statements in the next PRD-amendment
cycle, and update the `DOC-P0-005` and `DOC-P0-008` descriptions in
`requirements/01-must-ship/module.json` in the same pass so `prd_ref`
stays honest. Suggested wording — `OT-P0-005`: "Word, presentations,
spreadsheets, OpenDocument, RTF, EPUB, CSV, Markdown and plain text
parsed locally through the declared handler chain for its format."
`OT-P0-008`: "Every unit carries a document hash plus the coordinates
its handler chain could prove — page and bounding box, sheet and cell
range, or structural path and character offset." The Tech Direction
snapshot should name the handler registry as the format source of truth
rather than naming parsers.

A further target is missing entirely: nothing covers the unsupported
terminal states (`no_handler_for_format`, `handler_unavailable`,
`handler_failed`, `blocked_by_policy`, `unsupported_variant`), and
those are a user-facing contract, not an implementation detail. It
belongs in P0 — a corpus that silently drops the files it cannot parse
fails the custody claim.

**Owner:** whoever authors the implementation plan.

**Refs:** `PRD.md` (Operational Targets, Tech Direction Snapshot,
Dependencies), `docs/reference/format-matrix.md`,
`docs/internal/DECISIONS.md` (the three 2026-08-06 `anydoc` rows).

### 2026-08-06 — Image intake is assumed by the anchor design but funded by no target

**Symptom:** `DECISIONS.md` states geometric anchors are available for
"PDF and image sources", and the format matrix lists raster images at
tier 3. No operational target ingests an image.

**Root cause:** The anchor-kinds decision reasoned about which sources
have fixed geometry and correctly included images; the target list was
derived from a document-centric brief that never mentioned them.

**Workaround:** None needed yet — nothing is built.

**Real fix:** Add a P1 target for raster image intake, or narrow the
anchor decision's wording to PDF only. The first is better: images share
tier 3 with scanned PDFs and need no separate parse path, so the marginal
cost is close to zero once `vision.default` lands.

**Owner:** whoever authors the implementation plan.

**Refs:** `docs/reference/format-matrix.md` (Gaps),
`docs/internal/DECISIONS.md` (anchor-kinds row).

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
