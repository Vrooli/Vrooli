#!/usr/bin/env bash
################################################################################
# MinIO Performance Tuning Library
# 
# Configure MinIO performance parameters for optimal operation
################################################################################

set -euo pipefail

# Source dependencies
source "${MINIO_CLI_DIR}/lib/common.sh"
source "${MINIO_CLI_DIR}/lib/docker.sh"

################################################################################
# Performance Configuration
################################################################################

minio::performance::configure() {
    local profile="${1:-balanced}"
    
    log::info "Configuring MinIO performance profile: $profile"
    
    case "$profile" in
        "minimal")
            # Minimal resource usage for development
            MINIO_CACHE_SIZE="64MB"
            MINIO_CACHE_WATERMARK_LOW="70"
            MINIO_CACHE_WATERMARK_HIGH="80"
            MINIO_CACHE_DRIVES="/cache"
            MINIO_PERF_BROWSER="off"  # Disable console to save resources
            GOMAXPROCS="2"
            ;;
            
        "balanced")
            # Balanced for general use
            MINIO_CACHE_SIZE="256MB"
            MINIO_CACHE_WATERMARK_LOW="80"
            MINIO_CACHE_WATERMARK_HIGH="90"
            MINIO_CACHE_DRIVES="/cache"
            MINIO_PERF_BROWSER="on"
            GOMAXPROCS="4"
            ;;
            
        "performance")
            # High performance for production
            MINIO_CACHE_SIZE="1GB"
            MINIO_CACHE_WATERMARK_LOW="85"
            MINIO_CACHE_WATERMARK_HIGH="95"
            MINIO_CACHE_DRIVES="/cache"
            MINIO_PERF_BROWSER="on"
            GOMAXPROCS="8"
            MINIO_API_REQUESTS_MAX="16000"
            MINIO_API_REQUESTS_DEADLINE="30s"
            ;;
            
        "custom")
            # Use environment variables as-is
            log::info "Using custom performance configuration from environment"
            ;;
            
        *)
            log::error "Unknown performance profile: $profile"
            log::info "Available profiles: minimal, balanced, performance, custom"
            return 1
            ;;
    esac
    
    # Save configuration
    local config_file="${HOME}/.minio/config/performance.conf"
    mkdir -p "$(dirname "$config_file")"
    
    cat > "$config_file" <<EOF
# MinIO Performance Configuration
# Profile: $profile
# Generated: $(date)

MINIO_CACHE_SIZE="${MINIO_CACHE_SIZE:-256MB}"
MINIO_CACHE_WATERMARK_LOW="${MINIO_CACHE_WATERMARK_LOW:-80}"
MINIO_CACHE_WATERMARK_HIGH="${MINIO_CACHE_WATERMARK_HIGH:-90}"
MINIO_CACHE_DRIVES="${MINIO_CACHE_DRIVES:-/cache}"
MINIO_PERF_BROWSER="${MINIO_PERF_BROWSER:-on}"
GOMAXPROCS="${GOMAXPROCS:-4}"
MINIO_API_REQUESTS_MAX="${MINIO_API_REQUESTS_MAX:-8000}"
MINIO_API_REQUESTS_DEADLINE="${MINIO_API_REQUESTS_DEADLINE:-10s}"
EOF
    
    log::success "Performance profile '$profile' configured"
    return 0
}

################################################################################
# Performance Monitoring
################################################################################

minio::performance::monitor() {
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    
    log::info "MinIO Performance Metrics"
    log::info "========================="
    
    # Check if container is running
    if ! minio::is_running; then
        log::error "MinIO is not running"
        return 1
    fi
    
    # Get container stats
    local stats
    stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$container_name" 2>/dev/null)
    
    if [[ -n "$stats" ]]; then
        echo "$stats"
    fi
    
    # Get MinIO server info via mc admin
    log::info ""
    log::info "Server Information:"
    if docker exec "$container_name" mc admin info local 2>/dev/null; then
        log::success "Server info retrieved"
    else
        log::warning "Could not retrieve detailed server info (admin privileges required)"
    fi
    
    # Get connection count (approximate via netstat)
    local connections
    connections=$(docker exec "$container_name" sh -c 'netstat -an 2>/dev/null | grep -c ":9000.*ESTABLISHED"' 2>/dev/null || echo "0")
    log::info "Active connections: $connections"
    
    # Get cache info if available
    if docker exec "$container_name" mc admin config get local cache 2>/dev/null; then
        log::info "Cache configuration retrieved"
    fi
    
    return 0
}

################################################################################
# Performance Tuning Commands
################################################################################

minio::performance::apply_profile() {
    local profile="${1:-balanced}"
    
    # Check if MinIO is running
    if ! minio::is_running; then
        log::error "MinIO must be running to apply performance profile"
        return 1
    fi
    
    # Configure the profile
    minio::performance::configure "$profile"
    
    # Load the configuration
    local config_file="${HOME}/.minio/config/performance.conf"
    if [[ -f "$config_file" ]]; then
        # shellcheck disable=SC1090
        source "$config_file"
    fi
    
    # Apply settings that can be changed at runtime
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    
    # Update container environment (requires restart)
    log::warning "Performance settings will be fully applied on next restart"
    log::info "Run 'vrooli resource minio manage restart' to apply all settings"
    
    # Save profile preference
    echo "$profile" > "${HOME}/.minio/config/performance.profile"
    
    return 0
}

minio::performance::get_profile() {
    local profile_file="${HOME}/.minio/config/performance.profile"
    
    if [[ -f "$profile_file" ]]; then
        cat "$profile_file"
    else
        echo "balanced"  # Default
    fi
}

minio::performance::benchmark() {
    local size="${1:-100}"  # MB
    local iterations="${2:-5}"
    
    log::info "Running MinIO performance benchmark..."
    log::info "File size: ${size}MB, Iterations: $iterations"
    
    if ! minio::is_healthy; then
        log::error "MinIO is not healthy"
        return 1
    fi
    
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    local test_bucket="benchmark-$(date +%s)"
    local test_file="/tmp/benchmark-${size}mb.dat"
    
    # Create test file
    log::info "Creating ${size}MB test file..."
    dd if=/dev/zero of="$test_file" bs=1M count="$size" 2>/dev/null
    
    # Create test bucket
    docker exec "$container_name" mc mb "local/$test_bucket" &>/dev/null
    
    # Run upload benchmark
    log::info "Testing upload performance..."
    local upload_times=()
    for i in $(seq 1 "$iterations"); do
        local start_time=$(date +%s%N)
        docker cp "$test_file" "$container_name:/tmp/test.dat"
        docker exec "$container_name" mc cp "/tmp/test.dat" "local/$test_bucket/test-$i.dat" &>/dev/null
        local end_time=$(date +%s%N)
        local duration=$((($end_time - $start_time) / 1000000))  # Convert to ms
        upload_times+=("$duration")
        echo -n "."
    done
    echo ""
    
    # Run download benchmark
    log::info "Testing download performance..."
    local download_times=()
    for i in $(seq 1 "$iterations"); do
        local start_time=$(date +%s%N)
        docker exec "$container_name" mc cp "local/$test_bucket/test-$i.dat" "/tmp/download-$i.dat" &>/dev/null
        local end_time=$(date +%s%N)
        local duration=$((($end_time - $start_time) / 1000000))  # Convert to ms
        download_times+=("$duration")
        echo -n "."
    done
    echo ""
    
    # Calculate averages
    local upload_sum=0
    for time in "${upload_times[@]}"; do
        upload_sum=$((upload_sum + time))
    done
    local upload_avg=$((upload_sum / iterations))
    
    local download_sum=0
    for time in "${download_times[@]}"; do
        download_sum=$((download_sum + time))
    done
    local download_avg=$((download_sum / iterations))
    
    # Calculate throughput
    local upload_throughput=$(echo "scale=2; ($size * 1024) / ($upload_avg / 1000)" | bc 2>/dev/null || echo "N/A")
    local download_throughput=$(echo "scale=2; ($size * 1024) / ($download_avg / 1000)" | bc 2>/dev/null || echo "N/A")
    
    # Clean up
    docker exec "$container_name" mc rb --force "local/$test_bucket" &>/dev/null
    rm -f "$test_file"
    
    # Display results
    log::info ""
    log::info "Benchmark Results:"
    log::info "=================="
    log::info "Upload avg: ${upload_avg}ms (${upload_throughput} KB/s)"
    log::info "Download avg: ${download_avg}ms (${download_throughput} KB/s)"
    
    # Performance rating
    if [[ "$upload_avg" -lt 1000 ]] && [[ "$download_avg" -lt 1000 ]]; then
        log::success "Performance: EXCELLENT"
    elif [[ "$upload_avg" -lt 2000 ]] && [[ "$download_avg" -lt 2000 ]]; then
        log::success "Performance: GOOD"
    else
        log::warning "Performance: NEEDS OPTIMIZATION"
    fi
    
    return 0
}

################################################################################
# Export Functions
################################################################################

export -f minio::performance::configure
export -f minio::performance::monitor
export -f minio::performance::apply_profile
export -f minio::performance::get_profile
export -f minio::performance::benchmark