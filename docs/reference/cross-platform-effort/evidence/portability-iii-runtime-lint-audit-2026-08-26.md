# Portability III Runtime Lint Audit — 2026-08-26

The AST rules are wired into `make lint-portability` and every current finding is dated in `.vrooli/portability-lint-allowlist.json`. These are recorded exceptions, not evidence that the underlying boundary is portable.

## Unguarded shell findings

| File | Line | Verdict |
|---|---:|---|
| `scenarios/app-monitor/api/services/metrics_service.go` | 148 | allowlisted pending split or native replacement |
| `scenarios/app-monitor/api/services/metrics_service.go` | 160 | allowlisted pending split or native replacement |
| `scenarios/app-monitor/api/services/metrics_service.go` | 217 | allowlisted pending split or native replacement |
| `scenarios/app-monitor/api/services/metrics_service.go` | 246 | allowlisted pending split or native replacement |
| `internal/safeguards/vrooli-launcher/handler_test.go` | 119 | allowlisted pending split or native replacement |
| `internal/safeguards/vrooli-launcher/handler_test.go` | 149 | allowlisted pending split or native replacement |
| `internal/process/process_linux_test.go` | 11 | allowlisted pending split or native replacement |
| `packages/cli-core/cliapp/scenario_app_test.go` | 693 | allowlisted pending split or native replacement |
| `scenarios/scenario-to-cloud/api/vps/native_cli_test.go` | 205 | allowlisted pending split or native replacement |
| `scenarios/test-genie/api/internal/orchestrator/requirements/requirements_syncer_test.go` | 70 | allowlisted pending split or native replacement |
| `scenarios/test-genie/api/internal/orchestrator/requirements/requirements_syncer_test.go` | 150 | allowlisted pending split or native replacement |
| `scenarios/algorithm-library/api/local_executor.go` | 199 | allowlisted pending split or native replacement |
| `scenarios/algorithm-library/api/local_executor.go` | 368 | allowlisted pending split or native replacement |
| `scenarios/algorithm-library/api/local_executor.go` | 455 | allowlisted pending split or native replacement |
| `scenarios/agent-manager/api/internal/adapters/runner/exit_code_test.go` | 24 | allowlisted pending split or native replacement |
| `scenarios/agent-manager/api/internal/adapters/runner/idle_scanner_test.go` | 47 | allowlisted pending split or native replacement |
| `scenarios/agent-manager/api/internal/adapters/runner/idle_scanner_test.go` | 73 | allowlisted pending split or native replacement |
| `scenarios/agent-manager/api/internal/adapters/runner/idle_scanner_test.go` | 108 | allowlisted pending split or native replacement |
| `scenarios/agent-manager/api/internal/adapters/runner/idle_scanner_test.go` | 139 | allowlisted pending split or native replacement |
| `scenarios/agent-manager/api/internal/adapters/runner/idle_scanner_test.go` | 192 | allowlisted pending split or native replacement |
| `scenarios/content-desk/api/internal/claims/check_runner.go` | 47 | allowlisted pending split or native replacement |
| `scenarios/scenario-to-desktop/api/screenrecording/media_test.go` | 49 | allowlisted pending split or native replacement |
| `scenarios/scenario-to-desktop/api/screenrecording/media_test.go` | 51 | allowlisted pending split or native replacement |
| `scenarios/api-library/api/main.go` | 2263 | allowlisted pending split or native replacement |
| `scenarios/vrooli-onboarding/api/coverage_gaps_test.go` | 165 | allowlisted pending split or native replacement |

The plan inventory described 19 shell-out sites. The AST rule currently reports 25 syntax findings (including repeated calls and test fixtures); the allowlist preserves each exact file/line so newly introduced calls fail the gate.
