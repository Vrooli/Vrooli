# Meta-Orchestrator Summary

## Source

Follow-up from `execute/deployment-manager-lpbs-desktop-release-orchestration` completion. The CLI `releases verify --deep` already exists but `releases start` does not expose deep-verify. Two-step flow (start, then re-verify with --deep) is awkward for operators who already know this is a critical release.

## Codebase Inspection Already Done

- `scenarios/deployment-manager/cli/releases/commands.go:202-217` — CLI `verify` subcommand has a `--deep` flag that sets `q.Set("deep", "true")` on the request to `POST /api/v1/releases/{id}/verify`. The `start` subcommand does not have this flag.
- `scenarios/deployment-manager/api/releases/handlers.go` — The `Verify` handler reads `deep` from the query string. The `Start` handler's `StartRequest` struct has no `Deep` field.
- `scenarios/deployment-manager/api/releases/types.go` — `StartRequest` has `Channel`, `GitCommitHash`, `ReleaseVersion`, `ReleaseNotes`, `ReleasedBy`, `Platforms`. No `Deep`.
- `scenarios/deployment-manager/api/deployments/orchestrator_release.go:144-171` — `deployVerifyUpdateEndpoints` constructs `LPBSVerifyRequest` without setting `Deep`. The LPBS client honors the field; the orchestrator just never sets it.
- `scenarios/deployment-manager/api/deployments/lpbs_release_client.go:57` — `LPBSVerifyRequest.Deep bool` exists and is already wired through to `q.Set("deep", "true")` when present.

## Decisions Made

- Scope: thread a `Deep` boolean from the start request through to the orchestrator's verify step.
- All the LPBS-side plumbing is already in place; this is pure config-routing inside DM.

## Unresolved Questions Deferred To Workshop

- **Payload field name**: `deep` (matches existing verify endpoint query param) vs. `deep_verify` (more explicit). Recommendation: `deep` for consistency with the verify endpoint.
- **Default behavior**: Always lightweight (current) vs. profile-configurable default. Recommendation: lightweight default, explicit opt-in via flag.

## Dependency Notes

Depends on `execute/deployment-manager-lpbs-desktop-release-orchestration` (completed). No other blockers.

## Effort Assessment

Genuinely XS: one field on `StartRequest`, one line in the orchestrator that sets `Deep: ds.req.Deep`, one CLI flag, and tests. Roughly 20-30 lines plus 2 tests.
