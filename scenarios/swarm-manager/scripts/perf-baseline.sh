#!/usr/bin/env bash
# perf-baseline.sh — measures swarm-manager read-path latency against the
# running scenario.
#
# The operator inbox reads three projections on every board interaction:
# the ranked next-action feed, the Plan board, and the backlog summary. This
# script times each one over several warm runs and reports the median, so a
# projection change that reintroduces per-item store scans is visible as a
# number instead of a feeling.
#
#   scripts/perf-baseline.sh            # human report
#   scripts/perf-baseline.sh --json     # machine report
#   scripts/perf-baseline.sh --check    # non-zero exit when a budget is blown
#
# Budgets are deliberately loose relative to measured medians: this guards
# against regression, not against noise. Override with PERF_FEED_MAX_MS /
# PERF_PLAN_MAX_MS / PERF_SUMMARY_MAX_MS.

set -euo pipefail

SCENARIO_NAME="swarm-manager"

RUNS="${PERF_RUNS:-5}"
FEED_MAX_MS="${PERF_FEED_MAX_MS:-300}"
PLAN_MAX_MS="${PERF_PLAN_MAX_MS:-600}"
SUMMARY_MAX_MS="${PERF_SUMMARY_MAX_MS:-200}"

OUTPUT="human"
CHECK=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --json) OUTPUT="json" ;;
    --check) CHECK=1 ;;
    --runs) RUNS="$2"; shift ;;
    -h|--help)
      sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "perf-baseline: unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

resolve_api_port() {
  if [[ -n "${SWARM_MANAGER_API_PORT:-}" ]]; then
    echo "$SWARM_MANAGER_API_PORT"
    return 0
  fi
  local port
  port="$(vrooli scenario status "$SCENARIO_NAME" --json 2>/dev/null | jq -r '.scenario.ports.API_PORT // empty')"
  if [[ -z "$port" || "$port" == "null" ]]; then
    echo "perf-baseline: cannot resolve API port — start the scenario with 'vrooli scenario start $SCENARIO_NAME'" >&2
    return 1
  fi
  echo "$port"
}

API_PORT="$(resolve_api_port)"
BASE_URL="http://localhost:${API_PORT}/api/v1"

# median_ms <path> — one warm-up request followed by $RUNS timed requests.
# Prints "<median_ms> <bytes>".
median_ms() {
  local path="$1"
  local url="${BASE_URL}/${path}"
  local bytes=""
  local -a samples=()

  # Warm-up: the first read after a restart pays cold page-cache cost that no
  # operator interaction repeats.
  curl -fsS -o /dev/null "$url" >/dev/null 2>&1 || {
    echo "perf-baseline: request failed: $url" >&2
    return 1
  }

  local i
  for ((i = 0; i < RUNS; i++)); do
    local measured
    measured="$(curl -fsS -o /dev/null -w '%{time_total} %{size_download}' "$url")"
    samples+=("$(awk -v t="${measured%% *}" 'BEGIN { printf "%.0f", t * 1000 }')")
    bytes="${measured##* }"
  done

  local median
  median="$(printf '%s\n' "${samples[@]}" | sort -n | awk -v n="$RUNS" 'NR == int((n + 1) / 2) { print; exit }')"
  echo "$median $bytes"
}

declare -A MEDIAN BYTES BUDGET
ENDPOINTS=("next-actions/feed" "plan" "backlog/summary")
BUDGET["next-actions/feed"]="$FEED_MAX_MS"
BUDGET["plan"]="$PLAN_MAX_MS"
BUDGET["backlog/summary"]="$SUMMARY_MAX_MS"

for endpoint in "${ENDPOINTS[@]}"; do
  read -r m b <<<"$(median_ms "$endpoint")"
  MEDIAN["$endpoint"]="$m"
  BYTES["$endpoint"]="$b"
done

breached=0
for endpoint in "${ENDPOINTS[@]}"; do
  if (( MEDIAN["$endpoint"] > BUDGET["$endpoint"] )); then
    breached=1
  fi
done

if [[ "$OUTPUT" == "json" ]]; then
  {
    printf '{"runs":%s,"api_port":%s,"endpoints":[' "$RUNS" "$API_PORT"
    sep=""
    for endpoint in "${ENDPOINTS[@]}"; do
      status="ok"
      if (( MEDIAN["$endpoint"] > BUDGET["$endpoint"] )); then status="over_budget"; fi
      printf '%s{"path":"/api/v1/%s","median_ms":%s,"bytes":%s,"budget_ms":%s,"status":"%s"}' \
        "$sep" "$endpoint" "${MEDIAN[$endpoint]}" "${BYTES[$endpoint]}" "${BUDGET[$endpoint]}" "$status"
      sep=","
    done
    printf '],"within_budget":%s}\n' "$([[ $breached -eq 0 ]] && echo true || echo false)"
  }
else
  printf 'swarm-manager read-path latency (median of %s warm runs, API port %s)\n\n' "$RUNS" "$API_PORT"
  printf '  %-28s %10s %10s %10s\n' "ENDPOINT" "MEDIAN" "BYTES" "BUDGET"
  for endpoint in "${ENDPOINTS[@]}"; do
    marker=""
    if (( MEDIAN["$endpoint"] > BUDGET["$endpoint"] )); then marker="  OVER BUDGET"; fi
    printf '  %-28s %9sms %10s %9sms%s\n' \
      "/api/v1/$endpoint" "${MEDIAN[$endpoint]}" "${BYTES[$endpoint]}" "${BUDGET[$endpoint]}" "$marker"
  done
  echo
fi

if (( CHECK == 1 && breached == 1 )); then
  echo "perf-baseline: at least one endpoint exceeded its latency budget" >&2
  exit 1
fi
