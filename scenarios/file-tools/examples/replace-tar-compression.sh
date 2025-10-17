#!/usr/bin/env bash
# Example: Replace tar compression with file-tools API
# Use case: data-backup-manager, document-manager, any scenario doing backups

set -euo pipefail

# Configuration
FILE_TOOLS_PORT="${FILE_TOOLS_PORT:-15458}"  # Default from port registry
API_URL="http://localhost:${FILE_TOOLS_PORT}"

# ============================================================================
# BEFORE: Custom tar-based compression (typical pattern in scenarios)
# ============================================================================
compress_with_tar() {
    local source_path="$1"
    local output_file="$2"

    echo "❌ OLD WAY: Using tar directly"
    tar -czf "$output_file" "$source_path"

    # No built-in integrity check
    # No progress tracking
    # No error recovery
    # Manual compression level tuning required
}

# ============================================================================
# AFTER: Using file-tools API for compression
# ============================================================================
compress_with_file_tools() {
    local source_path="$1"
    local output_file="$2"

    echo "✅ NEW WAY: Using file-tools API"

    # Compress with automatic integrity verification
    response=$(curl -sf "${API_URL}/api/v1/files/compress" \
        -H "Content-Type: application/json" \
        -d "{
            \"files\": [\"$source_path\"],
            \"archive_format\": \"gzip\",
            \"output_path\": \"$output_file\",
            \"compression_level\": 6,
            \"options\": {
                \"preserve_permissions\": true,
                \"exclude_patterns\": [\".git\", \"node_modules\", \"*.tmp\"]
            }
        }")

    # Parse response for operation details
    operation_id=$(echo "$response" | jq -r '.operation_id')
    original_size=$(echo "$response" | jq -r '.original_size_bytes')
    compressed_size=$(echo "$response" | jq -r '.compressed_size_bytes')
    compression_ratio=$(echo "$response" | jq -r '.compression_ratio')
    checksum=$(echo "$response" | jq -r '.checksum')

    echo "📦 Compression complete:"
    echo "   Operation ID: $operation_id"
    echo "   Original: $(numfmt --to=iec "$original_size")"
    echo "   Compressed: $(numfmt --to=iec "$compressed_size")"
    echo "   Ratio: ${compression_ratio}x"
    echo "   Checksum: $checksum"

    # Optional: Add integrity verification
    verify_checksum "$output_file" "$checksum"
}

verify_checksum() {
    local file="$1"
    local expected_checksum="$2"

    echo "🔐 Verifying archive integrity..."

    response=$(curl -sf "${API_URL}/api/v1/files/checksum" \
        -H "Content-Type: application/json" \
        -d "{
            \"files\": [\"$file\"],
            \"algorithm\": \"sha256\"
        }")

    actual_checksum=$(echo "$response" | jq -r '.results[0].checksums.sha256')

    if [[ "$actual_checksum" == "$expected_checksum" ]]; then
        echo "✅ Checksum verified: Archive integrity confirmed"
    else
        echo "❌ Checksum mismatch: Archive may be corrupted"
        return 1
    fi
}

# ============================================================================
# DEMO: Compare both approaches
# ============================================================================
main() {
    local test_dir="/tmp/file-tools-example"
    local test_data="${test_dir}/test-data"
    local tar_output="${test_dir}/backup-tar.tar.gz"
    local ft_output="${test_dir}/backup-file-tools.tar.gz"

    # Create test data
    mkdir -p "$test_data"
    for i in {1..100}; do
        echo "Test data $i" > "${test_data}/file${i}.txt"
    done

    echo "📊 Compression Comparison"
    echo "========================="
    echo ""

    # Test tar approach
    echo "1. Traditional tar compression:"
    time compress_with_tar "$test_data" "$tar_output"
    echo ""

    # Test file-tools approach
    echo "2. File-tools API compression:"
    time compress_with_file_tools "$test_data" "$ft_output"
    echo ""

    # Compare results
    echo "📈 Results:"
    echo "   tar size: $(du -h "$tar_output" | cut -f1)"
    echo "   file-tools size: $(du -h "$ft_output" | cut -f1)"
    echo ""
    echo "✨ Benefits of file-tools approach:"
    echo "   ✓ Automatic integrity verification"
    echo "   ✓ Detailed operation metadata"
    echo "   ✓ Standardized error handling"
    echo "   ✓ Progress tracking support"
    echo "   ✓ Configurable compression options"
    echo "   ✓ Built-in checksum calculation"

    # Cleanup
    rm -rf "$test_dir"
}

# Run demo if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
