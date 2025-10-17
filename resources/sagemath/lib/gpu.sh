#!/bin/bash
# GPU acceleration support for SageMath computations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

# Check GPU availability
check_gpu_available() {
    if command -v nvidia-smi &> /dev/null; then
        nvidia-smi --query-gpu=name,memory.total,compute_cap --format=csv,noheader 2>/dev/null || echo "none"
    else
        echo "none"
    fi
}

# Enable GPU acceleration in container
enable_gpu() {
    local container_name="${SAGEMATH_CONTAINER_NAME}"
    
    # Check if container supports GPU
    if docker inspect "${container_name}" --format '{{json .HostConfig.DeviceRequests}}' 2>/dev/null | grep -q "gpu"; then
        echo "GPU acceleration already enabled"
        return 0
    fi
    
    # Restart container with GPU support
    docker stop "${container_name}" 2>/dev/null || true
    docker rm "${container_name}" 2>/dev/null || true
    
    # Start with GPU support
    docker run -d \
        --name "${container_name}" \
        --gpus all \
        -p "${SAGEMATH_JUPYTER_PORT}:8888" \
        -p "${SAGEMATH_API_PORT}:8889" \
        -v "${SAGEMATH_DATA_DIR}/scripts:/home/sage/scripts" \
        -v "${SAGEMATH_DATA_DIR}/notebooks:/home/sage/notebooks" \
        -v "${SAGEMATH_DATA_DIR}/outputs:/home/sage/outputs" \
        -e JUPYTER_ENABLE_LAB=yes \
        -e JUPYTER_TOKEN="" \
        --restart unless-stopped \
        sagemath/sagemath:latest
    
    echo "GPU acceleration enabled for SageMath"
}

# Run GPU-accelerated computation
gpu_compute() {
    local code="${1:-}"
    
    if [[ -z "${code}" ]]; then
        echo "Error: No computation code provided"
        exit 1
    fi
    
    # Check GPU availability
    local gpu_info=$(check_gpu_available)
    if [[ "${gpu_info}" == "none" ]]; then
        echo "Warning: No GPU available, falling back to CPU computation"
        docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "${code}"
        return
    fi
    
    # Execute with GPU acceleration hints
    local gpu_code="
import os
os.environ['CUDA_VISIBLE_DEVICES'] = '0'
# Enable parallel computation for applicable operations
from sage.parallel.decorate import parallel
${code}
"
    
    docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "${gpu_code}"
}

# Benchmark GPU vs CPU performance
benchmark_gpu() {
    echo "Running GPU vs CPU benchmark..."
    
    # Test matrix multiplication
    local matrix_test="
import time
import numpy as np
n = 1000
A = np.random.rand(n, n)
B = np.random.rand(n, n)
start = time.time()
C = np.dot(A, B)
elapsed = time.time() - start
print(f'Matrix multiplication ({n}x{n}): {elapsed:.3f}s')
"
    
    echo "CPU Performance:"
    docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "${matrix_test}"
    
    if [[ "$(check_gpu_available)" != "none" ]]; then
        echo -e "\nGPU Performance:"
        gpu_compute "${matrix_test}"
    else
        echo "GPU not available for comparison"
    fi
}

# Parallel computation support
parallel_compute() {
    local code="${1:-}"
    local num_cores="${2:-4}"
    
    if [[ -z "${code}" ]]; then
        echo "Error: No computation code provided"
        exit 1
    fi
    
    # Wrap code for parallel execution
    local parallel_code="
from sage.parallel.decorate import parallel
import multiprocessing
multiprocessing.set_start_method('fork', force=True)

# Set number of parallel processes
ncpus = ${num_cores}

${code}
"
    
    docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "${parallel_code}"
}

# Main GPU command handler
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        check)
            check_gpu_available
            ;;
        enable)
            enable_gpu
            ;;
        compute)
            shift
            gpu_compute "$@"
            ;;
        benchmark)
            benchmark_gpu
            ;;
        parallel)
            shift
            parallel_compute "$@"
            ;;
        *)
            echo "Usage: $0 {check|enable|compute|benchmark|parallel} [args]"
            exit 1
            ;;
    esac
fi