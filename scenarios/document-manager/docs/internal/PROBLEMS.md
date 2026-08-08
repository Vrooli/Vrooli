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

**Symptom:** All 60 requirements are single-line stubs with one manual
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

**Symptom:** All 60 requirements carry exactly one validation entry of
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

> **RESOLVED 2026-08-07** by an operator-authorized amendment pass. Six
> statements were wrong, not three — a review found `OT-P0-009` and
> `OT-P0-019` still describing a two-value anchor enum after `tabular`
> was added, and `OT-P1-001` still calling ordinary scanned pages tier 3
> after Tesseract was placed at tier 2. All six are corrected in `PRD.md`,
> the matching requirement descriptions are synced so `prd_ref` stays
> honest, and `vrooli scenario requirements validate` passes at L3. Two
> previously unfunded targets were added in the same pass: `OT-P0-027`
> (unsupported terminal states) and `OT-P1-024` (raster image intake).
> Kept rather than deleted because the *shape* of the failure recurs —
> a decision row lands, and the targets it invalidates are not swept.

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

> **RESOLVED 2026-08-07.** `OT-P1-024` / `DOC-P1-024` now fund raster
> image intake, taking the recommended option rather than narrowing the
> anchor decision to PDF. Images route to local OCR at tier 2 alongside
> scanned PDFs, so the marginal cost is close to zero and it does not wait
> on `vision.default`.

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

### 2026-08-07 — search-hub cannot express caller identity, so federation serves only the unrestricted corpus

**Symptom:** `DOC-P0-018` requires the corpus to be discoverable through
federated search. `DOC-P0-024` requires that a unit never surface to a
caller who cannot read its collection or privacy class. Both are P0 and
they cannot both be fully satisfied, because a federated query arrives
with no caller attached.

**Root cause:** Upstream contract shape, not a defect here.
`search-hub/v1/routing.QueryRequest` carries `query`, `types`, `all`,
`limit`, `group`, `explain`, `overrides` and `control_token` — no
principal. The provider side is thinner still: registration is a
descriptor whose `body_template` interpolates only `{{query}}` and
`{{limit}}`, so there is no seam through which an identity could travel
even if the caller had one. `control_token` is not a substitute; it
proves provider ownership for tuning overrides, not caller authorization.

**Workaround:** Per-collection opt-in, decided 2026-08-07 and recorded in
`DECISIONS.md`. A collection carries a `federated` flag, default off,
with a ceiling no flag overrides: confidential and secret units never
federate. Honest and safe, but it is a policy substitute for an identity
the contract cannot express, and the cost is real — **federation cannot
serve the restricted half of the corpus, which is the half the product
exists for.** Direct Connect callers are unaffected.

**Real fix:** A caller principal on `search-hub`'s query contract,
propagated to providers, so this scenario can apply `DOC-P0-024`'s filter
to federated queries the same way it applies it to direct ones. Filed as
an upstream ask. Until then the per-collection flag stands and
`DOC-P0-018` should be read as "discoverable to the extent an anonymous
caller may see," not "fully queryable."

**Owner:** `search-hub` for the contract; this scenario for the flag.

**Refs:** `packages/proto/schemas/search-hub/v1/routing/routing.proto`,
`scenarios/vrooli-memory/.vrooli/search.json` (descriptor shape),
`docs/internal/DECISIONS.md` (federation row).

### 2026-08-07 — The tier-1 speed claim is unmeasured across the Go/Rust boundary

**Symptom:** The free tier's latency story rests on `anydoc`'s ~4.4 ms
median. Nothing has measured what that costs from this scenario.

**Root cause:** Neither `anydoc` nor `pdf-inspector` ships a Go binding —
both are Rust with Node, Python, browser and CLI bindings. The API is Go,
so the call path is a subprocess per handler per document: two spawns for
a text-native PDF, which needs `anydoc` for content and `pdf-inspector`
for the geometry `anydoc` discards. The published figure is in-process
Rust and excludes spawn, argument marshalling and result deserialization.
`format-matrix.md` described this as "In-process Rust via CLI/bindings",
which is true of the library and false of our call path; corrected
2026-08-07.

**Workaround:** None needed yet — no code exists, and the first slice is
tier-2-only, so this blocks nothing at milestone one.

**Real fix:** The resource-packaging work measures real per-document cost
including spawn, and states it in `PERFORMANCE.md` next to the retrieval
budget. If it threatens the latency claim, promote a long-lived handler
process — a resource-shape change, not an architecture change. Both
libraries must also clear `scenario-dependency-analyzer` before use;
neither is in `.vrooli/dependencies/approved-dependencies.json` today,
and both are days old, so treat the maturity risk as live.

**Owner:** whoever packages the tier-1 resources.

**Refs:** `docs/reference/format-matrix.md` (The Handlers),
`docs/internal/DECISIONS.md` (CLI-shaped resources row).

### 2026-08-07 — The anchor URI is the ledger contract and has no specified scheme

> **RESOLVED 2026-08-07** by [`../reference/anchor-uri.md`](../reference/anchor-uri.md),
> funded by `DOC-P0-028` and recorded as four decision rows. Specifying it
> against the real proto rather than the prose corrected two working
> assumptions that had been repeated across three documents: provenance is
> **not a single opaque string** but `ImportProvenance{runtime,
> source_locator, content_hash}`, and the ledger's dedupe key is a
> **byte-exact join** — `runtime + ":" + source_locator + ":" +
> content_hash`, unique on `(scope, import_key)`. That second fact
> promoted canonical form from a style rule to a correctness requirement:
> a trailing zero or an attribute reorder makes two identical citations
> distinct to the ledger, and `DOC-P1-023` silently stops deduplicating.
> Kept rather than deleted because the *shape* of the failure recurs — a
> contract described in prose on both sides and specified on neither, with
> each side assuming the other had defined it.

**Symptom:** `INTEGRATIONS.md`, `DATA.md` and the ledger integration row
all say a publication carries "an anchor URI as provenance". No document
anywhere defines what that string looks like.

**Root cause:** The ledger treats provenance as an **opaque URI it never
dereferences** — deliberately, so it cannot learn what a PDF is. That
design is correct and has a consequence nobody wrote down: the URI string
*is* the entire interface between this scenario and the ledger. It was
described in prose on both sides and specified on neither.

**Workaround:** None needed today. `handoff` is P1 and unscaffolded, so
nothing depends on it yet.

**Real fix:** Specify the scheme — how it names a document, derivation
version, unit and anchor kind, and how it degrades when an anchor
resolves to its minting version — alongside the first consumer, not
after. Two independent things now depend on it: the sources-plus-findings
pattern the sibling pair exists to enable, and the write spine, where a
generated report's citations are anchor URIs pointing back into this
corpus.

**Owner:** this scenario, before `DOC-P1-020` is scheduled.

**Refs:** `docs/concepts/INTEGRATIONS.md` (ledger row, Known Gaps),
`docs/concepts/DATA.md` (Publications, Import/Export).

### 2026-08-07 — The write spine roughly doubles the scenario and must not enter milestone one

**Symptom:** Three new domains (`templates`, `composition`, `render`),
two new UI surfaces, an agent chat, a renderer registry and 18 new
operational targets — added to a scenario whose P0 is already recorded
above as "a program, not a first milestone", with zero domains
implemented and two parse resources unpackaged.

**Root cause:** Not drift. The generation boundary was designed
deliberately and in full, because a boundary improvised later is a
boundary that will be wrong. But designing it in full makes its size
visible, and visible size invites scheduling.

**Workaround:** Everything is P2 and explicitly unscaffolded. The
decision rows, domain map, data ownership, flows, seams and requirements
all exist; no proto, schema, repository, service, handler, mocks or tests
do. This is the same treatment `handoff` received on the same date.

**Real fix:** Open the write spine only at launch-sequencing step 8. The
ordering is not preference: the round-trip fidelity gate needs the read
spine to parse what it renders, generated anchors reuse the read spine's
resolver, and custody and corpus are shared. Building generation earlier
means building it twice. When it is scheduled, split it the way P0 should
be split — one vertical slice (`spec → render one target → ingest back →
round-trip assertion`) before the template registry, the switch, or the
chat.

**Owner:** whoever authors the implementation plan.

**Refs:** `PRD.md` (P2 generation spine, Launch sequencing),
`docs/internal/DECISIONS.md` (write-spine rows), the "P0 as written is a
program" entry above.

### 2026-08-07 — No render toolchain exists anywhere in the repo

**Symptom:** Nothing produces `.pptx`, `.docx`, `.xlsx` or PDF. A search
across `scenarios/`, `docs/` and `resources/` for pandoc, typst,
gotenberg, docxtemplater, unioffice, python-pptx, reportlab, weasyprint,
headless-Chrome PDF, marp and reveal.js returned no toolchain — only
unrelated build artifacts and log noise.

**Root cause:** Generation was never anyone's charter. `chart-generator`
owns charts, `graph-studio` diagrams, `asset-studio` media,
`content-desk` copy, `brand-manager` tokens — and the container layer
that assembles them into a file fell between all of them.

**Workaround:** None. The write spine cannot start without this
selection, which is exactly why it is P2 rather than optimistically P1.

**Real fix:** Select and package a renderer as a resource, governed
through `scenario-dependency-analyzer` like the parse resources. Two
selection criteria are non-negotiable and are easy to discover too late:
**fidelity coverage** (which of `paged-geometry`, `cell-structure`,
`speaker-notes`, `vector-embed`, `styled-text` it can honor) and
**whether it can emit a block→region alignment** — a renderer that
cannot is a renderer whose generated documents lose their durable-anchor
guarantee, which is one of the two strongest reasons generation lives in
this scenario at all. Measure real per-document cost including spawn;
the read side already learned that a published in-process figure is not
a budget.

**Owner:** whoever opens the write spine.

**Refs:** `docs/reference/render-matrix.md` (Gaps),
`docs/concepts/INTEGRATIONS.md` (render toolchain row).

### 2026-08-07 — Template-agnostic specs are an authoring discipline nothing enforces

**Symptom:** `DOC-P2-014` promises an existing spec re-renders under a
different template. That holds only if the spec never encoded
presentation — and nothing structurally prevents a spec block from
saying "two-column, image right".

**Root cause:** The rule ("the spec declares content and intent; the
template decides presentation") is a contract about *what authors write*,
not about what the schema permits. A spec schema permissive enough to be
useful is permissive enough to be abused, and the abuse is invisible
until the first switch fails.

**Workaround:** None yet — nothing is built. Recorded before it can bite,
because the failure surfaces long after the authoring mistake.

**Real fix:** Make the spec schema refuse presentation vocabulary
outright rather than relying on discipline, and treat a switch failure
across two well-formed templates as a **spec defect** rather than a
template gap. The related hazard is overrides: every per-document
override is a bet against future template changes, so `DOC-P2-015`'s
enumerability requirement is load-bearing, not reporting polish. A
corpus where most documents carry overrides has lost switchability
without anyone deciding to give it up.

**Owner:** whoever designs the spec schema.

**Refs:** `docs/internal/DECISIONS.md` (template-switching row),
`docs/reference/render-matrix.md` (Templates Declare Fidelity Per Target).

### 2026-08-07 — Templates stored as corpus documents is the deliberately clever choice

**Symptom:** Not yet observable. Templates are declared as corpus
documents under a distinguished kind so they inherit versioning, custody,
export, diff and access control for free.

**Root cause:** It is genuinely the right trade today — the alternative
reimplements five mechanisms this scenario already has — but it makes a
template subject to a schema designed for *received* material. Documents
carry a privacy class, parse confidence, OCR provenance, page geometry
and a derivation chain. A template has none of those, and the fit is
comfortable only while nobody asks it to be.

**Workaround:** None needed. Recorded as a watched risk rather than a
settled comfort, so that whoever hits the friction recognises it as the
predicted failure rather than a puzzle.

**Real fix:** If template kinds start needing document fields that make
no sense, or start needing document fields *suppressed* in more than one
or two places, split templates into their own store. That is a
mechanical migration while templates are few, and an expensive one after
a corpus of them exists — so the decision point is early, not when the
pain is large.

**Owner:** whoever builds `templates`.

**Refs:** `docs/internal/DECISIONS.md` (templates-as-documents row),
`docs/concepts/DATA.md` (Templates rows).

### 2026-08-07 — The Composer's agent chat is an unowned build, not a panel

**Symptom:** `DOC-P2-021` through `DOC-P2-025` describe an in-UI agent
that edits documents conversationally. A search for a reusable or
embeddable chat surface in this repo found nothing adoptable.

**Root cause:** Chat surfaces look like a component and are a feature
area: streaming responses, tool access, approval gates on destructive
edits, session and turn state, stale-completion handling, and undo
semantics. Treating it as "add a panel" is how it gets underestimated.

**Workaround:** None. Flagged so it is scoped as a build rather than
discovered mid-implementation.

**Real fix:** At launch-sequencing step 8, either find an existing
scenario willing to own a reusable chat surface, or build one here and
publish it through `react-component-library` so the next scenario that
needs one does not repeat this. Either way the parity constraint holds:
the chat constructs only generated clients (`DOC-P2-021`), so whatever
is built is a *client* and cannot acquire a privileged path by
convenience.

**Owner:** unassigned.

**Refs:** `docs/concepts/INTEGRATIONS.md` (Known Gaps),
`docs/internal/SEAMS.md` (Composer chat client boundary).

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
