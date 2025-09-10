#!/usr/bin/env bash
# Gazebo Smoke Test Phase

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Run smoke tests
test_smoke