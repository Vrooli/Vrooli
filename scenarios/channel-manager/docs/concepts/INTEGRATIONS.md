# Integrations — Channel Manager

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | identities, platforms, warming, queue, signals | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| credential authority | native platform service | no at P0 | future executor (at execution time only) | Authority identity/field reference resolved through the control-plane credential contract | P0 manual work stores only a reference, so the console and test suite run without the authority. A future credential-consuming executor must fail terminally and never cache a value. |
| content-desk | scenario | no (inbound) | queue, identities | Connect-RPC: release handoff and eligibility query | This scenario does not call it at P0; it answers. If nothing calls in, warming and manual actions continue unaffected. |
| browser-automation-studio | scenario | no (P1) | queue (browser executor) | `workflows`, `executions`, `session-profiles` | The action degrades to the manual executor with the dispatch failure recorded. It is never marked complete on a dispatch error. |
| asset-studio | scenario | no (P1) | queue (asset uniqueness) | Connect-RPC: has this asset been published, and by whom | **Refuse, not allow.** A lookup failure blocks the post, matching the fail-closed posture used for eligibility. |
| vrooli-events | scenario | no | all | api-core receipt publication (automatic) | Run correlation is absent; nothing else is affected. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | active | Identity, queue, warming, and signal tables are single-writer, local, and modest in size. | If the scenario ever becomes multi-host. |
| credential authority | manual / executor-only | Canonical credential authority. This scenario holds a reference and never persists a value; P0 manual execution does not read it. | Enable only when a credential-consuming executor is approved. |
| browser-automation-studio resources | indirect (P1) | Reached through the BAS scenario, never driven directly. Per-identity session profiles are what keep browser state from leaking between identities. | If BAS stops exposing session profiles, multi-account browser execution is not safe and the executor must be withdrawn. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| content-desk | required (inbound) | Owns campaigns, drafts, claims, review, and approval. It hands this scenario an approved draft and asks whether an identity may carry a lane. | **Exactly two questions cross the boundary**, per that scenario's own `INTEGRATIONS.md`: *release this draft* and *is this identity eligible for this lane*. No account state, warming detail, or credential crosses in either direction. |
| browser-automation-studio | optional (P1) | The browser executor and per-identity session profiles. | Create a profile with `session-profiles create`; dispatch with `workflows execute <workflow-id> --parameters-file <file>`, where parameters carry both `session_profile_id` and `save_session_profile_id`. Full operator command sequence: `docs/internal/SEAMS.md`. |
| asset-studio | optional (P1) | Owns render provenance, so it is the only scenario that can answer whether an asset has already been published and by which identity. | Read-only lookup. Fails closed. |
| vrooli-events | automatic | Run correlation for actions taken inside an agent run. No integration work — api-core publishes receipts for every endpoint already. | Correlation ids only; run payloads are never copied. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Social platforms (TikTok, Instagram, X, LinkedIn, YouTube, Threads, Bluesky, Reddit) | indirect | Reached only through an executor — manually by an operator at P0, through BAS at P1, through official APIs at P2. **The API executors are P2 and unscoped**; no platform API tier has been checked for current availability. | Per-platform behaviour is declared in `data/platforms/<platform>.json`, never in code. |

## Marketing Canon — read-only input

These are not dependencies in the runtime sense, but changing them changes this
scenario's behaviour, and this scenario must never write to them.

| Document | Supplies | Rule |
|---|---|---|
| `docs/marketing/strategy/CHANNELS.md` | Which channels are active, account counts, and purpose tags per channel. | Explicitly forbids credentials and handles from living there and routes them here. This scenario reads it and never edits it. |
| `docs/marketing/strategy/patterns/ai-ugc-personas.md` | Which persona shapes are permitted and what disclosure each platform requires. | Drives `CHANMGR-P1-004`. The banned-claim set is enforced by `content-desk` at review time, not here — this scenario enforces the *disclosure marker*, not the content. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| credential authority | credential read error or timeout | The action fails terminally and is recorded as such. It is never marked complete, and no credential is cached to survive the outage. | `CHANMGR-P0-002` |
| content-desk (inbound release) | malformed draft, or an identity not eligible for its lane | Refused with a typed error. A refused release is never queued. | `CHANMGR-P0-014` |
| content-desk (inbound eligibility) | internal repository or descriptor failure while evaluating | Returns **unknown**, never eligible. The caller's contract states that a permissive default would post from an unwarmed account. | `CHANMGR-P0-013` |
| browser-automation-studio | dispatch error, or session profile unavailable | The action degrades to manual with the failure recorded. Never marked complete. | `CHANMGR-P1-001` |
| asset-studio | lookup error or timeout | The post is refused rather than allowed. Fail-closed, matching eligibility. | `CHANMGR-P1-003` |
| Platform (during execution) | rejected credential, rate limit, or content rejection | Classified as retryable or terminal per the platform descriptor. Retries consume cadence budget so a retry storm cannot breach the ceiling. | `CHANMGR-P1-005` |
| Descriptor file | malformed or schema-invalid JSON at boot | **Seeding fails loudly.** A descriptor is never partially applied or silently skipped — a skipped platform descriptor would remove a cadence ceiling. | `CHANMGR-P0-003`, `CHANMGR-P0-004` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
