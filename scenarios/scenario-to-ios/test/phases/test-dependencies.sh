#!/bin/bash
set -e

echo \"=== Dependency Tests ===\"

# Check Go dependencies

go mod tidy

echo \"✅ Dependency tests completed\"