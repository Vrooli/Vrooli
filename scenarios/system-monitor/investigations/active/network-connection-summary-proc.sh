#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Network Connection Summary (proc)
# DESCRIPTION: Summarizes TCP states and listening ports using /proc/net to avoid netlink permission issues
# CATEGORY: network
# TRIGGERS: netlink_unavailable, restricted_caps, high_tcp_count
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-12-22
# LAST_MODIFIED: 2025-12-22
# VERSION: 1.0

set -euo pipefail

SCRIPT_NAME="network-connection-summary-proc"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
RESULTS_FILE="${OUTPUT_DIR}/results.json"

mkdir -p "${OUTPUT_DIR}"

cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "network-connection-summary-proc",
  "timestamp": "",
  "summary": {},
  "listening_ports": [],
  "findings": [],
  "recommendations": []
}
EOF

jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting ${SCRIPT_NAME}..."

total_all=0
listen_all=0
established_all=0
time_wait_all=0
close_wait_all=0
files_read=0

declare -A port_counts=()

hex2dec_awk='
function hex2dec(h, i, d, c) {
  d=0
  for (i=1; i<=length(h); i++) {
    c=tolower(substr(h,i,1))
    d = d*16 + index("0123456789abcdef", c) - 1
  }
  return d
}
'

for file in /proc/net/tcp /proc/net/tcp6; do
  if [[ -r "${file}" ]]; then
    files_read=$((files_read + 1))
    total=0
    listen=0
    established=0
    time_wait=0
    close_wait=0
    if read -r total listen established time_wait close_wait < <(
      awk "${hex2dec_awk}
        NR>1 {
          total++
          state=\$4
          if (state==\"0A\") {listen++}
          else if (state==\"01\") {established++}
          else if (state==\"06\") {time_wait++}
          else if (state==\"08\") {close_wait++}
        }
        END {print total+0, listen+0, established+0, time_wait+0, close_wait+0}
      " "${file}"
    ); then
      total_all=$((total_all + total))
      listen_all=$((listen_all + listen))
      established_all=$((established_all + established))
      time_wait_all=$((time_wait_all + time_wait))
      close_wait_all=$((close_wait_all + close_wait))
    fi

    while read -r port count; do
      [[ -z "${port}" ]] && continue
      port_counts["${port}"]=$(( ${port_counts["${port}"]:-0} + count ))
    done < <(
      awk "${hex2dec_awk}
        NR>1 && \$4==\"0A\" {
          split(\$2, addr, \":\")
          port=hex2dec(addr[2])
          counts[port]++
        }
        END {for (p in counts) print p, counts[p]}
      " "${file}"
    )
  fi
done

ports_json='[]'
if (( ${#port_counts[@]} > 0 )); then
  ports_lines=""
  for port in "${!port_counts[@]}"; do
    ports_lines+="${port}\t${port_counts[$port]}\n"
  done
  ports_json=$(printf "%b" "${ports_lines}" | sort -n | jq -Rsc 'split("\n") | map(select(length>0)) | map(split("\t")) | map({port:(.[0]|tonumber), count:(.[1]|tonumber)})')
fi

jq ".summary = {
  \"files_read\": ${files_read},
  \"total_connections\": ${total_all},
  \"listening\": ${listen_all},
  \"established\": ${established_all},
  \"time_wait\": ${time_wait_all},
  \"close_wait\": ${close_wait_all}
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

jq ".listening_ports = ${ports_json}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

findings=""
recommendations=""

if [[ ${files_read} -eq 0 ]]; then
  findings="\"Unable to read /proc/net/tcp or /proc/net/tcp6\""
  recommendations="\"Check permissions for /proc/net files\""
fi

if [[ ${time_wait_all} -gt 1000 ]]; then
  findings="${findings}${findings:+,}\"High TIME_WAIT count: ${time_wait_all}\""
  recommendations="${recommendations}${recommendations:+,}\"Review connection pooling and TCP settings\""
fi

if [[ ${close_wait_all} -gt 100 ]]; then
  findings="${findings}${findings:+,}\"High CLOSE_WAIT count: ${close_wait_all}\""
  recommendations="${recommendations}${recommendations:+,}\"Investigate services not closing sockets\""
fi

if [[ -n "${findings}" ]]; then
  jq ".findings = [${findings}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

if [[ -n "${recommendations}" ]]; then
  jq ".recommendations = [${recommendations}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ ${SCRIPT_NAME} complete! Results: ${RESULTS_FILE}"
cat "${RESULTS_FILE}"
