#!/usr/bin/env bash
# Standalone Cleanup Script - Clean all test resources
# Can be run manually to clean up accumulated test resources

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Script directory
SCRIPTS_TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPTS_TEST_DIR

# Source trash module for safe cleanup
# shellcheck disable=SC1091
source "${SCRIPTS_TEST_DIR}/../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load cleanup manager
source "$SCRIPTS_TEST_DIR/fixtures/cleanup-manager.sh"

# Parse command line arguments
ACTION="status"
FORCE=false
VERBOSE=false

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Clean up test resources from Vrooli test runs.

OPTIONS:
    -s, --status       Show cleanup status (default)
    -c, --clean        Run smart cleanup
    -a, --aggressive   Run aggressive cleanup
    -f, --full         Run full system cleanup
    -o, --old          Clean only old resources
    --force            Skip confirmation prompts
    -v, --verbose      Enable verbose output
    -h, --help         Show this help message

EXAMPLES:
    $0                 # Show status
    $0 --clean         # Run smart cleanup
    $0 --aggressive    # Run aggressive cleanup
    $0 --full --force  # Full cleanup without confirmation

CLEANUP LEVELS:
    status      - Show current resource usage
    clean       - Smart cleanup based on usage
    aggressive  - Remove all test resources
    old         - Remove resources older than 24 hours
    full        - Complete system cleanup including Docker

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--status)
            ACTION="status"
            shift
            ;;
        -c|--clean)
            ACTION="clean"
            shift
            ;;
        -a|--aggressive)
            ACTION="aggressive"
            shift
            ;;
        -f|--full)
            ACTION="full"
            shift
            ;;
        -o|--old)
            ACTION="old"
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Confirmation prompt
confirm_action() {
    local action_desc="$1"
    
    if [[ "$FORCE" == "true" ]]; then
        return 0
    fi
    
    echo
    log_warning "About to perform: $action_desc"
    echo -n "Are you sure you want to continue? (y/N): "
    read -r response
    
    case "$response" in
        [yY][eE][sS]|[yY])
            return 0
            ;;
        *)
            echo "Cancelled."
            exit 0
            ;;
    esac
}

# Main execution
main() {
    echo "======================================="
    echo "    Vrooli Test Resource Cleanup      "
    echo "======================================="
    echo
    
    case "$ACTION" in
        status)
            log_info "Generating cleanup status report..."
            vrooli_cleanup_status
            
            # Show additional info if verbose
            if [[ "$VERBOSE" == "true" ]]; then
                echo
                log_info "Recent test directories in /tmp:"
                find /tmp -maxdepth 1 -name "*vrooli*" -o -name "*test*" -o -name "bats*" 2>/dev/null | head -20 || true
                
                if [[ -d "/dev/shm" ]]; then
                    echo
                    log_info "Recent test directories in /dev/shm:"
                    find /dev/shm -maxdepth 1 -name "*vrooli*" -o -name "*test*" 2>/dev/null | head -20 || true
                fi
            fi
            ;;
            
        clean)
            confirm_action "Smart cleanup based on current usage"
            log_info "Running smart cleanup..."
            
            if vrooli_smart_cleanup; then
                log_success "Smart cleanup completed successfully"
                vrooli_validate_cleanup || log_warning "Some resources may remain"
            else
                log_error "Smart cleanup encountered errors"
                exit 1
            fi
            ;;
            
        aggressive)
            confirm_action "Aggressive cleanup - remove all test resources"
            log_info "Running aggressive cleanup..."
            
            if vrooli_aggressive_cleanup; then
                log_success "Aggressive cleanup completed successfully"
                vrooli_validate_cleanup || log_warning "Some resources may remain"
            else
                log_error "Aggressive cleanup encountered errors"
                exit 1
            fi
            ;;
            
        old)
            confirm_action "Clean resources older than ${CLEANUP_AGE_HOURS:-24} hours"
            log_info "Cleaning old resources..."
            
            if vrooli_cleanup_old_resources; then
                log_success "Old resource cleanup completed successfully"
            else
                log_error "Old resource cleanup encountered errors"
                exit 1
            fi
            ;;
            
        full)
            confirm_action "FULL SYSTEM CLEANUP - This will remove all test resources and Docker containers"
            log_warning "This is the most aggressive cleanup option!"
            sleep 2
            
            log_info "Running full system cleanup..."
            
            if vrooli_full_system_cleanup; then
                log_success "Full system cleanup completed successfully"
                vrooli_validate_cleanup || log_warning "Some resources may remain"
            else
                log_error "Full system cleanup encountered errors"
                exit 1
            fi
            ;;
            
        *)
            log_error "Unknown action: $ACTION"
            exit 1
            ;;
    esac
    
    echo
    echo "======================================="
    log_info "Cleanup operation complete"
    
    # Show final status if not already showing status
    if [[ "$ACTION" != "status" ]]; then
        echo
        log_info "Final status:"
        vrooli_cleanup_status
    fi
}

# Check for required permissions
check_permissions() {
    # Check if we can write to /tmp
    if ! touch /tmp/.cleanup_test_$$ 2>/dev/null; then
        log_error "Cannot write to /tmp - insufficient permissions"
        exit 1
    fi
    trash::safe_remove /tmp/.cleanup_test_$$ --test-cleanup
    
    # Check if we can write to /dev/shm (if it exists)
    if [[ -d "/dev/shm" ]]; then
        if ! touch /dev/shm/.cleanup_test_$$ 2>/dev/null; then
            log_warning "Cannot write to /dev/shm - some cleanup may be limited"
        else
            trash::safe_remove /dev/shm/.cleanup_test_$$ --test-cleanup
        fi
    fi
}

# Run checks and main
check_permissions
main