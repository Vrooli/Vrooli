#!/usr/bin/env bash
# Port Registry - Central source of truth for all service port assignments
# This file prevents port conflicts between Vrooli services and local resources

# ============================================================================
# LOCAL RESOURCE SERVICES - CAREFULLY CHOSEN TO AVOID CONFLICTS
# ============================================================================
declare -g -A RESOURCE_PORTS=(
    # Core Services
    ["vrooli-api"]="8092"      # Vrooli Unified API (was 8090, moved due to Whisper conflict)
    ["vrooli-orchestrator"]="9500"  # Vrooli Orchestrator for managing scenarios/apps
    
    # AI Services (11xxx range)
    ["ollama"]="11434"         # LLM inference engine
    ["litellm"]="11435"        # LiteLLM unified LLM proxy server
    ["whisper"]="8090"         # Speech-to-text service (keeping original port)
    ["unstructured-io"]="11450" # Document processing and extraction
    ["nsfw-detector"]="11451"  # NSFW content detection and moderation AI
    ["qwencoder"]="11452"      # QwenCoder code generation LLM
    ["deepstack"]="11453"      # DeepStack computer vision AI (object detection, face recognition)
    ["segment-anything"]="11454"  # Segment Anything Model (SAM2/HQ-SAM) foundation segmentation service
    ["ultralytics-yolo"]="11455"  # Ultralytics YOLOv8 vision intelligence suite (detection, segmentation, tracking)
    ["gazebo"]="11456"         # Gazebo robotics simulation platform (3D physics, sensors, robot modeling)
    ["godot"]="11457"          # Godot Engine game development platform (GDScript, 2D/3D engine)
    ["mathlib"]="11458"        # Mathlib4 formal mathematical theorem proving with Lean 4
    ["papermc"]="11461"        # PaperMC Minecraft server health endpoint
    ["pybullet"]="11460"       # PyBullet physics simulation engine for robotics and ML
    ["ros2"]="11501"           # ROS2 Robot Operating System middleware for distributed robotics
    
    # Automation Services (56xx range)
    ["n8n"]="5678"             # Workflow automation
    ["comfyui"]="8188"         # AI-powered image generation workflows
    ["node-red"]="1880"        # Flow-based automation (Node-RED default port)
    ["windmill"]="5681"        # Developer-centric workflow automation
    ["temporal"]="7233"        # Temporal workflow orchestration (frontend)
    ["temporal-grpc"]="7234"   # Temporal gRPC service
    
    # Storage Services (9xxx range + vector databases)
    ["minio"]="9000"           # Object storage (S3 compatible)
    ["vault"]="8200"           # HashiCorp Vault secret management
    ["step-ca"]="9010"         # Private certificate authority with ACME support
    ["qdrant"]="6333"          # Vector database for AI embeddings
    ["questdb"]="9009"         # Time-series database for analytics
    ["postgres"]="5433"        # PostgreSQL instances (5433-5499 range for client instances)
    ["postgis"]="5434"         # PostGIS spatial database extension for PostgreSQL
    ["redis"]="6380"           # Redis in-memory data store (6380 to avoid conflict with internal Redis on 6379)
    ["neo4j"]="7474"           # Neo4j graph database (HTTP)
    ["neo4j-bolt"]="7687"      # Neo4j Bolt protocol
    # Note: sqlite is serverless and doesn't use a port
    
    # Agent Services (41xx range - safely above Vrooli range)
    ["browserless"]="4110"     # Browserless.io Chrome service
    ["huginn"]="4111"          # Workflow automation platform
    ["agent-s2"]="4113"        # Agent S2 autonomous computer interaction
    ["obs-studio"]="4455"      # OBS Studio WebSocket API
    # Note: claude-code is a CLI tool and doesn't use a port
    
    # Search Services (82xx range - moved to avoid conflict with debug range)
    ["searxng"]="8280"         # SearXNG metasearch engine
    
    # Security Services (81xx range)
    ["owasp-zap"]="8180"       # OWASP ZAP security scanner API
    ["wireguard"]="51820"      # WireGuard VPN (UDP port)
    
    # Infrastructure Services (80xx range)
    ["restic"]="8085"          # Encrypted backup and recovery service
    ["pihole"]="8087"          # Pi-hole network-wide ad blocking and DNS management
    
    # Execution Services (23xx range)
    ["judge0"]="2358"          # Code execution sandbox (official Judge0 port)
    ["llamaindex"]="8091"      # RAG and document processing
    ["autogen-studio"]="8081"  # Multi-agent conversation framework
    ["pandas-ai"]="8095"       # AI-powered data analysis and manipulation
    ["splink"]="8096"          # Probabilistic record linkage and deduplication at scale
    ["sagemath"]="8888"        # Open-source mathematics software system (Jupyter interface)
    ["sagemath-api"]="8889"    # SageMath REST API endpoint
    ["haystack"]="8075"        # End-to-end framework for question answering and search
    ["crewai"]="8084"          # Multi-agent AI framework for collaborative AI systems
    ["keycloak"]="8070"        # Enterprise identity and access management
    ["erpnext"]="8020"         # Complete open-source ERP suite
    ["mifos"]="8030"           # Mifos X digital finance platform (Apache Fineract)
    ["blender"]="8093"         # 3D creation suite with Python API
    ["vocr"]="9420"            # Vision OCR - Advanced screen recognition and AI-powered image analysis
    ["matrix-synapse"]="8008"  # Matrix Synapse federated communication server
    ["audiocraft"]="7862"      # Meta's comprehensive audio generation suite (MusicGen + AudioGen + EnCodec)
    ["cncjs"]="8194"           # Web-based CNC controller for Grbl, Marlin, Smoothieware, TinyG
    ["esphome"]="6587"         # ESPHome IoT firmware framework for ESP32/ESP8266 microcontrollers
    ["freecad"]="8195"         # Parametric 3D CAD modeler with Python API for engineering design
    ["elmer-fem"]="8192"       # Elmer FEM multiphysics finite element solver for coupled simulations
    ["meep"]="8193"            # MEEP electromagnetic FDTD simulation engine with Python bindings
    ["octoprint"]="8197"       # Web-based 3D printer management platform with REST API
    ["ggwave"]="8196"          # Air-gapped data transmission via FSK-modulated audio signals
    ["lnbits"]="5001"          # Lightning Network Bitcoin micropayments and wallet system
    ["strapi"]="1337"          # Strapi v5 headless CMS for structured content management
    
    # Payment Processing Services (230xx range)
    ["btcpay"]="23000"         # BTCPay Server self-hosted cryptocurrency payment processor
    
    # Simulation Services (95xx range)
    ["simpy"]="9510"           # SimPy discrete-event simulation framework (existing)
    ["gridlabd"]="9511"        # GridLAB-D power distribution system simulator
    ["mesa"]="9512"            # Mesa agent-based modeling framework for complex adaptive systems
    ["openrocket"]="9513"      # OpenRocket model rocket design and flight simulation
    ["su2"]="9514"             # SU2 aerospace CFD and optimization platform
    
    # Collaboration Services (80xx range)
    ["nextcloud"]="8086"       # Self-hosted file sync, share, and collaboration platform
    ["wikijs"]="3010"          # Wiki.js documentation platform with Git backend
    
    # Transportation Services (80xx range)
    ["opentripplanner"]="8080" # OpenTripPlanner multimodal routing engine
    ["traccar"]="8082"         # Traccar GPS tracking and fleet management server
    ["geonode"]="8100"         # GeoNode geospatial content management system (Django portal)
    ["geonode-geoserver"]="8101" # GeoNode GeoServer for OGC map services
    ["open-data-cube"]="8850"  # Open Data Cube Earth observation analytics platform
    ["open-data-cube-db"]="5450" # Open Data Cube PostGIS database
    ["open-data-cube-ows"]="8851" # Open Data Cube OGC Web Services (WMS/WCS)
    
    # Data Integration Services (80xx range)
    ["airbyte"]="8002"         # Airbyte ELT platform webapp (moved from 8000)
    ["airbyte-server"]="8003"  # Airbyte API server (moved from 8001)
    
    # Messaging & Streaming Services (290xx range)
    ["kafka"]="29092"          # Apache Kafka broker port
    ["kafka-controller"]="29093" # Kafka KRaft controller port
    ["kafka-external"]="29094" # Kafka external access port
    ["kafka-jmx"]="29099"      # Kafka JMX monitoring port
    
    # Agricultural Management Services (80xx range)
    ["farmos"]="8004"          # farmOS agricultural management platform
    ["airbyte-temporal"]="8006" # Airbyte workflow orchestration
    
    # IoT Services (80xx range)
    ["eclipse-ditto"]="8089"   # Eclipse Ditto Digital Twin Platform for IoT
    
    # Energy Management Services (80xx range)
    ["openems"]="8294"         # OpenEMS HTTP API and Web UI
    ["openems-jsonrpc"]="8295" # OpenEMS JSON-RPC WebSocket
    ["openems-backend"]="8296" # OpenEMS Backend API
    ["openems-modbus"]="502"   # OpenEMS Modbus TCP (standard port)
    
    # Healthcare Services (80xx range)
    ["openmrs"]="8005"         # OpenMRS web interface
    ["openmrs-api"]="8006"     # OpenMRS REST API
    ["openmrs-fhir"]="8007"    # OpenMRS FHIR API
    ["openemr"]="8010"         # OpenEMR web interface
    ["openemr-api"]="8011"     # OpenEMR REST API
    ["openemr-fhir"]="8012"    # OpenEMR FHIR R4 API
    
    # Interactive Computing Services (80xx range)
    ["jupyterhub"]="8000"      # JupyterHub multi-user notebook server
    ["jupyterhub-api"]="8001"  # JupyterHub API endpoint
    ["jupyterhub-proxy"]="8081" # JupyterHub proxy API
    
    # Monitoring Services (90xx range)
    ["prometheus"]="9090"       # Prometheus metrics database
    ["grafana"]="3030"         # Grafana visualization (non-standard port to avoid conflicts)
    ["alertmanager"]="9093"    # Prometheus Alertmanager
    ["node-exporter"]="9100"   # Node metrics exporter
    ["openmct"]="8099"         # NASA Open MCT mission control telemetry visualization
    
    # Analytics Services (80xx range)  
    ["apache-superset"]="8088"        # Apache Superset analytics platform
    ["apache-superset-postgres"]="5436" # Superset metadata database
    ["apache-superset-redis"]="6381"    # Superset cache and message broker
    
    # Future services can use:
    # - 11xxx for AI services
    # - 56xx for automation (avoiding 5678)
    # - 90xx for storage
    # - 41xx for agents (above Vrooli range)
    # - 81xx for search services
    # - 23xx for execution services
)

# ============================================================================
# RESERVED PORT RANGES
# ============================================================================
declare -g -A RESERVED_RANGES=(
    ["vrooli_core"]="3000-4100"       # Reserved for Vrooli core services
    ["databases"]="5432-5499"         # Database services
    ["cache"]="6379-6399"             # Cache services
    ["debug"]="9200-9299"             # Debug ports
    ["system"]="1-1023"               # System ports (require root)
)

# Export RESOURCE_PORTS for use in subshells and child processes
export RESOURCE_PORTS

# ============================================================================
# PostgreSQL Instance Configuration
# ============================================================================
# PostgreSQL instances (5433-5499 range for client instances)  
[[ -z "${POSTGRES_DEFAULT_PORT:-}" ]] && readonly POSTGRES_DEFAULT_PORT=5433
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_START:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_START=5433
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_END:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_END=5499

# ============================================================================
# Helper Functions
# ============================================================================

#######################################
# Get port for a resource service
# Arguments:
#   $1 - resource name
# Returns:
#   Port number or empty string
#######################################
ports::get_resource_port() {
    local resource="$1"
    echo "${RESOURCE_PORTS[$resource]:-}"
}

#######################################
# Validate port assignment
# Arguments:
#   $1 - port number
#   $2 - service name (for error messages)
# Returns:
#   0 if valid, 1 if invalid
#######################################
ports::validate_assignment() {
    local port="$1"
    local service="$2"
    
    # Check if port is a valid number
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 || "$port" -gt 65535 ]]; then
        echo "ERROR: Invalid port number '$port' for $service"
        return 1
    fi
    
    # Check if it's a system port (requires root)
    if [[ "$port" -lt 1024 ]]; then
        echo "WARNING: Port $port for $service requires root privileges"
    fi
    
    return 0
}

#######################################
# Get all ports in use (both Vrooli and resources)
# Returns:
#   Array of all assigned ports
#######################################
ports::get_all_assigned() {
    local -a all_ports=()
    
    # Add resource ports
    for port in "${RESOURCE_PORTS[@]}"; do
        all_ports+=("$port")
    done
    
    printf '%s\n' "${all_ports[@]}" | sort -n | uniq
}

#######################################
# Find next available port in a range
# Arguments:
#   $1 - start of range
#   $2 - end of range
# Returns:
#   Next available port or empty string
#######################################
ports::find_available_in_range() {
    local start="$1"
    local end="$2"
    local -a assigned=($(ports::get_all_assigned))
    
    for ((port=start; port<=end; port++)); do
        local found=0
        for assigned_port in "${assigned[@]}"; do
            if [[ "$port" == "$assigned_port" ]]; then
                found=1
                break
            fi
        done
        
        if [[ "$found" -eq 0 ]]; then
            echo "$port"
            return 0
        fi
    done
    
    return 1
}

#######################################
# Display port registry information
#######################################
ports::show_registry() {
    echo "=== Vrooli Port Registry ==="

    echo
    echo "LOCAL RESOURCES:"
    for resource in "${!RESOURCE_PORTS[@]}"; do
        printf "  %-15s : %s\n" "$resource" "${RESOURCE_PORTS[$resource]}"
    done | sort
    
    echo
    echo "RESERVED RANGES:"
    for range_name in "${!RESERVED_RANGES[@]}"; do
        printf "  %-15s : %s\n" "$range_name" "${RESERVED_RANGES[$range_name]}"
    done | sort
}

#######################################
# Export resource ports as JSON
# Used by TypeScript modules for dynamic loading
#######################################
ports::export_json() {
    local output='{"resource_ports":{'
    local first=true
    
    for resource in "${!RESOURCE_PORTS[@]}"; do
        if [[ "$first" == true ]]; then
            first=false
        else
            output+=","
        fi
        # Ensure port is a valid number
        local port="${RESOURCE_PORTS[$resource]}"
        if [[ "$port" =~ ^[0-9]+$ ]]; then
            output+="\"$resource\":$port"
        fi
    done
    
    output+="}}"
    echo "$output"
}

#######################################
# Show help information
#######################################
ports::show_help() {
    echo "Port Registry - Central source of truth for all service port assignments"
    echo
    echo "Usage: $0 [OPTION]"
    echo
    echo "Options:"
    echo "  --export-json    Export resource ports as JSON (for TypeScript integration)"
    echo "  --help          Show this help message"
    echo "  (no options)    Show port registry information"
}

#######################################
# SCENARIO INTEGRATION FUNCTIONS
# Enhanced functionality for scenario port conflict detection
#######################################

# Get all currently allocated scenario ports
ports::get_scenario_allocated_ports() {
    local scenario_state_dir="${HOME}/.vrooli/state/scenarios"
    local -a all_ports=()
    
    if [[ -d "$scenario_state_dir" ]]; then
        for state_file in "$scenario_state_dir"/*.json; do
            [[ -f "$state_file" ]] || continue
            
            if command -v jq >/dev/null 2>&1; then
                local scenario_ports
                scenario_ports=$(jq -r '.allocated_ports | to_entries[] | .value' "$state_file" 2>/dev/null || echo "")
                while IFS= read -r port; do
                    [[ -n "$port" && "$port" != "null" ]] && all_ports+=("$port")
                done <<< "$scenario_ports"
            fi
        done
    fi
    
    printf '%s\n' "${all_ports[@]}" | sort -n | uniq
}

# Get ALL allocated ports (resources + scenarios)
ports::get_all_allocated_ports() {
    local -a all_ports=()
    
    # Add resource ports
    for port in "${RESOURCE_PORTS[@]}"; do
        all_ports+=("$port")
    done
    
    # Add scenario ports
    local scenario_ports
    scenario_ports=$(ports::get_scenario_allocated_ports)
    while IFS= read -r port; do
        [[ -n "$port" ]] && all_ports+=("$port")
    done <<< "$scenario_ports"
    
    printf '%s\n' "${all_ports[@]}" | sort -n | uniq
}

# Validate scenario port configuration against registry
ports::validate_scenario_config() {
    local scenario_name="$1"
    local service_json_path="$2"
    
    if [[ ! -f "$service_json_path" ]]; then
        echo "{\"success\": false, \"error\": \"Service JSON not found: $service_json_path\"}"
        return 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        echo "{\"success\": false, \"error\": \"jq is required for JSON processing\"}"
        return 1
    fi
    
    # Parse ports configuration
    local ports_config
    ports_config=$(jq -r '.ports // {}' "$service_json_path" 2>/dev/null)
    
    if [[ "$ports_config" == "{}" || "$ports_config" == "null" ]]; then
        echo "{\"success\": true, \"message\": \"No ports configured for validation\"}"
        return 0
    fi
    
    local conflicts=()
    local -a allocated_ports=($(ports::get_all_allocated_ports))
    
    # Check each port configuration for conflicts
    while IFS= read -r port_entry; do
        [[ -n "$port_entry" ]] || continue
        
        local port_name port_config
        port_name=$(echo "$port_entry" | jq -r '.key')
        port_config=$(echo "$port_entry" | jq -r '.value')
        
        # Check fixed ports (official schema only)
        local fixed_port
        fixed_port=$(echo "$port_config" | jq -r '.port // empty' 2>/dev/null)
        
        if [[ -n "$fixed_port" && "$fixed_port" != "null" ]]; then
            # Check against resource ports
            for resource in "${!RESOURCE_PORTS[@]}"; do
                if [[ "${RESOURCE_PORTS[$resource]}" == "$fixed_port" ]]; then
                    conflicts+=("Fixed port $fixed_port for $port_name conflicts with resource '$resource'")
                fi
            done
            
            # Check against allocated scenario ports
            for allocated_port in "${allocated_ports[@]}"; do
                if [[ "$allocated_port" == "$fixed_port" ]]; then
                    conflicts+=("Fixed port $fixed_port for $port_name is already allocated")
                fi
            done
        fi
        
        # Check port ranges against reserved ranges
        local port_range
        port_range=$(echo "$port_config" | jq -r '.range // empty' 2>/dev/null)
        
        if [[ -n "$port_range" && "$port_range" != "null" ]] && [[ "$port_range" =~ ^([0-9]+)-([0-9]+)$ ]]; then
            local range_start="${BASH_REMATCH[1]}"
            local range_end="${BASH_REMATCH[2]}"
            
            # Check if range overlaps with system ports
            if [[ "$range_start" -lt 1024 ]]; then
                conflicts+=("Port range $port_range for $port_name includes system ports (< 1024)")
            fi
            
            # Check against reserved ranges
            for range_name in "${!RESERVED_RANGES[@]}"; do
                local reserved_range="${RESERVED_RANGES[$range_name]}"
                if [[ "$reserved_range" =~ ^([0-9]+)-([0-9]+)$ ]]; then
                    local reserved_start="${BASH_REMATCH[1]}"
                    local reserved_end="${BASH_REMATCH[2]}"
                    
                    # Check for overlap
                    if [[ "$range_start" -le "$reserved_end" && "$range_end" -ge "$reserved_start" ]]; then
                        conflicts+=("Port range $port_range for $port_name overlaps with reserved range '$range_name' ($reserved_range)")
                    fi
                fi
            done
        fi
        
    done < <(echo "$ports_config" | jq -c 'to_entries[]' 2>/dev/null)
    
    # Return results
    if [[ ${#conflicts[@]} -gt 0 ]]; then
        local conflicts_json="["
        local first=true
        for conflict in "${conflicts[@]}"; do
            [[ "$first" != "true" ]] && conflicts_json+=", "
            conflicts_json+="\"$conflict\""
            first=false
        done
        conflicts_json+="]"
        
        echo "{\"success\": false, \"conflicts\": $conflicts_json}"
        return 1
    else
        echo "{\"success\": true, \"message\": \"No port conflicts found for scenario: $scenario_name\"}"
        return 0
    fi
}

# Get detailed port allocation status
ports::get_allocation_status() {
    local -a resource_ports_array=()
    local -a scenario_ports_array=()
    
    # Collect resource ports
    for resource in "${!RESOURCE_PORTS[@]}"; do
        resource_ports_array+=("\"$resource\": ${RESOURCE_PORTS[$resource]}")
    done
    
    # Collect scenario ports
    local scenario_state_dir="${HOME}/.vrooli/state/scenarios"
    if [[ -d "$scenario_state_dir" ]]; then
        for state_file in "$scenario_state_dir"/*.json; do
            [[ -f "$state_file" ]] || continue
            
            local scenario_name
            scenario_name=$(basename "$state_file" .json)
            
            if command -v jq >/dev/null 2>&1; then
                local scenario_data
                scenario_data=$(jq -c '{allocated_ports, allocated_at}' "$state_file" 2>/dev/null || echo "{}")
                scenario_ports_array+=("\"$scenario_name\": $scenario_data")
            fi
        done
    fi
    
    # Build comprehensive status JSON
    echo "{"
    echo "  \"resource_ports\": {"
    local first=true
    for entry in "${resource_ports_array[@]}"; do
        [[ "$first" != "true" ]] && echo ", "
        echo -n "    $entry"
        first=false
    done
    echo ""
    echo "  },"
    echo "  \"scenario_ports\": {"
    
    first=true
    for entry in "${scenario_ports_array[@]}"; do
        [[ "$first" != "true" ]] && echo ", "
        echo -n "    $entry"
        first=false
    done
    echo ""
    echo "  },"
    echo "  \"reserved_ranges\": {"
    
    first=true
    for range_name in "${!RESERVED_RANGES[@]}"; do
        [[ "$first" != "true" ]] && echo ", "
        echo -n "    \"$range_name\": \"${RESERVED_RANGES[$range_name]}\""
        first=false
    done
    echo ""
    echo "  }"
    echo "}"
}

# Enhanced help with scenario functions
ports::show_help() {
    echo "Port Registry - Central source of truth for all service port assignments"
    echo
    echo "Usage: $0 [OPTION]"
    echo
    echo "Options:"
    echo "  --export-json               Export resource ports as JSON (for TypeScript integration)"
    echo "  --get-all-allocated         Get all allocated ports (resources + scenarios)"
    echo "  --get-scenario-ports        Get only scenario allocated ports"
    echo "  --validate-scenario <name> <service.json>  Validate scenario port config"
    echo "  --allocation-status         Get detailed allocation status"
    echo "  --help                      Show this help message"
    echo "  (no options)               Show port registry information"
    echo
    echo "Examples:"
    echo "  $0 --validate-scenario my-scenario /path/to/service.json"
    echo "  $0 --get-all-allocated | head -20"
    echo "  $0 --allocation-status | jq '.scenario_ports'"
}

# If script is run directly, handle CLI arguments
# Only execute when directly run as a script (not when sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]] && [[ -z "${SOURCED_PORT_REGISTRY:-}" ]]; then
    # Set flag to prevent re-execution when this script sources itself indirectly
    export SOURCED_PORT_REGISTRY=1
    case "${1:-}" in
        --export-json)
            ports::export_json
            ;;
        --get-all-allocated)
            ports::get_all_allocated_ports
            ;;
        --get-scenario-ports)
            ports::get_scenario_allocated_ports
            ;;
        --validate-scenario)
            ports::validate_scenario_config "$2" "$3"
            ;;
        --allocation-status)
            ports::get_allocation_status
            ;;
        --help)
            ports::show_help
            ;;
        *)
            ports::show_registry
            ;;
    esac
fi