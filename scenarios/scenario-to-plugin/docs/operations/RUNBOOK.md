# Runbook — Scenario to Plugin

Operational procedures for running this scenario and responding when
something goes wrong.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and check this scenario?
- What do I do when a gate closes unexpectedly?
- What do I do when a publication or revocation goes wrong?
- What is backed up, and how is it restored?
- When do I escalate, and to whom?

## Start / Stop / Status

Always go through the lifecycle. Never run built binaries directly — that
bypasses process naming, port allocation, and health checks.

```bash
make setup                 # one-time environment preparation
make start                 # start API + UI
make status                # lifecycle metadata and health
make logs                  # recent logs
make test                  # scenario suite
make stop
```

Useful read-only checks:

```bash
scenario-to-plugin status                       # API health and dependency reachability
scenario-to-plugin readiness --json             # fleet publish-readiness
scenario-to-plugin package show <package-id>    # gate ladder position for one package
scenario-to-plugin publish status <package-id>  # per-channel publication state
```

## Common Incidents

| Symptom | Likely cause | First check | Action |
|---|---|---|---|
| Every drift check fails at once | `cli-health` unreachable, or the pinned manifest revision is unavailable | `scenario-to-plugin status`; `cli-health status` | Restore `cli-health`. **Do not** disable the drift gate to unblock a release — the gate failing closed is correct behavior. |
| A single skill's drift check fails | The wrapped CLI renamed or removed a command | Drift detail: the unresolved command and the pinned manifest revision | Decide which side is wrong. If the CLI moved deliberately, update the skill. If the skill was right, the CLI change is the regression. |
| Rehearsal fails with an undisclosed acquisition | The install pulls something the declaration does not name | Rehearsal journey: the named acquisition | Either disclose it in the declaration or remove it from the install. Do not suppress the check — this is exactly the stealth-bundling case it exists to catch. |
| Rehearsals hang, then time out | Sandbox provisioning is wedged | `workspace-sandbox` status; check for leaked sandbox instances | Tear down leaked sandboxes, restart `workspace-sandbox`. Investigate teardown-on-cancel if instances accumulate. |
| Attestation fails at signing | Managed release authority unavailable, or conformance record missing | `vrooli release-authority` status; conformance record for the package | Restore the authority. If conformance is missing, the ordering guard is working — re-run conformance rather than bypassing it. |
| Publication refused with no release decision | No `deployment-manager` decision for this exact source commit | `scenario-to-plugin publish status`; deployment-manager approvals | Obtain approval for the exact commit. An approval for an earlier commit deliberately does not carry forward. |
| Publication recorded as unconfirmed | Push succeeded but retrieval at the digest failed | Channel outcome detail | Retry — push is idempotent by digest. Persistent failure means the registry accepted and dropped the artifact; escalate to that registry. |
| Revocation reports `revoked_partial` | A channel cannot hard-delete a version | Revocation detail: the channels still carrying it | Escalate to each named registry's withdrawal process. This is a real terminal state, not a retryable one. |
| A published package is later found to drift | The wrapped CLI changed after publication | `manifest_pins` for the published version | Revoke, fix the skill, republish. Note this in `PROBLEMS.md` — automatic re-verification of published versions is a known gap. |

### Incident: a bad package was published

This is the highest-severity incident this scenario can produce, because
the artifact is on machines Vrooli does not control.

1. **Revoke first, diagnose second.** `scenario-to-plugin revoke <version>`.
   Getting the artifact withdrawn matters more than knowing why yet.
2. Confirm the fan-out covered every channel. If the result is
   `revoked_partial`, escalate to each named registry immediately.
3. Preserve evidence. Do not delete the package, its conformance run, its
   attestation, or its publication rows — they explain an artifact that
   still exists in the world.
4. Determine which gate should have caught it. Every incident of this kind
   is a missing or mis-scoped gate, not bad luck.
5. Add the fixture to the gate self-test in the release checklist so the
   same class cannot recur.
6. Follow the disclosure process in the skills security policy.

## Backup / Restore

| Asset | Backed up | Why |
|---|---|---|
| SQLite database | Yes | Contains publication history, from which the revocation fan-out is derived. Losing it loses the kill switch. |
| Capture store | Yes, for anything referenced by a publication | Explains published artifacts after the fact. Unreferenced build artifacts are regenerable. |
| Credentials | No | Held by `secrets-manager`; this scenario stores references only. |
| Signing keys | No | Held by the managed release authority. This scenario never has one. |

Restore is ordinary scenario restore. After restoring, verify that
`publications` is intact before publishing anything new — a truncated
publication table will silently under-report a revocation fan-out.

## Maintenance Tasks

| Task | Cadence | Notes |
|---|---|---|
| Prune unreferenced packages, artifacts, and rehearsal logs | Monthly | Retention rules are in `../concepts/DATA.md`. The job is not yet implemented; see `PROBLEMS.md`. |
| Re-verify published versions against current CLI manifests | Monthly | Not yet implemented. Until it is, drift in a published version is discovered by users rather than by us. |
| Review scanner versions | On scanner release | A scanner version change alters what a passing verdict means; record it with the attestation. |
| Review the Agent Plugins and Agent Skills specifications | Quarterly | Both are young. A specification revision changes what conformance means. |
| Rotate registry credential references | Per credential policy | Owned by `secrets-manager`; this scenario needs no change. |

## Escalation

| Situation | Escalate to |
|---|---|
| A published artifact is malicious or vulnerable | Security disclosure process, immediately, in parallel with revocation |
| Partial revocation that a registry will not complete | The registry's published withdrawal process; record the ticket in `PROBLEMS.md` |
| Signing authority compromise suspected | Platform release-authority owner. Stop all publication. |
| A gate is being bypassed to meet a deadline | Stop. There is no supported bypass, and adding one is the failure mode this scenario exists to prevent. |
| Repeated drift failures across many scenarios | Likely a `cli-manifest` contract change rather than many skill regressions |

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — release checklist and rollback
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — signals to check first
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency failure modes
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known gaps
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — template-level troubleshooting
