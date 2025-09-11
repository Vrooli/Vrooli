#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Network Anomaly Detector
# DESCRIPTION: Detects suspicious network connections, unusual ports, and potential security issues
# CATEGORY: network
# TRIGGERS: unusual_connections, high_tcp_count, suspicious_ports
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="network-anomaly-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=45

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "network-anomaly-detector",
  "timestamp": "",
  "connection_summary": {},
  "listening_ports": [],
  "external_connections": [],
  "suspicious_patterns": [],
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Network Anomaly Detection..."

# Connection Summary
echo "📊 Analyzing connection states..."
TOTAL_CONNECTIONS=$(ss -a | wc -l)
ESTABLISHED=$(ss -t state established | wc -l)
LISTENING=$(ss -lt | wc -l)
TIME_WAIT=$(ss -t state time-wait | wc -l)
CLOSE_WAIT=$(ss -t state close-wait | wc -l)

jq ".connection_summary = {
  \"total\": ${TOTAL_CONNECTIONS},
  \"established\": ${ESTABLISHED},
  \"listening\": ${LISTENING},
  \"time_wait\": ${TIME_WAIT},
  \"close_wait\": ${CLOSE_WAIT}
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Listening Ports Analysis
echo "🚪 Checking listening ports..."
LISTENING_PORTS=$(timeout ${TIMEOUT_SECONDS} ss -tlnp 2>/dev/null | awk 'NR>1 {
  split($4, addr, ":");
  port = addr[length(addr)];
  if (port ~ /^[0-9]+$/) {
    printf "{\"port\":%s,\"address\":\"%s\"},", port, $4
  }
}' | sed 's/,$//')

if [[ -n "${LISTENING_PORTS}" ]]; then
  jq ".listening_ports = [${LISTENING_PORTS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# External Connections Analysis
echo "🌐 Analyzing external connections..."
EXTERNAL_CONNS=$(timeout ${TIMEOUT_SECONDS} ss -tn state established 2>/dev/null | awk 'NR>1 && $4 !~ /^127\./ && $4 !~ /^::1/ {
  split($4, local, ":");
  split($5, remote, ":");
  if (remote[1] != "" && remote[1] !~ /^127\./ && remote[1] !~ /^10\./ && remote[1] !~ /^192\.168\./ && remote[1] !~ /^172\.(1[6-9]|2[0-9]|3[01])\./) {
    printf "{\"local_port\":\"%s\",\"remote_addr\":\"%s\",\"remote_port\":\"%s\"},", 
           local[length(local)], remote[1], remote[length(remote)]
  }
}' | sed 's/,$//')

if [[ -n "${EXTERNAL_CONNS}" ]]; then
  jq ".external_connections = [${EXTERNAL_CONNS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Suspicious Pattern Detection
echo "🚨 Detecting suspicious patterns..."
SUSPICIOUS=""

# Check for high number of connections from single IP
CONNECTION_COUNTS=$(ss -tn | awk '$1 == "ESTAB" {print $5}' | cut -d: -f1 | sort | uniq -c | sort -rn | head -5)
TOP_CONN_COUNT=$(echo "${CONNECTION_COUNTS}" | head -1 | awk '{print $1}')
TOP_CONN_IP=$(echo "${CONNECTION_COUNTS}" | head -1 | awk '{print $2}')

if [[ ${TOP_CONN_COUNT} -gt 50 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"excessive_connections\",\"details\":\"${TOP_CONN_COUNT} connections from ${TOP_CONN_IP}\"},"
fi

# Check for unusual high ports
HIGH_PORTS=$(ss -tln | awk '{split($4,a,":"); port=a[length(a)]; if(port+0 > 30000 && port+0 < 65535) print port}' | wc -l)
if [[ ${HIGH_PORTS} -gt 10 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"many_high_ports\",\"details\":\"${HIGH_PORTS} services on high ports\"},"
fi

# Check for TIME_WAIT/CLOSE_WAIT accumulation
if [[ ${TIME_WAIT} -gt 1000 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"time_wait_accumulation\",\"count\":${TIME_WAIT}},"
fi

if [[ ${CLOSE_WAIT} -gt 100 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"close_wait_accumulation\",\"count\":${CLOSE_WAIT}},"
fi

# Check for common attack ports
ATTACK_PORTS="22 23 445 3389 5900"
for port in ${ATTACK_PORTS}; do
  PORT_CONNS=$(ss -tn sport = :${port} 2>/dev/null | wc -l)
  if [[ ${PORT_CONNS} -gt 20 ]]; then
    SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"attack_port_activity\",\"port\":${port},\"connections\":${PORT_CONNS}},"
  fi
done

SUSPICIOUS=$(echo "${SUSPICIOUS}" | sed 's/,$//')
if [[ -n "${SUSPICIOUS}" ]]; then
  jq ".suspicious_patterns = [${SUSPICIOUS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate Recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

if [[ ${ESTABLISHED} -gt 1000 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"High number of established connections (${ESTABLISHED}) - check for connection leaks\","
fi

if [[ ${CLOSE_WAIT} -gt 100 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"CLOSE_WAIT accumulation detected - application not closing connections properly\","
fi

if [[ ${TIME_WAIT} -gt 1000 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"TIME_WAIT accumulation - consider tuning tcp_tw_reuse kernel parameter\","
fi

if [[ ${HIGH_PORTS} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Many services on high ports - audit for unauthorized services\","
fi

# Check if external connections exist
EXTERNAL_COUNT=$(echo "${EXTERNAL_CONNS}" | jq -s 'length' 2>/dev/null || echo "0")
if [[ ${EXTERNAL_COUNT} -gt 50 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"High number of external connections (${EXTERNAL_COUNT}) - verify all are legitimate\","
fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Network anomaly detection complete! Results: ${RESULTS_FILE}"
cat "${RESULTS_FILE}"