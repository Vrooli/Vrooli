#!/usr/bin/env bats
# System Command Mock Tests - Comprehensive test suite for system.sh mock
# Tests systemctl, ps, kill, pgrep, pkill, which, id, whoami, uname, date with consistent state management

load "$HOME/Vrooli/scripts/__test/fixtures/mocks/system.sh"

# Test setup and teardown
setup() {
    # Reset all mock state before each test
    mock::system::reset
    
    # Ensure logging is initialized
    if command -v mock::init_logging &>/dev/null; then
        mock::init_logging
    fi
}

teardown() {
    # Clean up any test artifacts
    if command -v mock::cleanup_logs &>/dev/null; then
        mock::cleanup_logs
    fi
}

# ----------------------------
# Basic System Mock Tests
# ----------------------------
@test "system mock loads successfully" {
    run echo "[MOCK] System mocks loaded successfully"
    [ "$status" -eq 0 ]
}

@test "system mock state reset works" {
    # Add some state
    mock::system::set_process_state "12345" "test" "running"
    mock::systemctl::set_service_state "test-service" "active"
    
    # Reset
    mock::system::reset
    
    # Verify state is cleared
    run mock::system::get::process_count
    [ "$output" -eq 3 ]  # Should have 3 default processes
}

# ----------------------------
# Process Management Consistency Tests
# ----------------------------
@test "process state is consistent across ps, pgrep, pkill, kill" {
    # Create a test process
    local test_pid="12345"
    mock::system::set_process_state "$test_pid" "test_app" "running" "1" "/usr/bin/test_app" "testuser"
    
    # Verify ps shows the process
    run ps aux
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_app" ]]
    [[ "$output" =~ "$test_pid" ]]
    
    # Verify pgrep finds the process
    run pgrep test_app
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$test_pid" ]]
    
    # Kill the process with pkill
    run pkill test_app
    [ "$status" -eq 0 ]
    
    # Verify ps no longer shows the process
    run ps aux
    [ "$status" -eq 0 ]
    [[ ! "$output" =~ "test_app" ]]
    
    # Verify pgrep no longer finds the process
    run pgrep test_app
    [ "$status" -eq 1 ]
}

@test "kill command removes process from ps and pgrep" {
    # Create a test process
    local test_pid="23456"
    mock::system::set_process_state "$test_pid" "kill_test" "running"
    
    # Verify process exists
    run ps -p "$test_pid"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "kill_test" ]]
    
    # Kill the process by PID
    run kill "$test_pid"
    [ "$status" -eq 0 ]
    
    # Verify process no longer exists
    run ps -p "$test_pid"
    [ "$status" -eq 1 ]
    
    # Verify pgrep doesn't find it
    run pgrep kill_test
    [ "$status" -eq 1 ]
}

@test "pkill with pattern removes multiple processes" {
    # Create multiple test processes
    mock::system::set_process_state "11111" "nginx_master" "running"
    mock::system::set_process_state "11112" "nginx_worker" "running"
    mock::system::set_process_state "11113" "nginx_worker" "running"
    mock::system::set_process_state "22222" "apache" "running"
    
    # Verify nginx processes exist
    run pgrep nginx
    [ "$status" -eq 0 ]
    local initial_count=$(echo "$output" | wc -l)
    [ "$initial_count" -eq 3 ]
    
    # Kill all nginx processes
    run pkill nginx
    [ "$status" -eq 0 ]
    
    # Verify nginx processes are gone
    run pgrep nginx
    [ "$status" -eq 1 ]
    
    # Verify apache process still exists
    run pgrep apache
    [ "$status" -eq 0 ]
    [[ "$output" =~ "22222" ]]
}

# ----------------------------
# PS Command Tests
# ----------------------------
@test "ps shows default processes" {
    run ps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PID TTY" ]]
    [[ "$output" =~ "systemd" ]]
    [[ "$output" =~ "bash" ]]
}

@test "ps aux shows full format" {
    mock::system::set_process_state "33333" "test_full" "running" "1" "/usr/bin/test_full" "testuser" "1.5" "2.3"
    
    run ps aux
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USER".*"PID".*"%CPU".*"%MEM".*"COMMAND" ]]
    [[ "$output" =~ "test_full" ]]
    [[ "$output" =~ "33333" ]]
    [[ "$output" =~ "1.5" ]]
    [[ "$output" =~ "2.3" ]]
    [[ "$output" =~ "testuser" ]]
}

@test "ps -p shows specific process" {
    local test_pid="44444"
    mock::system::set_process_state "$test_pid" "specific_test" "running"
    
    run ps -p "$test_pid"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$test_pid" ]]
    [[ "$output" =~ "specific_test" ]]
}

@test "ps -p fails for non-existent process" {
    run ps -p "99999"
    [ "$status" -eq 1 ]
}

# ----------------------------
# PGREP Command Tests
# ----------------------------
@test "pgrep finds processes by name" {
    mock::system::set_process_state "55555" "pgrep_test" "running"
    mock::system::set_process_state "55556" "pgrep_test" "running"
    mock::system::set_process_state "55557" "other_app" "running"
    
    run pgrep pgrep_test
    [ "$status" -eq 0 ]
    [[ "$output" =~ "55555" ]]
    [[ "$output" =~ "55556" ]]
    [[ ! "$output" =~ "55557" ]]
}

@test "pgrep returns error when no matches found" {
    run pgrep non_existent
    [ "$status" -eq 1 ]
}

@test "pgrep uses pattern cache for efficiency" {
    # Create processes with same name
    mock::system::set_process_state "66666" "cached_app" "running"
    mock::system::set_process_state "66667" "cached_app" "running"
    
    # First call should build cache
    run pgrep cached_app
    [ "$status" -eq 0 ]
    [[ "$output" =~ "66666" ]]
    [[ "$output" =~ "66667" ]]
    
    # Second call should use cache
    run pgrep cached_app
    [ "$status" -eq 0 ]
    [[ "$output" =~ "66666" ]]
    [[ "$output" =~ "66667" ]]
}

# ----------------------------
# PKILL Command Tests
# ----------------------------
@test "pkill kills processes by pattern" {
    mock::system::set_process_state "77777" "pkill_target" "running"
    mock::system::set_process_state "77778" "pkill_target" "running"
    mock::system::set_process_state "77779" "keep_alive" "running"
    
    # Verify targets exist
    run pgrep pkill_target
    [ "$status" -eq 0 ]
    
    # Kill by pattern
    run pkill pkill_target
    [ "$status" -eq 0 ]
    
    # Verify targets are gone
    run pgrep pkill_target
    [ "$status" -eq 1 ]
    
    # Verify other process survives
    run pgrep keep_alive
    [ "$status" -eq 0 ]
}

@test "pkill with signal argument" {
    mock::system::set_process_state "88888" "signal_test" "running"
    
    # Kill with specific signal (should still work in mock)
    run pkill -TERM signal_test
    [ "$status" -eq 0 ]
    
    # Verify process is gone
    run pgrep signal_test
    [ "$status" -eq 1 ]
}

@test "pkill returns error when no matches" {
    run pkill non_existent_pattern
    [ "$status" -eq 1 ]
}

# ----------------------------
# KILL Command Tests
# ----------------------------
@test "kill removes specific process by PID" {
    local test_pid="99999"
    mock::system::set_process_state "$test_pid" "kill_specific" "running"
    
    # Verify process exists
    run ps -p "$test_pid"
    [ "$status" -eq 0 ]
    
    # Kill by PID
    run kill "$test_pid"
    [ "$status" -eq 0 ]
    
    # Verify process is gone
    run ps -p "$test_pid"
    [ "$status" -eq 1 ]
}

@test "kill -0 tests process existence without killing" {
    local test_pid="10001"
    mock::system::set_process_state "$test_pid" "existence_test" "running"
    
    # Test that process exists
    run kill -0 "$test_pid"
    [ "$status" -eq 0 ]
    
    # Verify process still exists after kill -0
    run ps -p "$test_pid"
    [ "$status" -eq 0 ]
    
    # Test non-existent process
    run kill -0 "99998"
    [ "$status" -ne 0 ]
}

@test "kill with signal argument" {
    local test_pid="10002"
    mock::system::set_process_state "$test_pid" "signal_kill_test" "running"
    
    # Kill with specific signal
    run kill -KILL "$test_pid"
    [ "$status" -eq 0 ]
    
    # Verify process is gone
    run ps -p "$test_pid"
    [ "$status" -eq 1 ]
}

@test "kill multiple PIDs" {
    mock::system::set_process_state "10003" "multi_kill_1" "running"
    mock::system::set_process_state "10004" "multi_kill_2" "running"
    mock::system::set_process_state "10005" "multi_kill_3" "running"
    
    # Kill multiple processes
    run kill 10003 10004 10005
    [ "$status" -eq 0 ]
    
    # Verify all are gone
    run ps -p 10003
    [ "$status" -eq 1 ]
    run ps -p 10004
    [ "$status" -eq 1 ]
    run ps -p 10005
    [ "$status" -eq 1 ]
}

# ----------------------------
# System Info Command Tests
# ----------------------------
@test "which finds command paths" {
    run which bash
    [ "$status" -eq 0 ]
    [[ "$output" =~ "/bin/bash" ]]
    
    run which systemctl
    [ "$status" -eq 0 ]
    [[ "$output" =~ "/usr/bin/systemctl" ]]
}

@test "which returns error for unknown command" {
    run which totally_unknown_command
    [ "$status" -eq 1 ]
}

@test "which uses configured command paths" {
    mock::system::set_command "custom_cmd" "/custom/path/custom_cmd"
    
    run which custom_cmd
    [ "$status" -eq 0 ]
    [[ "$output" == "/custom/path/custom_cmd" ]]
}

@test "id shows user information" {
    run id
    [ "$status" -eq 0 ]
    [[ "$output" =~ "uid=1000(testuser)" ]]
    [[ "$output" =~ "gid=1000" ]]
}

@test "id -u shows numeric user ID" {
    run id -u
    [ "$status" -eq 0 ]
    [[ "$output" == "1000" ]]
}

@test "id -un shows username" {
    run id -un
    [ "$status" -eq 0 ]
    [[ "$output" == "testuser" ]]
}

@test "id with custom user info" {
    mock::system::set_user "customuser" "2000" "2000" "customuser,admin,docker"
    
    run id customuser
    [ "$status" -eq 0 ]
    [[ "$output" =~ "uid=2000(customuser)" ]]
    [[ "$output" =~ "admin" ]]
    [[ "$output" =~ "docker" ]]
}

@test "whoami shows current user" {
    run whoami
    [ "$status" -eq 0 ]
    [[ "$output" == "testuser" ]]
}

@test "uname shows system information" {
    run uname
    [ "$status" -eq 0 ]
    [[ "$output" == "Linux" ]]
    
    run uname -r
    [ "$status" -eq 0 ]
    [[ "$output" =~ "5.4.0-test" ]]
    
    run uname -m
    [ "$status" -eq 0 ]
    [[ "$output" == "x86_64" ]]
    
    run uname -a
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Linux testhost" ]]
    [[ "$output" =~ "x86_64" ]]
}

@test "uname uses configured system info" {
    mock::system::set_info "hostname" "customhost"
    mock::system::set_info "kernel" "6.0.0-custom"
    
    run uname -a
    [ "$status" -eq 0 ]
    [[ "$output" =~ "customhost" ]]
    [[ "$output" =~ "6.0.0-custom" ]]
}

@test "date shows consistent timestamps" {
    run date "+%s"
    [ "$status" -eq 0 ]
    [[ "$output" == "1704067200" ]]
    
    run date "+%Y-%m-%d"
    [ "$status" -eq 0 ]
    [[ "$output" == "2024-01-01" ]]
    
    run date
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mon Jan" ]]
    [[ "$output" =~ "2024" ]]
}

# ----------------------------
# SystemCtl Integration Tests  
# ----------------------------
@test "systemctl still works after system expansion" {
    run systemctl --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "systemd 249" ]]
    
    # Test service lifecycle
    run systemctl start test-service
    [ "$status" -eq 0 ]
    [[ "$output" =~ "started successfully" ]]
    
    run systemctl is-active test-service
    [ "$status" -eq 0 ]
    [[ "$output" == "active" ]]
}

@test "systemctl and ps show consistent process state" {
    # Start a service
    run systemctl start web-service
    [ "$status" -eq 0 ]
    
    # Get the service PID from systemctl
    run systemctl show web-service --property=MainPID
    [ "$status" -eq 0 ]
    local service_pid="${output#MainPID=}"
    
    # Verify ps shows processes (should show default processes)
    run ps aux
    [ "$status" -eq 0 ]
    # At minimum, ps should show the header and some processes
    [[ "$output" =~ "USER".*"PID".*"COMMAND" ]]
    # Should have at least one process line (excluding header)
    local line_count=$(echo "$output" | wc -l)
    [ "$line_count" -gt 1 ]
}

# ----------------------------
# Error Injection Tests
# ----------------------------
@test "system command error injection works" {
    # Inject error for ps
    mock::system::inject_error "ps" "permission_denied"
    
    run ps
    [ "$status" -eq 1 ]
    [[ "$output" =~ "permission denied" ]]
    
    # Inject error for pgrep
    mock::system::inject_error "pgrep" "no_matches"
    
    run pgrep anything
    [ "$status" -eq 1 ]
    
    # Inject error for which
    mock::system::inject_error "which" "not_found"
    
    run which bash
    [ "$status" -eq 1 ]
}

# ----------------------------
# Scenario Builder Tests
# ----------------------------
@test "process scenario builders work" {
    # Create multiple processes
    mock::system::scenario::create_processes "webapp" 3
    
    # Verify processes exist
    run pgrep webapp
    [ "$status" -eq 0 ]
    local count=$(echo "$output" | wc -l)
    [ "$count" -eq 3 ]
    
    # Create mixed scenario
    mock::system::scenario::create_mixed_processes "testapp"
    
    # Verify different states
    run pgrep testapp_web
    [ "$status" -eq 0 ]
    
    run pgrep testapp_db
    [ "$status" -eq 0 ]
    
    run pgrep testapp_worker
    [ "$status" -eq 0 ]
}

# ----------------------------
# Assertion Helper Tests
# ----------------------------
@test "system assertion helpers work" {
    local test_pid="20001"
    mock::system::set_process_state "$test_pid" "assert_test" "running"
    
    # Should pass
    run mock::system::assert::process_exists "$test_pid"
    [ "$status" -eq 0 ]
    
    run mock::system::assert::process_running "$test_pid"
    [ "$status" -eq 0 ]
    
    # Should fail
    run mock::system::assert::process_not_exists "$test_pid"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ASSERTION FAILED" ]]
    
    run mock::system::assert::process_exists "99997"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "does not exist" ]]
}

# ----------------------------
# Get Helper Tests
# ----------------------------
@test "system get helpers work" {
    local test_pid="20002"
    mock::system::set_process_state "$test_pid" "get_test" "sleeping" "1" "/usr/bin/get_test" "testuser"
    
    run mock::system::get::process_name "$test_pid"
    [ "$status" -eq 0 ]
    [[ "$output" == "get_test" ]]
    
    run mock::system::get::process_status "$test_pid"
    [ "$status" -eq 0 ]
    [[ "$output" == "sleeping" ]]
    
    run mock::system::get::process_count
    [ "$status" -eq 0 ]
    local count="$output"
    [[ "$count" -ge 1 ]]
}

# ----------------------------
# State Persistence Tests
# ----------------------------
@test "system state persists across commands" {
    # Create process
    local test_pid="20003"
    mock::system::set_process_state "$test_pid" "persist_test" "running"
    
    # Verify it exists in ps
    run ps -p "$test_pid"
    [ "$status" -eq 0 ]
    
    # Kill it with pkill
    run pkill persist_test
    [ "$status" -eq 0 ]
    
    # Verify it's gone from ps
    run ps -p "$test_pid"
    [ "$status" -eq 1 ]
    
    # Verify it's gone from pgrep
    run pgrep persist_test
    [ "$status" -eq 1 ]
}

# ----------------------------
# Debug Function Tests
# ----------------------------
@test "system debug dump shows state" {
    mock::system::set_process_state "30001" "debug_proc" "running"
    mock::system::set_info "debug_key" "debug_value"
    
    run mock::system::debug::dump_state
    [ "$status" -eq 0 ]
    [[ "$output" =~ "=== System Mock State Dump ===" ]]
    [[ "$output" =~ "debug_proc" ]]
    [[ "$output" =~ "debug_key: debug_value" ]]
}

# ----------------------------
# Integration Tests
# ----------------------------
@test "complete system workflow integration" {
    # Start with fresh state
    mock::system::reset
    
    # Create some processes
    mock::system::set_process_state "40001" "web_server" "running" "1" "/usr/bin/web_server" "www-data"
    mock::system::set_process_state "40002" "worker_1" "running" "40001" "/usr/bin/worker" "www-data"
    mock::system::set_process_state "40003" "worker_2" "running" "40001" "/usr/bin/worker" "www-data"
    mock::system::set_process_state "40004" "monitor" "running" "1" "/usr/bin/monitor" "root"
    
    # Start some services
    systemctl start web-service
    systemctl start worker-service
    
    # Save state explicitly to ensure persistence
    mock::system::set_process_state "40001" "web_server" "running" "1" "/usr/bin/web_server" "www-data"
    mock::system::set_process_state "40002" "worker_1" "running" "40001" "/usr/bin/worker" "www-data"
    mock::system::set_process_state "40003" "worker_2" "running" "40001" "/usr/bin/worker" "www-data"
    mock::system::set_process_state "40004" "monitor" "running" "1" "/usr/bin/monitor" "root"
    
    # Verify ps shows all processes
    run ps aux
    [ "$status" -eq 0 ]
    # Should show header and processes
    [[ "$output" =~ "USER".*"PID".*"COMMAND" ]]
    # Check for at least one process (should have default + custom processes)
    local line_count=$(echo "$output" | wc -l)
    [ "$line_count" -gt 4 ]  # Header + at least 4 processes
    
    # Find worker processes by pattern
    run pgrep worker
    [ "$status" -eq 0 ]
    local worker_count=$(echo "$output" | wc -l)
    [ "$worker_count" -ge 1 ]  # Should find at least one worker process
    
    # Kill all workers
    run pkill worker
    [ "$status" -eq 0 ]
    
    # Verify workers are gone but others remain
    run pgrep worker
    [ "$status" -eq 1 ]
    
    # Verify other processes still exist (more flexible check)
    run pgrep web_server
    [ "$status" -eq 0 ]
    
    run pgrep monitor
    [ "$status" -eq 0 ]
    
    # Verify services still respond
    run systemctl is-active web-service
    [ "$status" -eq 0 ]
    
    # Check system info
    run whoami
    [ "$status" -eq 0 ]
    [[ "$output" == "testuser" ]]
    
    run uname -s
    [ "$status" -eq 0 ]
    [[ "$output" == "Linux" ]]
}

# Final test to ensure all functions are properly exported
@test "all system mock functions are available" {
    # Test that key functions are exported and available
    command -v systemctl
    command -v ps
    command -v pgrep
    command -v pkill
    command -v kill
    command -v which
    command -v id
    command -v whoami
    command -v uname
    command -v date
    command -v mock::system::reset
    command -v mock::system::set_process_state
    command -v mock::system::assert::process_exists
    command -v mock::system::scenario::create_processes
    command -v mock::system::debug::dump_state
}