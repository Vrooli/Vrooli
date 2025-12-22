**Workspace Sandbox Integration Plan**

Goal: add the first Vrooli-specific feature in git-control-tower by surfacing
workspace-sandbox approved changes, staging them safely, and providing a
commit message assist flow.

---

**Context Summary**
- workspace-sandbox already exposes:
  - `GET /api/v1/pending`
  - `GET /api/v1/commit-preview` (includes pending files + suggestedMessage)
  - `POST /api/v1/commit-pending`
- git-control-tower already supports:
  - staging/unstaging, commit creation, and audit logging
  - UI with Change list, commit panel, and banner-style status UI
- Cross-scenario discovery is implemented in api-core:
  - `github.com/vrooli/api-core/discovery.ResolveScenarioURLDefault`
  - Used in `scenarios/system-monitor/api/internal/agentmanager/client.go`

---

**Scope of the First Feature**
1) Show approved changes availability in git-control-tower
   - A banner above the Changes list with a CTA to stage approved changes.
   - Per-file badge icon for files that have sandbox-approved pending changes.
2) Provide commit message assistance
   - Button in Commit panel to insert the sandbox suggested message.

---

**Integration Design**
**API Layer (git-control-tower)**
- Add a workspace-sandbox client that:
  - Resolves base URL using `discovery.ResolveScenarioURLDefault(ctx, "workspace-sandbox")`.
  - Calls `GET /api/v1/commit-preview?projectRoot=<repoDir>`.
  - Normalizes errors: if scenario is not running, return `available: false`.
- Add endpoint:
  - `GET /api/v1/repo/approved-changes`
  - Response shape:
    - `available` (bool)
    - `committableFiles` (int)
    - `suggestedMessage` (string)
    - `files` (list: `relativePath`, `status`, `sandboxId`, `sandboxOwner`)
    - `warning` (string, optional)

**UI Layer (git-control-tower)**
- New banner above Change list:
  - Trigger: approved changes available and `committableFiles > 0`.
  - CTA: "Stage approved changes".
  - Action: stage all `files[].relativePath` where `status == "pending"`.
  - After stage, set commit message to `suggestedMessage`.
- Per-file badge:
  - Add icon for files present in approved changes (pending only).
  - Align with binary-file badge pattern.
- Commit panel:
  - Add "Use approved message" button.
  - Visible only when staged files intersect approved files.
  - Click inserts `suggestedMessage` into commit textarea.
  - Requires lifting commit message state to `App.tsx`.

---

**Gap + Minimal Enhancement Needed in workspace-sandbox**
Problem:
- `commit-preview` is all pending changes; no API for staged subset.

Minimal fix:
- Add `POST /api/v1/commit-preview` with body:
  - `projectRoot`
  - `filePaths[]` (relative paths to filter)
  - Optional `sandboxIds[]`
- Return the same preview structure, but filtered to requested files.
- This enables "Use approved message" for manually staged subsets.

---

**Implementation Steps**
1) git-control-tower API
  - Add workspace-sandbox client with api-core discovery.
  - Implement `GET /api/v1/repo/approved-changes`.
  - Add unit tests for:
    - scenario not running -> `available:false`
    - preview response -> normalized payload.
2) git-control-tower UI
  - Add approved-changes query/hook.
  - Add banner above Change list.
  - Add per-file badge in Change list.
  - Lift commit message state to `App.tsx`.
  - Add "Use approved message" button in Commit panel.
  - Wire CTA: stage approved file list + insert suggested message.
3) workspace-sandbox API (if needed)
  - Add filtered commit-preview endpoint.
  - Update service + repository to filter by filePaths.
  - Tests for filter behavior + empty response.
4) Optional audit metadata
  - Add commit audit metadata with sandbox IDs when staged approved files
    are committed.

---

**Testing Plan**
- git-control-tower API tests:
  - Discovery failure -> `available:false`.
  - Successful preview normalization.
- git-control-tower UI tests:
  - Banner appears when approved changes exist.
  - CTA stages pending approved files.
  - Badge appears on approved files.
  - Commit message insert works.
- workspace-sandbox tests (if filter endpoint is added):
  - Filtered preview only includes requested file paths.
  - Suggested message reflects filtered set.

---

**Files Likely to Change**
- git-control-tower API:
  - `scenarios/git-control-tower/api/main.go`
  - `scenarios/git-control-tower/api/http_handler.go`
  - New client file (ex: `scenarios/git-control-tower/api/workspace_sandbox_client.go`)
- git-control-tower UI:
  - `scenarios/git-control-tower/ui/src/lib/api.ts`
  - `scenarios/git-control-tower/ui/src/lib/hooks.ts`
  - `scenarios/git-control-tower/ui/src/components/FileList.tsx`
  - `scenarios/git-control-tower/ui/src/components/CommitPanel.tsx`
  - `scenarios/git-control-tower/ui/src/App.tsx`
- workspace-sandbox API (if extending):
  - `scenarios/workspace-sandbox/api/internal/handlers/diff.go`
  - `scenarios/workspace-sandbox/api/internal/sandbox/service.go`
  - `scenarios/workspace-sandbox/api/internal/repository/sandbox_repo.go`

---

**Open Decisions**
- Whether to extend workspace-sandbox commit-preview or accept full preview.
- Whether to include audit metadata for sandbox IDs on commit.
