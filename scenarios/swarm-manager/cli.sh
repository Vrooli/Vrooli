#!/usr/bin/env bash

# Swarm Manager CLI
# Autonomous task orchestration for Vrooli

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKS_DIR="${SCRIPT_DIR}/tasks"
API_BASE="http://localhost:8095"
SERVICE_NAME="swarm-manager"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Ensure tasks directories exist
mkdir -p "${TASKS_DIR}"/{active,backlog/{manual,generated},staged,completed,failed}

# Main CLI function
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        start)
            start_service "$@"
            ;;
        stop)
            stop_service "$@"
            ;;
        status)
            show_status "$@"
            ;;
        add-task|add)
            add_task "$@"
            ;;
        list-tasks|list|ls)
            list_tasks "$@"
            ;;
        execute|exec)
            execute_task "$@"
            ;;
        analyze)
            analyze_task "$@"
            ;;
        scan-problems|scan)
            scan_problems "$@"
            ;;
        list-problems|problems)
            list_problems "$@"
            ;;
        resolve-problem|resolve)
            resolve_problem "$@"
            ;;
        agents)
            list_agents "$@"
            ;;
        config)
            manage_config "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        metrics)
            show_metrics "$@"
            ;;
        scan-problems)
            scan_problems "$@"
            ;;
        problems|problem)
            manage_problems "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}Unknown command: $command${NC}"
            show_help
            exit 1
            ;;
    esac
}

# Start the swarm manager service
start_service() {
    echo -e "${BLUE}Starting Swarm Manager...${NC}"
    
    # Start PostgreSQL if not running
    if ! pg_isready -h localhost -p 5433 &>/dev/null; then
        echo "Starting PostgreSQL..."
        vrooli resource postgres start || true
    fi
    
    # Start Redis if not running
    if ! redis-cli -p 6380 ping &>/dev/null; then
        echo "Starting Redis..."
        vrooli resource redis start || true
    fi
    
    # Start n8n if not running
    if ! curl -s http://localhost:5678/healthz &>/dev/null; then
        echo "Starting n8n..."
        vrooli resource n8n start || true
    fi
    
    # Initialize database schema
    echo "Initializing database..."
    PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -d postgres \
        -f "${SCRIPT_DIR}/initialization/postgres/01-schema.sql" 2>/dev/null || true
    
    # Start API server
    echo "Starting API server..."
    cd "${SCRIPT_DIR}/api/src"
    
    # Install Go dependencies if needed
    if [ ! -f go.mod ]; then
        go mod init swarm-manager
        go get github.com/gofiber/fiber/v2
        go get github.com/lib/pq
        go get github.com/google/uuid
        go get gopkg.in/yaml.v3
    fi
    
    # Build and run
    go build -o swarm-manager-api main.go
    SERVICE_PORT=8095 nohup ./swarm-manager-api > "${SCRIPT_DIR}/logs/api.log" 2>&1 &
    echo $! > "${SCRIPT_DIR}/api.pid"
    
    # Start UI server
    echo "Starting UI server..."
    cd "${SCRIPT_DIR}/ui"
    python3 -m http.server 4000 > "${SCRIPT_DIR}/logs/ui.log" 2>&1 &
    echo $! > "${SCRIPT_DIR}/ui.pid"
    
    # Import n8n workflows
    echo "Importing workflows..."
    for workflow in "${SCRIPT_DIR}/initialization/n8n"/*.json; do
        if [ -f "$workflow" ]; then
            echo "  Importing $(basename "$workflow")..."
            curl -X POST http://localhost:5678/api/v1/workflows \
                -H "Content-Type: application/json" \
                -d "@$workflow" &>/dev/null || true
        fi
    done
    
    echo -e "${GREEN}✓ Swarm Manager started successfully${NC}"
    echo ""
    echo "  API: http://localhost:8095"
    echo "  UI:  http://localhost:4000"
    echo ""
    echo "Run 'swarm-manager status' to check system status"
}

# Stop the service
stop_service() {
    echo -e "${BLUE}Stopping Swarm Manager...${NC}"
    
    # Stop API server
    if [ -f "${SCRIPT_DIR}/api.pid" ]; then
        kill "$(cat "${SCRIPT_DIR}/api.pid")" 2>/dev/null || true
        rm "${SCRIPT_DIR}/api.pid"
    fi
    
    # Stop UI server
    if [ -f "${SCRIPT_DIR}/ui.pid" ]; then
        kill "$(cat "${SCRIPT_DIR}/ui.pid")" 2>/dev/null || true
        rm "${SCRIPT_DIR}/ui.pid"
    fi
    
    echo -e "${GREEN}✓ Swarm Manager stopped${NC}"
}

# Show service status
show_status() {
    echo -e "${BLUE}Swarm Manager Status${NC}"
    echo "===================="
    
    # Check API
    if curl -s "${API_BASE}/health" &>/dev/null; then
        echo -e "API Server:  ${GREEN}● Running${NC}"
    else
        echo -e "API Server:  ${RED}○ Stopped${NC}"
    fi
    
    # Check UI
    if curl -s http://localhost:4000 &>/dev/null; then
        echo -e "UI Server:   ${GREEN}● Running${NC}"
    else
        echo -e "UI Server:   ${RED}○ Stopped${NC}"
    fi
    
    # Show task counts
    echo ""
    echo "Task Status:"
    echo "------------"
    
    for folder in active staged backlog/manual backlog/generated completed failed; do
        count=$(find "${TASKS_DIR}/${folder}" -name "*.yaml" -o -name "*.yml" 2>/dev/null | wc -l)
        folder_name=$(echo "$folder" | sed 's|/| |g')
        printf "  %-20s %3d tasks\n" "$folder_name:" "$count"
    done
    
    # Show problem discovery status
    echo ""
    echo "Problem Discovery:"
    echo "-----------------"
    problems_count=$(find /home/matthalloran8/Vrooli -name "PROBLEMS.md" 2>/dev/null | wc -l)
    troubleshooting_count=$(find /home/matthalloran8/Vrooli -name "TROUBLESHOOTING.md" 2>/dev/null | wc -l)
    printf "  %-20s %3d files\n" "PROBLEMS.md:" "$problems_count"
    printf "  %-20s %3d files\n" "TROUBLESHOOTING.md:" "$troubleshooting_count"
    
    # Show agents
    echo ""
    if command -v jq &>/dev/null; then
        agent_count=$(curl -s "${API_BASE}/api/agents" | jq '.count // 0')
        echo "Active Agents: $agent_count"
    fi
    
    # Show metrics
    echo ""
    if command -v jq &>/dev/null; then
        success_rate=$(curl -s "${API_BASE}/api/metrics/success-rate" | jq '.success_rate // 0')
        printf "Success Rate:  %.1f%%\n" "$success_rate"
    fi
}

# Add a new task
add_task() {
    local title="$*"
    
    if [ -z "$title" ]; then
        echo -e "${RED}Error: Task title required${NC}"
        echo "Usage: swarm-manager add-task <title>"
        exit 1
    fi
    
    # Generate task ID
    local task_id="task-$(date +%Y%m%d-%H%M%S)"
    local task_file="${TASKS_DIR}/backlog/manual/${task_id}.yaml"
    
    # Create task file
    cat > "$task_file" << EOF
id: ${task_id}
title: "${title}"
description: ""
type: general
target: ""
priority_estimates:
  impact: null
  urgency: null
  success_prob: null
  resource_cost: null
priority_score: null
dependencies: []
blockers: []
created_at: $(date -Iseconds)
created_by: human
analyzed_at: null
started_at: null
completed_at: null
attempts: []
notes: ""
EOF
    
    echo -e "${GREEN}✓ Task created: ${task_id}${NC}"
    echo "  Location: $task_file"
}

# List tasks
list_tasks() {
    local status="${1:-all}"
    
    echo -e "${BLUE}Tasks (${status})${NC}"
    echo "==============="
    
    if [ "$status" = "all" ]; then
        folders="active staged backlog/manual backlog/generated completed failed"
    else
        case "$status" in
            active|staged|completed|failed)
                folders="$status"
                ;;
            backlog)
                folders="backlog/manual backlog/generated"
                ;;
            *)
                echo -e "${RED}Invalid status: $status${NC}"
                exit 1
                ;;
        esac
    fi
    
    for folder in $folders; do
        echo ""
        echo -e "${YELLOW}$(echo "$folder" | sed 's|/| |g' | tr '[:lower:]' '[:upper:]')${NC}"
        echo "---"
        
        for task_file in "${TASKS_DIR}/${folder}"/*.yaml "${TASKS_DIR}/${folder}"/*.yml; do
            if [ -f "$task_file" ]; then
                # Extract key fields
                id=$(grep "^id:" "$task_file" | cut -d' ' -f2)
                title=$(grep "^title:" "$task_file" | cut -d'"' -f2)
                type=$(grep "^type:" "$task_file" | cut -d' ' -f2)
                priority=$(grep "^priority_score:" "$task_file" | cut -d' ' -f2)
                
                if [ "$priority" != "null" ] && [ -n "$priority" ]; then
                    printf "  %-20s %-40s %-15s [P:%.0f]\n" "$id" "$title" "$type" "$priority"
                else
                    printf "  %-20s %-40s %-15s\n" "$id" "$title" "$type"
                fi
            fi
        done
    done
}

# Execute a task
execute_task() {
    local task_id="${1:-}"
    
    if [ -z "$task_id" ]; then
        echo -e "${RED}Error: Task ID required${NC}"
        echo "Usage: swarm-manager execute <task-id>"
        exit 1
    fi
    
    echo -e "${BLUE}Executing task: ${task_id}${NC}"
    
    response=$(curl -s -X POST "${API_BASE}/api/tasks/${task_id}/execute")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to execute task${NC}"
        echo "$response"
    else
        echo -e "${GREEN}✓ Task execution started${NC}"
    fi
}

# Analyze a task
analyze_task() {
    local task_id="${1:-}"
    
    if [ -z "$task_id" ]; then
        echo -e "${RED}Error: Task ID required${NC}"
        echo "Usage: swarm-manager analyze <task-id>"
        exit 1
    fi
    
    echo -e "${BLUE}Analyzing task: ${task_id}${NC}"
    
    response=$(curl -s -X POST "${API_BASE}/api/tasks/${task_id}/analyze")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to analyze task${NC}"
        echo "$response"
    else
        echo -e "${GREEN}✓ Task analysis started${NC}"
    fi
}

# List agents
list_agents() {
    echo -e "${BLUE}Active Agents${NC}"
    echo "============="
    
    if command -v jq &>/dev/null; then
        curl -s "${API_BASE}/api/agents" | jq -r '.agents[] | 
            "\(.name)\t\(.status)\t\(.current_task_title // "idle")"' | \
        while IFS=$'\t' read -r name status task; do
            printf "  %-20s %-10s %s\n" "$name" "$status" "$task"
        done
    else
        curl -s "${API_BASE}/api/agents"
    fi
}

# Manage configuration
manage_config() {
    local action="${1:-get}"
    shift || true
    
    case "$action" in
        get|show)
            echo -e "${BLUE}Configuration${NC}"
            echo "============="
            if command -v jq &>/dev/null; then
                curl -s "${API_BASE}/api/config" | jq '.'
            else
                curl -s "${API_BASE}/api/config"
            fi
            ;;
        set)
            local key="${1:-}"
            local value="${2:-}"
            
            if [ -z "$key" ] || [ -z "$value" ]; then
                echo -e "${RED}Error: Key and value required${NC}"
                echo "Usage: swarm-manager config set <key> <value>"
                exit 1
            fi
            
            # Special handling for boolean values
            if [ "$value" = "true" ] || [ "$value" = "false" ]; then
                json="{\"$key\": $value}"
            else
                json="{\"$key\": \"$value\"}"
            fi
            
            response=$(curl -s -X PUT "${API_BASE}/api/config" \
                -H "Content-Type: application/json" \
                -d "$json")
            
            echo -e "${GREEN}✓ Configuration updated${NC}"
            ;;
        *)
            echo -e "${RED}Unknown config action: $action${NC}"
            echo "Usage: swarm-manager config [get|set] [key] [value]"
            exit 1
            ;;
    esac
}

# Show logs
show_logs() {
    local log_type="${1:-api}"
    
    case "$log_type" in
        api)
            tail -f "${SCRIPT_DIR}/logs/api.log" 2>/dev/null || \
                echo "No API logs found. Is the service running?"
            ;;
        ui)
            tail -f "${SCRIPT_DIR}/logs/ui.log" 2>/dev/null || \
                echo "No UI logs found. Is the service running?"
            ;;
        decisions)
            tail -f "${SCRIPT_DIR}/logs/decisions/orchestration.log" 2>/dev/null || \
                echo "No decision logs found."
            ;;
        *)
            echo -e "${RED}Unknown log type: $log_type${NC}"
            echo "Available: api, ui, decisions"
            exit 1
            ;;
    esac
}

# Show metrics
show_metrics() {
    echo -e "${BLUE}System Metrics${NC}"
    echo "=============="
    
    if command -v jq &>/dev/null; then
        metrics=$(curl -s "${API_BASE}/api/metrics")
        
        echo "$metrics" | jq -r '
            "Task Counts:",
            (.task_counts[] | "  \(.status): \(.count)"),
            "",
            "Performance:",
            "  Avg Duration: \(.avg_duration_seconds // 0) seconds",
            "  Success Rate: \(.success_rate // 0)%"
        '
    else
        curl -s "${API_BASE}/api/metrics"
    fi
}

# Scan for problems
scan_problems() {
    echo -e "${BLUE}Scanning for problems...${NC}"
    
    # Scan PROBLEMS.md files
    echo ""
    echo "Scanning PROBLEMS.md files:"
    echo "---------------------------"
    
    local problems_found=0
    local troubleshooting_found=0
    
    # Find and scan PROBLEMS.md files
    while IFS= read -r file; do
        if [ -f "$file" ]; then
            echo "  Scanning: $file"
            # Count active problems using embedded markers
            active_count=$(grep -c "<!-- EMBED:ACTIVEPROBLEM:START -->" "$file" 2>/dev/null || echo 0)
            if [ "$active_count" -gt 0 ]; then
                echo -e "    ${YELLOW}Found $active_count active problems${NC}"
                problems_found=$((problems_found + active_count))
            fi
        fi
    done < <(find /home/matthalloran8/Vrooli -name "PROBLEMS.md" 2>/dev/null)
    
    # Find and scan TROUBLESHOOTING.md files
    echo ""
    echo "Scanning TROUBLESHOOTING.md files:"
    echo "-----------------------------------"
    
    while IFS= read -r file; do
        if [ -f "$file" ]; then
            echo "  Scanning: $file"
            # Look for common troubleshooting patterns
            pattern_count=$(grep -E "(Known issue|Workaround|TODO|FIXME)" "$file" 2>/dev/null | wc -l)
            if [ "$pattern_count" -gt 0 ]; then
                echo -e "    ${YELLOW}Found $pattern_count potential issues${NC}"
                troubleshooting_found=$((troubleshooting_found + pattern_count))
            fi
        fi
    done < <(find /home/matthalloran8/Vrooli -name "TROUBLESHOOTING.md" 2>/dev/null)
    
    # Summary
    echo ""
    echo -e "${GREEN}Scan Complete${NC}"
    echo "-------------"
    echo "Active problems in PROBLEMS.md:         $problems_found"
    echo "Potential issues in TROUBLESHOOTING.md: $troubleshooting_found"
    
    # Trigger API scan if problems found
    if [ "$problems_found" -gt 0 ] || [ "$troubleshooting_found" -gt 0 ]; then
        echo ""
        echo "Triggering task generation for discovered problems..."
        response=$(curl -s -X POST "${API_BASE}/api/problems/scan" \
            -H "Content-Type: application/json" \
            -d '{"force": true}')
        
        if echo "$response" | grep -q "error"; then
            echo -e "${RED}Failed to trigger task generation${NC}"
        else
            echo -e "${GREEN}✓ Task generation initiated${NC}"
        fi
    fi
}

# Manage problems
manage_problems() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list|ls)
            list_problems "$@"
            ;;
        show)
            show_problem "$@"
            ;;
        resolve)
            resolve_problem "$@"
            ;;
        *)
            echo -e "${RED}Unknown problems action: $action${NC}"
            echo "Usage: swarm-manager problems [list|show|resolve]"
            exit 1
            ;;
    esac
}

# List discovered problems
list_problems() {
    local severity="${1:-all}"
    
    echo -e "${BLUE}Discovered Problems${NC}"
    echo "==================="
    
    # Query API for problems
    if command -v jq &>/dev/null; then
        problems=$(curl -s "${API_BASE}/api/problems?severity=${severity}")
        
        echo "$problems" | jq -r '
            .problems[] | 
            "\(.id)\t\(.severity)\t\(.status)\t\(.title)"' | \
        while IFS=$'\t' read -r id severity status title; do
            # Color code by severity
            case "$severity" in
                critical)
                    severity_color="${RED}${severity}${NC}"
                    ;;
                high)
                    severity_color="${YELLOW}${severity}${NC}"
                    ;;
                *)
                    severity_color="$severity"
                    ;;
            esac
            printf "  %-15s %-20s %-10s %s\n" "$id" "$severity_color" "$status" "$title"
        done
    else
        echo "Problems list requires jq to be installed"
    fi
}

# Show problem details
show_problem() {
    local problem_id="${1:-}"
    
    if [ -z "$problem_id" ]; then
        echo -e "${RED}Error: Problem ID required${NC}"
        echo "Usage: swarm-manager problems show <problem-id>"
        exit 1
    fi
    
    echo -e "${BLUE}Problem Details: ${problem_id}${NC}"
    echo "================"
    
    if command -v jq &>/dev/null; then
        curl -s "${API_BASE}/api/problems/${problem_id}" | jq '.'
    else
        curl -s "${API_BASE}/api/problems/${problem_id}"
    fi
}

# Resolve a problem
resolve_problem() {
    local problem_id="${1:-}"
    local resolution="${2:-}"
    
    if [ -z "$problem_id" ] || [ -z "$resolution" ]; then
        echo -e "${RED}Error: Problem ID and resolution required${NC}"
        echo "Usage: swarm-manager problems resolve <problem-id> <resolution>"
        exit 1
    fi
    
    echo -e "${BLUE}Resolving problem: ${problem_id}${NC}"
    
    response=$(curl -s -X PUT "${API_BASE}/api/problems/${problem_id}/resolve" \
        -H "Content-Type: application/json" \
        -d "{\"resolution\": \"$resolution\"}")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to resolve problem${NC}"
        echo "$response"
    else
        echo -e "${GREEN}✓ Problem marked as resolved${NC}"
    fi
}

# Scan for problems implementation (fixed)
scan_for_problems_impl() {
    local scan_path="${1:-/home/matthalloran8/Vrooli}"
    local output_file="${SCRIPT_DIR}/logs/problem-scan-$(date +%Y%m%d-%H%M%S).log"
    
    echo "Scanning path: $scan_path"
    echo "Results will be logged to: $output_file"
    
    # Scan for PROBLEMS.md files
    echo -e "\n${YELLOW}Scanning for PROBLEMS.md files...${NC}"
    local problems_files=$(find "$scan_path" -name "PROBLEMS.md" -type f 2>/dev/null | head -20)
    
    if [ -z "$problems_files" ]; then
        echo "  No PROBLEMS.md files found"
        echo "  Recommendation: Create PROBLEMS.md files in resources/scenarios"
    else
        echo "$problems_files" | while read -r file; do
            echo "  Found: $file"
            
            # Extract active problems using embedding markers
            if grep -q "<!-- EMBED:ACTIVEPROBLEM:START -->" "$file" 2>/dev/null; then
                local active_count=$(grep -c "### " "$file" 2>/dev/null || echo 0)
                echo "    Active problems: $active_count"
            fi
        done
    fi
    
    # Use Claude to analyze problems and create tasks
    local response=$(curl -s -X POST "${API_BASE}/api/problems/scan" \
        -H "Content-Type: application/json" \
        -d "{\"scan_path\": \"$scan_path\"}")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${YELLOW}Warning: API scan failed, using file-based scan${NC}"
        echo "$response"
    else
        echo -e "${GREEN}✓ Problem scan completed${NC}"
        if command -v jq &>/dev/null; then
            local problems_found=$(echo "$response" | jq -r '.problems_found // 0')
            local tasks_created=$(echo "$response" | jq -r '.tasks_created // 0')
            echo "  Problems found: $problems_found"
            echo "  Tasks created: $tasks_created"
        fi
    fi
}

# List discovered problems
list_problems() {
    local filter="${1:-all}"
    
    echo -e "${BLUE}System Problems (${filter})${NC}"
    echo "================"
    
    local response=$(curl -s "${API_BASE}/api/problems?filter=${filter}")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to fetch problems${NC}"
        echo "$response"
        return 1
    fi
    
    if command -v jq &>/dev/null; then
        echo "$response" | jq -r '.problems[] | 
            "\(.severity | ascii_upcase) | \(.title) | \(.frequency) | \(.status)"' | \
        while IFS='|' read -r severity title frequency status; do
            # Color code by severity
            case "$severity" in
                *CRITICAL*)
                    color="$RED"
                    ;;
                *HIGH*)
                    color="$YELLOW" 
                    ;;
                *)
                    color="$NC"
                    ;;
            esac
            printf "${color}%-10s${NC} %-50s %-12s %s\n" \
                "$severity" "$title" "$frequency" "$status"
        done
    else
        echo "$response"
    fi
}

# Mark a problem as resolved
resolve_problem() {
    local problem_id="${1:-}"
    local resolution="${2:-manually resolved}"
    
    if [ -z "$problem_id" ]; then
        echo -e "${RED}Error: Problem ID required${NC}"
        echo "Usage: swarm-manager resolve-problem <problem-id> [resolution]"
        exit 1
    fi
    
    echo -e "${BLUE}Resolving problem: ${problem_id}${NC}"
    
    local response=$(curl -s -X PUT "${API_BASE}/api/problems/${problem_id}/resolve" \
        -H "Content-Type: application/json" \
        -d "{\"resolution\": \"$resolution\"}")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to resolve problem${NC}"
        echo "$response"
    else
        echo -e "${GREEN}✓ Problem marked as resolved${NC}"
    fi
}

# Scan for problems
scan_for_problems() {
    local scan_path="${1:-}"
    
    echo -e "${BLUE}Scanning for problems...${NC}"
    
    # If no path provided, use configured defaults
    if [ -z "$scan_path" ]; then
        # Read from config
        scan_path="/home/matthalloran8/Vrooli"
    fi
    
    echo "Scanning path: $scan_path"
    
    # Call API to scan
    response=$(curl -s -X POST "${API_BASE}/api/problems/scan" \
        -H "Content-Type: application/json" \
        -d "{\"scan_path\": \"$scan_path\", \"force\": false}")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to scan for problems${NC}"
        echo "$response"
        return 1
    fi
    
    # Parse response
    if command -v jq &>/dev/null; then
        problems_found=$(echo "$response" | jq -r '.problems_found // 0')
        tasks_created=$(echo "$response" | jq -r '.tasks_created // 0')
        new_problems=$(echo "$response" | jq -r '.new_problems[]? // empty' | wc -l)
        
        echo -e "${GREEN}✓ Scan complete${NC}"
        echo "  Problems found: $problems_found"
        echo "  New problems: $new_problems"
        echo "  Tasks created: $tasks_created"
        
        if [ "$problems_found" -gt 0 ]; then
            echo ""
            echo "Run 'swarm-manager problems list' to view problems"
        fi
    else
        echo "$response"
    fi
}

# Manage problems
manage_problems() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list|ls)
            list_problems "$@"
            ;;
        show|get)
            show_problem "$@"
            ;;
        resolve)
            resolve_problem "$@"
            ;;
        *)
            echo -e "${RED}Unknown problems action: $action${NC}"
            echo "Usage: swarm-manager problems [list|show|resolve]"
            exit 1
            ;;
    esac
}

# List problems
list_problems() {
    local filter="${1:-all}"
    
    echo -e "${BLUE}Problems (${filter})${NC}"
    echo "================"
    
    # Query API for problems
    response=$(curl -s "${API_BASE}/api/problems")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to fetch problems${NC}"
        echo "$response"
        return 1
    fi
    
    if command -v jq &>/dev/null; then
        problems=$(echo "$response" | jq -r '.problems[]')
        
        # Filter based on request
        case "$filter" in
            critical)
                problems=$(echo "$problems" | jq -r 'select(.severity == "critical")')
                ;;
            high)
                problems=$(echo "$problems" | jq -r 'select(.severity == "high")')
                ;;
            active)
                problems=$(echo "$problems" | jq -r 'select(.status == "active")')
                ;;
            resolved)
                problems=$(echo "$problems" | jq -r 'select(.status == "resolved")')
                ;;
        esac
        
        # Display problems
        echo "$problems" | jq -r '
            "\(.id)\t\(.severity)\t\(.status)\t\(.title)"
        ' | while IFS=$'\t' read -r id severity status title; do
            # Color code by severity
            case "$severity" in
                critical)
                    severity_color="${RED}[CRITICAL]${NC}"
                    ;;
                high)
                    severity_color="${YELLOW}[HIGH]${NC}"
                    ;;
                medium)
                    severity_color="${BLUE}[MEDIUM]${NC}"
                    ;;
                low)
                    severity_color="${GREEN}[LOW]${NC}"
                    ;;
                *)
                    severity_color="[$severity]"
                    ;;
            esac
            
            printf "  %-20s %b %-10s %s\n" "$id" "$severity_color" "$status" "$title"
        done
    else
        echo "$response"
    fi
}

# Show specific problem
show_problem() {
    local problem_id="${1:-}"
    
    if [ -z "$problem_id" ]; then
        echo -e "${RED}Error: Problem ID required${NC}"
        echo "Usage: swarm-manager problems show <problem-id>"
        exit 1
    fi
    
    response=$(curl -s "${API_BASE}/api/problems/${problem_id}")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to fetch problem${NC}"
        echo "$response"
        return 1
    fi
    
    if command -v jq &>/dev/null; then
        echo "$response" | jq '.'
    else
        echo "$response"
    fi
}

# Resolve a problem
resolve_problem() {
    local problem_id="${1:-}"
    local resolution="${2:-Resolved}"
    
    if [ -z "$problem_id" ]; then
        echo -e "${RED}Error: Problem ID required${NC}"
        echo "Usage: swarm-manager problems resolve <problem-id> [resolution-note]"
        exit 1
    fi
    
    echo -e "${BLUE}Resolving problem: ${problem_id}${NC}"
    
    response=$(curl -s -X PUT "${API_BASE}/api/problems/${problem_id}/resolve" \
        -H "Content-Type: application/json" \
        -d "{\"resolution\": \"$resolution\"}")
    
    if echo "$response" | grep -q "error"; then
        echo -e "${RED}Failed to resolve problem${NC}"
        echo "$response"
        return 1
    fi
    
    echo -e "${GREEN}✓ Problem resolved${NC}"
}

# Show help
show_help() {
    cat << EOF
${BLUE}Swarm Manager - Autonomous Task Orchestrator${NC}

Usage: swarm-manager <command> [options]

Commands:
  start               Start the swarm manager service
  stop                Stop the swarm manager service
  status              Show service and task status
  
  add-task <title>    Add a new task to the backlog
  list-tasks [status] List tasks (all|active|staged|backlog|completed|failed)
  execute <task-id>   Execute a specific task
  analyze <task-id>   Analyze a task to calculate priority
  
  scan-problems [path] Scan for problems across resources and scenarios
  list-problems [filter] List discovered problems (all|active|critical|resolved)
  resolve-problem <id> Mark a problem as resolved
  
  agents              List active agents
  config [get|set]    Manage configuration
  logs [type]         Show logs (api|ui|decisions)
  metrics             Show system metrics
  
  scan-problems       Scan for problems in PROBLEMS.md and TROUBLESHOOTING.md
  problems list       List discovered problems
  problems show       Show specific problem details
  problems resolve    Mark a problem as resolved
  
  help                Show this help message

Examples:
  swarm-manager start
  swarm-manager add-task "Fix n8n webhook reliability"
  swarm-manager list-tasks active
  swarm-manager execute task-20240115-103000
  swarm-manager scan-problems
  swarm-manager list-problems critical
  swarm-manager resolve-problem prob-001 "Fixed in v2.1.0"
  swarm-manager config set yolo_mode true

For more information, see: ${SCRIPT_DIR}/README.md
EOF
}

# Create log directory
mkdir -p "${SCRIPT_DIR}/logs/"{decisions,executions}

# Run main function
main "$@"