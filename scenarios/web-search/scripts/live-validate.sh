#!/usr/bin/env bash
# Attended evidence collection. Completion never automatically proves answer quality.
set -euo pipefail
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
QUERY="${1:-What is the default TCP port for PostgreSQL? Verify the official documentation and capture one supported finding.}"
WAIT_SECONDS="${LIVE_VALIDATE_WAIT_SECONDS:-60}"
RUN_ID="${LIVE_VALIDATE_RUN_ID:-}"
KEY="${LIVE_VALIDATE_IDEMPOTENCY_KEY:-web-search-live-$(date -u +%Y%m%dT%H%M%SZ)}"
ARTIFACT_DIR="${SCENARIO_DIR}/coverage/manual-validations/artifacts/${KEY}"
[[ "$KEY" =~ ^[A-Za-z0-9_-]+$ ]] || { echo 'Idempotency key must be a safe artifact name' >&2; exit 1; }
[[ "$WAIT_SECONDS" =~ ^[0-9]+$ ]] && (( WAIT_SECONDS>=1 && WAIT_SECONDS<=90 )) || { echo 'Wait must be within 1..90 seconds' >&2; exit 1; }
for binary in jq curl web-search agent-manager; do command -v "$binary" >/dev/null; done
mkdir -p "$ARTIFACT_DIR"
API_PORT="$(vrooli scenario port web-search API_PORT)"
curl -fsS --max-time 5 "http://localhost:${API_PORT}/health" > "$ARTIFACT_DIR/health.json"
if [[ -z "$RUN_ID" ]]; then
  web-search research l3 "$QUERY" --idempotency-key "$KEY" --json > "$ARTIFACT_DIR/start.json"
  RUN_ID="$(jq -r '.runId // .run_id // empty' "$ARTIFACT_DIR/start.json")"
fi
[[ -n "$RUN_ID" ]] || { echo 'No execution identity returned' >&2; exit 1; }
printf '%s\n' "$RUN_ID" > "$ARTIFACT_DIR/execution-id.txt"
# One server-owned wait. A timeout preserves the execution and all evidence.
if ! web-search research wait "$RUN_ID" --timeout-seconds "$WAIT_SECONDS" --json > "$ARTIFACT_DIR/result.json"; then
  echo "Wait incomplete or execution failed. Evidence: $ARTIFACT_DIR" >&2
  echo "Reattach with LIVE_VALIDATE_RUN_ID=$RUN_ID LIVE_VALIDATE_IDEMPOTENCY_KEY=$KEY; do not start a replacement or poll." >&2
  exit 2
fi
agent-manager workflow trace "$RUN_ID" --json > "$ARTIFACT_DIR/trace.json"
agent-manager workflow execution-runs "$RUN_ID" --json > "$ARTIFACT_DIR/attempts.json"
jq -e '.status=="complete" and (.result.summary | length)>0 and (.result.status=="answered" or .result.status=="partial" or .result.status=="abstained")' "$ARTIFACT_DIR/result.json" >/dev/null
# Only IDs returned by this execution can contribute capture evidence.
jq -r '.result.finding_ids[]?' "$ARTIFACT_DIR/result.json" > "$ARTIFACT_DIR/finding-ids.txt"
while IFS= read -r finding_id; do
  [[ "$finding_id" =~ ^[A-Za-z0-9_-]+$ ]] || { echo 'Unsafe finding identity' >&2; exit 1; }
  web-search findings get "$finding_id" --json > "$ARTIFACT_DIR/finding-${finding_id}.json"
done < "$ARTIFACT_DIR/finding-ids.txt"
# Latency samples make no claim about unrelated traffic or source correctness.
URL="http://localhost:${API_PORT}/vrooli.web_search.v1.findings.FindingsService/SearchFindings"
BODY='{"query":"PostgreSQL default port","limit":5}'
curl -fsS --max-time 30 -H 'Content-Type: application/json' -d "$BODY" "$URL" >/dev/null
for _ in {1..20}; do
  curl -fsS --max-time 10 -o /dev/null -w '%{time_total}\n' -H 'Content-Type: application/json' -d "$BODY" "$URL"
done > "$ARTIFACT_DIR/warm-latency-seconds.txt"
jq -s 'sort | {samples:length,p95_ms:(.[18]*1000)}' "$ARTIFACT_DIR/warm-latency-seconds.txt" > "$ARTIFACT_DIR/latency.json"
echo "Evidence collected: $ARTIFACT_DIR"
echo 'Review citations, capture provenance, and execution-scoped tool events before recording manual requirement evidence. No requirement was automatically marked passed.'
jq '{status,result}' "$ARTIFACT_DIR/result.json"
