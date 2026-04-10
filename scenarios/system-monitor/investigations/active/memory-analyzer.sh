#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Memory and Swap Analyzer
# DESCRIPTION: Comprehensive memory/swap analysis with exhaustion pattern detection
# CATEGORY: resource-management
# TRIGGERS: high_memory_usage, memory_growth_pattern, resource_exhaustion, high_swap_usage, swap_thrashing, memory_pressure, process_exhaustion
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2026-02-18
# LAST_MODIFIED: 2026-02-18
# VERSION: 2.0

set -euo pipefail
source "$(dirname "$0")/lib/common.sh"

MODE="full"
while [[ $# -gt 0 ]]; do
  case "$1" in --mode) MODE="$2"; shift 2 ;; *) shift ;; esac
done

MEM_THRESHOLD="${INVESTIGATION_MEM_THRESHOLD:-60}"
SWAP_THRESHOLD="${INVESTIGATION_SWAP_THRESHOLD:-30}"

init_investigation "memory-analyzer"
jq -n --arg mode "${MODE}" '{
  investigation:"memory-analyzer", timestamp:(now|todateiso8601), mode:$mode,
  memory_overview:{}, swap_analysis:{}, memory_pressure:{}, cache_pressure:{},
  top_memory_consumers:[], swap_consuming_processes:[], memory_growth_patterns:[],
  process_limits:{}, system_tuning:{}, exhaustion_patterns:[],
  findings:[], recommendations:[], risk_level:"low"
}' > "${RESULTS_FILE}"

gather_memory_overview() {
  if ! command -v free >/dev/null 2>&1; then add_finding "free unavailable"; return; fi
  local total used free_val shared cache available
  read -r _ total used free_val shared cache available <<< "$(free -m | awk 'NR==2')"
  total=$(sanitize_int "${total}"); used=$(sanitize_int "${used}")
  local usage_pct=0
  [[ "${total}" -gt 0 ]] && usage_pct=$(awk -v u="${used}" -v t="${total}" 'BEGIN{printf "%.1f",(u/t)*100}')
  update_json_value '.memory_overview = $value' "$(jq -n \
    --argjson total "${total}" --argjson used "${used}" \
    --argjson fv "$(sanitize_int "${free_val}")" --argjson sh "$(sanitize_int "${shared}")" \
    --argjson ca "$(sanitize_int "${cache}")" --argjson av "$(sanitize_int "${available}")" \
    --arg up "${usage_pct}" \
    '{total_mb:$total,used_mb:$used,free_mb:$fv,shared_mb:$sh,cache_mb:$ca,available_mb:$av,usage_percent:($up|tonumber)}')"
}

analyze_swap() {
  local total used free_val
  read -r _ total used free_val <<< "$(free -m | awk 'NR==3')"
  total=$(sanitize_int "${total}"); used=$(sanitize_int "${used}")
  local swap_pct=0
  [[ "${total}" -gt 0 ]] && swap_pct=$(awk -v u="${used}" -v t="${total}" 'BEGIN{printf "%.1f",(u/t)*100}')
  local vline swap_in swap_out
  vline=$(safe_timeout 5 vmstat 1 2 | tail -1)
  swap_in=$(sanitize_int "$(awk '{print $7}' <<< "${vline}")")
  swap_out=$(sanitize_int "$(awk '{print $8}' <<< "${vline}")")
  local tsi tso
  tsi=$(sanitize_int "$(safe_timeout 3 vmstat -s | awk '/pages swapped in/{print $1}')")
  tso=$(sanitize_int "$(safe_timeout 3 vmstat -s | awk '/pages swapped out/{print $1}')")
  local active="false"
  [[ "${swap_in}" -gt 0 || "${swap_out}" -gt 0 ]] && active="true"
  update_json_value '.swap_analysis = $value' "$(jq -n \
    --argjson total "${total}" --argjson used "${used}" \
    --argjson fv "$(sanitize_int "${free_val}")" --arg sp "${swap_pct}" \
    --argjson si "${swap_in}" --argjson so "${swap_out}" \
    --argjson tsi "${tsi}" --argjson tso "${tso}" --argjson act "${active}" \
    '{total_mb:$total,used_mb:$used,free_mb:$fv,usage_percent:($sp|tonumber),recent_swap_in_kb:$si,recent_swap_out_kb:$so,total_swap_in_pages:$tsi,total_swap_out_pages:$tso,active_swapping:$act}')"
}

analyze_memory_pressure() {
  local a10=0 a60=0 a300=0 atot=0
  if [[ -f /proc/pressure/memory ]]; then
    local pl; pl=$(safe_timeout 3 awk '/^some/{print $2,$3,$4,$5}' /proc/pressure/memory)
    if [[ -n "${pl}" ]]; then
      a10=$(awk '{split($1,a,"=");print a[2]}' <<< "${pl}")
      a60=$(awk '{split($2,a,"=");print a[2]}' <<< "${pl}")
      a300=$(awk '{split($3,a,"=");print a[2]}' <<< "${pl}")
      atot=$(awk '{split($4,a,"=");print a[2]}' <<< "${pl}")
    fi
  fi
  local dirty wb slab srec
  dirty=$(sanitize_int "$(awk '/^Dirty:/{print $2}' /proc/meminfo 2>/dev/null)")
  wb=$(sanitize_int "$(awk '/^Writeback:/{print $2}' /proc/meminfo 2>/dev/null)")
  slab=$(sanitize_int "$(awk '/^Slab:/{print $2}' /proc/meminfo 2>/dev/null)")
  srec=$(sanitize_int "$(awk '/^SReclaimable:/{print $2}' /proc/meminfo 2>/dev/null)")
  update_json_value '.memory_pressure = $value' "$(jq -n \
    --argjson a10 "${a10:-0}" --argjson a60 "${a60:-0}" --argjson a300 "${a300:-0}" \
    --argjson at "${atot:-0}" --argjson d "${dirty}" --argjson w "${wb}" \
    '{psi_avg10:$a10,psi_avg60:$a60,psi_avg300:$a300,psi_total_us:$at,dirty_kb:$d,writeback_kb:$w}')"
  update_json_value '.cache_pressure = $value' "$(jq -n \
    --argjson d "${dirty}" --argjson w "${wb}" --argjson s "${slab}" --argjson r "${srec}" \
    '{dirty_mb:($d/1024|floor),writeback_mb:($w/1024|floor),slab_mb:($s/1024|floor),reclaimable_mb:($r/1024|floor)}')"
}

find_top_memory_consumers() {
  local json="[]"
  if command -v ps >/dev/null 2>&1; then
    local raw; raw=$(ps aux --sort=-rss | awk 'NR>1&&NR<=11{
      c=$11;for(i=12;i<=NF;i++)c=c" "$i;printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",$2,$6,$5,$3,$4,$1,substr(c,1,100)}')
    [[ -n "${raw}" ]] && json=$(jq -Rsc 'split("\n")|map(select(length>0))|map(split("\t"))|map({
      pid:(.[0]|tonumber),rss_mb:(.[1]|tonumber/1024|floor),vsz_mb:(.[2]|tonumber/1024|floor),
      cpu_percent:(.[3]|tonumber),mem_percent:(.[4]|tonumber),user:.[5],command:.[6]})' <<< "${raw}")
  fi
  update_json_value '.top_memory_consumers = $value' "${json}"
}

find_swap_consumers() {
  local sorted; sorted=$(safe_timeout 10 bash -c '
    for f in /proc/[0-9]*/status; do
      [[ -r "$f" ]] || continue
      s=$(awk "/^VmSwap:/{print \$2}" "$f" 2>/dev/null)
      [[ -n "$s" && "$s" -gt 0 ]] 2>/dev/null && echo "$(basename "$(dirname "$f")") $s"
    done | sort -k2 -rn | head -10')
  local json="[]"
  if [[ -n "${sorted}" ]]; then
    local entries=()
    while IFS=' ' read -r pid swap_kb; do
      [[ -z "${pid}" ]] && continue
      local cmd user rss_kb
      cmd=$(safe_timeout 2 cat "/proc/${pid}/comm" 2>/dev/null || echo "unknown")
      user=$(ps -p "${pid}" -o user= 2>/dev/null || echo "unknown")
      rss_kb=$(sanitize_int "$(awk '/^VmRSS:/{print $2}' "/proc/${pid}/status" 2>/dev/null)")
      swap_kb=$(sanitize_int "${swap_kb}")
      entries+=("$(jq -n --argjson p "${pid}" --argjson s "$((swap_kb/1024))" \
        --argjson r "$((rss_kb/1024))" --arg c "${cmd}" --arg u "${user}" \
        '{pid:$p,swap_mb:$s,rss_mb:$r,command:$c,user:$u}')")
    done <<< "${sorted}"
    [[ ${#entries[@]} -gt 0 ]] && json=$(printf '%s\n' "${entries[@]}" | jq -s '.')
  fi
  update_json_value '.swap_consuming_processes = $value' "${json}"
}

analyze_memory_growth() {
  local entries=()
  for pid in $(ps aux --sort=-%mem | awk 'NR>1&&NR<=11{print $2}'); do
    [[ -f "/proc/${pid}/status" ]] || continue
    local comm rss_kb peak_kb
    comm=$(safe_timeout 2 cat "/proc/${pid}/comm" 2>/dev/null || echo "unknown")
    rss_kb=$(sanitize_int "$(awk '/^VmRSS:/{print $2}' "/proc/${pid}/status" 2>/dev/null)")
    peak_kb=$(sanitize_int "$(awk '/^VmPeak:/{print $2}' "/proc/${pid}/status" 2>/dev/null)")
    [[ "${peak_kb}" -eq 0 || "${rss_kb}" -eq 0 ]] && continue
    local gp; gp=$(awk -v p="${peak_kb}" -v r="${rss_kb}" 'BEGIN{printf "%.1f",((p-r)/p)*100}')
    entries+=("$(jq -n --argjson p "${pid}" --arg c "${comm}" \
      --argjson cur "$((rss_kb/1024))" --argjson pk "$((peak_kb/1024))" --arg g "${gp}" \
      '{pid:$p,command:$c,current_mb:$cur,peak_mb:$pk,growth_potential_percent:($g|tonumber)}')")
  done
  local json="[]"
  [[ ${#entries[@]} -gt 0 ]] && json=$(printf '%s\n' "${entries[@]}" | jq -s '.')
  update_json_value '.memory_growth_patterns = $value' "${json}"
}

inspect_container_memory() {
  command -v docker >/dev/null 2>&1 || { add_finding "Docker CLI not available - skipping container memory"; return; }
  local out; out=$(safe_timeout 10 docker stats --no-stream --format '{{json .}}' 2>/dev/null) || true
  [[ -n "${out}" ]] && update_json_value '.container_memory = $value' \
    "$(jq -s 'map({container:.Name,memory_usage:.MemUsage,memory_percent:(.MemPerc|gsub("%";"")|tonumber?//0),cpu_percent:(.CPUPerc|gsub("%";"")|tonumber?//0)})' <<< "${out}")"
}

check_process_limits() {
  local mt ct tp mf cf fp
  mt=$(sanitize_int "$(safe_timeout 2 cat /proc/sys/kernel/threads-max 2>/dev/null)")
  ct=$(sanitize_int "$(ps -eLf 2>/dev/null | wc -l)")
  tp=0; [[ "${mt}" -gt 0 ]] && tp=$(awk -v c="${ct}" -v m="${mt}" 'BEGIN{printf "%.1f",(c/m)*100}')
  mf=$(sanitize_int "$(safe_timeout 2 cat /proc/sys/fs/file-max 2>/dev/null)")
  cf=$(sanitize_int "$(awk '{print $1}' /proc/sys/fs/file-nr 2>/dev/null)")
  fp=0; [[ "${mf}" -gt 0 ]] && fp=$(awk -v c="${cf}" -v m="${mf}" 'BEGIN{printf "%.1f",(c/m)*100}')
  update_json_value '.process_limits = $value' "$(jq -n \
    --argjson ct "${ct}" --argjson mt "${mt}" --arg tp "${tp}" \
    --argjson cf "${cf}" --argjson mf "${mf}" --arg fp "${fp}" \
    '{threads:{current:$ct,maximum:$mt,usage_percent:($tp|tonumber)},file_descriptors:{current:$cf,maximum:$mf,usage_percent:($fp|tonumber)}}')"
}

check_system_tuning() {
  local sw vc dr dbr mf
  sw=$(sanitize_int "$(safe_timeout 2 cat /proc/sys/vm/swappiness 2>/dev/null)")
  vc=$(sanitize_int "$(safe_timeout 2 cat /proc/sys/vm/vfs_cache_pressure 2>/dev/null)")
  dr=$(sanitize_int "$(safe_timeout 2 cat /proc/sys/vm/dirty_ratio 2>/dev/null)")
  dbr=$(sanitize_int "$(safe_timeout 2 cat /proc/sys/vm/dirty_background_ratio 2>/dev/null)")
  mf=$(sanitize_int "$(safe_timeout 2 cat /proc/sys/vm/min_free_kbytes 2>/dev/null)")
  update_json_value '.system_tuning = $value' "$(jq -n \
    --argjson sw "${sw}" --argjson vc "${vc}" --argjson dr "${dr}" \
    --argjson dbr "${dbr}" --argjson mf "${mf}" \
    '{swappiness:$sw,vfs_cache_pressure:$vc,dirty_ratio:$dr,dirty_background_ratio:$dbr,min_free_kbytes:$mf}')"
}

detect_exhaustion_patterns() {
  local mp sp si so psi tp fp cr
  mp=$(jq -r '.memory_overview.usage_percent//0' "${RESULTS_FILE}")
  sp=$(jq -r '.swap_analysis.usage_percent//0' "${RESULTS_FILE}")
  si=$(jq -r '.swap_analysis.recent_swap_in_kb//0' "${RESULTS_FILE}")
  so=$(jq -r '.swap_analysis.recent_swap_out_kb//0' "${RESULTS_FILE}")
  psi=$(jq -r '.memory_pressure.psi_avg10//0' "${RESULTS_FILE}")
  tp=$(jq -r '.process_limits.threads.usage_percent//0' "${RESULTS_FILE}")
  fp=$(jq -r '.process_limits.file_descriptors.usage_percent//0' "${RESULTS_FILE}")
  local tm cm; tm=$(jq -r '.memory_overview.total_mb//1' "${RESULTS_FILE}")
  cm=$(jq -r '.memory_overview.cache_mb//0' "${RESULTS_FILE}")
  cr=$(awk -v c="${cm}" -v t="${tm}" 'BEGIN{if(t>0)printf "%.2f",c/t;else print "0"}')
  local risk="low"
  if awk "BEGIN{exit !(${mp}>${MEM_THRESHOLD})}" && [[ "$(sanitize_int "${si}")" -gt 0 || "$(sanitize_int "${so}")" -gt 0 ]]; then
    update_json '.exhaustion_patterns += ["memory_pressure_with_swapping"]'
    add_finding "Critical: High memory usage (${mp}%) with active swapping"; risk="high"
  fi
  if awk "BEGIN{exit !(${sp}>${SWAP_THRESHOLD})}"; then
    update_json '.exhaustion_patterns += ["high_swap_usage"]'
    add_finding "Warning: Swap usage at ${sp}% - performance degradation likely"
    [[ "${risk}" == "low" ]] && risk="medium"
  fi
  if awk "BEGIN{exit !(${cr}<0.2)}"; then
    update_json '.exhaustion_patterns += ["cache_pressure"]'
    add_finding "Cache under pressure - ratio ${cr} below threshold"
  fi
  if awk "BEGIN{exit !(${tp}>70)}"; then
    update_json '.exhaustion_patterns += ["thread_exhaustion_risk"]'
    add_finding "Thread usage at ${tp}% of maximum"; risk="high"
  fi
  if awk "BEGIN{exit !(${fp}>70)}"; then
    update_json '.exhaustion_patterns += ["fd_exhaustion_risk"]'
    add_finding "File descriptor usage at ${fp}% of maximum"; risk="high"
  fi
  jq --arg r "${risk}" '.risk_level=$r' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

generate_recommendations() {
  local mp sp sw
  mp=$(jq -r '.memory_overview.usage_percent//0' "${RESULTS_FILE}")
  sp=$(jq -r '.swap_analysis.usage_percent//0' "${RESULTS_FILE}")
  sw=$(jq -r '.system_tuning.swappiness//60' "${RESULTS_FILE}")
  awk "BEGIN{exit !(${mp}>80)}" && add_recommendation "High memory usage (${mp}%) - investigate top memory consumers"
  if awk "BEGIN{exit !(${sp}>60)}"; then
    add_recommendation "Critical swap usage (${sp}%) - identify and stop memory-intensive processes"
  elif awk "BEGIN{exit !(${sp}>${SWAP_THRESHOLD})}"; then
    add_recommendation "Elevated swap usage (${sp}%) - consider adding more RAM"
  fi
  if [[ "$(jq -r '.swap_analysis.active_swapping//false' "${RESULTS_FILE}")" == "true" ]]; then
    local si so; si=$(jq -r '.swap_analysis.recent_swap_in_kb//0' "${RESULTS_FILE}")
    so=$(jq -r '.swap_analysis.recent_swap_out_kb//0' "${RESULTS_FILE}")
    [[ "$(sanitize_int "${si}")" -gt 100 || "$(sanitize_int "${so}")" -gt 100 ]] && \
      add_recommendation "Active swap thrashing detected - reduce memory pressure immediately"
  fi
  [[ "$(sanitize_int "${sw}")" -gt 60 ]] && add_recommendation "Swappiness set to ${sw} - consider reducing to 10-30"
  local psi; psi=$(jq -r '.memory_pressure.psi_avg10//0' "${RESULTS_FILE}")
  awk "BEGIN{exit !(${psi}>20)}" && add_recommendation "High memory pressure (PSI ${psi}%) - review memory allocation"
}

# ── Mode dispatch ──────────────────────────────────────────────────────────
run_common() { detect_exhaustion_patterns; generate_recommendations; }
case "${MODE}" in
  full)       gather_memory_overview; analyze_swap; analyze_memory_pressure
              find_top_memory_consumers; find_swap_consumers; analyze_memory_growth
              inspect_container_memory; check_process_limits; check_system_tuning; run_common ;;
  memory)     gather_memory_overview; analyze_memory_pressure; find_top_memory_consumers
              analyze_memory_growth; inspect_container_memory; check_process_limits
              check_system_tuning; run_common ;;
  swap)       gather_memory_overview; analyze_swap; find_swap_consumers
              check_system_tuning; run_common ;;
  exhaustion) gather_memory_overview; analyze_swap; check_process_limits
              check_system_tuning; run_common ;;
  *)          echo "Unknown mode: ${MODE}. Use full|memory|swap|exhaustion" >&2; exit 1 ;;
esac

finalize_investigation
