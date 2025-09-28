#!/usr/bin/env bash
################################################################################
# Judge0 Execution Pool Manager
# 
# Maintains warm Docker containers for frequently used languages
# to reduce execution latency and improve throughput
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"

# Source configuration
source "${SCRIPT_DIR}/../config/defaults.sh" 2>/dev/null || true

# Pool configuration
readonly POOL_DIR="${VROOLI_CACHE_DIR:-/tmp/.vrooli/judge0/pools}"
readonly POOL_STATE_FILE="${POOL_DIR}/state.json"
readonly POOL_METRICS_FILE="${POOL_DIR}/metrics.json"

# Pool settings
readonly DEFAULT_POOL_SIZE=3         # Containers per language
readonly MAX_IDLE_TIME=300           # 5 minutes idle before container removal
readonly MAX_ON_DEMAND_CONTAINERS=5  # Maximum on-demand containers per language
readonly WARMUP_LANGUAGES=("python:3.11-slim" "node:18-slim")
readonly POOL_HEALTH_CHECK_INTERVAL=60  # Check pool health every minute
readonly POOL_METRICS_RETENTION=3600    # Keep metrics for 1 hour

# Container limits (optimized for performance)
readonly CONTAINER_CPU="0.5"         # Half CPU core
readonly CONTAINER_MEMORY="256m"     # 256MB RAM
readonly CONTAINER_TMPFS="64m"       # 64MB tmpfs for /tmp

# Ensure pool directory exists
mkdir -p "$POOL_DIR"

# Initialize pool state
init_pool_state() {
    if [[ ! -f "$POOL_STATE_FILE" ]]; then
        echo '{"pools": {}, "metrics": {"created": 0, "reused": 0, "evicted": 0}}' > "$POOL_STATE_FILE"
    fi
}

# Get container name for pool
get_pool_container_name() {
    local language="$1"
    local index="$2"
    # Normalize image name for container naming
    local normalized="${language//[:.]/-}"
    normalized="${normalized//\//-}"  # Handle slashes in image names
    echo "judge0-pool-${normalized}-${index}"
}

# Create warm container
create_warm_container() {
    local image="$1"
    local container_name="$2"
    
    # Remove existing container if it exists
    if docker ps -a --format "{{.Names}}" | grep -q "^${container_name}$"; then
        docker rm -f "$container_name" &>/dev/null
    fi
    
    # Create container with optimized settings (removed tmpfs to allow docker cp)
    timeout 10 docker run -d \
        --name "$container_name" \
        --cpus "$CONTAINER_CPU" \
        --memory "$CONTAINER_MEMORY" \
        --memory-swap "$CONTAINER_MEMORY" \
        --network none \
        --security-opt="no-new-privileges" \
        --label "app=judge0" \
        --label "type=pool" \
        --label "language=${image}" \
        --label "created=$(date +%s)" \
        --entrypoint "/bin/sh" \
        "$image" \
        -c "while true; do sleep 30; done" \
        >/dev/null 2>&1
    
    return $?
}

# Warm up pool for a language
warmup_language_pool() {
    local language="$1"
    local pool_size="${2:-$DEFAULT_POOL_SIZE}"
    
    echo "  Warming up $language pool (size: $pool_size)..."
    
    # Skip pulling if image exists (just proceed)
    # Images should already be available
    
    local created=0
    for i in $(seq 1 "$pool_size"); do
        local container_name=$(get_pool_container_name "$language" "$i")
        
        # Create with timeout to avoid hanging
        if create_warm_container "$language" "$container_name"; then
            ((created++))
            echo "    Created container $i/$pool_size"
        else
            echo "    Skipped container $i (already exists or creation failed)"
        fi
    done
    
    # Update pool state safely
    if [[ -f "$POOL_STATE_FILE" ]]; then
        local timestamp=$(date +%s)
        local current_state=$(cat "$POOL_STATE_FILE" 2>/dev/null || echo '{"pools": {}, "metrics": {"created": 0, "reused": 0, "evicted": 0}}')
        local new_state=$(echo "$current_state" | jq ".pools.\"$language\" = {\"size\": $pool_size, \"created\": $created, \"last_used\": $timestamp}")
        echo "$new_state" > "$POOL_STATE_FILE"
    fi
    
    echo "    Successfully created $created/$pool_size containers"
}

# Get available container from pool
get_pool_container() {
    local language="$1"
    
    # Find idle container
    for i in $(seq 1 "$DEFAULT_POOL_SIZE"); do
        local container_name=$(get_pool_container_name "$language" "$i")
        
        # Check if container exists and is running
        if docker ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
            # Check if not in use (simplified check)
            local processes=$(docker exec "$container_name" sh -c 'ps aux | wc -l' 2>/dev/null || echo "0")
            if [[ "$processes" -le 3 ]]; then
                echo "$container_name"
                
                # Update metrics
                local state=$(cat "$POOL_STATE_FILE")
                local reused=$(echo "$state" | jq -r '.metrics.reused')
                ((reused++))
                state=$(echo "$state" | jq ".metrics.reused = $reused")
                local update_timestamp=$(date +%s)
                state=$(echo "$state" | jq ".pools.\"$language\".last_used = $update_timestamp")
                echo "$state" > "$POOL_STATE_FILE"
                
                return 0
            fi
        fi
    done
    
    return 1
}

# Execute code using pool container
execute_with_pool() {
    local language="$1"
    local code="$2"
    local input="${3:-}"
    
    # Debug: echo "execute_with_pool: language='$language' code='$code' input='$input'" >&2
    
    # Try to get container from pool
    local container=$(get_pool_container "$language")
    
    local on_demand=false
    if [[ -z "$container" ]]; then
        # Check if we've reached the on-demand limit
        local lang_pattern="${language//[:.]/-}"
        local on_demand_count=0
        if docker ps --filter "name=judge0-pool-.*-demand-" --format "{{.Names}}" | grep -q "$lang_pattern"; then
            on_demand_count=$(docker ps --filter "name=judge0-pool-.*-demand-" --format "{{.Names}}" | grep -c "$lang_pattern")
        fi
        
        if [[ $on_demand_count -ge $MAX_ON_DEMAND_CONTAINERS ]]; then
            echo '{"status": "error", "message": "Too many on-demand containers, please wait and retry"}'
            return 1
        fi
        
        # No container available, create on-demand with unique ID
        container=$(get_pool_container_name "$language" "demand-$$-$(date +%s%N)")
        create_warm_container "$language" "$container"
        on_demand=true
        
        if [[ $? -ne 0 ]]; then
            echo '{"status": "error", "message": "Failed to create container"}'
            return 1
        fi
    fi
    
    # Execute code in container - optimized for speed
    local start=$(date +%s%N)
    
    local output
    local runtime_cmd
    case "$language" in
        python*)
            runtime_cmd="python3 -c"
            ;;
        node*)
            runtime_cmd="node -e"
            ;;
        ruby*)
            runtime_cmd="ruby -e"
            ;;
        golang*)
            # Go requires a file, can't use -e
            runtime_cmd="sh -c 'cat > /tmp/main.go && go run /tmp/main.go'"
            ;;
        openjdk*)
            # Java requires compilation
            runtime_cmd="sh -c 'cat > /tmp/Main.java && javac /tmp/Main.java && java -cp /tmp Main'"
            ;;
        *)
            runtime_cmd="sh -c"
            ;;
    esac
    
    # Execute directly without file copy for better performance
    if [[ "$language" == "golang"* ]] || [[ "$language" == "openjdk"* ]]; then
        # Go and Java need files
        if [[ -n "$input" ]]; then
            output=$(echo "$code" | docker exec -i "$container" sh -c "cat > /tmp/code.tmp && echo '$input' | $runtime_cmd < /tmp/code.tmp" 2>&1)
        else
            output=$(echo "$code" | docker exec -i "$container" bash -c "$runtime_cmd" 2>&1)
        fi
    else
        # Direct execution for interpreted languages
        if [[ -n "$input" ]]; then
            output=$(docker exec -i "$container" sh -c "echo '$input' | $runtime_cmd '$code'" 2>&1)
        else
            output=$(docker exec "$container" $runtime_cmd "$code" 2>&1)
        fi
    fi
    
    local end=$(date +%s%N)
    local exec_time=$(( (end - start) / 1000000 ))
    
    # Clean up any temporary files in container
    docker exec "$container" rm -f /tmp/code.tmp /tmp/main.go /tmp/Main.java /tmp/Main.class 2>/dev/null || true
    
    # Clean up on-demand container
    if [[ "$on_demand" == true ]]; then
        docker stop "$container" &>/dev/null
        docker rm "$container" &>/dev/null
    fi
    
    # Return result
    cat <<EOF
{
    "status": "accepted",
    "output": $(echo "$output" | jq -Rs .),
    "execution_time_ms": $exec_time,
    "container": "$container",
    "pooled": true
}
EOF
}

# Evict idle containers
evict_idle_containers() {
    local current_time=$(date +%s)
    local evicted=0
    
    # Read pool state
    local state=$(cat "$POOL_STATE_FILE" 2>/dev/null || echo '{"pools": {}}')
    
    # Check each pool
    for language in $(echo "$state" | jq -r '.pools | keys[]'); do
        local last_used=$(echo "$state" | jq -r ".pools.\"$language\".last_used // 0")
        local idle_time=$((current_time - last_used))
        
        if [[ $idle_time -gt $MAX_IDLE_TIME ]]; then
            echo "  Evicting idle $language pool (idle: ${idle_time}s)"
            
            # Remove containers
            for i in $(seq 1 "$DEFAULT_POOL_SIZE"); do
                local container_name=$(get_pool_container_name "$language" "$i")
                docker rm -f "$container_name" 2>/dev/null && ((evicted++)) || true
            done
            
            # Update state
            state=$(echo "$state" | jq "del(.pools.\"$language\")")
        fi
    done
    
    # Update metrics
    if [[ $evicted -gt 0 ]]; then
        local total_evicted=$(echo "$state" | jq -r '.metrics.evicted // 0')
        ((total_evicted += evicted))
        state=$(echo "$state" | jq ".metrics.evicted = $total_evicted")
        echo "$state" > "$POOL_STATE_FILE"
    fi
    
    echo "  Evicted $evicted containers"
}

# Warmup all configured languages
warmup_all_pools() {
    echo "🔥 Warming up execution pools..."
    
    init_pool_state
    
    for language in "${WARMUP_LANGUAGES[@]}"; do
        warmup_language_pool "$language"
    done
    
    echo "✅ Pool warmup complete"
}

# Health check and recover unhealthy pool containers
check_and_recover_pools() {
    local recovered=0
    local checked=0
    
    if [[ ! -f "$POOL_STATE_FILE" ]]; then
        return 0
    fi
    
    local state=$(cat "$POOL_STATE_FILE")
    
    # Check each pool's containers
    for language in $(echo "$state" | jq -r '.pools | keys[]' 2>/dev/null); do
        local size=$(echo "$state" | jq -r ".pools.\"$language\".size" 2>/dev/null || echo "0")
        
        for i in $(seq 1 "$size"); do
            local container_name=$(get_pool_container_name "$language" "$i")
            ((checked++))
            
            # Check if container exists and is running
            local container_status=$(docker inspect "$container_name" 2>/dev/null | jq -r '.[0].State.Status' 2>/dev/null || echo "not_found")
            
            if [[ "$container_status" != "running" ]]; then
                echo "  ⚠️  Container $container_name is $container_status, recovering..."
                
                # Remove dead container
                docker rm -f "$container_name" &>/dev/null || true
                
                # Recreate container
                if create_warm_container "$language" "$container_name"; then
                    echo "  ✅ Recovered $container_name"
                    ((recovered++))
                else
                    echo "  ❌ Failed to recover $container_name"
                fi
            fi
        done
    done
    
    if [[ $recovered -gt 0 ]]; then
        echo "  Recovered $recovered containers (checked: $checked)"
    fi
    
    return 0
}

# Show pool status
show_pool_status() {
    echo "🏊 Execution Pool Status"
    echo "========================"
    
    if [[ ! -f "$POOL_STATE_FILE" ]]; then
        echo "No pools initialized"
        return 0
    fi
    
    local state=$(cat "$POOL_STATE_FILE")
    
    # Show pools
    echo ""
    echo "Active Pools:"
    for language in $(echo "$state" | jq -r '.pools | keys[]'); do
        local size=$(echo "$state" | jq -r ".pools.\"$language\".size")
        local last_used=$(echo "$state" | jq -r ".pools.\"$language\".last_used")
        local idle=$(($(date +%s) - last_used))
        
        echo "  • $language: $size containers (idle: ${idle}s)"
        
        # Show container status
        for i in $(seq 1 "$size"); do
            local container_name=$(get_pool_container_name "$language" "$i")
            if docker ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
                echo "    - $container_name: running"
            else
                echo "    - $container_name: not found"
            fi
        done
    done
    
    # Show metrics
    echo ""
    echo "Metrics:"
    echo "  Created: $(echo "$state" | jq -r '.metrics.created // 0') containers"
    echo "  Reused: $(echo "$state" | jq -r '.metrics.reused // 0') executions"
    echo "  Evicted: $(echo "$state" | jq -r '.metrics.evicted // 0') containers"
    
    # Show resource usage
    echo ""
    echo "Resource Usage:"
    local total_mem=0
    local total_cpu=0
    
    for container in $(docker ps --filter "label=type=pool" --format "{{.Names}}"); do
        local stats=$(docker stats --no-stream --format "{{json .}}" "$container" 2>/dev/null || echo '{}')
        local mem=$(echo "$stats" | jq -r '.MemPerc // "0%"' | tr -d '%')
        local cpu=$(echo "$stats" | jq -r '.CPUPerc // "0%"' | tr -d '%')
        
        total_mem=$(echo "$total_mem + $mem" | bc)
        total_cpu=$(echo "$total_cpu + $cpu" | bc)
    done
    
    echo "  Total Memory: ${total_mem}%"
    echo "  Total CPU: ${total_cpu}%"
    
    # Show enhanced performance metrics
    if [[ -f "$POOL_METRICS_FILE" ]]; then
        local metrics=$(cat "$POOL_METRICS_FILE" 2>/dev/null || echo '{}')
        local avg_exec_time=$(echo "$metrics" | jq -r '.avg_execution_time_ms // 0')
        local hit_rate=$(echo "$metrics" | jq -r '.pool_hit_rate // 0')
        local total_executions=$(echo "$metrics" | jq -r '.total_executions // 0')
        
        echo ""
        echo "Performance Metrics:"
        echo "  Average execution: ${avg_exec_time}ms"
        echo "  Pool hit rate: ${hit_rate}%"
        echo "  Total executions: ${total_executions}"
    fi
}

# Enhanced performance monitoring
update_pool_metrics() {
    local exec_time="${1:-0}"
    local pooled="${2:-false}"
    local language="${3:-unknown}"
    
    # Load existing metrics
    local metrics='{}'
    if [[ -f "$POOL_METRICS_FILE" ]]; then
        metrics=$(cat "$POOL_METRICS_FILE" 2>/dev/null || echo '{}')
    fi
    
    # Update execution count
    local total=$(echo "$metrics" | jq -r '.total_executions // 0')
    ((total++))
    
    # Update average execution time
    local avg=$(echo "$metrics" | jq -r '.avg_execution_time_ms // 0')
    avg=$(( (avg * (total - 1) + exec_time) / total ))
    
    # Update pool hit rate
    local hits=$(echo "$metrics" | jq -r '.pool_hits // 0')
    if [[ "$pooled" == "true" ]]; then
        ((hits++))
    fi
    local hit_rate=0
    if [[ $total -gt 0 ]]; then
        hit_rate=$(( (hits * 100) / total ))
    fi
    
    # Save updated metrics
    echo "$metrics" | jq \
        --arg total "$total" \
        --arg avg "$avg" \
        --arg hits "$hits" \
        --arg hit_rate "$hit_rate" \
        '.total_executions = ($total | tonumber) |
         .avg_execution_time_ms = ($avg | tonumber) |
         .pool_hits = ($hits | tonumber) |
         .pool_hit_rate = ($hit_rate | tonumber) |
         .last_updated = now' > "$POOL_METRICS_FILE"
}

# Pool health check with enhanced monitoring and recovery
check_pool_health() {
    local healthy=0
    local unhealthy=0
    local recovered=0
    local containers=$(docker ps --filter "label=app=judge0" --filter "label=type=pool" --format "{{.Names}}")
    
    echo "🏥 Checking pool health..."
    
    for container in $containers; do
        # Enhanced health check with timeout
        if timeout 1 docker exec "$container" sh -c "echo 'health' && exit 0" &>/dev/null; then
            ((healthy++))
        else
            ((unhealthy++))
            echo "  ⚠️ Unhealthy container: $container"
            
            # Try to recover the container first
            if docker restart "$container" &>/dev/null; then
                # Wait briefly and recheck
                sleep 1
                if timeout 1 docker exec "$container" sh -c "echo 'health' && exit 0" &>/dev/null; then
                    ((recovered++))
                    ((unhealthy--))
                    ((healthy++))
                    echo "  ✅ Recovered container: $container"
                else
                    # Remove if recovery failed
                    docker rm -f "$container" &>/dev/null || true
                    echo "  ❌ Removed unhealthy container: $container"
                fi
            else
                # Remove if restart failed
                docker rm -f "$container" &>/dev/null || true
                echo "  ❌ Removed failed container: $container"
            fi
        fi
    done
    
    # Auto-replenish removed containers
    if [[ $unhealthy -gt 0 ]]; then
        echo "  🔄 Replenishing pool with $unhealthy new containers..."
        local state=$(cat "$POOL_STATE_FILE" 2>/dev/null || echo '{"pools": {}}')
        
        for language in $(echo "$state" | jq -r '.pools | keys[]'); do
            local expected_size=$(echo "$state" | jq -r ".pools.\"$language\".size // $DEFAULT_POOL_SIZE")
            local current_count=$(docker ps --filter "label=type=pool" --filter "label=language=$language" --format "{{.Names}}" | wc -l)
            local needed=$((expected_size - current_count))
            
            if [[ $needed -gt 0 ]]; then
                echo "    Creating $needed containers for $language..."
                for i in $(seq 1 $needed); do
                    local idx=$((current_count + i))
                    local container_name=$(get_pool_container_name "$language" "$idx")
                    if create_warm_container "$language" "$container_name"; then
                        echo "    ✅ Created: $container_name"
                    fi
                done
            fi
        done
    fi
    
    # Update health metrics
    local health_data="{\"healthy\": $healthy, \"unhealthy\": $unhealthy, \"recovered\": $recovered, \"timestamp\": $(date +%s)}"
    echo "$health_data" > "${POOL_DIR}/health.json"
    
    echo "$health_data"
}

# Clean up orphaned on-demand containers (older than 5 minutes)
cleanup_orphaned() {
    echo "🧹 Cleaning up orphaned on-demand containers..."
    
    local removed=0
    local current_time=$(date +%s)
    
    # Find and remove old on-demand containers
    for container in $(docker ps --filter "name=judge0-pool-.*-demand-" --format "{{.Names}}"); do
        # Get container creation time
        local created=$(docker inspect -f '{{.Created}}' "$container" 2>/dev/null | xargs -I {} date -d {} +%s 2>/dev/null || echo 0)
        local age=$((current_time - created))
        
        # Remove if older than 5 minutes (300 seconds)
        if [[ $age -gt 300 ]]; then
            docker stop "$container" &>/dev/null
            docker rm "$container" &>/dev/null && ((removed++)) || true
            echo "  Removed orphaned container: $container (age: ${age}s)"
        fi
    done
    
    echo "  Removed $removed orphaned containers"
}

# Clean up all pool containers
cleanup_pools() {
    echo "🧹 Cleaning up execution pools..."
    
    local removed=0
    for container in $(docker ps -a --filter "label=type=pool" --format "{{.Names}}"); do
        docker rm -f "$container" 2>/dev/null && ((removed++)) || true
    done
    
    # Also clean up any on-demand containers
    for container in $(docker ps -a --filter "name=judge0-pool-.*-demand-" --format "{{.Names}}"); do
        docker rm -f "$container" 2>/dev/null && ((removed++)) || true
    done
    
    rm -f "$POOL_STATE_FILE" "$POOL_METRICS_FILE"
    
    echo "  Removed $removed containers"
}

# Main execution
case "${1:-status}" in
    warmup)
        warmup_all_pools
        ;;
    execute)
        shift  # Remove 'execute' from arguments
        # Debug: echo "After shift: args=$# 1='$1' 2='$2' 3='${3:-}'" >&2
        execute_with_pool "${1:-python:3.11}" "${2:-print('test')}" "${3:-}"
        ;;
    status)
        show_pool_status
        ;;
    evict)
        evict_idle_containers
        ;;
    cleanup)
        cleanup_pools
        ;;
    cleanup-orphaned)
        cleanup_orphaned
        ;;
    health)
        check_pool_health
        ;;
    *)
        echo "Usage: $0 {warmup|execute <language> <code> [input]|status|evict|cleanup|cleanup-orphaned|health}"
        echo ""
        echo "Commands:"
        echo "  warmup              - Initialize warm container pools"
        echo "  execute             - Execute code using pool"
        echo "  status              - Show pool status"
        echo "  cleanup-orphaned    - Remove orphaned on-demand containers"
        echo "  evict               - Remove idle containers"
        echo "  cleanup             - Remove all pool containers"
        echo "  health              - Check pool health"
        exit 1
        ;;
esac