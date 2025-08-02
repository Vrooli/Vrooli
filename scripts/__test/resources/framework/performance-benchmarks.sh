#!/bin/bash
# ====================================================================
# Performance Benchmarking Framework
# ====================================================================
#
# Provides standardized performance tests for each resource category.
# Measures key performance metrics and compares against baselines.
#
# Usage:
#   source "$SCRIPT_DIR/framework/performance-benchmarks.sh"
#   run_performance_benchmark "$RESOURCE_NAME" "$CATEGORY" "$PORT"
#
# ====================================================================

set -euo pipefail

# Performance benchmark counters
BENCHMARK_TESTS_RUN=0
BENCHMARK_TESTS_PASSED=0
BENCHMARK_TESTS_FAILED=0

# Colors for benchmark output
PB_GREEN='\033[0;32m'
PB_RED='\033[0;31m'
PB_YELLOW='\033[1;33m'
PB_BLUE='\033[0;34m'
PB_CYAN='\033[0;36m'
PB_BOLD='\033[1m'
PB_NC='\033[0m'

# Performance benchmark logging
pb_log_info() {
    echo -e "${PB_BLUE}[BENCHMARK]${PB_NC} $1"
}

pb_log_success() {
    echo -e "${PB_GREEN}[BENCHMARK]${PB_NC} ✅ $1"
}

pb_log_error() {
    echo -e "${PB_RED}[BENCHMARK]${PB_NC} ❌ $1"
}

pb_log_warning() {
    echo -e "${PB_YELLOW}[BENCHMARK]${PB_NC} ⚠️  $1"
}

pb_log_metric() {
    echo -e "${PB_CYAN}[METRIC]${PB_NC} 📊 $1"
}

# Performance baselines (in milliseconds unless specified)
declare -A PERFORMANCE_BASELINES=(
    # AI Resource Baselines
    ["ai_model_list_response_time"]="5000"      # 5 seconds
    ["ai_simple_inference_time"]="30000"        # 30 seconds
    ["ai_health_check_time"]="2000"             # 2 seconds
    
    # Storage Resource Baselines
    ["storage_connection_time"]="1000"          # 1 second
    ["storage_write_operation_time"]="5000"     # 5 seconds
    ["storage_read_operation_time"]="2000"      # 2 seconds
    ["storage_health_check_time"]="1000"        # 1 second
    
    # Automation Resource Baselines
    ["automation_ui_load_time"]="5000"          # 5 seconds
    ["automation_api_response_time"]="3000"     # 3 seconds
    ["automation_workflow_list_time"]="10000"   # 10 seconds
    
    # Agent Resource Baselines
    ["agent_screenshot_time"]="10000"           # 10 seconds
    ["agent_action_response_time"]="5000"       # 5 seconds
    ["agent_capability_check_time"]="3000"      # 3 seconds
    
    # Search Resource Baselines
    ["search_query_response_time"]="10000"      # 10 seconds
    ["search_index_load_time"]="5000"           # 5 seconds
    
    # Execution Resource Baselines
    ["execution_code_compile_time"]="15000"     # 15 seconds
    ["execution_simple_run_time"]="10000"       # 10 seconds
    ["execution_language_list_time"]="2000"     # 2 seconds
)

# Utility function to measure execution time
measure_execution_time() {
    local command="$1"
    local description="$2"
    local timeout_duration="${3:-60}"
    
    pb_log_info "Measuring: $description"
    
    local start_time=$(date +%s%3N)  # Milliseconds
    local result=""
    local exit_code=0
    
    # Execute command with timeout
    if timeout "${timeout_duration}s" bash -c "$command" >/dev/null 2>&1; then
        exit_code=$?
    else
        exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            pb_log_warning "Command timed out after ${timeout_duration}s"
        fi
    fi
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    pb_log_metric "$description: ${duration}ms (exit code: $exit_code)"
    
    echo "$duration"
    return $exit_code
}

# Compare performance against baseline
compare_with_baseline() {
    local metric_name="$1"
    local measured_value="$2"
    local baseline="${PERFORMANCE_BASELINES[$metric_name]:-0}"
    
    if [[ "$baseline" -eq 0 ]]; then
        pb_log_warning "No baseline defined for $metric_name"
        return 0
    fi
    
    local percentage=$((measured_value * 100 / baseline))
    
    if [[ $percentage -le 50 ]]; then
        pb_log_success "$metric_name: Excellent performance (${percentage}% of baseline)"
        return 0
    elif [[ $percentage -le 100 ]]; then
        pb_log_success "$metric_name: Good performance (${percentage}% of baseline)"
        return 0
    elif [[ $percentage -le 150 ]]; then
        pb_log_warning "$metric_name: Acceptable performance (${percentage}% of baseline)"
        return 0
    else
        pb_log_error "$metric_name: Poor performance (${percentage}% of baseline, needs optimization)"
        return 1
    fi
}

# AI Resource Performance Benchmarks
benchmark_ai_performance() {
    local resource_name="$1"
    local port="$2"
    
    pb_log_info "🧠 Running AI performance benchmarks for $resource_name"
    
    local benchmark_results=()
    
    # Benchmark 1: Health check response time
    local health_time
    health_time=$(measure_execution_time "curl -s --max-time 30 http://localhost:${port}/health || curl -s --max-time 30 http://localhost:${port}/" "AI health check" 30)
    benchmark_results+=("health_check:$health_time")
    compare_with_baseline "ai_health_check_time" "$health_time"
    
    # Benchmark 2: Model listing performance
    case "$resource_name" in
        "ollama")
            local model_list_time
            model_list_time=$(measure_execution_time "curl -s --max-time 30 http://localhost:${port}/api/tags" "Model listing" 30)
            benchmark_results+=("model_list:$model_list_time")
            compare_with_baseline "ai_model_list_response_time" "$model_list_time"
            
            # Benchmark 3: Simple inference (if models available)
            local models_response
            models_response=$(curl -s --max-time 10 "http://localhost:${port}/api/tags" 2>/dev/null)
            
            if [[ -n "$models_response" ]] && echo "$models_response" | jq -e '.models[0].name' >/dev/null 2>&1; then
                local model_name
                model_name=$(echo "$models_response" | jq -r '.models[0].name')
                
                local inference_command="curl -s --max-time 60 -X POST http://localhost:${port}/api/generate -H 'Content-Type: application/json' -d '{\"model\": \"$model_name\", \"prompt\": \"Hi\", \"stream\": false}'"
                local inference_time
                inference_time=$(measure_execution_time "$inference_command" "Simple inference" 60)
                benchmark_results+=("inference:$inference_time")
                compare_with_baseline "ai_simple_inference_time" "$inference_time"
            fi
            ;;
        "whisper")
            # Create minimal test audio file for transcription benchmark
            local test_audio="/tmp/benchmark_audio.wav"
            printf '\x52\x49\x46\x46\x24\x08\x00\x00\x57\x41\x56\x45\x66\x6d\x74\x20\x10\x00\x00\x00\x01\x00\x01\x00\x40\x1f\x00\x00\x80\x3e\x00\x00\x02\x00\x10\x00\x64\x61\x74\x61\x00\x08\x00\x00' > "$test_audio"
            dd if=/dev/zero bs=1 count=1024 >> "$test_audio" 2>/dev/null
            
            local transcription_command="curl -s --max-time 60 -X POST http://localhost:${port}/transcribe -F 'audio=@${test_audio}'"
            local transcription_time
            transcription_time=$(measure_execution_time "$transcription_command" "Audio transcription" 60)
            benchmark_results+=("transcription:$transcription_time")
            
            rm -f "$test_audio"
            ;;
        *)
            pb_log_warning "Specific AI benchmarks not implemented for $resource_name"
            ;;
    esac
    
    # Return benchmark summary
    echo "ai_benchmarks:$(IFS=,; echo "${benchmark_results[*]}")"
}

# Storage Resource Performance Benchmarks
benchmark_storage_performance() {
    local resource_name="$1"
    local port="$2"
    
    pb_log_info "💾 Running storage performance benchmarks for $resource_name"
    
    local benchmark_results=()
    
    # Benchmark 1: Connection establishment time
    local connection_time
    connection_time=$(measure_execution_time "timeout 10 bash -c '</dev/tcp/localhost/$port'" "Connection establishment" 10)
    benchmark_results+=("connection:$connection_time")
    compare_with_baseline "storage_connection_time" "$connection_time"
    
    # Benchmark 2: Health check response time
    case "$resource_name" in
        "postgres")
            if which pg_isready >/dev/null 2>&1; then
                local health_time
                health_time=$(measure_execution_time "pg_isready -h localhost -p $port -t 10" "PostgreSQL health check" 10)
                benchmark_results+=("health_check:$health_time")
                compare_with_baseline "storage_health_check_time" "$health_time"
            fi
            ;;
        "redis")
            if which redis-cli >/dev/null 2>&1; then
                local ping_time
                ping_time=$(measure_execution_time "redis-cli -h localhost -p $port ping" "Redis ping" 5)
                benchmark_results+=("ping:$ping_time")
                compare_with_baseline "storage_health_check_time" "$ping_time"
                
                # Benchmark 3: Simple read/write operations
                local write_time
                write_time=$(measure_execution_time "redis-cli -h localhost -p $port SET benchmark_test 'test_value'" "Redis write operation" 5)
                benchmark_results+=("write:$write_time")
                compare_with_baseline "storage_write_operation_time" "$write_time"
                
                local read_time
                read_time=$(measure_execution_time "redis-cli -h localhost -p $port GET benchmark_test" "Redis read operation" 5)
                benchmark_results+=("read:$read_time")
                compare_with_baseline "storage_read_operation_time" "$read_time"
                
                # Cleanup
                redis-cli -h localhost -p "$port" DEL benchmark_test >/dev/null 2>&1
            fi
            ;;
        "minio")
            local health_time
            health_time=$(measure_execution_time "curl -s --max-time 10 http://localhost:${port}/minio/health/live" "MinIO health check" 10)
            benchmark_results+=("health_check:$health_time")
            compare_with_baseline "storage_health_check_time" "$health_time"
            ;;
        "qdrant")
            local health_time
            health_time=$(measure_execution_time "curl -s --max-time 10 http://localhost:${port}/health" "Qdrant health check" 10)
            benchmark_results+=("health_check:$health_time")
            compare_with_baseline "storage_health_check_time" "$health_time"
            ;;
        *)
            pb_log_warning "Specific storage benchmarks not implemented for $resource_name"
            ;;
    esac
    
    echo "storage_benchmarks:$(IFS=,; echo "${benchmark_results[*]}")"
}

# Automation Resource Performance Benchmarks
benchmark_automation_performance() {
    local resource_name="$1"
    local port="$2"
    
    pb_log_info "⚙️ Running automation performance benchmarks for $resource_name"
    
    local benchmark_results=()
    
    # Benchmark 1: UI load time
    local ui_load_time
    ui_load_time=$(measure_execution_time "curl -s --max-time 30 http://localhost:${port}/" "UI page load" 30)
    benchmark_results+=("ui_load:$ui_load_time")
    compare_with_baseline "automation_ui_load_time" "$ui_load_time"
    
    # Benchmark 2: API response time
    case "$resource_name" in
        "n8n")
            local api_time
            api_time=$(measure_execution_time "curl -s --max-time 15 http://localhost:${port}/rest/workflows" "n8n API response" 15)
            benchmark_results+=("api_response:$api_time")
            compare_with_baseline "automation_api_response_time" "$api_time"
            ;;
        "node-red")
            local flows_time
            flows_time=$(measure_execution_time "curl -s --max-time 15 http://localhost:${port}/flows" "Node-RED flows API" 15)
            benchmark_results+=("flows_api:$flows_time")
            compare_with_baseline "automation_api_response_time" "$flows_time"
            ;;
        "windmill")
            local version_time
            version_time=$(measure_execution_time "curl -s --max-time 10 http://localhost:${port}/api/version" "Windmill version API" 10)
            benchmark_results+=("version_api:$version_time")
            compare_with_baseline "automation_api_response_time" "$version_time"
            ;;
        *)
            pb_log_warning "Specific automation benchmarks not implemented for $resource_name"
            ;;
    esac
    
    echo "automation_benchmarks:$(IFS=,; echo "${benchmark_results[*]}")"
}

# Agent Resource Performance Benchmarks
benchmark_agent_performance() {
    local resource_name="$1"
    local port="$2"
    
    pb_log_info "🤖 Running agent performance benchmarks for $resource_name"
    
    local benchmark_results=()
    
    case "$resource_name" in
        "agent-s2")
            # Benchmark 1: Screenshot capability
            local screenshot_time
            screenshot_time=$(measure_execution_time "curl -s --max-time 30 -X POST 'http://localhost:${port}/screenshot?format=png&response_format=binary'" "Screenshot capture" 30)
            benchmark_results+=("screenshot:$screenshot_time")
            compare_with_baseline "agent_screenshot_time" "$screenshot_time"
            
            # Benchmark 2: Mouse position query
            local mouse_time
            mouse_time=$(measure_execution_time "curl -s --max-time 10 -X POST http://localhost:${port}/mouse/position" "Mouse position query" 10)
            benchmark_results+=("mouse_query:$mouse_time")
            compare_with_baseline "agent_action_response_time" "$mouse_time"
            ;;
        "browserless")
            # Benchmark 1: Browser screenshot
            local browser_screenshot_time
            local test_payload='{"url": "data:text/html,<html><body>Test</body></html>"}'
            browser_screenshot_time=$(measure_execution_time "curl -s --max-time 30 -X POST http://localhost:${port}/chrome/screenshot -H 'Content-Type: application/json' -d '$test_payload'" "Browser screenshot" 30)
            benchmark_results+=("browser_screenshot:$browser_screenshot_time")
            compare_with_baseline "agent_screenshot_time" "$browser_screenshot_time"
            ;;
        "claude-code")
            # Benchmark CLI response time
            local cli_time="0"
            if which claude >/dev/null 2>&1; then
                cli_time=$(measure_execution_time "claude --version" "Claude CLI version" 10)
            elif which claude-code >/dev/null 2>&1; then
                cli_time=$(measure_execution_time "claude-code --version" "Claude Code CLI version" 10)
            fi
            benchmark_results+=("cli_response:$cli_time")
            ;;
        *)
            pb_log_warning "Specific agent benchmarks not implemented for $resource_name"
            ;;
    esac
    
    echo "agent_benchmarks:$(IFS=,; echo "${benchmark_results[*]}")"
}

# Search Resource Performance Benchmarks
benchmark_search_performance() {
    local resource_name="$1"
    local port="$2"
    
    pb_log_info "🔍 Running search performance benchmarks for $resource_name"
    
    local benchmark_results=()
    
    case "$resource_name" in
        "searxng")
            # Benchmark 1: Search query performance
            local search_time
            search_time=$(measure_execution_time "curl -s --max-time 30 'http://localhost:${port}/search?q=test&format=json'" "Search query" 30)
            benchmark_results+=("search_query:$search_time")
            compare_with_baseline "search_query_response_time" "$search_time"
            
            # Benchmark 2: Index/stats load time
            local stats_time
            stats_time=$(measure_execution_time "curl -s --max-time 15 http://localhost:${port}/stats" "Search stats" 15)
            benchmark_results+=("stats_load:$stats_time")
            compare_with_baseline "search_index_load_time" "$stats_time"
            ;;
        *)
            pb_log_warning "Specific search benchmarks not implemented for $resource_name"
            ;;
    esac
    
    echo "search_benchmarks:$(IFS=,; echo "${benchmark_results[*]}")"
}

# Execution Resource Performance Benchmarks
benchmark_execution_performance() {
    local resource_name="$1"
    local port="$2"
    
    pb_log_info "⚡ Running execution performance benchmarks for $resource_name"
    
    local benchmark_results=()
    
    case "$resource_name" in
        "judge0")
            # Benchmark 1: Language list performance
            local languages_time
            languages_time=$(measure_execution_time "curl -s --max-time 15 http://localhost:${port}/languages" "Languages list" 15)
            benchmark_results+=("languages_list:$languages_time")
            compare_with_baseline "execution_language_list_time" "$languages_time"
            
            # Benchmark 2: Simple code execution
            local execution_payload='{"source_code": "print(\"Hello\")", "language_id": 71}'
            local submission_time
            submission_time=$(measure_execution_time "curl -s --max-time 30 -X POST http://localhost:${port}/submissions -H 'Content-Type: application/json' -d '$execution_payload'" "Code submission" 30)
            benchmark_results+=("code_submission:$submission_time")
            compare_with_baseline "execution_simple_run_time" "$submission_time"
            ;;
        *)
            pb_log_warning "Specific execution benchmarks not implemented for $resource_name"
            ;;
    esac
    
    echo "execution_benchmarks:$(IFS=,; echo "${benchmark_results[*]}")"
}

# Main performance benchmark function
run_performance_benchmark() {
    local resource_name="${1:-unknown}"
    local category="${2:-}"
    local port="${3:-8080}"
    
    pb_log_info "🏁 Running performance benchmarks for: $resource_name (category: $category)"
    echo
    
    # Reset counters
    BENCHMARK_TESTS_RUN=0
    BENCHMARK_TESTS_PASSED=0
    BENCHMARK_TESTS_FAILED=0
    
    local benchmark_results=""
    
    case "$category" in
        "ai")
            benchmark_results=$(benchmark_ai_performance "$resource_name" "$port")
            ;;
        "storage")
            benchmark_results=$(benchmark_storage_performance "$resource_name" "$port")
            ;;
        "automation")
            benchmark_results=$(benchmark_automation_performance "$resource_name" "$port")
            ;;
        "agents")
            benchmark_results=$(benchmark_agent_performance "$resource_name" "$port")
            ;;
        "search")
            benchmark_results=$(benchmark_search_performance "$resource_name" "$port")
            ;;
        "execution")
            benchmark_results=$(benchmark_execution_performance "$resource_name" "$port")
            ;;
        *)
            pb_log_error "Unknown category for benchmarking: $category"
            return 1
            ;;
    esac
    
    echo
    
    # Print benchmark summary
    pb_log_info "Performance benchmark summary for $resource_name:"
    if [[ -n "$benchmark_results" ]]; then
        echo "  Results: $benchmark_results"
    else
        echo "  No benchmark results collected"
    fi
    
    pb_log_success "Performance benchmarking completed for $resource_name"
    return 0
}

# List performance baselines
list_performance_baselines() {
    pb_log_info "📊 Performance Baselines"
    echo
    
    echo "🧠 AI Resource Baselines:"
    echo "  • Model list response: ${PERFORMANCE_BASELINES[ai_model_list_response_time]}ms"
    echo "  • Simple inference: ${PERFORMANCE_BASELINES[ai_simple_inference_time]}ms"
    echo "  • Health check: ${PERFORMANCE_BASELINES[ai_health_check_time]}ms"
    echo
    
    echo "💾 Storage Resource Baselines:"
    echo "  • Connection time: ${PERFORMANCE_BASELINES[storage_connection_time]}ms"
    echo "  • Write operation: ${PERFORMANCE_BASELINES[storage_write_operation_time]}ms"
    echo "  • Read operation: ${PERFORMANCE_BASELINES[storage_read_operation_time]}ms"
    echo
    
    echo "⚙️ Automation Resource Baselines:"
    echo "  • UI load time: ${PERFORMANCE_BASELINES[automation_ui_load_time]}ms"
    echo "  • API response: ${PERFORMANCE_BASELINES[automation_api_response_time]}ms"
    echo
    
    echo "🤖 Agent Resource Baselines:"
    echo "  • Screenshot time: ${PERFORMANCE_BASELINES[agent_screenshot_time]}ms"
    echo "  • Action response: ${PERFORMANCE_BASELINES[agent_action_response_time]}ms"
    echo
    
    echo "🔍 Search Resource Baselines:"
    echo "  • Query response: ${PERFORMANCE_BASELINES[search_query_response_time]}ms"
    echo "  • Index load: ${PERFORMANCE_BASELINES[search_index_load_time]}ms"
    echo
    
    echo "⚡ Execution Resource Baselines:"
    echo "  • Code execution: ${PERFORMANCE_BASELINES[execution_simple_run_time]}ms"
    echo "  • Language list: ${PERFORMANCE_BASELINES[execution_language_list_time]}ms"
}

# Export functions for use in other scripts
export -f run_performance_benchmark
export -f list_performance_baselines
export -f measure_execution_time
export -f compare_with_baseline