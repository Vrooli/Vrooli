# Temporal Flows

## Flow Index

| Flow ID | Domain | Risk | Model Status | Source of Truth | Tests | Remaining Gaps |
|---|---|---|---|---|---|---|
| `channelmanager.action-lifecycle.api` | channelmanager | High | Level 5 checked formal model | `api/internal/channelmanager/flow/flow.json` and generated transition table | `api/internal/channelmanager/flow` plus formal verifier | Browser completion still requires operator verification. |
| `channelmanager.activity-ledger` | channelmanager | High | Level 3 trace-backed event projection | `api/internal/channelmanager/ledger.go` | `api/internal/channelmanager/service_test.go::TestActivityLedgerPreservesProvenanceAndRedactsSensitiveReferences` | Add a declarative ledger-event spec when event types materially expand. |

## Audit Notes

- 2026-07-29: The account timeline is an append-only evidence stream, not a second action scheduler. `browser.dispatched` and `action.completed` are distinct events so an operator can distinguish BAS execution from manual verification. Actions retain the formally checked lifecycle as their mutable projection.
- 2026-07-29: The ledger redacts credential-, cookie-, password-, and token-shaped details and artifact references before storage and presentation. Browser session material remains BAS-owned protected state.
