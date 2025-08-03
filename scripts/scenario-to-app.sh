#!/bin/bash
# Scenario-to-App Conversion Script
# This script converts Vrooli scenarios into deployable applications

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIOS_DIR="$SCRIPT_DIR/__test/resources/scenarios"
DEFAULT_DEPLOY_MODE="local"
DEFAULT_VALIDATION_MODE="full"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script state
DEPLOYMENT_MODE=""
VALIDATION_MODE=""
SCENARIO_PATH=""
SCENARIO_ID=""
SCENARIO_NAME=""
VERBOSE=false
DRY_RUN=false

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${CYAN}[STEP]${NC} $1"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${PURPLE}[VERBOSE]${NC} $1"
    fi
}

# Error handling
trap 'log_error "Script failed at line $LINENO"; exit 1' ERR

# Helper functions
show_banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    Vrooli Scenario-to-App                   ║"
    echo "║              Convert scenarios into applications             ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

show_help() {
    cat << EOF
Usage: $0 [options] <scenario-path>

Convert a Vrooli scenario into a deployable application.

ARGUMENTS:
  scenario-path     Path to the scenario directory (relative to scenarios dir or absolute)

OPTIONS:
  -m, --mode MODE          Deployment mode (local, docker, k8s) [default: $DEFAULT_DEPLOY_MODE]
  -v, --validate MODE      Validation mode (none, basic, full) [default: $DEFAULT_VALIDATION_MODE]
  -d, --dry-run           Show what would be done without executing
  -V, --verbose           Enable verbose logging
  -h, --help              Show this help message

EXAMPLES:
  # Deploy ai-content-assistant-example locally
  $0 ai-content-assistant-example

  # Deploy with full path and validation
  $0 --mode local --validate full /path/to/scenario

  # Dry run to see what would happen
  $0 --dry-run --verbose ai-content-assistant-example

DEPLOYMENT MODES:
  local     - Deploy to local development environment (default)
  docker    - Deploy using Docker containers
  k8s       - Deploy to Kubernetes cluster

VALIDATION MODES:
  none      - Skip all validation
  basic     - Basic structure and syntax validation
  full      - Complete validation including resource health checks

EOF
}

# Parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--mode)
                DEPLOYMENT_MODE="$2"
                shift 2
                ;;
            -v|--validate)
                VALIDATION_MODE="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -V|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
            *)
                if [[ -z "$SCENARIO_PATH" ]]; then
                    SCENARIO_PATH="$1"
                else
                    log_error "Multiple scenario paths provided"
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # Set defaults
    DEPLOYMENT_MODE="${DEPLOYMENT_MODE:-$DEFAULT_DEPLOY_MODE}"
    VALIDATION_MODE="${VALIDATION_MODE:-$DEFAULT_VALIDATION_MODE}"
    
    # Validate arguments
    if [[ -z "$SCENARIO_PATH" ]]; then
        log_error "Scenario path is required"
        show_help
        exit 1
    fi
    
    # Resolve scenario path
    if [[ "$SCENARIO_PATH" =~ ^/ ]]; then
        # Absolute path
        SCENARIO_PATH="$SCENARIO_PATH"
    else
        # Relative to scenarios directory
        SCENARIO_PATH="$SCENARIOS_DIR/$SCENARIO_PATH"
    fi
    
    if [[ ! -d "$SCENARIO_PATH" ]]; then
        log_error "Scenario directory not found: $SCENARIO_PATH"
        exit 1
    fi
    
    # Extract scenario info
    if [[ -f "$SCENARIO_PATH/metadata.yaml" ]]; then
        SCENARIO_ID=$(grep "^[[:space:]]*id:" "$SCENARIO_PATH/metadata.yaml" | awk -F': ' '{print $2}' | tr -d '"' | xargs)
        SCENARIO_NAME=$(grep "^[[:space:]]*name:" "$SCENARIO_PATH/metadata.yaml" | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    else
        log_error "metadata.yaml not found in scenario directory"
        exit 1
    fi
    
    log_verbose "Deployment mode: $DEPLOYMENT_MODE"
    log_verbose "Validation mode: $VALIDATION_MODE"
    log_verbose "Scenario path: $SCENARIO_PATH"
    log_verbose "Scenario ID: $SCENARIO_ID"
    log_verbose "Scenario name: $SCENARIO_NAME"
    log_verbose "Dry run: $DRY_RUN"
}

# Validation functions
validate_scenario_structure() {
    log_step "Validating scenario structure..."
    
    local validation_script="$SCENARIO_PATH/deployment/validate.sh"
    
    if [[ -f "$validation_script" && -x "$validation_script" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would run: $validation_script structure"
        else
            log_verbose "Running structure validation..."
            if "$validation_script" structure; then
                log_success "Scenario structure validation passed"
            else
                log_error "Scenario structure validation failed"
                return 1
            fi
        fi
    else
        log_warning "No validation script found, performing basic checks..."
        
        # Basic structure checks
        local required_files=(
            "metadata.yaml"
            "manifest.yaml"
            "test.sh"
            "deployment/startup.sh"
            "initialization/database/schema.sql"
            "initialization/configuration/app-config.json"
        )
        
        for file in "${required_files[@]}"; do
            if [[ -f "$SCENARIO_PATH/$file" ]]; then
                log_success "✓ $file exists"
            else
                log_error "✗ $file missing"
                return 1
            fi
        done
    fi
}

validate_resources() {
    if [[ "$VALIDATION_MODE" == "full" ]]; then
        log_step "Validating required resources..."
        
        local validation_script="$SCENARIO_PATH/deployment/validate.sh"
        
        if [[ -f "$validation_script" && -x "$validation_script" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Would run: $validation_script with-resources"
            else
                log_verbose "Running resource validation..."
                if "$validation_script" with-resources; then
                    log_success "Resource validation passed"
                else
                    log_error "Resource validation failed"
                    log_error "Please ensure all required resources are running"
                    return 1
                fi
            fi
        else
            log_warning "No validation script found, skipping resource validation"
        fi
    else
        log_info "Skipping resource validation (mode: $VALIDATION_MODE)"
    fi
}

# Deployment functions
generate_resource_config() {
    log_step "Generating resource configuration..."
    
    local required_resources
    required_resources=$(grep -A 20 "required:" "$SCENARIO_PATH/metadata.yaml" | grep "^[[:space:]]*-" | sed 's/^[[:space:]]*-[[:space:]]*//' | tr '\n' ' ')
    
    log_verbose "Required resources: $required_resources"
    
    # Generate .vrooli/resources.local.json with only required resources
    local config_dir="$HOME/.vrooli"
    local config_file="$config_dir/resources.local.json"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would generate resource config with resources: $required_resources"
        log_info "Would update: $config_file"
    else
        # Backup existing config
        if [[ -f "$config_file" ]]; then
            cp "$config_file" "${config_file}.backup.$(date +%s)"
            log_verbose "Backed up existing resource config"
        fi
        
        # Create minimal config with only required resources
        mkdir -p "$config_dir"
        
        # This is a simplified version - in production you'd use proper JSON manipulation
        cat > "$config_file" << EOF
{
  "services": {
    "ai": {
EOF
        
        for resource in $required_resources; do
            case "$resource" in
                "ollama")
                    cat >> "$config_file" << 'EOF'
      "ollama": {
        "enabled": true,
        "baseUrl": "http://localhost:11434"
      },
EOF
                    ;;
                "whisper")
                    cat >> "$config_file" << 'EOF'
      "whisper": {
        "enabled": true,
        "baseUrl": "http://localhost:8090"
      },
EOF
                    ;;
                "comfyui")
                    cat >> "$config_file" << 'EOF'
      "comfyui": {
        "enabled": true,
        "baseUrl": "http://localhost:8188"
      },
EOF
                    ;;
            esac
        done
        
        cat >> "$config_file" << 'EOF'
      "dummy": {"enabled": false}
    },
    "automation": {
EOF
        
        for resource in $required_resources; do
            case "$resource" in
                "n8n")
                    cat >> "$config_file" << 'EOF'
      "n8n": {
        "enabled": true,
        "baseUrl": "http://localhost:5678"
      },
EOF
                    ;;
                "windmill")
                    cat >> "$config_file" << 'EOF'
      "windmill": {
        "enabled": true,
        "baseUrl": "http://localhost:5681"
      },
EOF
                    ;;
            esac
        done
        
        cat >> "$config_file" << 'EOF'
      "dummy": {"enabled": false}
    },
    "storage": {
EOF
        
        for resource in $required_resources; do
            case "$resource" in
                "postgres")
                    cat >> "$config_file" << 'EOF'
      "postgres": {
        "enabled": true,
        "baseUrl": "http://localhost:5433"
      },
EOF
                    ;;
                "redis")
                    cat >> "$config_file" << 'EOF'
      "redis": {
        "enabled": true,
        "baseUrl": "redis://localhost:6380"
      },
EOF
                    ;;
                "minio")
                    cat >> "$config_file" << 'EOF'
      "minio": {
        "enabled": true,
        "baseUrl": "http://localhost:9000"
      },
EOF
                    ;;
                "qdrant")
                    cat >> "$config_file" << 'EOF'
      "qdrant": {
        "enabled": true,
        "baseUrl": "http://localhost:6333"
      },
EOF
                    ;;
            esac
        done
        
        cat >> "$config_file" << 'EOF'
      "dummy": {"enabled": false}
    }
  }
}
EOF
        
        # Clean up trailing commas (basic cleanup)
        sed -i 's/,$//' "$config_file"
        
        log_success "Generated resource configuration: $config_file"
    fi
}

deploy_application() {
    log_step "Deploying application..."
    
    local startup_script="$SCENARIO_PATH/deployment/startup.sh"
    
    if [[ -f "$startup_script" && -x "$startup_script" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would run: $startup_script deploy"
        else
            log_verbose "Running deployment script..."
            if "$startup_script" deploy; then
                log_success "Application deployed successfully"
            else
                log_error "Application deployment failed"
                return 1
            fi
        fi
    else
        log_error "Deployment script not found or not executable: $startup_script"
        return 1
    fi
}

start_monitoring() {
    log_step "Starting application monitoring..."
    
    local monitor_script="$SCENARIO_PATH/deployment/monitor.sh"
    
    if [[ -f "$monitor_script" && -x "$monitor_script" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would run: $monitor_script start"
        else
            log_verbose "Starting monitoring..."
            if "$monitor_script" start >/dev/null 2>&1; then
                log_success "Application monitoring started"
            else
                log_warning "Failed to start monitoring (non-critical)"
            fi
        fi
    else
        log_warning "Monitoring script not found, skipping monitoring setup"
    fi
}

run_integration_tests() {
    log_step "Running integration tests..."
    
    local test_script="$SCENARIO_PATH/test.sh"
    
    if [[ -f "$test_script" && -x "$test_script" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would run: $test_script"
        else
            log_verbose "Running integration tests..."
            if "$test_script"; then
                log_success "Integration tests passed"
            else
                log_error "Integration tests failed"
                return 1
            fi
        fi
    else
        log_warning "Test script not found or not executable: $test_script"
    fi
}

# Output functions
show_deployment_summary() {
    echo ""
    echo -e "${GREEN}🎉 Deployment Summary${NC}"
    echo "════════════════════════════════════════════════════════════════"
    echo -e "📋 Scenario: ${CYAN}$SCENARIO_NAME${NC} ($SCENARIO_ID)"
    echo -e "📁 Path: ${BLUE}$SCENARIO_PATH${NC}"
    echo -e "🚀 Mode: ${YELLOW}$DEPLOYMENT_MODE${NC}"
    echo -e "✅ Status: ${GREEN}Successfully Deployed${NC}"
    echo ""
    
    # Extract and show application URLs
    log_info "Application Access:"
    
    # Check metadata for UI requirements
    local requires_ui
    requires_ui=$(grep "requires_ui:" "$SCENARIO_PATH/metadata.yaml" | awk '{print $2}' || echo "false")
    
    if [[ "$requires_ui" == "true" ]]; then
        echo -e "  🖥️  UI Application: ${BLUE}http://localhost:5681/app/$SCENARIO_ID${NC}"
        echo -e "  ⚙️  Admin Interface: ${BLUE}http://localhost:5681/admin/$SCENARIO_ID${NC}"
    fi
    
    # Check for n8n workflows
    local required_resources
    required_resources=$(grep -A 20 "required:" "$SCENARIO_PATH/metadata.yaml" | grep "^[[:space:]]*-" | sed 's/^[[:space:]]*-[[:space:]]*//' | tr '\n' ' ')
    
    if [[ "$required_resources" =~ "n8n" ]]; then
        echo -e "  📡 Webhook: ${BLUE}http://localhost:5678/webhook/${SCENARIO_ID}-webhook${NC}"
        echo -e "  🔧 n8n Editor: ${BLUE}http://localhost:5678${NC}"
    fi
    
    if [[ "$required_resources" =~ "postgres" ]]; then
        echo -e "  🗄️  Database: ${BLUE}postgresql://postgres:postgres@localhost:5433/${SCENARIO_ID//-/_}${NC}"
    fi
    
    echo ""
    echo -e "${YELLOW}📊 Management Commands:${NC}"
    echo -e "  Monitoring: ${BLUE}$SCENARIO_PATH/deployment/monitor.sh status${NC}"
    echo -e "  Logs: ${BLUE}$SCENARIO_PATH/deployment/monitor.sh logs${NC}"
    echo -e "  Validation: ${BLUE}$SCENARIO_PATH/deployment/validate.sh post-deployment${NC}"
    echo -e "  Tests: ${BLUE}$SCENARIO_PATH/test.sh${NC}"
    echo ""
}

# Main execution function
main() {
    show_banner
    
    log "Starting scenario-to-app conversion..."
    log_info "Scenario: $SCENARIO_NAME ($SCENARIO_ID)"
    log_info "Mode: $DEPLOYMENT_MODE"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No actual changes will be made"
    fi
    
    # Phase 1: Validation
    if [[ "$VALIDATION_MODE" != "none" ]]; then
        log_step "Phase 1: Validation"
        validate_scenario_structure
        validate_resources
        log_success "Validation phase completed"
    else
        log_info "Skipping validation phase"
    fi
    
    echo ""
    
    # Phase 2: Configuration
    log_step "Phase 2: Configuration"
    generate_resource_config
    log_success "Configuration phase completed"
    
    echo ""
    
    # Phase 3: Deployment
    log_step "Phase 3: Deployment"
    deploy_application
    log_success "Deployment phase completed"
    
    echo ""
    
    # Phase 4: Monitoring & Testing
    log_step "Phase 4: Monitoring & Testing"
    start_monitoring
    run_integration_tests
    log_success "Monitoring & testing phase completed"
    
    echo ""
    
    # Summary
    if [[ "$DRY_RUN" == "true" ]]; then
        log_success "Dry run completed successfully!"
        log_info "Run without --dry-run to perform actual deployment"
    else
        show_deployment_summary
    fi
}

# Entry point
parse_arguments "$@"
main