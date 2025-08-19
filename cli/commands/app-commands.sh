#!/usr/bin/env bash
################################################################################
# Vrooli CLI - App Management Commands
# 
# Thin wrapper around the Vrooli App HTTP API
#
# Usage:
#   vrooli app <subcommand> [options]
#
################################################################################

set -euo pipefail

# API configuration
API_PORT="${VROOLI_API_PORT:-8092}"
API_BASE="http://localhost:${API_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Source utilities for display
# shellcheck disable=SC1091
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="$(cd "$CLI_DIR/../.." && pwd)"
# Unset source guards to ensure utilities are properly loaded when exec'd from CLI
unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED 2>/dev/null || true
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${CLI_DIR}/../lib/arg-parser.sh"
# shellcheck disable=SC1091
source "${CLI_DIR}/../lib/output-formatter.sh"

# Check if API is running
check_api() {
	if ! curl -s --connect-timeout 2 "${API_BASE}/health" >/dev/null 2>&1; then
		return 1  # Return failure silently, let callers handle it
	fi
	return 0
}

# Show help
show_app_help() {
	cat << EOF
🚀 Vrooli App Management Commands

USAGE:
    vrooli app <subcommand> [options]

RUNTIME COMMANDS:
    start <name>            Start a specific app (runs setup if needed, then develop)
    stop <name>             Stop a specific app
    restart <name>          Restart a specific app
    logs <name>             Show logs for a specific app

MANAGEMENT COMMANDS:
    list                    List all generated apps with their status
    status <name>           Show detailed status of a specific app
    protect <name>          Mark app as protected from auto-regeneration
    unprotect <name>        Remove protection from app
    diff <name>             Show changes from original generation
    regenerate <name>       Regenerate app from scenario
    backup <name>           Create manual backup of app
    restore <name>          Restore app from backup
    clean                   Remove old backups

OPTIONS:
    --help, -h              Show this help message
    --json                  Output in JSON format (alias for --format json)
    --format <type>         Output format: text, json
    --follow                Follow logs in real-time (for logs command)

EXAMPLES:
    vrooli app start research-assistant       # Start an app (setup + develop)
    vrooli app stop research-assistant        # Stop an app
    vrooli app restart research-assistant     # Restart an app
    vrooli app logs research-assistant        # Show app logs
    vrooli app logs research-assistant --follow # Follow logs real-time
    vrooli app list                           # List all apps
    vrooli app status research-assistant      # Check app status
    vrooli app protect research-assistant     # Protect from regeneration

For more information: https://docs.vrooli.com/cli/app-management
EOF
}

# List all apps
app_list() {
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
		show_app_help
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
	
	local apps
	local use_api=false
	
	# Try to use API if available
	if check_api; then
		local response
		response=$(curl -s --connect-timeout 2 --max-time 5 "${API_BASE}/apps" 2>/dev/null)
		
		if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
			apps=$(echo "$response" | jq -r '.data')
			use_api=true
		fi
	fi
	
	# Fallback to filesystem if API not available
	if [[ "$use_api" == "false" ]]; then
		# Build JSON array from filesystem
		apps="[]"
		local generated_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
		
		if [[ -d "$generated_dir" ]]; then
			for app_dir in "$generated_dir"/*; do
				[[ ! -d "$app_dir" ]] && continue
				
				local app_name
				app_name=$(basename "$app_dir")
				
				# Skip hidden directories and backups
				[[ "$app_name" == .* && "$app_name" != .tmp-* ]] && continue
				[[ "$app_name" == "backups" ]] && continue
				
				local has_git="false"
				[[ -d "$app_dir/.git" ]] && has_git="true"
				
				local customized="false"
				if [[ "$has_git" == "true" ]]; then
					# Check if there are uncommitted changes
					if cd "$app_dir" 2>/dev/null && git diff --quiet HEAD 2>/dev/null; then
						customized="false"
					else
						customized="true"
					fi
					cd - >/dev/null 2>&1
				fi
				
				local modified
				modified=$(stat -c '%Y' "$app_dir" 2>/dev/null || stat -f '%m' "$app_dir" 2>/dev/null || echo "0")
				modified=$(date -d "@$modified" '+%Y-%m-%d' 2>/dev/null || date -r "$modified" '+%Y-%m-%d' 2>/dev/null || echo "unknown")
				
				local protected="false"
				[[ -f "$app_dir/.protected" ]] && protected="true"
				
				# Add to apps array
				local app_json
				app_json=$(jq -n \
					--arg name "$app_name" \
					--arg path "$app_dir" \
					--arg has_git "$has_git" \
					--arg customized "$customized" \
					--arg protected "$protected" \
					--arg modified "$modified" \
					'{name: $name, path: $path, has_git: ($has_git == "true"), customized: ($customized == "true"), protected: ($protected == "true"), modified: $modified}')
				
				apps=$(echo "$apps" | jq --argjson app "$app_json" '. + [$app]')
			done
		fi
	fi
	
	# Source process manager for status checks
	local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
	if [[ -f "$process_manager" ]]; then
		# shellcheck disable=SC1090
		source "$process_manager" 2>/dev/null || true
	fi
	
	# Build rows for output
	local rows=()
	local app_count=0
	while IFS= read -r app; do
		# Skip empty lines
		[[ -z "$app" ]] && continue
		
		# Process app in subshell to prevent errors from breaking the loop
		(
			# Extract app data (with error handling)
			local name git_status modified runtime_state url_display
			name=$(echo "$app" | jq -r '.name' 2>/dev/null)
			[[ -z "$name" || "$name" == "null" ]] && exit 0
			
			modified=$(echo "$app" | jq -r '.modified' 2>/dev/null || echo "unknown")
			[[ "$modified" == "null" ]] && modified="unknown"
			modified=$(echo "$modified" | cut -d'T' -f1 2>/dev/null || echo "$modified")
		
		# Determine git status
		git_status="✅ Clean"
		local has_git customized protected
		has_git=$(echo "$app" | jq -r '.has_git' 2>/dev/null || echo "false")
		customized=$(echo "$app" | jq -r '.customized' 2>/dev/null || echo "false")
		protected=$(echo "$app" | jq -r '.protected' 2>/dev/null || echo "false")
		
		if [[ "$has_git" == "false" ]]; then
			git_status="⚠️  No Git"
		elif [[ "$customized" == "true" ]]; then
			git_status="🔧 Modified"
		fi
		
		if [[ "$protected" == "true" ]]; then
			git_status="$git_status 🔒"
		fi
		
		# Get runtime state using process manager
		runtime_state="○ Stopped"
		url_display="-"
		
		# Check if any background processes are running for this app
		if type -t pm::list >/dev/null 2>&1; then
			local has_running_processes=false
			while IFS= read -r process; do
				if [[ "$process" == vrooli.develop.* ]] && pm::is_running "$process" 2>/dev/null; then
					has_running_processes=true
					break
				fi
			done < <(pm::list 2>/dev/null)
			
			if [[ "$has_running_processes" == "true" ]]; then
				runtime_state="● Running"
				# Try to get port from app's config
				local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$name"
				local port=""
				if [[ -f "$app_path/package.json" ]]; then
					port=$(jq -r '.config.port // .port // ""' "$app_path/package.json" 2>/dev/null)
				fi
				if [[ -z "$port" ]] && [[ -f "$app_path/.env" ]]; then
					port=$(grep -E '^PORT=' "$app_path/.env" | cut -d= -f2 | tr -d '"' 2>/dev/null)
				fi
				if [[ -n "$port" ]]; then
					url_display="http://localhost:$port"
				fi
			fi
		fi
		
		rows+=("${name}:${runtime_state}:${url_display}:${git_status}:${modified}")
		) || true  # Continue even if subshell fails
		
		app_count=$((app_count + 1))
	done < <(echo "$apps" | jq -c '.[]' 2>/dev/null)
	
	local generated_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
	local count
	count=$(echo "$apps" | jq '. | length')
	
	# Count running apps
	local running_count=0
	if type -t pm::list >/dev/null 2>&1; then
		while IFS= read -r process; do
			if [[ "$process" == vrooli.*.develop ]] && pm::is_running "$process" 2>/dev/null; then
				running_count=$((running_count + 1))
			fi
		done < <(pm::list 2>/dev/null)
	fi
	
	# Format output using CLI formatter
	if [[ "$output_format" == "json" ]]; then
		# Build JSON structure
		local table_json
		table_json=$(format::table json "Name" "Runtime" "URL" "Git" "Modified" -- "${rows[@]}")
		
		cli::format_result json true "App list retrieved successfully" \
			"location" "$generated_dir" \
			"total" "$count" \
			"running" "$running_count" \
			"apps" "$table_json"
	else
		# Text output
		cli::format_header text "📦 Generated Applications"
		cli::format_output text kv location "$generated_dir"
		echo ""
		cli::format_table text "Name" "Runtime" "URL" "Git" "Modified" -- "${rows[@]}"
		echo ""
		cli::format_summary text total "$count" running "$running_count"
		echo "Use 'vrooli app start <name>' to start an app"
		echo "Use 'vrooli app logs <name>' to view logs"
	fi
}

# Show app status
app_status() {
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
		show_app_help
		return 0
	fi
	
	# Get app name from remaining arguments
	local args_array
	mapfile -t args_array < <(args_to_array "$remaining_args")
	local app_name="${args_array[0]:-}"
	
	# Check for unknown arguments
	for arg in "${args_array[@]:1}"; do
		if [[ "$arg" =~ ^- ]]; then
			cli::format_error "$output_format" "Unknown option: $arg"
			return 1
		fi
	done
	
	[[ -z "$app_name" ]] && { 
		cli::format_error "$output_format" "App name required" 
		return 1 
	}
	
	check_api || return 1
	
	local response
	response=$(curl -s --connect-timeout 2 --max-time 5 "${API_BASE}/apps/${app_name}")
	
	# Check if response is valid JSON first
	if ! echo "$response" | jq . >/dev/null 2>&1; then
		cli::format_error "$output_format" "App not found: $app_name"
		return 1
	fi
	
	if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
		local error_msg
		error_msg=$(echo "$response" | jq -r '.error // "Failed to get app status"' 2>/dev/null || echo "Failed to get app status")
		cli::format_error "$output_format" "$error_msg"
		return 1
	fi
	
	local app backups
	app=$(echo "$response" | jq -r '.data.app')
	backups=$(echo "$response" | jq -r '.data.backups')
	
	# Format output using CLI formatter
	if [[ "$output_format" == "json" ]]; then
		# Extract app data
		local name path modified has_git customized protected runtime_status
		name="$app_name"
		path=$(echo "$app" | jq -r '.path // ""')
		modified=$(echo "$app" | jq -r '.modified // ""')
		has_git=$(echo "$app" | jq -r '.has_git // false')
		customized=$(echo "$app" | jq -r '.customized // false')
		protected=$(echo "$app" | jq -r '.protected // false')
		runtime_status=$(echo "$app" | jq -r '.runtime_status // "unknown"')
		
		# Build app data JSON
		local app_json
		app_json=$(format::key_value json \
			"name" "$name" \
			"path" "$path" \
			"modified" "$modified" \
			"has_git" "$has_git" \
			"customized" "$customized" \
			"protected" "$protected" \
			"runtime_status" "$runtime_status")
		
		cli::format_result json true "App status retrieved successfully" \
			"app" "$app_json" \
			"backups" "$backups"
	else
		# Text output
		cli::format_header text "📦 App Status: $app_name"
		
		cli::format_output text kv \
			"Location" "$(echo "$app" | jq -r '.path')" \
			"Modified" "$(echo "$app" | jq -r '.modified' | cut -d'T' -f1)"
		
		# Git status
		if [[ $(echo "$app" | jq -r '.has_git') == "true" ]]; then
			echo ""
			echo "Git Repository:"
			echo "━━━━━━━━━━━━━━━"
			if [[ $(echo "$app" | jq -r '.customized') == "true" ]]; then
				echo "🔧 Has customizations"
			else
				echo "✅ Clean (no customizations)"
			fi
		else
			echo ""
			echo "⚠️  No Git repository"
		fi
		
		# Protection
		echo ""
		echo "Protection Status:"
		echo "━━━━━━━━━━━━━━━━━"
		if [[ $(echo "$app" | jq -r '.protected') == "true" ]]; then
			echo "🔒 Protected"
		else
			echo "🔓 Not Protected"
		fi
		
		# Backups
		echo ""
		echo "Backups:"
		echo "━━━━━━━━━━━━━━━"
		if [[ $backups -gt 0 ]]; then
			echo "💾 Available backups: $backups"
			echo "   Use 'vrooli app restore $app_name --from latest' to restore"
		else
			echo "📭 No backups found"
		fi
	fi
}

# Protect app
app_protect() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	check_api || return 1
	
	if curl -s -X POST "${API_BASE}/apps/${app_name}/protect" | jq -e '.success' >/dev/null 2>&1; then
		log::success "✅ App protected: $app_name"
		echo "This app will not be automatically regenerated during 'vrooli setup'"
	else
		log::error "Failed to protect app"
		return 1
	fi
}

# Unprotect app
app_unprotect() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	check_api || return 1
	
	if curl -s -X DELETE "${API_BASE}/apps/${app_name}/protect" | jq -e '.success' >/dev/null 2>&1; then
		log::success "✅ Protection removed: $app_name"
	else
		log::error "Failed to unprotect app"
		return 1
	fi
}

# Show diff (git-based, doesn't need API)
app_diff() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$app_name"
	
	if [[ ! -d "$app_path/.git" ]]; then
		log::error "No git repository found for: $app_name"
		return 1
	fi
	
	log::header "Changes in: $app_name"
	
	cd "$app_path" || return 1
	
	# Get initial commit
	local initial_commit
	initial_commit=$(git rev-list --max-parents=0 HEAD 2>/dev/null | head -1)
	
	echo ""
	echo "Comparing against initial generation"
	echo ""
	
	# Show diff stats
	git diff --stat "$initial_commit" HEAD
	
	cd - >/dev/null 2>&1 || true
}

# Regenerate app
app_regenerate() {
	local app_name="${1:-}"
	shift
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	local force=false no_backup=false
	while [[ $# -gt 0 ]]; do
		case "$1" in
			--force) force=true ;;
			--no-backup) no_backup=true ;;
			# Ignore shell redirections and operators that might be passed accidentally
			'2>&1'|'1>&2'|'&>'|'>'|'<'|'|'|'&&'|'||'|';') ;; # Silently ignore
			*) log::error "Unknown option: $1"; return 1 ;;
		esac
		shift
	done
	
	check_api || return 1
	
	# Check if protected
	local response
	response=$(curl -s "${API_BASE}/apps/${app_name}")
	
	if echo "$response" | jq -e '.data.app.protected' >/dev/null 2>&1; then
		if [[ $(echo "$response" | jq -r '.data.app.protected') == "true" ]] && [[ "$force" != "true" ]]; then
			log::error "App is protected: $app_name"
			echo "To regenerate anyway, use: vrooli app regenerate $app_name --force"
			return 1
		fi
	fi
	
	# Backup if customized and not skipped
	if [[ $(echo "$response" | jq -r '.data.app.customized // false') == "true" ]] && [[ "$no_backup" != "true" ]]; then
		log::info "Creating backup before regeneration..."
		curl -s -X POST "${API_BASE}/apps/${app_name}/backup" >/dev/null
	fi
	
	# Regenerate using scenario convert command
	log::info "Regenerating app: $app_name"
	
	# Use the CLI's own scenario convert command (dogfooding)
	if "${VROOLI_ROOT}/cli/vrooli" scenario convert "$app_name" --force; then
		log::success "✅ App regenerated successfully: $app_name"
		
		# Remove protection if forced
		if [[ "$force" == "true" ]]; then
			curl -s -X DELETE "${API_BASE}/apps/${app_name}/protect" >/dev/null 2>&1
		fi
	else
		log::error "Failed to regenerate app: $app_name"
		return 1
	fi
}

# Create backup
app_backup() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	check_api || return 1
	
	log::info "Creating backup: $app_name"
	
	local response
	response=$(curl -s -X POST "${API_BASE}/apps/${app_name}/backup")
	
	if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
		log::success "✅ Backup created: $(echo "$response" | jq -r '.data.backup')"
	else
		log::error "$(echo "$response" | jq -r '.error // "Failed to create backup"')"
		return 1
	fi
}

# Restore app
app_restore() {
	local app_name="${1:-}"
	shift
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	local backup_name=""
	while [[ $# -gt 0 ]]; do
		case "$1" in
			--from) backup_name="${2:-}"; shift 2 ;;
			# Ignore shell redirections and operators that might be passed accidentally
			'2>&1'|'1>&2'|'&>'|'>'|'<'|'|'|'&&'|'||'|';') ;; # Silently ignore
			*) log::error "Unknown option: $1"; return 1 ;;
		esac
	done
	
	[[ -z "$backup_name" ]] && { log::error "Backup name required (use --from)"; return 1; }
	
	check_api || return 1
	
	# Confirm restoration
	local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$app_name"
	if [[ -d "$app_path" ]]; then
		log::warning "This will replace the current app at: $app_path"
		flow::confirm "Continue with restoration?" || { log::info "Cancelled"; return 0; }
	fi
	
	log::info "Restoring from: $backup_name"
	
	local response
	response=$(curl -s -X POST "${API_BASE}/apps/${app_name}/restore" \
		-H "Content-Type: application/json" \
		-d "{\"backup\": \"$backup_name\"}")
	
	if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
		log::success "✅ $(echo "$response" | jq -r '.data')"
	else
		log::error "$(echo "$response" | jq -r '.error // "Failed to restore"')"
		return 1
	fi
}

# Clean backups (simplified - just removes old backups locally)
app_clean() {
	local days="${1:-30}"
	
	log::header "Cleaning Old Backups"
	echo "Removing backups older than $days days..."
	
	local backup_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}/.backups"
	if [[ -d "$backup_dir" ]]; then
		find "$backup_dir" -type f -mtime "+$days" -print -delete 2>/dev/null || true
		log::success "Cleanup complete"
	else
		log::warning "No backups directory found: $backup_dir"
	fi
}

# Logs
app_logs() {
	local app_name="${1:-}"
	local follow=false
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	# Parse options
	shift || true
	while [[ $# -gt 0 ]]; do
		case "$1" in
			--follow) follow=true ;;
			*) log::error "Unknown option: $1"; return 1 ;;
		esac
		shift
	done
	
	# Uses process manager logs if available
	if ! type -t pm::list >/dev/null 2>&1; then
		log::error "Process manager not available"
		return 1
	fi
	
	local found=false
	while IFS= read -r process; do
		if [[ "$process" == vrooli.develop.$app_name ]]; then
			found=true
			log::info "Showing logs for $app_name"
			if [[ "$follow" == "true" ]]; then
				pm::logs --follow "$process"
			else
				pm::logs "$process" 20
			fi
		fi
	done < <(pm::list 2>/dev/null)
	
	if [[ "$found" == "false" ]]; then
		log::warning "No logs found for $app_name"
	fi
	
	# Tail all logs if following
	if [[ "$follow" == "true" ]]; then
		while IFS= read -r process; do
			if [[ "$process" == vrooli.develop.* ]]; then
				pm::logs "$process" 20
			fi
		done
		
		if [[ "$follow" != "true" ]]; then
			echo ""
			echo "Use --follow to see logs in real-time"
		fi
	fi
}

# Main handler
main() {
	if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
		show_app_help
		return 0
	fi
	
	local subcommand="$1"
	shift
	
	case "$subcommand" in
		# Runtime commands
		start) app_start "$@" ;;
		stop) app_stop "$@" ;;
		restart) app_restart "$@" ;;
		logs) app_logs "$@" ;;
		
		# Management commands
		list) app_list "$@" ;;
		status) app_status "$@" ;;
		protect) app_protect "$@" ;;
		unprotect) app_unprotect "$@" ;;
		diff) app_diff "$@" ;;
		regenerate) app_regenerate "$@" ;;
		backup) app_backup "$@" ;;
		restore) app_restore "$@" ;;
		clean) app_clean "$@" ;;
		*)
			log::error "Unknown app command: $subcommand"
			echo ""
			show_app_help
			return 1
			;;
	esac
}

main "$@"