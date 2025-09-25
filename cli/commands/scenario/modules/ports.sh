#!/usr/bin/env bash
# Scenario Port Management Module
# Handles port queries and allocation

set -euo pipefail

scenario::ports::get() {
    local json_output=false
    local -a positional=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                ;;
            *)
                positional+=("$1")
                ;;
        esac
        shift
    done

    local positional_count=${#positional[@]}
    local scenario_name=""
    local port_name=""

    if (( positional_count == 0 )); then
        log::error "Scenario name required"
        scenario::ports::print_usage
        return 1
    fi

    scenario_name="${positional[0]}"
    if (( positional_count > 2 )); then
        log::error "Too many arguments provided"
        scenario::ports::print_usage
        return 1
    elif (( positional_count == 2 ));
    then
        port_name="${positional[1]}"
    fi

    if [[ -z "$port_name" ]]; then
        scenario::ports::get_all "$scenario_name" "$json_output"
    else
        scenario::ports::get_single "$scenario_name" "$port_name" "$json_output"
    fi
}

scenario::ports::print_usage() {
    echo "Usage: vrooli scenario port <scenario-name> [<port-name>] [--json]"
    echo ""
    echo "Examples:"
    echo "  vrooli scenario port ecosystem-manager"  # List all ports
    echo "  vrooli scenario port ecosystem-manager API_PORT"
    echo "  vrooli scenario port ecosystem-manager UI_PORT --json"
}

scenario::ports::get_single() {
    local scenario_name="$1"
    local port_name="$2"
    local json_output="$3"

    if [[ -z "$port_name" ]]; then
        log::error "Port name required when requesting a specific port"
        scenario::ports::print_usage
        return 1
    fi

    local -a possible_steps=()
    case "$port_name" in
        API_PORT)
            possible_steps=("start-api" "start-app" "api" "app")
            ;;
        UI_PORT)
            possible_steps=("start-ui" "ui")
            ;;
        *)
            local base_name="${port_name%_PORT}"
            possible_steps=("start-${base_name,,}" "${base_name,,}")
            ;;
    esac

    local step_name
    local port_value=""

    for step_name in "${possible_steps[@]}"; do
        local process_file="$HOME/.vrooli/processes/scenarios/${scenario_name}/${step_name}.json"

        if [[ -f "$process_file" ]]; then
            port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)

            if [[ -n "$port_value" ]] && [[ "$port_value" != "null" ]]; then
                if [[ "$json_output" == "true" ]]; then
                    scenario::ports::print_single_json "$scenario_name" "$port_name" "$step_name" "$port_value"
                else
                    echo "$port_value"
                fi
                return 0
            fi
        fi
    done

    if [[ "$json_output" == "true" ]]; then
        jq -n --arg scenario "$scenario_name" --arg port_name "$port_name" '{
            success: false,
            scenario: $scenario,
            port_name: $port_name,
            error: "Port not found"
        }'
    fi
    return 1
}

scenario::ports::print_single_json() {
    local scenario_name="$1"
    local port_name="$2"
    local step_name="$3"
    local port_value="$4"

    jq -n \
        --arg scenario "$scenario_name" \
        --arg port_name "$port_name" \
        --arg step "$step_name" \
        --arg port_value "$port_value" '
        {
            success: true,
            scenario: $scenario,
            port_name: $port_name,
            step: $step,
            port: (if ($port_value | test("^[0-9]+$")) then ($port_value | tonumber) else $port_value end)
        }
    '
}

scenario::ports::get_all() {
    local scenario_name="$1"
    local json_output="$2"
    local scenario_dir="$HOME/.vrooli/processes/scenarios/${scenario_name}"
    local service_json="${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json"

    if [[ ! -d "$scenario_dir" ]]; then
        if [[ "$json_output" == "true" ]]; then
            jq -n --arg scenario "$scenario_name" '{
                success: false,
                scenario: $scenario,
                ports: [],
                error: "No running processes found for scenario"
            }'
        else
            log::error "No running processes found for scenario '$scenario_name'"
        fi
        return 1
    fi

    local ports_found=false
    local -a port_lines=()
    local entries_json=""

    # Load port metadata from service.json if available
    declare -A port_env_map=()
    declare -A port_fixed_map=()
    declare -A port_range_start_map=()
    declare -A port_range_end_map=()

    if [[ -f "$service_json" ]] && command -v jq >/dev/null 2>&1; then
        while IFS=$'\t' read -r port_key env_var fixed_port range; do
            [[ -n "$port_key" ]] || continue

            local normalized_key
            normalized_key=$(echo "$port_key" | tr '[:lower:]' '[:upper:]' | sed 's/[^A-Z0-9]/_/g')

            if [[ -z "$env_var" || "$env_var" == "null" ]]; then
                env_var="${normalized_key}_PORT"
            fi

            port_env_map["$port_key"]="$env_var"

            if [[ -n "$fixed_port" && "$fixed_port" != "null" ]]; then
                port_fixed_map["$port_key"]="${fixed_port}"
            fi

            if [[ -n "$range" && "$range" != "null" && "$range" =~ ^([0-9]+)-([0-9]+)$ ]]; then
                port_range_start_map["$port_key"]="${BASH_REMATCH[1]}"
                port_range_end_map["$port_key"]="${BASH_REMATCH[2]}"
            fi
        done < <(jq -r '
            (.ports // {}) | to_entries[] |
            [
                .key,
                (.value.env_var // ""),
                (if (.value.port? != null) then (.value.port | tostring) else "" end),
                (.value.range // "")
            ] | @tsv
        ' "$service_json" 2>/dev/null)
    fi

    shopt -s nullglob
    local process_file
    for process_file in "${scenario_dir}"/*.json; do
        [[ -f "$process_file" ]] || continue

        local step_name
        step_name=$(basename "$process_file" .json)
        local port_value
        port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)

        if [[ -n "$port_value" ]] && [[ "$port_value" != "null" ]]; then
            ports_found=true
            local env_var=""

            # Attempt to match against service.json definitions
            if [[ ${#port_env_map[@]} -gt 0 ]]; then
                local port_name
                for port_name in "${!port_env_map[@]}"; do
                    local matched=false
                    local fixed_port="${port_fixed_map[$port_name]:-}"
                    local range_start="${port_range_start_map[$port_name]:-}"
                    local range_end="${port_range_end_map[$port_name]:-}"

                    if [[ -n "$fixed_port" && "$port_value" == "$fixed_port" ]]; then
                        matched=true
                    elif [[ -n "$range_start" && -n "$range_end" && "$port_value" =~ ^[0-9]+$ ]]; then
                        if (( port_value >= range_start && port_value <= range_end )); then
                            matched=true
                        fi
                    fi

                    if [[ "$matched" == "true" ]]; then
                        env_var="${port_env_map[$port_name]}"
                        break
                    fi
                done
            fi

            # Final fallback: derive from step name
            if [[ -z "$env_var" ]]; then
                local derived_name="$step_name"
                if [[ "$derived_name" == start-* ]]; then
                    derived_name="${derived_name#start-}"
                fi
                derived_name="${derived_name//[^a-zA-Z0-9]/_}"
                derived_name="${derived_name^^}"
                env_var="${derived_name}_PORT"
            fi

            port_lines+=("${env_var}=${port_value}")

            if [[ "$json_output" == "true" ]]; then
                local entry
                entry=$(jq -n \
                    --arg key "$env_var" \
                    --arg step "$step_name" \
                    --arg port_value "$port_value" '
                    {
                        key: $key,
                        step: $step,
                        port: (if ($port_value | test("^[0-9]+$")) then ($port_value | tonumber) else $port_value end)
                    }
                ')
                entries_json+="${entry}"$'\n'
            fi
        fi
    done
    shopt -u nullglob

    if [[ "$ports_found" != "true" ]]; then
        if [[ "$json_output" == "true" ]]; then
            jq -n --arg scenario "$scenario_name" '{
                success: false,
                scenario: $scenario,
                ports: [],
                error: "No ports exposed by scenario processes"
            }'
        else
            log::error "No ports exposed by scenario '$scenario_name'"
        fi
        return 1
    fi

    if [[ "$json_output" == "true" ]]; then
        local ports_json
        ports_json=$(printf '%s' "$entries_json" | jq -s '.')
        jq -n --arg scenario "$scenario_name" --argjson ports "$ports_json" '{
            success: true,
            scenario: $scenario,
            ports: $ports,
            metadata: {
                count: ($ports | length)
            }
        }'
    else
        printf '%s\n' "${port_lines[@]}"
    fi
}

# Map port names to process step names (utility function)
# Returns the first/most likely step name for compatibility
scenario::ports::map_port_to_step() {
    local port_name="$1"
    
    case "$port_name" in
        API_PORT)
            echo "start-api"  # Most common convention
            ;;
        UI_PORT)
            echo "start-ui"
            ;;
        *)
            # For other port names, try lowercase without _PORT suffix
            # e.g., WORKER_PORT -> start-worker
            local base_name="${port_name%_PORT}"
            echo "start-${base_name,,}"
            ;;
    esac
}

# Get port value from process file (utility function)
scenario::ports::get_from_process_file() {
    local scenario_name="$1"
    local step_name="$2"
    
    local process_file="$HOME/.vrooli/processes/scenarios/${scenario_name}/${step_name}.json"
    
    if [[ ! -f "$process_file" ]]; then
        return 1
    fi
    
    # Extract port directly from JSON file
    local port_value
    port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)
    
    if [[ -n "$port_value" ]] && [[ "$port_value" != "null" ]]; then
        echo "$port_value"
        return 0
    else
        return 1
    fi
}
