#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Judge0 Memory Optimizer
# DESCRIPTION: Detects and optimizes Judge0 worker memory exhaustion patterns
# CATEGORY: resource-management
# TRIGGERS: [judge0_memory_high, worker_exhaustion, container_oom]
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-15
# LAST_MODIFIED: 2025-09-15
# VERSION: 1.0

set -euo pipefail

# Output directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_DIR="../results/${TIMESTAMP}_judge0-memory-optimizer"
mkdir -p "$RESULT_DIR"

echo "🔍 Starting Judge0 Memory Optimization Analysis..."

# Function to get container memory stats
get_container_stats() {
    docker stats --no-stream --format "table {{.Container}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.CPUPerc}}" 2>/dev/null | \
        grep -E "judge0|resque" || echo ""
}

# Function to analyze worker processes
analyze_workers() {
    local worker_data="[]"
    for container in $(docker ps --filter "name=judge0-workers" --format "{{.Names}}"); do
        local proc_count=$(docker exec "$container" ps aux 2>/dev/null | grep -c ruby || echo 0)
        local mem_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "$container" 2>/dev/null || echo "unknown")
        local mem_perc=$(docker stats --no-stream --format "{{.MemPerc}}" "$container" 2>/dev/null || echo "0%")
        
        worker_data=$(echo "$worker_data" | jq ". += [{
            \"container\": \"$container\",
            \"process_count\": $proc_count,
            \"memory_usage\": \"$mem_usage\",
            \"memory_percent\": \"$mem_perc\"
        }]")
    done
    echo "$worker_data"
}

# Function to check for memory leaks
check_memory_patterns() {
    local patterns="[]"
    
    # Check for containers near memory limit
    docker stats --no-stream --format "{{.Container}}\t{{.MemPerc}}" 2>/dev/null | while IFS=$'\t' read -r container mem_perc; do
        if [[ "$container" == *"judge0"* ]]; then
            mem_value=${mem_perc%\%}
            if (( $(echo "$mem_value > 90" | bc -l) )); then
                patterns=$(echo "$patterns" | jq ". += [{
                    \"type\": \"high_memory\",
                    \"container\": \"$container\",
                    \"memory_percent\": \"$mem_perc\",
                    \"severity\": \"critical\"
                }]")
            fi
        fi
    done
    
    echo "$patterns"
}

# Function to generate optimization recommendations
generate_optimizations() {
    local recommendations="[]"
    
    # Check current memory limits
    for container in $(docker ps --filter "name=judge0" --format "{{.Names}}"); do
        local mem_limit=$(docker inspect "$container" 2>/dev/null | jq -r '.[0].HostConfig.Memory // 0')
        local current_usage=$(docker stats --no-stream --format "{{.MemPerc}}" "$container" 2>/dev/null | sed 's/%//')
        
        if [[ "$mem_limit" != "0" ]] && (( $(echo "$current_usage > 95" | bc -l) )); then
            recommendations=$(echo "$recommendations" | jq ". += [
                \"Container $container at ${current_usage}% memory - consider increasing limit or reducing workers\",
                \"Current limit: $(( mem_limit / 1073741824 ))GB\"
            ]")
        fi
    done
    
    # Check for zombie processes in workers
    for container in $(docker ps --filter "name=judge0-workers" --format "{{.Names}}"); do
        local zombie_count=$(docker exec "$container" ps aux 2>/dev/null | grep -c "<defunct>" || echo 0)
        if [[ "$zombie_count" -gt 0 ]]; then
            recommendations=$(echo "$recommendations" | jq ". += [
                \"Found $zombie_count zombie processes in $container - restart recommended\"
            ]")
        fi
    done
    
    # General recommendations based on patterns
    recommendations=$(echo "$recommendations" | jq ". += [
        \"Monitor worker queue depth to prevent memory buildup\",
        \"Consider implementing worker process recycling after N jobs\",
        \"Review STACK_LIMIT and MEMORY_LIMIT environment variables\"
    ]")
    
    echo "$recommendations"
}

echo "📊 Gathering Judge0 container stats..."
container_stats=$(get_container_stats)

echo "🔍 Analyzing worker processes..."
worker_analysis=$(analyze_workers)

echo "🧪 Checking memory patterns..."
memory_patterns=$(check_memory_patterns)

echo "💡 Generating optimization recommendations..."
optimizations=$(generate_optimizations)

# Compile results
results=$(jq -n \
    --arg investigation "judge0-memory-optimizer" \
    --arg timestamp "$(date -Iseconds)" \
    --argjson container_stats "$(echo "$container_stats" | jq -Rs '.')" \
    --argjson worker_analysis "$worker_analysis" \
    --argjson memory_patterns "$memory_patterns" \
    --argjson optimizations "$optimizations" \
    '{
        investigation: $investigation,
        timestamp: $timestamp,
        container_overview: {
            raw_stats: $container_stats
        },
        worker_analysis: $worker_analysis,
        memory_patterns: $memory_patterns,
        optimizations: $optimizations,
        auto_fix_available: true,
        fix_commands: [
            "docker restart config-judge0-workers-1",
            "docker restart config-judge0-workers-2",
            "docker exec config-judge0-workers-1 kill -HUP 1",
            "docker exec config-judge0-workers-2 kill -HUP 1"
        ]
    }')

# Save results
echo "$results" | jq '.' > "$RESULT_DIR/results.json"

echo "✅ Judge0 memory optimization analysis complete!"
echo "📁 Results saved to: $RESULT_DIR/results.json"

# Output results for immediate use
echo "$results" | jq '.'