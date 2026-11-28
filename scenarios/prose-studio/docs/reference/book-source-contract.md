# Book brief and manuscript source contract

Prose Studio accepts a consumer-owned `book_brief` on its existing durable
document record. The v1 shape is in [`book-brief.schema.json`](book-brief.schema.json)
and the Go contract is `api/internal/prose/book_contract.go`. It captures the
title, creative direction, audience, reader promise, declared structure,
chapter intent, front/back matter intent, and typed production metadata. The
brief is stored with the document so document reads and resume retain it. Generation forwards the full
serialized brief to both outline and section prompts.

`evidence_boundary.approved_facts` is the only supplied support for concrete
factual claims; title, creative direction, audience, promise, and intent are
not evidence. Concrete claims are admitted only when the entire normalized
sentence exactly matches an approved fact; facts are not paraphrased or
extended. If the boundary or `unknown_details` is omitted, Prose declares
time, location, and source identity unknown in the prompt. `unknown_details`
accepts `time`, `location`, and `source_identity`. Requested field-card values
do not license invented values. Concrete time patterns, common source examples,
and named intersection/street patterns are rejected unless the detected detail
appears in an approved fact. This is a deterministic lexical boundary, not a
comprehensive fact checker; unfamiliar paraphrases, named-place forms, and
source vocabularies can evade it and require editorial review.

The canonical `ProseStudioService` protobuf carries `book_brief` on `Document`
for Create/Get and exposes read-only `ExportBookSource(id)`. The RPC returns
`manuscript-source/v1`: title and ordered chapter source, planned matter
intent, source brief, and plain-text reading order. It reads each committed
candidate by ID and fails closed if metadata is incomplete or any section is
absent or uncommitted. The
typed result maps title and ordered source parts to the existing shared book
schema. Prose Studio does not manufacture a second renderer schema;
NB9/shared engine remains the authority for figures, validation, rendering,
and HTML/PDF/EPUB packaging. A book brief deterministically creates ordered
front matter, chapter, then back matter slots from its intent lists. Each slot
is generated and committed as a typed section; export carries authored
`front_matter`, `chapters`, and `back_matter` parts plus one `reading_order`
array with section IDs, committed candidate IDs, kind, position, intent, and
text. It refuses export unless every requested intent has a matching committed
authored part. The separate intent arrays remain available in the brief/source
metadata. The engine may generate a title page from book metadata.

The shared NB9 book schema also requires a subtitle, language, physical trim,
and provenance. These are carried as typed `production_metadata` in the brief
and export. Source export refuses an incomplete brief instead of filling these
values from defaults. For AI-written manuscripts, generation runs carry the
request-query and output SHA-256, model, and date captured from the generation
session and selected candidate. `external_asset_refs` stays empty until an
approved asset workflow provides references; the NB9 adapter must resolve
references to its asset provenance objects and may not invent them. This is a mapping contract,
not a claim that prose-studio emits NB9's final validated book JSON.

Legacy article documents omit `book_brief` and retain their prior behavior.
This addition owns prose generation and source serialization only; Studio,
page maps, feedback and packaging are downstream consumers.

The declaration `book-manuscript-local-profile.json` pins the supported
`write.default` role to `local_only`, with one sampler and a maximum of 2,048
output tokens per section. This uses the Prose Studio section output floor;
the earlier 1,200-token override was below that truncation-tested default.
The adjustment preserves the 440-word chapter floor. It supports 1–16 ordered slots and 440–660 word
chapter bands (matter slots use 200–300 words). The local route's runtime
availability and manuscript quality have not been verified.

An opt-in `book-manuscript-local-quality-profile.json` pins the same book
contract to `chat.quality` with `local_only`; it preserves the same section
target, output cap, slot limit, and word gates. It is a separate declaration
and does not change `book-manuscript-local`'s `write.default` route. The route
was shape-validated without invoking a model; generated output quality remains
unverified.

Book section prompts serialize the approved brief once, then add the exact
current slot, kind, and word band. The deterministic brief already contains
all ordered intents, so the same section prompt omits a duplicate full outline
and repeated title/intent prefix. Each chapter part has a distinct sequence
purpose, and later parts receive the preceding part's final sentences as
continuation context. A lexical setup-action check rejects repeated opening
staging across parts; semantic restarts expressed with different vocabulary
are not comprehensively detected. The profile retains up to 1,024 tokens of
full prior prose and summarizes older sections when that budget is exceeded;
the summary role is explicitly `local_only`. Summary quality is not guaranteed,
and these deterministic prompt tests do not prove that the model will meet the
section word band. Output quality and local-route runtime remain unverified.

New book exports use `book-section-chunks/v2`: a deterministic plan assigns each
part a distinct purpose grounded in the complete brief and the current typed
slot. Parts use the existing `write.default` role with `local_only`; every
candidate remains durable with its own prompt/output hash and route/token
provenance. The owner checks part bands, filler-like repetition, pairwise
substance novelty, and explicit species-identification/wellness prohibitions
before assembly. It then combines the parts into one derived candidate and
re-applies the unchanged full-section word gate (440–660 for a 550-word
chapter; matter keeps its own 200–300 band). A failed check leaves no committed
canonical section. The committed section and `manuscript-source/v1` retain the
canonical text plus each part's section/candidate IDs, purpose, word count,
provider/model, token counts, and prompt/output hashes. These deterministic
checks catch obvious repetition and listed safety violations; they do not
establish literary quality or comprehensively judge factual grounding. A real
local generation still needs operator-side review before downstream
integration. No model was invoked to validate this implementation.

Before exporting AI-generated book source, Prose reloads the pinned profile,
durable section, canonical candidate, every derived part candidate, and their
round/session records. It recomputes text hashes and word counts from stored
text, verifies exact section and part ordering/purposes, checks the full
440–660 chapter band and part-level evidence, and reruns repetition and
evidence-boundary checks. It rejects stale or missing provenance and will not
export a committed canonical candidate whose parts no longer assemble to its
stored text. This path is intentionally stricter than generic manuscript
serialization. Explicit caller-authored non-AI source remains supported
without generated chunk records; it still requires a typed brief and committed
section text. Export-time checks use deterministic lexical repetition and
unsupported-detail rules, so paraphrased semantic restarts or unfamiliar
time/place/source forms can still evade them. The grounded positive fixture
uses only its explicitly approved fictional facts; it demonstrates contract
acceptance, not model output quality.

When a section candidate fails eligibility, the returned error receipt records
the generation session ID, candidate ID, provider/model, output-token count,
measured word count, gate reason, and SHA-256 of the prompt components. Candidate
text is deliberately excluded from the receipt; the candidate itself remains
in the durable Prose record for owner-authorized inspection. The prompt hash is
for correlation and does not expose prompt text.
