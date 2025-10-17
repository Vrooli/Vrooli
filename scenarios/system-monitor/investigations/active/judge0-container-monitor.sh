#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Judge0 Container Health Monitor
# DESCRIPTION: Monitors judge0 worker containers for memory exhaustion and performance issues
# CATEGORY: resource-management
# TRIGGERS: container_memory_high, judge0_worker_issues, resque_workers_stuck
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-14
# LAST_MODIFIED: 2025-09-14
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="judge0-container-monitor"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30
MEMORY_THRESHOLD=95

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "judge0-container-monitor",
  "timestamp": "",
  "judge0_workers": [],
  "memory_analysis": {},
  "process_analysis": [],
  "block_io_analysis": {},
  "recommendations": [],
  "auto_fix_actions": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Judge0 Container Health Monitoring..."

# Check judge0 worker containers
echo "📊 Analyzing judge0 worker containers..."
WORKERS=$(docker ps --filter "name=judge0-worker" --format "{{.Names}}" 2>/dev/null || true)

if [ -z "$WORKERS" ]; then
    echo "No judge0 worker containers found."
    jq '.recommendations += ["No judge0 worker containers detected"]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
else
    for WORKER in $WORKERS; do
        echo "Checking $WORKER..."
        
        # Get container stats
        STATS=$(docker stats --no-stream --format "json" "$WORKER" 2>/dev/null || echo '{}')
        
        if [ "$STATS" != '{}' ]; then
            # Parse memory usage
            MEM_PERCENT=$(echo "$STATS" | jq -r '.MemPerc' | sed 's/%//')
            MEM_USAGE=$(echo "$STATS" | jq -r '.MemUsage')
            CPU_PERCENT=$(echo "$STATS" | jq -r '.CPUPerc' | sed 's/%//')
            BLOCK_IO=$(echo "$STATS" | jq -r '.BlockIO')
            NET_IO=$(echo "$STATS" | jq -r '.NetIO')
            
            # Check process count inside container
            PROCESS_COUNT=$(docker exec "$WORKER" ps aux 2>/dev/null | wc -l || echo "0")
            
            # Get resque worker status
            RESQUE_WORKERS=$(docker exec "$WORKER" ps aux 2>/dev/null | grep -c "resque" || echo "0")
            
            # Record findings
            jq --arg worker "$WORKER" \
               --arg mem_percent "$MEM_PERCENT" \
               --arg mem_usage "$MEM_USAGE" \
               --arg cpu_percent "$CPU_PERCENT" \
               --arg block_io "$BLOCK_IO" \
               --arg net_io "$NET_IO" \
               --arg process_count "$PROCESS_COUNT" \
               --arg resque_workers "$RESQUE_WORKERS" \
               '.judge0_workers += [{
                   "container": $worker,
                   "memory_percent": ($mem_percent | tonumber),
                   "memory_usage": $mem_usage,
                   "cpu_percent": ($cpu_percent | tonumber),
                   "block_io": $block_io,
                   "net_io": $net_io,
                   "process_count": ($process_count | tonumber),
                   "resque_workers": ($resque_workers | tonumber)
               }]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
            
            # Check if memory is critically high
            if (( $(echo "$MEM_PERCENT > $MEMORY_THRESHOLD" | bc -l) )); then
                echo "⚠️ High memory usage detected in $WORKER: ${MEM_PERCENT}%"
                
                # Check for memory leaks in resque workers
                LONG_RUNNING=$(docker exec "$WORKER" ps aux 2>/dev/null | grep "resque" | awk '$10 ~ /[0-9]+:[0-9]+/ {print $2, $10}' | head -5 || true)
                
                if [ -n "$LONG_RUNNING" ]; then
                    jq --arg worker "$WORKER" \
                       --arg processes "$LONG_RUNNING" \
                       '.memory_analysis[$worker] = {
                           "status": "critical",
                           "long_running_processes": $processes
                       }' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
                fi
                
                # Recommendation for high memory
                jq --arg worker "$WORKER" \
                   --arg mem "$MEM_PERCENT" \
                   '.recommendations += ["Container " + $worker + " at " + $mem + "% memory - consider restarting or increasing memory limit"]' \
                   "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
            fi
        fi
    done
fi

# Analyze block I/O patterns
echo "💾 Analyzing block I/O for judge0 containers..."
BLOCK_IO_STATS=$(docker stats --no-stream --format "table {{.Container}}\t{{.BlockIO}}" 2>/dev/null | grep -E "judge0|config-judge0" || true)

if [ -n "$BLOCK_IO_STATS" ]; then
    # Check for excessive I/O
    HIGH_IO=$(echo "$BLOCK_IO_STATS" | awk '$2 ~ /GB/ {print $1, $2}' || true)
    
    if [ -n "$HIGH_IO" ]; then
        jq --arg io "$HIGH_IO" '.block_io_analysis = {"high_io_containers": $io}' \
           "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
        
        jq '.recommendations += ["High block I/O detected in judge0 containers - may indicate excessive compilation or temp file usage"]' \
           "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
    fi
fi

# Check for stuck processes
echo "🔍 Checking for stuck resque workers..."
for WORKER in $WORKERS; do
    STUCK_WORKERS=$(docker exec "$WORKER" ps aux 2>/dev/null | grep "resque" | grep -E "Waiting for [0-9]+\.[0-9]+\.[0-9]+" | wc -l || echo "0")
    
    if [ "$STUCK_WORKERS" -gt "10" ]; then
        jq --arg worker "$WORKER" \
           --arg count "$STUCK_WORKERS" \
           '.process_analysis += [{
               "container": $worker,
               "stuck_workers": ($count | tonumber),
               "status": "degraded"
           }]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
        
        jq --arg worker "$WORKER" \
           --arg count "$STUCK_WORKERS" \
           '.recommendations += ["Container " + $worker + " has " + $count + " stuck resque workers - restart recommended"]' \
           "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
    fi
done

# Final recommendations
jq '.recommendations += [
    "Monitor judge0 worker memory usage regularly",
    "Consider implementing memory limits per compilation job",
    "Set up automatic container restart on memory exhaustion",
    "Review code submission patterns for memory-intensive operations"
]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "✅ Judge0 container monitoring complete!"

# Output results
cat "${RESULTS_FILE}" | jq '.'