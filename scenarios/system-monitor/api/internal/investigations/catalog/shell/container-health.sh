#!/usr/bin/env bash
# INVESTIGATION_SCRIPT
# NAME: Container Health Analyzer
# DESCRIPTION: Reports container runtime availability and a bounded health summary.
# CATEGORY: resource-management
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2026-02-18
# LAST_MODIFIED: 2026-08-26

set -euo pipefail
printf '%s\n' 'collecting container health' >&2
timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
if ! docker info >/dev/null 2>&1; then
  jq -n --arg ts "$timestamp" '{investigation:"container-health", timestamp:$ts, status:"unavailable", reason:"Docker daemon is unavailable", findings:[], recommendations:[]}'
  exit 0
fi

running="$(docker ps -q 2>/dev/null | wc -l | tr -d ' ')"
unhealthy="$(docker ps --filter health=unhealthy -q 2>/dev/null | wc -l | tr -d ' ')"
exited="$(docker ps -a --filter status=exited -q 2>/dev/null | wc -l | tr -d ' ')"
jq -n --arg ts "$timestamp" --argjson running "${running:-0}" --argjson unhealthy "${unhealthy:-0}" --argjson exited "${exited:-0}" \
  '{investigation:"container-health", timestamp:$ts, status:"completed", summary:{total_running:$running, unhealthy:$unhealthy, exited:$exited}, findings:[], recommendations:[]}'
