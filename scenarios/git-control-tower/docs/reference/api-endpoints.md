# API Reference: git-control-tower

40 REST routes across 12 functional groups, including the two operational
health probes. Source of truth:
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

The health JSON payload is typed by
`packages/proto/schemas/git-control-tower/v1/shared/health.proto`. Health is
the intentional operational REST exception: lifecycle managers and load
balancers must be able to probe it without a Connect client.

## Source distributions (read-only)

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/api/v1/source-distributions` | Projection of `scenario-to-repository/ListDistributions`; optional `repository_context` and `scenario` filters. |
| GET | `/api/v1/source-distributions/{id}` | Projection of the source-ramp distribution, contents, publication handoff, and drift read surfaces. |

These routes never assemble, publish, approve, create repositories, access
credentials, or modify Git/host state. A source-ramp outage is returned as a
typed `503` unavailable response; a partial detail read retains the durable
distribution and identifies the unavailable evidence rather than fabricating
it. See [source distribution integration](../source-distribution-integration.md)
for field ownership and the complete evidence tuple.

## Human authority and mutation intents

Authority status, exact mutation previews, and single-use mutation intents are
Connect-RPC methods on `HumanControlService`:
`GetAuthorityStatus`, `PrepareMutation`, and `ConfirmMutation`. There is no
parallel REST authority surface. The typed RepoService commit method consumes
the returned `intent_id` before invoking the Git writer.

## Same-origin sign-in facade

`AuthService.Login` is the browser entry point for GCT sign-in. The GCT API
forwards the request to scenario-authenticator's typed `AccountsService` and
relays the short-lived access token; it does not store credentials, mint
tokens, or call the authenticator from browser JavaScript. The UI stores the
returned access token in the existing `gct_access_token` cookie so the normal
relying-party verifier can establish human authority. Refresh tokens are not
stored by GCT.

## Repo state — status, diff, history

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetRepoStatus` | Typed working-tree status, file categorisation, hotspots. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetRepoDiff` | Typed unified diff for one file or view mode. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetFiles` | Typed tracked/untracked file listing with bounded search. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetDirectoryContents` | Typed immediate directory listing. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetRelatedFiles` | Typed related-file suggestions for a path. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/SearchContent` | Typed bounded content search across tracked files. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetRepoHistory` | Typed Git log with optional grep / limit / file-detail include. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetSyncStatus` | Typed push/pull-needed assessment vs upstream. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetGroupingRules` | Typed grouping configuration read. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/SaveGroupingRules` | Typed grouping configuration save with exact single-use intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetGitignoreHealth` | Typed .gitignore health analysis. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/MoveGitignoreEntry` | Typed .gitignore remediation with exact single-use intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetTrackedBinaries` | Typed tracked-binary analysis. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/UntrackBinary` | Typed binary untracking with exact single-use intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetApprovedChanges` | Typed approved-change catalog or path-scoped preview. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetProvenance` | Typed AI-attribution lookup by run. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetBlame` | Bounded typed native blame joined with historical sandbox evidence, work references, standings, and change bundles. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/SearchProvenance` | Typed bounded exact/lexical provenance search with explicit repository scope. |
| GET | `/api/v1/collaboration/status` | Host-neutral capability status; configured identity is distinct from active Integration Hub access. |
| POST | `/vrooli.git_control_tower.v1.advisory.AdvisoryService/Draft` | Typed evidence-bound summary/commit/PR/release draft envelope. |
| POST | `/api/v1/advisory/mention` | `webhook_receiver` exception: authenticated normalized provider mention admission with durable event deduplication; no provider transport or reply publication. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/SaveFileContent` | Typed file save with exact single-use intent and optimistic concurrency. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/DeletePath` | Typed path deletion with exact single-use intent. |

## Repo registry (multi-repo prep)

Repository registry reads and mutations use `RepoService`:
`ListRepositories`, `GetActiveRepository`, `SetActiveRepository`,
`OpenRepository`, `CloneRepository`, and `RemoveRepository`.

## Staging, commit, discard, ignore

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/DiscardFiles` | Typed discard with exact single-use intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/IgnorePath` | Typed .gitignore update with exact single-use intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetRepoGroups` | Typed changed-file groups ordered as manual rules, contract targets by fixed kind, then `Other`. |

### Repo history includes

`RepoService/GetRepoHistory` accepts:
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
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/PushToRemote` | Typed remote push with exact intent and step-up policy. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/PullFromRemote` | Typed remote pull with exact intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/RunUpstreamAction` | Typed bounded upstream action with exact intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetPrecommitConfig` | Typed precommit configuration read. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/SavePrecommitConfig` | Typed precommit configuration save with exact intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/RunPrecommit` | Typed bounded precommit execution. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/UpdateRemoteURL` | Update remote URL with exact intent. |

## Branches

| Method | Path | Notes |
| --- | --- | --- |

## Credentials & SSH

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/ListCredentials` | List stored repository credentials. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/SaveCredential` | Save a credential with exact intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/DeleteCredential` | Remove a credential with exact intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/TestCredential` | Test credential connectivity. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/ListSSHKeys` | List SSH keys. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/DeleteSSHKey` | Remove an SSH key with exact intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GenerateSSHKey` | Generate an SSH keypair with exact intent. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetSSHPublicKey` | Return a public key for paste. |
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/TestSSHConnection` | Test an SSH key against the remote. |

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
| GET    | `/api/v1/repo/test-runs/{runId}/artifacts/{artifactId}`    | Stream opaque typed run evidence; requires `scenario`. |

## Test runs and evidence (Connect-RPC)

`EvidenceService` exposes `StartRun`, `ListRuns`, `GetRun`, and `ListEvidence` on the generated Connect procedure paths. It preserves Test Genie's canonical `RunInfo`, captured descriptors, comparison/applicability metadata, and open `ArtifactRef.kind` strings without a GCT phase registry. List operations support server-side metadata filtering and pagination; artifact bodies are never embedded in list/detail responses.

## Tidiness & rules / auditor

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/repo/tidiness-scan`     | Run a tidiness scan. |
| GET  | `/api/v1/repo/tidiness-issues`   | Pending issues. |
| GET  | `/api/v1/repo/tidiness-scenario` | Per-scenario summary. |
| GET  | `/api/v1/repo/tidiness-score`    | Aggregate score. |
| GET  | `/api/v1/repo/tidiness-staleness`| Staleness metrics. |
| GET  | `/api/v1/repo/blame`            | Bounded native Git line attribution (`path` may repeat; optional `revision`, `start_line`, `end_line`, `enrich=true`). Enrichment preserves exact-content, commit-only, run-overlap, private, unavailable, and unknown standings; path overlap never proves authorship. |
| POST | `/vrooli.git_control_tower.v1.auditor.AuditorService/StartCheck` | Start auditor rules. |
| POST | `/vrooli.git_control_tower.v1.auditor.AuditorService/GetJobStatus` | Auditor job status. |
| POST | `/vrooli.git_control_tower.v1.auditor.AuditorService/ListRules` | Auditor rules catalog. |
| POST | `/vrooli.git_control_tower.v1.auditor.AuditorService/PreviewFix` | Typed advisory auditor-fix preview. |
| POST | `/vrooli.git_control_tower.v1.auditor.AuditorService/ApplyFix` | Typed auditor fix with exact single-use intent; no REST writer remains. |
| POST | `/vrooli.git_control_tower.v1.auditor.AuditorService/ListViolations` | Outstanding violations. |

The typed auditor methods currently proxy the separately owned
`scenario-auditor` scenario's existing HTTP client contract. `scenario-auditor`
does not yet publish an owner proto/Connect surface, so that inter-scenario
hop remains an explicit follow-up; no GCT UI, CLI, or GCT REST auditor writer
depends on it.

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
| POST | `/vrooli.git_control_tower.v1.review.ReviewService/Start` | Typed start for a scenario-review run. |
| GET  | `/api/v1/review/run/{jobId}`    | Job status / results. |
| GET  | `/api/v1/review/summary`        | Summary across runs. |

## Scenarios & audit log

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/api/v1/scenarios`                       | List scenarios. |
| GET | `/api/v1/scenarios/{slug}/envelope`       | Scenario envelope info. |
| GET | `/api/v1/audit`                           | Query audited operations. |
