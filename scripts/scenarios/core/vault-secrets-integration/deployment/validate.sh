#!/bin/bash
# Validation script for Vault Secrets Integration
# This script performs comprehensive pre-deployment and post-deployment validation

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_ID="vault-secrets-integration"
SCENARIO_NAME="Vault Secrets Integration"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validation state
VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0
VALIDATION_PASSED=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((VALIDATION_PASSED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((VALIDATION_WARNINGS++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((VALIDATION_ERRORS++))
}

# Test functions
test_file_exists() {
    local file_path="$1"
    local description="$2"
    
    if [[ -f "$file_path" ]]; then
        log_success "$description exists: $file_path"
        return 0
    else
        log_error "$description missing: $file_path"
        return 1
    fi
}

test_directory_exists() {
    local dir_path="$1"
    local description="$2"
    
    if [[ -d "$dir_path" ]]; then
        log_success "$description exists: $dir_path"
        return 0
    else
        log_error "$description missing: $dir_path"
        return 1
    fi
}

test_json_valid() {
    local file_path="$1"
    local description="$2"
    
    if [[ -f "$file_path" ]]; then
        if jq empty "$file_path" 2>/dev/null; then
            log_success "$description has valid JSON"
            return 0
        else
            log_error "$description has invalid JSON"
            return 1
        fi
    else
        log_warning "$description file not found: $file_path"
        return 1
    fi
}

test_yaml_valid() {
    local file_path="$1"
    local description="$2"
    
    if [[ -f "$file_path" ]]; then
        # Basic YAML syntax check (requires python)
        if command -v python3 >/dev/null 2>&1; then
            if python3 -c "import yaml; yaml.safe_load(open('$file_path'))" 2>/dev/null; then
                log_success "$description has valid YAML"
                return 0
            else
                log_error "$description has invalid YAML"
                return 1
            fi
        else
            log_warning "Cannot validate YAML (python3 not available)"
            return 0
        fi
    else
        log_warning "$description file not found: $file_path"
        return 1
    fi
}

test_service_health() {
    local service_name="$1"
    local health_url="$2"
    local timeout="${3:-5}"
    
    if curl -sf --max-time "$timeout" "$health_url" >/dev/null 2>&1; then
        log_success "$service_name is healthy ($health_url)"
        return 0
    else
        log_error "$service_name is not healthy ($health_url)"
        return 1
    fi
}

test_database_connection() {
    local db_name="$1"
    
    if pg_isready -h localhost -p 5433 -d "$db_name" >/dev/null 2>&1; then
        log_success "Database connection successful: $db_name"
        return 0
    else
        log_error "Database connection failed: $db_name"
        return 1
    fi
}

test_sql_file() {
    local file_path="$1"
    local description="$2"
    
    if [[ -f "$file_path" ]]; then
        # Basic SQL syntax check
        if grep -q "CREATE\|INSERT\|SELECT" "$file_path" 2>/dev/null; then
            log_success "$description contains SQL statements"
            return 0
        else
            log_warning "$description might not contain valid SQL"
            return 1
        fi
    else
        log_warning "$description file not found: $file_path"
        return 1
    fi
}

# Validation phases
validate_structure() {
    log_info "=== Validating Scenario Structure ==="
    
    # Core files
    test_file_exists "$SCENARIO_DIR/metadata.yaml" "Metadata file"
    test_file_exists "$SCENARIO_DIR/manifest.yaml" "Manifest file"
    test_file_exists "$SCENARIO_DIR/test.sh" "Test script"
    test_file_exists "$SCENARIO_DIR/README.md" "README file"
    
    # Initialization structure
    test_directory_exists "$SCENARIO_DIR/initialization" "Initialization directory"
    test_directory_exists "$SCENARIO_DIR/initialization/database" "Database initialization directory"
    test_directory_exists "$SCENARIO_DIR/initialization/workflows" "Workflows directory"
    test_directory_exists "$SCENARIO_DIR/initialization/configuration" "Configuration directory"
    
    # Deployment structure
    test_directory_exists "$SCENARIO_DIR/deployment" "Deployment directory"
    test_file_exists "$SCENARIO_DIR/deployment/startup.sh" "Startup script"
    test_file_exists "$SCENARIO_DIR/deployment/validate.sh" "Validation script"
    
    # Database files
    test_file_exists "$SCENARIO_DIR/initialization/database/schema.sql" "Database schema"
    test_file_exists "$SCENARIO_DIR/initialization/database/seed.sql" "Database seed data"
    
    # Configuration files
    test_file_exists "$SCENARIO_DIR/initialization/configuration/app-config.json" "App configuration"
    test_file_exists "$SCENARIO_DIR/initialization/configuration/resource-urls.json" "Resource URLs configuration"
    test_file_exists "$SCENARIO_DIR/initialization/configuration/feature-flags.json" "Feature flags configuration"
    
    # Workflow files
    test_file_exists "$SCENARIO_DIR/initialization/workflows/triggers.yaml" "Workflow triggers configuration"
    
    echo ""
}

validate_content() {
    log_info "=== Validating File Content ==="
    
    # Validate YAML files
    test_yaml_valid "$SCENARIO_DIR/metadata.yaml" "Metadata"
    test_yaml_valid "$SCENARIO_DIR/manifest.yaml" "Manifest"
    test_yaml_valid "$SCENARIO_DIR/initialization/workflows/triggers.yaml" "Workflow triggers"
    
    # Validate JSON files
    test_json_valid "$SCENARIO_DIR/initialization/configuration/app-config.json" "App configuration"
    test_json_valid "$SCENARIO_DIR/initialization/configuration/resource-urls.json" "Resource URLs"
    test_json_valid "$SCENARIO_DIR/initialization/configuration/feature-flags.json" "Feature flags"
    
    # Validate SQL files
    test_sql_file "$SCENARIO_DIR/initialization/database/schema.sql" "Database schema"
    test_sql_file "$SCENARIO_DIR/initialization/database/seed.sql" "Database seed data"
    
    # Validate workflow files
    for workflow_file in "$SCENARIO_DIR/initialization/workflows/n8n"/*.json; do
        if [[ -f "$workflow_file" ]]; then
            test_json_valid "$workflow_file" "n8n workflow $(basename "$workflow_file")"
        fi
    done
    
    # Validate UI files
    if [[ -f "$SCENARIO_DIR/initialization/ui/windmill-app.json" ]]; then
        test_json_valid "$SCENARIO_DIR/initialization/ui/windmill-app.json" "Windmill app configuration"
    fi
    
    echo ""
}

validate_configuration() {
    log_info "=== Validating Configuration Values ==="
    
    # Extract and validate metadata
    if [[ -f "$SCENARIO_DIR/metadata.yaml" ]]; then
        local scenario_id
        scenario_id=$(grep "^[[:space:]]*id:" "$SCENARIO_DIR/metadata.yaml" | awk -F': ' '{print $2}' | tr -d '"' | xargs)
        
        if [[ "$scenario_id" == "$SCENARIO_ID" ]]; then
            log_success "Scenario ID matches: $scenario_id"
        else
            log_error "Scenario ID mismatch: expected $SCENARIO_ID, got $scenario_id"
        fi
        
        # Check for required fields
        if grep -q "required:" "$SCENARIO_DIR/metadata.yaml"; then
            log_success "Required resources are specified"
        else
            log_error "No required resources specified"
        fi
        
        if grep -q "business:" "$SCENARIO_DIR/metadata.yaml"; then
            log_success "Business configuration is present"
        else
            log_warning "Business configuration is missing"
        fi
    fi
    
    # Validate resource URLs match expected services
    if [[ -f "$SCENARIO_DIR/initialization/configuration/resource-urls.json" ]]; then
        local required_resources
        required_resources=$(grep -A 10 "required:" "$SCENARIO_DIR/metadata.yaml" | grep "^[[:space:]]*-" | sed 's/^[[:space:]]*-[[:space:]]*//' | tr '\n' ' ')
        
        for resource in $required_resources; do
            case "$resource" in
                "ollama")
                    if jq -e '.ai.ollama.base_url' "$SCENARIO_DIR/initialization/configuration/resource-urls.json" >/dev/null 2>&1; then
                        log_success "Ollama configuration present in resource URLs"
                    else
                        log_error "Ollama required but not configured in resource URLs"
                    fi
                    ;;
                "n8n")
                    if jq -e '.automation.n8n.base_url' "$SCENARIO_DIR/initialization/configuration/resource-urls.json" >/dev/null 2>&1; then
                        log_success "n8n configuration present in resource URLs"
                    else
                        log_error "n8n required but not configured in resource URLs"
                    fi
                    ;;
                "postgres")
                    if jq -e '.storage.postgres.connection_url' "$SCENARIO_DIR/initialization/configuration/resource-urls.json" >/dev/null 2>&1; then
                        log_success "PostgreSQL configuration present in resource URLs"
                    else
                        log_error "PostgreSQL required but not configured in resource URLs"
                    fi
                    ;;
            esac
        done
    fi
    
    echo ""
}

validate_resources() {
    log_info "=== Validating Required Resources ==="
    
    # Extract required resources from metadata
    local required_resources
    if [[ -f "$SCENARIO_DIR/metadata.yaml" ]]; then
        required_resources=$(grep -A 10 "required:" "$SCENARIO_DIR/metadata.yaml" | grep "^[[:space:]]*-" | sed 's/^[[:space:]]*-[[:space:]]*//' | tr '\n' ' ')
        
        log_info "Required resources: $required_resources"
        
        for resource in $required_resources; do
            case "$resource" in
                "ollama")
                    test_service_health "Ollama" "http://localhost:11434/api/tags"
                    ;;
                "n8n")
                    test_service_health "n8n" "http://localhost:5678/healthz"
                    ;;
                "postgres")
                    if command -v pg_isready >/dev/null 2>&1; then
                        test_database_connection "${SCENARIO_ID//-/_}"
                    else
                        test_service_health "PostgreSQL" "localhost:5433"
                    fi
                    ;;
                "redis")
                    if command -v redis-cli >/dev/null 2>&1; then
                        if redis-cli -h localhost -p 6380 ping >/dev/null 2>&1; then
                            log_success "Redis is healthy"
                        else
                            log_error "Redis is not responding"
                        fi
                    else
                        log_warning "redis-cli not available, skipping Redis health check"
                    fi
                    ;;
                "windmill")
                    test_service_health "Windmill" "http://localhost:5681/api/version"
                    ;;
                "whisper")
                    test_service_health "Whisper" "http://localhost:8090/"
                    ;;
                "comfyui")
                    test_service_health "ComfyUI" "http://localhost:8188/"
                    ;;
                "minio")
                    test_service_health "MinIO" "http://localhost:9000/minio/health/live"
                    ;;
                "qdrant")
                    test_service_health "Qdrant" "http://localhost:6333/"
                    ;;
                "questdb")
                    test_service_health "QuestDB" "http://localhost:9010/"
                    ;;
                *)
                    log_warning "Unknown resource: $resource"
                    ;;
            esac
        done
    else
        log_error "Cannot read metadata.yaml to determine required resources"
    fi
    
    echo ""
}

validate_deployment() {
    log_info "=== Validating Deployment Readiness ==="
    
    # Check if deployment scripts are executable
    if [[ -x "$SCENARIO_DIR/deployment/startup.sh" ]]; then
        log_success "Startup script is executable"
    else
        log_error "Startup script is not executable"
    fi
    
    if [[ -x "$SCENARIO_DIR/test.sh" ]]; then
        log_success "Test script is executable"
    else
        log_warning "Test script is not executable"
    fi
    
    # Validate disk space
    local available_space
    available_space=$(df "$SCENARIO_DIR" | awk 'NR==2 {print $4}')
    if [[ "$available_space" -gt 1048576 ]]; then # 1GB in KB
        log_success "Sufficient disk space available"
    else
        log_warning "Low disk space: $(($available_space / 1024))MB available"
    fi
    
    # Check port availability for common services
    local ports_to_check=("3000" "5678" "5681" "1880")
    for port in "${ports_to_check[@]}"; do
        if lsof -Pi ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
            log_info "Port $port is in use (expected for running services)"
        else
            log_info "Port $port is available"
        fi
    done
    
    echo ""
}

# Summary function
print_summary() {
    echo ""
    echo "=== Validation Summary ==="
    echo -e "✅ Passed: ${GREEN}$VALIDATION_PASSED${NC}"
    echo -e "⚠️  Warnings: ${YELLOW}$VALIDATION_WARNINGS${NC}"
    echo -e "❌ Errors: ${RED}$VALIDATION_ERRORS${NC}"
    echo ""
    
    if [[ $VALIDATION_ERRORS -eq 0 ]]; then
        echo -e "${GREEN}🎉 Scenario validation passed!${NC}"
        echo -e "${BLUE}ℹ️  Scenario is ready for deployment${NC}"
        return 0
    else
        echo -e "${RED}❌ Scenario validation failed${NC}"
        echo -e "${YELLOW}⚠️  Please fix the errors before deploying${NC}"
        return 1
    fi
}

# Main validation function
main() {
    echo "🔍 Validating scenario: $SCENARIO_NAME ($SCENARIO_ID)"
    echo "📁 Scenario directory: $SCENARIO_DIR"
    echo ""
    
    validate_structure
    validate_content
    validate_configuration
    
    # Only validate resources if requested or in post-deployment mode
    if [[ "${1:-}" == "--with-resources" || "${1:-}" == "post-deployment" ]]; then
        validate_resources
    else
        log_info "=== Skipping Resource Validation ==="
        log_info "Use '--with-resources' to validate running services"
        echo ""
    fi
    
    validate_deployment
    
    print_summary
}

# Handle command line arguments
case "${1:-validate}" in
    "validate"|"pre-deployment")
        main
        ;;
    "post-deployment")
        main "post-deployment"
        ;;
    "with-resources"|"--with-resources")
        main "--with-resources"
        ;;
    "structure")
        validate_structure
        print_summary
        ;;
    "content")
        validate_content
        print_summary
        ;;
    "config"|"configuration")
        validate_configuration
        print_summary
        ;;
    "resources")
        validate_resources
        print_summary
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  validate         - Full validation without resource health checks (default)"
        echo "  pre-deployment   - Same as validate"
        echo "  post-deployment  - Full validation including resource health checks"
        echo "  with-resources   - Full validation including resource health checks"
        echo "  structure        - Validate only file/directory structure"
        echo "  content          - Validate only file content (JSON/YAML syntax)"
        echo "  configuration    - Validate only configuration values"
        echo "  resources        - Validate only resource health"
        echo "  help             - Show this help message"
        echo ""
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac