# AUTH-02 finite-effort contract — DEPRECATED

> **Status: DARK and DEPRECATED (2026-10-15).** The finite-effort / kernel-broker / custody
> machinery this document described is gated behind `VROOLI_FINITE_ENABLED`, which is set nowhere,
> so none of it runs. The operator rejected it as too much friction.

The authorization role it was built for — letting unattended and human-spawned callers do costly
work without a human sitting there each time — is now served by three simpler mechanisms that share
one `OperationGrant` representation: **standing grants**, **ask-don't-block** approval, and
**human-session delegation**. See [`docs/concepts/AUTHORIZATION.md`](../../../docs/concepts/AUTHORIZATION.md)
for the authoritative model.

The dark broker **code** still compiles in the tree (it is structurally entangled with live
run-creation paths), so its removal is a tracked, separately-scoped follow-up — not part of this
documentation pass. The precise delete/keep map is recorded in
[`docs/internal/auth-consolidation/phase7-broker-deletion-DEFERRED.md`](../../../docs/internal/auth-consolidation/phase7-broker-deletion-DEFERRED.md).

Do not build new work against this contract.
