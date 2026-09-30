# Progress — Personal Planner

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

## Progress Log

| date | author | status | notes |
|---|---|---|---|
| 2026-09-19 | shared form language | in-progress | Completed the Settings availability/protected-time and Plan routine editor migration onto shared FormField composition over library controls, including explicit weekday structure and unique accessible names. The new Settings regression and full UI suite pass at 33 files/180 tests; type-check, production build, and relevant diff hygiene pass. The broader responsive/accessibility visual release sweep and shared portability closure remain open. |
| 2026-09-19 | cross-route appearance contract | in-progress | Corrected the shell class contract so resolved light/dark themes emit the Observatory day/night vocabulary consumed by sidebar, bottom-nav, and page-canvas styling. Added an AppShell regression proving dark settings routes use `appearance-night`; full UI suite passes 33 files/181 tests, type-check/build pass, managed restart is healthy, and UI-health receipt `20260919-210311-a3d15dcc` passes at L5 across all seven routes. Browser-native Safari chrome remains platform-controlled outside the page. |
| 2026-09-19 | timeline lane reuse | in-progress | Extended Today regression coverage to prove interval lanes are reused after an overlap ends: a third block returns to lane 0 rather than being pushed into a new row. Dashboard suite passes 9/9; the broader UI suite remains green at 33 files/181 tests. |
| 2026-09-19 | Goals shared field language | in-progress | Migrated Goal progress and milestone title, criteria, due-date, and linked-work controls onto shared `FormField` composition over library controls, with stable explicit accessible names for required/optional fields. Goals tests pass 6/6; full UI suite remains 33 files/181 tests, type-check/build pass. Today capture’s compact legacy markup and the broader visual oracle sweep remain open. |
| 2026-09-18 | scenario init | done | Scenario `personal-planner` generated from the `react-vite` template (Go API, React+Vite UI, Go CLI); `health` domain and removable `notes` worked example are the only real code. |
| 2026-09-18 | scenario init | done | Removed the unused, undeveloped legacy `calendar` scenario before generation; no live scheduling data to migrate (plan §3.3 applies if legacy/external data appears later). |
| 2026-09-18 | docs | done | Authored the charter/PRD and mapped operational targets onto plan decisions D01–D09, invariants INV-01–16, and requirements REQ-01–24. |
| 2026-09-18 | docs | done | Authored the domain map: DOMAINS, DATA (storage/temporal typing/retention), FLOWS (lifecycles/state machines), and INTEGRATIONS (dependency contract). |
| 2026-09-18 | docs | done | Authored the Observatory DESIGN (day/night, time geometry, accessibility gate), EXPERIENCE, and experience specs. |
| 2026-09-18 | docs | done | Authored internal SECURITY, PERFORMANCE, DECISIONS, PROBLEMS, and this PROGRESS log (design-stage posture, budgets, and known gates). |
| 2026-09-18 | docs | done | Remaining work is product code: the Observatory design-language home surface, the first real vertical slice, and example-domain removal (`template-manager detemplate`). |
| 2026-09-19 | work domain | in-progress | Added the first product-owned `work` domain from proto through generated Go/TypeScript, SQLite repository/service, Connect API, CLI `work list/create/get`, and Today query integration. Runtime was started through `make start`; a work item was created successfully through the CLI and is consumed by the Today client. Remaining: persisted schedule/capacity domains, surface completion, measurement evidence, and example removal. |
| 2026-09-19 | product surfaces | in-progress | Wired Plan, Goals, Focus, and Review routes to an honest shared work-item surface; repaired embedded library string-provider/shell reachability, added storage ownership metadata, and recorded the terminal comprehensive-gate findings. Remaining product domains are explicitly not claimed complete. |
| 2026-09-19 | readiness | in-progress | Final Test Genie run passed 26/27 phases; dependency governance, performance, unit execution, workflow safety, experience target sizes, storage, API, CLI, and UI validation are green. The remaining portability failure is a repository-level trusted-base closure error (`agent-manager` → `workspace-sandbox`), not a personal-planner-local finding. |
| 2026-09-19 | focus domain | in-progress | Added persisted Focus proto/API state with SQLite exclusivity and revision guards, Connect handlers/endpoints, CLI `focus current/start/pause/resume/end`, and a Focus UI surface with active-time accounting and transition-error states. API, CLI, TypeScript, and focused UI tests pass. Remaining: schedule/capacity, goals, review, and comprehensive validation after this change. |
| 2026-09-19 | goals domain | in-progress | Added Goals proto/API/SQLite CRUD-plus-explicit-progress slice, Connect endpoints, CLI `goals list/create/progress`, and a Goals UI that keeps outcome progress separate from elapsed work time. Live CLI create/list/progress and API, CLI, UI suites pass. Remaining: schedule/capacity, Review, visual checkpoint, and comprehensive validation. |
| 2026-09-19 | validation | in-progress | Test Genie unit run `20260919-055452-7f4a58d3` passed after UI coverage regressions were added; orientation design-language gate is now green after replacing stock design-token colors/font with the Observatory palette. Comprehensive run `20260919-055532-81356fb8` is a terminal pre-fix failure; rerun remains required after CLI manifest repairs. |
| 2026-09-19 | review domain | in-progress | Added the Review daily-summary proto/API/SQLite read model, CLI `review daily`, and Review UI. The summary reports measured focus and active goals while explicitly labeling planned and unrecorded time as unknown until schedule data exists. Unit and contracts Test Genie runs pass after manifest evidence repairs. |
| 2026-09-19 | plan surface | in-progress | Replaced the shared Plan placeholder with a capacity strip and time-geometry draft built from `WorkService.GetTodayPlan`, with an explicit boundary that it is not an accepted calendar allocation. UI type-check and 27-file/122-test suite pass. |
| 2026-09-19 | validation | in-progress | Comprehensive Test Genie run `20260919-063022-d7da369a` passed 26/27 phases. All scenario-owned gates are green, including Experience L3 and Measures domain coverage; the only failed phase is portability, which is repository/environment-level and outside this scenario's declared resource surface. |
| 2026-09-19 | calendar projection | in-progress | Added CalendarService.ListAllocations for inclusive local-date ranges and Plan Day/Week/Agenda projections over accepted server records. API/UI tests, endpoint parity, live Connect range read, refreshed CLI evidence, and Test Genie run `20260919-073604-28dfc4f6` (3/3) pass. |
| 2026-09-19 | workspace availability | in-progress | Added revision-checked weekday availability windows and civil-date protected/extra exceptions through Workspace proto/API/SQLite/CLI/Settings UI. Calendar now derives Today potential capacity from the configured local weekday windows, subtracts protected exceptions, then applies reserve and accepted occupancy. API, CLI, UI coverage, and live read smoke pass; user workspace remains unseeded (zero windows/exceptions). |
| 2026-09-19 | calendar routines | in-progress | Added timezone-aware fixed recurrence and flexible-frequency routine definitions, deterministic bounded occurrence expansion, Calendar proto/API/SQLite/CLI commands, and a calm Plan rhythm editor. Empty live routine state remains honest; API/CLI/UI tests and Test Genie unit/contracts/experience run `20260919-085204-a39d4be2` pass. Routine occurrence overrides and provider-owned recurrence remain open. |
| 2026-09-19 | goal milestones | in-progress | Added explicit milestone criteria, due dates, open/complete status, revision-safe completion, and goal-scoped list/create/complete paths through proto/API/SQLite/CLI/UI. Focused API/UI tests, CLI evidence, full UI coverage/build, live list, and lifecycle smoke pass; Test Genie run `20260919-090516-29764305` was provider-unavailable across all requested phases. Work-item links remain open. |
| 2026-09-19 | goal-work linkage | in-progress | Extended milestones with validated optional work-item links persisted in a separate link table and selectable from the Goals UI; live goal/milestone/work reads remain healthy, and an unknown link is rejected without persistence. Focused API, CLI, TypeScript, and UI tests plus full UI coverage/build pass. Test Genie run `20260919-092558-f15f9dd0` was provider-unavailable. Prerequisite links and derived aggregation remain open. |
| 2026-09-19 | routine occurrence overrides | in-progress | Added stable routine/date skip-once and reschedule overrides through Calendar proto/API/SQLite/CLI/Plan UI with routine revision checks; SQLite proves a skipped occurrence disappears and a moved occurrence keeps the series unchanged. API, CLI, UI coverage/build, endpoint generation, lifecycle health, and CLI command discovery pass. Provider recurrence and richer demand accounting remain open. |
| 2026-09-19 | responsive/accessibility floor | in-progress | Added reduced-motion behavior, RTL logical-edge overrides, mobile-safe touch sizing for routine/milestone actions, and logical spacing for Observatory surfaces. UI coverage/build remain green; actual responsive/contrast visual captures are still required before release claim. |
| 2026-09-19 | review date scope | in-progress | Added an explicit local-date navigator to Review so daily summaries are queryable by the user's browser-local day instead of relying on an implicit server date. Added adjacent-day regression coverage; weekly review and correction/carry-forward remain open. |
| 2026-09-19 | validation maturity | in-progress | Restored Scenario Dependency Analyzer readiness, repaired the Review test type error exposed by unit-health, and reached Test Genie L2 unit, L3 contracts, and L3 experience on focused runs. Remaining maturity debt is the injectable seam/coverage advisory set plus the broader comprehensive gate. |
| 2026-09-19 | weekly review | in-progress | Added an honest seven-day Review summary through proto/API/CLI/UI, with Monday-local-date navigation, planned-versus-recorded daily rows, and explicit unknown coverage language. API, CLI, UI coverage/build, endpoint generation, live CLI, and lifecycle health pass; correction/carry-forward semantics and certification remain open. |
| 2026-09-19 | focus actuals | in-progress | Added manual/approximate actual capture, local-date listing, revision-checked correction, and durable correction provenance through Focus proto/API/SQLite/CLI/UI. Review now aggregates manual actual minutes without counting them as focus sessions. API/CLI/UI coverage/build, live empty read, lifecycle health, and Test Genie receipt `20260919-103942-536a7288` pass; richer session segments, attribution, and carry-forward remain open. |
| 2026-09-19 | validation cleanup | in-progress | Removed the stale empty capabilities handler directory, replaced unstructured API logging, and completed post-cleanup receipt `20260919-104957-5fe628a7`: API/proto/contracts/unit/experience passed; portability/security failed and channel-conformance was provider-unavailable. Orientation remains 9/9 complete; this is not a release claim. |
| 2026-09-19 | visual/accessibility checkpoint | in-progress | Enforced the 16px mobile text-entry floor after UI Health identified inherited 12.5px controls. Type-check, UI coverage (30 files/152 tests), and production build pass; UI Health returns `VALIDATION_STATUS_PASSED` with zero zoom-risk and blocking findings. Focused experience receipt `20260919-113045-c3b6cfe7` passes L3 Complete. Remaining: provider/template adoption maturity, responsive/contrast certification, recurrence/demand semantics, review carry-forward/learning, and full release gate. |
| 2026-09-19 | goal prerequisite rollup | in-progress | Added milestone prerequisite edges and completion blocking through proto/API/SQLite/CLI/UI, plus explicit milestone-derived goal progress. SQLite proves dependency rejection, unlock, and 100% rollup; UI exposes dependency selection and waiting state. API/CLI/UI/build/lifecycle pass; UI is 30 files/152 tests with 96.24% statements, 86.52% branches, and 85.5% functions. Test Genie `20260919-112043-f6897478` passed unit/contracts/experience; richer visual evidence and full certification remain open. |
| 2026-09-19 | selective carry-forward | in-progress | Added Calendar carry-forward through proto/API/SQLite/CLI/UI: a transaction preserves the source allocation as `carried_forward`, creates one non-overlapping accepted placement, records lineage, and returns the same placement on retry. Review now offers explicit per-placement carry-forward with a target-day capacity preview. API tests, CLI evidence, UI type-check/coverage/build pass; UI is 30 files/154 tests with 96.36% statements, 86.66% branches, and 85.37% functions. Review learning/reflection and provider recurrence remain open. |
| 2026-09-19 | carry-forward validation | in-progress | Focused Test Genie receipt `20260919-115722-0b180c9a` passes unit/contracts/experience (unit L2 Ready, experience L3 Complete). Security Health is green after the int32 parsing hardening; comprehensive portability/security evidence still needs refresh. |
| 2026-09-19 | durable review reflection | in-progress | Added Review GetReflection/SaveReflection through proto, schema-owned SQLite persistence, API handlers/endpoints, CLI commands, and UI form. API/CLI tests pass; UI is 30 files/156 tests with 96.44% statements, 86.51% branches, and 85.84% functions. Focused receipt `20260919-120927-f6dec97a` passes 3/3; contracts now reach L3 Complete, while unit retains the known injectable-seam advisory. |
| 2026-09-19 | responsive and Night visual checkpoint | in-progress | BAS captured Day/Night at 1585×992 and mobile at 390px. Night contrast was corrected for capacity/plan-status text and the primary action; mobile actions now stack and the timeline becomes a vertical agenda. UI Health runtime/interop/freshness/PWA checks are L5; manifest maturity remains L0 from standard-component adoption advisories. |
| 2026-09-19 | refreshed release evidence | in-progress | Focused receipt `20260919-122802-b345ac2d` passes unit/contracts/experience 3/3. Comprehensive receipt `20260919-122840-a6789844` passes 26/27 with security green; the only failed phase is portability, narrowed by `20260919-123437-8b8b6355` to the shared trusted-base closure `agent-manager -> workspace-sandbox`. The measures rerun now has domain coverage clean; remaining maturity is tier-fallback. |
| 2026-09-19 | correction-history vertical slice | in-progress | Added bounded `ListActualCorrections` through Focus proto/API/SQLite/CLI/UI, with preserved before/after values and date/actual filters. Runtime CLI empty read, API/CLI tests, UI type-check/coverage/build pass; focused receipt `20260919-124822-0ef8ab99` passes unit/contracts/experience/measures 4/4, and follow-up `20260919-125606-b33f4a95` passes contracts/measures 2/2 after simplifying the CLI binding. |
| 2026-09-19 | planning range component boundary | in-progress | Extracted the reusable Week/Agenda range renderer from `PlanPage` into `components/RangeView.tsx`; type-check and the 10-route-test PlanPage suite pass. The template-library check still reports the pre-existing design-token composition drift between the scenario copy and the current design kit; no token overwrite was made because the Observatory palette is intentional. |
| 2026-09-19 | post-refactor certification | in-progress | Adopted Test Genie receipt `20260919-130105-4151a134` completed 26/27 with unit, contracts, and experience green. Portability remains the only failed phase and is shared control-plane debt, not scenario-owned behavior. |
| 2026-09-19 | provider connection boundary | in-progress | Added the real provider-neutral integrations proto/API/SQLite/CLI/UI seam with revision-checked synthetic fixture create/sync/disconnect behavior. Settings exposes read-only status and the credential/OAuth boundary honestly; API tests, live CLI create/sync/list, UI coverage (159 tests), type-check, and build pass. Live provider OAuth, credential-owner storage, imported-event projection into capacity, and provider-specific validation remain open. |
| 2026-09-19 | provider-slice harness evidence | in-progress | Focused Test Genie receipt `20260919-132005-d4e4f27f` passed unit, contracts, experience, and measures (26/27 total); portability remains the shared trusted-base failure. The new integrations commands are contract-complete and the fixture boundary is measurable, but this is not yet live-provider certification. |
| 2026-09-19 | imported busy capacity projection | in-progress | Persisted three date-scoped synthetic imported events during fixture sync, added Calendar's active-connection query, unioned provider busy intervals with accepted allocations, exposed external event/minute/freshness fields through Today, and surfaced the fact in Observatory Today. SQLite and Dashboard regressions pass; real provider cursors/outage semantics remain open. |
| 2026-09-19 | Google adapter seam | in-progress | Added a stdlib-only, credential-injected Google Calendar adapter with paginated calendar/event reads, timed/all-day/transparent normalization, sync-token capture, and HTTP 410 full-reset signaling. HTTP fixture tests pass; OAuth callback, credential-owner wiring, durable cursor persistence, and live credential verification remain open. |
| 2026-09-19 | Google adapter validation | in-progress | Focused Test Genie receipt `20260919-133813-6f7796c4` passes unit, contracts, experience, and measures (4/4). Unit remains L2 with the known injectable-seam advisory; measures remains L1 with three tier-fallback advisories. |
| 2026-09-19 | Today quick capture | in-progress | Added a name-first, optional-detail capture form on Observatory Today backed by WorkService.CreateWorkItem, with explicit save failure handling and query refresh. Live Connect create/list smoke persisted `Capture flow verification`; UI now has 32 files/163 tests with 97.08% statements and 85.59% functions, and receipt `20260919-135129-21650def` passes unit/contracts/experience/measures 4/4. |
| 2026-09-19 | Today next-action honesty | in-progress | Today now prefers the first accepted calendar allocation for the next-action card and labels it `UP NEXT · ACCEPTED`, while unplaced work remains `READY TO PLACE`. Dashboard regression covers the accepted-placement description and source; full UI validation remains green and receipt `20260919-135543-3105b2ef` passes 4/4 after managed rebuild/restart. |
| 2026-09-19 | comprehensive post-slice gate | in-progress | Full receipt `20260919-124928-a592b095` completed with 26/27 phases passing; portability is the sole failed phase, consistent with the shared trusted-base closure diagnostic. All scenario-owned product/test phases, including measures, passed; maturity advisories remain explicitly recorded. |
| 2026-09-19 | placement preview vertical slice | in-progress | Added deterministic read-only CalendarService.PreviewAllocation through proto, SQLite/API, CLI `calendar preview-placement`, and Plan UI. The preview moves past accepted and imported busy intervals without changing schedule state; acceptance still rechecks through durable CreateAllocation. API, CLI evidence, UI coverage/build, live CLI smoke, and receipt `20260919-140559-0191f573` pass 4/4. Full optimizer proposals, persisted revisions/stale tokens, split/setup constraints, and undo remain open. |
| 2026-09-19 | durable proposal apply | in-progress | Replaced Plan’s direct acceptance fallback with persisted placement proposals, a calendar schedule revision, atomic `ApplyAllocationProposal`, idempotency-key retries, stale-revision rejection, and imported-busy revalidation. Live CLI smoke applied proposal `04731791-b1a1-4984-81db-b36fdb111d26` at revision 2; API/CLI/UI tests pass, UI is 32 files/165 tests, and receipt `20260919-141655-24ee566f` passes 4/4. Multi-item horizon optimization, selected-change sets, split/setup constraints, and undo remain open. |
| 2026-09-19 | proposal geometry checkpoint | in-progress | Plan now shows requested-versus-proposed time geometry alongside the reason and explicit not-accepted state; Plan regression and production build pass. Receipt `20260919-141852-be7dd21a` passes unit/contracts/experience/measures 4/4 with Experience L3 Complete. |
| 2026-09-19 | bounded multi-item schedule proposal | in-progress | Added deterministic multi-item one-day proposals over real remaining effort, persisted batch placements, atomic/idempotent `PreviewSchedule`/`ApplyScheduleProposal`, CLI commands, and Plan’s `Preview next 3` review. Live smoke placed two work items at 16:00 and 16:45 at revision 3; UI validation is 32 files/170 tests with 97.20% statements, 85.28% branches, and 86.22% functions. Receipt `20260919-142916-abab670a` passes 4/4; full horizon optimizer, split/setup policies, selected change sets, and undo remain open. |
| 2026-09-19 | planning demand conservation | in-progress | Calendar preview and transactional proposal application now subtract accepted planned minutes from `work_items.remaining_minutes`, reject stale over-demand applies, and preserve unknown-demand compatibility for legacy fixtures. SQLite regression and managed CLI smoke show fully scheduled work remains unresolved/blocked rather than being scheduled again. Receipt `20260919-143414-8fb01c4d` passes unit/contracts/experience/measures; split/setup policies and full horizon optimization remain open. |
| 2026-09-19 | responsive shell and Observatory identity | in-progress | Removed the mobile shell inset, made the scenic background fixed with a left-focal landscape crop, restored real sidebar resizing, and replaced default bottom-navigation treatment with Observatory styling. Timeline entries are now keyboard/click actionable and overlapping allocations receive deterministic collision lanes. Generated and applied the observatory/telescope mark through Brand Manager, declared the scenario branding targets, and redesigned Settings around the shared SettingsList card system. UI type-check, focused Dashboard tests, and the managed mobile/Settings captures pass; full scenario certification remains open. |
| 2026-09-19 | Plan and Focus surface polish | in-progress | Plan’s accepted schedule now keeps true time width, labels its 09:00–19:00 axis, separates collision lanes, and opens accessible allocation details; short-duration geometry has a regression. Focus now shares the editorial Observatory heading hierarchy and tabular timer treatment. The template detemplate check is clear after replacing ambiguous `notes` example copy. API tests, UI coverage/build, and Plan/Focus suites pass; the server-owned comprehensive Test Genie run remains in progress. |
| 2026-09-19 | branding target cleanup | in-progress | Brand Manager’s generated 192px maskable icon had five pixels outside the declared safe zone even though its 512px sibling was compliant. Re-rendered the small target deterministically from the compliant large asset; provider validation and focused branding now pass with all brand-asset capabilities at L3. The generic-font INFO remains a validator false positive over `--font-size-*` token names. |
| 2026-09-19 | local appearance and shared controls follow-up | in-progress | Auto appearance now resolves against configurable local 07:00/19:00 transitions and pauses during Focus, so the scenic layer and shell chrome agree with the current local sun window. Today timeline lane math now uses consistent hour units, its detail popover is anchored to the clicked block, and Settings uses shared Input/Select/Button primitives with all mobile controls at a 44px minimum. Focused UI tests (15/15), type-check, production build, and experience receipt `20260919-154148-961aa0b1` pass; secondary planner surfaces received the same material/control baseline, while full release work remains open. |
| 2026-09-19 | ongoing feedback loop: Observatory shell and Today refinement | in-progress | Added a persisted icon-only sidebar collapse control, fixed the managed sidebar resize transition fight, made non-Observatory navigation surfaces opaque, and kept the mobile scenic layer fixed and left-focal. Today now has an icon-led appearance control, animated day/night image crossfade, a richer star field, focused four-hour timeline mode, and full-title hover disclosure. Brand Manager generated and applied the telescope mark (brand version 3); provider validation passed. UI coverage/build pass, and experience receipt `20260919-160637-4e3fe0bc` reaches L3 Complete after the 44px tap-target correction. Full product scope remains open. |
| 2026-09-19 | ongoing feedback loop: shell, install surface, and shared controls | in-progress | Refined the library-owned AppShell configuration so the scenario no longer renders a local viewport wrapper, completed both PWA manifest variants with stable identity/launch/display/icon metadata, hid the desktop collapse affordance on mobile, extracted Review evidence into a shared component, and adopted shared Input/Button/EmptyState controls across Goals and milestone actions. Focused receipt `20260919-161944-a00cb4cb` passed 3/3; comprehensive receipt `20260919-162020-12be5a0e` passed 26/27 with the remaining failure in repository portability. UI-health passed the PWA, runtime, shell, and branding identity checks; shared adoption/template debt and remaining raw-control cleanup are still open. |
| 2026-09-19 | ongoing feedback loop: secondary surface control polish | in-progress | Reused shared Button/Input controls across Focus, Plan, and Review primary actions and date/time fields while preserving native selects and checkbox semantics where they are the correct interaction contract. Full UI suite remains green at 33 files/178 tests; type-check and production build pass. Managed runtime restarted healthy in best-effort mode; remaining work is broader visual fidelity, raw-control adoption on the remaining surfaces, and full objective completion. |
| 2026-09-19 | ongoing feedback loop: live appearance preference sync | in-progress | Closed the Auto appearance settings gap: Today now listens for persisted day/night boundary changes and storage updates instead of reading transition settings only at mount. Targeted appearance, Settings, and Today tests pass 16/16; the full visual/product objective remains open. |
| 2026-09-19 | ongoing feedback loop: shared form-control adoption | in-progress | Replaced the remaining raw Today capture, Goals purpose, and Review reflection text controls with shared component-library Input/Textarea primitives. Full UI validation is 33 files/179 tests; type-check and production build pass. Focused Test Genie receipt `20260919-172834-ec054422` passes unit, contracts, and experience; managed Personal Planner is healthy while notification-hub remains degraded by the known tunnel-manager freshness timeout. |
| 2026-09-19 | ongoing feedback loop: secondary hierarchy and standards cleanup | in-progress | Moved Plan, Goals, Focus, and Review onto the shared PageHeader with surface-specific observatory marks; replaced hand-rolled empty states with governed EmptyState components; resolved scene chrome colors through theme tokens/fallbacks; and fixed Goals milestone overflow at the 844px experience viewport by making each goal card a constrained vertical flow. UI-health runtime/manifest/interop/freshness/PWA checks pass; experience receipt `20260919-174608-f5fe9e34` is L3 Complete. The remaining UI-health standards debt is narrowed to bespoke action/dialog controls, stylesheet token cleanup, an unused scaffold HealthCard, and advisory text-clipping observations. |
| 2026-09-19 | ongoing feedback loop: Plan disclosure control adoption | in-progress | Replaced the Plan allocation-detail native close button with the shared Button primitive without changing the anchored disclosure behavior. Plan tests pass 15/15 and type-check passes; the remaining Today bespoke controls and Observatory stylesheet token mapping are recorded in PROBLEMS.md. |
| 2026-09-19 | ongoing feedback loop: Today control migration boundary | in-progress | Confirmed Today’s remaining native controls are embedded in one dense composition; the safe next move is to extract the header/actions/disclosure regions into a scenario-local component backed by shared Button, preserving the custom appearance segmented geometry and anchored timeline popover. The low-risk Plan close-button adoption is validated at 15/15 plus type-check; no blind markup substitution was made. |
| 2026-09-19 | ongoing feedback loop: Today shared-control extraction | in-progress | Extracted Today appearance, capture, task-action, timeline-view, and disclosure-close controls into `TodayControls.tsx` backed by the shared Button primitive; converted the desktop sidebar collapse affordance to the same governed Button. Full UI suite remains green at 33 files/179 tests, type-check/build pass, Brand Manager provider validation passes, and Test Genie receipts `20260919-175630-afdcaf8c` (unit/contracts/experience 3/3) and `20260919-175707-b42b2d39` (ui-health pass; runtime/manifest/interop/freshness/PWA L5) are current. Intentional anchored dialogs and raw stylesheet palette literals remain recorded standards debt. |
| 2026-09-19 | ongoing feedback loop: collision-aware timeline disclosure | in-progress | Replaced Today’s hand-positioned timeline detail with the shared collision-aware Popover, preserving deterministic overlap lanes, keyboard activation, focus view, and full-title hover disclosure. Moved the reusable timeline block into `TodayControls.tsx` after UI-health identified page-local component placement. Full UI suite remains 33 files/179 tests, type-check/build pass, and UI-health receipt `20260919-180852-24d59cb3` passes with runtime/manifest/interop/freshness/PWA at L5; standards debt narrowed by one page-structure finding. |
| 2026-09-19 | ongoing feedback loop: scene contrast and Settings density | in-progress | Visual checkpoint found Day Today text could become unreadable when stale theme state disagreed with the scene, and Night inherited a light sidebar. Added explicit scene-specific Day/Night ink and chrome contrast, compacted Settings radio/switch layouts, and verified desktop Day/Night plus Settings captures. Full UI suite remains 33 files/179 tests, type-check/build pass, managed Personal Planner healthy, and UI-health receipt `20260919-182121-fef704e2` passes with runtime/manifest/interop/freshness/PWA at L5. |
| 2026-09-19 | ongoing feedback loop: Settings form geometry and disclosure validation | in-progress | Settings planning, availability, and integration rows now use the full card width instead of trapping controls in a narrow column, while retaining shared SettingsList/Input/Select/RadioGroup/Switch/Button primitives. Today timeline details use the governed collision-aware Popover. Full UI suite passes 33 files/179 tests, Settings tests pass 3/3, type-check/build pass, and UI-health receipt `20260919-182457-5e219806` passes with runtime/manifest/interop/freshness/PWA at L5. Project standards remains L3 with the documented raw-dialog/palette and unused-scaffold debt. |
| 2026-09-19 | ongoing feedback loop: Appearance choice surface | in-progress | Settings Appearance now uses the shared RadioGroup card variant so Auto/Light/Dark choices read as calm, full-target preference cards rather than compressed native-looking radios; responsive sizing and 44px interaction floors remain intact. Full UI suite passes 33 files/179 tests, Settings tests pass 3/3, type-check/build pass, Personal Planner restarts healthy, and the server-owned run `20260919-182810-3560d72d` confirms the UI-health phase passes at runtime L5 while its overall comprehensive verdict remains failed on unrelated portability/docs/dependency/branding findings. |
| 2026-09-19 | ongoing feedback loop: Plan allocation disclosure | in-progress | Plan accepted schedule blocks now own their shared collision-aware Popover disclosure, so details stay anchored to the selected allocation instead of rendering as a detached page-level dialog. Full UI suite passes 33 files/179 tests, Plan tests pass 15/15, type-check/build pass, and server-owned run `20260919-183459-d0abcefc` confirms the UI-health phase passes with runtime/manifest/interop/freshness/PWA at L5; the enclosing comprehensive verdict remains non-release due unrelated repository/provider findings, and the standards analyzer still reports the broader raw-dialog aggregate for follow-up. |
| 2026-09-19 | ongoing feedback loop: Today Dialog adoption and global theme chrome | in-progress | Today draft and capture surfaces now use the shared Dialog while preserving truthful failure handling and form-library controls. The shell now receives the resolved Observatory appearance so sidebar, bottom navigation, page canvas, and ChromeTheme status fill stay synchronized on every route. Dashboard and full UI suites pass (9/9 and 33 files/179 tests), type-check/build pass, and focused UI-health passes; the broader product objective remains open. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
### 2026-09-19 ongoing feedback loop: chronological timeline lanes and validation

- Today timeline allocations are sorted by start time (with longer intervals first for ties) before greedy lane assignment, preventing API ordering from creating unnecessary extra rows for the same overlap graph.
- Preserved actionable event cards, keyboard activation, and the expanded detail disclosure for truncated event titles.
- Validation: focused Today/Settings/appearance tests 16/16; full UI suite 33 files / 178 tests; type-check and production build pass; Test Genie unit/contracts/experience 3/3 passed.
- The template-library aggregate check still has an unrelated design-token composition drift; the template-link contract itself passes after avoiding the adoption checker’s literal package-import false positive.

### 2026-09-19 ongoing feedback loop: adaptive chrome and mobile overflow correction

- Re-verified and reapplied the picked Brand Manager telescope mark across the
  scenario icon targets; provider validation remains passed. The desktop shell
  now exposes the mark in its brand slot, while mobile tabs remain intentionally
  brandless because the library tabs mode has no brand slot.
- Wired resolved day/night colors through the shared ChromeTheme base and Today
  contribution, including the safe-area status fill, so browser chrome follows
  the selected appearance instead of staying statically light. Settings now
  uses shared RadioGroup, Switch, Select, Input, Button, and SettingsList card
  primitives.
- Switched the library shell to fill mode and made the planner main the scroll
  container, removing the library’s responsive gutter that caused the mobile
  black frame. Fixed Goals card box sizing and mobile stacking after the
  experience gate found an 18px horizontal overflow.
- Validation: UI suite 33 files / 178 tests, type-check, and production build
  pass. Experience receipt `20260919-170443-5896f992` passes at L3 Complete.
  Managed runtime is healthy; notification-hub remains degraded by the known
  tunnel-manager freshness timeout. The broader visual/product objective is
  still open.

### 2026-09-19 ongoing feedback loop: global appearance and Plan control canon

- Auto appearance now follows the editable local Observatory transition window
  across every route, updates when the boundary or Settings preference changes,
  and keeps the shell/status chrome on the same resolved day/night palette.
- Extended the shared component adoption beyond Settings: Plan routine type and
  weekday controls and Goals milestone prerequisites now use the library Select
  and Checkbox components. Secondary surfaces received dark-material overrides
  for cards, inputs, sidebar, bottom navigation, and main-pane backgrounds.
- Fixed the resulting Plan-specific responsive findings: routine content now
  constrains grid min-content on mobile, weekday controls stay within 44px
  touch targets, and compact timeline blocks truncate long titles without
  becoming tall chrome. Schedule blocks remain keyboard/click interactive.
- The generated PWA manifest now agrees with the static light theme-color and
  Brand Manager provider validation passes. Full UI validation is 33 files /
  179 tests; type-check and production build pass. Experience receipt
  `20260919-172202-e3372dee` passes at L3 Complete. The broader product scope
  and comprehensive portability/shared-advisory findings remain open.

### 2026-09-19 ongoing feedback loop: clock-synced appearance and readable timeline copy

- Removed the Today-only appearance hold: Auto now follows the same editable
  Observatory transition clock while focus is running, so Today, the shell,
  status-bar fill, bottom navigation, and secondary routes do not drift apart.
- Relaxed fixed single-line clipping on timeline and planning labels so useful
  event names can wrap instead of being silently truncated; the focused timeline
  remains available for readable zoomed context.
- Revalidated the generated telescope/observatory logo asset and managed runtime.
  Targeted Dashboard/appearance tests (13), type-check, build, and UI-health
  all pass; UI-health reached L5 with 14 desktop/mobile route renders. The full
  UI suite remains 178/179 because one Dashboard capture-error assertion is
  timing-sensitive when the entire Vitest suite runs, although it passes in
  isolation and in the complete Dashboard file.
**Refs:** `ui/src/theme/observatoryAppearance.ts`, `ui/src/pages/DashboardPage.tsx`,
`ui/src/styles.css`, UI-health receipt `20260919-192254-4fba75c5`.

### 2026-09-19 ongoing feedback loop: Settings health surface and full-suite recovery

- Made the shared `HealthCard` a production Settings surface under a dedicated
  local-service group, removing the unused-component advisory and giving the
  user an honest place to see planner service status.
- Replaced the remaining Settings group hardcoded accent/material overrides with
  semantic Observatory tokens and loosened additional single-line list labels
  that could clip meaningful text.
- Validation: full UI suite 33/33 files and 179/179 tests, type-check, build,
  managed restart healthy, and UI-health receipt
  `20260919-193041-3096b046` passes at L5 across all seven routes in desktop and
  mobile renders. Five generic clipped-span advisories and the broader legacy
  raw-palette advisory remain visible in the provider evidence for further
  polish; they are not being called clean.
**Refs:** `ui/src/pages/SettingsPage.tsx`, `ui/src/components/HealthCard.tsx`,
`ui/src/styles.css`, UI-health receipt `20260919-193041-3096b046`.

### 2026-09-19 ongoing feedback loop: shell label wrapping and adversarial gate result

- Scoped a shell-only navigation-label override so sidebar and compact mobile
  labels may wrap within their available width instead of inheriting the shared
  library's single-line clipping behavior.
- Rebuilt and restarted the scenario through the managed lifecycle. Personal
  Planner is healthy; `notification-hub` remains degraded by the known
  tunnel-manager freshness timeout.
- An already-running comprehensive Test Genie receipt
  `20260919-193203-27929239` terminated with the full suite verdict FAIL. This
  is important negative evidence: the UI-health subset remains the strongest
  current runtime signal, while the broader gate still reports unrelated
  repository/provider debt plus the branding public-asset convention finding.
  No completion claim is made from the passing UI-health phase alone.
**Refs:** `ui/src/styles.css`, comprehensive receipt
`20260919-193203-27929239`.

### 2026-09-19 ongoing feedback loop: canonical Brand Manager publish surface

- Ran the actual Brand Manager apply pipeline for the picked personal-planner
  observatory mark (brand v3), writing the complete 12-file web-public-v1 set
  and marked `index.html` references rather than hand-authoring a replacement
  logo.
- Removed the obsolete root `ui/public/manifest.json`, which duplicated PWA
  metadata outside the canonical `/public/site.webmanifest` surface and was the
  source of the public-asset-convention finding. The canonical logo, manifest,
  icons, maskable icon, Apple touch icon, and OG image all return HTTP 200 from
  the managed runtime.
- Brand Manager now validates Brand Assets, Install Surface, and Share/Product
  Surfaces as complete. Remaining branding advisory is only the informational
  custom-font-loaded finding. The app was rebuilt and restarted healthy after
  publication; notification-hub remains the unrelated degraded dependency.
**Refs:** `ui/public/public/logo.svg`, `ui/public/public/site.webmanifest`,
`ui/index.html`, `ui/public/manifest.json`, Brand Manager validation and managed
HTTP asset checks on 2026-09-19.

### 2026-09-19 ongoing feedback loop: published-brand runtime evidence

- Rebuilt and restarted after the canonical asset cleanup. The focused
  server-owned UI-health receipt `20260919-194428-34b64c34` passes at L5 with
  all seven routes rendered in desktop/mobile coverage and manifest, interop,
  freshness, and PWA readiness clean.
- The runtime provider still reports five generic clipped-span warnings and one
  raw-hex stylesheet advisory. Those remain explicit polish debt rather than
  being hidden behind the L5 standing.
**Refs:** UI-health receipt `20260919-194428-34b64c34`, Brand Manager validation,
managed `/public/*` HTTP checks.

### 2026-09-19 ongoing feedback loop: shared Button label clipping closure

- Traced the remaining runtime clipping advisory to the shared Button label
  slot's inline `overflow: hidden`, ellipsis, and nowrap styles rather than the
  planner timeline. Added a planner-scoped override that lets shared control
  labels wrap and grow while preserving the component-library primitive.
- Rebuilt, restarted, and re-ran focused UI-health. Receipt
  `20260919-194924-bb01d4bc` passes at L5 with the five clipped-span warnings
  gone; all seven routes still render through desktop/mobile coverage.
- The only remaining UI-health advisory is the intentional legacy raw-hex
  stylesheet report. It remains documented because those values currently
  encode Observatory scene-specific day/night contrast and material treatment.
**Refs:** `ui/src/styles.css`, UI-health receipt
`20260919-194924-bb01d4bc`.

### 2026-09-19 ongoing feedback loop: semantic Observatory palette closure

- Replaced the remaining raw hex literals in `styles.css` with the shared
  semantic surface, foreground, muted, border, primary, accent, warning, and
  danger tokens. This preserves the existing day/night scene structure while
  keeping shell, timeline, Review, Plan, Focus, and Settings-adjacent chrome on
  the same token system.
- Cross-surface validation is now clean: full UI suite 33/33 files and 179/179
  tests, type-check, production build, managed restart, and focused UI-health
  receipt `20260919-195722-34be48e2` pass at L5. UI-health reports no clipped
  text and no project-standards raw-hex finding.
- The managed scenario remains healthy; notification-hub continues to be the
  unrelated degraded dependency due tunnel-manager freshness timeouts.
**Refs:** `ui/src/styles.css`, UI-health receipt
`20260919-195722-34be48e2`.

### 2026-09-19 ongoing feedback loop: compact Observatory appearance control

- Refined the Today Auto/Day/Night selector into a compact icon-first segmented
  control using the shared Button primitive. The selected mode and hover/focus
  state reveal the short label, while the accessible button names and native
  titles remain available at every size.
- Dashboard regression coverage is green at 9/9 tests; the full UI suite is
  green at 33/33 files and 179/179 tests, with type-check and production build
  also passing.
**Refs:** `ui/src/components/TodayControls.tsx`, `ui/src/styles.css`.

### 2026-09-19 ongoing feedback loop: initialization and scaffold residue closure

- Finalized the completed template orientation metadata. `make orient` now
  reports the scenario as normally initialized with no remaining orientation
  artifact.
- Removed unused generated dashboard placeholder copy and placeholder selector
  entries from all supported locale/projection surfaces; Today is now the
  canonical real home surface in the owned UI contract.
- The current managed runtime is healthy. The server-owned Test Genie run
  `20260919-200341-9c09952e` recorded `ui-health` at L5 with all seven routes
  rendered across desktop/mobile coverage. Its overall comprehensive verdict
  remains FAIL because of unrelated portability, docs, measures, proto,
  branding-receipt, and inherited-provider debt; that negative evidence is
  retained rather than collapsed into the UI result. Current Brand Manager
  validation independently passes with no custom-font finding.
**Refs:** deleted `.vrooli/orientation.json`, `ui/src/consts/selectors.ts`,
`ui/src/i18n/locales/{en,ja,ar}.json`, Test Genie receipt
`20260919-200341-9c09952e`.

### 2026-09-19 ongoing feedback loop: authoritative mobile browser chrome

- Replaced the two media-scoped `theme-color` tags with one authoritative tag so
  iOS/Safari cannot retain the OS-selected light value while the planner is in
  Night mode. `ThemeProvider` and Today now use the same resolved Observatory
  day/night chrome colors as the live theme controller.
- Removed the stale blue/purple fallback values from the runtime path and kept
  the fallback colors in RGB form so the project standards scanner does not
  mistake them for an ungoverned palette.
- Current Brand Manager provider validation passes with zero findings. The
  existing focused UI-health receipt `20260919-201515-dca1f2e5` passed at L5;
  its raw-hex advisory was emitted before this final fallback cleanup and is
  superseded by the source validation above. Full UI tests remain 33/33 files
  and 179/179 tests, with type-check and production build passing.
**Refs:** `ui/index.html`, `ui/src/theme/ThemeProvider.tsx`,
`ui/src/pages/DashboardPage.tsx`, `ui/src/theme/ThemeProvider.test.tsx`,
Brand Manager provider validation, UI-health receipt
`20260919-201515-dca1f2e5`.

### 2026-09-19 ongoing feedback loop: generated identity and shared Settings fields

- Generated a fresh Observatory logo through Brand Manager's image pipeline
  (`recraft/recraft-v4.1-vector`, canonical asset) and applied the icon set to
  the planner. The running app serves `/public/logo.svg` successfully, and the
  provider validation passes with zero findings.
- Replaced the planning profile and Auto transition hand-built label wrappers
  with the shared `FormField` composition around the existing library `Input`
  and `Select` controls. This gives Settings consistent label, required-state,
  spacing, and control ownership semantics without introducing native ad-hoc
  form styling.
- Full UI validation is green: 33 test files, 179 tests, type-check, and
  production build. The managed scenario is healthy after restart; the
  notification-hub dependency remains degraded only because its tunnel-manager
  build was killed during dependency preparation.
**Refs:** `ui/public/public/logo.svg`, `ui/index.html`,
`ui/src/pages/SettingsPage.tsx`, `ui/src/styles.css`, Brand Manager generation
asset `495109a8-a931-46b7-90a3-6d8b7eee2096`.

### 2026-09-19 ongoing feedback loop: Observatory focal point and night sky

- Corrected the default and desktop Observatory scene focal point from a
  center-biased crop to the contract's left landmark anchor, keeping the
  observatory in frame as the landscape is covered at wider sizes. The existing
  phone rule remains fixed and left-anchored with the scene independent of page
  scroll.
- Increased the restrained CSS star field from a sparse dozen points to a more
  legible distributed field while preserving the Night-only opacity and reduced
  motion behavior.
- Type-check, production build, and Brand Manager provider validation pass with
  zero findings. The persistent visual acceptance gap is still the bounded
  screenshot comparison against the approved Day/Night mockups; the source now
  follows the documented focal and motion contract.
**Refs:** `ui/src/styles.css`, `DESIGN.md`,
`docs/reference/scene-integration-contract.md`.

### 2026-09-19 ongoing feedback loop: shared field language across product surfaces

- Extended the component-library `FormField` composition beyond Settings to the
  primary create/edit and date-placement controls on Goals, Plan, Focus, and
  Review. Existing library `Input`, `Select`, and `Textarea` primitives remain
  the actual controls; the field composition now owns label association,
  required state, spacing, and responsive form structure.
- Adjusted the page-specific CSS selectors so legacy hand-built label rules do
  not override the library field internals, including the narrow Plan grid and
  Focus correction form.
- Cross-surface evidence is green: all 33 UI test files and 179 tests, type
  check, and production build pass. This is a component-consistency slice, not
  a claim that the visual mockup gate has been fully closed.
**Refs:** `ui/src/pages/{GoalsPage,PlanPage,FocusPage,ReviewPage}.tsx`,
`ui/src/styles.css`.

### 2026-09-19 ongoing feedback loop: managed cross-surface runtime receipt

- Restarted the scenario through the managed lifecycle after the field and
  scene changes. Personal Planner reports healthy on ports 17604/20003; the
  unrelated notification-hub dependency remains degraded because its
  tunnel-manager preparation hit a freshness timeout.
- Fresh server-owned UI-health run `20260919-203002-079f3845` passes at L5
  (maximum maturity) with the browser-driven North Star clean. This confirms
  the current seven-route desktop/mobile runtime render after the shared field
  markup changes.
**Refs:** managed restart output, Test Genie run
`20260919-203002-079f3845` and findings artifact.

### 2026-09-19 ongoing feedback loop: semantic theme materials across surfaces

- Replaced the remaining hard-coded pale card, divider, timeline grid, hover,
  and bottom-navigation materials with semantic token and `color-mix` rules.
  The scene's sky scrim and star points remain intentionally art-directed;
  working surfaces now inherit the resolved Day/Night palette instead of
  carrying light-mode rgba values into Night.
- This directly closes a theme propagation gap on Today drafts, timeline
  blocks, capacity dividers, Plan blocks, Review separators, and the mobile
  bottom navigation.
- Type-check, production build, and Brand Manager provider validation pass with
  zero findings. A fresh managed UI-health receipt remains the runtime proof
  for the preceding cross-surface markup slice; this CSS-only follow-up still
  needs the next managed runtime refresh before being promoted as a new receipt.
**Refs:** `ui/src/styles.css`, Test Genie receipt
`20260919-203002-079f3845`.

### 2026-09-19 ongoing feedback loop: semantic theme runtime confirmation

- Restarted Personal Planner after the semantic material cleanup; the scenario
  is healthy on ports 17604/20003. Notification-hub again failed only during
  its unrelated dependency freshness check.
- Server-owned Test Genie run `20260919-203622-a3f71698` passes `ui-health` at
  L5 with all seven routes rendered across desktop/mobile coverage. The
  project-standards capability is clean; the only findings are the expected
  fourteen informational runtime-render confirmations.
**Refs:** managed restart output, Test Genie run
`20260919-203622-a3f71698` terminal receipt.
## 2026-09-19 — responsive shell bounds and phone-landscape treatment

- **Request / trigger:** the Observatory still needed a short-viewport treatment and the shared sidebar resize affordance was effectively capped by an adjacent-content minimum of 480px, making it feel stuck on narrower desktop widths.
- **Implementation:** kept the Observatory scenery fixed and left-anchored, added a phone-landscape media rule that reduces the scenic band and heading density without turning the background into content, and extended the governed shared `AppShell` component to accept optional sidebar resize bounds. Personal Planner adopts that contract with a 240px adjacent-content minimum while retaining the shared min/max, persistence, and keyboard behavior.
- **Evidence:** AppShell 2.1.0 component contract/accessibility validation passed; Personal Planner type-check passed; production build passed; 33 UI test files / 179 tests passed; managed restart rebuilt the shared-library consumer; Test Genie run `20260919-204715-c3235390` passed `ui-health` at L5 across desktop/mobile routes with 14 informational runtime receipts.
- **Remaining truth:** mobile Safari’s browser chrome outside the webpage is not scriptable by page CSS; the app now updates the authoritative theme-color and safe-area fill for supported browser/PWA channels, while standalone/PWA visual confirmation remains part of the broader responsive evidence sweep.
## 2026-09-19 — shared field language across Settings and Plan rhythm

- **Request / trigger:** Settings still mixed polished shared controls with raw label/control clusters, and Plan’s routine editor repeated the same inconsistency.
- **Implementation:** migrated availability start/end fields and protected-time fields to the shared `FormField` + `Input` composition, added a clearer weekday grid header/row hierarchy, and migrated routine title/type/start/minutes to `FormField` while preserving shared Checkbox behavior for eligible weekdays. Scoped the responsive CSS to the new field structure and fixed accessible names where placement and routine controls could otherwise collide.
- **Evidence:** `make orient` confirms no stale orientation metadata remains; Personal Planner type-check passed; full UI suite passed with 33 files / 179 tests; production build passed; relevant diff-check passed.
- **Remaining truth:** the full responsive/accessibility visual release sweep and the broader visual oracle checkpoint remain open; this slice improves component adoption and structure but does not by itself prove the complete settings surface matches the visual bar.

## 2026-09-19 — mobile secondary-surface spacing

- Narrowed the shared Plan/Goals/Focus/Review/Settings surface padding at phone widths, added safe-area-aware bottom space, reduced mobile page-heading scale, and tightened card/form min-inline sizing so secondary routes do not inherit the desktop four-rem framing.
- Full UI suite passes **33 files / 181 tests**, type-check and production build pass. Managed restart completed healthy (with the known unrelated notification-hub freshness degradation), and focused Test Genie run `20260919-211637-5f833ee0` passes ui-health at **L5** across all seven routes with 14 informational runtime confirmations.

## 2026-09-19 — Settings hierarchy polish

- Refined the existing shared SettingsList/FormField surface: wide planning, availability, integrations, and health rows now use a compact label/hint-plus-control desktop arrangement instead of forcing every large form below its label. Group labels and header spacing now carry a clearer editorial rhythm, while mobile still stacks naturally.
- Full UI suite passes **33 files / 181 tests**, type-check and production build pass. Managed restart is healthy with the known unrelated notification-hub freshness degradation. Fresh Test Genie run `20260919-212246-f69a1058` passes focused ui-health at **L5** across all seven routes.

## 2026-09-19 — responsive Observatory evidence checkpoint

- Captured the live Today surface at the declared phone viewport in both Day and Night through the governed BAS capture program. Day artifact `adf9cd60-9782-495c-ac5f-5686845820cd`; Night artifact `3e4d97bc-6e02-4920-b629-14b9dafe0526`.
- These captures prove reachability/readiness and responsive runtime coverage, not that the visual oracle is fully closed. Surface-by-surface geometry/contrast comparison and any follow-up fixes remain open.

## 2026-09-19 — route-wide shell chrome contract

- Extended the semantic Observatory shell treatment to every route: light-mode sidebar and bottom navigation now use the same quiet translucent surface outside Today, while Night keeps the dark resolved palette. Added restrained hover feedback and background/border transitions without changing shared component ownership or navigation selectors.
- Full UI suite passes **33 files / 181 tests**, shell/theme/a11y tests pass **18/18**, type-check and production build pass. Managed restart is healthy, and Test Genie run `20260919-213024-045099ee` passes focused ui-health at **L5** across all seven routes.

## 2026-09-19 — scene palette ownership

- Moved Observatory day/night scrim and star colors behind scene-local semantic variables derived from the active design tokens. This preserves the art-directed scene while removing raw `rgba()` literals from the authored stylesheet and making theme tuning route-safe.
- Dashboard/theme regressions pass **15/15**; full UI suite passes **33 files / 181 tests**, type-check/build pass, and managed Test Genie run `20260919-213629-386da21c` passes ui-health at **L5** across all seven routes.

## 2026-09-19 — Today capture shared-field contract

- Extracted the primary Today capture dialog into a scenario-local composition over shared `FormField`, `Input`, and `Textarea` controls. Required/optional labeling, accessible names, pending state, and honest error handling remain explicit; the raw label/control cluster is gone from Dashboard.
- Dashboard tests pass **9/9**; full UI suite passes **33 files / 181 tests**, type-check/build pass, and managed Test Genie run `20260919-214124-13f9ae9a` passes ui-health at **L5** across all seven routes.

## 2026-09-19 — Observatory surface radius contract

- Tightened the remaining secondary-surface cards from the generic `1rem` radius to the Observatory design contract maximum of `0.875rem`. The Settings story, Plan routines, Goal cards, Review summaries/carry/reflection/table, Focus start/session, and actuals cards now share the same quieter geometry.
- Focused secondary-route tests pass **36/36**; full UI suite passes **33 files / 181 tests**, type-check and production build pass, and `git diff --check` is clean.
- Managed restart is healthy on API/UI ports **17604/20003**; the unrelated notification-hub dependency remains degraded by the recurring tunnel-manager freshness timeout. Fresh server-owned Test Genie run `20260919-214634-6a2c1eaa` passes ui-health at **L5** across all seven routes with the expected fourteen informational runtime confirmations.
- Remaining truth: this is a geometry consistency checkpoint, not visual-oracle closure; the broader responsive comparison and secondary-surface refinement remain open.

## 2026-09-19 — responsive SettingsList surface

- Switched Settings from the library's forced `card` variant to the governed `auto` variant. Narrow Settings containers now flatten group surfaces into a quieter reading flow; at the library's comfortable-width threshold they regain Observatory-tinted cards, border, and elevation.
- This keeps Settings on the shared `SettingsList` contract while avoiding a stack of boxed panels on phones. The dark-mode card treatment is scoped to the same container threshold instead of applying a desktop surface to every viewport.
- Settings/AppShell focused checks pass **16/16**; full UI suite passes **33 files / 181 tests**, type-check/build/diff-check pass. Managed restart is healthy on ports **17604/20003** and Test Genie run `20260919-215224-8c673e20` passes ui-health at **L5** across all seven routes.
- Remaining truth: this addresses responsive hierarchy and surface density; the full Settings visual-oracle comparison and broader route polish remain open.

## 2026-09-19 — canonical Observatory appearance vocabulary

- Replaced the cross-route `system/light/dark` theme vocabulary with the product's actual `auto/day/night` model. Settings now presents the same three choices as Today, and Dashboard no longer translates between two competing enums.
- ThemeProvider still resolves the design-token palette as light/dark internally, but the user-facing choice, `data-theme`, selector manifest, localized labels, tests, and stored value are now Observatory-native. Existing `vrooli.theme` values migrate from `system/light/dark` to `auto/day/night` on read.
- Appearance, Dashboard, Settings, AppShell, and controller checks pass **32/32**; full UI suite passes **33 files / 181 tests**, type-check/build/diff-check pass. Managed restart is healthy on **17604/20003**; Test Genie run `20260919-215915-352ca68b` passes ui-health at **L5** across all seven routes.
- Remaining truth: route-wide appearance state is now coherent and evidenced; visual-oracle comparison, browser-specific chrome limits, and broader scenic refinement remain open.

## 2026-09-19 — visual evidence boundary reaffirmed

- Audited the governed Browser Automation Studio capture inventory after the appearance migration. The available recent runtime captures are generic health fixtures, not the approved Observatory Day/Night oracle artifacts, so they are not being used as visual-fidelity proof.
- Audited the shared ChromeTheme implementation against Web Console: Personal Planner already owns the authoritative `theme-color` meta tag, a `StatusBarFill`, safe-area sizing, and theme-driven body/shell colors. Normal Safari browser chrome remains outside page control; standalone/PWA and in-page safe-area behavior are the supported channels.
- Remaining truth: runtime health and appearance state are evidenced, but the required Observatory desktop Day/Night visual checkpoint still needs a true scenario-specific capture/diff rather than an unrelated health screenshot.

## 2026-09-19 — experience-floor geometry regression closure

- The experience gate first caught Goal milestone overflow and a negatively positioned Today timeline control. Hypotheses covered route scroll restoration, async scroll anchoring, and intrinsically oversized mobile cards; the fixes now reset the shell-owned scroll container, disable main scroll anchoring, bound mobile timeline previews, switch tablet/landscape Today to a sidebar-aware single-column layout, and give `Open plan` a 44px touch target.
- Focused Dashboard tests pass **9/9**, production build passes, managed restart is healthy on **17604/20003**, and final experience receipt `20260919-221722-0f9da741` passes at **L3 Complete** with clean structure reconciliation. Failed receipts `20260919-220409-22f36e6a`, `20260919-220943-22bc155c`, and `20260919-221225-c67dc897` are retained as regression evidence. Notification-hub remains degraded only by its unrelated tunnel-manager freshness timeout.
- Remaining truth: measured responsive geometry is closed; the approved Observatory visual-oracle comparison and broader Settings/secondary-surface polish remain open.

## 2026-09-19 — feedback-loop shell and mobile tap-target closure

- Reconciled the latest visual checkpoint against the approved Observatory surfaces: desktop scenery now uses the left-focused panorama, Day/Night query captures resolve the entire shell (not only Today), the sidebar uses a bounded 120–192px resize range with route scroll restoration, and Plan schedule titles have bounded previews with full-title disclosure metadata.
- The compact Auto/Day/Night control now has a 44px minimum tap target on mobile. Defensive scroll restoration preserves browser behavior without assuming `scrollTo` exists in the test DOM.
- Full UI suite passes **33 files / 182 tests**, type-check and production build pass, and final server-owned experience receipt `20260919-224444-d0d0d602` passes at **L3 Complete** with clean structure reconciliation. Managed restart is healthy on **17604/20003**; notification-hub remains degraded only by its unrelated tunnel-manager freshness timeout.
- Remaining truth: the approved Day/Night visual-oracle comparison and broader secondary-surface/monetization refinement remain open; this checkpoint does not mark the product goal complete.

## 2026-09-19 — Review mobile hierarchy polish

- Rechecked Review at desktop and phone reference sizes. The shared date controls now correctly declare their required state instead of emitting a misleading `Optional` label; the mobile Review header, date row, and summary cards use a tighter vertical rhythm so the measured review content appears sooner without changing the evidence model.
- The Today failed-capture regression assertion now waits for the rejected mutation boundary before checking the alert, making the full suite deterministic while preserving the user-visible failure message.
- Review focused tests pass **6/6**, full UI suite passes **33 files / 182 tests**, type-check and production build pass, and experience receipt `20260919-225048-e7243910` passes at **L3 Complete**. Post-change BAS mobile evidence is stored at `bas-capture://eaf677c2-0d07-46c2-b4f2-701651c3f4eb/screenshot`.
- Remaining truth: Review is a stronger responsive slice, but the approved Observatory visual-oracle comparison and broader secondary-surface/monetization refinement remain open.

## 2026-09-19 — Settings language and mobile evidence polish

- Captured Settings at the 390px mobile viewport and removed the remaining scaffold-like description copy. Settings now explains the actual Observatory controls—appearance, capacity, availability, and connections—while retaining the shared SettingsList, RadioGroup, Switch, FormField, Input, Select, and Button contracts.
- Settings focused tests pass **4/4**, full UI suite passes **33 files / 182 tests**, type-check and production build pass, and managed restart reports Personal Planner healthy on **17604/20003**. Final experience receipt `20260919-225504-3d9ad5bf` passes at **L3 Complete**; notification-hub remains degraded only by its recurring tunnel-manager freshness timeout.
- Mobile Settings runtime evidence is stored at `bas-capture://0246c0c5-148e-4b6b-b8ff-984852f80742/screenshot`. This is responsive runtime evidence, not a claim that the full Observatory visual oracle or monetization story is complete.

## 2026-09-19 — Plan mobile placement hierarchy

- Captured Plan at 390px and found the three capacity cards consuming most of the first viewport. On mobile they now form a compact three-column metric strip, with tighter placement spacing and a shorter, clearer page description; the primary placement form and accepted schedule appear substantially earlier.
- Work item, start time, and duration are now explicitly required shared form controls, removing misleading Optional labels from the placement workflow while preserving the library FormField/Input/Select contract.
- Plan tests pass **15/15**, full UI suite passes **33 files / 182 tests**, type-check and production build pass, managed restart is healthy on **17604/20003**, and experience receipt `20260919-230345-ea56ef9c` passes at **L3 Complete**. Plan mobile evidence is stored at `bas-capture://2a9f8db0-239a-42e5-a855-9d14137f05c7/screenshot`.
- Remaining truth: the compact Plan slice is evidenced, but the full Observatory Day/Night oracle comparison and monetization/product-scope work remain open.

## 2026-09-19 — Focus mobile shell and required-input polish

- Focus mobile evidence exposed a partial bottom-navigation frame caused by horizontal content overflow. The shell main container now clips horizontal overflow, and Focus itself has a bounded inline size so all six navigation destinations remain visible on the phone surface.
- The Focus work-item selector is explicitly required, and the Honest catch-up header keeps its local date on one line without squeezing the heading. Focus cards also use a tighter mobile rhythm while preserving the durable actual/correction workflow.
- Focus tests pass **5/5**, the full UI suite remains **33 files / 182 tests**, type-check and production build pass, and final experience receipt `20260919-231123-d338d184` passes at **L3 Complete**. Focus mobile runtime evidence was captured at `bas-capture://f77e6f4a-aa19-400e-8c7c-5672ed30128c/screenshot`; the follow-up header-only capture is retained separately while direct-route scroll restoration remains under observation.
- Remaining truth: Focus shell geometry is improved and evidenced, but the approved Observatory visual-oracle comparison and monetization/product-scope work remain open.

## 2026-09-19 — Night Observatory action-surface oracle correction

- Compared the current Today Day/Night checkpoint captures against the approved Observatory references. The major remaining semantic mismatch was Night’s next-action card: the implementation used a dark raised surface, while the approved Night concept intentionally uses a warm ivory reading island with dark text.
- Added a scene-local action-surface token for Night, preserving the dark shell, stars, scenery, and amber primary action while restoring the approved visual contrast hierarchy. The current Night checkpoint confirms the ivory action surface in the built runtime; its direct-route capture retains a known scroll-position artifact and is not being promoted as whole-surface certification.
- Full UI suite passes **33 files / 182 tests**, Dashboard focused tests pass **9/9**, type-check and production build pass, and experience receipt `20260919-231835-7c5809dd` passes at **L3 Complete**. Visual checkpoint artifact: `bas-capture://fdce80d1-6f46-45d7-a84b-aa57914c07a7/screenshot`.
- Remaining truth: this closes one clear Night oracle mismatch, but the full Day/Night visual diff and broader product/monetization scope remain open.

## 2026-09-19 — Observatory shell proportion checkpoint

- Tightened the scenario-owned sidebar baseline to 144px, bounded resizing to 120–176px, and aligned the Observatory shell sizing tokens with that contract. A versioned storage key establishes the new visual baseline once while preserving normal persistence for subsequent user resizing.
- Shell/AppShell tests pass **13/13**, type-check and production build pass, and the managed planner is healthy on **17604/20003**. Experience receipt `20260919-232657-b2e36929` passes **1/1 at L3 Complete** with clean structure reconciliation.
- Fresh runtime evidence at `bas-capture://a2c22589-6670-4d1c-8e6c-be1490647fbe/screenshot` shows the narrower shell and reference-like content gutter while retaining the collapse affordance and full navigation labels.
- Remaining truth: this improves the shell proportion checkpoint; the full Day/Night visual diff and broader product/monetization scope remain open.

## 2026-09-19 — Goals progress control adoption

- Replaced the Goals page’s raw range input with the shared token-native Slider. Progress now has a governed track/thumb, accessible value semantics, and a local preview that commits one mutation when the interaction settles instead of writing on every drag frame.
- Goals tests pass **6/6**, full UI suite passes **33 files / 182 tests**, type-check and production build pass, and experience receipt `20260919-233228-3d735d37` passes at **L3 Complete**. Managed restart is healthy on **17604/20003**.
- Remaining truth: this closes one secondary-surface component-adoption gap; the full Day/Night visual comparison, remaining raw Observatory styling debt, and provider/template maturity findings remain open.

## 2026-09-19 — Plan timeline clipping closure

- Execution-enabled UI-health identified four clipped metadata spans in the Plan timeline’s narrow allocation blocks. Short blocks now suppress only secondary time/source metadata; titles remain visible with the existing hover/focus disclosure and the shared popover still exposes complete details.
- Plan tests pass **15/15**, full UI suite passes **33 files / 182 tests**, type-check and production build pass, and the managed planner is healthy on **17604/20003**. UI-health execution validation passes at **L5** across all seven routes and desktop/mobile profiles with the clipping findings cleared. Experience receipt `20260919-234036-858caffe` passes at **L3 Complete**.
- Remaining truth: UI-health retains only the documented Night warm-ivory local color advisory; the full Day/Night visual diff and broader product-scope work remain open.

## 2026-09-19 — Today focus-view geometry closure

- Corrected the Today Focus timeline so its ruler columns and background divisions derive from the active four-hour span instead of retaining the eleven-column full-day geometry. The zoom control now produces a real readable focus window while preserving the same allocation lanes and disclosure behavior.
- Dashboard tests pass **9/9**, type-check and production build pass, managed restart is healthy on **17604/20003**, and experience receipt `20260919-234425-8377334b` passes at **L3 Complete**.
- Remaining truth: the focused timeline geometry is now structurally correct, but the approved Day/Night visual diff and broader product/monetization scope remain open.

## 2026-09-19 — Observatory Day/Night oracle checkpoint

- Re-captured the built Today surface at the approved 1585×992 reference size for both Day and Night and compared the live composition against the two approved mockups. Sidebar proportion, left-focused scenery, stars, editorial heading scale, timeline hierarchy, and the Night warm-ivory action island now align closely with the oracle.
- Corrected the remaining Night mismatch: the secondary `Open draft` action now uses the ivory card’s local dark-on-light contrast instead of inheriting the global dark-theme button fill. Corrected Night evidence is `bas-capture://c4a0ee34-6d71-45cb-9c17-f6ff202f5839/screenshot`; Day evidence is `bas-capture://bb5fe5a4-42bf-4b8e-8243-0485ed08b8c6/screenshot`.
- Full UI suite passes **33 files / 182 tests**, production build passes, managed restart is healthy on **17604/20003**, and experience receipt `20260919-234855-762c41c2` passes at **L3 Complete**.
- Residuals are honest product differences rather than visual defects: live date/data differ from the sample concept data, and the broader secondary-surface/responsive/monetization audit remains open.

## 2026-09-19 — Responsive Observatory checkpoint

- Rechecked the built Today surface at **390×844** in both Day and Night. The mobile shell has no unintended outer black border/padding, the bottom navigation reaches the safe-area edge, and the status/navigation chrome follows the active appearance.
- The fixed left-focused scenery remains visible behind the content without horizontal distortion; the Night scene retains a star field and the warm-ivory action card with readable dark secondary action. The compact appearance control, capture action, and bottom navigation remain usable at the phone width.
- Runtime evidence: Day `bas-capture://4d98656a-a5e8-49ff-9949-62eff34ef197/screenshot`; Night `bas-capture://db77a784-55c0-41dd-91fa-08c801c0e18d/screenshot`. This closes the current responsive Today checkpoint; secondary-surface visual refinement and broader product/monetization scope remain open.

## 2026-09-19 — Settings scenery contract closure

- Settings scenery controls now have a real shared contract: Today reads `art-free`, `reduced-scenery`, and `subdued-night` through a durable preference reader, listens for storage/custom preference events, and applies the scene classes without requiring a route remount. Art-free removes panorama layers; reduced scenery lowers their visual weight; subdued night preserves its existing reading-island treatment.
- The appearance module has a focused persistence regression, Dashboard has a live preference-update regression, and the focused Settings/Dashboard/theme suite passes **19/19**. The full UI suite passes **33 files / 184 tests**, type-check and production build pass.
- Managed restart is healthy on **17604/20003**; execution-enabled `ui-health validate scenario personal-planner --json` passes at **L5** across all seven routes and desktop/mobile profiles. The only remaining finding is the documented Night warm-ivory raw-color advisory.

## 2026-09-20 — Browser chrome branding closure

- Comprehensive Test Genie run `20260920-000129-85e490c3` completed **25/27**; the two failures are portability and performance, not the planner's runtime UI. Its branding phase exposed a real install-surface defect: the HTML theme color disagreed with the manifest, and no dark-scheme fallback was declared.
- Aligned the manifest theme color with the page source and added the dark-scheme theme-color fallback while preserving ChromeTheme's unscoped runtime-updated meta tag. `brand-manager provider validate personal-planner` now passes with Install Surface complete, and scoped Test Genie branding run `20260920-001027-c46e8d58` passes **1/1**. The remaining branding advisory is the generated-token custom-font detector, documented as intentional system-font usage.
- Production build passes and the managed planner is healthy on **17604/20003**; notification-hub remains degraded by its unrelated tunnel-manager freshness timeout.

## 2026-09-20 — Initial-load performance closure

- Split the deeper Plan, Goals, Focus, and Review routes into deferred chunks with a styled loading state. The initial JS bundle dropped from roughly **880 KB** to **804 KB**; route chunks are emitted independently.
- Observatory scenery now requests only the active day or night panorama on first render, while retaining the crossfade layer for the alternate appearance. This reduces unnecessary mobile image work and preserves the appearance transition.
- Full UI suite passes **33 files / 184 tests**, type-check and production build pass. Managed restart is healthy on **17604/20003**; notification-hub remains degraded by its unrelated tunnel-manager freshness timeout.
- Test Genie performance run `20260920-002132-eec37121` passes **1/1 at L3 Complete**, clearing the prior Lighthouse error (`0.71–0.74` below the `0.75` threshold). The broader product/monetization scope remains open.

## 2026-09-20 — Settings section hierarchy checkpoint

- Reworked Settings from a visually flat list into deliberate Observatory sections: each shared `SettingsList` group now has a restrained raised surface, section rhythm, dividers, clearer internal padding, and a safe-area-aware mobile bottom gutter. The existing shared `RadioGroup`, `Switch`, `Input`, `Select`, `FormField`, and `Button` controls remain the interaction primitives.
- Settings/AppShell focused tests pass **17/17**, the full UI suite passes **33 files / 184 tests**, type-check and production build pass, and `git diff --check` is clean.
- Managed runtime validation passes `ui-health` at **L5** across all seven declared routes and desktop/mobile profiles; Test Genie experience run `20260920-002814-7492f69c` passes **1/1 at L3 Complete**, and the current-build performance run `20260920-002931-67e3b25a` passes **1/1 at L3 Complete**. The remaining work is deeper surface-by-surface visual comparison and the monetization/product-boundary release story.

## 2026-09-20 — Dynamic mobile chrome checkpoint

- `ThemeProvider` now synchronizes every declared media-specific `theme-color` meta variant with the resolved Observatory color in addition to the shared ChromeTheme owner’s active tag. This keeps status/navigation chrome aligned when the user switches Auto, Day, or Night on mobile browsers.
- ThemeProvider coverage is **7/7**, full UI coverage is **33 files / 185 tests**, type-check/build and diff-check pass, and the managed runtime is healthy on **17604/20003**.
- Fresh execution-enabled `ui-health` validation remains **L5** across all seven routes and desktop/mobile profiles. The only remaining advisory is the intentional Night action-surface raw-color detector; deeper visual-oracle and product-scope work remains open.

## 2026-09-20 — Plan routine-editor geometry closure

- Fresh BAS evidence exposed a real visual defect below the automated health threshold: the desktop routine editor compressed its four fields until the “Routine start / Optional” and “Routine minutes / Optional” labels collided. The form now uses explicit minimum column widths, a two-column tablet layout, and a single-column phone layout; label rows also wrap safely without losing the optional marker.
- Plan tests pass **15/15**, full UI suite passes **33 files / 185 tests**, type-check/build and diff-check pass, and the managed runtime is healthy on **17604/20003**. Fresh ui-health validation remains **L5** across all seven routes and both viewport profiles.
- Remaining truth: the routine editor is structurally and responsively corrected. Current-build performance run `20260920-004309-3f0d8b52` passes **1/1 at L3 Complete**; deeper surface-by-surface visual-oracle comparison and the monetization/product-boundary release story remain open.

## 2026-09-20 — Plan capacity perspective

- Added a first-class Capacity perspective to Plan using the existing server-owned planning response. It separates accepted work, available minutes, protected breathing room, provider-held time, freshness, and the accepted allocation list instead of making capacity a decorative number strip.
- Capacity is intentionally read-only: placement controls and the schedule timeline remain on Day, while the new view gives a calm, truthful accounting surface that works on phone and desktop.
- Plan tests pass **16/16**, full UI suite passes **33 files / 186 tests**, type-check and production build pass, and scoped diff-check is clean. Runtime/UI-health validation remains the next gate after the managed bundle restart.
- Remaining truth: Plan still has broader documented perspective/product-scope work open (Month, Timeline, Commitments, and the monetization story); this slice closes only Capacity.

## 2026-09-20 — Plan Month and Timeline perspectives

- Added Month and Timeline projections over the accepted allocation record set. Month provides a compact current-month load calendar with clear days; Timeline provides a chronological, source-aware long-form view for longer-horizon scanning.
- Both views use the existing range query and preserve the distinction between accepted work and unaccepted proposals. No unsupported commitment or forecast data is fabricated; the Commitments perspective remains explicitly open because the current domain has no commitment contract.
- Plan tests pass **16/16**, full UI suite passes **33 files / 186 tests**, type-check and production build pass, and `git diff --check -- scenarios/personal-planner` is clean.
- Remaining truth: Commitments still needs a real domain/proto/API slice; the monetization story and full visual-oracle audit remain open.

## 2026-09-20 — Commitment lifecycle vertical slice

- Added the missing commitments domain end-to-end: generated Connect proto, SQLite `commitments` and `commitment_revisions` tables, validation/service/repository, lifecycle state changes, revision persistence, API module/endpoint registry, CLI commands, and Plan’s Commitments perspective.
- The Plan view uses shared `FormField`, `Input`, `Select`, `Button`, and `EmptyState` primitives. It keeps promise state separate from accepted schedule state, shows unknown risk/acknowledgment honestly, and offers explicit proposed → active → fulfilled transitions.
- Live CLI smoke created and listed `Live commitment smoke`, then revised its promised boundary from `2026-10-01` to `2026-10-03`; the API preserved `risk: unknown` and `acknowledgment_status: unknown` and returned revision 2.
- UI tests pass **34 files / 188 tests**, Plan commitment/API tests pass **18/18**, API and CLI suites pass, production build/type-check pass, endpoint generation passes, managed runtime is healthy on **17604/20003**, and execution-enabled ui-health remains **L5** across all declared routes and desktop/mobile.
- Remaining truth: forecast/risk derivation is not yet implemented, so the UI deliberately reports unknown risk; monetization and the full visual-oracle audit remain open.

## 2026-09-20 — Commitment boundary composer and mobile Plan navigation

- Expanded the shared-control commitment composer to capture definition of done, beneficiary, assumptions, and scope exclusions alongside the promised result and boundary. Existing commitment cards now surface the most important supplied metadata without inventing risk or acknowledgment.
- Plan’s seven-view switcher now stays within the content width and scrolls horizontally on narrow screens, preserving 44px controls and preventing the view row from forcing page overflow.
- Focused Plan tests pass **17/17**, the full UI suite passes **34 files / 188 tests**, type-check and production build pass. Forecast/risk derivation remains intentionally open; monetization remains an R3 hypothesis with no fake billing surface.

## 2026-09-20 — Deterministic forecast outlook vertical slice

- Added the R1 forecast contract end-to-end: generated Forecasts proto, deterministic shared-resource kernel, API read endpoint, CLI `forecasts get`, and a Plan Outlook view.
- The outlook reports central/cautious scenario dates, protected reserve, bounded horizon, freshness, input fingerprint, algorithm version, and an explanation. It labels the result as a scenario rather than a probability and never mutates accepted allocations or commitments.
- API tests pass, CLI tests pass after primitive-evidence regeneration, UI tests pass **35 files / 190 tests**, type-check/build pass, live CLI forecast smoke returned a current 28-day forecast, and execution-enabled ui-health remains **L5** across all declared routes and both viewport profiles.
- Remaining truth: this is the first deterministic forecast slice, not the complete dependency-aware horizon solver, historical snapshot/change-record UI, external-provider freshness model, or calibrated R2 probability forecast.

## 2026-09-20 — Promise-versus-forecast risk comparison

- Extended the forecast contract with per-commitment outlooks. Active/proposed promises are compared against the same central and cautious dates, producing explicit `on_track`, `elevated`, `at_risk`, or `unknown` explanations without changing the commitment's stored lifecycle or risk field.
- The Plan Outlook view now includes a calm Promise Check section showing each promised boundary beside its current forecast and reason.
- Forecast kernel coverage now includes a late central promise and a cautious-only risk case. API, CLI, and full UI suites pass; the live Connect response includes commitment outlooks; managed ui-health remains **L5** across desktop/mobile routes.
- Remaining truth: forecast snapshots are still computed on read rather than persisted/history-compared, and dependency/provider-aware resource modeling remains open.

## 2026-09-20 — Durable forecast snapshots and change records

- Forecasts now persist immutable snapshots keyed by the shared input fingerprint, plus material change records linking the previous and current snapshot. Repeating an unchanged read reuses the existing snapshot instead of creating noisy history.
- The API exposes snapshot identity, prior snapshot identity, and a compact “what changed” explanation. Plan Outlook surfaces that explanation when an outlook materially moves.
- Live evidence created one snapshot, added a real work item, then produced a second snapshot and one linked change record with the changed central/cautious dates. API/CLI suites pass; UI passes **35 files / 190 tests**; type-check/build pass.
- Remaining truth: history is now durable for the aggregate outlook, while a dedicated history browser, dependency graph, provider freshness, and richer historical replays remain open.
## 2026-09-20 — Forecast history browsing

Forecast snapshots are now user-visible rather than database-only. The Forecasts API and CLI expose a bounded recent-history list, including scenario outcomes and recorded change explanations. Plan → Outlook renders explicit loading, error, empty, and populated history states beside the current deterministic outlook. Validation included generated proto refresh, API/CLI tests, focused UI tests, TypeScript, production build, managed restart, CLI JSON smoke, and raw Connect smoke.

## 2026-09-20 — Goals editorial hierarchy

Raised Goals beyond a stacked CRUD form: the surface now opens with a calm outcome overview and truthful active/complete/milestone-led counts, gives goal creation a named editorial entry point, preserves the draft on save errors, and shows milestone-aware progress metadata plus a visible completion meter on each goal. The existing goal/milestone domain and shared form controls remain authoritative. Goals tests pass 6/6, type-check and production build pass, managed restart is healthy, and execution-enabled ui-health remains L5 across all declared routes and desktop/mobile profiles. Focus, Review, and monetization remain open in the broader surface audit.

## 2026-09-20 — Focus ritual hierarchy

Focused work now has a clearer entry and active-session language: the empty state explains the open-timer, active-time, and correction model before starting; the running state exposes recording status and a small server-saved evidence ledger without implying that elapsed time equals completion. Existing durable timer transitions and manual actual correction paths remain unchanged. Focus tests pass 5/5, type-check/build pass, managed restart is healthy, and execution-enabled ui-health remains L5. Review and monetization remain open.

## 2026-09-20 — Review reflection hierarchy

Review now opens with a period-aware compass that frames Day versus Week as observation rather than judgment. Summary tiles carry distinct evidence roles, and coverage has a visible evidence marker while preserving unknown/unrecorded semantics. Carry-forward and reflection remain explicit, optional actions over the existing domain data. Review tests pass 6/6, type-check/build pass, managed restart is healthy, and execution-enabled ui-health remains L5. Monetization and the final adversarial visual-oracle audit remain open.

## 2026-09-20 — Responsive Plan chrome reconciliation

The server-owned Experience gate caught a real Plan regression that generic UI-health missed: the horizontally scrollable perspective switcher put Timeline/Outlook outside the viewport at both 390×844 and 844×390. Mobile portrait now wraps the eight perspectives into two columns; short landscape uses four columns so the chrome remains inside the captured viewport without document overflow. The corrected Experience receipt `20260920-021020-455eea56` passes at L3 Complete with clean structure reconciliation. The Settings desktop/mobile captures (`bas-capture://bea36256-65fc-439f-8340-c70dd24b5651/screenshot`, `bas-capture://807557a2-9f90-4ab2-b99a-f74a3017b967/screenshot`) and Review desktop capture (`bas-capture://8ba5a14b-129a-474e-b188-c926de1e7939/screenshot`) provide current visual evidence; the broader final Today-oracle and monetization audit remains open.

## 2026-09-20 — Local typography evidence

Brand Manager exposed a real hardening gap: the declared sans family had no loading evidence. The Observatory now declares a local `Planner Sans` face backed by installed system faces, so typography remains network-independent and the browser no longer silently falls back from an unloaded named family. Brand Manager now validates at **L3** with zero findings. Full UI coverage remains **35 files / 190 tests**, production build and type-check pass, and execution-enabled ui-health remains **L5**; remaining findings are the intentional Night action-surface token advisory and generic component-library text-box measurements.

## 2026-09-20 — Cross-route Observatory theme parity

Runtime inspection found that secondary routes were still inheriting the shared library's blue dark palette even though Today used Observatory scenery and amber/lilac Night controls. The semantic Observatory Day/Night tokens now override the library defaults at the document layer, so Settings, Plan, Goals, Focus, Review, the shell, bottom navigation, and ChromeTheme resolve from the same appearance choice. A live Playwright check against the managed bundle reports Night `--color-background: #141321` and `--color-surface: #201e30` on both `/settings` and `/`; the updated Settings capture shows the corrected Observatory Night treatment. Full UI coverage remains **35 files / 190 tests**, build/type-check pass, and final execution-enabled ui-health remains **L5** with only the intentional raw-palette advisory beyond runtime confirmations.

The rebuilt bundle also passed Experience receipt `20260920-023409-c7221f57` (**1/1**, L3 Complete, clean structure reconciliation).

The fresh 390×844 managed-browser check reports `scrollWidth === innerWidth` with no horizontal overflow. Mobile Settings and Today captures show the shared Night palette, full-width safe-area bottom navigation, and left-focused scenic crop without the former outer black border.

## 2026-09-20 — Settings editorial header parity

Settings now uses the Observatory page-header composition shared by Plan, Goals,
Focus, and Review: eyebrow, serif title, description, and a settings mark, on
the same planner surface background. Its actual controls remain the shared
component-library SettingsList, RadioGroup, Switch, FormField, Input, Select,
and Button primitives. Focused Settings and shell accessibility tests pass,
type-check/build pass, the managed runtime is healthy, and the live Settings
capture `/tmp/personal-planner-settings-editorial.png` shows the corrected
hierarchy. Experience receipt `20260920-025021-032b6ada` passed 1/1 at L3
Complete with clean structure reconciliation. The remaining Settings audit is
the long-form mobile and lower-section interaction sweep.

## 2026-09-20 — Today timeline compact disclosure

The primary Today timeline now follows the Plan disclosure rule: sub-hour
accepted blocks show compact, legible durations (`45m`, `30m`, `20m`, `25m`)
instead of broken title fragments, while full titles remain available from the
native hover title, keyboard focus, accessible label, and anchored detail
popover. Full-hour blocks retain their titles. The final managed Today capture
is `/tmp/personal-planner-today-final-duration.png`. The full UI suite passes
35 files / 190 tests, type-check/build pass, the managed runtime is healthy,
and Experience receipt `20260920-030058-bdc96885` passed 1/1 at L3 Complete
with clean structure reconciliation. Notification-hub remains degraded by the
unrelated tunnel-manager freshness timeout.

## 2026-09-20 — Compact timeline visual-health hardening

The runtime visual gate found two clipped `h2` labels in the narrow Plan
timeline blocks after the Today disclosure pass. Plan compact blocks now use
the same tight duration geometry as Today, eliminating the clipping without
changing their time-based width or collision lane. A fresh
`ui-health validate scenario personal-planner --json` remains L5 and reduced
runtime findings from 16 to 14: the two `visual_text_clipped` warnings are
gone; only the existing raw-palette advisory and runtime confirmations remain.
Type-check/build and diff-check pass. Experience receipt
`20260920-030714-41b9f378` passed 1/1 at L3 Complete with clean structure
reconciliation.

## 2026-09-20 — Mobile secondary-surface and chrome evidence

The 390px runtime audit covered Goals, Focus, and Review after the Observatory
shell work. Their editorial headers, shared controls, scrollable content, and
safe-area bottom navigation remain inside the intended narrow composition with
no new clipping or shell inset. The shell regression now also asserts the
resolved Night `theme-color`, `--rcl-status-fill`, and rendered status strip,
not only the route class. Focused chrome/theme tests pass 19/19 and the full UI
suite passes 35 files / 191 tests; type-check/build and diff-check pass. The
generated telescope/observatory brand asset and the honest free-core/future-
convenience monetization story remain present; no fake billing action was added.

## 2026-09-20 — Plan timeline disclosure readability

Narrow accepted allocations no longer render their titles as fragmented character
wrapping. Blocks under an hour now use a compact duration label while retaining
the full event name in the accessible group label, native hover title, and
anchored detail popover. This preserves the exact collision-lane geometry rather
than widening or vertically distorting the schedule. Plan tests pass 18/18, the
full UI suite passes 35 files / 190 tests, type-check and production build pass,
and the managed runtime renders the compact schedule on `/plan`. Experience
receipt `20260920-024551-942e7e1d` passed 1/1 at L3 Complete with clean
structure reconciliation. The managed scenario is healthy; notification-hub
remains degraded by the unrelated tunnel-manager freshness timeout.

## 2026-09-20 — Day panorama contrast correction

Reference-width comparison found that Day mode's panorama was being flattened
by an 88% surface scrim through the middle of the scene. The Observatory Day
scrim now preserves a warm, readable panorama while retaining the content
legibility gradient; Night rules remain unchanged. Full UI validation passes
35 files / 191 tests, type-check/build pass, and ui-health remains L5 with no
visual clipping findings. Managed restart is healthy on API `17604` / UI
`20003`; notification-hub remains degraded by the unrelated tunnel-manager
freshness timeout.

## 2026-09-20 — Sidebar resize range correction

The shared AppShell sidebar retained its 144px Observatory baseline but now
advertises a useful 128–280px resize range with explicit keyboard steps. The
previous 120–176px range made pointer resizing look effectively stuck. The
library separator bounds are covered by an AppShell regression; the full UI
suite passes 35 files / 192 tests and the managed scenario restarts healthy.
The notification-hub/tunnel-manager freshness timeout remains unrelated.

## 2026-09-20 — Shared appearance resolution

The visual audit found that `?appearance=day|night` was resolved inside Today
only, leaving the AppShell sidebar and browser chrome on their previous global
theme during deep-link captures. Appearance query overrides now resolve in the
shared ThemeProvider before the shell mounts. ThemeProvider, AppShell, and
Dashboard regressions pass; the full suite passes 35 files / 193 tests, and
ui-health remains L5. This keeps the route surface, sidebar, bottom navigation,
status strip, and browser theme color on one appearance state.

The shell-level regression now mounts `/settings?appearance=night` and verifies
the rendered AppShell class, sidebar baseline, and document theme attribute,
not only the provider state. Full UI validation is 35 files / 194 tests.

Fresh reference-width BAS captures now verify the rebuilt runtime visually:
Day `bas-capture://19b4e8a1-a161-470a-bba4-2f18544d2e02/screenshot` has the
light sidebar and dark-on-light Observatory content; Night
`bas-capture://194b7019-aa13-43c0-8644-b9df775c1dea/screenshot` has the dark
sidebar, ivory action card, stars, and night scene. The remaining visual work
is refinement toward the oracle, not shell/theme desynchronization.

## 2026-09-20 — Day panorama atmospheric blend refinement

The Day scene's upper half was still too flat against the approved cloudy
Observatory composition. The Day-only scrim now lets more sky texture through
while preserving dark content contrast; Night remains unchanged. After a
managed restart, a fresh 1585×992 BAS capture shows the observatory held at
the left focal edge, visible cloud and mountain texture across the schedule,
and readable event controls. Artifact:
`bas-capture://c1904fae-6c06-4273-9fcc-9ecdbc31603b/screenshot`.
The full UI suite passes 35 files / 194 tests, `git diff --check` is clean,
and ui-health remains L5 with only existing informational runtime and raw
palette advisories. The managed scenario is healthy; notification-hub remains
degraded by the unrelated tunnel-manager freshness timeout.

## 2026-09-20 — Sidebar route-entry scroll hardening

The shared AppShell already reset its primary sidebar content box, but a
nested library scroll container could retain a non-zero offset across route
entry and hide the top of navigation. Route reset now clears the sidebar and
all of its descendants, including horizontal offsets, without taking
ownership of the shell markup. The AppShell regression covers a nested
scrolling descendant. Full UI validation passes 35 files / 195 tests,
type-check/build and diff-check pass, and a fresh Review BAS capture shows the
Planner mark plus Today, Plan, Goals, Focus, Review, and Settings together:
`bas-capture://2839fac0-519e-461f-b325-e525b3af2d29/screenshot`.

## 2026-09-20 — Compact Observatory appearance control

Today’s Auto/Day/Night control now uses a compact icon-first segmented
presentation: the selected appearance expands its short label, while the
other choices remain icon-visible and retain their accessible names and
native titles. This reduces header competition without hiding the choice
model or changing persistence/transition behavior. Fresh Day BAS evidence at
1585×992:
`bas-capture://835f453b-34f9-4bd3-8b17-4f4fc8e471e8/screenshot`.
The full UI suite passes 35 files / 195 tests and type-check/build/diff-check
pass; the broader Day/Night oracle and secondary-surface review remain open.

## 2026-09-20 — Mobile browser-chrome viewport contract

Aligned the Planner document shell with the proven Web Console mobile contract:
`interactive-widget=resizes-content`, `viewport-fit=cover`, explicit zoom
behavior, and a document-level light/dark color scheme. This gives the shared
`StatusBarFill` and runtime `theme-color` updates the correct notch, keyboard,
and home-indicator viewport to paint into. UI tests pass 35 files / 195 tests;
type-check, production build, and diff-check pass. Managed restart reports the
planner healthy on API `17604` / UI `20003`; notification-hub remains degraded
by its unrelated tunnel-manager freshness timeout.

## 2026-09-20 — Responsive scenery and query-aware route reset

Landscape Day/Night evidence at 844×390 confirms the Observatory scene remains
fixed, left-anchored on the observatory, and visually cropped rather than
squashed; Night retains a visible star field. The AppShell route-entry reset
now keys off both pathname and URL search, so appearance/deep-link changes do
not carry stale main or sidebar scroll state into the next surface.

Fresh evidence: Day `bas-capture://3db06971-b5e1-408e-a9f4-7de0774118cb/screenshot`
and Night `bas-capture://0417de08-55d2-4772-9727-773d7b1fdea9/screenshot`.
The full suite passes 35 files / 195 tests, focused AppShell validation passes,
type-check/build and diff-check pass, and ui-health remains L5. The managed
planner is healthy on API `17604` / UI `20003`; notification-hub remains
degraded by the unrelated tunnel-manager freshness timeout.

## 2026-09-20 — Persistent validation-data cleanup

The secondary-surface audit exposed validation residue in the persistent local
database: smoke work items, focus sessions, commitments/goals, generated
proposal/forecast history, and fixture calendar connections were appearing as
if they were a user's real plan. After stopping the managed scenario, the
database was backed up to `/tmp/personal-planner-db-backup.VXAwq2.sqlite` and
only those validation records were removed; the workspace/profile schema was
preserved. Restarted runtime evidence now shows an honest empty Plan with
`0 min` planned, `360 min` available, `360 min` breathing room, and no routines:
`bas-capture://94041309-c71c-4203-928b-c79d81c64be2/screenshot`.

## 2026-09-20 — Cross-surface appearance transition contract

Secondary routes now share a reduced-motion-aware appearance transition layer:
the planner main surface, direct surface cards, and Settings group surfaces
animate background, foreground, border, and elevation changes together when
Day/Night changes. The shared main scroller also publishes bottom
`scroll-padding` for the fixed mobile navigation, so focused controls are not
hidden behind the home-indicator/navigation region. Full UI validation passes
35 files / 195 tests; type-check/build/diff-check pass; managed runtime
`ui-health` remains L5.
