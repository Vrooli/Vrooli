#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Comprehensive Container Health Monitor
# DESCRIPTION: Detects unhealthy containers, health check failures, and provides recovery recommendations
# CATEGORY: resource-management
# TRIGGERS: [container_unhealthy, health_check_failures, docker_errors]
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-16
# LAST_MODIFIED: 2025-09-16
# VERSION: 1.0

set -e

# Create results directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_DIR="../results/${TIMESTAMP}_container-health-comprehensive"
mkdir -p "$RESULT_DIR"

# Output file
OUTPUT_FILE="$RESULT_DIR/results.json"

echo "🔍 Starting Comprehensive Container Health Analysis..."

# Get unhealthy containers
echo "🚨 Checking unhealthy containers..."
UNHEALTHY_CONTAINERS=$(docker ps --filter health=unhealthy --format '{"name": "{{.Names}}", "status": "{{.Status}}", "id": "{{.ID}}"}' 2>/dev/null | jq -s '.' || echo '[]')

# Get health check failures from docker logs
echo "📊 Analyzing health check failures..."
HEALTH_CHECK_ERRORS=$(journalctl -xe --since "30 minutes ago" --no-pager 2>/dev/null | grep -i "health check.*error" | tail -20 | jq -Rs 'split("\n") | map(select(. != ""))' || echo '[]')

# Get container restart counts
echo "🔄 Checking container restarts..."
RESTART_COUNTS=$(docker ps --format '{{.Names}}' | while read container; do
    RESTARTS=$(docker inspect "$container" 2>/dev/null | jq -r '.[0].RestartCount // 0')
    if [ "$RESTARTS" -gt 0 ]; then
        echo "{\"container\": \"$container\", \"restart_count\": $RESTARTS}"
    fi
done | jq -s '.' || echo '[]')

# Check containers missing health checks
echo "⚠️ Finding containers without health checks..."
NO_HEALTH_CHECK=$(docker ps --format '{{.Names}}' | while read container; do
    HAS_HEALTH=$(docker inspect "$container" 2>/dev/null | jq -r '.[0].State.Health // null')
    if [ "$HAS_HEALTH" = "null" ]; then
        echo "\"$container\""
    fi
done | jq -s '.' || echo '[]')

# Get detailed info for unhealthy containers
echo "📋 Gathering detailed unhealthy container info..."
UNHEALTHY_DETAILS=$(echo "$UNHEALTHY_CONTAINERS" | jq -r '.[].id // empty' | while read id; do
    if [ -n "$id" ]; then
        docker inspect "$id" 2>/dev/null | jq '{
            name: .[0].Name,
            state: .[0].State.Status,
            health_status: .[0].State.Health.Status // "no_healthcheck",
            last_health_output: (.[0].State.Health.Log[0].Output // "none" | split("\n")[0]),
            exit_code: .[0].State.Health.Log[0].ExitCode // 0,
            restart_policy: .[0].HostConfig.RestartPolicy.Name,
            memory_limit: .[0].HostConfig.Memory,
            cpu_limit: .[0].HostConfig.CpuQuota
        }' 2>/dev/null || echo '{}'
    fi
done | jq -s 'map(select(. != {}))' || echo '[]')

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=()

if [ "$(echo "$UNHEALTHY_CONTAINERS" | jq 'length')" -gt 0 ]; then
    RECOMMENDATIONS+=("Multiple unhealthy containers detected - investigate health check commands")
    RECOMMENDATIONS+=("Consider implementing proper health endpoints in containerized services")
fi

if [ "$(echo "$RESTART_COUNTS" | jq '[.[] | select(.restart_count > 5)] | length')" -gt 0 ]; then
    RECOMMENDATIONS+=("Containers with high restart counts detected - check logs for crash patterns")
fi

if [ "$(echo "$NO_HEALTH_CHECK" | jq 'length')" -gt 10 ]; then
    RECOMMENDATIONS+=("Many containers lack health checks - consider adding HEALTHCHECK directives")
fi

# Check for specific known issues
if echo "$UNHEALTHY_DETAILS" | jq -e '.[] | select(.name | contains("mifos"))' >/dev/null 2>&1; then
    RECOMMENDATIONS+=("Mifos backend unhealthy - check database connectivity and API endpoints")
fi

if echo "$UNHEALTHY_DETAILS" | jq -e '.[] | select(.name | contains("superset"))' >/dev/null 2>&1; then
    RECOMMENDATIONS+=("Superset worker unhealthy - verify Redis connectivity and worker configuration")
fi

if echo "$HEALTH_CHECK_ERRORS" | grep -q "curl.*not found"; then
    RECOMMENDATIONS+=("Health check failures due to missing curl - install curl in container or use alternative health check method")
fi

# Convert recommendations to JSON array
RECOMMENDATIONS_JSON=$(printf '%s\n' "${RECOMMENDATIONS[@]}" | jq -Rs 'split("\n") | map(select(. != ""))')

# Generate final report
echo "📊 Compiling results..."
jq -n \
    --argjson unhealthy "$UNHEALTHY_CONTAINERS" \
    --argjson details "$UNHEALTHY_DETAILS" \
    --argjson restarts "$RESTART_COUNTS" \
    --argjson no_health "$NO_HEALTH_CHECK" \
    --argjson health_errors "$HEALTH_CHECK_ERRORS" \
    --argjson recommendations "$RECOMMENDATIONS_JSON" \
    '{
        investigation: "container-health-comprehensive",
        timestamp: now | strftime("%Y-%m-%dT%H:%M:%S%z"),
        summary: {
            unhealthy_count: ($unhealthy | length),
            containers_with_restarts: ($restarts | length),
            containers_without_healthcheck: ($no_health | length),
            health_check_errors: ($health_errors | length)
        },
        unhealthy_containers: $unhealthy,
        unhealthy_details: $details,
        restart_counts: $restarts,
        containers_without_healthcheck: $no_health,
        recent_health_errors: $health_errors,
        recommendations: $recommendations
    }' > "$OUTPUT_FILE"

echo "✅ Container health analysis complete! Results: $OUTPUT_FILE"

# Output results
cat "$OUTPUT_FILE"