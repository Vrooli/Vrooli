#!/bin/bash
# SU2 Unit Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Load configuration
source "${RESOURCE_DIR}/config/defaults.sh"

echo "=== SU2 Unit Tests ==="
echo "Testing configuration and dependencies..."

# Test 1: Configuration values
echo -n "1. Configuration loading... "
if [[ -n "$SU2_PORT" ]] && [[ "$SU2_PORT" == "9514" ]]; then
    echo "✓"
else
    echo "✗"
    echo "   Configuration not loaded properly"
    exit 1
fi

# Test 2: Directory structure
echo -n "2. Directory structure... "
all_dirs_exist=true
for dir in "${SU2_MESHES_DIR}" "${SU2_CONFIGS_DIR}" "${SU2_RESULTS_DIR}" "${SU2_CACHE_DIR}"; do
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir" 2>/dev/null || all_dirs_exist=false
    fi
done

if [[ "$all_dirs_exist" == "true" ]]; then
    echo "✓"
else
    echo "✗"
    echo "   Failed to create required directories"
    exit 1
fi

# Test 3: Docker availability
echo -n "3. Docker availability... "
if docker --version > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "   Docker not available"
    exit 1
fi

# Test 4: Required files exist
echo -n "4. Required files... "
required_files=(
    "${RESOURCE_DIR}/cli.sh"
    "${RESOURCE_DIR}/lib/core.sh"
    "${RESOURCE_DIR}/lib/test.sh"
    "${RESOURCE_DIR}/config/defaults.sh"
    "${RESOURCE_DIR}/config/runtime.json"
    "${RESOURCE_DIR}/config/schema.json"
)

all_files_exist=true
for file in "${required_files[@]}"; do
    if [[ ! -f "$file" ]]; then
        all_files_exist=false
        echo -e "\n   Missing: $file"
    fi
done

if [[ "$all_files_exist" == "true" ]]; then
    echo "✓"
else
    echo "✗"
    exit 1
fi

# Test 5: CLI executable
echo -n "5. CLI executable... "
if [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "✓"
else
    echo "✗"
    chmod +x "${RESOURCE_DIR}/cli.sh"
    echo "   Fixed: Made CLI executable"
fi

echo -e "\n✓ Unit tests passed"
exit 0