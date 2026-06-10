#!/usr/bin/env bash
# live-validate.sh — attended live-validation runbook for web-search.
#
# Exercises the parts of the requirements registry that genuinely cannot be
# validated hermetically: a real L3 agent-manager research run (REQ-P1-002),
# capture-flag-free auto-capture of findings (REQ-P1-004), and the
# served-from-learnings latency SLO (REQ-P0-004's 500ms-p95 half).
#
# This script is NOT wired into any test phase. Run it deliberately
# (monthly — manual evidence expires after 30 days) via:
#
#   cd scenarios/web-search && make validate-live
#   # or: scripts/live-validate.sh ["custom research query"]
#
# On success it logs manual-validation evidence for each covered requirement
# via `test-genie requirements manual-log` (30-day TTL). On any failed
# assertion it logs NOTHING and exits non-zero — after the TTL lapses the
# requirements regress honestly until the runbook is re-run; that is the
# designed behavior, not a bug.
#
# See docs/operations/LIVE_VALIDATION.md for the full runbook.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
SCENARIO_NAME="web-search"

# --- Tunables (env-overridable) ---------------------------------------------
# Maximum seconds to wait for the L3 run to reach a terminal state.
L3_BUDGET_SECS="${LIVE_VALIDATE_L3_BUDGET:-900}"
# Poll interval while the L3 run is in flight.
POLL_SECS="${LIVE_VALIDATE_POLL_SECS:-15}"
# Latency SLO for a warm served-from-learnings query (REQ-P0-004), in ms.
LEARNINGS_P95_MS="${LIVE_VALIDATE_LEARNINGS_P95_MS:-500}"
# Ollama base URL for the preflight reachability check.
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"

API_LOG="$HOME/.vrooli/logs/scenarios/web-search/vrooli.develop.web-search.start-api.log"

# Rotating research queries: pick by day-of-month so monthly runs vary and a
# stale cached answer can't satisfy the assertion. $1 overrides.
QUERIES=(
  "What are the headline features of the latest stable Go release?"
  "What is the latest stable Rust release and its headline features?"
  "What are the most recent changes in the latest PostgreSQL major release?"
  "What is the newest LTS release of Node.js and what changed in it?"
)
QUERY="${1:-${QUERIES[$(( $(date +%-d) % ${#QUERIES[@]} ))]}}"

fail() {
  echo ""
  echo "✗ LIVE VALIDATION FAILED: $*" >&2
  echo "  No manual-validation evidence was logged." >&2
  exit 1
}
step() { echo ""; echo "── $* ──"; }

require_bin() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool '$1' not on PATH"
}
require_bin jq
require_bin curl
require_bin web-search
require_bin test-genie

# --- (a) Preflight ------------------------------------------------------------
step "Preflight"

API_PORT="$(vrooli scenario port web-search API_PORT 2>/dev/null)" || fail "web-search not running (no API_PORT claim)"
HEALTH="$(curl -fsS --max-time 5 "http://localhost:${API_PORT}/health" 2>/dev/null)" || fail "web-search API /health unreachable on :${API_PORT}"
[[ "$(jq -r '.status' <<<"$HEALTH")" == "healthy" ]] || fail "web-search API reports unhealthy: $HEALTH"
echo "✓ web-search API healthy on :${API_PORT}"

if command -v resource-searxng >/dev/null 2>&1; then
  ENGINES="$(resource-searxng engine-health --json 2>/dev/null | jq '.responsive_engines | length')" || fail "searxng engine-health probe failed"
  (( ENGINES >= 2 )) || fail "searxng has only ${ENGINES} responsive engine(s); need >=2 for a trustworthy live run"
  echo "✓ searxng healthy (${ENGINES} responsive engines)"
else
  fail "resource-searxng CLI not installed (needed for engine-health preflight)"
fi

AM_PORT="$(vrooli scenario port agent-manager API_PORT 2>/dev/null)" || fail "agent-manager not running (L3 runs require it)"
curl -fsS --max-time 5 "http://localhost:${AM_PORT}/health" >/dev/null 2>&1 || fail "agent-manager /health unreachable on :${AM_PORT}"
echo "✓ agent-manager healthy on :${AM_PORT}"

if BAS_PORT="$(vrooli scenario port browser-automation-studio API_PORT 2>/dev/null)"; then
  echo "✓ browser-automation-studio up on :${BAS_PORT}"
else
  echo "⚠ browser-automation-studio not running — L2 fetch falls back to direct HTTP; continuing"
fi

curl -fsS --max-time 5 "${OLLAMA_URL}/api/tags" >/dev/null 2>&1 || fail "ollama unreachable at ${OLLAMA_URL} (synthesis + distillation need it)"
echo "✓ ollama reachable at ${OLLAMA_URL}"

[[ -f "$API_LOG" ]] || fail "API access log not found at $API_LOG (log-delta assertions need it)"

# --- (b) Served-from-learnings latency (REQ-P0-004, 500ms-p95 half) ----------
step "Learnings-query latency (warm, p95 over 5 calls, SLO ${LEARNINGS_P95_MS}ms)"

SEARCH_URL="http://localhost:${API_PORT}/vrooli.web_search.v1.findings.FindingsService/SearchFindings"
SEARCH_BODY='{"query":"latest stable release features","limit":5}'

live_lines_before="$(grep -c 'LiveSearchService/Search' "$API_LOG" || true)"

# One warm-up call (model/embedder load is exempt from the warm SLO).
curl -fsS --max-time 30 -X POST -H 'Content-Type: application/json' -d "$SEARCH_BODY" "$SEARCH_URL" >/dev/null \
  || fail "warm-up SearchFindings call failed"

max_ms=0
for i in 1 2 3 4 5; do
  t="$(curl -fsS --max-time 10 -o /dev/null -w '%{time_total}' -X POST -H 'Content-Type: application/json' -d "$SEARCH_BODY" "$SEARCH_URL")" \
    || fail "timed SearchFindings call #${i} failed"
  ms="$(awk -v t="$t" 'BEGIN{printf "%d", t*1000}')"
  echo "  call ${i}: ${ms}ms"
  (( ms > max_ms )) && max_ms=$ms
done
# With N=5, p95 == max.
(( max_ms <= LEARNINGS_P95_MS )) || fail "learnings query p95 ${max_ms}ms exceeds ${LEARNINGS_P95_MS}ms SLO"

live_lines_after="$(grep -c 'LiveSearchService/Search' "$API_LOG" || true)"
(( live_lines_after == live_lines_before )) || fail "learnings queries triggered $(( live_lines_after - live_lines_before )) live-web call(s); expected zero"
echo "✓ p95 ${max_ms}ms <= ${LEARNINGS_P95_MS}ms with zero LiveSearchService calls"
LEARNINGS_P95_RESULT="$max_ms"

# --- (c) Kick an L3 research run ----------------------------------------------
step "L3 research run: \"$QUERY\""

START_ISO="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
api_log_lines_before="$(wc -l < "$API_LOG")"
l3_findings_before="$(web-search findings list --json --limit 500 | jq '[.findings[]? | select(.source=="FINDING_SOURCE_L3")] | length')"

RUN_JSON="$(web-search research l3 "$QUERY" --json)" || fail "research l3 kick-off failed"
RUN_ID="$(jq -r '.run_id // .runId // empty' <<<"$RUN_JSON")"
[[ -n "$RUN_ID" ]] || fail "research l3 returned no run id: $RUN_JSON"
echo "  run id: $RUN_ID (budget ${L3_BUDGET_SECS}s)"

STATUS=""
SUMMARY=""
deadline=$(( $(date +%s) + L3_BUDGET_SECS ))
while (( $(date +%s) < deadline )); do
  sleep "$POLL_SECS"
  STATUS_JSON="$(web-search research status "$RUN_ID" --json)" || fail "research status poll failed for run $RUN_ID"
  STATUS="$(jq -r '.status // empty' <<<"$STATUS_JSON")"
  case "$STATUS" in
    complete) SUMMARY="$(jq -r '.summary // empty' <<<"$STATUS_JSON")"; break ;;
    failed|cancelled) fail "L3 run $RUN_ID terminated as '$STATUS': $(jq -r '.error_msg // .errorMsg // "no error message"' <<<"$STATUS_JSON")" ;;
    *) echo "  …${STATUS:-pending}" ;;
  esac
done
[[ "$STATUS" == "complete" ]] || fail "L3 run $RUN_ID did not complete within ${L3_BUDGET_SECS}s (last status: ${STATUS:-unknown})"
echo "✓ run $RUN_ID complete"

# --- (d) Post-run assertions ---------------------------------------------------
step "Post-run assertions"

# >=1 L2/livesearch tool invocation during the run window (API log delta).
api_log_delta="$(tail -n "+$(( api_log_lines_before + 1 ))" "$API_LOG")"
tool_calls="$(grep -cE 'ResearchService/RunL2|LiveSearchService/Search' <<<"$api_log_delta" || true)"
(( tool_calls >= 1 )) || fail "no L2/livesearch tool invocations observed in the run window"
echo "✓ ${tool_calls} L2/livesearch tool invocation(s) during the run window"

# Re-search-on-gap proxy (REQ-P1-002): a gap-iterating run invokes the search
# tools more than once. Attended caveat: concurrent API traffic could inflate
# this — run on a quiet system.
if (( tool_calls >= 2 )); then
  RESEARCHED="yes (${tool_calls} tool calls)"
  echo "✓ agent searched more than once (gap iteration observed)"
else
  RESEARCHED="not observed (single tool call)"
  echo "⚠ only one tool invocation — gap-iteration evidence weak; REQ-P1-002's re-search entry will note this"
fi

# >=1 NEW finding with source=L3, created during the window, NO capture flag
# (the l3 CLI verb has no capture flag at all — auto-capture is the default).
NEW_FINDINGS="$(web-search findings list --json --limit 500 | jq --arg ts "$START_ISO" \
  '[.findings[]? | select(.source=="FINDING_SOURCE_L3" and .created_at >= $ts)] | length')"
l3_findings_after="$(web-search findings list --json --limit 500 | jq '[.findings[]? | select(.source=="FINDING_SOURCE_L3")] | length')"
(( NEW_FINDINGS >= 1 )) || fail "no new L3-sourced findings after the run (before=${l3_findings_before}, after=${l3_findings_after})"
echo "✓ ${NEW_FINDINGS} new L3-sourced finding(s) auto-captured (no capture flag passed)"

# Brief non-empty.
[[ -n "$SUMMARY" ]] || fail "run completed but the brief/summary is empty"
echo "✓ brief non-empty ($(wc -c <<<"$SUMMARY") bytes)"

# Index freshness (capture-kick): a finding captured by the run must become
# semantically searchable within seconds of run completion, not after the
# periodic sync interval (WEB_SEARCH_SYNC_INTERVAL is the repair cadence only).
NEW_IDS="$(web-search findings list --json --limit 500 | jq -r --arg ts "$START_ISO" \
  '.findings[]? | select(.source=="FINDING_SOURCE_L3" and .created_at >= $ts) | .id')"
fresh_ok=""
for _ in $(seq 1 6); do
  HITS="$(web-search findings search "$QUERY" --json 2>/dev/null | jq -r '.hits[]?.finding.id // empty')" || HITS=""
  while IFS= read -r id; do
    [[ -n "$id" ]] && grep -qx "$id" <<<"$HITS" && fresh_ok="$id" && break 2
  done <<<"$NEW_IDS"
  sleep 5
done
if [[ -n "$fresh_ok" ]]; then
  echo "✓ fresh finding ${fresh_ok} semantically searchable within ~30s of completion (capture-kick)"
else
  fail "no fresh finding became semantically searchable within 30s of run completion (capture-kick regression?)"
fi

# --- (e) Log manual-validation evidence ---------------------------------------
step "Logging manual-validation evidence (30-day TTL)"

ARTIFACT_DIR="coverage/manual-validations/artifacts"
mkdir -p "${SCENARIO_DIR}/${ARTIFACT_DIR}"
ARTIFACT="${ARTIFACT_DIR}/live-validate-$(date -u +%Y%m%d-%H%M%S).json"
jq -n --arg run_id "$RUN_ID" --arg query "$QUERY" --arg started "$START_ISO" \
      --argjson tool_calls "$tool_calls" --argjson new_findings "$NEW_FINDINGS" \
      --argjson p95_ms "$LEARNINGS_P95_RESULT" \
      '{run_id:$run_id, query:$query, started_at:$started, tool_calls:$tool_calls,
        new_l3_findings:$new_findings, learnings_query_p95_ms:$p95_ms}' \
  > "${SCENARIO_DIR}/${ARTIFACT}"

log_evidence() {
  local req="$1" notes="$2"
  test-genie requirements manual-log \
    --dir "$SCENARIO_DIR" --scenario "$SCENARIO_NAME" \
    --requirement "$req" --status passed --expires-in 30 \
    --artifact "$ARTIFACT" --notes "$notes" \
    || fail "manual-log failed for $req"
  echo "✓ evidence logged for $req"
}

log_evidence "REQ-P1-002" "live-validate.sh: L3 run ${RUN_ID} for '${QUERY}' completed within budget; ${tool_calls} L2/livesearch tool invocations; re-search-on-gap: ${RESEARCHED}; brief non-empty."
log_evidence "REQ-P1-004" "live-validate.sh: L3 run ${RUN_ID} auto-captured ${NEW_FINDINGS} finding(s) with source=L3 and no capture flag (the l3 verb exposes none)."
log_evidence "REQ-P0-004" "live-validate.sh: default learnings query (SearchFindings) warm p95 ${LEARNINGS_P95_RESULT}ms <= ${LEARNINGS_P95_MS}ms with zero LiveSearchService calls (log-delta). Zero-external-calls half also pinned hermetically by search-hub router tests."

echo ""
echo "✓ LIVE VALIDATION PASSED — evidence valid for 30 days (re-run monthly)."
