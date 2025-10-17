#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Memory Leak and Process Health Detector
# DESCRIPTION: Detects memory leaks, long-running processes, and resource consumption patterns
# CATEGORY: resource-management
# TRIGGERS: high_memory_usage, memory_growth_pattern, resource_exhaustion
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-14
# LAST_MODIFIED: 2025-09-20
# VERSION: 1.1

set -euo pipefail

SCRIPT_NAME="memory-leak-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
RESULTS_FILE="${OUTPUT_DIR}/results.json"
mkdir -p "${OUTPUT_DIR}"

jq -n '{
  investigation: "memory-leak-detector",
  timestamp: "",
  memory_overview: {},
  high_memory_processes: [],
  long_running_processes: [],
  container_memory: [],
  python_processes: [],
  findings: [],
  recommendations: []
}' > "${RESULTS_FILE}"

update_json() {
  local filter="$1"
  jq "${filter}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

update_json_value() {
  local filter="$1"
  local json_payload="$2"
  jq --argjson value "${json_payload}" "${filter}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

add_finding() {
  local text="$1"
  jq --arg text "${text}" '.findings += [$text]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

add_recommendation() {
  local text="$1"
  jq --arg text "${text}" '.recommendations += [$text]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

echo "🔍 Starting Memory Leak and Process Health Detection..."
update_json ' .timestamp = (now | todateiso8601) '

# Memory overview
if command -v free >/dev/null 2>&1; then
  total_mem=$(free -m | awk 'NR==2{print $2}')
  used_mem=$(free -m | awk 'NR==2{print $3}')
  free_mem=$(free -m | awk 'NR==2{print $4}')
  cache_mem=$(free -m | awk 'NR==2{print $6}')
  if [[ -z "${total_mem}" || "${total_mem}" == "0" ]]; then
    usage_percent=0
  else
    usage_percent=$(awk -v u="${used_mem}" -v t="${total_mem}" 'BEGIN{ if (t==0) {print 0} else {printf "%.1f", (u/t)*100} }')
  fi
  update_json_value '.memory_overview = $value' "$(jq -n --argjson total "${total_mem:-0}" --argjson used "${used_mem:-0}" --argjson free "${free_mem:-0}" --argjson cache "${cache_mem:-0}" --arg usage "${usage_percent}" '{total_mb: $total, used_mb: $used, free_mb: $free, cache_mb: $cache, usage_percent: ($usage | tonumber)}')"
  mem_percent_numeric=$(printf '%.1f' "${usage_percent}")
else
  add_finding "free command unavailable - memory overview skipped"
  mem_percent_numeric=0
fi

echo "🔥 Gathering high-memory processes..."
if command -v ps >/dev/null 2>&1; then
  high_mem_raw=$(ps -eo pid=,user=,%mem=,rss=,comm= --sort=-%mem | head -n 10)
  if [[ -n "${high_mem_raw}" ]]; then
    high_mem_json=$(printf '%s
' "${high_mem_raw}" | awk '{printf "%s\t%s\t%s\t%s\t%s\n", $1,$2,$3,$4,$5}' | jq -Rsc 'split("\n") | map(select(length>0)) | map(split("\t")) | map({pid:(.[0]|tonumber), user:.[1], mem_percent:(.[2]|tonumber), rss_kb:(.[3]|tonumber), command:(.[4] // "")})')
  else
    high_mem_json='[]'
  fi
  update_json_value '.high_memory_processes = $value' "${high_mem_json:-[]}" || true
else
  add_finding "ps command unavailable - high-memory process check skipped"
  high_mem_json='[]'
fi

echo "⏱️ Checking long-running processes..."
if command -v ps >/dev/null 2>&1; then
  long_runner_raw=$(ps -eo pid=,user=,%mem=,%cpu=,etime=,comm= --sort=-etime | head -n 10)
  if [[ -n "${long_runner_raw}" ]]; then
    long_runner_json=$(printf '%s
' "${long_runner_raw}" | awk '{printf "%s\t%s\t%s\t%s\t%s\t%s\n", $1,$2,$3,$4,$5,$6}' | jq -Rsc 'split("\n") | map(select(length>0)) | map(split("\t")) | map({pid:(.[0]|tonumber), user:.[1], mem_percent:(.[2]|tonumber), cpu_percent:(.[3]|tonumber), elapsed: (.[4] // ""), command:(.[5] // "")})')
  else
    long_runner_json='[]'
  fi
  update_json_value '.long_running_processes = $value' "${long_runner_json:-[]}" || true
else
  add_finding "ps command unavailable - long-running process check skipped"
fi

echo "🐳 Inspecting container memory..."
container_mem_json='[]'
if command -v docker >/dev/null 2>&1; then
  if container_output=$(timeout 5 docker stats --no-stream --format '{{json .}}' 2>/dev/null || true); then
    if [[ -n "${container_output}" ]]; then
      container_mem_json=$(printf '%s
' "${container_output}" | jq -s 'map({
        container: .Name,
        memory_usage: .MemUsage,
        memory_percent: (.MemPerc | gsub("%";"") | tonumber? // 0),
        cpu_percent: (.CPUPerc | gsub("%";"") | tonumber? // 0)
      })')
    fi
  fi
else
  add_finding "Docker CLI not available - skipping container memory analysis"
fi
update_json_value '.container_memory = $value' "${container_mem_json}"

echo "🐍 Reviewing Python processes..."
python_json='[]'
if command -v ps >/dev/null 2>&1; then
  python_raw=$(ps -eo pid=,user=,%mem=,%cpu=,state=,comm= | awk 'tolower($6) ~ /python/' | head -n 25)
  if [[ -n "${python_raw}" ]]; then
    python_json=$(printf '%s
' "${python_raw}" | awk '{printf "%s\t%s\t%s\t%s\t%s\t%s\n", $1,$2,$3,$4,$5,$6}' | jq -Rsc 'split("\n") | map(select(length>0)) | map(split("\t")) | map({pid:(.[0]|tonumber), user:.[1], mem_percent:(.[2]|tonumber), cpu_percent:(.[3]|tonumber), state:(.[4] // ""), command:(.[5] // "")})')
  fi
fi
update_json_value '.python_processes = $value' "${python_json}"

# Recommendations
if (( $(printf '%.0f' "${mem_percent_numeric}") > 80 )); then
  add_recommendation "High memory usage (${mem_percent_numeric}%) - investigate top memory consumers"
fi

python_count=$(jq 'length' <<<"${python_json}" 2>/dev/null || echo 0)
if (( python_count > 20 )); then
  add_recommendation "Many Python processes (${python_count}) detected - review subprocess management"
fi

if [[ "${container_mem_json}" != '[]' ]]; then
  high_container_count=$(jq '[.[] | select(.memory_percent > 50)] | length' <<<"${container_mem_json}" 2>/dev/null || echo 0)
  if (( high_container_count > 0 )); then
    add_recommendation "${high_container_count} containers using >50% memory - review container limits"
  fi
fi

if jq -e 'map(.command) | map(test("sage")) | any' <<<"${high_mem_json}" >/dev/null 2>&1; then
  add_recommendation "SageMath process consuming significant memory - check for computation issues"
fi

echo "✅ Memory leak detection complete! Results: ${RESULTS_FILE}"
cat "${RESULTS_FILE}"
