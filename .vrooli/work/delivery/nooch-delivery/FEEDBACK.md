### NOOCH-AUTH-REMOVAL-20261017 — current explicit operator direction — ADOPTED

> “Nooch requires NO authentication and it must be FULLY removed.”

This direction supersedes the earlier AF-002 connected-authentication work,
authentication clauses in the frozen product baseline, and any instruction to
retain Scenario Authenticator for Nooch. It applies only to
`scenarios/nutrition-planner`; do not change authentication for other apps, the
platform, or the operator's account. Remove authentication from every Nooch
entry point, including the service manifest/public `/api` boundary, API
middleware and handlers, UI flow, the registered
`nutrition-planner.itsagitime.com` Cloudflare Access/Tunnel policy, and
generated/installed serving configuration. Remove the edge gate only for
Nooch's own hostname. Do not replace it with another sign-in, a fabricated principal
or token, hidden header, agent grant, or development-only bypass.

The data model must retain existing Nooch workspaces and records without
caller-supplied owner identity. Do not merge existing owner/workspace data or
silently switch to a new empty anonymous workspace. Anyone who can reach the
intentionally unauthenticated Nooch URL can access the Nooch data that the app
serves; do not describe it as multi-user-isolated or authenticated. This does
not waive any product, safety, budget, connected-journey, visual, or acceptance
gate. First reconcile the current owner/workspace mapping, then remove the
Nooch-only enforcement through its owning files and verify the exact serving
build without a sign-in prompt or authentication error.

### RELIABILITY-NOOCH-RETURN-20261017 — reliability coordinator — superseded by ROOT-20261019-NOOCH-RECEIVING-AND-STALE-GATES

The shared native AM interactive route now accepts the existing approved authority: on-demand supervisorafe095ec received its prompt and durably completed its assessment, and original BAS E46 childd2e88109 has native_verified goal delivery. Native A99 management repair adopted19:52:30; the exact previously refused RCLwake now succeeds without new permission. This qualifies shared infrastructure, not an unattempted E30 child or product gate.

Exact independent A97 readback of failed E30 attempt02e08dc1-0ef8-4a4c-9fb2-9b309e2220ce says no Create submitted. Native ledger has one stored user input, two lifecycle events, one log, zero assistant/tool activity; no session/start/result. The stored input is not WC delivery. The unavailable diff does not prove absence of unrelated repository edits. Sources: docs/internal/delivery-reliability-research/sources/A97-nooch-old-attempt-native-readback.md and adjacent sanitized JSON.

On this retained original owner16d59037/task27847f20, reconcile changed inputs and preserve E30 task51b75231-d745-4279-9aed-435b43c439a4 and all original gates. Its +832 total runtime (+183 handwritten/+649 generated) and +192 tests are historical 2026-10-14 D1 evidence. A198 changed eight admitted paths afterward and the original 52-path byte baseline is unavailable; current G6/G7 are unreconciled. Keep +450/+600 unchanged and do not infer a current count or a 382-line reduction. No generated hand edits, cap waiver, broad feature expansion, new leader/task, or replay merely because task says queued. The 2026-10-19 signal 9 authorizes bounded E1 functionality on the same owner/task/profile while preserving original gates. Native admission, not a Planner lease or per-step Root confirmation, governs any isolated browser context. Ordinary authorized maintenance needs no invented grant; human-only review boundaries remain.

# Nooch delivery feedback

## Current supervisor direction — acknowledged 2026-10-14

- **SUP-NATIVE-RESUME-20261014:** Native consent approved for existing-run management/investigation only. Existing leader resumed through retained identity; no wider authority implied.
- **SUP-OWNER-APPROVAL-20261013:** Owner-approved continuation of the five existing efforts within their native profiles, budgets and safeguards. Preserve historical verdicts and attribution gaps.
- **SUP-FOLLOWUP-20261013:** Continue the existing Nooch effort at `nooch-selected@2026-10-03:b013666f214be361`; qualify the W3 startup repair; resume admissible connected/E30 work individually. Preserve E3 and all acceptance gates. No Authenticator restart or human-only login inference.
- **SUP-STORAGE-RECOVERY-20261013:** Consume storage-manager unit PASS `20261013-231238-5ce819c4` and Nooch storage PASS `20261013-231714-b3c867ad`. The earlier failed run is historical; two `SQL_DB_HANDLE_CAPTURE` warnings remain distinct from startup repair.

Receiver qualification: Nooch now serves build `sha256:7f48d443cc5f74ddac4c419859e31913593ca1019bdd0782799fb4c81551a8a0`; lifecycle health/readiness and database connectivity pass, freshness is clean, and running API binary SHA256 `e7471054516e3340ce00ba12049b155ea47242621ef6832ceceeecaac3c6903f` matches `/proc/3352953/exe`. Authenticator remained healthy on the same PIDs. E30 child creation against its existing task/profile/parent was refused: `run claims do not match the retained parent identity`; no retry, route substitution or authority change. Keep E30 and connected work open under their current gates.

## Open

- **NC-REC-001:** Exact-build visual/connected capture still needs a supported capture route and authorized existing fixture/session. Original reference hashes and E3 boundary are preserved.
- **AF-002 (superseded for Nooch by `NOOCH-AUTH-REMOVAL-20261017`):** The isolated connected-auth fixture remains historical evidence only. It does not prove that Nooch authentication has been removed. Other E1/E2/E3 journeys and visual acceptance remain open.
- **AF-001 (superseded for Nooch only):** The former authenticated identity boundary is no longer a Nooch requirement. This does not alter identity boundaries for other applications or platform services.
- **VF-001:** Exact-build matched visual comparisons and justified adaptations remain open.
- **OC-001 / E3:** Await owner occurrence/allocation and revision-safe undo contract plus supported capture route.
- **OF-003:** Curated assets and app-owned generation remain subject to ND-024 and current budget/activation gates.

## Resolved

Verbatim resolved history: [through 2026-10-23](feedback/resolved-through-20261023.md). This move does not resolve any open feedback.
