#!/bin/bash

# Open Data Cube Sample Query Examples
# Demonstrates various ODC query and export operations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Open Data Cube Sample Query Examples"
echo "===================================="
echo ""

# Example 1: Query by geographic area
echo "1. Querying data for Canberra region..."
vrooli resource open-data-cube query area '{
  "type": "Polygon",
  "coordinates": [[[
    149.0, -35.3],
    [149.2, -35.3],
    [149.2, -35.1],
    [149.0, -35.1],
    [149.0, -35.3
  ]]]
}'

echo ""

# Example 2: Query by time range
echo "2. Querying data for 2024..."
vrooli resource open-data-cube query time '2024-01-01/2024-12-31'

echo ""

# Example 3: Query specific product
echo "3. Querying Sentinel-2 data..."
vrooli resource open-data-cube query product 'sentinel2_l2a'

echo ""

# Example 4: Combined query and export
echo "4. Query and export to GeoTIFF..."
cat << 'EOF' > /tmp/query.py
import datacube
import numpy as np

# Initialize datacube
dc = datacube.Datacube()

# Define query parameters
query = {
    'product': 'sentinel2_l2a',
    'time': ('2024-01-01', '2024-01-31'),
    'lat': (-35.1, -35.3),
    'lon': (149.0, 149.2),
    'measurements': ['red', 'green', 'blue', 'nir']
}

# Load data
data = dc.load(**query)

# Calculate NDVI
ndvi = (data.nir - data.red) / (data.nir + data.red)

# Export to GeoTIFF
ndvi.rio.to_raster('ndvi_output.tif')
print("NDVI exported to ndvi_output.tif")

# Export RGB composite
rgb = data[['red', 'green', 'blue']].to_array()
rgb.rio.to_raster('rgb_composite.tif')
print("RGB composite exported to rgb_composite.tif")
EOF

docker exec open-data-cube-api python3 /tmp/query.py

echo ""

# Example 5: Climate anomaly detection
echo "5. Detecting temperature anomalies..."
cat << 'EOF' > /tmp/anomaly.py
import datacube
import xarray as xr
import numpy as np

dc = datacube.Datacube()

# Load land surface temperature data
lst = dc.load(
    product='landsat8_lst',
    time=('2023-01-01', '2024-12-31'),
    lat=(-35.3, -35.1),
    lon=(149.0, 149.2)
)

# Calculate monthly mean
monthly_mean = lst.groupby('time.month').mean()

# Calculate anomalies
anomalies = lst.groupby('time.month') - monthly_mean

# Find extreme events (>2 std deviations)
std = anomalies.std('time')
extreme_events = anomalies.where(abs(anomalies) > 2 * std)

print(f"Found {extreme_events.count().values} extreme temperature events")

# Export anomaly map
anomalies.mean('time').rio.to_raster('temperature_anomalies.tif')
print("Temperature anomaly map exported")
EOF

docker exec open-data-cube-api python3 /tmp/anomaly.py

echo ""

# Example 6: Agricultural monitoring
echo "6. Agricultural monitoring with vegetation indices..."
cat << 'EOF' > /tmp/agriculture.py
import datacube
import numpy as np

dc = datacube.Datacube()

# Load Sentinel-2 data for farm area
farm_data = dc.load(
    product='sentinel2_l2a',
    time=('2024-01-01', '2024-03-31'),
    lat=(-35.25, -35.20),
    lon=(149.05, 149.10),
    measurements=['red', 'nir', 'swir1', 'swir2']
)

# Calculate vegetation indices
# NDVI - Normalized Difference Vegetation Index
ndvi = (farm_data.nir - farm_data.red) / (farm_data.nir + farm_data.red)

# NDMI - Normalized Difference Moisture Index
ndmi = (farm_data.nir - farm_data.swir1) / (farm_data.nir + farm_data.swir1)

# NBR - Normalized Burn Ratio (useful for crop stress)
nbr = (farm_data.nir - farm_data.swir2) / (farm_data.nir + farm_data.swir2)

# Calculate statistics
ndvi_mean = ndvi.mean(['latitude', 'longitude'])
print(f"Average NDVI: {ndvi_mean.values.mean():.3f}")
print(f"Crop health trend: {'Improving' if ndvi_mean[-1] > ndvi_mean[0] else 'Declining'}")

# Export time series
ndvi_mean.to_netcdf('farm_ndvi_timeseries.nc')
print("NDVI time series exported to NetCDF")
EOF

docker exec open-data-cube-api python3 /tmp/agriculture.py

echo ""
echo "===================================="
echo "Sample queries completed!"
echo ""
echo "Generated outputs:"
echo "  - ndvi_output.tif: NDVI vegetation index"
echo "  - rgb_composite.tif: True color composite"
echo "  - temperature_anomalies.tif: Climate anomaly map"
echo "  - farm_ndvi_timeseries.nc: Agricultural monitoring data"