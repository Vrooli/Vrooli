# Document fidelity investigation and design

Observed and researched: **2026-10-14**. Status: source-backed preparation; implementation and live-account qualification are not performed. Governing direction: `docs/architecture/external-knowledge-search.md`. Read with `GOAL.md`, `RESEARCH.md`, `QUEUE.md`, and `SETUP.md` in this goal home. This report proposes owner changes and acceptance evidence; it does not replace scenario product contracts or admit an epoch. The external-knowledge-delivery team remains disabled.

## Findings that change delivery admission

Use structured Google-native reads for Docs, Sheets, and Slides, then normalize before chunking. Use Drive binary download and qualified local parser adapters for uploaded Office files and PDFs. Export is a tested fallback or visual derivative, with explicit losses. Preserve typed spreadsheet cells alongside searchable prose. Text embeddings, generated questions, OCR, vision descriptions, and image embeddings are different capabilities and need separate evidence.

The current repository does **not** substantiate a complete Google-native extraction path, raster OCR, or document-manager intake-to-derived-search publication. The following are especially material:

- `doc-ocr` currently implements an embedded-text baseline. Decoded PNG/JPEG inputs yield the literal label `image page`, confidence 0.35, and a full-image box. PDF inputs use printable-byte extraction; an empty result yields `scanned page`, confidence 0.55, and a synthetic box. These are not transcriptions. They MUST NOT become observed-text passages, semantic evidence, or spreadsheet/chart answers. Its own operations document records raster character error rate as N/A. [L06]
- The derivation registry chooses the first matching MIME row; PDF selects `doc-parse`. `Service.Derive` invokes one runner and has no scan classification, low-confidence escalation, or fallback chain. `NativeRunner` invokes every resource as `parse - --capabilities …`, but doc-parse expects a regular file path and doc-ocr exposes `ocr`, not `parse`. Resource output is not the scenario's `{units:[…]}` model. A valid resource JSON response without units can decode into an empty model and be marked parsed. These seams need actual contract adaptation and tests. [L02–L05]
- `api/main.go` and the module registry mount intake, corpus, enrichment and retrieval, but no derivation handler/schema. Intake stores original bytes and creates a document; it does not call derivation or populate retrieval units. Enrichment's mounted module passes a nil Gateway and therefore records unenriched or rejects embedding. Search fixtures or fake-runner tests do not prove a connected production path. [L07–L10]
- The format matrix describes fallback handlers, `gateway-vision`, formats and fidelity beyond `registry.json`. Google-native MIME types are absent from that registry. Doc-parse's Office response returns Markdown with `tables:[]`; no typed cell/formula/slide-note model is emitted. Its portable-output equality measurements prove equality between two runtimes, not independently correct content, structure or locators. [L01–L05]

These are in-scope admission findings, not permission to repair code during preparation. Reconcile the format claims through document-manager and resource owners before an implementation slice claims them.

## Investigation method and bounded result

The investigation used one Search Hub recall query, local canonical documents and source inspection, and bounded read-only browsing of official Google API reference and guide pages. Search Hub returned relevant architecture, prior parse/OCR work and shared search records; it also reported degraded reranker/provider circuits. No search result was treated as qualified format support without reading its owner source.

Four research questions bounded the external investigation:

| Question | Result | Residual gap |
|---|---|---|
| Can native reads preserve more than exports? | Yes: Docs tab structure, Sheets distinct cell values/formatting/ranges, and Slides body/notes/objects are exposed separately. [G01–G13] | No account call or cross-format fidelity comparison was performed. |
| Do structured reads require a write grant? | No. The read methods accept `drive.readonly` and their respective service read-only scope; per-file `drive.file` is also accepted for authorized files. [G02, G05, G07, G11] | Existing and future descendants must be qualified for the actual connection scope and selection route. |
| Can portable local OCR/vision be claimed today? | Raster transcription is absent from current doc-ocr source; optional Unstructured and AI Gateway attachment seams exist with limits. [L06, L14–L15] | No representative raster OCR, vision accuracy or image-vector retrieval is qualified by this report. |
| Can versions stay citable without permanent originals? | Retained normalized evidence can remain version-bound, with separate live links and honest missing-visual/history states. [G15–G18] | Cross-call snapshot consistency and old-version retention duration need selection and tests. |

This is a bounded deep investigation, not a native Web Search L3 run. The discovered L3 contract delegates an execution and can capture findings. That would conflict with the assigned research-only/no-native-mutation/no-further-agent boundary. No delegated research run, finding-store mutation, source-account read, credential operation, installation, product edit or test run occurred. Official API facts below are dated observations; recommendations are marked as proposed.

## Actual support versus declared support

“Implemented” below means visible code, not fresh runtime certification. “Declared” means owner prose or registry, not acceptance evidence.

| Source | Declaration / existing mechanism | Actual visible support and limit | Proposed delivery posture |
|---|---|---|---|
| Google Docs | Architecture requires a tested structured-read or export adapter. | No Google-native registry row or adapter in inspected document-manager path. Existing text path splits nonempty lines; it does not parse headings, lists or XML/HTML semantics. | Structured Docs read; preserve all tabs and logical structure. Metadata-only until admitted adapter gates pass. |
| Google Sheets | Tabular anchors and common spreadsheet parsing declared. | No Google-native adapter. Native CSV/TSV handler splits lines and marks comma-containing lines tabular without parsing escaped CSV cells. XLSX parser returns Markdown, not distinct formulas/typed/effective/display values. | Native typed cell projection plus range/table chunks. Exact queries use structured evidence. |
| Google Slides | PPTX support declared; architecture proposes slide units and notes. | No native adapter. Generic Office-to-Markdown response has no explicit slide identity, notes/body distinction or image region contract. | Structured presentation read; separate body and notes, retain object locators and visual availability. |
| Text PDF | doc-parse declares classification, text and geometry. | Rust shim invokes pdf-inspector, exposes positioned text, page count, scan/encoding flags and Markdown. Scenario runner fails to map that response into units. Correct reading order, table structure and normalized geometry are unqualified. | Qualify page/block adapter; preserve raw coordinate conventions and verified normalized boxes. |
| Scanned/mixed PDF | Matrix declares local OCR/fallback. | doc-parse exposes `pages_needing_ocr`; no scenario fallback. doc-ocr does not recognize raster glyphs. | Text pages may publish with per-page degraded coverage; scanned pages stay OCR unavailable until a real engine passes. |
| Raster PNG/JPEG | Registry includes local OCR. | Decoder obtains image dimensions but returns a generic label; no glyph recognition. [L06] | Index source metadata/real supplied alt text only; expose OCR unavailable. |
| TIFF/BMP/HEIC/WebP / diagrams | Matrix extends raster breadth; registry has TIFF/BMP but not HEIC/WebP. | OCR source imports PNG/JPEG decoders only. Printable binary strings are not a safe fallback. Diagram relations are not extracted. | Format-specific decoder and OCR gates; optional labelled vision; otherwise metadata-only/unsupported variant. |
| DOCX/ODT/EPUB/RTF | doc-parse MIME rows; 42-fixture portable measurements. | anydoc Markdown path exists. Preservation of headings, tables, images, notes, protection/error categories and scenario integration needs content-ground-truth tests. | Qualified structural text lane; explicit unsupported objects/variants. |
| XLSX/PPTX uploaded to Drive | Same doc-parse lanes. | These are binary Office files; no evidence they become native Sheets/Slides automatically. Formulas, cached results and notes fidelity are not represented by current output contract. | Download original transiently; type-specific extraction; partial support until fixtures qualify each field. |
| Legacy DOC/XLS/PPT, XLSM, ODS/ODP | Some appear in the human format matrix/long-tail fallback. | Absent from inspected registry; no fallback runtime registration demonstrated. | Metadata-only/no handler, or explicitly qualified optional lane. Never run macros, refresh workbook links or activate embedded content. |
| Semantic image vectors | Generic gateway modalities include image/vector vocabulary. | Shared Embedder accepts text; document-manager embedding request is TEXT_EMBEDDING. No qualified cross-modal image index/query pipeline was established. | Separate optional extension with compatible model/query space and own evals; never inferred from text vectors. |

## Google-native reads, exports and scopes

### Docs

Request `documents.get` with `includeTabsContent=true`, then traverse `tabs` and `childTabs`; the default legacy body contains only the first tab. Preserve tab ID/title/path and each tab's segments. [G01, G02] Normalize paragraphs, heading/list context, nested tables, footnotes, links and object references into separate units; handle unknown element types explicitly. The API indexes structural text using UTF-16 code units, so do not substitute byte/rune offsets in source locators. [G03]

Choose and persist a suggestions view. Proposed initial default: existing accepted document body (`PREVIEW_WITHOUT_SUGGESTIONS`), comments omitted; expose that choice in fidelity status. Inline proposed changes and comments can be a later explicit additional lane, not silently mixed operating instructions. Current official `documents.get` has comments options with compatibility conditions; recheck this evolving surface at admission. [G02]

An export to Markdown/DOCX can reuse a qualified parser, but acceptance must cover nested tabs, tables and links; export support does not prove their preservation. Exported PDF offers a rendered visual snapshot with page geometry, not stable native tab/heading indices. Native embedded drawings have limited structural detail. Embedded image URLs are short-lived requester-authorized links and should be transient inputs, never durable citation URLs or exposed payloads. [G03, G13–G14]

### Sheets

Prefer `spreadsheets.get` with explicit field masks and bounded `ranges`. Include grid fields deliberately: setting a field mask overrides `includeGridData`; a metadata-only request is not extraction. [G05] A cell record must keep `userEnteredValue`, `effectiveValue`, `formattedValue`, effective number format and errors distinct. A formula is entered evidence; its effective result is the server-observed calculated value; its displayed string is a presentation of that result. Preserve booleans, numbers, strings, blanks and error types without string coercion. [G04, G20]

`values.batchGet` is a lower-payload option. It offers FORMULA, UNFORMATTED_VALUE and FORMATTED_VALUE views, not all three in one scalar response. Multiple calls need a consistency check and coordinate reconstruction; retain requested ranges and leading/trailing blank positions rather than compacting rows. [G07–G08] Proposed default is one bounded grid read that includes the needed distinct cell fields; use values-only reads for an explicitly lower-fidelity lane.

Store spreadsheet locale/timezone and number formats. Dates/times/durations are numeric serials; preserve the original number and date format/type before producing a normalized date. Google documents a 1899-12-30 day origin. Formatted text follows spreadsheet locale, so a displayed `1.234,50` must not be parsed using the host's US locale. Date-only, duration and datetime are different; a serial is not automatically a UTC timestamp. [G08–G10]

Preserve sheet ID, title, grid bounds, named ranges, merges, hidden-row/column/filter metadata where admitted. Google GridData begins at explicit zero-based row/column coordinates; do not treat returned arrays as origin A1. [G19] A hidden row is not an authorization boundary. Proposed extraction includes it with visibility facets unless collection policy excludes it; aggregation must state whether hidden/filtered rows count.

Charts need chart ID/spec/title/alt text/axes and linked data-range units. The chart specification describes series and hidden-dimension behavior; it is not a replacement for the cells or a pixel-faithful rendering. [G06] CSV/TSV export includes only the first sheet. XLSX export may carry useful workbook artifacts but requires a distinct parser qualification; PDF export loses typed/formula semantics and may paginate/crop. [G14]

For uploaded XLSX, distinguish formula text from the cached workbook result and cached display, with the workbook date system and format captured by a qualified parser. Do not execute formulas to manufacture a missing cache. Record cache absent/unknown freshness; Google effective values and uploaded workbook caches are different provenance. Exact retrieval should return an error or missing-value state if evidence is unavailable.

### Slides

Read `presentations.get`; normalize slide IDs, shape text, tables, image/alt-text metadata, groups, transforms and links. Resolve inherited layout/master presentation where needed; tag repeated template text to avoid dominating retrieval. [G11–G12] Speaker notes live in each slide's notes page; locate the notes shape through `speakerNotesObjectId` and keep its text separate from audience-visible body. A missing notes object means no notes, not failed content. [G13a]

Retain image/object regions and crop/transform information; OCR handles actual raster text, while vision may generate a labelled interpretation. Image `contentUrl` expires by default after 30 minutes and grants access as the requester. Never persist it in serving snippets, logs or federated metadata. Linked Sheets charts and unlinked chart images differ. Reading a deck never grants permission to fetch an out-of-scope linked workbook. [G12, G13]

PPTX/PDF/plain-text exports are declared by Google, but the report does not certify note inclusion, image retention, chart fidelity or reading order for those exports. Their fixtures must prove the intended fields. [G14]

### Scope and connection behavior

All three native read methods accept `drive.readonly`; they also accept their service-specific read-only scope and per-file `drive.file`. Enable the corresponding APIs and advertise the actual granted operations; API enablement and OAuth scopes are distinct. Drive export accepts Drive scopes, so a service-only read grant is not a universal binary-download/export grant. [G02, G05, G07, G11, G16]

Folder allowlisting remains document-manager policy. `drive.file` is per-file access, while broader Drive scopes have different access/verification implications; do not claim folder selection limits the Google token or grants all future descendants. [G17] The future early outcome is a repeatable integration-hub CLI and UI connection journey: consent, scope/capability inspection, refresh, reauthorization and revoke. Operator live consent gates only dependent live work; fixture-backed driver, normalization and extraction preparation need not wait. Activation and credentials are outside this report.

## Extraction-to-unit contract and publication

Extend existing document-manager derivation/corpus repositories rather than create another content authority. Keep an immutable normalized representation for each observed extraction. Proposed minimum fields:

| Entity | Required data / meaning |
|---|---|
| Source | Connection/account reference + remote file ID; membership/exclusions/grants; native MIME; source title/URL; provider catalog version/time; current visibility. Hash alone is not source identity. |
| Capture | Read start/end, provider metadata before/after, optional Docs/Slides revision, extraction method, source projection digest, consistency status and covered/excluded regions. Never claim atomic consistency across unrelated API calls. |
| Unit | Source/capture ID; stable version-local unit ID; parent/order/type; observed text and language; structure and typed cell/table payload; source locator; confidence/engine only where meaningful; available visual evidence; coverage losses. |
| Derivation | Adapter/parser/normalization version and artifact identity; successful/partial/failed state; evidence origin (`native`, `binary-parse`, `ocr`, `supplied-alt`, `generated-vision`) and recipe. Generated text cannot overwrite observed text. |
| Index view | Source-unit IDs; raw/contextual/overview/question/vision-description role; view recipe/prompt/model version; policy/ACL version; content hash; lexical/semantic publication version/time. |

Local `Unit.Metadata map[string]string` is a useful transitional seam but cannot safely encode all typed cells, coordinates and graph relationships by convention. Reconcile and extend the owning schema/protobuf contracts, then regenerate through the owner; do not hand-edit generated code. Keep distinct extraction, chunking and embedding versions so a new recipe need not redownload unchanged originals. A source-level hash used by shared search should reflect the normalized projection plus extraction policy, not merely raw byte hash or Drive modified time.

Publication sequence: catalog membership and durable capture work; bounded extraction into normalized units; atomic publish of citable units and lexical lane; independent optional semantic/enrichment work; serving publication version. Separate source-feed checkpoints from index promotion checkpoints. Shared ChangeSetSource commits its cursor after generation promotion; an embedding outage must not lose already captured provider changes. Stable pages should read a durable local capture, not walk a mutable remote source and call it one snapshot. [L12–L13]

## Type-specific chunks and locators

| Shape | Proposed searchable units / bounded context | Locator requirements |
|---|---|---|
| Docs / DOCX / flowing text | Heading-aware sections, lists and explicit table units; tab/title/heading prefix; overlap only within coherent sections. Shared Markdown chunker can handle a proven normalized Markdown projection. | Capture hash/version + tab ID + segment ID + structural path + UTF-16 range where native; link/heading ID when available. Export coordinates belong to the export, not native content. |
| Sheets / XLSX / CSV | Workbook/sheet/table overview; bounded contiguous row groups with repeated labels/headers/units. Keep typed cell store separate. Wide/oversized rows split into labelled column groups retaining row identity, or stay structured-only. | Capture + stable sheet ID + numeric grid range + A1 display; named-range ID where available. Current anchor sheet ordinal is insufficient for mutable native reorder/rename. |
| Slides / PPTX | Slide body, separate notes, table rows and object-specific visual units. Keep slide title/context and avoid joining unrelated slides to fill a token target. | Capture + slide ID + body/notes surface + object ID + text range or transformed region. Position/slide ordinal is version-local. |
| Text PDF | Page-positioned blocks in tested reading order; heading/table-aware grouping across pages only with block map. Avoid repeated header/footer dominance. | Capture + page + verified box + coordinate basis, page dimensions and rotation. Table cells need a source block map. |
| OCR PDF / images | Actual recognized runs grouped by page/region; language/confidence/engine; low-confidence spans visible. | Capture + page/image region + dimensions/DPI/rotation and engine; confidence is not geometry or correctness proof. |
| Charts / diagrams | Supplied metadata and OCR text; optional generated diagram relations/description as hints linked to the object. Exact chart values come from linked typed data if authorized and coherent. | Capture + chart/object/region; linked-data source/version separately. No numeric answer from inferred pixel heights. |

Current `vrooli-anchor:1` supports content-addressed derivation plus logical numeric paths, positive sheet ordinal/A1 and normalized page boxes. Its resolver considers tabular/geometric anchors intrinsically resolvable when source bytes exist. Remote mode needs a version-bound locator envelope or governed anchor version that explicitly records retained evidence versus retrievable original visual/source. Do not alter v1 meanings silently or report `resolved` merely because the current remote URL opens. Historical normalized passages may resolve while the historical original visual is unavailable; label both. Unknown-version, degraded and forbidden outcomes remain necessary. [L11]

Stable tab/sheet/slide/object IDs help follow edits, but deletion/recreation, row insertions and changed offsets invalidate old coordinates. A live link is a navigation aid. The citation is to the captured version, with optional tested alignment to a later version; guessed current coordinates must not replace it.

## Exact retrieval and semantic hints

Semantic recall selects likely passages/tables; exact cell reads and deterministic authorized computation answer numeric questions. An exact-answer receipt should identify requested range, captured version/time, representation (entered/effective/display/cached), locale/unit interpretation, blank/error/hidden-row policy, calculation and input range coverage. Require a single coherent capture or disclose a cross-source temporal mismatch. Exact identifiers, numbers, codes and phrases also need lexical/range filters; vectors do not preserve arithmetic or equality.

For example, “What is the Q3 total?” may discover a Revenue sheet through contextual/semantic views, then fetch typed Q3 cells or the captured total formula/result. It must distinguish a displayed rounded `$1.2M` from the effective `1234567.89`, and a stale uploaded cache from a fresh server-observed value. Summaries and anticipated questions can route to that range; neither is evidence for the number.

Start with raw citable units and deterministic title/tab/sheet/slide/heading context. Add bounded overview, summary and anticipated-question views only after baseline evaluation, with every view mapping to real units. A question such as “Where is the calibration procedure?” is an index hint, never a new cited source. Keep held-out queries separate from generated questions, collapse duplicate views before result emission, and inherit source grants/withdrawal/residency. Existing enrichment stores summary and text vectors, with content version hardcoded to 1; it does not establish this versioned multi-view pipeline. [L08, L12]

Shared search is text-oriented and its point IDs are based on source identity/chunk ordinal. Retain independent citable unit IDs so re-chunking does not turn an old retrieval point into a citation to different evidence. Preserve document-manager's SQLite adapter; shared reconciliation/fusion mechanics do not require moving its authority into Qdrant.

## Limits, streaming and honest degradation

Transport streaming, parser streaming, source paging and model context bounds solve different problems. Drive binary content supports ranged download; Google Workspace export via `files.export` is limited to 10 MB and does not support partial downloads. [G15] Docs/Slides native gets have no content-pagination parameter in their read reference; stream decoding/field masks can bound local work but do not create a provider cursor. Sheets can tile ranges; Google recommends approximately 2 MB payloads and has read quotas of 300/minute/project and 60/minute/user/project at this access date. [G18]

Current derivation input is a byte slice; NativeRunner copies it into a string and buffers command output. Doc-parse reads/compiles its WASI module and parses a regular staged file. Doc-ocr reads the entire input **before** enforcing 32 MiB. Current shared source paging bounds documents per page, not one document's bytes, units or composed embedding text. These paths need limits before allocation/download, decompressed archive expansion limits, and bounded per-document extraction/output/page counts. [L02–L06, L12–L13]

Proposed initial policy: downloadable binary cap 32 MiB, normalized response/capture cap 32 MiB, sheet reads targeting at most 2 MiB response per range tile with adaptive subdivision, parser timeout 60 seconds, OCR/vision at most 20 pages/regions per job and a durable continuation rather than silent truncation. These values are conservative admission hypotheses, not measured operational budgets; the owner must benchmark and select them. Reject compressed expansion bombs and excessive pixels before decode. Record covered ranges/pages and resumable continuation for partial work. A large file limit state keeps a source catalog entry without retaining original bytes by default.

Model bounds must come from the selected role/model, including text tokens, image MIME/pixel/byte/count limits, and output budget. Never use an invented universal context window. Compose context first, count with the appropriate tokenizer or documented conservative bound, and split/reject before embedding. The shared Markdown defaults estimate 4 characters/token, target 900 and hard nominal 1200; an atomic table/fence can exceed that maximum. Add a final real input bound and a type-aware split/structured-only decision. [L12]

AI Gateway accepts resolved inline image attachments in existing provider transport; this proves a request seam, not diagram accuracy or multimodal embeddings. Document-manager's current builder has no attachment option. Future vision/OCR calls must pass through an owner seam and policy-qualified role; no direct provider client or image URL leakage in Search Hub. Local-first permits fallback and cannot establish local-only residency. Text and visual payloads inherit sensitivity before inference. [L14]

Use explicit machine-readable support/fidelity states: `native-structured`, `parsed-text`, `partial-structure`, `ocr-unavailable`, `ocr-low-confidence`, `visual-interpretation-unavailable`, `unsupported-format`, `unsupported-variant`, `blocked-by-policy`, `too-large`, `handler-unavailable`, `handler-failed`, `source-unavailable`, and `reauthorization-required`. Map them through existing terminal states plus capability/coverage facets where suitable; these names are proposed, not a pre-existing API contract. A parsed empty document differs from an adapter that omitted content. One boolean `parsed` or `partial` cannot convey all of these losses.

Local OCR ownership belongs in doc-ocr or a qualified local extraction resource; document-manager orchestrates, normalizes and records outcomes. Optional Unstructured currently uses a large Linux-amd64-only pinned runtime, and its high-resolution lane is explicitly partial without detectron2. Its CLI is `process --input …`, not the generic runner's `parse`. It cannot serve as an assumed portable fallback. [L15] Vision inference belongs to AI Gateway role/profile governance; diagram relation/description storage belongs to document-manager. Cross-modal embeddings are a separately selected model/index/eval extension.

## Mutable remote versions and retention

Drive `version` is a monotonically increasing catalog change signal including non-visible changes; head revision ID is binary-only. Metadata size is not an export/extraction-size guarantee. [G19a] Native Docs/Slides revision IDs are opaque, edit-access-only, requester-bound and only guaranteed valid for 24 hours, so read-only ingestion must work without them. [G03, G12a] Use connection/file identity, observed provider version/time and a normalized capture digest; store revision fields opportunistically, not as durable universally retrievable revision pointers.

For multi-range/image/export capture, read provider metadata before and after; if it changes, retain the attempt as inconsistent and schedule a bounded retry rather than publish false snapshot fidelity. Matching versions support an observed consistency claim, not proof of a transactional snapshot. Volatile formulas, external data and linked charts can have freshness concerns beyond an edit timestamp. Default exact answers disclose capture time and may request a live authorized refresh for freshness-critical values; selecting a refresh TTL remains open.

Default remote retention is normalized text/structure/typed cells, citable passages, extraction/index manifests and derived lexical/semantic views. Original bytes, image crops, exports and bearer-like image URLs are transient; remove them on success, failure and crash recovery through scoped expiry/cleanup. Optional archival originals require explicit policy. Keep source catalog/version histories separately from deduplicated content: identical files in different accounts/collections must never share authorization by hash.

Retained historical normalized evidence cannot be regenerated by rereading a changed remote file. Declare its storage and pruning policy accordingly; do not inherit the archival assumption that all derivations are freely regenerable from permanent originals. Text indexes and question views can be rebuilt from retained units, while discarded original layouts/images cannot. Current corpus `Export` emits empty artifacts and `Prune` is a safe no-op, so neither currently establishes the required remote retention/export behavior. [L01, L09]

Proposed initial serving retention: current normalized capture plus one prior successful capture for up to 30 days, subject to the selected policy; discard older passage content while retaining a minimal non-content citation tombstone if permitted. This is an open default, not acceptance of indefinite retention. Cite unavailable history honestly. Google revision availability is not a permanent archive guarantee; downloadable older binary revisions have retention restrictions, and this read-only design must not set keepForever or modify source revision policy. [G15, G21]

ACL/membership withdrawal blocks all current and old retained serving generations, question views, summaries, typed stores, vectors, caches and context fetch before scoring/emission. Physical cleanup may follow, but generation rollback must not restore access. Extracted text, typed values, vector payload bodies and logs are content copies too. Disconnect/revoke behavior and physical deletion deadlines must follow the selected collection policy. A stale original visual must not be served under the privilege of an expired requester image URL.

## Existing owners and required seams

| Owner / seam | Needed outcome |
|---|---|
| integration-hub + Google drivers | Repeatable early CLI/UI connection outcome after activation; bounded authenticated Docs/Sheets/Slides/read/download/export operations, grants/scopes, API-disabled/permission/export-size errors, expired image inputs, refresh/revoke. Expose operations and capability receipts rather than credentials. |
| document-manager intake/corpus | Remote-backed source/member/version/retention mode independent of permanent-byte intake; durable captures/queue; typed cells and coverage; no hash-only remote source identity. |
| document-manager derivation/anchors | Adapt resource command and result shapes; classify/route PDF pages; explicit fallbacks and loss states; native normalized-unit adapters; version-bound remote locator envelope; wire schemas/handlers and end-to-end publication. |
| doc-parse | Preserve and verify Office structure, formula/cache/date-system fields and slide/notes/image maps; normalized PDF geometry/reading-order/table evidence; typed terminal errors; bounded staged inputs and expansion. |
| doc-ocr | Select a real raster recognizer or advertise transcription unavailable; do not emit generic labels as source text; genuine raster/language/rotation/geometry and accuracy gates. |
| unstructured-io | Qualify optional process adapter/platform/memory/high-resolution limits; no universal fallback claim. |
| document-manager enrichment + AI Gateway | Wire actual client through gatewayreq; image attachments/residency; versioned grounded views; explicit model input bounds; separately measured vision and potential cross-modal model compatibility. |
| packages/ai-go/search | Reuse source paging, changelists, composer/chunker/fusion/version seams. Add portable bounded-input/view/locator metadata seams only where reusable; keep Google selection, typed computation, retention and policy in document-manager. |
| search-hub | Carry typed provenance, representation/coverage/degradation and authenticated grant context; no Google client, OCR, credentials or corpus storage. Federation must match direct evidence and privacy. |
| Scenario contract / generated-artifact owners | Reconcile PRD/requirements/format claims before admitting implementation, govern dependencies, extend shared protos through their owner, and verify generated freshness before consumer diagnosis. |

## Proposed fixtures and acceptance gates

Fixtures must be synthetic and source-controlled, with independent expected text/typed data/locators. Tests must assert desired content fidelity, not merely nonempty runs, valid JSON or equality between two copies of the same parser. Reuse existing resource corpus where it contains suitable evidence; extend owner fixture homes rather than place product fixtures in this goal home.

| Gate | Given / When / Then acceptance intent |
|---|---|
| F01 Native Docs | Given nested tabs, headings, lists, non-BMP Unicode, nested tables, footnote, link, image and suggestions, when all-tab extraction runs, then every admitted segment is represented once, ranges use UTF-16 correctly, and omitted/unknown elements and selected suggestion view are explicit. Default first-tab-only failure must be caught. |
| F02 Native Sheets values | Given formulas, effective results, formatted currency/percent, booleans, errors, string `00123`, empty versus zero and absent caches, when extraction/search/context run, then all representations remain distinct and exact answers cite the intended one. |
| F03 Sheets identity/ranges | Given sheet ID 0, quoted/special titles, rename/reorder, row insertions, sparse ranges beginning outside A1, merges, named ranges and hidden/filtered rows, then bounded reads preserve numeric coordinates, identity, blank positions and declared aggregation policy. |
| F04 Dates/locales | Given US/German displayed numbers, date-only/datetime/duration, spreadsheet timezone and XLSX 1900/1904 date systems, then source numbers/formats survive and any normalized date is explicitly typed and correctly interpreted. |
| F05 Charts and links | Given chart specs with missing values, filters, labels and out-of-scope linked workbook, then chart metadata is searchable, authorized source data drives numbers, and no link expands grants. Visual-only chart queries degrade honestly. |
| F06 Slides | Given body plus notes, no notes, grouped/transformed objects, repeated master text, tables, image-only chart and linked chart, then body/notes identities and regions are preserved, no generic whole-deck flattening is certified, and requester image URLs never appear in durable results/logs. |
| F07 Office binaries | Given multi-sheet XLSX with formulas/caches, DOCX headings/tables/images and PPTX notes/body/images, then binary-download-to-parser-to-normalized-unit outputs match independent ground truth. Markdown-only lanes identify their losses. Macros/links/embedded actions never execute. |
| F08 PDF | Given text, two-column, rotated, table, mixed text/scan, malformed and protected PDFs, then reading order/page/box maps are correct, scanned pages route to qualified OCR or an explicit unavailable state, and errors never become parsed-empty success. |
| F09 Genuine OCR | Given actual PNG/JPEG/TIFF/BMP raster glyphs with known text, rotated and low-resolution pages, then transcription is compared to ground truth using character/word error and locator accuracy; text-labelled fixtures cannot qualify raster OCR. Placeholder page labels produce no observed passages. Select quantitative thresholds before admission. |
| F10 Bounds/residency | Given oversized exports, compressed expansion bombs, enormous image dimensions, giant table/fence and context overflow, then preflight bounds prevent unbounded work; continuation/coverage is honest; no secret content takes a remote inference route. |
| F11 Version/retention | Given edits during tiled reads, expired image URL, missing revision ID, removed source, disconnect/revoke and crash during transient staging, then citations bind the captured version, inconsistent snapshots do not publish, retrieval withdraws across all views/generations, and transient bytes expire. No permanent archive is created by remote default. |
| F12 Full path/search | Given fixture connection and captured units, when intake/capture, derivation, lexical publication, optional embedding and direct/federated passage fetch run, then source-backed exact and paraphrased answers work, generated hints cannot be cited, and semantic/OCR/vision outages preserve authorized available evidence with precise status. |

Future validation scope follows `docs/TESTING.md`: focused owner regressions plus relevant structure/unit/integration/business phases, adding UI journey/e2e evidence for the repeatable connection and source-status flows. Run via scoped Test Genie or the owner governed iteration program; block once on the returned run, never poll. A separate operator-consented live overlay qualifies selected folder scope, real API responses, export fidelity and recovery. Fixture progress is not blocked by absent live consent, and passing fixtures do not certify private account behavior.

## Open decisions and proposed defaults

| Decision | Proposed default / return condition |
|---|---|
| Native strategy | Structured Docs/Sheets/Slides first; independently qualified export fallback. Acceptance must cover the selected fields, not every format matrix row. |
| Comments/suggestions/notes | Notes separate and searchable within the same grant; accepted Docs body and comments omitted initially. Select any comment/proposed-change lane explicitly. |
| Hidden sheet content | Include admitted hidden rows/cells with visibility facets; disclose aggregation selection. Owner selection can exclude them. |
| OCR and vision scope | Metadata-only raster support until genuine local OCR passes. Optional vision descriptions require selected residency/budget and measured grounding. No initial image-vector claim. |
| Version retention | Current normalized capture + one prior for 30 days is a proposal; operator policy selects duration/purge deadlines and whether any originals/visual crops may persist. |
| Size/context limits | Start bounded hypotheses above; benchmark admitted formats and resolved models before calling budgets qualified. |
| Numeric freshness | Capture-time labelled exact values; freshness-critical reads can require authorized refresh. Decide TTL/volatile-formula behavior and cross-source snapshot rules. |
| Locator schema | Extend owner envelope or introduce governed anchor version; preserve v1 meaning. Decide historical normalized passage versus original visual resolution outcomes. |
| Quality thresholds | Select recall/precision, exact-answer correctness, OCR CER/WER, geometry accuracy, latency/cost/coverage budgets before admission. Require zero fabricated observed text and zero unauthorized content in every lane. |

Integration-hub's repeatable CLI/UI Google connection is an early future delivery outcome. Live operator steps gate its live evidence and dependent live corpus work, not independent implementation/fixture work. The team remains disabled during this report.

## Local evidence index

Paths are relative to `/home/matthalloran8/Vrooli`. Observations are source inspection on 2026-10-14; no runtime freshness or passing test claim is implied.

| ID | Exact owner evidence |
|---|---|
| L01 | `scenarios/document-manager/docs/reference/format-matrix.md`; `scenarios/document-manager/docs/reference/anchor-uri.md`; `scenarios/document-manager/docs/concepts/DOMAINS.md`; `scenarios/document-manager/docs/concepts/DATA.md` |
| L02 | `scenarios/document-manager/api/internal/derivation/registry.json`; `scenarios/document-manager/api/internal/derivation/types.go` (`Registry.Match`, `Unit`, `Input`); `scenarios/document-manager/api/internal/derivation/service.go` (`Derive`, `NativeRunner.Run`, `nativeModel`) |
| L03 | `resources/doc-parse/src/main.rs` (`pdf_response`, `office_response`, `process`) |
| L04 | `resources/doc-parse/cli/internal/domain/service.go` (`Parse`, `Classify`, `Capabilities`); `resources/doc-parse/cli/internal/parse/runner.go` (`Run`, regular-file staging contract) |
| L05 | `resources/doc-parse/docs/OPERATIONS.md`; `resources/doc-parse/docs/measurements.md`; `resources/doc-parse/docs/measurements.raw.csv`; `resources/doc-parse/testdata/corpus/`; `resources/doc-parse/cli/internal/parse/runner_test.go` |
| L06 | `resources/doc-ocr/cli/internal/ocr/ocr.go` (`Recognize`, `extractPDFText`, synthetic boxes); `resources/doc-ocr/cli/internal/app/app.go` (`ocr` command); `resources/doc-ocr/cli/internal/domain/service.go` (`Health`, capabilities); `resources/doc-ocr/docs/OPERATIONS.md`; `resources/doc-ocr/cli/internal/ocr/ocr_test.go`; `resources/doc-ocr/testdata/{clean-300dpi,low-resolution,rotated,multi-column,table}.txt` |
| L07 | `scenarios/document-manager/api/internal/intake/types.go`; `scenarios/document-manager/api/internal/intake/service.go` (`Ingest`, `sniff`, `RoutedFileStore.Put`); `scenarios/document-manager/api/handlers/intake/connect_handler.go` (`Ingest`); `scenarios/document-manager/api/handlers/intake/module.go` |
| L08 | `scenarios/document-manager/api/internal/enrichment/service.go` (`Enrich`, `Embed`); `scenarios/document-manager/api/internal/enrichment/types.go`; `scenarios/document-manager/api/internal/enrichment/readers.go`; `scenarios/document-manager/api/handlers/enrichment/module.go` (nil Gateway) |
| L09 | `scenarios/document-manager/api/internal/corpus/types.go`; `scenarios/document-manager/api/internal/corpus/service.go` (`Export`, safe no-op `Prune`); `scenarios/document-manager/api/internal/retrieval/service.go`; `scenarios/document-manager/api/internal/retrieval/sqlite.go` (FTS writes, substring ranking, vector scan); `scenarios/document-manager/api/internal/retrieval/types.go` |
| L10 | `scenarios/document-manager/api/main.go`; `scenarios/document-manager/api/internal/modules/registry.go`; `scenarios/document-manager/api/internal/derivation/service_test.go` (fakeRunner); `scenarios/document-manager/api/handlers/retrieval/module.go` |
| L11 | `scenarios/document-manager/api/internal/anchors/uri.go`; `scenarios/document-manager/api/internal/anchors/resolution.go`; `scenarios/document-manager/api/internal/anchors/uri_test.go` |
| L12 | `packages/ai-go/search/contracts.go`; `packages/ai-go/search/markdown.go`; `packages/ai-go/search/chunker.go`; `packages/ai-go/search/embedder.go`; `packages/ai-go/search/hash.go` |
| L13 | `packages/ai-go/search/streaming.go`; `packages/ai-go/search/README.md`; `packages/ai-go/search/reconcile_types.go` |
| L14 | `scenarios/document-manager/api/internal/gatewayreq/gatewayreq.go`; `scenarios/document-manager/api/internal/sensitivity/policy.go`; `packages/proto/schemas/ai-gateway/v1/shared/gateway.proto`; `scenarios/ai-gateway/api/internal/providers/adapter.go` (`multimodalStdin`); `scenarios/ai-gateway/api/internal/providers/adapter_test.go` |
| L15 | `resources/unstructured-io/README.md`; `resources/unstructured-io/resource.json` |
| L16 | `scenarios/web-search/.vrooli/program-runtime/research-l3.json`; `docs/TESTING.md`; `docs/architecture/external-knowledge-search.md`; this goal home's `GOAL.md`, `RESEARCH.md`, `QUEUE.md` |

## Official primary sources

All sources below were opened/read on **2026-10-14**. Recheck relevant references at implementation admission, especially recently added comments/tabs and scope behavior. The links support API capability facts, not a passing Vrooli integration. No third-party source or private account content contributed.

| ID | Official source | Fact used |
|---|---|---|
| G01 | [Docs tabs guide](https://developers.google.com/workspace/docs/api/how-tos/tabs) | includeTabsContent, child-tab traversal, first-tab legacy behavior and tab-aware links. |
| G02 | [documents.get](https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/get) | Read parameters, suggestions/comments conditions and accepted scopes. |
| G03 | [Document resource](https://developers.google.com/workspace/docs/api/reference/rest/v1/documents) | UTF-16 structure, revision limits, limited embedded drawings and transient image URI. |
| G04 | [Sheets CellData](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/cells) | Entered/effective/display values, number formats and errors. |
| G05 | [spreadsheets.get](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/get) | Grid field masks, range restriction and accepted scopes. |
| G06 | [Sheets charts](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/charts) | Chart specs, series, axes, alt text, filters and hidden dimensions. |
| G07 | [values.batchGet](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets.values/batchGet) | Range/major-dimension/render views and scopes. |
| G08 | [ValueRenderOption](https://developers.google.com/workspace/sheets/api/reference/rest/v4/ValueRenderOption) | Formula versus formatted/unformatted calculated values; spreadsheet locale. |
| G09 | [DateTimeRenderOption](https://developers.google.com/workspace/sheets/api/reference/rest/v4/DateTimeRenderOption) | Numeric serial date origin and formatted-string behavior. |
| G10 | [Spreadsheet resource](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets) | Locale, timezone and recalculation metadata. |
| G11 | [presentations.get](https://developers.google.com/workspace/slides/api/reference/rest/v1/presentations/get) | Latest structured presentation read and scopes. |
| G12 | [Slide pages/elements](https://developers.google.com/workspace/slides/api/reference/rest/v1/presentations.pages) | Object IDs, groups/transforms, alt text and linked-versus-image charts. |
| G12a | [Presentation resource](https://developers.google.com/workspace/slides/api/reference/rest/v1/presentations) | Revision limitations and layout/master inheritance. |
| G13 | [Slides images](https://developers.google.com/workspace/slides/api/reference/rest/v1/presentations.pages/images) | 30-minute requester-authorized content URL. |
| G13a | [Speaker notes](https://developers.google.com/workspace/slides/api/guides/notes) | Notes page/body shape and speakerNotesObjectId. |
| G14 | [Export MIME types](https://developers.google.com/workspace/drive/api/guides/ref-export-formats) | Supported export shapes; CSV/TSV first-sheet-only; current Markdown export. |
| G15 | [Download/export guide](https://developers.google.com/workspace/drive/api/guides/manage-downloads) | Binary media/ranges, 10 MB export cap, no partial export and older binary revision restrictions. |
| G16 | [files.export](https://developers.google.com/workspace/drive/api/reference/rest/v3/files/export) | Native export parameters and Drive scope requirements. |
| G17 | [Drive scope guide](https://developers.google.com/workspace/drive/api/guides/api-specific-auth) | Per-file drive.file and broader Drive scope categories. |
| G18 | [Sheets usage limits](https://developers.google.com/workspace/sheets/api/limits) | Recommended payload size and current read quotas. |
| G19 | [Sheets/grid resource](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/sheets) | Numeric grid offsets, sheet and visibility metadata. |
| G19a | [Drive File resource](https://developers.google.com/workspace/drive/api/reference/rest/v3/files) | File version, size, binary-only head revision and source links. |
| G20 | [Sheets ExtendedValue/GridRange types](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/other) | Typed cell variants and numeric range structure. |
| G21 | [Manage file revisions](https://developers.google.com/workspace/drive/api/guides/manage-revisions) | Revision availability and retention are not permanent-original guarantees. |
