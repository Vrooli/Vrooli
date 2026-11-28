# Aquila launch and marketing-system maturity: source brief

Captured: 2026-09-10, America/New_York. Target sales launch: 2026-09-17.
Product identity: market `web-console` as **Aquila**; retain its internal scenario identifier.
Status: preserved user intent and proposed delivery strategy. This is not an approved implementation plan, live capability registry, campaign ledger, or active execution schedule.

## Outcome and immediate boundary

The operator wants genuinely usable Aquila marketing material within seven days, a mature marketing team that can repeat this for other scenarios, searchable and validated marketing capability/production state, and the Prompt Manager UX and agent-runtime improvements listed below. The initial response must use only light investigation and recommend a strategy before detailed investigation or implementation. All requested improvements remain in scope for subsequent planning; the long-term capability catalog is broader than the capabilities that must be implemented this week.

The operator should not need to repeat the request after an agent handoff, interruption, or resumed session. Plans must reference this source intent and map every requirement below to an owner and acceptance evidence. Operational facts must migrate to their authoritative scenarios rather than accumulate in this brief or the marketing plan of record.

## Complete request inventory

| ID | Required outcome |
|---|---|
| R01 | Use Aquila as the public product name everywhere in the launch package, while joining records through `web-console` and supporting both names in search. |
| R02 | Mature the existing marketing team, its configuration, responsibilities, skills, instruments, objectives, and work selection so it can produce high-quality marketing repeatedly. |
| R03 | Define a broad ideal marketing capability catalog for any deployable scenario, with priorities, current evidence, gaps, and near-term improvements. Cover desktop, hosted/web, other delivery modes, human buyers, developers, and other agents. |
| R04 | Produce an Aquila campaign with actual usable copy, channel adaptations, video scripts/storyboards, blog/technical content, and calls to action. Real screenshots and screen recordings are desired launch outputs. Generated imagery is desirable; polished generated video is not a prerequisite for this week's initial package. |
| R05 | Include promotional social campaigns, image ads, demonstrations, TikTok/Reels/Shorts concepts, YouTube videos, release updates, true advertisements, store listings, landing-page-business-suite placements, and feature/funnel marketing in the capability inventory. |
| R06 | Persist capability definitions, current state, evidence, prerequisites, priorities, and next actions as validated scenario-owned data. Evaluate extending Offer Desk and existing marketing instruments before introducing another owner. |
| R07 | Give the marketing team one understandable read surface for capability, campaign, production, and feedback state. Changing operational status must not require editing plan-of-record prose. |
| R08 | Track each scenario's campaigns and artifacts durably across multiple producers, pauses, resumption, and later revisions. Include versions, source material, approval state, owners, and a way to identify the current approved deliverable. Assess Document Manager alongside Content Desk and Asset Studio. |
| R09 | Distinguish prelaunch readiness work from ongoing release marketing, distribution, measurement, refresh, and retirement. Answer what is ready, what is next, and what blocks launch. |
| R10 | Record real views, impressions, likes, clicks, conversions and related performance where integrations allow. Model future connectors without presenting missing data as zero, fabricated engagement, or a successful integration. Use feedback to steer subsequent content and capability work. |
| R11 | Integrate relevant Offer Desk data, including the added marketing capability view, through the shared AI search package and Search Hub's provider and evaluation conventions. Include the existing Offer Desk corpus, not only new launch records. |
| R12 | Validate actual agent questions through ordinary `search-hub query`, including general capabilities, marketing capabilities, Aquila state, missing assets, priorities, evidence, and next actions. Repair relevant Search Hub, provider, shared package, and discovery friction within scope. |
| R13 | Use representative anticipated questions, synonyms and metadata to improve discovery; test paraphrases and questions not used during tuning. Return authoritative current records and usable follow-up reads rather than only historical conversation or stale prose. |
| R14 | Simplify creating and configuring teams in Prompt Manager. Expose meaningful existing configuration currently hidden from the UI and explain it clearly. |
| R15 | Provide intuitive UI creation/editing of terminal objectives (ends sought) and instrumental objectives (means toward those ends), after verifying the existing model and ownership. Make their relationships to teams and missions understandable. |
| R16 | Show objectives attached to each team on its page, with editing and persistent priority ordering, including drag-to-reorder and an accessible keyboard/touch equivalent. |
| R17 | Consolidate duplicate or confusing objective/configuration representations into one authoritative model with full validation. UI, CLI, API, generated prompts and agents must resolve the same identities, relationships, revisions and ordering. Preserve existing configured intent during migration. |
| R18 | Overhaul the 3-D world HUD using progressive disclosure: a simpler default view, camera controls behind a camera icon, related controls grouped, and a professional consolidated settings overlay. |
| R19 | Use the shared React Component Library and overlay components. Prefer desktop side tabs and mobile tabs where appropriate. Ensure responsive behavior, focus handling, keyboard access, touch usability and clear save/error states. |
| R20 | Make objective management reachable from global settings/management and team pages using the same editor and data. Keep frequently used team context easy to reach rather than requiring navigation through world settings. |
| R21 | Verify that spawning agents, heartbeats and chat reconstruct the right prompt with base-agent identity, selected team-member role, applicable objectives, priorities, skills and current operating context. Make prompt previews explain that composition. |
| R22 | Inspect Prompt Manager prompt assembly against Swarm Manager's existing cache-aware approach. Reuse stable shared composition where suitable; keep stable prompt sections before volatile run context and measure correctness and actual caching evidence where available. |
| R23 | Verify scheduled/cron heartbeats, direct/manual runs and their frequency configuration. Distinguish these modes visibly and behaviorally from interactive conversations. |
| R24 | Support real back-and-forth chat with the selected agent/team member in the 3-D world and a proper chat interface in the agent detail view. Retain conversation continuity and useful failure/recovery state. |
| R25 | Chat should respond to the user's conversational request rather than silently run a heartbeat agenda. Explicit work requests within a chat may still authorize work; selecting chat must not imply autonomous scheduled-task execution. |
| R26 | Let the user choose base-agent context or a specific team-member context when chatting. Agents can belong to multiple teams. Preserve the choice throughout the conversation and make it visible in the conversation and prompt preview. |
| R27 | Reproduce and fix the user-reported agent Files-tab error, “Unable to load files.” Validate populated, empty and failure states and both applicable identity contexts. The report is not yet independently reproduced. |
| R28 | Use existing scenarios and suitable governed programs for production and coordination. Fix or extend the owning capabilities when needed so future agents can repeat the work; avoid an untracked collection of one-off launch files. |
| R29 | Support safe parallel work, interruption and resumption through durable plans, explicit ownership/dependencies, evidence, and supervision. Prevent conflicting writers and duplicated production. A failure in one independent workstream must not stop the others. |
| R30 | Aim to deliver both the launch package and the listed Prompt Manager/system improvements within seven days, with honest estimates, visible risks and no silent loss of scope. Infrastructure friction must have bounded diagnosis and a concrete owner; production should continue wherever its actual prerequisites are satisfied. |

## Evidence from the light investigation

These observations identify existing foundations; they are not end-to-end capability certification. No marketing artifact was generated and no Prompt Manager bug was reproduced in this pass.

- `scenarios/prose-studio/README.md` describes governed text generation, versioned styles/profiles, measured candidates, provenance and section-composed documents.
- The `content-desk` usage skill identifies Content Desk as the editorial ledger for drafts, claims, reviews, campaign capacity and publish records. It explicitly does not generate copy or publish to social platforms.
- `scenarios/asset-studio/README.md` describes an implemented identity workbench: specifications, candidate selection, conformance and released asset references. Production output quality and live rendering remain unverified in this pass.
- Discovery returned `browser-automation-studio.capture-surface`; its combined-read program reported no contract validation error and described durable screenshots and accessibility snapshots associated with a BAS execution. No capture was run here. Browser recording/export must be qualified separately.
- `docs/marketing/catalogs/post-types/README.md` lists dev-log and scenario-spotlight as v1. Several image and video forms are v0 skeletons whose approval gate requires activation. A format definition is not proof of a functioning production workflow.
- The registered Video Studio skill explicitly says it is in the planning phase. There is no `scenarios/video-studio` directory in this checkout.
- A Social Media Scheduler skill describes scheduling and analytics, but there is no `scenarios/social-media-scheduler` directory in this checkout. The marketing team's own configuration also flags its assigned capability as unowned. These are conflicting documentation signals, not evidence of connected live publishing or metrics.
- `scenarios/prompt-manager/store/teams/marketing-crew/team.json` links the team to objectives T1 and T3 with acknowledged revisions. Its instrument is explicitly **partial**, covering Content Desk and Asset Studio and noting the missing aggregate read surface. Members include producer, brand-manager and marketing-contrarian.
- `scenarios/prompt-manager/ui/src/world/hud/ConversationPanel.tsx` contains message history, continuation checks and a full-conversation link. `components/chat/ChatPanel.tsx` and `StartChatDialog.tsx` also exist. Live role selection, dialogue continuity and heartbeat isolation remain unverified. A limited UI surface scan did not establish whether complete objective editing exists.
- `components/editor/tabs/FilesTab.tsx` contains the reported error text. The underlying failure has not been diagnosed.
- `scenarios/document-manager/README.md` describes content-addressed ingestion, citable anchors and custody. Its rendering/write spine is explicitly planned. It should not be assumed to be the existing campaign authoring or version-selection authority.
- A lifecycle-managed attempt to start Document Manager for brief ingestion failed: required `doc-parse` is disabled by operator state. The CLI's automatic setup fallback then failed its API build. No document was ingested. The source brief is retained locally and is being captured in the existing memory journal; document ingestion is a named qualification gap for the later work. The lifecycle refreshed/restarted Storage Manager successfully and rebuilt shared dependency outputs during the attempt.
- `scenarios/plan-manager/skills/plan-family-orchestration/SKILL.md` describes reviewed family graphs, resource claims, derived runnable batches and a durable Agent Manager owner workflow, `plan-manager/plan-family-drain`, with stable start keys and persisted child attempts.
- Search Hub returned useful results but reported its `cli-health.commands` provider unreachable because no running runtime port was available. This limits command discovery; it does not establish that any requested capability does not exist. Existing recall also returned historical monetization discussions, which must not be mistaken for current offer/pricing state.

## Ideal capability catalog and proposed priority

Priority is a proposed sequencing policy for the Aquila launch, not an edit to Offer Desk's authoritative release ranking. P0 means required for credible launch; P1 means the next repeatable production/measurement capability; P2 means expansion after evidence identifies demand. Some P1 outputs can be delivered this week using existing foundations. The catalog describes a supported toolbox, not a requirement to use every channel for every product.

| Capability | Ideal outputs | Priority | Initial evidence/state |
|---|---|---|---|
| Product truth and positioning | Brand identity, audience, problem, benefits, proof, differentiators, feature/platform availability and claim library | P0 | Existing marketing canon and Offer Desk; Aquila-specific current facts need reconciliation. |
| Campaign and funnel planning | Briefs, goals, channel choices, calendar, budgets, content-to-CTA mapping and launch checklist | P0 | Content Desk campaign foundation; full launch workflow unverified. |
| Website conversion | Landing-page-business-suite copy and visuals, pricing explanations, FAQ, objections, demo/download/purchase CTAs | P0 | Existing landing-page capability discovered; actual Aquila offer and placement need inspection. |
| Store and distribution presence | Desktop/download listing, app-store metadata, screenshots, release notes and install expectations | P0 for chosen route | Existing deployment skills; selected route and store-specific acceptance unverified. |
| Product screenshots | Reproducible real UI captures, annotated feature shots, hero/social crops and alt text | P0 | BAS capture program exists; qualify with Aquila. |
| Written marketing | Launch article, technical post/tutorial, feature spotlight, social copy, release updates and comparison copy grounded in evidence | P0 | Prose Studio and active text post types; qualify generation-to-review workflow. |
| Demonstration recording | Short feature recordings, complete walkthrough, GIF/loop and caption/transcript variants | P0 target for this launch | Capture foundation exists; actual record/export path unverified. |
| Static and generated design | Image ads, carousels, infographics, consistent product/brand assets and reusable templates | P1 | Asset Studio exists; many marketing format definitions remain v0. |
| Short-form promotional video | TikTok, Reels, Shorts, teaser ads, vertical screen demos and hooks | P1 | Scripts can use text pipeline; finished video workflow unproven, Video Studio planned. |
| Long-form/audio production | YouTube demos, tutorials, narration, captions, interviews, webinars and reusable clips | P1/P2 | Scripts/storyboards feasible to qualify first; integrated production unknown/planned. |
| Organic distribution | Reviewed multi-platform posts, calendars, delivery receipts, community/launch-directory material and repurposing | P1 | Publishing ledger exists; scheduler skill is not evidence of an implemented local publisher. |
| Paid acquisition | Ad variants, audience/placement hypotheses, landing-page variants, tracking and budget-aware experiment plans | P1 | Copy/creative foundations; live ad-account operations and attribution unverified. |
| Lifecycle communication | Waitlist, launch, onboarding, activation, upgrade, retention and reactivation email/in-product campaigns | P1 | Content can be authored; delivery integrations unverified. |
| Sales and partnerships | One-pager, pitch/demo deck, use-case sheets, objections, partner/affiliate kits and press kit | P1/P2 | Text/assets can contribute; integrated export/templates unverified. |
| Developer and agent adoption | Quickstarts, integration examples, capability descriptions, machine-readable offers and discoverable install/use paths | P0/P1 for those audiences | Existing docs/skills/search ecosystem; product-specific package unverified. |
| Trust and advocacy | Evidence-based comparisons, genuine customer stories, testimonials, referrals and community feedback | P2 | Requires real external evidence and customer participation. |
| Measurement and improvement | Reach/engagement, clicks, trial/activation, conversion/revenue, cohorts, experiments and recommendations | P0 schema/manual evidence; P1 connectors | No live marketing metrics integrations verified. |
| Production governance | Provenance, revisions, review/approval, current approved artifact, release association, reusable recipes, ownership and retirement | P0 | Content Desk, Asset Studio and Document Manager foundations; cross-owner handoff needs validation. |
| Scale and adaptation | Localization, accessibility, regional/channel variants and automated repurposing | Accessibility P0; broader scale P2 | Channel/format-specific automation unverified. |

Proposed capability records include identity/aliases, desired output, audience/delivery mode, producing scenario/program, maturity, environment, prerequisites, latest qualifying artifact/run, validation date, failure reason, next action and owner. Keep definition, implementation, operational readiness, output quality and connected distribution as separate facts. An implementation or healthy service alone does not establish usable marketing output.

Derive priorities from launch prerequisites, outstanding committed work, dependencies unlocked, evidence/confidence, expected effort and current capacity. Expose the reasons and source facts. Treat numerical ROI estimates as hypotheses until measured. Do not invent precision or overwrite the offer board's own ranking.

## Recommended ownership

Confirm this split during the bounded first investigation; retain existing owners where their contracts fit.

| Owner | Authority |
|---|---|
| Prompt Manager | Agent/team identities, member roles, objective relationships, ordering, configuration and prompt context. |
| Offer Desk | What is offered, audience/delivery relationships, commercial/release priorities and readiness projections from other owners. |
| Content Desk | Marketing capability/production overview, campaigns, asset requirements, editorial revisions, claims, review, approval and publishing references; likely the marketing team's single read address. |
| Asset Studio / Prose Studio / BAS | Their production methods, source specifications, outputs and execution provenance. |
| Document Manager | Citable document ingestion/custody and references where its existing contract fits. Asset binaries remain with the appropriate asset/blob owner. |
| Metrics and distribution owners | Connected account state, delivery receipts and actual observations. Content Desk/Offer Desk project these facts rather than fabricate their own measurements. |
| Search Hub and shared AI search | Discovery, provider contracts, authoritative links, indexing/evaluation and retrieval quality. |
| Plan Manager / Agent Manager | Implementation dependencies, resource claims, execution state, durable attempts and supervision. |

One source of truth means one owner per fact with joined views, not one scenario storing every kind of fact. In particular, let Offer Desk answer how marketing affects an offer's readiness without duplicating the editorial ledger. Keep stable voice/strategy in canon and changing operational state in data.

## Proposed delivery structure

Use one Plan Manager family with separately verifiable child outcomes and one durable supervisor. Start with three active worker slots at most, plus the supervisor, and tighten concurrency when shared resources require it. Do not start a second executor after a disconnect; resume the persisted execution. The actual launch order must come from reviewed family state after claims and dependencies are known.

| Candidate child | Scope | Requirement coverage |
|---|---|---|
| A: Marketing data and readiness | Capability catalog, ownership, editorial/asset/document relationships, versioning, next work, launch readiness and feedback model | R03, R05–R10, R28 |
| B: Prompt Manager model and runtime | Objectives/team configuration, validation, prompt assembly, cache-aware composition, schedules, chat modes/identity and Files bug | R14–R17, R21–R27 |
| C: Prompt Manager experience | HUD/settings overlay, team and objective editing, agent detail/chat, context selection, shared component adoption and responsive UX | R14–R20, R23–R27 |
| D: Discovery and capability qualification | Search provider/package conformance, natural questions, aliases, freshness, exact follow-up commands, production smoke examples and gaps | R03, R11–R13, R28 |
| E: Marketing team and Aquila launch package | Team operating setup, actual campaign/assets, factual/brand review, launch handoff and reusable recipes | R01–R05, R07–R10, R21, R28 |

R29–R30 apply to every child. A and D may combine if they prove to be one atomic validation boundary. B and C must agree on contracts before concurrent implementation; UI design and component inventory can proceed earlier. E begins source/positioning/copy work immediately from existing owners and adopts new data surfaces as they become available. Shared package work gets explicit resource claims. Existing marketing member ownership must be reconciled before assigning multiple writers; its current configuration names a sole producer.

Proposed milestones, subject to first-day evidence:

- **Sept 10:** preserve scope; reconcile prior decisions; qualify one text artifact and one real Aquila screenshot; establish current offer/audience/delivery truth, model ownership, source-to-acceptance mapping and reviewed work graph. Assess all seven-day estimates and surface risk early.
- **Sept 11–12:** establish the marketing data workflow; draft the complete campaign; implement objective/runtime foundations; design and begin the shared UI surfaces. Exercise screenshot/recording production early enough to fix the real limiting capability.
- **Sept 13–14:** assemble and review launch outputs; integrate team/HUD/chat changes; run retrieval question suites and strengthen the marketing team's operating instrument. Complete the first usable handoff by Sept 14 so feedback can change it.
- **Sept 15–16:** correct review findings; validate actual end-to-end journeys, context isolation and pause/resume; finalize files, links, screenshots/recordings and launch measurement setup.
- **Sept 17:** deliver the approved launch bundle and recorded completion evidence for the platform improvements. Keep continuing marketing work visible through the same instruments.

The target is all requested near-term changes within the week. A light investigation cannot establish that they will all fit. If evidence shows a conflict, the supervisor must present the affected requirement, remaining work, impact and options early; it must not silently defer a requested feature or label partial work complete.

## Proposed Aquila launch bundle

Minimum concrete handoff: one evidence-backed positioning/claims brief; one campaign/funnel brief and publishing calendar; landing-page-business-suite copy/placement assets; listing copy for the chosen sales/distribution route; one launch article; one technical tutorial or use-case article; a set of channel-native social/ad variants; release-update copy; real screenshots of key workflows; a short real demonstration recording if the qualified route supports it; short-form and longer demo scripts/storyboards with shot lists, captions and CTAs; an indexed delivery manifest linking current reviewed versions and production sources. Image and finished-video additions are prioritized by proven usefulness and production readiness.

Proposed starting scope for estimating the package: 6–8 screenshots, a 30–60 second feature recording, 6–10 social variants, three short-form scripts and one longer walkthrough script. These counts are recommendations, not prior user requirements or quality substitutes. Select the featured workflows after verifying Aquila's actual release behavior and existing positioning.

Public claims must match the release and delivery mode being sold. Reuse prior approved offer/pricing/audience decisions; ask the operator only for unresolved commercial choices. Preparing usable material does not itself authorize external publication, messages, account changes or advertising spend.

## Acceptance evidence that survives a handoff

- Given this brief, when plans are authored, then every R01–R30 row maps to an owner, acceptance check and eventual evidence reference; no requirement disappears because a workstream was split.
- Given an Aquila campaign, when a producer resumes after interruption, then the owner scenario identifies outstanding work, relevant source versions and the current approved assets without reconstructing state from a chat transcript.
- Given multiple producers, when they request campaign work, then claims/reservations and revisions prevent duplicate ownership and conflicting updates.
- Given an objective update/reorder in the UI, when another UI/CLI reader and a newly composed agent prompt read it, then they resolve the same validated identities, order and revision. Invalid/cyclic/orphaned relationships are rejected or clearly reported according to the owner model.
- Given a base agent with memberships in two teams, when the user selects each chat context, then the conversation identifies its context and uses only the intended role/objectives; a normal conversational greeting does not trigger the member's heartbeat agenda. Context switching must not silently reuse the wrong session.
- Given scheduled and manual heartbeat runs, when they compose prompts, then expected objective/role context remains correct and interactive history does not leak between unrelated contexts. Cache optimization must preserve these semantics.
- Given desktop or mobile use, when the user opens settings, camera controls, team objectives or agent chat, then each workflow works with keyboard/touch and shared overlays, with no hidden critical controls or trapped focus.
- Given an agent whose files exist, when Files opens, then files load; empty/error states are accurate and recovery is actionable. Reproduce and retain the regression evidence for the originally reported failure.
- Given a marketed claim or visual, when a reviewer follows its source, then it resolves to real release behavior/evidence and the correct artifact version, using Aquila branding consistently.
- Given unavailable engagement data, when the dashboard or an agent asks for performance, then the response distinguishes unknown/unconnected/stale from a measured zero and identifies the next integration or observation needed.
- Given one blocked child, when the supervisor computes eligible work, then unrelated admitted work can proceed; a disconnect/resume does not double-spawn or replay completed writes.

Use focused regressions and the relevant Test Genie phases per `docs/TESTING.md`. For server-owned suite runs, retain IDs and use the documented one-time wait, not polling. Code changes need the owning scenario's work-ladder diagnosis and checks; a polished screenshot alone is not functional acceptance.

## Search acceptance corpus

Include queries such as:

1. What marketing material can we produce today, and what proves it?
2. Can we make a TikTok for Aquila? What part is ready and what is missing?
3. What blocks marketing web-console for its paid launch?
4. Where are Aquila's latest approved screenshots and launch article?
5. What should the marketing team work on next, and why?
6. Which capabilities would unlock the most currently blocked launch assets?
7. What is required to market a desktop scenario versus an agent-facing capability?
8. What assets are drafted, awaiting review, approved, published or stale?
9. How did the last Aquila campaign perform, and which measurements are unavailable?
10. What marketing and non-marketing readiness facts exist for this offer?
11. Which objectives does the marketing team serve, and in what priority?
12. Which reusable program produces this asset, and what inputs/prerequisites does it need?

Test ordinary automatic routing and provider-direct retrieval separately. Include exact identifiers, Aquila/web-console aliases, paraphrases, held-out questions, unavailable sources, stale/deleted/superseded records and negative cases. Check the expected authoritative entity near the top, usable citations/follow-up reads, freshness and response budgets. Use existing provider conventions and evidence gates rather than create a private marketing-only search system.

Anticipated questions are useful retrieval metadata and evaluation inputs. They do not replace accurate canonical content, appropriate chunking, lexical identifier matching, semantic retrieval, correct provider routing and current-state reads. Queries requiring joins or priority decisions should find the owning typed read/program; a semantically similar historical paragraph is insufficient.

## Continuation instructions

The next step is bounded fact-finding and plan authoring from this preserved scope, after the strategy response. Reconcile existing plans, commercial decisions and marketing work before creating duplicates. Store actual family membership, claims, review, execution IDs and progress only in Plan Manager/Agent Manager. Store evolving marketing facts in their owning scenario. Link their IDs back to the source requirements through the plans. This brief preserves the request and proposal; it must not become a second operational ledger.
