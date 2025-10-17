#!/bin/bash
set -e
echo "=== Structure Checks ==="

# Check Go code formatting and linting
if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then
  echo "Checking Go code structure..."

  # Run go vet
  cd api
  echo "Running go vet..."
  go vet ./... || echo "⚠️  Some vet warnings found"

  # Check gofmt
  echo "Checking gofmt compliance..."
  unformatted=$(gofmt -l . 2>/dev/null || true)
  if [ -n "$unformatted" ]; then
    echo "⚠️  Some files not formatted:"
    echo "$unformatted"
  else
    echo "✅ All Go files properly formatted"
  fi

  cd ..
fi

# Check for required files
echo "Checking required scenario files..."
required_files=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "PROBLEMS.md"
  "Makefile"
)

for file in "${required_files[@]}"; do
  if [ -f "$file" ]; then
    echo "✅ $file exists"
  else
    echo "❌ $file missing"
    exit 1
  fi
done

# Check CLI binary
if [ -f "cli/financial-calculators-hub" ]; then
  echo "✅ CLI binary exists"
else
  echo "❌ CLI binary missing"
  exit 1
fi

# Check API binary (may not exist if not built yet)
if [ -f "api/financial-calculators-hub-api" ]; then
  echo "✅ API binary exists"
else
  echo "ℹ️  API binary not built (this is OK - built during startup)"
fi

# Check UI build
if [ -d "ui/dist" ] || [ -d "ui/build" ]; then
  echo "✅ UI build artifacts exist"
else
  echo "ℹ️  UI not built (this is OK - built during startup)"
fi

echo "✅ Structure checks completed"
