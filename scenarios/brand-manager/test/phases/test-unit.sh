#!/bin/bash

set -e

echo "=== Testing Unit Tests ==="

# Run unit tests
go test ./...

echo "Unit tests passed."