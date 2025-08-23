#!/usr/bin/env bash
# DNS (dig) Command Mock for Bats tests
# Provides comprehensive DNS query mocking with a clear mock::dig::* namespace.

# Prevent duplicate loading
if [[ "${DIG_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export DIG_MOCK_LOADED="true"

# ----------------------------
# Global mock state & options
# ----------------------------
# Default mode: normal (can be set to: normal, offline, slow, nxdomain)
export DIG_MOCK_MODE="${DIG_MOCK_MODE:-normal}"

# In-memory state
declare -gA MOCK_DIG_RECORDS=()        # domain:type -> "value1|value2|value3"
declare -gA MOCK_DIG_TTLS=()           # domain:type -> ttl
declare -gA MOCK_DIG_FAILURES=()       # domain -> failure_type (timeout, nxdomain, servfail)
declare -gA MOCK_DIG_RESPONSE_TIMES=() # domain -> milliseconds
declare -gA MOCK_DIG_QUERY_COUNT=()    # domain:type -> count
declare -gA MOCK_DIG_NAMESERVERS=()    # index -> nameserver

# File-based state persistence for subshell access (BATS compatibility)
export DIG_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/dig_mock_state.$$"

# Initialize state file
_dig_mock_init_state_file() {
    if [[ -n "${DIG_MOCK_STATE_FILE}" ]]; then
        {
            echo "declare -gA MOCK_DIG_RECORDS=()"
            echo "declare -gA MOCK_DIG_TTLS=()"
            echo "declare -gA MOCK_DIG_FAILURES=()"
            echo "declare -gA MOCK_DIG_RESPONSE_TIMES=()"
            echo "declare -gA MOCK_DIG_QUERY_COUNT=()"
            echo "declare -gA MOCK_DIG_NAMESERVERS=()"
        } > "$DIG_MOCK_STATE_FILE"
    fi
}

# Save current state to file
_dig_mock_save_state() {
    if [[ -n "${DIG_MOCK_STATE_FILE}" ]]; then
        {
            declare -p MOCK_DIG_RECORDS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DIG_RECORDS=()"
            declare -p MOCK_DIG_TTLS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DIG_TTLS=()"
            declare -p MOCK_DIG_FAILURES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DIG_FAILURES=()"
            declare -p MOCK_DIG_RESPONSE_TIMES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DIG_RESPONSE_TIMES=()"
            declare -p MOCK_DIG_QUERY_COUNT 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DIG_QUERY_COUNT=()"
            declare -p MOCK_DIG_NAMESERVERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DIG_NAMESERVERS=()"
        } > "$DIG_MOCK_STATE_FILE"
    fi
}

# Load state from file
_dig_mock_load_state() {
    if [[ -n "${DIG_MOCK_STATE_FILE}" && -f "$DIG_MOCK_STATE_FILE" ]]; then
        eval "$(cat "$DIG_MOCK_STATE_FILE")" 2>/dev/null || true
    fi
}

# Initialize state file
_dig_mock_init_state_file

# ----------------------------
# Public functions used by tests
# ----------------------------
mock::dig::reset() {
    # Clear all state
    declare -gA MOCK_DIG_RECORDS=()
    declare -gA MOCK_DIG_TTLS=()
    declare -gA MOCK_DIG_FAILURES=()
    declare -gA MOCK_DIG_RESPONSE_TIMES=()
    declare -gA MOCK_DIG_QUERY_COUNT=()
    declare -gA MOCK_DIG_NAMESERVERS=()
    
    # Reset mode to normal
    export DIG_MOCK_MODE="normal"
    
    # Initialize state file for subshell access
    _dig_mock_init_state_file
    
    # Save the cleared state
    _dig_mock_save_state
    
    echo "[MOCK] DNS (dig) state reset"
}

# Set a single DNS record
mock::dig::set_record() {
    local domain="$1"
    local record_type="$2"
    local value="$3"
    local ttl="${4:-300}"  # Default TTL of 300 seconds
    
    local key="${domain}:${record_type}"
    MOCK_DIG_RECORDS["$key"]="$value"
    MOCK_DIG_TTLS["$key"]="$ttl"
    
    # Save state to file for subshell access
    _dig_mock_save_state
    
    echo "[MOCK] Set DNS record: $domain $record_type -> $value (TTL: $ttl)"
}

# Set multiple DNS records
mock::dig::set_records() {
    local domain="$1"
    local record_type="$2"
    shift 2
    local values=("$@")
    
    local key="${domain}:${record_type}"
    # Join values with pipe separator
    local joined_values=""
    for value in "${values[@]}"; do
        if [[ -z "$joined_values" ]]; then
            joined_values="$value"
        else
            joined_values="${joined_values}|${value}"
        fi
    done
    
    MOCK_DIG_RECORDS["$key"]="$joined_values"
    MOCK_DIG_TTLS["$key"]="300"  # Default TTL
    
    # Save state to file for subshell access
    _dig_mock_save_state
    
    echo "[MOCK] Set DNS records: $domain $record_type -> ${#values[@]} values"
}

# Set DNS failure condition
mock::dig::set_failure() {
    local domain="$1"
    local failure_type="$2"  # timeout, nxdomain, servfail
    
    MOCK_DIG_FAILURES["$domain"]="$failure_type"
    
    # Save state to file for subshell access
    _dig_mock_save_state
    
    echo "[MOCK] Set DNS failure: $domain -> $failure_type"
}

# Set response time for a domain
mock::dig::set_response_time() {
    local domain="$1"
    local milliseconds="$2"
    
    MOCK_DIG_RESPONSE_TIMES["$domain"]="$milliseconds"
    
    # Save state to file for subshell access
    _dig_mock_save_state
    
    echo "[MOCK] Set DNS response time: $domain -> ${milliseconds}ms"
}

# Get query count for verification
mock::dig::get_query_count() {
    local domain="$1"
    local record_type="${2:-}"
    
    # Load state from file for subshell access
    _dig_mock_load_state
    
    local key
    if [[ -n "$record_type" ]]; then
        key="${domain}:${record_type}"
    else
        # Count all queries for the domain
        local total=0
        for k in "${!MOCK_DIG_QUERY_COUNT[@]}"; do
            if [[ "$k" == "$domain:"* ]]; then
                total=$((total + ${MOCK_DIG_QUERY_COUNT["$k"]:-0}))
            fi
        done
        echo "$total"
        return 0
    fi
    
    echo "${MOCK_DIG_QUERY_COUNT["$key"]:-0}"
}

# Check if domain was queried
mock::dig::was_queried() {
    local domain="$1"
    local record_type="$2"
    
    local count=$(mock::dig::get_query_count "$domain" "$record_type")
    [[ "$count" -gt 0 ]]
}

# ----------------------------
# Main dig command override
# ----------------------------
dig() {
    # Load state from file for subshell access
    _dig_mock_load_state
    
    # Parse arguments
    local domain=""
    local record_type="A"  # Default to A record
    local short_output=false
    local timeout=5
    local tries=1
    local nameserver=""
    local show_stats=true
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            +short)
                short_output=true
                show_stats=false
                shift
                ;;
            +time=*)
                timeout="${1#*=}"
                shift
                ;;
            +tries=*)
                tries="${1#*=}"
                shift
                ;;
            @*)
                nameserver="${1#@}"
                shift
                ;;
            +nostats|+stats)
                show_stats=[[ "$1" == "+stats" ]]
                shift
                ;;
            A|AAAA|MX|TXT|NS|SOA|PTR|CNAME)
                record_type="$1"
                shift
                ;;
            -*)
                # Ignore other options
                shift
                ;;
            *)
                # Assume it's the domain
                if [[ -z "$domain" ]]; then
                    domain="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Check for global mock mode
    case "$DIG_MOCK_MODE" in
        offline)
            echo ";; connection timed out; no servers could be reached" >&2
            return 9
            ;;
        nxdomain)
            if [[ "$short_output" == "false" ]]; then
                echo ";; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN"
            fi
            return 0
            ;;
    esac
    
    # Check for domain-specific failures
    if [[ -n "${MOCK_DIG_FAILURES[$domain]:-}" ]]; then
        case "${MOCK_DIG_FAILURES[$domain]}" in
            timeout)
                # Simulate timeout delay if configured
                if [[ -n "${MOCK_DIG_RESPONSE_TIMES[$domain]:-}" ]]; then
                    sleep "$((${MOCK_DIG_RESPONSE_TIMES[$domain]} / 1000))"
                fi
                echo ";; connection timed out; no servers could be reached" >&2
                return 9
                ;;
            nxdomain)
                if [[ "$short_output" == "false" ]]; then
                    echo ";; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN"
                fi
                return 0
                ;;
            servfail)
                if [[ "$short_output" == "false" ]]; then
                    echo ";; ->>HEADER<<- opcode: QUERY, status: SERVFAIL"
                fi
                return 2
                ;;
        esac
    fi
    
    # Simulate response time if configured
    if [[ -n "${MOCK_DIG_RESPONSE_TIMES[$domain]:-}" ]]; then
        local delay=$((${MOCK_DIG_RESPONSE_TIMES[$domain]} / 1000))
        [[ $delay -gt 0 ]] && sleep "$delay"
    fi
    
    # Update query count
    local key="${domain}:${record_type}"
    MOCK_DIG_QUERY_COUNT["$key"]=$((${MOCK_DIG_QUERY_COUNT["$key"]:-0} + 1))
    _dig_mock_save_state
    
    # Get configured records
    local records="${MOCK_DIG_RECORDS[$key]:-}"
    
    # If no records configured, return empty (but successful)
    if [[ -z "$records" ]]; then
        if [[ "$short_output" == "false" ]]; then
            echo ";; ANSWER SECTION:"
            echo ";; (no records found)"
        fi
        return 0
    fi
    
    # Output records
    if [[ "$short_output" == "true" ]]; then
        # Simple output - one IP per line
        echo "$records" | tr '|' '\n'
    else
        # Full dig output format
        echo ""
        echo "; <<>> DiG Mock <<>> $record_type $domain"
        [[ -n "$nameserver" ]] && echo ";; SERVER: $nameserver"
        echo ";; QUESTION SECTION:"
        echo ";$domain.            IN      $record_type"
        echo ""
        echo ";; ANSWER SECTION:"
        
        local ttl="${MOCK_DIG_TTLS[$key]:-300}"
        echo "$records" | tr '|' '\n' | while read -r value; do
            echo "$domain.        $ttl    IN      $record_type     $value"
        done
        
        if [[ "$show_stats" == "true" ]]; then
            echo ""
            echo ";; Query time: ${MOCK_DIG_RESPONSE_TIMES[$domain]:-1} msec"
            echo ";; SERVER: ${nameserver:-127.0.0.1}#53(${nameserver:-127.0.0.1})"
            echo ";; WHEN: $(date)"
            echo ";; MSG SIZE  rcvd: 100"
        fi
    fi
    
    return 0
}

# Export all functions so they're available in subshells
export -f dig
export -f mock::dig::reset
export -f mock::dig::set_record
export -f mock::dig::set_records
export -f mock::dig::set_failure
export -f mock::dig::set_response_time
export -f mock::dig::get_query_count
export -f mock::dig::was_queried
export -f _dig_mock_init_state_file
export -f _dig_mock_save_state
export -f _dig_mock_load_state

echo "[MOCK] DNS (dig) mock loaded successfully"