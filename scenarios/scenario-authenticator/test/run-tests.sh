#!/bin/bash
set -euo pipefail

# Comprehensive test runner for scenario-authenticator

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../" && pwd)"
cd "$SCENARIO_DIR"

echo "==================================="
echo "Scenario Authenticator - Phased Testing"
echo "==================================="

# Color codes for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Run all phase tests in order
PHASES_DIR="test/phases"
FAILED_PHASES=()

if [ -d "$PHASES_DIR" ]; then
  # Define phase execution order
  PHASE_ORDER=(
    "test-structure.sh"
    "test-dependencies.sh"
    "test-unit.sh"
    "test-integration.sh"
    "test-business.sh"
    "test-performance.sh"
  )

  for phase_name in "${PHASE_ORDER[@]}"; do
    phase="$PHASES_DIR/$phase_name"
    if [ -f "$phase" ]; then
      echo ""
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo "Running phase: $(basename "$phase")"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

      if bash "$phase"; then
        echo -e "${GREEN}✓ Phase $(basename "$phase") completed${NC}"
      else
        echo -e "${YELLOW}⚠ Phase $(basename "$phase") failed${NC}"
        FAILED_PHASES+=("$phase_name")
      fi
    else
      echo -e "${YELLOW}⚠ Phase $phase_name not found, skipping${NC}"
    fi
  done
else
  echo "✗ Phases directory missing: $PHASES_DIR"
  exit 1
fi

# Run legacy tests if they exist (for backward compatibility)
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Running legacy test scripts (if present)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Run auth-flow test
if [ -f test/auth-flow.sh ]; then
  echo ""
  echo "Running legacy auth-flow test..."
  if bash test/auth-flow.sh; then
    echo -e "${GREEN}✓ Legacy auth-flow test completed${NC}"
  else
    echo -e "${YELLOW}⚠ Legacy auth-flow test failed${NC}"
    FAILED_PHASES+=("auth-flow.sh")
  fi
fi

# Run 2FA test
if [ -f test/test-2fa.sh ]; then
  echo ""
  echo "Running 2FA test..."
  if bash test/test-2fa.sh; then
    echo -e "${GREEN}✓ 2FA test completed${NC}"
  else
    echo -e "${YELLOW}⚠ 2FA test failed${NC}"
    FAILED_PHASES+=("test-2fa.sh")
  fi
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ ${#FAILED_PHASES[@]} -eq 0 ]; then
  echo -e "${GREEN}✅ All tests completed successfully!${NC}"
  exit 0
else
  echo -e "${YELLOW}⚠ Some tests failed:${NC}"
  for failed in "${FAILED_PHASES[@]}"; do
    echo "  - $failed"
  done
  exit 1
fi
