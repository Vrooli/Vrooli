# Error Handling

## Shared contract

Read the [Error Handling shared guide](../../../template-manager/docs/internal/ERROR-HANDLING.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

Record scenario-specific domain errors, recovery behavior, and transport
exceptions here when they differ from the shared pattern.

## Safety review outcomes

The content reviewer has three distinct outcomes, and they must not collapse into
one wire response:

| Outcome | Job | HTTP (submit edge) | Counts against the model? |
|---|---|---|---|
| Genuine refusal (`action=refuse`) | `failure_kind=refused`, refused bytes deleted | 403 `forbidden` | No |
| Reviewer unavailable (timeout, crash, missing weights, invalid output) | `failure_kind=safety_unavailable`, unreviewed bytes deleted | 503 `unavailable` | No |
| Reviewer configured but absent on a public deployment | refused before the job runs | 503 `unavailable` | No |

A technical reviewer failure is never a policy refusal. The job records
`safety_action=unavailable`, `safety_reason=reviewer_unavailable` (or
`weights_missing`), and the attempted evidence if any. The refused job record
keeps `safety_*` keys (action, reason codes, evidence, refused refs); it does not
keep output references or success-only cost totals.

## Reviewer feedback errors

`RecordSafetyFeedback` labels an existing verdict, so its refusals describe the
request rather than the reviewer:

| Condition | Code |
|---|---|
| Job id is unknown | `404 not_found` |
| The job has no reviewer refusal/flag, no structured evidence, or the `output_ref` was not in the reviewed batch | `412 failed_precondition` |
| The label is unspecified or the note exceeds 1000 characters | `400 invalid_argument` |
| The feedback store is not wired (a test-only module) | `501 unimplemented` |

A technical store failure is `500 internal` and is logged; it never changes a
stored verdict or restores an output.

Quarantine review adds two transport outcomes:

| Condition | Code |
|---|---|
| A pending quarantined entry expired before it was fetched | `410 gone` on the image route |
| A public deployment is missing or given the wrong admin token | `403 forbidden` on the image route; `403 permission_denied` / `412 failed_precondition` on the list RPC |

A missing or already-purged entry is `404 not_found`. Local deployments do not
require a token. A quarantine write failure during a refusal deletes the original
as the fail-closed fallback and records `safety_quarantine_error`; it is not a
separate HTTP outcome because the job has already failed or dropped the
variation.

## AI Gateway generation outcomes

- Image Tools makes one Gateway Submit and one Wait for each variation. It does
  not retry an ambiguous Submit or Wait.
- If the AI job reaches orderly FAILED/CANCELED after Gateway work, its
  `result_meta.gateway_execution_evidence_v1` value may preserve the execution
  ID, observed Gateway status, per-variation outcome, and optional known cost.
  Known zero is represented by `cost_state=known` plus `cost_usd: 0`; unknown
  cost uses `cost_state=unknown` and omits the amount.
- These incomplete jobs keep their FAILED/CANCELED state, empty `result_ref`,
  and no top-level cost total/provenance or successful result refs. A terminal
  Gateway failure can still include known cost; `outcome_unknown` always leaves
  cost unknown. Successful job totals continue to require every dispatched
  variation's provider-reported cost.
- Evidence is retained only when orderly terminal persistence succeeds. It is
  not checkpointed after each Submit, so process death before finalization can
  lose the receipt link. A terminal database write failure is not evidence of
  durable storage.
