#!/usr/bin/env bash
# Mock Configuration Helpers
# Provides convenient functions to configure mock behaviors for tests

# Ensure mock response directory exists
_ensure_mock_dir() {
    mkdir -p "${MOCK_RESPONSES_DIR:-/tmp/mock_responses}"
}

#######################################
# Generic mock response configuration
#######################################
mock::set_response() {
    local command="$1"
    local subcommand="$2"
    local response="$3"
    local exitcode="${4:-0}"
    
    _ensure_mock_dir
    echo "$response" > "$MOCK_RESPONSES_DIR/${command}_${subcommand}_response"
    echo "$exitcode" > "$MOCK_RESPONSES_DIR/${command}_${subcommand}_exitcode"
}

# Set response for specific URL
mock::set_curl_response() {
    local url="$1"
    local response="$2"
    local exitcode="${3:-0}"
    
    _ensure_mock_dir
    local url_hash=$(echo -n "$url" | md5sum | cut -d' ' -f1)
    echo "$response" > "$MOCK_RESPONSES_DIR/curl_${url_hash}_response"
    echo "$exitcode" > "$MOCK_RESPONSES_DIR/curl_${url_hash}_exitcode"
}

#######################################
# Docker-specific mock helpers
#######################################
mock::docker::set_container_state() {
    local container="$1"
    local state="$2"  # running, stopped, not_found
    
    _ensure_mock_dir
    
    case "$state" in
        "running")
            echo "running" > "$MOCK_RESPONSES_DIR/container_${container}_state"
            cat > "$MOCK_RESPONSES_DIR/container_${container}_inspect" <<EOF
{
    "State": {
        "Running": true,
        "Status": "running",
        "Pid": 12345
    },
    "Config": {
        "Image": "${container}:latest",
        "Env": [],
        "ExposedPorts": {
            "8080/tcp": {}
        }
    },
    "NetworkSettings": {
        "Ports": {
            "8080/tcp": [{"HostPort": "8080"}]
        }
    }
}
EOF
            ;;
        "stopped")
            echo "stopped" > "$MOCK_RESPONSES_DIR/container_${container}_state"
            cat > "$MOCK_RESPONSES_DIR/container_${container}_inspect" <<EOF
{
    "State": {
        "Running": false,
        "Status": "exited",
        "ExitCode": 0
    },
    "Config": {
        "Image": "${container}:latest"
    }
}
EOF
            ;;
        "not_found")
            rm -f "$MOCK_RESPONSES_DIR/container_${container}_state"
            rm -f "$MOCK_RESPONSES_DIR/container_${container}_inspect"
            ;;
    esac
}

# Set Docker exec response
mock::docker::set_exec_response() {
    local response="$1"
    local exitcode="${2:-0}"
    
    _ensure_mock_dir
    echo "$response" > "$MOCK_RESPONSES_DIR/docker_exec_response"
    echo "$exitcode" > "$MOCK_RESPONSES_DIR/docker_exec_exitcode"
}

# Set container logs
mock::docker::set_container_logs() {
    local container="$1"
    local logs="$2"
    
    _ensure_mock_dir
    echo "$logs" > "$MOCK_RESPONSES_DIR/container_${container}_logs"
}

# Set Docker image existence
mock::docker::set_image_exists() {
    local image="$1"
    local exists="$2"  # "true" or "false"
    
    _ensure_mock_dir
    
    # Replace colons with double underscores for safe filenames
    local safe_image="${image//:/__}"
    
    if [[ "$exists" == "true" ]]; then
        echo "true" > "$MOCK_RESPONSES_DIR/image_${safe_image}_exists"
    else
        rm -f "$MOCK_RESPONSES_DIR/image_${safe_image}_exists"
    fi
}

# Set container health status
mock::docker::set_container_health() {
    local container="$1"
    local health="$2"  # "healthy", "unhealthy", "" (unknown/missing)
    
    _ensure_mock_dir
    
    if [[ -n "$health" ]]; then
        echo "$health" > "$MOCK_RESPONSES_DIR/container_${container}_health"
    else
        rm -f "$MOCK_RESPONSES_DIR/container_${container}_health"
    fi
}

#######################################
# HTTP endpoint mock helpers
#######################################
mock::http::set_endpoint_state() {
    local url="$1"
    local state="$2"  # healthy, unhealthy, timeout, not_found
    
    _ensure_mock_dir
    
    case "$state" in
        "healthy")
            mock::set_curl_response "$url" '{"status":"ok","healthy":true}' 0
            ;;
        "unhealthy")
            mock::set_curl_response "$url" '{"status":"error","healthy":false}' 1
            ;;
        "timeout")
            mock::set_curl_response "$url" "" 28
            ;;
        "not_found")
            mock::set_curl_response "$url" '{"error":"Not Found"}' 22
            ;;
    esac
}

# Set specific HTTP response for endpoint
mock::http::set_endpoint_response() {
    local url="$1"
    local status_code="$2"
    local response_body="$3"
    
    _ensure_mock_dir
    
    # Handle special status codes
    case "$status_code" in
        "connection_refused")
            mock::set_curl_response "$url" "" 7
            ;;
        "timeout")
            mock::set_curl_response "$url" "" 28
            ;;
        *)
            # Convert HTTP status codes to curl exit codes
            local exit_code=0
            if [[ "$status_code" -ge 400 ]]; then
                exit_code=22  # HTTP error
            fi
            mock::set_curl_response "$url" "$response_body" "$exit_code"
            ;;
    esac
}

# Set sequence of HTTP responses for retry testing
mock::http::set_endpoint_sequence() {
    local url="$1"
    local status_sequence="$2"  # e.g., "503,503,200"
    local response_sequence="$3"  # e.g., "unhealthy,unhealthy,healthy"
    
    _ensure_mock_dir
    
    # Split sequences into arrays
    IFS=',' read -ra status_array <<< "$status_sequence"
    IFS=',' read -ra response_array <<< "$response_sequence"
    
    local url_hash=$(echo -n "$url" | md5sum | cut -d' ' -f1)
    
    # Store sequence metadata
    echo "${#status_array[@]}" > "$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_count"
    echo "0" > "$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_index"
    
    # Store each response in the sequence
    for i in "${!status_array[@]}"; do
        local status="${status_array[$i]}"
        local response="${response_array[$i]:-}"
        local exit_code=0
        
        # Convert status codes to exit codes
        case "$status" in
            "connection_refused") exit_code=7 ;;
            "timeout") exit_code=28 ;;
            *) if [[ "$status" -ge 400 ]]; then exit_code=22; fi ;;
        esac
        
        echo "$response" > "$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_${i}_response"
        echo "$exit_code" > "$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_${i}_exitcode"
    done
}

# Set endpoint delay for timeout testing
mock::http::set_endpoint_delay() {
    local url="$1"
    local delay_seconds="$2"
    
    _ensure_mock_dir
    
    local url_hash=$(echo -n "$url" | md5sum | cut -d' ' -f1)
    echo "$delay_seconds" > "$MOCK_RESPONSES_DIR/curl_${url_hash}_delay"
    
    # Set a timeout response for this endpoint
    mock::set_curl_response "$url" "" 28  # 28 = curl timeout error
}

#######################################
# Service mock helpers
#######################################
mock::systemctl::set_service_state() {
    local service="$1"
    local state="$2"  # active, inactive, failed
    
    _ensure_mock_dir
    
    case "$state" in
        "active")
            cat > "$MOCK_RESPONSES_DIR/systemctl_${service}_status" <<EOF
● ${service}.service - Mock ${service} Service
     Loaded: loaded (/etc/systemd/system/${service}.service; enabled)
     Active: active (running) since Mon 2024-01-01 00:00:00 UTC
   Main PID: 12345 (${service})
     Memory: 100.0M
     CGroup: /system.slice/${service}.service
EOF
            echo "0" > "$MOCK_RESPONSES_DIR/systemctl_${service}_exitcode"
            ;;
        "inactive")
            cat > "$MOCK_RESPONSES_DIR/systemctl_${service}_status" <<EOF
● ${service}.service - Mock ${service} Service
     Loaded: loaded (/etc/systemd/system/${service}.service; disabled)
     Active: inactive (dead)
EOF
            echo "3" > "$MOCK_RESPONSES_DIR/systemctl_${service}_exitcode"
            ;;
        "failed")
            cat > "$MOCK_RESPONSES_DIR/systemctl_${service}_status" <<EOF
● ${service}.service - Mock ${service} Service
     Loaded: loaded (/etc/systemd/system/${service}.service; enabled)
     Active: failed (Result: exit-code)
   Main PID: 12345 (code=exited, status=1/FAILURE)
EOF
            echo "3" > "$MOCK_RESPONSES_DIR/systemctl_${service}_exitcode"
            ;;
    esac
}

#######################################
# Network mock helpers
#######################################
mock::network::set_offline() {
    _ensure_mock_dir
    touch "$MOCK_RESPONSES_DIR/network_offline"
}

mock::network::set_online() {
    _ensure_mock_dir
    rm -f "$MOCK_RESPONSES_DIR/network_offline"
}

# Mock port usage
mock::port::set_in_use() {
    local port="$1"
    _ensure_mock_dir
    touch "$MOCK_RESPONSES_DIR/port_${port}_in_use"
}

mock::port::set_available() {
    local port="$1"
    _ensure_mock_dir
    rm -f "$MOCK_RESPONSES_DIR/port_${port}_in_use"
}

#######################################
# Resource-specific mock setups
#######################################
mock::resource::setup() {
    local resource="$1"
    local state="$2"
    local port="${3:-$(resources::get_default_port "$resource" 2>/dev/null || echo 8080)}"
    
    case "$state" in
        "installed_running")
            mock::docker::set_container_state "$resource" "running"
            mock::http::set_endpoint_state "http://localhost:$port" "healthy"
            mock::systemctl::set_service_state "$resource" "active"
            ;;
        "installed_stopped")
            mock::docker::set_container_state "$resource" "stopped"
            mock::http::set_endpoint_state "http://localhost:$port" "not_found"
            mock::systemctl::set_service_state "$resource" "inactive"
            ;;
        "not_installed")
            mock::docker::set_container_state "$resource" "not_found"
            mock::http::set_endpoint_state "http://localhost:$port" "not_found"
            mock::systemctl::set_service_state "$resource" "inactive"
            ;;
        "unhealthy")
            mock::docker::set_container_state "$resource" "running"
            mock::http::set_endpoint_state "http://localhost:$port" "unhealthy"
            mock::systemctl::set_service_state "$resource" "active"
            ;;
    esac
}

#######################################
# Cleanup functions
#######################################
mock::cleanup() {
    if [[ -d "${MOCK_RESPONSES_DIR:-}" ]]; then
        rm -rf "$MOCK_RESPONSES_DIR"
    fi
    _ensure_mock_dir
}

# Clean specific mock
mock::clean() {
    local type="$1"
    local identifier="$2"
    
    _ensure_mock_dir
    
    case "$type" in
        "container")
            rm -f "$MOCK_RESPONSES_DIR/container_${identifier}_"*
            ;;
        "url")
            local url_hash=$(echo -n "$identifier" | md5sum | cut -d' ' -f1)
            rm -f "$MOCK_RESPONSES_DIR/curl_${url_hash}_"*
            ;;
        "service")
            rm -f "$MOCK_RESPONSES_DIR/systemctl_${identifier}_"*
            ;;
        "port")
            rm -f "$MOCK_RESPONSES_DIR/port_${identifier}_in_use"
            ;;
    esac
}

#######################################
# Debugging helpers
#######################################
mock::debug::list_mocks() {
    if [[ -d "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "Current mocks in $MOCK_RESPONSES_DIR:"
        ls -la "$MOCK_RESPONSES_DIR"
    else
        echo "No mock directory found"
    fi
}

mock::debug::show_mock() {
    local mock_file="$1"
    if [[ -f "$MOCK_RESPONSES_DIR/$mock_file" ]]; then
        echo "=== $mock_file ==="
        cat "$MOCK_RESPONSES_DIR/$mock_file"
    else
        echo "Mock file not found: $mock_file"
    fi
}

# Export mock helper functions for use in subshells and tests
export -f _ensure_mock_dir
export -f mock::set_response mock::set_curl_response
export -f mock::docker::set_container_state mock::docker::set_container_health
export -f mock::docker::set_exec_response mock::docker::set_container_logs mock::docker::set_image_exists
export -f mock::http::set_endpoint_state mock::http::set_endpoint_response
export -f mock::http::set_endpoint_sequence mock::http::set_endpoint_delay
export -f mock::systemctl::set_service_state
export -f mock::network::set_offline mock::network::set_online
export -f mock::port::set_in_use mock::port::set_available
export -f mock::resource::setup mock::cleanup mock::clean
export -f mock::debug::list_mocks mock::debug::show_mock