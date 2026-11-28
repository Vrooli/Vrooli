# Account-credit lifecycle API

LPBS owns account wallets, app caps, reservations, and settlements. Apps that
need a long-running provider session use this API with both an authenticated
LPBS user session and a short-lived service token carrying the
`lpbs:account-credit.lifecycle` capability. The service token source supplies
the app key. The app must not send an account ID: LPBS resolves the user's
default account from the verified user identity for reservation, and resolves
membership again from the reservation's stored account before later operations.

Service token header:

```http
X-Vrooli-Account-Credit-Authorization: Bearer <short-lived-service-token>
```

The lifecycle endpoints are mounted under `/api/v1/account-credits/reservations`:

| Operation | Method and path | Meaning |
|---|---|---|
| Reserve | `POST /` | Reserve a positive amount with an idempotent UUIDv4 `operation_id`. |
| Begin dispatch | `POST /{id}/dispatch` | One-time owner transition authorizing the provider call. Dispatch only when `dispatch_authorized` is true. |
| Heartbeat | `POST /{id}/heartbeat` | Extend the pre-dispatch lease by 30 seconds to 5 minutes. |
| Settle | `POST /{id}/settle` | Record measured `actual_amount`, `cost_micros`, active/paused seconds, token counts, model, provider, and provider receipt ID after dispatch. |
| Release | `POST /{id}/release` | Release only while the owner state is `reserved`; a dispatch transition makes release impossible. |
| Retain uncertainty | `POST /{id}/retain-uncertain` | Keep a dispatched or settling hold pending for owner recovery. |
| Recover | `GET /{id}/recovery` | Read lifecycle phase, lease, and wallet settlement state. |

Reservation replay with the same operation ID and exact stored account, actor,
app, and amount returns the existing reservation with `created: false`; callers
must not dispatch on that response. Reusing the ID with different scope or
amount is rejected. Dispatch and release race through a conditional owner state
transition, so at most one succeeds. Once dispatch is authorized, LPBS removes
the expiry from the pending wallet hold. A lost provider response therefore
cannot make those credits available to another operation. The client must
settle from measured provider usage or retain the hold for reconciliation.

Settlement requires non-negative measured active and paused seconds, and
replays must match the same measured receipt. LPBS stores the provider receipt
ID and a digest of the complete measured settlement; conflicting replay data is
rejected. For GPT-Live, the session ID can identify the provider's terminal
usage record, but LPBS does not validate provider receipt authenticity and the
session ID is not an invoice or proof of billing. An uncertain hold stays pending until a measured settlement
or explicit owner recovery; it is not automatically released. The API does not
enable GPT-Live or make a provider call. A client must honor the dispatch gate,
heartbeat before dispatch when needed, send actual provider measurements, and
retain outcomes it cannot prove.
