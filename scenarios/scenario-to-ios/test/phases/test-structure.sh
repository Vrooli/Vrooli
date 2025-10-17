#!/bin/bash
set -e

echo \"=== Structure Tests ===\"

# Check for required files and structure

if [ ! -f .vrooli/service.json ]; then
  echo \"Missing service.json\"
  exit 1
fi

if [ ! -f api/main.go ]; then
  echo \"Missing api/main.go\"
  exit 1
fi

echo \"✅ Structure tests completed\"