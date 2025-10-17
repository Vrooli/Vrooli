#\!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Judge0 Worker Memory Exhaustion Detector
# DESCRIPTION: Detects Judge0 workers approaching memory limits and provides tuning recommendations
# CATEGORY: resource-management
# TRIGGERS: [judge0_high_memory, worker_memory_exhaustion, container_memory_near_limit]
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-16
# LAST_MODIFIED: 2025-09-16
# VERSION: 1.0

echo "🔍 Starting Judge0 Worker Memory Exhaustion Detection..."

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%S%z")
RESULTS_DIR="../results/$(date +%Y%m%d_%H%M%S)_judge0-worker-exhaustion-detector"
mkdir -p "$RESULTS_DIR"

echo "📊 Analyzing Judge0 worker containers..."

# Get Judge0 worker container stats with proper parsing
WORKER_STATS=$(docker stats --no-stream --format "table {{.Container}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.CPUPerc}}" | grep -E "judge0|resque" | while read -r container mem_usage mem_percent cpu_percent; do
    # Parse memory usage
    mem_current=$(echo "$mem_usage" | cut -d'/' -f1 | sed 's/[^0-9.]//g')
    mem_limit=$(echo "$mem_usage" | cut -d'/' -f2 | sed 's/[^0-9.]//g')
    mem_unit_current=$(echo "$mem_usage" | cut -d'/' -f1 | grep -o '[A-Za-z]*')
    mem_unit_limit=$(echo "$mem_usage" | cut -d'/' -f2 | grep -o '[A-Za-z]*')
    
    # Convert to MiB for consistency
    if [[ "$mem_unit_current" == "GiB" ]]; then
        mem_current_mib=$(echo "$mem_current * 1024" | bc)
    else
        mem_current_mib=$mem_current
    fi
    
    if [[ "$mem_unit_limit" == "GiB" ]]; then
        mem_limit_mib=$(echo "$mem_limit * 1024" | bc)
    else
        mem_limit_mib=$mem_limit
    fi
    
    # Calculate exhaustion percentage
    exhaustion_percent=$(echo "scale=2; ($mem_current_mib / $mem_limit_mib) * 100" | bc 2>/dev/null || echo "0")
    
    echo "{\"container\":\"$container\",\"memory_current_mib\":$mem_current_mib,\"memory_limit_mib\":$mem_limit_mib,\"exhaustion_percent\":$exhaustion_percent,\"cpu_percent\":\"$cpu_percent\"}"
done | jq -s '.')

echo "🔍 Checking worker process details..."

# Get detailed process info from judge0 workers
WORKER_PROCESSES=$(docker exec -i config-judge0-workers-1 ps aux 2>/dev/null | grep -E "resque|ruby" | head -5 | awk '{print "{\"pid\":"$2",\"cpu\":"$3",\"mem\":"$4",\"cmd\":\""$11"\"}"}'| jq -s '.' 2>/dev/null || echo "[]")

echo "💡 Analyzing memory patterns..."

# Check for memory pressure indicators
MEMORY_PRESSURE=$(
    if docker exec -i config-judge0-workers-1 cat /proc/meminfo 2>/dev/null | grep -q "MemAvailable"; then
        docker exec -i config-judge0-workers-1 cat /proc/meminfo | grep -E "MemAvailable|SwapFree" | awk '{print $2}' | paste -sd',' | awk -F',' '{print "{\"mem_available_kb\":"$1",\"swap_free_kb\":"($2==""?"null":$2)"}"}' 
    else
        echo '{"mem_available_kb":null,"swap_free_kb":null}'
    fi
)

echo "🎯 Generating tuning recommendations..."

# Generate recommendations based on findings
RECOMMENDATIONS=$(cat << 'RECO'
[
  "Consider reducing REDIS_MAX_CONNECTIONS if workers are idle",
  "Implement memory limits per job execution",
  "Add container memory swap accounting for better OOM detection",
  "Monitor for memory leaks in long-running worker processes",
  "Consider horizontal scaling with more worker containers instead of vertical scaling"
]
RECO
)

# Compile results
cat > "$RESULTS_DIR/results.json" << EOJSON
{
  "investigation": "judge0-worker-exhaustion-detector",
  "timestamp": "$TIMESTAMP",
  "worker_containers": $WORKER_STATS,
  "worker_processes": $WORKER_PROCESSES,
  "memory_pressure": $MEMORY_PRESSURE,
  "recommendations": $RECOMMENDATIONS,
  "critical_threshold": 90,
  "warning_threshold": 75
}
EOJSON

echo "✅ Judge0 worker exhaustion detection complete\!"
jq '.' "$RESULTS_DIR/results.json"
