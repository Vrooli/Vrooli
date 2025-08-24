#\!/usr/bin/env bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
MOCK_DIR="${APP_ROOT}/__test/mocks/tier2"

echo "Loading Redis mock..."
source "${MOCK_DIR}/redis.sh"

echo "Testing Redis..."
redis-cli ping
redis-cli set "test" "value"
result=$(redis-cli get "test")
echo "Result: $result"

if [[ "$result" == "value" ]]; then
    echo "✅ Test passed\!"
else
    echo "❌ Test failed\!"
fi
