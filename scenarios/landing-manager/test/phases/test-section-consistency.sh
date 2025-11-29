#!/bin/bash
# test-section-consistency.sh - Validates section consistency across all files
#
# This test ensures that:
# 1. All implemented sections have matching schemas, components, and registrations
# 2. TypeScript types, DB constraints, and renderer are in sync
# 3. No orphaned entries exist

set -euo pipefail

SCENARIO_ROOT="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"

# Source the section management script for shared functions
source "${SCENARIO_ROOT}/scripts/manage-sections.sh" &>/dev/null || {
    echo "FAIL: Could not source manage-sections.sh"
    exit 1
}

# Run validation
echo "=== Section Consistency Test ==="
echo ""

if cmd_validate; then
    echo ""
    echo "PASS: All section consistency checks passed"
    exit 0
else
    echo ""
    echo "FAIL: Section consistency validation failed"
    exit 1
fi
