# Configuration authority retirement ledger

This ledger records the phase 19 ownership audit. It distinguishes durable
configuration from derived observations and from bootstrap markers so a future
cleanup does not merge unrelated state into a new parallel authority.

| Concern | Current authority | Current readers/consumers | Retired or rejected path | Evidence |
|---|---|---|---|---|
| Operator choices and forward-compatible fields | `internal/operatorstate` writing `.vrooli/operator-state.json` | onboarding API/UI/CLI, resource catalog, host requirements, capacity, trust posture, lifecycle readers | Direct file writes outside `internal/operatorstate` | `internal/operatorstate/operatorstate_test.go:TestOnlyOperatorStatePackageWrites` |
| Scenario/resource enablement | `operator-state.resources.*` and `operator-state.scenarios.*` | `internal/resources/catalog`, lifecycle dependency resolution, autoheal supervision projection | Mutable `.vrooli/service.json` values after one-time adoption | `internal/resources/resources.go`, catalog tests |
| Core supervision membership | `vrooli supervision-set` projection from `operator-state.core` | autoheal bootstrap/reconcile, CLI consumers | Autoheal-owned membership lists or duplicated scenario-specific sets | `internal/app/supervision`, `scenarios/vrooli-autoheal/api/internal/bootstrap/supervision.go` |
| Apply operation and completion digest | onboarding apply service plus `operator-state.completion` | onboarding readiness/session, CLI/API/UI peers | Client-local completion flags or a second apply authority | onboarding completion and apply-plan tests |
| Bootstrap completion | `internal/projectstate` `.bootstrap-complete` marker | setup status and setup readiness | Treating bootstrap completion as configuration completion | `internal/projectstate/setup_markers.go` |
| Configuration completion handoff | `internal/projectstate` `.configuration-complete` marker, written only after onboarding’s verified apply | setup status/result and deployment handoff consumers | A bootstrap phase claiming configuration completion before readiness | `internal/setup/readiness_test.go`, `scenarios/vrooli-onboarding/api/apply.go` |
| Remote handoff | typed `setup/v1.Selection` and Bridge target-aware onboarding procedure | vrooli-bridge and onboarding target proxy | Bridge-owned shadow operator state or onboarding-owned transport/session state | Bridge onboarding client/handler tests |
| Draft/session pointer | operator-state service `drafts` and `session` fields | onboarding session API and UI/CLI re-entry | Browser-only progress or an independent CLI progress database | `internal/operatorstate/operatorstate_test.go` draft tests |

## Migration and recovery rules

- Unknown operator-state fields are retained by `RawFields`; an older binary
  cannot silently erase fields owned by a newer binary.
- A stale revision fails with `ErrRevisionConflict`; the caller reloads the
  authoritative document and retries a field-scoped patch.
- Drafts are target- and actor-scoped, reject secret-like keys, and remain
  inspectable after an interrupted save.
- Bootstrap and configuration markers are intentionally separate. A missing,
  malformed, or unreadable marker is reported as pending or degraded; it is not
  inferred from a process exit or a partial apply.
- A failed or interrupted migration remains retryable through the projectstate
  migration ledger. No migration-only reader is retained as an undocumented
  runtime fallback.

No deletion is pending from this audit. The legacy paths above are either
retired by the current typed readers or are explicitly retained as distinct
bootstrap/configuration marker contracts with separate owners.
