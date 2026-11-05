# Aquila: get more from the coding agents you already have

Aquila is a shared workspace for coding-agent conversations, outputs and the
next decision. Its primary audience is a developer or technical builder who
already uses one or more coding agents and wants to use them together with less
context switching. The positioning below incorporates operator daily use and
product inspection on 2026-10-21. It is a marketing hypothesis with source-backed
feature boundaries, not a measured savings or conversion result.

## Customer job and promise

**Get more from the coding agents you already have.** Keep their conversations
organized, review what they made, and carry useful context into the next task.

A person may already pay for several coding-agent subscriptions, have capable
local compute, or use API keys. Aquila makes those configured tools easier to
work with through one interface. It helps the person choose and coordinate their
existing tools; it does not merge vendor subscriptions, pool their quotas, move a
native thread between providers, or prove that every task takes the cheapest route.

The useful loop is: choose a configured agent, notice where attention is needed,
read the answer, open the files from that conversation, and carry selected context
forward. This addresses cognitive overhead and avoidable switching as well as the
possibility of better utilization of existing spend. Dollar savings require a
buyer-specific baseline and evidence; convenience alone does not establish ROI.

## A short customer pitch

Keep the coding agents you like. Give their conversations and outputs a shared
home. Aquila helps you put existing agent subscriptions to work in one workspace:
choose a configured tool, see what needs a look, inspect what it made, and reuse
the decisions worth keeping. Eligible voice and AI assistance can fit the local
compute, provider account or API key you already have.

## Who should care

Start with individual developers, technical founders and small teams who already
coordinate several coding-agent conversations. A good demonstration answers:
"Why use this instead of another terminal tab?" Show a cross-agent review and
handoff, rich messages, file inspection and clear attention state. Explain the
configured-provider and setup boundaries rather than suggesting Aquila replaces
the underlying agents or includes every provider entitlement.

A trial should demonstrate a repeatable daily job before asking someone to pay.
Measure whether people return to review outputs, recover a conversation and reuse
a handoff. Treat willingness to subscribe, reduced switching and improved use of
existing capacity as hypotheses to test. Do not invent a time or money saving.

## Selling points and proof

| Capability | Customer benefit | Evidence and limit |
|---|---|---|
| Configured coding-agent launch choices | Work with different tools from a consistent workspace | `api/shortcut_profiles.go`, provider/session adapters and the actual launcher. Installation, authentication, transcript support and native features vary by provider; a menu entry is not successful execution evidence |
| Sessions, groups, unread state and customization | Spend less attention finding the next conversation | Sidebar/workspace navigation and native group/collapse captures. Unread means unread, not verified success or required approval |
| Markdown, tables, diagrams and full reader | Review structured answers clearly | `ui/src/components/MessagesPane.tsx` and messages/reader components. Transcript/rendering availability is build-specific |
| Linked files and multiple viewers | Inspect what the agent made without losing its conversation | `ui/src/components/file-preview/renderers/index.ts`; actual message → sidebar → expansion workflow. Format limits apply; workbooks are read-only and do not recalculate formulas |
| Selected-message export/copy | Reuse decisions across conversations and agent choices | Message export drawer and formatters. This transfers selected text, not an entire native provider session or its hidden state |
| Native agent voice or configured dictation | Use a suitable voice path without requiring the same service for every agent | `api/agent_voice.go`, `useAgentVoice.ts`, `useRoutedVoiceInput.ts`. Current browser-native adapter is Codex/version/session gated. Dictation is a separate Audio Tools path; microphone, provider, account and service readiness still matter |
| Local, BYOK and Vrooli-assisted AI options | Use available resources and choose how supported assistance is funded | AI service/configuration and credential/billing owners. These are feature-specific routes, not universal coding-agent substitution or a guaranteed least-cost optimizer |
| Vrooli ecosystem | Combine the workspace with deeper agent operations and local tools | Aquila handles daily interaction/review; Agent Manager handles supported orchestration, evidence and efficiency investigation. Bundling and account entitlements follow the actual offer |

## Voice and funding: explain the distinctions

Native agent voice uses a supported agent's own live connection and eligibility.
The inspected implementation currently supports qualified Codex sessions; do not
advertise native voice for every agent. Auto mode attaches to eligible already
active native voice; it is not a universal "subscription voice first" promise.
An explicit Agent Voice choice and Dictation choice are available. Native failures
report recovery rather than proving a transparent paid fallback occurred.

Dictation, speech playback and AI command assistance are separate capabilities.
Their local/BYOK/Vrooli routes have their own configuration, permissions, health
and billing. Existing provider subscriptions do not automatically supply API-key
credits. Do not imply that a Vrooli subscription includes unlimited third-party
inference, that local compute has no cost, or that every route is interchangeable.
See [Monetization](MONETIZATION.md) for funding boundaries.

## Aquila and Agent Manager

Aquila is the daily workspace: choose, follow, review and hand off. Agent Manager
(Mizar) is the deeper operational instrument: inspect supported workloads,
recurring friction, outcomes, duration and attributed usage, then compare a change.
This is a complementary efficiency story, not a claim that Aquila itself performs
all analysis or that Agent Manager observes every external terminal session.
See [Agent Manager positioning](../../../agent-manager/docs/business/GO-TO-MARKET.md).

## Future opportunity: provider limits and reset visibility

A unified configured-agent usage/remaining-limit/reset view could help people
choose which entitlement to use next. This is a discovery opportunity, not a
current selling point or an implementation commitment in this update. First
verify each provider's supported read path, account scope, freshness and terms.
Keep unavailable values unknown, distinguish subscription allowances from API
billing, and avoid auto-switching an active thread or disclosing credentials.
Do not show a fictional quota dashboard as product evidence in a launch film.

## Offer, availability and launch direction

Offer Desk's 2026-10-21 release-path read ranks web-console first and marks it
`TRIGGER_MET`, with AI Gateway and the landing-page suite still listed as blockers.
That graph describes commercial intent, not installer or integration certification.
Resolve current package/target and published claims before a new commercial export.
Desktop remains the lead target. Mobile can be useful, but secure remote access,
tunnel and home-screen setup prevent it from being the default turnkey promise.

The launch film should establish the existing-agent problem, show the configured
choice and daily review loop, explain eligible voice/assistance routes, and give
one qualified next action. Deliberate browser/app framing and optional qualified
Device Studio laptop footage should support readable native proof. Duration follows
the necessary explanation and actual capture; the previous48-second cut is not a
ceiling and90seconds is not a requirement. Keep rate-limit intelligence, quantified
savings, effortless setup and unverified voice execution out of demonstrated claims.
