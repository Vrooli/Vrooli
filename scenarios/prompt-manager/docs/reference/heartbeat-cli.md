# Heartbeat CLI Reference

CLI commands for managing heartbeat configurations, execution, and member documents.

## Overview

Heartbeat commands are subcommands of `prompt-manager team`. They manage:
- Heartbeat configuration (schedule, enabled state)
- Manual triggering and execution logs
- Bounded per-member dispatch-attempt history
- Member documents (RESPONSIBILITIES.md, HEARTBEAT.md)

---

## Heartbeat Auto-Pause Control

### prompt-manager heartbeat-control status

Show global heartbeat control state and per-team summaries.

```bash
prompt-manager heartbeat-control status [--json]
```

### prompt-manager heartbeat-control pause

Manually pause future heartbeat starts globally.

```bash
prompt-manager heartbeat-control pause [--reason "quiet period"] [--json]
```

### prompt-manager heartbeat-control resume

Resume global heartbeat scheduling and reschedule enabled heartbeat configs for enabled teams.

```bash
prompt-manager heartbeat-control resume [--json]
```

### prompt-manager heartbeat-control policy

Show or update the global auto-pause policy.

```bash
prompt-manager heartbeat-control policy show [--json]
prompt-manager heartbeat-control policy set --enabled=true --pause-after=14d --warning-after=10d --resume-mode=manual [--json]
```

### prompt-manager team heartbeat-control

Show or update one team's control state.

```bash
prompt-manager team heartbeat-control <team-id> status [--json]
prompt-manager team heartbeat-control <team-id> pause [--reason "quiet period"] [--json]
prompt-manager team heartbeat-control <team-id> resume [--json]
prompt-manager team heartbeat-control <team-id> policy show [--json]
prompt-manager team heartbeat-control <team-id> policy set --mode=inherit|disabled|custom --pause-after=21d --warning-after=14d [--json]
```

Pause is separate from member heartbeat `enabled`. A paused team can still have enabled heartbeat configs; they simply will not start until resumed.

---

## Heartbeat Configuration

### Native delivery team commissioning

Prompt Manager does not have finite-leader bind, amend, retire, completion, reopen, or restart commands. Configure a delivery team through the existing team contract: `EffortRefs` identifies the effort, `Coordination.LeadAgentID` identifies its leader, the operating contract carries member and plan-of-record/source paths, and the leader member heartbeat selects its Agent Manager `ProfileKey`. Keep the team and heartbeat disabled during preparation.

Before enablement, follow the reusable [product receiver-qualification procedure](../../store/skills/packs/core/large-effort-orchestration/references/commissioning.md) when the team declares a product home. Qualify its actual member/profile/owner route and selected sources; retain the receipt in that home. Managed finite plans, families, qualifications and investigations instead retain their declared owner and existing handoff. Verify the same actual identity/source boundary there, without creating a product home or copying another effort's source list.

The ordinary member heartbeat route (`team heartbeat-enable`, `team heartbeat-trigger`) creates and dispatches the task/run. For the configured finite delivery lead, Prompt Manager requires a valid team contract, declared leader, active member, matching profile, and exact current Agent Manager enrollment before creating a task or run. It adds the enrollment's non-withdrawn `TargetRevision` as a public, verified `orchestrator` WorkReference to the immutable CreateRun request; it does not derive verification from environment labels or accept a locally supplied revision. A missing, unavailable, or mismatched identity fails admission. If a prior run ID exists, PM preserves that exact owner obligation as uncertain until it can be verified and does not launch competing work. Active/parked verified owners are watched through the existing owner path; a verified terminal owner permits a fresh dispatch using current configuration. Finite non-lead members and ordinary standing/non-delivery heartbeats remain supported without orchestrator association. WorkReferences describe work; they do not grant authority.

Use the existing Agent Manager effort enrollment and run identity as canonical state. A terminal run does not accept the effort, and the normal parked orchestrator/child-watch route remains the qualified continuation model. Event check-ins are proposed and unavailable in Prompt Manager.

A declared product home requires a running Agent Manager delivery commission before leader
dispatch. An enrollment that already has a commission remains subject to its running state,
start timestamp and budget even without a home hint. Managed finite work with neither keeps
its own owner admission and stopping model; it still needs the exact enrollment/revision,
active member, matching profile, qualified caller and current authority. Preparing member
instructions or registering `team:<id>` in Source Ledger does not enable scheduling or runs.

### Effort supervisor heartbeat

The bundled team `effort-supervision` is disabled. Its `effort-supervisor`
member is an ordinary heartbeat (daily schedule, `--wake-admission=on-change
--wake-sources=team,member,inbox`) on `prompt-manager/delivery-review`;
orchestrators wake it with `team message-send` plus `team heartbeat-trigger`.

### prompt-manager team heartbeat-list

List all heartbeat configurations for a team.

```bash
prompt-manager team heartbeat-list <team-id> [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |

**Example:**
```bash
prompt-manager team heartbeat-list my-team
# Output:
# Heartbeats for my-team:
#   agent-1: enabled (0 */6 * * *) - last: completed 2h ago
#   agent-2: disabled (0 0 * * *) - never run
```

---

### prompt-manager team heartbeat-fleet-health

Report the rolling 24-hour success aggregate across enabled heartbeat members of enabled teams.

```bash
prompt-manager team heartbeat-fleet-health [--json]
```

The numerator uses each heartbeat record's durable `lastSuccessfulExecution`. Starting a new run therefore does not erase that member's earlier completion from the rolling window, and the aggregate does not depend on event-history retention. The JSON response includes `successPercent`, `thresholdPercent`, and the integer-arithmetic `meetsThreshold` verdict; consumers should use that verdict instead of rounding the percentage. `membersWithTwoFailures` reports the number of enabled members whose consecutive-failure streak is at least two.

---

### prompt-manager team heartbeat

Get heartbeat configuration for a specific member.

```bash
prompt-manager team heartbeat <team-id> <agent-id> [--json]
```

**Example:**
```bash
prompt-manager team heartbeat my-team agent-1
# Output:
# Heartbeat for my-team/agent-1:
#   Enabled:  true
#   Schedule: 0 */6 * * * (every 6 hours)
#   Profile:  prompt-manager/heartbeat
#   Last Run: 2026-02-01T10:00:00Z (completed)
#   Next Run: 2026-02-01T16:00:00Z
```

---

### prompt-manager team heartbeat-enable

Enable or create a heartbeat configuration for a member.

```bash
prompt-manager team heartbeat-enable <team-id> <agent-id> [--schedule=<cron>] [--profile=<key>] [--wake-admission=always|on-change] [--wake-sources=team,inbox] [--json]
```

**Options:**
| Flag | Required | Description |
|------|----------|-------------|
| `--schedule` | No | Cron expression for execution schedule; new heartbeats default to `0 */6 * * *`, while updates preserve the existing schedule unless this flag is supplied |
| `--profile` | No | Declared Agent Manager profile key override. Defaults to `prompt-manager/heartbeat-judgment` (declared in `.vrooli/agent-manager/heartbeat.json`; role `code.economy.judgment`) for multi-process teams and `prompt-manager/heartbeat-inspection` (declared in `.vrooli/agent-manager/heartbeat-single-process.json`; role `code.flatrate`) for single-process teams. The runner, model, and effort come from the resource-owned role policy, not this document or the profile name; resolve them from the live catalog before changing a route. |
| `--wake-admission` | No | `always` (default) runs on every schedule; `on-change` admits only when a selected bounded source identity changes. |
| `--wake-sources` | No | Comma-separated `team`, `member`, `inbox`, or `corpus` sources required with `--wake-admission=on-change`. |
| `--json` | No | Output as JSON |

**Schedule Examples:**
| Expression | Description |
|------------|-------------|
| `0 * * * *` | Every hour |
| `0 */6 * * *` | Every 6 hours |
| `0 0 * * *` | Daily at midnight |
| `0 9 * * *` | Daily at 9am |
| `0 0 * * 1` | Weekly on Monday |

**Example:**
```bash
prompt-manager team heartbeat-enable my-team agent-1 --schedule="0 */6 * * *"
# Output: Heartbeat enabled for my-team/agent-1 (0 */6 * * *)
```

---

### prompt-manager team heartbeat-disable

Disable a heartbeat configuration.

```bash
prompt-manager team heartbeat-disable <team-id> <agent-id> [--json]
```

**Example:**
```bash
prompt-manager team heartbeat-disable my-team agent-1
# Output: Heartbeat disabled for my-team/agent-1
```

---

### prompt-manager team heartbeat-trigger

Manually trigger a heartbeat execution.

For a caller carrying an operation grant or request-operation authority, the exact
member-trigger route checks the existing `run.create` operation on `agent-manager:*`
with `agent-manager:write`. Team configuration uses its separate `team.manage`
contract. The trigger still enters the existing caller, membership, pause and
queue-deduplication checks; AUTH-01 retains its declared caller-proof mode.

```bash
prompt-manager team heartbeat-trigger <team-id> <agent-id> [--json]
```

**Example:**
```bash
prompt-manager team heartbeat-trigger my-team agent-1
# Output:
# Heartbeat triggered for my-team/agent-1
# Run ID: run-xyz789
# Status: running
# Log: 2026-02-01T15-30-00Z.log
```

---

### prompt-manager team heartbeat-logs

List execution logs for a member.

```bash
prompt-manager team heartbeat-logs <team-id> <agent-id> [--json]
```

**Example:**
```bash
prompt-manager team heartbeat-logs my-team agent-1
# Output:
# Execution logs for my-team/agent-1:
#   2026-02-01T10-00-00Z.log (completed)
#   2026-02-01T04-00-00Z.log (completed)
#   2026-01-31T22-00-00Z.log (failed)
```

### prompt-manager team heartbeat-attempts

Read a bounded newest-first page of dispatch attempts for exactly one team
member. The default limit is 20; values from 1 through 200 are accepted. If
the newest 1,000 event records do not exhaust the source, `total` is a lower
bound, `totalExact` is false, and `hasMore` remains true. `--offset` pages the
currently observed recent window; it does not continue through all retained
history or guarantee a stable snapshot. JSON output retains the complete API
response. An API or identity error is returned to the caller.

```bash
prompt-manager team heartbeat-attempts <team-id> <agent-id> [--limit 1..200] [--offset N] [--json]
```

Use `--json` for a machine-readable record set. This command does not trigger a
heartbeat or change scheduling state.

---

## Member Documents

### prompt-manager team responsibilities

Get or set RESPONSIBILITIES.md for a team member.

```bash
# Get
prompt-manager team responsibilities <team-id> <agent-id> [--json]

# Set from string
prompt-manager team responsibilities <team-id> <agent-id> --set='content'

# Set from file
prompt-manager team responsibilities <team-id> <agent-id> --file=path
```

**Options:**
| Flag | Description |
|------|-------------|
| `--set` | Set content from a string value |
| `--file` | Set content from a file path |
| `--json` | Output as JSON |

**Examples:**
```bash
# Get responsibilities
prompt-manager team responsibilities my-team agent-1
# Output: (content of RESPONSIBILITIES.md)

# Set from inline content
prompt-manager team responsibilities my-team agent-1 --set='# Responsibilities

- Monitor system health
- Report anomalies'

# Set from file
prompt-manager team responsibilities my-team agent-1 --file=responsibilities.md
# Output: Updated RESPONSIBILITIES.md for my-team/agent-1 (142 bytes)
```

---

### prompt-manager team heartbeat-instructions

Get or set HEARTBEAT.md for a team member.

```bash
# Get
prompt-manager team heartbeat-instructions <team-id> <agent-id> [--json]

# Set from string
prompt-manager team heartbeat-instructions <team-id> <agent-id> --set='content'

# Set from file
prompt-manager team heartbeat-instructions <team-id> <agent-id> --file=path
```

**Options:**
| Flag | Description |
|------|-------------|
| `--set` | Set content from a string value |
| `--file` | Set content from a file path |
| `--json` | Output as JSON |

**Examples:**
```bash
# Get heartbeat instructions
prompt-manager team heartbeat-instructions my-team agent-1
# Output: (content of HEARTBEAT.md)

# Set from inline content
prompt-manager team heartbeat-instructions my-team agent-1 --set='# Heartbeat Task

On each heartbeat:
1. Check pending issues
2. Review recent commits
3. Update status report'

# Set from file
prompt-manager team heartbeat-instructions my-team agent-1 --file=heartbeat-task.md
```

---

## Agent Soul

### prompt-manager agent soul

Get or set SOUL.md for an agent.

```bash
# Get
prompt-manager agent soul <agent-id> [--json]

# Set from string
prompt-manager agent soul <agent-id> --set='content'

# Set from file
prompt-manager agent soul <agent-id> --file=path
```

**Options:**
| Flag | Description |
|------|-------------|
| `--set` | Set content from a string value |
| `--file` | Set content from a file path |
| `--json` | Output as JSON |

**Examples:**
```bash
# Get soul
prompt-manager agent soul agent-1
# Output: (content of SOUL.md)

# Set from inline content
prompt-manager agent soul agent-1 --set='# Agent Personality

I am a meticulous and thorough assistant who values:
- Clarity in communication
- Systematic problem-solving
- Continuous improvement'

# Set from file
prompt-manager agent soul agent-1 --file=soul.md
```

---

## Prompt Preview and Member Context

### prompt-manager team prompt-preview

Preview the full runtime heartbeat prompt for a member. This includes the active `HEARTBEAT.md` task and should be used when auditing exactly what a heartbeat run receives.

```bash
prompt-manager team prompt-preview <team-id> <agent-id> [--json]
```

### prompt-manager team prompt-preview-structured

Preview the same runtime prompt as backend-ordered sections. This is the CLI equivalent of the UI's prompt pipeline surface.

```bash
prompt-manager team prompt-preview-structured <team-id> <agent-id> [--json]
```

### prompt-manager team prompt-matrix

Show prompt section coverage and character counts for every member in a team. Use `--json` to inspect the complete structured prompt matrix.

```bash
prompt-manager team prompt-matrix <team-id> [--json]
```

### prompt-manager team member-context

Get standing context for a team member without the active `HEARTBEAT.md` task. This includes agent files, responsibilities, org context, coordination guidance, storage-map guidance, and inbox content when enabled. Use this for external or leader-led bootstrapping that needs taskless context; use `prompt-preview` to audit the full runtime heartbeat prompt.

```bash
prompt-manager team member-context <team-id> <agent-id> [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--json` | Output as JSON (includes teamId and agentId fields) |

**Examples:**
```bash
# Get context as plain text (useful for piping)
prompt-manager team member-context my-team agent-1

# Get context as JSON
prompt-manager team member-context my-team agent-1 --json
# Output:
# {
#   "teamId": "my-team",
#   "agentId": "agent-1",
#   "prompt": "# Agent Files (Markdown)\n\n..."
# }
```

---

## Common Workflows

### Setting Up a New Heartbeat

```bash
# 1. Create agent if not exists
prompt-manager agent create "Monitor Bot"

# 2. Add agent to team
prompt-manager team add-member ops-team monitor-bot

# 3. Set agent soul
prompt-manager agent soul monitor-bot --file=soul.md
# Or inline: prompt-manager agent soul monitor-bot --set='# Monitor Bot ...'

# 4. Set responsibilities for this team
prompt-manager team responsibilities ops-team monitor-bot --file=responsibilities.md
# Or inline: prompt-manager team responsibilities ops-team monitor-bot --set='...'

# 5. Set heartbeat instructions
prompt-manager team heartbeat-instructions ops-team monitor-bot --file=heartbeat-task.md
# Or inline: prompt-manager team heartbeat-instructions ops-team monitor-bot --set='...'

# 6. Enable heartbeat
prompt-manager team heartbeat-enable ops-team monitor-bot --schedule="0 */6 * * *"
```

### Testing a Heartbeat

```bash
# Manually trigger
prompt-manager team heartbeat-trigger ops-team monitor-bot

# View logs
prompt-manager team heartbeat-logs ops-team monitor-bot

# Check specific log (use filename from logs output)
# Logs are stored in: store/teams/{team}/members/{agent}/logs/
```

### Disabling a Heartbeat

```bash
# Disable (keeps config)
prompt-manager team heartbeat-disable ops-team monitor-bot

# Or delete entirely
prompt-manager team remove-member ops-team monitor-bot --force
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | API connection failed |
| 3 | Resource not found |
| 4 | Invalid schedule expression |

---

## Implementation Reference

- [CODE: cli/teams/teams.go] - Team and heartbeat CLI commands
- [CODE: cli/agents/agents.go] - Agent and soul CLI commands
- [CODE: api/heartbeat/handlers.go] - HTTP handlers
- [CODE: api/heartbeat/scheduler.go] - Cron scheduler
