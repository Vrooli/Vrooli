# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario switchboard`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Switchboard adds the permanent capability of *being reachable*. It accepts a message that arrives from outside Vrooli — an iMessage, an SMS, a Telegram or Slack message, a call, or a thread inside its own app — resolves which agent that address belongs to, decides what that particular sender is allowed to ask for, and runs the agent inside those bounds. Every scenario in the ecosystem already produces something worth saying; none of them should own a transport, a thread, or a trust decision. Those live here once. The direction that does not exist anywhere in Vrooli today is *inbound*: `notification-hub` already delivers to a human and even sends iMessage from a Mac, but nothing turns an arriving message into an agent turn.

- **The distinction that justifies the scenario**: `notification-hub` owns one-shot delivery — a title, a body, an address, a receipt. Switchboard owns a *conversation*: a durable thread with a roster, media in both directions, reply-to identity, and a sender whose permissions must be resolved before a model reads a word. `agent-inbox` and `ai-chatbot-manager` are surfaces the operator opens; Switchboard is a surface that opens itself when somebody outside sends something. `channel-manager` owns outbound platform identities and posting cadence for social accounts and never receives a message addressed to an agent.

- **Primary users/verticals**: The machine owner, who creates agents and gives them handles. People the owner chooses to expose an agent to — family in a group thread, colleagues in a Slack channel, customers on a published number — each reaching a deliberately narrower agent than the owner does. Agents and scenarios that need a human in the loop and should not implement a transport to get one. Beyond Vrooli: operators who want a personal agent on their own phone number without routing their private conversations through a third party's servers.

- **Deployment surfaces**: Go API over Connect-RPC (the contract other scenarios call), a typed Go CLI with full headless parity, a React operator console (agent roster, agent authoring, conversations, contacts and trust tiers, channel catalogue), an in-app conversation and call surface that is itself an ordinary channel adapter, and a set of channel adapters bound to validated descriptors.

- **Value promise**: An agent that is *somewhere*, reachable the way a person is, and safe to give out. The differentiator no hosted iMessage or SMS agent product can copy is custody: the thread, the agent, and the credentials stay on the owner's machine, and reaching an Apple ecosystem means reaching the owner's own Mac through their own fleet link rather than a vendor's relay. Behind the message sits every scenario the ecosystem has, which is depth a chatbot with a tool list cannot match.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Channel descriptor registry | The system shall load every channel definition from validated descriptor files at boot, shall treat the descriptor file as the source of truth and the database as a cache, and shall fail startup naming the file and field when a descriptor is invalid.
- [ ] OT-P0-002 | Channel capability questions answered by descriptor | The system shall answer every capability, limit, and policy question about a channel by reading its descriptor, and shall contain no branch on a channel identifier above the adapter layer.
- [ ] OT-P0-003 | Channel adapter contract | The system shall define one bidirectional adapter contract covering connect, receive, send, and normalise, and shall bind an adapter to a descriptor by identifier.
- [ ] OT-P0-004 | Normalised message envelope | The system shall convert every inbound and outbound message to a single envelope shape, and no component above the adapter shall receive a channel-native type.
- [ ] OT-P0-005 | Ingress idempotency | The system shall discard a redelivered inbound message at ingress using the channel identifier and the remote message identifier, before any run is started or any metered call is made.
- [ ] OT-P0-006 | Durable thread sessions | The system shall persist every conversation as a thread carrying its roster, its messages, and its media references, and shall resume a thread across a restart without losing position.
- [ ] OT-P0-007 | Address to agent binding | The system shall resolve an inbound address, handle, or thread to exactly one agent binding, and shall reject an ambiguous binding rather than choosing one.
- [ ] OT-P0-008 | Agent profiles held by reference | The system shall read agent profiles from prompt-manager by reference and shall never store a copy of a descriptor it does not own.
- [ ] OT-P0-009 | Contact trust tiers | The system shall assign every known contact a trust tier of owner, trusted, known, or stranger, and shall treat an unrecognised sender as a stranger.
- [ ] OT-P0-010 | Fail-closed scope resolution | The system shall compute the effective scope for a turn as the narrowest of the sender tier, the thread ceiling, and the agent capability grant, shall resolve it before the agent reads the message, and shall refuse the turn when the result is empty.
- [ ] OT-P0-011 | Owner-only scope is unreachable from a non-owner tier | The system shall make it impossible to grant an owner-only scope to any tier below owner, by construction rather than by configuration.
- [ ] OT-P0-012 | Attenuated run dispatch | The system shall start every agent turn as an agent-manager run carrying the resolved scope through one-way attenuation, and shall never widen a scope inside a turn.
- [ ] OT-P0-013 | Reply egress through the ingressing adapter | The system shall return a reply through the adapter that received the message, on the same thread, and shall not route a conversational reply through a one-shot notification path.
- [ ] OT-P0-014 | In-app conversation adapter | The system shall expose its own in-app conversation surface as an ordinary channel adapter, indistinguishable to the core from any external channel.
- [ ] OT-P0-015 | Telegram adapter | The system shall send and receive text, images, and files on Telegram through the Bot API, requiring no account provisioning and no public origin.
- [ ] OT-P0-016 | Channel conformance suite | The system shall provide one shared conformance suite that every adapter passes, covering text round-trip, media round-trip, declared-limit rejection, threaded reply, rate-limit backoff, unknown-sender ingress, and refusal to auto-reply to an agent-authored message.
- [ ] OT-P0-017 | Operator console | The system shall provide a console for the agent roster, conversations, contacts and their tiers, and the channel catalogue, rendering loading, error, and empty states throughout.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Group threads with a roster-derived ceiling | The system should compute a group thread's ceiling as the lowest trust tier present in its roster, so that adding a lower-tier participant narrows the room rather than widening that participant.
- [ ] OT-P1-002 | No cross-member disclosure in a group | The system should withhold from a group thread any content that only one member's tier unlocks, and should state the withholding rather than answering partially.
- [ ] OT-P1-003 | Speaking discipline | The system should stay silent in a thread with more than one human unless the agent is addressed by mention, reply, or wake word, while still recording every message to the thread.
- [ ] OT-P1-004 | Agent-authored message loop breaker | The system should mark every agent-authored message and should never start a turn in response to one.
- [ ] OT-P1-005 | Per-thread turn budget | The system should enforce an hourly turn budget per thread, should fail closed when it is exhausted, and should notify the owner once rather than on every suppressed turn.
- [ ] OT-P1-006 | Per-thread spend cap | The system should enforce a spend cap per thread so that a participant who is not the owner cannot exhaust the owner's balance, and should surface the cap to the owner.
- [ ] OT-P1-007 | Capability grant on the agent descriptor | The system should read a capability grant from the agent descriptor that names existing agent-manager scopes and program-runtime bindings, and should never define a second policy vocabulary.
- [ ] OT-P1-008 | Assisted agent authoring | The system should draft a complete agent descriptor from a plain-language description, should present it for confirmation before writing, and should write it through a typed action rather than a free-form model write.
- [ ] OT-P1-009 | Approval gate with expiry | The system should let an agent request a scope it lacks, should route that request to the owner only, and should expire an unanswered request rather than leaving the thread pending.
- [ ] OT-P1-010 | iMessage adapter through a Mac fleet node | The system should send and receive iMessage through a Mac in the owner's fleet reached by vrooli-bridge, and should report a stated unavailable reason when no Mac is reachable.
- [ ] OT-P1-011 | Slack adapter | The system should send and receive text, images, files, and threaded replies on Slack, using the same adapter contract as every other channel.
- [ ] OT-P1-012 | Channel availability computed from host facts | The system should evaluate each descriptor's requirements against the facts vrooli-bridge and tunnel-manager already hold, and should present an unavailable channel as a prompt to satisfy the requirement rather than as a selectable option.
- [ ] OT-P1-013 | Attach catalogue ordered by declared friction | The system should present available channels ordered by the setup friction each descriptor declares, so that the least-effort path is first without any maintained ordering.
- [ ] OT-P1-014 | Metered inference through the business suite | The system should charge cost-bearing inference and hosted speech through landing-page-business-suite, and should fall through to an operator-supplied provider key with no charge when one is configured.
- [ ] OT-P1-015 | Notification hub shares the adapter registry | The system should let notification-hub deliver one-shot notifications through the same conversational adapters, so that one channel is never implemented twice.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | SMS through a provisioned number | The system may send and receive SMS and MMS through Twilio, treating carrier registration as a recorded operator handoff rather than an automated step.
- [ ] OT-P2-002 | Voice calls on a phone number | The system may answer and place voice calls, streaming speech through audio-tools transcription and synthesis.
- [ ] OT-P2-003 | Call mode with camera capture | The system may hold a live voice conversation in its own app and may accept a photo or video captured mid-call as turn context.
- [ ] OT-P2-004 | FaceTime audio adapter | The system may place and answer FaceTime Audio calls through a macOS control surface on a Mac fleet node, binding each call to an exact authorised caller identity.
- [ ] OT-P2-005 | GitHub comment adapter | The system may treat issue and pull-request comments that mention the agent as a conversational channel, while leaving every other GitHub capability to the scenario that already owns it.
- [ ] OT-P2-006 | Additional conversational channels | The system may add Discord, Signal, WhatsApp, and Matrix as descriptor-and-adapter pairs with no change to the core, the console, or the conformance suite.
- [ ] OT-P2-007 | Hosted relay for owners without a Mac | The system may offer a third-party iMessage relay as an explicitly labelled fallback that states its custody trade-off at the point of purchase, and never as a default path.
- [ ] OT-P2-008 | Provisioned handle as a gated offering | The system may offer number provisioning and carrier registration as a gated convenience, without gating any capability an owner could run with their own credentials.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API over Connect-RPC with proto-owned wire contracts; typed Go CLI over that API; React + Vite + TypeScript + Tailwind console built on the React Component Library and the `vrooli-default` design kit. No new language and no framework the template does not already carry.

- Data + storage expectations: SQLite through the `api-core/storage` seam, resolved from the scenario identity. Per-domain schemas owned by the code that interprets them. **Channel descriptors are versioned JSON files under `data/channels/`, validated against a schema and loaded at boot; the descriptor file is the source of truth and the database table is a cache** — the same contract `channel-manager` already implements for platform descriptors. No PostgreSQL and no Redis: neither is recorded `supported` on macOS — `postgres` is build-verified there with no macOS hardware run performed, and the platform matrix's two tables disagree about `redis` — and this scenario must run on the Mac fleet node that the iMessage lane exists to reach. Neither is Docker-backed — both are `managed-service` — so the constraint is macOS evidence rather than a container runtime.

- Integration strategy: compose, never reimplement. Agent identity and profiles are read from `prompt-manager` by reference and never copied. Run execution, approvals, scope attenuation, and DAG workflows belong to `agent-manager`. Governed script execution is `program-runtime`. Speech is `audio-tools` with its local `whisper`, `kyutai-stt`, and `kokoro` resources. Outward-facing declared identity and one-time-code retrieval are `persona`. Reaching another machine is `vrooli-bridge`, which owns the reach plane and never learns what a channel is. Public origins for inbound webhooks are `tunnel-manager`. Metered capabilities route through `landing-page-business-suite`; nothing here reinvents credits or entitlements.

- Non-goals / guardrails: This is not a multi-tenant messaging SaaS, not a general remote shell, and not a replacement for `notification-hub` — one-shot notifications keep their own path and are never metered. It does not store credentials, only references to a credential authority. It does not own the GitHub, Jira, or calendar capability: those are tools an agent uses and live in the scenarios that already own them. It never branches on a channel identifier above the adapter layer; a capability question is answered by the descriptor. It ships no hosted third-party message relay as a default path, because doing so would forfeit the custody promise that justifies the scenario.

## 🤝 Dependencies & Launch Plan

- Required resources: none at P0. The first two channels — the in-app adapter and Telegram — need no resource at all. Later channels arrive through optional `cloud-api` resources that are credential-and-reachability only and carry no local process. `twilio` already exists as a `cloud-api` resource with typed Go credentials and is adopted when SMS and voice activate. `whisper`, `kyutai-stt`, and `kokoro` are consumed indirectly through `audio-tools` when call mode activates.

- Scenario dependencies: `prompt-manager` (required) for agent descriptors, skills, and capability grants, read by reference. `agent-manager` (required) for run execution, approvals, and one-way scope attenuation. `scenario-authenticator` (required) for identity on every authenticated surface, verified locally against the published JWKS. `tunnel-manager` (required once any webhook-ingress channel activates) for a stable public origin. `vrooli-bridge` (optional) as the reach plane for host-bound channels such as iMessage on a Mac node. `notification-hub` (optional) as a second caller into the same adapter registry for one-shot notifications. `program-runtime`, `audio-tools`, and `persona` (optional) as capabilities an agent may be granted.

- Operational risks: The highest-severity risk in the scenario is that an inbound message is untrusted text from an unauthenticated sender arriving at a system that can reach a terminal, which makes the trust guard load-bearing rather than advisory and makes granting owner-only scope to a non-owner tier something that must be impossible by construction rather than discouraged by prompt. Runtime injection defence has no existing owner: `prompt-injection-arena` is stale, predates the `api-core` layout, and is shaped as an offline tournament rather than a runtime guard, so it must be redesigned from its own operational targets before anything here depends on it. Two agents in one room will loop without a turn budget, and every turn is a cost-bearing metered call, so the loop breaker is a spend control and not a politeness feature. Group threads broadcast, so an answer one member's tier unlocks must never be rendered in a room where a lower tier is present. Apple provides no supported inbound iMessage interface, so the `chat.db` path is fragile across macOS releases and must degrade to a stated unavailable reason rather than a silent failure. A2P 10DLC registration rejects VoIP numbers and requires a real mobile carrier number, which is an operator handoff rather than an automatable step. Dictation currently never sends `engine_id`, so `audio-tools` has been silently running CPU transcription; that defect is a prerequisite for call mode, not a detail.

- Launch sequencing: descriptors, the adapter contract, and the message envelope first, because every later decision reads them. Then the in-app adapter and Telegram together — two adapters on day one is the only cheap proof that the contract is channel-neutral before anything depends on it. Then threads, address-to-agent binding, and profile-by-reference, which is the point at which a real agent answers a real message. Then trust tiers and fail-closed scope resolution with the conformance phase, which is the point at which the scenario is safe to expose to anybody who is not the owner. Then the operator console. Group semantics, the loop breaker, and the budget follow, because they are only reachable once more than one participant exists. iMessage through a Mac node comes after that, as an adapter swap rather than new architecture. SMS, voice, call mode, and FaceTime are last: they carry the carrier-registration wall, the marginal cost, and the platform fragility, and everything above them must work without them.

## 🎨 UX & Branding

- Look & feel: The `vrooli-default` design kit as generated, unmodified at P0. Two moods coexist deliberately. The console is dense and operational — a roster, a thread list, a contact table, a channel catalogue — and reads at a glance through state encoded in form as well as in text: a live agent, a paused one, a channel that cannot run on this host, a thread near its spend cap. The conversation surface is the opposite and is calm, ordinary, and unmistakably a chat, because it is the one screen a person who has never heard of Vrooli will see first.

- Accessibility: WCAG 2.1 AA. Every state the operator must act on is carried by text or shape as well as colour, because trust tier, channel availability, and budget exhaustion are all conditions a colour-blind operator must resolve without hovering. Keyboard reachability for the whole console including the thread view, visible focus on every interactive element, correct roles and `aria-*` on the conversation transcript so a screen reader announces sender and turn boundaries, and `prefers-reduced-motion` honoured. The template's accessibility primitives, i18n wiring, and `data-testid` selectors are durable seams and are preserved rather than replaced.

- Voice & messaging: Plain and exact, never chatty. The product's most important sentences are refusals, and a refusal always states what was withheld and what would unblock it — "account detail needs the trusted tier; I can send it to you directly" — rather than apologising or going vague. Costs and limits are stated in the units the operator is billed in. The agent never claims a capability it was not granted, and never implies a message was delivered before a receipt exists.

- Branding hooks: The generated `vrooli-default` tokens, the seeded PWA install surface — `site.webmanifest`, `sw.js`, maskable icons, relative install asset URLs, and the safe-area CSS tokens — kept valid throughout. Generic template icons are replaced only when real product branding exists. Per-agent visual identity is not a branding decision here: it is read from the `appearance` colour triple that a `prompt-manager` agent descriptor already carries, so an agent looks the same everywhere it appears.

## 📎 Appendix

- Design record and mockups for this scenario: the Agent Switchboard artifact, 2026-09-01, covering the channel ladder, the conversion funnel, the trust matrix, and the six implementation diagrams this PRD's targets are drawn from.
- `path:docs/concepts/ECOSYSTEM.md` — the interface map whose Embodied/embedded *inbound* cell this scenario fills.
- `path:docs/concepts/PAID_FEATURES.md` — the free / metered / gated contract and the Class A versus Class B meter split.
- `docs/director-swarm/strategy/OBJECTIVES.md` — objective `T2 · Personal agency`, unserved at the time of writing.
- `scenarios/notification-hub/docs/internal/DECISIONS.md` — the 2026-08-17 decision to ship one channel before building any abstraction, and the 2026-08-18 statement that adding a channel should touch only an adapter registry.
- `scenarios/channel-manager` — the descriptor-as-source-of-truth loader this scenario copies.
- `github.com/kingbootoshi/facetime-bridge` — MIT; the FaceTime Audio control surface the P2 adapter would wrap.
