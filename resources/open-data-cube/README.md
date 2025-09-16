# Open Data Cube Resource

Open Data Cube (ODC) is a comprehensive Earth observation platform that transforms satellite imagery into analysis-ready data cubes. It provides powerful geospatial analytics capabilities for climate monitoring, agricultural intelligence, and urban planning scenarios.

## Features

- **Complete Stack**: PostgreSQL/PostGIS + datacube-core + OWS services in Docker
- **Satellite Data Management**: Index and query Sentinel-2, Landsat, and other datasets
- **Data Cube Analytics**: Time-series analysis across space and time dimensions
- **Export Formats**: GeoTIFF, GeoJSON, NetCDF for various applications
- **OGC Services**: WMS/WCS/WFS endpoints for GIS integration
- **Sample Datasets**: Pre-configured Sentinel and Landsat samples

## Quick Start

```bash
# Check ODC status
vrooli resource open-data-cube status

# Install and start ODC stack
vrooli resource open-data-cube manage install
vrooli resource open-data-cube manage start --wait

# Index sample dataset
vrooli resource open-data-cube dataset index /data/sentinel2/sample.yaml

# Query by area and time
vrooli resource open-data-cube query area '{"type":"Polygon","coordinates":[[[149.0,-35.3],[149.2,-35.3],[149.2,-35.1],[149.0,-35.1],[149.0,-35.3]]]}'

# Export to GeoTIFF
vrooli resource open-data-cube export geotiff output.tif
```

## Architecture

The Open Data Cube stack consists of:

- **odc-db**: PostgreSQL 13 with PostGIS for spatial data storage
- **odc-api**: Datacube core API and processing engine
- **odc-ows**: OGC Web Services (WMS/WCS/WFS)
- **odc-redis**: Caching layer for performance

## CLI Commands

### Lifecycle Management
```bash
vrooli resource open-data-cube manage install   # Install dependencies
vrooli resource open-data-cube manage start     # Start ODC stack
vrooli resource open-data-cube manage stop      # Stop ODC stack
vrooli resource open-data-cube manage restart   # Restart services
vrooli resource open-data-cube manage uninstall # Remove installation
```

### Dataset Operations
```bash
# Index new dataset
vrooli resource open-data-cube dataset index /path/to/dataset.yaml

# List indexed datasets
vrooli resource open-data-cube dataset list

# Search datasets
vrooli resource open-data-cube dataset search "product='sentinel2'"
```

### Data Queries
```bash
# Query by geographic area (GeoJSON)
vrooli resource open-data-cube query area '<geojson>'

# Query by time range
vrooli resource open-data-cube query time '2024-01-01/2024-12-31'

# Query by product type
vrooli resource open-data-cube query product 'landsat8'
```

### Export Functions
```bash
# Export to various formats
vrooli resource open-data-cube export geotiff output.tif
vrooli resource open-data-cube export geojson output.json
vrooli resource open-data-cube export netcdf output.nc
```

## Sample Datasets

The resource includes sample configurations for:
- **Sentinel-2**: 10m resolution multispectral imagery
- **Landsat 8**: 30m resolution with thermal bands
- **Custom Products**: Define your own data products

## Integration Examples

### With MinIO for Storage
```bash
# Configure MinIO backend for large raster storage
export ODC_MINIO_INTEGRATION=true
vrooli resource open-data-cube manage restart
```

### With n8n for Automation
```bash
# Import satellite ingestion workflow
vrooli resource n8n workflow import examples/satellite-ingestion.json
```

### With Qdrant for Similarity Search
```bash
# Enable vector embeddings for scene similarity
export ODC_QDRANT_INTEGRATION=true
vrooli resource open-data-cube manage restart
```

## Use Cases

### Climate Monitoring
```python
# Analyze temperature trends over time
datacube query product='lst' time='2020/2024' | \
  vrooli scenario climate-analyzer process
```

### Agricultural Intelligence
```python
# Monitor crop health with NDVI
datacube query product='sentinel2' | \
  calculate-ndvi | \
  vrooli resource farmos import
```

### Urban Planning
```python
# Track urban expansion
datacube query area='city_boundary.json' | \
  detect-changes | \
  export-report
```

## API Endpoints

- **Health Check**: `http://localhost:${ODC_PORT}/health`
- **WMS Service**: `http://localhost:${DATACUBE_OWS_PORT}/wms`
- **WCS Service**: `http://localhost:${DATACUBE_OWS_PORT}/wcs`
- **Database**: `postgresql://datacube:password@localhost:${ODC_DB_PORT}/datacube`

## Testing

```bash
# Run all tests
vrooli resource open-data-cube test all

# Run specific test phases
vrooli resource open-data-cube test smoke       # Quick health check
vrooli resource open-data-cube test integration # Full functionality
vrooli resource open-data-cube test unit        # Library tests
```

## Troubleshooting

### Container Issues
```bash
# Check container logs
vrooli resource open-data-cube logs api
vrooli resource open-data-cube logs db
vrooli resource open-data-cube logs ows

# Restart specific service
docker restart open-data-cube-api
```

### Database Connection
```bash
# Test database connection
docker exec open-data-cube-db pg_isready -U datacube

# Access database directly
docker exec -it open-data-cube-db psql -U datacube
```

### Performance Optimization
- Increase `ODC_MAX_WORKERS` for parallel processing
- Adjust `ODC_CHUNK_SIZE` for memory management
- Enable Redis caching for repeated queries

## Resources

- [Open Data Cube Documentation](https://datacube-core.readthedocs.io/)
- [OGC Services Guide](https://datacube-ows.readthedocs.io/)
- [Sample Datasets](https://registry.opendata.aws/sentinel-2/)

## License

This resource integrates the Open Data Cube project, which is licensed under Apache 2.0.