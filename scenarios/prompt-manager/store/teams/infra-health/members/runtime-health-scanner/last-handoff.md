### Window inspected
2026-05-20 durable autoheal incidents, autoheal status, latest 50 autoheal actions, 24h trends/transitions, action timeline, owned pending decisions, and prior runtime-health knowledge.

### Signal counts
46 open incidents: 34 host_integrity, 11 unclean_boot, 1 autoheal_failure. Autoheal status: 62 checks, 43 ok, 11 warning, 8 critical. Latest 50 actions: 15 successes, 35 failures: 26 `missing_go_sum`, 4 `sudo_requires_tty`, 2 `go_mod_tidy_needed`, 1 `generated_strings_stale`, 1 `lifecycle_lock`, 1 timeout.

### Signal picked
Repeat privileged infra DNS/network remediation-capability failure.

### Pattern observed
Action IDs `11025`/`11028` for `infra-network flush-arp-cache` and `11026`/`11029` for `infra-dns restart-resolved` failed on 2026-05-18 and 2026-05-19 in 4-8ms with `sudo: a terminal is required` / password required. Current 24h trends show both checks 100% ok, so remediation failed while checks recovered naturally.

### Hypothesized root cause
Autoheal action contracts do not distinguish privileged host mutation requirements or noninteractive privilege unavailability from generic command failure.

### Proposed action
No new decision raised: four runtime-health-scanner owned-context decisions are already pending. When capacity allows, propose privileged action preflight plus operator-artifact routing and recovery-aware suppression.

### Measurement plan
Track action privilege requirement, preflight result, remediation mode, action/check/action IDs, transition timestamps, current/post-action health, natural recovery timing, suppression decision, and operator artifact path.

### Missing CLI or telemetry surfaces
Action history lacks structured `requires_privilege`, `privilege_preflight`, `remediation_mode`, `duplicate_of`, `post_action_health`, and `operator_artifact_path`.

### Decisions raised
None; skipped due owned-context pending decision cap.

### Knowledge entries written
`knw-1779319945357063769` on `runtime-health-audit/2026-05-20`.