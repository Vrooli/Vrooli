# Adaptive Personal Planner — Complete Vrooli Implementation Plan

**Version:** 1.0  
**Prepared:** September 18, 2026  
**Audience:** The local coding agent implementing the product in Vrooli and the developer reviewing its work.  
**Status:** Product direction and six launch decisions accepted. Engineering defaults below make the work actionable; repository-specific facts must be established locally.  
**Working label:** Planner. “Observatory” names the approved visual direction, not an approved permanent product name.  
**Suggested technical slug:** `time-planner`, subject to repository discovery and the identity of the existing scheduling scenario. Do not create a duplicate owner merely to use this suggestion.  
**Deliverable:** One self-contained product specification, implementation sequence, domain contract, UX description, and verification plan. No hosted prototype or second document is necessary to understand the intended application.

> Build a personal planning and focus application that helps people make realistic commitments, follow through, and learn to plan better from what actually happens.

**Reading guide:** Sections 1–18 define the product and its behavior; 19–23 define implementation contracts; 24–25 define examples and verification; 26–29 define delivery, defaults, and completion. Start implementation with P00 in section 26 after reading the full specification.

**Contents**

- [1. Implementation mandate and authority](#1-implementation-mandate-and-authority)
- [2. Product outcomes and release boundaries](#2-product-outcomes-and-release-boundaries)
- [3. Repository discovery and legacy transition](#3-repository-discovery-and-legacy-transition)
- [4. Domain concepts and semantic boundaries](#4-domain-concepts-and-semantic-boundaries)
- [5. User journeys and information architecture](#5-user-journeys-and-information-architecture)
- [6. Approved visual design and production interpretation](#6-approved-visual-design-and-production-interpretation)
- [7. Today, capture, and everyday task interaction](#7-today-capture-and-everyday-task-interaction)
- [8. Calendar, hybrid planning, and recurrence](#8-calendar-hybrid-planning-and-recurrence)
- [9. Goals, milestones, commitments, and dependencies](#9-goals-milestones-commitments-and-dependencies)
- [10. Capacity and effort accounting](#10-capacity-and-effort-accounting)
- [11. Planning engine, proposals, and what-if behavior](#11-planning-engine-proposals-and-what-if-behavior)
- [12. Forecasts, risk, and explainable delays](#12-forecasts-risk-and-explainable-delays)
- [13. Focus sessions and actual activity](#13-focus-sessions-and-actual-activity)
- [14. Review, learning, and preference calibration](#14-review-learning-and-preference-calibration)
- [15. Shared Vrooli scheduling contracts](#15-shared-vrooli-scheduling-contracts)
- [16. Read-only external calendars and interoperability](#16-read-only-external-calendars-and-interoperability)
- [17. Selective read-only sharing](#17-selective-read-only-sharing)
- [18. Notifications, jobs, and bounded assistance](#18-notifications-jobs-and-bounded-assistance)
- [19. Persistence model and data integrity](#19-persistence-model-and-data-integrity)
- [20. Application commands, queries, and API behavior](#20-application-commands-queries-and-api-behavior)
- [21. CLI, tools, and native program integration](#21-cli-tools-and-native-program-integration)
- [22. Application architecture and frontend implementation](#22-application-architecture-and-frontend-implementation)
- [23. Reliability, privacy, portability, and operations](#23-reliability-privacy-portability-and-operations)
- [24. Worked examples and canonical fixtures](#24-worked-examples-and-canonical-fixtures)
- [25. Verification strategy and acceptance tests](#25-verification-strategy-and-acceptance-tests)
- [26. Detailed implementation sequence](#26-detailed-implementation-sequence)
- [27. Requirement traceability and release checklist](#27-requirement-traceability-and-release-checklist)
- [28. Configuration defaults, unresolved discovery, and risks](#28-configuration-defaults-unresolved-discovery-and-risks)
- [29. Handoff, definition of done, and reference notes](#29-handoff-definition-of-done-and-reference-notes)

## 1. Implementation mandate and authority

### 1.1 What to build

Create a persistent, usable Vrooli application connecting goals, projects, milestones, tasks, routines, commitments, calendars, focus sessions, and learning. It must serve both as a standalone personal planner and as the shared scheduling and availability service for specialized Vrooli applications. Daily, a food-planning application, and Cadence, a content-planning application, are the originating integration examples. Their display names do not establish actual repository slugs, API availability, or implementation status.

The ordinary experience is simple: inspect a realistic day, choose the next useful action, work, record a small correction when necessary, and see how that affects the plan. The system should progressively reveal deeper planning tools. Advanced modeling must not turn everyday use into continuous data entry.

The originating problem is repeated optimistic commitments, overloaded days, unclear completion criteria, and difficulty explaining how interruptions, misunderstandings, waiting, and extra work affect timelines. The product must improve visibility before commitments fail and provide evidence for better future planning. It is not a promise that software will guarantee punctuality or maximize productivity.

### 1.2 Authority of statements

| Classification | Meaning | Implementation treatment |
| --- | --- | --- |
| Confirmed | Explicitly requested or accepted in the conversation | Preserve unless the user changes the decision. |
| Approved design direction | User selected the Observatory layout with day/night landscape environments | Reproduce the visual hierarchy and atmosphere, then fix mockup shortcuts. |
| Recommended default | Specific implementation choice supplied by this plan | Implement or record a concrete repository-grounded reason to adjust. |
| Discovery item | Requires inspection of the actual Vrooli repository or user configuration | Resolve locally; never substitute a historical memory for evidence. |
| Later capability | Retained vision outside the first complete release | Preserve affordable extension points; do not claim it already works. |

Within the selected release, “must” describes required behavior. Examples labeled illustrative are fixtures, not assertions about the user's appointments, habits, productivity, or account connections.

### 1.3 Accepted decisions

| ID | Decision | Consequence |
| --- | --- | --- |
| D01 | Suggestions and previews, applied by the user | Background work may compute forecasts and proposals. It must not silently move accepted work sessions. |
| D02 | Hybrid planning | Support fixed events, flexible timed sessions, day-assigned effort, and an unscheduled backlog. Day-assigned effort consumes capacity. |
| D03 | Individual users with private workspaces | Build tenant isolation from the start. Household and team editing are later features. |
| D04 | Optional read-only external calendar connections | Bring external obligations into capacity calculations. No provider event mutation is needed for launch. |
| D05 | Optional timers, quick corrections, and brief reviews | Manual use remains complete. Missing activity data is unknown, never automatically productive time. |
| D06 | Selective read-only commitment sharing | The owner explicitly selects records and fields. Guests cannot edit the calendar or see unrelated activity. |
| D07 | One shared source for scheduling across apps | Source apps keep their own UX and domain content; schedule changes use shared contracts. |
| D08 | Observatory UX with coordinated day and night scenes | Preserve the composition and location across appearances; provide Auto, Day, and Night controls. |
| D09 | Local Vrooli implementation | This handoff author has not inspected the destination repository. The local agent must do so before choosing framework, schema, or service details. |

### 1.4 Agent working rules

1. Read this file fully and inspect local `AGENTS.md`, testing instructions, architecture, discovery mechanisms, and relevant existing scenarios before implementation.
2. Maintain this specification's intent and requirement IDs. Store implementation decisions and gate evidence alongside the canonical repository copy.
3. Deliver complete vertical slices: interaction, domain command, persistence, recalculation, failure behavior, and meaningful verification.
4. Use actual repository conventions for languages, frameworks, auth, service discovery, shared UI, jobs, and migrations. Example TypeScript types describe contracts and do not mandate the server language.
5. Keep facts, estimates, forecasts, user reports, and AI suggestions distinguishable.
6. Do not call a demo complete because cards and sample calendars render. Exercise persistence, multiple browser sessions, non-happy paths, and the end-to-end planning loop.
7. Do not overwrite source-domain content, existing production data, or old scheduling records to simplify implementation.
8. Every visible action must work or visibly explain its supported prerequisite. Do not ship controls that silently do nothing.
9. Do not require an AI provider for deterministic planning, timers, capacity, sharing, or history.
10. Use the defaults here to resolve routine choices. Record genuine product conflicts and continue independent work while they are resolved.

## 2. Product outcomes and release boundaries

### 2.1 Outcomes

The application should help users answer:

- What matters and what have I actually promised?
- How much usable time do I have, including its fragmentation?
- What work fits before the next fixed commitment?
- What does accepting one more task displace?
- What remains to finish, and when is completion currently feasible?
- Why did a forecast change?
- What patterns in my own history should change future plans?

Success measures should emphasize usable information and follow-through: fewer unrecognized overloads, earlier identification of risks, less repeated planning work, clear commitment changes, and better estimate calibration when enough data exists. Do not optimize for filling every gap, streak length, or maximizing tracked hours.

### 2.2 Releases

| Release | Purpose | Required scope |
| --- | --- | --- |
| R0: persistent foundation | A reviewable vertical slice | Identity, private workspaces, setup, native tasks/events, correct time semantics, manual hybrid planning, Today, basic week/agenda, shared schedule read/write contracts, approved theme foundation. |
| R1: first complete personal product | The user's actual planning–focus–learning loop | Goals, milestones, commitments, dependencies, capacity, proposal/apply, explainable forecasts, actuals, focus modes, review, basic history-based recommendations, recurring routines, read-only external integration, selective sharing, notifications, portability, mobile/accessibility, and source adapter verification. |
| R2: stronger assistance | Improve planning sophistication with measured evidence | Additional providers, richer what-if comparisons, calibrated statistical forecasts where supportable, richer learning controls, optional conversational assistance, advanced scheduling search. |
| R3: broader collaboration | Separate product expansion | Opt-in automatic flexible rescheduling, shared household editing, team capacity/assignment, provider writeback, commercial billing if selected. |

R0 is not the promised complete application. R1 is the default finish line for the implementation effort described here. R2/R3 items are not preconditions for personal use. Build R1 in several demonstrable increments, not as one large unreviewable change.

Optional means the user need not enable a capability, not that an accepted R1 feature can be replaced by a nonfunctional button. At least one real read-only calendar provider must be implemented and validated before declaring the integration gate complete. If credentials are unavailable, ship manual use, test fixtures, setup instructions, and an explicitly unverified provider gate.

### 2.3 Not part of R1

Two-way external calendar sync; emailing invitations without a user action; automatic renegotiation of promises; employee monitoring; automatic screenshots or keystroke capture; medical conclusions about attention or energy; billing; group resource optimization; arbitrary business workflow orchestration; full recipe/content editing; unrestricted autonomous agents; guaranteed deadline probabilities without calibration evidence.

### 2.4 Guard against two failure modes

**A generic calendar with extra tabs:** It would miss remaining-effort modeling, commitment history, delays, capacity, and learning. These are core requirements.

**An all-purpose project-management platform:** It would duplicate existing domain owners and overwhelm personal use. Start with simple objects and progressive disclosure, and integrate authoritative records where they already exist.

## 3. Repository discovery and legacy transition

### 3.1 Required inventory

Before writing migrations or choosing the scenario identity, produce a short evidence-backed inventory in the implementation log.

| Area | Inspect | Decision to record |
| --- | --- | --- |
| Old scheduler | Scenario slug, routes, database, jobs, users, dependents, event IDs | Reuse identity, replace internals, or create successor with migration adapter. |
| Runtime | Current service language, UI stack, package manager, routing, deployment | Follow active conventions rather than an old architectural description. |
| Auth | Session model, user identity, scopes, tenant isolation, guest support | Reuse identity; define owner and viewer capabilities. |
| Tasks/goals/projects | Which scenarios own which concepts and whether they are operational | Reference owners or supply native records when no usable owner exists. |
| Notifications | Notification-hub/Fama or current equivalent; delivery receipts, quiet hours | Publish domain notifications through the existing owner. |
| Events | Existing event bus, receipts, replay, schemas, retry mechanisms | Reuse transport and envelope; do not start an independent bus. |
| Integrations | Calendar/credentials/connectors already available | Reuse secure adapters and credential storage. |
| UI | Shared accessible controls, themes, date pickers, calendar components | Reuse primitives while preserving the approved visual design. |
| Agents/programs | Current CLI, tool, skill and program-runtime conventions | Expose bounded domain verbs with the same policies as the UI. |
| Daily/Cadence | Current source app identities, owner contracts, existing scheduling | Implement actual adapters only where the owner exists. |
| Operations | Metrics, audit logging, backups, migrations, job runner | Integrate rather than invent parallel management paths. |

A reference in documentation is not proof a capability works. Record the command or source location that establishes it and, when applicable, an observed runtime check.

### 3.2 Decision procedure

Prefer replacing or extending the existing calendar scenario if it remains the recognized scheduling owner and can support the new model without unsafe compatibility compromises. Use a successor scenario when the old API/data model is materially incompatible, but provide an explicit owner map and cutover path. Never leave both services accepting independent authoritative writes to the same schedule.

Keep framework selection local. Do not migrate the whole ecosystem to match this plan. The domain boundaries and semantics are mandatory; folder names and package choices are illustrative.

### 3.3 Legacy data handling

Inventory records and dependencies before changes. Back up data and validate a restore in the normal repository workflow. Produce a mapping for old events, recurrences, reminders, task references, timezones, and completion fields. Unknown timezone or ambiguous completion status must produce an import issue, not an invented fact.

Use stable ID mappings, repeatable migration commands, dry-run counts, per-record errors, and a migration version. Mark migrated provenance. Do not treat old scheduled events as actual activity. Keep unsupported records available for inspection/export. Verify recurrence expansion and representative dates before cutover. Stop old write-producing jobs only as part of the defined cutover. Provide a rollback path that accounts for new writes after migration; a database snapshot alone is not a complete rollback design.

The old UI may temporarily link to the new app, but there must be one documented write authority. Retire compatibility endpoints only after callers are identified and migrated.

## 4. Domain concepts and semantic boundaries

### 4.1 Core objects

| Object | Meaning | Essential distinction |
| --- | --- | --- |
| Goal | Desired outcome and evidence of progress | Allocating time is not achieving the outcome. |
| Initiative/project | A bounded grouping of related work | May be owned by another Vrooli scenario. |
| Milestone | Observable checkpoint with completion criteria | Does not consume time itself; its work does. |
| Work item | A task or other unit requiring human effort | Can take multiple sessions and can be only partly complete. |
| Event | Something fixed or externally scheduled | Attendance is not inferred from its scheduled end. |
| Routine definition | Repeating intention or schedule | A cadence target is not necessarily a fixed event recurrence. |
| Routine occurrence | A dated instance of a routine | Skip/move one occurrence without rewriting the series. |
| Schedule allocation | Accepted reservation or date-level allocation of effort | Separate from task identity, total estimate, and actual activity. |
| Commitment | Explicit promised result or availability with an agreed boundary | Forecasts never silently change it. |
| Focus session | A user's working session | May contain work, pauses, and breaks; timer end is not task completion. |
| Actual activity | Recorded evidence of elapsed active work or other activity | May be timed, manually reported, or corrected. |
| Forecast | Derived completion outlook from an input snapshot | Does not reserve time or alter an accepted schedule. |
| Proposal | A reviewable candidate change set | Applying it is a separate revision-checked command. |
| Delay/change record | Facts and optional explanation about a meaningful change | Observed difference and claimed cause are different fields. |

### 4.2 Time values that must never collapse into one field

Store original effort estimate, revised estimate history, explicit remaining effort, desired finish, committed finish, earliest allowed start, scheduled sessions, latest forecast, actual finish, and completion criteria independently. A user may omit most of them for a simple task. The UI reveals advanced distinctions when relevant.

An estimate range describes uncertain active effort. A completion forecast concerns elapsed calendar time and includes capacity, sequencing, and waiting. A 60-minute task blocked until next week remains 60 minutes of work but may have a next-week finish.

Source apps may have domain timestamps such as publication time or a meal slot. They must explicitly declare whether those timestamps reserve human attention. Automated publication is normally a marker; drafting and review consume human time. Oven time or agent runtime is not automatically exclusive human work.

### 4.3 Ownership

The planner owns native planning records, accepted scheduling allocations, availability, private activity records, forecast snapshots, proposals, and sharing projections. A source app owns its content and authoritative domain status. External calendar providers own their imported events. Imported and referenced objects are projections with origin identity and freshness.

An operation such as “mark story published” belongs to Cadence. An operation such as “move my drafting session to tomorrow” belongs to the planner, subject to declared source constraints. Recipe restrictions remain Daily's rules. The planner may request an alternative but must not invent eligibility or silently alter a meal.

### 4.4 Invariants

| ID | Invariant |
| --- | --- |
| INV-01 | All reads, writes, jobs, projections, and caches are scoped to the authorized workspace and subject. |
| INV-02 | One authoritative owner exists for each mutable fact; projections retain origin and revision. |
| INV-03 | Passing scheduled time, running a timer, and completing a task are separate events. |
| INV-04 | Forecast/proposal generation cannot alter accepted schedules or commitment revisions. |
| INV-05 | Past actuals and original promises remain available after corrections, subject to explicit deletion/privacy operations. |
| INV-06 | Fixed commitments, protected time, source constraints, and locks constrain proposal placement. |
| INV-07 | Date-assigned work consumes capacity; a timed session linked to that work must not count twice. |
| INV-08 | Unknown effort, unavailable provider state, and unrecorded activity cannot silently become zero. |
| INV-09 | One person cannot have two simultaneously running exclusive focus sessions. Breaks and pauses are not active effort. |
| INV-10 | Repeated requests and replayed events cannot create duplicate allocations, actuals, shares, or notices. |
| INV-11 | Applying a proposal revalidates all material inputs and commits its local changes atomically. |
| INV-12 | Shared views expose only explicitly allowed fields and records, including nested data and derived explanations. |
| INV-13 | Routine exceptions retain stable occurrence identity after rescheduling. |
| INV-14 | Learning recommendations never silently rewrite promises, hard constraints, or historical measurements. |
| INV-15 | Forecast allocation is hypothetical and never counted as accepted reserved time. |
| INV-16 | UI geometry, accessible text, API results, and analytics describe the same time values. |

## 5. User journeys and information architecture

### 5.1 Destinations

Primary navigation is **Today, Plan, Goals, Focus, Review**. Search/quick capture is globally reachable. Settings contains availability, appearance, focus preferences, integrations, sharing management, notifications, data, and learning controls. Commitments have a dedicated filter/view inside Plan and contextual access from Today; promote them to primary navigation only if usage demonstrates the need.

Plan provides Day, Week, Month, Agenda, Timeline, Capacity, and Commitments perspectives. Do not build seven independent stores. These are projections and interactions over the same records. Goals contains outcomes, milestone status, linked work, and time allocation. Focus may be opened without a task for spontaneous work, with optional classification later.

### 5.2 First-use journey

Ask short, skippable questions over approximately five steps:

1. Confirm timezone and week start; show the detected timezone as a suggestion, not a silent assertion.
2. Establish typical availability and protected personal periods. Allow different weekdays and no regular schedule.
3. Ask what the user wants help with: overload, estimating, starting, commitments, or understanding time. These prioritize explanatory UI; they are not diagnostic labels.
4. Offer read-only calendar connection and source-app discovery. Skipping leaves full manual setup available.
5. Capture one meaningful task or commitment with a rough estimate, then produce a small draft day for review.

Show an immediate usable result before requesting a detailed goal hierarchy. An advanced step can collect preferred session lengths, fragmentation tolerance, recurring routines, and maximum daily demanding-work allocation. Do not infer availability from all waking hours or assign a universal focus limit.

The first draft identifies assumptions. The user explicitly chooses Apply to my day. Imported fixed events may appear immediately as external facts; they must not be described as accepted new work. A fresh workspace contains no fictional personal commitments. Offer an explicitly separate example workspace for demos.

### 5.3 Everyday journeys

**Morning:** Today shows available capacity and relevant commitments; the user adds or selects a manageable set of work, inspects a proposal, and applies it.

**Starting:** Start focus opens the linked session goal and domain material. Starting spontaneously records actual activity without silently reshuffling the calendar.

**Interruptions:** Pause or report blocked; optionally capture a thought. Resume later from a short note. No reason is mandatory.

**Change:** Update remaining effort or record a new obligation. Current forecasts update; accepted future schedule changes appear as proposals.

**Evening:** A brief review records important differences, remaining work, and tomorrow's considerations. Unfinished work does not automatically become an endless overdue queue.

**Weekly:** Inspect commitments, feasible capacity, progress toward goals, and one or two evidence-linked insights. Apply a chosen planning adjustment and retain the previous setting in history.

**Before promising:** Capture result/definition of done, inspect a forecast with assumptions, and explicitly record the commitment. The application may warn but does not prevent the user from knowingly making an ambitious promise.

**Sharing:** Select commitments, preview exactly what a guest would see, grant access, and copy a link. The viewer gets no editing tools or access to the owner's full calendar.

## 6. Approved visual design and production interpretation

### 6.1 Design intent

The user rejected a functional but generic white-card/blue-button dashboard. The selected direction uses the Observatory composition with a coordinated landscape that changes between daytime and nighttime. Beauty is a product requirement alongside clarity. Do not substitute a default component-library dashboard and label it equivalent.

The recurring environment is a small observatory on a nearby hillside overlooking a valley, lake, and layered distant mountains. Daytime shows warm sunlit terrain and atmospheric depth. Nighttime depicts the same buildings, trees, shoreline, and viewpoint with dark silhouettes and restrained settlement lights. Keep the scene mostly below the main working area. It should create a sense of place without compromising the usability of text, inputs, or the timeline.

There is no approved permanent product name, logo, motivational slogan, or commercial claim. The reference labels Planner and Observatory are replaceable configuration. Decorative slogans accidentally present in generated concepts are optional and should be omitted from the working UI by default.

### 6.2 Desktop composition

At an approximately 1440–1600 px desktop width:

- Use a narrow left navigation rail around 120–152 px wide. Put the wordmark at top, primary destinations below, and profile/settings near the bottom. Source apps are contextual labels, not a second large navigation hierarchy.
- Give the main canvas around 32–48 px outer padding, adjusting with viewport width. Use a responsive grid rather than fixed image coordinates.
- Put Today and the readable local date in an expressive serif heading. Keep the heading around 42–72 px depending on width, with a maximum so dates do not push useful work below the fold.
- Place Auto / Day / Night appearance control at the top right, with accessible selected state.
- The upper working row contains one prominent next-action surface around 58–62% of usable width and a quieter capacity/commitment region in the remainder.
- The next-action surface has an eyebrow, task title, source, duration or uncertainty label, session objective, dominant Start focus action, and quieter Open source action. Avoid more than one primary action.
- Capacity uses aligned numerals and labels, fine separators, and clearly scoped totals. The commitment notice remains visible but visually secondary until selected.
- A wide horizontal Today timeline sits below the working row. It has a real time scale, a now indicator, fixed/flexible distinctions, and open space that means something.
- The lower 20–30% may contain the scenic environment when there is room. Content can extend beyond it; the art is never a fixed overlay obscuring scrollable work.

At 1024–1279 px, shrink the rail or collapse its labels, reduce heading size, and stack the commitment summary below capacity if needed. At narrower tablet widths, stack the action and capacity regions. Avoid truncating the sole accessible representation of a task title.

### 6.3 Recommended initial tokens

These values are starting points to tune and verify, not sampled exact colors from a generated image.

| Token | Day | Night |
| --- | --- | --- |
| Canvas | `#F5F0E6` warm ivory | `#141321` midnight plum |
| Raised surface | `#FFFDF7` | `#201E30` |
| Primary text | `#192622` forest ink | `#F5EEDF` warm cream |
| Secondary text | `#565E59` | `#C0B9CB` |
| Primary action | `#A64327` with near-white text | `#F2B75A` with near-black text |
| Divider | `#D7CEC0` | `#444054` |
| Cadence example accent | `#7055AE` | `#B69DE7` |
| Daily example accent | `#456B58` | `#A6C6B5` |
| Attention | Deep burnt orange with label/icon | Warm amber with label/icon |

The night next-action surface may remain warm ivory with dark text, matching the approved concept; offer a subdued surface setting if it is uncomfortable in low light. It must use its own on-surface tokens instead of inheriting white text. Use a licensed or repository-approved expressive serif for headings and a highly readable sans serif for controls and dense labels. Fraunces or an equivalent editorial serif plus the existing UI sans is a possible starting point; verify font availability and licensing locally. Use tabular numerals for timers and aligned time values.

Prefer restrained 8–14 px radii on major surfaces, thinner rounding on time blocks, subtle shadows in day appearance, and borders/surface separation in night appearance. Avoid repeated large shadowed cards, aggressive neon, uncontrolled glass transparency, and decorative charts.

### 6.4 Scene assets

Treat the landscape as a replaceable decorative asset with meaningful accessible content entirely outside it. Author separate day/night assets from the same master composition or use a matching pair. Export responsive sizes and a modern compressed format with a compatible fallback. Never bake task text, buttons, the date, or timeline into the image.

Suggested asset brief: “A restrained, finely rendered hillside observatory in the bottom-left foreground, overlooking a lake and distant valley settlements, layered mountains across the horizon, plenty of quiet sky above; straight-on wide panoramic environment designed behind a readable app; matched daylight and night versions from exactly the same viewpoint; no text, logos, people, enormous moon, aurora, or galaxy.”

Use locally bundled or appropriately cached assets; a third-party image host must not be a prerequisite for opening Today. Decorative assets should not block the initial task view. If artwork is unavailable during early development, a deliberate tonal placeholder is acceptable for R0, but the approved scenic experience is a visual release gate for R1. Provide an art-free setting and reduced scenery intensity. Avoid generating a new image on every visit.

### 6.5 Appearance behavior

Support Auto, Day, and Night. Default Auto uses the user's local display timezone and editable daytime/nighttime transition hours; initial suggested values may be 07:00 and 19:00 and must be visible as defaults. It does not require location access or claim accurate sunrise/sunset. System appearance can be an optional alternative auto policy. Astronomical transitions are later refinement if wanted.

Manual override persists per user; device-specific override may be added if the settings model supports it clearly. Theme changes do not alter scheduling timezone, data selection, current date, or plan. During focus, defer an automatic transition until the next break or session boundary. Respect reduced motion; otherwise use a restrained opacity/color transition without moving content. A “Night” appearance selected at 10:00 is valid.

### 6.6 Time geometry and visual semantics

On a linear timeline, position and width represent time accurately. A 45-minute interval cannot occupy 90 minutes to accommodate text. Use an adjacent label, popover, stacked collision lane, or alternate agenda representation. Day and week scales must use actual instants when DST changes produce nonstandard days. A collapsed nonworking interval must have an explicit break marker; never silently distort the scale.

Represent fixed commitments with an anchor/lock and clear boundary; flexible sessions with move affordances; pending proposals with labeled original/proposed geometry; actual activity with a distinct comparison layer. Forecast ranges must be labeled as scenario ranges or calibrated intervals, whichever applies. Color supplements shape/text rather than carrying sole meaning.

Distinguish **unscheduled time**, **unavailable time**, and **reserved breathing room**. White space can include nonworking time; it is not automatically usable capacity. Show a legend or explanation when relevant. Source colors remain semantically stable across themes, even if shades change for contrast.

### 6.7 Mobile and accessibility

Phone Today defaults to a vertical agenda, next-action surface, compact capacity line, and commitment notice. Use a bottom navigation with the five main destinations or an equivalent accessible pattern. A compressed horizontal desktop timeline is not the primary phone experience. Week can become a day-grouped list with daily capacity; detailed Timeline can scroll with clear context.

All drag operations need keyboard and menu alternatives. Task start/stop must be reachable without precise pointer movement. Announce meaningful timer state changes, not every ticking second. Ensure visible focus, sensible focus return from dialogs, semantic headings, reflow, text zoom, and no color-only status. Target WCAG 2.2 AA; verify normal text contrast of at least 4.5:1, relevant non-text contrast, and the standard's target-size requirements and exceptions. Use larger approximately 44 px touch targets where practical. See [WCAG 2.2](https://www.w3.org/TR/WCAG22/).

### 6.8 Visual verification gates

Review Today in both appearances, a busy week, an empty workspace, a long title, many overlapping fixed events, a commitment at risk, a running focus session, a narrow phone, and 200% zoom. Check actual screenshots and interact with the app. Verify landscape contrast in the brightest and darkest regions. The screenshots from the conversation are style references; their example arithmetic, block widths, repeated slogans, and crowded labels are not authoritative behavior.

## 7. Today, capture, and everyday task interaction

### 7.1 Today hierarchy

Today must answer “What can I usefully do now?” and “What needs my attention?” The primary card is selected from currently feasible work using explicit user order, commitments, availability, dependencies, and session fit. Show a short reason when helpful, such as “Fits the next 40 minutes and unblocks tomorrow's recording.” Never claim an item is objectively the best use of life.

Offer an editable session objective distinct from the task's full completion criteria. A task named Build installer may have today's objective Reproduce the Windows launch issue. Preserve that objective in the session record. Let the user choose a different next action without a lengthy explanation.

Capacity scope must be clear: remaining today, whole day, or selected week. “3 h planned / 4 h available / 1 h reserve” is valid only if all three use the same accounting model. If demand exceeds room, show the amount and a review action. If estimates are missing, display “3 h known + 2 unestimated tasks.”

Commitment notices show result, promised date, current outlook, freshness, and a short reason or missing-data status. Use a small number of high-value notices, not a wall of alarms. Explicitly protected personal time is visible with the same dignity as work obligations.

### 7.2 Quick capture

Allow a name-only task. Common inline additions are estimate, desired day, goal/project, due boundary, and source link. Advanced options include effort range, earliest start, minimum session, split policy, dependencies, definition of done, and commitment conversion. Do not require every field for every task.

Natural-language capture may parse “draft tomorrow for 45 minutes,” but show interpreted fields before material changes. Ambiguous dates or timezone-sensitive expressions need a visible interpretation. Manual fields remain available. Capture must work without a model provider.

The inbox/backlog has no implied promise or allocated capacity. Moving an item to a day turns it into planned demand even without a clock time. A name-only day assignment remains visible as unestimated demand and limits confidence in the capacity assessment.

### 7.3 Task detail

Task detail includes title, source/origin, status, goal/project links, definition of done, estimate history, remaining effort, scheduled allocations, actual activity, dependencies, notes, and relevant commitment/forecast. Show the most useful subset initially. Group history and advanced constraints in disclosure panels.

Native tasks can be edited directly. For external/source-owned fields, show the owner and a supported Open in source or owner command. A local planning alias may be allowed but must not masquerade as the source's title. Completion must route to the correct owner when that owner controls status.

### 7.4 Common operations

Support rename, estimate, assign to a day, schedule session, split, move, lock, unschedule, skip occurrence, complete, reopen, archive, and undo where meaningful. Unscheduling a session does not delete its task. Completing a task with future sessions offers a preview to release those sessions; retain earlier history. Reopening adds a new status revision and asks for remaining effort when needed.

Delete/archive behavior follows object ownership. Archiving a task preserves linked historical actuals. Destructive deletion has a clear consequence summary and follows the data-removal rules. Fast reversible changes may use Undo; cross-record or promise changes need an explicit preview.

### 7.5 Empty and unavailable states

| State | Required behavior |
| --- | --- |
| No tasks | Offer capture, routine setup, or optional source connection. No fake personal plan. |
| Nothing fits | Explain the limiting window, block, or missing estimate; offer a smaller session or a different task. |
| Over capacity | Show known excess and unknown demand; offer scope/order/day adjustments. |
| Source unavailable | Keep last-known projection with freshness and safe local schedule actions; disable unsupported owner writes. |
| Saving failed | Keep the user's input, show pending/failed state, allow retry/export; never claim saved. |
| Stale forecast | Mark why stale and recompute when possible; do not silently display an old result as current. |
| No history | Explain that learning needs records; provide editable manual defaults. |
| Task blocked | Show the dependency or user-reported block and optional next check-in. |

## 8. Calendar, hybrid planning, and recurrence

### 8.1 Native events and allocations

Native fixed events support title, start/end or all-day dates, timezone, busy/free effect, optional location/link, privacy, reminders, recurrence, and notes. A deadline marker need not reserve any attention. Define “Show on calendar” and “Consumes time” independently where the domain warrants it.

An allocation can be timed or assigned to a local date/part-of-day. Timed allocations include actual start/end instants and planned effort. Date-level allocations include local date, planning timezone, effort quantity, and optional preferred windows. Both have accepted/draft/cancelled states and links to their work item or routine occurrence.

Moving a date-level 90-minute allocation into a 30-minute session must consume 30 minutes from that date allocation and leave 60 minutes date-assigned, rather than creating 120 minutes of demand. Support a planning-demand parent with child session allocations or an equivalent conserved-quantity model. The UI must explain splits and remainder.

### 8.2 Views

Day and Week allow placement and adjustment; Month emphasizes commitments, milestones, and daily load rather than tiny task text. Agenda is fully usable with a keyboard and screen reader. Timeline shows dependency-driven work and milestones over longer horizons. Capacity shows demand against usable time across days/weeks. Commitments shows result, owner, promise, forecast, status, latest change, and check-in.

Filters include source, goal/project, event/task/routine, commitment status, and privacy. Filtering the visible items must not remove hidden obligations from feasibility calculations. Display “Capacity includes all calendars” where needed. Use query state in deep links; returning to the originating app should preserve context.

### 8.3 Editing behavior

Dragging a flexible session creates a local preview with conflicts and ripple effects. A simple isolated move can be applied with a clear confirmation or the existing direct-manipulation convention plus Undo. Moving an item must not silently move other accepted items. The gesture authorizes that move, not a global optimization.

Dragging a fixed event edits it only if the user owns it and explicitly chooses the edit. Read-only provider events open provider context and cannot be dragged into a mutation. If an external fact creates a conflict, keep both facts visible and propose moving local flexible work.

Lock means preserve placement in automatic proposals. Fixed means the object's nature or agreement constrains placement. A user can explicitly unlock a flexible session; an external event's edit permissions do not change because a lock icon is clicked.

### 8.4 Native routines

Support two distinct types:

- **Fixed recurrence:** e.g., a weekly appointment at a particular local time. Generate occurrences using a supported recurrence library.
- **Flexible frequency:** e.g., three runs per week or one weekly review within a window. Generate planning demand with frequency, eligible days, effort, and optional spacing rules; do not create duplicate fixed appointments.

R1 routine editor supports daily, selected weekdays, weekly, interval, and common monthly patterns, with start date, optional end/count, timezone, effort, and exception controls. Unsupported imported recurrence remains inspectable and must not be rewritten into a superficially similar pattern.

Every recurrence edit asks whether it affects this occurrence, this and following, or the series. Persist occurrence overrides separately. A series revision changes future instances without rewriting actual activity or already completed occurrences. R1 must define behavior for moving an occurrence across the week boundary; frequency fulfillment uses the configured routine period and counts an occurrence once.

### 8.5 Civil time and DST

Use local dates for date-only concepts, instants for actual timed intervals, and a named timezone for recurring wall-clock schedules. Never represent an all-day item by an invented UTC midnight task. Display zone and schedule zone may differ; changing display zone must not move a fixed instant.

For a newly entered ambiguous or nonexistent local time, show the resolved meaning and allow correction. For native recurring schedules, choose and document a deterministic policy: default shift a nonexistent time forward by the DST gap; use the earlier offset for a repeated time unless an occurrence override selects otherwise. Persist the policy and visibly flag adjusted occurrences. Source-owned recurrence uses provider semantics, not the native policy. A real 60-minute session spans 60 elapsed minutes even when local clock labels repeat or skip.

For date-only commitments, capture whether the intended boundary is end of local day or a specific finish time. The UI must show that boundary. Date-only milestones do not require minute precision, but forecasting against their deadline needs an explicit zone and boundary.

Preserve iCalendar date, recurrence, and exception semantics using a mature parser and tests. In particular, maintain exclusive end boundaries and original recurrence identities. See [RFC 5545](https://www.rfc-editor.org/info/rfc5545/) for the interchange standard; the native product policies above are explicit implementation defaults.

## 9. Goals, milestones, commitments, and dependencies

### 9.1 Goals and initiatives

Native goals have name, purpose, optional target horizon, status, and progress method. Methods include manual assessment, an explicitly defined metric, or milestone completion. Linked hours are effort exposure, not automatic outcome progress. For numerical metrics, store baseline, target, unit, direction, observed values with dates, and origin.

Allow a small number of active priorities without imposing a universal number. Show goals with no linked planned work and work with no goal as different valid states. Maintenance, rest, relationships, and obligations can be intentionally allocated without requiring a productivity goal.

Initiatives group work and milestones. If a project manager already owns them, store a reference and the minimum scheduling projection; do not mirror full project editing. Native lightweight initiatives exist for standalone users.

### 9.2 Milestones

A milestone has observable completion criteria, linked prerequisite work, optional target and commitment, status, and forecast. Completion can be manually confirmed or derived only from a declared rule over authoritative prerequisite status. A nonempty checklist is not inherently a complete deliverable. Show what evidence is missing.

Example: “Installable release verified” requires packaging, required tests, and a successful installation check. The planner understands dependencies and status; it does not claim software verification from a task title.

### 9.3 Commitments

Creation captures result or availability promise, definition of done, promised boundary, timezone, optional beneficiary, assumptions, and optional scope exclusions. Beneficiary can be private free text; selecting a person does not automatically share or send anything. Distinguish a draft intention from an explicitly accepted commitment.

Store original promise and every revision. A forecast update is a derived event; a renegotiation is an explicit user action with old/new boundary, optional reason, and optional acknowledgement status. If another person has not acknowledged a change, the product must not imply mutual agreement. The owner can record an agreement made elsewhere without forcing the other person to use the app.

Lifecycle states are proposed, active, fulfilled, and cancelled. Risk is a separate assessment; renegotiation is a revision event, not a competing lifecycle state. Cancellation and renegotiation do not count as on-time fulfillment in summaries.

For availability promises, support backward planning: “Ready to leave at 18:00” may reserve wrap-up and getting-ready intervals before 18:00. The latest safe work stopping point is calculated from explicit preparation/transition assumptions. Avoid notifying only when it is already time to leave.

### 9.4 Dependencies and waiting

R1 supports finish-to-start edges with optional nonnegative lag. Reject cycles. Edges are scoped to accessible records in the same personal workspace or an authorized source projection. Unknown/deleted/stale dependencies are visible unresolved conditions, not satisfied edges.

A waiting dependency has an expected release date/range or remains unknown. It consumes elapsed time, not exclusive attention, unless a monitoring/review task explicitly reserves effort. Model review after an agent run as human work. Do not assume more agents remove the user's review bottleneck.

Critical-path language should be used carefully: resource competition can delay tasks even when graph paths are independent. The forecast must schedule across the user's competing work, not simply sum one project's task durations in isolation.

Changing completion criteria or adding a dependency produces a scope/change record and invalidates relevant forecasts/proposals. It does not rewrite the original commitment definition.

## 10. Capacity and effort accounting

### 10.1 Capacity is both a budget and a set of intervals

A daily total alone cannot establish feasibility. A 90-minute indivisible task does not fit into six 15-minute gaps. Maintain both a normalized interval set and scalar summaries. All intervals are half-open `[start, end)` so adjacent items do not overlap.

For a person and local date:

1. Expand the explicitly configured availability windows into instants.
2. Subtract the union of fixed busy events, protected periods, and explicit fixed transitions that intersect them. Overlapping busy events subtract their union, not their summed duration.
3. Apply the configured daily workload cap if one exists. Do not invent a cap based on a population average.
4. Account for breathing-room reserve. A reserve may be a scalar amount or explicit reserved intervals; these representations must share identity so they cannot be subtracted twice.
5. Evaluate accepted timed occupancy, date-level demand, setup/break requirements, and remaining feasible fragments.

Define:

```text
potential_capacity = min(duration(usable_intervals), configured_daily_cap_if_any)
reserve = configured_uncommitted_allowance counted exactly once
allocatable_budget = max(0, potential_capacity - reserve)
planned_occupancy = timed_block_occupancy + date_assigned_occupancy
budget_spare = allocatable_budget - planned_occupancy
unallocated_room = potential_capacity - planned_occupancy
```

These are planning quantities, not measurements of actual work. Planned occupancy includes required setup and planned breaks when they occupy the block. Active effort is reported separately. If timed allocations overlap, report a conflict even when summed effort fits the daily budget. Exclude already elapsed time when computing remaining-today capacity; preserve whole-day totals for review.

The cap in this equation limits occupied flexible-planning time. A separate demanding-work cap limits active effort of the relevant work category. Label their units and scopes in settings; do not compare an active-effort cap directly with break-inclusive occupancy. Check both constraints when both are configured.

Example: four usable hours, one hour reserve, and three hours accepted occupancy produce three hours planned, four hours available, and one hour breathing room. There is zero spare task budget under the reserve policy. Display the reserve assumption when expanding the summary. Do not advertise the reserved hour as another hour available for commitments without noting the tradeoff.

### 10.2 Demand conservation

A work item has remaining active effort; an allocation describes some of that effort assigned to a day or session. Splitting a 90-minute day allocation into a 30-minute timed block leaves 60 minutes on the day. The 30-minute session is a child of the same planning demand, not an additional copy.

Use stable allocation lineage and a transactional split command. The aggregate planned effort for a given allocation parent equals the sum of its noncancelled children plus its unscheduled remainder. Replanning moves those quantities. Completing actual work updates observed effort and may update remaining effort, but never assumes a strict one-to-one reduction without an explicit progress rule or user correction.

Allocations in excess of explicit remaining effort are flagged as overallocated and offered for review. They can be intentional practice/review time, but that purpose must be distinguishable. Planned work beyond a task estimate is not silently discarded.

### 10.3 Scope and unknown data

Pure backlog ideas do not consume a specific day's capacity. Accepted work with deadlines enters the relevant forecast workload even if it has not been placed on a date. A date-level assignment consumes that date's budget. Flexible routine demand consumes its eligible period's budget and is checked for placement across days.

Unknown estimates are listed separately and prevent an unqualified “everything fits” assertion. Optional conservative placeholders may be used for a scenario if clearly labeled and editable. Keep the underlying estimate unknown. Source outages, missing dependencies, or stale calendars similarly lower the completeness of the feasibility result.

Apply unknown-demand warnings to the queried planning scope: an unestimated task assigned today matters to today's fit claim; an unaccepted backlog idea does not make every future day indeterminate. Show accepted deadline workload in forecast completeness even if it has no dated allocation yet.

For remaining-today calculations, clip usable and reserved intervals at now and include only future occupied portions of timed allocations. Keep unfinished dated demand in scope until explicitly completed, released, or revised. Passing the start/end of an allocation never establishes actual progress: unconfirmed past work remains unresolved, and its independently reported remaining effort still participates in forecasts. Expose work no longer covered by future sessions instead of assuming the elapsed schedule completed it.

Do not reset the daily cap at noon. Reduce the day's occupied-time cap by earlier effective flexible occupancy: retain elapsed accepted reservations as a conservative budget deduction until the user reconciles them, or use corrected actual occupied time when available, without counting both. This deduction is a planning assumption, not proof that work occurred. Apply only the remaining reserve balance; never deduct both a scalar reserve and its explicit intervals. Example: a 240-minute daily cap, 120 minutes of earlier accounted occupancy, 180 minutes of physically usable time left, and 60 minutes of unspent reserve permit at most 60 minutes of additional flexible occupancy. If 60 minutes are already planned later, spare budget is zero.

### 10.4 Energy and context

R1 can support user-declared preferred focus windows, demanding/light work labels, location prerequisites, minimum useful session length, and a demanding-work daily cap. These are editable preferences and constraints, not inferred medical or psychological states. Avoid a fabricated numerical energy meter.

Separate active preparation, passive wait, elapsed session time, and transition time. Parallel passive processes are permitted only if they do not reserve exclusive attention. Never double-book the same person's active effort based on an assumption that they can multitask.

### 10.5 Capacity explanations

Every summary should be expandable into configured windows, fixed exclusions, protected time, reserves, allocated work, unestimated demand, and conflicts. Explain why a time that looks blank is unavailable. Preserve the filter-independent calculation so hiding Daily tasks does not free their preparation time.

## 11. Planning engine, proposals, and what-if behavior

### 11.1 Deterministic core

R1 uses a deterministic scheduling core over validated inputs. It returns candidates and explanations without writing accepted records. AI may help capture text or summarize a proposal, but cannot substitute unvalidated prose for constraints or reservations.

Inputs include a person/workspace, horizon, planning/display zones, work and occurrence revisions, remaining-effort assumptions, authoritative status/dependencies, accepted allocations, availability and caps, fixed events, reserve policy, source constraints/freshness, user priorities, and planner version. Results include placements, unchanged/changed rows, unresolved demand, conflicts, capacity differences, milestone effects, assumptions, and input fingerprints.

### 11.2 Initial algorithm

Use this behavior as the baseline; implementation may use a mature solver if the repository already has one and equivalent explanations are retained.

```text
createProposal(snapshot, requestedScope):
    validateScopeAndOwnership(snapshot, requestedScope)
    normalizeIntervalsAndCivilDates(snapshot)
    identifyImmutableFactsAndUserLocks()
    validateDependenciesAndDetectCycles()
    deriveDemandWithoutDoubleCounting()
    preserveAcceptedPlacementsOutsideScope()
    reserveProtectedTimeAndBreathingRoom()
    queue = dependencyReadyItemsOrderedByExplicitPolicy()
    for item in queue:
        candidates = feasibleWindows(item, state)
        candidate = chooseStableLowDisruptionPlacement(candidates)
        if candidate exists:
            placeHypotheticallyAndUpdateReadyQueue(item, candidate)
        else:
            recordUnplacedItemAndLimitingConditions(item)
    boundedRepairOfFlexiblePlacementsWithinScope()
    evaluateAllHardConstraintsAgain()
    computeBeforeAfterCapacityAndForecasts()
    return immutableProposalWithInputRevisionsAndExplanations()
```

Hard constraints are checked before preference ranking. Initial ordering should be explainable and stable: user-pinned order where feasible, accepted near-term commitments and available slack, prerequisite readiness, user priority, goal preference, then stable identity as tie-breaker. Do not represent an arbitrary weighted score as objective human value. When priorities conflict, show which rule determined the recommendation.

Candidate selection prefers retaining an existing feasible placement, meeting constraints, reducing new fragmentation/transitions, fitting declared focus windows, and avoiding unnecessary movement. Favor schedule stability; adding a small task should not reshuffle the entire week.

### 11.3 Splitting and session policies

Each item declares whether it can split, minimum useful chunk, preferred chunk, maximum continuous effort if specified, setup cost per session, and whether a final shorter remainder is permitted. Suggested defaults are splittable 25–90 minute sessions for generic knowledge work, with all values editable; these are product defaults, not physiological claims. Sources can supply different constraints.

Breaks consume elapsed capacity when planned inside a session. For a 25/5/25 Pomodoro sequence, active effort is 50 minutes and occupancy is 55 minutes. If the user leaves the end break outside the reservation, do not append it twice. Session planning and timer preferences must use a shared duration calculation.

For an unsplittable item with no matching interval, distinguish “no suitable placement found by this planner” from a proven impossibility. A simple budget excess is provable; a heuristic placement failure may not be. Return reason codes and concrete alternatives without relaxing constraints silently.

### 11.4 Proposal UX and application

The preview shows original and proposed placements, locks, changed demand, remaining conflicts, and affected commitments. Recommended apply scope is selected unlocked future work. Users can deselect changes, but the remaining subset must be revalidated before acceptance. Partial selection can invalidate downstream placements.

Apply is a separate command with proposal ID, expected schedule revision, selected changes, and idempotency key. Recheck relevant source revisions, event changes, constraints, and permissions. If materially stale, return a stale-proposal response with the changed reasons and an option to refresh. Do not silently apply an old proposal using new assumptions.

Commit local allocation changes, the accepted plan revision, change history, and outbound events atomically. Undo creates a compensating change if affected revisions still match. If newer edits exist, offer a reverse preview instead of overwriting them.

### 11.5 What-if scenarios

R1 supports at least adding a new task, reducing scope/remaining effort, moving a deadline target, protecting a day, and changing available hours within a draft scenario. Compare changes against the accepted baseline. Hypothetical scenarios do not send notifications or write to source apps.

“What moves if I accept this?” should show displaced work and affected commitments. A reduced scope scenario creates an explicit hypothetical scope revision; applying it must route any source-owned scope change to that source or require the user to make it there.

### 11.6 Planning horizons

Default interactive scheduling horizon is the selected week plus a bounded lookahead, initially 28 days. Support longer milestone forecasts with a configurable maximum, initially 90 days. These are engineering defaults to verify against realistic workloads. Work that cannot finish within the horizon gets `beyond_horizon` or `unresolved`, never a fabricated finish date.

Distant routines and events are expanded on demand within bounded windows. A user can choose a longer explicit forecast; show when data coverage or source freshness does not support it.

## 12. Forecasts, risk, and explainable delays

### 12.1 Forecast model

Forecast all competing accepted work in one personal resource model. Do not let independent project forecasts each assume exclusive use of the same free hours. Use a shared snapshot and scenario allocation across the workload, then derive each task/milestone finish from it.

A forecast must include generated time, source freshness, input revision/fingerprint, horizon, algorithm version, assumed priority/scope, remaining effort assumptions, dependency waits, result state, and explanation. States include feasible_in_scenario, at_risk, unresolved_estimate, blocked_unknown_release, stale_inputs, and beyond_horizon. Risk is an assessment alongside lifecycle status.

R1 exposes a central scenario and a cautious scenario using explicit effort ranges and capacity assumptions. Label their finish dates as a scenario range, not “80% confidence.” If remaining effort is unknown, show the missing information and only clearly labeled conditional scenarios.

For reproducibility, the central scenario uses a supplied central remaining-effort value when available; for a bounded range without one, use its arithmetic midpoint rounded up to a minute and label that as a scenario assumption, not an expected value. The cautious scenario uses the supplied upper bound. With only a point estimate, both use that point unless the user selects another explicit assumption; do not fabricate a statistically meaningful spread. For a known waiting-release range without a chosen date, initially use its latest boundary in both scenarios and disclose the choice. An unknown release remains unresolved. Both scenarios respect accepted reservations, reserves, dependencies, and the same global competing workload; any additional capacity assumption must be visible.

Statistical probabilities are R2 work. They require enough relevant data, a documented method, evaluation against held-out outcomes, and calibration reporting. Summing unrelated task percentile estimates does not create a valid project percentile. Correlated interruptions, resource contention, unfinished work, and scope change need explicit treatment.

### 12.2 Promises and forecasts

Forecast updates can happen automatically after relevant facts change. They never mutate `promised_finish`, protected availability, or accepted session placement. The commitment view always permits comparison with the original promise and latest explicit revision. A user can see an ambitious commitment and a late forecast at the same time.

Moving a desired target in a what-if scenario is not renegotiating a commitment. Fulfillment requires authoritative or user-confirmed completion with evidence appropriate to the result. A timer finish cannot fulfill a milestone by itself.

### 12.3 Change explanations

Keep a compact structured difference between successive material forecasts:

- Remaining effort changed from X to Y.
- Available intervals changed by named external events or user rules, subject to permission.
- A dependency release changed or became unknown.
- Work was reprioritized or a higher-priority commitment was added.
- Scope/completion criteria changed.
- Actual progress differed from the prior assumption.
- A source became stale or was refreshed.

User-supplied reasons can classify extra work, clarification, rework, waiting, interruption, energy/context, intentional reprioritization, or other. A calculated change is evidence; a causal story is user-reported or inferred and must be labeled. Do not infer that an application window being open means distraction or poor focus.

For multiple changes, explanations must not claim additive day counts unless the engine actually computed independent effects. A one-hour effort increase may move a finish by days because the next feasible session is later. Show that next available interval and the relevant constraints. A later sensitivity analysis can estimate individual contributions, but it must identify order dependence.

### 12.4 Risk notifications and review

Notify on meaningful state transitions or material forecast changes, with a default minimum lead threshold and deduplication. Do not send a new notification on every re-evaluation of the same risk. Present the result, promise, changed outlook, and one useful action: review scope, review schedule, clarify remaining work, or prepare an update.

Shared viewers see only the current allowed projection and any explanation explicitly shared. A private calendar event causing a delay must not leak through a reason such as its title or location. Generic “Available time changed” is preferable when the underlying fact is private.

### 12.5 Baselines and history

Retain the first estimate, accepted planning snapshots, commitment revisions, and meaningful forecast revisions. Do not store a huge complete snapshot every time a second ticks. Debounce transient changes, save material input revisions and derived outputs, and use explicit user checkpoints for significant baseline comparisons.

“Original plan,” “Latest accepted plan,” and “Current forecast” must be distinct selectors. Historical review uses the data and assumptions available at the historical point, rather than recalculating all past forecasts from current task metadata.

## 13. Focus sessions and actual activity

### 13.1 Entry points and modes

Start focus is available from Today, an eligible task/session in Plan, task detail, a supported source app, and quick start. An external event can instead offer Join or Open; a meal can offer Start cooking. These domain actions are not all Pomodoro sessions.

R1 supports Pomodoro, a fixed countdown, an open timer, and untimed focus. Defaults can be 25 minutes work/5 minutes break for Pomodoro, a task/session-derived countdown, and manual ending for open/untimed modes. All are editable. Never imply one timing method is universally optimal.

Untimed focus can record a session's start/end context without asserting all elapsed time was active work. Let the user optionally supply actual effort at wrap-up. Timed mode also requires correction paths because timers can be forgotten.

### 13.2 Focus screen

The focused view centers task objective, timer/mode, a minimal checklist or next step, source link, next fixed commitment/stop time, and pause/end controls. Navigation recedes but remains accessible. A capture-thought action saves an inbox item without exiting. Resume notes are easy to find on re-entry.

The wrap-up panel appears at an interval/session boundary. Do not display a large review form throughout focused work. Choices are Done, More to do, and Blocked, with optional remaining effort, next step, interruption reason, and notes. All optional feedback must genuinely be optional.

If the next appointment is in 15 minutes, offer a shorter session or an explicit overrun choice with consequences. Extending a timer never moves the appointment. Foreground presentation may surface a gentle stopping cue while the underlying schedule remains unchanged.

### 13.3 State machines

Represent session lifecycle separately from segment type and task status:

| Session state | Allowed next states | Segment treatment |
| --- | --- | --- |
| Created | Running, abandoned | No active work before an explicit start. |
| Running | Paused, ended, abandoned | Close the current segment on transition. |
| Paused | Running, ended, abandoned | Paused elapsed time is not active effort. |
| Ended or abandoned | No resumed lifecycle; correction remains available | Further work starts a new session. |

Segment kinds are work, break, interruption, and unclassified. A phase change can close one segment and open another without changing the session's running lifecycle. Task states are independently open, in_progress, blocked, done, cancelled, or archived.

Entering a Pomodoro break closes the work segment and opens a break segment. Ending the break does not automatically start counting work while the user is away; default to awaiting explicit Resume, with any auto-start preference clearly configured. Ending a work interval does not complete the task. A pause is not an activity category unless the user chooses to classify it.

One person can have at most one current exclusive session across devices and workspaces in the same identity domain, including its running, break, and paused phases. A new start offers Continue existing session or explicitly switch, closing the previous current session without erasing its segments. Historical paused segments can coexist. Break state and ownership must not allow a second client to accidentally start duplicate work counters.

### 13.4 Durable timers and multiple devices

The server persists start/stop transitions, segment timestamps, command IDs, and revision/lease ownership. Client intervals render elapsed time; they are not the source of truth. Use monotonic time for in-process display smoothing and persisted instants for reconstruction. Clamp negative display duration and reconcile clock drift against server time.

Refresh, tab suspension, process restart, and phone lock must not reset a session. On reconnect, fetch the current session and revision. Do not infer productive time from heartbeat continuity. If a session was left running far beyond expectations, flag it for correction rather than confidently assigning many hours.

Start, pause, resume, switch, end, and correct are idempotent commands. Serialize competing commands with a revision check and a database-enforced active-session constraint. A second client receives the current session and a conflict response; it cannot win by incrementing a local timer.

For R1, reliable server-backed operation is the baseline. Offline users may create a local draft activity record or stop their local display, clearly marked pending. Reconciliation must not overwrite a newer server session. Full seamless offline multi-device timer ownership is later work; preserve notes and offered corrections even when synchronization fails.

### 13.5 Actuals and corrections

An actual activity record contains subject, optional work item/allocation/session link, start/end when known, reported active seconds, recording method, certainty/completeness, source, and correction history. Manual “about 45 minutes” is valid without exact start/end. Do not fabricate a precise interval to satisfy a schema.

Track uncertainty in actuals: timed_observed, user_reported_approximate, imported_report, inferred_needs_confirmation. Keep inferred values out of authoritative calibration until confirmed or explicitly modeled. External calendar attendance remains unknown unless recorded.

Allow splitting a forgotten timer into work and interruption, shortening it, reassigning activity to another task, and removing erroneous activity. Recompute relevant summaries and learning candidates. Preserve the correction audit within the user's retention policy. CSV/summary exports default to current effective actuals; native backup exports include the selected history needed for restoration and label that choice explicitly.

Remaining effort is independently editable. Default wrap-up asks “Anything left?” rather than subtracting all tracked time from the original estimate. A completed task's final active effort is the sum of effective attributed activity, with missing-data status if coverage is incomplete.

### 13.6 Source-owned completion

If a user finishes a Cadence drafting task, route the configured completion command to the source. When the source is unavailable, the planner may save a private “session finished; source update pending” record, but must not claim the story's authoritative state changed. Retry safely with an idempotency key or show a manual resolution action. Source publication, meal consumption, and task completion remain distinct domain facts.

## 14. Review, learning, and preference calibration

### 14.1 Daily review

Show planned versus recorded activity with unrecorded periods explicitly visible. Ask about only material differences, and allow Skip. Summarize completed results, partial work, changed commitments, and remaining demand. The user can choose which unfinished items to assign forward; a mass rollover offers a capacity preview.

Do not make “all tasks completed” the definition of a successful day. A deliberate scope reduction or early renegotiation can be recorded as a useful decision without counting as on-time delivery of the original promise.

### 14.2 Weekly review

Provide: workload versus usable capacity, estimate/actual comparisons by relevant task family, unplanned work, sources of recorded delay, protected time, goals receiving attention, commitments fulfilled/renegotiated/cancelled, and forecast freshness. Display denominators and unknown coverage. If only two days were tracked, do not characterize the entire week as precisely measured.

Render planned and actual time on the same scale with a usable table alternative. Select at most a few insights initially; each has supporting records, an explanation, and a practical proposed adjustment. Users can inspect, dismiss, mute, or reset recommendations.

### 14.3 R1 learning behavior

Learning begins with descriptive history and recommendations, not an opaque model. Group comparable tasks by explicit type/source/workflow where possible, with user correction. Never assume all coding tasks or all writing tasks have comparable effort.

Recommended initial rules:

| Evidence | Candidate recommendation | Boundary |
| --- | --- | --- |
| Repeated complete records show preparation omitted | Add explicit setup allowance to similar future work | Preview the change; preserve original estimates. |
| Daily accepted demand repeatedly exceeds recorded feasible work with adequate coverage | Review the daily workload cap/reserve | Present missing coverage and competing explanations. |
| Frequent user-reported interruption in a chosen period | Prefer another declared work window | Do not conclude causality from app usage alone. |
| Many tiny sessions with repeated setup | Suggest longer chunks or batching | Retain fixed commitments and user's preferences. |
| Commitments repeatedly need scope clarification | Offer definition-of-done prompts at creation | Do not demand lengthy forms for every task. |

Use a conservative initial evidence threshold, for example at least eight comparable completed records across at least three distinct days before suggesting a recurring numerical adjustment. This threshold is a product guardrail to evaluate, not a scientifically validated constant. Below it, show examples and say there is insufficient history for a stable pattern.

A robust median of actual-to-original-estimated active effort can inform a suggestion if estimates are nonzero and coverage is adequate. Store sample size, window, group definition, excluded records, missing records, outliers, and model/rule version. Do not silently multiply every future estimate. The user can accept a scoped adjustment, after which the planner applies the accepted policy visibly.

### 14.4 Avoid misleading learning

Completed-only samples omit abandoned or still-running work. Show that limitation and include unresolved workload counts in reviews. Do not use completed-task ratios to claim deadline reliability for all work. Treat scope revisions separately; a task doubled in scope is not simply evidence the original effort estimate was wrong.

Preference changes, improved skill, new tooling, seasonality, and changing responsibilities can make old observations less relevant. R1 uses a visible recent-history window and user-controlled reset/exclusion. R2 may add evaluated decay or change-point methods. Maintain historical reports when policies change.

Separate human understanding, accepted planning-policy changes, and forecast algorithm improvements. A user accepting “reserve another 30 minutes on Tuesdays” changes future capacity policy. It does not rewrite past capacity or reclassify earlier outcomes.

### 14.5 Learning controls

Settings shows which data informs insights, grouping rules, accepted adjustments, disabled insight types, and reset/export controls. Provide “This comparison is not relevant,” “Use a different task type,” “Keep my current estimate,” and “Apply to future similar tasks.” Record the decision as feedback. Confidence language must correspond to the available evidence.

AI-generated interpretations must cite internal record IDs for the UI to resolve, distinguish user reports from calculations, and pass a structured validation layer. They cannot change the schedule, share data, or modify estimates without an authorized command.

## 15. Shared Vrooli scheduling contracts

### 15.1 Integration principle

Specialized apps submit scheduling intent and read accepted schedules through one owner. They do not write planner tables directly. The planner maintains the user's global availability and reservations, while each source owns content, eligibility, and its authoritative domain completion state.

Provide three distinct surfaces: a source registration/capability contract, a work/constraint projection contract, and a schedule command/query contract. Share normalized schemas and a lightweight client library if consistent with Vrooli conventions. A shared database is not the integration API.

### 15.2 Source registration

Register a stable source ID, display name/icon, deep-link template or trusted resolver, supported item kinds, readable fields, allowed commands, actor scopes, constraint validation mode, event schema versions, and health/freshness behavior. Discover operational capabilities from current source APIs. A declared capability that fails validation must not render an enabled action.

The registry is separate from user consent. A registered Cadence integration cannot read every user's calendar. Bind every source projection and command to the authorized account/workspace. A source can usually query its own scheduling records and aggregate availability, without private titles from other apps.

### 15.3 Normalized scheduling intent

The following TypeScript is an illustrative wire contract to translate into repository conventions. Branded IDs and runtime schema validation should be added locally.

```ts
/** Source-owned content is referenced; this payload describes scheduling needs. */
interface SchedulingIntentV1 {
  schemaVersion: 1;
  source: {
    appId: string;
    objectType: string;
    objectId: string;
    occurrenceKey?: string;
    sourceRevision: string;
  };
  idempotencyKey: string;
  title: string;
  kind: 'human_work' | 'fixed_event' | 'marker' | 'passive_wait';
  effort?: {
    originalEstimateMinutes?: number | null;
    remaining: {
      centralMinutes: number | null;
      lowMinutes?: number;
      highMinutes?: number;
    };
    basis: 'source' | 'user' | 'accepted_history_policy';
  };
  timing: {
    zone: string;
    preferredLocalDate?: string;
    earliestStart?: string;
    desiredFinish?: string;
    eligibleWindows?: Array<{ start: string; end: string }>;
    fixedInterval?: { start: string; end: string };
    allDay?: { startDate: string; endDateExclusive: string };
  };
  attention: 'exclusive' | 'passive' | 'none';
  splitPolicy?: {
    allowed: boolean;
    minChunkMinutes: number;
    preferredChunkMinutes?: number;
    setupMinutesPerChunk?: number;
    allowShortFinalChunk: boolean;
  };
  constraints: {
    version: string;
    validatedAt: string;
    validUntil?: string;
    prerequisiteRefs: string[];
    allowedActions: string[];
  };
  // A source may request a draft commitment; explicit user acceptance is separate.
  proposedCommitment?: { definitionOfDone: string; finish: string; zone: string };
}
```

Do not trust a payload-supplied workspace or actor. Resolve them from authenticated identity and scoped registration. Validate IDs, ranges, maximum strings, URLs, enum values, interval order, and cross-workspace references. `marker` with no attention cannot reserve human capacity. An unestimated human task must not be coerced to a zero-minute marker.

The effort range describes remaining human work; the optional original estimate is historical context and cannot overwrite an existing original estimate during an upsert. An absent/unknown remaining value is not automatically the original estimate minus timer time. Validate nonnegative finite values, ordered bounds, and a supplied central value within those bounds. A fixed event uses either `fixedInterval` or `allDay`, never both; date strings retain civil-date meaning in the declared zone. The final runtime schema should use discriminated variants to reject incompatible combinations.

### 15.4 Identity and idempotency

The canonical source key includes owner scope, app, object type, object ID, and occurrence identity when relevant. A replayed upsert updates the projection for the same object and revision; it does not create another task. Maintain last accepted source revision or an ordered sequence plus event deduplication. Do not compare opaque provider revisions lexicographically.

The source object and planner work item have separate stable IDs linked by this key. Allocation IDs are owned by the planner and stable across moves. Event transport IDs identify delivery, not business identity. A source deletion creates a tombstone; historical activity can retain a minimal archived label subject to privacy/deletion policy.

### 15.5 Scheduling and source changes

Source apps propose windows and constraints, not arbitrary authoritative clock placements that bypass conflict checks. A direct user placement from Daily/Cadence uses the same preview/validation semantics as the central UI. A source-originated new fact may invalidate accepted work; record the conflict and propose recovery rather than rewriting promises.

Each accepted allocation stores the source/constraint revision against which it was validated. If stronger cross-service validation is needed, use an existing source capability that provides a short-lived validation token or lease. Do not claim distributed atomicity from a read followed by a local write. Without such a capability, acceptance is explicitly valid against the observed revision; later source revisions trigger revalidation and visible conflicts. Domain safety/eligibility still belongs to the source at execution time.

Avoid two services both owning one timestamp. If Cadence stores a derived scheduled-drafting time, it is a projection of the planner allocation. If Cadence owns automated publication time, it publishes that as a domain marker and controls it through its own command. The planner can propose changing it but does not secretly assume ownership.

### 15.6 Deep links and embedded UI

Source detail links open the full planner with item ID, date, source filter, and optional return context. The full planner can show all relevant scheduling conflicts regardless of the originating filter. Links must resolve through current routing/service discovery rather than hardcoded localhost ports.

Provide embeddable source-scoped agenda and capacity summaries if the ecosystem has a widget contract. Share domain schemas and accessible components where sensible. Each app can keep a distinct visual style; a shared source of truth does not require identical page layouts.

### 15.7 Example source behaviors

| Source | Submit | Planner response | Source remains responsible for |
| --- | --- | --- | --- |
| Daily | Meal preparation effort, allowed meal window, batch prerequisites | Accepted preparation allocation or feasible alternatives | Recipe, dietary rules, ingredients, meal suitability, consumption status. |
| Cadence | Draft/review work, content milestone, publication marker | Human work sessions and conflict/forecast information | Draft body/title, channel versions, publication status, audience feedback. |
| Agent work | Expected review/check-in tasks and passive run milestones | Human attention reservations plus nonblocking run markers | Agent execution, actual run results, tools and artifacts. |
| Native entry | User task, appointment, personal promise | Full native planning record | Planner owns native content and status. |

A dinner recipe with 10 minutes active preparation and 30 minutes unattended cooking should not become 40 minutes of exclusive attention by default. A publish-at-09:00 automation may reserve zero human minutes while its final review reserves 20 minutes earlier. The adapter must supply the distinction explicitly.

### 15.8 Integration acceptance

Demonstrate one source-created work item visible in both interfaces, a move reflected in both, a source revision invalidating a proposal, unavailable-source recovery, duplicate event replay, and privacy-preserving availability lookup. Implement Daily/Cadence adapters when those scenarios are operational. When one does not yet exist, deliver its exact contract, contract fixtures, and a reference adapter; do not fabricate a live integration. At least one real Vrooli consumer or equivalent repository integration harness must validate the shared owner before the integration architecture is called proven.

## 16. Read-only external calendars and interoperability

### 16.1 Provider selection and configuration

The user accepted optional read-only connections but did not identify a provider. First reuse the repository's supported connector for the user's actual calendar. If no provider is established, implement a provider-neutral interface and select a practical first provider through local configuration; Google Calendar is a recommended fallback implementation target, not an assertion that the user uses it. Microsoft Graph and Apple/CalDAV-compatible sources are later adapters unless already supported.

Configuration must explain prerequisites, granted read scopes, chosen calendars, busy/free treatment, sync freshness, disconnect, and data removal. Use existing credential storage and auth flows. Never paste tokens into app logs, frontend bundles, exported plans, or this document. Read-only access must be enforced at both requested permissions and adapter implementation; there should be no provider mutation path hidden behind the UI.

### 16.2 Adapter interface

Adapters list calendars, perform initial/incremental reads, normalize event identities and time semantics, report deletions, refresh credentials through the owner, expose health, and optionally handle provider change notifications. The planner owns normalization, projections, and conflict evaluation; adapters preserve provider-specific identifiers and opaque cursor state.

Store a separate cursor per connection/calendar/query definition. Commit cursor advancement only after the corresponding data batch has been durably processed. Retries must be idempotent. A provider notification requests synchronization; it is not itself a complete authoritative event payload.

### 16.3 Google-specific reference behavior

Google's documented incremental flow begins with a full read, retains the final synchronization token, and then fetches changes, including deletions. Pagination must complete before adopting the next sync token. Invalidated tokens can require a full resynchronization. Preserve compatible query parameters across a cursor's lifecycle. See [Google incremental synchronization](https://developers.google.com/workspace/calendar/api/guides/sync).

In this product, a full reset concerns that connection's imported projection only. Build a replacement snapshot and switch it in after successful completion; do not erase native tasks, accepted schedules, history, or unrelated calendars. If the replacement read fails, retain last-known data with a stale marker and mark feasibility uncertain.

Moved recurring instances need a stable original-instance key in addition to their current start. Preserve cancelled occurrences and series membership. See [Google recurring events](https://developers.google.com/workspace/calendar/api/guides/recurringevents). Do not mistake a moved exception for a second appointment.

### 16.4 Microsoft-specific reference behavior

Graph calendar-view delta tracking is scoped to a calendar view and time range, uses opaque next/delta links, and requires handling removed events. Treat query-window changes as a new synchronization scope and follow the provider's current reset behavior. See [Microsoft Graph event delta queries](https://learn.microsoft.com/en-us/graph/delta-query-events). Do not reuse Google's cursor assumptions for Graph.

### 16.5 Normalization and overlap

Preserve origin calendar, provider event and series IDs, occurrence identity, revision/ETag, start/end or all-day dates, timezone, status, busy/free effect, title visibility, and last successful observation. Imported events remain read-only. An Open original action uses a validated provider URL when available.

Recurring and all-day events require explicit tests. Busy all-day treatment is configurable per calendar/event where appropriate, with visible defaults. Tentative appointments can conservatively block by default but must be identified as tentative. Free events are visible without consuming capacity. Deleted/cancelled events stop blocking after confirmed synchronization.

The same meeting on multiple calendars must not consume capacity twice. Interval union handles physical occupancy, but the UI may still need user-confirmed grouping. Provider identity/iCalendar UID can inform candidates; do not merge two separate events merely because title and time match. Never merge confidential copies into a more permissive projection.

### 16.6 Freshness and outage policy

Expose last successful sync, pending work, errors requiring reconnection, and retry timing. Default polling can target several minutes when active, slower when idle, subject to provider limits and current deployment constraints. These are adjustable operational defaults, not assumed contractual SLAs. Use bounded exponential backoff with jitter and honor retry guidance.

Stale imported busy intervals remain last-known constraints until a successful reconciliation or explicit user policy override. Display coverage and uncertainty when suggesting new commitments. A disconnected account is not equivalent to a calendar with no events. Let users explicitly choose whether to retain or remove imported projections on disconnect, explaining the capacity effect.

### 16.7 ICS and exports

Support user-initiated ICS import as a manual fallback, with preview, duplicate detection, errors, and provenance. Distinguish one-time import from a live connection. Validate file size, recurrence expansion bounds, text escaping, and URL fields. Do not fetch arbitrary URLs embedded in imported calendar descriptions.

Provide user-initiated ICS export for selected native events/accepted sessions, with clear boundaries and privacy choices. Export is not two-way sync; another app may retain a copy after local deletion. A live external subscription feed is later scope unless its revocation and cache semantics are designed explicitly. Never export private notes or a whole calendar through a commitment-share route.

## 17. Selective read-only sharing

### 17.1 Product behavior

The owner chooses commitments and previews a viewer page before granting access. Show a separate public-facing title/summary when needed so private source titles do not automatically become shared. Default fields are agreed result, promised boundary, current status, optionally a coarse current forecast, and last updated. Completion criteria, reasons, notes, and history are opt-in fields. Unselected commitments and full availability are absent.

Private workspaces remain the underlying model. Sharing adds a narrow projection and viewer grant; it does not give membership or editing rights. The page should work well on a phone and clearly say whether a date is promised or forecast. Viewers cannot renegotiate by editing. The owner can record an agreement made elsewhere and choose whether to expose it.

### 17.2 Access model

Default to authenticated, recipient-bound viewer access using the repository's existing guest/account mechanism. If guest identity is unavailable, implement the compatible identity flow as a bounded prerequisite or keep the gate explicitly blocked; do not silently downgrade to a public link. Anonymous capability links can be a separate later option with clear semantics.

Grants have stable ID, owner, allowed viewer identity, explicit record set, field mask, expiry, revoked timestamp, and version. Record selection is explicit rather than “all future commitments in this project” by default. New records require deliberate inclusion.

If invitation links are needed, use high-entropy single-use redemption tokens stored hashed, scoped to the intended identity and grant. Creating a grant or copying a link does not send email automatically. Use existing communication workflows only after the user chooses a send action. Viewer access checks apply on every query, export, subscription, and cached response.

### 17.3 Leakage controls

Generate shared DTOs server-side from the allowlist. Never send a full private object and hide fields in CSS. Apply the same rule to error text, nested dependencies, source deep links, thumbnails, activity counts, timeline history, forecast explanations, notifications, and metadata previews.

A private appointment that delays a shared commitment can yield “Available time changed” if reasons are shared, not its title or attendees. Aggregate capacity and other commitment counts remain private unless separately designed. Source links requiring broader permissions must be omitted or replaced with the allowed commitment detail.

Use scoped cache keys, no public caching of private shared pages, referrer controls for token-bearing redemption routes, and audit entries for grant/revoke/view where consistent with repository practice. Revocation takes effect on the next authorized fetch and terminates active subscriptions where supported. No product can retract a viewer's screenshots; avoid promising that it can.

### 17.4 Sharing acceptance

Verify allowed viewer access, forbidden viewer access, revoked/expired grant, direct ID enumeration, changed source title, hidden private delay reason, new commitment exclusion, and attempted writes. Test the actual network payload, not only what the page renders.

## 18. Notifications, jobs, and bounded assistance

### 18.1 Notification types

R1 supports upcoming fixed commitments, latest safe stopping time, optional session boundaries, explicit review reminders, material deadline-risk changes, source/calendar reconnection, and failed queued owner updates. Notification settings allow per-type/channel controls, quiet hours, and lead times. Do not publish a separate alert for each dependent task when one root change explains them.

Use the ecosystem notification owner when available. The planner emits an intent with recipient scope, dedup key, relevance/expiry, deep link, and minimum safe content. Delivery receipts belong to that owner. A retry can resend an intent idempotently without sending duplicate human-facing messages. A private reason must not appear in a notification directed to a shared viewer.

### 18.2 Job catalog

| Job | Trigger | Required behavior |
| --- | --- | --- |
| Provider sync | Poll or validated provider change notice | Scoped lock, paginated idempotent processing, durable cursor, visible health. |
| Occurrence expansion | Horizon request or daily maintenance | Bounded expansion, stable keys, exception preservation. |
| Forecast refresh | Material input revision | Coalescing, snapshot fingerprint, no accepted-schedule mutation. |
| Reminder dispatch | Due intent | Recheck current state, cancellation, quiet hours, recipient scope, dedup. |
| Source command retry | Pending supported owner command | Idempotent retry, bounded policy, visible unresolved failures. |
| Learning analysis | Enough new effective actuals or weekly review | Scoped evidence, versioned recommendation, no silent preference writes. |
| Share maintenance | Expiry/revocation | Revoke access and cached subscriptions consistently. |
| Cleanup | Retention policy | Preserve user history; bound transient job/log data. |

Use existing job infrastructure with leases and observable statuses. A crashed worker must not strand a permanent active lock. Keep attempts, next retry, last error class, and terminal unresolved state. Avoid tight loops on invalid credentials or unsupported source commands.

### 18.3 AI assistance boundary

The app is useful with no model configured. R1 may include optional text parsing and explanation if infrastructure already supports them. A conversational planner can be a later interface to the same bounded commands, not a second scheduling engine.

If using a model, obtain it through the repository's current role/router mechanism rather than hardcoding a provider. Treat imported text, source titles, and model outputs as data. Validate structured intent, ground explanations in current records, and require normal policy checks and user application of changes. Track cost/latency and allow budget limits. A missing model never breaks the core workflow.

## 19. Persistence model and data integrity

### 19.1 Storage approach

Use the repository's established durable database and migration tooling. A relational database with transactions, unique constraints, and explicit indexes is the recommended fallback if the scenario has no established persistence. Do not introduce a graph database, distributed event-sourcing platform, or separate analytical warehouse for R1. Dependencies are a directed graph represented by ordinary edges; forecasts and learning can begin with bounded queries and background jobs.

Store authoritative current state plus revisioned history for meaningful edits. An append-only change log supports explanations and auditing, but rebuilding every entity exclusively from events is not required. Read models are disposable projections, not additional authorities. Soft deletion supports ordinary undo, but explicit permanent deletion must also remove or anonymize associated personal history according to the deletion workflow.

The names below are logical entities. They are not instructions to create one table per row when an existing owner or a simpler normalized representation already supplies the concept.

### 19.2 Logical entity catalog

| Entity | Important fields and relationships | Integrity and lifecycle |
| --- | --- | --- |
| Workspace | ID, owner subject, default timezone, locale, week start, created time | Ownership comes from auth; no client-supplied owner reassignment. |
| Planning profile | Workspace, person/resource identity, availability policy, capacity caps, reserve policy, focus preferences | Version changes invalidate affected proposals; preserve accepted setting history. |
| Availability rule | Local weekday/date rule, local intervals, timezone, effective dates, priority | Date exceptions override normal rules deterministically; reject ambiguous overlapping policy definitions. |
| Availability exception | Civil date or interval, unavailable/extra-available, reason visibility | Protected time is not silently relaxed by scheduling. |
| Source registration | Trusted app identity, adapter version, capabilities, allowed actions, credential reference | Installation is privileged; an ordinary task cannot register executable callbacks. |
| Source object projection | Source identity, source object ID, occurrence key, source revision, normalized kind, freshness, constraints | Unique origin identity; only the verified adapter updates source-owned fields. |
| Native work item | Title, definition of done, status, priority, context, owner, optional initiative/milestone | Simple tasks need no initiative; revision-check edits. |
| Effort revision | Work reference, original/current/remaining kind, minutes or range, author, time, method | Original estimate remains identifiable; unknown is nullable, not zero. |
| Initiative reference | Native project or source-owned reference, title, state, owner | Referenced source content is not copied into a second editable project system. |
| Goal | Outcome, measure type, target, baseline, observed progress, optional target date | Manual outcome updates and work-derived indicators are visibly distinct. |
| Milestone | Completion criterion, desired date, linked work or source condition, state | Date-only checkpoints carry timezone and an explicit deadline boundary policy. |
| Commitment | Promised result, beneficiary label if supplied, scope, current accepted revision, lifecycle | Risk is a separate derived value; do not infer recipient identity from a free-text label. |
| Commitment revision | Original/revised boundary, scope/definition of done, accepted by, reason, supersedes | Preserve original and renegotiated promises; record external acknowledgment only if actually supplied. |
| Dependency edge | Predecessor, successor, kind, lag/wait policy, owner, revision | Enforce same authorized scope, no self-edge, no cycle; unknown release time remains unknown. |
| Native event | Title, timed interval or all-day dates, timezone, busy status, recurrence reference | Exactly one temporal representation per event. |
| Routine definition | Fixed recurrence or frequency target, effort template, timezone, policy, revision | Fixed recurrence and flexible frequency are separate variants. |
| Routine occurrence | Routine ID, stable original occurrence key, current date/state, exception | Moving an occurrence does not generate a new identity. |
| Allocation | Work/source reference, kind, date or interval, active effort, occupied minutes, lock, accepted revision | Timed/date assignments conserve demand; canceled allocation is not a completed task. |
| Proposal | Scope, base revisions/fingerprint, assumptions, candidate operations, unresolved work, expiry | Draft/applied/rejected/expired/superseded; immutable after apply. |
| Proposal application | Proposal ID, operation results, actor, idempotency key, resulting revision, undo relation | One application per proposal; atomic local write set. |
| Forecast snapshot | Subject/scope, input fingerprint, model version, generated time, scenarios, limitations | Immutable derived snapshot; no accepted allocations created. |
| Focus session | Person identity, work reference, state, timestamps, mode, revision, device metadata | At most one running exclusive session per person; no unbounded device metadata. |
| Session segment | Session ID, start/end, work/break/interruption/unclassified, correction provenance | No negative intervals; exclusive work segments cannot overlap for the same person. |
| Manual actual | Work reference, duration or interval, date attribution, precision, reported by | Duration-only entries do not invent clock times; overlapping known intervals require resolution. |
| Change record | Subject, before/after references, observed changes, optional reported reason, actor/source | Separate evidence from interpretation; redact source payloads. |
| Review | Date/week, confirmed actual corrections, carry-forward choices, optional reflection | Completing a review is not required for using the next day. |
| Learning insight | Rule/model version, eligible sample, evidence IDs, recommendation, status | Inspect, accept, dismiss, expire; accepting creates a separate setting command. |
| Provider connection | Provider/account opaque ID, credential reference, status, scopes | Store secrets in existing credential infrastructure, not UI records or exports. |
| Provider calendar | Connection, remote calendar ID, selection, busy policy, cursor/query scope, freshness | Cursor belongs to one documented query scope. |
| Imported event | Provider/calendar identity, remote event or occurrence ID, remote revision, normalized time, tombstone | Identity is not title/start matching; deletion behavior follows provider semantics. |
| Share grant | Owner, verified viewer or redemption identity, explicit record set, field mask, expiry, revision | Revocable; no recursive grants through linked private objects. |
| Notification intent | Recipient subject, type, subject/version, dedupe key, delivery state | Current permission and preference checks before delivery. |
| Integration receipt | Source event identity, status, result reference, retry metadata | Replays return prior outcome or safely resume; they do not repeat effects. |
| Outbox record | Domain event ID, schema version, transaction ID, delivery attempts | Created with local mutation; retained until safely delivered and policy permits cleanup. |
| Import/export job | Actor, format/version, progress, artifact reference, expiry, validation report | No credentials or reusable share tokens in exports; download authorization remains enforced. |

Use a typed subject reference for native/source work where necessary. Avoid unconstrained strings such as `entity_type + entity_id` without validation: the command layer must verify that the referenced entity exists, belongs to the authorized scope, and is an allowed kind. Native foreign keys should remain real database constraints whenever practical.

### 19.3 Temporal and numeric types

The implementation language should make these differences explicit, even if the transport ultimately uses strings and integers:

```ts
// Illustrative transport/domain types; runtime validation is still required.
type Instant = string;       // RFC 3339 instant with offset; normalized for storage
type CivilDate = string;     // YYYY-MM-DD, never parsed as a UTC midnight instant
type LocalTime = string;     // HH:mm[:ss], interpreted with an explicit time zone
type TimeZoneId = string;    // Supported IANA time-zone identifier
type Revision = string;     // Opaque equality token, not lexically sortable
type Minutes = number;       // Finite nonnegative integer at the allocation boundary

type Effort =
  | { kind: "unknown" }
  | { kind: "point"; minutes: Minutes }
  | { kind: "range"; lowMinutes: Minutes; highMinutes: Minutes };

type TemporalExtent =
  | { kind: "timed"; start: Instant; end: Instant; displayZone: TimeZoneId }
  | { kind: "all_day"; startDate: CivilDate; endDateExclusive: CivilDate;
      timeZone: TimeZoneId };

type AllocationPlacement =
  | { kind: "timed"; start: Instant; end: Instant }
  | { kind: "date"; date: CivilDate; timeZone: TimeZoneId };
```

Validate `lowMinutes <= highMinutes`, positive duration for occupied intervals, supported timezones, and exclusive end dates. A zero-effort milestone can be valid; a zero-length busy event is not an occupied interval. Preserve seconds or higher precision for actual session timestamps and round only for display or a documented scheduling grid. Do not accumulate rounding errors by rounding every timer tick or segment separately.

### 19.4 Revisions, invalidation, and indexing

Maintain entity revisions and a workspace planning revision for mutations that affect planning. Derive a proposal fingerprint from relevant entity/policy revisions, source constraint revisions, provider freshness state, algorithm version, and scope. It need not be a hash of every workspace field: a cosmetic label change need not invalidate a placement, but a source constraint, date, estimate, dependency, availability, or busy interval change must.

Within the database transaction, lock or compare the planning revision before applying operations. Record the resulting revision and enqueue invalidation events. A provider update or source event that arrives during apply must either serialize before validation or afterward trigger a visible new conflict/forecast; do not claim external state was frozen globally.

Recommended indexes include workspace plus interval start/end for allocations/events; workspace plus date for dated demand; source identity tuple; provider/calendar/remote occurrence tuple; predecessor/successor dependency lookups; person plus session state; pending job schedule; and recipient plus grant state. Use the database's supported range/index strategy only after inspecting actual query patterns. Bound recurrence expansion by requested range and cap query sizes.

Where PostgreSQL is the established database, a partial unique constraint is a useful final protection for one current exclusive focus session, for example on person ID where state is running or paused and attention is exclusive. Group that predicate explicitly as `(state IN ('running', 'paused')) AND attention = 'exclusive'`. Use equivalent transactional protection elsewhere. A preflight `SELECT` alone is insufficient. Break segments remain within the current session without being labeled work. A new start offers to close or replace the current session, keeping only one current session per person.

### 19.5 Cross-workspace and resource scope

R1 presents one active personal planning workspace by default. Do not silently offer multiple independent workspaces that each assume the same person is fully available. If the host already supports several workspaces, either explicitly aggregate authorized busy time into a personal resource view or show that other-workspace availability is not included. Private titles and details must not cross workspace boundaries merely to reserve time.

The single-running-session constraint is tied to the authenticated person's resource identity, not a browser tab or task. A conflict with a session in another workspace may reveal only that a session is already active, plus a permitted navigation action. Forecast resource aggregation must use the same declared planning scope as the capacity display.

## 20. Application commands, queries, and API behavior

### 20.1 Transport conventions

Follow existing Vrooli routing and error conventions. The routes below describe a complete logical surface; they are illustrative names under an appropriately versioned namespace, not assertions that these paths already exist. Keep one application-command layer behind HTTP, CLI, internal tools, and source adapters.

Every mutation obtains actor, workspace, and capabilities from authenticated context; accepts an idempotency key when effects could be retried; checks expected revisions where overwrites are possible; validates domain constraints; commits locally; and returns the resulting revision plus a trace/request ID. Return canonical stored records, not only `success: true` when the caller needs to reconcile its state.

Use cursor pagination with stable ordering for growing collections. Queries require bounded date ranges. Support conditional requests or equivalent revision-aware caching where the stack already provides it. Server authorization always remains required, including for cached and subscription results.

### 20.2 Logical endpoint inventory

| Area | Commands and queries | Important behavior |
| --- | --- | --- |
| Setup/profile | Get/update profile; list/update availability rules and exceptions | Preview material capacity effects before broad policy changes; accepted settings invalidate proposals. |
| Native work | Create/read/update/archive work; revise effort; update remaining; complete/reopen | Source-owned status routes through its owner; completion is explicit. |
| Events/routines | Create/update/cancel event; create/update routine; move/skip occurrence | Require instance/series scope for recurring edits. |
| Goals/milestones | Create/update goal; record outcome progress; link work; complete milestone | Do not derive achievement solely from elapsed time. |
| Commitments | Create draft; accept promise; propose/accept revision; withdraw; list history | Forecast updates cannot invoke revision acceptance. |
| Dependencies | Add/update/remove edge; inspect blockers | Cycle validation in a consistent graph snapshot. |
| Schedule | Query range; allocate to date; allocate timed session; split/move/lock/release allocation | Preserve demand and constraints; a move is not delete-plus-unrelated-create. |
| Capacity | Query date/range and scope; explain deductions and demand | Return known/unknown quantities and fragmentation, not one ambiguous percentage. |
| Proposals | Generate; get; compare; reject; apply; undo an application | Generation may be asynchronous; apply must revalidate. |
| Forecasts | Get latest; request recompute; get snapshot/history/explanation | Freshness and model/scenario labels accompany results. |
| Focus | Start; pause; resume; change phase; finish; get current session | One authoritative session; expected session revision on transitions. |
| Actuals | Add manual duration/interval; correct; classify; remove | Provenance and coverage survive corrections; overlap checks. |
| Reviews | Get daily/weekly summary; save reflection and explicit follow-up choices | No mandatory completion ceremony or automatic rollover. |
| Learning | List/inspect insight; accept scoped adjustment; dismiss; reset derived profile | Evidence visibility; acceptance is an auditable setting change. |
| Source apps | Register capability; ingest intent/change; query authorized schedule projection | Registration is privileged; app scope cannot become whole-workspace access by default. |
| Provider calendars | Initiate connection; handle callback; list/select calendars; sync; disconnect | No provider write endpoints in R1; secrets never returned to UI. |
| Sharing | Preview projection; create/redeem/revoke grant; list owner grants; view granted records | Separate viewer DTOs and authorization path. |
| Notifications | Get preferences; update preferences; list/acknowledge in-app notices | Delivery and read state are separate. |
| Data/operations | Export; validate import; apply import; request deletion; get authorized job status | Native restore/import requires preview and explicit apply. |
| Health | Process liveness; dependency readiness; authorized integration status | Public health must not expose account names, calendar titles, or secrets. |

### 20.3 Proposal response contract

Return inspectable operations and unresolved demand, not just a generated calendar. This abbreviated fixture uses illustrative identifiers:

```json
{
  "proposalId": "proposal_example_01",
  "status": "ready",
  "basePlanningRevision": "rev_example_17",
  "inputFingerprint": "fp_example_abc",
  "algorithmVersion": "deterministic-v1",
  "scope": {
    "startDate": "2026-09-21",
    "endDateExclusive": "2026-09-22",
    "timeZone": "America/New_York"
  },
  "assumptions": [
    { "code": "RESERVE_POLICY", "reserveMinutes": 60 },
    { "code": "EFFORT_ESTIMATE", "workItemId": "work_example_a",
      "activeMinutes": 50, "origin": "user" }
  ],
  "operations": [
    {
      "operationId": "op_example_1",
      "kind": "replace_date_allocation_with_session",
      "allocationId": "allocation_example_a",
      "expectedRevision": "allocation_rev_3",
      "dateEffortRemovedMinutes": 50,
      "session": {
        "start": "2026-09-21T09:00:00-04:00",
        "end": "2026-09-21T09:55:00-04:00",
        "activeEffortMinutes": 50,
        "breakMinutes": 5
      },
      "reasonCodes": ["FITS_BEFORE_FIXED_EVENT", "ADVANCES_COMMITMENT"]
    }
  ],
  "unresolved": [
    { "workItemId": "work_example_b", "code": "UNKNOWN_REMAINING_EFFORT" }
  ],
  "limitations": ["UNKNOWN_EFFORT_EXCLUDED_FROM_NUMERIC_DEMAND"],
  "expiresAt": "2026-09-21T12:00:00Z"
}
```

The 50 minutes are removed from the date allocation before adding the timed session; occupied time becomes 55 minutes because of the planned break. Expiry is a usability and storage bound, not a substitute for revision checks. A proposal can be stale one second after generation.

### 20.4 Apply contract and atomicity

An apply request contains the proposal ID, expected base fingerprint/revision, selected operation IDs if partial apply is supported, and an idempotency key. The server loads the stored proposal rather than trusting the client to submit altered operations. Recompute feasibility for the selected subset because operation dependencies may change. Reject unsupported subsets with a clear explanation.

On success, return the application ID, resulting planning revision, applied operation IDs, canonical changed records, resulting capacity summary, and undo availability. On a stale proposal, return a structured conflict with safe reason codes and an action to regenerate. Do not partially apply unrelated writes and then report a stale failure. Outbound source commands, if required, have their own explicit workflow and receipts; a local database transaction cannot atomically commit another service's database.

Recommended R1 placement operations only mutate planner-owned allocations. Source-domain changes appear as separate, clearly labeled requests to the source app. If a proposed schedule depends on a source-domain alternative, obtain source acceptance first, then regenerate or revalidate the placement.

### 20.5 Error model and recovery

| Error code | Typical condition | UI or caller response |
| --- | --- | --- |
| `VALIDATION_FAILED` | Invalid date, effort range, unsupported interval | Show field error; preserve entered values. |
| `FORBIDDEN` / `NOT_FOUND` | No authorization or record unavailable | Follow host policy without leaking existence across tenants. |
| `REVISION_CONFLICT` | Concurrent edit | Fetch current record, show relevant difference, offer retry. |
| `PROPOSAL_STALE` | Material input changed | Keep old proposal inspectable; generate an updated proposal. |
| `CONSTRAINT_VIOLATION` | Protected time, source restriction, dependency | Identify permitted explanation and available alternatives. |
| `SESSION_ALREADY_ACTIVE` | Concurrent start or another current session | Resume, end, or switch through a revision-checked command. |
| `SOURCE_UNAVAILABLE` | Source-owned action cannot be confirmed | Show pending/failed source action; do not display false completion. |
| `SOURCE_CONSTRAINT_STALE` | Source validation too old for action | Refresh constraints or leave action unapplied. |
| `PROVIDER_REAUTH_REQUIRED` | Credentials expired or revoked | Preserve last-known busy facts and show reconnect path. |
| `INSUFFICIENT_INPUT` | Unknown effort/release time prevents a forecast | Request the specific missing value or show partial result. |
| `LIMIT_EXCEEDED` | Recurrence, query, import, or proposal bound | Explain the bound and offer smaller scope. |
| `RATE_LIMITED` / `TEMPORARILY_UNAVAILABLE` | Infrastructure/provider delay | Respect retry metadata; keep current accepted state visible. |

Avoid exposing raw provider errors, SQL details, private source payloads, or hidden record IDs in user-facing messages. Log a safe diagnostic code and request ID for investigation.

### 20.6 Idempotency and events

Scope idempotency by authenticated caller, command type, and workspace. Store a request fingerprint with the key; reuse with a different payload is an error. Recommended interactive command retention is seven days, with durable entity uniqueness protecting older replays. Import batches and source event receipts use their own stable identities and retention, not only an expiring request cache.

Domain event names should describe completed facts, for example `AllocationMoved`, `RemainingEffortRevised`, `CommitmentRevisionAccepted`, `SessionFinished`, `SourceConstraintChanged`, and `ProviderCalendarRefreshed`. Include schema version, event ID, workspace, subject, actor category, correlation/causation IDs, and occurred time. Publish minimal references and safe change metadata; consumers fetch authorized details when needed.

Receipt handling must tolerate duplicates and out-of-order events. A revision token can establish equality but not necessarily order; when ordering cannot be established, fetch the current authoritative object. Never compare opaque revision strings numerically or lexically to guess which is newest.

## 21. CLI, tools, and native program integration

### 21.1 One policy surface

The local agent should inspect existing Vrooli conventions for scenario CLI commands, tool registration, and program discovery. Supply domain capabilities through those mechanisms. UI, CLI, and program actions must share authorization, idempotency, validation, and audit behavior. A tool must not gain unrestricted database access merely because it runs locally.

Recommended logical verbs are `schedule.query`, `capacity.explain`, `work.capture`, `work.revise_remaining`, `proposal.create`, `proposal.inspect`, `proposal.apply`, `forecast.inspect`, `focus.start`, `focus.pause`, `focus.finish`, `actuals.record`, `review.summarize`, and `commitment.inspect`. These are naming suggestions to map into the actual runtime, not preexisting command names.

Provide machine-readable JSON output and concise human-readable output. Bulk operations accept structured input through files/stdin or the host's structured arguments. Do not parse shell-like natural language to construct execution commands. Return documented exit/status codes and safe recovery hints.

### 21.2 Read versus write capabilities

| Capability class | Examples | Recommended default |
| --- | --- | --- |
| Read planning context | Schedule, capacity, goals, permitted source summaries | Scoped to the caller and active workspace. |
| Compute without applying | Forecast, proposal, hypothetical comparison | Allowed when input access exists; clearly labeled draft. |
| Capture/update owned facts | Create native task, record manual actual, revise remaining | Requires write capability and attributable actor. |
| Apply schedule changes | Apply stored proposal, move allocation | Explicit command under user-granted capability; no automatic invocation by forecast jobs. |
| Sensitive scope changes | Create share, connect account, accept commitment revision, delete data | Explicit user intent and host policy; never inferred from imported text. |

“Manual apply” means background scheduling does not change the accepted plan without a user-authorized application command. The UI button and an explicitly authorized CLI/tool invocation can use the same command. It does not mean every low-risk API write needs a new conversational confirmation.

### 21.3 Bounded reusable programs

Useful optional wrappers, if the existing runtime supports them, include morning-plan preparation, an end-of-session remaining-effort prompt, weekly-review preparation, and a commitment-risk digest. Each wrapper declares inputs, permissible reads/writes, output schema, retries, and cancellation behavior. A morning preparation program can create a draft; it cannot silently apply it. A digest may produce an in-app summary under user preferences; it cannot message another person by default.

Automated work from other scenarios should represent human review/setup effort separately from machine execution. The planner may show agent progress as context and revise availability when the owner supplies a new release estimate. It should not become the agent's job runner or fabricate completion when a scheduled marker passes.

### 21.4 Developer-facing documentation and contract evidence

Publish the actual discovered route/CLI/tool mapping, request/response schemas, permission table, error codes, event versions, and realistic examples alongside the scenario. Include a minimal source-adapter example and a contract test harness with duplicate delivery, stale constraint, cancellation, and unauthorized cross-workspace cases. Fixtures must be clearly synthetic.

If Daily or Cadence is absent or lacks an API, implement the shared contract and a fixture adapter, document the missing owner capability, and keep that live integration gate open. Do not create an unrelated substitute source application merely to mark the gate complete.

## 22. Application architecture and frontend implementation

### 22.1 Recommended module boundaries

Start with a modular application using the host's established process/service layout. Keep a pure planning kernel and adapters around it; do not turn every domain noun into a microservice.

| Module | Responsibility | Must not do |
| --- | --- | --- |
| Identity/policy adapter | Resolve subject/workspace/capabilities | Trust user-supplied tenant identifiers as authorization. |
| Work and commitment domain | Native work, effort, dependencies, promise revisions | Mutate source-domain content directly. |
| Temporal normalization | Timezones, intervals, recurrence/instance identity | Treat civil dates as arbitrary UTC instants. |
| Capacity kernel | Interval union/difference, budgets, demand accounting | Count hypothetical forecast placements as reservations. |
| Planning kernel | Candidate placement and explanation | Write database records or call arbitrary tools. |
| Forecast service | Snapshot workloads, compute scenarios, classify risk | Replace original promise fields. |
| Command service | Validation, concurrency, transactions, history/outbox | Bypass constraints for CLI or assistant callers. |
| Focus/actuals service | Durable sessions, correction, coverage | Infer task completion from elapsed time. |
| Review/learning service | Summaries and evidence-linked recommendations | Treat missing data as tracked zero effort. |
| Source/provider adapters | Normalize authoritative external state | Own a second independent accepted schedule. |
| Sharing projection | Enforce explicit field/record grants | Serialize private domain objects and hide fields in CSS. |
| Read models | Efficient authorized views | Become write authorities. |
| Jobs/notifications | Reliable background work and delivery | Silently apply proposals or send to unselected recipients. |

Pass an explicit clock and timezone service into date-sensitive logic. Seed deterministic tie-breaks and inject provider/source interfaces for tests. A solver function should accept a normalized immutable snapshot and return a candidate result with reasons. Keep persistence, auth, HTTP, and visual rendering outside that function.

### 22.2 Frontend composition

Use one application shell, accessible navigation, appearance provider, authenticated data layer, and route-level boundaries. Build reusable domain components rather than separate versions of each task card per screen:

- `NextActionCard`: criterion, available window, estimate, start/resume, open source, alternative action.
- `CapacitySummary`: explicit scope, usable time, planned time, reserve, unknown demand, details.
- `TimeHorizon`: proportional intervals, now marker, density controls, accessible list equivalent.
- `WorkItemRow/Card`: status, source, remaining effort, target/commitment badges, allocation summary.
- `CommitmentStatus`: original/current promise, forecast, risk, definition of done, history link.
- `ProposalDiff`: move/add/remove operations, rationale, conflicts, selected subset validation, apply.
- `FocusPanel`: current intention, authoritative phase/time, pause/finish, interruption capture.
- `HistoryDrawer`: observed changes, user explanations, provenance, revision comparisons.
- `InsightCard`: evidence size/coverage, suggested setting change, preview, accept/dismiss.
- `SharePreview`: exact viewer projection and selected field mask.

Names are illustrative. Adapt to the repository's component and naming conventions. Separate presentation from mutation hooks so the same domain rules are exercised by a compact source-app widget and the full planner.

### 22.3 Client state and invalidation

The server is authoritative for accepted schedules, revisions, session state, grants, and source status. Local state holds incomplete forms, selected view/date, unsaved scenario adjustments, and drag previews. A drag preview can be immediate, but acceptance waits for validation; show conflict without losing the user's intended target.

Use optimistic updates only for straightforward reversible fields with revision-aware rollback. For proposal apply, source completion, share creation, and timer start, show pending state until canonical success. If the request times out, query by idempotency/application ID before asking the user to repeat it.

Subscribe to the host's established push mechanism if available. Otherwise use bounded active-view polling and conditional queries; do not add a second realtime stack solely for a ticking clock. Subscriptions are scoped and authorization is rechecked when grants/session access change. Timer digits render from a confirmed timestamp while periodic reconciliation detects authoritative transitions.

### 22.4 Route loading and error handling

Load the shell and cached authorized context promptly, then the selected day/range. Fetch forecast and learning detail lazily when the view needs it. Display “updating” separately from “no data.” Preserve the accepted schedule if an auxiliary forecast request fails. A provider outage must not blank the whole application.

Use independent error boundaries for the scene, timeline, source summaries, and optional insights. If artwork fails, preserve readable themed surfaces. If a visualization cannot render, keep its textual values and actions. Empty onboarding, a truly empty date, a failed query, a stale cache, and a filtered-to-zero result need distinct copy.

### 22.5 Search and navigation

R1 search covers authorized native titles, source projection titles, commitments, goals, and milestones. Respect field visibility and source freshness. Keep the initial implementation simple using the established database/search service; semantic search is not required. Search results deep-link into the relevant detail and show source identity. The full planner button from a source app preserves filters in the route, but capacity remains aware of all authorized busy time in the chosen resource scope.

Global quick capture and keyboard navigation must work without a mouse. Provide conventional documented shortcuts with collision handling; do not override browser or assistive-technology keys. Restore focus after dialogs and announce meaningful asynchronous completion or conflict states.

## 23. Reliability, privacy, portability, and operations

### 23.1 Transaction boundaries and concurrency

Define transaction boundaries around commands, not screen renders. A split/move command writes its allocation changes, effort conservation checks, revision, history, and outbox records together. A session transition writes the session and segment boundary together. A commitment revision changes the accepted revision pointer and history together. A failed write must not leave half of these effects committed.

Use database-enforced uniqueness and a documented isolation/locking strategy for concurrent session starts, proposal applications, imports, and dependency edits. Keep transactions short; do not hold locks while waiting for provider APIs or a model response. Fetch external data first, then validate its recorded revision/freshness during the local command. A later source change triggers reconciliation rather than pretending there was a distributed transaction.

Outbox dispatch is at least once unless the actual infrastructure proves otherwise. Consumers deduplicate by stable event identity. Record handler completion after effects are safely committed. A process restart between commit and delivery must lead to safe replay, not a lost notification or duplicate actual entry.

### 23.2 Offline and degraded operation

R1 is an online authoritative application with deliberate degraded behavior, not a full offline multi-master system. Cache the last authorized read view only under the host's security model. Mark it stale and preserve its timestamp. Allow local text drafts and hypothetical edits while disconnected; require revalidation before saving accepted schedule changes.

A running timer can continue rendering elapsed time from its last confirmed session timestamp. Offline pause/finish intentions may be retained as pending local corrections with their device timestamps. On reconnect, compare the server revision; if another device changed the session, show a reconciliation choice instead of applying an obsolete transition. Do not start a second authoritative session offline.

Clear local private caches on logout/account change and revoke access to expired shared views. Do not cache shared private records in a public service worker cache. Document what survives browser storage clearing and what is only an unsaved local draft.

### 23.3 Security boundaries

Reuse established identity, session, secret storage, encryption in transit, and deployment controls. Apply authorization to every record lookup, nested relationship, export, job, and subscription. For browser sessions, follow the host's CSRF/origin protections. Provider connection callbacks must bind to the initiating authenticated session and expected connection flow; request only the scopes needed for read-only functionality.

Validate URLs and source registrations. Deep links may use trusted registered application origins; do not allow arbitrary executable URLs. Calendar descriptions and task titles are untrusted text, rendered safely. HTML in imported descriptions must be sanitized or displayed as plain text. An ICS attachment cannot instruct the app to fetch arbitrary URLs, run code, or change access policies.

Keep bearer secrets, credentials, tokenized share URLs, and private calendar text out of general logs and analytics. Redact query parameters where tokens could appear. Apply bounded import sizes, recurrence expansion limits, request limits, and source payload limits. Document ownership of security-sensitive configuration without inventing a new compliance program.

### 23.4 Data export and restore

R1 provides a native versioned JSON export of user-owned planning data and explicit history, plus useful CSV summaries for actuals/estimates and ICS export of supported calendar records. A forecast/commitment report can be printed through a readable browser view; a custom PDF engine is not required.

The native export includes format version, export time, timezone/policy metadata, stable IDs, provenance, native records, allocations, meaningful revisions, actuals, and references needed to interpret them. Exclude credentials, access/session tokens, private infrastructure IDs where unnecessary, and reusable share secrets. Include source projections only under an explicit documented export policy and retain their owner identity. External data is a snapshot, not a transfer of provider authority.

Native import first validates the complete package, produces counts/conflicts/warnings, and offers a preview. Recommended default is import into a new private workspace. Merging into an existing workspace requires explicit identity mapping and conflict rules. Reimporting the same package must not duplicate objects. Preserve unsupported records in the report; never silently flatten commitment history or recurrences into unrelated events.

Restored connections require reconnection. Restored sharing records are inactive until the owner deliberately grants access again. Do not resume an exported running session as if uninterrupted work occurred during the gap. Restore it as needing review, preserving the recorded timestamps and uncertainty.

### 23.5 Deletion, retention, and history

Provide ordinary archive/delete with undo where useful, alongside an explicit permanent-deletion path for user data. Dependency, history, forecast, and learning references need a consistent response when a record is permanently removed: remove sensitive payloads and recompute affected derived data, retaining only a minimal nonidentifying structural tombstone if necessary and disclosed.

Recommended defaults: retain user-authored planning history until the user deletes it; keep unapplied proposal payloads for 30 days unless explicitly saved; keep routine diagnostic logs for 30 days; keep delivered notification/job diagnostics for 30 days; expire downloadable export artifacts after 24 hours; and retain interactive idempotency results for seven days. These are engineering defaults to align with the host, not legal retention claims. Applied proposal changes remain in plan history even after transient preview payload cleanup.

Source receipt cleanup cannot allow an old replay to resurrect deleted work. Keep durable origin uniqueness/tombstones or a source replay checkpoint contract sufficient to prevent resurrection. Backup retention follows the host's documented policy; deletion documentation must distinguish removal from live views from eventual backup expiry.

### 23.6 Performance targets and bounded workloads

Treat these as initial acceptance targets on a recorded development/deployment environment, not published performance claims:

| Operation | Initial target | Measurement conditions |
| --- | --- | --- |
| Today/day API | p95 below 500 ms | Warm service, realistic personal dataset, no external call on the request path. |
| Native command acknowledgment | p95 below 500 ms | Database write and outbox commit, excluding external delivery. |
| First useful Today content | Within 2 seconds | Document browser/device/network; scene artwork must not block content. |
| Interactive proposal | Within 2 seconds for normal scope | Up to 200 candidate work items and 28 days; otherwise asynchronous with progress/cancel. |
| Forecast refresh after local edit | Visible within 5 seconds under normal load | Coalesced background job; old forecast labeled updating meanwhile. |
| Focus transition UI | Immediate pending feedback; prompt canonical response | Measure server latency; never hide failure behind an optimistic timer. |
| Large import/export | Asynchronous, cancelable, bounded memory | Stream or batch records; report progress and partial validation errors. |

Use at least one reference dataset with 1,000 work items, 10,000 historical event/occurrence records, and 10,000 actual/change records to expose unbounded queries. These are test sizes, not product limits. Query only the requested horizon, paginate history, and virtualize dense lists where measurements warrant it. Record any target adjustment with the actual bottleneck and user impact.

Do not run a full long-horizon solver on every keystroke, timer tick, hover, or theme transition. Debounce and coalesce material changes; cancel obsolete work by revision. Use deterministic fallback explanations when a bounded planner stops, not a frozen UI.

### 23.7 Observability and support

Track command errors by safe code, conflict rates, proposal computation time, stale-proposal rate, provider freshness, sync failures, pending outbox age, job retries, forecast age, and unexpected timer reconciliation. Keep personal content out of metric labels. Correlate a user-visible request ID with the internal command/job trail.

Provide an owner-facing integration health panel with last successful refresh, current problem, next retry or reconnect action, and affected data scope. Provide a developer diagnostic report with versions, configuration categories, and redacted IDs. Collect optional usage telemetry only through the host's consent model; it is not a dependency of personal learning, which uses the user's own authorized records.

### 23.8 Deployment and migration runbook

The local agent must document the actual start, stop, health, migration, backup, restore, and test commands after repository discovery. Use the established deployment system. Configuration should cover public/base URLs, auth integration, database, job infrastructure, provider credential references, timezone data/runtime support, limits, and optional model routing. Do not hardcode ports, hostnames, tokens, or user identities from this plan.

Before cutover, verify backup restoration, migration dry-run reconciliation, old/new source routing, representative recurring events, and rollback accounting for new writes. After cutover, verify one schedule owner, source event delivery, provider freshness, focus resume, and share authorization. An unavailable optional provider can leave manual use operational, but its release gate remains explicitly incomplete.

## 24. Worked examples and canonical fixtures

These synthetic examples establish semantics. They are not claims about the user's real routine. Implement the important ones as domain or integration fixtures and reuse the same values in screenshots and demos so the displayed math stays credible.

### 24.1 F01 — daily capacity with overlapping deductions

Availability is 09:00–17:00: 480 minutes. Meeting A occupies 10:00–11:00 and meeting B occupies 10:30–11:30. Their union is 90 minutes, not 120. Lunch is protected 12:00–13:00, and two explicitly fixed transitions occupy 15 minutes each without overlapping another exclusion. Usable intervals total `480 - 90 - 60 - 30 = 300` minutes. There is no additional daily cap. A 60-minute reserve leaves 240 minutes allocatable.

Accepted timed occupancy is 120 minutes, and date-assigned occupancy is 90 minutes. Planned occupancy is 210; budget spare is 30; unallocated room is 90, of which 60 is reserve. The UI must not call all 90 minutes safely available for extra work under the current policy. If fixed transitions overlap a meeting instead, subtract only the union and recompute.

### 24.2 F02 — converting dated demand into sessions

A task has 90 minutes assigned to Tuesday. Converting 30 minutes into a timed session leaves 60 untimed minutes on Tuesday. Total planned active effort remains 90. If that session also reserves a five-minute setup, occupancy becomes 95 minutes while active effort stays 90. Moving the 30-minute session to Wednesday leaves Tuesday with 60 minutes and Wednesday with 35 minutes occupancy. No clone task is created.

### 24.3 F03 — fragmentation and hard boundaries

A day contains six separate 15-minute usable gaps and 90 minutes total budget. A 90-minute unsplittable task cannot be placed. A task allowing 15-minute chunks may fit, provided per-session setup and other constraints permit it. A 25-minute minimum chunk cannot fit. The explanation refers to longest usable interval and chunk policy, not only total hours.

### 24.4 F04 — timer time, actual work, and remaining effort

A planned task originally had a 40-minute estimate. A focus session spans 45 wall-clock minutes: 30 active work and 15 paused. At wrap-up the user reports 20 minutes still needed. Show 30 recorded work minutes, 20 reported remaining, and a revised total-effort expectation of 50 if that summary is displayed. Keep the original estimate at 40. Do not report 45 active minutes, mark the task complete, or assume only 10 remain.

The original estimate ratio is not a completed-task measurement yet. The task may continue changing. Only when completion and adequate actual coverage are established can it enter the appropriate completed-task calibration sample.

### 24.5 F05 — additional work pushes a finish through a weekend

After the remaining Thursday workload, Friday has one usable 60-minute interval before a Friday 17:00 commitment. The task has 60 minutes remaining and no intervening dependencies. Its central scenario finishes Friday. The user revises remaining work to 120 minutes; the next eligible 60-minute interval after Friday is Monday morning. The forecast moves to Monday. The original Friday promise remains visible and becomes at risk/late in the current scenario.

The explanation is “Remaining work increased by 60 minutes; the next eligible interval is Monday,” with the relevant availability visible to the owner. It must not claim the person worked three days too slowly. A selected shared view may disclose the changed forecast without exposing the owner's weekend plans.

### 24.6 F06 — competing initiatives

Two initiatives each require four hours before Wednesday and share one person with four total usable hours before Wednesday. Separate forecasts cannot both report on-time completion by assigning those same hours twice. The shared workload forecast shows the chosen priority allocation and the remaining shortfall. A what-if swap demonstrates which initiative benefits and which is displaced.

### 24.7 F07 — unknown and passive work

A report waits for an external answer with unknown release time. Drafting requires 60 human minutes once the answer arrives. Show known active demand and an unknown release constraint; do not forecast a precise finish without a labeled assumption. An automated build taking two hours occupies machine time and requires ten minutes of human review afterward. It blocks downstream review until completion but does not subtract two hours of personal active capacity.

### 24.8 F08 — source constraints change during planning

Daily supplies a preparation intent with source revision A. A proposal uses that revision. Before apply, Daily changes eligibility or preparation duration to revision B. Apply detects the material mismatch, leaves accepted allocations unchanged, and requests refreshed inputs. If Daily changes after a valid local apply, reconciliation flags affected future sessions and produces a proposal. The planner never claims to have locked Daily's database.

### 24.9 F09 — provider reset and recurring exception

A remote recurring event is moved to another time while retaining its provider-defined occurrence identity. Sync updates that occurrence instead of creating two. A later cursor invalidation starts a scoped full refresh. Native tasks, manual actuals, accepted commitments, and unrelated calendars remain intact. After the complete replacement snapshot commits, removed provider events are reflected appropriately. A failed refresh retains last-known busy state with a visible freshness warning.

### 24.10 F10 — two devices and uncertain responses

Two authenticated devices request focus start simultaneously with different idempotency keys. Exactly one current exclusive session is created; the other receives the active-session response. If the successful client's response is lost, retrying with its original key returns the existing session. Refreshing either device shows the same authoritative state. A second device cannot overwrite a later pause using an old revision.

### 24.11 F11 — selective sharing and revocation

The owner shares one commitment's public summary, accepted promise, and current forecast with a selected viewer. That viewer cannot query the underlying task, private event, dependency, actual log, or another commitment by changing IDs. API responses and errors contain no hidden titles. Revocation removes access on the next authorized request and closes/invalidates active subscriptions as supported; downloaded information cannot be remotely retracted.

### 24.12 F12 — timezones, all-day events, and DST

Generate fixtures from the chosen timezone library for a forward transition, a backward transition, a zone without seasonal changes, and an event viewed from a different zone. Assert actual elapsed durations separately from local clock labels. Native recurrence exceptions follow the explicit ambiguity policy; imported recurrence behavior follows its owner. An all-day item remains on its civil date when viewed elsewhere under the documented all-day policy; it is not shifted by parsing its date as UTC midnight.

Include a cross-midnight work session, an event ending exactly when another begins, a date-only commitment boundary, and a viewer traveling between zones. Do not hardcode the belief that every civil day is 24 hours. Store the effective timezone-data/runtime version in diagnostics for reproducibility when available.

### 24.13 F13 — incomplete history and honest learning

Ten tasks are complete, but only four have adequate actual coverage. The insight sample says four, not ten. Three unfinished long tasks remain visible as unfinished and are not assigned zero durations. A task whose scope doubled is either excluded from a like-for-like estimate ratio or explicitly grouped as changed scope. Before the configured evidence threshold, show descriptive history and explain why no adjustment is suggested.

### 24.14 F14 — visual geometry and appearance

Render a four-hour horizon with adjacent 30-, 60-, and 90-minute blocks. Their occupied widths follow a 1:2:3 ratio at the same zoom, allowing for separately drawn borders. Labels and accessible descriptions give the same times as the API. Switching Day to Night changes scene/palette but not card order, block placement, capacity values, or scroll position. Reduced motion disables ambient animation and animated theme transitions.

### 24.15 F15 — commitment revision and acknowledged expectations

The original promise is Friday 17:00 with a stated definition of done. The user explicitly records a revised promise of Monday 12:00. History retains both; the current accepted revision points to Monday. If the owner has not recorded that the other person agreed, show acknowledgment as unknown/unconfirmed. A forecast moving to Monday alone creates neither this promise revision nor an acknowledgment.

### 24.16 F16 — forgotten timer and overlapping corrections

A session appears to have run through the night. Flag the unusually long uninterrupted interval for review using a configurable rule; do not automatically label all of it productive work. The user corrects it to 35 minutes with approximate timing. Keep the correction provenance, update actual totals and learning eligibility, and leave the original recording inspectable until deletion. If a manual interval overlaps a known session, ask whether it corrects that session, duplicates it, or represents nonexclusive context; do not sum both as exclusive active effort.

## 25. Verification strategy and acceptance tests

### 25.1 Test strategy

Use the repository's established tools. Concentrate automated tests on semantic boundaries, concurrency, integration replay, and user-visible flows with meaningful failure risk. Avoid tests that merely repeat constant values or internal implementation steps. Rendering a card is insufficient evidence that its command, persistence, and derived results work.

Use pure domain/property tests for interval algebra, effort conservation, recurrence identity, dependency acyclicity, deterministic proposals, and nonmutation of forecasts. Use real database integration tests for transactions, unique constraints, tenant isolation, and idempotency. Use adapter contract tests with recorded/synthetic provider responses and live sandbox checks for the configured provider. Use a small end-to-end suite for the complete planning–focus–review loop and security-critical sharing interactions.

No real user's account or calendar should be modified by automated fixtures. R1 provider access is read-only, but integration setup and selecting calendars still require the appropriate configured test account. Redact captured provider examples and never commit credentials.

### 25.2 Acceptance case catalog

| ID | Case and required result | Evidence type |
| --- | --- | --- |
| T01 | Fresh manual onboarding reaches a persisted usable Today without provider or model configuration. | End-to-end, reload check. |
| T02 | Native task/event edits survive service/browser restart; unauthorized subject cannot read them. | Database/API integration. |
| T03 | F01 interval union, reserves, caps, and remaining-today scope reconcile to visible totals. | Domain fixture and UI/API comparison. |
| T04 | F02 split/move/cancel preserves demand lineage and does not double-count dated effort. | Transaction/property test. |
| T05 | F03 fragmentation respects chunk/setup constraints; no unqualified fit claim. | Planning fixture. |
| T06 | Dependency cycles are rejected; blocked/unknown release stays unresolved. | Domain/API integration. |
| T07 | Fixed events, locks, protected time, and source constraints survive every generated proposal. | Property/fixture tests. |
| T08 | Generate, compare, and discard proposals without changing accepted records or outbound messages. | Database integration. |
| T09 | Apply once, retry safely, reject stale input, and revalidate partial selection. | Transaction/concurrency integration. |
| T10 | Undo respects intervening edits; reverse preview appears when direct compensation would overwrite them. | API/end-to-end. |
| T11 | F06 shares capacity across initiatives; hypothetical allocations are absent from accepted occupancy. | Forecast/domain fixture. |
| T12 | F05/F15 preserve original promises and show effort/availability changes separately from reported causes. | API/history/UI. |
| T13 | Unknown effort, stale provider input, and beyond-horizon work prevent false certainty. | Forecast fixture and UI states. |
| T14 | F10 enforces one current exclusive session across devices and handles a lost response. | Concurrent database/API test. |
| T15 | Pause/break/resume/reload and clock reconciliation produce correct segments; timer end does not complete work. | Session integration/end-to-end. |
| T16 | F04/F16 manual correction preserves provenance, handles overlaps, and updates summaries without inventing timestamps. | Actuals integration. |
| T17 | Untimed/manual use can complete a review and plan tomorrow without running a timer. | End-to-end. |
| T18 | F13 uses only eligible evidence, includes unfinished-work caveats, and changes settings only on acceptance. | Learning integration. |
| T19 | Source duplicate/out-of-order deliveries preserve identity; canceled/deleted source items do not resurrect. | Adapter contract. |
| T20 | F08 rejects stale source constraints; pending source completion never displays as confirmed. | Adapter/API integration. |
| T21 | Daily/Cadence supported adapters open filtered full views while other busy facts still constrain capacity. | Live owner integration where available. |
| T22 | F09 provider pagination, deletion, moved recurrence, expired cursor, retry, and reconnect preserve local facts. | Provider contract and sandbox evidence. |
| T23 | F12 handles civil dates, recurrence exceptions, DST, all-day boundaries, and timezone changes. | Temporal/domain plus UI fixture. |
| T24 | F11 recipient binding, exact field mask, ID guessing, nested leaks, expiry, and revocation behave correctly. | Security integration/end-to-end. |
| T25 | Notification retries deduplicate, quiet hours apply, and sensitive reasons are absent from shared delivery. | Job/notification integration. |
| T26 | What-if changes cannot mutate source records, accepted commitments, shares, or notifications. | Transaction/adapter-spy test. |
| T27 | Native export/import round-trip preserves semantics; restored tokens/grants/sessions are safe. | Import/export integration. |
| T28 | Archive/undo/permanent delete update history visibility and learning without cross-tenant effects. | Data lifecycle integration. |
| T29 | F14 scene/layout/time geometry match; long labels, conflicts, and empty states remain readable. | Recorded visual/browser review. |
| T30 | Keyboard-only planning, focus, proposal application, and sharing preview work; labels/contrast/reduced motion pass review. | Automated accessibility plus manual checks. |
| T31 | Mobile Today/agenda/focus function at narrow widths with touch controls and no obscured primary action. | Responsive browser review. |
| T32 | Background crash/replay, outbox delay, lost job lease, and provider outage recover without duplicate effects. | Failure-injection integration. |
| T33 | CLI/tool commands enforce the same permissions and revisions as UI commands. | Contract/API parity. |
| T34 | Performance targets are measured with bounded realistic datasets; no external provider blocks Today. | Recorded benchmark/profile. |
| T35 | Legacy dry-run, stable ID mapping, recurrence reconciliation, restore, and post-cutover owner checks pass. | Migration rehearsal, if legacy data exists. |
| T36 | No model configured, model failure, or malicious imported text can bypass normal application commands. | Optional-assistance contract. |

### 25.3 Property-level invariants worth testing

For valid interval inputs, normalization is idempotent; interval union duration never exceeds the sum of component durations; adjacent half-open intervals do not overlap; and subtraction never creates negative durations. For allocation splits, total active demand is conserved except through an explicit effort revision. Repeated projection ingestion converges to the same authoritative state. For any generated proposal, all accepted changes satisfy hard constraints under its validated snapshot. Forecast generation leaves authoritative rows unchanged.

For histories, applying a correction changes effective totals without deleting the original measurement except through the explicit deletion workflow. For sharing, removing permissions can only maintain or reduce visible data. For sessions, any successful concurrent start sequence leaves at most one current exclusive session per person.

### 25.4 UX review script

Use a synthetic workspace containing a busy day, an overloaded day, uncertain work, two commitments, recurring routines, a source-owned task, and one stale integration. At desktop and phone widths, ask the reviewer to identify what fits today, start a task, record an interruption, revise remaining effort, inspect the affected promise, preview a schedule change, and share only the permitted summary.

Record whether the reviewer can distinguish available time from reserve, promise from forecast, planned from actual, source status from planner state, and an empty view from a failed query. Check that the beautiful scene does not compete with task text, that time geometry is truthful, and that all relevant actions are discoverable without explanatory developer narration.

### 25.5 Evidence and completion reporting

For each gate, record the actual command/test name, environment, date, result, and material limitation. Screenshots establish visual state; they do not prove database isolation. A fixture adapter establishes contract behavior; it does not prove a live Daily/Cadence connection. A mocked provider test does not establish live credentials or callback routing. Report those distinctions plainly.

Once a gate is adequately verified, avoid expanding tests merely to increase a count. Fix concrete failures and proceed to the next delivery dependency. Preserve a concise set of reproducible acceptance commands for the local developer.

## 26. Detailed implementation sequence

### 26.1 How to execute the work packages

The packages below define deliverables and dependency order, not calendar-time estimates. Repository discovery must precede reliable effort estimates. The local agent should turn each package into repository-native issues/checklists if useful, then implement small reviewable slices. Each package includes working UI, commands, persistence, and relevant verification; do not finish every screen with mocked data before starting the backend.

Packages may be interleaved when dependencies permit, but do not use that flexibility to postpone time semantics, authorization, visual quality, or source ownership until the end. R0 is reached after P02 with the stated basic shared contract. R1 requires P00–P11 and all applicable gates, with missing external credentials or unavailable source owners explicitly reported as incomplete integration gates.

| Package | Main dependency | Visible result |
| --- | --- | --- |
| P00 | None | Evidence-backed repository and migration decisions. |
| P01 | P00 | Persistent private app shell with the approved day/night foundation. |
| P02 | P01 | Manual calendar and hybrid day planning; R0 checkpoint. |
| P03 | P02 | Goals, milestones, explicit commitments, and dependency context. |
| P04 | P02; P03 for commitment context | Correct capacity and an honest, useful Today. |
| P05 | P03, P04 | Proposal/apply, scenario forecasts, delay history, what-if preview. |
| P06 | P02, P04 | Durable focus and correctable actual activity. |
| P07 | P03, P05, P06 | Daily/weekly review and evidence-linked learning. |
| P08 | P02, P05; P00 source discovery | Real native adapters and read-only calendar integration. |
| P09 | P03, P05; P08 for imported-context redaction | Selective read-only sharing and useful notifications. |
| P10 | P05–P09 | Portability, reliability, CLI/tool parity, operator readiness. |
| P11 | P00–P10 | Verified, polished R1 release and migration/cutover evidence. |

### 26.2 P00 — repository discovery and architectural decisions

**Deliver:** the actual scenario identity, owner map, stack/runtime map, integration capability inventory, migration decision, and verification commands.

Tasks:

1. Read repository instructions and inspect the old calendar scenario, actual Daily/Cadence implementations, auth, jobs, notifications, event transport, and shared primitives.
2. Trace existing scheduling writes and references so replacement does not create a competing authority.
3. Inspect old data volume, recurrence/timezone representation, dependent callers, and current backup mechanisms without mutating records.
4. Choose reuse versus successor using section 3; record why and identify compatibility/migration work.
5. Map the plan's logical modules and entity owners to the actual codebase. Identify which native task/goal records must be supplied by the planner.
6. Establish provider target from evidence; document required local credentials and callback configuration without requesting secrets in chat.
7. Create the implementation log, requirement/gate status file, and repository-native run instructions.

**Exit gate:** another developer can identify where each authoritative fact lives, how the app will run, how old data will survive, and what remains unavailable. No implementation claim rests only on a stale document reference. Discovery may reveal a blocker for one adapter without preventing the manual vertical slice.

### 26.3 P01 — durable foundation and visual shell

**Deliver:** a running authenticated private workspace, versioned migrations, application shell, Today route, quick capture, and both appearances with real persisted data.

Tasks:

1. Scaffold within actual scenario conventions; wire health/readiness, configuration, database migrations, and auth.
2. Implement workspace/profile/timezone initialization and tenant-scoped repository access.
3. Add native work creation/editing and event creation with minimal validated temporal types and revision handling.
4. Build the shared command/error/idempotency skeleton, history/outbox foundation, and injectable clock.
5. Implement navigation, typography, surface tokens, daytime/nighttime palettes, responsive shell, and reduced-motion preferences.
6. Produce the first paired landscape assets using the approved composition; keep text and time blocks as live UI.
7. Add quick capture and a real empty state. Reload and restart to demonstrate persistence.

**Exit gate:** T01/T02 pass for the foundation; a real user can create and reopen records; a second identity cannot read them; no model/provider is needed. Review desktop Day/Night and phone screenshots now, so the design is not deferred to release week.

### 26.4 P02 — temporal model, manual planning, and shared contract foundation

**Deliver:** native events, dated/timed allocations, basic Day/Week/Agenda, routine instances, and a minimal shared schedule read/write contract.

Tasks:

1. Implement interval normalization, date/instant/timezone distinctions, native recurrence policies, and occurrence identity.
2. Implement allocation lineage with transactional split, move, cancel/release, and lock commands.
3. Build date-assigned lists alongside timed placement, with drag preview and equivalent non-drag controls.
4. Implement recurring editor scope and instance overrides; preserve past actual references even before actual capture is enabled.
5. Introduce source registration/normalized identity/projection contracts and a synthetic source adapter for contract development.
6. Add deep-link/filter state and an embeddable minimal schedule summary using the same API.
7. Verify core temporal and conservation cases before using these values in forecasts.

**Exit gate / R0 checkpoint:** manual planning is usable and persistent; T04/T23 and relevant parts of T19/T33 pass. Fixed and flexible items remain distinct. The app has real Day/Week/Agenda views and the visual foundation, but is explicitly not yet the complete promised product.

### 26.5 P03 — outcomes, milestones, commitments, and dependencies

**Deliver:** goal/project context, measurable milestones, explicit promises with revision history, and trustworthy dependency status.

Tasks:

1. Implement native or referenced initiatives/goals according to discovered ownership; add lightweight standalone paths.
2. Build milestone completion criteria and declared derivation rules; expose missing evidence.
3. Implement draft/accept/revise/fulfill/cancel commitment commands and distinct acknowledgment metadata.
4. Add definition-of-done prompts, scope assumptions, beneficiary labels, and desired-versus-promised boundaries.
5. Implement dependency edges, cycle checking, wait/release uncertainty, and source-owned prerequisite status.
6. Build Goals, commitment detail/history, and the Commitments view in Plan.
7. Add preparation/wrap-up assumptions for availability promises and latest safe stopping calculations.

**Exit gate:** T06 and the promise-preservation portions of T12 pass. A changed target or forecast cannot rewrite a promise. Standalone tasks remain easy to capture without constructing a project hierarchy.

### 26.6 P04 — capacity accounting and the everyday Today experience

**Deliver:** interval-aware capacity, reserves, explicit demand, next-action explanations, and usable load views.

Tasks:

1. Implement availability rules/exceptions, protected time, overlapping exclusion union, and configured caps.
2. Account for reserve exactly once, including explicit reserve intervals if supported.
3. Aggregate timed occupancy, dated demand, routine period demand, setup/breaks, unknown estimates, and overallocations.
4. Build whole-day and remaining-today scopes with visible labels and detailed arithmetic.
5. Implement Today priority hierarchy and a deterministic next-action suggestion that respects available window, eligibility, user priorities, and stopping boundaries.
6. Add Capacity and Month load views; finish useful empty, overloaded, fragmented, and unknown-input states.
7. Use identical domain values in visual widths, summary text, accessible labels, and API output.

**Exit gate:** T03/T05 and relevant T07/T29 pass. F01/F02/F03 numbers reconcile. Filtering a source cannot make its busy time disappear from planning. Today is useful before the full planner automatically proposes placements.

### 26.7 P05 — proposal engine, forecasts, and explainable change

**Deliver:** deterministic candidate scheduling, review/apply/undo, shared-resource forecasts, commitment risk, and baseline comparison.

Tasks:

1. Define immutable planner input/output schemas and scenario fingerprints; implement the pure kernel.
2. Implement stable priority ordering, eligible window construction, split/setup policies, and bounded repair.
3. Evaluate hard constraints after construction and return unresolved items/reasons rather than silently dropping them.
4. Build proposal diff UI, operation selection, scope controls, stale detection, atomic apply, idempotent retry, and conditional undo.
5. Forecast all competing workload from the same resource snapshot; add central/cautious scenarios with explicit assumptions.
6. Build milestone/project Timeline and promise-versus-forecast comparisons, including original/latest/current selectors.
7. Record material change explanations and optional user-reported causes; keep causality labels precise.
8. Implement the five R1 what-if categories in section 11.5 with no external side effects.
9. Add background refresh coalescing and clear stale/updating/beyond-horizon displays.

**Exit gate:** T07–T13 and T26 pass. Demonstrate a source/availability revision arriving before apply, a partial apply that needs revalidation, F05 Friday-to-Monday movement, and F06 resource contention. No background job changes accepted sessions.

### 26.8 P06 — focus, sessions, and actuals

**Deliver:** event/task-to-focus entry, four focus modes, durable session state, interruption capture, and correctable actual effort.

Tasks:

1. Implement the session/segment model, person-scoped current-session constraint, command idempotency, and revision-checked transitions.
2. Build start/resume/pause/finish/switch UI with Pomodoro/countdown/open/untimed modes and domain-specific source actions.
3. Render from authoritative timestamps and reconcile on reconnect, refresh, tab visibility change, and process restart.
4. Add quiet stopping cues tied to the next fixed obligation and optional wrap-up notes.
5. Implement Done/More to do/Blocked wrap-up with independently editable remaining effort and source-owner completion requests.
6. Support manual durations, exact intervals, approximate entries, segment correction, reassignment, and overlap resolution.
7. Add pending offline correction handling and forgotten-timer review without inventing continuous productive effort.

**Exit gate:** T14–T17 pass. Demonstrate F04/F10/F16 and manual-only use. A timer refresh retains state; a second device cannot create duplicate active work; a failed source update remains visibly pending.

### 26.9 P07 — daily/weekly review and learning

**Deliver:** useful planned-versus-actual review, visible tracking coverage, comparable estimate history, and inspectable setting suggestions.

Tasks:

1. Build daily and weekly read models with effective actuals, original estimates, scope revisions, promise outcomes, and coverage.
2. Implement brief daily review and selective carry-forward with a capacity preview.
3. Add weekly comparisons, goal attention, unresolved work, commitment outcomes, and meaningful change history.
4. Implement the initial rule set and evidence eligibility/grouping controls; preserve sample counts and exclusions.
5. Present recommendation preview, inspect evidence, accept/dismiss/mute, and reset/exclude controls.
6. Apply accepted adjustments through profile/policy commands, with effective dates and history.
7. Explain insufficient evidence and avoid probabilities or causal claims not supported by the model.

**Exit gate:** T18 plus applicable T12/T16/T17 pass. F13 excludes untracked/changed-scope records appropriately. The app works for users who skip every review; learning never changes settings in the background.

### 26.10 P08 — real source integration and read-only calendars

**Deliver:** supported Daily/Cadence adapters, full-calendar deep links, and at least one configured real read-only provider path.

Tasks:

1. Map source contracts to actual owner APIs and permissions, including human attention versus automated markers/wait.
2. Implement ingestion, origin uniqueness, revisions, cancellation, freshness, retry receipts, and source constraint validation.
3. Implement supported source-owned command callbacks with pending/failed/succeeded status and safe idempotency.
4. Add source-app schedule summaries and Open full planner actions, preserving appropriate source context.
5. Implement provider connection setup through existing credential infrastructure, calendar selection, initial sync, incremental sync, normalized recurrence/deletion, and health.
6. Implement cursor-reset staging, pagination failure recovery, retry/backoff, reconnect/disconnect, and stale-busy behavior.
7. Add ICS preview/import/export with declared supported semantics and origin/conflict handling.
8. Run the adapter harness and real provider sandbox checks; record unavailable source capabilities separately.

**Exit gate:** T19–T23 pass for available owners/providers, with T21/T22 live evidence explicitly distinguished from fixtures. Scheduling a meal preparation session in one app constrains the full day; an automatic content publication marker does not consume human time unless its intent says it should.

### 26.11 P09 — selective sharing and notifications

**Deliver:** recipient-bound commitment projections, exact preview/revocation, and useful deduplicated owner notifications.

Tasks:

1. Implement explicit record/field grant schema and a separate viewer query/projection layer.
2. Build owner selection, exact viewer preview, recipient binding or trusted redemption, expiry, and revocation.
3. Show current forecast only when selected; sanitize derived explanations and nested references.
4. Add the read-only viewer experience at narrow widths; no private calendar navigation leaks.
5. Publish notification intents through the discovered notification owner; implement preferences, quiet hours, lead times, deduplication, and state recheck before delivery.
6. Consolidate related risk notices and support review/open actions without automatically messaging another person.
7. Exercise direct API enumeration and revocation, not just visual masking.

**Exit gate:** T24/T25 pass. F11 viewer cannot retrieve private data via ID manipulation, exports, error strings, or subscriptions. Creating a share provides the explicit sharing mechanism; it does not silently email a beneficiary.

### 26.12 P10 — portability, failure recovery, and integration surface

**Deliver:** native export/restore, actuals CSV and calendar ICS, reliable jobs, developer CLI/tools, redacted diagnostics, and deployment instructions.

Tasks:

1. Implement versioned native export and validate/preview/apply import with stable identity mapping.
2. Add archive/undo/permanent deletion and derived-data cleanup under the documented retention policy.
3. Harden transaction/outbox replay, worker leases, retry terminal states, and reconciliation after outages.
4. Implement actual CLI/tool mappings over shared commands with structured inputs/outputs and permission parity.
5. Add bounded program wrappers only where the current runtime supports them and they are useful.
6. Measure normal and reference-scale workloads; fix unbounded queries and long request paths.
7. Implement owner integration-health views, safe request diagnostics, and operational metrics.
8. Rehearse backup/restore and old scheduler migration; finalize actual runbook commands.

**Exit gate:** T27/T28/T32–T36 pass where applicable. A round-trip restore retains promises/actuals and does not reactivate shares, credentials, or a running timer. The developer can diagnose a stale provider and pending source action without reading private content from logs.

### 26.13 P11 — complete release verification and product polish

**Deliver:** the first complete personal product, a verified owner transition, and an honest release report.

Tasks:

1. Run the small end-to-end planning–focus–review journey through real persistence and supported integrations.
2. Review all main views in Day/Night, phone/desktop, keyboard-only use, reduced motion, 200% zoom, long content, empty/error/stale states.
3. Finish all accepted views: Today; Day/Week/Month/Agenda/Timeline/Capacity/Commitments; Goals; Focus; Review; and Settings.
4. Verify shared source widgets and the full planner have consistent schedule values and capacity scope.
5. Remove demo-only controls, placeholder metrics, invented confidence values, and mock data from real workspaces.
6. Reconcile the requirement matrix and mark every gate complete, incomplete, or not applicable with evidence and justification.
7. Perform migration/cutover under the established deployment authorization, verify single ownership, and keep rollback/restore instructions available.
8. Produce the release report: what works, exact limitations, live versus fixture verification, migration result, known defects, and R2 backlog.

**Exit gate:** all applicable R1 requirements and T01–T36 have sufficient evidence, including visual T29–T31 and measured T34. Any live provider/source prerequisites that remain unavailable are named; the release cannot be described as fully integrated until they are verified.

## 27. Requirement traceability and release checklist

The local implementation should maintain this matrix with actual evidence links. An empty evidence cell is unfinished verification, not assumed success.

| Requirement | Definition | Specification | Delivery | Main acceptance evidence |
| --- | --- | --- | --- | --- |
| REQ-01 | Private standalone manual use with persistent native records | 1–5, 19, 22 | P01–P02 | T01–T02 |
| REQ-02 | Observatory layout and matched Living Landscape day/night scenes | 6, 22 | P01, P11 | T29–T31 |
| REQ-03 | Hybrid fixed/timed/dated/backlog planning without double counting | 7–8, 10 | P02, P04 | T03–T05 |
| REQ-04 | Day, Week, Month, Agenda, Timeline, Capacity, and Commitments views | 5, 8, 12 | P02–P05, P11 | T03, T12, T29–T31 |
| REQ-05 | Correct recurring instances, civil dates, and timezones | 8, 16, 19 | P02, P08 | T22–T23 |
| REQ-06 | Goals, lightweight projects/references, and evidence-based milestones | 9, 19 | P03 | T06, T12, journey review |
| REQ-07 | Explicit promise/definition of done/revision/acknowledgment history | 9, 12 | P03, P05 | T12, F15 |
| REQ-08 | Dependencies, unknown waits, passive work, and shared resource contention | 9–12, 15 | P03–P05, P08 | T06–T07, T11, T20 |
| REQ-09 | Capacity, reserves, fragmentation, unknown effort, and preparation time | 10–11 | P04 | T03–T05, T13 |
| REQ-10 | Manual proposal preview/apply, partial revalidation, and safe undo | 11, 20 | P05 | T07–T10 |
| REQ-11 | Explainable scenario forecasts and historical comparison | 12 | P05 | T11–T13 |
| REQ-12 | Side-effect-free what-if tradeoff exploration | 11–12 | P05 | T26 |
| REQ-13 | Event/task-to-focus, four modes, stopping cues, cross-device durability | 13, 19, 23 | P06 | T14–T15, T31 |
| REQ-14 | Optional tracking, manual correction, provenance, independent remaining effort | 13 | P06 | T16–T17 |
| REQ-15 | Daily/weekly reviews, coverage, and accepted evidence-linked adjustments | 14 | P07 | T18, F13 |
| REQ-16 | Shared scheduling owner, normalized intents, source revisions, and deep links | 3, 15, 21 | P00, P02, P08 | T19–T21, T33, T35 |
| REQ-17 | Real read-only external calendar integration and honest freshness | 16 | P08 | T22–T23 plus live setup evidence |
| REQ-18 | Explicit selected read-only commitment sharing without private leakage | 17 | P09 | T24 |
| REQ-19 | Useful notifications with preferences and deduplicated delivery | 18 | P09 | T25, T32 |
| REQ-20 | Native portability, CSV/ICS, deletion, and safe restoration | 16, 23 | P08, P10 | T27–T28 |
| REQ-21 | Identity isolation, revision checks, idempotency, and durable jobs | 4, 19–23 | P01 onward | T02, T09, T14, T24, T32–T33 |
| REQ-22 | Accessible responsive UI and truthful time geometry | 6, 22 | P01 onward, P11 | T29–T31 |
| REQ-23 | Useful without AI; bounded optional assistant behavior | 18, 21 | P01 onward | T01, T26, T36 |
| REQ-24 | Measured performance, diagnostics, and deployment/migration runbook | 3, 23 | P00, P10–P11 | T32, T34–T35 |

Before declaring R1 complete, confirm that D01–D09 are still reflected in the implementation and that INV-01–INV-16 have a concrete enforcement point. If a requirement cannot be fulfilled because an external owner is unavailable, keep the requirement visible and describe the supported fallback and exact missing contract. Do not silently redefine “done.”

## 28. Configuration defaults, unresolved discovery, and risks

### 28.1 Initial defaults

Defaults here are deliberately explicit so the local agent can implement without asking a new question for every routine choice. They are adjustable product/engineering defaults, not conclusions about the user's actual capacity or preferences.

| Setting | Initial default | User-visible or operational treatment |
| --- | --- | --- |
| Workspace | One private personal workspace | Map to existing host identity; no team editing UI. |
| Timezone | Detected suggestion, confirmed during setup | Persist independently from browser changes. |
| Week start | Locale suggestion, user editable | Used consistently for routines/review/grouping. |
| Availability | User-defined; quick editable template allowed | Never treat all waking hours as work capacity. |
| Daily occupied-time cap | Unset until configured | Calendar availability still bounds capacity. |
| Demanding-work active cap | Unset until configured | Separate quantity from break-inclusive occupancy. |
| Reserve | Offer 20% of potential capacity as an editable onboarding suggestion | No claim of learned suitability; apply only after the user accepts the profile. Allow fixed minutes or explicit intervals instead. |
| Appearance | Auto with suggested Day 07:00/Night 19:00 | Editable, no location required; postpone transition during focus. |
| Ambient motion | Subtle and optional | Disable for reduced-motion preference. |
| Scheduling action | Generate preview, apply explicitly | Never automatically move accepted work. |
| Placement scope | Selected unlocked future work | Preserve outside-scope allocations and fixed facts. |
| Interactive horizon | 28 days maximum by default | Selected day/week remains the primary UX scope. |
| Forecast horizon | 90 days initially | Configurable and bounded; unresolved beyond coverage. |
| Generic splitting | 25-minute minimum, up to 90-minute preferred maximum; short final remainder allowed | Editable by type/item; source constraints take precedence. |
| Scheduling grid | Five-minute UI snapping; exact source times retained | Store exact instants; keyboard/detail editor allows finer precision. |
| Pomodoro | 25 work / 5 break | Editable; next work interval awaits explicit resume. |
| Default focus mode | Last chosen mode; initial Pomodoro suggestion | Untimed and manual use are equally accessible. |
| Effort certainty | Unknown until supplied | Optional scenario placeholders are labeled assumptions. |
| Forecast labels | Central/cautious scenario | No statistical probability in R1. |
| Learning window | Most recent 90 days initially | Show scope and allow reset/exclusion; preserve older history. |
| Numerical adjustment evidence | At least eight comparable completed records over three distinct days | Adequate actual coverage still required; configurable product guardrail. |
| Provider | Reuse existing adapter/account support; Google fallback implementation target | Confirm actual local capability; not a claim about the user's provider. |
| Provider refresh | Initial target five minutes when active, slower when idle | Respect quotas/backoff and display last success; exact policy is adapter/deployment configuration. |
| Stale busy facts | Retain with visible freshness warning | Never treat provider failure as new free time. |
| Share mode | Selected recipient, selected records and fields, read-only | Proposed default expiry 30 days, editable or no expiry by explicit choice; no automatic send. |
| Review reminders | Off until opted in | User chooses time/channel; skipping has no penalty. |
| Risk notices | In-app on newly crossed commitment boundary; deduplicated | Same-boundary date/time changes update the existing notice; external channels opt-in. |
| Routine reminders | Off until configured per routine/type | Avoid default notification overload. |
| Native import | Validate/preview, then new private workspace | Merge requires explicit conflict mapping. |
| Proposal expiry | 24 hours maximum by default | Material revision invalidates earlier; history retention is separate. |
| Interactive idempotency results | Seven days | Durable entity/source uniqueness protects older replay. |
| Export download artifacts | 24 hours | Authenticated access; user can regenerate. |
| User-authored history | Retain until user deletes | Explicit deletion/backup policy remains visible. |
| Diagnostics/transient previews | 30 days | Align with host policy; never log private content by default. |

For percentage reserve, retain the policy ratio and compute at the backend's canonical precision. Round the proposed reserve conservatively to a documented minute boundary once, then use that same result in UI and solver. Do not display one rounded reserve while scheduling against another hidden value.

### 28.2 Discovery items that remain legitimate

These require local evidence rather than another product brainstorming round:

- The actual old scheduler identity, data quality, dependent apps, and preferred cutover method.
- Current Vrooli language/framework, auth/workspace/guest model, database, event bus, notification infrastructure, and deployment commands.
- Whether another working scenario owns tasks, goals, or initiatives.
- Actual Daily/Cadence APIs, source slugs, constraints, and supported command callbacks.
- Available provider integration and test credentials; account-specific calendar selection is runtime configuration.
- Available fonts/assets/licenses and the repository's current accessible UI primitives.
- Whether guest identity supports selected sharing directly or requires a scoped viewer flow.
- Performance on the target machine and runtime; adjust bounded limits from measured evidence.
- Product name and public branding, which do not block domain or UI implementation under the working label.

The accepted product decisions do not need to be reopened because these details are not yet known. The local agent should record discoveries, use the documented defaults, and surface only conflicts that materially change scope, ownership, access, or data safety.

### 28.3 Risk register

| Risk | Consequence | Mitigation and evidence |
| --- | --- | --- |
| Duplicate scheduling owners | Apps disagree and overwrite one another | P00 owner map, shared command surface, cutover gate T35. |
| Attractive visuals conceal wrong arithmetic | Trust collapses when the plan overbooks | Shared domain values, fixtures F01–F03/F14, T03/T29. |
| Tracking becomes burdensome | Users abandon the app before learning helps | Manual path, optional wrap-up, brief reviews, progressive disclosure, T17. |
| Forecasts appear more certain than inputs | Users repeat unrealistic promises | Scenario labels, unknown/stale states, independent promise history, T12–T13. |
| Too much schedule churn | Planner feels disruptive | Stable placement preference, scope/locks, preview/apply, T07–T10. |
| Source business rules leak into generic scheduler | Duplicated domain logic drifts | Normalized constraints, source validation/revisions, T19–T20. |
| Recurrence/provider edge cases corrupt records | Missing or duplicated obligations | Stable occurrence identity, staged resets, temporal fixtures, T22–T23. |
| Private causes leak through sharing | Viewer sees unrelated personal information | Separate DTOs, derived-text redaction, network-level tests, T24. |
| Timers overstate productive time | Learning becomes misleading | Independent segments, corrections, coverage, one current session, T14–T18. |
| Early learning fits tiny biased samples | Bad recommendations become defaults | Evidence thresholds, exclusions, unfinished-work disclosure, explicit acceptance. |
| Full feature set leads to an unfinished shell | User receives another underdeveloped calendar | Vertical slices, persistent R0, complete R1 gate, no dead controls. |
| Repository assumptions are wrong | Rework or incompatible architecture | Discovery before implementation; logical contracts rather than invented file paths. |
| Integration setup unavailable | False claim of completed ecosystem support | Fixture/live evidence distinction; maintain open gates and manual usefulness. |
| Scope expands into general project management | Core personal loop becomes hard to use | Respect owners, lightweight native fallback, retain R2/R3 boundary. |

## 29. Handoff, definition of done, and reference notes

### 29.1 Instructions for the local agent

Read this document and the repository's current instructions, establish the actual ownership/stack/migration facts in P00, then implement the R1 scope through the listed vertical slices. Preserve the approved Observatory layout with matched day/night scenery from the start. Keep accepted schedules, promises, actuals, and forecasts distinct. Use existing Vrooli owners and infrastructure where they work, and supply native standalone functionality where no owner exists. Maintain requirement/gate evidence and continue through working persistence, error paths, integrations, responsive accessibility, and deployment readiness. Report actual blockers and unverified external capabilities precisely; do not replace them with mock success.

### 29.2 Product definition of done

A person can open the app, understand today's real constraints, decide what fits, see how a new commitment affects other work, apply a manageable plan, start focused work, record or correct what happened, inspect why a forecast changed, and make a better next plan. They can do this without connecting a provider or using a timer, and gain additional context by enabling those features. They can share a selected commitment without exposing their private calendar. Daily/Cadence retain their own domain experiences while using the same authoritative schedule.

The interface feels like one coherent place throughout the day: a quiet living landscape in daylight and an observatory at night, with the same readable controls and truthful timeline. Its beauty supports repeated use; its accounting and history support trust.

### 29.3 Required implementation report

At each meaningful checkpoint, the local agent should report the implemented user behavior, changed domain/contracts, actual verification, remaining risks, and next dependency. At R1, provide:

1. Actual scenario location and run/access instructions.
2. Completed requirement matrix with verification evidence.
3. Owner/adapter map and live versus fixture integration status.
4. Migration/backup/restore/cutover result, including any unsupported legacy records.
5. Screenshots of Today Day/Night, a busy Plan view, focus, review, mobile, and the shared viewer.
6. Known limitations and any incomplete gates, without overstating completion.
7. The bounded R2/R3 backlog and decisions intentionally deferred.

Do not treat the number of screens, lines of code, tests, or generated screenshots as the success criterion. The product must support the complete observable user loop and preserve the invariants behind it.

### 29.4 References and scope of verification

This plan is a product/engineering specification, not a claim that the destination repository or accounts have been inspected. Provider/standard references were consulted while preparing this plan on September 18, 2026. Recheck current official documentation during adapter implementation, especially requested scopes, pagination, recurrence, error/reset handling, and callback requirements.

| Reference | Why it matters |
| --- | --- |
| [Google Calendar synchronization guide](https://developers.google.com/workspace/calendar/api/guides/sync) | Incremental state, pagination, deletions, and invalidated synchronization tokens. |
| [Google Calendar recurring events](https://developers.google.com/workspace/calendar/api/guides/recurringevents) | Series/instance identity and changed occurrences. |
| [Microsoft Graph event delta queries](https://learn.microsoft.com/en-us/graph/delta-query-events) | Scoped change tracking and continuation state for a future or existing Microsoft adapter. |
| [RFC 5545: iCalendar](https://www.rfc-editor.org/info/rfc5545/) | Interchange date/time, recurrence, and exception semantics. |
| [WCAG 2.2](https://www.w3.org/TR/WCAG22/) | Accessibility requirements to apply through automated and manual review. |

These sources inform specific interoperability/accessibility behavior. Product choices such as reserve percentage, focus defaults, forecast horizons, learning thresholds, visual tokens, and release sequencing are recommendations in this plan; they are not scientific findings or claims made by those references.

### 29.5 Implementation log template

Maintain this compact record in the repository rather than editing historical decisions silently:

| Date/checkpoint | Discovery or decision | Evidence/location | Requirements affected | Verification/remaining gate |
| --- | --- | --- | --- | --- |
| First local discovery | Populate with actual repository facts | Actual source paths or commands | Relevant REQ IDs | Observed result and limitation |

For an intentional change to a recommended default, record the old value, new value, reason, and resulting behavior. For a change to an accepted product decision, record the user's explicit new direction. Keep this document's version and the implemented contract/model versions distinguishable.
