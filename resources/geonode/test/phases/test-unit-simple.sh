#!/bin/bash

# Simple GeoNode Unit Tests

set -e

RESOURCE_DIR="/home/matthalloran8/Vrooli/resources/geonode"
cd "$RESOURCE_DIR"

echo "Running GeoNode unit tests..."

# Load configuration
source lib/core.sh
load_config

# Test configuration
echo -n "Testing port configuration... "
if [[ -n "$GEONODE_PORT" ]] && [[ "$GEONODE_PORT" -gt 1024 ]]; then
    echo "✓"
else
    echo "✗"
    exit 1
fi

echo -n "Testing database configuration... "
if [[ -n "$GEONODE_DB_NAME" ]]; then
    echo "✓"
else
    echo "✗"
    exit 1
fi

echo -n "Testing file structure... "
if [[ -f cli.sh ]] && [[ -f lib/core.sh ]] && [[ -f lib/test.sh ]]; then
    echo "✓"
else
    echo "✗"
    exit 1
fi

echo -n "Testing JSON configs... "
if jq empty config/schema.json && jq empty config/runtime.json; then
    echo "✓"
else
    echo "✗"
    exit 1
fi

echo "All unit tests passed!"