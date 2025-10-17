# Open Data Cube - Known Issues and Solutions

## Current Issues

### 1. Sample Dataset Required
**Problem**: The resource doesn't include sample satellite data for testing  
**Impact**: Cannot demonstrate full functionality without actual datasets  
**Solution**: Future improvers should add Sentinel-2 or Landsat sample data scripts

### 2. Port Allocation Method
**Problem**: Ports are hardcoded in core.sh as fallbacks due to lifecycle complexity  
**Impact**: Not fully compliant with Vrooli's dynamic port allocation  
**Solution**: Once lifecycle v2.0 is fully implemented, remove hardcoded defaults

### 3. OWS Configuration Missing
**Problem**: The OWS container needs a proper configuration file  
**Impact**: WMS/WCS endpoints may not work correctly without config  
**Solution**: Add datacube_ows_cfg.py configuration file

## Resolved Issues

### 1. Docker Image Name (FIXED 2025-01-16)
**Problem**: Used incorrect image name `opendatacube/datacube-ows`  
**Solution**: Changed to correct name `opendatacube/ows`

### 2. Port Fallbacks (FIXED 2025-01-16)
**Problem**: Had port fallbacks which violate Vrooli standards  
**Solution**: Removed fallbacks, added to port_registry.sh

## Future Improvements

1. **Data Seeding**: Add scripts to download and index sample Sentinel-2 scenes
2. **MinIO Integration**: Connect to MinIO for large raster storage
3. **Qdrant Integration**: Use for similarity search on satellite scenes
4. **Performance**: Add Dask for distributed processing
5. **Monitoring**: Add Prometheus metrics for query performance

## Testing Notes

- Smoke tests verify structure but need running containers for full validation
- Integration tests require actual dataset to be meaningful  
- Unit tests successfully validate configuration and structure