#!/usr/bin/env bash
# Apache Airflow Resource - Smoke Test Phase

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "${SCRIPT_DIR}")")"

# Run smoke tests
"${RESOURCE_DIR}/lib/test.sh" smoke "$@"