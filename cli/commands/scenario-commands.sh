#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Scenario Management Commands
# 
# Thin wrapper around the Vrooli Scenario HTTP API
#
# Usage:
#   vrooli scenario <subcommand> [options]
#
################################################################################

set -euo pipefail

# API configuration
API_PORT="${VROOLI_API_PORT:-8090}"
API_BASE="http://localhost:${API_PORT}"

# Get CLI directory
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
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

# Check if API is running
check_api() {
	if ! curl -s --connect-timeout 2 --max-time 5 "${API_BASE}/health" >/dev/null 2>&1; then
		log::error "Scenario API is not running"
		echo "Start it with: ${var_ROOT_DIR}/api/start.sh"
		return 1
	fi
}

# Show help for scenario commands
show_scenario_help() {
	cat << EOF
🚀 Vrooli Scenario Commands

USAGE:
    vrooli scenario <subcommand> [options]

SUBCOMMANDS:
    list                    List all available scenarios
    info <name>             Show detailed information about a scenario
    validate <name>         Validate scenario configuration
    convert <name>          Convert scenario to standalone app
    convert-all             Convert all enabled scenarios
    enable <name>           Enable scenario in catalog
    disable <name>          Disable scenario in catalog

OPTIONS:
    --help, -h              Show this help message
    --verbose, -v           Show detailed output
    --json                  Output in JSON format (alias for --format json)
    --format <type>         Output format: text, json

EXAMPLES:
    vrooli scenario list                      # List all scenarios
    vrooli scenario list --json               # List as JSON
    vrooli scenario info research-assistant   # Show scenario details
    vrooli scenario validate my-scenario      # Validate configuration
    vrooli scenario convert my-scenario       # Generate app from scenario
    vrooli scenario convert-all --force       # Regenerate all apps

For more information: https://docs.vrooli.com/cli/scenarios
EOF
}

# List all scenarios (output delegated to format.sh)
scenario_list() {
	check_api || return 1
	
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
		show_scenario_help
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
	
	local response
	response=$(curl -s --connect-timeout 2 --max-time 5 "${API_BASE}/scenarios")
	
	if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
		log::error "Failed to list scenarios"
		return 1
	fi
	
	local rows=()
	echo "" 1>/dev/null
	
	# Build rows for format.sh
	while IFS= read -r scenario; do
		local name enabled category description enabled_display
		name=$(echo "$scenario" | jq -r '.name')
		enabled=$(echo "$scenario" | jq -r '.enabled')
		category=$(echo "$scenario" | jq -r '.category // "general"')
		description=$(echo "$scenario" | jq -r '.description // "No description"' | cut -c1-60)
		
		if [[ "$output_format" == "json" ]]; then
			enabled_display="$enabled"
		else
			enabled_display=$([ "$enabled" == "true" ] && echo "✅" || echo "❌")
		fi
		rows+=("${name}:${enabled_display}:${category}:${description}")
	done < <(echo "$response" | jq -c '.data[]')
	
	format::table "$output_format" "Name" "Enabled" "Category" "Description" -- "${rows[@]}"
	
	if [[ "$output_format" == "text" ]]; then
		log::header "Available Scenarios"
		local total_count enabled_count
		total_count=$(echo "$response" | jq '.data | length')
		enabled_count=$(echo "$response" | jq '[.data[] | select(.enabled == true)] | length')
		echo ""
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
		echo "Total: $total_count scenarios ($enabled_count enabled)"
	fi
}

# Show detailed info about a scenario
scenario_info() {
	check_api || return 1
	
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
		show_scenario_help
		return 0
	fi
	
	# Get scenario name from remaining arguments
	local args_array
	mapfile -t args_array < <(args_to_array "$remaining_args")
	local scenario_name="${args_array[0]:-}"
	
	# Check for unknown arguments
	for arg in "${args_array[@]:1}"; do
		if [[ "$arg" =~ ^- ]]; then
			cli::format_error "$output_format" "Unknown option: $arg"
			return 1
		fi
	done
	
	if [[ -z "$scenario_name" ]]; then
		cli::format_error "$output_format" "Scenario name required"
		echo "Usage: vrooli scenario info <scenario-name> [--json]"
		return 1
	fi
	
	local response
	response=$(curl -s --connect-timeout 2 --max-time 5 "${API_BASE}/scenarios/${scenario_name}")
	
	if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
		log::error "$(echo "$response" | jq -r '.error // "Failed to get scenario"')"
		return 1
	fi
	
	local scenario_data
	scenario_data=$(echo "$response" | jq -r '.data')
	
	local name enabled description category location has_service path
	name=$(echo "$scenario_data" | jq -r '.scenario.name')
	enabled=$(echo "$scenario_data" | jq -r '.scenario.enabled')
	description=$(echo "$scenario_data" | jq -r '.scenario.description // "No description"')
	category=$(echo "$scenario_data" | jq -r '.scenario.category // "general"')
	location=$(echo "$scenario_data" | jq -r '.scenario.location')
	has_service=$(echo "$scenario_data" | jq -r '.has_service')
	path=$(echo "$scenario_data" | jq -r '.path')
	
	if [[ "$output_format" == "json" ]]; then
		# Build a structured JSON object via format helpers
		local base
		base=$(format::key_value json \
			name "$name" \
			enabled "$enabled" \
			category "$category" \
			location "$location" \
			path "$path")
		# Add description and has_service
		local desc_json
		desc_json=$(format::key_value json description "$description" "has_service" "$has_service")
		local combined
		combined=$(format::json_object info "$base" extra "$desc_json")
		echo "$combined" | jq '.' 2>/dev/null || echo "$combined"
		return 0
	fi
	
	# Text output
	if [[ "$output_format" == "text" ]]; then
		log::header "Scenario: $scenario_name"
		echo ""
		echo "📍 Location: $path"
		echo "📁 Category: $category"
		echo "🔧 Status: $([ "$enabled" == "true" ] && echo "✅ Enabled" || echo "❌ Disabled")"
		echo "📝 Description: $description"
		
		if [[ "$has_service" == "true" ]]; then
			echo ""
			echo "✅ Has service.json configuration"
		else
			echo ""
			log::warning "No service.json found"
		fi
		
		# Check for generated app
		local app_path="$HOME/generated-apps/$scenario_name"
		echo ""
		echo "Generated App:"
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
		if [[ -d "$app_path" ]]; then
			echo "✅ App exists: $app_path"
		else
			echo "❌ Not generated yet"
			echo "   Run: vrooli scenario convert $scenario_name"
		fi
	fi
}

# Validate a scenario
scenario_validate() {
	local scenario_name="${1:-}"
	
	if [[ -z "$scenario_name" ]]; then
		log::error "Scenario name required"
		echo "Usage: vrooli scenario validate <scenario-name>"
		return 1
	fi
	
	check_api || return 1
	
	log::info "Validating scenario: $scenario_name"
	
	local response
	response=$(curl -s -X POST "${API_BASE}/scenarios/${scenario_name}/validate")
	
	if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
		log::error "Validation failed"
		return 1
	fi
	
	local result
	result=$(echo "$response" | jq -r '.data')
	
	if [[ $(echo "$result" | jq -r '.valid') == "true" ]]; then
		log::success "✅ Scenario is valid"
	else
		log::error "❌ Scenario is invalid"
	fi
}

# Convert a scenario to an app
scenario_convert() {
	local scenario_name="${1:-}"
	shift || true
	[[ -z "$scenario_name" ]] && { log::error "Scenario name required"; return 1; }
	
	check_api || return 1
	
	local force=false
	while [[ $# -gt 0 ]]; do
		case "$1" in
			--force) force=true ;;
			*) log::error "Unknown option: $1"; return 1 ;;
		esac
		shift
	done
	
	log::info "Converting scenario to app: $scenario_name"
	local response
	response=$(curl -s -X POST "${API_BASE}/scenarios/${scenario_name}/convert" \
		-H "Content-Type: application/json" \
		-d "{\"force\": ${force}}")
	
	if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
		log::success "✅ $(echo "$response" | jq -r '.data')"
	else
		log::error "$(echo "$response" | jq -r '.error // "Conversion failed"')"
		return 1
	fi
}

# Main handler
main() {
	if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
		show_scenario_help
		return 0
	fi
	
	local subcommand="$1"; shift
	case "$subcommand" in
		list) scenario_list "$@" ;;
		info) scenario_info "$@" ;;
		validate) scenario_validate "$@" ;;
		convert) scenario_convert "$@" ;;
		convert-all)
			check_api || return 1
			log::info "Converting all enabled scenarios..."
			curl -s -X POST "${API_BASE}/scenarios/convert-all" | jq -r '.data // .error'
			;;
		enable)
			check_api || return 1
			[[ -z "${1:-}" ]] && { log::error "Scenario name required"; return 1; }
			curl -s -X POST "${API_BASE}/scenarios/${1}/enable" | jq -r '.data // .error'
			;;
		disable)
			check_api || return 1
			[[ -z "${1:-}" ]] && { log::error "Scenario name required"; return 1; }
			curl -s -X POST "${API_BASE}/scenarios/${1}/disable" | jq -r '.data // .error'
			;;
		*)
			log::error "Unknown scenario command: $subcommand"
			echo ""
			show_scenario_help
			return 1
			;;
	esac
}

main "$@"