#!/usr/bin/env bash
# Scenario-owned seed for routed tests and isolated presentation instances.
# The caller supplies the exact target and its hook token; never discover live.
set -euo pipefail

TARGET_API_BASE="${TEST_GENIE_TARGET_API_BASE:-${BAS_TARGET_API_BASE:-}}"
TARGET_API_BASE="${TARGET_API_BASE%/}"
if [[ ! "${TARGET_API_BASE}" =~ ^http://(127\.0\.0\.1|localhost|\[::1\]):[0-9]+$ ]]; then
  echo '[seed] seed_target_invalid: supply a loopback HTTP URL with an explicit port' >&2
  exit 2
fi
LIVE_API_PORT="${TEST_GENIE_LIVE_WEB_CONSOLE_API_PORT:-${BAS_LIVE_WEB_CONSOLE_API_PORT:-}}"
if [[ -n "${LIVE_API_PORT}" && "${TARGET_API_BASE##*:}" == "${LIVE_API_PORT}" ]]; then
  echo '[seed] seed_target_live_instance_refused' >&2
  exit 3
fi
TOKEN_FILE="${BAS_WEB_CONSOLE_HOOK_TOKEN_PATH:-${WC_HOOK_TOKEN_PATH:-}}"
# Test Genie's routed pool uses the serving primary API's existing webhook
# authentication. Resolve that path before the test storage overlay; a separate
# presentation instance always supplies its own path explicitly.
if [[ -z "${TOKEN_FILE}" && "${TEST_GENIE_SEEDS:-}" == 1 && -n "${PLAYBOOKS_SQLITE_DSN:-}" ]]; then
  TOKEN_FILE="$(env -u VROOLI_STORAGE_ROOT -u VROOLI_STORAGE_NAMESPACE -u VROOLI_VARIANT -u VROOLI_SCENARIO web-console hooks token-path)"
fi
if [[ -z "${TOKEN_FILE}" || ! -f "${TOKEN_FILE}" ]]; then
  echo '[seed] seed_hook_token_missing: supply the target-owned BAS_WEB_CONSOLE_HOOK_TOKEN_PATH; live token discovery is disabled' >&2
  exit 4
fi
HOOK_TOKEN="$(tr -d '\r\n' < "${TOKEN_FILE}")"
if [[ -z "${HOOK_TOKEN//[[:space:]]/}" ]]; then
  echo '[seed] seed_hook_token_empty' >&2
  exit 4
fi

# A routed seed must name the caller-owned lease. Reattach idempotently and
# verify BOTH storage legs before creating any sessions or linked files.
routed=false
if [[ "${BAS_ROUTED_TEST_MODE:-}" == 1 || -n "${PLAYBOOKS_SQLITE_DSN:-}" ]]; then
  routed=true
  lease_id="${TEST_GENIE_ROUTING_LEASE_ID:-${BAS_ROUTING_LEASE_ID:-}}"
  if [[ -z "${lease_id}" || -z "${PLAYBOOKS_SQLITE_DSN:-}" ]]; then
    echo '[seed] seed_routing_context_missing: routed mode requires lease ID and DSN' >&2
    exit 4
  fi
  routing="$(curl -fsS --connect-timeout 2 --max-time 15 -X POST \
    "${TARGET_API_BASE}/vrooli.dev_routing.v1.routing.RoutingService/InstallTestPool" \
    -H 'Content-Type: application/json' \
    -d "$(jq -cn --arg lease "${lease_id}" --arg dsn "${PLAYBOOKS_SQLITE_DSN}" \
      '{leaseId:$lease,dsn:$dsn,leaseTtlMs:"180000"}')")"
  jq -e --arg lease "${lease_id}" '.activeLeaseId == $lease and .fileRootsInstalled == true' <<< "${routing}" >/dev/null
fi

# A fresh per-invocation directory prevents concurrent runs from replacing each
# other's linked files. Retain it with the receipt until caller-owned cleanup.
seed_parent="${TMPDIR:-/tmp}"
route_headers=()
if [[ "${routed}" == true ]]; then
  route_headers=(-H 'X-Vrooli-Test-Mode: 1')
  if [[ -n "${VROOLI_STORAGE_ROOT:-}" ]]; then
    seed_parent="${VROOLI_STORAGE_ROOT}/fixtures"
    mkdir -p "${seed_parent}"
  fi
fi
fixture_dir="$(mktemp -d "${seed_parent}/web-console-bas-fixtures.XXXXXX")"
created_sessions=()
complete=false
post() {
  curl -fsS --connect-timeout 2 --max-time 15 -X POST "${TARGET_API_BASE}$1" \
    -H 'Content-Type: application/json' -H "X-Hook-Token: ${HOOK_TOKEN}" "${route_headers[@]}" -d "$2"
}
cleanup_failed_seed() {
  if [[ "${complete}" == true ]]; then return; fi
  local cleanup_failed=false
  for session_id in "${created_sessions[@]}"; do
    post '/vrooli.web_console.v1.sessions.SessionsService/Delete' \
      "$(jq -cn --arg id "${session_id}" '{id:$id,confirmation:("DELETE:"+$id)}')" >/dev/null || cleanup_failed=true
  done
  if [[ -f "${fixture_dir}/seed-receipt.json" ]]; then
    jq --argjson failed "${cleanup_failed}" '.status="failed" | .cleanup_failed=$failed' \
      "${fixture_dir}/seed-receipt.json" > "${fixture_dir}/receipt.tmp"
    mv "${fixture_dir}/receipt.tmp" "${fixture_dir}/seed-receipt.json"
  fi
  echo "[seed] seed_failed: retained partial evidence at ${fixture_dir}" >&2
}
trap cleanup_failed_seed EXIT

curl -fsS --connect-timeout 2 --max-time 10 "${TARGET_API_BASE}/health" >/dev/null

create_session() {
  local label="$1" response session_id
  response="$(post '/vrooli.web_console.v1.sessions.SessionsService/Create' \
    "$(jq -cn --arg label "${label}" '{shell:"/bin/sh",backend:"standard",cols:120,rows:40,displayLabel:$label}')")"
  session_id="$(jq -er '.session.id | select(type == "string" and length > 0)' <<< "${response}")"
  created_sessions+=("${session_id}")
  printf '%s\n' "${session_id}" >> "${fixture_dir}/sessions.txt"
  # Persist as each operation succeeds; cleanup can address only these IDs.
  jq -cn --arg target "${TARGET_API_BASE}" --arg files "${fixture_dir}" \
    --slurpfile ids <(jq -R . "${fixture_dir}/sessions.txt") \
    '{version:1,target_api:$target,fixture_dir:$files,session_ids:$ids,status:"seeding",proof_scope:"authored messages through real API; no agent execution"}' \
    > "${fixture_dir}/seed-receipt.json"
  CREATED_SESSION_ID="${session_id}"
}
user_message() {
  post '/api/v1/hooks/prompt-submit' \
    "$(jq -cn --arg id "$1" --arg text "$2" '{webConsoleSessionId:$id,userPrompt:$text}')" >/dev/null
}
assistant_message() {
  post '/api/v1/hooks/stop' \
    "$(jq -cn --arg id "$1" --arg text "$2" '{web_console_session_id:$id,session_id:("bas-"+$id),last_assistant_message:$text}')" >/dev/null
}
name_pane() {
  post '/vrooli.web_console.v1.workspace.WorkspaceService/UpdatePane' \
    "$(jq -cn --arg id "$1" --arg name "$2" '{sessionId:$id,name:$name,hasName:true,supportsMessagesView:true,hasSupportsMessagesView:true}')" >/dev/null
}

create_session 'Northline — release review'
navigator_id="${CREATED_SESSION_ID}"
user_message "${navigator_id}" 'NAVIGATOR_SEED deploy the release pipeline'
assistant_message "${navigator_id}" $'NAVIGATOR_SEED deploying now — see src/pipeline.ts for the steps.\n\n| Check | Result |\n| --- | --- |\n| Unit tests | Passed |\n| Staging | Ready for review |\n\n```mermaid\nflowchart LR\n  Build --> Tests --> Review\n```'
assistant_message "${navigator_id}" 'Here is where the release stands. The build finished and every unit test passed, so the next step is the staging deploy. I will run the migration first, then roll the new version to one instance, watch its health for a few minutes, and only then move the rest of the fleet across. If anything looks wrong I will stop, roll back to the previous version, and write down what I saw so we can decide together what to change before trying again. Nothing in this plan touches production until staging has been quiet for at least an hour.'
name_pane "${navigator_id}" 'Northline — release review'

# Seed through the same typed service used by the CLI, retaining the test-mode
# header instead of losing it across a second client invocation.
for row in \
  '{"id":"ba500000-0000-4000-8000-000000000001","name":"BAS Plain Snippet","body":"BAS_PLAIN_BODY ready for insertion","color":"#38bdf8","pinned":true,"hasPinned":true,"sort_order":10}' \
  '{"id":"ba500000-0000-4000-8000-000000000002","name":"BAS Variable Snippet","body":"Investigate {{topic}} and preserve {{missing}}","color":"#a78bfa","pinned":true,"hasPinned":true,"sort_order":20}'; do
  printf '%s\n' "${row}" > "${fixture_dir}/snippet.json"
  post '/vrooli.web_console.v1.snippets.SnippetsService/UpsertSnippet' "${row}" >/dev/null
done

mkdir -p "${fixture_dir}/preview-dir/nested"
printf '# FILEPREVIEW_SEED fixture\n\nHello from the markdown preview fixture.\n' > "${fixture_dir}/preview-fixture.md"
printf '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4"/></svg>\n' > "${fixture_dir}/preview-fixture.svg"
printf '# nested fixture\n\nInside the nested directory.\n' > "${fixture_dir}/preview-dir/nested/inner.md"
printf 'top level entry\n' > "${fixture_dir}/preview-dir/top.md"
printf 'SECRET=should-not-show-by-default\n' > "${fixture_dir}/preview-dir/.hidden-env"
create_session 'Northline — artifact review'
files_id="${CREATED_SESSION_ID}"
user_message "${files_id}" 'FILEPREVIEW_SEED open the generated artifacts'
assistant_message "${files_id}" "FILEPREVIEW_SEED here are the artifacts: [markdown fixture](${fixture_dir}/preview-fixture.md) and [svg fixture](${fixture_dir}/preview-fixture.svg) and [directory fixture](${fixture_dir}/preview-dir)."
name_pane "${files_id}" 'Northline — artifact review'

# Verify the stored result, not just webhook HTTP success. Retain both
# responses as the seed's backend evidence before any browser takes begin.
for pair in "${navigator_id}:NAVIGATOR_SEED" "${files_id}:FILEPREVIEW_SEED"; do
  session_id="${pair%%:*}"
  marker="${pair#*:}"
  post '/vrooli.web_console.v1.conversation.ConversationService/Get' \
    "$(jq -cn --arg id "${session_id}" '{sessionId:$id,limit:100}')" > "${fixture_dir}/${marker}.json"
  if ! jq -e --arg marker "${marker}" 'any(.events[]?; .text | contains($marker))' \
    "${fixture_dir}/${marker}.json" >/dev/null; then
    echo "[seed] seed_messages_missing: ${marker} was not persisted" >&2
    exit 7
  fi
done

# The maintained SQL migration is the single continuity fixture. Test Genie
# already applies it before this script; standalone routed calls may replay it.
db_target="${PLAYBOOKS_SQLITE_DSN:-${PLAYBOOKS_SQLITE_PATH:-}}"
if [[ -n "${db_target}" ]]; then
  sqlite3 -bail "${db_target}" < "$(dirname "${BASH_SOURCE[0]}")/migrations/sqlite/001-continuity-incident.sql"
fi
jq '.status="seeded"' "${fixture_dir}/seed-receipt.json" > "${fixture_dir}/receipt.tmp"
mv "${fixture_dir}/receipt.tmp" "${fixture_dir}/seed-receipt.json"
complete=true
printf '[seed] Seeding complete. Receipt: %s/seed-receipt.json\n' "${fixture_dir}"
