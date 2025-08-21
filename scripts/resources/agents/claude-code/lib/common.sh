#!/usr/bin/env bash
# Claude Code Common Utility Functions
# Shared utilities used across all modules

#######################################
# Check if Node.js is installed and meets version requirements
# Returns: 0 if valid, 1 otherwise
#######################################
claude_code::check_node_version() {
	if command -v system::is_command >/dev/null 2>&1; then
		if ! system::is_command node; then
			return 1
		fi
	else
		command -v node >/dev/null 2>&1 || return 1
	fi
	
	local node_version
	node_version=$(node --version | sed 's/v//' | cut -d'.' -f1)
	
	if [[ "$node_version" -lt "$MIN_NODE_VERSION" ]]; then
		return 1
	fi
	
	return 0
}

#######################################
# Check if Claude Code is installed
# Returns: 0 if installed, 1 otherwise
#######################################
claude_code::is_installed() {
	if command -v system::is_command >/dev/null 2>&1; then
		if system::is_command claude; then
			return 0
		fi
	else
		if command -v claude >/dev/null 2>&1; then
			return 0
		fi
	fi
	return 1
}

#######################################
# Get Claude Code version
# Outputs: version string or "not installed"
#######################################
claude_code::get_version() {
	if claude_code::is_installed; then
		claude --version 2>/dev/null || echo "unknown"
	else
		echo "not installed"
	fi
}

#######################################
# Set timeout environment variables for Claude operations
# Arguments:
#   $1 - timeout in seconds
#######################################
claude_code::set_timeouts() {
	local timeout="${1:-$DEFAULT_TIMEOUT}"
	export BASH_DEFAULT_TIMEOUT_MS=$((timeout * 1000))
	export BASH_MAX_TIMEOUT_MS=$((timeout * 1000))
	export MCP_TOOL_TIMEOUT=$((timeout * 1000))
}

#######################################
# Build allowed tools parameter string
# Arguments:
#   $1 - comma-separated list of tools
# Outputs: parameter string for claude command
#######################################
claude_code::build_allowed_tools() {
	local tools_list="$1"
	local result=""
	
	if [[ -n "$tools_list" ]]; then
		IFS=',' read -ra TOOLS <<< "$tools_list"
		for tool in "${TOOLS[@]}"; do
			result="$result --allowedTools \"$tool\""
		done
	fi
	
	echo "$result"
}

#######################################
# Check if running in a TTY environment
# Returns: 0 if TTY, 1 otherwise
#######################################
claude_code::is_tty() {
	[[ -t 0 && -t 1 ]]
}

#######################################
# Check if command requires TTY and handle appropriately
# Arguments:
#   $1 - command name (e.g., "doctor", "interactive")
# Returns: 0 if can proceed, 1 if should skip
#######################################
claude_code::check_tty_requirement() {
	local command="$1"
	
	if ! claude_code::is_tty; then
		case "$command" in
			"doctor")
				log::warn "Interactive 'claude doctor' not supported in non-TTY environment"
				log::info "Use --action health-check instead for non-interactive diagnostics"
				return 1
				;;
			"interactive")
				log::error "Interactive mode requires a TTY environment"
				log::info "Use --action run with --prompt for non-interactive execution"
				return 1
				;;
			*)
				# Command doesn't require TTY
				return 0
				;;
		esac
	fi
	
	return 0
}

#######################################
# Detect rate limit from Claude output and exit code
# Arguments:
#   $1 - Output from Claude command
#   $2 - Exit code from Claude command
# Outputs: JSON with rate limit detection results
#######################################
claude_code::detect_rate_limit() {
	local output="$1"
	local exit_code="$2"
	
	local rate_limit_detected="false"
	local retry_after=""
	local reset_time=""
	local limit_type=""
	local error_message=""
	
	# Check for various rate limit patterns
	if [[ $exit_code -eq 129 ]] || \
	   [[ "$output" =~ "429" ]] || \
	   [[ "$output" =~ "rate_limit_error" ]] || \
	   [[ "$output" =~ [Rr]ate[[:space:]]limit ]] || \
	   [[ "$output" =~ [Uu]sage[[:space:]]limit ]] || \
	   [[ "$output" =~ "Too Many Requests" ]] || \
	   [[ "$output" =~ "would exceed your.*limit" ]] || \
	   [[ "$output" =~ "limit.*reached" ]]; then
		
		rate_limit_detected="true"
		
		# Extract retry-after if available (in seconds)
		if [[ "$output" =~ retry[[:space:]]after[[:space:]]([0-9]+) ]]; then
			retry_after="${BASH_REMATCH[1]}"
		elif [[ "$output" =~ "retry_after"[[:space:]]*:[[:space:]]*([0-9]+) ]]; then
			retry_after="${BASH_REMATCH[1]}"
		fi
		
		# Extract reset time if available
		if [[ "$output" =~ reset[s]?[[:space:]]at[[:space:]]([0-9]+[amp]*) ]]; then
			reset_time="${BASH_REMATCH[1]}"
		elif [[ "$output" =~ reset[s]?[[:space:]]in[[:space:]]([0-9]+)[[:space:]]hour ]]; then
			local hours="${BASH_REMATCH[1]}"
			reset_time="$(date -d "+${hours} hours" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "")"
		elif [[ "$output" =~ "typically every 5 hours" ]]; then
			reset_time="$(date -d "+5 hours" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "")"
		fi
		
		# Determine limit type
		if [[ "$output" =~ "weekly.*limit" ]]; then
			limit_type="weekly"
		elif [[ "$output" =~ "daily.*limit" ]]; then
			limit_type="daily"
		elif [[ "$output" =~ "5.*hour" ]]; then
			limit_type="5_hour"
		else
			limit_type="unknown"
		fi
		
		# Extract error message
		if [[ "$output" =~ \"message\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
			error_message="${BASH_REMATCH[1]}"
		fi
	fi
	
	# Output JSON result
	cat <<-EOF
	{
		"detected": $rate_limit_detected,
		"retry_after": "${retry_after:-300}",
		"reset_time": "$reset_time",
		"limit_type": "$limit_type",
		"error_message": "$error_message",
		"timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	}
	EOF
}

#######################################
# Initialize usage tracking file if it doesn't exist
# Creates the usage tracking JSON file with default structure
#######################################
claude_code::init_usage_tracking() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Create directory if it doesn't exist
	mkdir -p "$(dirname "$usage_file")"
	
	# Initialize file if it doesn't exist
	if [[ ! -f "$usage_file" ]]; then
		cat > "$usage_file" <<-EOF
		{
			"hourly_requests": {},
			"daily_requests": {},
			"weekly_requests": {},
			"rate_limit_encounters": [],
			"last_5hour_reset": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
			"last_weekly_reset": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
			"subscription_tier": "unknown",
			"estimated_limits": {
				"free": {"5_hour": 45, "daily": 50, "weekly": 350},
				"pro": {"5_hour": 216, "daily": 1000, "weekly": 7000},
				"max_100": {"5_hour": 1080, "daily": 5000, "weekly": 35000},
				"max_200": {"5_hour": 4320, "daily": 20000, "weekly": 140000}
			}
		}
		EOF
	fi
}

#######################################
# Track a Claude Code request
# Updates hourly, daily, and weekly counters
#######################################
claude_code::track_request() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	local current_hour=$(date +%Y%m%d%H)
	local current_day=$(date +%Y%m%d)
	local current_week=$(date +%Y%W)
	
	# Update counters using jq
	local temp_file=$(mktemp)
	jq --arg hour "$current_hour" \
	   --arg day "$current_day" \
	   --arg week "$current_week" \
	   '.hourly_requests[$hour] = ((.hourly_requests[$hour] // 0) + 1) |
	    .daily_requests[$day] = ((.daily_requests[$day] // 0) + 1) |
	    .weekly_requests[$week] = ((.weekly_requests[$week] // 0) + 1)' \
	   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
}

#######################################
# Record a rate limit encounter
# Arguments:
#   $1 - Rate limit info JSON (from detect_rate_limit)
#######################################
claude_code::record_rate_limit() {
	local rate_info="$1"
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	# Add to rate limit encounters array (keep last 100)
	local temp_file=$(mktemp)
	jq --argjson info "$rate_info" \
	   '.rate_limit_encounters = ([$info] + .rate_limit_encounters)[0:100]' \
	   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
}

#######################################
# Get current usage statistics
# Outputs: JSON with usage data
#######################################
claude_code::get_usage() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	local current_hour=$(date +%Y%m%d%H)
	local current_day=$(date +%Y%m%d)
	local current_week=$(date +%Y%W)
	
	# Calculate 5-hour window (Claude resets every 5 hours)
	local five_hours_ago=$(date -d "5 hours ago" +%Y%m%d%H)
	
	# Get usage data
	jq --arg hour "$current_hour" \
	   --arg day "$current_day" \
	   --arg week "$current_week" \
	   --arg five_hours_ago "$five_hours_ago" \
	   '{
	     current_hour_requests: (.hourly_requests[$hour] // 0),
	     current_day_requests: (.daily_requests[$day] // 0),
	     current_week_requests: (.weekly_requests[$week] // 0),
	     last_5_hours: (
	       [.hourly_requests | to_entries[] | 
	        select(.key >= $five_hours_ago and .key <= $hour) | 
	        .value] | add // 0
	     ),
	     last_rate_limit: (.rate_limit_encounters[0] // null),
	     subscription_tier: .subscription_tier,
	     estimated_limits: .estimated_limits
	   }' "$usage_file"
}

#######################################
# Check if approaching rate limits
# Returns: 0 if safe, 1 if warning, 2 if critical
# Outputs: Warning message if needed
#######################################
claude_code::check_usage_limits() {
	local usage_json=$(claude_code::get_usage)
	
	local last_5_hours=$(echo "$usage_json" | jq -r '.last_5_hours')
	local current_day=$(echo "$usage_json" | jq -r '.current_day_requests')
	local current_week=$(echo "$usage_json" | jq -r '.current_week_requests')
	local tier=$(echo "$usage_json" | jq -r '.subscription_tier')
	
	# Get limits based on tier (default to free if unknown)
	local tier_key="${tier:-free}"
	[[ "$tier_key" == "unknown" ]] && tier_key="free"
	
	local limit_5h=$(echo "$usage_json" | jq -r ".estimated_limits.${tier_key}.\"5_hour\"")
	local limit_daily=$(echo "$usage_json" | jq -r ".estimated_limits.${tier_key}.daily")
	local limit_weekly=$(echo "$usage_json" | jq -r ".estimated_limits.${tier_key}.weekly")
	
	# Calculate usage percentages
	local pct_5h=$((last_5_hours * 100 / limit_5h))
	local pct_daily=$((current_day * 100 / limit_daily))
	local pct_weekly=$((current_week * 100 / limit_weekly))
	
	# Check thresholds
	if [[ $pct_5h -ge 95 ]] || [[ $pct_daily -ge 95 ]] || [[ $pct_weekly -ge 95 ]]; then
		log::error "⚠️  CRITICAL: Approaching rate limits!"
		log::warn "  5-hour: ${last_5_hours}/${limit_5h} (${pct_5h}%)"
		log::warn "  Daily: ${current_day}/${limit_daily} (${pct_daily}%)"
		log::warn "  Weekly: ${current_week}/${limit_weekly} (${pct_weekly}%)"
		# TODO: Implement automatic fallback to LiteLLM when rate limits are critical
		# This would involve:
		# 1. Checking if LiteLLM resource is available
		# 2. Switching the execution backend to use LiteLLM API
		# 3. Translating Claude prompts to LiteLLM format
		# 4. Managing the fallback state and recovery
		return 2
	elif [[ $pct_5h -ge 80 ]] || [[ $pct_daily -ge 80 ]] || [[ $pct_weekly -ge 80 ]]; then
		log::warn "⚠️  WARNING: High usage detected"
		log::info "  5-hour: ${last_5_hours}/${limit_5h} (${pct_5h}%)"
		log::info "  Daily: ${current_day}/${limit_daily} (${pct_daily}%)"
		log::info "  Weekly: ${current_week}/${limit_weekly} (${pct_weekly}%)"
		# TODO: Prepare LiteLLM fallback when usage is high
		# Pre-check LiteLLM availability and configuration
		return 1
	fi
	
	return 0
}

#######################################
# Estimate time until rate limit reset
# Arguments:
#   $1 - Limit type (5_hour, daily, weekly)
# Outputs: Human-readable time until reset
#######################################
claude_code::time_until_reset() {
	local limit_type="${1:-5_hour}"
	local now=$(date +%s)
	local reset_time
	
	case "$limit_type" in
		"5_hour")
			# Calculate next 5-hour boundary
			local hours_since_midnight=$(($(date +%H) % 5))
			local hours_until_reset=$((5 - hours_since_midnight))
			reset_time=$(date -d "+${hours_until_reset} hours" +%s)
			;;
		"daily")
			# Reset at midnight UTC
			reset_time=$(date -d "tomorrow 00:00:00 UTC" +%s)
			;;
		"weekly")
			# Reset on Monday 00:00:00 UTC
			local days_until_monday=$(( (8 - $(date +%u)) % 7 ))
			[[ $days_until_monday -eq 0 ]] && days_until_monday=7
			reset_time=$(date -d "+${days_until_monday} days 00:00:00 UTC" +%s)
			;;
		*)
			echo "Unknown limit type"
			return 1
			;;
	esac
	
	local seconds_until_reset=$((reset_time - now))
	local hours=$((seconds_until_reset / 3600))
	local minutes=$(((seconds_until_reset % 3600) / 60))
	
	if [[ $hours -gt 0 ]]; then
		echo "${hours}h ${minutes}m"
	else
		echo "${minutes}m"
	fi
}

#######################################
# Clean up old usage data
# Removes data older than 30 days
#######################################
claude_code::cleanup_usage_data() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	[[ ! -f "$usage_file" ]] && return 0
	
	local cutoff_day=$(date -d "30 days ago" +%Y%m%d)
	local cutoff_week=$(date -d "30 days ago" +%Y%W)
	
	# Clean up old data
	local temp_file=$(mktemp)
	jq --arg cutoff_day "$cutoff_day" \
	   --arg cutoff_week "$cutoff_week" \
	   '.hourly_requests |= with_entries(select(.key >= $cutoff_day + "00")) |
	    .daily_requests |= with_entries(select(.key >= $cutoff_day)) |
	    .weekly_requests |= with_entries(select(.key >= $cutoff_week))' \
	   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
}

#######################################
# Set subscription tier for accurate limit tracking
# Arguments:
#   $1 - Tier (free, pro, max_100, max_200)
#######################################
claude_code::set_subscription_tier() {
	local tier="$1"
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Validate tier
	case "$tier" in
		free|pro|max_100|max_200)
			;;
		*)
			log::error "Invalid tier: $tier"
			log::info "Valid tiers: free, pro, max_100, max_200"
			return 1
			;;
	esac
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	# Update tier
	local temp_file=$(mktemp)
	jq --arg tier "$tier" \
	   '.subscription_tier = $tier' \
	   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
	
	log::info "Subscription tier set to: $tier"
	
	# Show new limits
	local usage_json=$(claude_code::get_usage)
	local limit_5h=$(echo "$usage_json" | jq -r ".estimated_limits.${tier}.\"5_hour\"")
	local limit_daily=$(echo "$usage_json" | jq -r ".estimated_limits.${tier}.daily")
	local limit_weekly=$(echo "$usage_json" | jq -r ".estimated_limits.${tier}.weekly")
	
	log::info "New limits:"
	log::info "  5-hour: $limit_5h requests"
	log::info "  Daily: $limit_daily requests"
	log::info "  Weekly: $limit_weekly requests"
}

#######################################
# Reset usage counters (for testing or manual reset)
# Arguments:
#   $1 - Reset type (hourly, daily, weekly, all)
#######################################
claude_code::reset_usage_counters() {
	local reset_type="${1:-all}"
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	[[ ! -f "$usage_file" ]] && return 0
	
	local temp_file=$(mktemp)
	
	case "$reset_type" in
		hourly)
			jq '.hourly_requests = {}' "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
			log::info "Reset hourly usage counters"
			;;
		daily)
			jq '.daily_requests = {}' "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
			log::info "Reset daily usage counters"
			;;
		weekly)
			jq '.weekly_requests = {}' "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
			log::info "Reset weekly usage counters"
			;;
		all)
			jq '.hourly_requests = {} | .daily_requests = {} | .weekly_requests = {}' \
			   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
			log::info "Reset all usage counters"
			;;
		*)
			log::error "Invalid reset type: $reset_type"
			log::info "Valid types: hourly, daily, weekly, all"
			return 1
			;;
	esac
}