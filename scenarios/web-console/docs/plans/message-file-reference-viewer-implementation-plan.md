# Message File Reference Viewer — Implementation Plan

## 1. Purpose

Add first-class file-reference handling to the web-console messages view so agent-generated file paths open a safe in-app viewer instead of behaving like broken external URLs. The implementation must resolve file references against session context, support markdown/code preview for common text files, and fail clearly when a path cannot be resolved.

## 2. Required Reading

```bash
# Skills
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement

# Current web-console UI
cat scenarios/web-console/ui/src/components/MessagesPane.tsx
cat scenarios/web-console/ui/src/components/markdown/MarkdownRenderer.tsx
cat scenarios/web-console/ui/src/__tests__/MarkdownRenderer.test.tsx
cat scenarios/web-console/ui/src/stores/useConversationStore.ts
cat scenarios/web-console/ui/src/stores/useWorkspaceStore.ts
cat scenarios/web-console/ui/src/lib/api.ts

# Current web-console API seams
cat scenarios/web-console/api/main.go
cat scenarios/web-console/api/errors.go
cat scenarios/web-console/api/session_handlers.go
cat scenarios/web-console/api/workspace_handlers.go
cat scenarios/web-console/api/pty.go
cat scenarios/web-console/api/pty_tmux.go
cat scenarios/web-console/api/config.go

# Safe path/file serving precedent
cat scenarios/scenario-to-desktop/api/handlers_docs.go

# Reusable file typing / preview precedent
cat scenarios/git-control-tower/ui/src/lib/fileTypes.ts
cat scenarios/git-control-tower/ui/src/components/MarkdownPreview.tsx
```

## 3. Problem Statement

The messages pane currently renders markdown links as ordinary external anchors. In [MarkdownRenderer.tsx](../../ui/src/components/markdown/MarkdownRenderer.tsx), the custom `a` renderer always emits `target="_blank"` with the raw `href`. This works for real URLs, but it fails for agent-generated file references such as:

- Relative repo paths like `docs/plans/foo.md`
- Absolute local paths like `PROJECT_ROOT/src/app.ts:42`
- Web-console-style file links like `[app.ts](/abs/path/app.ts:12)`

The result is misleading UX:

- The file reference looks interactive and trustworthy.
- Clicking it attempts browser navigation to a non-web path.
- The app has no current file-view surface or resolution feedback.

This is common enough in coding-agent workflows that the messages pane loses a substantial amount of its utility precisely when users need it most: implementation plans, code review findings, test failures, and change summaries.

## 4. Scope

### In scope

- Detect file-like links in rendered markdown inside the messages view
- Resolve clicked paths against session/workspace context on the backend
- Add a safe read-only file preview API for text/markdown/code use cases
- Add an in-app viewer surface for resolved files from the messages pane
- Support `:line` suffixes on file references
- Provide explicit unresolved/error states instead of broken navigation
- Add unit/integration coverage for path parsing, resolution, API semantics, and viewer UX
- Update relevant internal docs if seams or error semantics change

### Out of scope

- General-purpose file editing from the messages pane
- Arbitrary filesystem browsing outside message-triggered navigation
- Preview support for every binary/document type in v1
- Changes to conversation event storage format
- Automatic inference from plain text that is not already rendered as a link
- Full IDE-style diff viewer, outline view, or cross-file navigation

## 5. Current Technical Context

### Relevant UI files

- `scenarios/web-console/ui/src/components/MessagesPane.tsx`
  Renders the messages list and already owns session-aware message interactions.
- `scenarios/web-console/ui/src/components/markdown/MarkdownRenderer.tsx`
  Central markdown rendering seam. This is where anchor click interception should begin.
- `scenarios/web-console/ui/src/lib/api.ts`
  Canonical API client layer. New file-resolution/view endpoints belong here.
- `scenarios/web-console/ui/src/stores/useConversationStore.ts`
  Stores conversation state and pane view mode; does not currently model file viewing state.
- `scenarios/web-console/ui/src/stores/useWorkspaceStore.ts`
  Stores pane metadata and layout state; likely the right place for viewer panel state if the viewer becomes pane-level.

### Relevant API files

- `scenarios/web-console/api/main.go`
  Registers all current HTTP routes. New routes must be added here and covered in `main_test.go`.
- `scenarios/web-console/api/errors.go`
  Structured error catalog. New path/file errors must fit existing category/recovery semantics.
- `scenarios/web-console/api/session_handlers.go`
  Current session JSON shape. No cwd/root metadata is exposed today.
- `scenarios/web-console/api/pty.go`
  PTY seam currently supports I/O, resize, liveness, and readiness, but not cwd lookup.
- `scenarios/web-console/api/pty_tmux.go`
  Tmux backend already has a proven way to ask tmux for `#{pane_current_path}`.
- `scenarios/web-console/api/config.go`
  Defines default working-directory resolution for new sessions. This is only the initial cwd, not necessarily the live cwd after user `cd`.

### Existing precedents

- `scenarios/scenario-to-desktop/api/handlers_docs.go`
  Demonstrates safe path normalization, root checks, file serving, and markdown content endpoints.
- `scenarios/git-control-tower/ui/src/lib/fileTypes.ts`
  Provides a lightweight file-type classification precedent worth reusing or copying.
- `scenarios/git-control-tower/ui/src/components/MarkdownPreview.tsx`
  Shows a read-only markdown/code preview approach already accepted elsewhere in this repo.

### Key technical constraints

1. File resolution must not be delegated to browser URL behavior.
2. Relative paths are ambiguous without session context.
3. Absolute local paths must be constrained to allowed roots.
4. The messages pane is session-scoped, so file resolution should also be session-scoped.
5. The initial backend should prioritize correctness and safety over broad preview support.

## 6. Target End State

When a user clicks a file reference in the messages pane:

1. The link is intercepted instead of opening a new browser tab.
2. The UI asks the backend to resolve the link for the current session.
3. If resolution succeeds within allowed roots:
   - A viewer opens in-app.
   - Markdown files render as markdown.
   - Code/text files render read-only with syntax highlighting/plain text fallback.
   - If a line suffix was provided, the viewer scrolls/focuses that line.
4. If resolution fails:
   - The user sees a clear reason such as `file_not_found`, `path_not_allowed`, or `path_resolution_ambiguous`.
   - No broken browser navigation occurs.

The viewer should feel like a natural extension of the messages pane, not a separate documentation subsystem.

## 7. Implementation Strategy

### Phase 0: Contract and seam preparation

Goal: establish the minimal backend/UI contracts before adding any viewer UI.

1. Add a file-reference parsing utility in the UI that can classify:
   - external URL
   - absolute local file path
   - relative file path
   - optional `:line` suffix
2. Keep parsing conservative. Only intercept links that are clearly file-like.
3. Define backend response types in `ui/src/lib/api.ts` before implementing UI behavior.
4. Decide viewer state ownership:
   - Preferred v1: local state owned by `MessagesPane` or a dedicated child surface.
   - If a pane-level reusable viewer is desired later, keep the initial state model easy to lift.

### Phase 1: Backend file-resolution API

Goal: make the backend the source of truth for path resolution.

Add session-scoped endpoints under `/api/v1/sessions/{id}/files/...`.

Recommended endpoints:

- `POST /api/v1/sessions/{id}/files/resolve`
  Input: `{ "path": string }`
  Output: resolution metadata only
- `GET /api/v1/sessions/{id}/files/content?path=<resolved-or-canonical>`
  Output: content + preview metadata for allowed text-like files

Why split resolution from content:

- Resolution errors are easier to test and reason about separately.
- The UI can decide whether to open an error state, fetch content, or short-circuit.
- It avoids overloading a single endpoint with too many responsibilities.

Implementation work:

1. Add a small path-resolution module/handler pair rather than embedding logic in `main.go`.
2. Introduce a session-aware cwd lookup seam.
3. Normalize and validate incoming paths.
4. Resolve relative paths in deterministic order:
   - live session cwd
   - session default working dir / project root fallback
5. Enforce allowed-root checks before any content read.
6. Return structured metadata including:
   - original input
   - resolved path
   - whether resolution used live cwd or project-root fallback
   - file existence
   - file type classification
   - detected line number

### Phase 2: PTY cwd seam

Goal: resolve relative paths correctly for both PTY backends.

Current gap:

- `PTY` has no cwd query capability.
- `resolveWorkingDir()` only gives the launch directory, not the live directory.

Recommended seam change:

- Extend `PTY` with a cwd query method, for example `CurrentDir(ctx context.Context) (string, error)`.

Backend strategy:

- `tmuxPTY`: use `tmux display-message -t <session> -p "#{pane_current_path}"`.
- `realPTY`: resolve from `/proc/<pid>/cwd` for Linux, with graceful fallback to the launch/default cwd if unavailable.

Important rule:

- Failure to fetch live cwd must not fail the entire request if project-root fallback can still be attempted.
- The API response should indicate when fallback resolution was used.

### Phase 3: File content API and classification

Goal: safely serve previewable files.

Implement a content handler that:

1. Reads only after path-resolution succeeds.
2. Restricts preview to text-like files in v1.
3. Applies size limits to avoid loading huge files into the messages viewer.
4. Returns metadata sufficient for the UI to choose the renderer.

Recommended preview classes:

- `markdown`
- `code`
- `text`
- `binary`
- `unsupported`
- `not_found`

Recommended v1 limits:

- hard cap on preview size
- no binary body preview
- no recursive directory listing

Reuse guidance:

- Copy or adapt `git-control-tower` file-type classification rather than introducing a broad new package.
- Reuse existing web-console markdown renderer for markdown content where practical.

### Phase 4: Messages-pane link interception

Goal: make file links behave correctly without regressing external links.

Update `MarkdownRenderer` so the custom `a` renderer:

1. Leaves real external URLs alone.
2. Intercepts file-like links and routes them to a callback prop such as `onFileReferenceClick`.
3. Preserves accessibility and visible link styling.
4. Avoids `target="_blank"` for intercepted local-file references.

Recommended design:

- `MarkdownRenderer` remains presentation-focused.
- `MessagesPane` passes session-aware click handlers into the renderer.
- The renderer does not import API clients directly.

### Phase 5: In-app viewer surface

Goal: give the user a clear, contained preview experience.

Recommended v1 UI:

- Open a modal/sheet/panel from the messages pane rather than a full new route.
- Header shows:
  - basename
  - resolved path
  - line reference if present
  - resolution mode such as `resolved from session cwd`
- Body behavior:
  - markdown renderer for markdown
  - code/text block viewer for code/text
  - explicit unsupported/binary state otherwise

Why modal/panel first:

- Lower implementation cost than introducing route-level file navigation
- Keeps the interaction anchored to the message that produced it
- Easier to validate before deciding whether the feature should grow into a general workspace file viewer

### Phase 6: Error and unresolved states

Goal: replace broken navigation with clear product behavior.

Add structured API/UI handling for at least:

- `file_reference_invalid`
- `file_reference_not_found`
- `file_reference_not_allowed`
- `file_reference_too_large`
- `file_reference_unresolvable`

UI behavior:

- Show inline banner/toast or viewer error panel.
- Include recovery guidance such as:
  - path appears relative to a different cwd
  - file is outside the allowed workspace roots
  - file is too large to preview

## 8. Contract Decisions

### API contract

Recommended resolve request:

```json
{
  "path": "scenarios/web-console/ui/src/components/MessagesPane.tsx:166"
}
```

Recommended resolve response:

```json
{
  "input_path": "scenarios/web-console/ui/src/components/MessagesPane.tsx:166",
  "resolved_path": "/home/matthalloran8/Vrooli/scenarios/web-console/ui/src/components/MessagesPane.tsx",
  "line": 166,
  "exists": true,
  "resolution_basis": "project_root",
  "category": "code",
  "can_preview": true
}
```

Recommended content response:

```json
{
  "path": "/home/matthalloran8/Vrooli/scenarios/web-console/ui/src/components/MessagesPane.tsx",
  "line": 166,
  "category": "code",
  "content_type": "text/plain; charset=utf-8",
  "content": "..."
}
```

### Allowed roots

V1 should explicitly allow:

- the resolved project/workspace root used by web console sessions
- any session-scoped upload/cache area that the product already intentionally surfaces, if needed

V1 should explicitly reject:

- arbitrary absolute paths outside allowed roots
- traversal escapes
- directory reads unless directory support is intentionally added later

### Resolution basis semantics

Use explicit values, for example:

- `session_cwd`
- `project_root`
- `absolute_allowed`
- `unresolved`

This prevents hidden ambiguity when debugging why a file opened or failed.

### UI ownership

V1 recommendation:

- Keep file-view interactions local to the messages feature.
- Do not add a global “workspace file browser” abstraction unless the implementation reveals a strong reuse case.

## 9. Testing Plan

### API unit tests

Add tests for:

- file-path parsing with and without `:line`
- relative path resolution from session cwd
- fallback resolution from project root
- absolute path acceptance within allowed roots
- absolute path rejection outside allowed roots
- traversal rejection
- missing file behavior
- oversized preview behavior
- non-previewable/binary classification

Likely files:

- new `file_reference_handlers_test.go`
- new resolver-focused test file if logic is extracted into a helper/module
- `main_test.go` route registration updates

### PTY seam tests

Add tests for:

- tmux cwd lookup path
- standard PTY cwd lookup fallback behavior
- resolver behavior when live cwd lookup fails

### UI unit tests

Extend or add tests for:

- `MarkdownRenderer` leaves `https://...` links untouched
- `MarkdownRenderer` intercepts file-like links
- `MessagesPane` opens the viewer for resolved files
- unresolved paths show a clear error state
- markdown preview renders markdown files
- code preview renders code/text files
- line suffix is carried through to viewer state

Likely files:

- `scenarios/web-console/ui/src/__tests__/MarkdownRenderer.test.tsx`
- new messages-pane viewer tests
- new API client tests in `ui/src/__tests__/api.test.ts`

### Manual validation

Validate with real message content containing:

- relative project path
- absolute repo path
- markdown file
- code file
- nonexistent path
- path outside allowed root
- file with line suffix

## 10. Rollout / Validation Checklist

- Add backend routes and register them in `main.go`
- Add structured errors to `errors.go` and document semantics if needed
- Extend PTY seam for cwd lookup
- Implement resolver + content handlers
- Add API client methods in `ui/src/lib/api.ts`
- Update markdown link renderer to intercept file references
- Add viewer UI in messages flow
- Add unit/integration tests on both API and UI
- Run targeted UI tests
- Run targeted Go tests
- Run the scenario test suite for web-console
- Manually validate clicking file links from real agent output

Suggested validation commands:

```bash
cd scenarios/web-console/ui && pnpm test -- MarkdownRenderer
cd scenarios/web-console/ui && pnpm test -- MessagesPane
cd scenarios/web-console/api && go test ./...
vrooli scenario test web-console
```

If the UI test script requires the scenario’s normal wrapper, prefer the existing scenario conventions over ad hoc commands.

## 11. Risks + Mitigations

### Risk: relative paths still resolve incorrectly

Cause:
- live cwd is unavailable or stale

Mitigation:
- make `resolution_basis` explicit
- fall back deterministically
- surface unresolved state clearly rather than silently guessing

### Risk: security regression through arbitrary file reads

Cause:
- weak root checks or path normalization

Mitigation:
- backend-only resolution
- explicit allowed-root policy
- traversal tests
- content-size limits

### Risk: feature grows into an accidental generic file browser

Cause:
- broadening scope during implementation

Mitigation:
- keep v1 message-triggered and read-only
- defer directory navigation/editing

### Risk: UI coupling inside `MarkdownRenderer`

Cause:
- importing session/API concerns directly into the renderer

Mitigation:
- keep the renderer callback-driven and presentation-only

## 12. Non-goals / Prohibited Patterns

- Do not treat local filesystem paths as browser URLs.
- Do not resolve paths only in the frontend.
- Do not allow preview reads outside explicit allowed roots.
- Do not add write/edit capabilities in v1.
- Do not introduce a heavyweight new document-view subsystem unless the existing markdown/code components are clearly insufficient.
- Do not silently fall back to the wrong file when resolution is ambiguous.

## 13. Definition of Done

The feature is done when all of the following are true:

1. Clicking a file-like markdown link in the messages pane no longer attempts broken browser navigation.
2. External web links still behave normally.
3. Relative file references resolve correctly using session context when available, with explicit fallback behavior when not.
4. Absolute local paths are previewable only when they are inside allowed roots.
5. Markdown and code/text files open in a read-only in-app viewer.
6. Unsupported, missing, too-large, and disallowed paths produce clear user-facing states.
7. The backend path/file API is covered by tests for normalization, safety, and resolution behavior.
8. The UI link interception and viewer behavior are covered by tests.
9. Web-console scenario validation passes without regressing existing messages-pane behavior.

## 14. Open Questions / Assumptions

### Assumptions

- The primary allowed root should be the same project/workspace root already implied by `resolveWorkingDir()` and scenario launch environment.
- Linux `/proc/<pid>/cwd` is available for the standard PTY backend in the current deployment target.
- A modal/panel viewer is sufficient for v1 and preferable to adding a new route.

### Questions to resolve during implementation

1. Should unsupported-but-textual formats such as JSON/YAML be classified as `code` or `text` in v1?
2. Should session upload paths be previewable from the same viewer if agents reference them?
3. Do we want a reusable viewer component under `src/components/file-viewer/` immediately, or should v1 keep it local to messages until reuse appears?
