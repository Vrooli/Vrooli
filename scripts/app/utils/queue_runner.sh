#!/usr/bin/env bash
# queue_runner.sh  (v4.20)
# Smart gate-keeper for heavy jobs (lint, type-check, etc.)
#
# ▸ Guarantees ≤ MAX_PARALLEL concurrent jobs
# ▸ Refuses to enqueue if ≥ MAX_QUEUE other jobs are already waiting
# ▸ Starts a job only while CPU-load & free-RAM stay healthy
# ▸ Cleans up lock files even on signals; prunes truly stale locks on start-up
#
# Linux only – Bash ≥ 4, GNU coreutils, flock

set -euo pipefail
IFS=$'\n\t'

# ---------------------------------------------------------------------
# CONFIG  (override with env vars:  VAR=value ./queue_runner.sh <cmd …>)
# ---------------------------------------------------------------------
LOCK_DIR="/tmp/monorepo-lock"              # dir for slot / wait files
SLOT_PREFIX="slot"                         # slot.1, slot.2, …
MAX_PARALLEL=${MAX_PARALLEL:-1}            # max concurrent running jobs
MAX_QUEUE=${MAX_QUEUE:-3}                  # max *waiting* jobs
RESERVE_GIB=${RESERVE_GIB:-1}            # keep ≥ N GiB free RAM
STALE_MINUTES=${STALE_MINUTES:-30}         # treat older locks as dead (0 = never)
TIMEOUT=${TIMEOUT:-0}                      # secs to wait for a slot (0 = forever)
STRICT_QUEUE=${STRICT_QUEUE:-0}            # abort if queue grows past limit later
HEADROOM_POLL=5                            # seconds between headroom polls

# Clamp bad env input
(( MAX_PARALLEL > 0 )) || MAX_PARALLEL=1

# ---------------------------------------------------------------------
# HOUSEKEEPING
# ---------------------------------------------------------------------
mkdir -p "$LOCK_DIR"

if (( STALE_MINUTES > 0 )); then
  # -- 1) purge long-dead wait markers outright -----------------------
  find "$LOCK_DIR" -maxdepth 1 -type f -name 'wait.*.lock' \
       -mmin "+$STALE_MINUTES" -delete 2>/dev/null || true

  # -- 2) probe *slot* files and delete only if *not locked* ----------
  while IFS= read -r slot; do
    exec {fd}>>"$slot" || continue           # open (create if absent)
    if flock -n "$fd"; then                  # lock succeeded ⇒ nobody owns it
      rm -f -- "$slot"
    fi
    exec {fd}>&-                            # close test FD
  done < <(find "$LOCK_DIR" -maxdepth 1 -type f -name "${SLOT_PREFIX}.*" \
           -mmin "+$STALE_MINUTES" 2>/dev/null)
fi

# ---------------------------------------------------------------------
# CLEANUP HANDLER
# ---------------------------------------------------------------------
cleanup() {
  [[ -n ${fd:-}            ]] && exec {fd}>&-      # unlock slot on exit
  [[ -n ${acquired_slot:-} ]] && rm -f -- "$acquired_slot"
  [[ -n ${wait_marker:-}   ]] && rm -f -- "$wait_marker"
}
trap cleanup EXIT INT TERM HUP QUIT

# ---------------------------------------------------------------------
# UTILS
# ---------------------------------------------------------------------
queue_len() {
  find "$LOCK_DIR" -maxdepth 1 -type f -name 'wait.*.lock' \
       ! -name "wait.$$.lock" \
       -printf '%f\n' |           # one line per wait-file
       wc -l                      # => exact count (0-safe)
}

enough_headroom() {
  # Restore default IFS locally so /proc/stat splits on spaces
  local IFS=$' \t\n'

  # --- CPU % ---------------------------------------------------------
  read -ra a1 < /proc/stat || return 1
  sleep 0.5
  read -ra a2 < /proc/stat || return 1

  local idle1=$(( a1[4] + a1[5] ));  local idle2=$(( a2[4] + a2[5] ))
  local total1=0 total2=0 v
  for v in "${a1[@]:1}"; do (( total1 += v )); done
  for v in "${a2[@]:1}"; do (( total2 += v )); done

  # avoid divide-by-zero on weird reads
  (( total2 > total1 )) || return 1
  local usage_pct=$(( 100 * (total2 - total1 - (idle2 - idle1)) / (total2 - total1) ))
  CPU_USAGE_PCT=$usage_pct 

  # --- RAM -----------------------------------------------------------
  local avail_kib
  avail_kib=$(awk '/^MemAvailable:/ {print $2; exit}' /proc/meminfo 2>/dev/null || echo 0)
  # Fallback for ancient kernels lacking MemAvailable
  if (( avail_kib == 0 )); then
    avail_kib=$(free -k | awk '/^Mem:/ {print $7; exit}')
  fi
  AVAIL_KIB=$avail_kib

  local reserve_kib=$(( RESERVE_GIB * 1024 * 1024 ))

  (( usage_pct < 90 && avail_kib >= reserve_kib ))
}

touch_slot() { printf '%s\n' "$$" >"$1"; }

randseq() {
  if command -v shuf >/dev/null; then
    shuf -i 1-"$MAX_PARALLEL"
  else
    for ((n=1; n<=MAX_PARALLEL; n++)); do printf '%s\n' "$n"; done
  fi
}

acquire_slot_now() {
  for i in $(randseq); do
    local slot="$LOCK_DIR/$SLOT_PREFIX.$i"
    exec {fd}>>"$slot"               # keep FD open; no O_TRUNC
    if flock -n "$fd"; then
      acquired_slot="$slot"
      touch_slot "$slot"             # refresh mtime for stale-pruning
      echo "🏁  acquired $slot" >&2
      return 0
    fi
    exec {fd}>&-                     # unlock & try next slot
  done
  return 1
}

announce_slot_wait() {
  local queued running
  queued=$(queue_len)
  running=$(find "$LOCK_DIR" -maxdepth 1 -type f -name "${SLOT_PREFIX}.*" | wc -l)
  echo "⌛  No free slot yet — $running running, $queued queued (limit $MAX_QUEUE). Waiting…" >&2
}

announce_headroom_wait() {
  # convert KiB → GiB with one decimal
  local avail_gib
  avail_gib=$(awk -v k="$AVAIL_KIB" 'BEGIN{printf "%.1f", k/1024/1024}')

  echo "💤  System busy — CPU ${CPU_USAGE_PCT}%  |  RAM ${avail_gib} GiB free." \
       "(need CPU < 90 % and ≥ ${RESERVE_GIB} GiB free). Polling every ${HEADROOM_POLL}s…" >&2
}

wait_for_slot() {
  local start_ts announced=0
  start_ts=$(date +%s)
  wait_marker="$LOCK_DIR/wait.$$.lock"; : >"$wait_marker"

  while :; do
    acquire_slot_now && { rm -f -- "$wait_marker"; wait_marker=""; return 0; }

    (( announced )) || { announce_slot_wait; announced=1; }

    if (( TIMEOUT > 0 )); then
      local now_ts
      now_ts=$(date +%s)
      (( now_ts - start_ts >= TIMEOUT )) && {
        echo "⏳  Couldn't get slot within ${TIMEOUT}s — giving up." >&2
        return 1
      }
    fi

    sleep 1
  done
}

# ---------------------------------------------------------------------
# MAIN SLOT-ACQUISITION LOGIC
# ---------------------------------------------------------------------
(( $# )) || { echo "Usage: $0 <cmd …>" >&2; exit 2; }

acquired_slot=""
acquire_slot_now || wait_for_slot || exit 0   # refused / timed-out

# ---------------------------------------------------------------------
# OPTIONAL SECOND GUARD – still okay with queue size?
# ---------------------------------------------------------------------
if (( STRICT_QUEUE )) && (( $(queue_len) >= MAX_QUEUE )); then
  echo "🚦  Queue limit exceeded after lock — exiting (STRICT_QUEUE)." >&2
  exit 0
fi

# ---------------------------------------------------------------------
# FINAL HEADROOM CHECK(S)
# ---------------------------------------------------------------------
headroom_announced=0
while ! enough_headroom; do
  (( headroom_announced )) || { announce_headroom_wait; headroom_announced=1; }
  sleep "$HEADROOM_POLL"
done

# ---------------------------------------------------------------------
# GO GO GO   (FD stays open → lock persists while child runs)
# ---------------------------------------------------------------------
"$@"
exit "$?"