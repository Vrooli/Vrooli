#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: CPU Analyzer
# DESCRIPTION: Comprehensive CPU analysis with infinite loop and busy loop detection
# CATEGORY: performance
# TRIGGERS: cpu_usage, load_average, high_cpu_api, infinite_loop_suspected, go_service_issue, high_go_cpu, busy_loop_suspected
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2026-02-18
# LAST_MODIFIED: 2026-02-18
# VERSION: 2.0

set -euo pipefail

source "$(dirname "$0")/lib/common.sh"

# ── Argument parsing ────────────────────────────────────────────────────────
MODE="full"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) MODE="$2"; shift 2 ;;
    *) shift ;;
  esac
done

# ── Thresholds ──────────────────────────────────────────────────────────────
CPU_THRESHOLD="${INVESTIGATION_CPU_THRESHOLD:-10.0}"
GO_THRESHOLD="${INVESTIGATION_CPU_GO_THRESHOLD:-50.0}"

# ── Initialize ──────────────────────────────────────────────────────────────
init_investigation "cpu-analyzer"

jq -n --arg mode "${MODE}" '{
  investigation: "cpu-analyzer",
  timestamp: (now | todateiso8601),
  mode: $mode,
  system_overview: {},
  high_cpu_processes: [],
  process_trees: [],
  api_loop_analysis: {},
  go_busy_loop_analysis: {},
  patterns: {},
  findings: [],
  recommendations: []
}' > "${RESULTS_FILE}"

# ── Functions ───────────────────────────────────────────────────────────────

gather_system_overview() {
  local load_avg cpu_count mem_percent
  load_avg=$(uptime | awk -F'load average:' '{print $2}' | tr -d ' ')
  cpu_count=$(nproc)
  mem_percent=$(free -m | awk 'NR==2{ if ($2>0) printf "%.1f", $3/$2*100; else print "0" }')

  update_json_value '.system_overview = $value' "$(jq -n \
    --arg load "${load_avg}" \
    --argjson cores "${cpu_count}" \
    --argjson mem "${mem_percent}" \
    '{load_average: $load, cpu_cores: $cores, memory_usage_percent: $mem}')"
}

find_high_cpu_processes() {
  local top_output proc_json="[]"
  top_output=$(safe_timeout 30 top -b -n 1) || true

  if [[ -n "${top_output}" ]]; then
    proc_json=$(printf '%s\n' "${top_output}" | awk -v threshold="${CPU_THRESHOLD}" '
      NR>7 && $9+0 >= threshold {
        printf "%s\t%s\t%s\t%s\t%s\n", $1, $2, $9, $10, $12
      }' | jq -Rsc '
        split("\n") | map(select(length>0)) | map(split("\t")) |
        map({
          pid: (.[0] | tonumber),
          user: .[1],
          cpu: (.[2] | tonumber),
          mem: (.[3] | tonumber),
          command: (.[4] // "")
        })')
  fi

  update_json_value '.high_cpu_processes = $value' "${proc_json}"
}

build_process_trees() {
  local pids tree_json="[]"
  pids=$(jq -r '.[].pid' <<< "$(jq '.high_cpu_processes' "${RESULTS_FILE}")" 2>/dev/null) || true

  if [[ -n "${pids}" ]]; then
    local entries=""
    while IFS= read -r pid; do
      [[ "${pid}" =~ ^[0-9]+$ ]] || continue
      local tree parent_info
      tree=$(safe_timeout 10 pstree -p "${pid}" 2>/dev/null | head -20) || true
      parent_info=$(safe_timeout 5 ps -o pid=,ppid=,comm= --no-headers -p "${pid}" 2>/dev/null) || true
      if [[ -n "${tree}" ]]; then
        entries="${entries}$(jq -n --argjson pid "${pid}" --arg tree "${tree}" --arg parent "${parent_info}"),"
      fi
    done <<< "${pids}"
    if [[ -n "${entries}" ]]; then
      tree_json=$(printf '[%s]' "${entries%,}" | jq 'map({pid: .pid, tree: .tree, parent_info: .parent})')
    fi
  fi

  update_json_value '.process_trees = $value' "${tree_json}"
}

detect_api_infinite_loops() {
  local api_procs result_json
  api_procs=$(ps aux --sort=-%cpu | head -20 | grep -E "\.(api|server|monitor)" | grep -v grep | head -5 || true)

  if [[ -z "${api_procs}" ]]; then
    update_json_value '.api_loop_analysis = $value' '{"status":"healthy","message":"No high CPU API processes detected","processes":[]}'
    return
  fi

  local entries=""
  while IFS= read -r proc_line; do
    [[ -n "${proc_line}" ]] || continue
    local pid cpu cmd io_read=0 io_write=0 threads=0 io_rate=0 loop_suspected="false" loop_reason=""
    pid=$(awk '{print $2}' <<< "${proc_line}")
    cpu=$(awk '{print $3}' <<< "${proc_line}")
    cmd=$(awk '{print $11}' <<< "${proc_line}")

    if [[ -r "/proc/${pid}/io" ]]; then
      io_read=$(awk -F: '/^read_bytes/{gsub(/ /,"",$2); print $2}' "/proc/${pid}/io" 2>/dev/null || echo 0)
      io_write=$(awk -F: '/^write_bytes/{gsub(/ /,"",$2); print $2}' "/proc/${pid}/io" 2>/dev/null || echo 0)
    fi
    threads=$(awk '/^Threads:/{print $2}' "/proc/${pid}/status" 2>/dev/null || echo 0)

    if (( $(echo "${cpu} > 0" | bc -l 2>/dev/null || echo 0) )); then
      io_rate=$(echo "scale=2; (${io_read} + ${io_write}) / ${cpu}" | bc 2>/dev/null || echo 0)
    fi

    if (( $(echo "${cpu} > 100" | bc -l 2>/dev/null || echo 0) )); then
      if (( $(echo "${io_rate} < 10000" | bc -l 2>/dev/null || echo 0) )); then
        loop_suspected="true"
        loop_reason="High CPU with minimal I/O activity"
      fi
    fi

    entries="${entries}$(jq -n \
      --argjson pid "${pid}" --arg cmd "${cmd}" --argjson cpu "${cpu}" \
      --argjson threads "${threads:-0}" --argjson io_read "${io_read:-0}" \
      --argjson io_write "${io_write:-0}" --argjson io_rate "${io_rate:-0}" \
      --argjson loop "${loop_suspected}" --arg reason "${loop_reason}" \
      '{pid:$pid, command:$cmd, cpu_percent:$cpu, threads:$threads,
        io_read_bytes:$io_read, io_write_bytes:$io_write, io_per_cpu:$io_rate,
        loop_suspected:$loop, loop_reason:$reason}'),"
  done <<< "${api_procs}"

  local procs_array status="healthy" message="No infinite loop patterns detected" affected=0
  procs_array=$(printf '[%s]' "${entries%,}")
  affected=$(jq '[.[] | select(.loop_suspected == true)] | length' <<< "${procs_array}")

  if [[ "${affected}" -gt 0 ]]; then
    status="critical"
    message="Infinite loop conditions detected"
    add_finding "API infinite loop detected in ${affected} process(es): high CPU with minimal I/O"
    add_recommendation "IMMEDIATE: Restart affected API services to restore normal operation"
    add_recommendation "Review recent code changes for polling loops without delays"
    add_recommendation "Add pprof CPU profiling to identify hot code paths"
  fi

  result_json=$(jq -n --arg st "${status}" --arg msg "${message}" \
    --argjson n "${affected}" --argjson procs "${procs_array}" \
    '{status:$st, message:$msg, affected_processes:$n, processes:$procs}')
  update_json_value '.api_loop_analysis = $value' "${result_json}"
}

detect_go_busy_loops() {
  local go_procs
  go_procs=$(ps aux | grep -E "(system-monitor-api|api/main|go-build)" | grep -v grep \
    | awk -v threshold="${GO_THRESHOLD}" '$3+0 >= threshold {print $2":"$3":"$11}') || true

  if [[ -z "${go_procs}" ]]; then
    update_json_value '.go_busy_loop_analysis = $value' '{"status":"healthy","processes":[]}'
    return
  fi

  local entries=""
  for proc in ${go_procs}; do
    local pid cpu cmd
    pid=$(cut -d: -f1 <<< "${proc}")
    cpu=$(cut -d: -f2 <<< "${proc}")
    cmd=$(cut -d: -f3 <<< "${proc}")

    [[ -e "/proc/${pid}/io" ]] || continue

    local read_bytes write_bytes syscalls runtime_secs
    read_bytes=$(awk -F: '/^read_bytes/{gsub(/ /,"",$2); print $2}' "/proc/${pid}/io" 2>/dev/null || echo 0)
    write_bytes=$(awk -F: '/^write_bytes/{gsub(/ /,"",$2); print $2}' "/proc/${pid}/io" 2>/dev/null || echo 0)
    syscalls=$(awk -F: '/^syscr/{gsub(/ /,"",$2); print $2}' "/proc/${pid}/io" 2>/dev/null || echo 0)
    runtime_secs=$(ps -o etimes= -p "${pid}" 2>/dev/null | tr -d ' ' || echo 1)
    runtime_secs=$((runtime_secs > 0 ? runtime_secs : 1))

    local read_rate=$((read_bytes / runtime_secs))
    local write_rate=$((write_bytes / runtime_secs))
    local syscall_rate=$((syscalls / runtime_secs))

    local vol_sw nonvol_sw
    vol_sw=$(awk '/^voluntary_ctxt_switches:/{print $2}' "/proc/${pid}/status" 2>/dev/null || echo 0)
    nonvol_sw=$(awk '/^nonvoluntary_ctxt_switches:/{print $2}' "/proc/${pid}/status" 2>/dev/null || echo 0)

    local confidence=0
    if (( $(echo "${cpu} > 100" | bc -l) )); then confidence=$((confidence + 30)); fi
    if [[ ${read_rate} -lt 1000 && ${write_rate} -lt 1000 ]]; then confidence=$((confidence + 20)); fi
    if [[ ${runtime_secs} -gt 60 ]]; then
      local vol_rate=$((vol_sw / runtime_secs))
      if [[ ${vol_rate} -lt 100 ]]; then confidence=$((confidence + 30)); fi
    fi
    if [[ ${syscall_rate} -gt 1000 && ${read_rate} -lt 1000 ]]; then confidence=$((confidence + 20)); fi

    local busy_loop="false"
    if [[ ${confidence} -ge 60 ]]; then busy_loop="true"; fi

    entries="${entries}$(jq -n \
      --argjson pid "${pid}" --arg cmd "${cmd}" --argjson cpu "${cpu}" \
      --argjson rt "${runtime_secs}" --argjson rr "${read_rate}" --argjson wr "${write_rate}" \
      --argjson sr "${syscall_rate}" --argjson vs "${vol_sw:-0}" --argjson nvs "${nonvol_sw:-0}" \
      --argjson bl "${busy_loop}" --argjson conf "${confidence}" \
      '{pid:$pid, command:$cmd, cpu_percent:$cpu, runtime_seconds:$rt,
        read_bytes_per_sec:$rr, write_bytes_per_sec:$wr, syscalls_per_sec:$sr,
        voluntary_ctxt_switches:$vs, nonvoluntary_ctxt_switches:$nvs,
        busy_loop_detected:$bl, confidence_score:$conf}'),"
  done

  local procs_array status="healthy"
  procs_array=$(printf '[%s]' "${entries%,}")
  local busy_count
  busy_count=$(jq '[.[] | select(.busy_loop_detected == true)] | length' <<< "${procs_array}")

  if [[ "${busy_count}" -gt 0 ]]; then
    status="critical"
    add_finding "Go busy loop detected in ${busy_count} process(es) with high confidence"
    add_recommendation "Add time.Sleep() to polling loops in Go services"
    add_recommendation "Consider using channels or condition variables instead of polling"
  fi

  update_json_value '.go_busy_loop_analysis = $value' "$(jq -n \
    --arg st "${status}" --argjson procs "${procs_array}" '{status:$st, processes:$procs}')"
}

analyze_patterns() {
  local dup_cmds total_procs lsof_count
  dup_cmds=$(safe_timeout 20 ps aux | awk '{print $11}' | sort | uniq -c | sort -rn | head -10 \
    | awk '$1 > 5 {printf "%s\t%s\n", $1, $2}' \
    | jq -Rsc 'split("\n") | map(select(length>0)) | map(split("\t")) |
        map({command: .[1], count: (.[0] | ltrimstr(" ") | tonumber)})') || true
  dup_cmds="${dup_cmds:-[]}"

  lsof_count=$(ps aux | grep -c "[l]sof" || echo 0)
  total_procs=$(ps aux | wc -l)

  update_json_value '.patterns = $value' "$(jq -n \
    --argjson dups "${dup_cmds}" --argjson lsof "${lsof_count}" \
    --argjson total "${total_procs}" \
    '{duplicate_commands: $dups, lsof_processes: $lsof, total_processes: $total}')"

  if [[ ${lsof_count} -gt 10 ]]; then
    add_finding "Excessive lsof processes (${lsof_count})"
    add_recommendation "Replace lsof polling with /proc filesystem reads"
  fi
  if [[ ${total_procs} -gt 1000 ]]; then
    add_finding "High process count: ${total_procs}"
    add_recommendation "Investigate process spawning - ${total_procs} processes running"
  fi
}

generate_recommendations() {
  local load_1min cpu_count
  load_1min=$(jq -r '.system_overview.load_average' "${RESULTS_FILE}" | cut -d',' -f1)
  cpu_count=$(jq -r '.system_overview.cpu_cores' "${RESULTS_FILE}")
  if [[ -n "${load_1min}" && -n "${cpu_count}" ]]; then
    if (( $(echo "${load_1min} > ${cpu_count}" | bc -l 2>/dev/null || echo 0) )); then
      add_recommendation "Load average (${load_1min}) exceeds CPU count (${cpu_count}) - system overloaded"
    fi
  fi
}

# ── Execution by mode ───────────────────────────────────────────────────────
gather_system_overview

case "${MODE}" in
  full)
    find_high_cpu_processes
    build_process_trees
    detect_api_infinite_loops
    detect_go_busy_loops
    analyze_patterns
    ;;
  general)
    find_high_cpu_processes
    build_process_trees
    analyze_patterns
    ;;
  api-loop)
    detect_api_infinite_loops
    ;;
  go-busy)
    detect_go_busy_loops
    ;;
esac

generate_recommendations
finalize_investigation
