#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: [Your Investigation Name]
# DESCRIPTION: [Brief description of what this script investigates]
# CATEGORY: [performance|process-analysis|resource-management|network|storage]
# TRIGGERS: [condition1, condition2, ...]
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: $(date +%Y-%m-%d)
# LAST_MODIFIED: $(date +%Y-%m-%d)
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="your-script-name"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=60
RESULTS_FILE="${OUTPUT_DIR}/results.json"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results structure
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "",
  "timestamp": "",
  "summary": {},
  "findings": [],
  "patterns": {},
  "recommendations": [],
  "raw_data": {}
}
EOF

# Set script name and timestamp
jq ".investigation = \"${SCRIPT_NAME}\" | .timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting ${SCRIPT_NAME} investigation..."

# Your investigation logic here
echo "📊 Gathering data..."

# Example: Basic system info
SYSTEM_INFO="{\"hostname\":\"$(hostname)\",\"uptime\":\"$(uptime -p)\"}"
jq ".summary = ${SYSTEM_INFO}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Example: Collect findings
echo "🔍 Analyzing..."
FINDINGS=""

# Add your specific investigation steps here
# Example pattern:
# METRIC=$(your_command_here | process_output)
# FINDING_JSON="{\"metric\":\"value\",\"status\":\"normal|warning|critical\"}"
# FINDINGS="${FINDINGS}${FINDING_JSON},"

# Remove trailing comma and update results
FINDINGS=$(echo "${FINDINGS}" | sed 's/,$//')
if [[ -n "${FINDINGS}" ]]; then
  jq ".findings = [${FINDINGS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Add your recommendation logic here
# Example:
# if [[ condition ]]; then
#   RECOMMENDATIONS="${RECOMMENDATIONS}\"Your recommendation here\","
# fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Investigation complete! Results saved to: ${RESULTS_FILE}"

# Output results for API consumption
cat "${RESULTS_FILE}"

# Cleanup temporary files if needed
# rm -f /tmp/your_temp_files*