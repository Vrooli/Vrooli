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
	
	# First, check if output is JSON with usage limit error
	# Handle the JSON response format: {"type":"result","is_error":true,"result":"Claude AI usage limit reached|1755806400",...}
	if echo "$output" | jq -e '.' >/dev/null 2>&1; then
		# It's valid JSON, let's check for the specific error structure
		local is_error=$(echo "$output" | jq -r '.is_error // false' 2>/dev/null)
		local result=$(echo "$output" | jq -r '.result // ""' 2>/dev/null)
		
		if [[ "$is_error" == "true" ]] && [[ "$result" =~ "Claude AI usage limit reached" ]]; then
			rate_limit_detected="true"
			error_message="Claude AI usage limit reached"
			
			# Extract timestamp from result field (format: "Claude AI usage limit reached|1755806400")
			if [[ "$result" =~ \|([0-9]+)$ ]]; then
				local timestamp="${BASH_REMATCH[1]}"
				# Convert Unix timestamp to human-readable format
				reset_time="$(date -d "@$timestamp" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "")"
				
				# Calculate retry_after in seconds
				local current_time=$(date +%s)
				if [[ -n "$timestamp" && "$timestamp" -gt "$current_time" ]]; then
					retry_after=$((timestamp - current_time))
				fi
			fi
			
			# For Claude Code, it's typically a 5-hour rolling limit
			limit_type="5_hour"
		fi
	fi
	
	# If not detected as JSON error, check for various text patterns
	if [[ "$rate_limit_detected" == "false" ]]; then
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
	mkdir -p "${usage_file%/*"
	
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
				"free": {"5_hour": 30, "daily": 50, "weekly": 350},
				"pro": {"5_hour": 45, "daily": 216, "weekly": 1500},
				"max_100": {"5_hour": 225, "daily": 1080, "weekly": 7500},
				"max_200": {"5_hour": 900, "daily": 4320, "weekly": 30000}
			},
			"limits_source": {
				"description": "Estimates based on Claude Code weekly hour limits",
				"sources": [
					"TechCrunch: 'Anthropic unveils new rate limits' (2025-07-28)",
					"Anthropic Help: 'Using Claude Code with Pro/Max' (2024)",
					"Note: Pro=40-80hr/week, Max100=140-280hr/week, Max200=240-480hr/week Sonnet 4"
				],
				"last_verified": "2025-08-21",
				"accuracy_note": "These are ESTIMATES. Actual limits vary by message length and complexity.",
				"rolling_window": "5-hour limit uses rolling window, not fixed boundaries"
			}
		}
		EOF
	else
		# Upgrade existing file to include new fields if missing
		local temp_file=$(mktemp)
		
		# Check if limits_source exists
		local has_limits_source=$(jq 'has("limits_source")' "$usage_file" 2>/dev/null || echo "false")
		
		if [[ "$has_limits_source" == "false" ]]; then
			# Add limits_source to existing file
			jq '.limits_source = {
				"description": "Estimates based on Claude Code weekly hour limits",
				"sources": [
					"TechCrunch: \"Anthropic unveils new rate limits\" (2025-07-28)",
					"Anthropic Help: \"Using Claude Code with Pro/Max\" (2024)",
					"Note: Pro=40-80hr/week, Max100=140-280hr/week, Max200=240-480hr/week Sonnet 4"
				],
				"last_verified": "2025-08-21",
				"accuracy_note": "These are ESTIMATES. Actual limits vary by message length and complexity.",
				"rolling_window": "5-hour limit uses rolling window, not fixed boundaries"
			}' "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
		fi
		
		# Check if estimated_limits exists and has correct structure
		local has_estimated_limits=$(jq 'has("estimated_limits")' "$usage_file" 2>/dev/null || echo "false")
		
		if [[ "$has_estimated_limits" == "false" ]]; then
			# Add estimated_limits to existing file
			jq '.estimated_limits = {
				"free": {"5_hour": 30, "daily": 50, "weekly": 350},
				"pro": {"5_hour": 45, "daily": 216, "weekly": 1500},
				"max_100": {"5_hour": 225, "daily": 1080, "weekly": 7500},
				"max_200": {"5_hour": 900, "daily": 4320, "weekly": 30000}
			}' "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
		fi
	fi
}

#######################################
# Try to auto-detect subscription tier
# Checks various indicators to guess the tier
#######################################
claude_code::auto_detect_tier() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Check if tier is already set
	local current_tier=$(jq -r '.subscription_tier // "unknown"' "$usage_file" 2>/dev/null || echo "unknown")
	[[ "$current_tier" != "unknown" ]] && return 0
	
	# Check environment variable
	if [[ -n "${CLAUDE_SUBSCRIPTION_TIER:-}" ]]; then
		claude_code::set_subscription_tier "$CLAUDE_SUBSCRIPTION_TIER"
		return 0
	fi
	
	# Check for Pro/Teams/Enterprise indicators in Claude CLI output
	local claude_info
	if command -v claude &>/dev/null; then
		claude_info=$(timeout 2s claude --version 2>&1 || true)
		
		# Look for tier indicators in version output
		if [[ "$claude_info" =~ [Pp]ro ]]; then
			log::info "Auto-detected Pro subscription"
			claude_code::set_subscription_tier "pro"
			return 0
		elif [[ "$claude_info" =~ [Tt]eams ]]; then
			log::info "Auto-detected Teams subscription"
			claude_code::set_subscription_tier "teams"
			return 0
		elif [[ "$claude_info" =~ [Ee]nterprise ]]; then
			log::info "Auto-detected Enterprise subscription"
			claude_code::set_subscription_tier "enterprise"
			return 0
		fi
	fi
	
	# Analyze historical usage patterns to guess tier
	if [[ -f "$usage_file" ]]; then
		local max_5h_usage=$(jq '[.hourly_requests | to_entries[] | .value] | max // 0' "$usage_file" 2>/dev/null || echo "0")
		
		# Guess based on max usage without hitting limits
		if [[ $max_5h_usage -gt 200 ]]; then
			log::info "Based on usage patterns, guessing Enterprise tier"
			claude_code::set_subscription_tier "enterprise"
		elif [[ $max_5h_usage -gt 100 ]]; then
			log::info "Based on usage patterns, guessing Teams tier"
			claude_code::set_subscription_tier "teams"
		elif [[ $max_5h_usage -gt 30 ]]; then
			log::info "Based on usage patterns, guessing Pro tier"
			claude_code::set_subscription_tier "pro"
		fi
	fi
	
	# Default to unknown (will use free tier limits)
	return 0
}

#######################################
# Track a Claude Code request
# Updates hourly, daily, and weekly counters
#######################################
claude_code::track_request() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	# Try to auto-detect tier on first run
	claude_code::auto_detect_tier
	
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
	# Also update the last_rate_limit_hit timestamp for immediate reflection
	local temp_file=$(mktemp)
	local current_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	jq --argjson info "$rate_info" \
	   --arg time "$current_time" \
	   '.rate_limit_encounters = ([$info] + .rate_limit_encounters)[0:100] |
	    .last_rate_limit_hit = $time' \
	   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
	
	# If we hit a rate limit, we should also set usage counters to the limit
	# This ensures the usage command immediately reflects that we're at the limit
	local limit_type=$(echo "$rate_info" | jq -r '.limit_type // "unknown"')
	if [[ "$limit_type" != "unknown" ]]; then
		claude_code::force_set_at_limit "$limit_type"
	fi
}

#######################################
# Force set usage counters to indicate we're at the limit
# Arguments:
#   $1 - Limit type (5_hour, daily, weekly)
#######################################
claude_code::force_set_at_limit() {
	local limit_type="$1"
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	# Get current tier and limits
	local usage_json=$(claude_code::get_usage)
	local tier=$(echo "$usage_json" | jq -r '.subscription_tier // "free"')
	[[ "$tier" == "unknown" ]] && tier="free"
	
	local temp_file=$(mktemp)
	
	case "$limit_type" in
		"5_hour")
			# Set the last 5 hours of requests to the limit
			local limit=$(echo "$usage_json" | jq -r ".estimated_limits.${tier}.\"5_hour\" // 45")
			local current_hour=$(date +%Y%m%d%H)
			jq --arg hour "$current_hour" \
			   --arg limit "$limit" \
			   '.hourly_requests[$hour] = ($limit | tonumber)' \
			   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
			;;
		"daily")
			# Set today's requests to the daily limit
			local limit=$(echo "$usage_json" | jq -r ".estimated_limits.${tier}.daily // 216")
			local current_day=$(date +%Y%m%d)
			jq --arg day "$current_day" \
			   --arg limit "$limit" \
			   '.daily_requests[$day] = ($limit | tonumber)' \
			   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
			;;
		"weekly")
			# Set this week's requests to the weekly limit
			local limit=$(echo "$usage_json" | jq -r ".estimated_limits.${tier}.weekly // 1500")
			local current_week=$(date +%Y%W)
			jq --arg week "$current_week" \
			   --arg limit "$limit" \
			   '.weekly_requests[$week] = ($limit | tonumber)' \
			   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
			;;
	esac
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
	
	# Calculate rolling 5-hour window
	# Note: This is a ROLLING window, not fixed boundaries
	# Requests from >5 hours ago don't count toward current limit
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
	     estimated_limits: .estimated_limits,
	     limits_source: .limits_source
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
			# Rolling 5-hour window - estimate based on oldest request in window
			# This is an ESTIMATE since we don't track the exact window start
			local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
			if [[ -f "$usage_file" ]]; then
				# Find oldest request in last 5 hours
				local five_hours_ago=$(date -d "5 hours ago" +%Y%m%d%H)
				local oldest_hour=$(jq -r ".hourly_requests | to_entries[] | select(.key >= \"$five_hours_ago\") | .key" "$usage_file" 2>/dev/null | sort | head -1)
				if [[ -n "$oldest_hour" ]]; then
					# Calculate 5 hours from oldest request
					local oldest_timestamp=$(date -d "${oldest_hour:0:8} ${oldest_hour:8:2}:00:00" +%s)
					reset_time=$((oldest_timestamp + 18000))  # 5 hours = 18000 seconds
				else
					# No recent requests, assume 5 hours from now
					reset_time=$(date -d "+5 hours" +%s)
				fi
			else
				# Default to 5 hours from now
				reset_time=$(date -d "+5 hours" +%s)
			fi
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
	
	# Validate tier and normalize names
	case "$tier" in
		free|pro)
			;;
		teams|max_100)
			tier="max_100"  # Normalize to internal name
			;;
		enterprise|max_200)
			tier="max_200"  # Normalize to internal name
			;;
		*)
			log::error "Invalid tier: $tier"
			log::info "Valid tiers: free, pro, teams (or max_100), enterprise (or max_200)"
			return 1
			;;
	esac
	
	# Initialize if needed
	claude_code::init_usage_tracking
	
	# Update tier and record when it was set
	local temp_file=$(mktemp)
	local current_date=$(date +%Y-%m-%d)
	jq --arg tier "$tier" \
	   --arg date "$current_date" \
	   '.subscription_tier = $tier |
	    .limits_source.last_verified = $date |
	    .limits_source.user_confirmed = "Tier set by user on " + $date' \
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

#######################################
# Validate and update limits based on observed rate limits
# This helps keep limits accurate over time
# Arguments:
#   $1 - Observed limit type (5_hour, daily, weekly)
#   $2 - Observed limit value
#######################################
claude_code::update_observed_limit() {
	local limit_type="$1"
	local observed_value="$2"
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	[[ ! -f "$usage_file" ]] && return 1
	
	# Get current tier
	local tier=$(jq -r '.subscription_tier // "unknown"' "$usage_file")
	[[ "$tier" == "unknown" ]] && tier="free"
	
	# Update the observed limit
	local temp_file=$(mktemp)
	local current_date=$(date +%Y-%m-%d)
	
	jq --arg tier "$tier" \
	   --arg type "$limit_type" \
	   --arg value "$observed_value" \
	   --arg date "$current_date" \
	   '.estimated_limits[$tier][$type] = ($value | tonumber) |
	    .limits_source.last_observed = $date |
	    .limits_source.observed_values = (.limits_source.observed_values // {}) |
	    .limits_source.observed_values[$tier + "_" + $type] = {
	      "value": ($value | tonumber),
	      "date": $date
	    }' \
	   "$usage_file" > "$temp_file" && mv "$temp_file" "$usage_file"
	
	log::info "Updated $tier tier $limit_type limit to $observed_value based on observation"
}

#######################################
# Analyze rate limit encounters to refine limits
# Looks at historical rate limit encounters to improve accuracy
#######################################
claude_code::analyze_rate_limits() {
	local usage_file="${CLAUDE_USAGE_FILE:-$HOME/.claude/usage_tracking.json}"
	
	[[ ! -f "$usage_file" ]] && return 1
	
	# Analyze recent rate limit encounters
	local analysis=$(jq -r '
		.rate_limit_encounters[:10] | 
		map(select(.detected == true)) |
		group_by(.limit_type) |
		map({
			type: .[0].limit_type,
			count: length,
			last_seen: (.[0].timestamp // "unknown")
		})
	' "$usage_file")
	
	echo "Rate Limit Analysis:"
	echo "$analysis" | jq -r '.[] | "  \(.type): \(.count) occurrences (last: \(.last_seen))"'
	
	# Suggest tier upgrade if hitting limits frequently
	local limit_count=$(echo "$analysis" | jq '[.[].count] | add // 0')
	if [[ $limit_count -gt 5 ]]; then
		log::warn "Frequent rate limits detected. Consider upgrading your Claude subscription."
	fi
}