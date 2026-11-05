# Installs

External tools installed for this goal, authorized by the orchestrator
(`large-effort-orchestration` §5, D24). Dependency installs go through Scenario
Dependency Analyzer. Before the goal completes, stop everything it started outside
the project and uninstall large goal-only installs.

| Tool | State | Size | Why | Cleanup rule |
|---|---|---|---|---|
| `test-genie` CLI (`~/.vrooli/bin/test-genie`) | Present; auto-provisioned by a project test command during E14, not explicitly authorized; used for the E14 tidiness gate | 40 MB | Scoped BAS Test Genie validation | Keep only if shared; otherwise remove before goal close |
| `tidiness-manager` CLI (`~/.vrooli/bin/tidiness-manager`) | Present; auto-provisioned by a project test command during E14, not explicitly authorized; used by the tidiness provider | 18 MB | Scoped BAS Test Genie tidiness validation | Keep only if shared; otherwise remove before goal close |
| `@stryker-mutator/core`, `jest-runner`, `vitest-runner`, `typescript-checker` ^9 | Approved in SDA 2026-09-29 (dev tooling); not installed | ~60 MB of node_modules when installed | Mutation score on key driver/UI packages before and after test-consolidation epochs | Small dev tool: may stay; remove if no consolidation epoch used it |
| `github.com/go-gremlins/gremlins` ^0.5 | Approved in SDA 2026-09-29 (dev tooling); not installed | ~20 MB binary | Mutation score on key Go API packages | Small dev tool: may stay |

Goal-owned running state (not an external install, listed so it is not forgotten):

| Item | State | Cost | Why | Cleanup rule |
|---|---|---|---|---|
| Shadow engagement `bas-goal` (second BAS instance `@shadow`, own database) | Running since 2026-09-29, no idle TTL | One extra BAS API, driver and UI process set | The journey suite and epoch workers rebuild and run changed BAS code here (D18) | Keep while the goal runs; `promote` at stable points; `git-control-tower baseline abandon --scenario browser-automation-studio --name bas-goal` if the goal is paused for long or completes |
