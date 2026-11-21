# Error Handling

## Shared contract

Read the [Error Handling shared guide](../../../template-manager/docs/internal/ERROR-HANDLING.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

Record scenario-specific domain errors, recovery behavior, and transport
exceptions here when they differ from the shared pattern.

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
