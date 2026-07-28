# Integrations — Content Desk

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
| SQLite | embedded storage | yes | API, persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| social-media-scheduler | scenario | no (P1) | ledger, review | Connect-RPC: publish handoff, account eligibility | Approved drafts queue locally and are released when the scheduler returns; eligibility degrades to a recorded unknown, never a false eligible. |
| ai-gateway | scenario | no (P1) | claims | Connect-RPC: assisted claim extraction | **Extraction is optional by design.** Author-declared claims are the P0 path, so inference being unavailable never blocks a draft. |
| search-hub | scenario | no (P1) | ledger | `.vrooli/search.json` descriptor + registration | Local reads keep working; only federated query loses the editorial provider. |
| prompt-manager | scenario | no | posttypes, review | Read-only: post-type canon, team decisions, paired skills | Registry seeding falls back to the last seeded state; nothing blocks. |
| vrooli-events | scenario | no | artifacts | api-core receipt publication (automatic) | Run correlation is absent; drafting and approval are unaffected. |
| image-tools | scenario | no (P2) | artifacts | Connect-RPC: `ai` generation, `ops` resize/crop, `looks` style recipes | Image attachment is unavailable; drafting, verification, and approval are unaffected. Text-only posts never touch it. |
| agent-manager | scenario | no (P2) | artifacts | Connect-RPC: run a drafting or evidence-hunting agent from the UI | The UI loses commissioning; the CLI path that agents already use is unchanged. Never on the approval path. |

## Vrooli Resources

This scenario declares **no external resources**, and that is a decision
rather than an unfinished state. Everything it stores is text with a lifecycle:
campaigns, drafts, claims, evidence, review verdicts, and publish history. All
of it is single-writer, local, and small. Introducing a shared resource would
buy nothing and would add a failure mode to a surface whose whole job is to be
trustworthy.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | active | Domain tables are single-writer, local, and modest. No shared resource is warranted. | If editorial history outgrows embedded storage, which volume makes implausible. |
| ollama | indirect | Reached **through ai-gateway**, never directly, and only for P1 assisted claim extraction. | Never call inference directly from this scenario; the gateway owns model policy and routing. |
| vault | not-applicable | **No credential ever enters this scenario.** Account identity and platform tokens belong to the scheduler, which holds vault references. | Never. A schema change introducing a token column is a defect, not a new integration. |
| qdrant | not-applicable | Federated retrieval goes through the shared search stack used by every other provider, so this scenario's index shape matches the fleet. | If corpus size ever makes an embedded index untenable. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| social-media-scheduler | P1 | Owns publishing, account identity, warming, cadence, and credentials. The desk hands off an approved draft and records what comes back. | Two narrow questions only: *release this draft* and *is this account eligible for this lane*. No account state, warming detail, or credential crosses the boundary. |
| prompt-manager | P1, read-only | Owns the marketing team, its decisions, and the paired `x-<type>` skills that actually produce copy. | The desk reads post-type canon to seed its registry. It never writes team state or edits canon. |
| ai-gateway | P1, optional | Assisted claim extraction as a cross-check on author self-reporting. | Proposals only. Never an authority — extraction output does not satisfy the verification gate. |
| search-hub | P1 | Makes editorial history reachable from federated query without a router change. | Descriptor plus boot registration; the router holds no content. |
| vrooli-events | automatic | Run correlation for drafts written inside an agent run. **No integration work** — api-core publishes receipts for every endpoint already. | Correlation ids only; run payloads are never copied. |

### Boundaries this scenario deliberately does not cross

| Concern | Owner | Why not here |
|---|---|---|
| Generating draft copy | The paired `x-<type>` skills | Keeps strategic reasoning and executable procedure separate, and lets prompts change without a deploy. |
| Account identity, warming, cadence, credentials | social-media-scheduler | Marketing canon already routes these there. Warming state and the post queue are coupled; splitting them would put a cross-scenario call on every publish. |
| Image generation and deterministic editing | image-tools | It is already a capability. The desk asks for an image and stores the reference; it never implements generation, resizing, or style recipes (D-018). |
| Character and scene identity; multi-frame and video media | A future asset-production scenario | Narrowed by D-018, not removed. Single images with no continuity requirement are a call into image-tools; identity consistency across frames is a different problem with a different failure mode — visual drift and generation spend, not a false claim. |
| Visual identity, palettes, brand assets | brand-manager | Unrelated capability that happens to share the word "brand". |
| Strategic marketing canon | `docs/marketing/`, decision-gated | Judgement the operator owns. The desk holds state and gates, never strategy. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | This scenario reaches no external network service. Publishing goes through the scheduler, inference through ai-gateway, and all storage is local. That isolation is deliberate: the desk is the surface that decides whether a public claim is true, and it should not itself depend on anything that can be unavailable or wrong. | Add only if a hosted service is ever introduced — which would route through an owning scenario, not directly from here. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| ai-gateway (extraction) | call error or timeout | The draft stays workable with author-declared claims only. **Inference failure never blocks a draft**, because extraction is a cross-check and not the authority. | `CONTENTD-P1-007` |
| scheduler (eligibility) | call error or timeout | Recorded as unknown and surfaced to the operator. **Never resolved as eligible on failure** — a permissive default here would post from an unwarmed account. | `CONTENTD-P1-005` |
| scheduler (release) | call error, or no post id returned | The draft stays approved rather than published, and the failure is recorded. A retried release must not create a second publish record. | `CONTENTD-P1-006` |
| search-hub | registration failure at boot | Scenario starts and serves local reads; registration retries. Federated query loses the editorial provider until it succeeds. | `CONTENTD-P1-008` |
| Import source unreadable | path missing or permission denied | That source is skipped and reported; other sources import normally. A partial sweep is never recorded as complete. | `CONTENTD-P0-013` |
| Import source format changed | adapter yields zero items from a non-empty source | **Import aborts for that source rather than importing nothing silently.** A changed layout must surface as a failure, not as an empty diff that reads as "nothing new". | `CONTENTD-P0-013` |
| Claim check command fails to execute | non-zero exit, or command missing | The claim moves to stale, not verified. An unrunnable check is not evidence. | `CONTENTD-P1-001` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
