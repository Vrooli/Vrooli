# LPBS progress

## Sign-in and session hardening — 2026-09-17

Delivered durable browser-bound sign-in codes and links, native PKCE grants,
bounded/revocable customer and administrator sessions, tracking-free delivery
with janitor cleanup, administrator MFA and step-up policy, customer account
security, and customer/administrator WebAuthn passkeys.

Evidence includes focused Go and Vitest suites, the BAS workflow run
`20260917-011223-9f0aaff9`, customer browser proof in
`/home/matthalloran8/.vrooli/plan-artifacts/lpbs-signin-session-hardening/evidence/phase-12/customer-passkey-proof.json`,
and administrator browser proof in
`/home/matthalloran8/.vrooli/plan-artifacts/lpbs-signin-session-hardening/evidence/phase-12/admin-passkey-proof.json`.

Operator actions before production rollout: configure sending-domain DNS and
SendGrid authentication/webhook credentials, disable account-level click
tracking, set `ADMIN_REQUIRE_MFA=true`, enroll the first administrator factor
on `vrooli.com`, and configure any additional WebAuthn origins when `www` is
served. Google sign-in remains owned by `scenario-authenticator`.
