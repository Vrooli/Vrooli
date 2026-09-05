# Portability III Kernel Filesystem Audit — 2026-08-26

The AST rule is wired into `make lint-portability`. Current matches are dated in `.vrooli/portability-lint-allowlist.json`; platform-split files remain recorded until each source-level fallback is reviewed.

## Current matches

| File | Line | Verdict |
|---|---:|---|
| `internal/hostinventory/collector.go` | 171 | allowlisted pending split, fallback, or package-scope review |
| `internal/hostinventory/hostmemory_linux.go` | 7 | allowlisted pending split, fallback, or package-scope review |
| `internal/hostinventory/storage_candidates_unix.go` | 63 | allowlisted pending split, fallback, or package-scope review |
| `internal/resources/managed_service_supervisor.go` | 394 | allowlisted pending split, fallback, or package-scope review |
| `internal/resources/managed_service_supervisor_test.go` | 138 | allowlisted pending split, fallback, or package-scope review |
| `internal/resources/securestore/keyringdaemon_linux.go` | 37 | allowlisted pending split, fallback, or package-scope review |
| `internal/resources/securestore/keyringdaemon_linux.go` | 41 | allowlisted pending split, fallback, or package-scope review |
| `internal/resources/securestore/keyringdaemon_linux.go` | 93 | allowlisted pending split, fallback, or package-scope review |
| `internal/safeguards/nvidia-driver/handler.go` | 153 | allowlisted pending split, fallback, or package-scope review |
| `internal/safeguards/ollama-resource-controls/handler.go` | 154 | allowlisted pending split, fallback, or package-scope review |
| `internal/safeguards/pstore-native/handler.go` | 135 | allowlisted pending split, fallback, or package-scope review |
| `packages/cli-core/cliutil/agent_context.go` | 256 | allowlisted pending split, fallback, or package-scope review |
| `packages/platform-go/process_linux.go` | 102 | allowlisted pending split, fallback, or package-scope review |
| `packages/platform-go/process_linux.go` | 132 | allowlisted pending split, fallback, or package-scope review |
| `packages/platform-go/process_linux.go` | 146 | allowlisted pending split, fallback, or package-scope review |
| `packages/platform-go/process_linux.go` | 159 | allowlisted pending split, fallback, or package-scope review |
| `packages/platform-go/process_linux.go` | 87 | allowlisted pending split, fallback, or package-scope review |
| `resources/doc-parse/cli/cmd/portable-compare/main.go` | 151 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/agent-manager/api/internal/orchestration/reconciler.go` | 1176 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/agent-manager/api/internal/orchestration/reconciler.go` | 590 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/agent-manager/api/internal/orchestration/reconciler.go` | 607 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/code-facts/api/handlers/health/handler.go` | 108 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/program-runtime/api/internal/programs/runner.go` | 505 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/program-runtime/api/internal/programs/runner.go` | 524 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/scenario-to-desktop/api/procmetrics/proc_reader.go` | 165 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/scenario-to-desktop/api/procmetrics/proc_reader.go` | 173 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/storage-manager/api/hostfs/process_liveness_unix.go` | 40 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/storage-manager/api/internal/migration/reconcile/isolation_process_test.go` | 121 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/disk.go` | 122 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/disk.go` | 132 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/disk.go` | 245 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/disk.go` | 321 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/disk.go` | 387 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/network.go` | 271 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/platform_cpu_linux.go` | 16 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/platform_cpu_linux.go` | 181 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/platform_cpu_linux.go` | 215 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/platform_cpu_linux.go` | 233 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/platform_cpu_linux.go` | 234 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/platform_forkrate_linux.go` | 14 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/platform_fragmentation_linux.go` | 15 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/pressure.go` | 26 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/pressure.go` | 66 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/pressure.go` | 80 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/process.go` | 258 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/socketowners_linux.go` | 118 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/socketowners_linux.go` | 52 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/system-monitor/api/internal/collectors/socketowners_linux.go` | 57 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/unit-health/api/internal/executor/leaks_linux.go` | 13 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/unit-health/api/internal/executor/leaks_linux.go` | 32 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/vrooli-autoheal/api/internal/checks/filesystem.go` | 267 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/vrooli-autoheal/api/internal/checks/filesystem.go` | 345 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/vrooli-autoheal/api/internal/checks/system/stale_service_binary_linux.go` | 24 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/vrooli-autoheal/api/internal/checks/system/swap.go` | 207 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/vrooli-autoheal/api/internal/watchdog/watchdog.go` | 158 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/workspace-sandbox/api/internal/driver/exec/run_test.go` | 171 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/workspace-sandbox/api/internal/driver/overlay.go` | 303 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/workspace-sandbox/api/internal/driver/probe.go` | 108 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/workspace-sandbox/api/internal/driver/probe.go` | 20 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/workspace-sandbox/api/internal/driver/probe.go` | 90 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/workspace-sandbox/api/internal/namespace/namespace.go` | 120 | allowlisted pending split, fallback, or package-scope review |
| `scenarios/workspace-sandbox/api/internal/namespace/namespace.go` | 86 | allowlisted pending split, fallback, or package-scope review |

The authored plan inventory expected 53 files. The current syntactic rule reports 80 call matches across 35 files, including already platform-suffixed files and tests; this discrepancy is retained as a finding rather than hidden by reducing the scan.
