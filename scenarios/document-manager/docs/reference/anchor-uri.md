# Anchor URI — Document Manager

The citation format. One string that names an exact region of an exact
reading of an exact document, durable enough to sit in an append-only
ledger for years, and meaningful without this scenario's database.

This is the contract between Document Manager and every consumer that
cites it — most importantly the ledger, which stores the string and
**never dereferences it**. That opacity is deliberate and it has a
consequence: this string *is* the interface. Nothing else crosses.

## Purpose Of This Document

Use this document to answer:

- What exactly goes in a citation, and what must never go in one?
- How is a URI minted, parsed, and compared for equality?
- What does resolution return when a document has been re-derived?
- How does a URI map onto the ledger's `ImportProvenance` fields?
- What has to happen to add an anchor kind or a hash algorithm?

Anchor storage belongs in [`../concepts/DATA.md`](../concepts/DATA.md).
Which anchor kind a source can carry belongs in
[`format-matrix.md`](format-matrix.md). Narrative rationale belongs in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## The Shape

```
vrooli-anchor:<scheme-version>/<document>/<derivation>/<kind>/<coordinates>[?<attributes>]
```

```
vrooli-anchor:1/sha256-9f2b7c…41/3/geometric/p7@0.118000,0.342000,0.560000,0.411000
vrooli-anchor:1/sha256-9f2b7c…41/3/tabular/sheet:2!B4:D9
vrooli-anchor:1/sha256-9f2b7c…41/3/logical/7!2/1@120-260
vrooli-anchor:1/sha256-9f2b7c…41/3/logical/0/2/1@120-260
```

```abnf
anchor-uri     = "vrooli-anchor:" scheme-version "/" document "/" derivation
                 "/" kind "/" coordinates [ "?" attributes ]

scheme-version = 1*DIGIT                    ; "1" today
document       = hash-alg "-" 1*hex-lower   ; content address, full length
hash-alg       = "sha256"                   ; extensible; see Adding A Hash Algorithm
derivation     = 1*DIGIT                    ; the derivation version cited
kind           = "geometric" / "tabular" / "logical"

coordinates    = geo-coords / tab-coords / log-coords
geo-coords     = "p" 1*DIGIT "@" unit "," unit "," unit "," unit
unit           = ( "0" / "1" ) "." 6DIGIT   ; normalized to the page box
tab-coords     = "sheet:" 1*DIGIT "!" cell [ ":" cell ]
cell           = 1*ALPHA-upper 1*DIGIT      ; A1 notation
log-coords     = [ index-path "!" ] index-path "@" 1*DIGIT "-" 1*DIGIT
index-path     = 1*DIGIT *( "/" 1*DIGIT )

attributes     = attr *( "&" attr )
attr           = attr-name "=" attr-value
```

**Parsing rule.** Split on `/` exactly four times; everything after the
fourth separator up to an unescaped `?` is `coordinates`. This matters
because `logical` coordinates contain `/` and a naive split loses them.

**No authority component.** There is no `//`. A citation names a
document, not a host — see *What Is Deliberately Absent*.

## The Three Coordinate Grammars

### `geometric` — page and box

`p7@0.118000,0.342000,0.560000,0.411000` is page 7, box from
(x0,y0) to (x1,y1).

- **Pages are 1-based.** People cite "page 7", and an off-by-one in a
  compliance artifact is worse than a slightly less natural encoding.
- **Coordinates are normalized to the page box**, `0.000000`–`1.000000`,
  always six decimal places. This is a deliberate refinement of "the
  original document's coordinate space": raw units are *parser-dependent*
  (points, pixels, twips), and an anchor whose meaning changes with the
  parser is precisely what the anchor design exists to prevent. The page
  box comes from the original bytes, so normalization keeps the anchor a
  property of the source.
- **Origin is top-left, y increasing downward.** PDF is natively
  bottom-left, so the geometry handler converts once at mint time. Pinning
  this is not pedantry — an unpinned origin means half of all citations
  are vertically mirrored and every one of them looks plausible.

### `tabular` — sheet and cell range

`sheet:2!B4:D9` is sheet index 2, cells B4 through D9.

- **Sheets are addressed by 1-based index, never by name.** A sheet name
  is content — `Q3 Layoffs` in a URI leaks into a ledger that may have
  different access control than the corpus. Index is stable and
  leak-free.
- Single-table sources (CSV, TSV) always use `sheet:1`.
- Cell references are uppercase A1 notation. A single cell omits the
  range: `sheet:1!B4`.

### `logical` — structural path and character offsets

`7!2/1@120-260` is: stable prefix `7` (slide 7), then child 2, child 1,
characters 120–260.

- **The path indexes the normalized document model, not a parser's
  element tree.** `DOC-P0-006` guarantees every tier emits the same
  normalized model; pathing over a parser's own vocabulary would make the
  anchor depend on which handler ran, which contradicts the model.
- **Indices only, never element names.** `/body/sec[@title='…']` would
  leak content. This is the same rule as sheet names, and it is absolute.
- **Offsets are Unicode scalar (code point) offsets over NFC-normalized
  unit text**, half-open `[start, end)`. Byte offsets vary with encoding;
  un-normalized text varies with the parser's composition choices. Both
  produce anchors that drift without anything visibly changing.
- **The `!` marks the end of the intrinsically stable prefix.** A slide
  index, an EPUB spine item, an email part: these do not change when the
  parser improves. Everything after `!` does. Formats with no stable outer
  coordinate — a Word document — simply omit the `!`.

That last rule is what makes the documented degradation behavior
mechanical rather than a judgement call. See *Resolution Outcomes*.

## Canonical Form Is Load-Bearing

The ledger's dedupe key is a **byte-exact string join**:

```
import_key = runtime + ":" + source_locator + ":" + content_hash
UNIQUE (scope, import_key)
```

Nothing parses that key back apart. So two URIs that differ by a trailing
zero, a percent-escape case, or an attribute order are *different
citations* to the ledger, and `DOC-P1-023`'s idempotency silently stops
working — republishing produces duplicates that look like new findings.
Canonical form is therefore a correctness requirement, not tidiness.

A canonical anchor URI satisfies all of:

| # | Rule |
|---|---|
| 1 | Scheme is exactly `vrooli-anchor:`, lowercase, no `//` |
| 2 | Hash algorithm lowercase; digest lowercase hex at **full length** — never truncated for readability |
| 3 | All integers decimal with **no leading zeros**; pages and sheets 1-based |
| 4 | Geometric units always **exactly six decimal places**, including trailing zeros (`0.500000`, never `0.5`) |
| 5 | Cell references uppercase; range omitted when a single cell |
| 6 | Logical text NFC-normalized **before** offsets are computed |
| 7 | Attributes sorted lexicographically by name; default-valued attributes omitted entirely |
| 8 | Percent-encoding only where RFC 3986 requires it, with **uppercase** hex digits |
| 9 | No trailing `?` when there are no attributes |
| 10 | No whitespace anywhere |

**Equality is byte equality of the canonical form.** There is no
semantic comparison, no normalization at read time, and no tolerance
window on coordinates. A minting implementation that emits non-canonical
output is a defect even when every URI it produces resolves correctly,
because the damage appears downstream in a system that cannot detect it.

## Mapping To The Ledger

`ImportProvenance` has three fields, and each has exactly one correct
source:

| Field | Value | Why |
|---|---|---|
| `runtime` | `document-manager` | Names the producing capability. Constant for every publication from here. |
| `source_locator` | the canonical anchor URI | The citation. Opaque to the ledger by design. |
| `content_hash` | hash of the **finding body**, not the document | The document is already identified inside the URI. Hashing the finding is what makes "the same claim about the same region" a duplicate while leaving "a different claim about the same region" a distinct entry. |

That third row is the one worth not getting wrong. Putting the document
hash here would collapse every finding about a document into one entry;
omitting it would make every republish a duplicate.

Because the unique index is `(scope, import_key)`, the same finding
published into two scopes is correctly two entries.

**A published URI is never rewritten.** `DOC-P1-020` gives `handoff`
"re-derivation propagation", and the tempting reading — update old
entries to point at the new version — is wrong twice: the journal is
append-only, and the citation was a statement about *what version 3
said*, which stays true after version 4 exists. Propagation means
optionally appending a **new** entry noting the newer reading, with the
old one left standing.

## Resolution Outcomes

Resolution takes a URI and returns one of six outcomes. There is no
seventh, and in particular there is no "best guess".

| Outcome | Meaning | When |
|---|---|---|
| `resolved` | An exact source region | Geometric and tabular always; logical when the version matches or an alignment exists |
| `resolved-degraded` | The **stable prefix only** — e.g. "slide 7", not the passage within it | A logical anchor whose version has no alignment but which has a stable prefix |
| `unresolved` | The anchor is well-formed and cannot be located | A logical anchor with neither an alignment nor a stable prefix |
| `unknown-version` | The document exists; that derivation version does not | The version was pruned, or the URI came from another corpus |
| `unknown-document` | No document with that content address here | Portable URI, different corpus |
| `forbidden` | The caller may not read this document's collection or privacy class | `DOC-P0-024`, asserted as a failure rather than an empty result |

Two invariants sit above the table:

1. **Never return a region the anchor did not prove.** Degraded is an
   honest answer; a plausible wrong region is the failure the entire
   anchor design exists to prevent.
2. **Resolution never reads a `regenerable: true` parse output.**
   Geometric resolves against the original bytes; tabular against the
   source's intrinsic cell coordinates; logical through stored
   alignments in SQLite. A resolver that reads a parse output works
   perfectly until the first prune.

### Why the version is mandatory

The URI pins the derivation version it was minted against, always — even
for geometric and tabular anchors that would resolve identically at any
version.

For logical anchors this is forced: offsets are version-relative, so an
unpinned logical anchor silently means something different after a
re-derivation. For geometric and tabular it costs one integer and buys
something an evidentiary product needs — a citation is a claim about a
specific *reading*, and "I cited what version 3 said" stays checkable
after version 4 disagrees. An unversioned citation quietly becomes a
claim about whatever the document says today, which is not what the
author asserted.

## What Is Deliberately Absent

| Absent | Why |
|---|---|
| **Any database identifier** — unit id, row id, collection id | These do not survive export and import. A URI built from them is a foreign key wearing a citation's clothes, meaningless the moment it leaves the machine that minted it. |
| **Corpus, collection, or host identity** | `DOC-P0-017` promises full export with no lock-in. A URI naming its corpus would be exactly that lock-in, and would leak deployment topology into a compliance artifact. Which corpus can resolve a URI is the resolver's problem, not the citation's. |
| **Any content-derived token** — sheet names, element names, headings, titles | A URI travels into a ledger whose access control may differ from the corpus's. Structure is safe to expose; content is not. |
| **A scheme host or `//` authority** | This is local-first. There is no stable host, and an `https://` form would imply network dereference semantics the ledger explicitly does not use. |
| **Load-bearing attributes** | Attributes are hints and telemetry only. Resolution uses the path alone, so a URI arriving from outside cannot steer resolution through a crafted attribute. |

### An accepted property, stated plainly

A content address permits **confirmation of possession**: someone who
already holds a document can confirm it is in this corpus by comparing
hashes. For most buyers this is unremarkable; for eDiscovery it is worth
knowing before a reviewer asks.

The mitigation — a per-corpus HMAC of the content address — would defeat
portability, which `DOC-P0-017` requires, and would make the same
document uncitable across two corpora. The trade is resolved in favour of
portability and recorded here rather than left to be discovered.

## Attributes

Optional, never load-bearing, sorted, omitted when default.

| Name | Values | Meaning |
|---|---|---|
| `align` | `authored` | The alignment for this anchor was **emitted by a renderer** rather than computed from a parse, so the logical anchor is durable without a computed alignment map. A hint that must be verified against the derivation record — never trusted from the URI alone. |

`align=authored` exists because the write spine's renderers emit their
block→region mapping as a byproduct (`DOC-P2-017`), which is what makes
generated documents' logical anchors durable. A reader of an attestation
benefits from seeing that; a resolver still checks the record.

## Minting And Parsing

`anchors` owns both directions. The verbs are `document-manager anchors
cite` (mint) and `document-manager anchors resolve` (parse and resolve),
both proto-backed with CLI parity per `DOC-P0-020`.

Minting rules:

1. Kind is **what the handler chain actually proved**, never what the
   format could support at best. A PDF parsed without its geometry
   handler mints `logical`.
2. The derivation version is the one that produced the unit, never
   "latest".
3. Emit canonical form only. There is no non-canonical minting path.
4. One anchor names one region. A citation spanning two units is two
   anchors — ranges do not span the structure.

## Adding An Anchor Kind

1. A new kind needs a genuinely new **coordinate system**, not a new
   format. A transcript's time anchor (`DOC-P2-003`) is the likely
   fourth; a new document format almost never is.
2. Adding a kind is additive per the anchor-format rule in
   [`../concepts/DATA.md`](../concepts/DATA.md): existing anchors keep
   resolving, and the resolver gains a dispatch branch.
3. Define its canonical form in the table above **before** any minting
   code exists, or the first implementation defines it by accident.
4. State whether it is durable unconditionally (like `geometric` and
   `tabular`) or only through an alignment (like `logical`). This is the
   single most consequential property of a kind.

## Adding A Hash Algorithm

The `hash-alg` prefix exists so the digest can change without a scheme
version bump. Adding one is additive: new documents mint under the new
algorithm, existing URIs keep resolving under the old one, and the
resolver dispatches on the prefix. Never re-mint existing URIs — a
re-minted citation is a different string, which breaks every published
`import_key` that referenced it.

## Versioning This Scheme

The leading `<scheme-version>` covers **breaking** grammar changes only.
Additive changes — a new kind, a new hash algorithm, a new attribute — do
not bump it, because a parser for version 1 must reject what it cannot
understand rather than guess, and additions are already rejected safely
by the kind and algorithm dispatch.

If version 2 is ever needed, version 1 URIs must keep resolving forever.
They are in an append-only journal that does not rewrite history.

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — anchor storage, alignments, and the anchor-format rule
- [`format-matrix.md`](format-matrix.md) — which anchor kind each source shape can carry
- [`render-matrix.md`](render-matrix.md) — why generated documents get durable logical anchors
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — the ledger contract this maps onto
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the anchor-kind and anchor-URI decisions
- `packages/proto/schemas/source-ledger/v1/journal/journal.proto` — `ImportProvenance`
