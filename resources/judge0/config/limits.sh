#!/usr/bin/env bash
################################################################################
# Judge0 Optimized Resource Limits Configuration
# 
# Performance-tuned limits based on system capabilities
################################################################################

# Execution time limits (seconds)
export JUDGE0_CPU_TIME_LIMIT="${JUDGE0_CPU_TIME_LIMIT:-5}"           # CPU time limit
export JUDGE0_WALL_TIME_LIMIT="${JUDGE0_WALL_TIME_LIMIT:-10}"       # Wall clock time limit
export JUDGE0_COMPILE_TIME_LIMIT="${JUDGE0_COMPILE_TIME_LIMIT:-10}" # Compilation time limit

# Memory limits (MB)
export JUDGE0_MEMORY_LIMIT="${JUDGE0_MEMORY_LIMIT:-256}"            # Execution memory limit
export JUDGE0_STACK_LIMIT="${JUDGE0_STACK_LIMIT:-64}"              # Stack size limit
export JUDGE0_COMPILE_MEMORY_LIMIT="${JUDGE0_COMPILE_MEMORY_LIMIT:-384}" # Compilation memory

# Process and file limits
export JUDGE0_MAX_PROCESSES="${JUDGE0_MAX_PROCESSES:-30}"           # Max processes per submission
export JUDGE0_MAX_FILE_SIZE="${JUDGE0_MAX_FILE_SIZE:-5}"           # Max file size (MB)
export JUDGE0_MAX_OPEN_FILES="${JUDGE0_MAX_OPEN_FILES:-256}"       # Max open file descriptors

# Worker configuration
export JUDGE0_WORKERS_MIN="${JUDGE0_WORKERS_MIN:-2}"                # Minimum workers
export JUDGE0_WORKERS_MAX="${JUDGE0_WORKERS_MAX:-10}"              # Maximum workers
export JUDGE0_WORKERS_SCALE_THRESHOLD="${JUDGE0_WORKERS_SCALE_THRESHOLD:-5}" # Queue size to trigger scaling

# Performance optimization
export JUDGE0_ENABLE_BATCHED_SUBMISSIONS="${JUDGE0_ENABLE_BATCHED_SUBMISSIONS:-true}"
export JUDGE0_MAX_BATCH_SIZE="${JUDGE0_MAX_BATCH_SIZE:-20}"
export JUDGE0_SUBMISSION_CACHE_DURATION="${JUDGE0_SUBMISSION_CACHE_DURATION:-300}" # Cache identical submissions for 5 minutes

# Security limits
export JUDGE0_ENABLE_NETWORK="${JUDGE0_ENABLE_NETWORK:-false}"      # Network access in sandbox
export JUDGE0_ENABLE_SYSCALL_FILTER="${JUDGE0_ENABLE_SYSCALL_FILTER:-true}" # Syscall filtering
export JUDGE0_MAX_SUBMISSION_SIZE="${JUDGE0_MAX_SUBMISSION_SIZE:-65536}" # Max source code size (bytes)

# Database connection pool
export JUDGE0_DB_POOL_SIZE="${JUDGE0_DB_POOL_SIZE:-10}"
export JUDGE0_DB_TIMEOUT="${JUDGE0_DB_TIMEOUT:-5000}"

# Redis configuration
export JUDGE0_REDIS_POOL_SIZE="${JUDGE0_REDIS_POOL_SIZE:-10}"
export JUDGE0_REDIS_TIMEOUT="${JUDGE0_REDIS_TIMEOUT:-5000}"

# Apply optimized limits based on system resources
judge0::limits::optimize() {
    local total_memory=$(free -m | awk '/^Mem:/{print $2}')
    local cpu_count=$(nproc)
    
    # Adjust workers based on CPU cores
    if [[ $cpu_count -ge 16 ]]; then
        export JUDGE0_WORKERS_MAX=10
        export JUDGE0_DB_POOL_SIZE=20
    elif [[ $cpu_count -ge 8 ]]; then
        export JUDGE0_WORKERS_MAX=6
        export JUDGE0_DB_POOL_SIZE=15
    else
        export JUDGE0_WORKERS_MAX=4
        export JUDGE0_DB_POOL_SIZE=10
    fi
    
    # Adjust memory limits based on available RAM
    if [[ $total_memory -ge 32768 ]]; then  # 32GB+
        export JUDGE0_MEMORY_LIMIT=512
        export JUDGE0_COMPILE_MEMORY_LIMIT=768
    elif [[ $total_memory -ge 16384 ]]; then  # 16GB+
        export JUDGE0_MEMORY_LIMIT=384
        export JUDGE0_COMPILE_MEMORY_LIMIT=512
    elif [[ $total_memory -ge 8192 ]]; then   # 8GB+
        export JUDGE0_MEMORY_LIMIT=256
        export JUDGE0_COMPILE_MEMORY_LIMIT=384
    else
        export JUDGE0_MEMORY_LIMIT=128
        export JUDGE0_COMPILE_MEMORY_LIMIT=256
    fi
    
    log "Optimized limits: Workers=$JUDGE0_WORKERS_MAX, Memory=${JUDGE0_MEMORY_LIMIT}MB"
}

# Validate limits are within safe ranges
judge0::limits::validate() {
    local errors=0
    
    # Check time limits
    if [[ $JUDGE0_CPU_TIME_LIMIT -gt 60 ]]; then
        log "⚠️  CPU time limit too high: ${JUDGE0_CPU_TIME_LIMIT}s (max: 60s)"
        errors=$((errors + 1))
    fi
    
    if [[ $JUDGE0_WALL_TIME_LIMIT -gt 120 ]]; then
        log "⚠️  Wall time limit too high: ${JUDGE0_WALL_TIME_LIMIT}s (max: 120s)"
        errors=$((errors + 1))
    fi
    
    # Check memory limits
    if [[ $JUDGE0_MEMORY_LIMIT -gt 2048 ]]; then
        log "⚠️  Memory limit too high: ${JUDGE0_MEMORY_LIMIT}MB (max: 2048MB)"
        errors=$((errors + 1))
    fi
    
    # Check worker limits
    if [[ $JUDGE0_WORKERS_MAX -gt 20 ]]; then
        log "⚠️  Too many workers: ${JUDGE0_WORKERS_MAX} (max: 20)"
        errors=$((errors + 1))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log "✅ All resource limits are within safe ranges"
        return 0
    else
        log "❌ $errors limit validation errors found"
        return 1
    fi
}

# Export limits to Docker environment
judge0::limits::export_docker_env() {
    cat <<EOF
CPU_TIME_LIMIT=$JUDGE0_CPU_TIME_LIMIT
WALL_TIME_LIMIT=$JUDGE0_WALL_TIME_LIMIT
MEMORY_LIMIT=$JUDGE0_MEMORY_LIMIT
STACK_LIMIT=$JUDGE0_STACK_LIMIT
MAX_PROCESSES_AND_OR_THREADS=$JUDGE0_MAX_PROCESSES
MAX_FILE_SIZE=$JUDGE0_MAX_FILE_SIZE
ENABLE_NETWORK=$JUDGE0_ENABLE_NETWORK
NUMBER_OF_WORKERS=$JUDGE0_WORKERS_MAX
EOF
}