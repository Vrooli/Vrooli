### Slice audited
`harness`: shared Go test harness packages, primarily `packages/testkit-go`, `packages/api-core/connectxtest`, `packages/api-core/databasetest`, and `packages/cli-core/cliapptest`. Excluded scenario-owned UI/prompt-manager-specific harnesses.

### Dimension grades
Architecture B-; security B; test coverage B; documentation B-; cross-platform C+; signal C+; instrumentation C.

### Top finding
Shared scenario fixtures are only partly repo-contract-aware. `testkit-go` supports `WithScenarioDir`, and `scenariofixture.WriteScenarioService` follows `scenario.ServicePath`, but `WriteScenarioCLIGoMod` and `WriteScenarioTestPhaseFixture` still hardcode `scenarios/`. Custom-layout tests can split artifacts between `apps/<name>` and `scenarios/<name>`.

### Measurement plan
Add `scenariofixture` contract tests using `NewRepoFixture(t, WithScenarioDir("apps"))`, `WriteRepoContract`, `WriteScenarioCLIGoMod`, and `WriteScenarioTestPhaseFixture`; assert all generated scenario artifacts land under `apps/<name>`. Update helpers to resolve the scenario root through the repo contract or accept the fixture scenario dir. Add/document a wrapper or module-root verification command for separate-module harness tests.

### Plan-of-record diffs proposed
None. Queue pressure blocked a new decision.

### Decisions raised
None. Four owned-context pending decisions were visible; no supersession candidate found.

### Knowledge entries written
`knw-1779148991395060774` on `platform-code-audit/2026-05-19`.

Next run should audit `repo-contract` unless a newer runtime signal justifies an override.