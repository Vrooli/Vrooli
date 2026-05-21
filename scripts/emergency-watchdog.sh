#!/usr/bin/env bash
# emergency-watchdog.sh — Pure-POSIX last-line-of-defense.
#
# Runs every 5 minutes from systemd timer; checks both the runtime supervisor
# and the autoheal loop. If either has been inactive for more than the
# threshold (default 10 minutes), runs `go mod download` at the repo root,
# then restarts both units. Has no Go dependency itself: if Go is fully
# broken the script can still detect the outage and trigger the restart,
# which exercises the systemd ExecStartPre known-good fallback.
#
# Hysteresis prevents flapping: state file records the first observed
# unhealthy timestamp; recovery only fires once enough wall-clock has passed.

set -u

VROOLI_ROOT="${VROOLI_ROOT:-/home/matthalloran8/Vrooli}"
STATE_DIR="${HOME}/.vrooli/state"
LOG_FILE="${HOME}/.vrooli/logs/emergency-watchdog.log"
LAST_FAIL_FILE="${STATE_DIR}/emergency-watchdog.last-fail"
THRESHOLD_SECONDS="${EMERGENCY_WATCHDOG_THRESHOLD:-600}"

mkdir -p "$STATE_DIR" "$(dirname "$LOG_FILE")"

log() {
  printf '%s %s\n' "$(date -Iseconds)" "$*" >>"$LOG_FILE"
}

is_active() {
  systemctl --user is-active --quiet "$1"
}

now() { date +%s; }

UNITS="vrooli-runtime-supervisor.service vrooli-autoheal.service"

any_down=0
for u in $UNITS; do
  if ! is_active "$u"; then
    any_down=1
  fi
done

if [ "$any_down" -eq 0 ]; then
  # Healthy — clear hysteresis state.
  rm -f "$LAST_FAIL_FILE"
  exit 0
fi

# At least one unit is down — apply hysteresis.
first_fail=""
if [ -f "$LAST_FAIL_FILE" ]; then
  first_fail="$(cat "$LAST_FAIL_FILE" 2>/dev/null || true)"
fi
if [ -z "$first_fail" ] || ! [ "$first_fail" -gt 0 ] 2>/dev/null; then
  printf '%s\n' "$(now)" >"$LAST_FAIL_FILE"
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
  (cd "$VROOLI_ROOT" && go mod download 2>>"$LOG_FILE") || log "go mod download exited non-zero"
else
  log "skipping go mod download (no go.mod or go binary)"
fi

# Attempt 2: restart the systemd units; ExecStartPre will swap in known-good
# binaries if the live ones are corrupt.
for u in $UNITS; do
  if systemctl --user restart "$u" 2>>"$LOG_FILE"; then
    log "restart ok: $u"
  else
    log "restart FAILED: $u"
  fi
done

# Reset hysteresis so we don't immediately escalate again; the next tick
# will re-observe the situation freshly.
rm -f "$LAST_FAIL_FILE"
exit 0
