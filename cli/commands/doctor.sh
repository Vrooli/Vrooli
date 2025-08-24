#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Doctor / Preflight Checks
#
# Verifies local environment prerequisites and common tools, emitting either
# human-readable text or JSON using the shared format.sh helpers.
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/cli/commands"
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${CLI_DIR}/../lib/arg-parser.sh"
# shellcheck disable=SC1091
source "${CLI_DIR}/../lib/output-formatter.sh"

show_help() {
	cat << EOF
🩺 Vrooli Doctor - Environment Preflight

USAGE:
    vrooli doctor [--json|--format <type>]

Checks:
    - jq, curl, git, docker, go, lsof, tput
    - API port availability (VROOLI_API_PORT)
    - Config presence: ".vrooli/service.json"

OPTIONS:
    --json              Emit JSON output (alias for --format json)
    --format <type>     Output format: text, json
    --help,-h           Show this help message
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
	# Parse common arguments
	local parsed_args
	parsed_args=$(parse_combined_args "$@")
	if [[ $? -ne 0 ]]; then
		return 1
	fi
	
	local verbose
	verbose=$(extract_arg "$parsed_args" "verbose")
	local help_requested
	help_requested=$(extract_arg "$parsed_args" "help")
	local output_format
	output_format=$(extract_arg "$parsed_args" "format")
	local remaining_args
	remaining_args=$(extract_arg "$parsed_args" "remaining")
	
	# Handle help request
	if [[ "$help_requested" == "true" ]]; then
		show_help
		return 0
	fi
	
	# Check for unknown arguments
	local args_array
	mapfile -t args_array < <(args_to_array "$remaining_args")
	
	for arg in "${args_array[@]}"; do
		if [[ "$arg" =~ ^- ]]; then
			cli::format_error "$output_format" "Unknown option: $arg"
			return 1
		fi
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
	local api_port="${VROOLI_API_PORT:-8092}"
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
	
	# Determine overall health status
	local has_critical_errors=false
	local has_warnings=false
	
	# Check for critical errors (missing essential tools)
	for row in "${rows[@]}"; do
		local check_name="${row%%:*}"
		local status="${row#*:}"
		
		# Critical errors: missing essential tools
		if [[ "$status" == "missing" ]] && [[ "$check_name" =~ ^(jq|curl|git|docker)$ ]]; then
			has_critical_errors=true
		fi
		
		# Warnings: missing optional tools or port in use
		if [[ "$status" == "missing" ]] || [[ "$status" == "in_use" ]]; then
			has_warnings=true
		fi
	done
	
	# Output
	if [[ "$output_format" == "json" ]]; then
		format::output json "table" "check" "status" -- "${rows[@]}"
	else
		log::header "Vrooli Doctor"
		format::output "$output_format" "table" "Check" "Status" -- "${rows[@]}"
		
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
	
	# Return appropriate exit code
	if [[ "$has_critical_errors" == "true" ]]; then
		return 1  # Critical errors
	elif [[ "$has_warnings" == "true" ]]; then
		return 0  # Warnings only (success with warnings)
	else
		return 0  # All good
	fi
}

main "$@" 