# API Reference: git-control-tower

num[sot]:15 REST routes across num[sot]:12 functional groups, including the two operational
health probes. Integration capability/collaboration status, audit and staged-message metrics, review summary/job status, scenario list/envelope/isolation, source-distribution, tidiness-manager and Agent Manager profile/run reads moved to typed Connect. Source of truth:
[CODE: api/routes.go].

REST endpoints are mounted under `http://localhost:<API_PORT>` (printed
by `vrooli scenario port git-control-tower`) at `/api/v1`; Connect procedures
are mounted on the same origin at their generated service paths. The API mounts
num[sot]:18 GCT Connect services / num[sot]:155 RPCs plus ui-health's external
VisualHealth service. Typed GCT mutations are audited via [CODE: api/audit_logger.go]; retained REST exceptions are listed below with owner-specific authority and audit behavior.

### Retained REST exception register

These 15 registrations are intentionally separate from the typed repository,
review, and audit owner contracts. Keep each REST boundary until its owner
provides an appropriate typed or binary-streaming contract; this inventory is
the next-step map, not a claim that every exception is permanently closed.

| REST route group | Owner | Why REST remains / next step |
| --- | --- | --- |
| `GET /health`; `GET /api/v1/health` | Vrooli runtime / infrastructure probes | Load balancers and lifecycle probes require unauthenticated standard HTTP health. There are no current GCT runtime UI consumers (the exported `fetchHealth` helper has no caller); `/api/v1/health` is a compatibility alias. Retain as `RESTException`; no Connect caller should replace it. |
| `POST /api/v1/advisory/mention` | Provider webhook ingress (`webhook_receiver`) | Provider delivery uses its signed/normalized webhook payload and durable dedupe. Keep HTTP ingress; next step is only a provider-supported typed downstream handoff, not a Connect webhook requirement. |
| `POST /api/v1/repo/visual-capture`; `GET /api/v1/repo/visual-captures/{id}/screenshot/{filename}`; `GET /api/v1/repo/visual-captures/{id}/video/{filename}`; `DELETE /api/v1/repo/visual-captures/{id}`; `DELETE /api/v1/repo/visual-capture-storage` | GCT capture/storage with Browser Automation Studio (BAS) | Three UI hook callsites trigger/delete/clear capture storage; two components consume screenshot URLs and the video URL builder has no current caller. Capture trigger uses `RepoRead` then coordinates a long-running BAS request and writes snapshot files. The two deletes also use `RepoRead` and directly delete storage; no Connect mutation intent or audit call is currently wired for these handlers. Retain all three as route-specific `RESTException` entries until approved mutation authority/audit semantics are defined and tested; moving them unchanged would carry an unreviewed mutation contract. Screenshot/video return bytes and stay REST pending an owner streaming/download contract. Four metadata/path reads use `VisualCaptureService`; BAS remains capture owner. |
| `GET /api/v1/repo/test-runs/{runId}/artifacts/{artifactId}` | Test Genie artifact owner; GCT opaque proxy | Three UI components use the same-origin `runArtifactUrl` helper. The endpoint forwards Range and streaming headers to Test Genie, whose artifact body is opaque. Keep HTTP until Test Genie provides a compatible authenticated streaming or signed-download contract; next step is owner-defined streaming/download binding. |
| `POST /api/v1/agent/attachments/upload`; `POST /api/v1/agent/run`; `POST /api/v1/agent/runs/{id}/continue`; `POST /api/v1/agent/runs/{id}/approve`; `POST /api/v1/agent/runs/{id}/reject`; `POST /api/v1/agent/runs/{id}/stop` | Agent Manager | The UI has six REST calls here: multipart attachment upload and five human-controlled run operations. Agent Manager already offers Connect RPCs for the five operations and GCT's client uses those RPCs upstream; this same-origin browser facade is still REST. Migrating it needs a GCT typed facade with verified human identity and exact request-bound intent, preserving task/run binding and approval semantics. Attachment upload has no typed owner RPC and stays multipart HTTP. Do not bypass the GCT policy gate with a direct browser-to-Agent-Manager call. |

## Health

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/health`                       | Infrastructure health probe (root). |
| GET | `/api/v1/health`                | Same handler, client-friendly path. |

The health JSON payload is typed by
`packages/proto/schemas/git-control-tower/v1/shared/health.proto`. Health is
the intentional operational REST exception: lifecycle managers and load
balancers must be able to probe it without a Connect client.

## Integration status

Capability registry and collaboration status reads use the generated
`IntegrationService` procedures `GetCapabilities` and
`GetCollaborationStatus`. Capability entries retain their complete registry
fields. Collaboration reads preserve the host-neutral identity validation and
return `unconfigured` or `unavailable` status without claiming a live provider
connection.

| Method | Procedure | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.integration.IntegrationService/GetCapabilities` | Full GCT capability registry projection, including timestamp and owner-defined extension fields. |
| POST | `/vrooli.git_control_tower.v1.integration.IntegrationService/GetCollaborationStatus` | Host-neutral identity validation and capability standing; does not perform provider reads or writes. |

## Source distributions (read-only)

| Method | Procedure | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.source_distributions.SourceDistributionService/ListDistributions` | Projection of source-ramp candidates; optional `repository_context` and `scenario` filters; default limit 50, max 200. |
| POST | `/vrooli.git_control_tower.v1.source_distributions.SourceDistributionService/GetDistributionDetails` | Projection of distribution, contents, handoff, and drift reads. Partial evidence retains its unavailable reason. |

These methods never assemble, publish, approve, create repositories, access
credentials, or modify Git/host state. A source-ramp outage is returned as a
typed Connect `unavailable` error; a partial detail read retains the durable
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
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/GetRepoStatus` | Typed working-tree status, file categorisation, hotspots; optional `page_size`/`page_token` continuation capped at 256 KiB per response. Legacy full requests fail with ResourceExhausted when the complete response exceeds the cap. |
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
| POST | `/vrooli.git_control_tower.v1.advisory.AdvisoryService/Draft` | Typed evidence-bound summary/commit/PR/release draft envelope. |
| POST | `/vrooli.git_control_tower.v1.advisory.AdvisoryService/GetStagedMessageMetrics` | Durable failure, duration, proven zero-work refusal, duplicate retry, and stuck lifecycle aggregates. |
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

## Commit proposals (Connect-RPC)

`vrooli.git_control_tower.v1.proposals.ProposalService`
([CODE: api/proposal_connect.go], domain in [CODE: api/internal/proposals/service.go]).
Draft methods write only GCT's database and are agent-callable; the caller
principal and asserted run are recorded on every revision.

| Method | Notes |
| --- | --- |
| `AnchorScope` | Records HEAD plus raw SHA-256, canonical blob and effective Git mode of each dirty scoped file. |
| `CreateProposal` | Retains exact candidate/base bytes and modes; records flags/exclusions and trailer-linked message. Preserves older proposals; `validate_only` stores nothing. Limits: 2,000 paths, 8 MiB/file, 64 MiB captured package. |
| `ListProposals` / `GetProposal` | Live freshness/flags, destination identity, actual retained revisions. `revision` reads an immutable payload; legacy missing captures remain explicit. |
| `RepoService.GetRepoDiff` | `proposal_id`, exact positive `proposal_revision`, `path`: retained base-to-candidate diff with no working/history selectors. `compare_current` explicitly compares candidate to current files without updating proposal storage. Binary/mode/empty-file standing is explicit; diff output capped at 2 MiB. |
| `EditProposal` | Creates a revision against `expected_revision`; removing paths preserves working files/index; unknown trailers/operator order retained. |
| `CombineProposals` | 2–20 exact open captured source revisions; requires shared base/branch and matching shared-file versions. Atomic source closure/new package with retained source manifests and union provenance, never live rehash or silent truncation. |
| `WithdrawProposal` | Optional reason/replacement, exact source revision; closes only the review record, preserving files, index and history. Records paths omitted by the replacement. |
| `RefreshProposal` | Explicitly captures current contents into a new revision and drops clean/unchanged/no-contribution paths. Source evidence remains attributed rather than certified against new bytes. |
| `ApplyProposal` | Verified human only. Consumes `repo.apply_proposal` intent bound to revision, branch/HEAD, files/modes/message and index. Uncertain prior work requires `acknowledge_prior_work` and intent context suffix `;prior-work=acknowledged`. Checks/hook effects run in a private candidate; tree/message verified before ref publication under repository, Git and proposal revision leases. `keep_later_changes=true` binds `;working-files=preserved` to the intent, stages retained blobs/modes and keeps later working edits untouched. Default false refuses working drift. Both modes refuse affected base/branch/foreign-index changes; saved-version mode also refuses different staged versions on proposal paths; failed pre-publication operations restore the index. A published OID is reported truthfully even when recording fails. |

### Repo history includes

Detailed entries include `full_message`, `body`, parsed/resolved `trailers`, and
server-rendered `continuation_message`. `RepoService/GetRepoHistory` accepts:
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
| POST | `/vrooli.git_control_tower.v1.repo.RepoService/PushToRemote` | Typed remote push with exact single-use intent. |
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
| GET    | `/api/v1/repo/visual-captures/{id}/screenshot/{filename}`  | Serve screenshot bytes. |
| GET    | `/api/v1/repo/visual-captures/{id}/video/{filename}`       | Serve video bytes. |
| DELETE | `/api/v1/repo/visual-captures/{id}`                        | Delete capture. |
| DELETE | `/api/v1/repo/visual-capture-storage`                      | Clear all captures. |
| GET    | `/api/v1/repo/test-runs/{runId}/artifacts/{artifactId}`    | Stream opaque typed run evidence; requires `scenario`. |

`VisualCaptureService` owns scoped read projections and the guarded local path
lookup:

| Method | Procedure | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.visual_captures.VisualCaptureService/ListCaptures` | Repository/scenario scoped capture list and staleness. |
| POST | `/vrooli.git_control_tower.v1.visual_captures.VisualCaptureService/GetCaptureDetail` | Capture and screenshot/video metadata, without embedding bytes. |
| POST | `/vrooli.git_control_tower.v1.visual_captures.VisualCaptureService/GetStorageStats` | Repository storage accounting. |
| POST | `/vrooli.git_control_tower.v1.visual_captures.VisualCaptureService/GetScreenshotPath` | Same repository-scoped, traversal-checked local screenshot path for agent prompt consumers. |

## Test runs and evidence (Connect-RPC)

`EvidenceService` exposes `StartRun`, `ListRuns`, `GetRun`, and `ListEvidence` on the generated Connect procedure paths. It preserves Test Genie's canonical `RunInfo`, captured descriptors, comparison/applicability metadata, and open `ArtifactRef.kind` strings without a GCT phase registry. List operations support server-side metadata filtering and pagination; artifact bodies are never embedded in list/detail responses.

## Tidiness & rules / auditor

`TidinessService` provides typed score, issue-list, staleness, light-scan, and
scenario-detail operations. These methods validate repository context through
GCT's repository resolver and delegate to `tidiness-manager`; an unavailable
owner is returned as `unavailable`, not an empty successful result. Light scans
resolve a scenario path within the selected repository before calling the owner.

| Method | Procedure | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.tidiness.TidinessService/GetScore` | Owner score and optional metrics. |
| POST | `/vrooli.git_control_tower.v1.tidiness.TidinessService/ListIssues` | Filtered owner issues; default limit 100, maximum 1,000. |
| POST | `/vrooli.git_control_tower.v1.tidiness.TidinessService/GetStaleness` | Owner staleness state. |
| POST | `/vrooli.git_control_tower.v1.tidiness.TidinessService/TriggerLightScan` | Owner scan constrained to resolved scenario path. |
| POST | `/vrooli.git_control_tower.v1.tidiness.TidinessService/GetScenarioDetail` | Per-scenario owner summary. |

| Method | Path | Notes |
| --- | --- | --- |
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

`AgentReadService` exposes typed profile and run projections from Agent Manager.
GCT calls Agent Manager's generated Connect client, forwarding only request-scoped
verified owner proof; Agent Manager independently checks its `agent-manager:read`
scope and remains authoritative. Missing identity or permission stays a typed
authentication/permission failure; capability or transport outages stay
unavailable. Per-run reads require both the exact `gct-<scenario-slug>` tag and
the Agent Manager task's `scope_path` to match the selected repository's
canonical scenario path. Runs with a missing task or mismatched binding are not
exposed. `ListRuns` asks Agent Manager for its latest 50 prefix candidates,
filters exact tag and verified task scope, then applies the requested page size;
its `total` is the number returned by that bounded filtered page, not an
all-time count. Event payload details use a protobuf `Struct`; the diff remains
a JSON contract with file summaries and patch text. Upload and run mutation
routes are separate owner contracts and remain REST for now.

| Method | Procedure | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.agent_reads.AgentReadService/ListProfiles` | Agent Manager-owned profiles. |
| POST | `/vrooli.git_control_tower.v1.agent_reads.AgentReadService/ListRuns` | GCT scenario-tagged runs; default 5, maximum 50. |
| POST | `/vrooli.git_control_tower.v1.agent_reads.AgentReadService/GetRun` | One run's current owner projection. |
| POST | `/vrooli.git_control_tower.v1.agent_reads.AgentReadService/GetRunEvents` | Ordered events; default 100, maximum 500. |
| POST | `/vrooli.git_control_tower.v1.agent_reads.AgentReadService/GetRunDiff` | Run-produced diff and per-file summaries. |

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/agent/run`                       | Start an agent run. |
| POST | `/api/v1/agent/runs/{id}/approve`         | Approve a run. |
| POST | `/api/v1/agent/runs/{id}/continue`        | Continue a paused run. |
| POST | `/api/v1/agent/runs/{id}/reject`          | Reject a run. |
| POST | `/api/v1/agent/runs/{id}/stop`            | Stop a run. |
| POST | `/api/v1/agent/attachments/upload`        | Upload attachment for runs. |

## Review pipeline

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.review.ReviewService/Start` | Typed start for a scenario-review run. |
| POST | `/vrooli.git_control_tower.v1.review.ReviewService/GetSummary` | Typed summary across available review dimensions; supports optional detail count and repository context. |
| POST | `/vrooli.git_control_tower.v1.review.ReviewService/GetJobStatus` | Typed status/results read for a review job in repository context. |

The retired REST paths `/api/v1/review/summary` and
`/api/v1/review/run/{jobId}` have no route registrations. Browser and CLI
review operations use the generated ReviewService client.

## Audit log

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.audit.AuditService/ListEntries` | GCT-owned audit query with operation/branch filters, bounded page size, total and timestamp. |

The CLI audit list command uses the generated AuditService client; `/audit` is
not a registered REST route.

## Scenario metadata (Connect)

Scenario listing and envelope reads moved to `ScenarioService`:

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/vrooli.git_control_tower.v1.scenarios.ScenarioService/ListScenarios` | Typed scenario status list from the canonical CLI status contract. |
| POST | `/vrooli.git_control_tower.v1.scenarios.ScenarioService/GetScenarioEnvelope` | Typed scenario metadata and lifecycle envelope from `service.json`. |
| POST | `/vrooli.git_control_tower.v1.scenarios.ScenarioService/GetScenarioIsolation` | Typed proxy to Test Genie's eligibility contract; downstream failures retain the explicit `unknown` status. |

The retired REST paths `/api/v1/scenarios` and
`/api/v1/scenarios/{slug}/envelope` and
`/api/v1/scenarios/{slug}/isolation` have no route registrations. The
scenario picker and review panel use the generated ScenarioService client.
