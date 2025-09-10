#!/usr/bin/env bash
# Apache Airflow Resource - DAG Management

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Add a new DAG file
add() {
    local dag_file=""
    local dag_name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file) dag_file="$2"; shift ;;
            --name) dag_name="$2"; shift ;;
            *) ;;
        esac
        shift
    done
    
    if [[ -z "$dag_file" ]]; then
        echo "Error: --file parameter required"
        return 1
    fi
    
    if [[ ! -f "$dag_file" ]]; then
        echo "Error: DAG file not found: $dag_file"
        return 1
    fi
    
    if [[ -z "$dag_name" ]]; then
        dag_name=$(basename "$dag_file")
    fi
    
    echo "Adding DAG: $dag_name"
    cp "$dag_file" "${AIRFLOW_DAG_DIR}/$dag_name"
    
    # Trigger DAG refresh
    if docker exec airflow-scheduler airflow dags list &>/dev/null; then
        echo "DAG added successfully: $dag_name"
    else
        echo "Warning: Could not verify DAG was parsed"
    fi
    
    return 0
}

# List available DAGs
list() {
    echo "Available DAGs:"
    echo "==============="
    
    if docker exec airflow-scheduler airflow dags list 2>/dev/null; then
        return 0
    else
        # Fallback to file listing
        if [[ -d "${AIRFLOW_DAG_DIR}" ]]; then
            for dag_file in "${AIRFLOW_DAG_DIR}"/*.py; do
                [[ -f "$dag_file" ]] && echo "  - $(basename "$dag_file")"
            done
        else
            echo "No DAGs directory found"
        fi
    fi
    
    return 0
}

# Get DAG definition
get() {
    local dag_id=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dag) dag_id="$2"; shift ;;
            *) dag_id="$1" ;;
        esac
        shift
    done
    
    if [[ -z "$dag_id" ]]; then
        echo "Error: DAG ID required"
        return 1
    fi
    
    # Try to get DAG details from Airflow
    if docker exec airflow-scheduler airflow dags show "$dag_id" 2>/dev/null; then
        return 0
    fi
    
    # Fallback to showing file content
    local dag_file="${AIRFLOW_DAG_DIR}/${dag_id}.py"
    if [[ -f "$dag_file" ]]; then
        echo "DAG Source: $dag_file"
        echo "===================="
        cat "$dag_file"
    else
        echo "DAG not found: $dag_id"
        return 1
    fi
    
    return 0
}

# Remove a DAG
remove() {
    local dag_id=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dag) dag_id="$2"; shift ;;
            *) dag_id="$1" ;;
        esac
        shift
    done
    
    if [[ -z "$dag_id" ]]; then
        echo "Error: DAG ID required"
        return 1
    fi
    
    echo "Removing DAG: $dag_id"
    
    # Pause the DAG first
    docker exec airflow-scheduler airflow dags pause "$dag_id" 2>/dev/null || true
    
    # Remove DAG file
    local dag_file="${AIRFLOW_DAG_DIR}/${dag_id}.py"
    if [[ -f "$dag_file" ]]; then
        rm "$dag_file"
        echo "DAG removed: $dag_id"
    else
        echo "DAG file not found: $dag_id"
        return 1
    fi
    
    return 0
}

# Execute/trigger a DAG
execute() {
    local dag_id=""
    local execution_date=""
    local conf=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dag) dag_id="$2"; shift ;;
            --date) execution_date="$2"; shift ;;
            --conf) conf="$2"; shift ;;
            *) dag_id="$1" ;;
        esac
        shift
    done
    
    if [[ -z "$dag_id" ]]; then
        echo "Error: DAG ID required"
        return 1
    fi
    
    if [[ -z "$execution_date" ]]; then
        execution_date=$(date -I)
    fi
    
    echo "Triggering DAG: $dag_id"
    echo "Execution date: $execution_date"
    
    # Trigger the DAG
    local trigger_cmd="airflow dags trigger $dag_id"
    if [[ -n "$conf" ]]; then
        trigger_cmd="$trigger_cmd --conf '$conf'"
    fi
    
    if docker exec airflow-scheduler sh -c "$trigger_cmd" 2>/dev/null; then
        echo "DAG triggered successfully"
        echo ""
        echo "View execution at: http://localhost:${AIRFLOW_WEBSERVER_PORT}/dags/${dag_id}/grid"
    else
        echo "Failed to trigger DAG"
        return 1
    fi
    
    return 0
}

# Export functions
export -f add
export -f list
export -f get
export -f remove
export -f execute

# Handle direct script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    command="${1:-list}"
    shift || true
    
    case "$command" in
        add|list|get|remove|execute)
            "$command" "$@"
            ;;
        *)
            echo "Usage: $0 {add|list|get|remove|execute}"
            exit 1
            ;;
    esac
fi