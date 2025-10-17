#!/bin/bash
set -euo pipefail

echo "Testing GeoNode configuration..."
source lib/core.sh
load_config
echo "Port: $GEONODE_PORT"
echo "GeoServer Port: $GEONODE_GEOSERVER_PORT"
echo "Test complete"