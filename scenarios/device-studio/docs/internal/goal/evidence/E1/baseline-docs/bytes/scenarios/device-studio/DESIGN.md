---
id: vrooli-default
version: 0.2.0
name: Vrooli Operational Console
description: Dense, responsive, customizable operational UI for generated Vrooli scenarios.
components:
  button-primary:
    tokenSource: design-tokens.css
  button-primary-loading:
    tokenSource: design-tokens.css
  button-disabled:
    tokenSource: design-tokens.css
  input-error:
    tokenSource: design-tokens.css
  alert-error:
    tokenSource: design-tokens.css
  toast-success:
    tokenSource: design-tokens.css
  empty-state:
    tokenSource: design-tokens.css
  skeleton:
    tokenSource: design-tokens.css
  inline-progress:
    tokenSource: design-tokens.css
  retry-action:
    tokenSource: design-tokens.css
constraints:
  letterSpacing: "0"
  cardRadiusMaximum: "0.5rem"
  defaultMode: "light"
  supportedModes: ["light", "dark", "system"]
  responsiveBaseline: "mobile-first"
  dominantPalette: "neutral-operational-with-blue-cyan-and-semantic-status"
---

# Device Studio design contract

**Status: design only, not implemented or visually approved.** The generated UI still has a placeholder home/health surface, removable Notes example and template Settings. No shot builder, library, account integration, renderer or creation workflow exists in that UI. `experience/index.json` records real template routes separately from proposed product routes; every entry remains draft. Schema validation proves contract structure, not accessibility, beauty, renderer quality or completed journeys.

## Intent

The inherited Vrooli Operational Console tokens and shared component ownership above remain binding. The [source design kit](../../templates/design/vrooli-default/DESIGN.md) defines the common floor; this document adapts it for builders, designers and their agents composing inspectable device shots. The product adaptation is a **studio workbench**: a quiet operational shell around a large, trustworthy view of the shot. The image is the work, not decorative wallpaper. People should compare the primary editing experience with a focused product-shot editor: a scene/viewer and properties inspector with a motion timeline, rather than a marketing dashboard or a replacement for Blender. The professional reference is the clarity of a video editor's stage/inspector/timeline, adapted to device presentations and evidence-backed assets.

This document defines intended behavior. Read `docs/START-HERE.md`, `PRD.md`, `docs/concepts/EXPERIENCE.md`, the linked requirements, and `experience/README.md` before implementation. The [research index](docs/research/README.md) links portable source snapshots and the original retained evidence at `~/.vrooli/evidence/device-presentation-2026-10-18`; it distinguishes historical experiments from current product authority.

## Current truth and evidence frontier

| Surface | Route truth now | Product/evidence truth |
|---|---|---|
| Home | `/` exists in the generated router | Placeholder home plus real template health feature; not a project browser |
| Notes | `/notes` exists | Removable worked example, not a Device Studio domain |
| Settings | `/settings` exists | Template preferences; not implemented account, privacy or worker settings |
| Product surfaces below | Proposed routes only | Draft design intent; no runtime selectors, state setup hooks, captures or attestations |

Retain the example spec and real Notes binding until the governed detemplate operation removes its UI. Do not mark a planned route active because a document names it. Do not remove the home placeholder marker or claim the design orientation gate passed in this docs-only change. There are no ratified desktop/mobile visual mockups. The visual design maturity is **pre-L1: concept and semantic contracts authored, mockups and implementation absent**, not visually ratified or beauty verified. This visual frontier is separate from experience-manager's parser/reference capability levels; a clean draft contract does not raise visual or runtime maturity.

## Information architecture and task hierarchy

Primary navigation has three destinations: **Projects**, **Library**, **Jobs**. Account and global preferences use the shared shell utility/settings entry. Project settings stay within the selected project. Detail routes preserve a breadcrumb/back action, selection, filters and scroll position; they are not additional top-level destinations.

The first useful flow is **Create shot → choose device and scene → attach authentic media → choose presentation and motion → inspect final-look preview → render → review/download/reopen**. A useful photographic milestone does not complete the requested physical phone rotation or laptop lid motion. A returning user sees their latest projects, pinned shots, current work and resumable jobs before setup advice.

| Planned page | Proposed route | Primary user decision |
|---|---|---|
| Projects | `/projects` | Open a project, create one, or import a portable project |
| Project | `/projects/:projectId` | Choose a shot or resume relevant work with this project's privacy/budget context |
| Library | `/library` | Find a suitable device, scene, presentation or motion by usable scope |
| Asset detail | `/library/:assetId` | Understand exact revision, rights, references, compatibility and review limits |
| Import | `/imports/new` | Admit a free asset/reference/media source safely and review its adaptation |
| Creation | `/creations/:creationId` | Initiate or supervise disclosed AI/agent candidate work within one budget |
| Review | `/reviews/:reviewId` | Compare exact candidates/references and record a scoped craft/qualification decision |
| Shot workbench | `/projects/:projectId/shots/:shotId` | Compose and inspect one reproducible shot |
| Jobs | `/jobs` | Find running, queued or attention-needed jobs without losing ownership/context |
| Job detail | `/jobs/:jobId` | Understand progress, stop/recovery/accounting and retained checkpoints |
| Result | `/results/:resultId` | Inspect actual output, provenance, limitations and direct downloadable artifacts |
| Account and providers | `/settings/account` | Connect optional services, choose account/provider and recover access safely |
| Project settings | `/projects/:projectId/settings` | Set this project's external-media permissions, aggregate budget and defaults |

Names of routes are design proposals, not an API or router implementation. Stable domain IDs, future route choices and links must remain aligned during implementation. Settings owns theme, font scale, locale, reduced motion, keyboard shortcuts, handedness where useful, notifications, renderer availability and account access. Quick shell controls are shortcuts to the same settings, never independent preference stores.

## Visual concept and design-system ownership

Use the inherited neutral operational language deliberately: a restrained slate/white work area, optional dark shell and a neutral stage whose surround helps judge the shot. In dark mode, preserve UI contrast without recoloring the rendered photograph or app texture. Theme changes alter controls and chrome; they do not alter a shot's pinned display transform. Blue identifies the current command/selection, cyan technical inspection, green a completed verified outcome, amber attention/staleness and red failures or destructive actions. Every status also has text and an icon/shape.

Media may be richly photographic; application chrome remains calm. Avoid decorative gradients, fake charts, giant welcome cards, sparkle-only AI controls, and ornamental glows behind the canvas. Thumbnails are stored previews with provenance, not silently generated artwork. Empty library/project states explain the next useful action without pretending sample assets are qualified.

Keep typography grounded in the inherited Inter/platform sans and monospace roles. Body/input text remains 16px on phones; dense desktop lists can use 14px. Paths/hashes/measurements use monospace with wrapping/copy/detail access. Product anatomy, source image, generated scene, simulated OS/browser chrome, proxy preview and final render use separate labels. A polished camera does not certify measured geometry or native app availability.

Configure the library `AppShell/2` rather than redrawing chrome. The workbench needs a filled main pane; lists/settings use scrolling pages. Preserve the shell's skip link, focus, safe areas, theme/i18n, and mobile navigation. Reuse working RCL page headers, list/detail patterns, async/empty/error regions, forms, dialogs, settings rows and status components through governed adoption. `AccountSettingsSurface` was a null placeholder in v8 research; qualify a real shared account surface instead of assuming its catalog name provides working login. Do not hand-edit released component versions or introduce a private token system.

## Compose the content once, then design each medium

| Archetype | Desktop composition | Mobile composition |
|---|---|---|
| Projects/Library lists | Search/filter toolbar, dense rows or deliberate thumbnail grid, optional detail pane; saved filters | Search and one key filter visible, compact record rows with one decisive status; detail drill-in preserves filters |
| Project overview | Shots dominate; jobs/recent activity and project limits remain adjacent | Shot/resume action first, compact budget/privacy summary, activity below; no wall of dashboard cards |
| Shot workbench | Scene/shot outline left, large stage center, selected properties right, timeline below; resizable/collapsible panes | Full-width Stage, Controls, Timeline and Checks views over the same draft; bottom tool sheet or full-screen inspector; no tiny three-pane imitation |
| Reference/candidate review | Synchronized side-by-side candidates/reference, detail zoom and decision/limitations panel | Named candidate/reference toggle with identical zoom/pose, swipe only as optional shortcut; comparison facts and decision remain accessible |
| Creation/Job detail | Stage/preview and timeline/activity with summary/stop/budget alongside | Current stage, preview and stop action first; activity/receipts/accounting through focused panels |
| Forms/Settings | Summary groups with clear edit affordances and adjacent explanations | Summary rows open focused sections, sticky safe-area action only when editing; keyboard-aware confirmation |
| Results | Actual video/still at useful scale, versions/comparison and artifact list beside it | Media first, explicit fit/fullscreen controls, then review and direct artifact actions; no HTML gallery dependency |

Do not drop functionality on a phone. Every pose/camera/calibration/keyframe operation has numeric controls, step/reset actions and a textual selection hierarchy. Gestures supplement visible actions. When the keyboard opens, collapse secondary chrome, keep the edited field and error visible, and avoid covering Save/Cancel or the source crop. A timeline can scroll inside its own bounded pane; the document must not overflow horizontally.

## Shot workbench: one draft, guided and advanced

Keep shot title, project, saved/draft identity and renderer/strategy visible without competing with the stage. Group guided controls into **Setup, Content, Motion, Look, Output**. Advanced switches reveal exact properties of the same draft, not a second configuration that can drift. Contextual selection highlights the instance/part/display/track being edited and provides a visible textual equivalent.

The dominant action follows work readiness: attach missing source, resolve incompatible setup, request final-look preview, then render. Save remains discoverable and reports its real state. Camera motion or timeline scrubbing is ordinary deterministic editing; it never triggers an AI call, critique, model download or agent launch.

- **Setup:** choose pinned device and scene revisions, exact/generic identity, configuration, attachments and support slot. Show `Compatible`, `Incompatible` or `Needs review` with the violated constraint and valid alternatives. A wall mount needs a compatible interface, not a free-standing display drawn near a wall. Camera protrusions/support contact and moving swept clearance remain visible diagnostics.
- **Content:** name each physical display and app viewport, select the genuine source, preserve aspect/whole capture by default, expose explicit crop/contain and orientation, and show source dimensions. OS status, gesture, browser and desktop chrome are configurable simulated layers outside the capture. A bright diagnostic makes colored pixels around the black island obvious. Curved/folding/multiple surfaces use their actual mapping; hinge gaps stay empty. Media intake identifies app-only versus full-screen capture so a recording with existing system chrome does not receive a second status and home bar.
- **Calibration:** show source and output views with corner/mask/occlusion overlays, zoom, undo/reset, measured versus estimated/unknown fields and numeric point editing. A planar homography is offered only to an eligible planar plate. Dragging is optional; select a named point and enter coordinates or nudge with keys. Preserve the original capture and calibration revision. Do not stretch geometry or screenshots to hide a reference mismatch.
- **Motion:** separate camera, whole-device, joints and source playback tracks. Show output time/frame and source time independently; support keyframe add/edit/delete, easing, duration, limits, snapping and a readable hold. A phone spin means physical rotation; a photographic crop is not an equivalent. Laptop lid and screen move together around the qualified hinge while the base stays seated. A screen-entry target is the active app aperture with an explicit handoff to genuine capture.
- **Look:** expose camera pose/lens, projection, framing, depth-of-field, lights/environment, support/placement, and qualified material/finish presets. The browser stage is clearly `Interactive preview — approximate lighting`; an actual renderer still is `Final-look preview` with its recipe identity. Reference photographs and generated imagery are labeled. No slider invents details or claims exact manufacturer CAD.
- **Output:** select qualified 1080p SDR24/30 landscape or portrait composition and high-resolution still profiles when eligible. Reframe each aspect rather than silently crop the device away. Show source-density/legibility warnings, actual color transform/encoding, available engine/profile, estimate and resource ceiling. 4K video/HDR remain later qualification; an upscaled result is labeled as such.

Source audio is explicit: identify whether it exists, mute by default for visual studies, and expose an intentional preserve/trim policy with timing. Do not imply new voice/native behavior. Source clips, holds and repeats are visible in the source track; processing must not silently change actual event timing. Display source presentation timestamps and rational output frame rate; variable-frame-rate input is not treated as one input frame per output frame.

## Library, import and reference-based creation

Library tabs distinguish Devices, Scenes, Presentations and Motions. Search includes names/variants; filters include family, qualification, supported configurations/motions, screen count/shape, support/mount, license and local availability. Sort by useful decisions such as recently used, name, family or review state. Cards/rows show one useful image plus identity, supported scope and current revision, not a generic `realistic` badge.

The first complete release covers slab phone, laptop, tablet, foldable/dual-screen, round watch, monitor/TV, smart display/frame and embedded vehicle host representatives. A sourced listing is a lead; inspected geometry, rights, measured references and qualified pose/motion are separate statuses. `Free download` never implies CC0 or redistribution rights. If an account gate blocks acquisition, retain the lead and offer another free source or a clearly labeled original/generic construction path. Do not substitute a mismatched generation without showing the adaptation.

Import has file/source selection, bounded admission/progress, dependency/format inspection, unit/axis/body-versus-envelope review, display/rig identification and a retained source/adaptation receipt. Failures preserve useful input and identify the exact rejection or missing dependency. Imported native files and generated scripts are untrusted; no automatic execution of embedded code. UI recovery calls the supported owner path; it does not install packages or repair the host privately.

Creation starts with **Create device with an agent**, **Refine with an agent**, or **Generate scene with AI** as appropriate. Before initiation, show the exact source/candidate revision, allowed references/tools/workspace, child-work allowance, funding route and aggregate time/iteration/spend ceilings. Search/import/build/fit/review/refine/pack stages retain candidate revisions and comparisons. A normal library selection simply loads an approved stored revision and initiates no AI.

Candidate comparison includes neutral front/side/rear or relevant configurations, actual intended-scene appearance, reference landmarks and uncertainty. Numeric silhouette errors require reviewed calibration; otherwise label comparison qualitative. Keep rejected iterations and unresolved anatomy, motion or material defects. Approval is scoped to exact bytes, view/configuration/output, and the required reviewer boundary; agent critique cannot self-release a shared asset.

## AI, privacy and aggregate budgets

Every AI call and agent launch is disclosed in visible action text before initiation, not only an icon, tooltip or after-the-fact log. The preflight summary names local/BYOK/Vrooli source, accountable project/account, expected outgoing inputs, known estimate or `Unknown`, enforceable cap and stop behavior. Local model inference is still AI; it does not imply external upload. Optional AI denoising/upscaling is labeled before the user starts that processing, including when it runs locally. A local deterministic render is not a new AI call merely because its input asset was generated earlier, but that asset's provenance remains available.

A bounded autonomous workflow is authorized once for its declared envelope. Activity lists every nested model call, tool stage and child agent under the same aggregate budget. Do not show a modal for every permitted iteration. Funding changes, newly sensitive data, undisclosed agent work or expanded ceilings stop at the boundary for an explicit revised choice. A missing enforceable cost bound blocks paid/BYOK dispatch; a displayed estimate is not that bound. Unknown actual cost remains pending with the applicable hold, never zero/free.

Real app media stays local by default. Project-specific external-media opt-in covers generation, critique and remote agent vision, including screen-bearing renders, thumbnails, contact sheets and derived text. Account sign-in, a key or a spending cap is not media consent. Default external references have blank/redacted screens. Show the actual reviewable egress manifest/provider/purpose and pin the consent/redaction to the transmitted bytes. Ordinary editing, preview reuse and rendering never upload implicitly.

## Jobs, results and safe recovery

Jobs have durable identity, immutable shot/creation revision, stage, source/funding route, progress when measurable, elapsed time, estimate/ceiling, worker capability and accounting state. Show `Queued` or indeterminate preparation rather than invented percentages. Preserve a useful existing preview while refreshing; stale previews name the revision and never masquerade as current final output.

Stop acknowledges immediately as `Stopping`; terminal stopped/cancelled status waits for owner evidence. Future submissions/child work cease; supported remote cancellation is attempted and already-started external work may still cost money. Preserve checkpoints/partial outputs and accounting reconciliation. On reconnect/restart, reconcile the same operation before retry; no blind resubmission of a paid call or duplicated render. Queues support search, state/project filters, useful sort and stale/refresh/retry affordances.

Results lead with the actual encoded still/movie and distinguish technical validation, craft review and operator/release status. Inspect continuous first/middle/last/critical motion, readable holds, changing reflections/contact, masks, source clock and aliasing at intended output size. The v5 technical success did not establish photographic quality; v6's stationary photographic direction does not qualify 3D motion. More samples or a successful decoder are not aesthetic acceptance.

Direct PNG/MP4 and eligible model/native downloads are primary; an HTML inspector is supplemental. An artifact list identifies raw/final/composited/upscaled/portable/native classes, dimensions, duration/fps, color, review scope, rights and recipe/provenance. Reopen/duplicate creates a new draft from the pinned recipe; a portable package excludes secrets and machine-private paths. Missing source files offer explicit relink/hash checking; never silently replace a dependency. Local approval, shared release pending, failed handoff and published state remain distinct.

## Account, providers and offline use

Separate local operator authority, optional Vrooli commercial account, verified entitlement and live credit authorization. Personal local projects and available local engines work without cloud sign-in. Download entitlement is a separate delivery concern. Account controls show selected business account, valid/expired/unavailable session/lease states, balance freshness, provider metadata and a supported disconnect/recovery path. A configured credential is not proof of valid login or funds.

Desktop optional commerce uses the qualified scoped account-link path, not copied website tokens. Browser access uses its declared ingress and qualified same-origin adapter. The current Web Console callback defect and desktop linked-lease-to-funded-operation gap remain implementation obligations, not patterns to copy. Never store secrets in projects, renderer storage or logs. Account sign-out leaves projects intact; explicit unlink removes the local lease and shows remote revocation/pending status. Offline valid leases can permit only their declared local capacities until hard expiry; they cannot mint spendable credits.

Provider routing is explicit. Local failure does not silently switch to BYOK or Vrooli-funded work. BYOK costs the user's provider account and is not billed again as equivalent Vrooli credit usage. Unsupported local renderer capability is separate from offline account service: show a useful proxy and the supported setup path while preserving the shot, without claiming final render support.

## Feedback & State

Every action reports its actual effect near the work: selected instance/track, changed draft, saving revision, queued request, sample ready or terminal output. Async controls acknowledge immediately and prevent harmful duplicates. Stable skeletons retain layout, old previews remain visible with a stale-revision label, and incomplete data is never presented as confirmed empty. Forms preserve input, associate field errors with the control and provide a submit summary. Toasts are supplementary; blocking issues remain in the relevant panel with a safe next step and sanitized diagnostic reference.

Library, source/media, stage, inspector, timeline, job progress, account and result surfaces have independently meaningful readiness. An unavailable optional account or catalog service does not blank the local editor. At implementation each independent async surface receives its real lifecycle region and component binding; planned surfaces in `x-design` are not yet such proof. Selection, focus, panel dimensions and compatible edits survive transitions.

## UX-State Contract

These are required **intended** states, not assertions that the generated template already exposes them. Page descriptions specify the content and recovery for each. Runtime evidence and deterministic state setups are absent until implemented.

- `idle`: no pending operation; the current task and available next action remain legible.
- `pending`: acknowledged work is in flight or queued with observed progress or an honest indeterminate state.
- `retrying`: a safe retry is actively reconciling or executing and cannot create a blind duplicate.
- `default`: current revision/selection, meaningful primary action and honest readiness.
- `loading`: stable-layout pending state; distinguish missing response from empty data.
- `submitting`: immediate acknowledgement and duplicate-effect prevention.
- `saving`: unsaved/persisting/saved revision truth, retained edits on failure.
- `syncing`: owner/input being reconciled and usable retained context.
- `refreshing`: previous result stays visible with age/revision indication.
- `empty`: confirmed empty state with next action and no fabricated examples.
- `partial`: usable subset/checkpoint plus precise omissions; no whole-job success claim.
- `stale`: age/changed inputs and consequences for decisions or accepted previews.
- `success`: exact saved result/receipt identity and relevant next step.
- `validation-error`: field/constraint issue with inline correction and retained input.
- `request-error`: failed action, safe remaining work and a concrete recovery path.
- `permission-denied`: explain unmet owner/account/release authority without bypass instructions.
- `offline`: identify which local work remains possible and which remote work is unavailable.
- `retry`: reconcile prior operation identity before a safe retry; no hidden paid duplicate.

Additional states include incompatible setup, missing media, calibration, unsupported renderer, budget exhausted, consent required, candidate comparison, release pending, stopping, interrupted, recovering and accounting pending. Model these distinctly rather than forcing every failure into `Something went wrong`. Form feedback is near the field plus a summary; input survives errors. Toasts may confirm transient success but cannot be the only record of a blocking failure.

## Request Lifecycle

For every network call, long local/file task, generation step and mutation, design idle, pending, success, failure, retrying and disabled/unavailable feedback. The canonical UX states above make those lifecycle meanings specific: validation/request errors explain failure, permission/offline/unsupported states explain unavailability, and submitting/saving/syncing describe pending work. Do not silently remove an action when its prerequisite is unavailable; explain the prerequisite without implying that connecting an account authorizes an operation.

Local edits can be optimistic only with clear undo or rollback. A failed save retains a visibly unsaved revision; a stale response cannot replace a newer edit or preview. Expensive submission uses an immutable snapshot and durable idempotency identity. Resolve a lost acknowledgement through that identity before offering a new request. A compiled policy snapshot is provenance, not perpetual authorization: dispatch, child work, resume and retry recheck current consent, actor/account authority, selected route and remaining aggregate cap. An account or funding change never silently retargets queued work.

Cancel is immediately acknowledged as stopping. Show the owner's serialized result: already-terminal if success committed first; otherwise late output after cancellation stays diagnostic/partial rather than becoming success. Keep uncertain remote outcomes reconciling, retain pending cost/holds and distinguish execution cleanup from settlement. Do not claim stopped, refunded or uploaded bytes recalled before the relevant owner confirms it. A UI tab closing leaves a server-owned job intact unless the user explicitly requests cancellation.

## Accessibility, input and responsive acceptance

Use semantic headings, landmarks, lists/tables and form labels. Canvas information has a named scene/part/display tree and textual compatibility/result summary. All drag handles, pose/camera controls, crop/mask points and keyframes have keyboard and numeric equivalents; undo/reset never requires pointer precision. Keyboard shortcuts are discoverable/remappable and do not steal typing shortcuts or Space from focused inputs. Dialogs/sheets trap focus appropriately, Escape closes safely, and focus returns to the initiating control. Confirm only destructive/irreversible or actual authority-expanding actions; ordinary reversible edits use undo.

Support WCAG AA text/controls in light/dark, 44px comfortable touch targets, visible focus, safe-area insets, font/200% zoom, RTL-aware reading order and reduced motion. Short labels do not conceal exact names: wrap/break paths and provide full-name detail/copy for truncated records. Reduced motion affects editor feedback and automatic preview playback; it never silently changes an authored export track. Live progress/status announces stage changes without narrating every frame. Never rely on green/red or a spinning animation alone.

The baseline **future runtime** capture matrix is mobile320×568 and390×844, tablet768×1024, desktop1280×800 and1440×900, plus a wide workbench. Include virtual-keyboard-open field editing, 200% zoom, keyboard-only use, long names/paths, light/dark/system, RTL and reduced motion. Capture actual route/state interactions, not only static screenshots. Source media's color/readability is evaluated separately from application theme.

## Baseline Capture Matrix

The future default machine-evidence run is a bounded covering set of at most 12 captures per active surface, rather than a Cartesian product. Cover declared viewport families, light/dark desktop, English/Arabic direction, both motion preferences and rest/hover/focus-visible/pressed interaction states. The viewport and accessibility cases above determine which combinations carry the most risk. Additional scoped claims opt into the smallest extra targets needed. Interactive keyboard, virtual-keyboard, 200% zoom and recovery paths need real execution evidence in addition to still captures. No baseline capture exists for the planned product UI in this foundation.

## Review and implementation frontier

Before UI feature work, produce paired desktop/mobile mockups for every primary surface under the governed mockup location, then obtain the experiential skill's two checkpoints: concept and mockups. No ratification or captures are claimed here. Product asset/craft/release reviews remain distinct from those UI design checkpoints.

Implement real product routes and RCL adoption, then add stable roles/names/selectors to the existing selector registry. Only then promote relevant draft claims to implemented deterministic types and join actual BAS accessibility/capture evidence. Keep custom/aspirational intent visible until its checker exists. Do not insert fake bindings, attestations or setup query parameters to make validation green. Preserve the real template route specs until their UI is deliberately replaced/removed.

This pass does not satisfy the design-language orientation gate: shell configuration, token application, placeholder replacement, adoptions, responsive compositions and runtime proofs are still absent. Requirements and the delivery plan retain those obligations. `experience/README.md` records the actual structural validation and remaining execution frontier.
