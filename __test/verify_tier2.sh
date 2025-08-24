#!/usr/bin/env bash
# Quick verification script for Tier 2 mocks

# Use cached APP_ROOT pattern for path robustness
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/.." && builtin pwd)}"
export APP_ROOT

echo "=== Tier 2 Mock Verification ==="
echo ""

# Test a few key mocks
test_mock() {
    local mock_name="$1"
    local mock_file="${APP_ROOT}/__test/mocks/tier2/${mock_name}.sh"
    
    echo -n "Testing $mock_name: "
    
    if [[ ! -f "$mock_file" ]]; then
        echo "✗ File not found"
        return 1
    fi
    
    # Try to source and test
    if bash -c "
        source '$mock_file' 2>/dev/null
        if declare -F test_${mock_name}_connection >/dev/null 2>&1; then
            test_${mock_name}_connection >/dev/null 2>&1 && echo '✓ Connection test passed' && exit 0
            echo '✗ Connection test failed'
            exit 1
        else
            echo '✗ No test function'
            exit 1
        fi
    "; then
        return 0
    else
        return 1
    fi
}

# Test key mocks
success=0
failed=0

for mock in redis postgres n8n docker ollama minio; do
    if test_mock "$mock"; then
        ((success++))
    else
        ((failed++))
    fi
done

echo ""
echo "Summary: $success passed, $failed failed"

# Test integration
echo ""
echo "=== Integration Test ==="
bash << EOF
    source '${APP_ROOT}/__test/mocks/tier2/redis.sh'
    source '${APP_ROOT}/__test/mocks/tier2/postgres.sh'
    
    # Reset
    redis_mock_reset 2>/dev/null || true
    postgres_mock_reset 2>/dev/null || true
    
    # Test Redis
    redis-cli set testkey testvalue 2>/dev/null
    result=\$(redis-cli get testkey 2>/dev/null)
    if [[ "\$result" == "testvalue" ]]; then
        echo "✓ Redis get/set works"
    else
        echo "✗ Redis get/set failed"
    fi
    
    # Test PostgreSQL
    if psql -c "SELECT 1" >/dev/null 2>&1; then
        echo "✓ PostgreSQL query works"
    else
        echo "✗ PostgreSQL query failed"
    fi
EOF

echo ""
echo "=== Mock Statistics ==="
echo "Total Tier 2 mocks: $(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" | wc -l)"
echo "Executable: $(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" -executable | wc -l)"
echo "Average lines: $(wc -l "${APP_ROOT}"/__test/mocks/tier2/*.sh 2>/dev/null | tail -1 | awk '{print int($1/24)}')"

exit $failed