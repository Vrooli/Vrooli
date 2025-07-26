#!/usr/bin/env bash
# Test runner for Node-RED management script tests
# Provides convenient ways to run the bats test suite

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default options
VERBOSE=false
PARALLEL=false
FILTER=""
OUTPUT_FORMAT="pretty"
JOBS=4

show_help() {
    cat << EOF
Node-RED Test Runner

Usage: $0 [OPTIONS] [TEST_CATEGORIES...]

Options:
    -h, --help          Show this help message
    -v, --verbose       Enable verbose output
    -p, --parallel      Run tests in parallel
    -j, --jobs NUM      Number of parallel jobs (default: 4)
    -f, --filter PATTERN   Run only tests matching pattern
    -o, --output FORMAT    Output format: pretty, tap, junit (default: pretty)
    --list             List all available tests
    --check            Check test environment setup

Test Categories:
    all                Run all tests (default)
    unit               Run all unit tests (config/ and lib/)
    integration        Run integration tests (manage.bats)
    config             Run configuration tests (config/)
    lib                Run all library tests (lib/)
    defaults           Run config/defaults.bats
    messages           Run config/messages.bats
    common             Run lib/common.bats
    docker             Run lib/docker.bats
    install            Run lib/install.bats
    status             Run lib/status.bats
    api                Run lib/api.bats
    testing            Run lib/testing.bats
    manage             Run manage.bats

Examples:
    $0                          # Run all tests
    $0 unit                     # Run all unit tests
    $0 lib                      # Run all library tests
    $0 config                   # Run config tests only
    $0 common docker            # Run specific test files
    $0 -v -p                    # Verbose parallel execution
    $0 -f "install"             # Run tests matching "install"
    $0 --output junit > results.xml  # Generate JUnit XML

Test Structure (following n8n pattern):
    config/defaults.bats        # Tests for config/defaults.sh
    config/messages.bats        # Tests for config/messages.sh
    lib/common.bats            # Tests for lib/common.sh
    lib/docker.bats            # Tests for lib/docker.sh
    lib/install.bats           # Tests for lib/install.sh
    lib/status.bats            # Tests for lib/status.sh
    lib/api.bats               # Tests for lib/api.sh
    lib/testing.bats           # Tests for lib/testing.sh
    manage.bats                # Tests for manage.sh (integration)

EOF
}

check_bats() {
    if ! command -v bats >/dev/null 2>&1; then
        echo -e "${RED}Error: bats is not installed${NC}"
        echo "Install with:"
        echo "  Ubuntu/Debian: sudo apt-get install bats"
        echo "  macOS: brew install bats-core"
        echo "  Or from source: https://github.com/bats-core/bats-core"
        exit 1
    fi
    
    echo -e "${GREEN}✓ bats found: $(bats --version)${NC}"
}

check_environment() {
    echo "Checking test environment..."
    
    check_bats
    
    # Check test files exist
    local test_files=(
        "test-fixtures/test_helper.bash"
        "config/defaults.bats"
        "config/messages.bats" 
        "lib/common.bats"
        "lib/docker.bats"
        "lib/install.bats"
        "lib/status.bats"
        "lib/api.bats"
        "lib/testing.bats"
        "manage.bats"
    )
    
    for file in "${test_files[@]}"; do
        if [[ -f "$SCRIPT_DIR/$file" ]]; then
            echo -e "${GREEN}✓ $file${NC}"
        else
            echo -e "${RED}✗ $file (missing)${NC}"
        fi
    done
    
    # Check source files exist
    local source_files=(
        "config/defaults.sh"
        "config/messages.sh"
        "lib/common.sh"
        "lib/docker.sh" 
        "lib/install.sh"
        "lib/status.sh"
        "lib/api.sh"
        "lib/testing.sh"
        "manage.sh"
    )
    
    for file in "${source_files[@]}"; do
        if [[ -f "$SCRIPT_DIR/$file" ]]; then
            echo -e "${GREEN}✓ $file${NC}"
        else
            echo -e "${YELLOW}⚠ $file (missing)${NC}"
        fi
    done
    
    echo -e "${GREEN}Environment check complete${NC}"
}

list_tests() {
    echo "Available test files:"
    echo
    
    # Config tests
    for test_file in "$SCRIPT_DIR/config"/*.bats; do
        if [[ -f "$test_file" ]]; then
            local basename=$(basename "$test_file" .bats)
            echo -e "${BLUE}$basename (config/$basename.bats)${NC}"
            grep -E "^@test" "$test_file" | sed 's/@test "/  - /' | sed 's/" {$//' || true
            echo
        fi
    done
    
    # Library tests
    for test_file in "$SCRIPT_DIR/lib"/*.bats; do
        if [[ -f "$test_file" ]]; then
            local basename=$(basename "$test_file" .bats)
            echo -e "${BLUE}$basename (lib/$basename.bats)${NC}"
            grep -E "^@test" "$test_file" | sed 's/@test "/  - /' | sed 's/" {$//' || true
            echo
        fi
    done
    
    # Main script tests
    if [[ -f "$SCRIPT_DIR/manage.bats" ]]; then
        echo -e "${BLUE}manage (manage.bats)${NC}"
        grep -E "^@test" "$SCRIPT_DIR/manage.bats" | sed 's/@test "/  - /' | sed 's/" {$//' || true
        echo
    fi
}

get_test_files() {
    local category="$1"
    local files=()
    
    case "$category" in
        "all")
            files+=(
                "$SCRIPT_DIR/config"/*.bats
                "$SCRIPT_DIR/lib"/*.bats
                "$SCRIPT_DIR/manage.bats"
            )
            ;;
        "unit")
            files+=(
                "$SCRIPT_DIR/config"/*.bats
                "$SCRIPT_DIR/lib"/*.bats
            )
            ;;
        "integration"|"manage")
            files+=("$SCRIPT_DIR/manage.bats")
            ;;
        "config")
            files+=("$SCRIPT_DIR/config"/*.bats)
            ;;
        "lib")
            files+=("$SCRIPT_DIR/lib"/*.bats)
            ;;
        "defaults"|"messages")
            local test_file="$SCRIPT_DIR/config/${category}.bats"
            if [[ -f "$test_file" ]]; then
                files+=("$test_file")
            else
                echo -e "${RED}Error: Test file '$category.bats' not found in config/${NC}"
                return 1
            fi
            ;;
        "common"|"docker"|"install"|"status"|"api"|"testing")
            local test_file="$SCRIPT_DIR/lib/${category}.bats"
            if [[ -f "$test_file" ]]; then
                files+=("$test_file")
            else
                echo -e "${RED}Error: Test file '$category.bats' not found in lib/${NC}"
                return 1
            fi
            ;;
        *)
            echo -e "${RED}Error: Unknown test category '$category'${NC}"
            echo "Available categories: all, unit, integration, config, lib, defaults, messages, common, docker, install, status, api, testing, manage"
            return 1
            ;;
    esac
    
    # Filter out non-existent files and expand globs
    local existing_files=()
    for file in "${files[@]}"; do
        if [[ -f "$file" ]]; then
            existing_files+=("$file")
        fi
    done
    
    printf '%s\n' "${existing_files[@]}"
}

run_bats() {
    local files=("$@")
    local bats_args=()
    
    if [[ ${#files[@]} -eq 0 ]]; then
        echo -e "${RED}Error: No test files to run${NC}"
        return 1
    fi
    
    # Build bats arguments
    if [[ "$VERBOSE" == "true" ]]; then
        bats_args+=(--verbose-run --print-output-on-failure)
    fi
    
    if [[ "$PARALLEL" == "true" ]]; then
        bats_args+=(--jobs "$JOBS")
    fi
    
    if [[ -n "$FILTER" ]]; then
        bats_args+=(--filter "$FILTER")
    fi
    
    case "$OUTPUT_FORMAT" in
        "tap")
            bats_args+=(--formatter tap)
            ;;
        "junit")
            bats_args+=(--formatter junit)
            ;;
        "pretty"|*)
            # Default pretty format
            ;;
    esac
    
    echo -e "${BLUE}Running tests...${NC}"
    echo "Files: ${files[*]}"
    if [[ ${#bats_args[@]} -gt 0 ]]; then
        echo "Args: ${bats_args[*]}"
    fi
    echo
    
    # Change to script directory for consistent relative paths
    cd "$SCRIPT_DIR"
    
    # Run bats
    if bats "${bats_args[@]}" "${files[@]}"; then
        echo -e "${GREEN}✓ All tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        return 1
    fi
}

main() {
    local categories=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -p|--parallel)
                PARALLEL=true
                shift
                ;;
            -j|--jobs)
                JOBS="$2"
                shift 2
                ;;
            -f|--filter)
                FILTER="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            --list)
                list_tests
                exit 0
                ;;
            --check)
                check_environment
                exit 0
                ;;
            all|unit|integration|config|lib|defaults|messages|common|docker|install|status|api|testing|manage)
                categories+=("$1")
                shift
                ;;
            *.bats)
                # Direct file specification
                if [[ -f "$1" ]]; then
                    run_bats "$1"
                    exit $?
                else
                    echo -e "${RED}Error: Test file not found: $1${NC}"
                    exit 1
                fi
                ;;
            *)
                echo -e "${RED}Error: Unknown option '$1'${NC}"
                show_help
                exit 1
                ;;
        esac
    done
    
    # Check environment first
    check_bats
    
    # Default to all tests if no categories specified
    if [[ ${#categories[@]} -eq 0 ]]; then
        categories=("all")
    fi
    
    # Collect all test files from categories
    local all_files=()
    for category in "${categories[@]}"; do
        echo -e "${BLUE}Collecting tests for category: $category${NC}"
        
        local category_files
        if ! category_files=$(get_test_files "$category"); then
            exit 1
        fi
        
        # Add files to array
        while IFS= read -r file; do
            if [[ -n "$file" && ! " ${all_files[*]} " =~ " $file " ]]; then
                all_files+=("$file")
            fi
        done <<< "$category_files"
    done
    
    # Run tests
    if [[ ${#all_files[@]} -gt 0 ]]; then
        run_bats "${all_files[@]}"
    else
        echo -e "${RED}Error: No test files found${NC}"
        exit 1
    fi
}

# Run main function
main "$@"