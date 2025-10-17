#!/bin/bash
# Database Setup Script for Accessibility Compliance Hub
# Initializes PostgreSQL schema and default patterns

set -euo pipefail

# Colors
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly SCHEMA_FILE="${SCRIPT_DIR}/../initialization/storage/postgres/schema.sql"
readonly DB_NAME="${POSTGRES_DB:-accessibility_hub}"
readonly DB_USER="${POSTGRES_USER:-postgres}"
readonly DB_HOST="${POSTGRES_HOST:-localhost}"
readonly DB_PORT="${POSTGRES_PORT:-5432}"

# Helper functions
info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ Error: $1${NC}" >&2
    exit 1
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" >&2
}

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."

    if ! command -v psql &> /dev/null; then
        error "psql not found. Install PostgreSQL client tools."
    fi

    if [ ! -f "${SCHEMA_FILE}" ]; then
        error "Schema file not found: ${SCHEMA_FILE}"
    fi

    success "Prerequisites check passed"
}

# Check if PostgreSQL resource is running
check_postgres_resource() {
    info "Checking PostgreSQL resource status..."

    if ! vrooli resource status postgres --json &> /dev/null; then
        warning "PostgreSQL resource not running"
        info "Starting PostgreSQL resource..."
        vrooli resource start postgres || error "Failed to start PostgreSQL resource"
        sleep 3
    fi

    success "PostgreSQL resource is running"
}

# Test database connection
test_connection() {
    info "Testing database connection..."

    if ! PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c '\q' &> /dev/null; then
        error "Cannot connect to PostgreSQL at ${DB_HOST}:${DB_PORT}"
    fi

    success "Database connection successful"
}

# Initialize database
initialize_database() {
    info "Initializing database schema..."

    # Execute schema file
    if PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -f "${SCHEMA_FILE}" &> /dev/null; then
        success "Database schema initialized successfully"
    else
        error "Failed to initialize database schema. Check ${SCHEMA_FILE} for syntax errors."
    fi
}

# Verify installation
verify_installation() {
    info "Verifying database setup..."

    # Check if tables exist
    local table_count
    table_count=$(PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE';" 2>/dev/null | tr -d ' ')

    if [ "${table_count}" -ge 8 ]; then
        success "Database verification passed (${table_count} tables created)"
    else
        warning "Expected at least 8 tables, found ${table_count}"
    fi

    # Check if default patterns exist
    local pattern_count
    pattern_count=$(PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c "SELECT COUNT(*) FROM accessible_patterns;" 2>/dev/null | tr -d ' ')

    if [ "${pattern_count}" -ge 4 ]; then
        success "Default patterns loaded (${pattern_count} patterns)"
    else
        warning "Expected at least 4 default patterns, found ${pattern_count}"
    fi
}

# Show database info
show_database_info() {
    info "Database Information:"
    echo ""
    echo "  Database: ${DB_NAME}"
    echo "  Host:     ${DB_HOST}:${DB_PORT}"
    echo "  User:     ${DB_USER}"
    echo ""

    info "Tables created:"
    PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE' ORDER BY table_name;" 2>/dev/null || true

    echo ""
    success "Database setup complete!"
    echo ""
    info "Next steps:"
    echo "  1. Connect API to database"
    echo "  2. Implement /api/v1/accessibility/audit endpoint"
    echo "  3. Integrate axe-core scanning"
    echo ""
}

# Main execution
main() {
    echo ""
    info "Accessibility Compliance Hub - Database Setup"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    check_prerequisites
    check_postgres_resource
    test_connection
    initialize_database
    verify_installation
    show_database_info
}

# Run main function
main "$@"
