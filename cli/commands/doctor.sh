#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Doctor / Preflight Checks
#
# Verifies local environment prerequisites and common tools, emitting either
# human-readable text or JSON using the shared format.sh helpers.
################################################################################

set -euo pipefail

CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/format.sh"

show_help() {
	cat << EOF
🩺 Vrooli Doctor - Environment Preflight

USAGE:
    vrooli doctor [--json]

Checks:
    - jq, curl, git, docker, go, lsof, tput
    - API port availability (VROOLI_API_PORT)
    - Config presence: ".vrooli/service.json"

OPTIONS:
    --json      Emit JSON output
    --help,-h   Show this help message
EOF
}

check_cmd() {
	local name="$1"
	if command -v "$name" >/dev/null 2>&1; then
		printf "%s:%s" "$name" "ok"
	else
		printf "%s:%s" "$name" "missing"
	fi
}

check_port_free() {
	local port="$1"
	if command -v lsof >/dev/null 2>&1 && lsof -i ":${port}" >/dev/null 2>&1; then
		printf "api_port:%s" "in_use"
	else
		printf "api_port:%s" "free"
	fi
}

main() {
	local output_format="text"
	while [[ $# -gt 0 ]]; do
		case "$1" in
			--json) output_format="json" ;;
			--help|-h) show_help; return 0 ;;
			*) ;;
		esac
		shift || true
	done
	
	local checks=(jq curl git docker go lsof tput)
	local rows=()
	
	for c in "${checks[@]}"; do
		local result
		result=$(check_cmd "$c")
		local status
		status="${result#*:}"
		if [[ "$output_format" == "json" ]]; then
			rows+=("${c}:${status}")
		else
			rows+=("${c}:$([[ "$status" == ok ]] && echo OK || echo MISSING)")
		fi
	done
	
	# API port
	local api_port="${VROOLI_API_PORT:-8090}"
	local port_status
	port_status=$(check_port_free "$api_port")
	local pstat="${port_status#*:}"
	rows+=("api_port_${api_port}:${pstat}")
	
	# Config
	local cfg="${var_SERVICE_JSON_FILE}"
	if [[ -f "$cfg" ]]; then
		rows+=("service_json:present")
	else
		rows+=("service_json:missing")
	fi
	
	# Output
	if [[ "$output_format" == "json" ]]; then
		format::table json "check" "status" -- "${rows[@]}"
	else
		log::header "Vrooli Doctor"
		format::table text "Check" "Status" -- "${rows[@]}"
		
		# Hints
		echo ""
		log::subheader "Hints"
		[[ " ${rows[*]} " =~ jq:missing ]] && echo "- Install jq: sudo apt-get install -y jq"
		[[ " ${rows[*]} " =~ curl:missing ]] && echo "- Install curl: sudo apt-get install -y curl"
		[[ " ${rows[*]} " =~ git:missing ]] && echo "- Install git: sudo apt-get install -y git"
		[[ " ${rows[*]} " =~ docker:missing ]] && echo "- Install Docker: see https://docs.docker.com/engine/install/"
		[[ " ${rows[*]} " =~ go:missing ]] && echo "- Install Go: https://go.dev/dl/"
		[[ " ${rows[*]} " =~ lsof:missing ]] && echo "- Install lsof: sudo apt-get install -y lsof"
		[[ " ${rows[*]} " =~ tput:missing ]] && echo "- Install ncurses-utils: sudo apt-get install -y ncurses-bin"
		[[ " ${rows[*]} " =~ api_port_${api_port}:in_use ]] && echo "- API port ${api_port} is in use. Stop the process or set VROOLI_API_PORT to another port."
		[[ " ${rows[*]} " =~ service_json:missing ]] && echo "- Missing .vrooli/service.json. Run 'vrooli setup' or create a config."
	fi
}

main "$@" 