#!/usr/bin/env bash
# Debug filesystem mock issues

set -e

# Set up test environment
export MOCK_LOG_DIR="${TMPDIR:-/tmp}/filesystem-test-$$"
mkdir -p "$MOCK_LOG_DIR"

echo "Loading filesystem mock..."
source ./filesystem.sh

echo "=== Testing mkdir issue ==="
echo "Root dir exists? ${MOCK_FS_FILES[/]}"
mkdir "/newdir"
echo "After mkdir, newdir exists? ${MOCK_FS_FILES[/newdir]}"

echo "=== Testing cp issue ==="
mock::fs::create_file "/source.txt" "content"
echo "Source exists? ${MOCK_FS_FILES[/source.txt]}"
cp "/source.txt" "/dest.txt"
echo "Dest exists? ${MOCK_FS_FILES[/dest.txt]}"

echo "=== Testing rm issue ==="
echo "Source before rm? ${MOCK_FS_FILES[/source.txt]}"
rm "/source.txt"
echo "Source after rm? ${MOCK_FS_FILES[/source.txt]}"

echo "=== Dumping all files ==="
for path in "${!MOCK_FS_FILES[@]}"; do
    echo "  $path: ${MOCK_FS_FILES[$path]}"
done