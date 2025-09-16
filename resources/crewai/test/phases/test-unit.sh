#!/usr/bin/env bash
################################################################################
# CrewAI Unit Tests - v2.0 Universal Contract Compliant
#
# Library function validation tests (<60s) for CrewAI resource
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
CREWAI_ROOT="${APP_ROOT}/resources/crewai"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${CREWAI_ROOT}/lib/test.sh"

# Run unit tests directly
crewai::test::unit