# Plan: Add Manual Allow/Deny Controls for Backlog Items

## Required Reading
- `prompt-manager skill read ux react-coherence api-steer scenario-generation`
- `prompt-manager skill read implementation-plan-authoring`

## Problem Statement

Currently, the `acceptance_allow` and `acceptance_deny` glob patterns on backlog items are displayed as **read-only chips** on the `BacklogDetailsPage`. There is no UI for a user to manually add, edit, or remove these patterns. When workshop agents fail to set them during processing, users have no fallback mechanism in the UI.

## Scope

### In Scope
- Modal dialog for editing `acceptance_allow` and `acceptance_deny` glob arrays on `BacklogDetailsPage`
- Single modal with two stacked textareas (one for allow, one for deny)
- One pattern per line in each textarea, with per-line validation
- Glob syntax validation client-side + API-side path existence check via new `POST /backlog/validate-globs` endpoint
- Validation triggers on blur + debounced typing (500ms)
- Error summary below each textarea with line numbers
- Explicit save button in modal (changes are not persisted until user confirms)
- Lock editing when item is in `queued` or `in_progress` status
- Mobile-responsive modal layout
- Empty state placeholder on detail page: "No patterns set — click to add" when arrays are empty
- Textarea placeholder text with example patterns + helper text below each label

### Out of Scope
- Changes to the execution system's use of acceptance globs
- Bulk editing of acceptance globs across items
- Changes to the workshop agent's auto-population logic
- Approve/reject gating controls (separate feature)
- Updating BacklogFormDialog's comma-separated input (cosmetic improvement, separate backlog item)
- Playwright E2E tests — all testing via vitest component/unit tests

### Target Files
- `scenarios/swarm-manager/ui/src/pages/BacklogDetailsPage.tsx` — add edit button next to acceptance chips, wire to modal, add empty state placeholder
- `scenarios/swarm-manager/ui/src/components/backlog/AcceptanceGlobDialog.tsx` — **new file**: modal dialog with two stacked textareas for editing globs
- `scenarios/swarm-manager/ui/src/lib/glob-validation.ts` — **new file**: client-side glob validation utilities
- `scenarios/swarm-manager/api/` — new `POST /backlog/validate-globs` endpoint for path-existence checking

### Acceptance Scope Globs
- `acceptance_allow`: `scenarios/swarm-manager/ui/src/components/backlog/**`, `scenarios/swarm-manager/ui/src/pages/BacklogDetailsPage.tsx`, `scenarios/swarm-manager/ui/src/lib/glob-validation.ts`, `scenarios/swarm-manager/api/**`

## Resolved Decisions

### From Round 1
1. **Scope**: Editable acceptance_allow/acceptance_deny globs only (no gating controls)
2. **Interaction pattern**: Modal/dialog editor — textarea with one pattern per line
3. **Validation**: Glob syntax check client-side + API call to verify paths exist in project
4. **Save behavior**: Explicit save button in modal (batch changes, don't auto-save)
5. **Locked state**: Lock glob editing when item is queued or in_progress (consistent with existing locking)

### From Round 2
6. **Modal layout**: Single modal with two stacked textareas (allow and deny sections)
7. **API path validation**: New `POST /backlog/validate-globs` endpoint that accepts an array of globs and returns match results
8. **Validation UX timing**: Validate on blur + debounced typing (500ms)
9. **Error display**: Error summary below textarea with line numbers (e.g., "Line 3: absolute path not allowed")

### From Round 3
10. **Phasing**: Ship all phases together — UI + API validation endpoint delivered as one unit
11. **Testing**: Vitest component tests only — no Playwright E2E tests
12. **Empty state**: Add "No patterns set — click to add" placeholder on detail page when arrays are empty
13. **Helper copy**: Placeholder text in textareas (example patterns) + helper text below each label ("One glob pattern per line. Relative to project root.")

## Approach

### Phase 1: Client-Side Validation Utilities (`glob-validation.ts`)
1. Create `glob-validation.ts` with functions:
   - `validateGlobLine(line: string): { valid: boolean; error?: string }` — checks non-empty, relative path (no leading `/`), valid glob syntax (balanced braces, valid wildcards)
   - `validateGlobLines(text: string): ValidationResult[]` — splits by newline, validates each line, returns array with line numbers
   - `parseGlobTextarea(text: string): string[]` — splits text into trimmed, non-empty lines
2. Unit tests in `glob-validation.test.ts` covering: empty lines, absolute paths, invalid brace syntax, valid patterns, edge cases (whitespace-only lines, trailing newlines)

### Phase 2: API Validate-Globs Endpoint
1. Add `POST /backlog/validate-globs` to the swarm-manager API
2. Request body: `{ "patterns": ["scenarios/foo/**", "nonexistent/**"] }`
3. Response: `{ "results": [{ "pattern": "scenarios/foo/**", "matchCount": 12, "valid": true }, { "pattern": "nonexistent/**", "matchCount": 0, "valid": true, "warning": "No files match this pattern" }] }`
4. Syntax errors return `valid: false` with error message
5. The endpoint walks the project directory using `doublestar.Glob()` or similar
6. Tests: valid patterns with matches, valid patterns with no matches, invalid syntax, empty array

### Phase 3: AcceptanceGlobDialog Component
1. Create `AcceptanceGlobDialog.tsx` using the existing `Dialog` component from `components/ui/dialog.tsx`
2. Props: `isOpen`, `onClose`, `initialAllow: string[]`, `initialDeny: string[]`, `onSave: (allow: string[], deny: string[]) => void`, `isSubmitting?: boolean`
3. Two labeled sections, each with:
   - Label (e.g., "Allowed Paths" with FolderOpen icon, "Denied Paths" with X icon)
   - Helper text below label: "One glob pattern per line. Relative to project root."
   - `<textarea>` in monospace font, one pattern per line, pre-populated from initial props
   - Placeholder text in textarea: `scenarios/my-app/ui/**\nsrc/components/*.tsx`
   - Error summary below the textarea showing validation errors with line numbers
4. Validation triggers:
   - Client-side: on blur and on debounced typing (500ms) — uses `validateGlobLines()`
   - API path check: on blur only (not on every keystroke) — calls `POST /backlog/validate-globs`
   - Warnings (no matches) shown as amber, errors (invalid syntax) shown as red
5. Save button disabled when any red (error) validation issues exist; amber warnings do not block save
6. Cancel button always enabled
7. Component tests: renders both textareas, shows validation errors, disables save on errors, calls onSave with parsed arrays

### Phase 4: Integration with BacklogDetailsPage
1. Add an "Edit" pencil icon button next to the acceptance section header (near line 1556)
2. Button disabled when `isLocked` (queued/in_progress)
3. When `acceptance_allow` and `acceptance_deny` are both empty, show a clickable placeholder: "No patterns set — click to add" that opens the modal
4. On click (edit button or empty state placeholder), open `AcceptanceGlobDialog` pre-populated with current `item.acceptanceAllow` and `item.acceptanceDeny`
5. On save, call `backlogService.update(kind, name, { acceptanceAllow, acceptanceDeny })`
6. On success, invalidate/refetch the item to update the chip display
7. On error, show toast notification with error message
8. Integration tests: edit button visibility based on lock state, save triggers correct PATCH payload, chips update after save, empty state placeholder renders and opens modal

## Technical Context

### Existing Infrastructure
- **Dialog component**: `components/ui/dialog.tsx` — portal-based, escape/click-outside close, animation support, ARIA attributes, `isLoading` prop to block close
- **API PATCH endpoint**: `/api/v1/backlog/{kind}/{name}` accepts `acceptance_allow` and `acceptance_deny` arrays
- **Validation (API-side)**: Server-side validates relative, non-empty, syntactically valid glob patterns (update_patch.go:147-191)
- **UI state**: Items in Zustand `backlogStore`, mutations via `updateMutation`
- **Current display**: Read-only `<code>` chips with show-more expand (BacklogDetailsPage.tsx:1556-1612), FolderOpen icon for allow, X icon for deny
- **Locked state**: `LOCKED_STATUSES = Set(["queued", "in_progress"])` in `backlog-queue-utils.ts` — controls editability across the page via `itemActions.locked`
- **Payload builder**: `buildBacklogUpdatePayload()` in `backlog-service.ts` handles camelCase → snake_case conversion, only includes defined fields

### Existing Patterns to Follow
- `BacklogFormDialog` — dialog with Zustand form store, client-side validation in handleSubmit, error auto-clear on field change
- `RequirementFormDialog` — simpler dialog with local useState, validation in handleSubmit
- Current acceptance inputs in `BacklogFormDialog` use comma-separated text (lines 360-398) — the new modal improves on this
- All dialogs use the `Dialog` component with consistent `maxWidth`, `title`, `isLoading` patterns

## Testing Strategy

All tests are vitest component/unit tests. No Playwright E2E tests.

### Unit Tests
- **`glob-validation.test.ts`**: Validate rules match API-side behavior
  - Empty string → error
  - Absolute path (leading `/`) → error
  - Unbalanced braces → error
  - Valid glob patterns (`**/*.ts`, `src/{a,b}/**`) → pass
  - Whitespace-only lines → filtered out by `parseGlobTextarea`
  - Trailing newlines → handled gracefully

### Component Tests
- **`AcceptanceGlobDialog.test.tsx`**:
  - Renders two textareas with correct labels and helper text
  - Pre-populates from `initialAllow` / `initialDeny` props (one pattern per line)
  - Shows placeholder text when textarea is empty
  - Shows client-side validation errors below textarea after typing + blur
  - Save button disabled when validation errors exist
  - Save button enabled when only warnings (no matches) exist
  - Calls `onSave` with correctly parsed arrays on save click
  - Calls `onClose` on cancel click
  - Shows loading state when `isSubmitting` is true

### Integration Tests
- **BacklogDetailsPage acceptance editing**:
  - Edit button visible when item is not locked
  - Edit button disabled/hidden when item is queued or in_progress
  - Empty state placeholder renders when no patterns set, opens modal on click
  - Modal opens pre-populated with current patterns
  - Save triggers PATCH with `{ acceptance_allow: [...], acceptance_deny: [...] }`
  - Chips update after successful save
  - Error toast shown on PATCH failure

### API Tests
- **`POST /backlog/validate-globs`**:
  - Valid patterns return match counts
  - Invalid syntax returns `valid: false` with error message
  - Empty array returns empty results
  - Non-matching valid patterns return `matchCount: 0` with warning

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| User enters invalid glob patterns | Low — API rejects with error | Client-side validation before submit; disable save button when errors present |
| Accidental deletion of agent-set patterns | Medium — could affect execution | Modal shows current patterns pre-populated; user must explicitly clear and save |
| API validate-globs endpoint slow on large repos | Medium — poor UX | Run path check on blur only (not debounced typing); show spinner during check; timeout after 3s and skip |
| Modal UX on mobile with long pattern lists | Low — cosmetic | Textarea scrolls naturally; use full-width modal on mobile (`maxWidth: 'max-w-xl'`) |
| CLI `backlog update` 405 error | Low — orthogonal | Tracked separately; UI uses HTTP PATCH directly, not CLI |

## Acceptance Criteria

- [ ] User can open a modal dialog from BacklogDetailsPage to edit acceptance_allow and acceptance_deny patterns
- [ ] Modal displays two labeled textareas with one pattern per line
- [ ] Each textarea has helper text ("One glob pattern per line. Relative to project root.") and placeholder examples
- [ ] When both arrays are empty, detail page shows "No patterns set — click to add" placeholder
- [ ] Invalid glob syntax shows error summary below textarea with line numbers
- [ ] Valid patterns with no matches show amber warning (not blocking)
- [ ] Save button is disabled when validation errors exist
- [ ] Changes persist via PATCH endpoint on save
- [ ] Edit button is disabled when item is queued or in_progress
- [ ] Modal pre-populates with existing patterns on open
- [ ] Chips on detail page update after successful save
- [ ] Modal is usable on mobile devices
- [ ] Unit tests pass for glob-validation.ts
- [ ] Component tests pass for AcceptanceGlobDialog
- [ ] Integration tests pass for BacklogDetailsPage acceptance editing
- [ ] API tests pass for POST /backlog/validate-globs
- [ ] No regressions to existing backlog detail functionality
