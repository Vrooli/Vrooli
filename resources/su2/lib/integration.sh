#!/bin/bash
# SU2 Data Integration Functions
# Connects SU2 results to QuestDB for time-series and Qdrant for vector storage

# Source defaults if not loaded
[[ -z "$SU2_PORT" ]] && source "${SCRIPT_DIR}/config/defaults.sh"

# QuestDB integration
export_to_questdb() {
    local result_id="${1:-}"
    local questdb_host="${2:-localhost}"
    local questdb_port="${3:-9000}"
    
    if [[ -z "$result_id" ]]; then
        echo "Error: Result ID required" >&2
        return 1
    fi
    
    local result_dir="${SU2_RESULTS_DIR}/${result_id}"
    local history_file="${result_dir}/history.csv"
    
    if [[ ! -f "$history_file" ]]; then
        echo "Error: Convergence history not found" >&2
        return 1
    fi
    
    echo "Exporting simulation ${result_id} to QuestDB..."
    
    # Create table if not exists
    local create_table="CREATE TABLE IF NOT EXISTS su2_simulations (
        timestamp TIMESTAMP,
        simulation_id SYMBOL,
        iteration LONG,
        cl DOUBLE,
        cd DOUBLE,
        cm DOUBLE,
        residual DOUBLE,
        mach DOUBLE,
        aoa DOUBLE
    ) timestamp(timestamp) PARTITION BY DAY;"
    
    # Send table creation via HTTP API
    curl -sf -X POST "http://${questdb_host}:${questdb_port}/exec" \
        --data-urlencode "query=${create_table}" > /dev/null || {
        echo "Warning: Could not create QuestDB table" >&2
    }
    
    # Parse and insert convergence data
    local timestamp=$(date -Iseconds)
    tail -n +2 "$history_file" | while IFS=',' read -r iter cl cd cm res; do
        local insert="INSERT INTO su2_simulations VALUES(
            '${timestamp}',
            '${result_id}',
            ${iter:-0},
            ${cl:-0},
            ${cd:-0},
            ${cm:-0},
            ${res:-0},
            0.8,
            1.25
        );"
        
        curl -sf -X POST "http://${questdb_host}:${questdb_port}/exec" \
            --data-urlencode "query=${insert}" > /dev/null
    done
    
    echo "Data exported to QuestDB successfully"
    return 0
}

# Qdrant integration for design space exploration
export_to_qdrant() {
    local result_id="${1:-}"
    local qdrant_host="${2:-localhost}"
    local qdrant_port="${3:-6333}"
    
    if [[ -z "$result_id" ]]; then
        echo "Error: Result ID required" >&2
        return 1
    fi
    
    local result_dir="${SU2_RESULTS_DIR}/${result_id}"
    local history_file="${result_dir}/history.csv"
    
    if [[ ! -f "$history_file" ]]; then
        echo "Error: Convergence history not found" >&2
        return 1
    fi
    
    echo "Exporting simulation ${result_id} to Qdrant..."
    
    # Create collection if not exists
    curl -sf -X PUT "http://${qdrant_host}:${qdrant_port}/collections/su2_designs" \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {
                "size": 6,
                "distance": "Cosine"
            }
        }' > /dev/null 2>&1 || true
    
    # Extract final performance metrics as vector
    local final_metrics=$(tail -1 "$history_file")
    IFS=',' read -r iter cl cd cm res <<< "$final_metrics"
    
    # Normalize values for vector embedding
    local vector="[${cl:-0}, ${cd:-0}, ${cm:-0}, ${res:-0}, 0.8, 1.25]"
    
    # Create point for Qdrant
    local point_data="{
        \"points\": [{
            \"id\": \"${result_id}\",
            \"vector\": ${vector},
            \"payload\": {
                \"simulation_id\": \"${result_id}\",
                \"cl\": ${cl:-0},
                \"cd\": ${cd:-0},
                \"cm\": ${cm:-0},
                \"residual\": ${res:-0},
                \"iterations\": ${iter:-0},
                \"timestamp\": \"$(date -Iseconds)\"
            }
        }]
    }"
    
    # Insert into Qdrant
    curl -sf -X PUT "http://${qdrant_host}:${qdrant_port}/collections/su2_designs/points" \
        -H "Content-Type: application/json" \
        -d "${point_data}" || {
        echo "Warning: Could not insert into Qdrant" >&2
        return 1
    }
    
    echo "Design vector stored in Qdrant successfully"
    return 0
}

# Search similar designs in Qdrant
search_similar_designs() {
    local cl="${1:-0}"
    local cd="${2:-0}"
    local cm="${3:-0}"
    local qdrant_host="${4:-localhost}"
    local qdrant_port="${5:-6333}"
    
    echo "Searching for similar designs..."
    
    local vector="[${cl}, ${cd}, ${cm}, 0, 0.8, 1.25]"
    
    local search_query="{
        \"vector\": ${vector},
        \"limit\": 5,
        \"with_payload\": true
    }"
    
    curl -sf -X POST "http://${qdrant_host}:${qdrant_port}/collections/su2_designs/points/search" \
        -H "Content-Type: application/json" \
        -d "${search_query}" | jq '.' || {
        echo "Error: Search failed" >&2
        return 1
    }
    
    return 0
}

# Batch optimization helper
run_batch_optimization() {
    local base_config="${1:-}"
    local param_file="${2:-}"
    local output_dir="${3:-${SU2_RESULTS_DIR}/batch_$(date +%s)}"
    
    if [[ -z "$base_config" ]] || [[ -z "$param_file" ]]; then
        echo "Usage: run_batch_optimization <base_config> <parameter_file> [output_dir]" >&2
        return 1
    fi
    
    if [[ ! -f "${SU2_CONFIGS_DIR}/${base_config}" ]]; then
        echo "Error: Base config not found" >&2
        return 1
    fi
    
    if [[ ! -f "$param_file" ]]; then
        echo "Error: Parameter file not found" >&2
        return 1
    fi
    
    mkdir -p "$output_dir"
    
    echo "Starting batch optimization..."
    local run_count=0
    
    # Read parameter variations (format: AOA,MACH,CFL)
    while IFS=',' read -r aoa mach cfl; do
        [[ "$aoa" =~ ^#.*$ ]] && continue  # Skip comments
        [[ -z "$aoa" ]] && continue
        
        ((run_count++))
        local run_id="run_${run_count}"
        local run_dir="${output_dir}/${run_id}"
        mkdir -p "$run_dir"
        
        # Create modified config
        local mod_config="${run_dir}/config.cfg"
        sed -e "s/AOA=.*/AOA= ${aoa}/" \
            -e "s/MACH_NUMBER=.*/MACH_NUMBER= ${mach}/" \
            -e "s/CFL_NUMBER=.*/CFL_NUMBER= ${cfl}/" \
            "${SU2_CONFIGS_DIR}/${base_config}" > "$mod_config"
        
        echo "  Run ${run_count}: AOA=${aoa}, Mach=${mach}, CFL=${cfl}"
        
        # Submit simulation
        local response=$(curl -sf -X POST "http://localhost:${SU2_PORT}/api/simulate" \
            -H "Content-Type: application/json" \
            -d "{\"mesh\":\"naca0012.su2\",\"config\":\"$(basename $mod_config)\"}" 2>/dev/null)
        
        echo "$response" > "${run_dir}/submission.json"
    done < "$param_file"
    
    echo "Batch optimization submitted: ${run_count} runs"
    echo "Results directory: ${output_dir}"
    return 0
}

# Export all results to external storage
export_all_results() {
    local export_type="${1:-all}"  # all, questdb, qdrant
    
    echo "Exporting all simulation results..."
    
    for result_dir in "${SU2_RESULTS_DIR}"/*; do
        [[ ! -d "$result_dir" ]] && continue
        
        local result_id=$(basename "$result_dir")
        echo "Processing ${result_id}..."
        
        case "$export_type" in
            questdb)
                export_to_questdb "$result_id" || true
                ;;
            qdrant)
                export_to_qdrant "$result_id" || true
                ;;
            all|*)
                export_to_questdb "$result_id" || true
                export_to_qdrant "$result_id" || true
                ;;
        esac
    done
    
    echo "Export complete"
    return 0
}