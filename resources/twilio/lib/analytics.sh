#!/usr/bin/env bash
################################################################################
# Twilio Analytics Library
# 
# Analytics dashboard for message statistics and cost tracking
################################################################################

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"
TWILIO_DATA_DIR="${TWILIO_DIR}/data"
TWILIO_HISTORY_FILE="${TWILIO_DATA_DIR}/message_history.json"
TWILIO_AUDIT_DIR="${TWILIO_DATA_DIR}/audit"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${TWILIO_DIR}/lib/common.sh"

################################################################################
# Cost Configuration
################################################################################

# Default SMS costs (in cents)
export SMS_COST_PER_MESSAGE="${SMS_COST_PER_MESSAGE:-0.75}"  # $0.0075 per SMS
export SMS_COST_PER_SEGMENT="${SMS_COST_PER_SEGMENT:-0.75}"  # Additional segments
export MMS_COST_PER_MESSAGE="${MMS_COST_PER_MESSAGE:-2.0}"   # $0.02 per MMS

################################################################################
# Analytics Functions
################################################################################

# Calculate message segments (SMS is 160 chars, or 70 if unicode)
twilio::analytics::calculate_segments() {
    local message="${1:-}"
    local length=${#message}
    
    # Check if message contains unicode
    if echo "$message" | grep -q '[^\x00-\x7F]'; then
        # Unicode message - 70 chars per segment
        echo $(( (length + 69) / 70 ))
    else
        # Regular message - 160 chars per segment
        echo $(( (length + 159) / 160 ))
    fi
}

# Calculate cost for a message
twilio::analytics::calculate_cost() {
    local message="${1:-}"
    local segments
    segments=$(twilio::analytics::calculate_segments "$message")
    
    # Cost in cents
    local cost=$((SMS_COST_PER_MESSAGE + (segments - 1) * SMS_COST_PER_SEGMENT))
    echo "$cost"
}

# Get dashboard summary
twilio::analytics::dashboard() {
    local period="${1:-today}"  # today, week, month, all
    
    if [[ ! -f "$TWILIO_HISTORY_FILE" ]]; then
        log::warn "No message history available"
        return 1
    fi
    
    # Calculate date filters
    local date_filter=""
    case "$period" in
        today)
            date_filter=$(date -u +"%Y-%m-%d")
            ;;
        week)
            date_filter=$(date -u -d "7 days ago" +"%Y-%m-%d")
            ;;
        month)
            date_filter=$(date -u -d "30 days ago" +"%Y-%m-%d")
            ;;
        all)
            date_filter=""
            ;;
    esac
    
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║               TWILIO ANALYTICS DASHBOARD                      ║"
    echo "║                   Period: ${period^^}                         ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    
    # Get message stats
    local total_messages
    local successful_messages
    local failed_messages
    
    if [[ -n "$date_filter" ]]; then
        total_messages=$(jq "[.messages[] | select(.timestamp >= \"$date_filter\")] | length" "$TWILIO_HISTORY_FILE")
        successful_messages=$(jq "[.messages[] | select(.timestamp >= \"$date_filter\" and (.status == \"sent\" or .status == \"delivered\"))] | length" "$TWILIO_HISTORY_FILE")
        failed_messages=$(jq "[.messages[] | select(.timestamp >= \"$date_filter\" and .status == \"failed\")] | length" "$TWILIO_HISTORY_FILE")
    else
        total_messages=$(jq ".messages | length" "$TWILIO_HISTORY_FILE")
        successful_messages=$(jq "[.messages[] | select(.status == \"sent\" or .status == \"delivered\")] | length" "$TWILIO_HISTORY_FILE")
        failed_messages=$(jq "[.messages[] | select(.status == \"failed\")] | length" "$TWILIO_HISTORY_FILE")
    fi
    
    # Calculate success rate
    local success_rate=0
    if [[ $total_messages -gt 0 ]]; then
        success_rate=$(echo "scale=2; $successful_messages * 100 / $total_messages" | bc)
    fi
    
    echo "📊 MESSAGE STATISTICS"
    echo "═══════════════════════════════════════════════════════════════"
    printf "  Total Messages:     %-10d\n" "$total_messages"
    printf "  Successful:         %-10d (%.1f%%)\n" "$successful_messages" "$success_rate"
    printf "  Failed:             %-10d\n" "$failed_messages"
    echo ""
    
    # Calculate costs
    local total_cost=0
    local messages
    if [[ -n "$date_filter" ]]; then
        messages=$(jq -c "[.messages[] | select(.timestamp >= \"$date_filter\")]" "$TWILIO_HISTORY_FILE")
    else
        messages=$(jq -c ".messages" "$TWILIO_HISTORY_FILE")
    fi
    
    # Calculate total segments and cost
    local total_segments=0
    while IFS= read -r msg; do
        local body
        body=$(echo "$msg" | jq -r '.body')
        if [[ -n "$body" ]]; then
            local segments
            segments=$(twilio::analytics::calculate_segments "$body")
            total_segments=$((total_segments + segments))
            local cost
            cost=$(twilio::analytics::calculate_cost "$body")
            total_cost=$((total_cost + cost))
        fi
    done < <(echo "$messages" | jq -c '.[]')
    
    # Convert cents to dollars
    local total_cost_dollars
    total_cost_dollars=$(echo "scale=2; $total_cost / 100" | bc)
    
    echo "💰 COST ANALYSIS"
    echo "═══════════════════════════════════════════════════════════════"
    printf "  Total Segments:     %-10d\n" "$total_segments"
    printf "  Estimated Cost:     \$%-9.2f\n" "$total_cost_dollars"
    if [[ $total_messages -gt 0 ]]; then
        local avg_cost
        avg_cost=$(echo "scale=4; $total_cost / $total_messages / 100" | bc)
        printf "  Average Cost/Msg:   \$%-9.4f\n" "$avg_cost"
    fi
    echo ""
    
    # Top recipients
    echo "📱 TOP RECIPIENTS"
    echo "═══════════════════════════════════════════════════════════════"
    if [[ -n "$date_filter" ]]; then
        jq -r "[.messages[] | select(.timestamp >= \"$date_filter\")] | group_by(.to) | map({to: .[0].to, count: length}) | sort_by(.count) | reverse | .[0:5] | .[] | \"  \\(.to): \\(.count) messages\"" "$TWILIO_HISTORY_FILE" 2>/dev/null || echo "  No data"
    else
        jq -r ".messages | group_by(.to) | map({to: .[0].to, count: length}) | sort_by(.count) | reverse | .[0:5] | .[] | \"  \\(.to): \\(.count) messages\"" "$TWILIO_HISTORY_FILE" 2>/dev/null || echo "  No data"
    fi
    echo ""
    
    # Hourly distribution (for today only)
    if [[ "$period" == "today" ]]; then
        echo "⏰ HOURLY DISTRIBUTION"
        echo "═══════════════════════════════════════════════════════════════"
        
        for hour in $(seq -w 0 23); do
            local hour_count
            hour_count=$(jq "[.messages[] | select(.timestamp | startswith(\"$(date -u +%Y-%m-%d)T${hour}:\"))] | length" "$TWILIO_HISTORY_FILE")
            if [[ $hour_count -gt 0 ]]; then
                printf "  %s:00 - %s:59:  " "$hour" "$hour"
                # Print bar chart
                for ((i=0; i<hour_count && i<50; i++)); do
                    printf "█"
                done
                printf " (%d)\n" "$hour_count"
            fi
        done
        echo ""
    fi
    
    # Delivery status breakdown
    echo "📬 DELIVERY STATUS"
    echo "═══════════════════════════════════════════════════════════════"
    if [[ -n "$date_filter" ]]; then
        jq -r "[.messages[] | select(.timestamp >= \"$date_filter\")] | group_by(.status) | map({status: .[0].status, count: length}) | .[] | \"  \\(.status): \\(.count)\"" "$TWILIO_HISTORY_FILE" 2>/dev/null || echo "  No data"
    else
        jq -r ".messages | group_by(.status) | map({status: .[0].status, count: length}) | .[] | \"  \\(.status): \\(.count)\"" "$TWILIO_HISTORY_FILE" 2>/dev/null || echo "  No data"
    fi
    echo ""
    
    echo "═══════════════════════════════════════════════════════════════"
    echo "Generated: $(date)"
}

# Export report to file
twilio::analytics::export_report() {
    local output_file="${1:-twilio_report_$(date +%Y%m%d_%H%M%S).txt}"
    local period="${2:-month}"
    
    log::info "Generating analytics report for period: $period"
    
    # Capture dashboard output
    twilio::analytics::dashboard "$period" > "$output_file"
    
    log::success "Report exported to: $output_file"
}

# Get cost breakdown by day
twilio::analytics::daily_costs() {
    local days="${1:-7}"
    
    echo "📅 Daily Cost Breakdown (Last $days days)"
    echo "═══════════════════════════════════════════════════════════════"
    
    for ((i=0; i<days; i++)); do
        local date
        date=$(date -u -d "$i days ago" +"%Y-%m-%d")
        
        local day_messages
        day_messages=$(jq "[.messages[] | select(.timestamp | startswith(\"$date\"))]" "$TWILIO_HISTORY_FILE")
        
        local day_count
        day_count=$(echo "$day_messages" | jq "length")
        
        if [[ $day_count -gt 0 ]]; then
            local day_cost=0
            while IFS= read -r msg; do
                local body
                body=$(echo "$msg" | jq -r '.body')
                if [[ -n "$body" ]]; then
                    local cost
                    cost=$(twilio::analytics::calculate_cost "$body")
                    day_cost=$((day_cost + cost))
                fi
            done < <(echo "$day_messages" | jq -c '.[]')
            
            local cost_dollars
            cost_dollars=$(echo "scale=2; $day_cost / 100" | bc)
            
            printf "  %s: %3d messages - \$%.2f\n" "$date" "$day_count" "$cost_dollars"
        fi
    done
}

# Performance metrics
twilio::analytics::performance() {
    echo "⚡ PERFORMANCE METRICS"
    echo "═══════════════════════════════════════════════════════════════"
    
    # Check audit logs for rate limiting events
    if [[ -d "$TWILIO_AUDIT_DIR" ]]; then
        local rate_limit_count
        rate_limit_count=$(grep -c "RATE_LIMIT" "$TWILIO_AUDIT_DIR"/*.log 2>/dev/null || echo "0")
        echo "  Rate Limit Events: $rate_limit_count"
    fi
    
    # Calculate average message length
    local avg_length
    avg_length=$(jq '[.messages[].body | length] | add / length' "$TWILIO_HISTORY_FILE" 2>/dev/null || echo "0")
    printf "  Avg Message Length: %.0f chars\n" "$avg_length"
    
    # Get template usage stats
    local template_count
    template_count=$(jq '[.messages[] | select(.body | contains("{{"))] | length' "$TWILIO_HISTORY_FILE" 2>/dev/null || echo "0")
    echo "  Template Usage: $template_count messages"
    
    echo ""
}

################################################################################
# Export Functions
################################################################################

export -f twilio::analytics::calculate_segments
export -f twilio::analytics::calculate_cost
export -f twilio::analytics::dashboard
export -f twilio::analytics::export_report
export -f twilio::analytics::daily_costs
export -f twilio::analytics::performance