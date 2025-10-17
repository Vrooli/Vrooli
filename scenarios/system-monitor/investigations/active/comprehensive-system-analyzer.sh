#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Comprehensive System Health Analyzer
# DESCRIPTION: Unified system health check covering processes, containers, memory, and services
# CATEGORY: resource-management
# TRIGGERS: routine_health_check, system_anomaly, performance_degradation
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-16
# LAST_MODIFIED: 2025-09-16
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="comprehensive-system-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_CMD="timeout 10"
HIGH_MEM_THRESHOLD=60
HIGH_CPU_THRESHOLD=20

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results file
RESULTS_FILE="${OUTPUT_DIR}/results.json"

echo "🔍 Starting Comprehensive System Health Analysis..."

# Initialize JSON structure
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "comprehensive-system-analyzer",
  "timestamp": "",
  "system_overview": {},
  "memory_status": {},
  "zombie_processes": [],
  "unhealthy_containers": [],
  "failed_services": [],
  "high_resource_processes": [],
  "network_anomalies": [],
  "findings": [],
  "recommendations": [],
  "auto_fixed": []
}
EOF

# Update timestamp
TIMESTAMP=$(date -Iseconds)
jq ".timestamp = \"$TIMESTAMP\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# System Overview
echo "📊 Gathering system overview..."
UPTIME=$(uptime -p | sed 's/up //')
LOAD_AVG=$(uptime | awk -F'load average:' '{print $2}' | xargs)
CPU_COUNT=$(nproc)
KERNEL=$(uname -r)

jq ".system_overview = {
  \"uptime\": \"$UPTIME\",
  \"load_average\": \"$LOAD_AVG\",
  \"cpu_count\": $CPU_COUNT,
  \"kernel\": \"$KERNEL\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Memory Status
echo "💾 Analyzing memory usage..."
MEM_INFO=$(free -m | awk 'NR==2{print $2, $3, $4, $6}')
read -r TOTAL_MEM USED_MEM FREE_MEM CACHE_MEM <<< "$MEM_INFO"
MEM_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($USED_MEM/$TOTAL_MEM)*100}")

SWAP_INFO=$(free -m | awk 'NR==3{print $2, $3}')
read -r TOTAL_SWAP USED_SWAP <<< "$SWAP_INFO"
SWAP_PERCENT=$(awk "BEGIN {if($TOTAL_SWAP > 0) printf \"%.1f\", ($USED_SWAP/$TOTAL_SWAP)*100; else print \"0.0\"}")

jq ".memory_status = {
  \"total_mb\": $TOTAL_MEM,
  \"used_mb\": $USED_MEM,
  \"free_mb\": $FREE_MEM,
  \"cache_mb\": $CACHE_MEM,
  \"usage_percent\": $MEM_PERCENT,
  \"swap_total_mb\": $TOTAL_SWAP,
  \"swap_used_mb\": $USED_SWAP,
  \"swap_percent\": $SWAP_PERCENT
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Zombie Processes
echo "🧟 Checking for zombie processes..."
ZOMBIES=()
while IFS= read -r line; do
    if [[ "$line" == *"<defunct>"* ]]; then
        PID=$(echo "$line" | awk '{print $2}')
        USER=$(echo "$line" | awk '{print $1}')
        PARENT_PID=$(ps -p "$PID" -o ppid= 2>/dev/null | xargs)
        CMD=$(echo "$line" | awk '{for(i=11;i<=NF;i++) printf "%s ", $i; print ""}' | sed 's/["\]/\\&/g')
        ZOMBIES+=("{\"pid\": \"$PID\", \"user\": \"$USER\", \"ppid\": \"$PARENT_PID\", \"command\": \"$CMD\"}")
    fi
done < <(ps aux 2>/dev/null | grep defunct | head -20)

if [ ${#ZOMBIES[@]} -gt 0 ]; then
    ZOMBIE_JSON=$(printf '%s\n' "${ZOMBIES[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".zombie_processes = $ZOMBIE_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Unhealthy Containers
echo "🐳 Checking Docker container health..."
if command -v docker &> /dev/null; then
    UNHEALTHY_CONTAINERS=()
    while IFS= read -r container; do
        if [[ -n "$container" ]]; then
            NAME=$(echo "$container" | awk '{print $1}')
            STATUS=$(echo "$container" | awk '{$1=""; print $0}' | xargs)
            UNHEALTHY_CONTAINERS+=("{\"name\": \"$NAME\", \"status\": \"$STATUS\"}")
        fi
    done < <($TIMEOUT_CMD docker ps --format "{{.Names}} {{.Status}}" 2>/dev/null | grep -E "unhealthy|Exited|Restarting" || true)
    
    if [ ${#UNHEALTHY_CONTAINERS[@]} -gt 0 ]; then
        UNHEALTHY_JSON=$(printf '%s\n' "${UNHEALTHY_CONTAINERS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
        jq ".unhealthy_containers = $UNHEALTHY_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
    fi
fi

# Failed Services
echo "🚨 Checking for failed services..."
FAILED_SERVICES=()
while IFS= read -r line; do
    if [[ -n "$line" ]]; then
        SERVICE=$(echo "$line" | awk '{print $2}')
        FAILED_SERVICES+=("\"$SERVICE\"")
    fi
done < <(systemctl list-units --failed --no-pager --no-legend 2>/dev/null || true)

if [ ${#FAILED_SERVICES[@]} -gt 0 ]; then
    FAILED_JSON=$(printf '%s\n' "${FAILED_SERVICES[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".failed_services = $FAILED_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# High Resource Processes
echo "🔥 Analyzing high resource processes..."
HIGH_RES_PROCS=()
while IFS= read -r line; do
    PID=$(echo "$line" | awk '{print $2}')
    USER=$(echo "$line" | awk '{print $1}')
    CPU=$(echo "$line" | awk '{print $3}')
    MEM=$(echo "$line" | awk '{print $4}')
    CMD=$(echo "$line" | awk '{for(i=11;i<=NF;i++) printf "%s ", $i; print ""}' | head -c 100 | sed 's/["\]/\\&/g')
    
    if (( $(echo "$CPU > $HIGH_CPU_THRESHOLD || $MEM > $HIGH_MEM_THRESHOLD" | bc -l) )); then
        HIGH_RES_PROCS+=("{\"pid\": \"$PID\", \"user\": \"$USER\", \"cpu\": $CPU, \"mem\": $MEM, \"command\": \"$CMD\"}")
    fi
done < <(ps aux --sort=-%cpu 2>/dev/null | head -10)

if [ ${#HIGH_RES_PROCS[@]} -gt 0 ]; then
    HIGH_RES_JSON=$(printf '%s\n' "${HIGH_RES_PROCS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".high_resource_processes = $HIGH_RES_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Network Anomalies (checking for unusually high connection counts)
echo "🌐 Checking network connections..."
TCP_COUNT=$(ss -t | wc -l)
ESTABLISHED=$(ss -t state established | wc -l)
TIME_WAIT=$(ss -t state time-wait | wc -l)

jq ".network_anomalies = {
  \"tcp_connections\": $TCP_COUNT,
  \"established\": $ESTABLISHED,
  \"time_wait\": $TIME_WAIT
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate Findings and Recommendations
echo "💡 Generating findings and recommendations..."
FINDINGS=()
RECOMMENDATIONS=()
AUTO_FIXED=()

# Check memory usage
if (( $(echo "$MEM_PERCENT > $HIGH_MEM_THRESHOLD" | bc -l) )); then
    FINDINGS+=("\"High memory usage detected: ${MEM_PERCENT}%\"")
    RECOMMENDATIONS+=("\"Review memory-consuming processes and consider adding swap or RAM\"")
fi

# Check swap usage
if (( $(echo "$SWAP_PERCENT > 50" | bc -l) )); then
    FINDINGS+=("\"High swap usage: ${SWAP_PERCENT}%\"")
    RECOMMENDATIONS+=("\"System is swapping heavily, performance may be degraded\"")
fi

# Check for zombies
ZOMBIE_COUNT=$(jq '.zombie_processes | length' "${RESULTS_FILE}")
if [ "$ZOMBIE_COUNT" -gt 0 ]; then
    FINDINGS+=("\"Found $ZOMBIE_COUNT zombie processes\"")
    RECOMMENDATIONS+=("\"Kill parent processes of zombies or restart affected services\"")
    
    # Auto-fix: Try to clean up zombies safely
    if [ "$ZOMBIE_COUNT" -gt 5 ]; then
        echo "🔧 Attempting to clean up zombie processes..."
        for pid in $(jq -r '.zombie_processes[].ppid' "${RESULTS_FILE}" | sort -u); do
            if [[ -n "$pid" && "$pid" != "1" ]]; then
                # Only kill if it's a safe process (not critical system process)
                PROC_NAME=$(ps -p "$pid" -o comm= 2>/dev/null || echo "unknown")
                if [[ "$PROC_NAME" =~ ^(python|node|ruby|java|defunct)$ ]]; then
                    if kill -SIGCHLD "$pid" 2>/dev/null; then
                        AUTO_FIXED+=("\"Sent SIGCHLD to parent process $pid ($PROC_NAME)\"")
                    fi
                fi
            fi
        done
    fi
fi

# Check for unhealthy containers
UNHEALTHY_COUNT=$(jq '.unhealthy_containers | length' "${RESULTS_FILE}")
if [ "$UNHEALTHY_COUNT" -gt 0 ]; then
    FINDINGS+=("\"Found $UNHEALTHY_COUNT unhealthy Docker containers\"")
    RECOMMENDATIONS+=("\"Review container logs and restart if necessary\"")
fi

# Check for failed services
FAILED_COUNT=$(jq '.failed_services | length' "${RESULTS_FILE}")
if [ "$FAILED_COUNT" -gt 0 ]; then
    FINDINGS+=("\"Found $FAILED_COUNT failed systemd services\"")
    RECOMMENDATIONS+=("\"Review service logs with 'journalctl -u <service-name>'\"")
fi

# Check for network issues
if [ "$TIME_WAIT" -gt 500 ]; then
    FINDINGS+=("\"High number of TIME_WAIT connections: $TIME_WAIT\"")
    RECOMMENDATIONS+=("\"Review application connection pooling and TCP settings\"")
fi

# Update findings
if [ ${#FINDINGS[@]} -gt 0 ]; then
    FINDINGS_JSON=$(printf '%s\n' "${FINDINGS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".findings = $FINDINGS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update recommendations
if [ ${#RECOMMENDATIONS[@]} -gt 0 ]; then
    RECS_JSON=$(printf '%s\n' "${RECOMMENDATIONS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".recommendations = $RECS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update auto-fixed items
if [ ${#AUTO_FIXED[@]} -gt 0 ]; then
    FIXED_JSON=$(printf '%s\n' "${AUTO_FIXED[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".auto_fixed = $FIXED_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Output final results
echo "✅ System health analysis complete!"
echo "📄 Results saved to: ${OUTPUT_DIR}/results.json"
cat "${RESULTS_FILE}" | jq '.'