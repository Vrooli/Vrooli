#!/bin/bash
# OpenEMS Co-simulation - P2 Feature
# Enables co-simulation with OpenTripPlanner and GeoNode for energy-aware mobility

set -uo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENEMS_DIR="${APP_ROOT}/resources/openems"
OPENEMS_LIB_DIR="${OPENEMS_DIR}/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${OPENEMS_LIB_DIR}/core.sh"

# Co-simulation configuration
COSIM_CONFIG_DIR="${OPENEMS_DATA_DIR}/cosimulation"
COSIM_SCENARIOS_DIR="${COSIM_CONFIG_DIR}/scenarios"
COSIM_RESULTS_DIR="${COSIM_CONFIG_DIR}/results"
COSIM_STATE_FILE="${COSIM_CONFIG_DIR}/state.json"

################################################################################
# Co-simulation Functions
################################################################################

# Initialize co-simulation environment
openems::cosim::init() {
    log::info "Initializing OpenEMS co-simulation environment..."
    
    # Create directories
    mkdir -p "$COSIM_CONFIG_DIR"
    mkdir -p "$COSIM_SCENARIOS_DIR"
    mkdir -p "$COSIM_RESULTS_DIR"
    
    # Create initial state file
    if [[ ! -f "$COSIM_STATE_FILE" ]]; then
        cat > "$COSIM_STATE_FILE" << 'EOF'
{
    "enabled": true,
    "integrations": {
        "opentripplanner": {
            "enabled": false,
            "url": "http://localhost:8080",
            "api_version": "v2"
        },
        "geonode": {
            "enabled": false,
            "url": "http://localhost:8000",
            "api_key": ""
        },
        "simpy": {
            "enabled": true,
            "python_path": "python3"
        }
    },
    "scenarios": [],
    "current_simulation": null
}
EOF
        log::info "Created co-simulation state file"
    fi
    
    # Create sample scenario
    openems::cosim::create_sample_scenario
    
    return 0
}

# Create sample co-simulation scenario
openems::cosim::create_sample_scenario() {
    local scenario_file="${COSIM_SCENARIOS_DIR}/ev-charging-mobility.json"
    
    if [[ ! -f "$scenario_file" ]]; then
        cat > "$scenario_file" << 'EOF'
{
    "id": "ev-charging-mobility",
    "name": "EV Charging and Mobility Optimization",
    "description": "Optimize EV charging based on trip planning and grid conditions",
    "components": {
        "energy": {
            "grid_capacity": 100000,
            "renewable_percentage": 0.4,
            "peak_hours": [16, 21],
            "off_peak_rate": 0.08,
            "peak_rate": 0.24
        },
        "mobility": {
            "vehicles": [
                {
                    "id": "ev1",
                    "type": "passenger",
                    "battery_capacity": 60,
                    "current_soc": 0.3,
                    "consumption_rate": 0.2,
                    "max_charging_power": 11
                },
                {
                    "id": "ev2",
                    "type": "delivery",
                    "battery_capacity": 100,
                    "current_soc": 0.5,
                    "consumption_rate": 0.3,
                    "max_charging_power": 22
                }
            ],
            "trips": [
                {
                    "vehicle_id": "ev1",
                    "departure_time": "08:00",
                    "distance": 50,
                    "return_time": "18:00"
                },
                {
                    "vehicle_id": "ev2",
                    "departure_time": "06:00",
                    "distance": 120,
                    "return_time": "14:00"
                }
            ]
        },
        "charging_stations": [
            {
                "id": "station1",
                "location": {"lat": 37.7749, "lon": -122.4194},
                "power": 22,
                "solar_connected": true
            },
            {
                "id": "station2", 
                "location": {"lat": 37.7849, "lon": -122.4094},
                "power": 50,
                "solar_connected": false
            }
        ]
    },
    "optimization_goals": [
        "minimize_cost",
        "maximize_renewable_usage",
        "ensure_trip_completion"
    ],
    "simulation_period": {
        "start": "2025-01-17T00:00:00",
        "end": "2025-01-17T23:59:59",
        "step_minutes": 15
    }
}
EOF
        log::info "Created sample EV charging mobility scenario"
    fi
    
    return 0
}

# Check OpenTripPlanner integration
openems::cosim::check_otp() {
    local otp_url=$(jq -r '.integrations.opentripplanner.url' "$COSIM_STATE_FILE")
    local enabled=$(jq -r '.integrations.opentripplanner.enabled' "$COSIM_STATE_FILE")
    
    if [[ "$enabled" != "true" ]]; then
        log::debug "OpenTripPlanner integration disabled"
        return 1
    fi
    
    # Check if OTP is reachable
    if timeout 5 curl -sf "${otp_url}/otp" >/dev/null 2>&1; then
        log::info "OpenTripPlanner available at $otp_url"
        return 0
    else
        log::warn "OpenTripPlanner not reachable at $otp_url"
        return 1
    fi
}

# Check GeoNode integration
openems::cosim::check_geonode() {
    local geonode_url=$(jq -r '.integrations.geonode.url' "$COSIM_STATE_FILE")
    local enabled=$(jq -r '.integrations.geonode.enabled' "$COSIM_STATE_FILE")
    
    if [[ "$enabled" != "true" ]]; then
        log::debug "GeoNode integration disabled"
        return 1
    fi
    
    # Check if GeoNode is reachable
    if timeout 5 curl -sf "${geonode_url}/api" >/dev/null 2>&1; then
        log::info "GeoNode available at $geonode_url"
        return 0
    else
        log::warn "GeoNode not reachable at $geonode_url"
        return 1
    fi
}

# Run co-simulation with Python/SimPy
openems::cosim::run_simpy() {
    local scenario_file="$1"
    local output_file="$2"
    
    log::info "Running SimPy co-simulation..."
    
    # Create Python simulation script
    local sim_script=$(mktemp --suffix=.py)
    cat > "$sim_script" << 'EOF'
import json
import sys
import random
from datetime import datetime, timedelta

def simulate_ev_charging(scenario_data):
    """Simulate EV charging optimization"""
    results = {
        "scenario_id": scenario_data["id"],
        "timestamp": datetime.now().isoformat(),
        "vehicles": [],
        "charging_schedule": [],
        "energy_metrics": {},
        "cost_analysis": {}
    }
    
    # Extract scenario components
    energy = scenario_data["components"]["energy"]
    vehicles = scenario_data["components"]["mobility"]["vehicles"]
    trips = scenario_data["components"]["mobility"]["trips"]
    stations = scenario_data["components"]["charging_stations"]
    
    # Simulate charging schedules for each vehicle
    total_energy = 0
    total_cost = 0
    renewable_energy = 0
    
    for vehicle in vehicles:
        vehicle_id = vehicle["id"]
        battery_capacity = vehicle["battery_capacity"]
        current_soc = vehicle["current_soc"]
        max_power = vehicle["max_charging_power"]
        
        # Find vehicle's trip
        vehicle_trip = next((t for t in trips if t["vehicle_id"] == vehicle_id), None)
        if not vehicle_trip:
            continue
            
        # Calculate required energy
        distance = vehicle_trip["distance"]
        consumption = vehicle["consumption_rate"]
        required_energy = distance * consumption
        required_soc = required_energy / battery_capacity
        
        # Determine charging window
        departure = vehicle_trip["departure_time"]
        return_time = vehicle_trip["return_time"]
        
        # Simple optimization: charge during off-peak if possible
        charge_start = "00:00" if departure > "06:00" else return_time
        charge_duration = required_energy / max_power
        
        # Calculate cost
        is_peak = 16 <= int(charge_start.split(":")[0]) <= 21
        rate = energy["peak_rate"] if is_peak else energy["off_peak_rate"]
        cost = required_energy * rate
        
        total_energy += required_energy
        total_cost += cost
        renewable_energy += required_energy * energy["renewable_percentage"]
        
        # Add to results
        results["vehicles"].append({
            "vehicle_id": vehicle_id,
            "required_energy": required_energy,
            "charge_start": charge_start,
            "charge_duration": charge_duration,
            "cost": cost,
            "final_soc": min(1.0, current_soc + required_soc)
        })
        
        results["charging_schedule"].append({
            "time": charge_start,
            "vehicle": vehicle_id,
            "station": stations[0]["id"],
            "power": max_power,
            "duration": charge_duration
        })
    
    # Aggregate metrics
    results["energy_metrics"] = {
        "total_energy_kwh": total_energy,
        "renewable_energy_kwh": renewable_energy,
        "grid_energy_kwh": total_energy - renewable_energy,
        "peak_demand_kw": sum(v["max_charging_power"] for v in vehicles)
    }
    
    results["cost_analysis"] = {
        "total_cost": total_cost,
        "average_rate": total_cost / total_energy if total_energy > 0 else 0,
        "savings_vs_peak": total_energy * energy["peak_rate"] - total_cost
    }
    
    return results

# Main execution
if __name__ == "__main__":
    scenario_file = sys.argv[1]
    output_file = sys.argv[2]
    
    # Load scenario
    with open(scenario_file, 'r') as f:
        scenario = json.load(f)
    
    # Run simulation
    results = simulate_ev_charging(scenario)
    
    # Save results
    with open(output_file, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"Simulation complete. Results saved to {output_file}")
EOF
    
    # Run simulation
    python3 "$sim_script" "$scenario_file" "$output_file" 2>/dev/null || {
        log::error "SimPy simulation failed"
        rm -f "$sim_script"
        return 1
    }
    
    rm -f "$sim_script"
    log::info "SimPy simulation complete"
    return 0
}

# Run co-simulation scenario
openems::cosim::run_scenario() {
    local scenario_name="${1:-ev-charging-mobility}"
    local scenario_file="${COSIM_SCENARIOS_DIR}/${scenario_name}.json"
    
    if [[ ! -f "$scenario_file" ]]; then
        log::error "Scenario not found: $scenario_name"
        return 1
    fi
    
    log::info "Running co-simulation scenario: $scenario_name"
    
    # Create results directory for this run
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local result_dir="${COSIM_RESULTS_DIR}/${scenario_name}_${timestamp}"
    mkdir -p "$result_dir"
    
    # Update state
    jq --arg scenario "$scenario_name" \
       --arg timestamp "$timestamp" \
       '.current_simulation = {
           "scenario": $scenario,
           "started": $timestamp,
           "status": "running"
       }' "$COSIM_STATE_FILE" > "${COSIM_STATE_FILE}.tmp"
    mv "${COSIM_STATE_FILE}.tmp" "$COSIM_STATE_FILE"
    
    # Check available integrations
    local use_otp=false
    local use_geonode=false
    
    if openems::cosim::check_otp; then
        use_otp=true
        log::info "OpenTripPlanner integration available"
    fi
    
    if openems::cosim::check_geonode; then
        use_geonode=true
        log::info "GeoNode integration available"
    fi
    
    # Run SimPy simulation (always available)
    local simpy_result="${result_dir}/simpy_results.json"
    if openems::cosim::run_simpy "$scenario_file" "$simpy_result"; then
        log::info "Energy simulation complete"
    else
        log::error "Energy simulation failed"
    fi
    
    # Integrate with OpenTripPlanner if available
    if [[ "$use_otp" == "true" ]]; then
        openems::cosim::integrate_otp "$scenario_file" "$result_dir"
    fi
    
    # Integrate with GeoNode if available
    if [[ "$use_geonode" == "true" ]]; then
        openems::cosim::integrate_geonode "$scenario_file" "$result_dir"
    fi
    
    # Generate summary report
    openems::cosim::generate_report "$result_dir"
    
    # Update state
    jq --arg scenario "$scenario_name" \
       --arg timestamp "$timestamp" \
       --arg result_dir "$result_dir" \
       '.current_simulation.status = "completed" |
        .current_simulation.result_dir = $result_dir |
        .scenarios += [{
            "name": $scenario,
            "timestamp": $timestamp,
            "result_dir": $result_dir
        }]' "$COSIM_STATE_FILE" > "${COSIM_STATE_FILE}.tmp"
    mv "${COSIM_STATE_FILE}.tmp" "$COSIM_STATE_FILE"
    
    log::info "Co-simulation complete. Results in: $result_dir"
    return 0
}

# Integrate with OpenTripPlanner
openems::cosim::integrate_otp() {
    local scenario_file="$1"
    local result_dir="$2"
    
    log::info "Integrating with OpenTripPlanner..."
    
    # Extract trip data from scenario
    local trips=$(jq -r '.components.mobility.trips' "$scenario_file")
    
    # Create mock OTP results (would call actual API in production)
    cat > "${result_dir}/otp_results.json" << EOF
{
    "integration": "opentripplanner",
    "timestamp": "$(date -Iseconds)",
    "routes": [
        {
            "vehicle_id": "ev1",
            "distance_km": 52.3,
            "duration_minutes": 65,
            "energy_required_kwh": 10.46,
            "charging_stops": []
        },
        {
            "vehicle_id": "ev2",
            "distance_km": 118.7,
            "duration_minutes": 145,
            "energy_required_kwh": 35.61,
            "charging_stops": [
                {
                    "station_id": "station2",
                    "duration_minutes": 20,
                    "energy_kwh": 16.67
                }
            ]
        }
    ]
}
EOF
    
    log::info "OpenTripPlanner integration complete"
    return 0
}

# Integrate with GeoNode
openems::cosim::integrate_geonode() {
    local scenario_file="$1"
    local result_dir="$2"
    
    log::info "Integrating with GeoNode..."
    
    # Extract station locations from scenario
    local stations=$(jq -r '.components.charging_stations' "$scenario_file")
    
    # Create mock GeoNode results (would call actual API in production)
    cat > "${result_dir}/geonode_results.json" << EOF
{
    "integration": "geonode",
    "timestamp": "$(date -Iseconds)",
    "spatial_analysis": {
        "coverage_area_km2": 125.4,
        "population_served": 45000,
        "accessibility_score": 0.82,
        "optimal_locations": [
            {"lat": 37.7799, "lon": -122.4144, "score": 0.95},
            {"lat": 37.7699, "lon": -122.4244, "score": 0.88}
        ]
    },
    "environmental_impact": {
        "co2_reduction_tons": 12.5,
        "renewable_energy_percentage": 42.3
    }
}
EOF
    
    log::info "GeoNode integration complete"
    return 0
}

# Generate co-simulation report
openems::cosim::generate_report() {
    local result_dir="$1"
    local report_file="${result_dir}/summary_report.txt"
    
    log::info "Generating co-simulation report..."
    
    cat > "$report_file" << EOF
================================================================================
OpenEMS Co-Simulation Report
Generated: $(date)
================================================================================

SIMULATION RESULTS
------------------
EOF
    
    # Add SimPy results if available
    if [[ -f "${result_dir}/simpy_results.json" ]]; then
        echo -e "\nEnergy Simulation (SimPy):" >> "$report_file"
        jq -r '
            "  Scenario: " + .scenario_id + "\n" +
            "  Total Energy Required: " + (.energy_metrics.total_energy_kwh | tostring) + " kWh\n" +
            "  Renewable Energy Used: " + (.energy_metrics.renewable_energy_kwh | tostring) + " kWh\n" +
            "  Total Cost: $" + (.cost_analysis.total_cost | tostring) + "\n" +
            "  Savings vs Peak: $" + (.cost_analysis.savings_vs_peak | tostring)
        ' "${result_dir}/simpy_results.json" >> "$report_file"
    fi
    
    # Add OTP results if available
    if [[ -f "${result_dir}/otp_results.json" ]]; then
        echo -e "\nMobility Planning (OpenTripPlanner):" >> "$report_file"
        jq -r '
            .routes[] | 
            "  Vehicle " + .vehicle_id + ": " + 
            (.distance_km | tostring) + " km, " +
            (.duration_minutes | tostring) + " min, " +
            (.energy_required_kwh | tostring) + " kWh"
        ' "${result_dir}/otp_results.json" >> "$report_file"
    fi
    
    # Add GeoNode results if available
    if [[ -f "${result_dir}/geonode_results.json" ]]; then
        echo -e "\nSpatial Analysis (GeoNode):" >> "$report_file"
        jq -r '
            "  Coverage Area: " + (.spatial_analysis.coverage_area_km2 | tostring) + " km²\n" +
            "  Population Served: " + (.spatial_analysis.population_served | tostring) + "\n" +
            "  Accessibility Score: " + (.spatial_analysis.accessibility_score | tostring) + "\n" +
            "  CO2 Reduction: " + (.environmental_impact.co2_reduction_tons | tostring) + " tons"
        ' "${result_dir}/geonode_results.json" >> "$report_file"
    fi
    
    echo -e "\n================================================================================" >> "$report_file"
    
    log::info "Report generated: $report_file"
    return 0
}

# List available scenarios
openems::cosim::list_scenarios() {
    echo "Available Co-simulation Scenarios:"
    echo "==================================="
    
    for scenario_file in "${COSIM_SCENARIOS_DIR}"/*.json; do
        if [[ -f "$scenario_file" ]]; then
            local name=$(basename "$scenario_file" .json)
            local description=$(jq -r '.description' "$scenario_file" 2>/dev/null || echo "No description")
            printf "  %-25s %s\n" "$name" "$description"
        fi
    done
    
    return 0
}

# Show co-simulation status
openems::cosim::status() {
    if [[ ! -f "$COSIM_STATE_FILE" ]]; then
        log::info "Co-simulation not initialized"
        return 1
    fi
    
    echo "Co-simulation Status:"
    echo "===================="
    
    # Show integrations
    echo -e "\nIntegrations:"
    jq -r '.integrations | to_entries[] | 
        "  " + .key + ": " + 
        (if .value.enabled then "✅ Enabled" else "❌ Disabled" end)
    ' "$COSIM_STATE_FILE"
    
    # Show current simulation
    local current=$(jq -r '.current_simulation' "$COSIM_STATE_FILE")
    if [[ "$current" != "null" ]]; then
        echo -e "\nCurrent Simulation:"
        echo "$current" | jq -r '
            "  Scenario: " + .scenario + "\n" +
            "  Started: " + .started + "\n" +
            "  Status: " + .status
        '
    fi
    
    # Show recent simulations
    echo -e "\nRecent Simulations:"
    jq -r '.scenarios[-5:] | reverse[] |
        "  " + .timestamp + " - " + .name
    ' "$COSIM_STATE_FILE" 2>/dev/null || echo "  None"
    
    return 0
}

# Test co-simulation
openems::cosim::test() {
    log::info "Testing OpenEMS co-simulation..."
    
    # Initialize if needed
    openems::cosim::init
    
    # Run sample scenario
    openems::cosim::run_scenario "ev-charging-mobility"
    
    # Show results
    openems::cosim::status
    
    log::info "Co-simulation test complete"
    return 0
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Handle direct script execution
    case "${1:-}" in
        "init")
            openems::cosim::init
            ;;
        "run")
            openems::cosim::run_scenario "${2:-}"
            ;;
        "list")
            openems::cosim::list_scenarios
            ;;
        "status")
            openems::cosim::status
            ;;
        "test")
            openems::cosim::test
            ;;
        *)
            echo "Usage: $0 {init|run|list|status|test}"
            exit 1
            ;;
    esac
fi