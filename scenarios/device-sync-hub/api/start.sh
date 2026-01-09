#!/bin/bash
# Wrapper script to start the Go API server
# This works around a lifecycle runner issue

# Ensure the authenticator scenario is running so we can resolve live ports
if command -v vrooli >/dev/null 2>&1; then
  vrooli scenario start scenario-authenticator >/dev/null 2>&1 || true

  # Populate SCENARIO_AUTHENTICATOR_API_PORT and UI port if not provided
  if [[ -z "${SCENARIO_AUTHENTICATOR_API_PORT:-}" ]] || [[ -z "${SCENARIO_AUTHENTICATOR_UI_PORT:-}" ]]; then
    while IFS='=' read -r key value; do
      case "$key" in
        API_PORT)
          export SCENARIO_AUTHENTICATOR_API_PORT="$value"
          ;;
        UI_PORT)
          export SCENARIO_AUTHENTICATOR_UI_PORT="$value"
          ;;
      esac
    done < <(vrooli scenario port scenario-authenticator 2>/dev/null)
  fi
fi

# Use environment variables with defaults
export API_PORT="${API_PORT:-17865}"
export UI_PORT="${UI_PORT:-37865}"

if [[ -z "${AUTH_SERVICE_URL:-}" ]]; then
  if [[ -n "${SCENARIO_AUTHENTICATOR_API_PORT:-}" ]]; then
    export AUTH_SERVICE_URL="http://localhost:${SCENARIO_AUTHENTICATOR_API_PORT}"
  else
    export AUTH_SERVICE_URL="http://localhost:15785"
  fi
fi

if [[ -z "${AUTH_PORT:-}" && -n "${SCENARIO_AUTHENTICATOR_API_PORT:-}" ]]; then
  export AUTH_PORT="${SCENARIO_AUTHENTICATOR_API_PORT}"
elif [[ -z "${AUTH_PORT:-}" ]]; then
  export AUTH_PORT="15785"
fi

if [[ -z "${AUTH_UI_URL:-}" && -n "${SCENARIO_AUTHENTICATOR_UI_PORT:-}" ]]; then
  export AUTH_UI_URL="http://localhost:${SCENARIO_AUTHENTICATOR_UI_PORT}"
fi

export STORAGE_PATH="${STORAGE_PATH:-${PWD}/data/files}"
export POSTGRES_URL="${POSTGRES_URL:-postgresql://vrooli:lUq9qvemypKpuEeXCV6Vnxak1@localhost:5433/vrooli?sslmode=disable}"
export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"
export MAX_FILE_SIZE="${MAX_FILE_SIZE:-10485760}"
export DEFAULT_EXPIRY_HOURS="${DEFAULT_EXPIRY_HOURS:-24}"
export THUMBNAIL_SIZE="${THUMBNAIL_SIZE:-200}"

# Change to the API directory
cd "$(dirname "$0")"

# Run the Go API server
exec ./device-sync-hub-api
