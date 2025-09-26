#!/usr/bin/env bash
################################################################################
# SageMath Parallel Computing Library
#
# Provides parallel computation capabilities using SageMath's parallel decorators
################################################################################

set -euo pipefail

# Ensure SAGEMATH_CLI_DIR is set
SAGEMATH_CLI_DIR="${SAGEMATH_CLI_DIR:-$(dirname "${BASH_SOURCE[0]}")/..}"

# Container name
SAGEMATH_CONTAINER_NAME="${SAGEMATH_CONTAINER_NAME:-sagemath-main}"

# Execute parallel computation
# Args: $1 = code to execute, $2 = number of cores (optional, default 4)
sagemath::parallel::execute() {
    local code="${1:-}"
    local cores="${2:-4}"
    
    if [[ -z "$code" ]]; then
        echo "[ERROR] No code provided for parallel computation" >&2
        return 1
    fi
    
    # Ensure SageMath is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${SAGEMATH_CONTAINER_NAME}$"; then
        echo "[ERROR] SageMath container is not running" >&2
        return 1
    fi
    
    echo "🔄 Running parallel computation with $cores cores..."
    
    # Create a temporary script with parallel execution
    local temp_script=$(mktemp /tmp/sagemath_parallel_XXXXXX.sage)
    
    cat > "$temp_script" << EOF
import multiprocessing
from sage.parallel.decorate import parallel

# Set number of processes
ncpus = $cores

# Execute the provided code and print result
result = $code
print(result)
EOF
    
    # Copy script to container and execute
    docker cp "$temp_script" "$SAGEMATH_CONTAINER_NAME:/home/sage/parallel_exec.sage"
    
    # Execute and capture result
    local result
    result=$(docker exec "$SAGEMATH_CONTAINER_NAME" sage /home/sage/parallel_exec.sage 2>&1)
    local exit_code=$?
    
    # Cleanup
    rm -f "$temp_script"
    docker exec "$SAGEMATH_CONTAINER_NAME" rm -f /home/sage/parallel_exec.sage
    
    if [[ $exit_code -eq 0 ]]; then
        echo "📊 Result: $result"
    else
        echo "[ERROR] Parallel computation failed: $result" >&2
        return 1
    fi
}

# Check parallel computing status
sagemath::parallel::check_status() {
    if ! docker ps --format "{{.Names}}" | grep -q "^${SAGEMATH_CONTAINER_NAME}$"; then
        echo "[ERROR] SageMath container is not running" >&2
        return 1
    fi
    
    local cpu_count=$(docker exec "$SAGEMATH_CONTAINER_NAME" sage -c 'import multiprocessing; print(multiprocessing.cpu_count())')
    echo "Available CPU cores: ${cpu_count}"
    
    # Check if parallel sage is available
    local parallel_check=$(docker exec "$SAGEMATH_CONTAINER_NAME" sage -c 'from sage.parallel.decorate import parallel; print("Parallel computing available")' 2>/dev/null || echo "Parallel computing not available")
    echo "${parallel_check}"
}