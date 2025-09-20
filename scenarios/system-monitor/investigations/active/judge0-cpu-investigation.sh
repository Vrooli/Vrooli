#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Judge0 High CPU Investigation
# DESCRIPTION: Investigates high CPU usage from Judge0 worker containers and Redis interactions
# CATEGORY: performance
# TRIGGERS: judge0_high_cpu, worker_cpu_overload, redis_high_cpu
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-16
# LAST_MODIFIED: 2025-09-16
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="judge0-cpu-investigation"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "judge0-cpu-investigation",
  "timestamp": "",
  "summary": {},
  "worker_analysis": [],
  "process_details": [],
  "redis_analysis": {},
  "patterns": {},
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Judge0 High CPU Investigation..."

# Analyze Judge0 worker CPU usage
echo "📊 Analyzing Judge0 worker CPU usage patterns..."
JUDGE0_WORKERS=$(docker ps --filter "name=judge0" --format "{{.Names}}" 2>/dev/null || true)

TOTAL_CPU=0
WORKER_COUNT=0
WORKER_DATA="["

for worker in $JUDGE0_WORKERS; do
    echo "  Analyzing worker: $worker"
    
    # Get container stats
    STATS=$(docker stats "$worker" --no-stream --format "{{json .}}" 2>/dev/null || echo "{}")
    
    # Extract CPU and memory info
    CPU_PCT=$(echo "$STATS" | jq -r '.CPUPerc // "0%"' | sed 's/%//')
    MEM_USAGE=$(echo "$STATS" | jq -r '.MemUsage // "0MiB / 0MiB"')
    MEM_PCT=$(echo "$STATS" | jq -r '.MemPerc // "0%"' | sed 's/%//')
    
    # Get process information inside container
    PROC_COUNT=$(docker exec "$worker" sh -c 'ps aux | wc -l' 2>/dev/null || echo "0")
    
    # Check for Resque workers (Judge0 uses Resque)
    RESQUE_COUNT=$(docker exec "$worker" sh -c 'ps aux | grep -c resque' 2>/dev/null || echo "0")
    
    WORKER_DATA="${WORKER_DATA}{
        \"name\": \"$worker\",
        \"cpu_percent\": $CPU_PCT,
        \"memory_usage\": \"$MEM_USAGE\",
        \"memory_percent\": $MEM_PCT,
        \"process_count\": $PROC_COUNT,
        \"resque_workers\": $RESQUE_COUNT
    },"
    
    TOTAL_CPU=$(echo "$TOTAL_CPU + $CPU_PCT" | bc)
    WORKER_COUNT=$((WORKER_COUNT + 1))
done

WORKER_DATA=$(echo "$WORKER_DATA" | sed 's/,$/]/')
jq ".worker_analysis = $WORKER_DATA" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Check for busy Resque workers
echo "🔎 Checking for busy Resque worker processes..."
RESQUE_PROCS=$(ps aux | grep -i resque | grep -v grep || true)
if [ -n "$RESQUE_PROCS" ]; then
    RESQUE_DATA=$(echo "$RESQUE_PROCS" | awk '{print "{\"pid\": " $2 ", \"cpu\": " $3 ", \"mem\": " $4 ", \"cmd\": \"" substr($0, index($0,$11)) "\"},"}' | sed '$ s/,$//')
    jq ".process_details = [$RESQUE_DATA]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Analyze Redis usage (Judge0 uses Redis for job queue)
echo "📈 Analyzing Redis queue status..."
REDIS_CONTAINERS=$(docker ps --filter "name=redis" --filter "name=judge0" --format "{{.Names}}" 2>/dev/null || true)

for redis in $REDIS_CONTAINERS; do
    echo "  Checking Redis: $redis"
    
    # Get queue sizes
    QUEUE_SIZE=$(docker exec "$redis" redis-cli LLEN resque:queue:default 2>/dev/null || echo "0")
    FAILED_SIZE=$(docker exec "$redis" redis-cli LLEN resque:failed 2>/dev/null || echo "0")
    
    # Get Redis CPU usage
    REDIS_CPU=$(docker stats "$redis" --no-stream --format "{{.CPUPerc}}" 2>/dev/null | sed 's/%//' || echo "0")
    
    jq ".redis_analysis = {
        \"container\": \"$redis\",
        \"queue_size\": $QUEUE_SIZE,
        \"failed_jobs\": $FAILED_SIZE,
        \"cpu_percent\": $REDIS_CPU
    }" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
done

# Analyze patterns
echo "🔍 Analyzing CPU usage patterns..."
AVG_CPU=0
if [ $WORKER_COUNT -gt 0 ]; then
    AVG_CPU=$(echo "scale=2; $TOTAL_CPU / $WORKER_COUNT" | bc)
fi

jq ".patterns = {
    \"total_judge0_cpu\": $TOTAL_CPU,
    \"average_worker_cpu\": $AVG_CPU,
    \"worker_count\": $WORKER_COUNT,
    \"high_cpu_detected\": $([ $(echo "$TOTAL_CPU > 20" | bc) -eq 1 ] && echo "true" || echo "false")
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

if [ $(echo "$TOTAL_CPU > 20" | bc) -eq 1 ]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"High total CPU usage ($TOTAL_CPU%) from Judge0 workers - consider scaling horizontally\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Review code submission patterns for infinite loops or resource-intensive operations\","
fi

if [ $(echo "$AVG_CPU > 10" | bc) -eq 1 ]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Average worker CPU is high ($AVG_CPU%) - check for stuck jobs or inefficient code executions\","
fi

# Check Redis queue
QUEUE_SIZE=$(jq -r '.redis_analysis.queue_size // 0' "${RESULTS_FILE}")
if [ "$QUEUE_SIZE" -gt 100 ]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Large Redis queue ($QUEUE_SIZE jobs) - workers may be overwhelmed\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider increasing worker pool size or optimizing job processing\","
fi

FAILED_SIZE=$(jq -r '.redis_analysis.failed_jobs // 0' "${RESULTS_FILE}")
if [ "$FAILED_SIZE" -gt 10 ]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Multiple failed jobs in queue ($FAILED_SIZE) - investigate failure reasons\","
fi

# Add general recommendations
RECOMMENDATIONS="${RECOMMENDATIONS}\"Monitor submission patterns for abuse or inefficient code\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Implement CPU and time limits for code executions\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider using Judge0 isolate for better resource control\","

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "✅ Judge0 CPU investigation complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"