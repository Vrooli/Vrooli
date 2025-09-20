#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Disk I/O Analyzer
# DESCRIPTION: Analyzes disk usage patterns, I/O performance, and identifies storage bottlenecks
# CATEGORY: storage
# TRIGGERS: high_disk_usage, io_wait, disk_errors
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0
trap 'status=$?; if [[ $status -eq 141 ]]; then exit 0; fi' EXIT

set -euo pipefail

# Configuration
SCRIPT_NAME="disk-io-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30
DISK_THRESHOLD=80

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "disk-io-analyzer",
  "timestamp": "",
  "disk_usage": [],
  "large_files": [],
  "growing_files": [],
  "io_stats": {},
  "suspicious_patterns": [],
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "💾 Starting Disk I/O Analysis..."

# Disk Usage Analysis
echo "📊 Analyzing disk usage..."
DISK_USAGE=$(df -h | awk 'NR>1 && $5 ~ /%/ {
  gsub("%", "", $5);
  printf "{\"filesystem\":\"%s\",\"size\":\"%s\",\"used\":\"%s\",\"available\":\"%s\",\"use_percent\":%d,\"mount\":\"%s\"},",
         $1, $2, $3, $4, $5, $6
}' | sed 's/,$//')

if [[ -n "${DISK_USAGE}" ]]; then
  jq ".disk_usage = [${DISK_USAGE}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Find Large Files
echo "🔍 Finding large files..."
LARGE_FILES=$(timeout ${TIMEOUT_SECONDS} find /home /tmp /var/log -type f -size +100M 2>/dev/null | head -20 | while read -r file; do
  if [[ -f "${file}" ]]; then
    SIZE=$(du -h "${file}" 2>/dev/null | cut -f1)
    FILE_ESC=$(echo "${file}" | sed 's/"/\\"/g')
    printf "{\"path\":\"%s\",\"size\":\"%s\"}," "${FILE_ESC}" "${SIZE}"
  fi
done | sed 's/,$//')

if [[ -n "${LARGE_FILES}" ]]; then
  jq ".large_files = [${LARGE_FILES}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Find Recently Modified Large Files (potentially growing)
echo "📈 Finding growing files..."
GROWING_FILES=$(timeout ${TIMEOUT_SECONDS} find /var/log /tmp -type f -mmin -60 -size +10M 2>/dev/null | head -10 | while read -r file; do
  if [[ -f "${file}" ]]; then
    SIZE=$(du -h "${file}" 2>/dev/null | cut -f1)
    MTIME=$(stat -c %y "${file}" 2>/dev/null | cut -d. -f1)
    FILE_ESC=$(echo "${file}" | sed 's/"/\\"/g')
    printf "{\"path\":\"%s\",\"size\":\"%s\",\"modified\":\"%s\"}," "${FILE_ESC}" "${SIZE}" "${MTIME}"
  fi
done | sed 's/,$//')

if [[ -n "${GROWING_FILES}" ]]; then
  jq ".growing_files = [${GROWING_FILES}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# I/O Statistics
echo "⚡ Gathering I/O statistics..."
IOSTAT_DATA=$(timeout 5 iostat -x 1 2 2>/dev/null | tail -n +4 | awk '
  NR>1 && $1 != "" {
    if ($1 != "Device" && $1 != "avg-cpu:") {
      total_io += $4 + $5;
      total_await += $10;
      device_count++;
      if ($14 > max_util) {
        max_util = $14;
        max_util_device = $1;
      }
    }
  }
  END {
    avg_await = device_count > 0 ? total_await / device_count : 0;
    printf "{\"total_io_per_sec\":%.2f,\"avg_await_ms\":%.2f,\"max_util_percent\":%.2f,\"busiest_device\":\"%s\"}",
           total_io, avg_await, max_util, max_util_device
  }
')

if [[ -n "${IOSTAT_DATA}" && "${IOSTAT_DATA}" != "{}" ]]; then
  jq ".io_stats = ${IOSTAT_DATA}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Suspicious Pattern Detection
echo "🚨 Detecting suspicious patterns..."
SUSPICIOUS=""

# Check for high disk usage
HIGH_USAGE_COUNT=$(echo "${DISK_USAGE}" | jq -s '[.[] | select(.use_percent >= 80)] | length' 2>/dev/null || echo "0")
if [[ ${HIGH_USAGE_COUNT} -gt 0 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"high_disk_usage\",\"count\":${HIGH_USAGE_COUNT}},"
fi

# Check for large log files
LARGE_LOG_COUNT=$(echo "${LARGE_FILES}" | jq -s '[.[] | select(.path | contains("/log"))] | length' 2>/dev/null || echo "0")
if [[ ${LARGE_LOG_COUNT} -gt 0 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"large_log_files\",\"count\":${LARGE_LOG_COUNT}},"
fi

# Check for /tmp filling up
TMP_USAGE=$(df /tmp 2>/dev/null | awk 'NR==2 {gsub("%",""); print $5}')
if [[ -n "${TMP_USAGE}" && ${TMP_USAGE} -gt 50 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"tmp_filling\",\"usage_percent\":${TMP_USAGE}},"
fi

# Check for swap usage
SWAP_USAGE=$(free | awk '/^Swap:/ && $2>0 {printf "%.0f", $3/$2*100}')
if [[ -n "${SWAP_USAGE}" && ${SWAP_USAGE} -gt 20 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"swap_usage\",\"percent\":${SWAP_USAGE}},"
fi

SUSPICIOUS=$(echo "${SUSPICIOUS}" | sed 's/,$//')
if [[ -n "${SUSPICIOUS}" ]]; then
  jq ".suspicious_patterns = [${SUSPICIOUS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate Recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Disk usage recommendations
if [[ ${HIGH_USAGE_COUNT} -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Critical: ${HIGH_USAGE_COUNT} filesystems above 80% usage - clean up or expand storage\","
fi

# Large file recommendations
LARGE_FILE_COUNT=$(echo "${LARGE_FILES}" | jq -s 'length' 2>/dev/null || echo "0")
if [[ ${LARGE_FILE_COUNT} -gt 5 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Found ${LARGE_FILE_COUNT} large files (>100MB) - review and archive if needed\","
fi

# Growing file recommendations
GROWING_COUNT=$(echo "${GROWING_FILES}" | jq -s 'length' 2>/dev/null || echo "0")
if [[ ${GROWING_COUNT} -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"${GROWING_COUNT} files growing rapidly - check log rotation and application output\","
fi

# I/O performance recommendations
MAX_UTIL=$(echo "${IOSTAT_DATA}" | jq '.max_util_percent' 2>/dev/null || echo "0")
if (( $(echo "${MAX_UTIL} > 80" | bc -l) )); then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"High disk utilization detected - investigate I/O bottlenecks\","
fi

# Swap recommendations
if [[ -n "${SWAP_USAGE}" && ${SWAP_USAGE} -gt 20 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Swap usage at ${SWAP_USAGE}% - increase memory or optimize applications\","
fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Disk I/O analysis complete! Results: ${RESULTS_FILE}"
cat "${RESULTS_FILE}"
