#!/usr/bin/env bash
# Smoke tests - quick verification of core functionality
# Target time: <30s

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Smoke Phase (Target: <30s) ==="

# Check if scenario is running
echo "  ✓ Checking if service is running..."
if ! pgrep -f "image-tools-api" > /dev/null; then
  echo -e "${RED}✗ Service not running${NC}"
  exit 1
fi

# Get API port - use environment variable first, then check the service status
if [ -n "$API_PORT" ]; then
  : # Use provided API_PORT
elif [ -f "${SCENARIO_DIR}/.vrooli/runtime/ports.json" ]; then
  API_PORT=$(jq -r '.api // empty' "${SCENARIO_DIR}/.vrooli/runtime/ports.json" 2>/dev/null || echo "")
fi

# If still not found, check vrooli scenario status output
if [ -z "$API_PORT" ]; then
  API_PORT=$(vrooli scenario status image-tools 2>/dev/null | grep -oP 'API.*http://localhost:\K[0-9]+' | head -1 || echo "")
fi

# Last resort: check process list
if [ -z "$API_PORT" ]; then
  API_PORT=$(ps aux | grep -E '[i]mage-tools.*--port[= ]([0-9]+)' | grep -oP 'port[= ]\K[0-9]+' | head -1 || echo "")
fi

if [ -z "$API_PORT" ]; then
  echo -e "${RED}✗ Could not determine API port${NC}"
  echo "  Try setting API_PORT environment variable"
  exit 1
fi

echo "  ✓ Using API port: $API_PORT"

# Test 1: Health check
echo "  ✓ Testing health endpoint..."
if ! curl -sf "http://localhost:${API_PORT}/health" > /dev/null; then
  echo -e "${RED}✗ Health check failed${NC}"
  exit 1
fi

# Test 2: Plugin registry
echo "  ✓ Testing plugin registry..."
PLUGINS=$(curl -sf "http://localhost:${API_PORT}/api/v1/plugins" || echo "{}")
if [ -z "$PLUGINS" ] || [ "$PLUGINS" = "{}" ]; then
  echo -e "${RED}✗ Plugin registry not responding${NC}"
  exit 1
fi

# Verify core plugins are loaded
if ! echo "$PLUGINS" | grep -q "jpeg"; then
  echo -e "${RED}✗ JPEG plugin not loaded${NC}"
  exit 1
fi

# Test 3: Presets endpoint
echo "  ✓ Testing presets endpoint..."
PRESETS=$(curl -sf "http://localhost:${API_PORT}/api/v1/presets" || echo "{}")
if [ -z "$PRESETS" ] || [ "$PRESETS" = "{}" ]; then
  echo -e "${RED}✗ Presets not responding${NC}"
  exit 1
fi

# Verify web-optimized preset exists
if ! echo "$PRESETS" | grep -q "web-optimized"; then
  echo -e "${RED}✗ web-optimized preset not found${NC}"
  exit 1
fi

# Test 4: CLI is installed and working
echo "  ✓ Testing CLI..."
if ! command -v image-tools > /dev/null; then
  echo -e "${YELLOW}⚠️  CLI not in PATH (this is okay if not installed)${NC}"
else
  if ! image-tools help > /dev/null 2>&1; then
    echo -e "${RED}✗ CLI help command failed${NC}"
    exit 1
  fi
fi

# Test 5: UI health (if UI_PORT is available)
UI_PORT="${UI_PORT:-$(grep -o 'UI_PORT=[0-9]*' "${SCENARIO_DIR}/.vrooli/service.json" | cut -d= -f2 | head -1)}"
if [ -n "$UI_PORT" ]; then
  echo "  ✓ Testing UI health on port $UI_PORT..."
  if ! curl -sf "http://localhost:${UI_PORT}/health" > /dev/null; then
    echo -e "${YELLOW}⚠️  UI health check failed (may not be critical)${NC}"
  fi
fi

echo "  ✓ Smoke tests complete"

exit 0
