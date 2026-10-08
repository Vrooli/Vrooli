#!/usr/bin/env bats

# This is deliberately opt-in. The case writes real blocks for the requested
# duration and must only run on a host prepared for the disk-recovery proof.
setup() {
	if [ "${STORAGE_MANAGER_CHAOS_E2E:-}" != "1" ]; then
		skip "set STORAGE_MANAGER_CHAOS_E2E=1 on a prepared host"
	fi
	base="${VROOLI_HOME:-$HOME/.vrooli}/tmp/go-work"
	root="$(mktemp -d "$base/storage-manager-chaos.XXXXXX")"
	# The chaos command must reclaim its governed file through recovery. Once
	# that completes, only the empty test directory remains; do not bypass the
	# storage controller with recursive deletion in the suite cleanup.
	trap 'rmdir "$root" 2>/dev/null || true' EXIT
}

@test "rate-aware recovery bounds a governed chaos writer" {
	duration="${STORAGE_MANAGER_CHAOS_DURATION:-8m}"
	events_base="${VROOLI_EVENTS_API_BASE:?VROOLI_EVENTS_API_BASE is required for chaos evidence}"
	started_at="$(date +%s)"
	output="$(storage-manager recovery chaos --root "$root" --rate 20GiB/h --duration "$duration" --json)"
	ended_at="$(date +%s)"
	history="$(storage-manager recovery history --limit 1 --json)"
	run_id="$(echo "$history" | jq -r '.runs[0].run_id')"

	# The command returns the server-owned run identity and terminal result. The
	# event and ledger assertions are retained in the JSON evidence emitted by
	# the command and queried by the plan's final proof step.
	echo "$output" | grep -q "recovery-"
	echo "$output" | grep -q "complete"
	[ -n "$run_id" ]
	[ "$((ended_at - started_at))" -lt 600 ]
	bytes="$(du -sb "$root" | awk '{print $1}')"
	[ "$bytes" -le 1073741824 ]
	hot="$(curl -fsS --max-time 5 "$events_base/api/v1/events?type=storage.writer.hot&limit=100")"
	cooled="$(curl -fsS --max-time 5 "$events_base/api/v1/events?type=storage.writer.cooled&limit=100")"
	started="$(curl -fsS --max-time 5 "$events_base/api/v1/events?type=storage.recovery.started&limit=100")"
	echo "$hot" | jq -e --arg root "$root" 'any(.. | scalars; . == $root)' >/dev/null
	echo "$cooled" | jq -e --arg root "$root" 'any(.. | scalars; . == $root)' >/dev/null
	echo "$started" | jq -e --arg run_id "$run_id" 'any(.. | scalars; . == $run_id)' >/dev/null
}
