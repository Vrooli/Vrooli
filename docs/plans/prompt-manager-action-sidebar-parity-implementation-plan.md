# Prompt-Manager Action Sidebar Feature Parity Implementation Plan

Date: 2026-05-01

## Purpose

Bring the prompt-manager Actions sidebar tab to practical feature parity with the Skill sidebar where the workflows make sense: richer search, AI search, select/share mode, saved copy sets, and copy formats including CLI commands. The implementation should maximize reuse by generalizing existing sidebar entity machinery instead of duplicating Skill-specific code.

## Required Reading

```bash
prompt-manager skill read implementation-plan-authoring react-coherence seam-discovery-and-enforcement test
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Blocked documentation read: `knowledge-observatory docs read prompt-manager seams` and `knowledge-observatory docs read prompt-manager problems` failed because the knowledge-observatory API base could not be resolved. Do not treat this plan as having refreshed those docs.

## Greenfield Rule

This is greenfield. Do not add compatibility shims, legacy aliases, duplicate Actions-only copies of shared sidebar logic, unused re-exports, `_unused` placeholders, or dead code. If a seam is wrong, replace it with the correct shared seam.

## Problem Statement

Actions are now a first-class prompt-manager entity, but the sidebar still treats them as a basic list. Skills have a mature browse surface with quick/content/AI search, view controls, filters, select mode, saved sets, and copy/share formats. Actions currently have only local row filtering and a New Action footer.

Evidence:
- `scenarios/prompt-manager/ui/src/components/tree/SkillTreeSidebar.tsx:70` disables content and AI search for Actions.
- `scenarios/prompt-manager/ui/src/components/tree/SkillTreeSidebar.tsx:79` omits Actions from `TAB_TO_ENTITY_TYPE`, so select mode and saved sets are unavailable.
- `scenarios/prompt-manager/ui/src/stores/combineStore.ts:15` omits Actions from `CombineEntityType`.
- `scenarios/prompt-manager/ui/src/components/action/ActionListPanel.tsx:11` has no select-mode, view-mode, saved-set, AI, or copy integration props.
- `scenarios/prompt-manager/ui/src/components/tree/SkillTreeSidebar.tsx:2135` renders Actions as only `ActionListPanel`, unlike Agents/Teams/Topics which already participate in select/copy footers.
- Backend Action AI search mostly exists in `api/aisearch`, but `scenarios/prompt-manager/api/main.go:387` registers skill, agent, and team AI routes only; there is no `/search/actions/ai` route.

## Scope

In scope:
- Action tab quick search parity over Action metadata and command contracts.
- Action tab AI search.
- Action tab select/share mode.
- Action saved copy sets.
- Action copy formats: XML, Markdown, JSON, CLI command.
- Shared sidebar refactors needed to avoid duplication.
- Tests for store/entity type support, Action list selection, sidebar action tab controls, Action AI client route, and copy formatting.

Out of scope:
- Changing Action contract semantics.
- Creating new Action schema fields.
- Reworking Skill content search internals beyond extracting reusable presentation/state seams.
- Broad visual redesign of the prompt-manager sidebar.

## Target UX

Move these Skill-sidebar capabilities to Actions:
- `Quick` search: search `id`, `name`, `description`, `status`, `owner`, `command.argv`, `tags`, input names/types/descriptions, output names/types/descriptions, permissions, and example descriptions.
- `AI` search: semantic search over indexed Actions, with text fallback just like other AI search surfaces.
- `Select` mode: checkboxes in Action rows and AI results, with a footer showing selected Action count.
- `Saved` sets: same saved-set panel/editor and localStorage persistence as other entity types, namespaced to Actions.
- Copy/share formats:
  - `json`: selected full Action contracts.
  - `markdown`: readable Action summaries with command, owner, inputs, outputs, permissions, examples.
  - `xml`: deterministic machine-readable Action blocks.
  - `cli`: `prompt-manager action show <id>` for one Action, and newline-separated `prompt-manager action show <id>` commands for many Actions.

Do not move these Skill-only affordances as-is:
- Skill folder/mode tree management.
- Skill content line search and editor line highlighting. If deep Action search is added, make it contract-field search, not fake Skill content search.
- Skill health badges unless an Action health score map already exists.

## Current Technical Context

Key UI files:
- `scenarios/prompt-manager/ui/src/components/tree/SkillTreeSidebar.tsx` owns tab feature flags, active-tab search state, AI search execution, saved-set wiring, and per-tab rendering.
- `scenarios/prompt-manager/ui/src/components/action/ActionListPanel.tsx` owns current Action rows and draft Action creation.
- `scenarios/prompt-manager/ui/src/stores/combineStore.ts` owns select/copy mode state and entity type.
- `scenarios/prompt-manager/ui/src/lib/copySetStorage.ts` persists saved copy sets by `CombineEntityType`; its comment already says entity types are skills, agents, teams, topics.
- `scenarios/prompt-manager/ui/src/components/tree/CombineActionBar.tsx` is already generic enough for Actions.
- `scenarios/prompt-manager/ui/src/components/search/SearchResultsList.tsx` already renders Action results inside discover results but its top-level `EntityType` excludes `actions`.
- `scenarios/prompt-manager/ui/src/components/layout/SkillManagerLayout.tsx` prefetches selected entity copy content. Skills use `api.displaySkills`; agents/teams/topics use local formatters in the layout.

Key API files:
- `scenarios/prompt-manager/api/aisearch/service.go` has `SearchActions`.
- `scenarios/prompt-manager/api/aisearch/models.go` has `AIActionSearchResult` and `AIActionSearchResponse`.
- `scenarios/prompt-manager/api/aisearch/handlers.go` does not expose `SearchActions` yet.
- `scenarios/prompt-manager/api/main.go` registers `/discover`, which supports `type=action`, but it does not register `/search/actions/ai`.

## Implementation Strategy

### Phase 1 - Establish Shared Entity Contracts

1. Replace ad hoc entity-tab mappings with a typed sidebar entity registry:
   - Add `actions` to `CombineEntityType`.
   - Add `actions` to tab-to-entity mapping.
   - Add a single entity config table with label, plural label, search placeholder, feature flags, saved-set support, copy support, and row identity helpers.
2. Update `copySetStorage` comments/tests to include Actions.
3. Ensure selection persistence accepts `entityType: "actions"` with no migration shim. Invalid persisted values can continue to be rejected by validation.

Acceptance:
- TypeScript compile catches every place that assumed only skills/agents/teams/topics.
- `pm.copySets.actions` is used for Action saved sets.

### Phase 2 - Refactor List Rendering Instead of Duplicating It

1. Extract a reusable selectable row/list primitive from `ActionListPanel`, `AgentListPanel`, `TeamListPanel`, and/or existing list view patterns only as far as needed:
   - Required API: `items`, `selectedId`, `onNavigate`, `isSelectMode`, `selectedIds`, `onToggleSelection`, `renderPrimary`, `renderSecondary`, `renderMeta`, `emptyState`.
   - Keep entity-specific row content in small render functions.
2. Update `ActionListPanel` to support:
   - `isSelectMode`
   - `selectedIds`
   - `onToggleSelection`
   - action row checkbox state
   - separate navigate button when in select mode, matching `SearchResultsList` behavior.
3. Keep Action creation in `ActionListPanel`; it is Action-specific and should not leak into the shared primitive.

Acceptance:
- Action rows select/deselect in select mode without opening the editor.
- In select mode, the user can still navigate to an Action through a separate control.
- Existing Action creation tests still pass.

### Phase 3 - Add Action Copy Formatting Through a Shared Display Service

1. Create or replace with a focused `entityDisplayService` that centralizes copy formatting for all selectable sidebar entities.
2. Preserve Skills as the only entity that calls `api.displaySkills`, because Skill list data may not include full content.
3. Move existing agent/team/topic formatting out of `SkillManagerLayout` into the service.
4. Add Action formatting:
   - JSON: stable `JSON.stringify(selectedActions, null, 2)`.
   - Markdown: one heading per Action plus owner, status, command, inputs, outputs, permissions, examples.
   - XML: deterministic `<action>` blocks with escaped values.
   - CLI: one `prompt-manager action show <id>` per line.
5. Update `SkillManagerLayout` prefetch logic to call the shared display service for every non-Skill entity, including Actions.

Acceptance:
- `CombineActionBar` works on the Actions tab.
- Copy success records an Action copy set.
- No copy formatting logic remains embedded in `SkillManagerLayout` except orchestration.

### Phase 4 - Add Action AI Search

1. Backend:
   - Add `SearchActions` handler to `api/aisearch/handlers.go`.
   - Register `POST /api/v1/search/actions/ai` in `api/main.go`.
   - Add handler tests using the existing Action search service seam.
2. Frontend schemas/client:
   - Add `AIActionSearchResponseSchema` and type exports.
   - Add `api.aiSearchActions(query, limit)`.
   - Add `actionService.aiSearchActions` only if keeping service-wrapper parity with skills is useful; otherwise use `api` consistently with agents/teams in the sidebar.
3. Sidebar:
   - Set Actions `aiSearch: true`.
   - Add Action AI results state.
   - Search Actions in the AI effect when `activeTab === "actions"`.
   - Extend `SearchResultsList` top-level `EntityType` to include `actions`.
   - Render Action results with name, status, owner, command if available, tags, score, checkbox, and navigate control.

Acceptance:
- Action tab shows Quick/AI mode toggle.
- AI mode searches Actions directly and falls back to text when vector search is unavailable.
- Existing Skill discover results can still include Action result badges.

### Phase 5 - Wire Saved Sets and Footer for Actions

1. Add Actions to:
   - entity lookup map
   - saved-set editor entity list
   - `handleApplySavedSet`
   - `handleSelectModeToggle`
   - tab rendering saved-set branch
2. Add Action tab footer:
   - In select mode, render `CombineActionBar` with `entityLabel="action"`.
   - Outside select mode, keep the existing `New Action` footer.
3. Avoid duplicating the saved-set blocks again. Extract a small `renderSavedSetContent(entityType)` helper or component if needed.

Acceptance:
- Actions tab exposes `Saved` while in select mode.
- Saved Action sets can be applied, edited, renamed, and copied.
- New Action remains available outside select mode.

### Phase 6 - Optional Contract Field Search

Implement only if Phase 1-5 are stable and tests are green.

1. Add a local Action contract search utility that returns grouped matches by Action and field path.
2. Reuse the existing content-search result presentation shape where possible, but name it contract search in code.
3. Do not add a backend endpoint unless local list data proves insufficient.

Acceptance:
- If added, Action deep search finds input/output/permission/example fields and navigates to the Action editor.
- The UI does not claim line-number source content for `action.json` unless the API actually returns line positions.

## Contract Decisions

- `CombineEntityType` must include `actions`.
- CLI copy format for Actions is `prompt-manager action show <id>`, not a new CLI alias.
- Direct AI search should use `/search/actions/ai`. Unified discover remains useful for cross-entity capability discovery, but the Actions tab should search Actions directly.
- Saved sets are local UI state and should use `pm.copySets.actions`.
- Action select mode should not auto-run or validate Actions. Selection is for copy/share only.

## Testing Plan

Frontend:
- `combineStore`/selection persistence tests: Actions entity type persists and restores.
- `copySetStorage.test.ts`: Actions copy sets are isolated from other entity types.
- New `entityDisplayService.test.ts`: Action XML/Markdown/JSON/CLI formatting, escaping, stable ordering, empty selection.
- `ActionListPanel.test.tsx`: select mode toggles rows, selected rows render checked state, navigate remains separate, creation still works.
- `SearchResultsList.test.tsx`: Action AI result rows render and support select/navigate.
- `SkillTreeSidebar.test.tsx`: Actions tab has Select, saved sets, footer copy bar, AI toggle, and calls `onEnterSelectMode("actions")`.
- `SkillManagerLayout` tests or hook-level tests: selected Actions prefetch and copy via shared display service.

Backend:
- `api/aisearch` handler tests for `POST /search/actions/ai`.
- Existing `DiscoverTyped` tests remain green.
- Route registration/parity tests updated if they assert known search routes.

Validation commands:

```bash
cd scenarios/prompt-manager/ui && pnpm test -- ActionListPanel SkillTreeSidebar SearchResultsList copySetStorage
cd scenarios/prompt-manager/ui && pnpm type-check
cd scenarios/prompt-manager && go test ./api/aisearch/... ./api/actions/... -timeout 300s
```

Scenario-level validation, if needed:

```bash
cd scenarios/prompt-manager && make test
```

## Rollout Checklist

- [x] Actions appear in the sidebar registry as a first-class selectable entity.
- [x] Actions tab quick search covers command contract fields.
- [x] Actions tab AI search works with text fallback.
- [x] Actions tab select mode works on normal rows and AI result rows.
- [x] Action selected content copies in XML, Markdown, JSON, and CLI formats.
- [x] Saved Action sets are isolated under `pm.copySets.actions`.
- [x] No duplicate Actions-only copy of saved-set, copy-bar, or AI-result components exists.
- [x] Tests listed above pass.

## Risks And Mitigations

| Risk | Mitigation |
| --- | --- |
| Sidebar file is already large and easy to make worse | Extract small targeted seams: entity registry, saved-set renderer, display service. Do not rewrite the whole sidebar. |
| Action AI search service exists but route is missing | Add the route and handler first, then wire UI to it. |
| Copy formatting drifts across entity types | One display service owns all non-Skill formatting; Skills remain backend-rendered. |
| Action contract search is forced into Skill content-search model | Keep Phase 6 optional and call it contract search. Do not fake file line matches. |
| Saved-set type expansion breaks persisted old values | Strict validation should reject invalid values; do not add migration compatibility code. |

## Non-Goals And Prohibited Patterns

- Do not duplicate SkillListView/SkillCardView as ActionListView/ActionCardView unless a reusable primitive cannot express the row.
- Do not add `prompt-manager action read` just for UI copy parity; use existing `action show`.
- Do not add legacy entity aliases or compatibility layers.
- Do not make Action selection imply execution.
- Do not restart prompt-manager from inside this task.

## Definition Of Done

- Actions have sidebar quick search, AI search, select/share, saved sets, and copy formats where those concepts apply.
- Shared code owns entity selection/copy behavior; Actions are data/config plugged into that machinery.
- The backend exposes direct Action AI search and tests it.
- The UI has focused tests that would fail if Actions fall out of sidebar parity again.
- No dead code, duplicate Action-only abstractions, compatibility shims, or unused exports remain.
