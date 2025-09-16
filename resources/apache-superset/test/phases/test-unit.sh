#!/usr/bin/env bash
# Unit test for Apache Superset
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "${SCRIPT_DIR}")")"

# Run unit test via CLI
"${RESOURCE_DIR}/cli.sh" test unit