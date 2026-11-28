#!/usr/bin/env sh
set -eu

bin_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
port=${WORKSPACE_SANDBOX_STANDALONE_PORT:-18765}
api_base="http://127.0.0.1:${port}"
state_dir=${XDG_STATE_HOME:-"$HOME/.local/state"}/workspace-sandbox
mkdir -p "$state_dir"
pid_file="$state_dir/api.pid"

if [ -f "$pid_file" ]; then
	pid=$(cat "$pid_file" 2>/dev/null || true)
	if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
		:
	else
		rm -f "$pid_file"
	fi
fi

if [ ! -f "$pid_file" ]; then
	API_PORT="$port" \
	WORKSPACE_SANDBOX_STANDALONE=1 \
	PROJECT_ROOT="${WORKSPACE_SANDBOX_PROJECT_ROOT:-$PWD}" \
	"$bin_dir/workspace-sandbox-api" >"$state_dir/api.log" 2>&1 &
	api_pid=$!
	printf '%s\n' "$api_pid" >"$pid_file"
	ready=0
	for _ in $(seq 1 10); do
		if kill -0 "$api_pid" 2>/dev/null; then
			ready=1
			sleep 0.2
			# Give the HTTP listener time to bind after the process is alive.
			if [ "$ready" -eq 1 ]; then break; fi
		fi
		sleep 0.1
	done
	if [ "$ready" -ne 1 ]; then
		cat "$state_dir/api.log" >&2 || true
		exit 1
	fi
fi

WORKSPACE_SANDBOX_API_BASE="$api_base" exec "$bin_dir/workspace-sandbox-cli" "$@"
