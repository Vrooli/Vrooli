#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Resource dir: $RESOURCE_DIR"

# Source libraries
echo "Sourcing core.sh..."
source "${RESOURCE_DIR}/lib/core.sh"

echo "Loading config..."
load_config

echo "GEONODE_PORT: $GEONODE_PORT"
echo "GEONODE_GEOSERVER_PORT: $GEONODE_GEOSERVER_PORT"

echo "Testing functions..."
type -t get_status && echo "get_status exists" || echo "get_status missing"
type -t is_healthy && echo "is_healthy exists" || echo "is_healthy missing"
type -t is_service_healthy && echo "is_service_healthy exists" || echo "is_service_healthy missing"

echo "Test complete!"