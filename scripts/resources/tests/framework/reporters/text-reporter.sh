#!/bin/bash
# ====================================================================
# Text Reporter - Layer 1 Validation Human-Readable Output
# ====================================================================
#
# Provides rich, human-readable validation reports with color coding,
# detailed error descriptions, fix recommendations, and progress tracking.
# Designed for developer use during development and debugging.
#
# Usage:
#   source text-reporter.sh
#   text_reporter_init
#   text_report_resource_result "resource_name" "status" "details" "duration_ms"
#   text_report_summary "total" "passed" "failed" "duration_s"
#   text_report_cache_stats "cache_stats_json"
#
# Features:
#   - Color-coded pass/fail indicators
#   - Detailed error descriptions with fix recommendations
#   - Progress tracking and summary statistics
#   - Resource-specific guidance and examples
#   - Cache performance reporting
#
# ====================================================================

set -euo pipefail

# Color definitions (auto-disable if not a TTY)
if [[ -t 1 ]]; then
    # Basic colors
    TEXT_REPORTER_RED='\033[0;31m'
    TEXT_REPORTER_GREEN='\033[0;32m'
    TEXT_REPORTER_YELLOW='\033[1;33m'
    TEXT_REPORTER_BLUE='\033[0;34m'
    TEXT_REPORTER_PURPLE='\033[0;35m'
    TEXT_REPORTER_CYAN='\033[0;36m'
    TEXT_REPORTER_WHITE='\033[1;37m'
    TEXT_REPORTER_BOLD='\033[1m'
    TEXT_REPORTER_DIM='\033[2m'
    TEXT_REPORTER_NC='\033[0m'
    
    # Status colors
    TEXT_REPORTER_SUCCESS="$TEXT_REPORTER_GREEN"
    TEXT_REPORTER_ERROR="$TEXT_REPORTER_RED"
    TEXT_REPORTER_WARNING="$TEXT_REPORTER_YELLOW"
    TEXT_REPORTER_INFO="$TEXT_REPORTER_BLUE"
    TEXT_REPORTER_CACHED="$TEXT_REPORTER_CYAN"
else
    # No colors for non-TTY
    TEXT_REPORTER_RED=''
    TEXT_REPORTER_GREEN=''
    TEXT_REPORTER_YELLOW=''
    TEXT_REPORTER_BLUE=''
    TEXT_REPORTER_PURPLE=''
    TEXT_REPORTER_CYAN=''
    TEXT_REPORTER_WHITE=''
    TEXT_REPORTER_BOLD=''
    TEXT_REPORTER_DIM=''
    TEXT_REPORTER_NC=''
    TEXT_REPORTER_SUCCESS=''
    TEXT_REPORTER_ERROR=''
    TEXT_REPORTER_WARNING=''
    TEXT_REPORTER_INFO=''
    TEXT_REPORTER_CACHED=''
fi

# Unicode symbols (with ASCII fallbacks)
if [[ "${LANG:-}" =~ UTF-8 ]] || [[ "${LC_ALL:-}" =~ UTF-8 ]]; then
    TEXT_REPORTER_CHECK_MARK="✅"
    TEXT_REPORTER_CROSS_MARK="❌"
    TEXT_REPORTER_WARNING_MARK="⚠️"
    TEXT_REPORTER_INFO_MARK="ℹ️"
    TEXT_REPORTER_CLOCK_MARK="⏱️"
    TEXT_REPORTER_CACHE_MARK="🗄️"
    TEXT_REPORTER_ROCKET_MARK="🚀"
    TEXT_REPORTER_TROPHY_MARK="🏆"
else
    # ASCII fallbacks
    TEXT_REPORTER_CHECK_MARK="[PASS]"
    TEXT_REPORTER_CROSS_MARK="[FAIL]"
    TEXT_REPORTER_WARNING_MARK="[WARN]"
    TEXT_REPORTER_INFO_MARK="[INFO]"
    TEXT_REPORTER_CLOCK_MARK="[TIME]"
    TEXT_REPORTER_CACHE_MARK="[CACHE]"
    TEXT_REPORTER_ROCKET_MARK="[DONE]"
    TEXT_REPORTER_TROPHY_MARK="[SUCCESS]"
fi

# Reporter state
TEXT_REPORTER_INITIALIZED=false
TEXT_REPORTER_RESOURCE_COUNT=0
TEXT_REPORTER_FAILED_RESOURCES=()

#######################################
# Initialize text reporter
# Arguments: None
# Returns: 0 on success
#######################################
text_reporter_init() {
    TEXT_REPORTER_INITIALIZED=true
    TEXT_REPORTER_RESOURCE_COUNT=0
    TEXT_REPORTER_FAILED_RESOURCES=()
    
    return 0
}

#######################################
# Print section header with formatting
# Arguments: $1 - header text, $2 - style (optional: major|minor)
# Returns: 0
#######################################
text_report_header() {
    local header_text="$1"
    local style="${2:-major}"
    
    echo
    case "$style" in
        "major")
            echo -e "${TEXT_REPORTER_BOLD}${TEXT_REPORTER_BLUE}============================================${TEXT_REPORTER_NC}"
            echo -e "${TEXT_REPORTER_BOLD}${TEXT_REPORTER_WHITE}$header_text${TEXT_REPORTER_NC}"
            echo -e "${TEXT_REPORTER_BOLD}${TEXT_REPORTER_BLUE}============================================${TEXT_REPORTER_NC}"
            ;;
        "minor")
            echo -e "${TEXT_REPORTER_BOLD}${TEXT_REPORTER_CYAN}--- $header_text ---${TEXT_REPORTER_NC}"
            ;;
        "subsection")
            echo -e "${TEXT_REPORTER_DIM}$header_text${TEXT_REPORTER_NC}"
            ;;
    esac
    echo
}

#######################################
# Report individual resource validation result
# Arguments: $1 - resource name, $2 - status, $3 - details, $4 - duration_ms, $5 - cached (optional)
# Returns: 0
#######################################
text_report_resource_result() {
    local resource_name="$1"
    local status="$2"
    local details="$3"
    local duration_ms="$4"
    local cached="${5:-false}"
    
    TEXT_REPORTER_RESOURCE_COUNT=$((TEXT_REPORTER_RESOURCE_COUNT + 1))
    
    # Format duration
    local duration_text
    if [[ $duration_ms -lt 1000 ]]; then
        duration_text="${duration_ms}ms"
    else
        duration_text="$((duration_ms / 1000)).$((duration_ms % 1000 / 100))s"
    fi
    
    # Cache indicator
    local cache_indicator=""
    if [[ "$cached" == "true" ]]; then
        cache_indicator=" ${TEXT_REPORTER_CACHED}${TEXT_REPORTER_CACHE_MARK}${TEXT_REPORTER_NC}"
    fi
    
    # Status-specific formatting
    case "$status" in
        "passed")
            echo -e "${TEXT_REPORTER_SUCCESS}${TEXT_REPORTER_CHECK_MARK} ${TEXT_REPORTER_BOLD}$resource_name${TEXT_REPORTER_NC}${cache_indicator} ${TEXT_REPORTER_DIM}($duration_text)${TEXT_REPORTER_NC}"
            if [[ -n "$details" && "$details" != *"All validations passed"* ]]; then
                echo -e "   ${TEXT_REPORTER_DIM}$details${TEXT_REPORTER_NC}"
            fi
            ;;
        "failed")
            TEXT_REPORTER_FAILED_RESOURCES+=("$resource_name")
            echo -e "${TEXT_REPORTER_ERROR}${TEXT_REPORTER_CROSS_MARK} ${TEXT_REPORTER_BOLD}$resource_name${TEXT_REPORTER_NC}${cache_indicator} ${TEXT_REPORTER_DIM}($duration_text)${TEXT_REPORTER_NC}"
            if [[ -n "$details" ]]; then
                # Parse and format error details
                text_report_error_details "$resource_name" "$details"
            fi
            ;;
        "warning")
            echo -e "${TEXT_REPORTER_WARNING}${TEXT_REPORTER_WARNING_MARK} ${TEXT_REPORTER_BOLD}$resource_name${TEXT_REPORTER_NC}${cache_indicator} ${TEXT_REPORTER_DIM}($duration_text)${TEXT_REPORTER_NC}"
            if [[ -n "$details" ]]; then
                echo -e "   ${TEXT_REPORTER_WARNING}$details${TEXT_REPORTER_NC}"
            fi
            ;;
    esac
    
    return 0
}

#######################################
# Format and display detailed error information with fix recommendations
# Arguments: $1 - resource name, $2 - error details
# Returns: 0
#######################################
text_report_error_details() {
    local resource_name="$1"
    local error_details="$2"
    
    # Parse error details and provide specific guidance
    echo -e "   ${TEXT_REPORTER_ERROR}Issues found:${TEXT_REPORTER_NC}"
    
    # Check for specific error patterns and provide targeted advice
    if [[ "$error_details" =~ "Missing required action" ]]; then
        local missing_action
        missing_action=$(echo "$error_details" | grep -o "Missing required action: [a-z]*" | cut -d: -f2 | tr -d ' ')
        echo -e "   ${TEXT_REPORTER_RED}•${TEXT_REPORTER_NC} Missing required action: ${TEXT_REPORTER_BOLD}$missing_action${TEXT_REPORTER_NC}"
        
        # Provide specific fix guidance
        case "$missing_action" in
            "install")
                echo -e "     ${TEXT_REPORTER_INFO_MARK} Add: ${TEXT_REPORTER_CYAN}\"install\") install_resource;;${TEXT_REPORTER_NC}"
                ;;
            "start")
                echo -e "     ${TEXT_REPORTER_INFO_MARK} Add: ${TEXT_REPORTER_CYAN}\"start\") start_resource;;${TEXT_REPORTER_NC}"
                ;;
            "stop")
                echo -e "     ${TEXT_REPORTER_INFO_MARK} Add: ${TEXT_REPORTER_CYAN}\"stop\") stop_resource;;${TEXT_REPORTER_NC}"
                ;;
            "status")
                echo -e "     ${TEXT_REPORTER_INFO_MARK} Add: ${TEXT_REPORTER_CYAN}\"status\") check_status;;${TEXT_REPORTER_NC}"
                ;;
            "logs")
                echo -e "     ${TEXT_REPORTER_INFO_MARK} Add: ${TEXT_REPORTER_CYAN}\"logs\") show_logs;;${TEXT_REPORTER_NC}"
                ;;
        esac
    fi
    
    if [[ "$error_details" =~ "Missing help pattern" ]]; then
        echo -e "   ${TEXT_REPORTER_RED}•${TEXT_REPORTER_NC} Help patterns not implemented"
        echo -e "     ${TEXT_REPORTER_INFO_MARK} Add support for: ${TEXT_REPORTER_CYAN}--help, -h, --version${TEXT_REPORTER_NC}"
    fi
    
    if [[ "$error_details" =~ "Missing error handling" ]]; then
        echo -e "   ${TEXT_REPORTER_RED}•${TEXT_REPORTER_NC} Error handling patterns missing"
        echo -e "     ${TEXT_REPORTER_INFO_MARK} Add at top: ${TEXT_REPORTER_CYAN}set -euo pipefail${TEXT_REPORTER_NC}"
        echo -e "     ${TEXT_REPORTER_INFO_MARK} Add cleanup: ${TEXT_REPORTER_CYAN}trap cleanup EXIT${TEXT_REPORTER_NC}"
    fi
    
    if [[ "$error_details" =~ "Missing file structure" ]]; then
        echo -e "   ${TEXT_REPORTER_RED}•${TEXT_REPORTER_NC} Required files missing"
        echo -e "     ${TEXT_REPORTER_INFO_MARK} Create: ${TEXT_REPORTER_CYAN}config/defaults.sh, config/messages.sh${TEXT_REPORTER_NC}"
        echo -e "     ${TEXT_REPORTER_INFO_MARK} Create: ${TEXT_REPORTER_CYAN}lib/common.sh${TEXT_REPORTER_NC}"
    fi
    
    # Generic error parsing for other issues
    if [[ "$error_details" =~ "Critical:" ]]; then
        local critical_info
        critical_info=$(echo "$error_details" | grep -o "Critical: [^,]*")
        echo -e "   ${TEXT_REPORTER_RED}•${TEXT_REPORTER_NC} $critical_info"
    fi
    
    if [[ "$error_details" =~ "Important:" ]]; then
        local important_info
        important_info=$(echo "$error_details" | grep -o "Important: [^,]*")
        echo -e "   ${TEXT_REPORTER_YELLOW}•${TEXT_REPORTER_NC} $important_info"
    fi
    
    echo
}

#######################################
# Report validation summary
# Arguments: $1 - total, $2 - passed, $3 - failed, $4 - duration_s
# Returns: 0
#######################################
text_report_summary() {
    local total="$1"
    local passed="$2"
    local failed="$3"
    local duration_s="$4"
    
    text_report_header "Validation Summary" "major"
    
    # Overall statistics
    echo -e "${TEXT_REPORTER_BOLD}Resources Validated:${TEXT_REPORTER_NC} $total"
    echo -e "${TEXT_REPORTER_SUCCESS}${TEXT_REPORTER_CHECK_MARK} Passed:${TEXT_REPORTER_NC} $passed"
    
    if [[ $failed -gt 0 ]]; then
        echo -e "${TEXT_REPORTER_ERROR}${TEXT_REPORTER_CROSS_MARK} Failed:${TEXT_REPORTER_NC} $failed"
    else
        echo -e "${TEXT_REPORTER_SUCCESS}${TEXT_REPORTER_CHECK_MARK} Failed:${TEXT_REPORTER_NC} 0"
    fi
    
    # Performance metrics
    echo -e "${TEXT_REPORTER_CLOCK_MARK} ${TEXT_REPORTER_BOLD}Duration:${TEXT_REPORTER_NC} ${duration_s}s"
    
    if [[ $total -gt 0 ]]; then
        local avg_ms=$((duration_s * 1000 / total))
        echo -e "${TEXT_REPORTER_CLOCK_MARK} ${TEXT_REPORTER_BOLD}Average:${TEXT_REPORTER_NC} ${avg_ms}ms per resource"
    fi
    
    # Success rate
    local success_rate=0
    if [[ $total -gt 0 ]]; then
        success_rate=$(( (passed * 100) / total ))
    fi
    echo -e "${TEXT_REPORTER_BOLD}Success Rate:${TEXT_REPORTER_NC} ${success_rate}%"
    
    echo
    
    # Failed resources summary
    if [[ $failed -gt 0 ]]; then
        text_report_header "Failed Resources" "minor"
        for resource in "${TEXT_REPORTER_FAILED_RESOURCES[@]}"; do
            echo -e "  ${TEXT_REPORTER_ERROR}${TEXT_REPORTER_CROSS_MARK}${TEXT_REPORTER_NC} $resource"
        done
        echo
        
        text_report_recommendations
    fi
    
    return 0
}

#######################################
# Display fix recommendations for common issues
# Arguments: None
# Returns: 0
#######################################
text_report_recommendations() {
    text_report_header "Fix Recommendations" "minor"
    
    echo -e "${TEXT_REPORTER_INFO_MARK} ${TEXT_REPORTER_BOLD}Quick Fixes:${TEXT_REPORTER_NC}"
    echo -e "  • Check contract compliance: ${TEXT_REPORTER_CYAN}cat contracts/v1.0/core.yaml${TEXT_REPORTER_NC}"
    echo -e "  • Validate specific resource: ${TEXT_REPORTER_CYAN}./validate-interfaces.sh --resource <name> --verbose${TEXT_REPORTER_NC}"
    echo -e "  • View detailed errors: ${TEXT_REPORTER_CYAN}./validate-interfaces.sh --format text --verbose${TEXT_REPORTER_NC}"
    echo
    
    echo -e "${TEXT_REPORTER_INFO_MARK} ${TEXT_REPORTER_BOLD}Common Patterns:${TEXT_REPORTER_NC}"
    echo -e "  • All resources need: install, start, stop, status, logs actions"
    echo -e "  • Add help support: --help, -h, --version flags"
    echo -e "  • Include error handling: set -euo pipefail, trap cleanup EXIT"
    echo -e "  • Create required files: config/defaults.sh, lib/common.sh"
    echo
    
    echo -e "${TEXT_REPORTER_INFO_MARK} ${TEXT_REPORTER_BOLD}Documentation:${TEXT_REPORTER_NC}"
    echo -e "  • Interface standards: ${TEXT_REPORTER_CYAN}docs/interface-standards.md${TEXT_REPORTER_NC}"
    echo -e "  • Testing strategy: ${TEXT_REPORTER_CYAN}docs/TESTING_STRATEGY.md${TEXT_REPORTER_NC}"
    echo -e "  • Layer 1 plan: ${TEXT_REPORTER_CYAN}docs/LAYER1_IMPLEMENTATION_PLAN.md${TEXT_REPORTER_NC}"
    echo
}

#######################################
# Report cache performance statistics
# Arguments: $1 - cache stats JSON
# Returns: 0
#######################################
text_report_cache_stats() {
    local cache_stats_json="$1"
    
    if [[ -z "$cache_stats_json" ]]; then
        return 0
    fi
    
    text_report_header "Cache Performance" "minor"
    
    # Parse JSON stats (basic parsing for compatibility)
    local cache_hits cache_misses hit_rate total_entries cache_size_kb
    cache_hits=$(echo "$cache_stats_json" | grep -o '"cache_hits":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' || echo "0")
    cache_misses=$(echo "$cache_stats_json" | grep -o '"cache_misses":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' || echo "0")
    hit_rate=$(echo "$cache_stats_json" | grep -o '"hit_rate_percent":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' || echo "0")
    total_entries=$(echo "$cache_stats_json" | grep -o '"total_entries":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' || echo "0")
    cache_size_kb=$(echo "$cache_stats_json" | grep -o '"cache_size_kb":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' || echo "0")
    
    echo -e "${TEXT_REPORTER_CACHE_MARK} ${TEXT_REPORTER_BOLD}Cache Hits:${TEXT_REPORTER_NC} $cache_hits"
    echo -e "${TEXT_REPORTER_CACHE_MARK} ${TEXT_REPORTER_BOLD}Cache Misses:${TEXT_REPORTER_NC} $cache_misses"
    echo -e "${TEXT_REPORTER_CACHE_MARK} ${TEXT_REPORTER_BOLD}Hit Rate:${TEXT_REPORTER_NC} ${hit_rate}%"
    echo -e "${TEXT_REPORTER_CACHE_MARK} ${TEXT_REPORTER_BOLD}Cached Entries:${TEXT_REPORTER_NC} $total_entries"
    echo -e "${TEXT_REPORTER_CACHE_MARK} ${TEXT_REPORTER_BOLD}Cache Size:${TEXT_REPORTER_NC} ${cache_size_kb}KB"
    
    # Performance insights
    if [[ $hit_rate -ge 80 ]]; then
        echo -e "   ${TEXT_REPORTER_SUCCESS}${TEXT_REPORTER_TROPHY_MARK} Excellent cache performance!${TEXT_REPORTER_NC}"
    elif [[ $hit_rate -ge 50 ]]; then
        echo -e "   ${TEXT_REPORTER_WARNING}${TEXT_REPORTER_WARNING_MARK} Good cache performance${TEXT_REPORTER_NC}"
    elif [[ $((cache_hits + cache_misses)) -gt 0 ]]; then
        echo -e "   ${TEXT_REPORTER_INFO_MARK} Cache warming up - performance will improve${TEXT_REPORTER_NC}"
    fi
    
    echo
}

#######################################
# Display final completion message
# Arguments: $1 - total failed count
# Returns: 0
#######################################
text_report_completion() {
    local failed_count="$1"
    
    if [[ $failed_count -eq 0 ]]; then
        text_report_header "${TEXT_REPORTER_TROPHY_MARK} All Resources Pass Validation!" "major"
        echo -e "${TEXT_REPORTER_SUCCESS}${TEXT_REPORTER_ROCKET_MARK} ${TEXT_REPORTER_BOLD}Congratulations!${TEXT_REPORTER_NC} All resources comply with interface standards."
        echo -e "   Your resource ecosystem is ready for production use."
    else
        text_report_header "${TEXT_REPORTER_WARNING_MARK} Validation Issues Found" "major"
        echo -e "${TEXT_REPORTER_WARNING}${TEXT_REPORTER_CROSS_MARK} ${TEXT_REPORTER_BOLD}$failed_count resource(s)${TEXT_REPORTER_NC} need attention."
        echo -e "   Please review the recommendations above and update the failing resources."
    fi
    
    echo
}

#######################################
# Get progress indicator for current validation
# Arguments: $1 - current number, $2 - total number
# Returns: 0, outputs progress indicator
#######################################
text_report_progress() {
    local current="$1"
    local total="$2"
    
    if [[ $total -le 1 ]]; then
        return 0
    fi
    
    local percentage=$((current * 100 / total))
    local progress_bar_width=20
    local filled_width=$((percentage * progress_bar_width / 100))
    
    printf "${TEXT_REPORTER_DIM}["
    for ((i=0; i<filled_width; i++)); do
        printf "="
    done
    for ((i=filled_width; i<progress_bar_width; i++)); do
        printf " "
    done
    printf "] %d%% (%d/%d)${TEXT_REPORTER_NC}\n" "$percentage" "$current" "$total"
}

# Export functions for use in other scripts
export -f text_reporter_init text_report_header text_report_resource_result
export -f text_report_error_details text_report_summary text_report_recommendations
export -f text_report_cache_stats text_report_completion text_report_progress