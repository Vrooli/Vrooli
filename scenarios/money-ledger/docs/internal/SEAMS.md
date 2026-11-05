# Money Ledger seams

| Seam | Interface | Owner | Failure contract |
|---|---|---|---|
| Routed database | `*database.RoutedDB` | API composition root | routed test pools remain isolated; database errors are internal failures |
| Clock | `schedule.Clock` / injected `func() time.Time` | API composition root | tests use a fixed clock; persisted timestamps are UTC |
| Adapter boundary | `api/internal/ingest.Store` | ingest domain | adapters emit typed events only; failures return receipts and availability |
| Journal admission | `ledger.Store.Ingest` | journal domain, called by ingest | adapter id + external id is idempotency key; no update/delete |
| Secret/config boundary | environment and scenario storage resolver | lifecycle | missing configuration is reported, never guessed |
| Connect transport | generated `*_v1connect` handlers | handlers | typed proto request/response and Connect status |
| CLI manifest bindings | `cli/manifest.json` | CLI composition root | every shipped RPC has a command binding or explicit omission |
| UI transport | generated TypeScript service descriptors and `createScenarioConnectTransport` | UI API layer | query errors remain visible; empty position is not zero |

The adapter boundary is deliberately the most volatile seam. Bank passwords are not accepted; manual entry, CSV export, and future aggregator APIs all normalize through the same contract.
