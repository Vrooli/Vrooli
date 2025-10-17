#!/usr/bin/env bash
# Pi-hole Regex Library - Regex filtering management functions
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source core library if not already sourced
if [[ -z "${CONTAINER_NAME:-}" ]]; then
    source "${SCRIPT_DIR}/core.sh"
fi

# Add regex blacklist pattern
add_regex_blacklist() {
    local pattern="$1"
    local comment="${2:-}"
    
    if [[ -z "$pattern" ]]; then
        echo "Error: Regex pattern required" >&2
        echo "Usage: add_regex_blacklist <pattern> [comment]" >&2
        echo "Example: add_regex_blacklist '^ad[0-9]*\\.' 'Block ad subdomains'" >&2
        return 1
    fi
    
    echo "Adding regex blacklist pattern: $pattern"
    
    # Use Pi-hole's regex command
    if [[ -n "$comment" ]]; then
        docker exec "${CONTAINER_NAME}" pihole --regex "${pattern}" --comment "${comment}"
    else
        docker exec "${CONTAINER_NAME}" pihole --regex "${pattern}"
    fi
    
    echo "Regex blacklist pattern added"
    return 0
}

# Remove regex blacklist pattern
remove_regex_blacklist() {
    local pattern="$1"
    
    if [[ -z "$pattern" ]]; then
        echo "Error: Regex pattern required" >&2
        return 1
    fi
    
    echo "Removing regex blacklist pattern: $pattern"
    
    # Use Pi-hole's regex delete command
    docker exec "${CONTAINER_NAME}" pihole --regex -d "${pattern}"
    
    echo "Regex blacklist pattern removed"
    return 0
}

# Add regex whitelist pattern
add_regex_whitelist() {
    local pattern="$1"
    local comment="${2:-}"
    
    if [[ -z "$pattern" ]]; then
        echo "Error: Regex pattern required" >&2
        echo "Usage: add_regex_whitelist <pattern> [comment]" >&2
        echo "Example: add_regex_whitelist '^(www\\.)?example\\.com$' 'Allow example.com'" >&2
        return 1
    fi
    
    echo "Adding regex whitelist pattern: $pattern"
    
    # Use Pi-hole's regex whitelist command
    if [[ -n "$comment" ]]; then
        docker exec "${CONTAINER_NAME}" pihole --regex-white "${pattern}" --comment "${comment}"
    else
        docker exec "${CONTAINER_NAME}" pihole --regex-white "${pattern}"
    fi
    
    echo "Regex whitelist pattern added"
    return 0
}

# Remove regex whitelist pattern
remove_regex_whitelist() {
    local pattern="$1"
    
    if [[ -z "$pattern" ]]; then
        echo "Error: Regex pattern required" >&2
        return 1
    fi
    
    echo "Removing regex whitelist pattern: $pattern"
    
    # Use Pi-hole's regex whitelist delete command
    docker exec "${CONTAINER_NAME}" pihole --regex-white -d "${pattern}"
    
    echo "Regex whitelist pattern removed"
    return 0
}

# List regex patterns
list_regex_patterns() {
    local filter_type="${1:-all}"  # all, black, white
    
    echo "Regex patterns:"
    echo "==============="
    
    case "$filter_type" in
        black|blacklist)
            echo "Blacklist patterns:"
            docker exec "${CONTAINER_NAME}" pihole --regex --list 2>/dev/null || echo "No blacklist patterns"
            ;;
        white|whitelist)
            echo "Whitelist patterns:"
            docker exec "${CONTAINER_NAME}" pihole --regex-white --list 2>/dev/null || echo "No whitelist patterns"
            ;;
        all|*)
            echo "Blacklist patterns:"
            docker exec "${CONTAINER_NAME}" pihole --regex --list 2>/dev/null || echo "No blacklist patterns"
            echo ""
            echo "Whitelist patterns:"
            docker exec "${CONTAINER_NAME}" pihole --regex-white --list 2>/dev/null || echo "No whitelist patterns"
            ;;
    esac
}

# Test regex pattern
test_regex_pattern() {
    local pattern="$1"
    local domain="$2"
    
    if [[ -z "$pattern" || -z "$domain" ]]; then
        echo "Error: Pattern and domain required" >&2
        echo "Usage: test_regex_pattern <pattern> <domain>" >&2
        echo "Example: test_regex_pattern '^ad[0-9]*\\.' 'ad123.example.com'" >&2
        return 1
    fi
    
    echo "Testing pattern: $pattern"
    echo "Against domain: $domain"
    
    # Test using grep with the pattern
    if echo "$domain" | grep -qE "$pattern"; then
        echo "✓ MATCH - Domain would be blocked by this pattern"
    else
        echo "✗ NO MATCH - Domain would not be affected by this pattern"
    fi
}

# Import regex patterns from file
import_regex_patterns() {
    local file="$1"
    local list_type="${2:-black}"  # black or white
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    echo "Importing regex patterns from: $file"
    
    local count=0
    while IFS= read -r pattern; do
        # Skip empty lines and comments
        [[ -z "$pattern" || "$pattern" =~ ^[[:space:]]*# ]] && continue
        
        # Add pattern based on type
        if [[ "$list_type" == "white" ]]; then
            add_regex_whitelist "$pattern" "Imported from file" || true
        else
            add_regex_blacklist "$pattern" "Imported from file" || true
        fi
        
        ((count++))
    done < "$file"
    
    echo "Imported $count regex patterns"
    return 0
}

# Export regex patterns to file
export_regex_patterns() {
    local file="$1"
    local list_type="${2:-all}"  # all, black, white
    
    echo "Exporting regex patterns to: $file"
    
    case "$list_type" in
        black|blacklist)
            docker exec "${CONTAINER_NAME}" pihole --regex -l > "$file" 2>/dev/null
            ;;
        white|whitelist)
            docker exec "${CONTAINER_NAME}" pihole --regex-white -l > "$file" 2>/dev/null
            ;;
        all|*)
            {
                echo "# Pi-hole Regex Patterns Export"
                echo "# Generated: $(date)"
                echo ""
                echo "# === BLACKLIST PATTERNS ==="
                docker exec "${CONTAINER_NAME}" pihole --regex -l 2>/dev/null || echo "# No blacklist patterns"
                echo ""
                echo "# === WHITELIST PATTERNS ==="
                docker exec "${CONTAINER_NAME}" pihole --regex-white -l 2>/dev/null || echo "# No whitelist patterns"
            } > "$file"
            ;;
    esac
    
    echo "Regex patterns exported to $file"
    return 0
}

# Common regex patterns library
add_common_regex_patterns() {
    local category="${1:-basic}"
    
    echo "Adding common regex patterns for category: $category"
    
    case "$category" in
        basic)
            # Basic ad and tracking patterns
            add_regex_blacklist '^ad[0-9]*[-.]' 'Block ad subdomains'
            add_regex_blacklist '^(.*\\.)?track(er|ing)?[0-9]*[-.]' 'Block tracking subdomains'
            add_regex_blacklist '^(.*\\.)?telemetry[-.]' 'Block telemetry'
            add_regex_blacklist '^(.*\\.)?analytic(s)?[0-9]*[-.]' 'Block analytics'
            ;;
        aggressive)
            # More aggressive blocking
            add_regex_blacklist '.*\\.doubleclick\\..*' 'Block DoubleClick'
            add_regex_blacklist '.*\\.googlesyndication\\..*' 'Block Google ads'
            add_regex_blacklist '.*\\.googleadservices\\..*' 'Block Google ad services'
            add_regex_blacklist '.*\\.google-analytics\\..*' 'Block Google Analytics'
            add_regex_blacklist '.*\\.amazon-adsystem\\..*' 'Block Amazon ads'
            add_regex_blacklist '.*\\.facebook\\.com\\/tr[\\/\\?]?' 'Block Facebook tracking'
            ;;
        social)
            # Social media tracking
            add_regex_blacklist '.*\\.scorecardresearch\\..*' 'Block comScore'
            add_regex_blacklist '.*\\.quantserve\\..*' 'Block Quantcast'
            add_regex_blacklist '.*\\.outbrain\\..*' 'Block Outbrain'
            add_regex_blacklist '.*\\.taboola\\..*' 'Block Taboola'
            ;;
        malware)
            # Known malware patterns
            add_regex_blacklist '^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}$' 'Block direct IP access'
            add_regex_blacklist '.*\\.tk$' 'Block .tk domains (often malicious)'
            add_regex_blacklist '.*\\.ml$' 'Block .ml domains (often malicious)'
            ;;
        *)
            echo "Unknown category: $category"
            echo "Available categories: basic, aggressive, social, malware"
            return 1
            ;;
    esac
    
    echo "Common regex patterns added for category: $category"
    return 0
}

# Export functions for use by other scripts
export -f add_regex_blacklist
export -f remove_regex_blacklist
export -f add_regex_whitelist
export -f remove_regex_whitelist
export -f list_regex_patterns
export -f test_regex_pattern
export -f import_regex_patterns
export -f export_regex_patterns
export -f add_common_regex_patterns