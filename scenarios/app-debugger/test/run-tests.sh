#!/bin/bash
set -e

echo "Running App Debugger tests..."

cd api
go test -v ./...

cd ../ui
npm test --if-present || true

echo "Tests completed"