# Progress — Document Manager

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/document-manager/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.
The table keeps milestone claims and explicit limitations; consult the archive before resuming historical work.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-17 | codex | done | **P0 closeout operator proof and final validation.** The installed `resource-doc-parse` CLI initially could not discover the checked WASI artifact from its generated build metadata; `cli/internal/discovery` now recovers the resource root from the sibling `.build.meta`, with regression tests, so the normal control-plane-installed command verifies and executes the checksum-verified `artifacts/doc-parse.wasm` without a shell-specific `RESOURCE_ROOT`. Verbatim proof on the running scenario: `resource-doc-parse classify resources/doc-parse/testdata/corpus/record-pdf-1.pdf` returned `format: pdf`, `pdf_type: TextBased`, `terminal_state: parsed`; `document-manager intake ingest` accepted `record-pdf-1.pdf`, `record-docx-1.docx`, `table-1.xlsx`, and `gate-rung-mapping.html` into confidential collection `c8831238-13fc-4ade-977c-6bea36806234` (the HTML fixture was conservatively detected as `application/xml`); `corpus export` returned `format: vrooli-document-corpus+json;version=1` and an archive containing four documents; importing the same archive under `P0 Operator Proof Reimport` returned `documents_imported: 4`; `document-manager enrichment embed ... --privacy confidential --json` returned `Error: unavailable: embedding unavailable` with exit code 1; and `GOWORK=off go test ./internal/gatewayreq -run Test -v` passed `TestConfidentialRequestFailsClosedForRemoteProfile` and the exclusivity check. Deferred as explicit platform debt: the unrefreshed baseline producer reports `not_found` for the plan's v3 collection name, and only Linux amd64 execution is exercised for the conditional desktop profiles. |
| 2026-08-05 | claude | done | **Scenario re-created from scratch.** Generated from `react-vite` v1.6.5 with the `vrooli-default` design kit, replacing three retired scenarios (`document-manager`, `secure-document-processing`, `data-structurer` — see [`DECISIONS.md`](DECISIONS.md)). The prior `document-manager` tree is parked at `scenarios/document-manager-retired/` pending removal. Remaining five post-hooks run by hand, all green. |
| 2026-08-06 | claude | done | **Boundary correction: this scenario is the ledger's sibling, not its ingestion half.** A design review found the 08-05 charter had the storage boundary wrong in a way that would have been expensive after code existed. |
| 2026-08-07 | claude | done | **Amendment pass: cleared the drift the `anydoc` decisions left behind, and closed four open questions.** A review found the PRD carried **six** superseded statements, not the three `PROBLEMS.md` had logged — `OT-P0-009` and `OT-P0-019` still described a two-value anchor enum after `tabular` was added, and `OT-P1-001` still called ordinary scanned pages tier 3 after Tesseract was placed at tier 2. |
| 2026-08-07 | claude | done | **Designed the generation (write) spine end to end, and scaffolded none of it.** A design review asked whether document *generation* belongs here. |
| 2026-08-07 | claude | done | **Specified the anchor URI — the contract between this scenario and the ledger — and reading the real proto corrected the premise.** The gap filed earlier the same day assumed the ledger's provenance was a single opaque string. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
