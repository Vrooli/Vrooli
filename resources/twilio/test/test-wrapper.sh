#!/usr/bin/env bash
# Test wrapper to isolate set -e issues

# Disable errexit for testing
set +e

# Change to script directory
cd "$(dirname "$0")/.." || exit 1

# Source test library
source lib/test.sh 2>/dev/null || {
    echo "Failed to source test library"
    exit 1
}

# Run the requested test
case "${1:-smoke}" in
    smoke)
        twilio::test::smoke
        ;;
    integration)
        twilio::test::integration
        ;;
    unit)
        twilio::test::unit
        ;;
    all)
        twilio::test::all
        ;;
    *)
        echo "Usage: $0 {smoke|integration|unit|all}"
        exit 1
        ;;
esac

# Return the test result
exit $?