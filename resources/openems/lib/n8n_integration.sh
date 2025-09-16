#!/usr/bin/env bash

# n8n Integration for OpenEMS
# Provides workflow automation capabilities for energy management

set -euo pipefail

# Source required libraries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh" 2>/dev/null || true
source "${SCRIPT_DIR}/../config/defaults.sh" 2>/dev/null || true

# Define logging functions if not available
if ! command -v log_info &> /dev/null; then
    log_info() { echo "[INFO] $*"; }
    log_success() { echo "[SUCCESS] ✅ $*"; }
    log_warn() { echo "[WARN] ⚠️  $*"; }
    log_error() { echo "[ERROR] ❌ $*"; }
fi

# n8n configuration
N8N_PORT="${N8N_PORT:-5678}"
N8N_URL="http://localhost:${N8N_PORT}"
N8N_WEBHOOK_PATH="/webhook/openems"

# OpenEMS API configuration
OPENEMS_REST_PORT="${OPENEMS_REST_PORT:-8084}"
OPENEMS_JSONRPC_PORT="${OPENEMS_JSONRPC_PORT:-8085}"
OPENEMS_REST_URL="http://localhost:${OPENEMS_REST_PORT}"
OPENEMS_JSONRPC_URL="ws://localhost:${OPENEMS_JSONRPC_PORT}/jsonrpc"

# -------------------------------
# n8n Workflow Management
# -------------------------------

create_energy_automation_workflow() {
    local workflow_name="${1:-energy-automation}"
    
    log_info "Creating n8n workflow for energy automation: ${workflow_name}"
    
    # Check if n8n is running (non-fatal warning)
    if ! curl -sf "${N8N_URL}/health" &>/dev/null; then
        log_warn "n8n is not running. Please start it with: vrooli resource n8n manage start"
        # Continue anyway - we're just creating templates
    fi
    
    # Create workflow JSON
    cat > "/tmp/openems-workflow.json" <<'EOF'
{
    "name": "OpenEMS Energy Automation",
    "nodes": [
        {
            "parameters": {
                "path": "openems-trigger",
                "responseMode": "onReceived",
                "responseData": "allEntries",
                "options": {}
            },
            "name": "OpenEMS Trigger",
            "type": "n8n-nodes-base.webhook",
            "typeVersion": 1,
            "position": [250, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/_sum/EssSoc",
                "responseFormat": "json",
                "options": {}
            },
            "name": "Get Battery SOC",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [450, 300]
        },
        {
            "parameters": {
                "conditions": {
                    "number": [
                        {
                            "value1": "={{$json[\"value\"]}}",
                            "operation": "smaller",
                            "value2": 20
                        }
                    ]
                }
            },
            "name": "Low Battery Check",
            "type": "n8n-nodes-base.if",
            "typeVersion": 1,
            "position": [650, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/ctrlGridOptimizedCharge0/SetChargeMaxPower",
                "method": "POST",
                "sendBody": true,
                "bodyParameters": {
                    "parameters": [
                        {
                            "name": "value",
                            "value": "5000"
                        }
                    ]
                },
                "options": {}
            },
            "name": "Start Charging",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [850, 200]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/_sum/GridActivePower",
                "responseFormat": "json",
                "options": {}
            },
            "name": "Monitor Grid Power",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [850, 400]
        }
    ],
    "connections": {
        "OpenEMS Trigger": {
            "main": [[{"node": "Get Battery SOC", "type": "main", "index": 0}]]
        },
        "Get Battery SOC": {
            "main": [[{"node": "Low Battery Check", "type": "main", "index": 0}]]
        },
        "Low Battery Check": {
            "main": [
                [{"node": "Start Charging", "type": "main", "index": 0}],
                [{"node": "Monitor Grid Power", "type": "main", "index": 0}]
            ]
        }
    },
    "active": true,
    "settings": {
        "saveDataErrorExecution": "all",
        "saveDataSuccessExecution": "all",
        "saveManualExecutions": true,
        "callerPolicy": "workflowsFromSameOwner"
    }
}
EOF
    
    log_info "Energy automation workflow template created"
    echo "Upload this workflow to n8n at ${N8N_URL}"
    echo "Workflow file: /tmp/openems-workflow.json"
    return 0
}

create_solar_optimization_workflow() {
    local workflow_name="${1:-solar-optimization}"
    
    log_info "Creating solar optimization workflow"
    
    cat > "/tmp/solar-optimization-workflow.json" <<'EOF'
{
    "name": "Solar Optimization",
    "nodes": [
        {
            "parameters": {
                "rule": "0 */15 6-18 * * *"
            },
            "name": "Every 15min (6am-6pm)",
            "type": "n8n-nodes-base.cron",
            "typeVersion": 1,
            "position": [250, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/meter0/ActivePower",
                "responseFormat": "json",
                "options": {}
            },
            "name": "Get Solar Production",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [450, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/_sum/ConsumptionActivePower",
                "responseFormat": "json",
                "options": {}
            },
            "name": "Get Consumption",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [450, 450]
        },
        {
            "parameters": {
                "jsCode": "const solar = $node[\"Get Solar Production\"].json.value;\nconst consumption = $node[\"Get Consumption\"].json.value;\nconst excess = solar - consumption;\n\nreturn [{\n  json: {\n    solar: solar,\n    consumption: consumption,\n    excess: excess,\n    recommendation: excess > 1000 ? \"charge_battery\" : excess < -1000 ? \"discharge_battery\" : \"maintain\"\n  }\n}];"
            },
            "name": "Calculate Strategy",
            "type": "n8n-nodes-base.code",
            "typeVersion": 1,
            "position": [650, 375]
        },
        {
            "parameters": {
                "conditions": {
                    "string": [
                        {
                            "value1": "={{$json[\"recommendation\"]}}",
                            "operation": "equals",
                            "value2": "charge_battery"
                        }
                    ]
                }
            },
            "name": "Should Charge?",
            "type": "n8n-nodes-base.if",
            "typeVersion": 1,
            "position": [850, 375]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/ess0/SetActivePowerEquals",
                "method": "POST",
                "sendBody": true,
                "bodyParameters": {
                    "parameters": [
                        {
                            "name": "value",
                            "value": "={{$node[\"Calculate Strategy\"].json[\"excess\"]}}"
                        }
                    ]
                },
                "options": {}
            },
            "name": "Set Battery Charge",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [1050, 300]
        }
    ],
    "connections": {
        "Every 15min (6am-6pm)": {
            "main": [[
                {"node": "Get Solar Production", "type": "main", "index": 0},
                {"node": "Get Consumption", "type": "main", "index": 0}
            ]]
        },
        "Get Solar Production": {
            "main": [[{"node": "Calculate Strategy", "type": "main", "index": 0}]]
        },
        "Get Consumption": {
            "main": [[{"node": "Calculate Strategy", "type": "main", "index": 0}]]
        },
        "Calculate Strategy": {
            "main": [[{"node": "Should Charge?", "type": "main", "index": 0}]]
        },
        "Should Charge?": {
            "main": [
                [{"node": "Set Battery Charge", "type": "main", "index": 0}],
                []
            ]
        }
    },
    "active": true
}
EOF
    
    log_info "Solar optimization workflow created at /tmp/solar-optimization-workflow.json"
    return 0
}

create_peak_shaving_workflow() {
    log_info "Creating peak shaving workflow for demand management"
    
    cat > "/tmp/peak-shaving-workflow.json" <<'EOF'
{
    "name": "Peak Shaving Control",
    "nodes": [
        {
            "parameters": {
                "path": "grid-peak-alert",
                "responseMode": "onReceived",
                "options": {}
            },
            "name": "Peak Alert Webhook",
            "type": "n8n-nodes-base.webhook",
            "typeVersion": 1,
            "position": [250, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/_sum/GridActivePower",
                "responseFormat": "json",
                "options": {}
            },
            "name": "Check Grid Power",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [450, 300]
        },
        {
            "parameters": {
                "conditions": {
                    "number": [
                        {
                            "value1": "={{$json[\"value\"]}}",
                            "operation": "larger",
                            "value2": 15000
                        }
                    ]
                }
            },
            "name": "Peak Threshold Check",
            "type": "n8n-nodes-base.if",
            "typeVersion": 1,
            "position": [650, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/ess0/SetActivePowerEquals",
                "method": "POST",
                "sendBody": true,
                "bodyParameters": {
                    "parameters": [
                        {
                            "name": "value",
                            "value": "-5000"
                        }
                    ]
                },
                "options": {}
            },
            "name": "Discharge Battery",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [850, 200]
        }
    ],
    "connections": {
        "Peak Alert Webhook": {
            "main": [[{"node": "Check Grid Power", "type": "main", "index": 0}]]
        },
        "Check Grid Power": {
            "main": [[{"node": "Peak Threshold Check", "type": "main", "index": 0}]]
        },
        "Peak Threshold Check": {
            "main": [
                [{"node": "Discharge Battery", "type": "main", "index": 0}],
                []
            ]
        }
    },
    "active": true
}
EOF
    
    log_info "Peak shaving workflow created at /tmp/peak-shaving-workflow.json"
    return 0
}

# -------------------------------
# SCADA/Modbus Integration
# -------------------------------

create_modbus_ingestion_workflow() {
    log_info "Creating Modbus data ingestion workflow for SCADA integration"
    
    cat > "/tmp/modbus-ingestion-workflow.json" <<'EOF'
{
    "name": "SCADA/Modbus Data Ingestion",
    "nodes": [
        {
            "parameters": {
                "rule": "0 */5 * * * *"
            },
            "name": "Poll Every 5min",
            "type": "n8n-nodes-base.cron",
            "typeVersion": 1,
            "position": [250, 300]
        },
        {
            "parameters": {
                "jsCode": "// Simulated Modbus read (replace with actual Modbus node)\nreturn [{\n  json: {\n    registers: {\n      voltage: Math.random() * 10 + 230,\n      current: Math.random() * 20,\n      power: Math.random() * 5000,\n      energy: Math.random() * 100,\n      frequency: 49.9 + Math.random() * 0.2,\n      powerFactor: 0.85 + Math.random() * 0.15\n    },\n    timestamp: new Date().toISOString()\n  }\n}];"
            },
            "name": "Read Modbus Registers",
            "type": "n8n-nodes-base.code",
            "typeVersion": 1,
            "position": [450, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/meter0/Voltage",
                "method": "POST",
                "sendBody": true,
                "bodyParameters": {
                    "parameters": [
                        {
                            "name": "value",
                            "value": "={{$json[\"registers\"][\"voltage\"]}}"
                        }
                    ]
                },
                "options": {}
            },
            "name": "Update Voltage",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [650, 200]
        },
        {
            "parameters": {
                "url": "http://localhost:8084/rest/channel/meter0/ActivePower",
                "method": "POST",
                "sendBody": true,
                "bodyParameters": {
                    "parameters": [
                        {
                            "name": "value",
                            "value": "={{$json[\"registers\"][\"power\"]}}"
                        }
                    ]
                },
                "options": {}
            },
            "name": "Update Power",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [650, 400]
        },
        {
            "parameters": {
                "jsCode": "const data = $node[\"Read Modbus Registers\"].json;\n\n// Format for QuestDB ingestion\nconst telemetry = {\n  measurement: \"scada_metrics\",\n  tags: {\n    device: \"modbus_meter_01\",\n    location: \"main_switchboard\"\n  },\n  fields: data.registers,\n  timestamp: data.timestamp\n};\n\nreturn [{json: telemetry}];"
            },
            "name": "Format for QuestDB",
            "type": "n8n-nodes-base.code",
            "typeVersion": 1,
            "position": [850, 300]
        },
        {
            "parameters": {
                "url": "http://localhost:9010/exec",
                "method": "POST",
                "sendQuery": true,
                "queryParameters": {
                    "parameters": [
                        {
                            "name": "query",
                            "value": "INSERT INTO scada_metrics VALUES('{{$json[\"timestamp\"]}}', '{{$json[\"tags\"][\"device\"]}}', {{$json[\"fields\"][\"voltage\"]}}, {{$json[\"fields\"][\"current\"]}}, {{$json[\"fields\"][\"power\"]}})"
                        }
                    ]
                },
                "options": {}
            },
            "name": "Store in QuestDB",
            "type": "n8n-nodes-base.httpRequest",
            "typeVersion": 3,
            "position": [1050, 300]
        }
    ],
    "connections": {
        "Poll Every 5min": {
            "main": [[{"node": "Read Modbus Registers", "type": "main", "index": 0}]]
        },
        "Read Modbus Registers": {
            "main": [[
                {"node": "Update Voltage", "type": "main", "index": 0},
                {"node": "Update Power", "type": "main", "index": 0},
                {"node": "Format for QuestDB", "type": "main", "index": 0}
            ]]
        },
        "Format for QuestDB": {
            "main": [[{"node": "Store in QuestDB", "type": "main", "index": 0}]]
        }
    },
    "active": true
}
EOF
    
    log_info "Modbus ingestion workflow created at /tmp/modbus-ingestion-workflow.json"
    return 0
}

# -------------------------------
# Workflow Deployment
# -------------------------------

deploy_workflow() {
    local workflow_file="${1}"
    
    if [[ ! -f "${workflow_file}" ]]; then
        log_error "Workflow file not found: ${workflow_file}"
        return 1
    fi
    
    log_info "To deploy this workflow to n8n:"
    echo "1. Open n8n at ${N8N_URL}"
    echo "2. Go to Workflows → Import from File"
    echo "3. Select: ${workflow_file}"
    echo "4. Activate the workflow"
    echo ""
    echo "Or use n8n CLI (if available):"
    echo "vrooli resource n8n content add openems-workflow ${workflow_file}"
    
    return 0
}

# -------------------------------
# Test n8n Integration
# -------------------------------

test_n8n_connectivity() {
    log_info "Testing n8n connectivity"
    
    if curl -sf "${N8N_URL}/health" &>/dev/null; then
        log_success "n8n is reachable at ${N8N_URL}"
        return 0
    else
        log_warn "n8n is not reachable. Please ensure it's running."
        echo "Start n8n with: vrooli resource n8n manage start"
        return 1
    fi
}

test_openems_apis() {
    log_info "Testing OpenEMS API connectivity from n8n perspective"
    
    # Test REST API
    if curl -sf "${OPENEMS_REST_URL}/rest/channel/_sum/EssSoc" &>/dev/null; then
        log_success "OpenEMS REST API is accessible"
    else
        log_warn "OpenEMS REST API not accessible at ${OPENEMS_REST_URL}"
    fi
    
    # Test health endpoint
    if curl -sf "http://localhost:${OPENEMS_REST_PORT}/health" &>/dev/null; then
        log_success "OpenEMS health endpoint is accessible"
    else
        log_warn "OpenEMS health endpoint not accessible"
    fi
    
    return 0
}

# -------------------------------
# Main execution
# -------------------------------

main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        create-workflows)
            create_energy_automation_workflow
            create_solar_optimization_workflow
            create_peak_shaving_workflow
            create_modbus_ingestion_workflow
            echo ""
            log_success "All workflow templates created in /tmp/"
            echo "Deploy them to n8n at ${N8N_URL}"
            ;;
        test)
            test_n8n_connectivity
            test_openems_apis
            ;;
        deploy)
            deploy_workflow "${1:-/tmp/openems-workflow.json}"
            ;;
        help|*)
            echo "OpenEMS n8n Integration"
            echo ""
            echo "Commands:"
            echo "  create-workflows  - Create all workflow templates"
            echo "  test             - Test n8n and OpenEMS connectivity"
            echo "  deploy <file>    - Instructions to deploy workflow"
            echo ""
            echo "Usage:"
            echo "  ${0} create-workflows"
            echo "  ${0} test"
            echo "  ${0} deploy /tmp/solar-optimization-workflow.json"
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi