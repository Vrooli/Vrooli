#!/usr/bin/env bash
################################################################################
# CrewAI Integration Tests - v2.0 Universal Contract Compliant
#
# End-to-end functionality tests (<120s) for CrewAI resource
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
CREWAI_ROOT="${APP_ROOT}/resources/crewai"

# Source test library
# shellcheck disable=SC1091
source "${CREWAI_ROOT}/lib/test.sh"

# Run integration tests
crewai::test::integration