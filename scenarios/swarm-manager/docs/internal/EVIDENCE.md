# Canonical evidence

Swarm Manager stores owner-neutral evidence in its SQLite event database. An
observation is immutable producer output; an evidence link attaches it to an
Agent Session or a workflow execution through the live owner index. Retired
operating-mode owner records remain readable only as historical provenance; a
zero-owner result remains unresolved and a dual-owner result remains ambiguous:
neither is guessed.

## Trust and completeness

`authoritative` observations come from the domain that committed a mutation.
`observed` observations come from bounded Agent Manager or CLI metadata.
`reported` observations are agent-reported declarations and are not promoted by
the ledger. `operator_verified` is reserved for an explicit operator repair
with actor and reason.

`swarm-review` is the producer for criterion-bound review observations. Review
agent narrative enters as `reported`; only bounded Test Genie, Git Control
Tower, or command artifacts are recorded as `observed`. Operator verification
and revocation are separate immutable operator observations, never a trust
upgrade of agent prose. The ledger projects the latest operator event for an
evidence item, so a later revocation or re-verification remains auditable.

Every review evidence item is bound to a stable criterion when it settles a
claim. The review result also contains one criterion verdict per declared
criterion, with the evidence identifiers that support it. `unavailable` is an
honest settlement only when it records both a specific reason and the
registered producer that was attempted; it is never a synonym for `refuted`.

Producers record a checkpoint only after their observations are linked. A
terminal watermark is scoped to producer, run, and fact kind. Therefore an
absent fact is `pending_evidence` until the requirement's producer supplies a
covering watermark; only then can the same absence become definitive
`missing_evidence` and put a round in `needs_attention`.

## Operating-mode gates

Phase data may declare `evidence_requirements`. Each requirement names a
normalized `subject_kind` and `action`, an optional producer, minimum
confidence, minimum count, and bounded metadata fields. The engine evaluates
the pinned phase data after output resolution and reconciliation, before round
completion. A late matching observation can move a prior evidence-specific
`needs_attention` round to completed on refresh.

Requirements without a producer can be satisfied by any producer, but absent
evidence remains pending: no unspecified producer set can establish a safe
negative result.

## Session projection

Session artifact APIs are a bounded projection over the ledger. Historical
`artifacts.jsonl` data is imported idempotently with its original timestamps
and trust level; migration does not upgrade legacy confidence. New attachments
write one atomic canonical-ledger batch and never append JSONL.

Before the projection is enabled, the importer verifies every `(session_id,
artifact_id)` pair has a ledger counterpart and writes the count plus stable
source/projection digests to `evidence_migration_audits` under
`agent-session-artifacts/v1`. A parity mismatch keeps the read-only JSONL
recovery path active. The local file is therefore a bounded recovery input,
not a second authority or an ongoing write target.

## Review migration and projection

Historical review rounds are imported under `review-evidence/v1`. The importer
records source and projection digests plus counts in `evidence_migration_audits`.
Until that audit proves parity, round-file verification flags remain the
recovery projection. After parity, the review surface derives verification
state from the latest linked operator verification or revocation observation.
