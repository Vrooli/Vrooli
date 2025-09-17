# MEEP Resource - Known Issues and Solutions

## Resolved Issues

### 1. Parameter Sweep API Endpoint Issue
**Problem**: The `/batch/sweep` endpoint was incorrectly expecting query parameters instead of JSON body
- **Symptom**: 422 Unprocessable Content error when calling parameter sweep
- **Root Cause**: FastAPI function signature was using direct parameters instead of request body
- **Solution**: Changed function to accept `request: dict = Body(...)` and parse JSON body
- **Date Fixed**: 2025-09-16

### 2. Docker Build Time
**Problem**: Docker image build takes 10-15 minutes due to conda package installation
- **Symptom**: Long wait times during installation
- **Root Cause**: Conda needs to download and install large scientific packages
- **Status**: Working but could be optimized
- **Potential Solution**: Pre-built base image or switch to pip-only installation

### 3. Integration Configuration
**Problem**: MinIO and PostgreSQL integrations are implemented but disabled by default
- **Symptom**: Storage falls back to filesystem instead of using MinIO/PostgreSQL
- **Root Cause**: Environment variables default to false for safety
- **Solution**: Enable via environment variables when resources are available

## Current Limitations

### 1. GPU Support Not Implemented
- GPU acceleration code is stubbed but not functional
- Requires CUDA/OpenCL setup in Docker container
- Would significantly speed up large simulations

### 2. Real MEEP Simulations Limited
- API runs simplified mock simulations for testing
- Full MEEP simulation runner needs more development
- Templates are placeholders, not production-ready

### 3. MPI Configuration
- MPI is installed but not fully utilized
- Parallel simulations not properly orchestrated
- Would benefit from job queue implementation

## Testing Notes

### Test Coverage
- **Smoke Tests**: 5/5 passing - Basic health and availability
- **Integration Tests**: 6/6 passing - API endpoints and workflows
- **Unit Tests**: 5/5 passing - Configuration and setup

### Performance Metrics
- Health check response: <500ms (target met)
- API response times: <1s for most endpoints
- Container startup: ~10-30s (within target)
- Docker image size: 3.59GB (large but acceptable for scientific software)

## Recommendations for Future Improvements

1. **Optimize Docker Build**: Create layered base image with conda pre-installed
2. **Enable Storage Integrations**: Connect to MinIO/PostgreSQL when available
3. **Implement Real Simulations**: Replace mock simulations with actual MEEP runs
4. **Add GPU Support**: Enable CUDA for performance-critical simulations
5. **Build Job Queue**: Implement proper async job handling for long simulations
6. **Enhance Templates**: Create production-ready simulation templates
7. **Add Monitoring**: Integrate with Prometheus/Grafana for metrics

## Dependencies

- **Ubuntu 22.04**: Base OS for Docker container
- **Conda**: Package manager for scientific Python packages
- **Python 3.x**: Runtime for API server
- **FastAPI**: Web framework for REST API
- **MEEP**: Electromagnetic simulation library
- **OpenMPI**: For parallel computing support

## Integration Points

- **Port 8193**: Allocated from central port registry
- **Data Storage**: `/home/matthalloran8/.vrooli/resources/meep/data`
- **Docker Container**: `meep-server`
- **Image**: `vrooli/meep:latest`