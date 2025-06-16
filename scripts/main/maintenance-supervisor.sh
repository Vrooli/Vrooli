#!/usr/bin/env bash
###############################################################################
# maintenance-supervisor.sh
#
# Lightweight bash scheduler for maintenance-agent.sh
#
# • Supports three schedule kinds:
#     tight           – run again immediately after each finish
#     every:<secs>    – fixed interval  (e.g. every:900  → every 15 min)
#     at:<HH:MM>      – once daily at given wall-clock time
#
# • Safeguards:
#   - flock locking  (only one supervisor instance)
#   - 120-second hard timeout per agent run (Claude Code tool limit)
#   - skip if previous run of same task still active
#   - privilege drop to FALLBACK_USER
###############################################################################
set -euo pipefail

###############################################################################
# CONFIG (edit here)
###############################################################################
FALLBACK_USER=${FALLBACK_USER:-matt}
AGENT_SCRIPT="/home/${FALLBACK_USER}/scripts/maintenance-agent.sh"
LOG_DIR="/home/${FALLBACK_USER}/agent-logs"
mkdir -p "${LOG_DIR}"

# Task table:  name|schedule|prompt|turns
TASKS=(
  "TEST_QUALITY|every:900|Do maintenance task TEST_QUALITY in the shared package. Make sure you `pnpm run test` modified files and address any failures|50"
  "TEST_COVERAGE|every:3600|Do maintenance task TEST_COVERAGE in the shared package|50"
  "UNSAFE_CASTS|every:3600|Do maintenance task UNSAFE_CASTS in the shared package|100"
  "CODE_QUALITY|every:3600|Do maintenance task CODE_QUALITY in the shared package|100"
  "TODO_CLEANUP|at:05:00|Do maintenance task TODO_CLEANUP in the shared package|300"
)

###############################################################################
# INTERNALS (you normally don’t touch these)
###############################################################################
LOCK_FD=200
exec {LOCK_FD}<>"/tmp/maintenance-supervisor.lock"
flock -n "$LOCK_FD" || {
  echo "🔒 Another maintenance-supervisor.sh is already running. Exiting."
  exit 1
}

trap 'echo -e "\n🚦  Supervisor aborted." >&2' INT TERM

# Always use the real coreutils timeout, never the npm-run-all shim
TIMEOUT_BIN=$(command -v timeout)
[[ $TIMEOUT_BIN == */node_modules/* ]] && TIMEOUT_BIN=/usr/bin/timeout

# --- helpers -----------------------------------------------------------------
next_at() {                        # next epoch for HH:MM TODAY or TOMORROW
  local hhmm=$1
  local tgt
  tgt=$(date -d "$(date +%F) $hhmm" +%s)
  [ "$tgt" -le "$(date +%s)" ] && tgt=$(date -d "tomorrow $hhmm" +%s)
  echo "$tgt"
}

task_running() {                   # returns 0 if a pid file is active
  local pid_file=$1
  [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null
}

run_task() {                       # (name schedule prompt turns)
  local name=$1 prompt=$3 turns=$4
  local start_ts logfile pid_file
  start_ts=$(date +%s)
  logfile="${LOG_DIR}/${name}_$(date +%F_%H%M%S).log"
  pid_file="/tmp/${name}.running"

  echo "[$(date)] ▶️  Starting $name" | tee -a "$logfile"

  # execute agent with 120-second ceiling
  timeout 120s sudo -u "$FALLBACK_USER" -- \
      "$AGENT_SCRIPT" "$prompt" "$turns" &>> "$logfile" &
  local run_pid=$!
  echo "$run_pid" > "$pid_file"

  # wait & harvest exit code
  wait "$run_pid"; exit_code=$?
  rm -f "$pid_file"

  if (( exit_code == 0 )); then
    echo "[$(date)] ✅  $name finished in $(( $(date +%s)-start_ts )) s" | tee -a "$logfile"
  else
    echo "[$(date)] ⚠️  $name exited with code $exit_code"            | tee -a "$logfile"
  fi
}

###############################################################################
# INITIALISE SCHEDULES
###############################################################################
declare -A NEXT_RUN
now=$(date +%s)

for entry in "${TASKS[@]}"; do
  IFS='|' read -r name sched _ <<<"$entry"
  case $sched in
    tight)      NEXT_RUN["$name"]=$now ;;
    every:*)    NEXT_RUN["$name"]=$(( now + ${sched#every:} )) ;;
    at:*)       NEXT_RUN["$name"]=$(next_at "${sched#at:}") ;;
    *)          echo "⛔️ Unknown schedule '$sched' for $name" >&2; exit 1 ;;
  esac
done

###############################################################################
# MAIN LOOP
###############################################################################
while true; do
  now=$(date +%s)
  soonest=$(( now + 365*24*3600 ))  # far future placeholder

  for entry in "${TASKS[@]}"; do
    IFS='|' read -r name sched prompt turns <<<"$entry"
    run_at=${NEXT_RUN["$name"]}
    pid_file="/tmp/${name}.running"

    # Skip if still running
    if task_running "$pid_file"; then
      # pick earliest next check, but don't reschedule yet
      (( run_at < soonest )) && soonest=$run_at
      continue
    fi

    # Launch when time has come
    if (( run_at <= now )); then
      run_task "$name" "$sched" "$prompt" "$turns" &
      case $sched in
        tight)      NEXT_RUN["$name"]=$now ;;                             # run again ASAP
        every:*)    NEXT_RUN["$name"]=$(( now + ${sched#every:} )) ;;
        at:*)       NEXT_RUN["$name"]=$(next_at "${sched#at:}") ;;
      esac
      run_at=${NEXT_RUN["$name"]}
    fi

    (( run_at < soonest )) && soonest=$run_at
  done

  sleep_for=$(( soonest - $(date +%s) ))
  (( sleep_for < 1 )) && sleep_for=1
  sleep "$sleep_for"
done
