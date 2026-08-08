# Format Matrix — Document Manager

What this scenario accepts, which handler chain parses it, what that
chain can prove about position, and what happens when nothing can parse
it at all.

This is the **read** side. Its twin is
[`render-matrix.md`](render-matrix.md), which answers what this scenario
can *produce*. The two share one normalized document model and the same
routing philosophy — handlers declare capabilities, renderers declare
fidelity — and a generated document goes back through *this* matrix on
ingest, so a target that renders something this matrix cannot parse is a
defect on both sides.

## Purpose Of This Document

Use this document to answer:

- Which file formats does intake accept, and at what priority?
- Which handler chain parses a given format, and at what cost tier?
- What anchor kind can a unit carry — and what does it degrade to when
  part of the chain is unavailable?
- What does a user see when no handler exists?
- What has to happen to add a format or a handler?

Narrative rationale belongs in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md). Anchor storage
belongs in [`../concepts/DATA.md`](../concepts/DATA.md). The runtime
routing sequence belongs in [`../concepts/FLOWS.md`](../concepts/FLOWS.md).

## The Registry Is The Source Of Truth

Format support is **declared data, not inferred behavior**. The handler
registry at `api/internal/derivation/registry.json` is the single source
of truth; this document is its human-readable projection, the same way
`reference/api-endpoints.md` mirrors `.vrooli/endpoints.json`.

A test asserts the two agree. A format present in one and absent from
the other fails the build. Nothing supports a format by accident, and
nothing silently stops supporting one.

The registry exists so that no library is load-bearing by default. Every
handler is a declared, replaceable participant — including `anydoc`.
Leveraging a good library means routing to it, not building the scenario
inside it.

## Handlers Declare Capabilities, Not Just Formats

The routing unit is not "format X uses parser Y." A single format
routinely needs more than one handler, because handlers provide
*different things*. A text-native PDF needs `anydoc` for clean content
and `pdf-inspector` for the page geometry `anydoc` discards.

So each handler declares the capabilities it provides:

| Capability | Meaning |
|---|---|
| `content` | Normalized text and structure — the document model |
| `geometry` | Page index and bounding boxes in source coordinate space |
| `tables` | Cell-addressable table structure rather than flattened prose |
| `elements` | Typed, hierarchical elements (title, body, list, header, table) |
| `ocr` | Text recovery from rasterized or scanned content |

A format declares which capabilities it *requires* and which are
*desirable*. The router selects the cheapest chain satisfying the
required set, then adds desirable capabilities from handlers whose tier
the sensitivity gate permits. This is why `anydoc` losing geometry is a
routing fact rather than a crisis: geometry is a capability sourced from
whichever handler has it.

## The Handlers

| Handler | Tier | Provides | Runtime | Formats |
|---|---|---|---|---|
| `anydoc` | 1 | `content`, `tables` | Rust, invoked as a **subprocess** through a CLI-shaped resource — there is no Go binding. No service, no ML weights. | 14 document formats (see matrix) |
| `pdf-inspector` | 1 | `geometry`, page classification | Rust, same subprocess model. Embedded inside `anydoc` but its geometry is not surfaced there, so it is invoked separately. | PDF only |
| `native-text` | 1 | `content` | In-process Go | `.txt`, `.md`, `.ipynb` |
| `unstructured-io` | 2 | `content`, `geometry`, `tables`, `elements`, `ocr` | Docker service; Tesseract for OCR, layout model for `hi_res` | 24+ formats including everything `anydoc` misses |
| `gateway-vision` | 3 | `content`, `geometry`, `ocr` | AI Gateway `vision.default` role — **metered** | Any rasterizable source |

**Tier is a cost class, not a sequence.** Tier 1 is instant and local.
Tier 2 is local and free but slow, and may require a running service.
Tier 3 is inference, is metered, and is the only tier the sensitivity
gate can refuse on residency grounds.

Tesseract OCR inside `unstructured-io` is **tier 2, not tier 3** — it is
a resource-owned local runtime, not a provider call, so it costs nothing
and routes nowhere. That placement has a consequence worth stating
plainly: **a confidential scanned document has a working local path.**
Under a model where all OCR was tier 3, such a document would be refused
by the sensitivity gate and never processable at all.

## The Matrix

| Family | Extensions | Required chain | Desirable | Anchor kind | Priority |
|---|---|---|---|---|---|
| PDF, text-native | `.pdf` | `pdf-inspector` + `anydoc` | — | `geometric` | P0 |
| PDF, scanned | `.pdf` | `unstructured-io` (`ocr_only`/`hi_res`) | `gateway-vision` | `geometric` | P1 |
| Word | `.doc` `.docx` `.docm` | `anydoc` | `unstructured-io` (`elements`) | `logical` | P0 |
| OpenDocument text | `.odt` | `anydoc` | `unstructured-io` | `logical` | P0 |
| Rich text | `.rtf` | `anydoc` | `unstructured-io` | `logical` | P0 |
| EPUB | `.epub` | `anydoc` | `unstructured-io` | `logical`, spine-scoped | P0 |
| Presentations | `.ppt` `.pps` `.pot` `.pptx` `.pptm` `.ppsx` `.ppsm` `.odp` | `anydoc` | `unstructured-io` | `logical`, slide-scoped | P0 |
| Spreadsheets | `.xls` `.xlsx` `.xlsm` `.xlsb` `.ods` | `anydoc` | — | `tabular` | P0 |
| Delimited text | `.csv` `.tsv` | `anydoc` (csv), `unstructured-io` (tsv) | — | `tabular` | P0 |
| Plain text, Markdown | `.txt` `.md` | `native-text` | — | `logical` | P0 |
| HTML, XML | `.html` `.htm` `.xml` | `unstructured-io` | — | `logical`, DOM-path-scoped | P0 |
| Raster images | `.png` `.jpg` `.jpeg` `.tiff` `.bmp` `.heic` `.webp` | `unstructured-io` (`ocr`) | `gateway-vision` | `geometric` | P1 |
| Email | `.eml` `.msg` | `unstructured-io` | — | `logical`, part-scoped | P1 |
| Notebooks | `.ipynb` | `native-text` | — | `logical`, cell-scoped | P2 |
| Markup text | `.rst`, `.org` | `unstructured-io` | — | `logical` | P2 |
| Archives | `.zip` `.tar.gz` | container — expand, route each member | — | Inherited from member | P2 |

## Why Both Parsers, And What Each Is For

`anydoc` is not a replacement for `unstructured-io`. They fail in
opposite directions, and the registry exists so both can be used where
each is strong.

| | `anydoc` | `unstructured-io` |
|---|---|---|
| Speed | ~4.4 ms median | Seconds; `hi_res` runs a layout model |
| Setup | Zero — no service, no Python, no ML weights | Docker service |
| Format breadth | 14 document formats | 24+, including HTML, XML, email, images, RST, Org, code |
| Position metadata | **None** — no pages, offsets or boxes | Page numbers and coordinates |
| Element typing | Flat document model | Typed hierarchy (title, body, list, table, header) |
| OCR | **None** — image-only PDFs return unsupported | Tesseract, 100+ languages |
| Tables | Yes | Yes, plus `text_as_html` |

Read as one sentence: **`anydoc` wins the common path, `unstructured-io`
wins the long tail and owns position.** The common path is most documents
most of the time, which is why `anydoc` is tier 1 and why the free tier
stays fast. The long tail is HTML, email, images and anything scanned —
none of which `anydoc` touches at all.

`unstructured-io` remains a declared dependency and still needs its
docker-service migration verified. It is not optional, and its absence
degrades specific formats in specific ways rather than being invisible.

**One caveat on the speed comparison.** `anydoc`'s ~4.4 ms is measured
in-process in Rust. Neither it nor `pdf-inspector` ships a Go binding, so
our call path is a subprocess per handler per document — two spawns for a
text-native PDF, which needs `anydoc` for content and `pdf-inspector` for
geometry. Process spawn and serialization sit on top of the published
figure and are not yet measured. The resource-packaging work owns that
measurement, and a long-lived handler process is the escape hatch if the
free tier's latency claim comes to depend on it. Treat the published
number as a floor, not a budget.

## Degradation And Anchor Kind

**Anchor kind is a property of the chain that actually ran, not of the
format.** The matrix column above states the *best available* kind. What
a unit records is what its chain could prove.

| Situation | Result |
|---|---|
| Full chain runs | Best available anchor kind; derivation records the chain and its versions |
| A `desirable` handler is unavailable | Partial success. Content is produced, the missing capability is recorded, and the receipt names what was skipped |
| PDF geometry source unavailable | The PDF yields `logical` anchors instead of `geometric`. Real degradation, recorded per unit — not silent |
| A `required` handler is unavailable | No derivation. Terminal state `handler_unavailable`, which is recoverable |

Because the derivation record carries its chain and version,
re-derivation when a better handler arrives is a query rather than a
guess: find every derivation produced by chain A and re-run it under
chain B. That is what makes `DOC-P1-003` mechanical, and it is why the
chain must be recorded even when everything succeeds.

## When Nothing Can Parse It

An unsupported file is a **terminal state on a stored document**, never a
rejected upload and never a silent absence. The bytes are kept, the
custody trail records the attempt, and the document appears in the Corpus
with an explicit state. A user who uploaded fifty files must be able to
see which four did not parse and why — losing the file would destroy the
evidence that a decision was ever made about it.

| State | Meaning | Recoverable | What the user is told |
|---|---|---|---|
| `no_handler_for_format` | No registry entry accepts this format | No, without a new handler | The format, and that support does not exist yet |
| `handler_unavailable` | A required handler is declared but not running | **Yes** — start the resource | Which handler, and how to start it |
| `handler_failed` | The handler ran and could not parse the bytes | Sometimes — retry a different chain | That the file may be corrupt, encrypted or malformed |
| `blocked_by_policy` | The only viable chain exceeds the document's privacy class | Yes — reclassify, or install a local handler | Which tier was needed and why it was refused |
| `unsupported_variant` | Format recognized, this instance is not handleable — password-protected, DRM'd, image-only where no OCR handler exists | Sometimes | The specific reason, never a generic failure |

These are five different remedies, so they must be five different states.
Collapsing them into one "parse failed" is the failure mode this table
exists to prevent — `handler_unavailable` is a one-command fix and
`no_handler_for_format` is a roadmap item, and a user cannot tell them
apart from a shared message.

Every terminal state writes a custody record. A document that failed to
parse still has provenance: what was attempted, by which chain, when, and
where it ran.

## Anchor Kinds By Source Shape

Three anchor kinds exist because sources have three genuinely different
intrinsic coordinate systems. A fourth needs a fourth coordinate system,
not merely a fourth format.

| Kind | Coordinates | Durable across re-derivation? | Available for |
|---|---|---|---|
| `geometric` | Page index plus bounding box in the original document's coordinate space | **Unconditionally.** The coordinates belong to the original bytes, which are `regenerable: false`. | PDF and raster images — the only sources with fixed page geometry |
| `tabular` | Sheet or table identity plus a cell range, e.g. `sheet:2!B4:D9` | **Unconditionally.** Cell coordinates are part of the source, not of the parse. | Spreadsheets and delimited text |
| `logical` | Structural path plus character offset | **Only through a recorded alignment.** A better parser changes whitespace, reading order and table handling, so the same offsets mean different things across versions. | Every flowing-text format |

### Why `tabular` is not a special case of `logical`

A spreadsheet has no pages and no flowing character stream. A character
offset into a serialized sheet is meaningless to a user — nobody cites
"character 4,120 of Q3-forecast.xlsx" — and it is unstable under any
reserialization, including a column-width change that alters nothing
semantically. Cell coordinates avoid both problems: they are what a user
already reasons in, and they survive re-derivation without an alignment
map because they are intrinsic to the source rather than produced by the
parser.

That property puts `tabular` on the `geometric` side of the durability
split despite looking structural. `DOC-P0-009`'s guarantee applies
unconditionally to `geometric` **and** `tabular`, and to `logical` only
through a recorded alignment.

### Slide-scoped, spine-scoped and part-scoped logical anchors

Presentations, EPUBs and emails are formally `logical` — offsets within a
slide, spine item or message part still shift between parser versions —
but their outer coordinate is intrinsic and stable. A slide index does
not change when the parser improves. Resolution should degrade
gracefully: an unaligned anchor into slide 7 still resolves *to slide 7*
rather than reporting `unresolved`, even when the within-slide offset
cannot be trusted. This is a narrower failure than a Word document, where
an unaligned anchor has no stable outer coordinate to fall back to.

Presentations carry a second content stream — speaker notes — that must
be separately addressable. A note is not body content on the same slide;
citing one and citing the other are different claims. Email has the same
shape: headers, body and each attachment are distinct addressable parts.

## Gaps

| Gap | Detail | Owner |
|---|---|---|
| ~~Image intake has no requirement~~ **Closed 2026-08-07** | Funded by `OT-P1-024` / `DOC-P1-024`. Raster images are parsed by local Tesseract OCR at **tier 2**, alongside scanned PDFs — not tier 3 — so they carry geometric anchors and do not wait on `vision.default`. | Closed |
| `unstructured-io` migration unverified | Its README describes a mid-flight move to the current `docker-service` structure. Six matrix rows depend on it, including all of HTML, email and images. | Verify before `derivation` is built |
| Encoding and language detection | Absent from every document. Affects FTS5 tokenization in `DOC-P0-023` before it affects the P2 multilingual target. `unstructured-io` reports language and can inform this. | Unassigned |
| Archive expansion bounds | `.zip` routing has no declared depth, count or expansion-ratio limit. Decompression bombs are a real intake surface for every ZIP-container format, which includes OOXML, ODF and EPUB. | Resolve with `SECURITY.md` |

## Deliberately Out Of Scope

| Format | Reason |
|---|---|
| Video and audio containers | `DOC-P2-003` bridges `audio-tools` instead. A transcript is a document with time anchors, which would be a fourth anchor kind. |
| CAD, proprietary legal formats (`.wpd`) | No credible parse path and no evidenced demand. These produce `no_handler_for_format`, which is the correct answer rather than a gap. |
| Macro execution | Macro-enabled formats (`.docm` `.xlsm` `.pptm`) are accepted and their macros parsed as inert bytes. Nothing in this scenario executes a macro, ever. |

## Adding A Format

1. Add the row to `api/internal/derivation/registry.json`, then mirror it
   here. The registry is the source of truth; the test that compares them
   is what keeps this document honest.
2. Name the required and desirable capabilities, not a parser. If no
   existing handler provides a required capability, the work is a new
   handler, not a new format row.
3. Decide the anchor kind by asking what coordinate system the *source*
   has, not what the parser emits. Add a new kind only when no existing
   one describes the source's intrinsic coordinates.
4. A new kind is an additive migration plus a resolver dispatch branch,
   per the anchor-format rule in
   [`../concepts/DATA.md`](../concepts/DATA.md). Existing anchors must
   keep resolving.
5. Declare the extension in `intake`'s accepted set. Detection is by
   content sniffing; the extension is a hint, never the decision.
6. Add or extend an operational target and a test carrying its
   `[REQ:DOC-...]` tag. A format with no test is a claim, not a
   capability.

## Adding A Handler

1. Declare the capabilities it provides and the tier it costs. A handler
   that provides nothing no existing handler provides is not worth
   adding.
2. Add it to the candidate chains of every format it improves. Existing
   chains keep working; a new handler is additive.
3. Handlers are replaceable by construction. If adding one requires
   changing routing logic rather than registry data, the router has
   drifted and that is the defect to fix first.

## Cross-References

- [`anchor-uri.md`](anchor-uri.md) — the citation format these anchor kinds are encoded into
- [`render-matrix.md`](render-matrix.md) — the write-side twin: render targets, fidelity, and the six write-side terminal states
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — handler registry, the two-parser split, and the anchor-kind decisions
- [`../concepts/DATA.md`](../concepts/DATA.md) — anchor storage and the anchor-format rule
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — the runtime routing sequence and the sensitivity gate
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — parse resources
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — unresolved drift
