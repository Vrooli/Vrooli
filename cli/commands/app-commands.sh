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
# Orchestrator API for app-specific operations
ORCHESTRATOR_PORT="${ORCHESTRATOR_PORT:-9500}"
ORCHESTRATOR_API_BASE="http://localhost:${ORCHESTRATOR_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/cli/commands"
VROOLI_ROOT="$APP_ROOT"
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
source "${APP_ROOT}/cli/lib/arg-parser.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/cli/lib/output-formatter.sh"

# Check if orchestrator API is running (REQUIRED - no fallback)
check_orchestrator_api() {
	if ! curl -s --connect-timeout 5 --max-time 10 "${ORCHESTRATOR_API_BASE}/health" >/dev/null 2>&1; then
		return 1  # Return failure silently, let callers handle it
	fi
	return 0
}

# Ensure orchestrator is available - exit with error if not
ensure_orchestrator() {
	local output_format="${1:-text}"
	if ! check_orchestrator_api; then
		cli::format_error "$output_format" "Orchestrator not running"
		echo "Start orchestrator with: vrooli develop" >&2
		return 1
	fi
	return 0
}

# Get app status from orchestrator API
orchestrator_get_app() {
	local app_name="$1"
	curl -s --connect-timeout 3 --max-time 5 "${ORCHESTRATOR_API_BASE}/apps" | \
		jq --arg name "$app_name" '.apps[] | select(.name == $name)' 2>/dev/null
}

# Start app via orchestrator API
orchestrator_start_app() {
	local app_name="$1"
	curl -s -X POST "${ORCHESTRATOR_API_BASE}/apps/$app_name/start" 2>/dev/null
}

# Stop app via orchestrator API
orchestrator_stop_app() {
	local app_name="$1"
	curl -s -X POST "${ORCHESTRATOR_API_BASE}/apps/$app_name/stop" 2>/dev/null
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
	if ! parsed_args=$(parse_combined_args "$@"); then
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
	
	# Use orchestrator API exclusively
	if ! ensure_orchestrator "$output_format"; then
		return 1
	fi
	
	local response
	response=$(curl -s --connect-timeout 3 --max-time 5 "${ORCHESTRATOR_API_BASE}/apps" 2>/dev/null)
	
	if ! echo "$response" | jq -e '.apps' >/dev/null 2>&1; then
		cli::format_error "$output_format" "Failed to get app data from orchestrator"
		return 1
	fi
	
	# Extract apps from orchestrator response
	local apps
	apps=$(echo "$response" | jq -r '.apps')
	
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
		
		# Check runtime protection
		local runtime_protection=""
		local service_json="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$name/service.json"
		if [[ -f "$service_json" ]]; then
			local protection_level
			if command -v jq >/dev/null 2>&1; then
				protection_level=$(jq -r '.runtime.protection // "none"' "$service_json" 2>/dev/null || echo "none")
				if [[ "$protection_level" == "none" ]] || [[ "$protection_level" == "null" ]]; then
					local is_runtime_protected
					is_runtime_protected=$(jq -r '.runtime.protected // false' "$service_json" 2>/dev/null || echo "false")
					if [[ "$is_runtime_protected" == "true" ]]; then
						protection_level="important"
					fi
				fi
			else
				protection_level="none"
			fi
			
			case "$protection_level" in
				critical)
					runtime_protection=" 🛡️"
					;;
				important)
					runtime_protection=" 🛡️"
					;;
			esac
			git_status="${git_status}${runtime_protection}"
		fi
		
		# Get runtime state from API response
		runtime_state="○ Stopped"
		url_display="-"
		
		# Get runtime status from orchestrator API
		local app_status=$(echo "$app" | jq -r '.status // "stopped"')
		if [[ "$app_status" == "running" ]]; then
			runtime_state="● Running"
			
			# Get port information from orchestrator API
			local allocated_ports=$(echo "$app" | jq -r '.allocated_ports // {}')
			local ui_port=$(echo "$allocated_ports" | jq -r '.ui // ""')
			local api_port=$(echo "$allocated_ports" | jq -r '.api // ""')
			
			# Prefer UI port for URL display
			local port=""
			if [[ -n "$ui_port" ]] && [[ "$ui_port" != "null" ]]; then
				port=$ui_port
			elif [[ -n "$api_port" ]] && [[ "$api_port" != "null" ]]; then
				port=$api_port
			fi
			
			if [[ -n "$port" ]] && [[ "$port" != "null" ]] && [[ "$port" != "" ]]; then
				url_display="http://localhost:$port"
			fi
		fi
		
		# Replace : in URL with a placeholder that won't break formatting
		local safe_url="${url_display//:/﹕}"  # Using fullwidth colon (U+FE55)
		rows+=("${name}:${runtime_state}:${safe_url}:${git_status}:${modified}")
		
		app_count=$((app_count + 1))
	done < <(echo "$apps" | jq -c '.[]' 2>/dev/null)
	
	local generated_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
	local count
	count=$(echo "$apps" | jq '. | length')
	
	# Count running apps based on what's actually displayed as "● Running"
	local running_count=0
	for row in "${rows[@]}"; do
		if [[ "$row" == *"● Running"* ]]; then
			running_count=$((running_count + 1))
		fi
	done
	
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
		echo ""
		echo "Legend:"
		echo "  🔒 = Regeneration protected (won't be overwritten)"
		echo "  🛡️ = Runtime protected (won't be stopped without --force)"
		echo ""
		echo "Use 'vrooli app start <name>' to start an app"
		echo "Use 'vrooli app logs <name>' to view logs"
	fi
}

# Show app status
app_status() {
	# Parse common arguments
	local parsed_args
	if ! parsed_args=$(parse_combined_args "$@"); then
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
	
	# Use orchestrator API exclusively
	if ! ensure_orchestrator "$output_format"; then
		return 1
	fi
	
	# Get app from orchestrator
	local response
	response=$(orchestrator_get_app "$app_name")
	
	if [[ -z "$response" || "$response" == "null" ]]; then
		cli::format_error "$output_format" "App not found: $app_name"
		return 1
	fi
	
	# Validate JSON response
	if ! echo "$response" | jq . >/dev/null 2>&1; then
		cli::format_error "$output_format" "Invalid response from orchestrator"
		return 1
	fi
	
	# Use orchestrator response directly (simplified)
	local app="$response"
	local backups="[]"  # TODO: Implement backup listing via orchestrator
	
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
		if [[ $(echo "$backups" | jq 'length') -gt 0 ]]; then
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
	
	ensure_orchestrator "text" || return 1
	
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
	
	ensure_orchestrator "text" || return 1
	
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
	
	ensure_orchestrator "text" || return 1
	
	# Check if protected
	local response
	response=$(curl -s "${API_BASE}/apps/${app_name}/status")
	
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
	
	ensure_orchestrator "text" || return 1
	
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
	
	ensure_orchestrator "text" || return 1
	
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
# Start an app (via orchestrator)
app_start() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	# Ensure orchestrator is available
	if ! ensure_orchestrator "text"; then
		return 1
	fi
	
	# Check if already running
	local app_data
	app_data=$(orchestrator_get_app "$app_name")
	if [[ -n "$app_data" ]] && [[ $(echo "$app_data" | jq -r '.status') == "running" ]]; then
		log::warning "App already running: $app_name"
		return 0
	fi
	
	# Start app via orchestrator API
	log::info "Starting $app_name..."
	local response
	response=$(orchestrator_start_app "$app_name")
	
	# Check if API call was successful
	if echo "$response" | jq -e '.message' >/dev/null 2>&1; then
		log::success "Started $app_name"
		
		# Get updated app data to show URL
		app_data=$(orchestrator_get_app "$app_name")
		if [[ -n "$app_data" ]]; then
			local allocated_ports
			allocated_ports=$(echo "$app_data" | jq -r '.allocated_ports // {}')
			local ui_port
			ui_port=$(echo "$allocated_ports" | jq -r '.ui // ""')
			local api_port
			api_port=$(echo "$allocated_ports" | jq -r '.api // ""')
			
			# Show URL (prefer UI port)
			local port=""
			if [[ -n "$ui_port" ]] && [[ "$ui_port" != "null" ]]; then
				port=$ui_port
			elif [[ -n "$api_port" ]] && [[ "$api_port" != "null" ]]; then
				port=$api_port
			fi
			
			if [[ -n "$port" ]]; then
				echo "  URL: http://localhost:$port"
			fi
		fi
		return 0
	else
		local error_msg
		error_msg=$(echo "$response" | jq -r '.detail // "Failed to start app"' 2>/dev/null || echo "Failed to start app")
		log::error "$error_msg"
		return 1
	fi
}

# Stop an app (via orchestrator)
app_stop() {
	local app_name="${1:-}"
	[[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
	
	# Ensure orchestrator is available
	if ! ensure_orchestrator "text"; then
		return 1
	fi
	
	# Stop app via orchestrator API
	log::info "Stopping $app_name..."
	local response
	response=$(orchestrator_stop_app "$app_name")
	
	# Check if API call was successful
	if echo "$response" | jq -e '.message' >/dev/null 2>&1; then
		log::success "Stopped $app_name"
		return 0
	else
		local error_msg
		error_msg=$(echo "$response" | jq -r '.detail // "Failed to stop app"' 2>/dev/null || echo "App may not have been running")
		log::warning "$error_msg"
		return 0  # Don't fail if app wasn't running
	fi
}

# Stop all apps
app_stop_all() {
	# Use unified stop manager
	local stop_manager="${VROOLI_ROOT}/scripts/lib/lifecycle/stop-manager.sh"
	
	if [[ -f "$stop_manager" ]]; then
		log::info "Using unified stop system..."
		source "$stop_manager"
		
		# Parse flags for stop manager
		local stop_args=()
		for arg in "$@"; do
			case "$arg" in
				--force)
					export FORCE_STOP=true
					;;
				--verbose|-v)
					export VERBOSE=true
					;;
				--dry-run|--check)
					export DRY_RUN=true
					;;
				*)
					stop_args+=("$arg")
					;;
			esac
		done
		
		stop::main apps "${stop_args[@]}"
		return $?
	else
		log::error "Unified stop manager not found at: $stop_manager"
		log::info "Attempting direct process termination..."
		
		# Fallback: Direct kill commands
		pkill -f "app_orchestrator" 2>/dev/null || true
		pkill -f "generated-apps.*manage.sh"
		pkill -f "generated-apps.*-api" 2>/dev/null || true
		
		log::success "Stop-all complete"
		return 0
	fi
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

# Logs - Fixed version with proper argument handling
app_logs() {
	# Use enhanced argument parser to handle shell operator artifacts
	local parsed_args
	if ! parsed_args=$(parse_combined_args "$@"); then
		return 1
	fi
	
	local help_requested
	help_requested=$(extract_arg "$parsed_args" "help")
	if [[ "$help_requested" == "true" ]]; then
		show_app_help
		return 0
	fi
	
	local remaining_args
	remaining_args=$(extract_arg "$parsed_args" "remaining")
	local args_array
	mapfile -t args_array < <(args_to_array "$remaining_args")
	
	local app_name="${args_array[0]:-}"
	local follow=false
	
	# Parse remaining options safely
	for arg in "${args_array[@]:1}"; do
		case "$arg" in
			--follow) 
				follow=true 
				;;
			--tail) 
				log::info "Note: --tail option is not yet supported, showing full log"
				;;
			--help|-h)
				show_app_help
				return 0
				;;
			-*) 
				log::error "Unknown option: $arg"
				return 1
				;;
			*)
				if [[ -z "$app_name" ]]; then
					app_name="$arg"
				else
					log::error "Extra argument: $arg"
					return 1
				fi
				;;
		esac
	done
	
	# Ensure orchestrator is available for app status
	if ! ensure_orchestrator "text"; then
		return 1
	fi
	
	# If no app name provided, show usage and available apps
	if [[ -z "$app_name" ]]; then
		log::error "App name required"
		echo ""
		log::info "Usage: vrooli app logs <app-name> [--follow]"
		echo ""
		
		# Show available apps with logs (both running and failed)
		log::info "Available apps with logs:"
		
		# First, show running apps from orchestrator
		local has_running=false
		echo ""
		echo "Running apps:"
		local response
		response=$(curl -s "${ORCHESTRATOR_API_BASE}/apps" 2>/dev/null)
		if echo "$response" | jq -e '.apps' >/dev/null 2>&1; then
			while IFS= read -r app; do
				local app_name
				app_name=$(echo "$app" | jq -r '.name')
				local app_status
				app_status=$(echo "$app" | jq -r '.status')
				if [[ "$app_status" == "running" ]]; then
					echo "  ✅ $app_name"
					has_running=true
				fi
			done < <(echo "$response" | jq -c '.apps[]')
		fi
		
		if [[ "$has_running" == "false" ]]; then
			echo "  (none running)"
		fi
		
		# Then, show apps with log files (including failed ones)
		echo ""
		echo "Apps with log files:"
		local log_dir="$HOME/.vrooli/logs"
		if [[ -d "$log_dir" ]]; then
			for log_file in "$log_dir"/*.log; do
				if [[ -f "$log_file" ]]; then
					local base_name=$(basename "$log_file" .log)
					# Check if file has content
					if [[ -s "$log_file" ]]; then
						# Check if app is running via orchestrator
						local is_running=false
						local app_data
						app_data=$(orchestrator_get_app "$base_name" 2>/dev/null)
						if [[ -n "$app_data" ]] && [[ $(echo "$app_data" | jq -r '.status') == "running" ]]; then
							is_running=true
						fi
						
						if [[ "$is_running" == "true" ]]; then
							echo "  ✅ $base_name"
						else
							# Check if log contains errors
							if tail -10 "$log_file" | grep -q -i "error\|fail\|exception"; then
								echo "  ❌ $base_name (has errors)"
							else
								echo "  ⚠️  $base_name"
							fi
						fi
					fi
				fi
			done
		fi
		
		echo ""
		log::info "Use 'vrooli app logs <name>' to see logs for any of these apps"
		return 1
	fi
	
	local found=false
	
	# Check if app is running via orchestrator
	local app_data
	app_data=$(orchestrator_get_app "$app_name" 2>/dev/null)
	local is_running=false
	if [[ -n "$app_data" ]] && [[ $(echo "$app_data" | jq -r '.status') == "running" ]]; then
		is_running=true
	fi
	
	# Show log file (whether app is running or not)
	local log_file="$HOME/.vrooli/logs/${app_name}.log"
	
	if [[ -f "$log_file" ]]; then
		found=true
		
		# Show appropriate status message
		if [[ "$is_running" == "true" ]]; then
			log::info "Showing logs for $app_name (running)"
		elif tail -20 "$log_file" | grep -q -i "error\|fail\|exception"; then
			log::warning "Showing logs for $app_name (failed - contains errors)"
		else
			log::info "Showing logs for $app_name (stopped)"
		fi
			
			# Show last error or important info
			echo ""
			echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
			
			if [[ "$follow" == "true" ]]; then
				tail -f "$log_file"
			else
				# Show last 50 lines for failed apps (they need more context)
				tail -50 "$log_file"
			fi
			
			echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
			echo ""
			
			# Provide helpful analysis of the failure
			if tail -20 "$log_file" | grep -q "package.json"; then
				log::error "❌ App tried to run as Node.js but is configured incorrectly"
				echo ""
				log::info "This app appears to be misconfigured. Possible issues:"
				echo "  • The app is a Go/Python app but lifecycle is trying to run 'npm start'"
				echo "  • The app's service.json lifecycle configuration is incorrect"
				echo ""
				log::info "To fix:"
				echo "  1. Check $HOME/generated-apps/$app_name/.vrooli/service.json"
				echo "  2. Ensure the 'develop' lifecycle uses the correct commands"
				echo "  3. Run 'vrooli setup' to regenerate if needed"
			elif tail -20 "$log_file" | grep -q "permission denied"; then
				log::error "❌ Permission denied error"
				echo ""
				log::info "To fix:"
				echo "  • Check file permissions in $HOME/generated-apps/$app_name/"
				echo "  • Ensure executable files have +x permission"
			elif tail -20 "$log_file" | grep -q "port.*already in use\|bind.*address already in use"; then
				log::error "❌ Port conflict - another process is using the required port"
				echo ""
				log::info "To fix:"
				echo "  • Run 'vrooli status --verbose' to see port allocations"
				echo "  • Kill conflicting processes or change port configuration"
			fi
		fi
	
	if [[ "$found" == "false" ]]; then
		log::warning "No logs found for app: $app_name"
		echo ""
		
		# Show available apps
		log::info "Available apps with logs:"
		
		# Show running apps first
		echo ""
		echo "Running apps:"
		local has_running=false
		while IFS= read -r process; do
			if [[ "$process" == vrooli.develop.* ]]; then
				local simple_name="${process#vrooli.develop.}"
				echo "  ✅ $simple_name"
				has_running=true
			fi
		done < <(pm::list 2>/dev/null)
		
		if [[ "$has_running" == "false" ]]; then
			echo "  (none running)"
		fi
		
		# Show apps with log files
		echo ""
		echo "Apps with log files:"
		local log_dir="$HOME/.vrooli/logs"
		if [[ -d "$log_dir" ]]; then
			for log_file in "$log_dir"/*.log; do
				if [[ -f "$log_file" ]]; then
					local base_name=$(basename "$log_file" .log)
					if [[ -s "$log_file" ]]; then
						if tail -10 "$log_file" | grep -q -i "error\|fail\|exception"; then
							echo "  ❌ $base_name (has errors)"
						else
							echo "  ⚠️  $base_name"
						fi
					fi
				fi
			done
		fi
		
		echo ""
		log::info "Use 'vrooli app logs <name>' to see logs for any of these apps"
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