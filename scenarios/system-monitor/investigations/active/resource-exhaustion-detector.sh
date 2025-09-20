#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Resource Exhaustion Pattern Detector
# DESCRIPTION: Detects resource exhaustion patterns including swap usage, memory pressure, and process limits
# CATEGORY: resource-management
# TRIGGERS: high_memory_usage, swap_usage, memory_pressure, process_exhaustion
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-18
# LAST_MODIFIED: 2025-09-18
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="resource-exhaustion-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_CMD="timeout 10"
HIGH_MEM_THRESHOLD=60
HIGH_SWAP_THRESHOLD=30
CACHE_RATIO_THRESHOLD=0.2

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results file
RESULTS_FILE="${OUTPUT_DIR}/results.json"

echo "🔍 Starting Resource Exhaustion Pattern Detection..."

# Initialize JSON structure
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "resource-exhaustion-detector",
  "timestamp": "",
  "memory_status": {},
  "swap_analysis": {},
  "cache_pressure": {},
  "top_memory_consumers": [],
  "process_limits": [],
  "exhaustion_patterns": [],
  "findings": [],
  "recommendations": []
}
EOF

# Update timestamp
TIMESTAMP=$(date -Iseconds)
jq ".timestamp = \"$TIMESTAMP\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Memory Status Deep Dive
echo "💾 Analyzing memory exhaustion patterns..."
MEM_INFO=$(free -m | awk 'NR==2')
TOTAL_MEM=$(echo "$MEM_INFO" | awk '{print $2}')
USED_MEM=$(echo "$MEM_INFO" | awk '{print $3}')
FREE_MEM=$(echo "$MEM_INFO" | awk '{print $4}')
SHARED_MEM=$(echo "$MEM_INFO" | awk '{print $5}')
CACHE_MEM=$(echo "$MEM_INFO" | awk '{print $6}')
AVAILABLE_MEM=$(echo "$MEM_INFO" | awk '{print $7}')

MEM_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($USED_MEM/$TOTAL_MEM)*100}")
CACHE_RATIO=$(awk "BEGIN {printf \"%.2f\", ($CACHE_MEM/$TOTAL_MEM)}")

jq ".memory_status = {
  \"total_mb\": $TOTAL_MEM,
  \"used_mb\": $USED_MEM,
  \"free_mb\": $FREE_MEM,
  \"shared_mb\": $SHARED_MEM,
  \"cache_mb\": $CACHE_MEM,
  \"available_mb\": $AVAILABLE_MEM,
  \"usage_percent\": $MEM_PERCENT,
  \"cache_ratio\": $CACHE_RATIO
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Swap Analysis
echo "🔄 Analyzing swap usage..."
SWAP_INFO=$(free -m | awk 'NR==3')
TOTAL_SWAP=$(echo "$SWAP_INFO" | awk '{print $2}')
USED_SWAP=$(echo "$SWAP_INFO" | awk '{print $3}')
FREE_SWAP=$(echo "$SWAP_INFO" | awk '{print $4}')
SWAP_PERCENT=0
if [ "$TOTAL_SWAP" -gt 0 ]; then
    SWAP_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($USED_SWAP/$TOTAL_SWAP)*100}")
fi

# Check swap I/O pressure
SWAP_IN=$(vmstat 1 2 | tail -1 | awk '{print $7}')
SWAP_OUT=$(vmstat 1 2 | tail -1 | awk '{print $8}')

jq ".swap_analysis = {
  \"total_mb\": $TOTAL_SWAP,
  \"used_mb\": $USED_SWAP,
  \"free_mb\": $FREE_SWAP,
  \"usage_percent\": $SWAP_PERCENT,
  \"swap_in_kb\": $SWAP_IN,
  \"swap_out_kb\": $SWAP_OUT,
  \"active_swapping\": $([ "$SWAP_IN" -gt 0 ] || [ "$SWAP_OUT" -gt 0 ] && echo "true" || echo "false")
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Cache Pressure Analysis
echo "📊 Analyzing cache pressure..."
DIRTY_KB=$(cat /proc/meminfo | grep "Dirty:" | awk '{print $2}')
WRITEBACK_KB=$(cat /proc/meminfo | grep "Writeback:" | awk '{print $2}')
SLAB_KB=$(cat /proc/meminfo | grep "^Slab:" | awk '{print $2}')
SRECLAIMABLE_KB=$(cat /proc/meminfo | grep "SReclaimable:" | awk '{print $2}')

CACHE_PRESSURE="low"
if (( $(echo "$CACHE_RATIO < $CACHE_RATIO_THRESHOLD" | bc -l) )); then
    CACHE_PRESSURE="high"
fi

jq ".cache_pressure = {
  \"dirty_mb\": $(($DIRTY_KB / 1024)),
  \"writeback_mb\": $(($WRITEBACK_KB / 1024)),
  \"slab_mb\": $(($SLAB_KB / 1024)),
  \"reclaimable_mb\": $(($SRECLAIMABLE_KB / 1024)),
  \"pressure_level\": \"$CACHE_PRESSURE\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Top Memory Consumers
echo "🎯 Identifying top memory consumers..."
TOP_CONSUMERS=()
while IFS= read -r line; do
    PID=$(echo "$line" | awk '{print $1}')
    RSS_KB=$(echo "$line" | awk '{print $2}')
    VSZ_KB=$(echo "$line" | awk '{print $3}')
    PCPU=$(echo "$line" | awk '{print $4}')
    PMEM=$(echo "$line" | awk '{print $5}')
    USER=$(echo "$line" | awk '{print $6}')
    CMD=$(echo "$line" | cut -d' ' -f7- | head -c 100 | sed 's/["\]/\\&/g')
    
    TOP_CONSUMERS+=("{
      \"pid\": $PID,
      \"rss_mb\": $(($RSS_KB / 1024)),
      \"vsz_mb\": $(($VSZ_KB / 1024)),
      \"cpu_percent\": $PCPU,
      \"mem_percent\": $PMEM,
      \"user\": \"$USER\",
      \"command\": \"$CMD\"
    }")
done < <(ps aux --sort=-rss | awk 'NR>1 && NR<=11 {print $2, $6, $5, $3, $4, $1, $11}')

if [ ${#TOP_CONSUMERS[@]} -gt 0 ]; then
    TOP_JSON=$(printf '%s\n' "${TOP_CONSUMERS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".top_memory_consumers = $TOP_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Process Limits Check
echo "🚫 Checking process limits..."
PROC_LIMITS=()

# Check system-wide limits
MAX_THREADS=$(cat /proc/sys/kernel/threads-max)
CURRENT_THREADS=$(ps -eLf | wc -l)
THREADS_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($CURRENT_THREADS/$MAX_THREADS)*100}")

MAX_FILES=$(cat /proc/sys/fs/file-max)
CURRENT_FILES=$(cat /proc/sys/fs/file-nr | awk '{print $1}')
FILES_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($CURRENT_FILES/$MAX_FILES)*100}")

PROC_LIMITS+=("{
  \"resource\": \"threads\",
  \"current\": $CURRENT_THREADS,
  \"maximum\": $MAX_THREADS,
  \"usage_percent\": $THREADS_PERCENT
}")

PROC_LIMITS+=("{
  \"resource\": \"file_descriptors\",
  \"current\": $CURRENT_FILES,
  \"maximum\": $MAX_FILES,
  \"usage_percent\": $FILES_PERCENT
}")

if [ ${#PROC_LIMITS[@]} -gt 0 ]; then
    LIMITS_JSON=$(printf '%s\n' "${PROC_LIMITS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".process_limits = $LIMITS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Identify Exhaustion Patterns
echo "🔬 Identifying exhaustion patterns..."
PATTERNS=()
FINDINGS=()
RECOMMENDATIONS=()

# Pattern 1: High memory with active swapping
if (( $(echo "$MEM_PERCENT > $HIGH_MEM_THRESHOLD" | bc -l) )) && [ "$SWAP_IN" -gt 0 -o "$SWAP_OUT" -gt 0 ]; then
    PATTERNS+=("\"memory_pressure_with_swapping\"")
    FINDINGS+=("\"Critical: High memory usage ($MEM_PERCENT%) with active swapping\"")
    RECOMMENDATIONS+=("\"Consider increasing system RAM or reducing memory-intensive workloads\"")
fi

# Pattern 2: High swap usage
if (( $(echo "$SWAP_PERCENT > $HIGH_SWAP_THRESHOLD" | bc -l) )); then
    PATTERNS+=("\"high_swap_usage\"")
    FINDINGS+=("\"Warning: Swap usage at $SWAP_PERCENT% - performance degradation likely\"")
    RECOMMENDATIONS+=("\"Identify and optimize memory-intensive processes\"")
fi

# Pattern 3: Low cache ratio
if [ "$CACHE_PRESSURE" = "high" ]; then
    PATTERNS+=("\"cache_pressure\"")
    FINDINGS+=("\"Cache under pressure - ratio $CACHE_RATIO below threshold\"")
    RECOMMENDATIONS+=("\"System may benefit from cache tuning or memory upgrade\"")
fi

# Pattern 4: Thread exhaustion risk
if (( $(echo "$THREADS_PERCENT > 70" | bc -l) )); then
    PATTERNS+=("\"thread_exhaustion_risk\"")
    FINDINGS+=("\"Thread usage at $THREADS_PERCENT% of maximum\"")
    RECOMMENDATIONS+=("\"Review applications spawning excessive threads\"")
fi

# Pattern 5: File descriptor pressure
if (( $(echo "$FILES_PERCENT > 70" | bc -l) )); then
    PATTERNS+=("\"fd_exhaustion_risk\"")
    FINDINGS+=("\"File descriptor usage at $FILES_PERCENT% of maximum\"")
    RECOMMENDATIONS+=("\"Check for file descriptor leaks in applications\"")
fi

# Update patterns in JSON
if [ ${#PATTERNS[@]} -gt 0 ]; then
    PATTERNS_JSON=$(printf '%s\n' "${PATTERNS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".exhaustion_patterns = $PATTERNS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update findings
if [ ${#FINDINGS[@]} -gt 0 ]; then
    FINDINGS_JSON=$(printf '%s\n' "${FINDINGS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".findings = $FINDINGS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update recommendations
if [ ${#RECOMMENDATIONS[@]} -gt 0 ]; then
    RECOMMENDATIONS_JSON=$(printf '%s\n' "${RECOMMENDATIONS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".recommendations = $RECOMMENDATIONS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Resource exhaustion detection complete!"
echo "📄 Results saved to: ${OUTPUT_DIR}/results.json"

# Output final results
cat "${RESULTS_FILE}" | jq .