#!/bin/bash
# Blender Physics Benchmark Functions
# Provides benchmarking and performance validation for physics simulations

# Source required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BLENDER_LIB_DIR="${APP_ROOT}/resources/blender/lib"

# Only source if not already loaded
if ! type blender::init &>/dev/null; then
    source "${BLENDER_LIB_DIR}/core.sh"
fi

# Run physics benchmark suite
blender::physics::benchmark() {
    local benchmark_type="${1:-all}"
    local output_file="${2:-/tmp/blender_physics_benchmark.json}"
    
    echo "[INFO] Running physics benchmark: $benchmark_type"
    
    # Use the example benchmark script
    local benchmark_script="${APP_ROOT}/resources/blender/examples/physics_benchmark.py"
    
    if [[ ! -f "$benchmark_script" ]]; then
        echo "[ERROR] Benchmark script not found: $benchmark_script"
        return 1
    fi
    
    # Set environment variables for the benchmark
    export BENCHMARK_TYPE="$benchmark_type"
    export BENCHMARK_OUTPUT="$output_file"
    
    # Run benchmark
    local benchmark_output
    benchmark_output=$(timeout 120 blender::run_script "$benchmark_script" 2>&1)
    
    if echo "$benchmark_output" | grep -q "BENCHMARK_COMPLETE"; then
        # Parse and display results
        if [[ -f "$output_file" ]]; then
            echo "[SUCCESS] Benchmark completed"
            echo "[INFO] Results:"
            
            # Display key metrics
            if command -v jq &>/dev/null; then
                jq -r '
                    .benchmarks | to_entries[] | 
                    "[METRIC] \(.key): FPS=\(.value.fps // "N/A" | tostring | .[0:6])"
                ' "$output_file" 2>/dev/null || true
            else
                # Display results from output
                echo "$benchmark_output" | grep "\[RESULT\]" || true
            fi
            
            return 0
        else
            echo "[ERROR] Benchmark results not generated"
            return 1
        fi
    else
        echo "[ERROR] Benchmark execution failed"
        echo "[DEBUG] Output: $benchmark_output" | head -20
        return 1
    fi
}

# Compare benchmark results
blender::physics::compare_benchmarks() {
    local baseline="${1:-/tmp/blender_physics_baseline.json}"
    local current="${2:-/tmp/blender_physics_benchmark.json}"
    
    if [[ ! -f "$baseline" ]]; then
        echo "[ERROR] Baseline file not found: $baseline"
        return 1
    fi
    
    if [[ ! -f "$current" ]]; then
        echo "[ERROR] Current benchmark file not found: $current"
        return 1
    fi
    
    echo "[INFO] Comparing benchmark results..."
    
    # Use Python for JSON comparison
    python3 << 'PYEOF'
import json
import sys

try:
    with open("${baseline}") as f:
        baseline = json.load(f)
    
    with open("${current}") as f:
        current = json.load(f)

    def compare_metric(name, baseline_val, current_val):
        if baseline_val and current_val:
            diff = ((current_val - baseline_val) / baseline_val) * 100
            symbol = "↑" if diff > 0 else "↓" if diff < 0 else "="
            color = "32" if diff > 0 else "31" if diff < -5 else "33"
            print(f"[\033[0;{color}mMETRIC\033[0m] {name}: {current_val:.2f} ({symbol} {abs(diff):.1f}%)")

    # Compare each benchmark
    for bench_name in current.get("benchmarks", {}):
        if bench_name in baseline.get("benchmarks", {}):
            print(f"\n[INFO] {bench_name}:")
            base = baseline["benchmarks"][bench_name]
            curr = current["benchmarks"][bench_name]
            
            if "fps" in curr and "fps" in base:
                compare_metric("FPS", base["fps"], curr["fps"])
            if "simulation_time" in curr and "simulation_time" in base:
                compare_metric("Sim Time", base["simulation_time"], curr["simulation_time"])
except Exception as e:
    print(f"[ERROR] Comparison failed: {e}")
    sys.exit(1)
PYEOF
    
    return $?
}

# Save current benchmark as baseline
blender::physics::save_baseline() {
    local benchmark_file="${1:-/tmp/blender_physics_benchmark.json}"
    local baseline_file="${2:-/tmp/blender_physics_baseline.json}"
    
    if [[ ! -f "$benchmark_file" ]]; then
        echo "[ERROR] Benchmark file not found: $benchmark_file"
        return 1
    fi
    
    cp "$benchmark_file" "$baseline_file"
    echo "[SUCCESS] Baseline saved to: $baseline_file"
    return 0
}

# Run physics optimization test
blender::physics::test_optimization() {
    echo "[INFO] Testing physics optimization..."
    
    # Test GPU optimization
    local gpu_script="${APP_ROOT}/resources/blender/examples/physics_gpu_optimized.py"
    if [[ -f "$gpu_script" ]]; then
        echo "[INFO] Testing GPU optimization..."
        if timeout 60 blender::run_script "$gpu_script" 2>&1 | grep -q "GPU"; then
            echo "[SUCCESS] GPU optimization available"
        else
            echo "[WARNING] GPU optimization not available"
        fi
    fi
    
    # Test cache streaming
    local cache_script="${APP_ROOT}/resources/blender/examples/physics_cache_streaming.py"
    if [[ -f "$cache_script" ]]; then
        echo "[INFO] Testing cache streaming..."
        if timeout 60 blender::run_script "$cache_script" 2>&1 | grep -q "Cache"; then
            echo "[SUCCESS] Cache streaming available"
        else
            echo "[WARNING] Cache streaming not available"
        fi
    fi
    
    return 0
}

# Export functions
export -f blender::physics::benchmark
export -f blender::physics::compare_benchmarks
export -f blender::physics::save_baseline
export -f blender::physics::test_optimization