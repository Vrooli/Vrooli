#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Swap and Memory Pressure Deep Analyzer
# DESCRIPTION: Comprehensive analysis of swap usage, memory pressure, and process behavior patterns
# CATEGORY: resource-management
# TRIGGERS: high_swap_usage, memory_pressure, performance_degradation
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-18
# LAST_MODIFIED: 2025-09-18
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="swap-memory-pressure-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_CMD="timeout 10"

# Create output directory
mkdir -p "${OUTPUT_DIR}"
RESULTS_FILE="${OUTPUT_DIR}/results.json"

echo "🔍 Starting Swap and Memory Pressure Deep Analysis..."

# Initialize JSON structure
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "swap-memory-pressure-analyzer",
  "timestamp": "",
  "memory_overview": {},
  "swap_analysis": {},
  "memory_pressure": {},
  "swap_consuming_processes": [],
  "memory_growth_patterns": [],
  "system_tuning": {},
  "findings": [],
  "recommendations": [],
  "risk_level": "low"
}
EOF

# Timestamp
TIMESTAMP=$(date -Iseconds)
jq ".timestamp = \"$TIMESTAMP\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Memory Overview
echo "💾 Analyzing memory distribution..."
MEM_INFO=$(free -m | awk 'NR==2{print $2, $3, $4, $6, $7}')
read -r TOTAL_MEM USED_MEM FREE_MEM CACHE_MEM AVAIL_MEM <<< "$MEM_INFO"
MEM_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($USED_MEM/$TOTAL_MEM)*100}")

jq ".memory_overview = {
  \"total_mb\": $TOTAL_MEM,
  \"used_mb\": $USED_MEM,
  \"free_mb\": $FREE_MEM,
  \"cache_mb\": $CACHE_MEM,
  \"available_mb\": $AVAIL_MEM,
  \"usage_percent\": $MEM_PERCENT
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Swap Analysis
echo "🔄 Analyzing swap usage patterns..."
SWAP_INFO=$(free -m | awk 'NR==3{print $2, $3, $4}')
read -r TOTAL_SWAP USED_SWAP FREE_SWAP <<< "$SWAP_INFO"
SWAP_PERCENT=$(awk "BEGIN {if($TOTAL_SWAP > 0) printf \"%.1f\", ($USED_SWAP/$TOTAL_SWAP)*100; else print \"0.0\"}")

# Check swap in/out activity
SWAP_IN=$(vmstat -s | grep "pages swapped in" | awk '{print $1}')
SWAP_OUT=$(vmstat -s | grep "pages swapped out" | awk '{print $1}')

# Check recent swap activity (last 2 seconds)
VMSTAT_OUTPUT=$(vmstat 1 2 | tail -1)
RECENT_SWAP_IN=$(echo "$VMSTAT_OUTPUT" | awk '{print $7}')
RECENT_SWAP_OUT=$(echo "$VMSTAT_OUTPUT" | awk '{print $8}')

jq ".swap_analysis = {
  \"total_mb\": $TOTAL_SWAP,
  \"used_mb\": $USED_SWAP,
  \"free_mb\": $FREE_SWAP,
  \"usage_percent\": $SWAP_PERCENT,
  \"total_swap_in_pages\": $SWAP_IN,
  \"total_swap_out_pages\": $SWAP_OUT,
  \"recent_swap_in_kb\": $RECENT_SWAP_IN,
  \"recent_swap_out_kb\": $RECENT_SWAP_OUT,
  \"active_swapping\": $([ "$RECENT_SWAP_IN" -gt 0 ] || [ "$RECENT_SWAP_OUT" -gt 0 ] && echo "true" || echo "false")
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Memory Pressure Analysis
echo "📊 Checking memory pressure indicators..."
if [ -f /proc/pressure/memory ]; then
  PSI_MEMORY=$(cat /proc/pressure/memory | grep -E "^some" | awk '{print $2, $3, $4, $5}')
  PSI_AVG10=$(echo "$PSI_MEMORY" | cut -d' ' -f1 | cut -d'=' -f2)
  PSI_AVG60=$(echo "$PSI_MEMORY" | cut -d' ' -f2 | cut -d'=' -f2)
  PSI_AVG300=$(echo "$PSI_MEMORY" | cut -d' ' -f3 | cut -d'=' -f2)
  PSI_TOTAL=$(echo "$PSI_MEMORY" | cut -d' ' -f4 | cut -d'=' -f2)
else
  PSI_AVG10=0
  PSI_AVG60=0
  PSI_AVG300=0
  PSI_TOTAL=0
fi

# Check cache pressure
DIRTY_KB=$(cat /proc/meminfo | grep "^Dirty:" | awk '{print $2}')
WRITEBACK_KB=$(cat /proc/meminfo | grep "^Writeback:" | awk '{print $2}')
CACHE_PRESSURE=$((DIRTY_KB + WRITEBACK_KB))

jq ".memory_pressure = {
  \"psi_avg10\": $PSI_AVG10,
  \"psi_avg60\": $PSI_AVG60,
  \"psi_avg300\": $PSI_AVG300,
  \"psi_total_us\": $PSI_TOTAL,
  \"dirty_kb\": $DIRTY_KB,
  \"writeback_kb\": $WRITEBACK_KB,
  \"cache_pressure_kb\": $CACHE_PRESSURE
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Find swap-consuming processes
echo "🔍 Identifying swap-consuming processes..."
SWAP_PROCS=()
COUNT=0
while IFS= read -r line; do
  if [ -n "$line" ] && [ "$COUNT" -lt 10 ]; then
    PID=$(echo "$line" | awk '{print $1}')
    SWAP_KB=$(echo "$line" | awk '{print $2}')
    if [ -d "/proc/$PID" ]; then
      CMD=$(cat /proc/$PID/comm 2>/dev/null || echo "unknown")
      USER=$(ps -p "$PID" -o user= 2>/dev/null || echo "unknown")
      RSS_KB=$(cat /proc/$PID/status 2>/dev/null | grep "^VmRSS:" | awk '{print $2}' || echo "0")
      VSZ_KB=$(cat /proc/$PID/status 2>/dev/null | grep "^VmSize:" | awk '{print $2}' || echo "0")
      
      SWAP_PROCS+=("{
        \"pid\": $PID,
        \"swap_mb\": $((SWAP_KB / 1024)),
        \"rss_mb\": $((RSS_KB / 1024)),
        \"vsz_mb\": $((VSZ_KB / 1024)),
        \"command\": \"$CMD\",
        \"user\": \"$USER\"
      }")
      COUNT=$((COUNT + 1))
    fi
  fi
done < <(for file in /proc/*/status; do
  [ -r "$file" ] || continue
  PID=$(basename $(dirname "$file"))
  SWAP=$(grep "^VmSwap:" "$file" 2>/dev/null | awk '{print $2}')
  [ -n "$SWAP" ] && [ "$SWAP" -gt 0 ] && echo "$PID $SWAP"
done | sort -k2 -rn)

if [ ${#SWAP_PROCS[@]} -gt 0 ]; then
  SWAP_JSON=$(printf '%s\n' "${SWAP_PROCS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
  jq ".swap_consuming_processes = $SWAP_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Analyze memory growth patterns (top growing processes)
echo "📈 Analyzing memory growth patterns..."
GROWTH_PROCS=()
COUNT=0
for pid in $(ps aux --sort=-%mem | awk 'NR>1 && NR<=11 {print $2}'); do
  if [ -f "/proc/$pid/status" ]; then
    COMM=$(cat /proc/$pid/comm 2>/dev/null || echo "unknown")
    RSS_KB=$(grep "^VmRSS:" /proc/$pid/status 2>/dev/null | awk '{print $2}' || echo "0")
    PEAK_KB=$(grep "^VmPeak:" /proc/$pid/status 2>/dev/null | awk '{print $2}' || echo "0")
    
    if [ "$PEAK_KB" -gt 0 ] && [ "$RSS_KB" -gt 0 ]; then
      GROWTH_PERCENT=$(awk "BEGIN {printf \"%.1f\", (($PEAK_KB - $RSS_KB) / $PEAK_KB) * 100}")
      GROWTH_PROCS+=("{
        \"pid\": $pid,
        \"command\": \"$COMM\",
        \"current_mb\": $((RSS_KB / 1024)),
        \"peak_mb\": $((PEAK_KB / 1024)),
        \"growth_potential_percent\": $GROWTH_PERCENT
      }")
    fi
  fi
done

if [ ${#GROWTH_PROCS[@]} -gt 0 ]; then
  GROWTH_JSON=$(printf '%s\n' "${GROWTH_PROCS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
  jq ".memory_growth_patterns = $GROWTH_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# System Tuning Parameters
echo "⚙️ Checking system tuning parameters..."
SWAPPINESS=$(cat /proc/sys/vm/swappiness)
CACHE_PRESSURE=$(cat /proc/sys/vm/vfs_cache_pressure)
DIRTY_RATIO=$(cat /proc/sys/vm/dirty_ratio)
DIRTY_BG_RATIO=$(cat /proc/sys/vm/dirty_background_ratio)
MIN_FREE_KB=$(cat /proc/sys/vm/min_free_kbytes)

jq ".system_tuning = {
  \"swappiness\": $SWAPPINESS,
  \"vfs_cache_pressure\": $CACHE_PRESSURE,
  \"dirty_ratio\": $DIRTY_RATIO,
  \"dirty_background_ratio\": $DIRTY_BG_RATIO,
  \"min_free_kbytes\": $MIN_FREE_KB
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate Findings and Recommendations
echo "💡 Generating findings and recommendations..."
FINDINGS=()
RECOMMENDATIONS=()
RISK_LEVEL="low"

# Check swap usage
if (( $(echo "$SWAP_PERCENT > 60" | bc -l) )); then
  FINDINGS+=("\"Critical: Swap usage at ${SWAP_PERCENT}% - severe performance impact\"")
  RECOMMENDATIONS+=("\"Immediate action needed: Identify and stop memory-intensive processes\"")
  RISK_LEVEL="high"
elif (( $(echo "$SWAP_PERCENT > 30" | bc -l) )); then
  FINDINGS+=("\"Warning: Swap usage at ${SWAP_PERCENT}% - performance degradation likely\"")
  RECOMMENDATIONS+=("\"Consider adding more RAM or optimizing memory-intensive applications\"")
  RISK_LEVEL="medium"
fi

# Check active swapping
if [ "$RECENT_SWAP_IN" -gt 100 ] || [ "$RECENT_SWAP_OUT" -gt 100 ]; then
  FINDINGS+=("\"Active swapping detected - system is thrashing\"")
  RECOMMENDATIONS+=("\"Reduce memory pressure immediately to restore performance\"")
  RISK_LEVEL="high"
fi

# Check memory pressure
if (( $(echo "$PSI_AVG10 > 20" | bc -l) )); then
  FINDINGS+=("\"High memory pressure detected (PSI: ${PSI_AVG10}%)\"")
  RECOMMENDATIONS+=("\"System experiencing memory stalls - review memory allocation\"")
  [ "$RISK_LEVEL" == "low" ] && RISK_LEVEL="medium"
fi

# Check swappiness
if [ "$SWAPPINESS" -gt 60 ]; then
  FINDINGS+=("\"Swappiness set to $SWAPPINESS - aggressive swapping enabled\"")
  RECOMMENDATIONS+=("\"Consider reducing swappiness to 10-30 for better performance: echo 30 > /proc/sys/vm/swappiness\"")
fi

# Check for swap-heavy processes
SWAP_COUNT=$(jq '.swap_consuming_processes | length' "${RESULTS_FILE}")
if [ "$SWAP_COUNT" -gt 5 ]; then
  FINDINGS+=("\"Multiple processes ($SWAP_COUNT) are using swap heavily\"")
  RECOMMENDATIONS+=("\"Review swap-consuming processes and restart memory-intensive services\"")
fi

# Update findings
if [ ${#FINDINGS[@]} -gt 0 ]; then
  FINDINGS_JSON=$(printf '%s\n' "${FINDINGS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
  jq ".findings = $FINDINGS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update recommendations
if [ ${#RECOMMENDATIONS[@]} -gt 0 ]; then
  RECS_JSON=$(printf '%s\n' "${RECOMMENDATIONS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
  jq ".recommendations = $RECS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update risk level
jq ".risk_level = \"$RISK_LEVEL\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Output results
echo "✅ Swap and memory pressure analysis complete!"
echo "📄 Results saved to: ${OUTPUT_DIR}/results.json"
cat "${RESULTS_FILE}" | jq '.'