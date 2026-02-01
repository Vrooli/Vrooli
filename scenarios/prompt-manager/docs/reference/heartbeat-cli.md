# Heartbeat CLI Reference

CLI commands for managing heartbeat configurations, execution, and member documents.

## Overview

Heartbeat commands are subcommands of `prompt-manager team`. They manage:
- Heartbeat configuration (schedule, enabled state)
- Manual triggering and execution logs
- Member documents (RESPONSIBILITIES.md, HEARTBEAT.md)

---

## Heartbeat Configuration

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
#   Profile:  prompt-manager-heartbeat
#   Last Run: 2026-02-01T10:00:00Z (completed)
#   Next Run: 2026-02-01T16:00:00Z
```

---

### prompt-manager team heartbeat-enable

Enable or create a heartbeat configuration for a member.

```bash
prompt-manager team heartbeat-enable <team-id> <agent-id> --schedule=<cron> [--profile=<key>] [--json]
```

**Options:**
| Flag | Required | Description |
|------|----------|-------------|
| `--schedule` | Yes | Cron expression for execution schedule |
| `--profile` | No | Agent-manager profile key (default: `prompt-manager-heartbeat`) |
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
prompt-manager team responsibilities <team-id> <agent-id>

# Set (reads from stdin)
prompt-manager team responsibilities <team-id> <agent-id> --set
```

**Options:**
| Flag | Description |
|------|-------------|
| `--set` | Set content from stdin |

**Examples:**
```bash
# Get responsibilities
prompt-manager team responsibilities my-team agent-1
# Output: (content of RESPONSIBILITIES.md)

# Set responsibilities from stdin
cat responsibilities.md | prompt-manager team responsibilities my-team agent-1 --set
# Output: RESPONSIBILITIES.md updated for my-team/agent-1

# Set using heredoc
prompt-manager team responsibilities my-team agent-1 --set << 'EOF'
# Agent Responsibilities

## Primary Duties
- Monitor system health
- Report anomalies

## Secondary Duties
- Assist with debugging
EOF
```

---

### prompt-manager team heartbeat-instructions

Get or set HEARTBEAT.md for a team member.

```bash
# Get
prompt-manager team heartbeat-instructions <team-id> <agent-id>

# Set (reads from stdin)
prompt-manager team heartbeat-instructions <team-id> <agent-id> --set
```

**Options:**
| Flag | Description |
|------|-------------|
| `--set` | Set content from stdin |

**Examples:**
```bash
# Get heartbeat instructions
prompt-manager team heartbeat-instructions my-team agent-1
# Output: (content of HEARTBEAT.md)

# Set heartbeat instructions
prompt-manager team heartbeat-instructions my-team agent-1 --set << 'EOF'
# Heartbeat Task

On each heartbeat:
1. Check pending issues
2. Review recent commits
3. Update status report
EOF
```

---

## Agent Soul

### prompt-manager agent soul

Get or set SOUL.md for an agent.

```bash
# Get
prompt-manager agent soul <agent-id>

# Set (reads from stdin)
prompt-manager agent soul <agent-id> --set
```

**Options:**
| Flag | Description |
|------|-------------|
| `--set` | Set content from stdin |

**Examples:**
```bash
# Get soul
prompt-manager agent soul agent-1
# Output: (content of SOUL.md)

# Set soul
prompt-manager agent soul agent-1 --set << 'EOF'
# Agent Personality

I am a meticulous and thorough assistant who values:
- Clarity in communication
- Systematic problem-solving
- Continuous improvement
EOF
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
prompt-manager agent soul monitor-bot --set << 'EOF'
# Monitor Bot

I am vigilant and detail-oriented. I catch issues before they become problems.
EOF

# 4. Set responsibilities for this team
prompt-manager team responsibilities ops-team monitor-bot --set << 'EOF'
# Responsibilities

- Monitor system health
- Alert on anomalies
- Generate daily reports
EOF

# 5. Set heartbeat instructions
prompt-manager team heartbeat-instructions ops-team monitor-bot --set << 'EOF'
# Heartbeat Task

1. Check all system metrics
2. Compare against baselines
3. Report any deviations
EOF

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
