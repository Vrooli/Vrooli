#!/usr/bin/env bash
# Dig Mock - Tier 2 (Stateful)
# 
# Provides DNS lookup mock for testing:
# - A, AAAA, CNAME, MX, TXT record queries
# - Reverse DNS lookups
# - DNS server configuration
# - Query statistics
# - Error injection for resilience testing
#
# Coverage: ~80% of common dig operations in 300 lines

# === Configuration ===
declare -gA DIG_CONFIG=(
    [status]="active"
    [server]="8.8.8.8"
    [timeout]="5"
    [retries]="3"
    [error_mode]=""
    [version]="9.16.1"
)

declare -gA DNS_RECORDS=()        # Domain -> Record mappings
declare -gA DIG_QUERIES=()        # Query history
declare -ga DIG_HISTORY=()        # Command history
declare -gi DIG_QUERY_COUNT=0

# Debug mode
declare -g DIG_DEBUG="${DIG_DEBUG:-}"

# === Helper Functions ===
dig_debug() {
    [[ -n "$DIG_DEBUG" ]] && echo "[MOCK:DIG] $*" >&2
}

dig_check_error() {
    case "${DIG_CONFIG[error_mode]}" in
        "server_failure")
            echo ";; Got answer:" >&2
            echo ";; ->>HEADER<<- opcode: QUERY, status: SERVFAIL, id: 12345" >&2
            return 2
            ;;
        "timeout")
            echo ";; connection timed out; no servers could be reached" >&2
            return 9
            ;;
        "nxdomain")
            echo ";; Got answer:" >&2
            echo ";; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN, id: 12345" >&2
            return 0
            ;;
    esac
    return 0
}

# === Main Dig Command ===
dig() {
    dig_debug "dig called with: $*"
    
    ((DIG_QUERY_COUNT++))
    DIG_HISTORY+=("dig $*")
    
    if ! dig_check_error; then
        return $?
    fi
    
    local domain=""
    local record_type="A"
    local server="${DIG_CONFIG[server]}"
    local short=""
    local trace=""
    local reverse=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            @*)
                server="${1#@}"
                shift
                ;;
            +short)
                short="true"
                shift
                ;;
            +trace)
                trace="true"
                shift
                ;;
            -x)
                reverse="true"
                domain="$2"
                shift 2
                ;;
            A|AAAA|CNAME|MX|TXT|NS|PTR|SOA)
                record_type="$1"
                shift
                ;;
            *)
                [[ -z "$domain" ]] && domain="$1"
                shift
                ;;
        esac
    done
    
    # Default domain
    [[ -z "$domain" ]] && domain="example.com"
    
    # Log query
    local query_key="${domain}:${record_type}"
    local count=${DIG_QUERIES[$query_key]:-0}
    DIG_QUERIES[$query_key]=$((count + 1))
    
    # Handle reverse DNS
    if [[ "$reverse" == "true" ]]; then
        if [[ "$short" == "true" ]]; then
            echo "reverse-${domain//./-}.example.com."
        else
            dig_full_output "$domain" "PTR" "reverse-${domain//./-}.example.com."
        fi
        return 0
    fi
    
    # Generate response based on record type
    local response=""
    case "$record_type" in
        A)
            response="192.0.2.1"
            ;;
        AAAA)
            response="2001:db8::1"
            ;;
        CNAME)
            response="canonical-${domain}"
            ;;
        MX)
            response="10 mail.${domain}."
            ;;
        TXT)
            response="\"v=spf1 include:${domain} ~all\""
            ;;
        NS)
            response="ns1.${domain}."
            ;;
        SOA)
            response="ns1.${domain}. admin.${domain}. 2021010101 3600 1800 604800 86400"
            ;;
        *)
            response="192.0.2.1"
            ;;
    esac
    
    # Check for custom records
    if [[ -n "${DNS_RECORDS[$query_key]:-}" ]]; then
        response="${DNS_RECORDS[$query_key]}"
    fi
    
    # Output format
    if [[ "$short" == "true" ]]; then
        echo "$response"
    else
        dig_full_output "$domain" "$record_type" "$response"
    fi
    
    return 0
}

# === Generate Full Dig Output ===
dig_full_output() {
    local domain="$1"
    local record_type="$2"
    local response="$3"
    
    cat << EOF
; <<>> DiG ${DIG_CONFIG[version]} <<>> $domain $record_type
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 12345
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 512
;; QUESTION SECTION:
;$domain.			IN	$record_type

;; ANSWER SECTION:
$domain.		300	IN	$record_type	$response

;; Query time: 15 msec
;; SERVER: ${DIG_CONFIG[server]}#53(${DIG_CONFIG[server]})
;; WHEN: $(date)
;; MSG SIZE  rcvd: 64

EOF
}

# === Additional DNS Tools ===
nslookup() {
    dig_debug "nslookup called with: $*"
    
    local domain="${1:-example.com}"
    local server="${2:-${DIG_CONFIG[server]}}"
    
    echo "Server:		$server"
    echo "Address:	$server#53"
    echo ""
    echo "Non-authoritative answer:"
    echo "Name:	$domain"
    echo "Address: 192.0.2.1"
    echo ""
}

host() {
    dig_debug "host called with: $*"
    
    local domain="${1:-example.com}"
    
    echo "$domain has address 192.0.2.1"
    echo "$domain mail is handled by 10 mail.$domain."
}

# === DNS Record Management ===
dig_set_record() {
    local domain="$1"
    local record_type="$2"
    local value="$3"
    
    local key="${domain}:${record_type}"
    DNS_RECORDS[$key]="$value"
    dig_debug "Set DNS record: $key = $value"
}

dig_get_record() {
    local domain="$1"
    local record_type="$2"
    
    local key="${domain}:${record_type}"
    echo "${DNS_RECORDS[$key]:-}"
}

dig_list_records() {
    echo "DNS Records:"
    for key in "${!DNS_RECORDS[@]}"; do
        echo "  $key -> ${DNS_RECORDS[$key]}"
    done
}

# === Query Statistics ===
dig_get_query_count() {
    local domain="${1:-}"
    local record_type="${2:-}"
    
    if [[ -n "$domain" ]] && [[ -n "$record_type" ]]; then
        echo "${DIG_QUERIES[${domain}:${record_type}]:-0}"
    else
        echo "$DIG_QUERY_COUNT"
    fi
}

# === Mock Control Functions ===
dig_mock_reset() {
    dig_debug "Resetting mock state"
    
    DNS_RECORDS=()
    DIG_QUERIES=()
    DIG_HISTORY=()
    DIG_QUERY_COUNT=0
    DIG_CONFIG[error_mode]=""
    DIG_CONFIG[server]="8.8.8.8"
    
    # Initialize common records
    dig_set_record "example.com" "A" "192.0.2.1"
    dig_set_record "test.com" "A" "192.0.2.2"
    dig_set_record "localhost" "A" "127.0.0.1"
}

dig_mock_set_error() {
    DIG_CONFIG[error_mode]="$1"
    dig_debug "Set error mode: $1"
}

dig_mock_dump_state() {
    echo "=== Dig Mock State ==="
    echo "Status: ${DIG_CONFIG[status]}"
    echo "Server: ${DIG_CONFIG[server]}"
    echo "Version: ${DIG_CONFIG[version]}"
    echo "Query Count: $DIG_QUERY_COUNT"
    echo "Records: ${#DNS_RECORDS[@]}"
    echo "History: ${#DIG_HISTORY[@]}"
    echo "Error Mode: ${DIG_CONFIG[error_mode]:-none}"
    echo "================="
}

# === Convention-based Test Functions ===
test_dig_connection() {
    dig_debug "Testing connection..."
    
    # Test basic dig functionality
    local result=$(dig +short example.com)
    if [[ -n "$result" ]]; then
        dig_debug "Connection test passed"
        return 0
    else
        dig_debug "Connection test failed"
        return 1
    fi
}

test_dig_health() {
    dig_debug "Testing health..."
    
    test_dig_connection || return 1
    
    # Test various record types
    dig A example.com >/dev/null 2>&1 || return 1
    dig MX example.com >/dev/null 2>&1 || return 1
    
    dig_debug "Health test passed"
    return 0
}

test_dig_basic() {
    dig_debug "Testing basic operations..."
    
    # Test A record
    local result=$(dig +short A example.com)
    [[ "$result" == "192.0.2.1" ]] || return 1
    
    # Test custom record
    dig_set_record "test.local" "A" "10.0.0.1"
    result=$(dig +short A test.local)
    [[ "$result" == "10.0.0.1" ]] || return 1
    
    # Test nslookup
    nslookup test.com >/dev/null 2>&1 || return 1
    
    dig_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f dig
export -f nslookup
export -f host
export -f dig_set_record
export -f dig_get_record
export -f dig_list_records
export -f dig_get_query_count
export -f test_dig_connection
export -f test_dig_health
export -f test_dig_basic
export -f dig_mock_reset
export -f dig_mock_set_error
export -f dig_mock_dump_state
export -f dig_debug

# Initialize
dig_mock_reset
dig_debug "Dig Tier 2 mock initialized"