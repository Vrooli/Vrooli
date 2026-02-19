#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Container Health Analyzer
# DESCRIPTION: Consolidated analysis of container health, errors, restarts, and resource usage
# CATEGORY: resource-management
# TRIGGERS: container_unhealthy, health_check_failures, docker_errors, container_memory_high, resource_inefficiency, container_limits_unset
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2026-02-18
# LAST_MODIFIED: 2026-02-18
# VERSION: 2.0

set -euo pipefail

source "$(dirname "$0")/lib/common.sh"
init_investigation "container-health"

jq -n '{
  investigation: "container-health",
  timestamp: (now | strftime("%Y-%m-%dT%H:%M:%S%z")),
  summary: {},
  unhealthy_containers: [],
  health_check_failures: [],
  container_errors: [],
  restart_counts: [],
  containers_without_healthcheck: [],
  resource_analysis: {},
  findings: [],
  recommendations: []
}' > "${RESULTS_FILE}"

if ! command -v docker &>/dev/null; then
    add_finding "Docker not available"
    finalize_investigation
    exit 0
fi

gather_summary() {
    local total healthy unhealthy exited
    total=$(safe_timeout 10 docker ps -q | wc -l)
    healthy=$(safe_timeout 10 docker ps --filter health=healthy -q | wc -l)
    unhealthy=$(safe_timeout 10 docker ps --filter health=unhealthy -q | wc -l)
    exited=$(safe_timeout 10 docker ps -a --filter status=exited -q | wc -l)
    total=$(sanitize_int "$total"); healthy=$(sanitize_int "$healthy")
    unhealthy=$(sanitize_int "$unhealthy"); exited=$(sanitize_int "$exited")

    update_json_value '.summary = $value' \
        "{\"total_running\":${total},\"healthy\":${healthy},\"unhealthy\":${unhealthy},\"exited\":${exited}}"
}

analyze_unhealthy() {
    local containers
    containers=$(safe_timeout 10 docker ps --filter health=unhealthy --format '{{.Names}}')
    [[ -z "${containers}" ]] && return

    while IFS= read -r name; do
        local info
        info=$(safe_timeout 10 docker inspect "$name" 2>/dev/null \
            | jq '.[0] | {name: .Name, image: .Config.Image, status: .State.Health.Status, last_log: (.State.Health.Log[-1] // {})}' 2>/dev/null || echo '{}')
        update_json_value '.unhealthy_containers += [$value]' "$info"

        local health_log
        health_log=$(safe_timeout 5 docker inspect "$name" 2>/dev/null \
            | jq --arg c "$name" '{container: $c, last_check: (.[0].State.Health.Log[-1] // {})}' 2>/dev/null || echo '{}')
        update_json_value '.health_check_failures += [$value]' "$health_log"
    done <<< "${containers}"

    add_finding "$(echo "${containers}" | wc -l | tr -d ' ') unhealthy container(s) detected"
}

check_restarts() {
    local containers
    containers=$(safe_timeout 10 docker ps --format '{{.Names}}')
    [[ -z "${containers}" ]] && return

    while IFS= read -r name; do
        local count
        count=$(safe_timeout 5 docker inspect "$name" 2>/dev/null | jq -r '.[0].RestartCount // 0' || echo 0)
        count=$(sanitize_int "$count")
        if [[ ${count} -gt 0 ]]; then
            update_json_value '.restart_counts += [$value]' "{\"container\":\"${name}\",\"restart_count\":${count}}"
        fi
        if [[ ${count} -gt 5 ]]; then
            add_finding "Container ${name} has restarted ${count} times"
        fi
    done <<< "${containers}"
}

check_missing_healthchecks() {
    local containers
    containers=$(safe_timeout 10 docker ps --format '{{.Names}}')
    [[ -z "${containers}" ]] && return

    while IFS= read -r name; do
        local health
        health=$(safe_timeout 5 docker inspect "$name" 2>/dev/null | jq -r '.[0].State.Health // "null"' || echo "null")
        if [[ "${health}" == "null" ]]; then
            jq --arg c "$name" '.containers_without_healthcheck += [$c]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" \
                && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
        fi
    done <<< "${containers}"
}

check_container_errors() {
    local containers
    containers=$(safe_timeout 10 docker ps --format '{{.Names}}' | head -15)
    [[ -z "${containers}" ]] && return

    while IFS= read -r name; do
        local err_count
        err_count=$(safe_timeout 5 docker logs "$name" 2>&1 | tail -100 | grep -ciE '(error|fatal|panic)' || true)
        err_count=$(sanitize_int "$err_count")
        if [[ ${err_count} -gt 0 ]]; then
            update_json_value '.container_errors += [$value]' "{\"container\":\"${name}\",\"error_count\":${err_count}}"
        fi
    done <<< "${containers}"
}

analyze_resource_usage() {
    local stats
    stats=$(safe_timeout 15 docker stats --no-stream --format '{{.Name}}|{{.MemUsage}}|{{.MemPerc}}|{{.CPUPerc}}')
    [[ -z "${stats}" ]] && return

    local overprovisioned='[]' underprovisioned='[]' unlimited='[]'

    while IFS='|' read -r name mem_usage mem_pct _cpu_pct; do
        local pct_val
        pct_val=$(echo "${mem_pct}" | tr -d '% ')
        # Skip non-numeric
        [[ "${pct_val}" =~ ^[0-9.]+$ ]] || continue

        if awk "BEGIN{exit !(${pct_val} < 10)}"; then
            overprovisioned=$(echo "$overprovisioned" | jq --arg n "$name" --arg m "$mem_usage" --arg p "$mem_pct" '. += [{"container":$n,"mem_usage":$m,"mem_percent":$p}]')
        elif awk "BEGIN{exit !(${pct_val} > 85)}"; then
            underprovisioned=$(echo "$underprovisioned" | jq --arg n "$name" --arg m "$mem_usage" --arg p "$mem_pct" '. += [{"container":$n,"mem_usage":$m,"mem_percent":$p}]')
        fi
    done <<< "${stats}"

    # Find containers without memory limits
    local container_ids
    container_ids=$(safe_timeout 10 docker ps -q)
    if [[ -n "${container_ids}" ]]; then
        while IFS= read -r cid; do
            local limit cname
            limit=$(safe_timeout 5 docker inspect "$cid" --format '{{.HostConfig.Memory}}' || echo "0")
            if [[ "${limit}" == "0" ]]; then
                cname=$(safe_timeout 5 docker inspect "$cid" --format '{{.Name}}' | sed 's|^/||')
                unlimited=$(echo "$unlimited" | jq --arg n "$cname" '. += [{"container":$n,"memory_limit":"unlimited"}]')
            fi
        done <<< "${container_ids}"
    fi

    update_json_value '.resource_analysis = $value' \
        "$(jq -n --argjson o "$overprovisioned" --argjson u "$underprovisioned" --argjson l "$unlimited" \
        '{overprovisioned:$o, underprovisioned:$u, unlimited:$l}')"
}

generate_recommendations() {
    local unhealthy_count restart_total no_health_count
    unhealthy_count=$(jq '.unhealthy_containers | length' "${RESULTS_FILE}")
    restart_total=$(jq '[.restart_counts[].restart_count] | add // 0' "${RESULTS_FILE}")
    no_health_count=$(jq '.containers_without_healthcheck | length' "${RESULTS_FILE}")

    [[ ${unhealthy_count} -gt 0 ]] && add_recommendation "Investigate ${unhealthy_count} unhealthy container(s) - check health endpoints and logs"
    [[ ${restart_total} -gt 10 ]] && add_recommendation "High total restart count (${restart_total}) - check for crash loops"
    [[ ${no_health_count} -gt 5 ]] && add_recommendation "Add HEALTHCHECK directives to ${no_health_count} container(s) without health checks"

    local over under unlim
    over=$(jq '.resource_analysis.overprovisioned | length' "${RESULTS_FILE}")
    under=$(jq '.resource_analysis.underprovisioned | length' "${RESULTS_FILE}")
    unlim=$(jq '.resource_analysis.unlimited | length' "${RESULTS_FILE}")
    [[ ${over} -gt 0 ]] && add_recommendation "Reduce memory limits for ${over} overprovisioned container(s) (<10% usage)"
    [[ ${under} -gt 0 ]] && add_recommendation "Increase memory limits for ${under} underprovisioned container(s) (>85% usage)"
    [[ ${unlim} -gt 0 ]] && add_recommendation "Set memory limits on ${unlim} unlimited container(s) to prevent resource exhaustion"
}

gather_summary
analyze_unhealthy
check_restarts
check_missing_healthchecks
check_container_errors
analyze_resource_usage
generate_recommendations
finalize_investigation
