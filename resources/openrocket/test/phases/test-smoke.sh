#!/usr/bin/env bash
# OpenRocket Smoke Tests

echo "==================== OpenRocket Smoke Tests ===================="
echo "Running OpenRocket smoke tests..."

failed=0

# Test container running
echo -n "  Testing container running... "
if docker ps | grep -q openrocket-server &>/dev/null; then
    echo -e "\033[0;32m✓\033[0m"
else
    echo -e "\033[0;31m✗\033[0m"
    ((failed++))
fi

# Test health endpoint
echo -n "  Testing health endpoint... "
if timeout 5 curl -sf http://localhost:9513/health &>/dev/null; then
    echo -e "\033[0;32m✓\033[0m"
else
    echo -e "\033[0;31m✗\033[0m"
    ((failed++))
fi

# Test API responsive
echo -n "  Testing API responsive... "
if timeout 5 curl -sf http://localhost:9513/api/designs &>/dev/null; then
    echo -e "\033[0;32m✓\033[0m"
else
    echo -e "\033[0;31m✗\033[0m"
    ((failed++))
fi

if (( failed > 0 )); then
    echo "Smoke tests failed: $failed errors"
    exit 1
fi

echo "All smoke tests passed!"
exit 0