#!/bin/bash
set -e

echo "Running comprehensive tests for Typing Test scenario"

# Run unit tests
echo "Running unit tests..."
bash test/phases/test-unit.sh
echo "Unit tests completed"

# Run integration tests
echo "Running integration tests..."
bash test/phases/test-integration.sh
echo "Integration tests completed"

# Run business logic tests
echo "Running business logic tests..."
bash test/phases/test-business.sh
echo "Business logic tests completed"

# Run dependency tests
echo "Running dependency tests..."
bash test/phases/test-dependencies.sh
echo "Dependency tests completed"

# Run performance tests
echo "Running performance tests..."
bash test/phases/test-performance.sh
echo "Performance tests completed"

echo "All tests completed successfully!"