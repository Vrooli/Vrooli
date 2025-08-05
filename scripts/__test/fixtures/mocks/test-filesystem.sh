#!/usr/bin/env bash
# Simple manual test for filesystem mock

set -e

# Set up test environment
export MOCK_LOG_DIR="${TMPDIR:-/tmp}/filesystem-test-$$"
mkdir -p "$MOCK_LOG_DIR"

echo "Loading filesystem mock..."
source ./filesystem-simple.bash

echo "Testing basic operations..."

# Test 1: Create file
echo "Test 1: Creating file..."
mock::fs::create_file "/test.txt" "Hello World"
echo "File created"

# Test 2: Check file exists
echo "Test 2: Checking file exists..."
if [[ -n "${MOCK_FS_FILES[/test.txt]}" ]]; then
    echo "File exists in state: ${MOCK_FS_FILES[/test.txt]}"
else
    echo "ERROR: File not found in state"
    exit 1
fi

# Test 3: Get file content
echo "Test 3: Getting file content..."
content=$(mock::fs::get::file_content "/test.txt")
echo "Content: '$content'"

# Test 4: Test cat command
echo "Test 4: Testing cat command..."
cat_output=$(cat "/test.txt")
echo "Cat output: '$cat_output'"

# Test 5: Test ls command
echo "Test 5: Testing ls command..."
mock::fs::create_directory "/mydir"
mock::fs::create_file "/mydir/file1.txt" "content1"
ls_output=$(ls "/mydir")
echo "LS output: '$ls_output'"

echo "All tests passed!"

# Clean up
rm -rf "$MOCK_LOG_DIR"