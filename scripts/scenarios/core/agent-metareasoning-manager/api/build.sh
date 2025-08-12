#!/bin/bash
# Build script for simplified metareasoning API

set -e

echo "Building simplified Metareasoning Coordinator API..."

# Clean previous build
rm -f metareasoning-api

# Build the simplified version
go build -o metareasoning-api cmd/server/main_simplified.go

echo "Build complete. Binary: metareasoning-api"
echo "Size: $(du -h metareasoning-api | cut -f1)"