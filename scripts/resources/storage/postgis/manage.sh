#!/bin/bash
# PostGIS Management Script

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Forward all commands to resource-postgis
"${SCRIPT_DIR}/resource-postgis" "$@"