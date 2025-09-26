#!/bin/bash
# Airbyte DBT Transformation Support Library

set -euo pipefail

# Resource metadata
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${RESOURCE_DIR}/data"
TRANSFORMATIONS_DIR="${DATA_DIR}/transformations"

# DBT configuration
DBT_PROJECT_NAME="airbyte_transformations"
DBT_VERSION="1.7.0"

# Logging functions
log_info() {
    echo "[INFO] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

# Initialize DBT project structure
init_dbt_project() {
    local project_name="${1:-$DBT_PROJECT_NAME}"
    
    log_info "Initializing DBT project: $project_name"
    
    # Create transformations directory
    mkdir -p "${TRANSFORMATIONS_DIR}"
    
    # Create DBT project structure
    cat > "${TRANSFORMATIONS_DIR}/dbt_project.yml" <<EOF
name: '$project_name'
version: '1.0.0'
config-version: 2

profile: 'airbyte_transformations'

model-paths: ["models"]
analysis-paths: ["analyses"]
test-paths: ["tests"]
seed-paths: ["data"]
macro-paths: ["macros"]
snapshot-paths: ["snapshots"]

clean-targets:
  - "target"
  - "dbt_packages"

models:
  $project_name:
    materialized: table
    staging:
      materialized: view
EOF

    # Create profiles.yml for database connection
    mkdir -p "${TRANSFORMATIONS_DIR}"
    cat > "${TRANSFORMATIONS_DIR}/profiles.yml" <<EOF
airbyte_transformations:
  target: dev
  outputs:
    dev:
      type: postgres
      host: localhost
      port: 5432
      user: postgres
      pass: postgres
      dbname: airbyte_destination
      schema: public
      threads: 4
      keepalives_idle: 0
EOF

    # Create model directories
    mkdir -p "${TRANSFORMATIONS_DIR}/models/staging"
    mkdir -p "${TRANSFORMATIONS_DIR}/models/marts"
    mkdir -p "${TRANSFORMATIONS_DIR}/macros"
    mkdir -p "${TRANSFORMATIONS_DIR}/tests"
    mkdir -p "${TRANSFORMATIONS_DIR}/data"
    
    log_info "DBT project initialized at ${TRANSFORMATIONS_DIR}"
}

# Install DBT
install_dbt() {
    log_info "Installing DBT..."
    
    # Check if DBT is already installed
    if command -v dbt &> /dev/null; then
        log_info "DBT is already installed"
        dbt --version
        return 0
    fi
    
    # Install DBT using pip in a virtual environment
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 is required for DBT installation"
        return 1
    fi
    
    # Create virtual environment for DBT
    python3 -m venv "${DATA_DIR}/dbt_venv"
    
    # Activate and install DBT
    source "${DATA_DIR}/dbt_venv/bin/activate"
    pip install --upgrade pip
    pip install "dbt-core==${DBT_VERSION}" "dbt-postgres==${DBT_VERSION}"
    
    log_info "DBT installed successfully"
    deactivate
}

# Create a DBT model
create_dbt_model() {
    local model_name="$1"
    local model_sql="$2"
    local model_type="${3:-staging}"  # staging or marts
    
    if [[ -z "$model_name" || -z "$model_sql" ]]; then
        log_error "Usage: create_dbt_model <name> <sql> [type]"
        return 1
    fi
    
    # Ensure project exists
    if [[ ! -f "${TRANSFORMATIONS_DIR}/dbt_project.yml" ]]; then
        init_dbt_project
    fi
    
    # Create model file
    local model_path="${TRANSFORMATIONS_DIR}/models/${model_type}/${model_name}.sql"
    echo "$model_sql" > "$model_path"
    
    # Create schema file for documentation
    local schema_path="${TRANSFORMATIONS_DIR}/models/${model_type}/schema.yml"
    if [[ ! -f "$schema_path" ]]; then
        cat > "$schema_path" <<EOF
version: 2

models:
EOF
    fi
    
    # Add model to schema
    cat >> "$schema_path" <<EOF
  - name: $model_name
    description: "Transformation model for $model_name"
    columns: []
EOF
    
    log_info "DBT model created: $model_path"
    echo "$model_path"
}

# Run DBT transformations
run_dbt_transformations() {
    local models="${1:-}"  # Optional specific models to run
    local full_refresh="${2:-false}"
    
    log_info "Running DBT transformations..."
    
    if [[ ! -f "${TRANSFORMATIONS_DIR}/dbt_project.yml" ]]; then
        log_error "DBT project not found. Initialize first with 'transform init'"
        return 1
    fi
    
    # Activate DBT virtual environment if it exists
    if [[ -f "${DATA_DIR}/dbt_venv/bin/activate" ]]; then
        source "${DATA_DIR}/dbt_venv/bin/activate"
    fi
    
    cd "${TRANSFORMATIONS_DIR}"
    
    # Run DBT deps to install dependencies
    dbt deps
    
    # Run transformations
    if [[ -n "$models" ]]; then
        if [[ "$full_refresh" == "true" ]]; then
            dbt run --models "$models" --full-refresh
        else
            dbt run --models "$models"
        fi
    else
        if [[ "$full_refresh" == "true" ]]; then
            dbt run --full-refresh
        else
            dbt run
        fi
    fi
    
    local exit_code=$?
    
    # Deactivate virtual environment
    if [[ -f "${DATA_DIR}/dbt_venv/bin/activate" ]]; then
        deactivate
    fi
    
    return $exit_code
}

# Test DBT models
test_dbt_models() {
    local models="${1:-}"
    
    log_info "Testing DBT models..."
    
    if [[ ! -f "${TRANSFORMATIONS_DIR}/dbt_project.yml" ]]; then
        log_error "DBT project not found"
        return 1
    fi
    
    # Activate DBT virtual environment
    if [[ -f "${DATA_DIR}/dbt_venv/bin/activate" ]]; then
        source "${DATA_DIR}/dbt_venv/bin/activate"
    fi
    
    cd "${TRANSFORMATIONS_DIR}"
    
    # Run tests
    if [[ -n "$models" ]]; then
        dbt test --models "$models"
    else
        dbt test
    fi
    
    local exit_code=$?
    
    # Deactivate virtual environment
    if [[ -f "${DATA_DIR}/dbt_venv/bin/activate" ]]; then
        deactivate
    fi
    
    return $exit_code
}

# Generate DBT documentation
generate_dbt_docs() {
    log_info "Generating DBT documentation..."
    
    if [[ ! -f "${TRANSFORMATIONS_DIR}/dbt_project.yml" ]]; then
        log_error "DBT project not found"
        return 1
    fi
    
    # Activate DBT virtual environment
    if [[ -f "${DATA_DIR}/dbt_venv/bin/activate" ]]; then
        source "${DATA_DIR}/dbt_venv/bin/activate"
    fi
    
    cd "${TRANSFORMATIONS_DIR}"
    
    # Generate docs
    dbt docs generate
    
    # Optionally serve docs
    if [[ "${1:-}" == "--serve" ]]; then
        log_info "Serving DBT documentation at http://localhost:8080"
        dbt docs serve --port 8080
    fi
    
    # Deactivate virtual environment
    if [[ -f "${DATA_DIR}/dbt_venv/bin/activate" ]]; then
        deactivate
    fi
}

# Create Airbyte-specific transformation models
create_airbyte_models() {
    log_info "Creating Airbyte-specific DBT models..."
    
    # Create staging model for raw Airbyte data
    local staging_sql='{{ config(materialized="view") }}

-- Staging model for Airbyte raw data
WITH source_data AS (
    SELECT 
        _airbyte_ab_id,
        _airbyte_emitted_at,
        _airbyte_data
    FROM {{ source("airbyte_raw", var("source_table")) }}
)

SELECT 
    _airbyte_ab_id AS record_id,
    _airbyte_emitted_at AS synced_at,
    _airbyte_data AS raw_data
FROM source_data
WHERE _airbyte_data IS NOT NULL'
    
    create_dbt_model "stg_airbyte_raw" "$staging_sql" "staging"
    
    # Create marts model for normalized data
    local marts_sql='{{ config(materialized="table") }}

-- Normalized data from Airbyte staging
WITH staged_data AS (
    SELECT * FROM {{ ref("stg_airbyte_raw") }}
),

parsed_data AS (
    SELECT 
        record_id,
        synced_at,
        raw_data::jsonb AS json_data
    FROM staged_data
)

SELECT 
    record_id,
    synced_at,
    json_data,
    NOW() AS transformed_at
FROM parsed_data'
    
    create_dbt_model "airbyte_normalized" "$marts_sql" "marts"
    
    log_info "Airbyte DBT models created"
}

# List available transformations
list_transformations() {
    if [[ ! -d "${TRANSFORMATIONS_DIR}/models" ]]; then
        log_info "No transformations found"
        return 0
    fi
    
    echo "Available DBT Models:"
    echo ""
    
    # List staging models
    if [[ -d "${TRANSFORMATIONS_DIR}/models/staging" ]]; then
        echo "Staging Models:"
        for model in "${TRANSFORMATIONS_DIR}/models/staging"/*.sql; do
            if [[ -f "$model" ]]; then
                basename "$model" .sql | sed 's/^/  - /'
            fi
        done
    fi
    
    # List marts models
    if [[ -d "${TRANSFORMATIONS_DIR}/models/marts" ]]; then
        echo ""
        echo "Marts Models:"
        for model in "${TRANSFORMATIONS_DIR}/models/marts"/*.sql; do
            if [[ -f "$model" ]]; then
                basename "$model" .sql | sed 's/^/  - /'
            fi
        done
    fi
}

# Apply a transformation to a connection
apply_transformation() {
    local connection_id="$1"
    local transformation_name="$2"
    
    if [[ -z "$connection_id" || -z "$transformation_name" ]]; then
        log_error "Usage: apply_transformation <connection_id> <transformation_name>"
        return 1
    fi
    
    log_info "Applying transformation '$transformation_name' to connection '$connection_id'"
    
    # Check if transformation exists
    local model_path="${TRANSFORMATIONS_DIR}/models"
    local found=false
    
    for dir in staging marts; do
        if [[ -f "${model_path}/${dir}/${transformation_name}.sql" ]]; then
            found=true
            break
        fi
    done
    
    if [[ "$found" != "true" ]]; then
        log_error "Transformation not found: $transformation_name"
        return 1
    fi
    
    # Run the specific transformation
    run_dbt_transformations "$transformation_name"
    
    if [[ $? -eq 0 ]]; then
        log_info "Transformation applied successfully"
        
        # Store transformation metadata
        local metadata_file="${DATA_DIR}/transformation_metadata.json"
        if [[ ! -f "$metadata_file" ]]; then
            echo "{}" > "$metadata_file"
        fi
        
        # Update metadata
        local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        jq --arg conn "$connection_id" \
           --arg trans "$transformation_name" \
           --arg ts "$timestamp" \
           '.[$conn] = {transformation: $trans, applied_at: $ts}' \
           "$metadata_file" > "${metadata_file}.tmp"
        mv "${metadata_file}.tmp" "$metadata_file"
        
        echo "Transformation metadata saved"
    else
        log_error "Transformation failed"
        return 1
    fi
}

# Get transformation status for a connection
get_transformation_status() {
    local connection_id="$1"
    
    if [[ -z "$connection_id" ]]; then
        log_error "Usage: get_transformation_status <connection_id>"
        return 1
    fi
    
    local metadata_file="${DATA_DIR}/transformation_metadata.json"
    
    if [[ ! -f "$metadata_file" ]]; then
        echo "No transformations applied"
        return 0
    fi
    
    local status=$(jq -r --arg conn "$connection_id" '.[$conn] // "none"' "$metadata_file")
    
    if [[ "$status" == "none" ]]; then
        echo "No transformation applied to connection: $connection_id"
    else
        echo "$status" | jq '.'
    fi
}

# Clean DBT artifacts
clean_dbt_artifacts() {
    log_info "Cleaning DBT artifacts..."
    
    if [[ -d "${TRANSFORMATIONS_DIR}/target" ]]; then
        rm -rf "${TRANSFORMATIONS_DIR}/target"
    fi
    
    if [[ -d "${TRANSFORMATIONS_DIR}/dbt_packages" ]]; then
        rm -rf "${TRANSFORMATIONS_DIR}/dbt_packages"
    fi
    
    if [[ -d "${TRANSFORMATIONS_DIR}/logs" ]]; then
        rm -rf "${TRANSFORMATIONS_DIR}/logs"
    fi
    
    log_info "DBT artifacts cleaned"
}

# Main command handler
cmd_transform() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        init)
            init_dbt_project "$@"
            ;;
        install)
            install_dbt
            ;;
        create)
            create_dbt_model "$@"
            ;;
        run)
            run_dbt_transformations "$@"
            ;;
        test)
            test_dbt_models "$@"
            ;;
        docs)
            generate_dbt_docs "$@"
            ;;
        list)
            list_transformations
            ;;
        apply)
            apply_transformation "$@"
            ;;
        status)
            get_transformation_status "$@"
            ;;
        clean)
            clean_dbt_artifacts
            ;;
        airbyte-models)
            create_airbyte_models
            ;;
        *)
            echo "Error: Unknown transform subcommand: $subcommand" >&2
            echo "Usage: transform [init|install|create|run|test|docs|list|apply|status|clean|airbyte-models]" >&2
            exit 1
            ;;
    esac
}