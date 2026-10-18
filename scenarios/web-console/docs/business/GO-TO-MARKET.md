# Aquila: customer value and current offer boundaries

Aquila is a workspace for people who work with multiple coding agents. Its useful
loop is: notice where attention is needed, read the answer, inspect the actual
files, then carry the useful context into the next conversation. This positioning
is based on operator daily-use feedback (2026-10-18) and the source surfaces below;
it is not a measured conversion result or a promise that every deployment includes
all integrations.

## Audience and recurring job

An individual developer or technical builder has several agent conversations in
flight. Switching between raw transcripts, terminals, files and copied context
makes it harder to find the next decision and review what an agent produced.
Aquila brings those review and coordination tasks into one workspace. The value
is control and clarity during ongoing work, not simply opening a terminal in a
browser. Competitive superiority and quantified time savings remain unmeasured.

## Discoverable selling points

| Capability | Useful outcome | Product evidence | Qualification |
|---|---|---|---|
| Sessions, groups, unread counts and sorting | Find conversations that deserve attention without opening each one | `ui/src/lib/workspaceNavigation.ts`, sidebar state and cursor handling | Unread is a reading signal; do not equate it with a verified approval queue or agent success |
| Markdown messages, tables, diagrams and reader view | Review an answer as structured material rather than scanning terminal escape sequences | `ui/src/components/MessagesPane.tsx`, `MessageJumpList.tsx`, Markdown and reader components | Provider transcript availability and diagram rendering must work in the selected build |
| File viewing across documents, workbooks, source, changes, HTML, media and folders | Check the deliverable while keeping its conversation close | `ui/src/components/file-preview/renderers/index.ts` and renderer implementations; README File inspection; `fixtures/file-preview/` | Format limits apply. Workbooks are read-only; saved formulas are not recalculated. DOCX layout can differ from Word. A UI demonstration does not certify every format |
| Selected-message copying/export | Carry a useful handoff into another conversation with less manual assembly | `MessageJumpList.tsx`, `MessageExportDrawer.tsx`, export formatters | Copy/export is distinct from executing the destination agent or proving the next task succeeded |
| Vrooli integration | Use the workspace alongside a growing set of local apps, services and agent capabilities | `docs/internal/BUNDLE_INTEGRATION.md`, `.vrooli/service.json`, provider/integration owners | Explain Vrooli as the surrounding collection of apps and local services. A standalone offer does not imply that the full ecosystem, models or paid inference are included |
| Voice and audio | Speak input and use supported audio workflows | voice controls, audio-tools integration, file-preview audio renderer | Verify microphone permission, provider, service dependencies and selected target before advertising hands-free or turnkey voice |
| Mobile access | Continue from a smaller device | responsive workspace, terminal keyboard and file viewers | Potentially valuable, but excluded from the present desktop-led offer story. Secure access/tunnel and home-screen setup require additional work |

This table is a discovery aid, not approved commercial copy. Read actual workflows
and tests when assessing a new build. Operator preferences can reveal overlooked
value; they do not establish package readiness.

## Current desktop positioning

Lead with **a workspace for your coding agents**. A coherent demonstration follows
one project through unread conversations, a readable answer, linked files and a
handoff. Show multiple file types where they contribute to review. Use real
controls and source pixels. Fictional presentation data may demonstrate UI
rendering and navigation; it cannot prove agent execution, persistence, installer
success or setup speed.

The Offer Desk snapshot read on 2026-10-18 lists web-console as a customer-facing,
marketed deliverable (`82881311-e8bd-4a12-a268-868cf8942581`, `TRIGGER_MET`). Its
desktop ramp (`7f935fe5-ecae-4aa1-9c55-5b6d4a2e4288`), developer audience
(`b62b4783-e872-4f6d-98b4-ab98d96ba818`) and tier-1 variant
(`54180a50-e4c4-49b6-9b11-0735fdb0b65e`) were `IDEA`. Commercial intent is not
package certification. Resolve current Offer Desk, actual release/package evidence,
Content Desk claims and the published landing route before each production.

Do not headline phone use, effortless remote access, bundled model providers,
automatic approval handling, full ecosystem installation, hands-free voice, or
unverified download availability. These can become future angles when their
specific delivery and setup evidence supports them. Preview/explore is a valid
next action when a download is not verified.

## Maintaining this brief

Keep customer jobs and evidence close to product changes. For each candidate,
record value (observed, operator-reported or inferred) separately from availability
(supported, conditional, excluded or unknown for the specific offer). Rank useful
workflows after discovery and qualification. Preserve worthwhile unknown candidates
for future offers. Recheck target, package, dependency and reliability assumptions
when either the offer or product changes. The launch-video skill owns production
selection and proof; Offer Desk owns commercial intent, and Content Desk and the
published presentation retain their claim-approval boundaries.
