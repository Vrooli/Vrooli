#\!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Container Resource Optimizer
# DESCRIPTION: Analyzes container resource usage vs limits and provides optimization recommendations
# CATEGORY: resource-management
# TRIGGERS: [container_memory_high, resource_inefficiency, container_limits_unset]
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-16
# LAST_MODIFIED: 2025-09-16
# VERSION: 1.0

echo "🔍 Starting Container Resource Optimization Analysis..."

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%S%z")
RESULTS_DIR="../results/$(date +%Y%m%d_%H%M%S)_container-resource-optimizer"
mkdir -p "$RESULTS_DIR"

echo "📊 Analyzing container resource usage vs limits..."

# Analyze containers with memory limits approaching usage
OVERPROVISIONED=$(docker stats --no-stream --format "{{.Container}}|{{.MemUsage}}|{{.MemPerc}}|{{.CPUPerc}}" | while IFS='|' read -r container mem_usage mem_percent cpu_percent; do
    mem_percent_val=$(echo "$mem_percent" | sed 's/%//')
    
    # Check if memory usage is very low (under-provisioned)
    if (( $(echo "$mem_percent_val < 10" | bc -l) )); then
        echo "{\"container\":\"$container\",\"type\":\"overprovisioned\",\"mem_usage\":\"$mem_usage\",\"mem_percent\":\"$mem_percent\"}"
    # Check if memory usage is very high (under-provisioned)
    elif (( $(echo "$mem_percent_val > 85" | bc -l) )); then
        echo "{\"container\":\"$container\",\"type\":\"underprovisioned\",\"mem_usage\":\"$mem_usage\",\"mem_percent\":\"$mem_percent\"}"
    fi
done | jq -s '.')

echo "🔍 Checking for containers without limits..."

# Find containers without memory limits
UNLIMITED=$(docker ps -q | while read -r container_id; do
    limits=$(docker inspect "$container_id" --format '{{.HostConfig.Memory}}')
    name=$(docker inspect "$container_id" --format '{{.Name}}' | sed 's/^\///')
    if [ "$limits" = "0" ]; then
        echo "{\"container\":\"$name\",\"memory_limit\":\"unlimited\",\"risk\":\"high\"}"
    fi
done | jq -s '.')

echo "💡 Generating optimization recommendations..."

# Compile results
cat > "$RESULTS_DIR/results.json" << EOJSON
{
  "investigation": "container-resource-optimizer",
  "timestamp": "$TIMESTAMP",
  "resource_inefficiencies": $OVERPROVISIONED,
  "unlimited_containers": $UNLIMITED,
  "recommendations": [
    "Set memory limits for all containers to prevent resource exhaustion",
    "Reduce memory limits for overprovisioned containers to improve efficiency",
    "Increase memory limits for underprovisioned containers to prevent OOM",
    "Consider using memory reservation instead of hard limits for elastic workloads",
    "Enable memory swap accounting for better container isolation"
  ]
}
EOJSON

echo "✅ Container resource optimization analysis complete\!"
jq '.' "$RESULTS_DIR/results.json"
