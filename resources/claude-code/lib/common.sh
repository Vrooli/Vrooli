#!/usr/bin/env bash
# Claude Code Common Functions - Simplified Version
# Core functions needed for Claude Code CLI functionality

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source required utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}" 2>/dev/null || true

# Configuration - avoid redeclaring readonly variables
: "${CLAUDE_USAGE_FILE:=$HOME/.claude/usage_tracking.json}"

#######################################
# Initialize usage tracking
#######################################
claude_code::init_usage_tracking() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	local usage_dir=$(dirname "$usage_file")
	
	# Create directory if needed
	mkdir -p "$usage_dir"
	
	# Initialize empty tracking file if it doesn't exist
	if [[ ! -f "$usage_file" ]]; then
		cat > "$usage_file" << 'EOF'
{
	"subscription_tier": "free",
	"hourly_requests": {},
	"daily_requests": {},
	"weekly_requests": {},
	"rate_limit_encounters": [],
	"estimated_limits": {
		"free": {
			"5_hour": 45,
			"daily": 50,
			"weekly": 350
		},
		"pro": {
			"5_hour": 200,
			"daily": 500,
			"weekly": 3500
		}
	}
}
EOF
	fi
}

#######################################
# Get current usage data
#######################################
claude_code::get_usage() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	# Read and calculate real usage data
	if [[ -f "$usage_file" ]]; then
		local current_hour=$(date +%Y%m%d%H)
		local current_date=$(date +%Y%m%d)
		local current_week=$(date +%Y%U)
		
		# Calculate current usage from actual data
		local hour_requests=$(jq -r ".hourly_requests.\"$current_hour\" // 0" "$usage_file")
		local day_requests=$(jq -r ".daily_requests.\"$current_date\" // 0" "$usage_file")
		local week_requests=$(jq -r ".weekly_requests.\"$current_week\" // 0" "$usage_file")
		
		# Calculate rolling 5-hour total
		local five_hour_total=0
		for i in {0..4}; do
			local hour_key=$(date -d "$i hours ago" +%Y%m%d%H)
			local hour_count=$(jq -r ".hourly_requests.\"$hour_key\" // 0" "$usage_file")
			five_hour_total=$((five_hour_total + hour_count))
		done
		
		# Get subscription tier and limits
		local tier=$(jq -r '.subscription_tier // "free"' "$usage_file")
		local limits=$(jq -r '.estimated_limits' "$usage_file")
		
		# Return calculated usage with limits
		echo "{
			\"current_hour_requests\": $hour_requests,
			\"current_day_requests\": $day_requests,
			\"current_week_requests\": $week_requests,
			\"last_5_hours\": $five_hour_total,
			\"subscription_tier\": \"$tier\",
			\"estimated_limits\": $limits
		}"
	else
		# Return actual file content if it exists
		cat "$usage_file"
	fi
}

#######################################
# Check if approaching usage limits
# Returns: 0 = OK, 1 = Warning, 2 = Critical
#######################################
claude_code::check_usage_limits() {
	local usage_json
	usage_json=$(claude_code::get_usage 2>/dev/null) || return 0
	
	# Parse usage data and check against limits
	local tier=$(echo "$usage_json" | jq -r '.subscription_tier // "free"')
	local five_hour_usage=$(echo "$usage_json" | jq -r '.last_5_hours // 0')
	local daily_usage=$(echo "$usage_json" | jq -r '.current_day_requests // 0')
	local weekly_usage=$(echo "$usage_json" | jq -r '.current_week_requests // 0')
	
	# Get limits based on tier (fallback to "free" tier limits if tier not found)
	local tier_key="$tier"
	[[ "$tier" == "max" ]] && tier_key="max_100"
	
	local five_hour_limit=$(echo "$usage_json" | jq -r ".estimated_limits.\"$tier_key\".\"5_hour\" // .estimated_limits.free.\"5_hour\"")
	local daily_limit=$(echo "$usage_json" | jq -r ".estimated_limits.\"$tier_key\".daily // .estimated_limits.free.daily")
	local weekly_limit=$(echo "$usage_json" | jq -r ".estimated_limits.\"$tier_key\".weekly // .estimated_limits.free.weekly")
	
	# Check thresholds
	local five_hour_pct=$((five_hour_usage * 100 / five_hour_limit))
	local daily_pct=$((daily_usage * 100 / daily_limit))
	local weekly_pct=$((weekly_usage * 100 / weekly_limit))
	
	# Critical threshold (95%+)
	if [[ $five_hour_pct -ge 95 ]] || [[ $daily_pct -ge 95 ]] || [[ $weekly_pct -ge 95 ]]; then
		return 2
	fi
	
	# Warning threshold (80%+)
	if [[ $five_hour_pct -ge 80 ]] || [[ $daily_pct -ge 80 ]] || [[ $weekly_pct -ge 80 ]]; then
		return 1
	fi
	
	# OK
	return 0
}

#######################################
# Record a rate limit encounter
# Arguments:
#   $1 - Rate limit info JSON
#######################################
claude_code::record_rate_limit() {
	local rate_info="${1:-}"
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	# Parse rate limit info and update tracking file
	local current_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	local temp_file=$(mktemp)
	
	# Create rate limit record
	local limit_record
	if [[ -n "$rate_info" ]] && echo "$rate_info" | jq . >/dev/null 2>&1; then
		# Use provided JSON rate info
		limit_record="$rate_info"
	else
		# Create basic rate limit record
		limit_record=$(echo '{}' | jq --arg time "$current_time" --arg info "$rate_info" '
			{"detected": true, "timestamp": $time, "error_message": $info, "retry_after": "300", "limit_type": "unknown", "reset_time": ""}')
	fi
	
	# Add to rate_limit_encounters array and update last_rate_limit_hit
	jq --argjson record "$limit_record" --arg time "$current_time" \
		'.rate_limit_encounters = [$record] + (.rate_limit_encounters // []) |
		 .rate_limit_encounters = .rate_limit_encounters[0:50] |
		 .last_rate_limit_hit = $time' \
		"$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
	
	log::warn "Rate limit recorded at $current_time"
	log::debug "Rate limit info: $rate_info"
	return 0
}

#######################################
# Track a request
#######################################
claude_code::track_request() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	# Update real usage tracking
	local current_hour=$(date +%Y%m%d%H)
	local current_date=$(date +%Y%m%d)
	local current_week=$(date +%Y%U)
	local temp_file=$(mktemp)
	
	# Update hourly, daily, and weekly counters atomically
	jq --arg hour "$current_hour" --arg date "$current_date" --arg week "$current_week" \
		'.hourly_requests[$hour] = (.hourly_requests[$hour] // 0) + 1 |
		 .daily_requests[$date] = (.daily_requests[$date] // 0) + 1 |
		 .weekly_requests[$week] = (.weekly_requests[$week] // 0) + 1' \
		"$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
	
	log::debug "Request tracked at $current_hour (date: $current_date, week: $current_week)"
	return 0
}

#######################################
# Detect rate limit from output
# Arguments:
#   $1 - Command output
#   $2 - Exit code
# Returns: JSON with rate limit info
#######################################
claude_code::detect_rate_limit() {
	local output="${1:-}"
	local exit_code="${2:-0}"
	
	# Basic rate limit detection patterns
	local detected="false"
	local limit_type="unknown"
	local retry_after="300"
	local reset_time=""
	
	if [[ "$output" =~ (usage|rate).*limit ]] || [[ "$exit_code" -eq 429 ]]; then
		detected="true"
		
		# Try to detect limit type
		if [[ "$output" =~ "5.*hour" ]]; then
			limit_type="5_hour"
		elif [[ "$output" =~ "daily" ]]; then
			limit_type="daily"
		elif [[ "$output" =~ "weekly" ]]; then
			limit_type="weekly"
		fi
		
		# Calculate reset time (simplified)
		reset_time=$(date -d "+5 hours" "+%Y-%m-%d %H:00:00")
	fi
	
	# Return JSON
	echo "{
		\"detected\": $detected,
		\"limit_type\": \"$limit_type\",
		\"retry_after\": \"$retry_after\",
		\"reset_time\": \"$reset_time\",
		\"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
		\"error_message\": \"Rate limit detected\"
	}"
}

#######################################
# Check if Claude Code CLI is installed
#######################################
claude_code::is_installed() {
	command -v claude >/dev/null 2>&1
}

#######################################
# Get Claude Code CLI version
#######################################
claude_code::get_version() {
	if claude_code::is_installed; then
		# Try multiple version command formats
		claude --version 2>/dev/null | head -1 || \
		claude version 2>/dev/null | head -1 || \
		echo "unknown"
	else
		echo "not installed"
	fi
}

#######################################
# Check TTY compatibility
#######################################
claude_code::is_tty() {
	[[ -t 0 && -t 1 ]]
}

#######################################
# Check Node.js version compatibility
#######################################
claude_code::check_node_version() {
	local min_version="${MIN_NODE_VERSION:-18}"
	
	if ! command -v node >/dev/null 2>&1; then
		return 1
	fi
	
	local node_version
	node_version=$(node --version | sed 's/v//')
	local major_version
	major_version=$(echo "$node_version" | cut -d. -f1)
	
	[[ "$major_version" -ge "$min_version" ]]
}

#######################################
# Health check for Claude Code
# Arguments:
#   $1 - Check type (basic, full)
#   $2 - Output format (text, json)
#######################################
claude_code::health_check() {
	local check_type="${1:-basic}"
	local output_format="${2:-text}"
	
	local status="healthy"
	local auth_status="unknown"
	local tty_compatible="unknown"
	
	# Basic checks
	if ! claude_code::is_installed; then
		status="unhealthy"
	fi
	
	# TTY compatibility
	if claude_code::is_tty; then
		tty_compatible="true"
	else
		tty_compatible="false"
	fi
	
	# Full checks
	if [[ "$check_type" == "full" ]]; then
		# Try to check authentication (simplified)
		if timeout 3 claude auth whoami >/dev/null 2>&1; then
			auth_status="authenticated"
		else
			auth_status="not_authenticated"
		fi
	fi
	
	# Output
	if [[ "$output_format" == "json" ]]; then
		echo "{
			\"status\": \"$status\",
			\"auth_status\": \"$auth_status\", 
			\"tty_compatible\": $tty_compatible
		}"
	else
		echo "Status: $status"
		echo "Auth: $auth_status"
		echo "TTY: $tty_compatible"
	fi
}

#######################################
# Set timeout environment variables for Claude execution
# Arguments:
#   $1 - Timeout in seconds
#######################################
claude_code::set_timeouts() {
	local timeout="${1:-600}"
	
	# Set Claude-specific timeout environment variables
	export CLAUDE_TIMEOUT="$timeout"
	export CLAUDE_REQUEST_TIMEOUT="$timeout"
	
	# Set general timeout variables that might be used by Claude
	export REQUEST_TIMEOUT="$timeout"
	export API_TIMEOUT="$timeout"
	
	log::debug "Timeouts set to $timeout seconds"
}

#######################################
# Export configuration for use by other modules
#######################################
claude_code::export_config() {
	# Set key variables for other modules (avoid readonly conflicts)
	: "${CLAUDE_USAGE_FILE:=$HOME/.claude/usage_tracking.json}"
	: "${CLAUDE_CONFIG_DIR:=$HOME/.claude}"
	: "${CLAUDE_SESSIONS_DIR:=$HOME/.claude/sessions}"
	
	# Only export if not already exported as readonly
	[[ -z "${CLAUDE_USAGE_FILE_EXPORTED:-}" ]] && { export CLAUDE_USAGE_FILE; export CLAUDE_USAGE_FILE_EXPORTED=1; } 2>/dev/null || true
	[[ -z "${CLAUDE_CONFIG_DIR_EXPORTED:-}" ]] && { export CLAUDE_CONFIG_DIR; export CLAUDE_CONFIG_DIR_EXPORTED=1; } 2>/dev/null || true
	[[ -z "${CLAUDE_SESSIONS_DIR_EXPORTED:-}" ]] && { export CLAUDE_SESSIONS_DIR; export CLAUDE_SESSIONS_DIR_EXPORTED=1; } 2>/dev/null || true
}

# Initialize configuration on load
claude_code::export_config

log::debug "Claude Code common functions loaded (simplified version)"