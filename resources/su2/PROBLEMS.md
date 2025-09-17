# SU2 Resource - Known Issues and Solutions

## Fixed Issues

### 1. Meson Build Options (FIXED)
**Problem**: Initial Docker build failed with `ERROR: Unknown options: "enable-mpi"`
**Root Cause**: SU2 v7.5.1 uses `with-mpi=enabled` instead of `enable-mpi=true`
**Solution**: Updated meson options in Dockerfile to use correct syntax
```bash
# Wrong
RUN ./meson.py build -Denable-mpi=true

# Correct
RUN ./meson.py build -Dwith-mpi=enabled
```

### 2. Health Check Timeout (FIXED)
**Problem**: Health checks were failing during service startup
**Root Cause**: Service takes 10-30 seconds to fully initialize
**Solution**: Added `--wait` flag to start command with proper timeout handling

## Current Limitations

### 1. Mesh Generation
**Limitation**: No direct CAD-to-mesh conversion within the resource
**Workaround**: Users must pre-generate meshes using external tools (Gmsh, Pointwise, etc.)
**Future Enhancement**: Add mesh generation capabilities or integrate mesh generation tools

### 2. Adjoint Solver Interface
**Limitation**: Adjoint solver not exposed through API yet
**Workaround**: Manual configuration file editing for adjoint runs
**Future Enhancement**: Add dedicated adjoint API endpoints

### 3. Large File Handling
**Limitation**: Large meshes (>100MB) may timeout during upload
**Workaround**: Mount mesh directory directly or use MinIO for large files
**Future Enhancement**: Implement chunked upload for large files

### 4. GPU Support
**Limitation**: GPU acceleration not available in current build
**Workaround**: Use MPI parallelization for performance
**Future Enhancement**: Build with GPU support when CUDA available

## Performance Considerations

### 1. Docker Build Time
**Issue**: Initial Docker build takes 15-20 minutes due to SU2 compilation
**Impact**: First-time setup is slow
**Mitigation**: Consider pre-built Docker images in registry

### 2. Memory Usage
**Issue**: Large simulations can consume significant memory (>4GB)
**Impact**: May cause OOM on resource-constrained systems
**Mitigation**: Configure memory limits appropriately in defaults.sh

### 3. MPI Process Scaling
**Issue**: Default 4 MPI processes may not be optimal for all cases
**Impact**: Sub-optimal performance for large/small problems
**Mitigation**: Allow dynamic MPI process configuration based on problem size

## Integration Gaps

### 1. Direct Resource Communication
**Gap**: No direct API calls between SU2 and other resources (QuestDB, Qdrant)
**Current**: Manual export/import commands required
**Improvement**: Add automatic data pipeline triggers

### 2. Workflow Orchestration
**Gap**: Multi-step workflows require manual coordination
**Current**: User must manually chain commands
**Improvement**: Add workflow templates or integrate with n8n

### 3. Result Caching
**Gap**: No caching of simulation results
**Current**: Re-runs always compute from scratch
**Improvement**: Add Redis integration for result caching

## Security Considerations

### 1. File Upload Validation
**Risk**: Malformed mesh files could cause crashes
**Mitigation**: Add file validation before processing

### 2. Resource Limits
**Risk**: Unbounded simulations could consume all resources
**Mitigation**: Implement job queuing and resource quotas

### 3. API Authentication
**Risk**: API is currently unauthenticated
**Mitigation**: Integrate with scenario-authenticator when available

## Troubleshooting Guide

### Container Won't Start
```bash
# Check Docker status
docker ps -a | grep su2

# View logs
docker logs vrooli-su2

# Verify port availability
lsof -i :9514
```

### Health Check Failures
```bash
# Test health endpoint directly
curl -v http://localhost:9514/health

# Check container networking
docker inspect vrooli-su2 | grep -A 10 NetworkMode
```

### Simulation Failures
```bash
# Check simulation logs
vrooli resource su2 logs

# Verify mesh and config files
vrooli resource su2 content list

# Test with simple case
vrooli resource su2 content execute naca0012.su2 test.cfg
```

### MPI Issues
```bash
# Check MPI installation in container
docker exec vrooli-su2 mpirun --version

# Test with single process
docker exec vrooli-su2 bash -c "SU2_CFD test.cfg"
```

## Recommendations

1. **Production Deployment**: Use pre-built Docker images from registry
2. **Large Simulations**: Configure appropriate resource limits
3. **Data Persistence**: Mount volume for results directory
4. **Monitoring**: Integrate with system monitoring for resource usage
5. **Backup**: Regular backup of simulation results and configurations