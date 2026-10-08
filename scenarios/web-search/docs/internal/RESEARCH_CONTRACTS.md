# Research contract

This document is the transition reference for the assurance contract in
`packages/proto/schemas/web-search/v1/research/research.proto`.

## Admission

`query` remains the required identity for every request. Existing `top_n`,
`max_age_seconds`, `source_domains`, `minimum_sources`, `capture`, and
`finding_id` fields retain their first-pass meanings. New callers may provide
the typed `policy` and `questions` fields; the handler resolves those fields
before dispatch and applies the same bounds for direct and L3 requests.

The server accepts at most 20 questions, each with a bounded prompt and
token-safe identifier. Evidence admitted to a request is bounded by
`max_evidence_bytes`, with zero meaning the owner default. Unknown or future
assessment values remain unknown and cannot satisfy support.

## Result meaning

`supported` requires claim-level evidence references. `contradicted` records
that named evidence conflicts with the claim. `unresolved` records a known
question or claim gap. `unknown` means no adequate assessment exists. A
structurally valid URL, citation index, or response shape does not by itself
produce `supported`.

Question coverage is evaluated per question. A final answer may be answered,
partial, abstained, unavailable, refused, or failed; execution completion is
separate from research completion. Missing source observations and unavailable
owner transports remain explicit gaps.

## Wire compatibility

The first-pass fields remain in place and no field numbers were reused. The
typed assurance additions are additive: `EvidencePolicy` and
`ResearchQuestion` are request inputs; `EvidenceReceipt`,
`EvidencePassageRef`, `ClaimAssessment`, and `QuestionCoverage` are portable
result references. Local artifact paths are never part of these messages.

L3 receives the resolved policy and questions inside the Agent Manager-owned
workflow input. The workflow remains the owner of execution state and can be
waited on or resumed by its execution identity.
