#!/bin/bash
set -e

echo \"=== Integration Tests ===\"

# Add integration tests here, e.g., curl health endpoint

curl -f http://localhost:8080/api/v1/health || (echo \"Integration test failed\"; exit 1)

echo \"✅ Integration tests completed\"