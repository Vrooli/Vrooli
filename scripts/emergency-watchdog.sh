#!/usr/bin/env bash
# emergency-watchdog.sh — Pure-POSIX last-line-of-defense.
#
# Runs every 5 minutes from a systemd timer. Checks two things:
#
#   1. Unit liveness — the runtime supervisor and the autoheal loop.
#   2. Free disk    — the condition that took the host down on 2026-07-31.
#
# Has no Go dependency itself: if Go is fully broken the script can still
# detect the outage and trigger the restart, which exercises the systemd
# ExecStartPre known-good fallback.
#
# Hysteresis prevents flapping: state files record the first observed
# unhealthy timestamp; recovery only fires once enough wall-clock has passed.
#
# DISK AWARENESS. During the incident this script was a victim rather than a
# responder: it had no disk check at all, and its own logging failed with
# "printf: write error: No space left on device" at 08:47:02. Both gaps are
# fixed here — it now watches free space, and its logging is bounded and
# tolerant of a full disk.

set -u

VROOLI_ROOT="${VROOLI_ROOT:-/home/matthalloran8/Vrooli}"
STATE_DIR="${HOME}/.vrooli/state"
LOG_FILE="${HOME}/.vrooli/logs/emergency-watchdog.log"
LAST_FAIL_FILE="${STATE_DIR}/emergency-watchdog.last-fail"
LAST_DISK_FILE="${STATE_DIR}/emergency-watchdog.last-disk"
THRESHOLD_SECONDS="${EMERGENCY_WATCHDOG_THRESHOLD:-600}"

# Free-space floor in MiB. Below this the host is treated as unhealthy and
# cleanup is requested. The default reserves enough room for the supervisor to
# write, the journal to rotate, and this script to log — the three things that
# all failed simultaneously during the incident.
DISK_FLOOR_MB="${EMERGENCY_WATCHDOG_DISK_FLOOR_MB:-10240}"

# Disk pressure escalates faster than a dead unit: a filesystem crossing the
# floor is already an emergency, and waiting ten minutes to react is most of
# the way to zero.
DISK_THRESHOLD_SECONDS="${EMERGENCY_WATCHDOG_DISK_THRESHOLD:-120}"

# Watched filesystem. Root and home share one filesystem on this host, so one
# mount point covers both.
WATCH_MOUNT="${EMERGENCY_WATCHDOG_MOUNT:-/}"

# Bound the log so the watchdog can never become a disk-pressure source itself.
LOG_MAX_BYTES="${EMERGENCY_WATCHDOG_LOG_MAX_BYTES:-1048576}"

mkdir -p "$STATE_DIR" "$(dirname "$LOG_FILE")" 2>/dev/null || true

# rotate_log keeps the log bounded. A watchdog that grows its own log without
# limit is a slow version of the problem it exists to catch.
rotate_log() {
  [ -f "$LOG_FILE" ] || return 0
  size=$( ( wc -c <"$LOG_FILE" ) 2>/dev/null || echo 0 )
  [ "$size" -gt "$LOG_MAX_BYTES" ] 2>/dev/null || return 0
  # Keep the tail; the recent history is what diagnoses an incident.
  ( tail -c $((LOG_MAX_BYTES / 2)) "$LOG_FILE" >"${LOG_FILE}.tmp" ) 2>/dev/null &&
    mv "${LOG_FILE}.tmp" "$LOG_FILE" 2>/dev/null
  return 0
}

# log never fails the script. During the incident a failed write here exited
# non-zero and took the watchdog down with the disk — the one moment it most
# needed to keep running.
# The subshell matters: when the redirection itself fails (a full or
# unwritable filesystem), bash reports it on stderr regardless of any
# redirection inside the command. Under systemd that stderr becomes journal
# writes — more disk pressure, at the exact moment there is none to spare.
# Wrapping the whole thing keeps the watchdog silent and alive.
log() {
  ( printf '%s %s\n' "$(date -Iseconds)" "$*" >>"$LOG_FILE" ) 2>/dev/null || true
  return 0
}

is_active() {
  systemctl --user is-active --quiet "$1"
}

now() { date +%s; }

# available_mb reports space available to an unprivileged writer, in MiB.
#
# It reads df's Available column, not Free. Free includes the superuser
# reserve, which on the incident host was 93 GB — enough to keep every
# threshold looking comfortable while the filesystem was unwritable for the
# supervisor. This is the same Bavail-vs-Bfree distinction the Go safeguards
# were corrected for.
available_mb() {
  df -PBM "$WATCH_MOUNT" 2>/dev/null | awk 'NR==2 {gsub(/M/,"",$4); print $4; found=1} END {if (!found) print ""}'
}

# read_state prints a stored epoch, or nothing when unset or malformed.
read_state() {
  [ -f "$1" ] || return 0
  value="$(cat "$1" 2>/dev/null || true)"
  [ -n "$value" ] && [ "$value" -gt 0 ] 2>/dev/null && printf '%s' "$value"
  return 0
}

# request_cleanup asks storage-manager to reclaim safe-tier space.
#
# Deliberately shells out to the CLI rather than linking anything: this script
# must keep working when the Go toolchain is broken, and the CLI is a
# prebuilt binary. A missing CLI is logged and skipped, never fatal.
request_cleanup() {
  band="$1"
  used_percent="$2"

  if ! command -v storage-manager >/dev/null 2>&1; then
    log "storage-manager CLI not on PATH; cannot request reclamation"
    return 0
  fi

  if ( storage-manager cleanup report-pressure \
    --partition "$WATCH_MOUNT" \
    --band "$band" \
    --used-percent "$used_percent" \
    --source emergency-watchdog >>"$LOG_FILE" 2>&1 ) 2>/dev/null; then
    log "requested $band cleanup for $WATCH_MOUNT"
  else
    log "cleanup request FAILED for $WATCH_MOUNT"
  fi
  return 0
}

rotate_log

# ---------------------------------------------------------------------------
# Disk check
# ---------------------------------------------------------------------------

avail="$(available_mb)"
if [ -z "$avail" ]; then
  log "could not read available space on $WATCH_MOUNT"
elif [ "$avail" -ge "$DISK_FLOOR_MB" ] 2>/dev/null; then
  # Healthy — clear disk hysteresis.
  rm -f "$LAST_DISK_FILE" 2>/dev/null || true
else
  used_percent="$(df -P "$WATCH_MOUNT" 2>/dev/null | awk 'NR==2 {gsub(/%/,"",$5); print $5}')"
  [ -n "$used_percent" ] || used_percent=0

  first_disk_fail="$(read_state "$LAST_DISK_FILE")"
  if [ -z "$first_disk_fail" ]; then
    printf '%s\n' "$(now)" >"$LAST_DISK_FILE" 2>/dev/null || true
    log "first observed low disk: ${avail}MB available on $WATCH_MOUNT (floor ${DISK_FLOOR_MB}MB, ${used_percent}% used)"
  else
    disk_elapsed=$(( $(now) - first_disk_fail ))
    if [ "$disk_elapsed" -lt "$DISK_THRESHOLD_SECONDS" ]; then
      log "low disk ${disk_elapsed}s/${DISK_THRESHOLD_SECONDS}s — not yet escalating (${avail}MB available)"
    else
      log "ESCALATING: ${avail}MB available on $WATCH_MOUNT for ${disk_elapsed}s (floor ${DISK_FLOOR_MB}MB, ${used_percent}% used)"
      # Half the floor is the point where the supervisor and the journal are
      # at risk, so authorise unattended reclamation rather than a preview.
      if [ "$avail" -lt $(( DISK_FLOOR_MB / 2 )) ] 2>/dev/null; then
        request_cleanup critical "$used_percent"
      else
        request_cleanup high "$used_percent"
      fi
      # Reset so the next tick re-observes freshly rather than escalating
      # every five minutes while cleanup is still taking effect.
      rm -f "$LAST_DISK_FILE" 2>/dev/null || true
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Unit liveness check
# ---------------------------------------------------------------------------

UNITS="vrooli-runtime-supervisor.service vrooli-autoheal.service"

any_down=0
for u in $UNITS; do
  if ! is_active "$u"; then
    any_down=1
  fi
done

if [ "$any_down" -eq 0 ]; then
  # Healthy — clear hysteresis.
  rm -f "$LAST_FAIL_FILE" 2>/dev/null || true
  exit 0
fi

first_fail="$(read_state "$LAST_FAIL_FILE")"
if [ -z "$first_fail" ]; then
  printf '%s\n' "$(now)" >"$LAST_FAIL_FILE" 2>/dev/null || true
  log "first observed unhealthy: $UNITS"
  exit 0
fi

elapsed=$(( $(now) - first_fail ))
if [ "$elapsed" -lt "$THRESHOLD_SECONDS" ]; then
  log "unhealthy ${elapsed}s/${THRESHOLD_SECONDS}s — not yet escalating"
  exit 0
fi

log "ESCALATING: units unhealthy for ${elapsed}s"

# Attempt 1: cheap, non-mutating dependency refresh at the repo root.
if [ -f "$VROOLI_ROOT/go.mod" ] && command -v go >/dev/null 2>&1; then
  ( cd "$VROOLI_ROOT" && go mod download 2>>"$LOG_FILE" ) 2>/dev/null || log "go mod download exited non-zero"
else
  log "skipping go mod download (no go.mod or go binary)"
fi

# Attempt 2: restart the systemd units; ExecStartPre will swap in known-good
# binaries if the live ones are corrupt.
for u in $UNITS; do
  if ( systemctl --user restart "$u" 2>>"$LOG_FILE" ) 2>/dev/null; then
    log "restart ok: $u"
  else
    log "restart FAILED: $u"
  fi
done

# Reset hysteresis so we don't immediately escalate again; the next tick
# will re-observe the situation freshly.
rm -f "$LAST_FAIL_FILE" 2>/dev/null || true
exit 0
