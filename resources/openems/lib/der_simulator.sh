#!/bin/bash

# DER (Distributed Energy Resource) Telemetry Simulator
# Simulates various DER assets for testing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"
source "${SCRIPT_DIR}/core.sh"

# ============================================
# DER Asset Simulation Functions
# ============================================

der::simulate_battery() {
    local capacity_kwh="${1:-20}"
    local soc_start="${2:-50}"
    local power_kw="${3:-5}"
    local duration="${4:-60}"
    
    echo "🔋 Simulating battery: ${capacity_kwh}kWh, SOC: ${soc_start}%, Power: ${power_kw}kW"
    
    local soc=$soc_start
    local count=0
    
    while [[ $count -lt $duration ]]; do
        # Calculate energy change
        local energy_change=$(awk "BEGIN {print $power_kw / 3600}")
        
        # Update SOC based on charge/discharge
        if [[ $(echo "$power_kw > 0" | bc -l) -eq 1 ]]; then
            # Charging
            soc=$(awk "BEGIN {print $soc + ($energy_change / $capacity_kwh * 100)}")
            [[ $(echo "$soc > 100" | bc -l) -eq 1 ]] && soc=100
        else
            # Discharging
            soc=$(awk "BEGIN {print $soc - ($energy_change / $capacity_kwh * 100)}")
            [[ $(echo "$soc < 0" | bc -l) -eq 1 ]] && soc=0
        fi
        
        # Send telemetry
        openems::send_telemetry "battery_01" "battery" \
            "$(echo "$power_kw * 1000" | bc)" \
            "$energy_change" \
            "400" \
            "$(echo "$power_kw * 1000 / 400" | bc)" \
            "$soc" \
            "25"
        
        echo "  Battery SOC: ${soc}%, Power: ${power_kw}kW"
        
        sleep 1
        ((count++))
    done
    
    echo "✅ Battery simulation completed"
    return 0
}

der::simulate_ev_charger() {
    local max_power="${1:-11}"  # kW
    local vehicle_connected="${2:-true}"
    local duration="${3:-30}"
    
    echo "🚗 Simulating EV charger: Max ${max_power}kW"
    
    local count=0
    local charging_power=0
    
    while [[ $count -lt $duration ]]; do
        if [[ "$vehicle_connected" == "true" ]]; then
            # Ramp up charging
            if [[ $count -lt 5 ]]; then
                charging_power=$(echo "$max_power * $count / 5" | bc -l)
            else
                charging_power=$max_power
            fi
        else
            charging_power=0
        fi
        
        # Send telemetry
        openems::send_telemetry "evcharger_01" "ev_charger" \
            "$(echo "$charging_power * 1000" | bc)" \
            "$(echo "$charging_power / 3600" | bc -l)" \
            "400" \
            "$(echo "$charging_power * 1000 / 400" | bc -l)" \
            "0" \
            "30"
        
        echo "  EV Charger: ${charging_power}kW"
        
        sleep 1
        ((count++))
    done
    
    echo "✅ EV charger simulation completed"
    return 0
}

der::simulate_wind_turbine() {
    local rated_power="${1:-50}"  # kW
    local wind_speed="${2:-8}"     # m/s
    local duration="${3:-30}"
    
    echo "💨 Simulating wind turbine: ${rated_power}kW rated"
    
    local count=0
    
    while [[ $count -lt $duration ]]; do
        # Wind speed variation
        local speed_variation=$(( RANDOM % 3 - 1 ))  # +/- 1 m/s
        local current_speed=$(( wind_speed + speed_variation ))
        
        # Power curve approximation
        local power=0
        if [[ $current_speed -ge 3 ]] && [[ $current_speed -le 25 ]]; then
            if [[ $current_speed -le 12 ]]; then
                # Cubic relationship below rated speed
                power=$(echo "$rated_power * ($current_speed / 12)^3" | bc -l)
            else
                # Rated power above 12 m/s
                power=$rated_power
            fi
        fi
        
        # Send telemetry
        openems::send_telemetry "wind_01" "wind_turbine" \
            "$(echo "$power * 1000" | bc)" \
            "$(echo "$power / 3600" | bc -l)" \
            "690" \
            "$(echo "$power * 1000 / 690" | bc -l)" \
            "0" \
            "15"
        
        echo "  Wind: ${current_speed}m/s, Power: ${power}kW"
        
        sleep 1
        ((count++))
    done
    
    echo "✅ Wind turbine simulation completed"
    return 0
}

der::simulate_microgrid() {
    local duration="${1:-60}"
    
    echo "🏘️ Simulating complete microgrid for ${duration}s"
    echo "Starting parallel DER simulations..."
    
    # Start simulations in background
    der::simulate_solar 10000 $duration &
    local solar_pid=$!
    
    der::simulate_battery 50 70 -5 $duration &
    local battery_pid=$!
    
    der::simulate_ev_charger 11 true $duration &
    local ev_pid=$!
    
    der::simulate_wind_turbine 20 10 $duration &
    local wind_pid=$!
    
    # Simulate load profile
    local count=0
    while [[ $count -lt $duration ]]; do
        # Residential load pattern (varies by time)
        local base_load=5000  # 5kW base
        local variation=$(( RANDOM % 2000 - 1000 ))  # +/- 1kW
        local total_load=$(( base_load + variation ))
        
        openems::send_telemetry "load_residential" "load" \
            "$total_load" \
            "$(echo "$total_load / 3600" | bc -l)" \
            "400" \
            "$(echo "$total_load / 400" | bc -l)" \
            "0" \
            "25"
        
        sleep 1
        ((count++))
    done &
    local load_pid=$!
    
    # Wait for all simulations
    wait $solar_pid $battery_pid $ev_pid $wind_pid $load_pid
    
    echo "✅ Microgrid simulation completed"
    return 0
}

# ============================================
# Grid Event Simulation
# ============================================

der::simulate_grid_outage() {
    local duration="${1:-30}"
    
    echo "⚡ Simulating grid outage for ${duration}s"
    
    # Signal grid disconnection
    echo "{\"event\":\"grid_outage\",\"timestamp\":\"$(date -Iseconds)\"}" \
        > "${DATA_DIR}/edge/data/grid_event.json"
    
    # Switch to island mode
    local count=0
    while [[ $count -lt $duration ]]; do
        # Battery provides backup power
        openems::send_telemetry "battery_backup" "battery" \
            "-8000" "2.22" "400" "20" "45" "28"
        
        echo "  Island mode: Battery discharging at 8kW"
        
        sleep 1
        ((count++))
    done
    
    # Grid reconnection
    echo "{\"event\":\"grid_restored\",\"timestamp\":\"$(date -Iseconds)\"}" \
        > "${DATA_DIR}/edge/data/grid_event.json"
    
    echo "✅ Grid outage simulation completed"
    return 0
}

# ============================================
# Main Entry Point
# ============================================

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being executed directly
    case "${1:-}" in
        solar)
            shift
            der::simulate_solar "$@"
            ;;
        battery)
            shift
            der::simulate_battery "$@"
            ;;
        ev-charger)
            shift
            der::simulate_ev_charger "$@"
            ;;
        wind)
            shift
            der::simulate_wind_turbine "$@"
            ;;
        microgrid)
            shift
            der::simulate_microgrid "$@"
            ;;
        grid-outage)
            shift
            der::simulate_grid_outage "$@"
            ;;
        *)
            echo "Usage: $0 {solar|battery|ev-charger|wind|microgrid|grid-outage} [options]"
            echo ""
            echo "Examples:"
            echo "  $0 solar 5000 30        # 5kW solar for 30 seconds"
            echo "  $0 battery 20 50 5 60   # 20kWh battery, 50% SOC, 5kW, 60s"
            echo "  $0 ev-charger 11 true 30 # 11kW charger with vehicle"
            echo "  $0 wind 50 8 30         # 50kW turbine, 8m/s wind"
            echo "  $0 microgrid 60         # Full microgrid for 60s"
            echo "  $0 grid-outage 30       # Grid outage for 30s"
            exit 1
            ;;
    esac
fi