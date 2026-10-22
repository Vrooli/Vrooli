# Bounded authenticated workspace ownership

Owner/run: personal-planner-delivery-orchestrator; exact parent `5d07e651-eaa6-48ec-89b4-5dc359059193`. Epoch: `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`; read its opening outcome/limits, actor-workspace qualification, and last 20 slice-log entries. Worker protocol: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`.

## Outcome

Bind new planning profiles and availability settings to the authenticated workspace principal. Current evidence: `api/internal/workspace/schema.sql` has singleton `planning_profiles(id='default')` and global availability tables without owner keys; E1 records actor-to-workspace ownership as unverified. This slice is independent of legacy record mapping.

## Authority and boundaries

- Existing approved Planner I18 scope: private authenticated workspace and identity parity, `GOAL.md`, `CONTRACT.md`, `AUTHORIZATION.md`; legacy decision is `separate_until_mapped`.
- Keep the legacy singleton profile and availability rows untouched and inaccessible to authenticated owners. Never copy, assign, expose, or mutate existing ownerless rows. New actor-owned workspace state starts from the documented defaults.
- Bind only verified principal subject from the existing identity context. Missing/unverified identity must fail closed; never accept a client-supplied owner.
- Exact write set: existing `scenarios/personal-planner/api/internal/workspace/schema.sql`, `sqlite.go`, `types.go`, plus new `sqlite_test.go` only. No Proto, generated files, CLI/UI source, external dependencies, credentials/OAuth, provider calls, or production/user data. Use temporary SQLite fixtures only.
- Estimate +180 runtime/+120 test lines; stop before cumulative E1 runtime growth exceeds +2,031 (current +1,731; hard ceiling +2,100) or before any need to widen scope. Preserve unrelated workspace data and concurrent edits.

## Required behavior and checks

- Two verified owners receive isolated profiles and availability; same-owner read/update preserves identity and revision guards.
- Legacy ownerless fixture rows remain byte/value unchanged and are not returned to either owner.
- Unknown/missing actor never falls back to the legacy singleton or a shared owner namespace.
- Capture exact admitted-path baseline hashes before writes. Add focused temporary-SQLite regressions and run `cd scenarios/personal-planner/api && go test ./internal/workspace -count=1`; report any module/build blocker exactly. Do not rerun Test Genie or lifecycle startup.
- Append one truthful E1 slice-log line; run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your exact run ID> --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`. Do not mark E1 accepted.

## Exit

The bounded behavior and focused tests pass, or return exact failing gate/source evidence without claiming the ownership gap closed. This slice does not prove browser UI/CLI/Connect durable-ID parity, full planning/domain journey, seven-phase Test Genie, LeaseStats, or E1/Q1 acceptance.
