#!/bin/bash
echo "=== Running Performance Tests ==="

cd "$(dirname "$0")/../../api"

echo "Running Go benchmark tests..."
go test -bench=. -benchmem -run=^$ -tags=testing . 2>&1 | tee /tmp/nutrition-tracker-bench.log

echo ""
echo "Running concurrency tests..."
go test -run=TestConcurrent -tags=testing -v .

echo ""
echo "Running response time tests..."
go test -run=TestResponseTime -tags=testing -v .

echo ""
echo "Running database connection pool tests..."
go test -run=TestDatabaseConnectionPool -tags=testing -v .

echo "✅ Performance tests completed"