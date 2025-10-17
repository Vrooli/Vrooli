#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Service Health Monitor
# DESCRIPTION: Detects failed systemd services and provides recovery recommendations
# CATEGORY: resource-management
# TRIGGERS: service_failures, systemd_issues, daemon_problems
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-15
# LAST_MODIFIED: 2025-09-15
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="service-health-monitor"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "service-health-monitor",
  "timestamp": "",
  "failed_services": [],
  "degraded_services": [],
  "recently_restarted": [],
  "service_details": [],
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Service Health Monitoring..."

# Check for failed services
echo "🚨 Checking for failed services..."
FAILED_JSON="[]"
while IFS= read -r line; do
    if [[ "$line" == *"●"* && "$line" == *"failed"* ]]; then
        SERVICE_NAME=$(echo "$line" | awk '{print $2}')
        DESCRIPTION=$(echo "$line" | cut -d' ' -f6-)
        FAILED_JSON=$(echo "$FAILED_JSON" | jq ". += [{\"service\": \"$SERVICE_NAME\", \"status\": \"failed\", \"description\": \"$DESCRIPTION\"}]")
    fi
done < <(systemctl list-units --failed --no-pager --no-legend 2>/dev/null || true)

jq ".failed_services = $FAILED_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Check for degraded services
echo "⚠️ Checking for degraded services..."
DEGRADED_JSON="[]"
while IFS= read -r line; do
    if [[ -n "$line" ]]; then
        SERVICE_NAME=$(echo "$line" | awk '{print $1}')
        DEGRADED_JSON=$(echo "$DEGRADED_JSON" | jq ". += [{\"service\": \"$SERVICE_NAME\", \"status\": \"degraded\"}]")
    fi
done < <(systemctl list-units --state=degraded --no-pager --no-legend 2>/dev/null || true)

jq ".degraded_services = $DEGRADED_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Check recently restarted services (last hour)
echo "🔄 Checking recently restarted services..."
RECENT_JSON="[]"
for service in $(systemctl list-units --type=service --no-pager --no-legend | awk '{print $1}'); do
    if systemctl show "$service" --property=ActiveEnterTimestamp 2>/dev/null | grep -q "$(date +'%Y-%m-%d %H')"; then
        RECENT_JSON=$(echo "$RECENT_JSON" | jq ". += [\"$service\"]")
    fi
done 2>/dev/null || true

jq ".recently_restarted = $RECENT_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Get details for failed services
echo "📋 Gathering service details..."
DETAILS_JSON="[]"
for service in $(jq -r '.failed_services[].service' "${RESULTS_FILE}"); do
    if [[ -n "$service" ]]; then
        STATUS=$(systemctl status "$service" 2>&1 | head -20 | sed 's/"/\\"/g' | tr '\n' ' ' || true)
        LOGS=$(journalctl -u "$service" --no-pager -n 10 2>&1 | sed 's/"/\\"/g' | tr '\n' ' ' || true)
        DETAILS_JSON=$(echo "$DETAILS_JSON" | jq ". += [{\"service\": \"$service\", \"status\": \"$STATUS\", \"recent_logs\": \"$LOGS\"}]")
    fi
done

jq ".service_details = $DETAILS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS="[]"

# Check if any services failed
if [ $(jq '.failed_services | length' "${RESULTS_FILE}") -gt 0 ]; then
    RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += ["Failed services detected - review logs and restart if appropriate"]')
fi

# Check if any services are degraded
if [ $(jq '.degraded_services | length' "${RESULTS_FILE}") -gt 0 ]; then
    RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += ["Degraded services found - investigate resource constraints"]')
fi

# Check for frequent restarts
if [ $(jq '.recently_restarted | length' "${RESULTS_FILE}") -gt 5 ]; then
    RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += ["Multiple services restarted recently - check system stability"]')
fi

jq ".recommendations = $RECOMMENDATIONS" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "✅ Service health monitoring complete! Results: ${OUTPUT_DIR}/results.json"
cat "${RESULTS_FILE}" | jq .