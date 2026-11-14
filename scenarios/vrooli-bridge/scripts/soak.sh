#!/usr/bin/env bash
# Run the Bridge multi-node reliability soak. The script is intentionally
# explicit about its destructive hooks: a real run must name how the operator
# will restart agents, restart Bridge through the lifecycle controller, and
# introduce/recover a network partition. No hook is silently skipped.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/soak.sh [options]

Required environment for a live run:
  BRIDGE_SOAK_LINUX_NODE_ID       Paired node id for swarminator
  BRIDGE_SOAK_MAC_NODE_ID         Paired node id for minimouse
  BRIDGE_SOAK_LINUX_HOST          SSH host for the Linux agent
  BRIDGE_SOAK_MAC_HOST            SSH host for the macOS agent
  BRIDGE_SOAK_NETWORK_PARTITION   Command that enters a bounded partition
  BRIDGE_SOAK_NETWORK_RESTORE     Command that restores that partition

Optional environment:
  BRIDGE_SOAK_SCENARIO             Typed scenario to dispatch (default: vrooli-bridge)
  BRIDGE_SOAK_VERB                 Typed verb to dispatch (default: status)
  BRIDGE_SOAK_ARGS                 Comma-separated typed args
  BRIDGE_SOAK_AGENT_KILL_LINUX     Remote command that kills the Linux agent
  BRIDGE_SOAK_AGENT_KILL_MAC       Remote command that kills the macOS agent
  BRIDGE_SOAK_CONTROL_RESTART      Lifecycle command (default: vrooli scenario restart vrooli-bridge)

Options:
  --duration DURATION     Run window (default: 24h)
  --interval SECONDS      Probe/dispatch interval (default: 30)
  --grace SECONDS         Deadline grace used by the invariant (default: 30)
  --fault-interval N      Inject one selected fault every N probes (default: 20)
  --db PATH               Bridge SQLite database path
  --report PATH           Markdown report path
  --dry-run               Validate prerequisites and print the planned run
  -h, --help              Show this help
EOF
}

DURATION=24h
INTERVAL=30
GRACE=30
FAULT_INTERVAL=20
DB_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/data/vrooli-bridge.db"
REPORT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/docs/internal/SOAK-REPORT.md"
DRY_RUN=false

while (($#)); do
  case "$1" in
    --duration) DURATION="$2"; shift 2 ;;
    --interval) INTERVAL="$2"; shift 2 ;;
    --grace) GRACE="$2"; shift 2 ;;
    --fault-interval) FAULT_INTERVAL="$2"; shift 2 ;;
    --db) DB_PATH="$2"; shift 2 ;;
    --report) REPORT_PATH="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

for value_name in INTERVAL GRACE FAULT_INTERVAL; do
  value="${!value_name}"
  [[ "$value" =~ ^[0-9]+$ ]] || { echo "$value_name must be a non-negative integer" >&2; exit 2; }
done
(( INTERVAL > 0 )) || { echo "INTERVAL must be positive" >&2; exit 2; }
(( FAULT_INTERVAL > 0 )) || { echo "FAULT_INTERVAL must be positive" >&2; exit 2; }

duration_seconds() {
  case "$1" in
    *h) printf '%s' "$(( ${1%h} * 3600 ))" ;;
    *m) printf '%s' "$(( ${1%m} * 60 ))" ;;
    *s) printf '%s' "${1%s}" ;;
    *) echo "duration must end in h, m, or s: $1" >&2; return 2 ;;
  esac
}

RUN_SECONDS="$(duration_seconds "$DURATION")"
[[ "$RUN_SECONDS" =~ ^[0-9]+$ ]] && (( RUN_SECONDS > 0 )) || exit 2

LINUX_NODE_ID="${BRIDGE_SOAK_LINUX_NODE_ID:-}"
MAC_NODE_ID="${BRIDGE_SOAK_MAC_NODE_ID:-}"
LINUX_HOST="${BRIDGE_SOAK_LINUX_HOST:-swarminator}"
MAC_HOST="${BRIDGE_SOAK_MAC_HOST:-minimouse}"
SCENARIO="${BRIDGE_SOAK_SCENARIO:-vrooli-bridge}"
VERB="${BRIDGE_SOAK_VERB:-status}"
JOB_ARGS="${BRIDGE_SOAK_ARGS:-}"
NETWORK_PARTITION="${BRIDGE_SOAK_NETWORK_PARTITION:-}"
NETWORK_RESTORE="${BRIDGE_SOAK_NETWORK_RESTORE:-}"
LINUX_KILL="${BRIDGE_SOAK_AGENT_KILL_LINUX:-}"
MAC_KILL="${BRIDGE_SOAK_AGENT_KILL_MAC:-}"
CONTROL_RESTART="${BRIDGE_SOAK_CONTROL_RESTART:-vrooli scenario restart vrooli-bridge}"

require_command() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing command: $1" >&2; exit 1; }
}

require_command sqlite3
require_command vrooli
require_command vrooli-bridge
[[ -f "$DB_PATH" ]] || { echo "Bridge database not found: $DB_PATH" >&2; exit 1; }

if [[ "$DRY_RUN" != true ]]; then
  for name in LINUX_NODE_ID MAC_NODE_ID NETWORK_PARTITION NETWORK_RESTORE; do
    [[ -n "${!name}" ]] || { echo "$name is required for a live soak" >&2; exit 1; }
  done
  [[ -n "$LINUX_KILL" ]] || { echo "BRIDGE_SOAK_AGENT_KILL_LINUX must be explicit for a live soak" >&2; exit 1; }
  [[ -n "$MAC_KILL" ]] || { echo "BRIDGE_SOAK_AGENT_KILL_MAC must be explicit for a live soak" >&2; exit 1; }
fi

if [[ "$DRY_RUN" == true ]]; then
  printf 'soak plan: duration=%s interval=%ss grace=%ss fault_interval=%s\n' "$DURATION" "$INTERVAL" "$GRACE" "$FAULT_INTERVAL"
  printf 'targets: linux=%s (%s) macos=%s (%s)\n' "$LINUX_NODE_ID" "$LINUX_HOST" "$MAC_NODE_ID" "$MAC_HOST"
  printf 'job: scenario=%s verb=%s args=%s\n' "$SCENARIO" "$VERB" "${JOB_ARGS:-<none>}"
  printf 'faults: random agent-kill/linux+macos, control-plane-restart, network-partition\n'
  printf 'report: %s\n' "$REPORT_PATH"
  exit 0
fi

mkdir -p "$(dirname "$REPORT_PATH")"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
started_epoch="$(date +%s)"
deadline_epoch=$(( started_epoch + RUN_SECONDS ))
probe_count=0
fault_count=0
late_count=0
declare -a fault_log=()

terminal_distribution() {
  sqlite3 -separator $'\t' "$DB_PATH" \
    "SELECT CASE status WHEN 1 THEN 'queued' WHEN 2 THEN 'running' WHEN 3 THEN 'passed' WHEN 4 THEN 'failed' WHEN 5 THEN 'aborted' WHEN 6 THEN 'pushed' WHEN 7 THEN 'acked' WHEN 8 THEN 'failed_delivery' ELSE 'unknown' END, COUNT(*) FROM runs GROUP BY status ORDER BY status;"
}

late_runs() {
  sqlite3 -separator $'\t' "$DB_PATH" \
    "SELECT id, status, created_at, timeout_seconds FROM runs WHERE status NOT IN (3,4,5,8) AND timeout_seconds > 0 AND (julianday('now') - julianday(created_at)) * 86400.0 > timeout_seconds + $GRACE ORDER BY created_at;"
}

write_report() {
  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  {
    printf '# Bridge multi-node soak report\n\n'
    printf -- '- Window: `%s` → `%s`\n' "$started_at" "$ended_at"
    printf -- '- Requested duration: `%s`\n' "$DURATION"
    printf -- '- Targets: Linux `%s` and macOS `%s`\n' "$LINUX_HOST" "$MAC_HOST"
    printf -- '- Probes: `%s`\n' "$probe_count"
    printf -- '- Injected faults: `%s`\n' "$fault_count"
    printf -- '- Non-terminal runs past deadline + grace: `%s`\n\n' "$late_count"
    printf '## Fault log\n\n'
    if ((${#fault_log[@]} == 0)); then printf -- '- none\n'; else printf -- '%s\n' "${fault_log[@]}"; fi
    printf '\n## Terminal-state distribution\n\n```text\n'
    terminal_distribution
    printf '```\n'
  } >"$REPORT_PATH"
}

on_exit() {
  write_report
  if (( late_count > 0 )); then
    echo "soak invariant failed: $late_count non-terminal run(s) exceeded deadline plus grace" >&2
    return 1
  fi
}
trap on_exit EXIT

while (( $(date +%s) < deadline_epoch )); do
  probe_count=$((probe_count + 1))
  if (( probe_count % 2 == 0 )); then
    vrooli-bridge dispatch job "$LINUX_NODE_ID" --scenario "$SCENARIO" --verb "$VERB" --args "$JOB_ARGS" --timeout 60 >/dev/null || true
    vrooli-bridge dispatch job "$MAC_NODE_ID" --scenario "$SCENARIO" --verb "$VERB" --args "$JOB_ARGS" --timeout 60 >/dev/null || true
  fi

  if late_output="$(late_runs)"; then
    if [[ -n "$late_output" ]]; then
      late_count=$((late_count + $(printf '%s\n' "$late_output" | wc -l)))
    fi
  fi

  if (( probe_count % FAULT_INTERVAL == 0 )); then
    fault_count=$((fault_count + 1))
    fault_kind=$((RANDOM % 4))
    if (( fault_kind == 0 )); then
      ssh "$LINUX_HOST" "$LINUX_KILL"
      fault_log+=("$(date -u +%Y-%m-%dT%H:%M:%SZ) agent kill: linux $LINUX_HOST")
    elif (( fault_kind == 1 )); then
      ssh "$MAC_HOST" "$MAC_KILL"
      fault_log+=("$(date -u +%Y-%m-%dT%H:%M:%SZ) agent kill: macOS $MAC_HOST")
    elif (( fault_kind == 2 )); then
      eval "$CONTROL_RESTART"
      fault_log+=("$(date -u +%Y-%m-%dT%H:%M:%SZ) control-plane restart: $CONTROL_RESTART")
    else
      eval "$NETWORK_PARTITION"
      sleep "$INTERVAL"
      eval "$NETWORK_RESTORE"
      fault_log+=("$(date -u +%Y-%m-%dT%H:%M:%SZ) network partition: bounded by $INTERVAL seconds")
    fi
  fi
  sleep "$INTERVAL"
done

exit 0
