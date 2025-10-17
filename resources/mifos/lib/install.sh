#!/usr/bin/env bash
################################################################################
# Mifos Install Library
# 
# Installation and uninstallation functions for Mifos X
################################################################################

set -euo pipefail

# ==============================================================================
# INSTALL
# ==============================================================================
mifos::install::execute() {
    log::header "Installing Mifos X Platform"
    
    # Check prerequisites
    mifos::install::check_prerequisites || return 1
    
    # Pull Docker images
    mifos::install::pull_images || return 1
    
    # Setup database
    mifos::install::setup_database || return 1
    
    # Create directories
    mifos::install::create_directories || return 1
    
    # Generate configuration
    mifos::docker::generate_compose || return 1
    
    # Seed demo data if enabled
    if [[ "${MIFOS_SEED_DEMO_DATA}" == "true" ]]; then
        mifos::install::prepare_demo_data || log::warning "Demo data preparation failed (non-critical)"
    fi
    
    log::success "Mifos X installation complete"
    return 0
}

# ==============================================================================
# UNINSTALL
# ==============================================================================
mifos::install::uninstall() {
    log::header "Uninstalling Mifos X Platform"
    
    # Stop containers
    mifos::docker::stop || log::warning "Failed to stop containers"
    
    # Remove containers
    log::info "Removing containers..."
    docker rm -f "${MIFOS_CONTAINER_PREFIX}-backend" "${MIFOS_CONTAINER_PREFIX}-webapp" 2>/dev/null || true
    
    # Remove network
    log::info "Removing network..."
    docker network rm "${MIFOS_NETWORK}" 2>/dev/null || true
    
    # Optionally remove data
    read -p "Remove all Mifos data? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log::info "Removing data directories..."
        rm -rf "${MIFOS_DATA_DIR}" "${MIFOS_LOG_DIR}"
        
        # Drop database using docker exec
        local pg_container="vrooli-postgres-main"
        if docker ps --format "{{.Names}}" | grep -q "${pg_container}"; then
            log::info "Dropping database..."
            docker exec "${pg_container}" psql -U postgres -c "DROP DATABASE IF EXISTS ${FINERACT_DB_NAME};" 2>/dev/null || true
            docker exec "${pg_container}" psql -U postgres -c "DROP USER IF EXISTS ${FINERACT_DB_USER};" 2>/dev/null || true
        fi
    fi
    
    log::success "Mifos X uninstallation complete"
    return 0
}

# ==============================================================================
# CHECK PREREQUISITES
# ==============================================================================
mifos::install::check_prerequisites() {
    log::info "Checking prerequisites..."
    
    local errors=0
    
    # Check Docker
    if ! command -v docker &>/dev/null; then
        log::error "Docker is not installed"
        ((errors++))
    fi
    
    # Check Docker Compose
    if ! (command -v docker-compose &>/dev/null || docker compose version &>/dev/null); then
        log::error "Docker Compose is not installed"
        ((errors++))
    fi
    
    # Check PostgreSQL resource
    if ! vrooli resource postgres status &>/dev/null; then
        log::warning "PostgreSQL resource is not running, will attempt to start"
    fi
    
    if [[ ${errors} -gt 0 ]]; then
        log::error "Prerequisites check failed"
        return 1
    fi
    
    log::success "Prerequisites check passed"
    return 0
}

# ==============================================================================
# PULL IMAGES
# ==============================================================================
mifos::install::pull_images() {
    log::info "Pulling Docker images..."
    
    local images=(
        "apache/fineract:${FINERACT_VERSION}"
        "${MIFOS_WEBAPP_IMAGE}:${MIFOS_WEBAPP_VERSION}"
    )
    
    for image in "${images[@]}"; do
        log::info "Pulling ${image}..."
        if docker pull "${image}"; then
            log::success "✓ ${image}"
        else
            log::error "Failed to pull ${image}"
            return 1
        fi
    done
    
    log::success "All images pulled successfully"
    return 0
}

# ==============================================================================
# SETUP DATABASE
# ==============================================================================
mifos::install::setup_database() {
    log::info "Setting up database..."
    
    # Ensure PostgreSQL is running
    if ! vrooli resource postgres status &>/dev/null; then
        log::info "Starting PostgreSQL..."
        vrooli resource postgres manage start || return 1
        sleep 5
    fi
    
    # Wait for PostgreSQL to be ready
    sleep 3
    
    # Create database and user
    log::info "Creating Mifos database and user..."
    
    # Use docker exec to run commands in PostgreSQL container
    local pg_container="vrooli-postgres-main"
    
    # Create database (using vrooli user as it has CREATEDB privilege)
    docker exec "${pg_container}" psql -U vrooli -c "CREATE DATABASE ${FINERACT_DB_NAME} ENCODING 'UTF8';" 2>/dev/null || log::info "Database may already exist"
    
    # Create user (using vrooli user as it has CREATEROLE privilege)
    docker exec "${pg_container}" psql -U vrooli -c "CREATE USER ${FINERACT_DB_USER} WITH PASSWORD '${FINERACT_DB_PASSWORD}';" 2>/dev/null || log::info "User may already exist"
    
    # Grant privileges
    docker exec "${pg_container}" psql -U vrooli -c "GRANT ALL PRIVILEGES ON DATABASE ${FINERACT_DB_NAME} TO ${FINERACT_DB_USER};" 2>/dev/null || true
    
    # Allow user to create schemas
    docker exec "${pg_container}" psql -U vrooli -c "ALTER DATABASE ${FINERACT_DB_NAME} OWNER TO ${FINERACT_DB_USER};" 2>/dev/null || true
    
    log::success "Database setup complete"
    return 0
}

# ==============================================================================
# CREATE DIRECTORIES
# ==============================================================================
mifos::install::create_directories() {
    log::info "Creating data directories..."
    
    mkdir -p "${MIFOS_DATA_DIR}"/{uploads,reports,exports}
    mkdir -p "${MIFOS_LOG_DIR}"
    mkdir -p "${MIFOS_CLI_DIR}/docker"
    
    log::success "Directories created"
    return 0
}

# ==============================================================================
# PREPARE DEMO DATA
# ==============================================================================
mifos::install::prepare_demo_data() {
    log::info "Preparing demo data configuration..."
    
    # Create a marker file for demo data seeding
    # This will be processed after Mifos starts
    cat > "${MIFOS_DATA_DIR}/demo_config.json" << EOF
{
  "seed_demo_data": true,
  "demo_clients_count": ${MIFOS_DEMO_CLIENTS_COUNT},
  "demo_loans_count": ${MIFOS_DEMO_LOANS_COUNT},
  "currencies": ["${MIFOS_CURRENCIES//,/\",\"}"],
  "base_currency": "${MIFOS_BASE_CURRENCY}"
}
EOF
    
    log::success "Demo data configuration prepared"
    return 0
}