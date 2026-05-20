### Window inspected
2026-05-19 durable autoheal incidents, autoheal status, latest 50 autoheal actions, 24h trends/transitions, and action timeline.

### Signal counts
42 open incidents: 30 host_integrity, 11 unclean_boot, 1 autoheal_failure. Autoheal status: 62 checks, 50 ok, 10 warning, 2 critical. Latest 50 actions: 22 successes, 28 failures: 26 `exit status 1`, 2 `action timed out`.

### Signal picked
Scenario restart setup/build prerequisite failures.

### Pattern observed
18 of latest 50 actions failed with `missing go.sum entry`: 8 `scenario-git-control-tower`, 5 `scenario-flow-verifier`, 4 `scenario-web-console`, 1 `scenario-tidiness-manager`. Most cited `github.com/go-chi/chi/v5` from `packages/api-core/connectx`; one cited `github.com/santhosh-tekuri/jsonschema/v5` from `packages/cli-core/cliapp`. Affected scenarios later reported healthy, so generic action failure overstates unresolved outage and hides a shared remediation blocker.

### Hypothesized root cause
Autoheal restart handling does not classify lifecycle setup/build prerequisite failures as deterministic remediation blockers keyed by package/module/source state, so retries continue as generic `exit status 1`.

### Proposed action
Raised `dec-1779233619542808422`: classify missing module-sum/setup-build failures as `permanent_build_prerequisite_failure`, suppress duplicate restart attempts until source state changes, keep health separate from remediation capability, and route a scenario-quality/platform-package payload.

### Measurement plan
Track action id, scenario, phase, step, error class, missing module/package, package path, source-state fingerprint, duplicate suppression decision, last health transition, post-action health status, and recovery path.

### Missing CLI or telemetry surfaces
Action history lacks structured `phase`, `step`, `package/module`, `source_fingerprint`, `duplicate_of`, and `post_action_health`.

### Decisions raised
`dec-1779233619542808422`.

### Knowledge entries written
`knw-1779233607585587538` on `runtime-health-audit/2026-05-19`.