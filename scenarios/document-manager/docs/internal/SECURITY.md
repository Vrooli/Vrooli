# Security — Document Manager

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

This scenario is **designed to hold regulated data** — PHI, privileged
legal material, personal information. That is a product feature rather
than a caveat, and the classification below is therefore load-bearing
rather than descriptive.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Original bytes | **as classified; up to secret** | intake | The document itself. Carries the `PrivacyClass` that selects the AI Gateway routing profile. `regenerable: false`. |
| Derivations, units, enrichments | inherits the document | derivation, anchors, enrichment | Derived text is as sensitive as its source. A summary of a privileged document is privileged. |
| Embeddings | inherits the document | enrichment | Vectors are not anonymised source. Treat them at the source's class; never pass them across a scenario boundary. |
| Detections (PII/PHI) | **at least as sensitive as the document** | sensitivity | A detection record names *where the sensitive material is*, which is a targeting aid if leaked separately. |
| Redactions + manifests | audit-sensitive; outlives the document | sensitivity | A manifest records what was removed and why. It is evidence, and it survives document deletion deliberately. |
| Custody records, access events | audit-sensitive | custody | Append-only. Reveals who read what and when — sensitive on its own terms, independent of document content. |
| Collections | metadata-sensitive | corpus | A collection name can disclose a matter, a patient or a deal. Carries `default_privacy_class` and the `federated` flag. |
| Specs (P2) | **inherits the generated document; often higher** | composition | A spec is the *reasoning* behind a document — sources, unused blocks, prior versions. It routinely contains more than the artifact shipped. |
| Source resolutions (P2) | inherits the binding's origin | composition | Snapshotted values from `command-center`, `content-desk` or a corpus anchor. Unreleased figures live here before they live anywhere else. |
| Templates (P2) | low, but brand-controlled | templates | Presentation only, referencing brand tokens. Carries no document content by construction — a template holding literal content is a defect. |
| Rendered bytes (P2) | inherits the spec | render | `regenerable: true`. The artifact someone was actually shown, which is what a citation points at. |
| Chat transcripts (P2) | **inherits the document** | composition (custody) | An agent turn about a confidential document is confidential. Recorded as custody events, subject to the same routing gate as any other inference. |

## The Central Control

One control carries the product claim, and everything else in this
document is secondary to it: **every outbound `GatewayRequest` is
constructed at the single `internal/gatewayreq` choke point**, which maps
privacy class to routing profile and refuses to emit a profile weaker
than the document's class (`DOC-P0-026`).

It sits where it does because the gateway's fail-closed behavior is a
property of the *profile*, not of the privacy class:
`PROFILE_LOCAL_FIRST` is documented to fall back to a permitted remote
provider, so a confidential document sent under the wrong profile routes
remote and the gateway is behaving *correctly*. A second construction
site anywhere in the tree voids the residency guarantee silently — no
test fails, no finding fires, the document just leaves. An AST check
asserts exactly one exists.

**The write spine is inside this control, not beside it** (`DOC-P2-023`).
The Composer's agent chat is an inference caller, and it is the one path
that would otherwise reach a confidential document without passing the
mapping. Routing it through the same choke point is what produces the
claim: *you can talk to your privileged documents and nothing leaves the
machine.* The default render path constructs no `GatewayRequest` at all.

## Auth And Authorization

No auth provider ships today; the scenario is single-operator and
lifecycle-managed. Two obligations are already recorded and must not be
retrofitted casually:

- **Per-collection access control** (`DOC-P1-014`) with every access
  event written to the append-only audit trail.
- **Privacy-filtered retrieval** (`DOC-P0-024`) — a unit never surfaces
  in a query whose caller cannot read its collection or privacy class,
  asserted as a failure rather than a filter applied late.

UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer. On the write spine, the
agent is **document-scoped by default**: a corpus-scoped action requires
an explicit, separately authorized scope binding (`DOC-P2-025`), so
blast radius and authorization move together.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None held by this scenario | n/a | no | Provider credentials, URLs and model slugs belong to AI Gateway; parsing and rendering runtimes are reached through resource CLIs. A provider HTTP client anywhere in this tree is an `ai-conformance` ERROR. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Confidential document routed to a remote provider | Voids the entire product claim; likely a regulatory breach for the buyer | Class→profile mapping at one choke point, sensitivity resolved *before* tier selection, AST check for a second construction site, and an explicit test asserting a confidential document fails closed | designed, `DOC-P0-013` / `DOC-P0-026` |
| Federated query leaks restricted units | A `search-hub` query arrives anonymous; a per-caller filter has nothing to filter on | Per-collection `federated` opt-in, default off, with a ceiling no flag overrides: confidential and secret units never federate | designed, `DOC-P0-018`; upstream gap recorded in `PROBLEMS.md` |
| Decompression bomb during intake | Storage exhaustion from a nested archive; every OOXML, ODF and EPUB file is a ZIP container | Declared depth, count and expansion-ratio limits on archive expansion | **unresolved** — named in `format-matrix.md` Gaps |
| Macro execution from an uploaded document | Arbitrary code execution | Macro-enabled formats are accepted and their macros parsed as **inert bytes**. Nothing in this scenario executes a macro, ever | designed |
| Custody trail edited or deleted | Evidence that can be altered is not evidence | Append-only repository with no `UPDATE`/`DELETE` path; document deletion writes a tombstone | designed, `DOC-P0-015` |
| Custody trail pruned by storage pressure | Same impact, arrived at accidentally | Custody kind declared without a prunable budget; `storage-manager`'s enforcer prunes any budgeted `kind=dir` regardless of intent | **watched** — see `PROBLEMS.md` |
| Automated redaction without review | Not defensible to a regulator; a wrong redaction is unrecoverable in the shipped copy | Proposals never auto-apply; explicit human confirmation before finalize; stale confirmations rejected | designed, `DOC-P1-010` |
| Agent chat reaches a confidential document off-gate (P2) | Same as the first row, through the newest path | Chat routes through `gatewayreq` under the document's class and fails closed identically | designed, `DOC-P2-023` |
| Agent chat acquires a privileged path (P2) | Breaks parity, and creates an unaudited verb surface | AST check that the chat constructs only generated clients; every action is a public verb with CLI parity | designed, `DOC-P2-021` |
| A template edit silently restyles a corpus (P2) | One person's tweak changes documents they never opened, including ones already sent | Template edits are proposals requiring confirmation; per-document overrides apply directly and are reversible by spec version | designed, `DOC-P2-024` |
| A generated document leaks unreleased source values (P2) | A spec holds more than the artifact ships — unused blocks, prior versions, raw resolutions | Specs inherit the document's privacy class and are exported only inside the corpus export; render output carries only what the spec placed | designed, `DOC-P2-010` |
| Rendered artifact diverges from its spec (P2) | The record explaining a document no longer describes it; citations and diffs become wrong while tests pass | Spec authority boundary: an AST check asserts no composition verb writes rendered bytes | designed, `DOC-P2-010` |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Archive expansion has no declared bounds | **high** | Before `intake` accepts any ZIP-container format, which includes OOXML, ODF and EPUB. |
| No auth model or per-collection access control | conditional | Required before multi-user access, and a prerequisite for loosening the agent's document scope (`DOC-P2-025`). |
| Receipts are self-attested | medium | `RouteEvidence` carries no caller correlation key, so a receipt has no independent gateway-side corroboration. Upstream ask on `ai-gateway`. |
| Federation cannot serve the restricted corpus | accepted | Resolved by policy, not mechanism. Revisit when `search-hub`'s query contract carries a caller principal. |
| Custody retention horizon undecided | medium | A policy decision, not an engineering one. Needed before a legal retention limit applies. |
| Renderer supply-chain risk unassessed (P2) | medium | A render toolchain is a large dependency that consumes attacker-influenced spec content. Assess at selection, through `scenario-dependency-analyzer`, before the write spine opens. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
