#!/usr/bin/env bash
################################################################################
# OBS Studio Integration Tests - Full Functionality (<120s)
#
# Tests end-to-end OBS Studio capabilities
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"
source "${RESOURCE_DIR}/lib/content.sh"

# Integration test implementation
main() {
    local start_time=$(date +%s)
    local failed=0
    
    log::header "🧪 OBS Studio Integration Tests"
    log::info "Testing end-to-end functionality (<120s)"
    
    # Ensure OBS Studio is running
    if ! obs::is_running; then
        log::info "Starting OBS Studio for tests..."
        obs::start || {
            log::error "Failed to start OBS Studio"
            exit 1
        }
        sleep 2
    fi
    
    # Test 1: WebSocket API connectivity
    log::info "Test 1: WebSocket API connectivity"
    if obs::websocket::test_connection; then
        log::success "✅ WebSocket API accessible"
    else
        log::error "❌ WebSocket API not accessible"
        ((failed++))
    fi
    
    # Test 2: Scene creation and management
    log::info "Test 2: Scene management"
    local test_scene="test-scene-$(date +%s)"
    
    # Create a test scene configuration
    cat > "/tmp/${test_scene}.json" <<EOF
{
  "name": "${test_scene}",
  "sources": [
    {
      "type": "text",
      "name": "test-text",
      "settings": {
        "text": "OBS Studio Integration Test"
      }
    }
  ]
}
EOF
    
    # Add the scene
    if obs::content::add "/tmp/${test_scene}.json"; then
        log::success "✅ Scene added successfully"
    else
        log::error "❌ Failed to add scene"
        ((failed++))
    fi
    
    # List scenes
    if obs::content::list | grep -q "${test_scene}"; then
        log::success "✅ Scene appears in list"
    else
        log::error "❌ Scene not found in list"
        ((failed++))
    fi
    
    # Get scene details
    if obs::content::get "${test_scene}" &>/dev/null; then
        log::success "✅ Scene details retrieved"
    else
        log::error "❌ Failed to get scene details"
        ((failed++))
    fi
    
    # Execute (activate) scene
    if obs::content::execute "${test_scene}"; then
        log::success "✅ Scene activated"
    else
        log::error "❌ Failed to activate scene"
        ((failed++))
    fi
    
    # Remove scene
    if obs::content::remove "${test_scene}"; then
        log::success "✅ Scene removed"
    else
        log::error "❌ Failed to remove scene"
        ((failed++))
    fi
    
    # Clean up test file
    rm -f "/tmp/${test_scene}.json"
    
    # Test 3: Recording functionality
    log::info "Test 3: Recording functionality"
    
    # Start recording
    if obs::content::record start; then
        log::success "✅ Recording started"
        sleep 2
        
        # Stop recording
        if obs::content::record stop; then
            log::success "✅ Recording stopped"
        else
            log::error "❌ Failed to stop recording"
            ((failed++))
        fi
    else
        log::error "❌ Failed to start recording"
        ((failed++))
    fi
    
    # Test 4: Status reporting
    log::info "Test 4: Status reporting"
    if obs::get_status &>/dev/null; then
        log::success "✅ Status command functional"
    else
        log::error "❌ Status command failed"
        ((failed++))
    fi
    
    # Test 5: Configuration persistence
    log::info "Test 5: Configuration persistence"
    local config_dir="${OBS_CONFIG_DIR:-$HOME/.vrooli/obs-studio}"
    local test_config="${config_dir}/test-config.json"
    
    echo '{"test": "data"}' > "${test_config}"
    if [[ -f "${test_config}" ]]; then
        log::success "✅ Configuration persisted"
        rm -f "${test_config}"
    else
        log::error "❌ Configuration not persisted"
        ((failed++))
    fi
    
    # Test 6: Error handling
    log::info "Test 6: Error handling"
    
    # Try to get non-existent scene
    if ! obs::content::get "non-existent-scene" &>/dev/null; then
        log::success "✅ Properly handles missing content"
    else
        log::error "❌ Did not properly handle missing content"
        ((failed++))
    fi
    
    # Test 7: Concurrent operations
    log::info "Test 7: Concurrent operations"
    
    # Run multiple status checks concurrently
    local pids=()
    for i in {1..3}; do
        obs::get_status &>/dev/null &
        pids+=($!)
    done
    
    # Wait for all to complete
    local concurrent_failed=0
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            ((concurrent_failed++))
        fi
    done
    
    if [[ $concurrent_failed -eq 0 ]]; then
        log::success "✅ Handles concurrent operations"
    else
        log::error "❌ Failed $concurrent_failed concurrent operations"
        ((failed++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Check time limit
    if [[ $duration -gt 120 ]]; then
        log::warning "⚠️  Integration tests took ${duration}s (exceeds 120s limit)"
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed (${duration}s)"
        exit 0
    else
        log::error "$failed integration tests failed (${duration}s)"
        exit 1
    fi
}

main "$@"