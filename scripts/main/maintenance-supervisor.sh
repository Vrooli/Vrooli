#!/usr/bin/env bash
###############################################################################
# maintenance-supervisor.sh
#
# Lightweight bash scheduler for maintenance-agent.sh
#
#   Schedule kinds
#     tight            – run again immediately after each finish (round-robin)
#     every:<secs>     – fixed interval (e.g. every:900 → every 15 min)
#     at:<HH:MM>       – once daily at given wall-clock time
#
#   Behaviour
#     • At most **one** maintenance-agent process at any time.
#     • Any “every:” job that becomes due is pushed onto a FIFO queue
#       (duplicates ignored).  The queue is drained **before** tight jobs run.
#     • Tight jobs run in round-robin order with a configurable cooldown.
#
#   Safeguards
#     • flock locking (one supervisor per user)
#     • watchdog kills agent if its logfile is idle for LONG_ENOUGH_S seconds
#     • skip if previous run of same task still active
#     • privilege drop to FALLBACK_USER when invoked as root
###############################################################################
set -euo pipefail

###############################################################################
# CONFIG (edit here)
###############################################################################
FALLBACK_USER=${FALLBACK_USER:-matt}
LONG_ENOUGH_S=${LONG_ENOUGH_S:-360}
TIGHT_COOLDOWN_S=${TIGHT_COOLDOWN_S:-30} # NOTE: Make sure this is greater than 0, to give non-tight jobs a chance to run

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
  'TEST_QUALITY|tight|Do maintenance task TEST_QUALITY in the shared package. Make sure you `pnpm run test` modified files and address any failures|50'
  'TEST_COVERAGE|every:1800|Do maintenance task TEST_COVERAGE in the shared package|50'
  'TYPE_SAFETY|every:3600|Do maintenance task TYPE_SAFETY in the shared package|100'
  'CODE_QUALITY|every:3600|Do maintenance task CODE_QUALITY in the shared package|100'
  'TODO_CLEANUP|at:05:00|Do maintenance task TODO_CLEANUP in the shared package|300'
)

###############################################################################
# PRE-FLIGHT CHECKS
###############################################################################
[[ -x $AGENT_SCRIPT ]] || { echo "⛔️  Agent not executable: $AGENT_SCRIPT" >&2; exit 1; }

# ensure log dir writable even if created by root previously
[[ $EUID -eq 0 ]] && chown "$FALLBACK_USER:$FALLBACK_USER" "$LOG_DIR" 2>/dev/null || true

###############################################################################
# INSTANCE LOCK
###############################################################################
LOCK_USER=${FALLBACK_USER:-$USER}
LOCK_FILE="/tmp/maintenance-supervisor.${LOCK_USER}.lock"
LOCK_FD=200

touch "$LOCK_FILE" && chmod 600 "$LOCK_FILE"
[[ $EUID -eq 0 ]] && chown "$LOCK_USER:$LOCK_USER" "$LOCK_FILE" 2>/dev/null || true

exec {LOCK_FD}<> "$LOCK_FILE" || { echo "⛔️  Cannot open lock file" >&2; exit 1; }
flock -n "$LOCK_FD" || { echo "🔒  Another supervisor for '$LOCK_USER' is already running." >&2; exit 0; }
trap 'rm -f "$LOCK_FILE"' EXIT

echo "🔍  maintenance-supervisor started for user '$FALLBACK_USER'"

###############################################################################
# UTILITIES
###############################################################################
DATE_BIN=$(command -v gdate || command -v date)

get_mtime() {           # portable file mtime (epoch)
  local f=$1
  "$DATE_BIN" -r "$f" +%s 2>/dev/null || stat -f %m "$f"
}

next_at() {             # HH:MM → next epoch (today or tomorrow)
  local hhmm=$1
  local tgt=$("$DATE_BIN" -d "$( $DATE_BIN +%F ) $hhmm" +%s 2>/dev/null || true)
  [[ -z $tgt ]] && tgt=$("$DATE_BIN" -j -f "%F %H:%M" "$( $DATE_BIN +%F ) $hhmm" +%s)
  (( tgt <= $("$DATE_BIN" +%s) )) && \
    tgt=$("$DATE_BIN" -d "tomorrow $hhmm" +%s 2>/dev/null || \
      "$DATE_BIN" -j -v+1d -f "%F %H:%M" "$( $DATE_BIN +%F ) $hhmm" +%s)
  echo "$tgt"
}

# privilege handling ----------------------------------------------------------
if [[ $(id -u) -eq 0 ]]; then
  SUDO=(sudo -u "$FALLBACK_USER" --)
elif [[ $(id -un) == "$FALLBACK_USER" ]]; then
  SUDO=()
else
  echo "⛔️  Must be run as root or ${FALLBACK_USER}" >&2; exit 1
fi

# running-agent detection (exact cmdline match prevents supervisor collision)
agent_running() { pgrep -f -- "$AGENT_SCRIPT" >/dev/null; }

task_running() {        # pid file → 0 if process alive
  local pf=$1
  [[ -f $pf ]] && kill -0 "$(cat "$pf")" 2>/dev/null
}

###############################################################################
# WATCHDOG + LAUNCHER
###############################################################################
run_task() {
  local name=$1 prompt=$2 turns=$3
  local logfile="${LOG_DIR}/${name}_$("$DATE_BIN" +%F_%H%M%S).log"
  :> "$logfile"                                 # touch so watchdog has a file
  local pid_file="/tmp/${name}.running"

  echo "[$("$DATE_BIN")] ▶️  Starting $name"            | tee -a "$logfile"

  # ---------- launch agent ---------------------------------------------------
  "${SUDO[@]}" "$AGENT_SCRIPT" "$prompt" "$turns" >>"$logfile" 2>&1 &
  local run_pid=$!
  echo "$run_pid" > "$pid_file"

  # ---------- watchdog: kill if no output for LONG_ENOUGH_S ------------------
  (
    local idle=$LONG_ENOUGH_S last now
    while kill -0 "$run_pid" 2>/dev/null; do
      last=$(get_mtime "$logfile")
      now=$("$DATE_BIN" +%s)
      if (( now - last > idle )); then
        echo "[$("$DATE_BIN")] ⏱  Idle >${idle}s – watchdog terminating $name" \
          | tee -a "$logfile"
        pkill -TERM -P "$run_pid" 2>/dev/null || true      # children first
        kill  -TERM  "$run_pid" 2>/dev/null || true
        sleep 3 && kill -KILL "$run_pid" 2>/dev/null || true
        break
      fi
      sleep 2
    done
  ) &
  local watch_pid=$!

  # ---------- wait (DON’T let ‘set -e’ kill the supervisor) ------------------
  set +e                              ### NEW: temporarily disable errexit
  wait "$run_pid"
  local ec=$?
  set -e                              ### NEW: re-enable errexit
  kill "$watch_pid" 2>/dev/null || true
  rm -f "$pid_file"

  # ---------- work out exit reason ------------------------------------------
  local reason
  if grep -qiE 'Idle .*watchdog' "$logfile";        then reason="idle_timeout"
  elif grep -qiE 'max.?turns|turn.?limit' "$logfile"; then reason="max_turns"
  elif (( ec == 0 ));                               then reason="success"
  else                                                   reason="exit_code_$ec"
  fi

  echo "[$("$DATE_BIN")] ⏹️  $name finished – ${reason}" | tee -a "$logfile"
}

###############################################################################
# SCHEDULER STATE
###############################################################################
declare -A NEXT_RUN         # job → next epoch timestamp
declare -A IN_QUEUE         # hash-set: job currently queued?
EVERY_QUEUE=()              # FIFO of "every:" jobs
TIGHT_NAMES=()              # ordered list of tight jobs
TIGHT_INDEX=0               # round-robin pointer

init_schedules() {
  local now=$("$DATE_BIN" +%s)
  for entry in "${TASKS[@]}"; do
    IFS='|' read -r name sched _ <<<"$entry"
    case $sched in
      tight)      NEXT_RUN["$name"]=0;                 TIGHT_NAMES+=("$name")   ;;
      every:*)    NEXT_RUN["$name"]=$(( now + ${sched#every:} ))               ;;
      at:*)       NEXT_RUN["$name"]=$(next_at "${sched#at:}")                  ;;
      *)          echo "⛔️  bad schedule for $name" >&2; exit 1 ;;
    esac
  done
}

enqueue_due_every() {
  local now=$("$DATE_BIN" +%s)
  for entry in "${TASKS[@]}"; do
    IFS='|' read -r name sched prompt turns <<<"$entry"
    [[ $sched != every:* ]] && continue

    local pf="/tmp/${name}.running"
    if (( ${NEXT_RUN[$name]} <= now )) \
       && [[ -z ${IN_QUEUE[$name]-} ]] \
       && ! task_running "$pf"; then         # <-- new guard
      EVERY_QUEUE+=("$name|$prompt|$turns")
      IN_QUEUE[$name]=1
      NEXT_RUN[$name]=$(( now + ${sched#every:} ))
    fi
  done
}

pick_next_task() {
  local now=$("$DATE_BIN" +%s)

  # 1) any due at:HH:MM job wins immediately
  for entry in "${TASKS[@]}"; do
    IFS='|' read -r name sched prompt turns <<<"$entry"
    [[ $sched != at:* ]] && continue
    local pf="/tmp/${name}.running"
    if (( now >= ${NEXT_RUN[$name]} )) && ! task_running "$pf"; then
      NEXT_RUN["$name"]=$(next_at "${sched#at:}")   # schedule tomorrow
      echo "$name|$prompt|$turns"
      return
    fi
  done

  # 2) every-queue wins
  if ((${#EVERY_QUEUE[@]})); then
    local item=${EVERY_QUEUE[0]}
    EVERY_QUEUE=("${EVERY_QUEUE[@]:1}")
    unset 'IN_QUEUE['"$(cut -d'|' -f1 <<<"$item")"']'
    echo "$item"
    return
  fi

  # 3) tight round-robin, but only if cool-down expired
  local now=$("$DATE_BIN" +%s) cnt=${#TIGHT_NAMES[@]}
  for ((i=0;i<cnt;i++)); do
    local idx=$(( (TIGHT_INDEX + i) % cnt ))
    local name=${TIGHT_NAMES[idx]}
    if (( now >= ${NEXT_RUN[$name]} )); then
      TIGHT_INDEX=$(( (idx + 1) % cnt ))   # advance pointer
      local entry
      for e in "${TASKS[@]}"; do [[ $e == "$name|"* ]] && entry=$e && break; done
      IFS='|' read -r _ _ prompt turns <<<"$entry"
      echo "$name|$prompt|$turns"
      return
    fi
  done

  echo "" # no task found
}

# remove stale pid files on startup
cleanup_pidfiles() {
  set +e            # CHG
  for entry in "${TASKS[@]}"; do
    IFS='|' read -r name _ <<<"$entry"
    local pf="/tmp/${name}.running"
    task_running "$pf" || rm -f "$pf"
  done
  set -e            # CHG
}

###############################################################################
# MAIN LOOP
###############################################################################
init_schedules
cleanup_pidfiles

while true; do
  enqueue_due_every

  # if an agent already running, wait a little
  if agent_running; then
    sleep 2
    continue
  fi

  task=$(pick_next_task)
  if [[ -z $task ]]; then
    sleep 2
    continue
  fi

  IFS='|' read -r name prompt turns <<<"$task"
  run_task "$name" "$prompt" "$turns"

  # reschedule tight job relative to finish-time
  local now=$("$DATE_BIN" +%s)
  for t in "${TIGHT_NAMES[@]}"; do
    [[ $t == "$name" ]] && NEXT_RUN["$t"]=$(( now + TIGHT_COOLDOWN_S ))
  done
done
