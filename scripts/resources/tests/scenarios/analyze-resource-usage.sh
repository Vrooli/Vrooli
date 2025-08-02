#!/bin/bash
# Script to analyze resource usage patterns in test scenarios

echo "=== Resource Usage Analysis ==="
echo

# Function to extract resources from a test file
extract_resources() {
    local file="$1"
    grep "^# @services:" "$file" 2>/dev/null | sed 's/# @services: //' | tr ',' ' '
}

# Count total scenarios
total_scenarios=$(find . -name "*.test.sh" -type f | wc -l)
echo "Total test scenarios: $total_scenarios"
echo

# List all available resources
echo "=== Available Resources ==="
all_resources=(
    "agent-s2" "browserless" "claude-code"
    "ollama" "unstructured-io" "whisper"
    "comfyui" "huginn" "n8n" "node-red" "windmill"
    "minio" "postgres" "qdrant" "questdb" "redis" "vault"
    "searxng" "judge0"
)
echo "Total available: ${#all_resources[@]}"
printf '%s\n' "${all_resources[@]}" | sort
echo

# Analyze resource usage
echo "=== Resource Usage Frequency ==="
{
    for file in $(find . -name "*.test.sh" -type f); do
        extract_resources "$file"
    done
} | tr ' ' '\n' | sort | uniq -c | sort -nr
echo

# Find unused resources
echo "=== Unused Resources ==="
used_resources=$(
    for file in $(find . -name "*.test.sh" -type f); do
        extract_resources "$file"
    done | tr ' ' '\n' | sort | uniq
)

for resource in "${all_resources[@]}"; do
    if ! echo "$used_resources" | grep -q "^$resource$"; then
        echo "- $resource"
    fi
done
echo

# Analyze resource combinations
echo "=== Resource Combinations by Frequency ==="
{
    for file in $(find . -name "*.test.sh" -type f); do
        services=$(extract_resources "$file")
        if [ -n "$services" ]; then
            echo "$services" | tr ' ' ',' | sed 's/,/, /g'
        fi
    done
} | sort | uniq -c | sort -nr
echo

# Show which scenarios use each resource
echo "=== Scenarios by Resource ==="
for resource in "${all_resources[@]}"; do
    scenarios=$(grep -l "# @services:.*$resource" $(find . -name "*.test.sh" -type f) 2>/dev/null)
    if [ -n "$scenarios" ]; then
        echo
        echo "[$resource]"
        echo "$scenarios" | while read -r scenario; do
            echo "  - $(basename "$(dirname "$scenario")")/$(basename "$scenario")"
        done
    fi
done

# Calculate coverage statistics
echo
echo "=== Coverage Statistics ==="
used_count=$(echo "$used_resources" | wc -l)
total_count=${#all_resources[@]}
coverage_percent=$((used_count * 100 / total_count))
echo "Resources with tests: $used_count/$total_count ($coverage_percent%)"

# Find most and least integrated resources
echo
echo "=== Integration Complexity ==="
echo "Scenarios with most resources:"
for file in $(find . -name "*.test.sh" -type f); do
    services=$(extract_resources "$file")
    if [ -n "$services" ]; then
        count=$(echo "$services" | wc -w)
        echo "$count resources: $(basename "$(dirname "$file")")/$(basename "$file")"
    fi
done | sort -nr | head -5

echo
echo "Average resources per scenario:"
total_resources=0
scenario_count=0
for file in $(find . -name "*.test.sh" -type f); do
    services=$(extract_resources "$file")
    if [ -n "$services" ]; then
        count=$(echo "$services" | wc -w)
        total_resources=$((total_resources + count))
        scenario_count=$((scenario_count + 1))
    fi
done
if [ $scenario_count -gt 0 ]; then
    avg=$((total_resources * 10 / scenario_count))
    echo "$(echo "scale=1; $avg / 10" | bc) resources per scenario"
fi