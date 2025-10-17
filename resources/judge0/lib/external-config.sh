#!/usr/bin/env bash
################################################################################
# Judge0 External API Configuration Helper
# 
# Manages external Judge0 API fallback configuration
################################################################################

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Simple logging
log() {
    echo -e "[$(date +'%H:%M:%S')] $*" >&2
}

# Setup external API with demo configuration
judge0::external::setup_demo() {
    local config_file="${HOME}/.vrooli/resources/judge0/external.conf"
    mkdir -p "$(dirname "$config_file")"
    
    cat > "$config_file" <<'EOF'
# Judge0 External API Configuration (Demo)
# WARNING: This is for testing only with limited requests.
# Get your own API key at: https://rapidapi.com/judge0-official/api/judge0-ce
export JUDGE0_EXTERNAL_ENABLED=true
export JUDGE0_EXTERNAL_URL=https://judge0-ce.p.rapidapi.com
export JUDGE0_EXTERNAL_MODE=demo
# Demo key is rate-limited, replace with your own for production:
export JUDGE0_RAPIDAPI_KEY="demo_limited_50_daily"
EOF
    
    log "${GREEN}✅ Demo external API configuration created at $config_file${NC}"
    log "${YELLOW}⚠️  Note: Demo mode has limited requests (50/day).${NC}"
    log "${BLUE}ℹ️  For production, get a key at: https://rapidapi.com/judge0-official/api/judge0-ce${NC}"
    
    # Auto-source the config
    source "$config_file"
    log "${GREEN}Configuration loaded into current shell${NC}"
}

# Check external API configuration
judge0::external::check() {
    local config_file="${HOME}/.vrooli/resources/judge0/external.conf"
    
    if [[ -f "$config_file" ]]; then
        source "$config_file"
        if [[ "${JUDGE0_EXTERNAL_ENABLED:-false}" == "true" ]]; then
            echo -e "${GREEN}✅ External API configured${NC}"
            echo -e "  Mode: ${JUDGE0_EXTERNAL_MODE:-custom}"
            echo -e "  URL: ${JUDGE0_EXTERNAL_URL:-not set}"
            if [[ -n "${JUDGE0_RAPIDAPI_KEY:-}" ]]; then
                local key_preview="${JUDGE0_RAPIDAPI_KEY:0:10}..."
                echo -e "  API Key: ${key_preview}"
            else
                echo -e "  API Key: ${YELLOW}not configured${NC}"
            fi
            return 0
        else
            echo -e "${RED}❌ External API disabled${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ No external API configuration found${NC}"
        echo -e "${BLUE}Run: resource-judge0 external setup-demo${NC}"
        return 1
    fi
}

# Test external API connectivity
judge0::external::test() {
    local config_file="${HOME}/.vrooli/resources/judge0/external.conf"
    
    if [[ ! -f "$config_file" ]]; then
        log "${RED}❌ No external configuration found.${NC}"
        log "${BLUE}Run: resource-judge0 external setup-demo${NC}"
        return 1
    fi
    
    source "$config_file"
    
    if [[ "${JUDGE0_EXTERNAL_ENABLED:-false}" != "true" ]]; then
        log "${RED}❌ External API is disabled${NC}"
        return 1
    fi
    
    log "${YELLOW}Testing external API connectivity...${NC}"
    
    # Test with actual code submission if API key is configured
    if [[ -n "${JUDGE0_RAPIDAPI_KEY:-}" ]]; then
        local test_code='print("External API Test")'
        local submission=$(cat <<EOF
{
    "source_code": "$test_code",
    "language_id": 92
}
EOF
        )
        
        local response=$(timeout 10 curl -sf -X POST \
            "${JUDGE0_EXTERNAL_URL}/submissions?base64_encoded=false&wait=true" \
            -H "Content-Type: application/json" \
            -H "X-RapidAPI-Key: ${JUDGE0_RAPIDAPI_KEY}" \
            -H "X-RapidAPI-Host: judge0-ce.p.rapidapi.com" \
            -d "$submission" 2>/dev/null || echo '{"error": "Request failed"}')
        
        if echo "$response" | jq -e '.stdout' &>/dev/null; then
            local output=$(echo "$response" | jq -r '.stdout // ""' | tr -d '\n')
            if [[ "$output" == "External API Test" ]]; then
                log "${GREEN}✅ External API is working perfectly!${NC}"
                log "  Output: $output"
                return 0
            else
                log "${YELLOW}⚠️  External API responded but output unexpected${NC}"
                log "  Got: $output"
                return 1
            fi
        else
            local error=$(echo "$response" | jq -r '.error // "Unknown error"')
            log "${RED}❌ External API test failed: $error${NC}"
            return 1
        fi
    else
        # Fallback to simple connectivity test
        local test_url="${JUDGE0_EXTERNAL_URL}/about"
        local response=$(timeout 5 curl -sf "$test_url" 2>/dev/null || echo "FAILED")
        
        if [[ "$response" != "FAILED" ]]; then
            log "${GREEN}✅ External API is reachable (connectivity only)${NC}"
            log "${YELLOW}ℹ️  Configure API key for full test${NC}"
            return 0
        else
            log "${RED}❌ Cannot reach external API${NC}"
            return 1
        fi
    fi
}

# Setup production API with user key
judge0::external::setup_production() {
    local api_key="${1:-}"
    
    if [[ -z "$api_key" ]]; then
        log "${RED}Error: API key required${NC}"
        log "${BLUE}Usage: resource-judge0 external setup-production YOUR_API_KEY${NC}"
        log ""
        log "Get your API key from: ${BLUE}https://rapidapi.com/judge0-official/api/judge0-ce${NC}"
        return 1
    fi
    
    local config_file="${HOME}/.vrooli/resources/judge0/external.conf"
    mkdir -p "$(dirname "$config_file")"
    
    cat > "$config_file" <<EOF
# Judge0 External API Configuration (Production)
# Configured on $(date)
export JUDGE0_EXTERNAL_ENABLED=true
export JUDGE0_EXTERNAL_URL=https://judge0-ce.p.rapidapi.com
export JUDGE0_EXTERNAL_MODE=production
export JUDGE0_RAPIDAPI_KEY="$api_key"
EOF
    
    log "${GREEN}✅ Production external API configured${NC}"
    
    # Test the configuration
    source "$config_file"
    if judge0::external::test; then
        log "${GREEN}🎉 External API is ready for production use!${NC}"
    else
        log "${YELLOW}⚠️  Please verify your API key is correct${NC}"
    fi
}

# Main function for CLI
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        setup-demo)
            judge0::external::setup_demo
            ;;
        setup-production)
            judge0::external::setup_production "$@"
            ;;
        check|status)
            judge0::external::check
            ;;
        test)
            judge0::external::test
            ;;
        help|*)
            echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
            echo -e "${BLUE}  Judge0 External API Configuration${NC}"
            echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
            echo ""
            echo -e "${GREEN}Commands:${NC}"
            echo -e "  setup-demo                    Setup demo external API (50 req/day limit)"
            echo -e "  setup-production <API_KEY>    Setup production external API with your key"
            echo -e "  check                         Check current configuration status"
            echo -e "  test                          Test external API connectivity"
            echo -e "  help                          Show this help message"
            echo ""
            echo -e "${GREEN}Usage Examples:${NC}"
            echo -e "  ${BLUE}resource-judge0 external setup-demo${NC}              # Quick start with demo"
            echo -e "  ${BLUE}resource-judge0 external setup-production KEY${NC}    # Production setup"
            echo -e "  ${BLUE}resource-judge0 external check${NC}                   # Check status"
            echo -e "  ${BLUE}resource-judge0 external test${NC}                    # Test connection"
            echo ""
            echo -e "${YELLOW}Get your production API key at:${NC}"
            echo -e "  ${BLUE}https://rapidapi.com/judge0-official/api/judge0-ce${NC}"
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi