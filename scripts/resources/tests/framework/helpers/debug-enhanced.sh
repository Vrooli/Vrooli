#!/bin/bash
# ====================================================================
# Enhanced Debug Mode Functions
# ====================================================================
#
# Provides enhanced debugging capabilities with timing, progress tracking,
# and better error context for integration tests.
#
# Functions:
#   - debug_start_timer()     - Start timing a debug section
#   - debug_end_timer()       - End timing and report duration
#   - debug_phase()           - Start a new debug phase
#   - debug_step()            - Log a debug step with context
#   - debug_error_context()   - Provide detailed error context
#   - debug_resource_status() - Show detailed resource status
#
# ====================================================================

# Source logging functions
if [[ -f "${BASH_SOURCE%/*}/logging.sh" ]]; then
    source "${BASH_SOURCE%/*}/logging.sh"
fi

# Debug timing variables
declare -A DEBUG_TIMERS
DEBUG_PHASE_COUNT=0
DEBUG_STEP_COUNT=0

# Enhanced debug configuration
DEBUG_SHOW_TIMING="${DEBUG_SHOW_TIMING:-true}"
DEBUG_SHOW_MEMORY="${DEBUG_SHOW_MEMORY:-false}"
DEBUG_SHOW_DISK="${DEBUG_SHOW_DISK:-false}"
DEBUG_CONTEXT_LINES="${DEBUG_CONTEXT_LINES:-5}"

# Start a debug timer
debug_start_timer() {
    local timer_name="$1"
    DEBUG_TIMERS["$timer_name"]=$(date +%s%3N 2>/dev/null || echo $(($(date +%s) * 1000)))
    
    if [[ "${DEBUG:-false}" == "true" && "$DEBUG_SHOW_TIMING" == "true" ]]; then
        log_debug "⏱️  Started timer: $timer_name"
    fi
}

# End a debug timer and report duration
debug_end_timer() {
    local timer_name="$1"
    local start_time="${DEBUG_TIMERS[$timer_name]:-0}"
    local end_time=$(date +%s%3N 2>/dev/null || echo $(($(date +%s) * 1000)))
    local duration=$((end_time - start_time))
    
    if [[ "${DEBUG:-false}" == "true" && "$DEBUG_SHOW_TIMING" == "true" ]]; then
        if [[ $duration -gt 1000 ]]; then
            local seconds=$((duration / 1000))
            local ms=$((duration % 1000))
            log_debug "⏱️  Timer $timer_name: ${seconds}.${ms}s"
        else
            log_debug "⏱️  Timer $timer_name: ${duration}ms"
        fi
    fi
    
    unset DEBUG_TIMERS["$timer_name"]
}

# Start a new debug phase
debug_phase() {
    local phase_name="$1"
    local phase_description="$2"
    
    DEBUG_PHASE_COUNT=$((DEBUG_PHASE_COUNT + 1))
    DEBUG_STEP_COUNT=0
    
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "\n${BLUE}[PHASE $DEBUG_PHASE_COUNT]${NC} 🎯 $phase_name" >&2
        if [[ -n "$phase_description" ]]; then
            log_debug "$phase_description"
        fi
        debug_start_timer "phase_$DEBUG_PHASE_COUNT"
    fi
}

# Log a debug step with context
debug_step() {
    local step_description="$1"
    local step_details="$2"
    
    DEBUG_STEP_COUNT=$((DEBUG_STEP_COUNT + 1))
    
    if [[ "${DEBUG:-false}" == "true" ]]; then
        log_debug "📍 Step $DEBUG_PHASE_COUNT.$DEBUG_STEP_COUNT: $step_description"
        if [[ -n "$step_details" ]]; then
            echo "    💡 $step_details" >&2
        fi
    fi
}

# Provide detailed error context
debug_error_context() {
    local error_message="$1"
    local context_info="$2"
    local exit_code="${3:-1}"
    
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "\n${RED}[ERROR CONTEXT]${NC} 🚨 Detailed Error Information" >&2
        log_error "$error_message"
        
        if [[ -n "$context_info" ]]; then
            echo "📋 Context: $context_info" >&2
        fi
        
        # Show system context
        echo "🖥️  System Context:" >&2
        echo "   - Exit Code: $exit_code" >&2
        echo "   - Current Directory: $(pwd)" >&2
        echo "   - User: $(whoami)" >&2
        echo "   - Time: $(date)" >&2
        
        # Show resource context if available
        if command -v docker >/dev/null 2>&1; then
            echo "🐳 Docker Context:" >&2
            echo "   - Running Containers: $(docker ps --format 'table {{.Names}}\t{{.Status}}' 2>/dev/null | wc -l || echo 'unknown')" >&2
        fi
        
        # Show memory if enabled
        if [[ "$DEBUG_SHOW_MEMORY" == "true" ]] && command -v free >/dev/null 2>&1; then
            echo "💾 Memory Usage:" >&2
            free -h | head -2 | sed 's/^/   /' >&2
        fi
        
        # Show disk if enabled
        if [[ "$DEBUG_SHOW_DISK" == "true" ]] && command -v df >/dev/null 2>&1; then
            echo "💿 Disk Usage:" >&2
            df -h . | tail -1 | sed 's/^/   /' >&2
        fi
        
        # Show recent logs if available
        if [[ -n "${HTTP_LOG_FILE:-}" && -f "$HTTP_LOG_FILE" ]]; then
            echo "📄 Recent HTTP Logs:" >&2
            tail -n "$DEBUG_CONTEXT_LINES" "$HTTP_LOG_FILE" 2>/dev/null | sed 's/^/   /' >&2
        fi
    fi
}

# Show detailed resource status
debug_resource_status() {
    local resource="$1"
    local status="$2"
    local details="$3"
    
    if [[ "${DEBUG:-false}" == "true" ]]; then
        case "$status" in
            "healthy")
                log_debug "✅ $resource: Ready and operational"
                ;;
            "unhealthy")
                log_debug "❌ $resource: Service responding but unhealthy"
                ;;
            "unreachable")
                log_debug "🔌 $resource: Cannot connect to service"
                ;;
            "timeout")
                log_debug "⏰ $resource: Connection timeout"
                ;;
            "starting")
                log_debug "🔄 $resource: Service is starting up"
                ;;
            *)
                log_debug "❓ $resource: Unknown status ($status)"
                ;;
        esac
        
        if [[ -n "$details" ]]; then
            echo "    💬 $details" >&2
        fi
    fi
}

# Show debug summary at the end of a phase
debug_phase_end() {
    local phase_name="$1"
    local success_count="${2:-0}"
    local failure_count="${3:-0}"
    
    if [[ "${DEBUG:-false}" == "true" ]]; then
        debug_end_timer "phase_$DEBUG_PHASE_COUNT"
        
        echo -e "\n${CYAN}[PHASE SUMMARY]${NC} 📊 $phase_name Results" >&2
        echo "   ✅ Successes: $success_count" >&2
        echo "   ❌ Failures: $failure_count" >&2
        echo "   📈 Steps Completed: $DEBUG_STEP_COUNT" >&2
        
        if [[ $failure_count -gt 0 ]]; then
            echo "   💡 Use --debug --verbose for more detailed error information" >&2
        fi
    fi
}

# Initialize enhanced debug mode
init_enhanced_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "\n${BOLD}${PURPLE}[DEBUG MODE]${NC} 🔍 Enhanced debugging enabled" >&2
        echo "   📝 HTTP logging: ${HTTP_LOG_ENABLED:-false}" >&2
        echo "   ⏱️  Timing: $DEBUG_SHOW_TIMING" >&2
        echo "   💾 Memory monitoring: $DEBUG_SHOW_MEMORY" >&2
        echo "   💿 Disk monitoring: $DEBUG_SHOW_DISK" >&2
        
        # Start overall timer
        debug_start_timer "total_execution"
    fi
}

# Finalize enhanced debug mode
finalize_enhanced_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        debug_end_timer "total_execution"
        echo -e "\n${BOLD}${PURPLE}[DEBUG COMPLETE]${NC} 🏁 Debug session ended" >&2
        
        # Clean up debug files if configured
        if [[ "${DEBUG_CLEANUP:-true}" == "true" ]]; then
            for timer in "${!DEBUG_TIMERS[@]}"; do
                unset DEBUG_TIMERS["$timer"]
            done
        fi
    fi
}