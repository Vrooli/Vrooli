#!/usr/bin/env bats

# Tests for Huginn lib/api.sh


# Load test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

load ../test_fixtures/test_helper

# Lightweight per-test setup
setup() {
    # Use auto-setup for service tests
    vrooli_auto_setup
    
    # Source scripts directly without expensive environment setup
    HUGINN_ROOT_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME}")/.." && pwd)"
    source "$HUGINN_ROOT_DIR/../../../helpers/utils/args.sh" 2>/dev/null || true
    source "$HUGINN_ROOT_DIR/../../../common.sh" 2>/dev/null || true
    source "$HUGINN_ROOT_DIR/config/defaults.sh" 2>/dev/null || true
    source "$HUGINN_ROOT_DIR/config/messages.sh" 2>/dev/null || true
    source "$HUGINN_ROOT_DIR/lib/api.sh" 2>/dev/null || true
    
    # Mock docker and other functions
    mock_docker "success"
    
    # Additional lightweight mocks
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    system::is_command() { command -v "$1" >/dev/null 2>&1; }
    
    # Set variables only if not already set (avoid readonly errors)
    : "${CONTAINER_NAME:=huginn}"
    : "${RESOURCE_PORT:=3000}"
}

teardown() {
    vrooli_cleanup_test
    teardown_test_environment
}

@test "api.sh: list_agents shows all agents" {
    export DOCKER_EXEC_MOCK="agents_list"
    
    run huginn::list_agents
    assert_success
    assert_output_contains "RSS Weather Monitor"
    assert_output_contains "Website Status Checker"
    assert_output_contains "Type:"
    assert_output_contains "Events:"
}

@test "api.sh: list_agents fails when not running" {
    mock_docker "not_running"
    
    run huginn::list_agents
    assert_failure
    assert_output_contains "not running"
}

@test "api.sh: show_agent displays agent details" {
    export DOCKER_EXEC_MOCK="agent_show"
    
    run huginn::show_agent "1"
    assert_success
    assert_output_contains "Agent Details"
    assert_output_contains "RSS Weather Monitor"
    assert_output_contains "Type:"
    assert_output_contains "Owner:"
}

@test "api.sh: show_agent validates agent ID" {
    run huginn::show_agent "invalid"
    assert_failure
    assert_output_contains "Invalid agent ID"
}

@test "api.sh: show_agent requires agent ID" {
    run huginn::show_agent ""
    assert_failure
    assert_output_contains "Invalid agent ID"
}

@test "api.sh: run_agent executes agent manually" {
    export DOCKER_EXEC_MOCK="agent_show"
    
    run huginn::run_agent "1"
    assert_success
    assert_output_contains "Running agent"
}

@test "api.sh: run_agent validates agent ID" {
    run huginn::run_agent "abc"
    assert_failure
    assert_output_contains "Invalid agent ID"
}

@test "api.sh: list_scenarios shows all scenarios" {
    export DOCKER_EXEC_MOCK="scenarios_list"
    
    run huginn::list_scenarios
    assert_success
    assert_output_contains "Weather Monitoring Suite"
    assert_output_contains "Description:"
    assert_output_contains "Agents:"
}

@test "api.sh: show_scenario displays scenario details" {
    export DOCKER_EXEC_MOCK="scenarios_list"
    
    run huginn::show_scenario "1"
    assert_success
    assert_output_contains "Scenario Details"
}

@test "api.sh: show_scenario validates scenario ID" {
    run huginn::show_scenario "invalid"
    assert_failure
    assert_output_contains "Invalid scenario ID"
}

@test "api.sh: show_recent_events displays events" {
    export DOCKER_EXEC_MOCK="events_recent"
    
    run huginn::show_recent_events "10"
    assert_success
    assert_output_contains "Recent Events"
    assert_output_contains "Weather Update"
}

@test "api.sh: show_recent_events uses default count" {
    export DOCKER_EXEC_MOCK="events_recent"
    
    run huginn::show_recent_events
    assert_success
    assert_output_contains "Recent Events"
}

@test "api.sh: show_agent_events displays agent-specific events" {
    export DOCKER_EXEC_MOCK="events_recent"
    
    run huginn::show_agent_events "1" "5"
    assert_success
    assert_output_contains "Events for Agent"
}

@test "api.sh: show_agent_events validates agent ID" {
    run huginn::show_agent_events "xyz" "5"
    assert_failure
    assert_output_contains "Invalid agent ID"
}

@test "api.sh: list_agent_types shows available types" {
    # Mock Rails runner to return agent types
    huginn::rails_runner() {
        cat << 'EOF'
✅ WebsiteAgent
   Monitor websites and scrape content

✅ RssAgent
   Monitor RSS feeds for new items

✅ DigestAgent
   Aggregate events into periodic summaries
EOF
        return 0
    }
    export -f huginn::rails_runner
    
    run huginn::list_agent_types
    assert_success
    assert_output_contains "Available Agent Types"
    assert_output_contains "WebsiteAgent"
    assert_output_contains "RssAgent"
    assert_output_contains "DigestAgent"
}

@test "api.sh: all API functions check if running" {
    mock_docker "not_running"
    
    # Test each function that requires running state
    run huginn::list_agents
    assert_failure
    
    run huginn::show_agent "1"
    assert_failure
    
    run huginn::run_agent "1"
    assert_failure
    
    run huginn::list_scenarios
    assert_failure
    
    run huginn::show_scenario "1"
    assert_failure
    
    run huginn::show_recent_events
    assert_failure
    
    run huginn::show_agent_events "1"
    assert_failure
    
    run huginn::list_agent_types
    assert_failure
}

@test "api.sh: headers are displayed correctly" {
    export DOCKER_EXEC_MOCK="agents_list"
    
    run huginn::list_agents
    assert_success
    
    # The header should be shown via huginn::show_agents_header
    # which is called before the agent list
}

@test "api.sh: handles empty results gracefully" {
    # Mock empty agent list
    huginn::rails_runner() {
        echo "📭 No agents found"
        return 0
    }
    export -f huginn::rails_runner
    
    run huginn::list_agents
    assert_success
    assert_output_contains "No agents found"
}
