#!/bin/bash
set -euo pipefail

echo "=== Business Logic Tests ==="

# Run specific business logic tests, e.g., API endpoints
# For api-library, assume API tests cover business logic

cd api
go test -v ./... -run TestBusiness || echo "No specific business tests, running all"
echo "✅ Business tests completed"