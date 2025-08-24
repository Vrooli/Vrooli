#!/usr/bin/env bash
# Mock Migration Helper Script
# 
# Assists with migrating from legacy mocks to Tier 2 architecture
# Usage: ./migrate.sh [command] [options]

set -euo pipefail

# Configuration
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test/mocks"
TIER2_DIR="${SCRIPT_DIR}/tier2"
LEGACY_DIR="${SCRIPT_DIR}/../mocks-legacy"
PROJECT_ROOT="${APP_ROOT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# === Helper Functions ===
print_success() { echo -e "${GREEN}✓${NC} $1"; }
print_error() { echo -e "${RED}✗${NC} $1" >&2; }
print_warning() { echo -e "${YELLOW}⚠${NC} $1"; }
print_info() { echo -e "  $1"; }

# === Check Status ===
check_status() {
    echo "=== Mock Migration Status ==="
    echo ""
    
    # Count files
    local tier2_count=$(find "$TIER2_DIR" -name "*.sh" -type f 2>/dev/null | wc -l)
    local legacy_count=$(find "$LEGACY_DIR" -name "*.sh" -type f 2>/dev/null | wc -l)
    
    echo "File Counts:"
    print_info "Tier 2 mocks: $tier2_count"
    print_info "Legacy mocks: $legacy_count"
    echo ""
    
    # Check permissions
    local exec_count=$(find "$TIER2_DIR" -name "*.sh" -type f -executable 2>/dev/null | wc -l)
    if [[ $exec_count -eq $tier2_count ]]; then
        print_success "All Tier 2 mocks have execute permissions"
    else
        print_warning "$(($tier2_count - $exec_count)) Tier 2 mocks lack execute permissions"
    fi
    echo ""
    
    # Check for duplicates
    echo "Coverage Analysis:"
    local both_count=0
    local tier2_only=0
    local legacy_only=0
    
    # Check each mock
    for mock in $(cd "$TIER2_DIR" 2>/dev/null && ls *.sh 2>/dev/null | sed 's/.sh$//' | sort); do
        if [[ -f "$LEGACY_DIR/${mock}.sh" ]]; then
            ((both_count++))
        else
            ((tier2_only++))
            print_info "[TIER2 ONLY] $mock"
        fi
    done
    
    for mock in $(cd "$LEGACY_DIR" 2>/dev/null && ls *.sh 2>/dev/null | sed 's/.sh$//' | sort); do
        if [[ ! -f "$TIER2_DIR/${mock}.sh" ]]; then
            ((legacy_only++))
            print_info "[LEGACY ONLY] $mock"
        fi
    done
    
    echo ""
    echo "Summary:"
    print_info "Both versions available: $both_count"
    print_info "Tier 2 only: $tier2_only"
    print_info "Legacy only: $legacy_only"
}

# === Test Mocks ===
test_mocks() {
    local type="${1:-tier2}"
    echo "=== Testing $type Mocks ==="
    echo ""
    
    local dir=""
    case "$type" in
        tier2) dir="$TIER2_DIR" ;;
        legacy) dir="$LEGACY_DIR" ;;
        *) print_error "Unknown type: $type"; exit 1 ;;
    esac
    
    # Test each mock
    for mock_file in "$dir"/*.sh; do
        [[ ! -f "$mock_file" ]] && continue
        local mock_name=$(basename "$mock_file" .sh)
        
        echo -n "Testing $mock_name: "
        
        # Try to source and run basic test
        if bash -c "source '$mock_file' 2>/dev/null && declare -F | grep -q test_${mock_name}_connection" 2>/dev/null; then
            if bash -c "source '$mock_file' && test_${mock_name}_connection" >/dev/null 2>&1; then
                print_success "connection test passed"
            else
                print_error "connection test failed"
            fi
        else
            print_warning "no test function found"
        fi
    done
}

# === Find Usage ===
find_usage() {
    echo "=== Finding Mock Usage in Codebase ==="
    echo ""
    
    # Find references to legacy mocks
    echo "Legacy mock references:"
    if rg -l "mocks-legacy" "$PROJECT_ROOT" --type sh 2>/dev/null | head -10; then
        local count=$(rg -l "mocks-legacy" "$PROJECT_ROOT" --type sh 2>/dev/null | wc -l)
        print_info "Found $count files referencing legacy mocks"
    else
        print_success "No references to legacy mocks found"
    fi
    echo ""
    
    # Find references to tier2 mocks
    echo "Tier 2 mock references:"
    if rg -l "mocks/tier2" "$PROJECT_ROOT" --type sh 2>/dev/null | head -10; then
        local count=$(rg -l "mocks/tier2" "$PROJECT_ROOT" --type sh 2>/dev/null | wc -l)
        print_info "Found $count files referencing Tier 2 mocks"
    else
        print_warning "No references to Tier 2 mocks found"
    fi
}

# === Validate Single Mock ===
validate_mock() {
    local mock_name="$1"
    echo "=== Validating Mock: $mock_name ==="
    echo ""
    
    local tier2_file="$TIER2_DIR/${mock_name}.sh"
    local legacy_file="$LEGACY_DIR/${mock_name}.sh"
    
    # Check existence
    [[ -f "$tier2_file" ]] && print_success "Tier 2 version exists" || print_error "Tier 2 version missing"
    [[ -f "$legacy_file" ]] && print_info "Legacy version exists" || print_info "No legacy version"
    
    # Check Tier 2 structure
    if [[ -f "$tier2_file" ]]; then
        echo ""
        echo "Tier 2 Structure Check:"
        
        # Check for required functions
        local required_functions=(
            "test_${mock_name}_connection"
            "test_${mock_name}_health"
            "test_${mock_name}_basic"
            "${mock_name}_mock_reset"
            "${mock_name}_mock_set_error"
        )
        
        for func in "${required_functions[@]}"; do
            if grep -q "^${func}()" "$tier2_file" || grep -q "^function ${func}" "$tier2_file"; then
                print_success "$func defined"
            else
                print_warning "$func not found"
            fi
        done
        
        # Check line count
        local lines=$(wc -l < "$tier2_file")
        if [[ $lines -le 600 ]]; then
            print_success "Line count: $lines (within 600 line target)"
        else
            print_warning "Line count: $lines (exceeds 600 line target)"
        fi
    fi
    
    # Size comparison
    if [[ -f "$tier2_file" ]] && [[ -f "$legacy_file" ]]; then
        echo ""
        echo "Size Comparison:"
        local tier2_lines=$(wc -l < "$tier2_file")
        local legacy_lines=$(wc -l < "$legacy_file")
        local reduction=$(( (legacy_lines - tier2_lines) * 100 / legacy_lines ))
        print_info "Legacy: $legacy_lines lines"
        print_info "Tier 2: $tier2_lines lines"
        if [[ $reduction -ge 40 ]]; then
            print_success "Reduction: ${reduction}% (target: 40-60%)"
        else
            print_warning "Reduction: ${reduction}% (below 40% target)"
        fi
    fi
}

# === Generate Report ===
generate_report() {
    local output_file="${1:-migration-report.md}"
    
    {
        echo "# Mock Migration Report"
        echo "Generated: $(date)"
        echo ""
        echo "## Executive Summary"
        
        local tier2_count=$(find "$TIER2_DIR" -name "*.sh" -type f 2>/dev/null | wc -l)
        local legacy_count=$(find "$LEGACY_DIR" -name "*.sh" -type f 2>/dev/null | wc -l)
        
        echo "- Tier 2 mocks: $tier2_count"
        echo "- Legacy mocks: $legacy_count"
        echo "- Migration progress: $(( tier2_count * 100 / (tier2_count + 4) ))%"
        echo ""
        
        echo "## Mock Status"
        echo ""
        echo "| Mock | Tier 2 | Legacy | Lines Saved | Reduction |"
        echo "|------|--------|--------|-------------|-----------|"
        
        for mock in $(ls "$TIER2_DIR"/*.sh "$LEGACY_DIR"/*.sh 2>/dev/null | xargs -n1 basename | sed 's/.sh$//' | sort -u); do
            local tier2_status="❌"
            local legacy_status="❌"
            local tier2_lines=0
            local legacy_lines=0
            local saved="-"
            local reduction="-"
            
            if [[ -f "$TIER2_DIR/${mock}.sh" ]]; then
                tier2_status="✅"
                tier2_lines=$(wc -l < "$TIER2_DIR/${mock}.sh")
            fi
            
            if [[ -f "$LEGACY_DIR/${mock}.sh" ]]; then
                legacy_status="✅"
                legacy_lines=$(wc -l < "$LEGACY_DIR/${mock}.sh")
            fi
            
            if [[ $tier2_lines -gt 0 ]] && [[ $legacy_lines -gt 0 ]]; then
                saved=$((legacy_lines - tier2_lines))
                reduction="$(( saved * 100 / legacy_lines ))%"
            fi
            
            echo "| $mock | $tier2_status | $legacy_status | $saved | $reduction |"
        done
        
        echo ""
        echo "## Next Steps"
        echo ""
        echo "1. Complete migration of remaining legacy-only mocks"
        echo "2. Update test files to use Tier 2 mocks"
        echo "3. Remove legacy mocks after verification"
        echo "4. Update documentation"
        
    } > "$output_file"
    
    print_success "Report generated: $output_file"
}

# === Main Command Handler ===
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        status)
            check_status
            ;;
        test)
            test_mocks "$@"
            ;;
        find)
            find_usage
            ;;
        validate)
            if [[ -z "${1:-}" ]]; then
                print_error "Usage: $0 validate <mock_name>"
                exit 1
            fi
            validate_mock "$1"
            ;;
        report)
            generate_report "$@"
            ;;
        fix-perms)
            echo "Fixing permissions on Tier 2 mocks..."
            chmod +x "$TIER2_DIR"/*.sh 2>/dev/null || true
            print_success "Permissions fixed"
            ;;
        help|*)
            echo "Mock Migration Helper"
            echo ""
            echo "Usage: $0 [command] [options]"
            echo ""
            echo "Commands:"
            echo "  status       - Check migration status"
            echo "  test [type]  - Test mocks (tier2|legacy)"
            echo "  find         - Find mock usage in codebase"
            echo "  validate <name> - Validate specific mock"
            echo "  report [file] - Generate migration report"
            echo "  fix-perms    - Fix execute permissions"
            echo "  help         - Show this help"
            ;;
    esac
}

# Run main function
main "$@"