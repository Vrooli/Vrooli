# Decisions — Device Sync Hub

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-17 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-17 | Two distinct credentials: the **owner JWT** (scenario-authenticator) identifies the human; a **hub-issued device token** carries device trust. | A device on the network/tunnel must be unable to act until *paired*, independent of whether it holds an owner login. Owner identity and device trust are different questions. | `auth` middleware best-effort-injects owner Identity; owner-gated RPCs call `RequireOwner`. Pairing redeem/request are open (a joining device has no owner token yet). Device tokens (Phase 3) gate transfer reads/pulls. Only token **hashes** (SHA-256) are stored. | A real multi-user requirement, or a need to let a device act with only an owner JWT and no pairing. |
| 2026-06-17 | Pairing is **fail-closed** and **single-use**: code claim is a conditional SQL UPDATE (`WHERE redeemed_at='' AND expires_at>now`); a redeemed/expired/unknown code returns one generic invalid error. | Short-TTL codes resist online guessing; a probing oracle would leak which codes exist. Concurrency safety needs an atomic claim, not read-then-write. | `ClaimPairingCode` disambiguates via rows-affected. 15-minute default TTL. `ErrInvalidPairingCode` is deliberately vague to redeemers. | If pairing UX needs richer redeemer feedback, or the TTL proves wrong in practice. |
| 2026-06-17 | Revocation severs **locally first**; authenticator session revoke is best-effort. | The hub's own access is gated by the device token-hash → flipping trust to REVOKED cuts access immediately regardless of authenticator reachability (fail-closed). | `Revoke` sets `TrustRevoked` then calls `auth.RevokeSession`; an auth failure is logged, not fatal. v1 single-owner: `RequestPairing` derives the owner from existing rows, so the **first** device must bootstrap via the authenticated code path. | Multi-user trust, or a requirement that authenticator-session revocation be transactional with local revoke. |
| 2026-06-17 | **Realtime is SSE, not WebSocket** (Phase 3). | The realtime requirement (OT-P0-005) is one-directional server→client push: presence, item-arrived, pairing-request. Every client *action* already rides a Connect RPC, so a bidirectional socket buys nothing. SSE needs zero new dependencies (pure `net/http` + `http.Flusher`); `gorilla/websocket` would add a dependency requiring explicit permission. | `GET /api/v1/realtime/events` is an SSE stream authed by the device token (header or `?token=` for `EventSource`). The hub (`internal/realtime`) is an in-memory fan-out; event payloads are the proto-typed `realtime.Event` marshaled with protojson. PRD/DOMAINS prose still says "WebSocket" — the transport changed, the capability did not. | A genuine need for client→server streaming (e.g. live cursor/collab) or for binary frames; or multi-instance presence (then back the hub with Redis pub/sub). |
| 2026-06-17 | **Device-token auth** gates transfer + realtime (separate from owner JWT). | Transfers are device→device within the owner's trust group; the actor is a *device*, not the human session. | `internal/deviceauth` middleware resolves `X-Device-Token` (or `?token=`) via `devices.Authenticator` to a TRUSTED device and injects it; handlers call `RequireDevice` (fail-closed). Unknown/PENDING/REVOKED tokens are indistinguishably untrusted (no probing oracle). The two middlewares (owner + device) compose; each surface reads its own credential. | A surface that must accept either credential, or a need to scope device tokens to specific capabilities. |
| 2026-06-17 | **Retention is the only relay knob**; per-item Live/Held/Pinned + a global default, swept by a background purge loop. | Keeps the relay simple: one policy field decides lifetime. Live drains on delivery (short TTL bound for the undelivered case); Held auto-purges after 24h; Pinned persists until deleted. | `internal/transfer` stamps `ExpiresAt` at create; `RunPurgeLoop` (5-min cadence, started in `main.go`) removes due rows + their blobs and emits `ItemDeleted`. Defaults (24h Held, 10-min Live, 5 GiB owner / 2 GiB device quota, 1 MiB text) are `Config`-overridable. | A dedicated `settings` domain that persists owner-tunable defaults (currently compile-time/Config), or per-item delivery receipts replacing the short Live TTL. |
| 2026-06-17 | Two new **REST-exception reasons** added: `binary_download` and `event_stream`. | A streaming file download (opaque bytes + original filename) and an SSE event stream are transports proto/Connect cannot express — the symmetric twins of the existing `multipart_upload`. Adding a reason is the sanctioned extension pattern (another scenario already added `host_hook_glue`). | `internal/module` + `cmd/gen-endpoints` `validRESTReason` accept both. Transfer download uses `binary_download`; realtime `/events` uses `event_stream`. Response metadata stays proto-typed (`GetItem`, `realtime.Event`). | If a download/stream shape can be expressed in proto after all, prefer Connect. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
