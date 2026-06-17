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

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
