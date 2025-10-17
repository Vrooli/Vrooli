#!/bin/bash
# SU2 Visualization Integration
# Connects SU2 results to OpenMCT and Superset dashboards

# Source defaults if not loaded
[[ -z "$SU2_PORT" ]] && source "${SCRIPT_DIR}/config/defaults.sh"

# OpenMCT telemetry adapter
setup_openmct_telemetry() {
    local openmct_host="${1:-localhost}"
    local openmct_port="${2:-8080}"
    
    echo "Setting up OpenMCT telemetry for SU2..."
    
    # Create telemetry configuration
    cat > "${SU2_DATA_DIR}/openmct_config.json" << EOF
{
    "name": "SU2 CFD Telemetry",
    "key": "su2.telemetry",
    "measurements": [
        {
            "key": "su2.cl",
            "name": "Lift Coefficient",
            "unit": "dimensionless",
            "format": "float",
            "min": -2,
            "max": 2
        },
        {
            "key": "su2.cd",
            "name": "Drag Coefficient", 
            "unit": "dimensionless",
            "format": "float",
            "min": 0,
            "max": 1
        },
        {
            "key": "su2.cm",
            "name": "Moment Coefficient",
            "unit": "dimensionless",
            "format": "float",
            "min": -1,
            "max": 1
        },
        {
            "key": "su2.residual",
            "name": "Convergence Residual",
            "unit": "log10",
            "format": "float",
            "min": -10,
            "max": 0
        },
        {
            "key": "su2.iterations",
            "name": "Iteration Count",
            "unit": "count",
            "format": "integer",
            "min": 0,
            "max": 10000
        }
    ]
}
EOF
    
    echo "OpenMCT configuration created"
    return 0
}

# Stream live telemetry during simulation
stream_telemetry() {
    local sim_id="${1:-}"
    local websocket_port="${2:-9515}"
    
    if [[ -z "$sim_id" ]]; then
        echo "Error: Simulation ID required" >&2
        return 1
    fi
    
    local result_dir="${SU2_RESULTS_DIR}/${sim_id}"
    local history_file="${result_dir}/history.csv"
    
    echo "Streaming telemetry for simulation ${sim_id}..."
    
    # Monitor history file and stream updates
    tail -F "$history_file" 2>/dev/null | while IFS=',' read -r iter cl cd cm res; do
        [[ "$iter" == "Iteration" ]] && continue  # Skip header
        
        # Create telemetry packet
        local timestamp=$(date +%s%3N)  # Milliseconds
        local telemetry="{
            \"timestamp\": ${timestamp},
            \"id\": \"su2.telemetry\",
            \"value\": {
                \"cl\": ${cl:-0},
                \"cd\": ${cd:-0},
                \"cm\": ${cm:-0},
                \"residual\": ${res:-0},
                \"iterations\": ${iter:-0}
            }
        }"
        
        # Send via websocket (placeholder - would need proper WS implementation)
        echo "$telemetry" >> "${result_dir}/telemetry.jsonl"
    done
}

# Superset dashboard configuration
setup_superset_dashboard() {
    local superset_host="${1:-localhost}"
    local superset_port="${2:-8088}"
    
    echo "Setting up Superset dashboard for SU2..."
    
    # Create SQL queries for Superset
    cat > "${SU2_DATA_DIR}/superset_queries.sql" << 'EOF'
-- Convergence History
SELECT 
    timestamp,
    simulation_id,
    iteration,
    cl as lift_coefficient,
    cd as drag_coefficient,
    cm as moment_coefficient,
    residual
FROM su2_simulations
WHERE simulation_id = '{{ simulation_id }}'
ORDER BY iteration;

-- Performance Comparison
SELECT 
    simulation_id,
    MAX(iteration) as total_iterations,
    MIN(residual) as final_residual,
    AVG(cl) as avg_lift,
    AVG(cd) as avg_drag,
    cl/cd as lift_drag_ratio
FROM su2_simulations
GROUP BY simulation_id
ORDER BY lift_drag_ratio DESC;

-- Design Space Exploration
SELECT 
    simulation_id,
    mach,
    aoa,
    cl,
    cd,
    cl/cd as efficiency
FROM su2_simulations
WHERE iteration = (
    SELECT MAX(iteration) 
    FROM su2_simulations s2 
    WHERE s2.simulation_id = su2_simulations.simulation_id
)
ORDER BY efficiency DESC;
EOF
    
    # Create dashboard configuration
    cat > "${SU2_DATA_DIR}/superset_dashboard.json" << EOF
{
    "dashboard_title": "SU2 CFD Analysis",
    "description": "Real-time CFD simulation monitoring and analysis",
    "charts": [
        {
            "name": "Convergence History",
            "type": "line",
            "query": "convergence_history",
            "x_axis": "iteration",
            "y_axis": ["residual"],
            "log_scale_y": true
        },
        {
            "name": "Aerodynamic Coefficients",
            "type": "line",
            "query": "convergence_history",
            "x_axis": "iteration",
            "y_axis": ["lift_coefficient", "drag_coefficient", "moment_coefficient"]
        },
        {
            "name": "Design Space",
            "type": "scatter",
            "query": "design_space",
            "x_axis": "aoa",
            "y_axis": "cl",
            "size": "efficiency",
            "color": "mach"
        },
        {
            "name": "Performance Comparison",
            "type": "bar",
            "query": "performance_comparison",
            "x_axis": "simulation_id",
            "y_axis": "lift_drag_ratio"
        }
    ]
}
EOF
    
    echo "Superset dashboard configuration created"
    return 0
}

# Generate VTK files for ParaView visualization
generate_vtk_visualization() {
    local sim_id="${1:-}"
    
    if [[ -z "$sim_id" ]]; then
        echo "Error: Simulation ID required" >&2
        return 1
    fi
    
    local result_dir="${SU2_RESULTS_DIR}/${sim_id}"
    
    echo "Generating VTK visualization for ${sim_id}..."
    
    # Check if VTK files exist
    if ls "${result_dir}"/*.vtk 1> /dev/null 2>&1; then
        echo "VTK files already exist"
    else
        # Convert SU2 output to VTK format
        if docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
            docker exec "${SU2_CONTAINER_NAME}" \
                SU2_SOL "${result_dir}/simulation.cfg" || {
                echo "Warning: VTK conversion failed" >&2
            }
        fi
    fi
    
    # Create ParaView state file for easy loading
    cat > "${result_dir}/paraview_state.pvsm" << EOF
<ParaView>
  <ServerManagerState>
    <Pipeline>
      <Source name="SU2_Results" type="LegacyVTKReader">
        <Property name="FileNames" value="${result_dir}/flow.vtk"/>
      </Source>
    </Pipeline>
    <ViewManager>
      <View type="RenderView">
        <Representation source="SU2_Results" type="Surface">
          <Property name="ColorArrayName" value="Pressure"/>
          <Property name="LookupTable" value="Rainbow"/>
        </Representation>
      </View>
    </ViewManager>
  </ServerManagerState>
</ParaView>
EOF
    
    echo "VTK visualization ready at: ${result_dir}"
    return 0
}

# Create animated convergence plot
create_convergence_animation() {
    local sim_id="${1:-}"
    local output_file="${2:-${SU2_RESULTS_DIR}/${sim_id}/convergence.gif}"
    
    if [[ -z "$sim_id" ]]; then
        echo "Error: Simulation ID required" >&2
        return 1
    fi
    
    local result_dir="${SU2_RESULTS_DIR}/${sim_id}"
    local history_file="${result_dir}/history.csv"
    
    if [[ ! -f "$history_file" ]]; then
        echo "Error: History file not found" >&2
        return 1
    fi
    
    echo "Creating convergence animation..."
    
    # Python script for animation (requires matplotlib)
    cat > "${result_dir}/plot_convergence.py" << 'EOF'
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.animation as animation
import sys

# Read data
df = pd.read_csv(sys.argv[1])
output_file = sys.argv[2] if len(sys.argv) > 2 else 'convergence.gif'

# Setup figure
fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(10, 8))

def animate(i):
    data = df.iloc[:i+1]
    
    ax1.clear()
    ax2.clear()
    
    # Residual plot
    ax1.semilogy(data.index, data['Residual'], 'b-')
    ax1.set_ylabel('Residual')
    ax1.set_xlabel('Iteration')
    ax1.grid(True)
    ax1.set_title('Convergence History')
    
    # Coefficients plot
    ax2.plot(data.index, data['CL'], 'r-', label='CL')
    ax2.plot(data.index, data['CD'], 'g-', label='CD')
    ax2.plot(data.index, data['CM'], 'b-', label='CM')
    ax2.set_ylabel('Coefficients')
    ax2.set_xlabel('Iteration')
    ax2.legend()
    ax2.grid(True)

# Create animation
ani = animation.FuncAnimation(fig, animate, frames=len(df), interval=50, repeat=True)
ani.save(output_file, writer='pillow')
print(f"Animation saved to {output_file}")
EOF
    
    # Run if Python is available in container
    if docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        docker exec "${SU2_CONTAINER_NAME}" \
            python3 "${result_dir}/plot_convergence.py" "$history_file" "$output_file" || {
            echo "Warning: Animation creation failed" >&2
        }
    fi
    
    echo "Convergence animation created: ${output_file}"
    return 0
}