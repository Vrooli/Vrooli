#!/usr/bin/env bash

# Energy Forecast Models for OpenEMS
# Provides solar, battery, and consumption forecasting capabilities

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

# QuestDB configuration for historical data
QUESTDB_PORT="${QUESTDB_PORT:-9010}"
QUESTDB_URL="http://localhost:${QUESTDB_PORT}"

# -------------------------------
# Solar Forecast Model
# -------------------------------

create_solar_forecast_model() {
    log_info "Creating solar forecast model"
    
    cat > "/tmp/solar_forecast.py" <<'EOF'
#!/usr/bin/env python3
"""
Solar Power Forecast Model for OpenEMS
Uses historical data and weather patterns for prediction
"""

import json
import math
from datetime import datetime, timedelta
import random  # Simplified for demo, replace with ML model

class SolarForecast:
    """Solar generation forecasting"""
    
    def __init__(self, capacity_kw=10):
        self.capacity = capacity_kw
        self.efficiency = 0.85
    
    def forecast_hourly(self, hours_ahead=24):
        """Generate hourly solar forecast"""
        forecasts = []
        base_time = datetime.now()
        
        for h in range(hours_ahead):
            forecast_time = base_time + timedelta(hours=h)
            hour = forecast_time.hour
            
            # Simple solar irradiance model
            if 6 <= hour <= 18:
                # Peak at noon, gaussian distribution
                solar_angle = (hour - 6) * 15  # degrees
                irradiance = 1000 * math.cos(math.radians(90 - solar_angle))
                irradiance = max(0, irradiance)
                
                # Add cloud cover factor (simplified)
                cloud_factor = 0.7 + random.random() * 0.3
                irradiance *= cloud_factor
                
                # Calculate power
                power = self.capacity * (irradiance / 1000) * self.efficiency
            else:
                power = 0
            
            forecasts.append({
                "timestamp": forecast_time.isoformat(),
                "hour": hour,
                "power_kw": round(power, 2),
                "confidence": 0.85 if 8 <= hour <= 16 else 0.95
            })
        
        return forecasts
    
    def forecast_daily(self, days_ahead=7):
        """Generate daily solar energy forecast"""
        daily_forecasts = []
        
        for d in range(days_ahead):
            date = datetime.now() + timedelta(days=d)
            hourly = self.forecast_hourly(24)
            
            # Sum hourly to get daily
            daily_energy = sum(h["power_kw"] * 0.25 for h in hourly)  # kWh
            
            daily_forecasts.append({
                "date": date.strftime("%Y-%m-%d"),
                "expected_energy_kwh": round(daily_energy, 2),
                "min_energy_kwh": round(daily_energy * 0.7, 2),
                "max_energy_kwh": round(daily_energy * 1.2, 2),
                "confidence": 0.8
            })
        
        return daily_forecasts
    
    def forecast_peak_hours(self):
        """Identify peak generation hours"""
        hourly = self.forecast_hourly(24)
        sorted_hours = sorted(hourly, key=lambda x: x["power_kw"], reverse=True)
        
        return {
            "peak_hours": [h["hour"] for h in sorted_hours[:3]],
            "peak_power_kw": sorted_hours[0]["power_kw"],
            "optimal_charging_window": {
                "start": 10,
                "end": 15,
                "expected_generation_kwh": sum(
                    h["power_kw"] * 0.25 
                    for h in hourly if 10 <= h["hour"] <= 15
                )
            }
        }

# Main execution
if __name__ == "__main__":
    import sys
    
    capacity = float(sys.argv[1]) if len(sys.argv) > 1 else 10
    forecast_type = sys.argv[2] if len(sys.argv) > 2 else "hourly"
    
    model = SolarForecast(capacity)
    
    if forecast_type == "hourly":
        result = model.forecast_hourly()
    elif forecast_type == "daily":
        result = model.forecast_daily()
    elif forecast_type == "peak":
        result = model.forecast_peak_hours()
    else:
        result = {"error": "Unknown forecast type"}
    
    print(json.dumps(result, indent=2))
EOF
    
    chmod +x /tmp/solar_forecast.py
    log_info "Solar forecast model created at /tmp/solar_forecast.py"
    return 0
}

# -------------------------------
# Battery Optimization Forecast
# -------------------------------

create_battery_forecast_model() {
    log_info "Creating battery optimization forecast model"
    
    cat > "/tmp/battery_forecast.py" <<'EOF'
#!/usr/bin/env python3
"""
Battery Optimization Forecast Model for OpenEMS
Predicts optimal charge/discharge schedules
"""

import json
from datetime import datetime, timedelta
import random  # Simplified, replace with ML/optimization

class BatteryForecast:
    """Battery operation forecasting and optimization"""
    
    def __init__(self, capacity_kwh=20, max_power_kw=5):
        self.capacity = capacity_kwh
        self.max_power = max_power_kw
        self.efficiency = 0.95
        self.min_soc = 20
        self.max_soc = 95
    
    def forecast_charge_schedule(self, solar_forecast, load_forecast, tariffs):
        """Generate optimal charge/discharge schedule"""
        schedule = []
        current_soc = 50  # Start at 50%
        
        for i in range(24):
            hour = (datetime.now() + timedelta(hours=i)).hour
            solar = solar_forecast[i] if i < len(solar_forecast) else 0
            load = load_forecast[i] if i < len(load_forecast) else 3
            tariff = tariffs.get(hour, 0.15)
            
            # Simple optimization logic
            net_power = solar - load
            
            if net_power > 0 and current_soc < self.max_soc:
                # Excess solar - charge battery
                charge_power = min(net_power, self.max_power)
                charge_energy = charge_power * self.efficiency
                new_soc = min(self.max_soc, 
                             current_soc + (charge_energy / self.capacity * 100))
                action = "charge"
            elif net_power < 0 and current_soc > self.min_soc:
                # Deficit - discharge battery
                discharge_power = min(abs(net_power), self.max_power)
                discharge_energy = discharge_power / self.efficiency
                new_soc = max(self.min_soc,
                            current_soc - (discharge_energy / self.capacity * 100))
                action = "discharge"
                charge_power = -discharge_power
            else:
                # Hold
                new_soc = current_soc
                charge_power = 0
                action = "hold"
            
            schedule.append({
                "hour": i,
                "timestamp": (datetime.now() + timedelta(hours=i)).isoformat(),
                "action": action,
                "power_kw": round(charge_power, 2),
                "soc_start": round(current_soc, 1),
                "soc_end": round(new_soc, 1),
                "tariff": tariff,
                "cost_benefit": round(charge_power * tariff * -1, 2)
            })
            
            current_soc = new_soc
        
        return schedule
    
    def forecast_revenue(self, schedule, market_prices):
        """Calculate potential revenue from optimized operation"""
        total_revenue = 0
        grid_services_revenue = 0
        arbitrage_revenue = 0
        
        for s in schedule:
            hour = s["hour"]
            power = s["power_kw"]
            price = market_prices.get(hour, 0.15)
            
            if power < 0:  # Discharging
                # Energy arbitrage
                arbitrage_revenue += abs(power) * price
                
                # Grid services (simplified)
                if 17 <= hour <= 21:  # Peak hours
                    grid_services_revenue += abs(power) * 0.05
        
        return {
            "total_revenue": round(arbitrage_revenue + grid_services_revenue, 2),
            "arbitrage_revenue": round(arbitrage_revenue, 2),
            "grid_services_revenue": round(grid_services_revenue, 2),
            "daily_cycles": sum(1 for s in schedule if s["action"] == "discharge") / 2
        }
    
    def forecast_lifetime(self, daily_cycles):
        """Predict battery lifetime based on usage"""
        max_cycles = 6000
        current_cycles = 100  # Assume some usage
        
        remaining_cycles = max_cycles - current_cycles
        remaining_days = remaining_cycles / daily_cycles if daily_cycles > 0 else 999999
        remaining_years = remaining_days / 365
        
        return {
            "estimated_lifetime_years": round(remaining_years, 1),
            "remaining_cycles": remaining_cycles,
            "daily_cycle_rate": daily_cycles,
            "health_percentage": 100 - (current_cycles / max_cycles * 20)
        }

# Main execution
if __name__ == "__main__":
    import sys
    
    # Simplified inputs
    solar_forecast = [0, 0, 0, 0, 0, 0, 1, 3, 5, 7, 8, 9, 9, 8, 7, 5, 3, 1, 0, 0, 0, 0, 0, 0]
    load_forecast = [2, 2, 2, 2, 3, 4, 5, 5, 4, 3, 3, 3, 3, 3, 4, 5, 6, 7, 6, 5, 4, 3, 2, 2]
    tariffs = {h: 0.25 if 17 <= h <= 21 else 0.15 for h in range(24)}
    market_prices = tariffs
    
    model = BatteryForecast()
    
    forecast_type = sys.argv[1] if len(sys.argv) > 1 else "schedule"
    
    if forecast_type == "schedule":
        result = model.forecast_charge_schedule(solar_forecast, load_forecast, tariffs)
    elif forecast_type == "revenue":
        schedule = model.forecast_charge_schedule(solar_forecast, load_forecast, tariffs)
        result = model.forecast_revenue(schedule, market_prices)
    elif forecast_type == "lifetime":
        result = model.forecast_lifetime(1.5)  # Average daily cycles
    else:
        result = {"error": "Unknown forecast type"}
    
    print(json.dumps(result, indent=2))
EOF
    
    chmod +x /tmp/battery_forecast.py
    log_info "Battery forecast model created at /tmp/battery_forecast.py"
    return 0
}

# -------------------------------
# Consumption Forecast Model
# -------------------------------

create_consumption_forecast_model() {
    log_info "Creating consumption forecast model"
    
    cat > "/tmp/consumption_forecast.py" <<'EOF'
#!/usr/bin/env python3
"""
Energy Consumption Forecast Model for OpenEMS
Predicts load patterns and peak demands
"""

import json
from datetime import datetime, timedelta
import random  # Simplified, replace with ML model

class ConsumptionForecast:
    """Energy consumption forecasting"""
    
    def __init__(self, base_load_kw=3):
        self.base_load = base_load_kw
        self.profiles = {
            "residential": [0.5, 0.4, 0.4, 0.4, 0.5, 0.7, 1.0, 1.2, 0.9, 0.7, 0.6, 0.6, 
                          0.7, 0.6, 0.6, 0.7, 0.9, 1.3, 1.4, 1.3, 1.1, 0.9, 0.7, 0.6],
            "commercial": [0.3, 0.3, 0.3, 0.3, 0.3, 0.5, 0.7, 0.9, 1.2, 1.3, 1.3, 1.3,
                         1.2, 1.3, 1.3, 1.2, 1.0, 0.8, 0.6, 0.5, 0.4, 0.3, 0.3, 0.3],
            "industrial": [0.8, 0.8, 0.8, 0.8, 0.9, 1.0, 1.1, 1.2, 1.2, 1.2, 1.2, 1.2,
                         1.2, 1.2, 1.2, 1.2, 1.1, 1.0, 0.9, 0.8, 0.8, 0.8, 0.8, 0.8]
        }
    
    def forecast_hourly(self, profile_type="residential", hours_ahead=24):
        """Generate hourly consumption forecast"""
        profile = self.profiles.get(profile_type, self.profiles["residential"])
        forecasts = []
        base_time = datetime.now()
        
        for h in range(hours_ahead):
            forecast_time = base_time + timedelta(hours=h)
            hour = forecast_time.hour
            day_of_week = forecast_time.weekday()
            
            # Base consumption from profile
            consumption = self.base_load * profile[hour]
            
            # Add day-of-week variation
            if day_of_week in [5, 6]:  # Weekend
                consumption *= 1.1
            
            # Add seasonal variation (simplified)
            month = forecast_time.month
            if month in [12, 1, 2]:  # Winter
                consumption *= 1.3  # Heating
            elif month in [6, 7, 8]:  # Summer
                consumption *= 1.2  # Cooling
            
            # Add random variation
            consumption *= (0.9 + random.random() * 0.2)
            
            forecasts.append({
                "timestamp": forecast_time.isoformat(),
                "hour": hour,
                "consumption_kw": round(consumption, 2),
                "confidence": 0.85,
                "profile": profile_type
            })
        
        return forecasts
    
    def forecast_peaks(self, days_ahead=7):
        """Forecast peak demand periods"""
        peaks = []
        
        for d in range(days_ahead):
            date = datetime.now() + timedelta(days=d)
            hourly = self.forecast_hourly("residential", 24)
            
            # Find peak hours
            sorted_hours = sorted(hourly, key=lambda x: x["consumption_kw"], reverse=True)
            daily_peak = sorted_hours[0]
            
            peaks.append({
                "date": date.strftime("%Y-%m-%d"),
                "peak_hour": daily_peak["hour"],
                "peak_demand_kw": daily_peak["consumption_kw"],
                "peak_period": "17:00-21:00" if 17 <= daily_peak["hour"] <= 21 else "other",
                "total_energy_kwh": sum(h["consumption_kw"] * 0.25 for h in hourly)
            })
        
        return peaks
    
    def forecast_anomalies(self, historical_data=None):
        """Detect potential consumption anomalies"""
        # Simplified anomaly detection
        hourly = self.forecast_hourly()
        mean_consumption = sum(h["consumption_kw"] for h in hourly) / len(hourly)
        std_dev = 1.0  # Simplified
        
        anomalies = []
        for h in hourly:
            deviation = abs(h["consumption_kw"] - mean_consumption)
            if deviation > 2 * std_dev:
                anomalies.append({
                    "timestamp": h["timestamp"],
                    "expected_kw": round(mean_consumption, 2),
                    "forecast_kw": h["consumption_kw"],
                    "deviation": round(deviation, 2),
                    "severity": "high" if deviation > 3 * std_dev else "medium"
                })
        
        return {
            "anomaly_count": len(anomalies),
            "anomalies": anomalies[:5],  # Top 5
            "baseline_consumption_kw": round(mean_consumption, 2)
        }
    
    def forecast_dr_potential(self):
        """Forecast demand response potential"""
        hourly = self.forecast_hourly()
        
        # Identify shiftable loads
        shiftable = 0
        curtailable = 0
        
        for h in hourly:
            hour = h["hour"]
            consumption = h["consumption_kw"]
            
            # Peak hours have more DR potential
            if 17 <= hour <= 21:
                shiftable += consumption * 0.2  # 20% shiftable
                curtailable += consumption * 0.1  # 10% curtailable
            else:
                shiftable += consumption * 0.1
                curtailable += consumption * 0.05
        
        return {
            "total_dr_potential_kwh": round((shiftable + curtailable) * 0.25, 2),
            "shiftable_load_kwh": round(shiftable * 0.25, 2),
            "curtailable_load_kwh": round(curtailable * 0.25, 2),
            "peak_reduction_potential_kw": round(max(h["consumption_kw"] for h in hourly) * 0.3, 2)
        }

# Main execution
if __name__ == "__main__":
    import sys
    
    profile = sys.argv[1] if len(sys.argv) > 1 else "residential"
    forecast_type = sys.argv[2] if len(sys.argv) > 2 else "hourly"
    
    model = ConsumptionForecast()
    
    if forecast_type == "hourly":
        result = model.forecast_hourly(profile)
    elif forecast_type == "peaks":
        result = model.forecast_peaks()
    elif forecast_type == "anomalies":
        result = model.forecast_anomalies()
    elif forecast_type == "dr":
        result = model.forecast_dr_potential()
    else:
        result = {"error": "Unknown forecast type"}
    
    print(json.dumps(result, indent=2))
EOF
    
    chmod +x /tmp/consumption_forecast.py
    log_info "Consumption forecast model created at /tmp/consumption_forecast.py"
    return 0
}

# -------------------------------
# Integrated Forecast Engine
# -------------------------------

run_solar_forecast() {
    local capacity="${1:-10}"
    local forecast_type="${2:-hourly}"
    
    log_info "Running solar forecast (${capacity}kW, ${forecast_type})"
    
    if [[ -f "/tmp/solar_forecast.py" ]]; then
        python3 /tmp/solar_forecast.py "${capacity}" "${forecast_type}"
    else
        log_error "Solar forecast model not found. Run 'create-models' first."
        return 1
    fi
}

run_battery_forecast() {
    local forecast_type="${1:-schedule}"
    
    log_info "Running battery optimization forecast (${forecast_type})"
    
    if [[ -f "/tmp/battery_forecast.py" ]]; then
        python3 /tmp/battery_forecast.py "${forecast_type}"
    else
        log_error "Battery forecast model not found. Run 'create-models' first."
        return 1
    fi
}

run_consumption_forecast() {
    local profile="${1:-residential}"
    local forecast_type="${2:-hourly}"
    
    log_info "Running consumption forecast (${profile}, ${forecast_type})"
    
    if [[ -f "/tmp/consumption_forecast.py" ]]; then
        python3 /tmp/consumption_forecast.py "${profile}" "${forecast_type}"
    else
        log_error "Consumption forecast model not found. Run 'create-models' first."
        return 1
    fi
}

run_integrated_forecast() {
    log_info "Running integrated energy forecast"
    
    # Run all forecasts and combine
    local solar_json=$(run_solar_forecast 10 hourly 2>/dev/null)
    local battery_json=$(run_battery_forecast schedule 2>/dev/null)
    local consumption_json=$(run_consumption_forecast residential hourly 2>/dev/null)
    
    # Create integrated forecast
    cat > "/tmp/integrated_forecast.json" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "forecast_horizon": "24h",
    "solar": ${solar_json:-"{}"},
    "battery": ${battery_json:-"{}"},
    "consumption": ${consumption_json:-"{}"},
    "summary": {
        "expected_solar_kwh": $(echo "${solar_json}" | jq '[.[] | .power_kw] | add * 0.25' 2>/dev/null || echo 0),
        "expected_consumption_kwh": $(echo "${consumption_json}" | jq '[.[] | .consumption_kw] | add * 0.25' 2>/dev/null || echo 0),
        "self_sufficiency": "calculated_on_demand",
        "optimization_potential": "high"
    }
}
EOF
    
    cat /tmp/integrated_forecast.json
    log_success "Integrated forecast saved to /tmp/integrated_forecast.json"
    return 0
}

# -------------------------------
# Store Forecasts in QuestDB
# -------------------------------

store_forecast_in_questdb() {
    local forecast_file="${1:-/tmp/integrated_forecast.json}"
    
    log_info "Storing forecast in QuestDB"
    
    # Create table if not exists
    local create_table_query="CREATE TABLE IF NOT EXISTS energy_forecasts (
        timestamp TIMESTAMP,
        forecast_type SYMBOL,
        horizon_hours INT,
        data STRING
    ) timestamp(timestamp) PARTITION BY DAY;"
    
    curl -G "${QUESTDB_URL}/exec" \
        --data-urlencode "query=${create_table_query}" &>/dev/null
    
    # Insert forecast
    local forecast_data=$(cat "${forecast_file}" | jq -c '.')
    local insert_query="INSERT INTO energy_forecasts VALUES(
        now(), 
        'integrated', 
        24, 
        '${forecast_data}'
    );"
    
    if curl -G "${QUESTDB_URL}/exec" \
        --data-urlencode "query=${insert_query}" &>/dev/null; then
        log_success "Forecast stored in QuestDB"
    else
        log_warn "Failed to store forecast in QuestDB"
    fi
    
    return 0
}

# -------------------------------
# Test Forecasting
# -------------------------------

test_forecast_models() {
    log_info "Testing all forecast models"
    
    # Test solar forecast
    if python3 /tmp/solar_forecast.py 10 hourly &>/dev/null; then
        log_success "Solar forecast model working"
    else
        log_error "Solar forecast model failed"
    fi
    
    # Test battery forecast  
    if python3 /tmp/battery_forecast.py schedule &>/dev/null; then
        log_success "Battery forecast model working"
    else
        log_error "Battery forecast model failed"
    fi
    
    # Test consumption forecast
    if python3 /tmp/consumption_forecast.py residential hourly &>/dev/null; then
        log_success "Consumption forecast model working"
    else
        log_error "Consumption forecast model failed"
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
        create-models)
            create_solar_forecast_model
            create_battery_forecast_model
            create_consumption_forecast_model
            echo ""
            log_success "All forecast models created in /tmp/"
            ;;
        solar)
            run_solar_forecast "$@"
            ;;
        battery)
            run_battery_forecast "$@"
            ;;
        consumption)
            run_consumption_forecast "$@"
            ;;
        integrated)
            run_integrated_forecast
            ;;
        store)
            store_forecast_in_questdb "$@"
            ;;
        test)
            test_forecast_models
            ;;
        help|*)
            echo "OpenEMS Energy Forecast Models"
            echo ""
            echo "Commands:"
            echo "  create-models      - Create all forecast models"
            echo "  solar [kW] [type]  - Run solar forecast"
            echo "  battery [type]     - Run battery optimization"
            echo "  consumption [profile] [type] - Run load forecast"
            echo "  integrated         - Run complete forecast"
            echo "  store [file]       - Store forecast in QuestDB"
            echo "  test              - Test all models"
            echo ""
            echo "Solar types: hourly, daily, peak"
            echo "Battery types: schedule, revenue, lifetime"
            echo "Consumption types: hourly, peaks, anomalies, dr"
            echo "Profiles: residential, commercial, industrial"
            echo ""
            echo "Examples:"
            echo "  ${0} create-models"
            echo "  ${0} solar 10 daily"
            echo "  ${0} battery revenue"
            echo "  ${0} consumption commercial peaks"
            echo "  ${0} integrated"
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi