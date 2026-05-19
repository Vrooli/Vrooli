### Window inspected
2026-05-18 durable autoheal incidents, autoheal status, latest 50 autoheal actions, 24h trends/transitions, platform capabilities, action help, and direct `sudo -n true` probe.

### Signal counts
38 open incidents: 27 host_integrity, 10 unclean_boot, 1 autoheal_failure. Autoheal status: 62 checks, 48 ok, 10 warning, 4 critical. Latest 50 actions: 16 successes, 34 failures: 18 `action timed out`, 16 `exit status 1`.

### Signal picked
Privileged infra-network / infra-dns remediation-capability failure.

### Pattern observed
`infra-network` and `infra-dns` went `ok -> critical` around 2026-05-18T04:44Z and recovered around 04:46Z. Autoheal fired `flush-arp-cache` and `restart-resolved` at 04:45:10Z, but both failed in 4-8ms because sudo required a terminal/password. Direct `sudo -n true` also failed in this runtime.

### Hypothesized root cause
Privileged host remedies are attempted without a noninteractive privilege preflight or structured `privilege_unavailable` classification, so missing host privilege is recorded as generic remediation failure.

### Proposed action
No decision raised due owned-context pending threshold. When capacity opens, propose privileged autoheal actions must preflight privilege, suppress unsupported mutation attempts, classify missing privilege separately, and route an operator-runnable artifact with rollback/post-checks.

### Measurement plan
Track action privilege requirement, preflight result, selected remediation mode, suppression vs execution decision, outage transition timestamps, action duration/error class, and natural recovery timing.

### Missing CLI or telemetry surfaces
Action history has raw sudo output but no structured `requires_privilege` or `privilege_unavailable` field. No durable incident or remediation artifact path appeared for the recovered network/DNS outage. `knowledge-add --by` still fails despite generated storage guidance; retry without `--by` worked. `prompt-manager` also warns that auto-rebuild cannot write into read-only `.vrooli/bin`.

### Decisions raised
None.

### Knowledge entries written
`knw-1779147151915581641` on `runtime-health-audit/2026-05-18`.