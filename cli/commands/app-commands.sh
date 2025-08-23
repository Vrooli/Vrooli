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
    stop-all                Stop all running apps
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
    vrooli app stop-all                       # Stop all running apps
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
		
		# Extract app data (with error handling)
		local name git_status modified runtime_state url_display
		name=$(echo "$app" | jq -r '.name' 2>/dev/null)
		[[ -z "$name" || "$name" == "null" ]] && continue
		
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
			local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$name"
			
			# Check all process manager entries for this app
			while IFS= read -r process; do
				# Check various process naming patterns:
				# 1. Direct app process: vrooli.develop.{app-name}
				# 2. Service processes: vrooli.develop.start-{service}
				# 3. Generic service processes that might belong to this app
				
				local check_process=false
				
				# Direct match for app name
				if [[ "$process" == "vrooli.develop.$name" ]] || \
				   [[ "$process" == "vrooli."*".$name" ]] || \
				   [[ "$process" == "vrooli.$name."* ]]; then
					check_process=true
				elif [[ "$process" == "vrooli.develop."* ]]; then
					# For any develop process, check if it belongs to this app
					# by examining the process info file
					local info_file="$HOME/.vrooli/processes/$process/info"
					if [[ -f "$info_file" ]]; then
						local working_dir
						working_dir=$(grep '^working_dir=' "$info_file" 2>/dev/null | cut -d= -f2- || echo "")
						if [[ "$working_dir" == "$app_path" ]] || [[ "$working_dir" == *"/$name" ]]; then
							check_process=true
						fi
					fi
				fi
				
				# Check if this process is actually running
				if [[ "$check_process" == "true" ]]; then
					if pm::is_running "$process" 2>/dev/null; then
						has_running_processes=true
						break
					fi
				fi
			done < <(pm::list 2>/dev/null)
			
			if [[ "$has_running_processes" == "true" ]]; then
				runtime_state="● Running"
				# Try to get port from app's config
				local port=""
				if [[ -f "$app_path/package.json" ]]; then
					port=$(jq -r '.config.port // .port // ""' "$app_path/package.json" 2>/dev/null)
				fi
				if [[ -z "$port" ]] && [[ -f "$app_path/.env" ]]; then
					port=$(grep -E '^PORT=' "$app_path/.env" | cut -d= -f2 | tr -d '"' 2>/dev/null)
				fi
				if [[ -z "$port" ]] && [[ -f "$app_path/.vrooli/service.json" ]]; then
					# Try to get port from service.json
					port=$(jq -r '.ports.ui.fixed // .ports.api.fixed // ""' "$app_path/.vrooli/service.json" 2>/dev/null)
				fi
				if [[ -n "$port" ]]; then
					url_display="http://localhost:$port"
				fi
			fi
		fi
		
		rows+=("${name}:${runtime_state}:${url_display}:${git_status}:${modified}")
		
		app_count=$((app_count + 1))
	done < <(echo "$apps" | jq -c '.[]' 2>/dev/null)
	
	local generated_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
	local count
	count=$(echo "$apps" | jq '. | length')
	
	# Count running apps (unique apps, not individual processes)
	local running_count=0
	local -A running_apps  # Associative array to track unique apps
	
	if type -t pm::list >/dev/null 2>&1; then
		while IFS= read -r process; do
			# Only count develop processes that are actually running
			if [[ "$process" == "vrooli.develop."* ]]; then
				if pm::is_running "$process" 2>/dev/null; then
					# Try to determine which app this process belongs to
					local app_name=""
					
					# Check if it's a direct app process
					if [[ "$process" =~ ^vrooli\.develop\.([^.]+)$ ]]; then
						app_name="${BASH_REMATCH[1]}"
					else
						# For service processes, check the working directory
						local info_file="$HOME/.vrooli/processes/$process/info"
						if [[ -f "$info_file" ]]; then
							local working_dir
							working_dir=$(grep '^working_dir=' "$info_file" 2>/dev/null | cut -d= -f2- || echo "")
							if [[ -n "$working_dir" ]]; then
								app_name=$(basename "$working_dir")
							fi
						fi
					fi
					
					# Track unique apps
					if [[ -n "$app_name" ]] && [[ -z "${running_apps[$app_name]:-}" ]]; then
						running_apps["$app_name"]=1
						running_count=$((running_count + 1))
					fi
				fi
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
# Start an app
app_start() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	# Parse options
	shift || true
	local skip_setup=false
	local fast_mode=false
	while [[ $# -gt 0 ]]; do
		case "$1" in
			--skip-setup) skip_setup=true ;;
			--fast) fast_mode=true ;;
			*) break ;;
		esac
		shift
	done
	
	local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$app_name"
	
	# Validate app exists
	if [[ ! -d "$app_path" ]]; then
		log::error "App not found: $app_name"
		return 1
	fi
	
	# Source process manager if not already available
	if ! type -t pm::is_running >/dev/null 2>&1; then
		local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
		if [[ -f "$process_manager" ]]; then
			# shellcheck disable=SC1090
			source "$process_manager" || {
				log::error "Failed to load process manager: $?"
				return 1
			}
		else
			log::error "Process manager not found at: $process_manager"
			return 1
		fi
	fi
	
	# Check if already running (check both exact match and child processes)
	local is_running=false
	
	# Check for exact match
	if pm::is_running "vrooli.develop.$app_name" 2>/dev/null; then
		is_running=true
	fi
	
	# Also check for any child processes of this app
	if ! $is_running && type -t pm::list >/dev/null 2>&1; then
		while IFS= read -r process; do
			# Check if process starts with app's namespace or contains app name
			if [[ "$process" == "vrooli.develop.$app_name" ]] || \
			   [[ "$process" == "vrooli.develop."*".$app_name" ]] || \
			   [[ "$process" == "vrooli.$app_name."* ]]; then
				is_running=true
				break
			fi
		done < <(pm::list 2>/dev/null)
	fi
	
	if $is_running; then
		log::warning "App already running: $app_name"
		return 0
	fi
	
	# Run setup if needed (unless skipped)
	if [[ "$skip_setup" == "false" ]] && [[ -f "$app_path/scripts/manage.sh" ]]; then
		log::info "Running setup for $app_name..."
		local setup_cmd="./scripts/manage.sh setup --yes yes"
		[[ "$fast_mode" == "true" ]] && setup_cmd="$setup_cmd --fast"
		if ! (cd "$app_path" && eval "$setup_cmd" 2>&1 | head -50); then
			log::warning "Setup had issues, continuing anyway..."
		fi
	fi
	
	# Start via develop phase
	log::info "Starting $app_name..."
	if [[ -f "$app_path/scripts/manage.sh" ]]; then
		# Extract port configuration from service.json and set environment variables
		local service_json="$app_path/.vrooli/service.json"
		local api_port_start=""
		local ui_port_start=""
		if [[ -f "$service_json" ]]; then
			# Get API port configuration
			local api_port_range=$(jq -r '.ports.api.range // ""' "$service_json" 2>/dev/null)
			if [[ -n "$api_port_range" ]]; then
				api_port_start=$(echo "$api_port_range" | cut -d'-' -f1)
			fi
			
			# Get UI port configuration
			local ui_port_range=$(jq -r '.ports.ui.range // ""' "$service_json" 2>/dev/null)
			if [[ -n "$ui_port_range" ]]; then
				ui_port_start=$(echo "$ui_port_range" | cut -d'-' -f1)
			fi
		fi
		
		# Start the app using process manager with port environment variables
		# Export ports to current environment for proper propagation
		if [[ -n "$api_port_start" ]]; then
			export SERVICE_PORT="${api_port_start}"
		fi
		if [[ -n "$ui_port_start" ]]; then
			export UI_PORT="${ui_port_start}"
		fi
		
		# Build develop command without inline exports (they're now in environment)
		local develop_cmd="./scripts/manage.sh develop"
		[[ "$fast_mode" == "true" ]] && develop_cmd="$develop_cmd --fast"
		
		# Pass environment through process manager
		if pm::start "vrooli.develop.$app_name" \
			"cd '$app_path' && SERVICE_PORT='${SERVICE_PORT:-}' UI_PORT='${UI_PORT:-}' $develop_cmd" \
			"$app_path"; then
			log::success "Started $app_name"
			
			# Wait briefly and verify it's running
			sleep 2
			
			# Check if app or any of its child processes are running
			local app_running=false
			if pm::is_running "vrooli.develop.$app_name" 2>/dev/null; then
				app_running=true
			elif type -t pm::list >/dev/null 2>&1; then
				# Check for any processes related to this app
				while IFS= read -r process; do
					if [[ "$process" == "vrooli.develop.$app_name" ]] || \
					   [[ "$process" == "vrooli.develop."*".$app_name" ]] || \
					   [[ "$process" == "vrooli.$app_name."* ]] || \
					   [[ "$process" == "vrooli.develop.start-"* && -d "$app_path" ]]; then
						# Additional check: if it's a generic process name, verify it's from this app's directory
						if [[ "$process" == "vrooli.develop.start-"* ]]; then
							local process_info
							process_info=$(pm::status "$process" 2>/dev/null || echo "")
							if [[ "$process_info" == *"$app_path"* ]]; then
								app_running=true
								break
							fi
						else
							app_running=true
							break
						fi
					fi
				done < <(pm::list 2>/dev/null)
			fi
			
			if $app_running; then
				log::info "App $app_name is running"
				
				# Try to get port from app's config
				local port=""
				if [[ -f "$app_path/package.json" ]]; then
					port=$(jq -r '.config.port // .port // ""' "$app_path/package.json" 2>/dev/null)
				fi
				if [[ -z "$port" ]] && [[ -f "$app_path/.env" ]]; then
					port=$(grep -E '^PORT=' "$app_path/.env" | cut -d= -f2 | tr -d '"' 2>/dev/null)
				fi
				if [[ -z "$port" ]] && [[ -f "$app_path/.vrooli/service.json" ]]; then
					# Try to get port from service.json
					port=$(jq -r '.ports.ui.fixed // .ports.api.fixed // ""' "$app_path/.vrooli/service.json" 2>/dev/null)
				fi
				if [[ -n "$port" ]]; then
					echo "  URL: http://localhost:$port"
				fi
				return 0
			else
				log::warning "App started but may not be running correctly"
				return 1
			fi
		else
			log::error "Failed to start $app_name"
			return 1
		fi
	else
		log::error "No manage.sh found for $app_name"
		return 1
	fi
}

# Stop an app
app_stop() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	# Source process manager if not already available
	if ! type -t pm::stop >/dev/null 2>&1; then
		local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
		if [[ -f "$process_manager" ]]; then
			# shellcheck disable=SC1090
			source "$process_manager" 2>/dev/null || {
				log::error "Failed to load process manager"
				return 1
			}
		else
			log::error "Process manager not found"
			return 1
		fi
	fi
	
	log::info "Stopping $app_name..."
	if pm::stop "vrooli.develop.$app_name"; then
		log::success "Stopped $app_name"
		return 0
	else
		log::warning "App may not have been running: $app_name"
		return 0  # Don't fail if app wasn't running
	fi
}

# Stop all apps
app_stop_all() {
	# Source process manager if not already available
	if ! type -t pm::list >/dev/null 2>&1; then
		local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
		if [[ -f "$process_manager" ]]; then
			# shellcheck disable=SC1090
			source "$process_manager" 2>/dev/null || {
				log::error "Failed to load process manager"
				return 1
			}
		else
			log::error "Process manager not found"
			return 1
		fi
	fi
	
	log::info "Stopping all running apps in parallel..."
	
	local stopped_count=0
	local orphaned_count=0
	local -A stopped_apps  # Track unique apps to avoid duplicate messages
	local -A orphaned_apps  # Track orphaned processes
	local -a managed_processes=()  # Array to store managed processes to stop
	local -a orphaned_processes=()  # Array to store orphaned processes to kill
	local -a background_jobs=()  # Track background job PIDs
	
	# Phase 1: Collect all managed processes that need stopping
	while IFS= read -r process; do
		# Only stop develop processes
		if [[ "$process" == "vrooli.develop."* ]]; then
			if pm::is_running "$process" 2>/dev/null; then
				managed_processes+=("$process")
			fi
		fi
	done < <(pm::list 2>/dev/null)
	
	# Phase 2: Collect all orphaned app processes
	local generated_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
	if [[ -d "$generated_dir" ]]; then
		while IFS= read -r line; do
			# Parse ps output: PID CMD
			local pid=$(echo "$line" | awk '{print $1}')
			local cmd=$(echo "$line" | awk '{$1=""; print $0}' | sed 's/^ *//')
			
			# Skip if we can't read the process working directory
			local cwd=""
			cwd=$(readlink "/proc/$pid/cwd" 2>/dev/null || echo "")
			
			# Check if process is related to generated-apps (either by working dir or command)
			local app_name=""
			local is_app_process=false
			
			# Method 1: Check if running from generated-apps directory
			if [[ "$cwd" == "$generated_dir"* ]] || [[ "$cwd" == *"/generated-apps/"* ]]; then
				if [[ "$cwd" =~ $generated_dir/([^/]+) ]]; then
					app_name="${BASH_REMATCH[1]}"
					is_app_process=true
				elif [[ "$cwd" =~ /generated-apps/([^/]+) ]]; then
					app_name="${BASH_REMATCH[1]}"
					is_app_process=true
				fi
			fi
			
			# Method 2: Check if command references generated-apps files
			if [[ -z "$app_name" ]] && [[ "$cmd" == *"/generated-apps/"* ]]; then
				if [[ "$cmd" =~ /generated-apps/([^/]+)/ ]]; then
					app_name="${BASH_REMATCH[1]}"
					is_app_process=true
				fi
			fi
			
			# Method 3: Check if running from trashed app directories
			if [[ -z "$app_name" ]] && [[ "$cwd" == *"/Trash/files/"* ]]; then
				if [[ "$cwd" =~ /Trash/files/([^/]+_[0-9]{8}_[0-9]{6})/ ]]; then
					local trashed_name="${BASH_REMATCH[1]}"
					if [[ "$trashed_name" =~ ^(.+)_[0-9]{8}_[0-9]{6}$ ]]; then
						app_name="${BASH_REMATCH[1]}"
						is_app_process=true
					fi
				fi
			fi
			
			# Only target app-related processes
			if [[ "$is_app_process" == "true" ]] && [[ -n "$app_name" ]]; then
				# Check if this looks like an app-related process
				local is_likely_app_process=false
				
				# Node.js and JavaScript build tools
				if [[ "$cmd" == *"node"* ]] || [[ "$cmd" == *"npm"* ]] || [[ "$cmd" == *"npx"* ]] || \
				   [[ "$cmd" == *"pnpm"* ]] || [[ "$cmd" == *"yarn"* ]] || [[ "$cmd" == *"bun"* ]]; then
					is_likely_app_process=true
				# JavaScript build/dev servers
				elif [[ "$cmd" == *"vite"* ]] || [[ "$cmd" == *"next"* ]] || [[ "$cmd" == *"webpack"* ]] || \
				     [[ "$cmd" == *"parcel"* ]] || [[ "$cmd" == *"rollup"* ]] || [[ "$cmd" == *"esbuild"* ]]; then
					is_likely_app_process=true
				# Python web servers
				elif [[ "$cmd" == *"python"*"manage.py"* ]] || [[ "$cmd" == *"flask"* ]] || \
				     [[ "$cmd" == *"django"* ]] || [[ "$cmd" == *"uvicorn"* ]] || [[ "$cmd" == *"gunicorn"* ]]; then
					is_likely_app_process=true
				# Ruby/Rails
				elif [[ "$cmd" == *"ruby"* ]] || [[ "$cmd" == *"rails"* ]] || [[ "$cmd" == *"rake"* ]] || \
				     [[ "$cmd" == *"bundle"* ]] || [[ "$cmd" == *"puma"* ]]; then
					is_likely_app_process=true
				# Shell scripts for app management
				elif [[ "$cmd" == *"manage.sh"* ]] || [[ "$cmd" == *"start.sh"* ]] || \
				     [[ "$cmd" == *"develop.sh"* ]] || [[ "$cmd" == *"server.sh"* ]]; then
					is_likely_app_process=true
				# Bash processes running develop scripts
				elif [[ "$cmd" == *"bash"* ]] && [[ "$cmd" == *"develop"* ]]; then
					is_likely_app_process=true
				fi
				
				if [[ "$is_likely_app_process" == "true" ]]; then
					if [[ -z "${orphaned_apps[$app_name]:-}" ]]; then
						orphaned_processes+=("$pid:$app_name")
						orphaned_apps["$app_name"]=1
					fi
				fi
			fi
		done < <(ps -eo pid,cmd --no-headers | grep -v grep)
	fi
	
	# Phase 3: Stop managed processes in parallel
	if [[ ${#managed_processes[@]} -gt 0 ]]; then
		log::info "Stopping ${#managed_processes[@]} managed processes in parallel..."
		for process in "${managed_processes[@]}"; do
			{
				# Extract app name for reporting
				local app_name=""
				if [[ "$process" =~ ^vrooli\.develop\.([^.]+)$ ]]; then
					app_name="${BASH_REMATCH[1]}"
				elif [[ "$process" =~ ^vrooli\.develop\.start-(.+)$ ]]; then
					local info_file="$HOME/.vrooli/processes/$process/info"
					if [[ -f "$info_file" ]]; then
						local working_dir
						working_dir=$(grep '^working_dir=' "$info_file" 2>/dev/null | cut -d= -f2- || echo "")
						if [[ -n "$working_dir" ]]; then
							app_name=$(basename "$working_dir")
						fi
					fi
				fi
				
				if pm::stop "$process" 2>/dev/null; then
					if [[ -n "$app_name" ]]; then
						echo "MANAGED_SUCCESS:$app_name"
					fi
				fi
			} &
			background_jobs+=($!)
		done
	fi
	
	# Phase 4: Kill orphaned processes in parallel
	if [[ ${#orphaned_processes[@]} -gt 0 ]]; then
		log::info "Stopping ${#orphaned_processes[@]} orphaned processes in parallel..."
		for orphan_info in "${orphaned_processes[@]}"; do
			{
				local pid="${orphan_info%%:*}"
				local app_name="${orphan_info#*:}"
				
				log::warning "Found orphaned process for $app_name (PID: $pid)"
				if kill -TERM "$pid" 2>/dev/null; then
					# Wait briefly for graceful shutdown
					sleep 1
					if kill -0 "$pid" 2>/dev/null; then
						# Force kill if still running
						kill -KILL "$pid" 2>/dev/null
					fi
					echo "ORPHANED_SUCCESS:$app_name"
				fi
			} &
			background_jobs+=($!)
		done
	fi
	
	# Phase 5: Wait for all background jobs to complete and collect results
	if [[ ${#background_jobs[@]} -gt 0 ]]; then
		log::info "Waiting for all ${#background_jobs[@]} stop operations to complete..."
		for job_pid in "${background_jobs[@]}"; do
			if wait "$job_pid" 2>/dev/null; then
				# Job completed successfully, check for success messages
				:
			fi
		done 2>&1 | while IFS= read -r line; do
			if [[ "$line" == "MANAGED_SUCCESS:"* ]]; then
				local app_name="${line#MANAGED_SUCCESS:}"
				if [[ -z "${stopped_apps[$app_name]:-}" ]]; then
					log::success "Stopped $app_name (managed)"
					stopped_apps["$app_name"]=1
					stopped_count=$((stopped_count + 1))
				fi
			elif [[ "$line" == "ORPHANED_SUCCESS:"* ]]; then
				local app_name="${line#ORPHANED_SUCCESS:}"
				log::success "Stopped $app_name (orphaned)"
				orphaned_count=$((orphaned_count + 1))
			fi
		done
	fi
	
	# Recalculate counts since the subshell above doesn't update the main shell variables
	stopped_count=${#managed_processes[@]}
	orphaned_count=${#orphaned_processes[@]}
	
	# Phase 6: Report results
	local total_stopped=$((stopped_count + orphaned_count))
	if [[ $total_stopped -eq 0 ]]; then
		log::info "No apps were running"
	else
		if [[ $stopped_count -gt 0 ]] && [[ $orphaned_count -gt 0 ]]; then
			log::success "Stopped $total_stopped app(s) in parallel: $stopped_count managed, $orphaned_count orphaned"
		elif [[ $stopped_count -gt 0 ]]; then
			log::success "Stopped $stopped_count managed app(s) in parallel"
		else
			log::success "Stopped $orphaned_count orphaned app(s) in parallel"
		fi
	fi
	
	return 0
}

# Restart an app
app_restart() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	log::info "Restarting $app_name..."
	
	# Stop the app (ignore errors if not running)
	app_stop "$app_name" 2>/dev/null || true
	
	# Wait briefly for clean shutdown
	sleep 2
	
	# Start the app
	if app_start "$app_name" "$@"; then
		log::success "Restarted $app_name"
		return 0
	else
		log::error "Failed to restart $app_name"
		return 1
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
		stop-all) app_stop_all "$@" ;;
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