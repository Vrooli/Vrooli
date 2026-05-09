# API Reference: git-control-tower

92 routes across 11 functional groups. Source of truth:
[CODE: api/routes.go].

All endpoints are mounted under `http://localhost:<API_PORT>` (printed
by `vrooli scenario port git-control-tower`). Mutating endpoints are
audited via [CODE: api/audit_logger.go].

## Health & capabilities

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/health`                       | Infrastructure health probe (root). |
| GET | `/api/v1/health`                | Same handler, client-friendly path. |
| GET | `/api/v1/capabilities`          | Server-reported feature flags. |

## Repo state — status, diff, history

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/api/v1/repo/status`           | Working-tree status, file categorisation, hotspots. |
| GET | `/api/v1/repo/diff`             | Unified diff for one file (`--path` query). |
| GET | `/api/v1/repo/history`          | Git log with optional grep / limit / file-detail include. |
| GET | `/api/v1/repo/sync-status`      | Push/pull-needed assessment vs upstream. |
| GET | `/api/v1/repo/provenance`       | AI-attribution lookup for a path. |
| GET | `/api/v1/repo/related`          | Related-file suggestions for a path. |
| GET | `/api/v1/repo/search/content`   | Content search (grep-like) across tracked files. |
| GET | `/api/v1/repo/files`            | File listing for a directory. |
| GET | `/api/v1/repo/files/dir`        | Directory tree listing. |
| PUT | `/api/v1/repo/files/content`    | Save file content (audited). |
| POST | `/api/v1/repo/files/delete`    | Delete a path (audited). |

## Repo registry (multi-repo prep)

| Method | Path | Notes |
| --- | --- | --- |
| GET    | `/api/v1/repos`                 | List registered repos. |
| GET    | `/api/v1/repos/active`          | Get current active repo. |
| POST   | `/api/v1/repos/active`          | Set active repo. |
| POST   | `/api/v1/repos/open`            | Register an existing local repo. |
| POST   | `/api/v1/repos/clone`           | Clone a remote into the registry. |
| DELETE | `/api/v1/repos/{id}`            | Unregister a repo. |

## Staging, commit, discard, ignore

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/repo/stage`             | Stage paths (audited). |
| POST | `/api/v1/repo/unstage`           | Unstage paths (audited). |
| POST | `/api/v1/repo/commit`            | Compose a commit (audited). |
| POST | `/api/v1/repo/discard`           | Discard working-tree changes (audited). |
| POST | `/api/v1/repo/ignore`            | Add patterns to .gitignore (audited). |
| GET  | `/api/v1/repo/approved-changes`  | Approved-changes catalog. |
| POST | `/api/v1/repo/approved-changes/preview` | Preview approved changes. |
| GET  | `/api/v1/repo/grouping-rules`    | File grouping rules. |
| PUT  | `/api/v1/repo/grouping-rules`    | Replace grouping rules. |
| GET  | `/api/v1/repo/gitignore/health`  | .gitignore lint status. |
| POST | `/api/v1/repo/gitignore/move`    | Move gitignore entries. |

### Repo history includes

`GET /api/v1/repo/history` accepts:
- `limit=<n>`: cap returned commits.
- `grep=<text>`: filter commits by message text.
- `include=files`: return detailed `entries` with changed files.
- `include=checks`: return detailed `entries` with captured commit-check runs.
- `include=files,checks`: return files and checks together.

Commit-check runs are commit-scoped evidence captured by git-control-tower during commit creation. They are repo-agnostic and expose the configured command as opaque text:

```json
{
  "hash": "abc1234",
  "subject": "fix: example",
  "files": ["src/example.ts"],
  "checks": [
    {
      "kind": "precommit",
      "status": "passed",
      "command": "custom check command",
      "exit_code": 0,
      "summary": "Precommit checks passed",
      "stdout": "ok",
      "duration_ms": 42,
      "timestamp": "2026-05-09T12:00:00Z"
    }
  ]
}
```

## Push / pull / upstream

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/repo/push`              | Push branch (with safety checks). |
| POST | `/api/v1/repo/pull`              | Pull from upstream. |
| POST | `/api/v1/repo/upstream-action`   | Compound upstream operations. |
| POST | `/api/v1/repo/remote/url`        | Update remote URL. |

## Branches

| Method | Path | Notes |
| --- | --- | --- |
| GET  | `/api/v1/repo/branches`          | List branches (ahead/behind). |
| POST | `/api/v1/repo/branch/create`     | Create a branch. |
| POST | `/api/v1/repo/branch/switch`     | Switch branch (uncommitted-change checks). |
| POST | `/api/v1/repo/branch/publish`    | Publish current branch upstream. |

## Credentials & SSH

| Method | Path | Notes |
| --- | --- | --- |
| GET    | `/api/v1/credentials`           | List stored credentials. |
| POST   | `/api/v1/credentials`           | Save a credential. |
| DELETE | `/api/v1/credentials/{id}`      | Remove credential. |
| POST   | `/api/v1/credentials/test`      | Test credential connectivity. |
| GET    | `/api/v1/ssh/keys`              | List SSH keys. |
| DELETE | `/api/v1/ssh/keys`              | Remove SSH key. |
| POST   | `/api/v1/ssh/keys/generate`     | Generate a new SSH keypair. |
| POST   | `/api/v1/ssh/keys/public`       | Return public key for paste. |
| POST   | `/api/v1/ssh/keys/test`         | Test SSH key against remote. |

## Visual & workflow capture

| Method | Path | Notes |
| --- | --- | --- |
| POST   | `/api/v1/repo/visual-capture`                              | Trigger a screenshot capture. |
| GET    | `/api/v1/repo/visual-captures`                             | List captures. |
| GET    | `/api/v1/repo/visual-captures/{id}`                        | Capture detail. |
| GET    | `/api/v1/repo/visual-captures/{id}/screenshot/{filename}`  | Serve screenshot bytes. |
| GET    | `/api/v1/repo/visual-captures/{id}/screenshot/{filename}/path` | Resolve on-disk path. |
| GET    | `/api/v1/repo/visual-captures/{id}/video/{filename}`       | Serve video. |
| DELETE | `/api/v1/repo/visual-captures/{id}`                        | Delete capture. |
| GET    | `/api/v1/repo/visual-capture-storage`                      | Storage stats. |
| DELETE | `/api/v1/repo/visual-capture-storage`                      | Clear all captures. |
| POST   | `/api/v1/repo/workflow-capture`                            | Trigger a workflow capture. |
| GET    | `/api/v1/repo/workflow-captures`                           | List workflow captures. |
| GET    | `/api/v1/repo/workflow-captures/{id}`                      | Capture detail. |
| GET    | `/api/v1/repo/workflow-captures/{id}/video/{filename}`     | Serve video. |
| DELETE | `/api/v1/repo/workflow-captures/{id}`                      | Delete capture. |

## Test execution

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/repo/test-execution`            | Trigger test run. |
| GET  | `/api/v1/repo/test-executions`           | List runs. |
| GET  | `/api/v1/repo/test-executions/{id}`      | Run detail. |

## Tidiness & rules / auditor

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/repo/tidiness-scan`     | Run a tidiness scan. |
| GET  | `/api/v1/repo/tidiness-issues`   | Pending issues. |
| GET  | `/api/v1/repo/tidiness-scenario` | Per-scenario summary. |
| GET  | `/api/v1/repo/tidiness-score`    | Aggregate score. |
| GET  | `/api/v1/repo/tidiness-staleness`| Staleness metrics. |
| GET  | `/api/v1/repo/rules`             | Auditor rules catalog. |
| POST | `/api/v1/repo/rules-run`         | Run auditor rules. |
| GET  | `/api/v1/repo/rules-job/{jobId}` | Job status. |
| POST | `/api/v1/repo/rules-fix`         | Apply auditor fix. |
| GET  | `/api/v1/repo/rules-violations`  | Outstanding violations. |

## Agent manager

| Method | Path | Notes |
| --- | --- | --- |
| GET  | `/api/v1/agent/profiles`                  | Agent profiles. |
| POST | `/api/v1/agent/run`                       | Start an agent run. |
| GET  | `/api/v1/agent/runs`                      | List runs. |
| GET  | `/api/v1/agent/runs/{id}`                 | Run detail. |
| GET  | `/api/v1/agent/runs/{id}/diff`            | Run-produced diff. |
| GET  | `/api/v1/agent/runs/{id}/events`          | SSE event stream. |
| POST | `/api/v1/agent/runs/{id}/approve`         | Approve a run. |
| POST | `/api/v1/agent/runs/{id}/continue`        | Continue a paused run. |
| POST | `/api/v1/agent/runs/{id}/reject`          | Reject a run. |
| POST | `/api/v1/agent/runs/{id}/stop`            | Stop a run. |
| POST | `/api/v1/agent/attachments/upload`        | Upload attachment for runs. |

## Review pipeline

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/review/run`            | Start a scenario-review run. |
| GET  | `/api/v1/review/run/{jobId}`    | Job status / results. |
| GET  | `/api/v1/review/summary`        | Summary across runs. |

## Scenarios & audit log

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/api/v1/scenarios`                       | List scenarios. |
| GET | `/api/v1/scenarios/{slug}/envelope`       | Scenario envelope info. |
| GET | `/api/v1/audit`                           | Query audited operations. |
