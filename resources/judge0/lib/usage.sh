#!/usr/bin/env bash
# Judge0 Usage Module
# Handles help text and usage information

#######################################
# Show usage/help information
#######################################
judge0::usage::show() {
    cat << EOF
${JUDGE0_DISPLAY_NAME} - ${JUDGE0_DESCRIPTION}

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -a, --action <ACTION>           Action to perform
                                   Default: install
                                   
    -h, --help                     Show this help message
    -y, --yes <yes|no>             Skip confirmation prompts
                                   Default: no
    -f, --force <yes|no>           Force action even if already installed/running
                                   Default: no

ACTIONS:
    install                        Install Judge0 with Docker
    uninstall                      Remove Judge0 and all data
    start                          Start Judge0 services
    stop                           Stop Judge0 services  
    restart                        Restart Judge0 services
    status                         Show detailed status information
    logs                           Show service logs
    info                           Display system information
    test                           Test API connectivity
    languages                      List available programming languages
    usage                          Show resource usage statistics
    submit                         Submit code for execution
    monitor                        Start security monitoring (runs in foreground)

CODE EXECUTION OPTIONS:
    --code <CODE>                  Source code to execute (for submit action)
    --language <LANG>              Programming language (default: javascript)
    --stdin <INPUT>                Standard input for the program
    --expected-output <OUTPUT>     Expected output for validation

RESOURCE LIMITS:
    --cpu-limit <SECONDS>          CPU time limit per submission
                                   Default: ${JUDGE0_CPU_TIME_LIMIT}
    --memory-limit <MB>            Memory limit per submission
                                   Default: $((JUDGE0_MEMORY_LIMIT / 1024))
    --workers <COUNT>              Number of worker containers
                                   Default: ${JUDGE0_WORKERS_COUNT}

SECURITY OPTIONS:
    --api-key <KEY>                API key for authentication
                                   (auto-generated if not provided)

EXAMPLES:
    # Install Judge0
    $0 --action install

    # Check status
    $0 --action status

    # Submit JavaScript code
    $0 --action submit --code 'console.log("Hello, World!");' --language javascript

    # Submit Python code with input
    $0 --action submit --code 'name = input(); print(f"Hello, {name}!")' \\
       --language python --stdin "Judge0"

    # List all programming languages
    $0 --action languages

    # View logs
    $0 --action logs

    # Uninstall Judge0
    $0 --action uninstall --force yes

SUPPORTED LANGUAGES:
    Judge0 supports 60+ programming languages including:
    - JavaScript, TypeScript, Python, Go, Rust, Java, C++, C
    - Ruby, PHP, Swift, Kotlin, R, Bash, SQL, and many more
    
    Run '$0 --action languages' for the complete list.

SECURITY NOTES:
    - Code execution happens in isolated Docker containers
    - Network access is disabled by default
    - Resource limits prevent infinite loops and memory bombs
    - API authentication is enabled by default

CONFIGURATION:
    - Port: ${JUDGE0_PORT}
    - API URL: ${JUDGE0_BASE_URL}
    - Config: ${JUDGE0_CONFIG_DIR}
    - Logs: ${JUDGE0_LOGS_DIR}

MORE INFORMATION:
    - Documentation: https://judge0.com/docs
    - Examples: ${SCRIPT_DIR}/examples/
    - Security: $0 --action security-validate

EOF
}

#######################################
# Show resource usage statistics
#######################################
judge0::usage::show_stats() {
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    log::header "📊 Judge0 Resource Usage"
    echo
    
    # Container statistics
    judge0::usage::show_container_stats
    
    # Submission statistics
    judge0::usage::show_submission_stats
    
    # Language usage
    judge0::usage::show_language_stats
    
    # Performance metrics
    judge0::usage::show_performance_metrics
}

#######################################
# Show container resource usage
#######################################
judge0::usage::show_container_stats() {
    log::info "🐳 Container Resources:"
    
    # Server stats
    local server_stats=$(judge0::get_container_stats)
    if [[ "$server_stats" != "{}" ]]; then
        local cpu=$(echo "$server_stats" | jq -r '.CPUPerc // "0%"')
        local mem=$(echo "$server_stats" | jq -r '.MemUsage // "0MiB / 0MiB"')
        local net_io=$(echo "$server_stats" | jq -r '.NetIO // "0B / 0B"')
        local block_io=$(echo "$server_stats" | jq -r '.BlockIO // "0B / 0B"')
        
        echo "  Server Container:"
        echo "    CPU: $cpu"
        echo "    Memory: $mem"
        echo "    Network I/O: $net_io"
        echo "    Block I/O: $block_io"
    fi
    
    # Worker stats
    local worker_stats=$(judge0::get_worker_stats)
    if [[ "$worker_stats" != "[]" ]]; then
        echo
        echo "  Worker Containers:"
        
        local total_cpu=0
        local total_mem=0
        local worker_count=0
        
        while IFS= read -r line; do
            local cpu=$(echo "$line" | jq -r '.CPUPerc // "0%"' | tr -d '%')
            local mem_usage=$(echo "$line" | jq -r '.MemUsage // "0MiB"' | cut -d' ' -f1 | tr -d 'MiB')
            
            total_cpu=$(echo "$total_cpu + $cpu" | bc)
            total_mem=$(echo "$total_mem + $mem_usage" | bc)
            ((worker_count++))
        done < <(echo "$worker_stats" | jq -c '.[]')
        
        if [[ $worker_count -gt 0 ]]; then
            local avg_cpu=$(echo "scale=2; $total_cpu / $worker_count" | bc)
            local avg_mem=$(echo "scale=2; $total_mem / $worker_count" | bc)
            
            echo "    Active Workers: $worker_count"
            echo "    Average CPU: ${avg_cpu}%"
            echo "    Average Memory: ${avg_mem}MiB"
        fi
    fi
    echo
}

#######################################
# Show submission statistics
#######################################
judge0::usage::show_submission_stats() {
    log::info "📈 Submission Statistics:"
    
    # Get stats from submissions directory
    if [[ -d "$JUDGE0_SUBMISSIONS_DIR" ]]; then
        local total=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" 2>/dev/null | wc -l)
        local today=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" -mtime -1 2>/dev/null | wc -l)
        local week=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" -mtime -7 2>/dev/null | wc -l)
        
        echo "  Total Submissions: $total"
        echo "  Last 24 Hours: $today"
        echo "  Last 7 Days: $week"
        
        # Status breakdown (if submissions are stored)
        if [[ $total -gt 0 ]]; then
            echo
            echo "  Status Breakdown:"
            local accepted=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" -exec grep -l '"status":{"id":3}' {} \; 2>/dev/null | wc -l)
            local errors=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" -exec grep -l '"status":{"id":[4-9]' {} \; 2>/dev/null | wc -l)
            
            echo "    Accepted: $accepted"
            echo "    Errors: $errors"
            
            if [[ $total -gt 0 ]]; then
                local success_rate=$(echo "scale=2; $accepted * 100 / $total" | bc)
                echo "    Success Rate: ${success_rate}%"
            fi
        fi
    else
        echo "  No submission data available"
    fi
    echo
}

#######################################
# Show language usage statistics  
#######################################
judge0::usage::show_language_stats() {
    log::info "🗣️  Language Usage:"
    
    # This would require tracking language usage
    # For now, show available language count
    if judge0::is_running; then
        local languages=$(judge0::api::get_languages 2>/dev/null || echo "[]")
        local count=$(echo "$languages" | jq 'length')
        
        echo "  Available Languages: $count"
        echo "  Popular Languages:"
        
        # Check for common languages
        for lang in "JavaScript" "Python" "Java" "C++" "Go"; do
            if echo "$languages" | jq -e ".[] | select(.name | contains(\"$lang\"))" >/dev/null 2>&1; then
                echo "    ✅ $lang"
            fi
        done
    else
        echo "  Service not running"
    fi
    echo
}

#######################################
# Show performance metrics
#######################################
judge0::usage::show_performance_metrics() {
    log::info "⚡ Performance Metrics:"
    
    # Disk usage
    local disk_usage=$(du -sh "$JUDGE0_DATA_DIR" 2>/dev/null | cut -f1 || echo "0")
    echo "  Disk Usage: $disk_usage"
    
    # Volume usage
    if docker volume inspect "$JUDGE0_VOLUME_NAME" >/dev/null 2>&1; then
        local volume_size=$(docker run --rm -v "$JUDGE0_VOLUME_NAME:/data" alpine du -sh /data 2>/dev/null | cut -f1 || echo "0")
        echo "  Volume Size: $volume_size"
    fi
    
    # Uptime
    if judge0::is_running; then
        local start_time=$(docker inspect "$JUDGE0_CONTAINER_NAME" --format '{{.State.StartedAt}}' 2>/dev/null)
        if [[ -n "$start_time" ]]; then
            local uptime=$(docker ps --filter "name=$JUDGE0_CONTAINER_NAME" --format "{{.Status}}" | grep -oE "[0-9]+ (seconds?|minutes?|hours?|days?)" || echo "unknown")
            echo "  Uptime: $uptime"
        fi
    fi
    
    echo
}

#######################################
# Show quick help for specific action
# Arguments:
#   $1 - Action name
#######################################
judge0::usage::action_help() {
    local action="$1"
    
    case "$action" in
        submit)
            cat << EOF
Submit code for execution

USAGE:
    $0 --action submit --code <CODE> [OPTIONS]

OPTIONS:
    --code <CODE>              Source code to execute (required)
    --language <LANG>          Programming language (default: javascript)
    --stdin <INPUT>           Standard input for the program
    --expected-output <OUT>    Expected output for validation

EXAMPLES:
    # Simple JavaScript
    $0 --action submit --code 'console.log("Hello!");' 

    # Python with input
    $0 --action submit --code 'print(input().upper())' --language python --stdin "hello"

    # C++ with validation
    $0 --action submit --code '#include <iostream>
    int main() { std::cout << "42\\n"; return 0; }' --language cpp --expected-output "42"
EOF
            ;;
        languages)
            cat << EOF
List available programming languages

USAGE:
    $0 --action languages

Shows all 60+ supported programming languages with their IDs and versions.
EOF
            ;;
        *)
            judge0::usage::show
            ;;
    esac
}