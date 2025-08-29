#!/usr/bin/env bash

# N8N Workflow Deployment Script
# Deploys and tests the Claude investigation workflows

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
N8N_BASE_URL=${N8N_BASE_URL:-"http://localhost:5678"}
N8N_API_KEY=${N8N_API_KEY:-""}
API_BASE_URL="http://localhost:8090"

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# Check if N8N is running
check_n8n() {
    log "Checking N8N connection..."
    
    if curl -s "${N8N_BASE_URL}/rest/active-workflows" >/dev/null 2>&1; then
        success "N8N is running at $N8N_BASE_URL"
        return 0
    else
        error "N8N is not running at $N8N_BASE_URL"
        return 1
    fi
}

# Start N8N using Docker
start_n8n() {
    log "Starting N8N workflow engine..."
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        error "Docker is required to run N8N. Please install Docker first."
        return 1
    fi
    
    # Check if N8N container is already running
    if docker ps | grep -q n8n; then
        warn "N8N container is already running"
        return 0
    fi
    
    # Start N8N container
    docker run -d \
        --name app-issue-tracker-n8n \
        -p 5678:5678 \
        -e GENERIC_TIMEZONE="UTC" \
        -e TZ="UTC" \
        -e N8N_SECURE_COOKIE=false \
        -v n8n_data:/home/node/.n8n \
        -v $(pwd):/data \
        n8nio/n8n:latest
    
    # Wait for N8N to be ready
    log "Waiting for N8N to be ready..."
    local retries=30
    while ! curl -s "${N8N_BASE_URL}/rest/active-workflows" >/dev/null 2>&1; do
        if [ $retries -eq 0 ]; then
            error "N8N failed to start within timeout"
            return 1
        fi
        sleep 3
        retries=$((retries - 1))
        echo -n "."
    done
    echo
    
    success "N8N started successfully"
}

# Import workflow to N8N
import_workflow() {
    local workflow_file="$1"
    local workflow_name=$(basename "$workflow_file" .json)
    
    log "Importing workflow: $workflow_name"
    
    if [ ! -f "$workflow_file" ]; then
        error "Workflow file not found: $workflow_file"
        return 1
    fi
    
    # Import the workflow
    local response
    response=$(curl -s -X POST "${N8N_BASE_URL}/rest/workflows/import" \
        -H "Content-Type: application/json" \
        -d "@${workflow_file}")
    
    if echo "$response" | grep -q '"id"'; then
        local workflow_id=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        success "Workflow imported successfully: $workflow_name (ID: $workflow_id)"
        
        # Activate the workflow
        curl -s -X POST "${N8N_BASE_URL}/rest/workflows/${workflow_id}/activate" >/dev/null
        success "Workflow activated: $workflow_name"
        
        return 0
    else
        error "Failed to import workflow: $workflow_name"
        echo "Response: $response"
        return 1
    fi
}

# Deploy all workflows
deploy_workflows() {
    log "Deploying N8N workflows..."
    
    local workflow_dir="${PROJECT_DIR}/initialization/automation/n8n"
    
    if [ ! -d "$workflow_dir" ]; then
        error "Workflow directory not found: $workflow_dir"
        return 1
    fi
    
    # Import each workflow
    for workflow_file in "$workflow_dir"/*.json; do
        if [ -f "$workflow_file" ]; then
            import_workflow "$workflow_file"
        fi
    done
}

# Test webhook endpoints
test_webhooks() {
    log "Testing webhook endpoints..."
    
    # Test investigation webhook
    local investigation_webhook="${N8N_BASE_URL}/webhook/investigate-issue"
    
    log "Testing investigation webhook: $investigation_webhook"
    
    # Create test payload
    local test_payload='{
        "issue_id": "test-issue-001",
        "agent_id": "agent-test",
        "priority": "high",
        "title": "Test issue for webhook",
        "description": "This is a test issue to verify webhook functionality",
        "type": "bug",
        "error_message": "Test error message",
        "stack_trace": "Test stack trace",
        "project_path": "/tmp/test-project",
        "prompt_template": "Investigate this test issue",
        "timestamp": '$(date +%s)'
    }'
    
    local webhook_response
    webhook_response=$(curl -s -X POST "$investigation_webhook" \
        -H "Content-Type: application/json" \
        -d "$test_payload")
    
    if echo "$webhook_response" | grep -q "success\|queued\|started"; then
        success "Investigation webhook is working"
        log "Response: $webhook_response"
    else
        warn "Investigation webhook may not be working properly"
        log "Response: $webhook_response"
    fi
}

# Create workflow monitoring script
create_monitoring_script() {
    log "Creating workflow monitoring script..."
    
    cat > "${PROJECT_DIR}/scripts/monitor_workflows.sh" << 'EOF'
#!/usr/bin/env bash

# N8N Workflow Monitoring Script

N8N_BASE_URL=${N8N_BASE_URL:-"http://localhost:5678"}

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}N8N Workflow Status${NC}"
echo "===================="

# Get all workflows
workflows=$(curl -s "${N8N_BASE_URL}/rest/workflows" | jq -r '.data[]? | "\(.id) \(.name) \(.active)"')

if [ -n "$workflows" ]; then
    while IFS=' ' read -r id name active; do
        if [ "$active" = "true" ]; then
            status="${GREEN}●${NC} Active"
        else
            status="${RED}○${NC} Inactive"
        fi
        printf "%-30s %s\n" "$name" "$status"
    done <<< "$workflows"
else
    echo "No workflows found or N8N not accessible"
fi

echo ""
echo -e "${YELLOW}Recent Executions${NC}"
echo "=================="

# Get recent executions
executions=$(curl -s "${N8N_BASE_URL}/rest/executions?limit=10" | jq -r '.data[]? | "\(.workflowData.name // "Unknown") \(.finished) \(.mode)"')

if [ -n "$executions" ]; then
    while IFS=' ' read -r name finished mode; do
        if [ "$finished" = "true" ]; then
            status="${GREEN}✓${NC} Completed"
        else
            status="${YELLOW}⏳${NC} Running"
        fi
        printf "%-30s %-15s %s\n" "$name" "$mode" "$status"
    done <<< "$executions"
else
    echo "No recent executions found"
fi

echo ""
echo "Webhook endpoints:"
echo "- Investigation: ${N8N_BASE_URL}/webhook/investigate-issue"
EOF
    
    chmod +x "${PROJECT_DIR}/scripts/monitor_workflows.sh"
    success "Monitoring script created at scripts/monitor_workflows.sh"
}

# Test integration with API
test_api_integration() {
    log "Testing API integration with N8N workflows..."
    
    # Check if API is running
    if ! curl -s "${API_BASE_URL}/health" >/dev/null 2>&1; then
        warn "API server is not running at $API_BASE_URL"
        warn "Start the API server to test full integration"
        return 0
    fi
    
    # Create a test issue
    log "Creating test issue..."
    local issue_response
    issue_response=$(curl -s -X POST "${API_BASE_URL}/api/issues" \
        -H "Content-Type: application/json" \
        -d '{
            "title": "Test N8N Integration Issue",
            "description": "This is a test issue to verify N8N workflow integration",
            "type": "bug",
            "priority": "medium",
            "error_message": "Sample error for testing",
            "stack_trace": "Sample stack trace",
            "tags": ["test", "n8n", "integration"],
            "app_token": "test-token"
        }')
    
    local issue_id
    if echo "$issue_response" | grep -q "success"; then
        issue_id=$(echo "$issue_response" | grep -o '"issue_id":"[^"]*"' | cut -d'"' -f4)
        success "Test issue created: $issue_id"
        
        # Trigger investigation
        log "Triggering investigation for issue: $issue_id"
        local investigation_response
        investigation_response=$(curl -s -X POST "${API_BASE_URL}/api/investigate" \
            -H "Content-Type: application/json" \
            -d '{
                "issue_id": "'$issue_id'",
                "agent_id": "test-agent",
                "priority": "high"
            }')
        
        if echo "$investigation_response" | grep -q "success"; then
            success "Investigation triggered successfully"
            log "Response: $investigation_response"
        else
            warn "Investigation triggering failed"
            log "Response: $investigation_response"
        fi
    else
        warn "Failed to create test issue"
        log "Response: $issue_response"
    fi
}

# Cleanup function
cleanup_test_data() {
    log "Cleaning up test data..."
    
    # This would clean up test issues, executions, etc.
    # For now, just log the action
    log "Test cleanup completed (manual cleanup may be needed)"
}

# Main deployment function
main() {
    log "Deploying and testing N8N workflows..."
    
    # Check if N8N is running, start if not
    if ! check_n8n; then
        start_n8n
    fi
    
    # Deploy workflows
    deploy_workflows
    
    # Test webhooks
    test_webhooks
    
    # Create monitoring script
    create_monitoring_script
    
    # Test API integration
    test_api_integration
    
    success "N8N workflow deployment completed!"
    log ""
    log "Next steps:"
    log "1. Monitor workflows: ./scripts/monitor_workflows.sh"
    log "2. Access N8N UI: $N8N_BASE_URL"
    log "3. Test investigation: curl -X POST $API_BASE_URL/api/investigate"
    log ""
    log "Webhook endpoints:"
    log "- Investigation: $N8N_BASE_URL/webhook/investigate-issue"
}

# Handle command line arguments
case "${1:-deploy}" in
    deploy)
        main
        ;;
    test)
        test_webhooks
        test_api_integration
        ;;
    monitor)
        "${PROJECT_DIR}/scripts/monitor_workflows.sh"
        ;;
    cleanup)
        cleanup_test_data
        ;;
    *)
        echo "Usage: $0 [deploy|test|monitor|cleanup]"
        exit 1
        ;;
esac