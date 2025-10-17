#!/usr/bin/env bash
# Apache Airflow Resource - Unit Test Phase

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "${SCRIPT_DIR}")")"

# Run unit tests
"${RESOURCE_DIR}/lib/test.sh" unit "$@"