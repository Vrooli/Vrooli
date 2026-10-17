# Security — Switchboard

## Purpose Of This Document

The security posture of this scenario: what data is sensitive, who may do what,
where secrets live, what an attacker would try, and what is not yet defended.

**Read this before building any domain.** This is the highest-risk scenario in
the ecosystem, and not by a small margin. Every other scenario is entered by a
person, an agent, or another scenario, through a surface Vrooli controls. This
one is additionally entered by **unauthenticated strangers, over transports
Vrooli does not own, with content that reaches a system able to run a terminal.**
Nothing else in Vrooli has that property.

## Data Sensitivity

| Data | Sensitivity | Notes |
|---|---|---|
| Message bodies | **Highest** | Private correspondence, routinely involving people who never agreed to any of this and cannot consent — the other participants in a group thread |
| Media | **Highest** | May carry more than the sender intended, location metadata being the common case |
| Contact addresses | High | Phone numbers, Apple IDs, workspace identities — personally identifying on their own |
| Trust assignments | High | Reveal the owner's social graph and their judgement of it |
| Scope resolution log | Medium | Records what was permitted, never what was said. Deliberately minimal |
| Channel descriptors | Low | Capability declarations, no secrets |
| Credentials | **Never stored** | Only a reference to the credential authority. A write carrying a credential value is rejected rather than scrubbed |

## Auth And Authorization

**Two distinct questions, deliberately answered by different mechanisms.**

*Who is the operator?* — `scenario-authenticator`. RS256 tokens verified locally
against the published JWKS through `api-core/owneridentity`. This scenario issues
no API key, stores no password, and owns no profile table.

*Who is this stranger who just texted us, and what may they reach?* — the `trust`
domain. This is not authentication and must never be confused with it. A sender
is **identified** by a channel-scoped address, which is an assertion by the
transport, not a proof. Tiers express how much the owner has chosen to trust that
assertion.

| Tier | Established by | May reach |
|---|---|---|
| `owner` | Verified identity, never promotion | Everything the agent's grant permits, with owner-only scopes gated by one-time approval |
| `trusted` | Explicit operator assignment | Read-level scopes; program execution only by approval |
| `known` | The owner recording a reply relationship | An allowlist of read scopes |
| `stranger` | Default for any unrecognised sender | Conversation only |

**Three structural rules.**

1. Effective scope is the narrowest of sender tier, thread ceiling, and agent
   grant, resolved **before the agent reads the message**. Resolving afterwards
   means the model has already seen content it may not act on.
2. An owner-only scope has **no representation reachable from a lower tier**.
   This is construction, not configuration — there is no setting that produces
   one, so no misconfiguration and no prompt can reach it.
3. Trust is enforced in the `trust` domain, **not in middleware**. A middleware
   filter is bypassed by every new call path added later; scope resolution before
   dispatch is not.

## Secrets

| Secret | Where it lives | How it is reached |
|---|---|---|
| Channel credentials (bot tokens, OAuth tokens, provider keys) | The credential authority | Referenced by identifier. Never persisted here, never logged, never returned by any read endpoint |
| `vrooli-bridge` dispatch token | `secrets-manager`, injected by the lifecycle | Authenticates dispatch under this scenario's consumer identity |
| LPBS metering credential | The credential authority | Used only for reserve/execute/finalise |

Message bodies must never be written to logs. This is stricter than the usual
"do not log secrets" rule, and it is easy to violate accidentally in a debug
statement on the ingress path.

## Threat Model

| # | Threat | Vector | Mitigation | Status |
|---|---|---|---|---|
| T1 | **Prompt injection from an inbound message** | Any stranger who learns the handle sends crafted text intended to make the agent exceed its scope | Scope resolved before the model reads anything; owner-only scope structurally unreachable; an injected instruction can at most attempt what the tier already permits | **Partially mitigated — see Gaps.** The scope ceiling holds. Everything *inside* the ceiling is undefended |
| T2 | **Cross-member disclosure in a group** | An answer one member's tier unlocks is rendered into a room where a lower tier is present | Thread ceiling is the lowest tier in the roster; a scoped answer goes to a direct thread or is refused out loud | Designed, unbuilt |
| T3 | **Wallet drain by a non-owner** | Anyone in a thread spends the owner's metered credits by talking | Per-thread spend cap and hourly turn budget, both fail-closed | Designed, unbuilt |
| T4 | **Agent-to-agent loop** | Two agents in one room exchange metered turns indefinitely | `author_kind` marks agent-authored messages; no turn starts in response to one | Designed, unbuilt |
| T5 | **Replay / duplicate charge** | A transport redelivers a message and it is answered and billed twice | Ingress de-duplication on `(channel_id, remote_message_id)` before any run | Designed, unbuilt |
| T6 | **Address spoofing** | A sender asserts an address belonging to a higher tier | Addresses are transport assertions, not proofs. Tier changes require explicit operator action; `owner` is never reachable by promotion | Designed |
| T7 | **Exfiltration through media** | An agent is induced to attach private content to a reply | Egress is scoped by the same resolution as the turn; media is bounded by descriptor limits | Partially designed |
| T8 | **Descriptor tampering** | A malicious or malformed descriptor widens a channel's declared capabilities | Descriptors are schema-validated at boot; an unrecognised `schemaVersion` fails boot loudly rather than being coerced | Designed |
| T9 | **Hostile content in the console** | Message bodies rendered as markup in the operator UI | Message bodies are untrusted input and must be rendered as text, never as markup, and never into a template that interprets them | Designed |
| T10 | **Third-party relay exposure** | A hosted iMessage or SMS relay reads private conversations | Rejected as a default path; only ever an explicitly labelled fallback stating its trade-off at the point of purchase | Decided |
| T11 | **Cross-node dispatch abuse** | A compromised caller reaches a Mac node through the bridge | `vrooli-bridge` executes only allowlisted, typed CLI verbs under per-node scopes — never arbitrary shell | Inherited from bridge |

## Security Gaps

Stated plainly. Each is a real gap, not a formality.

1. **Runtime injection defence has no owner.** T1's ceiling holds, but nothing
   detects or resists manipulation *within* a tier's permitted scope.
   `prompt-injection-arena` looks like the owner and is not: it is stale, it
   predates the `api-core` layout, and it is shaped as an offline tournament that
   ranks models against an injection library rather than as a runtime guard.
   **It must be redesigned from its own operational targets before any non-owner
   tier is exposed in production.** Until then, exposing a handle to anyone but
   the owner is an accepted-risk decision the operator makes explicitly, not a
   default.
2. **Address identity is an assertion, not a proof, on every channel.** SMS
   sender identity in particular is spoofable at the network level. No tier above
   `known` should be assigned to an SMS-only contact without an out-of-band
   confirmation.
3. **The local `spend_ledger` can drift from LPBS.** Cap enforcement reads the
   mirror, so a drift window exists in which a cap under-counts.
4. **The iMessage ingress path reads a local message store with broad access to
   the owner's messages**, including conversations that have nothing to do with
   any agent. Scoping that read to bound threads is a design obligation, not an
   optimisation.
5. **No rate limit on unknown senders yet.** A stranger cannot exceed
   conversation scope, but they can consume metered inference until the thread
   cap trips. A per-address admission limit is wanted below the thread cap.
6. **Media is stored as received, metadata included.** Deliberate — stripping it
   would destroy evidence the owner may want — but it means inbound media can
   carry location and device identifiers into scenario storage.

## Cross-References

- `docs/concepts/FLOWS.md` — F2 trust resolution, F5 group arbitration
- `docs/concepts/DATA.md` — sensitivity and retention of each table
- `docs/concepts/DOMAINS.md` — why `trust` is a domain rather than middleware
- `docs/internal/DECISIONS.md` — the structural decisions cited above
- `docs/internal/PROBLEMS.md` — these gaps as tracked entries
