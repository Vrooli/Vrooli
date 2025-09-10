#!/bin/bash
# ====================================================================
# [Scenario Name] Test
# ====================================================================
#
# This test validates [what the scenario demonstrates]
#
# ====================================================================

set -euo pipefail

# Source the test framework
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${SCRIPT_DIR:-${APP_ROOT}/scripts/scenarios}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/helpers/fixtures.sh"
source "$SCRIPT_DIR/framework/helpers/metadata.sh"
source "$SCRIPT_DIR/framework/helpers/secure-config.sh"

# Load scenario metadata
SCENARIO_DIR="${APP_ROOT}/scripts/scenarios/templates/full"
SERVICE_FILE="${SCENARIO_DIR}/service.json"

# Parse service configuration
REQUIRED_RESOURCES=($(jq -r '.resources | to_entries[] | select(.value.required == true) | .key' "$SERVICE_FILE" 2>/dev/null))
TEST_TIMEOUT=$(jq -r '.deployment.testing.timeout // "30m"' "$SERVICE_FILE" 2>/dev/null | sed 's/[ms]//g')
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Service configuration from secure config
export_service_urls

# Scenario-specific variables
TEST_SESSION_ID="scenario_session_$(date +%s)"

# ====================================================================
# Resource Integration Helpers (Optional)
# ====================================================================
# These functions help integrate resource-specific artifacts.
# Only use the ones relevant to your scenario.

# Import n8n workflow from resources directory
import_n8n_workflow() {
    local workflow_file="${1:-}"
    local workflow_path="${SCENARIO_DIR}/resources/n8n/${workflow_file}"
    
    if [[ ! -f "$workflow_path" ]]; then
        echo "⚠️  Workflow file not found: $workflow_path"
        return 1
    fi
    
    log_step "n8n" "Importing workflow: $workflow_file"
    
    # Import workflow via n8n API
    local response
    response=$(curl -s -X POST "${N8N_BASE_URL}/api/v1/workflows" \
        -H "X-N8N-API-KEY: ${N8N_API_KEY:-}" \
        -H "Content-Type: application/json" \
        -d @"$workflow_path")
    
    if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
        local workflow_id=$(echo "$response" | jq -r '.id')
        echo "✅ Workflow imported with ID: $workflow_id"
        
        # Activate the workflow
        curl -s -X PATCH "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" \
            -H "X-N8N-API-KEY: ${N8N_API_KEY:-}" \
            -H "Content-Type: application/json" \
            -d '{"active": true}' >/dev/null
        
        return 0
    else
        echo "❌ Failed to import workflow"
        return 1
    fi
}

# Initialize PostgreSQL database with schema and seed data
init_postgres_database() {
    local schema_file="${SCENARIO_DIR}/resources/postgres/schema.sql"
    local seed_file="${SCENARIO_DIR}/resources/postgres/seed.sql"
    
    if [[ -f "$schema_file" ]]; then
        log_step "postgres" "Applying database schema"
        PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" \
            -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
            -f "$schema_file" >/dev/null 2>&1 || {
            echo "❌ Failed to apply schema"
            return 1
        }
    fi
    
    if [[ -f "$seed_file" ]]; then
        log_step "postgres" "Loading seed data"
        PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" \
            -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
            -f "$seed_file" >/dev/null 2>&1 || {
            echo "❌ Failed to load seed data"
            return 1
        }
    fi
    
    echo "✅ Database initialized"
    return 0
}

# Deploy Windmill scripts
deploy_windmill_scripts() {
    local scripts_dir="${SCENARIO_DIR}/resources/windmill"
    
    if [[ ! -d "$scripts_dir" ]]; then
        echo "⚠️  No Windmill scripts found"
        return 0
    fi
    
    log_step "windmill" "Deploying scripts"
    
    # Check if wmill CLI is available
    if ! command -v wmill >/dev/null 2>&1; then
        echo "⚠️  Windmill CLI not installed, skipping deployment"
        return 0
    fi
    
    # Deploy scripts
    cd "$scripts_dir"
    wmill sync push --workspace "${WINDMILL_WORKSPACE:-default}" || {
        echo "❌ Failed to deploy Windmill scripts"
        return 1
    }
    cd - >/dev/null
    
    echo "✅ Windmill scripts deployed"
    return 0
}

# Load environment configuration
load_scenario_config() {
    local env_file="${SCENARIO_DIR}/resources/config/.env"
    local env_template="${SCENARIO_DIR}/resources/config/.env.template"
    
    # Create .env from template if it doesn't exist
    if [[ ! -f "$env_file" && -f "$env_template" ]]; then
        log_step "config" "Creating environment configuration"
        cp "$env_template" "$env_file"
    fi
    
    # Load environment variables if file exists
    if [[ -f "$env_file" ]]; then
        set -a
        source "$env_file"
        set +a
        echo "✅ Scenario configuration loaded"
    fi
}

# Import all declared artifacts from metadata
import_all_artifacts() {
    echo "📦 Importing resource artifacts..."
    
    # Check if artifacts are declared in metadata
    if ! grep -q "^artifacts:" "$METADATA_FILE" 2>/dev/null; then
        echo "  No artifacts declared in metadata"
        return 0
    fi
    
    # Import n8n workflows if declared
    if grep -q "workflows:" "$METADATA_FILE" 2>/dev/null; then
        local workflows=($(grep -A5 "n8n:" "$METADATA_FILE" | grep -E "^\s*-\s" | sed 's/^[[:space:]]*-[[:space:]]*"*//' | sed 's/"*$//' | tr -d "'"))
        for workflow in "${workflows[@]}"; do
            import_n8n_workflow "$workflow"
        done
    fi
    
    # Initialize database if declared
    if grep -q "postgres:" "$METADATA_FILE" 2>/dev/null; then
        init_postgres_database
    fi
    
    # Deploy Windmill scripts if declared
    if grep -q "windmill:" "$METADATA_FILE" 2>/dev/null; then
        deploy_windmill_scripts
    fi
    
    echo "✅ All artifacts imported"
}

# Business scenario setup
setup_business_scenario() {
    echo "🚀 Setting up [Scenario Name]..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Verify all required resources are available
    require_resources "${REQUIRED_RESOURCES[@]}"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    # Setup test environment
    create_test_env "$TEST_SESSION_ID"
    
    # Optional: Load scenario-specific configuration
    # load_scenario_config
    
    # Optional: Import all artifacts declared in metadata
    # import_all_artifacts
    
    # Optional: Or import specific resources manually
    # import_n8n_workflow "main-workflow.json"
    # init_postgres_database
    # deploy_windmill_scripts
    
    echo "✅ Scenario setup complete"
}

# Test 1: [First Test Name]
test_first_capability() {
    echo "🧪 Testing [first capability]..."
    
    log_step "1/3" "Step description"
    # Test implementation
    
    log_step "2/3" "Step description"
    # Test implementation
    
    log_step "3/3" "Step description"
    # Test implementation
    
    echo "✅ [First capability] test passed"
}

# Test 2: [Second Test Name]
test_second_capability() {
    echo "🧪 Testing [second capability]..."
    
    # Test implementation
    
    echo "✅ [Second capability] test passed"
}

# Business value assessment
assess_business_value() {
    echo ""
    echo "💼 Business Value Assessment:"
    echo "=================================="
    
    # Calculate business metrics based on test results
    local capabilities_met=0
    local total_capabilities=2
    
    # Add business logic to evaluate scenario success
    
    echo "📊 Business Readiness Score: $capabilities_met/$total_capabilities"
}

# Main execution
main() {
    export TEST_START_TIME=$(date +%s)
    
    echo "Starting [Scenario Name] Test"
    echo "Required Resources: ${REQUIRED_RESOURCES[*]}"
    echo "Test Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_business_scenario
    
    # Run tests
    test_first_capability
    test_second_capability
    
    # Business validation
    assess_business_value
    
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ [Scenario name] failed"
        exit 1
    else
        echo "✅ [Scenario name] passed"
        exit 0
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi