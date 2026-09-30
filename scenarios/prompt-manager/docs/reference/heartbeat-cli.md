# Heartbeat CLI Reference

CLI commands for managing heartbeat configurations, execution, and member documents.

## Overview

Heartbeat commands are subcommands of `prompt-manager team`. They manage:
- Heartbeat configuration (schedule, enabled state)
- Manual triggering and execution logs
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

### Finite effort leader provisioning

The operator order is: create or resume a finite delivery team; register and
verify `team:<team-id>` in Source Ledger; validate the team's operating
contract; qualify the selected profile and owner route with a disposable team;
provision the binding disabled; review the evidence and obtain separate
execution approval; then enable only the approved finite team and admit its
first run through team execution. Team metadata and a heartbeat binding never
grant an effort approval.

Create an unused heartbeat binding in disabled state through the canonical CLI:

```bash
prompt-manager team heartbeat-bind-effort <team-id> <leader-id> --request-file binding.json
prompt-manager team heartbeat <team-id> <leader-id> --json
```

`binding.json` contains the exact configuration, without an RPC envelope. The
`profileKey` must be an actually qualified Agent Manager route (for the current
Prompt Manager judgment profile this is `prompt-manager/heartbeat-judgment`),
not a display name or an unverified historical key:

```json
{
  "schedule": "*/5 * * * *",
  "profileKey": "<qualified-profile>",
  "finiteLeader": {
    "effortRef": "<exact-effort-ref>",
    "acceptedRevision": "<exact-accepted-revision>",
    "coordinatorPromptRef": "<coordinator-prompt-ref>",
    "sourceRefs": ["<accepted-source-ref>"]
  }
}
```

Add `--update` only to bind an existing unused disabled heartbeat. Provisioning
always sends `enabled:false`. It validates a single JSON object of at most 16 KiB
and refuses unknown fields, including an activation flag. The command verifies
the returned exact binding and disabled state; an older server ignoring the new
fields cannot be reported as successful provisioning. Neither command starts an
agent or supplies an effort grant.

Retire future finite dispatch while retaining the current run and reservation:

```bash
prompt-manager team heartbeat-retire-effort <team-id> <leader-id>
```

Retirement reads the current owner binding, preserves its references, disables
scheduling and sets irreversible retirement. It does not cancel the run or claim
effort completion. Ordinary `heartbeat-disable` remains a reversible scheduling
pause. The heartbeat read retains `finiteLeaderState` and any owner-read error.
Live adoption still requires qualified finite recurrence and purpose-bound
coordinator authority; the initial single-run binding is not that qualification.

Record the revision-checked completion receipt, or explicitly reopen a completed
effort. For a dispatched owner run that ended without a completion receipt, use
the explicit terminal-run restart operation:

```bash
prompt-manager team heartbeat-complete-effort <team-id> <leader-id> --request-file transition.json
prompt-manager team heartbeat-reopen-effort <team-id> <leader-id> --request-file transition.json
prompt-manager team heartbeat-restart-effort <team-id> <leader-id> --request-file transition.json
```

For a `keepAlive` orchestrator, restart is also the way out of the liveness
relaunch cap (`relaunch-capped`); it resets the consecutive-relaunch count.

`transition.json` is a single JSON object of at most 16 KiB with no unknown
fields:

```json
{ "revision": "<exact-accepted-or-replacement-revision>", "evidenceRef": "<retained-owner-evidence>" }
```

Completion records an idempotent receipt only when `revision` equals the
accepted binding revision, and permanently refuses later scheduled or manual
starts until an explicit reopen. Reopen requires a replacement revision
different from the completed one and retains the prior receipt in
`finiteLeaderState.completionHistory`. Both commands read the exact binding
first and send only `finiteEffortTransition`; they never combine the lifecycle
operation with scheduling or configuration changes. Restart requires the exact
owner run to be reread from Agent Manager and confirmed terminal; it preserves
the prior task/run/evidence in `finiteLeaderState.restartHistory` and clears
only the reusable reservation. It does not cancel an active run, replace an
uncertain dispatch, or claim effort acceptance on its own.

When an initial run admission loses its response, the owner retains the task
reservation. A later reconciliation may replay the same idempotency key only
after Agent Manager returns an authoritative zero-run result for that exact
task; truncated, conflicting, unavailable, or nonzero results remain fenced.

Focused CLI verification (2026-09-12): `go test -race ./teams -run
'^TestFiniteLeaderCLI' -count=1 -timeout=60s`
passes. The finite CLI tests cover disabled create/update, malformed and
activation-bearing input rejection, mismatched owner responses, retirement with
retained identity, explicit completion/reopen transition payloads that carry no
configuration change, refusal of ambiguous transition input or an unbound member,
and the real human/JSON read commands. No live configuration was provisioned by
these tests.

The CLI tests prove request validation and lifecycle payload shape only. Live
qualification must additionally show the team, Source Ledger scope, PM
heartbeat state, Agent Manager task/run, exact work reference and handoff or
receipt. A terminal owner run is not accepted effort; use the revision-checked
completion transition only after the owner evidence is retained. Leave
recurrence and fresh-run recovery disabled until their separate gates pass.

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
