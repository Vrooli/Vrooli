#!/usr/bin/env bash
# Generate Stress Test Files On-Demand
# Creates large files and many small files dynamically for testing
#
# Usage:
#   ./generate-stress-files.sh [command] [options]
#
# Commands:
#   many-files <dir> <count>  - Generate many small files
#   huge-text <file> <size>   - Generate large text file
#   all                       - Generate all standard test files
#   clean                     - Remove generated files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${BASH_SOURCE[0]%/*}"

# Default locations
DEFAULT_STRESS_DIR="${APP_ROOT}/__test/fixtures/negative-tests/stress"
DEFAULT_EDGE_DIR="${APP_ROOT}/__test/fixtures/documents/edge_cases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Generate many small files
generate_many_files() {
    local target_dir="${1:-$DEFAULT_STRESS_DIR/many_files}"
    local count="${2:-100}"
    
    log_info "Generating $count small files in $target_dir..."
    
    # Create directory if it doesn't exist
    mkdir -p "$target_dir"
    
    # Generate files
    for i in $(seq 1 "$count"); do
        echo "File $i" > "$target_dir/file_$i.txt"
        
        # Progress indicator every 10 files
        if (( i % 10 == 0 )); then
            echo -n "."
        fi
    done
    echo  # New line after dots
    
    log_info "✓ Generated $count files in $target_dir"
}

# Generate huge text file
generate_huge_text() {
    local target_file="${1:-$DEFAULT_EDGE_DIR/huge_text.txt}"
    local size_mb="${2:-10}"
    
    log_info "Generating ${size_mb}MB text file: $target_file..."
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$target_file")"
    
    # Generate file with repeated pattern
    # Using dd for efficiency with large files
    local size_bytes=$((size_mb * 1024 * 1024))
    
    # Create a 1KB pattern block
    local pattern="The quick brown fox jumps over the lazy dog. "
    local pattern_block=""
    for i in {1..22}; do
        pattern_block="${pattern_block}${pattern}"
    done
    # pattern_block is now ~1KB
    
    # Write the pattern repeatedly
    {
        local blocks=$((size_bytes / 1024))
        for ((i=0; i<blocks; i++)); do
            echo -n "$pattern_block"
            # Progress indicator every 100 blocks (~100KB)
            if (( i % 100 == 0 )) && (( i > 0 )); then
                echo -n "." >&2
            fi
        done
    } > "$target_file"
    echo >&2  # New line after dots
    
    local actual_size=$(du -h "$target_file" | cut -f1)
    log_info "✓ Generated $actual_size file: $target_file"
}

# Generate binary file
generate_binary_file() {
    local target_file="${1:-$DEFAULT_EDGE_DIR/binary_disguised.pdf}"
    local size_kb="${2:-100}"
    
    log_info "Generating ${size_kb}KB binary file: $target_file..."
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$target_file")"
    
    # Generate random binary data
    dd if=/dev/urandom of="$target_file" bs=1024 count="$size_kb" 2>/dev/null
    
    log_info "✓ Generated binary file: $target_file"
}

# Generate corrupted files
generate_corrupted_files() {
    local target_dir="${1:-$DEFAULT_EDGE_DIR}"
    
    log_info "Generating corrupted files in $target_dir..."
    
    mkdir -p "$target_dir"
    
    # Corrupted JSON
    cat > "$target_dir/corrupted.json" << 'EOF'
{
  "valid_start": true,
  "but_then": "missing closing brace and quote
EOF
    
    # Invalid XML
    cat > "$target_dir/invalid_xml.xml" << 'EOF'
<?xml version="1.0"?>
<root>
  <unclosed_tag>
  <another>value</wrong_closing>
</root>
EOF
    
    # Malformed CSV
    cat > "$target_dir/malformed.csv" << 'EOF'
header1,header2,header3
value1,value2
value3,value4,value5,extra_value
"unclosed quote,value
EOF
    
    # Empty PDF
    touch "$target_dir/empty.pdf"
    
    log_info "✓ Generated corrupted files"
}

# Generate unicode stress test file
generate_unicode_stress() {
    local target_file="${1:-$DEFAULT_EDGE_DIR/unicode_stress_test.txt}"
    
    log_info "Generating unicode stress test: $target_file..."
    
    mkdir -p "$(dirname "$target_file")"
    
    cat > "$target_file" << 'EOF'
Unicode Stress Test File
========================

Basic Unicode: café, naïve, résumé
Emojis: 🚀 🎉 🔥 💯 🤖 🦄 🌈
Mathematical: ∑ ∏ ∫ ∞ √ ≈ ≠ ≤ ≥
Greek: α β γ δ ε ζ η θ ι κ λ μ
Cyrillic: А Б В Г Д Е Ж З И Й К Л
Arabic: ا ب ت ث ج ح خ د ذ ر ز س
Chinese: 你好世界 测试文本 数据处理
Japanese: こんにちは テスト データ
Korean: 안녕하세요 테스트 데이터

Special Cases:
‮Right-to-left override test‬
Z̴̡͖̈́a̸̛̰̐l̵̨̉g̸̨̐o̵̧̐ ̸̨̐t̵̡̉ę̸̧̐x̵̨̉t̸̡̐
​Zero-width space​ between words
EOF
    
    log_info "✓ Generated unicode stress test"
}

# Generate long single line file
generate_long_line() {
    local target_file="${1:-$DEFAULT_EDGE_DIR/long_single_line.txt}"
    local length="${2:-1000000}"
    
    log_info "Generating file with $length character line..."
    
    mkdir -p "$(dirname "$target_file")"
    
    # Use printf for efficiency
    printf "%${length}s" | tr ' ' 'A' > "$target_file"
    
    log_info "✓ Generated long line file"
}

# Generate network timeout simulation file
generate_network_timeout() {
    local target_file="${1:-$DEFAULT_EDGE_DIR/network_timeout.url}"
    
    log_info "Generating network timeout test file..."
    
    mkdir -p "$(dirname "$target_file")"
    
    cat > "$target_file" << 'EOF'
[InternetShortcut]
URL=http://httpstat.us/504?sleep=30000
Comment=This URL simulates a network timeout for testing
EOF
    
    log_info "✓ Generated network timeout file"
}

# Generate all standard test files
generate_all() {
    log_info "Generating all standard test files..."
    echo
    
    generate_many_files
    echo
    generate_huge_text
    echo
    generate_binary_file
    echo
    generate_corrupted_files
    echo
    generate_unicode_stress
    echo
    generate_long_line
    echo
    generate_network_timeout
    
    echo
    log_info "✅ All test files generated successfully!"
}

# Clean generated files
clean_generated() {
    log_info "Cleaning generated test files..."
    
    # Remove many_files directory
    if [[ -d "$DEFAULT_STRESS_DIR/many_files" ]]; then
        rm -rf "$DEFAULT_STRESS_DIR/many_files"
        log_info "✓ Removed many_files directory"
    fi
    
    # Remove large generated files
    local files_to_remove=(
        "$DEFAULT_EDGE_DIR/huge_text.txt"
        "$DEFAULT_EDGE_DIR/binary_disguised.pdf"
        "$DEFAULT_EDGE_DIR/corrupted.json"
        "$DEFAULT_EDGE_DIR/invalid_xml.xml"
        "$DEFAULT_EDGE_DIR/malformed.csv"
        "$DEFAULT_EDGE_DIR/empty.pdf"
        "$DEFAULT_EDGE_DIR/unicode_stress_test.txt"
        "$DEFAULT_EDGE_DIR/long_single_line.txt"
        "$DEFAULT_EDGE_DIR/network_timeout.url"
    )
    
    for file in "${files_to_remove[@]}"; do
        if [[ -f "$file" ]]; then
            rm -f "$file"
            log_info "✓ Removed $(basename "$file")"
        fi
    done
    
    log_info "✅ Cleanup complete"
}

# Show usage
show_usage() {
    cat << EOF
Generate Test Files On-Demand
==============================

Usage: $0 [command] [options]

Commands:
    many-files [dir] [count]     Generate many small files (default: 100 files)
    huge-text [file] [size_mb]   Generate large text file (default: 10MB)
    binary [file] [size_kb]      Generate binary file (default: 100KB)
    corrupted [dir]              Generate corrupted files
    unicode [file]               Generate unicode stress test
    long-line [file] [length]    Generate file with long line
    network-timeout [file]       Generate network timeout test
    all                          Generate all standard test files
    clean                        Remove all generated files
    help                         Show this help message

Examples:
    $0 many-files                      # Generate 100 files in default location
    $0 many-files /tmp/test 500        # Generate 500 files in /tmp/test
    $0 huge-text /tmp/big.txt 50       # Generate 50MB text file
    $0 all                              # Generate all test files
    $0 clean                            # Remove generated files

EOF
}

# Main command dispatcher
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        many-files)
            generate_many_files "$@"
            ;;
        huge-text)
            generate_huge_text "$@"
            ;;
        binary)
            generate_binary_file "$@"
            ;;
        corrupted)
            generate_corrupted_files "$@"
            ;;
        unicode)
            generate_unicode_stress "$@"
            ;;
        long-line)
            generate_long_line "$@"
            ;;
        network-timeout)
            generate_network_timeout "$@"
            ;;
        all)
            generate_all
            ;;
        clean)
            clean_generated
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"