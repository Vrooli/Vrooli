# Frozen bounty replay receipt contract

Test Genie owns adjudication for a sealed bounty replay. Workspace Sandbox owns
the source, candidate, and merged-reference snapshots; Agent Manager owns the
solver and scorer identities; Test Genie owns execution outcomes and the final
receipt. The candidate role must not receive the reference snapshot or scorer
result before its candidate seal is recorded.

## Required inputs

`test-genie/internal/frozenreplay.Adjudicate` accepts a context, one `Receipt`
input, and a required `EvidenceVerifier`. The verifier must resolve every
`run_id` and `producer_receipt` through Test Genie's owner record and compare
the retained artifact digest, result, evaluator, and source identity. The
adjudicator rejects caller-supplied run claims that the owner cannot confirm.
The input contains:

- `case_id` and `project_id`, stable identifiers for the replay and upstream
  project family;
- `frozen_evaluator_digest`, the SHA-256 digest of the immutable test and its
  declared assertions. The producer intent must include a content root named
  `frozen-evaluator` containing that exact file, so the digest is independently
  present in Test Genie's admitted source manifest;
- `preimage_identity`, `candidate_seal_digest`, and `reference_identity`, each
  a SHA-256 identity recorded by Sandbox for that exact variant;
- exactly three `evidence` records, one each for `preimage`, `candidate`, and
  `reference`.

Each evidence record carries the Test Genie `run_id`, immutable validation
`producer_receipt`, source identity, evaluator digest, `passed` or `failed`
status, and optionally the SHA-256 digest of the immutable owner receipt that
records the test outcome. If omitted, Test Genie derives that digest from the
owner receipt. The broker verifies that the evidence run belongs to its receipt, that
the durable run record is terminal and source-stable, and that its evaluator
digest appears in the admitted manifest's `frozen-evaluator` root. All run IDs
must be distinct; every source and evaluator digest must match the sealed input.
Missing or mismatched provenance is an error and cannot produce a qualifying
receipt.

## Producing a variant result

Use the existing durable validation lifecycle for each sealed variant:

1. Resolve `SourceIdentity` with `test-genie validation resolve-identity` using
   the exact variant content root and a `ContentInputRoot` named
   `frozen-evaluator` that includes the frozen test file. Save that resolved
   identity as the intent's `expected_identity`.
2. Submit the typed intent with `test-genie validation create`. It must target
   exactly one immutable variant workspace and include the frozen evaluator
   root. Test Genie records the immutable validation receipt and starts its
   durable run child.
3. Wait with `test-genie validation wait <receipt_id> --wait-id <unique-id>`.
   A timeout or detached observer does not cancel the run; attach again with
   the same receipt ID and a new wait ID. The final receipt child supplies the
   Test Genie `run_id` for the F9 evidence record.

The expected red is accepted only when the run record is source-stable, its
terminal verdict is `FAIL`, at least one failed phase is classified
`test_failure`, and the suite has no top-level `FailureReason`. Infrastructure,
system, setup, interrupted, and source-drift failures cannot count as a red
preimage. Runs for distinct workspaces can proceed concurrently; starting a
divergent run for the same target while one is active is refused by Test Genie.

## Decision and immutability

The only qualifying result is preimage `failed`, candidate `passed`, and merged
reference `passed`. Every other complete triple produces `not_qualified`; an
incomplete triple is rejected. `Adjudicate` normalizes outcome ordering to
preimage, candidate, reference, then computes `receipt_digest` as
`sha256:` plus the SHA-256 of the JSON encoding of the receipt with an empty
`receipt_digest` field. `Verify` checks this checksum after retrieval. It detects
content changes when compared with the retained digest; it is not a signature
and does not authenticate a writer. The owning caller must persist returned
receipts append-only behind the Test Genie owner boundary; a later evaluator
version is a new receipt and cannot overwrite an earlier outcome.

## Connect and CLI binding

The typed Connect methods are
`vrooli.test_genie.v1.frozen_replay.FrozenReplayService/CreateFrozenReplayReceipt`
and `.../GetFrozenReplayReceipt`. Generated clients and the Test Genie CLI
consume these methods. CLI commands are `test-genie frozen-replay create` and
`test-genie frozen-replay get <receipt_id>`; Program Runtime discovers the same
bindings from the manifest IDs `test-genie/frozen-replay/create` and
`test-genie/frozen-replay/get`.

Create takes an idempotency key and typed `FrozenReplayAdjudication`. A first
create returns a new receipt; an identical retry with the same key returns the
original receipt, and reusing a key with different evidence returns
`AlreadyExists`. Unverified evidence returns `InvalidArgument`; an unavailable
store returns `Unavailable`. Receipts have no update or delete operation, and
SQLite triggers reject direct row mutation. The receipt ID is the 64-character
hex suffix of its digest. A compatibility JSON path remains at
`POST /api/v1/f9/replays` and `GET /api/v1/f9/replays/{receipt_id}`.

The validation lifecycle above launches and resumes the three runs through
Test Genie's owner. The F9 adjudicator consumes only terminal receipts and does
not sandbox workspaces or reveal the reference. A failed/interrupted run that
does not meet the typed assertion-failure checks remains unqualified and must
be resumed or retried by the orchestration owner before a complete three-way
receipt can qualify.
