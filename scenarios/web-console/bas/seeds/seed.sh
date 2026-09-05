#!/usr/bin/env bash
# Seed script for web-console BAS (Browser Automation Suite)
# Creates default session data for BAS tests and workflows.
set -uo pipefail

RESOLVED_API_PORT="$(vrooli scenario port web-console API_PORT 2>/dev/null || true)"
if [[ -n "${RESOLVED_API_PORT}" ]]; then
  API_PORT="${RESOLVED_API_PORT}"
else
  API_PORT="${API_PORT:-17086}"
fi
RESOLVED_UI_PORT="$(vrooli scenario port web-console UI_PORT 2>/dev/null || true)"
if [[ -n "${RESOLVED_UI_PORT}" ]]; then
  UI_PORT="${RESOLVED_UI_PORT}"
else
  UI_PORT="${UI_PORT:-36233}"
fi
API_BASE="http://127.0.0.1:${API_PORT}/api/v1"
# Sessions moved to Connect-RPC; the old REST routes 404.
SESSIONS_RPC="http://127.0.0.1:${API_PORT}/vrooli.web_console.v1.sessions.SessionsService"
HEALTH_URL="http://127.0.0.1:${API_PORT}/health"
UI_PROXY_HEALTH_URL="http://127.0.0.1:${UI_PORT}/api/v1/health"

wait_for_api() {
  local attempts=10
  local delay_s=1
  local i
  local api_ok=0
  local proxy_ok=0
  for i in $(seq 1 "${attempts}"); do
    if [[ "${api_ok}" -eq 0 ]] && curl -sf --connect-timeout 1 --max-time 1 "${HEALTH_URL}" >/dev/null; then
      api_ok=1
    fi
    if [[ "${proxy_ok}" -eq 0 ]] && curl -sf --connect-timeout 1 --max-time 1 "${UI_PROXY_HEALTH_URL}" >/dev/null; then
      proxy_ok=1
    fi
    if [[ "${api_ok}" -eq 1 && "${proxy_ok}" -eq 1 ]]; then
      return 0
    fi
    sleep "${delay_s}"
  done
  if [[ "${api_ok}" -ne 1 ]]; then
    echo "[seed] API did not become healthy at ${HEALTH_URL}" >&2
  fi
  if [[ "${proxy_ok}" -ne 1 ]]; then
    echo "[seed] UI proxy did not become healthy at ${UI_PROXY_HEALTH_URL}" >&2
  fi
  return 1
}

create_seed_session() {
  local attempts=5
  local delay_s=1
  local i
  for i in $(seq 1 "${attempts}"); do
    if curl -sf --connect-timeout 1 --max-time 2 -X POST "${SESSIONS_RPC}/Create" \
      -H "Content-Type: application/json" \
      -d '{"shell":"/bin/bash","cols":120,"rows":40}' \
      -o /dev/null; then
      return 0
    fi
    sleep "${delay_s}"
  done
  echo "[seed] Failed to create seed session via ${API_BASE}/sessions" >&2
  return 1
}

echo "[seed] Waiting for API readiness (direct + UI proxy)..."
if ! wait_for_api; then
  echo "[seed] Warning: API readiness gate did not pass in time; continuing without pre-seeded session." >&2
  exit 0
fi
echo "[seed] Creating default terminal session..."
if ! create_seed_session; then
  echo "[seed] Warning: failed to create optional seed session; continuing." >&2
  exit 0
fi

# ---------------------------------------------------------------------------
# Message navigator seed — provisions one deterministic conversation so the
# message-navigator BAS case has user / assistant / file-reference landmarks to
# search, filter, and jump to. Conversation events are only writable through the
# token-authenticated agent hooks, so we read the hook token from scenario state
# (host-readable) and post a user prompt + an assistant reply. A Stop-hook
# assistant event also flips the session's agent_type to "claude", which is what
# enables the Messages view toggle in the UI. Entirely best-effort: a missing
# token or endpoint must never fail the seed.
resolve_hook_token() {
  local candidates=(
    "${XDG_STATE_HOME:-$HOME/.local/state}/vrooli/web-console/hook-token.txt"
    "$HOME/.vrooli/state/vrooli/web-console/hook-token.txt"
  )
  local path
  for path in "${candidates[@]}"; do
    if [[ -f "${path}" ]]; then
      tr -d '[:space:]' < "${path}"
      return 0
    fi
  done
  return 1
}

seed_message_navigator_conversation() {
  local token
  token="$(resolve_hook_token || true)"
  if [[ -z "${token}" ]]; then
    echo "[seed] hook token not found; skipping message-navigator conversation seed." >&2
    return 0
  fi

  local session_id
  session_id="$(curl -sf --connect-timeout 1 --max-time 2 -X POST "${SESSIONS_RPC}/Create" \
    -H "Content-Type: application/json" \
    -d '{"shell":"/bin/bash","cols":120,"rows":40}' \
    | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  if [[ -z "${session_id}" ]]; then
    echo "[seed] could not create message-navigator session; skipping." >&2
    return 0
  fi

  # User prompt — a unique marker keeps the navigator search deterministic.
  curl -sf --connect-timeout 1 --max-time 2 -X POST "${API_BASE}/hooks/prompt-submit" \
    -H "Content-Type: application/json" -H "X-Hook-Token: ${token}" \
    -d "{\"userPrompt\":\"NAVIGATOR_SEED deploy the release pipeline\",\"webConsoleSessionId\":\"${session_id}\"}" \
    -o /dev/null || true

  # Assistant reply (also marks the session agent_type=claude → Messages view).
  curl -sf --connect-timeout 1 --max-time 2 -X POST "${API_BASE}/hooks/stop" \
    -H "Content-Type: application/json" -H "X-Hook-Token: ${token}" \
    -d "{\"last_assistant_message\":\"NAVIGATOR_SEED deploying now — see src/pipeline.ts for the steps.\",\"session_id\":\"seed-agent-uuid\",\"web_console_session_id\":\"${session_id}\"}" \
    -o /dev/null || true

  echo "[seed] message-navigator conversation seeded for session ${session_id}."
}

seed_message_navigator_conversation || true

# ---------------------------------------------------------------------------
# Snippet workflow seeds — fixed ids make this idempotent across cases while
# still exercising the public CLI contract required by the snippet scenarios.
seed_snippets() {
  local snippet_seed_dir
  snippet_seed_dir="$(mktemp -d)" || return 0
  trap 'rm -rf "${snippet_seed_dir}"' RETURN

  printf '%s\n' '{"id":"ba500000-0000-4000-8000-000000000001","name":"BAS Plain Snippet","body":"BAS_PLAIN_BODY ready for insertion","color":"#38bdf8","pinned":true,"sort_order":10}' > "${snippet_seed_dir}/plain.json"
  printf '%s\n' '{"id":"ba500000-0000-4000-8000-000000000002","name":"BAS Variable Snippet","body":"Investigate {{topic}} and preserve {{missing}}","color":"#a78bfa","pinned":true,"sort_order":20}' > "${snippet_seed_dir}/variables.json"

  web-console snippet upsert --body-file "${snippet_seed_dir}/plain.json" >/dev/null || {
    echo "[seed] Warning: failed to seed the plain snippet through the CLI." >&2
  }
  web-console snippet upsert --body-file "${snippet_seed_dir}/variables.json" >/dev/null || {
    echo "[seed] Warning: failed to seed the variable snippet through the CLI." >&2
  }
}

seed_snippets || true

# ---------------------------------------------------------------------------
# File-preview seed — writes deterministic fixture files (markdown + SVG) to a
# stable host path and posts an assistant message whose markdown links point at
# them by ABSOLUTE path (so resolution never depends on the session cwd). The
# file-preview-drilldown BAS case clicks those links and asserts the preview
# viewer + the per-kind renderer. Shares the same token/hook seeding constraint
# as the message-navigator seed above; entirely best-effort.
seed_file_preview_conversation() {
  local token
  token="$(resolve_hook_token || true)"
  if [[ -z "${token}" ]]; then
    echo "[seed] hook token not found; skipping file-preview conversation seed." >&2
    return 0
  fi

  local fixture_dir="${TMPDIR:-/tmp}/web-console-bas-fixtures"
  mkdir -p "${fixture_dir}" || return 0
  local md_path="${fixture_dir}/preview-fixture.md"
  local svg_path="${fixture_dir}/preview-fixture.svg"
  printf '# FILEPREVIEW_SEED fixture\n\nHello from the markdown preview fixture.\n' > "${md_path}" || return 0
  printf '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4"/></svg>\n' > "${svg_path}" || return 0

  # Directory fixture for the directory-drilldown case: a directory holding one
  # child directory and one markdown file, so the case can walk down two levels
  # and back. The hidden entry proves the default listing filters dotfiles.
  local dir_path="${fixture_dir}/preview-dir"
  rm -rf "${dir_path}"
  mkdir -p "${dir_path}/nested" || return 0
  printf '# nested fixture\n\nInside the nested directory.\n' > "${dir_path}/nested/inner.md" || return 0
  printf 'top level entry\n' > "${dir_path}/top.md" || return 0
  printf 'SECRET=should-not-show-by-default\n' > "${dir_path}/.hidden-env" || return 0

  local session_id
  session_id="$(curl -sf --connect-timeout 1 --max-time 2 -X POST "${SESSIONS_RPC}/Create" \
    -H "Content-Type: application/json" \
    -d '{"shell":"/bin/bash","cols":120,"rows":40}' \
    | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  if [[ -z "${session_id}" ]]; then
    echo "[seed] could not create file-preview session; skipping." >&2
    return 0
  fi

  curl -sf --connect-timeout 1 --max-time 2 -X POST "${API_BASE}/hooks/prompt-submit" \
    -H "Content-Type: application/json" -H "X-Hook-Token: ${token}" \
    -d "{\"userPrompt\":\"FILEPREVIEW_SEED open the generated artifacts\",\"webConsoleSessionId\":\"${session_id}\"}" \
    -o /dev/null || true

  # Assistant reply embeds absolute-path markdown links to the fixtures.
  local reply="FILEPREVIEW_SEED here are the artifacts: [markdown fixture](${md_path}) and [svg fixture](${svg_path}) and [directory fixture](${dir_path})."
  curl -sf --connect-timeout 1 --max-time 2 -X POST "${API_BASE}/hooks/stop" \
    -H "Content-Type: application/json" -H "X-Hook-Token: ${token}" \
    -d "{\"last_assistant_message\":\"${reply}\",\"session_id\":\"seed-filepreview-uuid\",\"web_console_session_id\":\"${session_id}\"}" \
    -o /dev/null || true

  echo "[seed] file-preview conversation seeded for session ${session_id} (fixtures in ${fixture_dir})."
}

seed_file_preview_conversation || true

echo "[seed] Seeding complete."
