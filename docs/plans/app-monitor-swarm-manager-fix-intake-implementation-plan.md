# App Monitor Swarm Manager Fix Intake Implementation Plan

## 1. Purpose

Replace App Monitor's App Issue Tracker integration with a greenfield Swarm Manager fix-backlog integration.

App Monitor should submit known-fix reports directly to Swarm Manager as `fix` backlog items, including selected diagnostics as attached evidence files. App Monitor should also show existing active and archived Swarm Manager fix backlog items for the monitored scenario.

## 2. Required Reading

Before implementing, read these files and commands:

- `prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement`
- `scenarios/swarm-manager/api/internal/backlog/handler.go`
- `scenarios/swarm-manager/api/internal/backlog/handler_create.go`
- `scenarios/swarm-manager/api/internal/backlog/service.go`
- `scenarios/swarm-manager/api/internal/backlog/files.go`
- `scenarios/swarm-manager/api/internal/fileserve/fileserve.go`
- `scenarios/swarm-manager/api/internal/scenarios/context.go`
- `scenarios/swarm-manager/cli/cmd_scenarios.go`
- `packages/proto/schemas/swarm-manager/v1/api/backlog.proto`
- `scenarios/app-monitor/api/services/app_reports.go`
- `scenarios/app-monitor/api/services/app_issues.go`
- `scenarios/app-monitor/api/services/app_types.go`
- `scenarios/app-monitor/api/handlers/apps.go`
- `scenarios/app-monitor/ui/src/components/report/useReportIssueState.ts`
- `scenarios/app-monitor/ui/src/components/report/useReportExistingIssues.ts`
- `scenarios/app-monitor/ui/src/state/scenarioIssuesStore.ts`
- `scenarios/app-monitor/ui/src/services/api.ts`

## 3. Problem Statement

App Monitor currently reports problems through App Issue Tracker. That scenario is planned for deprecation and does not match the desired future architecture. The correct destination is Swarm Manager because these reports are known fixes, not open-ended captures requiring classification.

The missing platform capability is a first-class way to create a Swarm Manager backlog item and include evidence files in the same operation. Swarm Manager already supports creating backlog items and uploading files separately, but App Monitor needs one atomic create-with-files operation so partial reports do not leave invalid backlog items behind.

## 4. Scope

In scope:

- Extend `POST /api/v1/backlog` in Swarm Manager to accept both JSON and `multipart/form-data`.
- Preserve the existing JSON create behavior.
- Add multipart create-with-files support with atomic item-and-file persistence.
- Add CLI parity for create-with-files.
- Update App Monitor to create Swarm Manager `fix` backlog items through shared API Core discovery.
- Update App Monitor's existing-issue feature to show Swarm Manager active and archived fix backlog items for the scenario.
- Update UI language, API types, tool descriptions, and tests away from App Issue Tracker concepts.
- Fail visibly when Swarm Manager is unavailable.

Out of scope:

- Duplicate detection.
- Queueing or starting agent work from App Monitor.
- Secret or PII redaction before agent use.
- App Issue Tracker adapter, migration layer, or legacy compatibility period.
- Captor-based workflows for this known-fix path.

## 5. Current Technical Context

Swarm Manager:

- `POST /api/v1/backlog` is registered in `scenarios/swarm-manager/api/internal/backlog/handler.go`.
- `Create` in `handler_create.go` currently only decodes strict proto JSON.
- Backlog item creation is centralized in `Service.Create` in `service.go`.
- Existing file upload is separate: `POST /api/v1/backlog/{kind}/{name}/files`.
- `fileserve.Upload` already validates safe relative paths and writes multipart uploads.
- `GET /api/v1/scenarios/{name}/context` already returns scenario fix history split into `fixes.active` and `fixes.archived`.
- The CLI command `swarm-manager scenarios fixes` already proves the context endpoint is the right source for scenario-tied fix history.

App Monitor:

- `ReportAppIssue` in `api/services/app_reports.go` builds the right evidence set today, but posts to App Issue Tracker.
- `ListScenarioIssues` in `api/services/app_issues.go` fetches App Issue Tracker summaries.
- The UI report flow expects `issue_id` and `issue_url`.
- The existing-issues store and report dialog use issue-tracker naming and open/active issue counts.
- `api/toolregistry/issue_tools.go` still describes App Issue Tracker behavior.

## 6. Target End State

App Monitor submits a fix report by calling Swarm Manager:

- Method: `POST /api/v1/backlog`
- Content type: `multipart/form-data`
- Metadata part: `item`, containing JSON for `CreateBacklogItemRequest`
- Manifest part: `files_manifest`, containing JSON entries that map multipart file fields to explicit relative paths
- File parts: one part per evidence file named by the manifest

The created Swarm Manager item:

- `kind`: `fix`
- `title`: generated from the App Monitor report summary
- `description`: generated from the report and evidence index
- `tags`: include `app-monitor`, the scenario name, and diagnostic/source tags
- `acceptance_allow`: `["scenarios/<scenario-name>/**"]`
- `acceptance_deny`: empty unless App Monitor later gains an explicit deny selector
- `priority`: mapped from App Monitor report severity or defaulted consistently
- `status`: created/open backlog state, not queued

The evidence files use explicit relative paths:

- `evidence/report.json`
- `evidence/console.json`
- `evidence/network.json`
- `evidence/health.json`
- `evidence/status.txt`
- `evidence/lifecycle.txt`
- `evidence/screenshot.png`
- `evidence/element-01.png`
- `evidence/element-02.png`

The App Monitor API returns Swarm Manager identity, not App Issue Tracker identity:

```json
{
  "kind": "fix",
  "name": "example-fix-name",
  "url": "http://localhost:<swarm-manager-port>/backlog/fix/example-fix-name"
}
```

App Monitor's existing-fixes view reads Swarm Manager scenario context and shows:

- Active fix backlog items by default.
- Archived fix backlog items when the archived toggle is enabled.
- Counts for active, archived, and total fixes.

## 7. Implementation Strategy

### Phase 1: Swarm Manager Multipart Create Contract

Update `backlog.Handler.Create` to branch on request content type:

- `application/json`: use the current strict proto JSON path unchanged.
- `multipart/form-data`: call a new multipart parser and service method.
- Unsupported content type: return a clear `415 Unsupported Media Type`.

Define the multipart contract:

```json
{
  "files": [
    {
      "field": "evidence_report",
      "path": "evidence/report.json",
      "content_type": "application/json"
    },
    {
      "field": "screenshot",
      "path": "evidence/screenshot.png",
      "content_type": "image/png"
    }
  ]
}
```

Validation rules:

- `item` is required and must decode as `CreateBacklogItemRequest`.
- `files_manifest` is required when files are present.
- Every manifest entry must reference exactly one uploaded file field.
- Every file must have an explicit relative destination path.
- Absolute paths, `..`, empty paths, directory targets, and unsafe paths are rejected.
- `spec.json` is protected and cannot be provided as an attached file.
- Duplicate destination paths are rejected.
- The upload size limit should initially match the existing file upload limit unless product requirements prove screenshots need more.

### Phase 2: Atomic Backlog Create With Files

Add a service-level operation instead of putting persistence logic in the handler:

- Candidate shape: `Service.CreateWithFiles(item *smpb.BacklogItem, files []PendingFile, ctx CreationContext)`.
- Keep request parsing in the handler and validation/persistence in the service or a focused helper under the backlog package.
- Stage all uploaded files before making the item visible.
- Validate the item and all file paths before writing the final item directory.
- Write `spec.json` and evidence files as one logical operation.
- If any write, validation, initiative attach, or side-effect step fails, roll back the item directory and any newly attached initiative reference.

Side-effect ordering:

1. Validate item and files.
2. Create item directory in a temporary or rollback-safe form.
3. Write `spec.json` and evidence files.
4. Finalize visibility of the item directory.
5. Attach initiative references if needed.
6. Index and emit events.
7. Trigger the same create-time workshop behavior as normal create, unless explicitly disabled by future settings.

This preserves normal backlog-create semantics while guaranteeing App Monitor does not create partial fix reports.

### Phase 3: Swarm Manager CLI Parity

Add CLI support for create-with-files to keep CLI and API capabilities aligned.

Suggested command shape:

```bash
swarm-manager backlog create --data item.json --attach evidence/report.json=/tmp/report.json --attach evidence/screenshot.png=/tmp/screenshot.png
```

Rules:

- `--data` uses the same JSON request shape as the API `item` part.
- `--attach <dest=src>` is repeatable.
- The CLI constructs the same multipart request as App Monitor.
- CLI output should include `kind`, `name`, and URL.

Do not add App Monitor-specific behavior to the Swarm Manager CLI.

### Phase 4: App Monitor API Cutover

Replace App Issue Tracker client code with a Swarm Manager client.

Implementation details:

- Resolve Swarm Manager through shared API Core discovery, not hard-coded ports.
- Introduce a small App Monitor service seam for scenario URL resolution so tests can inject a fake Swarm Manager server.
- Rename issue-oriented service methods and types to fix-oriented names.
- Keep route names only if external API compatibility is intentionally required; otherwise prefer greenfield routes such as:
  - `GET /apps/:id/fixes`
  - `POST /apps/:id/fixes/report`
- Remove App Issue Tracker constants, URLs, and response fields from the active path.
- Return a hard error when Swarm Manager cannot be resolved or rejects the report.

Report mapping:

- Build a `CreateBacklogItemRequest` with `kind = "fix"`.
- Generate a stable slug from the title, with a collision-safe suffix.
- Set `acceptance_allow = ["scenarios/<scenario-name>/**"]`.
- Include all currently selected App Monitor diagnostics as raw evidence files.
- Include `evidence/report.json` as the canonical machine-readable report index.
- Use the existing lifecycle, console, network, health, app status, screenshot, and element capture collection logic where it is still well-shaped.

### Phase 5: App Monitor Existing Fixes View

Replace `ListScenarioIssues` with Swarm Manager scenario-context loading:

- Call `GET /api/v1/scenarios/{scenarioName}/context`.
- Read `fixes.active` and `fixes.archived`.
- Return a fix history response to the UI with active and archived arrays.
- Include Swarm Manager item URL construction in one place.

UI changes:

- Rename issue-oriented state, services, and visible copy to fix backlog language.
- Default the list to active fixes.
- Add an archived toggle.
- Show active, archived, and total counts.
- Update the report success state to show the created Swarm Manager fix item.
- Remove "Open issue tracker" wording.

### Phase 6: Documentation and Contract Updates

Update Swarm Manager docs:

- `scenarios/swarm-manager/docs/reference/api-endpoints.md`
- `scenarios/swarm-manager/docs/reference/cli-commands.md`

Update App Monitor docs or tool descriptions:

- `scenarios/app-monitor/api/toolregistry/issue_tools.go`
- Any App Monitor user-facing docs that mention App Issue Tracker.

The docs should describe the multipart contract, evidence path convention, hard failure behavior, and scenario fix-history source.

## 8. Contract Decisions

- Atomicity: required for create-with-files.
- Multipart support: extend existing `POST /api/v1/backlog`; do not add a separate App Monitor-specific endpoint.
- Evidence paths: explicit relative paths only.
- Destination kind: `fix`.
- Action: create only; do not queue execution.
- Existing fixes: read active and archived fix backlog items from Swarm Manager scenario context.
- Targeting: use `acceptance_allow`, specifically `["scenarios/<scenario-name>/**"]`.
- Evidence handling: accept raw evidence.
- Cutover: no adapter or legacy App Issue Tracker period.
- Failure mode: fail visibly.
- Response identity: return `kind`, `name`, and `url`.

## 9. Testing Plan

Swarm Manager API tests:

- JSON `POST /api/v1/backlog` still creates a backlog item exactly as before.
- Multipart create succeeds with valid `item`, `files_manifest`, and evidence file parts.
- Multipart create writes `spec.json` and all evidence files to the backlog item directory.
- Multipart create rejects malformed item JSON.
- Multipart create rejects missing manifest entries.
- Multipart create rejects uploaded files not listed in the manifest.
- Multipart create rejects duplicate destination paths.
- Multipart create rejects traversal paths, absolute paths, empty paths, directory paths, and `spec.json`.
- Multipart create rolls back the item directory when any file validation or write fails.
- Multipart create rolls back initiative references if initiative attachment fails after file staging.
- Response includes `kind`, `name`, and item URL or enough fields for clients to build the URL.

Swarm Manager CLI tests:

- `backlog create --data item.json --attach dest=src` sends multipart and creates files.
- Invalid attach syntax fails before making an API call.
- CLI JSON output includes the created item identity.

App Monitor API tests:

- Report submission resolves Swarm Manager through the discovery seam.
- Report submission sends multipart data to `POST /api/v1/backlog`.
- Report submission sets `kind = fix`.
- Report submission sets `acceptance_allow = ["scenarios/<scenario-name>/**"]`.
- Report submission includes selected evidence files at the expected relative paths.
- Report submission returns `{ kind, name, url }`.
- Swarm Manager resolution failure returns a visible error.
- Swarm Manager non-2xx response returns a visible error.
- Existing fixes loading calls `/api/v1/scenarios/{scenarioName}/context`.
- Existing fixes loading maps both `fixes.active` and `fixes.archived`.

App Monitor UI tests:

- Report dialog calls the new fix-report service and displays the created fix item.
- Existing fixes panel defaults to active fixes.
- Archived toggle shows archived fixes.
- Counts are correct for active, archived, and total fixes.
- UI copy no longer points users to App Issue Tracker.

End-to-end validation:

```bash
go test ./scenarios/swarm-manager/api/internal/backlog ./scenarios/swarm-manager/api/internal/scenarios
go test ./scenarios/app-monitor/api/...
cd scenarios/swarm-manager && make test
cd scenarios/app-monitor && make test
vrooli scenario test swarm-manager
vrooli scenario test app-monitor
```

Use focused tests first, then scenario-level tests as final gates.

## 10. Rollout and Validation Checklist

- Swarm Manager accepts JSON backlog creates unchanged.
- Swarm Manager accepts multipart backlog creates with files.
- Multipart failure leaves no visible partial item.
- App Monitor no longer resolves or calls App Issue Tracker.
- App Monitor creates a Swarm Manager `fix` backlog item with evidence files.
- App Monitor fails visibly when Swarm Manager is down.
- App Monitor existing-fixes view shows active fixes.
- App Monitor archived toggle shows archived fixes.
- Documentation matches implemented API and CLI behavior.
- Scenario tests pass for Swarm Manager and App Monitor.

## 11. Risks and Mitigations

- Risk: atomic rollback is incomplete because `Service.Create` currently mixes persistence and side effects.
  - Mitigation: implement create-with-files at the service layer with explicit side-effect ordering and rollback for initiative references.

- Risk: multipart file path handling duplicates or weakens existing upload validation.
  - Mitigation: reuse existing file safety helpers and add tests for all rejected path forms.

- Risk: App Monitor UI and API retain issue-tracker names that make future maintenance confusing.
  - Mitigation: rename active code paths to fix/backlog terminology during the hard cutover.

- Risk: large screenshots exceed the upload limit.
  - Mitigation: start with the existing upload limit for consistency, test visible failure behavior, and raise the limit only with an explicit product decision.

- Risk: create-time workshop behavior surprises App Monitor users.
  - Mitigation: document that multipart create preserves normal Swarm Manager create semantics but does not call queue/start APIs.

## 12. Non-Goals and Prohibited Patterns

- Do not call App Issue Tracker from App Monitor.
- Do not add an App Issue Tracker adapter.
- Do not use captors for this known-fix workflow.
- Do not hard-code Swarm Manager ports.
- Do not add App Monitor-specific fields to the generic Swarm Manager backlog model unless a general backlog concept requires them.
- Do not attach files with implicit paths derived only from browser filenames.
- Do not create a backlog item and then upload evidence in a separate App Monitor workflow.
- Do not silently ignore Swarm Manager failures.

## 13. Definition of Done

- Swarm Manager `POST /api/v1/backlog` supports both JSON and multipart create requests.
- Multipart create persists the backlog item and evidence files atomically.
- Multipart create has focused tests for success, validation failures, protected paths, and rollback.
- Swarm Manager CLI can create a backlog item with attached files through the same API path.
- App Monitor creates Swarm Manager `fix` backlog items with raw evidence files.
- App Monitor uses shared API Core discovery for Swarm Manager.
- App Monitor existing-fixes UI shows active and archived scenario fix backlog items from Swarm Manager context.
- App Monitor returns and displays `{ kind, name, url }`.
- App Monitor contains no active App Issue Tracker integration path for fix reporting.
- Relevant docs and tool descriptions describe Swarm Manager fix backlog behavior.
- Focused unit tests and scenario-level validation pass.
