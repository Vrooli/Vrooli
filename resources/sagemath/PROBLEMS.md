# SageMath Resource - Known Problems and Solutions

## Overview
This document tracks known issues, limitations, and their solutions for the SageMath resource.

## Solved Problems

### 5. Parallel Compute --cores Parameter Issue (SOLVED)
**Problem**: The `parallel compute --cores N` command failed with NameError about 'cores' not being defined.

**Symptoms**:
- Command `resource-sagemath parallel compute --cores 4 "code"` failed
- Error: `NameError: name 'cores' is not defined`
- Parallel compute worked without --cores parameter

**Root Cause**: 
- CLI was using positional parameters instead of parsing --cores flag
- Missing parallel.sh library (was calling non-existent gpu.sh)
- Improper argument parsing in sagemath::parallel::compute function

**Solution**:
```bash
# Created lib/parallel.sh with proper implementation
# Updated CLI to parse --cores flag properly
while [[ $# -gt 0 ]]; do
    case $1 in
        --cores)
            cores="$2"
            shift 2
            ;;
        *)
            code="$1"
            shift
            ;;
    esac
done
```

**Status**: ✅ FIXED (2025-09-26)
**Additional Fix**: Fixed undefined `error` function and missing result output in parallel.sh

### 1. Performance Benchmarks Hanging (SOLVED)
**Problem**: Performance benchmarks would hang on complex matrix operations and missing library imports.

**Symptoms**:
- Test suite hangs indefinitely during performance tests
- No output from benchmark operations
- Container CPU stuck at 100%

**Root Cause**: 
- Matrix operations with 100x100 matrices were computationally expensive
- Missing source of `common.sh` in test.sh

**Solution**:
```bash
# Added timeouts to all benchmark operations
timeout 5 docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "$code"

# Reduced matrix sizes from 100x100 to 20x20
random_matrix(RR, 20, 20)  # Previously 100x100

# Added library source
source "${SAGEMATH_LIB_DIR}/common.sh"
```

**Status**: ✅ FIXED (2025-09-12)

### 2. Visualization Files Not Saved (SOLVED)
**Problem**: Plot commands returned text descriptions but didn't save actual plot files.

**Symptoms**:
- Plot commands execute successfully
- Only text output returned
- No PNG files generated

**Root Cause**: SageMath plots need explicit save() calls to generate files.

**Solution**:
Enhanced `content::execute` to detect plot operations and automatically save:
```python
if 'plot' in code or 'graph' in code:
    plot_file = f"/home/sage/outputs/plot_{timestamp}.png"
    result.save(plot_file)
```

**Status**: ✅ FIXED (2025-09-12)

### 3. Temporary File Accumulation (SOLVED)
**Problem**: Temporary calculation files accumulated in /tmp without cleanup.

**Symptoms**:
- 62+ temp files in /tmp/sage_calc_*
- Disk space gradually consumed
- No automatic cleanup

**Solution**:
Added automatic cleanup for files older than 1 day:
```bash
find /tmp -name "sage_calc_*" -mtime +1 -delete 2>/dev/null || true
```

**Status**: ✅ FIXED (2025-09-14)

### 4. GPU Command Not Working from CLI (SOLVED)
**Problem**: The main GPU command shows help instead of executing check.

**Symptoms**:
- `resource-sagemath gpu` shows help text
- Must use `resource-sagemath gpu check` explicitly
- Inconsistent with other command patterns

**Root Cause**: GPU, export, cache, parallel, and plot commands were registered as command groups but missing handler functions.

**Solution**: Added group handler functions for all command groups:
```bash
# Added handler functions
cli::_handle_group_gpu() {
    local subcmd="${1:-}"
    if [[ -z "$subcmd" ]] || [[ "$subcmd" == "--help" ]] || [[ "$subcmd" == "-h" ]]; then
        cli::_show_group_help "gpu"
        return 0
    fi
    shift
    cli::_dispatch_subcommand "gpu" "$subcmd" "$@"
}
# Similar handlers added for export, cache, parallel, plot
```

**Status**: ✅ FIXED (2025-09-16)

## Current Limitations

### 1. GPU Acceleration Requires Container Restart
**Limitation**: Enabling GPU requires stopping and restarting the container.

**Impact**: Brief service interruption when enabling GPU support.

**Mitigation**: Plan GPU enablement during maintenance windows.

### 2. Large Matrix Operations Memory Intensive
**Limitation**: Operations on matrices >1000x1000 can consume significant memory.

**Recommendation**: 
- Monitor memory usage for large computations
- Consider chunking large operations
- Use sparse matrices when possible

### 3. Jupyter Token Authentication Disabled
**Limitation**: Jupyter notebook interface has no authentication token.

**Security Note**: Only suitable for local development. Production deployments should enable authentication.

**To Enable Authentication**:
```bash
# Set JUPYTER_TOKEN environment variable
-e JUPYTER_TOKEN="your-secure-token"
```

## Performance Considerations

### Optimal Configuration
- **Memory**: 4GB recommended for complex computations
- **CPU**: 2+ cores for parallel processing
- **Storage**: 10GB for notebooks and outputs
- **GPU**: NVIDIA GPU with CUDA for acceleration

### Benchmark Baselines
- Symbolic computation: ~1500ms for 100 equations
- Matrix operations (20x20): ~800ms
- Prime factorization: ~750ms for large numbers
- Calculus operations: ~1200ms for integration suite

## Troubleshooting Guide

### Container Won't Start
```bash
# Check ports are available
lsof -i :8888
lsof -i :8889

# Check Docker status
docker ps -a | grep sagemath

# View logs
docker logs sagemath-main
```

### Calculations Timing Out
```bash
# Increase timeout in calculate function
timeout 30 docker exec ...  # Default is 10

# Check container resources
docker stats sagemath-main
```

### Plot Files Not Generated
```bash
# Verify output directory mounted
docker exec sagemath-main ls -la /home/sage/outputs

# Check permissions
ls -la /home/matthalloran8/Vrooli/data/resources/sagemath/outputs
```

### Parallel Computing Not Working
```bash
# Verify multiprocessing available
resource-sagemath parallel status

# Check CPU allocation
docker inspect sagemath-main | grep -i cpu
```

## Integration Notes

### Working with Other Resources
- **Jupyter Integration**: Compatible with JupyterHub deployments
- **GPU Sharing**: Can share NVIDIA GPUs with other containers
- **Data Exchange**: Use mounted volumes for data sharing

### API Endpoints
- Health: `http://localhost:8888/api` (Jupyter API)
- Notebooks: `http://localhost:8888` (Web interface)
- Calculations: Via CLI or docker exec

## Future Improvements

### Planned Enhancements
1. [ ] Add authentication to Jupyter interface
2. [x] Implement result caching for repeated calculations ✅ (Completed 2025-09-15)
3. [ ] Add support for SageMath Cloud features
4. [ ] Create REST API endpoint for calculations
5. [x] Add support for LaTeX output formatting ✅ (Completed 2025-09-15)

### Performance Optimizations
1. [ ] Implement connection pooling for docker exec
2. [x] Add computation result caching ✅ (Completed 2025-09-15)
3. [ ] Optimize container startup time
4. [ ] Implement lazy loading of heavy libraries

## References
- [SageMath Documentation](https://doc.sagemath.org/)
- [SageMath Docker Image](https://hub.docker.com/r/sagemath/sagemath)
- [Jupyter Configuration](https://jupyter-notebook.readthedocs.io/)
- [CUDA Setup Guide](https://docs.nvidia.com/cuda/)

---
*Last Updated: 2025-09-16*