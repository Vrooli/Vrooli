#!/bin/bash
# Test: Dependency checks
# Ensures all required dependencies are available

set -e

echo "  ✓ Checking Go dependencies..."
cd api
go mod verify &> /dev/null || {
    echo "  ❌ Go module verification failed"
    exit 1
}

echo "  ✓ Checking binary exists..."
if [ ! -f "image-tools-api" ]; then
    echo "  ⚠️  Binary not found, building..."
    go build -o image-tools-api . || {
        echo "  ❌ Build failed"
        exit 1
    }
fi

cd ..

echo "  ✓ Checking CLI installation..."
if ! command -v image-tools &> /dev/null; then
    echo "  ⚠️  CLI not installed, installing..."
    cd cli
    ./install.sh &> /dev/null || {
        echo "  ❌ CLI installation failed"
        exit 1
    }
    cd ..
fi

echo "  ✓ Dependency checks complete"