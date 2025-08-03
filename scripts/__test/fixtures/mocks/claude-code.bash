#!/usr/bin/env bash
# Claude Code Resource Mock Implementation
# Provides realistic mock responses for Claude Code AI assistant service

# Prevent duplicate loading
if [[ "${CLAUDE_CODE_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export CLAUDE_CODE_MOCK_LOADED="true"

#######################################
# Setup Claude Code mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::claude-code::setup() {
    local state="${1:-healthy}"
    
    # Configure Claude Code-specific environment
    export CLAUDE_CODE_PORT="${CLAUDE_CODE_PORT:-8000}"
    export CLAUDE_CODE_BASE_URL="http://localhost:${CLAUDE_CODE_PORT}"
    export CLAUDE_CODE_CONTAINER_NAME="${TEST_NAMESPACE}_claude-code"
    export CLAUDE_CODE_API_KEY="${CLAUDE_CODE_API_KEY:-test-api-key}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$CLAUDE_CODE_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::claude-code::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::claude-code::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::claude-code::setup_installing_endpoints
            ;;
        "stopped")
            mock::claude-code::setup_stopped_endpoints
            ;;
        *)
            echo "[CLAUDE_CODE_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[CLAUDE_CODE_MOCK] Claude Code mock configured with state: $state"
}

#######################################
# Setup healthy Claude Code endpoints
#######################################
mock::claude-code::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/health" \
        '{"status":"ok","version":"3.5.0","model":"claude-3-5-sonnet-20241022"}'
    
    # Chat completion endpoint
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/v1/messages" \
        '{
            "id": "msg_'$(date +%s)'",
            "type": "message",
            "role": "assistant",
            "content": [
                {
                    "type": "text",
                    "text": "I am Claude Code, an AI assistant designed to help with programming tasks. How can I assist you today?"
                }
            ],
            "model": "claude-3-5-sonnet-20241022",
            "stop_reason": "end_turn",
            "stop_sequence": null,
            "usage": {
                "input_tokens": 15,
                "output_tokens": 25
            }
        }' \
        "POST"
    
    # Code analysis endpoint
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/v1/code/analyze" \
        '{
            "analysis": {
                "language": "python",
                "complexity": "medium",
                "suggestions": [
                    "Consider adding type hints for better code clarity",
                    "Add error handling for file operations"
                ],
                "issues": [],
                "metrics": {
                    "lines_of_code": 50,
                    "cyclomatic_complexity": 3,
                    "maintainability_index": 85
                }
            }
        }' \
        "POST"
    
    # Code generation endpoint
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/v1/code/generate" \
        '{
            "generated_code": "def hello_world():\n    \"\"\"A simple hello world function.\"\"\"\n    print(\"Hello, World!\")\n    return \"Hello, World!\"",
            "language": "python",
            "explanation": "This function prints and returns a greeting message.",
            "tests": "def test_hello_world():\n    assert hello_world() == \"Hello, World!\""
        }' \
        "POST"
    
    # Session endpoint
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/v1/sessions" \
        '{
            "session_id": "session_'$(date +%s)'",
            "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
            "expires_at": "'$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ)'",
            "context": {
                "working_directory": "/workspace",
                "project_type": "python",
                "files_loaded": 0
            }
        }' \
        "POST"
}

#######################################
# Setup unhealthy Claude Code endpoints
#######################################
mock::claude-code::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/health" \
        '{"status":"error","error":"Model unavailable"}' \
        "GET" \
        "503"
    
    # Chat endpoint returns error
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/v1/messages" \
        '{"error":"Service temporarily unavailable"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing Claude Code endpoints
#######################################
mock::claude-code::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/health" \
        '{"status":"initializing","progress":98,"current_step":"Loading model weights"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$CLAUDE_CODE_BASE_URL/v1/messages" \
        '{"error":"Claude Code is still initializing"}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped Claude Code endpoints
#######################################
mock::claude-code::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$CLAUDE_CODE_BASE_URL"
}

#######################################
# Mock Claude Code-specific operations
#######################################

# Mock code review
mock::claude-code::review_code() {
    local code="$1"
    local language="${2:-python}"
    
    echo '{
        "review": {
            "overall_score": 8,
            "suggestions": [
                "Code is well-structured and readable",
                "Consider adding docstrings for better documentation",
                "Error handling could be improved"
            ],
            "issues": [
                {
                    "line": 5,
                    "severity": "warning",
                    "message": "Variable name could be more descriptive"
                }
            ],
            "positive_aspects": [
                "Good use of functions",
                "Clear variable names",
                "Proper indentation"
            ]
        }
    }'
}

# Mock code refactoring
mock::claude-code::refactor_code() {
    local code="$1"
    
    echo '{
        "refactored_code": "# Refactored version of the input code\n# with improved structure and readability",
        "changes": [
            "Extracted repeated logic into helper functions",
            "Improved variable naming",
            "Added type hints"
        ],
        "explanation": "The code has been refactored to improve maintainability and readability."
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::claude-code::setup
export -f mock::claude-code::setup_healthy_endpoints
export -f mock::claude-code::setup_unhealthy_endpoints
export -f mock::claude-code::setup_installing_endpoints
export -f mock::claude-code::setup_stopped_endpoints
export -f mock::claude-code::review_code
export -f mock::claude-code::refactor_code

echo "[CLAUDE_CODE_MOCK] Claude Code mock implementation loaded"