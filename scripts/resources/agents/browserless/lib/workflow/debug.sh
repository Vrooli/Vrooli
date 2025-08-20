#!/usr/bin/env bash

#######################################
# Browserless Workflow Debug System
# Handles debug data collection and visualization
#######################################

# Get script directory
WORKFLOW_DEBUG_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

#######################################
# Initialize debug directory for workflow execution
# Arguments:
#   $1 - Workflow name
#   $2 - Execution ID
# Returns:
#   Debug directory path
#######################################
debug::init_execution() {
    local workflow_name="${1:?Workflow name required}"
    local execution_id="${2:-$(date +%s)}"
    
    local debug_dir="${BROWSERLESS_DATA_DIR}/workflows/${workflow_name}/executions/${execution_id}"
    
    # Create directory structure
    mkdir -p "${debug_dir}/steps"
    mkdir -p "${debug_dir}/outputs"
    mkdir -p "${debug_dir}/debug"
    
    # Initialize metadata
    cat > "${debug_dir}/metadata.json" <<EOF
{
    "workflow": "$workflow_name",
    "execution_id": "$execution_id",
    "started_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "status": "running"
}
EOF
    
    echo "$debug_dir"
}

#######################################
# Store step debug data
# Arguments:
#   $1 - Debug directory
#   $2 - Step name
#   $3 - Step data (JSON)
#######################################
debug::store_step_data() {
    local debug_dir="${1:?Debug directory required}"
    local step_name="${2:?Step name required}"
    local step_data="${3:?Step data required}"
    
    local step_file="${debug_dir}/steps/${step_name}.json"
    echo "$step_data" > "$step_file"
}

#######################################
# Generate debug summary report
# Arguments:
#   $1 - Debug directory
# Returns:
#   HTML report path
#######################################
debug::generate_report() {
    local debug_dir="${1:?Debug directory required}"
    local report_file="${debug_dir}/summary.html"
    
    # Get execution metadata
    local metadata
    metadata=$(<"${debug_dir}/metadata.json")
    
    local workflow_name=$(echo "$metadata" | jq -r '.workflow')
    local execution_id=$(echo "$metadata" | jq -r '.execution_id')
    local started_at=$(echo "$metadata" | jq -r '.started_at')
    
    # Generate HTML report
    cat > "$report_file" <<'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Workflow Execution Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            padding: 20px;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #4CAF50;
            padding-bottom: 10px;
        }
        .metadata {
            background: #f9f9f9;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .metadata dt {
            font-weight: bold;
            display: inline-block;
            width: 150px;
        }
        .metadata dd {
            display: inline;
            margin: 0;
        }
        .step {
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-bottom: 10px;
            overflow: hidden;
        }
        .step-header {
            background: #f0f0f0;
            padding: 10px;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .step.success .step-header {
            background: #e8f5e9;
        }
        .step.failed .step-header {
            background: #ffebee;
        }
        .step-content {
            padding: 15px;
            display: none;
        }
        .step.expanded .step-content {
            display: block;
        }
        .status-badge {
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: bold;
        }
        .status-badge.success {
            background: #4CAF50;
            color: white;
        }
        .status-badge.failed {
            background: #f44336;
            color: white;
        }
        .screenshot {
            max-width: 100%;
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-top: 10px;
        }
        .console-log {
            background: #1e1e1e;
            color: #d4d4d4;
            padding: 10px;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            font-size: 12px;
            overflow-x: auto;
        }
        .network-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        .network-table th,
        .network-table td {
            border: 1px solid #ddd;
            padding: 8px;
            text-align: left;
        }
        .network-table th {
            background: #f0f0f0;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Workflow Execution Report</h1>
        
        <div class="metadata">
            <dl>
                <dt>Workflow:</dt>
                <dd id="workflow-name"></dd><br>
                <dt>Execution ID:</dt>
                <dd id="execution-id"></dd><br>
                <dt>Started:</dt>
                <dd id="started-at"></dd><br>
                <dt>Status:</dt>
                <dd id="status"></dd>
            </dl>
        </div>
        
        <h2>Execution Steps</h2>
        <div id="steps-container"></div>
    </div>
    
    <script>
        // Load execution data
        const metadata = 
EOF
    
    # Insert metadata
    echo "$metadata" >> "$report_file"
    
    cat >> "$report_file" <<'EOF'
        ;
        
        // Populate metadata
        document.getElementById('workflow-name').textContent = metadata.workflow;
        document.getElementById('execution-id').textContent = metadata.execution_id;
        document.getElementById('started-at').textContent = metadata.started_at;
        document.getElementById('status').textContent = metadata.status;
        
        // Toggle step expansion
        function toggleStep(stepElement) {
            stepElement.classList.toggle('expanded');
        }
        
        // Load steps data (would be populated from actual execution)
        // This is a placeholder for the actual implementation
    </script>
</body>
</html>
EOF
    
    echo "$report_file"
}

#######################################
# View workflow execution results
# Arguments:
#   $1 - Workflow name
#   $2 - Execution ID (optional, defaults to latest)
#######################################
debug::view_results() {
    local workflow_name="${1:?Workflow name required}"
    local execution_id="${2:-latest}"
    
    local workflow_dir="${BROWSERLESS_DATA_DIR}/workflows/${workflow_name}"
    
    if [[ ! -d "$workflow_dir" ]]; then
        log::error "Workflow not found: $workflow_name"
        return 1
    fi
    
    local execution_dir
    if [[ "$execution_id" == "latest" ]]; then
        # Find latest execution
        execution_dir=$(find "${workflow_dir}/executions" -maxdepth 1 -type d | sort -r | head -n 2 | tail -n 1)
        if [[ -z "$execution_dir" ]]; then
            log::error "No executions found for workflow: $workflow_name"
            return 1
        fi
    else
        execution_dir="${workflow_dir}/executions/${execution_id}"
        if [[ ! -d "$execution_dir" ]]; then
            log::error "Execution not found: $execution_id"
            return 1
        fi
    fi
    
    log::header "📊 Workflow Execution Results"
    echo
    
    # Show metadata
    local metadata
    metadata=$(<"${execution_dir}/metadata.json")
    
    echo "Workflow: $(echo "$metadata" | jq -r '.workflow')"
    echo "Execution: $(echo "$metadata" | jq -r '.execution_id')"
    echo "Started: $(echo "$metadata" | jq -r '.started_at')"
    echo "Status: $(echo "$metadata" | jq -r '.status')"
    echo
    
    # Show steps
    echo "Steps:"
    for step_file in "${execution_dir}"/steps/*.json; do
        if [[ -f "$step_file" ]]; then
            local step_name=$(basename "$step_file" .json)
            local step_data
            step_data=$(<"$step_file")
            
            local success=$(echo "$step_data" | jq -r '.success // false')
            local duration=$(echo "$step_data" | jq -r '.duration // 0')
            
            if [[ "$success" == "true" ]]; then
                echo "  ✅ $step_name (${duration}ms)"
            else
                local error=$(echo "$step_data" | jq -r '.error // "Unknown error"')
                echo "  ❌ $step_name - $error"
            fi
        fi
    done
    
    # Show outputs
    echo
    echo "Outputs:"
    for output_file in "${execution_dir}"/outputs/*; do
        if [[ -f "$output_file" ]]; then
            echo "  📄 $(basename "$output_file")"
        fi
    done
    
    # Show debug files
    echo
    echo "Debug Files:"
    for debug_file in "${execution_dir}"/debug/*; do
        if [[ -f "$debug_file" ]]; then
            echo "  🔍 $(basename "$debug_file")"
        fi
    done
    
    echo
    echo "Full results: $execution_dir"
}

# Export functions
export -f debug::init_execution
export -f debug::store_step_data
export -f debug::generate_report
export -f debug::view_results