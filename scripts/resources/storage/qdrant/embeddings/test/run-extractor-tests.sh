#!/usr/bin/env bash
# Test runner for embedding extractors

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EMBEDDING_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🧪 Running Embedding Extractor Tests"
echo

# Check if bats is available
if ! command -v bats >/dev/null; then
    echo "❌ BATS (Bash Automated Testing System) is required"
    echo "   Install with: npm install -g bats"
    echo "   Or: brew install bats-core"
    echo "   Or: apt-get install bats"
    exit 1
fi

# Check test fixtures
echo "🔍 Verifying test fixtures..."
source "$SCRIPT_DIR/helpers/setup.bash"

if ! verify_test_fixtures; then
    echo "❌ Test fixtures missing or invalid"
    exit 1
fi
echo "✅ Test fixtures verified"

# Run individual extractor tests
test_files=(
    "$SCRIPT_DIR/extractors/test-workflows.bats"
    "$SCRIPT_DIR/extractors/test-scenarios.bats"  
    "$SCRIPT_DIR/extractors/test-docs.bats"
    "$SCRIPT_DIR/extractors/test-code.bats"
    "$SCRIPT_DIR/extractors/test-resources.bats"
)

failed_tests=()
passed_tests=()

echo
echo "🏃 Running extractor tests..."

for test_file in "${test_files[@]}"; do
    extractor_name=$(basename "$test_file" .bats | sed 's/test-//')
    
    echo "  Testing $extractor_name extractor..."
    
    if bats "$test_file"; then
        passed_tests+=("$extractor_name")
        echo "  ✅ $extractor_name tests passed"
    else
        failed_tests+=("$extractor_name")
        echo "  ❌ $extractor_name tests failed"
    fi
    echo
done

# Report results
echo "📊 Test Results Summary"
echo "======================"
echo "✅ Passed: ${#passed_tests[@]}"
echo "❌ Failed: ${#failed_tests[@]}"
echo

if [ ${#passed_tests[@]} -gt 0 ]; then
    echo "Passed extractors:"
    printf "  • %s\n" "${passed_tests[@]}"
    echo
fi

if [ ${#failed_tests[@]} -gt 0 ]; then
    echo "Failed extractors:"
    printf "  • %s\n" "${failed_tests[@]}"
    echo
    echo "Run individual tests for more details:"
    for extractor in "${failed_tests[@]}"; do
        echo "  bats $SCRIPT_DIR/extractors/test-${extractor}.bats"
    done
    echo
    exit 1
fi

echo "🎉 All extractor tests passed!"
echo "The embedding system extractors are working correctly."