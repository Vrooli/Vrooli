#!/usr/bin/env bash
# CLI-only acceptance walkthrough for the investigation evidence spine.
set -euo pipefail

run_id="${1:-0a573109-4555-46b0-aec6-611c2fd98177}"
search_hub_run_id="${2:-71d0abf3-89c0-4da2-a9d2-270eac15f3fe}"
test_genie_run_id="${3:-42b12338-364c-4f9d-a39c-a9b1861de4ed}"
prompt_manager_run_id="${4:-149541d0-bdb2-4d06-83ad-da8a66e67d6f}"
episode_run_id="${5:-0a573109-4555-46b0-aec6-611c2fd98177}"
finding_run_id="${6:-1c254e20-7300-4e14-a6c2-720939cd6c0b}"
self_report_run_id="${7:-09822a7b-da48-415b-8d12-91d8d9e1fb18}"

require() { jq -e "$1" >/dev/null; }
require_verified_receipt() {
  local run="$1" target="$2"
  agent-manager run ledger "$run" --json | jq -e --arg target "$target" '[.calls[] | select(.target_scenario == $target and .verified == true and .outcome == "success")] | length > 0' >/dev/null
}

report="$(agent-manager run report "$run_id" --json)"
echo '1. Time attribution'; printf '%s' "$report" | require '.time_accounting.model_generating_ms > 0 and .time_accounting.tool_executing_ms > 0 and .time_accounting.idle_waiting_ms > 0 and .time_accounting.awaiting_human_ms > 0'
echo '2. Tool correctness'; agent-manager run invocation-facts "$run_id" --json | require '[.facts[] | select(.outcome == "failure" and .resultEventId != "" and .failureSignature != "")] | length > 0'
echo '3. Repetition'; agent-manager run episodes "$episode_run_id" --json | require '[.episodes[] | select(.evidence_event_ids | length >= 2) | select(.wall_clock_ms | tonumber > 0)] | length > 0'
echo '4. Self-reported friction'; agent-manager run messages-friction "$self_report_run_id" --json | require '[.spans[] | select(.text != "" and .rule_id != "")] | length > 0'
echo '5. Owned versus external tooling'; printf '%s' "$report" | require '.project_owned_tool_calls != null and .external_tool_calls > 0'
echo '6. Per-run cross-scenario evidence'; require_verified_receipt "$search_hub_run_id" search-hub; require_verified_receipt "$test_genie_run_id" test-genie; require_verified_receipt "$prompt_manager_run_id" prompt-manager
echo '7. Cohort recurrence'; agent-manager run episode-cohort --json | require '[.signals[] | select((.occurrences | tonumber) > 0 and (.distinct_runs | tonumber) > 0 and ((.summed_cost_ms // "0") | tonumber) > 0)] | length > 0'
echo '8. Applied-fix recurrence'; agent-manager findings list --json | require '[.findings[] | select(.investigationRunId == "'"$finding_run_id"'" and .beforeValue != null and .afterValue != null and (.effectiveness == "effective" or .effectiveness == "ineffective" or .effectiveness == "not_yet_measurable"))] | length > 0'; agent-manager measures finding-recurrence-rate --json | require '.validity.state == "available"'
echo 'Acceptance walkthrough passed.'
