#!/usr/bin/env bash

################################################################################
# CLI Interface Module for Scenario-to-App Converter
# 
# Provides command-line argument parsing, usage display, and logging functions
# for the scenario-to-app.sh script.
#
# Functions:
#   - show_usage()           Display help message
#   - parse_args()           Parse command line arguments
#   - log_info()            Log informational messages
#   - log_success()         Log success messages
#   - log_warning()         Log warning messages
#   - log_error()           Log error messages
#   - log_step()            Log step messages
#   - log_phase()           Log phase headers
#   - log_banner()          Log banners with spacing
#
# Global variables set by parse_args():
#   - DEPLOYMENT_MODE       Deployment mode (local, docker, k8s)
#   - VALIDATION_MODE       Validation mode (none, basic, full)
#   - DRY_RUN              Dry run flag (true/false)
#   - VERBOSE              Verbose flag (true/false)
#   - SCENARIO_NAME        Name of scenario to process
#
################################################################################

set -euo pipefail

# Default values for global variables
DEPLOYMENT_MODE=""
VALIDATION_MODE=""
DRY_RUN=""
VERBOSE=""
SCENARIO_NAME=""

################################################################################
# Logging Functions
################################################################################

# Log informational messages
log_info() { 
    echo "[INFO] $1" 
}

# Log success messages
log_success() { 
    echo "[SUCCESS] $1" 
}

# Log warning messages
log_warning() { 
    echo "[WARNING] $1" 
}

# Log error messages to stderr
log_error() { 
    echo "[ERROR] $1" >&2 
}

# Log step messages
log_step() { 
    echo "[STEP] $1" 
}

# Log phase headers
log_phase() { 
    echo "=== $1 ===" 
}

# Log banners with spacing
log_banner() { 
    echo ""
    echo "=== $1 ==="
    echo ""
}

################################################################################
# Usage Display
################################################################################

# Show usage information
show_usage() {
    cat << 'EOF'
Scenario-to-App Conversion Script

Converts Vrooli scenarios into deployable applications using the unified
service.json format and resource injection engine.

SCHEMA REFERENCE:
  All service.json files follow the official schema:
  .vrooli/schemas/service.schema.json
  
  Key paths used in this script:
  - .service.name           (scenario identifier)
  - .service.displayName    (human-readable name)
  - .resources.{category}.{resource}  (resource definitions)
  - .resources.{category}.{resource}.required  (not .optional!)
  - .resources.{category}.{resource}.initialization  (setup data)

Usage:
  ./scenario-to-app.sh <scenario-name> [options]

Options:
  --mode        Deployment mode (local, docker, k8s) [default: local]
  --validate    Validation mode (none, basic, full) [default: full]
  --dry-run     Show what would be done without executing
  --verbose     Enable verbose logging
  --help        Show this help message

Examples:
  ./scenario-to-app.sh multi-modal-ai-assistant
  ./scenario-to-app.sh test-injection --mode docker --verbose
  ./scenario-to-app.sh analytics-dashboard --dry-run --validate basic
EOF
}

################################################################################
# Argument Parsing
################################################################################

# Parse command line arguments
# Sets global variables based on provided arguments
parse_args() {
    # Initialize defaults
    DEPLOYMENT_MODE="local"
    VALIDATION_MODE="full"
    DRY_RUN=false
    VERBOSE=false
    SCENARIO_NAME=""
    
    local scenario_provided=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --mode)
                if [[ $# -lt 2 ]]; then
                    log_error "Option --mode requires a value"
                    show_usage
                    return 1
                fi
                DEPLOYMENT_MODE="$2"
                # Validate deployment mode
                case "$DEPLOYMENT_MODE" in
                    local|docker|k8s)
                        # Valid modes
                        ;;
                    *)
                        log_error "Invalid deployment mode: $DEPLOYMENT_MODE"
                        log_error "Valid modes are: local, docker, k8s"
                        return 1
                        ;;
                esac
                shift 2
                ;;
            --validate)
                if [[ $# -lt 2 ]]; then
                    log_error "Option --validate requires a value"
                    show_usage
                    return 1
                fi
                VALIDATION_MODE="$2"
                # Validate validation mode
                case "$VALIDATION_MODE" in
                    none|basic|full)
                        # Valid modes
                        ;;
                    *)
                        log_error "Invalid validation mode: $VALIDATION_MODE"
                        log_error "Valid modes are: none, basic, full"
                        return 1
                        ;;
                esac
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            --*)
                log_error "Unknown option: $1"
                show_usage
                return 1
                ;;
            *)
                if [[ "$scenario_provided" == true ]]; then
                    log_error "Multiple scenario names provided: '$SCENARIO_NAME' and '$1'"
                    show_usage
                    return 1
                fi
                SCENARIO_NAME="$1"
                scenario_provided=true
                shift
                ;;
        esac
    done
    
    if [[ "$scenario_provided" == false ]]; then
        log_error "No scenario specified"
        show_usage
        return 1
    fi
    
    # Validate scenario name format
    if [[ ! "$SCENARIO_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log_error "Invalid scenario name: $SCENARIO_NAME"
        log_error "Scenario names must contain only letters, numbers, hyphens, and underscores"
        return 1
    fi
    
    return 0
}

################################################################################
# Validation Helper Functions
################################################################################

# Validate that all required global variables are set
validate_parsed_args() {
    local missing_vars=()
    
    [[ -z "$DEPLOYMENT_MODE" ]] && missing_vars+=("DEPLOYMENT_MODE")
    [[ -z "$VALIDATION_MODE" ]] && missing_vars+=("VALIDATION_MODE")
    [[ -z "$SCENARIO_NAME" ]] && missing_vars+=("SCENARIO_NAME")
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log_error "Missing required variables after parsing: ${missing_vars[*]}"
        return 1
    fi
    
    return 0
}

# Display parsed configuration (for debugging/verbose mode)
display_parsed_config() {
    if [[ "$VERBOSE" == true ]]; then
        log_info "Parsed configuration:"
        log_info "  Scenario: $SCENARIO_NAME"
        log_info "  Deployment mode: $DEPLOYMENT_MODE"
        log_info "  Validation mode: $VALIDATION_MODE"
        log_info "  Dry run: $DRY_RUN"
        log_info "  Verbose: $VERBOSE"
    fi
}

################################################################################
# Export Functions and Variables
################################################################################

# Export functions so they can be used by other scripts
export -f log_info log_success log_warning log_error log_step log_phase log_banner
export -f show_usage parse_args validate_parsed_args display_parsed_config

# Note: Global variables are set by parse_args() and should be accessed directly
# by the calling script after successful parsing