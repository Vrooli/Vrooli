#!/bin/bash
set -e
echo "=== Dependencies Tests ==="
if [ -d api ]; then
  cd api && go mod tidy && go mod verify
  echo "✅ Go dependencies verified"
fi
echo "✅ Dependencies phase completed"