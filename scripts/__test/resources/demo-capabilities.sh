#!/bin/bash
# ====================================================================
# Demo Capabilities - Show What the Framework Can Do
# ====================================================================
#
# Demonstrates the testing framework's capabilities with live examples.
# Perfect for understanding what's possible before diving into details.
#
# Usage:
#   ./demo-capabilities.sh [--interactive] [--quick] [--verbose]
#
# ====================================================================

set -euo pipefail

# Colors for output
if [[ -t 1 ]]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    PURPLE='\033[0;35m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    GREEN='' RED='' YELLOW='' BLUE='' PURPLE='' CYAN='' BOLD='' NC=''
fi

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Options
INTERACTIVE=false
QUICK_DEMO=false
VERBOSE=false

# Demo functions
show_header() {
    echo -e "${BOLD}${BLUE}════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${BLUE}$1${NC}"
    echo -e "${BOLD}${BLUE}════════════════════════════════════════════════════════════════════${NC}"
    echo
}

show_section() {
    echo -e "${BOLD}${CYAN}🎯 $1${NC}"
    echo -e "${BLUE}$2${NC}"
    echo
}

pause_for_user() {
    if [[ "$INTERACTIVE" == "true" ]]; then
        echo -e "${YELLOW}Press Enter to continue...${NC}"
        read -r
        echo
    else
        sleep 2
    fi
}

run_demo_command() {
    local title="$1"
    local command="$2"
    local description="$3"
    
    echo -e "${PURPLE}Demo: $title${NC}"
    echo -e "${BLUE}Command: $command${NC}"
    echo -e "${BLUE}What it does: $description${NC}"
    echo
    echo -e "${YELLOW}Output:${NC}"
    echo "----------------------------------------"
    
    # Run the command and capture output
    if eval "$command" 2>&1; then
        echo "----------------------------------------"
        echo -e "${GREEN}✅ Demo completed successfully${NC}"
    else
        echo "----------------------------------------"
        echo -e "${YELLOW}⚠️ Demo completed (some features may not be available)${NC}"
    fi
    echo
    pause_for_user
}

# Show usage
show_usage() {
    cat << EOF
🎬 Demo Capabilities - See What the Framework Can Do

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --interactive       Pause between demos for user input
    --quick             Run shorter, faster demos
    --verbose           Show detailed output from all commands
    --help              Show this help message

WHAT IT SHOWS:
    🔍 Resource discovery and health checking
    🧪 Individual resource testing
    🎭 Business scenario demonstrations  
    🔗 Integration pattern testing
    📊 Performance benchmarking
    ⚙️ Advanced framework features

DURATION:
    • Quick demo: ~2-3 minutes
    • Full demo: ~5-10 minutes
    • Interactive: Depends on user pace

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
            --interactive|-i)
                INTERACTIVE=true
                shift
                ;;
            --quick|-q)
                QUICK_DEMO=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
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

# Demo 1: Resource Discovery
demo_resource_discovery() {
    show_section "Resource Discovery & Health Checking" \
        "See what services are available and their health status"
    
    run_demo_command "Discover Running Resources" \
        "../index.sh --action discover" \
        "Automatically finds and health-checks all available services"
    
    if [[ "$QUICK_DEMO" != "true" ]]; then
        run_demo_command "List All Available Resources" \
            "./run.sh --list-scenarios | head -20" \
            "Shows all testable resources and business scenarios"
    fi
}

# Demo 2: Single Resource Testing
demo_single_resource_testing() {
    show_section "Single Resource Testing" \
        "Test individual services with comprehensive validation"
    
    # Find an available resource to demo
    local demo_resource="ollama"
    if ../index.sh --action discover 2>/dev/null | grep -q "whisper.*running"; then
        demo_resource="whisper"
    elif ../index.sh --action discover 2>/dev/null | grep -q "minio.*running"; then
        demo_resource="minio"
    fi
    
    run_demo_command "Test Single Resource" \
        "./run.sh --resource $demo_resource --timeout 60" \
        "Comprehensive testing of $demo_resource including health, functionality, and performance"
    
    if [[ "$QUICK_DEMO" != "true" ]]; then
        run_demo_command "Verbose Resource Testing" \
            "./run.sh --resource $demo_resource --verbose --timeout 30 | head -30" \
            "Detailed output showing exactly what tests are being performed"
    fi
}

# Demo 3: Framework Features
demo_framework_features() {
    show_section "Advanced Framework Features" \
        "Showcase the sophisticated capabilities of the testing system"
    
    run_demo_command "Beginner-Friendly Help" \
        "./run.sh --help-beginner | head -20" \
        "Shows simplified help designed for new developers"
    
    if [[ "$QUICK_DEMO" != "true" ]]; then
        run_demo_command "JSON Output for CI/CD" \
            "./run.sh --resource ollama --output-format json --timeout 30 | jq '.summary // empty'" \
            "Machine-readable output perfect for continuous integration"
        
        run_demo_command "Interface Compliance Testing" \
            "./run.sh --interface-only --timeout 30 | head -15" \
            "Validates that resources follow standard interface patterns"
    fi
}

# Demo 4: Business Scenarios
demo_business_scenarios() {
    show_section "Business Scenario Testing" \
        "Real-world, revenue-generating workflows that combine multiple services"
    
    # Show available scenarios
    run_demo_command "Available Business Scenarios" \
        "ls -1 scenarios/ | grep -v '\\.' | head -8" \
        "Complete workflows representing actual client projects worth $10K-25K+"
    
    if [[ "$QUICK_DEMO" != "true" ]]; then
        # Show scenario metadata
        run_demo_command "Scenario Business Details" \
            "head -20 scenarios/multi-modal-ai-assistant/test.sh | grep '^#.*@'" \
            "Each scenario includes revenue potential, market demand, and business value"
        
        # Note: We don't run actual scenarios in demo as they take too long
        echo -e "${PURPLE}Note: Business scenarios take 10-15 minutes to run${NC}"
        echo -e "${BLUE}Try this for a real demo: ./run.sh --scenarios \"multi-modal-ai-assistant\"${NC}"
        echo
        pause_for_user
    fi
}

# Demo 5: Integration Patterns
demo_integration_patterns() {
    show_section "Integration Pattern Testing" \
        "Test how multiple services work together in realistic combinations"
    
    if [[ "$QUICK_DEMO" != "true" ]]; then
        echo -e "${PURPLE}Integration patterns test service combinations like:${NC}"
        echo -e "${BLUE}• AI + Storage: ollama + minio${NC}"
        echo -e "${BLUE}• Automation + Storage: n8n + postgres${NC}"
        echo -e "${BLUE}• AI + Automation: ollama + n8n${NC}"
        echo -e "${BLUE}• Multi-resource pipelines: 3+ services together${NC}"
        echo
        echo -e "${PURPLE}Example integration test (not run in demo):${NC}"
        echo -e "${YELLOW}./run.sh --scenarios \"category=ai-assistance,complexity=intermediate\"${NC}"
        echo
        pause_for_user
    fi
}

# Demo 6: Developer Experience
demo_developer_experience() {
    show_section "Developer-Friendly Experience" \
        "Features designed to make testing easy and approachable"
    
    run_demo_command "Simplified Quick Test" \
        "echo './quick-test.sh ollama' | head -1" \
        "One command to test any resource without complex options"
    
    run_demo_command "System Validation" \
        "./validate-setup.sh --verbose | head -20" \
        "Checks if your system is ready for testing and fixes common issues"
    
    if [[ "$QUICK_DEMO" != "true" ]]; then
        run_demo_command "Documentation Structure" \
            "ls -1 docs/*.md" \
            "Progressive documentation from beginner to advanced"
        
        echo -e "${PURPLE}Available Documentation:${NC}"
        echo -e "${BLUE}• docs/GETTING_STARTED.md - 2-minute quick start${NC}"
        echo -e "${BLUE}• docs/ARCHITECTURE_OVERVIEW.md - Visual system overview${NC}"
        echo -e "${BLUE}• docs/COMMON_PATTERNS.md - Copy-paste examples${NC}"
        echo -e "${BLUE}• docs/TROUBLESHOOTING.md - Problem solving guide${NC}"
        echo
        pause_for_user
    fi
}

# Demo 7: Performance & Quality
demo_performance_quality() {
    show_section "Performance & Quality Features" \
        "Enterprise-grade testing with performance monitoring and quality gates"
    
    echo -e "${PURPLE}Performance Features:${NC}"
    echo -e "${BLUE}• Baseline performance measurement${NC}"
    echo -e "${BLUE}• Category-specific benchmarks${NC}"
    echo -e "${BLUE}• Response time validation${NC}"
    echo -e "${BLUE}• Resource utilization monitoring${NC}"
    echo
    
    echo -e "${PURPLE}Quality Features:${NC}"
    echo -e "${BLUE}• No mocking - tests against real services${NC}"
    echo -e "${BLUE}• Test isolation with cleanup${NC}"
    echo -e "${BLUE}• Capability registry validation${NC}"
    echo -e "${BLUE}• Interface compliance checking${NC}"
    echo
    
    if [[ "$QUICK_DEMO" != "true" ]]; then
        run_demo_command "Framework Architecture" \
            "ls -1 framework/ | head -10" \
            "Sophisticated multi-layer architecture with specialized components"
    fi
    
    pause_for_user
}

# Show final summary
show_demo_summary() {
    show_header "🎉 Demo Complete - What You've Seen"
    
    echo -e "${BOLD}The Vrooli Resource Test Framework provides:${NC}"
    echo
    echo -e "${GREEN}✅ Automatic resource discovery and health checking${NC}"
    echo -e "${GREEN}✅ Comprehensive single-resource testing${NC}"
    echo -e "${GREEN}✅ Business scenario validation (complete workflows)${NC}"
    echo -e "${GREEN}✅ Integration pattern testing (multi-service)${NC}"
    echo -e "${GREEN}✅ Performance benchmarking and monitoring${NC}"
    echo -e "${GREEN}✅ Developer-friendly experience with progressive docs${NC}"
    echo -e "${GREEN}✅ Enterprise-grade quality and reliability${NC}"
    echo
    
    echo -e "${BOLD}🚀 Ready to Get Started?${NC}"
    echo
    echo -e "${BLUE}For beginners:${NC}"
    echo -e "  1. Check system: ${YELLOW}./validate-setup.sh${NC}"
    echo -e "  2. Quick test: ${YELLOW}./quick-test.sh${NC}"
    echo -e "  3. Read guide: ${YELLOW}cat docs/GETTING_STARTED.md${NC}"
    echo
    echo -e "${BLUE}For experienced developers:${NC}"
    echo -e "  1. See all options: ${YELLOW}./run.sh --help${NC}"
    echo -e "  2. Test everything: ${YELLOW}./run.sh${NC}"
    echo -e "  3. Try scenarios: ${YELLOW}./run.sh --scenarios-only${NC}"
    echo
    echo -e "${BLUE}For copy-paste examples:${NC}"
    echo -e "  • See ${YELLOW}docs/COMMON_PATTERNS.md${NC} for real-world usage patterns"
    echo -e "  • See ${YELLOW}docs/TROUBLESHOOTING.md${NC} for problem-solving"
    echo
    echo -e "${BOLD}${GREEN}🎯 The framework is ready for production use and can save significant development time!${NC}"
    echo
}

# Main demo function
main() {
    parse_args "$@"
    
    show_header "🎬 Vrooli Resource Test Framework - Capabilities Demo"
    
    if [[ "$INTERACTIVE" == "true" ]]; then
        echo -e "${YELLOW}Running in interactive mode - press Enter to advance through demos${NC}"
        echo
    fi
    
    if [[ "$QUICK_DEMO" == "true" ]]; then
        echo -e "${YELLOW}Quick demo mode - showing essential features only${NC}"
        echo
    fi
    
    # Check if we're in the right directory
    if [[ ! -f "$SCRIPT_DIR/run.sh" ]]; then
        echo -e "${RED}❌ Cannot find run.sh. Please run from the tests directory.${NC}"
        exit 1
    fi
    
    pause_for_user
    
    # Run demos
    demo_resource_discovery
    demo_single_resource_testing
    demo_framework_features
    demo_business_scenarios
    demo_integration_patterns
    demo_developer_experience
    demo_performance_quality
    
    show_demo_summary
}

main "$@"