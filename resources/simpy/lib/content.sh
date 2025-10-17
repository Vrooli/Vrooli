#!/usr/bin/env bash
# SimPy content management functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SIMPY_CONTENT_DIR="${APP_ROOT}/resources/simpy/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${SIMPY_CONTENT_DIR}/core.sh"

#######################################
# Add simulation script
#######################################
simpy::content::add() {
    local script_path="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$script_path" ]]; then
        log::error "Usage: simpy::content::add <script_path> [name]"
        return 1
    fi
    
    if [[ ! -f "$script_path" ]]; then
        log::error "Script file not found: $script_path"
        return 1
    fi
    
    # Generate name if not provided
    if [[ -z "$name" ]]; then
        name=$(basename "$script_path" .py)
    fi
    
    # Copy script to examples directory
    local dest_path="$SIMPY_EXAMPLES_DIR/${name}.py"
    cp "$script_path" "$dest_path"
    
    log::success "Added simulation script: $name"
}

#######################################
# List simulation scripts
#######################################
simpy::content::list() {
    local format="${1:-text}"
    
    log::info "Available simulation scripts:"
    
    if [[ ! -d "$SIMPY_EXAMPLES_DIR" ]]; then
        log::info "No examples directory found"
        return 0
    fi
    
    case "$format" in
        json)
            local scripts=()
            while IFS= read -r -d '' file; do
                local name=$(basename "$file" .py)
                scripts+=("\"$name\"")
            done < <(find "$SIMPY_EXAMPLES_DIR" -name "*.py" -type f -print0 2>/dev/null)
            
            echo "[$(IFS=,; echo "${scripts[*]}")]"
            ;;
        *)
            find "$SIMPY_EXAMPLES_DIR" -name "*.py" -type f 2>/dev/null | while read -r file; do
                local name=$(basename "$file" .py)
                echo "  • $name"
            done
            ;;
    esac
}

#######################################
# Get simulation script details
#######################################
simpy::content::get() {
    local name="${1:-}"
    local format="${2:-text}"
    
    if [[ -z "$name" ]]; then
        log::error "Usage: simpy::content::get <script_name> [format]"
        return 1
    fi
    
    local script_path="$SIMPY_EXAMPLES_DIR/${name}.py"
    
    if [[ ! -f "$script_path" ]]; then
        log::error "Script not found: $name"
        return 1
    fi
    
    case "$format" in
        json)
            local size=$(stat -c%s "$script_path" 2>/dev/null || echo "0")
            local modified=$(stat -c%Y "$script_path" 2>/dev/null || echo "0")
            echo "{\"name\":\"$name\",\"path\":\"$script_path\",\"size\":$size,\"modified\":$modified}"
            ;;
        *)
            log::info "Script: $name"
            log::info "Path: $script_path"
            log::info "Size: $(stat -c%s "$script_path" 2>/dev/null || echo "unknown") bytes"
            log::info "Modified: $(stat -c%y "$script_path" 2>/dev/null || echo "unknown")"
            echo ""
            log::info "Content preview:"
            head -20 "$script_path" | sed 's/^/  /'
            ;;
    esac
}

#######################################
# Remove simulation script
#######################################
simpy::content::remove() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Usage: simpy::content::remove <script_name>"
        return 1
    fi
    
    local script_path="$SIMPY_EXAMPLES_DIR/${name}.py"
    
    if [[ ! -f "$script_path" ]]; then
        log::error "Script not found: $name"
        return 1
    fi
    
    rm "$script_path"
    log::success "Removed simulation script: $name"
}

#######################################
# Execute simulation script
#######################################
simpy::content::execute() {
    local name="${1:-}"
    local output_path="${2:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Usage: simpy::content::execute <script_name> [output_path]"
        return 1
    fi
    
    local script_path="$SIMPY_EXAMPLES_DIR/${name}.py"
    
    if [[ ! -f "$script_path" ]]; then
        log::error "Script not found: $name"
        return 1
    fi
    
    # Generate output path if not provided
    if [[ -z "$output_path" ]]; then
        local timestamp=$(date +%Y%m%d_%H%M%S)
        output_path="$SIMPY_RESULTS_DIR/${name}_${timestamp}.json"
    fi
    
    # Ensure results directory exists
    mkdir -p "$SIMPY_RESULTS_DIR"
    
    log::info "Executing simulation: $name"
    
    # Run simulation
    if python3 "$script_path" > "$output_path" 2>&1; then
        log::success "Simulation completed successfully"
        log::info "Output saved to: $output_path"
        
        # Show results if they look like JSON
        if grep -q "^{" "$output_path" 2>/dev/null; then
            log::info "Results:"
            cat "$output_path" | jq '.' 2>/dev/null || cat "$output_path"
        fi
    else
        log::error "Simulation failed"
        log::error "Error output:"
        cat "$output_path"
        return 1
    fi
}