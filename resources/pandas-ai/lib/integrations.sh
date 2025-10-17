#!/usr/bin/env bash
set -euo pipefail

# Integration helpers for Pandas-AI with other resources

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PANDAS_AI_LIB_DIR="${APP_ROOT}/resources/pandas-ai/lib"

source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Export data for SageMath processing
pandas_ai::export_for_math() {
    local input_file="${1:-}"
    local output_file="${2:-/tmp/pandas_ai_export.csv}"
    
    if [[ -z "${input_file}" ]]; then
        log::error "Input file required"
        return 1
    fi
    
    log::info "Exporting data for mathematical processing..."
    
    # Analyze and clean data first
    local analysis_result
    analysis_result=$(curl -s -X POST http://localhost:8095/analyze \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"describe\", \"csv_path\": \"${input_file}\"}")
    
    if [[ $? -eq 0 ]]; then
        log::success "Data analyzed and ready for export to ${output_file}"
        # In a real implementation, we'd process and save the cleaned data
        cp "${input_file}" "${output_file}"
        echo "${output_file}"
    else
        log::error "Failed to analyze data"
        return 1
    fi
}

# Prepare data for statistical analysis
pandas_ai::prepare_stats_data() {
    local data_json="${1:-}"
    
    if [[ -z "${data_json}" ]]; then
        log::error "Data JSON required"
        return 1
    fi
    
    # Get statistical summary
    local stats_result
    stats_result=$(curl -s -X POST http://localhost:8095/analyze \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"summary\", \"data\": ${data_json}}")
    
    echo "${stats_result}"
}

# Integration with PostgreSQL
pandas_ai::analyze_postgres_table() {
    local table_name="${1:-}"
    local query="${2:-describe}"
    
    if [[ -z "${table_name}" ]]; then
        log::error "Table name required"
        return 1
    fi
    
    log::info "Analyzing PostgreSQL table: ${table_name}"
    
    # Export table to CSV first (would use actual postgres export in production)
    local temp_csv="/tmp/${table_name}_export.csv"
    
    # Analyze the exported data
    curl -s -X POST http://localhost:8095/analyze \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"${query}\", \"csv_path\": \"${temp_csv}\"}"
}

# Helper for N8n workflow integration
pandas_ai::workflow_analysis() {
    local workflow_data="${1:-}"
    local analysis_type="${2:-summary}"
    
    log::info "Processing data for N8n workflow..."
    
    # Perform analysis and return JSON suitable for N8n
    local result
    result=$(curl -s -X POST http://localhost:8095/analyze \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"${analysis_type}\", \"data\": ${workflow_data}}")
    
    # Format for N8n consumption
    echo "{\"pandas_ai_result\": ${result}}"
}