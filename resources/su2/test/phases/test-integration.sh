#!/bin/bash
# SU2 Integration Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Load configuration
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

echo "=== SU2 Integration Tests ==="
echo "Testing full functionality..."

# Test 1: List designs
echo -n "1. List designs... "
response=$(timeout 5 curl -sf "http://localhost:${SU2_PORT}/api/designs" 2>/dev/null || echo "failed")
if [[ "$response" != "failed" ]] && echo "$response" | grep -q "meshes"; then
    echo "✓"
else
    echo "✗"
    echo "   Failed to list designs"
    exit 1
fi

# Test 2: Example files exist
echo -n "2. Example files... "
if [[ -f "${SU2_MESHES_DIR}/naca0012.su2" ]] && [[ -f "${SU2_CONFIGS_DIR}/test.cfg" ]]; then
    echo "✓"
else
    echo "⚠ (Downloading examples...)"
    download_examples
    if [[ -f "${SU2_MESHES_DIR}/naca0012.su2" ]]; then
        echo "   ✓ Examples downloaded"
    else
        echo "   ✗ Failed to download examples"
        exit 1
    fi
fi

# Test 3: Submit test simulation
echo -n "3. Simulation submission... "
sim_response=$(curl -sf -X POST "http://localhost:${SU2_PORT}/api/simulate" \
    -H "Content-Type: application/json" \
    -d '{"mesh":"naca0012.su2","config":"test.cfg"}' 2>/dev/null || echo "failed")

if [[ "$sim_response" != "failed" ]] && echo "$sim_response" | grep -q "simulation_id"; then
    echo "✓"
    sim_id=$(echo "$sim_response" | grep -o '"simulation_id":"[^"]*"' | cut -d'"' -f4)
    echo "   Simulation ID: $sim_id"
else
    echo "✗"
    echo "   Failed to submit simulation"
    exit 1
fi

# Test 4: Check simulation status
echo -n "4. Simulation status... "
sleep 2
status_response=$(curl -sf "http://localhost:${SU2_PORT}/api/results/${sim_id}" 2>/dev/null || echo "failed")
if [[ "$status_response" != "failed" ]] && echo "$status_response" | grep -q "simulation"; then
    echo "✓"
else
    echo "✗"
    echo "   Failed to get simulation status"
    exit 1
fi

# Test 5: MPI availability
echo -n "5. MPI functionality... "
if docker exec "${SU2_CONTAINER_NAME}" which mpirun > /dev/null 2>&1; then
    echo "✓"
else
    echo "⚠ (MPI not available)"
fi

echo -e "\n✓ Integration tests passed"
exit 0