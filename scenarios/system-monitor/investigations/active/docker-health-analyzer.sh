#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Docker Container Health Analyzer
# DESCRIPTION: Analyzes Docker container health issues and provides detailed diagnostics
# CATEGORY: resource-management
# TRIGGERS: container_unhealthy, docker_errors, health_check_failures
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-14
# LAST_MODIFIED: 2025-09-14
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="docker-health-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=45

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "docker-health-analyzer",
  "timestamp": "",
  "summary": {},
  "unhealthy_containers": [],
  "container_errors": [],
  "health_check_failures": [],
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Docker Container Health Analysis..."

# Summary of Docker health
echo "📊 Gathering Docker health summary..."
TOTAL_CONTAINERS=$(docker ps -q | wc -l)
HEALTHY_COUNT=$(docker ps --filter health=healthy -q 2>/dev/null | wc -l)
UNHEALTHY_COUNT=$(docker ps --filter health=unhealthy -q 2>/dev/null | wc -l)
EXITED_COUNT=$(docker ps -a --filter status=exited -q 2>/dev/null | wc -l)

jq ".summary = {
  \"total_running\": ${TOTAL_CONTAINERS},
  \"healthy\": ${HEALTHY_COUNT},
  \"unhealthy\": ${UNHEALTHY_COUNT},
  \"exited_recently\": ${EXITED_COUNT}
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Analyze unhealthy containers
echo "🏥 Analyzing unhealthy containers..."
if [[ ${UNHEALTHY_COUNT} -gt 0 ]]; then
  UNHEALTHY_DATA=$(docker ps --filter health=unhealthy --format "json" | jq -s '[.[] | {
    "name": .Names,
    "image": .Image,
    "status": .Status,
    "created": .CreatedAt,
    "ports": .Ports
  }]')
  
  jq ".unhealthy_containers = $UNHEALTHY_DATA" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  
  # Get detailed health check logs for each unhealthy container
  for container in $(docker ps --filter health=unhealthy --format "{{.Names}}"); do
    echo "  Checking health logs for: $container"
    HEALTH_LOG=$(timeout 5 docker inspect "$container" 2>/dev/null | jq '.[0].State.Health.Log[-1] // {}' || echo '{}')
    
    jq ".health_check_failures += [{
      \"container\": \"$container\",
      \"last_check\": $HEALTH_LOG
    }]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  done
fi

# Check for container errors in logs
echo "🔎 Checking for container errors..."
CONTAINERS_WITH_ERRORS=()
for container in $(docker ps --format "{{.Names}}" | head -10); do
  ERROR_COUNT=$(timeout 2 docker logs "$container" 2>&1 | tail -100 | grep -ciE "(error|exception|fatal|panic)" || echo "0")
  if [[ ${ERROR_COUNT} -gt 0 ]]; then
    CONTAINERS_WITH_ERRORS+=("$container:$ERROR_COUNT")
  fi
done

if [[ ${#CONTAINERS_WITH_ERRORS[@]} -gt 0 ]]; then
  ERROR_JSON="["
  for error_info in "${CONTAINERS_WITH_ERRORS[@]}"; do
    IFS=':' read -r container count <<< "$error_info"
    ERROR_JSON="${ERROR_JSON}{\"container\":\"$container\",\"error_count\":$count},"
  done
  ERROR_JSON=$(echo "$ERROR_JSON" | sed 's/,$/]/')
  
  jq ".container_errors = $ERROR_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

if [[ ${UNHEALTHY_COUNT} -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"${UNHEALTHY_COUNT} unhealthy containers detected - investigate health check failures\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider restarting unhealthy containers after fixing root causes\","
fi

if [[ ${EXITED_COUNT} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"${EXITED_COUNT} exited containers found - clean up with 'docker container prune'\","
fi

if [[ ${#CONTAINERS_WITH_ERRORS[@]} -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"${#CONTAINERS_WITH_ERRORS[@]} containers have errors in logs - review container logs\","
fi

# Check for specific container issues
if docker ps | grep -q "vrooli-unstructured-io"; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"vrooli-unstructured-io container is unhealthy - may need curl installed or health check adjusted\","
fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Docker health analysis complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"