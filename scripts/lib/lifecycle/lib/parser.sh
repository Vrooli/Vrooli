#!/usr/bin/env bash
# Lifecycle Engine - Argument Parser Module
# Handles command-line argument parsing and validation

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$LIB_LIFECYCLE_LIB_DIR/../../utils/var.sh"

# Guard against re-sourcing
[[ -n "${_PARSER_MODULE_LOADED:-}" ]] && return 0
declare -gr _PARSER_MODULE_LOADED=1

# Global variables that will be set by parser
declare -g LIFECYCLE_PHASE=""
declare -g TARGET="${TARGET:-}"
declare -g DRY_RUN="${DRY_RUN:-false}"
declare -g VERBOSE="${VERBOSE:-false}"
declare -g SERVICE_JSON_PATH=""
declare -g SKIP_STEPS=()
declare -g ONLY_STEP=""
declare -g LIFECYCLE_TIMEOUT=""
declare -g EXTRA_ENV=()

# Phase-specific arguments (populated dynamically)
declare -Ag PHASE_ARGS=()

#######################################
# Show usage information
#######################################
parser::usage() {
    cat << EOF
Usage: $0 <phase> [options]

Lifecycle Phases:
  setup       Initialize environment
  develop     Start development servers
  build       Build for production
  deploy      Deploy to target environment
  test        Run test suites
  clean       Clean artifacts
  <custom>    Any custom phase defined in service.json

Common Options:
  --target <target>     Target platform (docker, native-linux, k8s, etc.)
  --dry-run            Show what would be executed without running
  --verbose            Enable verbose output
  --config <path>      Path to service.json (default: ./.vrooli/service.json)
  --env <key=value>    Set environment variable
  --skip <step>        Skip specific step by name
  --only <step>        Only run specific step
  --timeout <duration> Override default timeout
  --help               Show this help message

Phase-Specific Options:
  setup:
    --clean <yes/no>         Clean install (default: no)
    --resources <mode>       Resource installation mode (enabled/none/specific)
    --sudo-mode <mode>       Sudo handling (ask/skip/force)
    --environment <env>      Environment type (development/production)
    
  develop:
    --detached <yes/no>      Run in background (default: no)
    --clean-instances <y/n>  Stop all instances first
    --skip-port-check        Skip port conflict check
    --environment <env>      Development environment
    
  build:
    --test <yes/no>          Run tests before build (default: no)
    --lint <yes/no>          Run linting (default: no)
    --version <version>      Build version
    --bundles <types>        Bundle types (zip/tar/none)
    --artifacts <types>      Artifact types (docker/k8s/all)
    --environment <env>      Build environment
    
  deploy:
    --source <type>          Deployment source (docker/k8s/zip)
    --version <version>      Deploy version
    --environment <env>      Target environment
    --skip-health-check      Skip health checks
    
  test:
    --type <type>            Test type (all/unit/integration/shell)
    --pattern <pattern>      Test file pattern
    --coverage <yes/no>      Generate coverage report
    --bail-on-fail <y/n>     Stop on first failure

Environment Variables:
  TARGET               Default target platform
  DRY_RUN             Enable dry-run mode
  VERBOSE             Enable verbose output
  DEBUG               Enable debug output

Examples:
  $0 setup --target native-linux --clean yes
  $0 develop --target docker --detached yes
  $0 build --target k8s --version 2.0.0 --test yes
  $0 test --type unit --coverage yes
  $0 deploy --source k8s --environment production

EOF
}

#######################################
# Parse phase-specific argument
# Arguments:
#   $1 - Argument name
#   $2 - Argument value (if required)
#   $3 - Phase name
# Returns:
#   0 on success with shift count in SHIFT_COUNT
#   1 on unknown argument
#######################################
parser::parse_phase_arg() {
    local arg="$1"
    local value="${2:-}"
    local phase="$3"
    
    # Variable to indicate how many positions to shift
    declare -g SHIFT_COUNT=1
    
    case "$phase" in
        setup)
            case "$arg" in
                --clean)
                    [[ -z "$value" ]] && { echo "Error: --clean requires a value" >&2; return 1; }
                    PHASE_ARGS[CLEAN]="$value"
                    SHIFT_COUNT=2
                    ;;
                --resources)
                    [[ -z "$value" ]] && { echo "Error: --resources requires a value" >&2; return 1; }
                    PHASE_ARGS[RESOURCES]="$value"
                    SHIFT_COUNT=2
                    ;;
                --sudo-mode)
                    [[ -z "$value" ]] && { echo "Error: --sudo-mode requires a value" >&2; return 1; }
                    PHASE_ARGS[SUDO_MODE]="$value"
                    SHIFT_COUNT=2
                    ;;
                --environment)
                    [[ -z "$value" ]] && { echo "Error: --environment requires a value" >&2; return 1; }
                    PHASE_ARGS[ENVIRONMENT]="$value"
                    SHIFT_COUNT=2
                    ;;
                --is-ci)
                    PHASE_ARGS[IS_CI]="yes"
                    SHIFT_COUNT=1
                    ;;
                *)
                    return 1
                    ;;
            esac
            ;;
            
        develop)
            case "$arg" in
                --detached)
                    [[ -z "$value" ]] && { echo "Error: --detached requires a value" >&2; return 1; }
                    PHASE_ARGS[DETACHED]="$value"
                    SHIFT_COUNT=2
                    ;;
                --clean-instances)
                    [[ -z "$value" ]] && { echo "Error: --clean-instances requires a value" >&2; return 1; }
                    PHASE_ARGS[CLEAN_INSTANCES]="$value"
                    SHIFT_COUNT=2
                    ;;
                --skip-port-check)
                    PHASE_ARGS[SKIP_PORT_CHECK]="yes"
                    SHIFT_COUNT=1
                    ;;
                --skip-instance-check)
                    PHASE_ARGS[SKIP_INSTANCE_CHECK]="yes"
                    SHIFT_COUNT=1
                    ;;
                --environment)
                    [[ -z "$value" ]] && { echo "Error: --environment requires a value" >&2; return 1; }
                    PHASE_ARGS[ENVIRONMENT]="$value"
                    SHIFT_COUNT=2
                    ;;
                *)
                    return 1
                    ;;
            esac
            ;;
            
        build)
            case "$arg" in
                --test)
                    [[ -z "$value" ]] && { echo "Error: --test requires a value" >&2; return 1; }
                    PHASE_ARGS[TEST]="$value"
                    SHIFT_COUNT=2
                    ;;
                --lint)
                    [[ -z "$value" ]] && { echo "Error: --lint requires a value" >&2; return 1; }
                    PHASE_ARGS[LINT]="$value"
                    SHIFT_COUNT=2
                    ;;
                --version)
                    [[ -z "$value" ]] && { echo "Error: --version requires a value" >&2; return 1; }
                    PHASE_ARGS[VERSION]="$value"
                    SHIFT_COUNT=2
                    ;;
                --bundles)
                    [[ -z "$value" ]] && { echo "Error: --bundles requires a value" >&2; return 1; }
                    PHASE_ARGS[BUNDLES]="$value"
                    SHIFT_COUNT=2
                    ;;
                --artifacts)
                    [[ -z "$value" ]] && { echo "Error: --artifacts requires a value" >&2; return 1; }
                    PHASE_ARGS[ARTIFACTS]="$value"
                    SHIFT_COUNT=2
                    ;;
                --environment)
                    [[ -z "$value" ]] && { echo "Error: --environment requires a value" >&2; return 1; }
                    PHASE_ARGS[ENVIRONMENT]="$value"
                    SHIFT_COUNT=2
                    ;;
                *)
                    return 1
                    ;;
            esac
            ;;
            
        deploy)
            case "$arg" in
                --source)
                    [[ -z "$value" ]] && { echo "Error: --source requires a value" >&2; return 1; }
                    PHASE_ARGS[SOURCE_TYPE]="$value"
                    SHIFT_COUNT=2
                    ;;
                --version)
                    [[ -z "$value" ]] && { echo "Error: --version requires a value" >&2; return 1; }
                    PHASE_ARGS[VERSION]="$value"
                    SHIFT_COUNT=2
                    ;;
                --environment)
                    [[ -z "$value" ]] && { echo "Error: --environment requires a value" >&2; return 1; }
                    PHASE_ARGS[ENVIRONMENT]="$value"
                    SHIFT_COUNT=2
                    ;;
                --skip-health-check)
                    PHASE_ARGS[SKIP_HEALTH_CHECK]="yes"
                    SHIFT_COUNT=1
                    ;;
                *)
                    return 1
                    ;;
            esac
            ;;
            
        test)
            case "$arg" in
                --type)
                    [[ -z "$value" ]] && { echo "Error: --type requires a value" >&2; return 1; }
                    PHASE_ARGS[TEST_TYPE]="$value"
                    SHIFT_COUNT=2
                    ;;
                --pattern)
                    [[ -z "$value" ]] && { echo "Error: --pattern requires a value" >&2; return 1; }
                    PHASE_ARGS[TEST_PATTERN]="$value"
                    SHIFT_COUNT=2
                    ;;
                --coverage)
                    [[ -z "$value" ]] && { echo "Error: --coverage requires a value" >&2; return 1; }
                    PHASE_ARGS[COVERAGE]="$value"
                    SHIFT_COUNT=2
                    ;;
                --bail-on-fail)
                    [[ -z "$value" ]] && { echo "Error: --bail-on-fail requires a value" >&2; return 1; }
                    PHASE_ARGS[BAIL_ON_FAIL]="$value"
                    SHIFT_COUNT=2
                    ;;
                *)
                    return 1
                    ;;
            esac
            ;;
            
        *)
            # Unknown phase, can't parse phase-specific args
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Parse command line arguments
# Arguments:
#   $@ - All command line arguments
# Returns:
#   0 on success, 1 on error
# Sets:
#   Global variables defined above
#######################################
parser::parse_args() {
    [[ $# -eq 0 ]] && { parser::usage; return 1; }
    
    # Check for help first
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        parser::usage
        exit 0
    fi
    
    # First argument is the lifecycle phase
    LIFECYCLE_PHASE="$1"
    shift
    
    # Parse remaining options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help|-h)
                parser::usage
                exit 0
                ;;
            --target|-t)
                [[ $# -lt 2 ]] && { echo "Error: --target requires a value" >&2; return 1; }
                TARGET="$2"
                shift 2
                ;;
            --dry-run|-n)
                DRY_RUN="true"
                shift
                ;;
            --verbose|-v)
                VERBOSE="true"
                shift
                ;;
            --config|-c)
                [[ $# -lt 2 ]] && { echo "Error: --config requires a value" >&2; return 1; }
                SERVICE_JSON_PATH="$2"
                shift 2
                ;;
            --env|-e)
                [[ $# -lt 2 ]] && { echo "Error: --env requires a value" >&2; return 1; }
                # Validate format
                if [[ ! "$2" =~ ^[A-Za-z_][A-Za-z0-9_]*=.*$ ]]; then
                    echo "Error: Invalid environment variable format: $2" >&2
                    echo "Expected: KEY=value" >&2
                    return 1
                fi
                EXTRA_ENV+=("$2")
                export "$2"
                shift 2
                ;;
            --skip)
                [[ $# -lt 2 ]] && { echo "Error: --skip requires a value" >&2; return 1; }
                SKIP_STEPS+=("$2")
                # Also set environment variable for compatibility
                local skip_var="SKIP_${2^^}"
                skip_var="${skip_var//-/_}"
                export "$skip_var=true"
                shift 2
                ;;
            --only)
                [[ $# -lt 2 ]] && { echo "Error: --only requires a value" >&2; return 1; }
                if [[ -n "$ONLY_STEP" ]]; then
                    echo "Error: --only can only be specified once" >&2
                    return 1
                fi
                ONLY_STEP="$2"
                shift 2
                ;;
            --timeout)
                [[ $# -lt 2 ]] && { echo "Error: --timeout requires a value" >&2; return 1; }
                # Validate timeout format (number with optional s/m/h suffix)
                if [[ ! "$2" =~ ^[0-9]+[smh]?$ ]]; then
                    echo "Error: Invalid timeout format: $2" >&2
                    echo "Expected: number with optional s/m/h suffix (e.g., 30s, 5m, 1h)" >&2
                    return 1
                fi
                LIFECYCLE_TIMEOUT="$2"
                shift 2
                ;;
            -*)
                # Try to parse as phase-specific argument
                if parser::parse_phase_arg "$1" "${2:-}" "$LIFECYCLE_PHASE"; then
                    shift "$SHIFT_COUNT"
                else
                    echo "Error: Unknown option for phase '$LIFECYCLE_PHASE': $1" >&2
                    echo "Use --help to see available options" >&2
                    return 1
                fi
                ;;
            *)
                # Also try phase-specific parsing for non-hyphenated args (backwards compat)
                if [[ "$1" =~ ^[A-Z_]+= ]]; then
                    # Environment variable format
                    export "$1"
                    shift
                else
                    echo "Error: Unexpected argument: $1" >&2
                    parser::usage
                    return 1
                fi
                ;;
        esac
    done
    
    return 0
}

#######################################
# Validate parsed arguments
# Returns:
#   0 if valid, 1 if invalid
#######################################
parser::validate() {
    # Validate phase name
    if [[ -z "$LIFECYCLE_PHASE" ]]; then
        echo "Error: Lifecycle phase is required" >&2
        return 1
    fi
    
    # Validate phase name format (alphanumeric with hyphens)
    if [[ ! "$LIFECYCLE_PHASE" =~ ^[a-zA-Z][a-zA-Z0-9-]*$ ]]; then
        echo "Error: Invalid phase name: $LIFECYCLE_PHASE" >&2
        echo "Phase names must start with a letter and contain only letters, numbers, and hyphens" >&2
        return 1
    fi
    
    # Validate target if specified
    if [[ -n "$TARGET" ]]; then
        if [[ ! "$TARGET" =~ ^[a-zA-Z][a-zA-Z0-9-]*$ ]]; then
            echo "Error: Invalid target name: $TARGET" >&2
            return 1
        fi
    fi
    
    # Validate config path if specified
    if [[ -n "$SERVICE_JSON_PATH" ]] && [[ ! -f "$SERVICE_JSON_PATH" ]]; then
        echo "Error: Configuration file not found: $SERVICE_JSON_PATH" >&2
        return 1
    fi
    
    # Check for conflicting options
    if [[ -n "$ONLY_STEP" ]] && [[ ${#SKIP_STEPS[@]} -gt 0 ]]; then
        echo "Warning: --only and --skip are both specified. --only takes precedence." >&2
    fi
    
    return 0
}

#######################################
# Export parser results for use by other modules
#######################################
parser::export() {
    export LIFECYCLE_PHASE
    export TARGET
    export DRY_RUN
    export VERBOSE
    export SERVICE_JSON_PATH
    export ONLY_STEP
    export LIFECYCLE_TIMEOUT
    
    # Export skip steps as a colon-separated list for easier checking
    if [[ ${#SKIP_STEPS[@]} -gt 0 ]]; then
        export SKIP_STEPS_LIST
        SKIP_STEPS_LIST=$(IFS=:; echo "${SKIP_STEPS[*]}")
    fi
    
    # Export extra environment variables list
    if [[ ${#EXTRA_ENV[@]} -gt 0 ]]; then
        export EXTRA_ENV_LIST
        EXTRA_ENV_LIST=$(IFS=:; echo "${EXTRA_ENV[*]}")
    fi
    
    # Export phase-specific arguments as environment variables
    for key in "${!PHASE_ARGS[@]}"; do
        export "$key=${PHASE_ARGS[$key]}"
    done
}

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi