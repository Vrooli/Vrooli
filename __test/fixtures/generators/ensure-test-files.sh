#!/usr/bin/env bash
# Ensure Test Files Exist
# Automatically generates test files on-demand for tests
#
# Usage:
#   source ensure-test-files.sh
#   ensure_stress_files    # Generates stress test files if missing
#   ensure_edge_files      # Generates edge case files if missing

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GENERATOR_SCRIPT="${APP_ROOT}/__test/fixtures/generators/generate-stress-files.sh"

# Locations
STRESS_DIR="${APP_ROOT}/__test/fixtures/negative-tests/stress"
EDGE_DIR="${APP_ROOT}/__test/fixtures/documents/edge_cases"

# Track what we've generated this session
declare -gA GENERATED_FILES=()

# Ensure stress test files exist
ensure_stress_files() {
    local force="${1:-false}"
    
    # Check if many_files directory exists and has files
    if [[ ! -d "$STRESS_DIR/many_files" ]] || [[ -z "$(ls -A "$STRESS_DIR/many_files" 2>/dev/null)" ]] || [[ "$force" == "true" ]]; then
        echo "[TEST] Generating stress test files..."
        "$GENERATOR_SCRIPT" many-files "$STRESS_DIR/many_files" 100
        GENERATED_FILES["many_files"]=1
    fi
    
    # Check for other stress files
    if [[ ! -f "$STRESS_DIR/many_columns.csv" ]] || [[ "$force" == "true" ]]; then
        # Generate CSV with many columns
        python3 -c "
import csv
with open('$STRESS_DIR/many_columns.csv', 'w', newline='') as f:
    writer = csv.writer(f)
    headers = [f'col_{i}' for i in range(1000)]
    writer.writerow(headers)
    writer.writerow(['data'] * 1000)
" 2>/dev/null || echo "col1,col2,col3" > "$STRESS_DIR/many_columns.csv"
        GENERATED_FILES["many_columns"]=1
    fi
    
    if [[ ! -f "$STRESS_DIR/changing_content.txt" ]] || [[ "$force" == "true" ]]; then
        cat > "$STRESS_DIR/changing_content.txt" << EOF
This file simulates content that might change during processing.
Version: 1
Timestamp: $(date)
Note: In real tests, this could be updated by another process.
EOF
        GENERATED_FILES["changing_content"]=1
    fi
}

# Ensure edge case files exist
ensure_edge_files() {
    local force="${1:-false}"
    
    # Generate huge_text.txt if missing
    if [[ ! -f "$EDGE_DIR/huge_text.txt" ]] || [[ "$force" == "true" ]]; then
        echo "[TEST] Generating huge_text.txt..."
        "$GENERATOR_SCRIPT" huge-text "$EDGE_DIR/huge_text.txt" 10
        GENERATED_FILES["huge_text"]=1
    fi
    
    # Generate other edge case files if missing
    if [[ ! -f "$EDGE_DIR/binary_disguised.pdf" ]] || [[ "$force" == "true" ]]; then
        "$GENERATOR_SCRIPT" binary "$EDGE_DIR/binary_disguised.pdf" 100
        GENERATED_FILES["binary_disguised"]=1
    fi
    
    if [[ ! -f "$EDGE_DIR/corrupted.json" ]] || [[ "$force" == "true" ]]; then
        "$GENERATOR_SCRIPT" corrupted "$EDGE_DIR"
        GENERATED_FILES["corrupted_files"]=1
    fi
    
    if [[ ! -f "$EDGE_DIR/unicode_stress_test.txt" ]] || [[ "$force" == "true" ]]; then
        "$GENERATOR_SCRIPT" unicode "$EDGE_DIR/unicode_stress_test.txt"
        GENERATED_FILES["unicode_stress"]=1
    fi
    
    if [[ ! -f "$EDGE_DIR/long_single_line.txt" ]] || [[ "$force" == "true" ]]; then
        "$GENERATOR_SCRIPT" long-line "$EDGE_DIR/long_single_line.txt" 1000000
        GENERATED_FILES["long_line"]=1
    fi
    
    if [[ ! -f "$EDGE_DIR/network_timeout.url" ]] || [[ "$force" == "true" ]]; then
        "$GENERATOR_SCRIPT" network-timeout "$EDGE_DIR/network_timeout.url"
        GENERATED_FILES["network_timeout"]=1
    fi
}

# Ensure all test files exist
ensure_all_test_files() {
    ensure_stress_files "$1"
    ensure_edge_files "$1"
}

# Clean up generated files (call at test end if desired)
cleanup_generated_files() {
    if [[ ${#GENERATED_FILES[@]} -gt 0 ]]; then
        echo "[TEST] Cleaning up generated test files..."
        "$GENERATOR_SCRIPT" clean
        GENERATED_FILES=()
    fi
}

# Register cleanup on exit (optional - uncomment if desired)
# trap cleanup_generated_files EXIT

# Export functions for use in tests
export -f ensure_stress_files
export -f ensure_edge_files
export -f ensure_all_test_files
export -f cleanup_generated_files