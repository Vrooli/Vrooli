#!/usr/bin/env bash
# Scenario Completeness Scoring Module
# Provides objective metrics-based completeness assessment
#
# This module delegates to the scenario-completeness-scoring service for all scoring operations.
# The service provides full validation analysis with gaming detection.

set -euo pipefail

# Check if scenario-completeness-scoring service is available
_completeness::is_service_available() {
    local api_port
    api_port=$(vrooli scenario port scenario-completeness-scoring API_PORT 2>/dev/null || echo "")

    if [[ -z "$api_port" ]]; then
        return 1
    fi

    # Try health check
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${api_port}/health" 2>/dev/null | grep -q "200"; then
        return 0
    fi

    return 1
}

# Auto-start the scenario-completeness-scoring service and wait for it to be healthy
_completeness::ensure_service_running() {
    local timeout=60
    local interval=2

    # Check if already running
    if _completeness::is_service_available; then
        return 0
    fi

    # Start the service
    log::info "Starting scenario-completeness-scoring service..."
    if ! vrooli scenario start scenario-completeness-scoring >/dev/null 2>&1; then
        log::warn "Failed to start scenario-completeness-scoring service"
        return 1
    fi

    # Wait for service to be healthy
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if _completeness::is_service_available; then
            log::info "Service ready"
            return 0
        fi
        sleep $interval
        elapsed=$((elapsed + interval))
    done

    log::warn "Service failed to start within ${timeout}s"
    return 1
}

# Calculate completeness score for a scenario
scenario::completeness::score() {
    local scenario_name="${1:-}"
    local format="human"
    local extra_args=()

    # Parse options
    shift || true
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-human}"
                if [[ ! "$format" =~ ^(json|human)$ ]]; then
                    log::error "Invalid format: $format. Use 'json' or 'human'."
                    return 1
                fi
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|--metrics|-v|-m)
                # Pass through to CLI
                extra_args+=("$1")
                shift
                ;;
            --help|-h)
                cat <<'HELP'
Usage: vrooli scenario completeness <scenario-name> [options]

Calculate objective completeness score with validation quality analysis.

Score Components:
  Quality (50%):
    • Requirement pass rates (20%)
    • Operational target pass rates (15%)
    • Test pass rates (15%)
  Coverage (15%):
    • Test coverage ratio (8%)
    • Requirement depth (7%)
  Quantity (10%):
    • Absolute counts vs category thresholds (10%)
  UI Completeness (25%):
    • Template detection (10%)
    • Component complexity (5%)
    • API integration depth (5%)
    • Routing complexity (2.5%)
    • Code volume (2.5%)

Validation Quality Penalties:
  The score includes penalties for test anti-patterns:
  • Invalid test locations        (up to -25pts)
  • Monolithic test files         (up to -15pts)
  • Ungrouped operational targets (up to -10pts)
  • Insufficient validation layers(up to -20pts)
  • Superficial test implementation (up to -10pts)
  • Missing test automation       (up to -15pts)
  • Suspicious 1:1 ratio          (-5pts)

Options:
  --format <type>    Output format: json (machine-readable) or human (default)
  --json             Shorthand for --format json
  --verbose, -v      Show detailed explanations and per-requirement breakdowns
  --metrics, -m      Show full detailed metrics breakdown (implies --verbose)
  --help, -h         Show this help message

Examples:
  vrooli scenario completeness deployment-manager
  vrooli scenario completeness deployment-manager --verbose
  vrooli scenario completeness deployment-manager --metrics
  vrooli scenario completeness deployment-manager --json
  vrooli scenario completeness deployment-manager --format json

Score Classifications:
  96-100: production_ready    (Full increment toward finalization)
  81-95:  nearly_ready        (Half increment toward finalization)
  61-80:  mostly_complete     (No finalization credit)
  41-60:  functional_incomplete
  21-40:  foundation_laid
  0-20:   early_stage

NOTE: This command automatically starts the scenario-completeness-scoring service
if it's not running. The service stays running for subsequent calls.

HELP
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done

    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario completeness <scenario-name> [--format json|human] [--json]"
        echo "Run 'vrooli scenario completeness --help' for more information"
        return 1
    fi

    # Verify scenario exists
    local scenario_root="${VROOLI_ROOT:-$HOME/Vrooli}/scenarios/${scenario_name}"
    if [[ ! -d "$scenario_root" ]]; then
        log::error "Scenario not found: $scenario_name"
        log::info "Available scenarios: $(ls "${VROOLI_ROOT:-$HOME/Vrooli}/scenarios" | tr '\n' ' ')"
        return 1
    fi

    # Use scenario-completeness-scoring service
    local scs_cli="${VROOLI_ROOT:-$HOME/Vrooli}/scenarios/scenario-completeness-scoring/cli/scenario-completeness-scoring"

    if [[ ! -x "$scs_cli" ]]; then
        log::error "scenario-completeness-scoring CLI not found or not executable"
        log::info "Run setup first: vrooli scenario setup scenario-completeness-scoring"
        return 1
    fi

    # Auto-start service if not running
    if ! _completeness::ensure_service_running; then
        log::error "Failed to start scenario-completeness-scoring service"
        return 1
    fi

    # Build args for scenario-completeness-scoring CLI
    local scs_args=("score" "$scenario_name")

    if [[ "$format" == "json" ]]; then
        scs_args+=("--json")
    fi

    for arg in "${extra_args[@]+"${extra_args[@]}"}"; do
        if [[ -n "$arg" ]]; then
            scs_args+=("$arg")
        fi
    done

    if ! "$scs_cli" "${scs_args[@]}"; then
        log::error "Completeness calculation failed for $scenario_name"
        return 1
    fi

    return 0
}
