#!/bin/bash
# ====================================================================
# Validate Setup - Check System Readiness
# ====================================================================
#
# Validates that the testing environment is properly configured
# and shows what resources are available for testing.
#
# Usage:
#   ./validate-setup.sh [--fix] [--verbose]
#
# ====================================================================

set -euo pipefail

# Colors for output
if [[ -t 1 ]]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    GREEN='' RED='' YELLOW='' BLUE='' BOLD='' NC=''
fi

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Options
VERBOSE=false
FIX_ISSUES=false

# Counters
CHECKS_TOTAL=0
CHECKS_PASSED=0
CHECKS_FAILED=0
ISSUES_FOUND=()

# Helper functions
log_check() {
    CHECKS_TOTAL=$((CHECKS_TOTAL + 1))
    echo -n "  • $1... "
}

log_pass() {
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
    echo -e "${GREEN}✓${NC}"
    [[ "$VERBOSE" == "true" ]] && echo -e "    ${BLUE}$1${NC}"
}

log_fail() {
    CHECKS_FAILED=$((CHECKS_FAILED + 1))
    echo -e "${RED}✗${NC}"
    echo -e "    ${RED}$1${NC}"
    ISSUES_FOUND+=("$1")
    [[ $# -gt 1 ]] && echo -e "    ${BLUE}Fix: $2${NC}"
}

log_warn() {
    echo -e "    ${YELLOW}⚠ $1${NC}"
    [[ $# -gt 1 ]] && echo -e "    ${BLUE}Note: $2${NC}"
}

# Show usage
show_usage() {
    cat << EOF
🔍 Validate Setup - Check System Readiness

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --fix           Attempt to fix common issues automatically
    --verbose       Show detailed information about checks
    --help          Show this help message

WHAT IT CHECKS:
    ✓ Required tools (curl, jq, docker)
    ✓ File permissions and structure
    ✓ Resource discovery system
    ✓ Available services
    ✓ Configuration files
    ✓ Test framework components

EOF
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_usage
                exit 0
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --fix)
                FIX_ISSUES=true
                shift
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                show_usage
                exit 1
                ;;
        esac
    done
}

# Check required tools
check_required_tools() {
    echo -e "${BOLD}🛠️ Checking Required Tools${NC}"
    
    local tools=("curl" "jq" "docker" "bash")
    for tool in "${tools[@]}"; do
        log_check "Tool: $tool"
        if command -v "$tool" &> /dev/null; then
            local version
            case $tool in
                "bash") version=$(bash --version | head -1) ;;
                "curl") version=$(curl --version | head -1) ;;
                "jq") version=$(jq --version) ;;
                "docker") version=$(docker --version) ;;
            esac
            log_pass "$version"
        else
            log_fail "$tool is not installed" "Install with: sudo apt install $tool (or brew install $tool on macOS)"
        fi
    done
    echo
}

# Check file permissions and structure
check_file_structure() {
    echo -e "${BOLD}📁 Checking File Structure & Permissions${NC}"
    
    # Check main files
    local key_files=(
        "run.sh"
        "framework/discovery.sh"
        "framework/runner.sh"
        "framework/reporter.sh"
        "GETTING_STARTED.md"
    )
    
    for file in "${key_files[@]}"; do
        log_check "File: $file"
        if [[ -f "$SCRIPT_DIR/$file" ]]; then
            if [[ -r "$SCRIPT_DIR/$file" ]]; then
                if [[ "$file" == *.sh ]] && [[ -x "$SCRIPT_DIR/$file" ]]; then
                    log_pass "Exists and executable"
                elif [[ "$file" != *.sh ]]; then
                    log_pass "Exists and readable"
                else
                    log_fail "Not executable" "Run: chmod +x $SCRIPT_DIR/$file"
                    if [[ "$FIX_ISSUES" == "true" ]]; then
                        chmod +x "$SCRIPT_DIR/$file" && echo -e "    ${GREEN}Fixed: Made $file executable${NC}"
                    fi
                fi
            else
                log_fail "Not readable" "Check file permissions"
            fi
        else
            log_fail "Missing" "File should exist at $SCRIPT_DIR/$file"
        fi
    done
    
    # Check test directories
    local test_dirs=("single" "scenarios" "framework" "fixtures")
    for dir in "${test_dirs[@]}"; do
        log_check "Directory: $dir"
        if [[ -d "$SCRIPT_DIR/$dir" ]]; then
            local file_count
            file_count=$(find "$SCRIPT_DIR/$dir" -type f | wc -l)
            log_pass "$file_count files found"
        else
            log_fail "Missing" "Directory should exist at $SCRIPT_DIR/$dir"
        fi
    done
    echo
}

# Check Docker access
check_docker_access() {
    echo -e "${BOLD}🐳 Checking Docker Access${NC}"
    
    log_check "Docker daemon"
    if docker info &> /dev/null; then
        log_pass "Docker daemon running"
    else
        log_fail "Cannot connect to Docker daemon" "Run: sudo systemctl start docker"
        return
    fi
    
    log_check "Docker permissions"
    if docker ps &> /dev/null; then
        log_pass "Can list containers"
    else
        log_fail "Permission denied" "Run: sudo usermod -aG docker \$USER && newgrp docker"
        if [[ "$FIX_ISSUES" == "true" ]]; then
            echo -e "    ${BLUE}Note: You'll need to log out and back in for group changes to take effect${NC}"
        fi
    fi
    
    log_check "Container count"
    local container_count
    container_count=$(docker ps -q | wc -l)
    if [[ $container_count -gt 0 ]]; then
        log_pass "$container_count containers running"
    else
        log_warn "No containers running" "This is normal if you haven't started any services yet"
    fi
    echo
}

# Check resource discovery system
check_resource_discovery() {
    echo -e "${BOLD}🔍 Checking Resource Discovery System${NC}"
    
    log_check "Resource index script"
    if [[ -f "$RESOURCES_DIR/index.sh" && -x "$RESOURCES_DIR/index.sh" ]]; then
        log_pass "Found at $RESOURCES_DIR/index.sh"
    else
        log_fail "Missing or not executable" "Check $RESOURCES_DIR/index.sh"
        return
    fi
    
    log_check "Discovery functionality"
    local discovery_output
    if discovery_output=$(timeout 30s "$RESOURCES_DIR/index.sh" --action discover 2>&1); then
        local healthy_count
        healthy_count=$(echo "$discovery_output" | grep -c "✅.*is running" || echo "0")
        if [[ $healthy_count -gt 0 ]]; then
            log_pass "$healthy_count services discovered"
            [[ "$VERBOSE" == "true" ]] && echo "$discovery_output" | grep "✅" | sed 's/^/      /'
        else
            log_warn "No services discovered" "This is normal if no services are running"
        fi
    else
        log_fail "Discovery command failed" "Check: $RESOURCES_DIR/index.sh --action discover"
    fi
    echo
}

# Check configuration files
check_configuration() {
    echo -e "${BOLD}⚙️ Checking Configuration Files${NC}"
    
    log_check "User config directory"
    if [[ -d "$HOME/.vrooli" ]]; then
        log_pass "Found at $HOME/.vrooli"
    else
        log_warn "Not found" "Will be created automatically when needed"
    fi
    
    log_check "Resource configuration"
    local config_file="$HOME/.vrooli/resources.local.json"
    if [[ -f "$config_file" ]]; then
        if jq . "$config_file" &> /dev/null; then
            local service_count
            service_count=$(jq -r '.services | keys | length' "$config_file" 2>/dev/null || echo "0")
            log_pass "Valid JSON with $service_count service categories"
        else
            log_fail "Invalid JSON format" "Fix syntax errors in $config_file"
        fi
    else
        log_warn "Not found" "Will use defaults or example config"
    fi
    
    log_check "Example configuration"
    local example_config="$RESOURCES_DIR/../.vrooli/resources.example.json"
    if [[ -f "$example_config" ]]; then
        if jq . "$example_config" &> /dev/null; then
            log_pass "Example config is valid"
        else
            log_fail "Example config has syntax errors" "Check $example_config"
        fi
    else
        log_warn "Example config not found" "Should be at $example_config"
    fi
    echo
}

# Check test framework components
check_framework_components() {
    echo -e "${BOLD}🧪 Checking Test Framework Components${NC}"
    
    # Test sourcing key framework files
    local framework_files=(
        "discovery.sh"
        "runner.sh"
        "reporter.sh"
        "helpers/assertions.sh"
        "helpers/cleanup.sh"
    )
    
    for file in "${framework_files[@]}"; do
        log_check "Framework: $file"
        local framework_path="$SCRIPT_DIR/framework/$file"
        if [[ -f "$framework_path" ]]; then
            # Test if file can be sourced without errors
            if bash -n "$framework_path" 2>/dev/null; then
                log_pass "Syntax valid"
            else
                log_fail "Syntax errors found" "Check $framework_path for shell syntax errors"
            fi
        else
            log_fail "Missing" "File should exist at $framework_path"
        fi
    done
    
    # Check if main run.sh can show help
    log_check "Main runner help"
    if "$SCRIPT_DIR/run.sh" --help &> /dev/null; then
        log_pass "Help function works"
    else
        log_fail "Help function broken" "Check run.sh for syntax errors"
    fi
    echo
}

# Check network connectivity
check_network() {
    echo -e "${BOLD}🌐 Checking Network Connectivity${NC}"
    
    log_check "Localhost connectivity"
    if curl -s --max-time 5 http://localhost:80 &> /dev/null || \
       curl -s --max-time 5 http://127.0.0.1:80 &> /dev/null; then
        log_pass "Can connect to localhost"
    else
        log_warn "Localhost not responding" "This is normal if no web server is running"
    fi
    
    # Check common resource ports
    local common_ports=("11434:Ollama" "5678:n8n" "4113:Agent-S2" "8090:Whisper" "9000:MinIO")
    local accessible_count=0
    
    for port_info in "${common_ports[@]}"; do
        local port="${port_info%%:*}"
        local service="${port_info##*:}"
        log_check "Port $port ($service)"
        
        if timeout 2s bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
            log_pass "Service responding"
            accessible_count=$((accessible_count + 1))
        else
            log_warn "Not accessible" "Service may not be running"
        fi
    done
    
    if [[ $accessible_count -gt 0 ]]; then
        echo -e "  ${GREEN}✓ $accessible_count out of ${#common_ports[@]} common services are accessible${NC}"
    else
        echo -e "  ${YELLOW}⚠ No common services found running${NC}"
        echo -e "    ${BLUE}Start services with: ../index.sh --action install --resources \"essential\"${NC}"
    fi
    echo
}

# Generate summary and recommendations
generate_summary() {
    echo -e "${BOLD}📊 Validation Summary${NC}"
    echo
    
    local success_rate=$((CHECKS_PASSED * 100 / CHECKS_TOTAL))
    
    echo -e "  Total Checks: $CHECKS_TOTAL"
    echo -e "  Passed: ${GREEN}$CHECKS_PASSED${NC}"
    echo -e "  Failed: ${RED}$CHECKS_FAILED${NC}"
    echo -e "  Success Rate: ${success_rate}%"
    echo
    
    if [[ $CHECKS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}🎉 System is ready for testing!${NC}"
        echo
        echo -e "${BOLD}Next Steps:${NC}"
        echo -e "  1. Start some services: ${BLUE}../index.sh --action install --resources \"ollama\"${NC}"
        echo -e "  2. Run a quick test: ${BLUE}./quick-test.sh ollama${NC}"
        echo -e "  3. Try the full test suite: ${BLUE}./run.sh${NC}"
    elif [[ $CHECKS_FAILED -le 2 ]]; then
        echo -e "${YELLOW}⚠ System mostly ready with minor issues${NC}"
        echo
        echo -e "${BOLD}Issues to fix:${NC}"
        for issue in "${ISSUES_FOUND[@]}"; do
            echo -e "  • ${RED}$issue${NC}"
        done
        echo
        echo -e "${BOLD}Quick fixes:${NC}"
        echo -e "  1. Fix permissions: ${BLUE}chmod +x run.sh single/**/*.test.sh${NC}"
        echo -e "  2. Install missing tools: ${BLUE}sudo apt install curl jq docker.io${NC}"
        echo -e "  3. Fix Docker access: ${BLUE}sudo usermod -aG docker \\$USER${NC}"
    else
        echo -e "${RED}❌ System needs significant fixes before testing${NC}"
        echo
        echo -e "${BOLD}Critical issues:${NC}"
        for issue in "${ISSUES_FOUND[@]}"; do
            echo -e "  • ${RED}$issue${NC}"
        done
        echo
        echo -e "${BOLD}Recommended actions:${NC}"
        echo -e "  1. Check you're in the right directory: ${BLUE}pwd${NC}"
        echo -e "  2. Install missing dependencies${NC}"
        echo -e "  3. Fix file permissions and structure${NC}"
        echo -e "  4. Re-run with --fix option: ${BLUE}$0 --fix${NC}"
    fi
    echo
    
    echo -e "${BOLD}For more help:${NC}"
    echo -e "  • Read the guide: ${BLUE}cat docs/GETTING_STARTED.md${NC}"
    echo -e "  • See troubleshooting: ${BLUE}cat docs/TROUBLESHOOTING.md${NC}"
    echo -e "  • Get beginner help: ${BLUE}./run.sh --help-beginner${NC}"
    echo
}

# Main function
main() {
    parse_args "$@"
    
    echo -e "${BOLD}🔍 Vrooli Test Framework Validation${NC}"
    echo -e "Checking system readiness for resource testing..."
    echo
    
    check_required_tools
    check_file_structure
    check_docker_access
    check_resource_discovery
    check_configuration
    check_framework_components
    check_network
    
    generate_summary
    
    # Exit with appropriate code
    if [[ $CHECKS_FAILED -eq 0 ]]; then
        exit 0
    elif [[ $CHECKS_FAILED -le 2 ]]; then
        exit 1
    else
        exit 2
    fi
}

main "$@"