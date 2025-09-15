#!/bin/bash
# KiCad SPICE Simulation Functions

# Get script directory and APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_SIM_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source common functions if not already sourced
if ! declare -f kicad::init_dirs &>/dev/null; then
    source "${KICAD_SIM_LIB_DIR}/common.sh"
fi

# Source logging functions
source "${APP_ROOT}/scripts/lib/utils/logging.sh"

# Check for SPICE simulator
kicad::simulation::check_ngspice() {
    if command -v ngspice &>/dev/null; then
        return 0
    else
        log::warning "ngspice not installed"
        echo "To enable SPICE simulation:"
        echo "  Ubuntu/Debian: sudo apt-get install ngspice"
        echo "  macOS: brew install ngspice"
        return 1
    fi
}

# Extract SPICE netlist from schematic
kicad::simulation::extract_netlist() {
    local schematic="${1:-}"
    local output="${2:-}"
    
    if [[ -z "$schematic" ]]; then
        echo "Usage: resource-kicad simulation extract <schematic.kicad_sch> [output.net]"
        return 1
    fi
    
    # Check if schematic file exists
    if [[ ! -f "$schematic" ]]; then
        log::error "Schematic file not found: $schematic"
        return 1
    fi
    
    if [[ -z "$output" ]]; then
        output="${schematic%.kicad_sch}.net"
    fi
    
    kicad::init_dirs
    
    # Check if KiCad CLI is available
    if command -v kicad-cli &>/dev/null; then
        # Use kicad-cli to export netlist
        kicad-cli sch export netlist "$schematic" -o "$output" --format spice
        log::success "SPICE netlist exported to: $output"
    else
        # Create mock netlist for development
        cat > "$output" <<'EOF'
* KiCad Netlist (Mock)
* Generated from: schematic.kicad_sch

.title Mock Circuit Simulation

* Voltage source
V1 VCC 0 DC 5V

* Resistors
R1 VCC n1 1k
R2 n1 0 2k

* Capacitor
C1 n1 0 100n

* Transistor (NPN)
Q1 out n1 0 2N2222

* Load resistor  
RL out VCC 10k

* Analysis commands
.tran 1u 10m
.dc V1 0 10 0.1
.ac dec 10 1 100k

.control
run
plot v(out) v(n1)
.endc

.end
EOF
        log::info "Mock SPICE netlist created: $output"
    fi
    
    return 0
}

# Run SPICE simulation
kicad::simulation::run() {
    local netlist="${1:-}"
    local analysis="${2:-tran}"  # tran, dc, ac, op
    
    if [[ -z "$netlist" ]]; then
        echo "Usage: resource-kicad simulation run <netlist.net> [analysis-type]"
        echo "Analysis types: tran (transient), dc, ac, op (operating point)"
        return 1
    fi
    
    if [[ ! -f "$netlist" ]]; then
        log::error "Netlist file not found: $netlist"
        return 1
    fi
    
    kicad::init_dirs
    local output_dir="${KICAD_OUTPUTS_DIR}/simulation"
    mkdir -p "$output_dir"
    
    # Check for ngspice
    if ! kicad::simulation::check_ngspice; then
        # Create mock simulation results
        local result_file="${output_dir}/$(basename "$netlist" .net)_${analysis}.txt"
        cat > "$result_file" <<EOF
Mock SPICE Simulation Results
=============================
Netlist: $netlist
Analysis: $analysis
Date: $(date)

Sample Results (Mock Data):
Time     V(out)   V(n1)    I(R1)
0ms      0.0V     0.0V     0.0mA
1ms      2.5V     1.2V     3.8mA
2ms      4.2V     2.1V     2.9mA
3ms      4.8V     2.4V     2.6mA
4ms      4.95V    2.48V    2.52mA
5ms      5.0V     2.5V     2.5mA

Simulation completed successfully (mock mode)
EOF
        log::info "Mock simulation results saved to: $result_file"
        cat "$result_file"
        return 0
    fi
    
    # Run actual ngspice simulation
    local spice_script="${output_dir}/run_${analysis}.sp"
    
    # Create simulation script based on analysis type
    case "$analysis" in
        tran)
            cat > "$spice_script" <<EOF
.control
source $netlist
tran 1u 10m
plot all
print all > ${output_dir}/transient_results.txt
quit
.endc
EOF
            ;;
        dc)
            cat > "$spice_script" <<EOF
.control
source $netlist
dc V1 0 10 0.1
plot all
print all > ${output_dir}/dc_results.txt
quit
.endc
EOF
            ;;
        ac)
            cat > "$spice_script" <<EOF
.control
source $netlist
ac dec 10 1 100k
plot vdb(out)
print all > ${output_dir}/ac_results.txt
quit
.endc
EOF
            ;;
        op)
            cat > "$spice_script" <<EOF
.control
source $netlist
op
print all > ${output_dir}/operating_point.txt
quit
.endc
EOF
            ;;
        *)
            log::error "Unknown analysis type: $analysis"
            return 1
            ;;
    esac
    
    echo "Running SPICE simulation..."
    ngspice -b "$spice_script" -o "${output_dir}/simulation.log" 2>&1
    
    if [[ $? -eq 0 ]]; then
        log::success "Simulation completed successfully"
        echo "Results saved in: $output_dir"
        ls -la "$output_dir"
    else
        log::error "Simulation failed. Check log: ${output_dir}/simulation.log"
        return 1
    fi
    
    return 0
}

# Interactive simulation shell
kicad::simulation::interactive() {
    local netlist="${1:-}"
    
    if [[ -z "$netlist" ]]; then
        echo "Usage: resource-kicad simulation interactive <netlist.net>"
        return 1
    fi
    
    if ! kicad::simulation::check_ngspice; then
        log::error "Interactive mode requires ngspice installation"
        return 1
    fi
    
    echo "Starting interactive SPICE simulation..."
    echo "Type 'quit' to exit"
    echo ""
    
    ngspice "$netlist"
}

# Generate simulation report
kicad::simulation::report() {
    local project="${1:-}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad simulation report <project-name>"
        return 1
    fi
    
    kicad::init_dirs
    local output_dir="${KICAD_OUTPUTS_DIR}/simulation"
    local report_file="${output_dir}/${project}_simulation_report.html"
    
    cat > "$report_file" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>KiCad Simulation Report - $project</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; }
        .result { background: #f5f5f5; padding: 10px; margin: 10px 0; }
        pre { background: #333; color: #0f0; padding: 10px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>Simulation Report: $project</h1>
    <div class="section">
        <h2>Project Information</h2>
        <p>Generated: $(date)</p>
        <p>Project Path: ${KICAD_PROJECTS_DIR}/$project</p>
    </div>
    
    <div class="section">
        <h2>Simulation Results</h2>
EOF
    
    # Add any simulation results found
    if [[ -d "$output_dir" ]]; then
        for result in "$output_dir"/*.txt; do
            if [[ -f "$result" ]]; then
                echo "<div class='result'>" >> "$report_file"
                echo "<h3>$(basename "$result")</h3>" >> "$report_file"
                echo "<pre>" >> "$report_file"
                head -50 "$result" >> "$report_file"
                echo "</pre>" >> "$report_file"
                echo "</div>" >> "$report_file"
            fi
        done
    else
        echo "<p>No simulation results found</p>" >> "$report_file"
    fi
    
    cat >> "$report_file" <<EOF
    </div>
</body>
</html>
EOF
    
    log::success "Simulation report generated: $report_file"
    echo "View report: $report_file"
    
    return 0
}

# Create SPICE models library
kicad::simulation::create_models() {
    kicad::init_dirs
    local models_dir="${KICAD_LIBRARIES_DIR}/spice_models"
    mkdir -p "$models_dir"
    
    # Create basic SPICE models
    cat > "${models_dir}/basic_components.lib" <<'EOF'
* Basic Component Models for KiCad SPICE Simulation

* Standard Diodes
.model 1N4148 D (IS=2.52e-9 RS=0.568 N=1.752 CJO=4e-12 M=0.333 VJ=0.75)
.model 1N4007 D (IS=7.02e-9 RS=0.0341 N=1.8 CJO=2.5e-11 M=0.333 VJ=0.75)

* Zener Diodes  
.model BZX84C5V1 D (IS=1e-15 RS=10 N=1.5 BV=5.1 IBV=1e-3)

* NPN Transistors
.model 2N2222 NPN (IS=14.34e-15 BF=255.9 NF=1 VAF=74.03 IKF=0.2847 ISE=14.34e-15)
.model BC547 NPN (IS=1.8e-14 BF=400 NF=1 VAF=100 IKF=0.08 ISE=2e-14)

* PNP Transistors
.model 2N2907 PNP (IS=650.6e-18 BF=231.7 NF=1.151 VAF=115.7 IKF=0.08)

* MOSFETs
.model IRF540 NMOS (VTO=4 KP=20 LAMBDA=0.01)
.model IRF9540 PMOS (VTO=-4 KP=10 LAMBDA=0.01)

* Op-Amps (simplified)
.subckt LM741 1 2 3 4 5
* Pins: 1=IN-, 2=IN+, 3=V-, 4=V+, 5=OUT
E1 5 0 2 1 100000
.ends

.subckt TL082 1 2 3 4 5 6 7 8
* Dual op-amp: pins 1-4 for amp1, pins 5-8 for amp2
E1 1 0 3 2 100000
E2 7 0 5 6 100000
.ends
EOF
    
    log::success "SPICE models library created: ${models_dir}/basic_components.lib"
    
    # Create example simulation circuits
    cat > "${models_dir}/example_circuits.cir" <<'EOF'
* Example Circuits for Testing

* RC Low-Pass Filter
.subckt RC_LOWPASS in out gnd
R1 in out 10k
C1 out gnd 100n
.ends

* Voltage Divider
.subckt VDIVIDER in out gnd
R1 in out 10k
R2 out gnd 10k
.ends

* LED Driver
.subckt LED_DRIVER vcc control led_out gnd
Q1 led_out control gnd 2N2222
R1 vcc led_out 330
.ends
EOF
    
    echo "SPICE models and examples created in: $models_dir"
    ls -la "$models_dir"
    
    return 0
}

# Export functions for CLI framework
export -f kicad::simulation::check_ngspice
export -f kicad::simulation::extract_netlist
export -f kicad::simulation::run
export -f kicad::simulation::interactive
export -f kicad::simulation::report
export -f kicad::simulation::create_models