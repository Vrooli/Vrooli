#!/bin/bash
set -euo pipefail

# Scenario Generator V1 - Deployment Script
# This script initializes the autonomous scenario generation pipeline

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Configuration
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5433}"
POSTGRES_DB="${POSTGRES_DB:-vrooli_client}"
POSTGRES_USER="${POSTGRES_USER:-vrooli}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"

N8N_HOST="${N8N_HOST:-localhost}"
N8N_PORT="${N8N_PORT:-5678}"

WINDMILL_HOST="${WINDMILL_HOST:-localhost}"
WINDMILL_PORT="${WINDMILL_PORT:-5681}"

MINIO_HOST="${MINIO_HOST:-localhost}"
MINIO_PORT="${MINIO_PORT:-9000}"

REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6380}"

CLAUDE_CODE_AVAILABLE="${CLAUDE_CODE_AVAILABLE:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if a service is available
check_service() {
    local host="$1"
    local port="$2"
    local service_name="$3"
    
    if timeout 5 bash -c "cat < /dev/null > /dev/tcp/$host/$port" 2>/dev/null; then
        log_success "$service_name is available at $host:$port"
        return 0
    else
        log_error "$service_name is not available at $host:$port"
        return 1
    fi
}

# Execute SQL script
execute_sql() {
    local sql_file="$1"
    local description="$2"
    
    if [ ! -f "$sql_file" ]; then
        log_error "SQL file not found: $sql_file"
        return 1
    fi
    
    log_info "Executing $description..."
    
    # Use containerized psql if local psql is not available
    if command -v psql >/dev/null 2>&1; then
        # Use local psql if available
        if [ -n "$POSTGRES_PASSWORD" ]; then
            PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f "$sql_file"
        else
            psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f "$sql_file"
        fi
    else
        # Use psql from PostgreSQL container
        local container_name="vrooli-postgres-scenario-generator-v1"
        if docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
            docker exec -i "$container_name" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" < "$sql_file"
        else
            log_error "psql not found locally and PostgreSQL container '$container_name' not running"
            return 1
        fi
    fi
    
    if [ $? -eq 0 ]; then
        log_success "$description completed successfully"
    else
        log_error "$description failed"
        return 1
    fi
}

# Check Claude Code availability
check_claude_code() {
    if [ "$CLAUDE_CODE_AVAILABLE" != "true" ]; then
        log_warn "Claude Code is disabled in configuration"
        return 1
    fi
    
    log_info "Checking Claude Code availability..."
    
    # Use the existing Claude Code management script
    local claude_script="${VROOLI_ROOT:-/home/matthalloran8/Vrooli}/scripts/resources/agents/claude-code/manage.sh"
    
    if [ -f "$claude_script" ]; then
        if bash "$claude_script" --action health-check --check-type basic --format json >/dev/null 2>&1; then
            log_success "Claude Code is available and configured"
            return 0
        else
            log_warn "Claude Code is installed but not properly configured"
            log_info "You may need to run: claude auth login"
            return 1
        fi
    else
        log_error "Claude Code management script not found"
        return 1
    fi
}

# Main deployment function
main() {
    log_info "Starting Scenario Generator V1 deployment..."
    log_info "Scenario directory: $SCENARIO_DIR"
    
    # Check prerequisites
    log_info "Checking service availability..."
    
    if ! check_service "$POSTGRES_HOST" "$POSTGRES_PORT" "PostgreSQL"; then
        log_error "PostgreSQL is required but not available"
        exit 1
    fi
    
    # Check Claude Code
    if ! check_claude_code; then
        log_warn "Claude Code is not available - scenario generation will fail"
        log_info "To fix this, run: bash ${VROOLI_ROOT:-/home/matthalloran8/Vrooli}/scripts/resources/agents/claude-code/manage.sh --action install"
    fi
    
    # Initialize database schema
    log_info "Setting up database schema..."
    if execute_sql "$SCENARIO_DIR/initialization/storage/schema.sql" "Database schema creation"; then
        log_success "Database schema initialized"
    else
        log_error "Failed to initialize database schema"
        exit 1
    fi
    
    # Load seed data
    log_info "Loading seed data..."
    if execute_sql "$SCENARIO_DIR/initialization/storage/seed.sql" "Seed data loading"; then
        log_success "Seed data loaded"
    else
        log_warn "Failed to load seed data (this may be okay if data already exists)"
    fi
    
    log_success "✅ Scenario Generator V1 deployment completed!"
    echo ""
    log_info "🎯 Access Points:"
    log_info "   Dashboard: http://$WINDMILL_HOST:$WINDMILL_PORT/apps/scenario-generator"
    log_info "   n8n Workflows: http://$N8N_HOST:$N8N_PORT"
    log_info "   Database: postgresql://$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB"
    echo ""
    log_info "📚 Next Steps:"
    log_info "   1. Access the dashboard to create your first scenario"
    log_info "   2. Verify Claude Code authentication is working"
    log_info "   3. Check workflow execution in n8n interface"
    log_info "   4. Monitor scenario generation progress"
    echo ""
    log_info "🔧 Troubleshooting:"
    log_info "   - Run './test.sh' to validate the deployment"
    log_info "   - Check n8n logs if workflows fail to execute"
    log_info "   - Verify Claude Code setup with: bash ${VROOLI_ROOT:-/home/matthalloran8/Vrooli}/scripts/resources/agents/claude-code/manage.sh --action status"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy"|"start"|"install")
        main
        ;;
    "check"|"health")
        log_info "Performing health check only..."
        check_service "$POSTGRES_HOST" "$POSTGRES_PORT" "PostgreSQL"
        check_service "$N8N_HOST" "$N8N_PORT" "n8n"
        check_service "$WINDMILL_HOST" "$WINDMILL_PORT" "Windmill"
        check_service "$MINIO_HOST" "$MINIO_PORT" "MinIO"
        check_service "$REDIS_HOST" "$REDIS_PORT" "Redis"
        check_claude_code
        ;;
    "help"|"-h"|"--help")
        echo "Scenario Generator V1 - Deployment Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  deploy, start, install    Deploy the complete scenario generator (default)"
        echo "  check, health            Check service availability only"
        echo "  help                     Show this help message"
        ;;
    *)
        log_error "Unknown command: $1"
        log_info "Run '$0 help' for usage information"
        exit 1
        ;;
esac
