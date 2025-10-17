#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Hello Test Investigation
# DESCRIPTION: Test investigation script that provides friendly system greeting
# CATEGORY: test
# TRIGGERS: manual_test, hello_request
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="hello-test-investigation"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMESTAMP=$(date -Iseconds)

# Create output directory
mkdir -p "${OUTPUT_DIR}"
RESULTS_FILE="${OUTPUT_DIR}/results.json"

echo "👋 Hello! Starting friendly system investigation..."

# Gather friendly system facts
HOSTNAME=$(hostname)
UPTIME=$(uptime -p)
KERNEL=$(uname -r)
PROCESSES=$(ps aux | wc -l)
USERS=$(who | wc -l)

# Create results JSON
cat > "${RESULTS_FILE}" << EOF
{
  "investigation": "hello-test",
  "timestamp": "${TIMESTAMP}",
  "greeting": "Hello from the system-monitor investigation framework!",
  "system_facts": {
    "hostname": "${HOSTNAME}",
    "uptime": "${UPTIME}",
    "kernel": "${KERNEL}",
    "total_processes": ${PROCESSES},
    "active_users": ${USERS},
    "message": "Your investigation framework is working perfectly! 🎉"
  },
  "capabilities": [
    "High CPU analysis",
    "Process genealogy tracing",
    "Network anomaly detection",
    "Disk I/O analysis",
    "Zombie process detection",
    "Resource leak detection"
  ],
  "recommendations": [
    "Investigation framework operational",
    "All scripts available for use",
    "System ready for comprehensive analysis"
  ]
}
EOF

echo "✅ Hello test complete! Results saved to: ${RESULTS_FILE}"
cat "${RESULTS_FILE}"