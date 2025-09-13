#!/bin/bash
# CLI test runner - orchestrates all BATS tests
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🔧 Running CLI BATS tests..."

# Check if BATS is available
if ! command -v bats >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  BATS is not installed${NC}"
    echo "   Install with: npm install -g bats"
    echo "   Or on Ubuntu: sudo apt-get install bats"
    exit 1
fi

# Get the CLI directory (where BATS files should be)
CLI_DIR="${VROOLI_ROOT:-$(pwd)}/scenarios/visited-tracker/cli"
if [ ! -d "$CLI_DIR" ]; then
    echo -e "${RED}❌ CLI directory not found: $CLI_DIR${NC}"
    exit 1
fi

echo "🔍 Looking for BATS files in: $CLI_DIR"

# Find all BATS files
bats_files=()
while IFS= read -r -d '' file; do
    bats_files+=("$file")
done < <(find "$CLI_DIR" -name "*.bats" -type f -print0 2>/dev/null)

if [ ${#bats_files[@]} -eq 0 ]; then
    echo -e "${YELLOW}ℹ️  No BATS files found in $CLI_DIR${NC}"
    echo -e "${BLUE}💡 Creating basic BATS test structure...${NC}"
    
    # Create a basic BATS test file
    cat > "$CLI_DIR/visited-tracker.bats" << 'EOF'
#!/usr/bin/env bats

# Basic CLI tests for visited-tracker

setup() {
    # Ensure CLI is available
    if ! command -v visited-tracker >/dev/null 2>&1; then
        skip "visited-tracker CLI not installed"
    fi
}

@test "CLI help command works" {
    run visited-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visited Tracker CLI" ]]
}

@test "CLI version command works" {
    run visited-tracker version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visited-tracker CLI" ]]
}

@test "CLI status command works" {
    run visited-tracker status
    # Status might fail if service not running, but should not crash
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI help shows main commands" {
    run visited-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visit" ]]
    [[ "$output" =~ "sync" ]]
    [[ "$output" =~ "least-visited" ]]
    [[ "$output" =~ "most-stale" ]]
    [[ "$output" =~ "coverage" ]]
}

@test "CLI version shows correct format" {
    run visited-tracker version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v[0-9]+\.[0-9]+\.[0-9]+" ]]
}

@test "CLI handles unknown command gracefully" {
    run visited-tracker nonexistent-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "not found" ]] || [[ "$output" =~ "invalid" ]]
}

@test "CLI visit command shows usage when no args" {
    run visited-tracker visit
    # Should show usage or error message
    [ "$status" -ne 0 ] || [[ "$output" =~ "Usage" ]] || [[ "$output" =~ "usage" ]]
}

@test "CLI sync command shows usage when no args" {
    run visited-tracker sync
    # Should show usage or error message  
    [ "$status" -ne 0 ] || [[ "$output" =~ "Usage" ]] || [[ "$output" =~ "usage" ]]
}

# Integration tests (might fail if service not running)
@test "CLI least-visited works with service running" {
    # Skip if service is not running
    if ! curl -sf "http://localhost:${API_PORT:-20252}/health" >/dev/null 2>&1; then
        skip "visited-tracker service not running"
    fi
    
    run visited-tracker least-visited --limit 1 --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "files" ]]
}

@test "CLI coverage works with service running" {
    # Skip if service is not running
    if ! curl -sf "http://localhost:${API_PORT:-20252}/health" >/dev/null 2>&1; then
        skip "visited-tracker service not running"
    fi
    
    run visited-tracker coverage --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "coverage_percentage" ]]
}
EOF
    
    echo -e "${GREEN}✅ Created basic BATS test file: $CLI_DIR/visited-tracker.bats${NC}"
    bats_files=("$CLI_DIR/visited-tracker.bats")
fi

# Run all BATS files
test_count=0
failed_count=0
total_files=${#bats_files[@]}

echo ""
echo "🧪 Running $total_files BATS test files..."

for bats_file in "${bats_files[@]}"; do
    echo ""
    echo -e "${BLUE}📋 Running $(basename "$bats_file")...${NC}"
    
    if bats "$bats_file" --tap; then
        echo -e "${GREEN}✅ $(basename "$bats_file") passed${NC}"
        ((test_count++))
    else
        echo -e "${RED}❌ $(basename "$bats_file") failed${NC}"
        ((failed_count++))
        ((test_count++))
    fi
done

# Summary
echo ""
echo "📊 CLI Test Summary:"
echo "   Test files run: $test_count"
echo "   Test files passed: $((test_count - failed_count))"
echo "   Test files failed: $failed_count"

if [ $failed_count -eq 0 ]; then
    echo -e "${GREEN}✅ All $test_count CLI test suites passed${NC}"
    exit 0
else
    echo -e "${RED}❌ $failed_count of $test_count CLI test suites failed${NC}"
    exit 1
fi