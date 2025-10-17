#!/usr/bin/env bash

# Eclipse Ditto Integration for OpenEMS Digital Twins
# Creates digital twins of energy resources for simulation and co-simulation

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

# Ditto configuration
DITTO_PORT="${DITTO_PORT:-8080}"
DITTO_URL="http://localhost:${DITTO_PORT}"
DITTO_API="${DITTO_URL}/api/2"

# OpenEMS configuration
OPENEMS_REST_PORT="${OPENEMS_REST_PORT:-8084}"
OPENEMS_REST_URL="http://localhost:${OPENEMS_REST_PORT}"

# -------------------------------
# Digital Twin Models
# -------------------------------

create_solar_panel_twin() {
    local thing_id="${1:-openems:solar-panel-01}"
    local capacity="${2:-10000}"  # 10kW default
    
    log_info "Creating digital twin for solar panel: ${thing_id}"
    
    cat > "/tmp/solar-panel-twin.json" <<EOF
{
    "thingId": "${thing_id}",
    "policyId": "openems:energy-policy",
    "definition": "org.openems:SolarPanel:1.0.0",
    "attributes": {
        "manufacturer": "Generic Solar",
        "model": "PV-10KW",
        "serialNumber": "SP-2025-001",
        "location": {
            "latitude": 48.1351,
            "longitude": 11.5820,
            "installation": "rooftop"
        },
        "specifications": {
            "nominalPower": ${capacity},
            "efficiency": 0.22,
            "temperatureCoefficient": -0.0035,
            "degradationRate": 0.005
        }
    },
    "features": {
        "power": {
            "properties": {
                "currentPower": 0,
                "dailyEnergy": 0,
                "totalEnergy": 0,
                "status": "operational"
            }
        },
        "environmental": {
            "properties": {
                "irradiance": 0,
                "moduleTemperature": 25,
                "ambientTemperature": 20
            }
        },
        "performance": {
            "properties": {
                "efficiency": 0,
                "performanceRatio": 0,
                "availability": 1.0
            }
        }
    }
}
EOF
    
    log_info "Solar panel twin template created at /tmp/solar-panel-twin.json"
    return 0
}

create_battery_storage_twin() {
    local thing_id="${1:-openems:battery-01}"
    local capacity="${2:-20}"  # 20kWh default
    
    log_info "Creating digital twin for battery storage: ${thing_id}"
    
    cat > "/tmp/battery-storage-twin.json" <<EOF
{
    "thingId": "${thing_id}",
    "policyId": "openems:energy-policy",
    "definition": "org.openems:BatteryStorage:1.0.0",
    "attributes": {
        "manufacturer": "Generic Battery",
        "model": "ESS-20KWH",
        "serialNumber": "BAT-2025-001",
        "chemistry": "LiFePO4",
        "specifications": {
            "capacity": ${capacity},
            "nominalVoltage": 48,
            "maxChargePower": 5000,
            "maxDischargePower": 5000,
            "efficiency": 0.95,
            "cycleLife": 6000
        }
    },
    "features": {
        "soc": {
            "properties": {
                "stateOfCharge": 50,
                "stateOfHealth": 100,
                "voltage": 48.0,
                "current": 0,
                "power": 0
            }
        },
        "temperature": {
            "properties": {
                "cellTemperature": 25,
                "ambientTemperature": 20,
                "coolingStatus": "off"
            }
        },
        "statistics": {
            "properties": {
                "totalCycles": 0,
                "todayCharged": 0,
                "todayDischarged": 0,
                "lifetimeCharged": 0,
                "lifetimeDischarged": 0
            }
        },
        "control": {
            "properties": {
                "mode": "auto",
                "targetPower": 0,
                "minSoc": 20,
                "maxSoc": 95
            }
        }
    }
}
EOF
    
    log_info "Battery storage twin template created at /tmp/battery-storage-twin.json"
    return 0
}

create_ev_charger_twin() {
    local thing_id="${1:-openems:ev-charger-01}"
    local max_power="${2:-22000}"  # 22kW default
    
    log_info "Creating digital twin for EV charger: ${thing_id}"
    
    cat > "/tmp/ev-charger-twin.json" <<EOF
{
    "thingId": "${thing_id}",
    "policyId": "openems:energy-policy",
    "definition": "org.openems:EVCharger:1.0.0",
    "attributes": {
        "manufacturer": "Generic Charger",
        "model": "AC-22KW",
        "serialNumber": "EVC-2025-001",
        "type": "AC",
        "specifications": {
            "maxPower": ${max_power},
            "phases": 3,
            "connector": "Type2",
            "protocols": ["OCPP1.6", "OCPP2.0"]
        }
    },
    "features": {
        "status": {
            "properties": {
                "connectorStatus": "available",
                "vehicleConnected": false,
                "chargingActive": false,
                "currentPower": 0,
                "sessionEnergy": 0
            }
        },
        "vehicle": {
            "properties": {
                "vehicleId": null,
                "batteryCapacity": null,
                "targetSoc": null,
                "currentSoc": null
            }
        },
        "session": {
            "properties": {
                "sessionId": null,
                "startTime": null,
                "energy": 0,
                "cost": 0,
                "userId": null
            }
        },
        "control": {
            "properties": {
                "enabled": true,
                "maxCurrent": 32,
                "smartCharging": true,
                "priority": "normal"
            }
        }
    }
}
EOF
    
    log_info "EV charger twin template created at /tmp/ev-charger-twin.json"
    return 0
}

create_microgrid_twin() {
    local thing_id="${1:-openems:microgrid-01}"
    
    log_info "Creating digital twin for complete microgrid: ${thing_id}"
    
    cat > "/tmp/microgrid-twin.json" <<EOF
{
    "thingId": "${thing_id}",
    "policyId": "openems:energy-policy",
    "definition": "org.openems:Microgrid:1.0.0",
    "attributes": {
        "name": "OpenEMS Microgrid",
        "type": "hybrid",
        "location": "Main Campus",
        "components": {
            "solar": ["openems:solar-panel-01", "openems:solar-panel-02"],
            "batteries": ["openems:battery-01"],
            "evChargers": ["openems:ev-charger-01"],
            "loads": ["openems:load-building-01"]
        }
    },
    "features": {
        "gridConnection": {
            "properties": {
                "connected": true,
                "voltage": 230,
                "frequency": 50.0,
                "importPower": 0,
                "exportPower": 0
            }
        },
        "generation": {
            "properties": {
                "totalSolar": 0,
                "totalWind": 0,
                "totalGeneration": 0
            }
        },
        "storage": {
            "properties": {
                "totalCapacity": 20,
                "averageSoc": 50,
                "totalChargePower": 0,
                "totalDischargePower": 0
            }
        },
        "consumption": {
            "properties": {
                "totalLoad": 0,
                "criticalLoad": 0,
                "evCharging": 0
            }
        },
        "balance": {
            "properties": {
                "netPower": 0,
                "selfSufficiency": 0,
                "renewableShare": 0
            }
        },
        "optimization": {
            "properties": {
                "mode": "cost",
                "forecast": {
                    "solarNextHour": 0,
                    "loadNextHour": 0,
                    "priceNextHour": 0
                }
            }
        }
    }
}
EOF
    
    log_info "Microgrid twin template created at /tmp/microgrid-twin.json"
    return 0
}

# -------------------------------
# Twin Synchronization
# -------------------------------

sync_twin_with_openems() {
    local thing_id="${1}"
    
    log_info "Synchronizing digital twin ${thing_id} with OpenEMS"
    
    # Create synchronization script
    cat > "/tmp/twin-sync-${thing_id}.sh" <<'EOF'
#!/usr/bin/env bash

# Sync OpenEMS data to Ditto twin

THING_ID="${1}"
DITTO_URL="${2:-http://localhost:8080}"
OPENEMS_URL="${3:-http://localhost:8084}"

# Fetch OpenEMS data
SOLAR_POWER=$(curl -sf "${OPENEMS_URL}/rest/channel/meter0/ActivePower" | jq -r '.value // 0')
BATTERY_SOC=$(curl -sf "${OPENEMS_URL}/rest/channel/ess0/Soc" | jq -r '.value // 50')
GRID_POWER=$(curl -sf "${OPENEMS_URL}/rest/channel/_sum/GridActivePower" | jq -r '.value // 0')
CONSUMPTION=$(curl -sf "${OPENEMS_URL}/rest/channel/_sum/ConsumptionActivePower" | jq -r '.value // 0')

# Update Ditto twin
curl -X PATCH "${DITTO_URL}/api/2/things/${THING_ID}/features/power/properties/currentPower" \
    -H "Content-Type: application/json" \
    -d "${SOLAR_POWER}"

curl -X PATCH "${DITTO_URL}/api/2/things/${THING_ID}/features/soc/properties/stateOfCharge" \
    -H "Content-Type: application/json" \
    -d "${BATTERY_SOC}"

echo "Twin ${THING_ID} synchronized at $(date)"
EOF
    
    chmod +x "/tmp/twin-sync-${thing_id}.sh"
    log_info "Twin synchronization script created at /tmp/twin-sync-${thing_id}.sh"
    return 0
}

# -------------------------------
# Co-Simulation Support
# -------------------------------

create_simpy_cosim_bridge() {
    log_info "Creating SimPy co-simulation bridge for grid digital twins"
    
    cat > "/tmp/simpy-cosim-bridge.py" <<'EOF'
#!/usr/bin/env python3
"""
SimPy Co-Simulation Bridge for OpenEMS/Ditto Digital Twins
Enables discrete event simulation of energy systems
"""

import simpy
import requests
import json
from typing import Dict, Any
import time

class EnergyResource:
    """Base class for energy resources in simulation"""
    
    def __init__(self, env: simpy.Environment, thing_id: str, ditto_url: str):
        self.env = env
        self.thing_id = thing_id
        self.ditto_url = f"{ditto_url}/api/2/things/{thing_id}"
        self.state = {}
    
    def update_twin(self, feature: str, properties: Dict[str, Any]):
        """Update digital twin in Ditto"""
        url = f"{self.ditto_url}/features/{feature}/properties"
        response = requests.patch(url, json=properties)
        return response.status_code == 200

class SolarPanel(EnergyResource):
    """Solar panel simulation model"""
    
    def __init__(self, env, thing_id, ditto_url, capacity=10000):
        super().__init__(env, thing_id, ditto_url)
        self.capacity = capacity
        self.env.process(self.generate())
    
    def generate(self):
        """Simulate solar generation based on time of day"""
        while True:
            hour = (self.env.now // 3600) % 24
            
            # Simple irradiance model
            if 6 <= hour <= 18:
                irradiance = 1000 * (1 - abs(hour - 12) / 6)
                power = self.capacity * (irradiance / 1000) * 0.9
            else:
                irradiance = 0
                power = 0
            
            # Update digital twin
            self.update_twin("power", {"currentPower": power})
            self.update_twin("environmental", {"irradiance": irradiance})
            
            yield self.env.timeout(300)  # Update every 5 minutes

class Battery(EnergyResource):
    """Battery storage simulation model"""
    
    def __init__(self, env, thing_id, ditto_url, capacity=20):
        super().__init__(env, thing_id, ditto_url)
        self.capacity = capacity * 1000  # Convert to Wh
        self.soc = 50  # Start at 50% SOC
        self.power = 0
        self.env.process(self.manage())
    
    def manage(self):
        """Simulate battery charge/discharge"""
        while True:
            # Update SOC based on power flow
            energy_delta = self.power * (5/60)  # 5 minute interval
            self.soc += (energy_delta / self.capacity) * 100
            self.soc = max(20, min(95, self.soc))  # Keep within limits
            
            # Update digital twin
            self.update_twin("soc", {
                "stateOfCharge": self.soc,
                "power": self.power
            })
            
            yield self.env.timeout(300)  # Update every 5 minutes
    
    def charge(self, power):
        """Set charging power"""
        self.power = min(power, 5000)  # Max 5kW
    
    def discharge(self, power):
        """Set discharging power"""
        self.power = -min(power, 5000)  # Max 5kW

class MicrogridSimulation:
    """Main microgrid simulation controller"""
    
    def __init__(self, ditto_url="http://localhost:8080"):
        self.env = simpy.Environment()
        self.ditto_url = ditto_url
        self.resources = {}
    
    def add_solar(self, thing_id, capacity):
        """Add solar panel to simulation"""
        self.resources[thing_id] = SolarPanel(
            self.env, thing_id, self.ditto_url, capacity
        )
    
    def add_battery(self, thing_id, capacity):
        """Add battery to simulation"""
        self.resources[thing_id] = Battery(
            self.env, thing_id, self.ditto_url, capacity
        )
    
    def run(self, duration=86400):
        """Run simulation for specified duration (seconds)"""
        print(f"Starting microgrid simulation for {duration}s")
        self.env.run(until=duration)
        print("Simulation completed")

# Example usage
if __name__ == "__main__":
    # Create simulation
    sim = MicrogridSimulation()
    
    # Add resources
    sim.add_solar("openems:solar-panel-01", capacity=10000)
    sim.add_battery("openems:battery-01", capacity=20)
    
    # Run for 24 hours
    sim.run(duration=86400)
EOF
    
    log_info "SimPy co-simulation bridge created at /tmp/simpy-cosim-bridge.py"
    return 0
}

create_blender_cosim_config() {
    log_info "Creating Blender co-simulation configuration for 3D visualization"
    
    cat > "/tmp/blender-cosim-config.json" <<'EOF'
{
    "name": "OpenEMS Energy System Visualization",
    "description": "3D visualization of digital twin energy flows",
    "ditto_connection": {
        "url": "http://localhost:8080",
        "api_version": "2",
        "polling_interval": 5
    },
    "scene_elements": {
        "solar_panels": {
            "thing_pattern": "openems:solar-panel-*",
            "model": "solar_panel.blend",
            "visualizations": {
                "power_flow": {
                    "property": "features.power.properties.currentPower",
                    "type": "particle_system",
                    "color_map": "yellow_gradient"
                },
                "efficiency": {
                    "property": "features.performance.properties.efficiency",
                    "type": "heat_map",
                    "color_map": "red_green"
                }
            }
        },
        "batteries": {
            "thing_pattern": "openems:battery-*",
            "model": "battery_storage.blend",
            "visualizations": {
                "soc": {
                    "property": "features.soc.properties.stateOfCharge",
                    "type": "fill_level",
                    "color_map": "battery_gradient"
                },
                "power": {
                    "property": "features.soc.properties.power",
                    "type": "arrow_flow",
                    "bidirectional": true
                }
            }
        },
        "grid_connection": {
            "thing_pattern": "openems:microgrid-*",
            "model": "grid_connection.blend",
            "visualizations": {
                "power_flow": {
                    "property": "features.gridConnection.properties.importPower",
                    "type": "animated_line",
                    "color_map": "power_flow"
                }
            }
        }
    },
    "animation_settings": {
        "fps": 30,
        "time_scale": 60,
        "interpolation": "linear"
    }
}
EOF
    
    log_info "Blender co-simulation config created at /tmp/blender-cosim-config.json"
    return 0
}

# -------------------------------
# Testing
# -------------------------------

test_ditto_connectivity() {
    log_info "Testing Eclipse Ditto connectivity"
    
    if curl -sf "${DITTO_URL}/health" &>/dev/null; then
        log_success "Ditto is reachable at ${DITTO_URL}"
        return 0
    else
        log_warn "Ditto is not reachable. Please ensure it's running."
        echo "Start Ditto with: vrooli resource eclipse-ditto manage start"
        return 1
    fi
}

test_twin_creation() {
    local thing_id="openems:test-twin"
    
    log_info "Testing digital twin creation"
    
    # Create test twin
    curl -X PUT "${DITTO_API}/things/${thing_id}" \
        -H "Content-Type: application/json" \
        -d '{
            "thingId": "'${thing_id}'",
            "attributes": {"test": true},
            "features": {
                "test": {
                    "properties": {"value": 42}
                }
            }
        }' &>/dev/null
    
    # Verify creation
    if curl -sf "${DITTO_API}/things/${thing_id}" &>/dev/null; then
        log_success "Digital twin created successfully"
        
        # Clean up
        curl -X DELETE "${DITTO_API}/things/${thing_id}" &>/dev/null
        return 0
    else
        log_error "Failed to create digital twin"
        return 1
    fi
}

# -------------------------------
# Deployment Instructions
# -------------------------------

show_deployment_instructions() {
    echo "═══════════════════════════════════════════════════════════════"
    echo "           Eclipse Ditto Digital Twin Deployment              "
    echo "═══════════════════════════════════════════════════════════════"
    echo ""
    echo "1. START ECLIPSE DITTO:"
    echo "   vrooli resource eclipse-ditto manage start"
    echo ""
    echo "2. CREATE DIGITAL TWINS:"
    echo "   # Create individual twins"
    echo "   curl -X PUT ${DITTO_API}/things/openems:solar-panel-01 \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -d @/tmp/solar-panel-twin.json"
    echo ""
    echo "   curl -X PUT ${DITTO_API}/things/openems:battery-01 \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -d @/tmp/battery-storage-twin.json"
    echo ""
    echo "3. SET UP SYNCHRONIZATION:"
    echo "   # Run sync script periodically"
    echo "   watch -n 60 /tmp/twin-sync-openems:solar-panel-01.sh"
    echo ""
    echo "4. START CO-SIMULATION (Optional):"
    echo "   # SimPy simulation"
    echo "   python3 /tmp/simpy-cosim-bridge.py"
    echo ""
    echo "   # Blender visualization"
    echo "   vrooli resource blender content execute \\"
    echo "     load-config /tmp/blender-cosim-config.json"
    echo ""
    echo "5. ACCESS DITTO UI:"
    echo "   Open browser: ${DITTO_URL}"
    echo "   View twins at: ${DITTO_API}/things"
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
}

# -------------------------------
# Main execution
# -------------------------------

main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        create-twins)
            create_solar_panel_twin "$@"
            create_battery_storage_twin "$@"
            create_ev_charger_twin "$@"
            create_microgrid_twin "$@"
            echo ""
            log_success "All digital twin templates created in /tmp/"
            ;;
        create-cosim)
            create_simpy_cosim_bridge
            create_blender_cosim_config
            echo ""
            log_success "Co-simulation bridges created in /tmp/"
            ;;
        sync)
            sync_twin_with_openems "${1:-openems:solar-panel-01}"
            ;;
        test)
            test_ditto_connectivity
            test_twin_creation
            ;;
        deploy)
            show_deployment_instructions
            ;;
        help|*)
            echo "OpenEMS Eclipse Ditto Integration"
            echo ""
            echo "Commands:"
            echo "  create-twins     - Create digital twin templates"
            echo "  create-cosim     - Create co-simulation bridges"
            echo "  sync <thing-id>  - Create sync script for twin"
            echo "  test            - Test Ditto connectivity"
            echo "  deploy          - Show deployment instructions"
            echo ""
            echo "Usage:"
            echo "  ${0} create-twins"
            echo "  ${0} create-cosim"
            echo "  ${0} sync openems:battery-01"
            echo "  ${0} test"
            echo "  ${0} deploy"
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi