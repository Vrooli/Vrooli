#!/usr/bin/env bats

# Source the port registry
SCRIPT_PATH="$BATS_TEST_DIRNAME/port-registry.sh"
. "$SCRIPT_PATH"

# ============================================================================
# Basic Port Retrieval Tests
# ============================================================================

@test "ports::get_vrooli_port returns correct port for UI service" {
    [ "$(ports::get_vrooli_port "ui")" = "3000" ]
}

@test "ports::get_vrooli_port returns correct port for API service" {
    [ "$(ports::get_vrooli_port "api")" = "5329" ]
}

@test "ports::get_vrooli_port returns correct port for postgres" {
    [ "$(ports::get_vrooli_port "postgres")" = "5432" ]
}

@test "ports::get_vrooli_port returns empty for invalid service" {
    [ "$(ports::get_vrooli_port "nonexistent")" = "" ]
}

@test "ports::get_resource_port returns correct port for ollama" {
    [ "$(ports::get_resource_port "ollama")" = "11434" ]
}

@test "ports::get_resource_port returns correct port for browserless" {
    [ "$(ports::get_resource_port "browserless")" = "4110" ]
}

@test "ports::get_resource_port returns correct port for n8n" {
    [ "$(ports::get_resource_port "n8n")" = "5678" ]
}

@test "ports::get_resource_port returns empty for invalid resource" {
    [ "$(ports::get_resource_port "nonexistent")" = "" ]
}

# ============================================================================
# Port Reservation Tests
# ============================================================================

@test "ports::is_vrooli_reserved detects UI port as reserved" {
    run ports::is_vrooli_reserved "3000"
    [ "$status" -eq 0 ]
}

@test "ports::is_vrooli_reserved detects API port as reserved" {
    run ports::is_vrooli_reserved "5329"
    [ "$status" -eq 0 ]
}

@test "ports::is_vrooli_reserved detects Adminer port as reserved" {
    run ports::is_vrooli_reserved "8080"
    [ "$status" -eq 0 ]
}

@test "ports::is_vrooli_reserved detects port 3500 in core range" {
    run ports::is_vrooli_reserved "3500"
    [ "$status" -eq 0 ]
}

@test "ports::is_vrooli_reserved detects port 9250 in debug range" {
    run ports::is_vrooli_reserved "9250"
    [ "$status" -eq 0 ]
}

@test "ports::is_vrooli_reserved returns false for ollama port" {
    run ports::is_vrooli_reserved "11434"
    [ "$status" -eq 1 ]
}

@test "ports::is_vrooli_reserved returns false for n8n port" {
    run ports::is_vrooli_reserved "5678"
    [ "$status" -eq 1 ]
}

@test "ports::is_vrooli_reserved returns false for random high port" {
    run ports::is_vrooli_reserved "50000"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Port Validation Tests
# ============================================================================

@test "ports::validate_assignment accepts valid ollama port" {
    run ports::validate_assignment "11434" "ollama"
    [ "$status" -eq 0 ]
}

@test "ports::validate_assignment accepts valid n8n port" {
    run ports::validate_assignment "5678" "n8n"
    [ "$status" -eq 0 ]
}

@test "ports::validate_assignment accepts valid custom port" {
    run ports::validate_assignment "50000" "custom"
    [ "$status" -eq 0 ]
}

@test "ports::validate_assignment rejects port 0" {
    run ports::validate_assignment "0" "test"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid port number"* ]]
}

@test "ports::validate_assignment rejects port 65536" {
    run ports::validate_assignment "65536" "test"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid port number"* ]]
}

@test "ports::validate_assignment rejects non-numeric port" {
    run ports::validate_assignment "abc" "test"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid port number"* ]]
}

@test "ports::validate_assignment rejects Vrooli UI port" {
    run ports::validate_assignment "3000" "custom"
    [ "$status" -eq 1 ]
    [[ "$output" == *"conflicts with Vrooli services"* ]]
}

@test "ports::validate_assignment rejects Vrooli API port" {
    run ports::validate_assignment "5329" "custom"
    [ "$status" -eq 1 ]
    [[ "$output" == *"conflicts with Vrooli services"* ]]
}

@test "ports::validate_assignment warns about privileged port 80" {
    run ports::validate_assignment "80" "web"
    [ "$status" -eq 0 ]
    [[ "$output" == *"requires root privileges"* ]]
}

@test "ports::validate_assignment warns about privileged port 443" {
    run ports::validate_assignment "443" "https"
    [ "$status" -eq 0 ]
    [[ "$output" == *"requires root privileges"* ]]
}

# ============================================================================
# Port Discovery Tests
# ============================================================================

@test "ports::get_all_assigned returns ports and contains expected values" {
    run ports::get_all_assigned
    [ "$status" -eq 0 ]
    
    # Check that output contains expected ports
    [[ "$output" == *"3000"* ]]  # UI
    [[ "$output" == *"5329"* ]]  # API
    [[ "$output" == *"11434"* ]] # Ollama
}

@test "ports::find_available_in_range finds available port" {
    # Find available port in a range without conflicts
    run ports::find_available_in_range 50000 50010
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    
    # Port should be numeric
    [[ "$output" =~ ^[0-9]+$ ]]
    
    # Port should be in requested range
    [ "$output" -ge 50000 ]
    [ "$output" -le 50010 ]
}

@test "ports::find_available_in_range fails when no ports available" {
    # Try to find port in Vrooli UI port (single port, reserved)
    run ports::find_available_in_range 3000 3000
    [ "$status" -eq 1 ]
}

# ============================================================================
# Registry Display Tests
# ============================================================================

@test "ports::show_registry displays all sections" {
    run ports::show_registry
    [ "$status" -eq 0 ]
    
    # Check for main sections
    [[ "$output" == *"VROOLI CORE SERVICES:"* ]]
    [[ "$output" == *"LOCAL RESOURCES:"* ]]
    [[ "$output" == *"RESERVED RANGES:"* ]]
    
    # Check for specific entries
    [[ "$output" == *"ui"*"3000"* ]]
    [[ "$output" == *"ollama"*"11434"* ]]
    [[ "$output" == *"vrooli_core"*"3000-4100"* ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "no resource port conflicts with Vrooli ports" {
    # Ensure no resource port is in Vrooli's reserved list
    for resource in "${!RESOURCE_PORTS[@]}"; do
        local port="${RESOURCE_PORTS[$resource]}"
        run ports::is_vrooli_reserved "$port"
        if [ "$status" -eq 0 ]; then
            echo "Resource $resource port $port conflicts with Vrooli!"
            false
        fi
    done
}

@test "all Vrooli ports are valid numbers in range" {
    # Check all Vrooli ports
    for service in "${!VROOLI_PORTS[@]}"; do
        local port="${VROOLI_PORTS[$service]}"
        
        # Check if numeric
        if ! [[ "$port" =~ ^[0-9]+$ ]]; then
            echo "Invalid Vrooli port for $service: $port"
            false
        fi
        
        # Check range
        if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
            echo "Vrooli port out of range for $service: $port"
            false
        fi
    done
}

@test "all resource ports are valid numbers in range" {
    # Check all resource ports
    for resource in "${!RESOURCE_PORTS[@]}"; do
        local port="${RESOURCE_PORTS[$resource]}"
        
        # Check if numeric
        if ! [[ "$port" =~ ^[0-9]+$ ]]; then
            echo "Invalid resource port for $resource: $port"
            false
        fi
        
        # Check range
        if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
            echo "Resource port out of range for $resource: $port"
            false
        fi
    done
}

@test "no duplicate ports across all services" {
    # Collect all ports
    local all_ports=()
    for service in "${!VROOLI_PORTS[@]}"; do
        all_ports+=("${VROOLI_PORTS[$service]}")
    done
    for resource in "${!RESOURCE_PORTS[@]}"; do
        all_ports+=("${RESOURCE_PORTS[$resource]}")
    done
    
    # Sort and check for duplicates
    local sorted_ports=($(printf '%s\n' "${all_ports[@]}" | sort -n))
    local prev=""
    for port in "${sorted_ports[@]}"; do
        if [[ "$port" == "$prev" ]] && [[ -n "$port" ]]; then
            echo "Duplicate port found: $port"
            false
        fi
        prev="$port"
    done
}