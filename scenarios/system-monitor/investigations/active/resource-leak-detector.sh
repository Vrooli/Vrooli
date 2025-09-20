#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Resource Leak Detector
# DESCRIPTION: Detects file descriptor, memory, and connection leaks across processes
# CATEGORY: resource-management
# TRIGGERS: fd_usage > 80%, memory_growth_pattern, connection_buildup
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-20
# VERSION: 1.1

set -euo pipefail

SCRIPT_NAME="resource-leak-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
RESULTS_FILE="${OUTPUT_DIR}/results.json"
mkdir -p "${OUTPUT_DIR}"

jq -n '{
  investigation: "resource-leak-detector",
  timestamp: "",
  fd_summary: {},
  top_fd_processes: [],
  high_memory_processes: [],
  socket_summary: {},
  findings: [],
  recommendations: []
}' > "${RESULTS_FILE}"

update_json() {
  local filter="$1"
  jq "${filter}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

update_json_value() {
  local filter="$1"
  local payload="$2"
  jq --argjson value "${payload}" "${filter}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

add_finding() {
  local text="$1"
  jq --arg text "${text}" '.findings += [$text]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

add_recommendation() {
  local text="$1"
  jq --arg text "${text}" '.recommendations += [$text]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

echo "🔎 Starting Resource Leak Detection..."
update_json ' .timestamp = (now | todateiso8601) '

# File descriptor summary
if [[ -r /proc/sys/fs/file-nr ]]; then
  read -r allocated unused max_fds < /proc/sys/fs/file-nr
  used=$((allocated - unused))
  fd_percent=$(awk -v u="${used}" -v m="${max_fds}" 'BEGIN{ if (m==0) {print 0} else {printf "%.2f", (u/m)*100} }')
  update_json_value '.fd_summary = $value' "$(jq -n --argjson used "${used}" --argjson max "${max_fds}" --arg percent "${fd_percent}" '{used: $used, max: $max, percent: ($percent | tonumber)}')"
else
  add_finding "Unable to read /proc/sys/fs/file-nr"
fi

# Top FD consumers
fd_json='[]'
if [[ -d /proc ]]; then
  set +o pipefail
  mapfile -t fd_entries < <(for pid_dir in /proc/[0-9]*; do
    pid=${pid_dir#/proc/}
    fd_count=$( (find "${pid_dir}/fd" -mindepth 1 -maxdepth 1 2>/dev/null || true) | wc -l )
    fd_count=${fd_count// /}
    if (( fd_count > 0 )); then
      user=$(stat -c %U "${pid_dir}" 2>/dev/null || echo "unknown")
      cmd=$(tr -d '\0' < "${pid_dir}/comm" 2>/dev/null || echo "${pid}")
      printf '%s\t%s\t%s\t%s\n' "${fd_count}" "${pid}" "${user}" "${cmd}"
    fi
done | sort -t $'\t' -nrk1 | head -n 10)
  set -o pipefail
  if (( ${#fd_entries[@]} > 0 )); then
    fd_json=$(printf '%s\n' "${fd_entries[@]}" | jq -Rcs 'split("\n") | map(select(length>0)) | map(split("\t")) | map({fd_count:(.[0]|tonumber), pid:(.[1]|tonumber), user:.[2], command:(.[3] // "")})')
  fi
else
  add_finding "Proc filesystem not available - cannot inspect file descriptors"
fi
update_json_value '.top_fd_processes = $value' "${fd_json}"

# High memory processes snapshot (reuse ps data)
high_mem_json='[]'
if command -v ps >/dev/null 2>&1; then
  high_mem_raw=$(ps -eo pid=,user=,%mem=,rss=,comm= --sort=-%mem | head -n 5)
  if [[ -n "${high_mem_raw}" ]]; then
    high_mem_json=$(printf '%s\n' "${high_mem_raw}" | awk '{printf "%s\t%s\t%s\t%s\t%s\n", $1,$2,$3,$4,$5}' | jq -Rcs 'split("\n") | map(select(length>0)) | map(split("\t")) | map({pid:(.[0]|tonumber), user:.[1], mem_percent:(.[2]|tonumber), rss_kb:(.[3]|tonumber), command:(.[4] // "")})')
  fi
fi
update_json_value '.high_memory_processes = $value' "${high_mem_json}"

# Socket summary
if command -v ss >/dev/null 2>&1; then
  set +o pipefail
  established=$(ss -tan state established 2>/dev/null | tail -n +2 | wc -l || echo 0)
  time_wait=$(ss -tan state time-wait 2>/dev/null | tail -n +2 | wc -l || echo 0)
  listen=$(ss -tan state listen 2>/dev/null | tail -n +2 | wc -l || echo 0)
  set -o pipefail
  established=${established// /}
  time_wait=${time_wait// /}
  listen=${listen// /}
  update_json_value '.socket_summary = $value' "$(jq -n --argjson est "${established:-0}" --argjson tw "${time_wait:-0}" --argjson listen "${listen:-0}" '{established: $est, time_wait: $tw, listen: $listen}')"
else
  add_finding "ss utility not available - socket summary skipped"
fi

# Recommendations
if jq -e '.fd_summary.percent // 0 | tonumber > 80' "${RESULTS_FILE}" >/dev/null 2>&1; then
  add_recommendation "File descriptor usage above 80% - consider increasing limits or closing unused handles"
fi

if [[ "${fd_json}" != '[]' ]]; then
  add_recommendation "Review top file descriptor consumers for potential leaks"
fi

if [[ "${high_mem_json}" != '[]' ]]; then
  add_recommendation "High memory processes detected - investigate for leaks"
fi

add_recommendation "Monitor TIME_WAIT growth to detect socket leaks"

echo "✅ Resource leak detection complete! Results: ${RESULTS_FILE}"
cat "${RESULTS_FILE}"
