#!/usr/bin/env bash
###############################################################################
# maintenance-supervisor.sh
#
# Lightweight bash scheduler for maintenance-agent.sh
#
#   • Schedule kinds
#       tight           – run again immediately after each finish
#       every:<secs>    – fixed interval  (every 900 → every 15 min)
#       at:<HH:MM>      – once daily at given wall-clock time
#
#   • Safeguards
#       - flock locking  (only one supervisor instance per user)
#       - Kills the agent only if it stops writing to its logfile for LONG_ENOUGH_S.
#       - skip if previous run of same task still active
#       - privilege drop to FALLBACK_USER
###############################################################################
set -euo pipefail

###############################################################################
# CONFIG (edit here)
###############################################################################
FALLBACK_USER=${FALLBACK_USER:-matt}
LONG_ENOUGH_S=${LONG_ENOUGH_S:-360}
TIGHT_COOLDOWN_S=${TIGHT_COOLDOWN_S:-5}

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
AGENT_SCRIPT="${SCRIPT_DIR}/maintenance-agent.sh"
if [[ ! -x "$AGENT_SCRIPT" ]]; then
  echo "⛔️  Agent script not found or not executable: $AGENT_SCRIPT" >&2
  exit 1
fi

LOG_DIR="/home/${FALLBACK_USER}/agent-logs"
mkdir -p "${LOG_DIR}"
# ensure writable even if root created it first
if [[ $EUID -eq 0 ]]; then   ## only chown when we actually can
  chown "$FALLBACK_USER":"$FALLBACK_USER" "$LOG_DIR" 2>/dev/null || true
fi

# Task table:  name|schedule|prompt|turns
TASKS=(
  'TEST_QUALITY|every:900|Do maintenance task TEST_QUALITY in the shared package. Make sure you `pnpm run test` modified files and address any failures|50'
  'TEST_COVERAGE|every:3600|Do maintenance task TEST_COVERAGE in the shared package|50'
  'UNSAFE_CASTS|every:3600|Do maintenance task UNSAFE_CASTS in the shared package|100'
  'CODE_QUALITY|every:3600|Do maintenance task CODE_QUALITY in the shared package|100'
  'TODO_CLEANUP|at:05:00|Do maintenance task TODO_CLEANUP in the shared package|300'
)

###############################################################################
# INTERNALS (you normally don’t touch these)
###############################################################################
# ----- locking ---------------------------------------------------------------
LOCK_USER=${FALLBACK_USER:-$USER}                      # fallback for clarity
LOCK_FILE="/tmp/maintenance-supervisor.${LOCK_USER}.lock"
LOCK_FD=200

# create/touch first so the chmod succeeds even on first run
touch "$LOCK_FILE" 2>/dev/null || {
  echo "⛔️  Cannot create lock file $LOCK_FILE" >&2
  exit 1
}
chmod 600 "$LOCK_FILE"
# if root created it earlier, transfer ownership
if [[ $EUID -eq 0 ]]; then
  chown "$LOCK_USER":"$LOCK_USER" "$LOCK_FILE" 2>/dev/null || true
fi

exec {LOCK_FD}<> "$LOCK_FILE" || {
  echo "⛔️  Cannot open lock file $LOCK_FILE" >&2
  exit 1
}

flock -n "$LOCK_FD" || {
  echo "🔒 Another maintenance-supervisor for user '$LOCK_USER' is already running." >&2
  exit 1
}

# clean up lock file in all cases
trap 'rm -f "$LOCK_FILE"' EXIT
trap 'echo -e "\n🚦  Supervisor exited." >&2; exit' INT TERM

# --- helpers -----------------------------------------------------------------
# choose coreutils date (gdate on macOS, date elsewhere)
DATE_BIN=$(command -v gdate || command -v date)

# return mtime (epoch s) portable across GNU/BSD
get_mtime() {
  local file=$1
  if "$DATE_BIN" --version >/dev/null 2>&1; then
    "$DATE_BIN" -r "$file" +%s
  else
    stat -f %m "$file"
  fi
}

# conditional sudo drop (see note-to-AI)
if [[ $(id -u) -eq 0 ]]; then
  SUDO=(sudo -u "$FALLBACK_USER" --)
elif [[ $(id -un) == "$FALLBACK_USER" ]]; then
  SUDO=()
else
  echo "⛔️  Must be run as root or ${FALLBACK_USER}; refusing to continue." >&2
  exit 1
fi

next_at() {             # HH:MM → next epoch (today or tomorrow)
  local hhmm=$1 tgt
  tgt=$("$DATE_BIN" -d "$( "$DATE_BIN" +%F ) $hhmm" +%s 2>/dev/null || true)
  if [[ -z $tgt ]]; then  # fallback for BSD date w/out -d
    local today=$("$DATE_BIN" +%F)
    tgt=$("$DATE_BIN" -j -f "%F %H:%M" "$today $hhmm" +%s)
  fi
  if [[ $tgt -le $("$DATE_BIN" +%s) ]]; then
    tgt=$("$DATE_BIN" -d "tomorrow $hhmm" +%s 2>/dev/null || \
          "$DATE_BIN" -j -f "%F %H:%M" "$( "$DATE_BIN" -v+1d +%F ) $hhmm" +%s)
  fi
  echo "$tgt"
}

task_running() {        # pid-file → 0 if process alive
  local pid_file=$1
  [[ -f $pid_file ]] && kill -0 "$(cat "$pid_file")" 2>/dev/null
}

run_task() {
  local name=$1 prompt=$2 turns=$3
  local logfile="${LOG_DIR}/${name}_$("$DATE_BIN" +%F_%H%M%S).log"
  :> "$logfile"                       # create empty file for watchdog
  local pid_file="/tmp/${name}.running"

  echo "[$("$DATE_BIN")] ▶️  Starting $name" | tee -a "$logfile"

  "${SUDO[@]}" "$AGENT_SCRIPT" "$prompt" "$turns" >>"$logfile" 2>&1
  local run_pid=$!
  echo "$run_pid" > "$pid_file"

  # ---------- watchdog -------------------------------------------------------
  (
    local idle_secs=$LONG_ENOUGH_S
    local check_int=2
    while kill -0 "$run_pid" 2>/dev/null; do
      local last_mod=$(get_mtime "$logfile")
      local now=$("$DATE_BIN" +%s)
      if (( now - last_mod > idle_secs )); then
        echo "[$("$DATE_BIN")] ⏱  No log output for $idle_secs s – terminating $name" | tee -a "$logfile"
        pkill -TERM -P "$run_pid" 2>/dev/null || true
        kill -TERM "$run_pid" 2>/dev/null || true
        sleep 3
        kill -KILL "$run_pid" 2>/dev/null || true
        break
      fi
      sleep "$check_int"
    done
  ) &
  local watch_pid=$!

  wait "$run_pid"; local ec=$?
  kill "$watch_pid" 2>/dev/null || true
  rm -f "$pid_file"

  if (( ec == 0 )); then
    echo "[$("$DATE_BIN")] ✅  $name finished" | tee -a "$logfile"
  else
    echo "[$("$DATE_BIN")] ⚠️  $name exit code $ec" | tee -a "$logfile"
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
    tight)      NEXT_RUN["$name"]=0 ;;
    every:*)    NEXT_RUN["$name"]=$(( now + ${sched#every:} )) ;;
    at:*)       NEXT_RUN["$name"]=$(next_at "${sched#at:}") ;;
    *)          echo "⛔️ Unknown schedule '$sched' for $name" >&2; exit 1 ;;
  esac
done

# remove stale pid-files on startup
for entry in "${TASKS[@]}"; do
  IFS='|' read -r name _ <<<"$entry"
  pid_file="/tmp/${name}.running"
  task_running "$pid_file" || rm -f "$pid_file"
done

###############################################################################
# MAIN LOOP
###############################################################################
while true; do
  now=$("$DATE_BIN" +%s)
  soonest=$(( now + 365*24*3600 ))      # far future placeholder

  for entry in "${TASKS[@]}"; do
    IFS='|' read -r name sched prompt turns <<<"$entry"
    run_at=${NEXT_RUN["$name"]}
    pid_file="/tmp/${name}.running"

    # still running → just pick earliest next check
    if task_running "$pid_file"; then
      (( run_at < soonest )) && soonest=$run_at
      continue
    fi

    # time to launch?
    if (( run_at <= now )); then
      run_task "$name" "$prompt" "$turns" &
      case $sched in
        tight)   NEXT_RUN["$name"]=$(( $( "$DATE_BIN" +%s ) + TIGHT_COOLDOWN_S )) ;;  # cooldown
        every:*) NEXT_RUN["$name"]=$(( $( "$DATE_BIN" +%s ) + ${sched#every:} )) ;;
        at:*)    NEXT_RUN["$name"]=$(next_at "${sched#at:}") ;;
      esac
      run_at=${NEXT_RUN["$name"]}
    fi

    (( run_at < soonest )) && soonest=$run_at
  done

  sleep_for=$(( soonest - $( "$DATE_BIN" +%s ) ))
  (( sleep_for < 1 )) && sleep_for=1
  sleep "$sleep_for"
done
