#!/usr/bin/env bash
#
# triage.sh - Automated diagnostic script for Browser Automation Studio
#
# This script collects all diagnostic information and outputs it as JSON
# for easy consumption by AI agents or automated debugging systems.
#
# Usage:
#   ./scripts/triage.sh              # Output to stdout (human-readable summary + JSON)
#   ./scripts/triage.sh --json       # Output pure JSON only (for automation)
#   ./scripts/triage.sh --quiet      # Suppress status messages
#
# Environment Variables:
#   API_URL - API endpoint base URL (default: http://localhost:19771)
#   API_PORT - API port (default: 19771)

set -o pipefail

# Configuration
API_PORT="${API_PORT:-19771}"
API_URL="${API_URL:-http://localhost:${API_PORT}}"
TIMEOUT=5

# Parse arguments
JSON_ONLY=false
QUIET=false
for arg in "$@"; do
    case "$arg" in
        --json) JSON_ONLY=true ;;
        --quiet) QUIET=true ;;
    esac
done

# Helper to log messages (respects --quiet)
log() {
    if [ "$QUIET" = "false" ] && [ "$JSON_ONLY" = "false" ]; then
        echo "$@" >&2
    fi
}

# Helper to make API calls with timeout
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"

    if [ -n "$data" ]; then
        curl -s -m "$TIMEOUT" -X "$method" "${API_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data" 2>/dev/null
    else
        curl -s -m "$TIMEOUT" -X "$method" "${API_URL}${endpoint}" 2>/dev/null
    fi
}

# Collect health check
log "Checking health..."
health_result=$(api_call GET "/health")
health_status=$?
if [ $health_status -ne 0 ] || [ -z "$health_result" ]; then
    health_result='{"status": "unreachable", "error": "API not responding"}'
fi

# Collect sessions
log "Listing sessions..."
sessions_result=$(api_call GET "/api/v1/observability/sessions")
if [ -z "$sessions_result" ]; then
    sessions_result='{"sessions": [], "error": "Could not fetch sessions"}'
fi

# Run quick diagnostics
log "Running quick diagnostics..."
diag_result=$(api_call POST "/api/v1/observability/diagnostics/run" '{"level": "quick"}')
if [ -z "$diag_result" ]; then
    diag_result='{"error": "Could not run diagnostics"}'
fi

# Run pipeline test
log "Running pipeline test..."
pipeline_result=$(api_call POST "/api/v1/observability/pipeline-test")
if [ -z "$pipeline_result" ]; then
    pipeline_result='{"error": "Could not run pipeline test"}'
fi

# Build combined JSON output
combined_json=$(jq -n \
    --arg timestamp "$(date -Iseconds)" \
    --arg api_url "$API_URL" \
    --argjson health "$health_result" \
    --argjson sessions "$sessions_result" \
    --argjson diagnostics "$diag_result" \
    --argjson pipeline "$pipeline_result" \
    '{
        triageReport: {
            timestamp: $timestamp,
            apiUrl: $api_url,
            health: $health,
            sessions: $sessions,
            diagnostics: $diagnostics,
            pipelineTest: $pipeline
        }
    }')

# Output
if [ "$JSON_ONLY" = "true" ]; then
    echo "$combined_json" | jq .
else
    # Print human-readable summary
    echo ""
    echo "=========================================="
    echo "  TRIAGE REPORT"
    echo "  $(date -Iseconds)"
    echo "=========================================="
    echo ""

    # Health status
    health_status=$(echo "$health_result" | jq -r '.status // "unknown"')
    if [ "$health_status" = "healthy" ] || [ "$health_status" = "ok" ]; then
        echo "Health: OK"
    else
        echo "Health: DEGRADED ($health_status)"
    fi

    # Session count
    session_count=$(echo "$sessions_result" | jq -r '.sessions | length // 0')
    echo "Active Sessions: $session_count"

    # Diagnostic summary
    diag_ready=$(echo "$diag_result" | jq -r '.results.recording.ready // false')
    diag_issues=$(echo "$diag_result" | jq -r '.results.recording.issues | length // 0')
    echo "Recording Ready: $diag_ready"
    echo "Diagnostic Issues: $diag_issues"

    # Pipeline test result
    pipeline_success=$(echo "$pipeline_result" | jq -r '.success // false')
    if [ "$pipeline_success" = "true" ]; then
        echo "Pipeline Test: PASSED"
    else
        failure_point=$(echo "$pipeline_result" | jq -r '.failure_point // "unknown"')
        echo "Pipeline Test: FAILED at $failure_point"
    fi

    echo ""
    echo "=========================================="
    echo "  FULL JSON OUTPUT"
    echo "=========================================="
    echo ""
    echo "$combined_json" | jq .
fi
