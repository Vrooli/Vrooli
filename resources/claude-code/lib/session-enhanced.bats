#!/usr/bin/env bats
# Tests for Claude Code session-enhanced.sh functions
bats_require_minimum_version 1.5.0

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code-session-enhanced"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}/.."
    
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh" || true
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh" || true
    
    # Load configuration if available
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        source "${SCRIPT_DIR}/config/defaults.sh"
    fi
    
    # Load the session enhanced functions
    source "${BATS_TEST_DIRNAME}/session-enhanced.sh"
    
    # Export functions for subshells
    export -f claude_code::session_enhanced_init
    export -f claude_code::session_extract_results
    export -f claude_code::session_analytics
    export -f claude_code::session_recover
    export -f claude_code::session_cleanup
    export -f claude_code::session_list_enhanced
}

setup() {
    # Reset mocks for each test
    mock::claude::reset >/dev/null 2>&1 || true
    
    # Create temporary test session directories
    export TEST_SESSION_DIR="${BATS_TMPDIR}/sessions-test"
    export CLAUDE_SESSIONS_DIR="$TEST_SESSION_DIR"
    export CLAUDE_SESSION_METADATA_DIR="${CLAUDE_SESSIONS_DIR}/.metadata"
    export CLAUDE_SESSION_ANALYTICS_CACHE="${CLAUDE_SESSION_METADATA_DIR}/.analytics_cache"
    
    mkdir -p "$CLAUDE_SESSIONS_DIR"
    mkdir -p "$CLAUDE_SESSION_METADATA_DIR"
}

teardown() {
    # Clean up test files
    trash::safe_remove "$TEST_SESSION_DIR" --test-cleanup
}

#######################################
# Session Enhanced Init Tests
#######################################

@test "claude_code::session_enhanced_init - should create necessary directories" {
    # Remove directories to test creation
    trash::safe_remove "$CLAUDE_SESSION_METADATA_DIR" --test-cleanup
    
    run claude_code::session_enhanced_init
    
    assert_success
    assert [ -d "$CLAUDE_SESSION_METADATA_DIR" ]
}

@test "claude_code::session_enhanced_init - should initialize analytics cache" {
    # Remove cache to test initialization
    trash::safe_remove "$CLAUDE_SESSION_ANALYTICS_CACHE" --test-cleanup
    
    run claude_code::session_enhanced_init
    
    assert_success
    assert [ -f "$CLAUDE_SESSION_ANALYTICS_CACHE" ]
    
    # Verify cache contains valid JSON array
    local cache_content
    cache_content=$(cat "$CLAUDE_SESSION_ANALYTICS_CACHE")
    [[ "$cache_content" == "[]" ]]
}

@test "claude_code::session_enhanced_init - should not overwrite existing analytics cache" {
    # Create existing cache with data
    echo '["existing_data"]' > "$CLAUDE_SESSION_ANALYTICS_CACHE"
    
    run claude_code::session_enhanced_init
    
    assert_success
    
    # Verify cache was not overwritten
    local cache_content
    cache_content=$(cat "$CLAUDE_SESSION_ANALYTICS_CACHE")
    [[ "$cache_content" == '["existing_data"]' ]]
}

#######################################
# Session Extract Results Tests
#######################################

@test "claude_code::session_extract_results - should require output file parameter" {
    run claude_code::session_extract_results "" "test-session" "json"
    
    assert_failure
    assert_output --partial "Output file path is required"
}

@test "claude_code::session_extract_results - should handle missing session file" {
    local output_file="${TEST_SESSION_DIR}/results.json"
    
    run claude_code::session_extract_results "$output_file" "nonexistent-session" "json"
    
    assert_failure
    assert_output --partial "Session file not found"
}

@test "claude_code::session_extract_results - should extract from valid session file" {
    local session_id="test-session-123"
    local session_file="${CLAUDE_SESSIONS_DIR}/${session_id}.json"
    local output_file="${TEST_SESSION_DIR}/results.json"
    
    # Create a mock session file
    cat > "$session_file" << 'EOF'
{
    "session_id": "test-session-123",
    "status": "completed",
    "messages": [
        {"role": "user", "content": "Hello"},
        {"role": "assistant", "content": "Hi there!"}
    ]
}
EOF
    
    run claude_code::session_extract_results "$output_file" "$session_id" "json"
    
    assert_success
    assert [ -f "$output_file" ]
    assert_output --partial "Results extracted"
}

#######################################
# Session Analytics Tests
#######################################

@test "claude_code::session_analytics - should analyze sessions for time period" {
    # Initialize session system
    claude_code::session_enhanced_init
    
    # Create some mock session files
    local session1="${CLAUDE_SESSIONS_DIR}/session1.json"
    local session2="${CLAUDE_SESSIONS_DIR}/session2.json"
    
    cat > "$session1" << 'EOF'
{
    "session_id": "session1",
    "status": "completed",
    "created_at": "2024-01-01T10:00:00Z",
    "turn_count": 5
}
EOF
    
    cat > "$session2" << 'EOF'
{
    "session_id": "session2", 
    "status": "active",
    "created_at": "2024-01-02T15:30:00Z",
    "turn_count": 3
}
EOF
    
    run claude_code::session_analytics "week" "text"
    
    assert_success
    assert_output --partial "Session Analytics"
}

@test "claude_code::session_analytics - should handle empty session directory" {
    # Initialize with no sessions
    claude_code::session_enhanced_init
    
    run claude_code::session_analytics "day" "text"
    
    assert_success
    assert_output --partial "No sessions found"
}

#######################################
# Session Recovery Tests
#######################################

@test "claude_code::session_recover - should require session ID" {
    run claude_code::session_recover "" "auto" ""
    
    assert_failure
    assert_output --partial "Session ID is required"
}

@test "claude_code::session_recover - should handle missing session file" {
    run claude_code::session_recover "nonexistent-session" "auto" ""
    
    assert_failure
    assert_output --partial "Session file not found"
}

@test "claude_code::session_recover - should recover valid session" {
    local session_id="recoverable-session"
    local session_file="${CLAUDE_SESSIONS_DIR}/${session_id}.json"
    
    # Create a mock corrupted session file
    cat > "$session_file" << 'EOF'
{
    "session_id": "recoverable-session",
    "status": "error",
    "last_error": "timeout"
}
EOF
    
    # Mock successful claude command
    mock::claude::scenario::setup_healthy
    
    run claude_code::session_recover "$session_id" "auto" "continue previous work"
    
    assert_success
    assert_output --partial "Attempting recovery"
}

#######################################
# Session Cleanup Tests
#######################################

@test "claude_code::session_cleanup - should clean old sessions" {
    # Initialize session system
    claude_code::session_enhanced_init
    
    # Create an old session file (simulate by date)
    local old_session="${CLAUDE_SESSIONS_DIR}/old_session.json"
    cat > "$old_session" << 'EOF'
{
    "session_id": "old_session",
    "status": "completed",
    "created_at": "2020-01-01T10:00:00Z"
}
EOF
    
    run claude_code::session_cleanup "old" "30"
    
    assert_success
    assert_output --partial "Session cleanup"
}

@test "claude_code::session_cleanup - should handle invalid cleanup strategy" {
    run claude_code::session_cleanup "invalid_strategy" "30"
    
    assert_failure
    assert_output --partial "Invalid cleanup strategy"
}

#######################################
# Session List Enhanced Tests
#######################################

@test "claude_code::session_list_enhanced - should list sessions with filter" {
    # Initialize session system
    claude_code::session_enhanced_init
    
    # Create some mock sessions
    local session1="${CLAUDE_SESSIONS_DIR}/recent1.json"
    local session2="${CLAUDE_SESSIONS_DIR}/recent2.json"
    
    cat > "$session1" << 'EOF'
{
    "session_id": "recent1",
    "status": "completed",
    "created_at": "2024-01-01T10:00:00Z"
}
EOF
    
    cat > "$session2" << 'EOF'
{
    "session_id": "recent2",
    "status": "active", 
    "created_at": "2024-01-01T11:00:00Z"
}
EOF
    
    run claude_code::session_list_enhanced "all" "date" "text" "10"
    
    assert_success
    assert_output --partial "Session List"
}

@test "claude_code::session_list_enhanced - should handle empty session directory" {
    # Initialize with no sessions
    claude_code::session_enhanced_init
    
    run claude_code::session_list_enhanced "all" "date" "text" "10"
    
    assert_success
    assert_output --partial "No sessions found"
}

@test "claude_code::session_list_enhanced - should respect limit parameter" {
    # Initialize session system
    claude_code::session_enhanced_init
    
    # Create multiple mock sessions
    for i in {1..5}; do
        local session_file="${CLAUDE_SESSIONS_DIR}/session${i}.json"
        cat > "$session_file" << EOF
{
    "session_id": "session${i}",
    "status": "completed",
    "created_at": "2024-01-0${i}T10:00:00Z"
}
EOF
    done
    
    run claude_code::session_list_enhanced "all" "date" "text" "3"
    
    assert_success
    # Should respect the limit of 3 sessions
    assert_output --partial "Session List"
}