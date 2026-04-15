#!/usr/bin/env bash
################################################################################
# Twilio Integration Test Phase
# 
# End-to-end functionality tests that complete in <120 seconds
################################################################################

set -euo pipefail

# Get the directory of this script
PHASE_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${PHASE_DIR}/.." && builtin pwd)"
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
TWILIO_DIR="${RESOURCE_DIR}"

# Source test library
source "${TWILIO_DIR}/lib/test.sh"

# Run integration tests
twilio::test::integration