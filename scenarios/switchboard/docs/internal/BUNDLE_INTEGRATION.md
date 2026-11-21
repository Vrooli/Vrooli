# Bundle Integration Status — switchboard

## Last Updated

2026-09-01

## Integration Status

| Area | Status | Notes |
|---|---|---|
| Shared identity | ⚠️ | Switchboard does not own accounts. Channel sender identity is transport-provided; authenticated LPBS consumer-session forwarding is not wired into channel dispatch. |
| Entitlements | ⚠️ | No local subscription replica or symmetric JWT verification exists. Cost-bearing inference is delegated to the existing `ai-gateway` path, which owns the LPBS session resolver. |
| Credit consumption | ⚠️ | `api/internal/metering/` has a tested authenticated LPBS reservation client and reserve/execute/finalize/release lifecycle, but no production call path from Switchboard dispatch. |
| LPBS registration | ⚠️ | The monetization declaration exists. LPBS download catalog registration has not been proven for this scenario. |
| Error handling | ⚠️ | The LPBS client maps non-2xx responses to bounded errors; a complete user-facing LPBS error-shape path is not yet wired into Switchboard handlers. |

## Gated Features Inventory

| Feature | Gate Type | Credit Cost | Idempotent? | Notes |
|---|---|---:|---|---|
| Conversational inference | Class A / LPBS | `ai_credits` estimate | LPBS reservation lifecycle | Must execute through `ai-gateway`; Switchboard must not charge from client-controlled code. |
| Local channel attach | Free / Class B capacity | none | n/a | Availability is derived from channel descriptor requirements and host facts. |

## Issues Found

- The Switchboard gate REST endpoint currently accepts `actor_id` in the request body; it is not an authenticated identity boundary. See `SWBD-PROB-011`.
- The LPBS gateway is reusable but needs a verified consumer access-token handoff before production wiring. See `SWBD-PROB-012`.
- LPBS's authenticated reservation routes derive identity from the bearer token. Do not add `user_identity` to outbound request bodies or a shared service secret.

## Priority Actions

1. Carry the verified LPBS consumer access token through the authenticated conversation boundary or delegate the complete inference call to `ai-gateway`.
2. Route capability-gate answers through authenticated channel envelopes or scenario-authenticator middleware.
3. Register and verify Switchboard's LPBS download catalog records through the delivery catalog service.
