# Independent review — routed CLI actor propagation test harness

## Native provenance

- Reviewer task/run: `8c3a10cb-8f9a-4c91-b807-1214732b9ab0` / `dd1291f7-281c-40d5-aae1-75dcfae77f17`.
- Reviewer profile: `b93d5233-4867-4569-8371-171bea404b7e` (`prompt-manager/heartbeat-judgment`), exact parent `5d07e651-eaa6-48ec-89b4-5dc359059193`, COMPLETE/exit 0/APPROVED.
- Selected final candidate: `2da5e07e-36ef-4927-b974-e36e5007b9af`; native final-output selection is SELECTED (`unique_terminal_main_assistant`).
- Worker task/run: `9924b9af-32d1-45de-b0c6-c2dd8e1dce4a` / `69ef5a17-940f-4bd0-b09e-4cdfc4d8aa49`, registered profile `4c19aef1-36a2-4cc5-8961-42648692883b`, exact parent above, COMPLETE/exit 0. Worker final-output selection was ambiguous; review relies on source bytes, native tool evidence and durable receipt files.

## Scope and checks

Baseline manifest SHA-256 `5a18e4b7557efa37bd8f1e3476ff789f6714891c575b1f83cfd61096604ff28d`; current manifest SHA-256 `ae9886a0a1778c87fc01e25232587e94480a7b403efd8235ee8a8caaaee9a0a0`. Only `api/handlers/calendar/connect_events_test.go` changed, from 506 to 514 lines (+8), current SHA-256 `421c22171f00b2ca39ea5abac079abd161a0c4e0ddeaeddf6628ae21903404cb`. `cli/app.go` and `cli/domains/calendar/handlers.go` match the baseline. Exact patch and full manifests are retained beside this report.

## Selected reviewer output (verbatim)

## PASS — bounded test-harness correction

The current test file’s SHA-256 matches the supplied value, and comparing it with the retained baseline bytes confirms the change is limited to the reported eight lines: **+8 lines**. The baseline and current manifest hashes also match the supplied values. The patch changes only `connect_events_test.go`; I made no edits and did not rerun tests.

Filtering inherited `VROOLI_AGENT_IDENTITY_TOKEN` and API-token variables from the CLI and UI child environments is appropriate for this harness: it prevents ambient credentials from reaching a server configured for synthetic bearer authentication, while the test explicitly supplies its synthetic bearer token. This isolates the fixture without changing product authentication behavior.

The actor assertion accepts only `synthetic-planner-actor` and `synthetic-planner-owner-b`, rejects every other observed actor, and requires owner B to have been observed. That preserves unknown-actor rejection while covering the test’s cross-owner requests.

The retained focused-run receipt matches its supplied hash and reports both calendar packages `ok`, with empty stderr. That supports accepting this bounded correction. It does **not** establish full E1/Q1 acceptance: the epoch receipt explicitly says the broader gates remain open. Nor does this synthetic harness qualify real human/agent identity behavior across surfaces.
