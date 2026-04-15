#!/usr/bin/env bash
#######################################
# Setup State Management Library
# Provides setup state tracking and condition checking for Vrooli platform
#######################################

set -euo pipefail

# Global variables for setup reasons
SETUP_REASONS=()

setup::resolve_path() {
    local root="${1}"
    local path="${2}"
    if [[ "$path" =~ ^/ ]]; then
        printf '%s\n' "$path"
    else
        printf '%s\n' "${root}/${path}"
    fi
}

setup::local_replace_paths() {
    local go_mod="${1}"
    [[ -f "$go_mod" ]] || return 0

    awk '
        /^replace[[:space:]]+\($/ { in_block=1; next }
        in_block && /^\)/ { in_block=0; next }
        /^replace[[:space:]]+/ && !in_block {
            print $NF
            next
        }
        in_block && /=>/ {
            print $NF
        }
    ' "$go_mod" | grep '^\.\./' || true
}

setup::state_dir() {
    printf '%s\n' "${APP_ROOT}/.vrooli/state/setup"
}

setup::marker_exists() {
    local current_path="${1}"
    [[ -f "$current_path" ]]
}

setup::check_condition() {
    local app_root="${1}"
    local check="${2}"
    local check_type="${3}"

    case "$check_type" in
        binaries|"")
            local target
            while IFS= read -r target; do
                [[ -z "$target" ]] && continue
                local resolved
                resolved=$(setup::resolve_path "$app_root" "$target")
                if [[ ! -f "$resolved" || ! -x "$resolved" ]]; then
                    return 0
                fi

                local binary_dir
                binary_dir=$(dirname "$resolved")
                if find "$binary_dir" -name "*.go" -newer "$resolved" 2>/dev/null | head -1 | grep -q .; then
                    return 0
                fi
                for dep_file in go.mod go.sum; do
                    if [[ -f "$binary_dir/$dep_file" && "$binary_dir/$dep_file" -nt "$resolved" ]]; then
                        return 0
                    fi
                done
                while IFS= read -r replace_path; do
                    [[ -z "$replace_path" ]] && continue
                    local replace_dir
                    replace_dir=$(setup::resolve_path "$binary_dir" "$replace_path")
                    if [[ -d "$replace_dir" ]]; then
                        if find "$replace_dir" \( -name "*.go" -o -name "go.mod" \) -newer "$resolved" 2>/dev/null | head -1 | grep -q .; then
                            return 0
                        fi
                    fi
                done < <(setup::local_replace_paths "$binary_dir/go.mod")
            done < <(echo "$check" | jq -r '.targets[]?' 2>/dev/null)
            return 1
            ;;
        cli)
            local command
            command=$(echo "$check" | jq -r '.command // empty' 2>/dev/null)
            [[ -z "$command" ]] && return 1

            local cli_path
            cli_path=$(command -v "$command" 2>/dev/null || true)
            [[ -z "$cli_path" ]] && return 0

            local cli_source_dir="${app_root}/cli"
            [[ -d "$cli_source_dir" ]] || return 1
            if find "$cli_source_dir" -name "*.go" -newer "$cli_path" 2>/dev/null | head -1 | grep -q .; then
                return 0
            fi
            for dep_file in go.mod go.sum; do
                if [[ -f "$cli_source_dir/$dep_file" && "$cli_source_dir/$dep_file" -nt "$cli_path" ]]; then
                    return 0
                fi
            done
            while IFS= read -r replace_path; do
                [[ -z "$replace_path" ]] && continue
                local replace_dir
                replace_dir=$(setup::resolve_path "$cli_source_dir" "$replace_path")
                if [[ -d "$replace_dir" ]]; then
                    if find "$replace_dir" \( -name "*.go" -o -name "go.mod" \) -newer "$cli_path" 2>/dev/null | head -1 | grep -q .; then
                        return 0
                    fi
                fi
            done < <(setup::local_replace_paths "$cli_source_dir/go.mod")
            return 1
            ;;
        ui-bundle)
            local bundle_path source_dir watch_file_dependencies
            bundle_path=$(echo "$check" | jq -r '.bundle_path // "ui/dist/index.html"' 2>/dev/null)
            source_dir=$(echo "$check" | jq -r '.source_dir // "ui/src"' 2>/dev/null)
            watch_file_dependencies=$(echo "$check" | jq -r 'if has("watch_file_dependencies") then .watch_file_dependencies else true end' 2>/dev/null)

            bundle_path=$(setup::resolve_path "$app_root" "$bundle_path")
            source_dir=$(setup::resolve_path "$app_root" "$source_dir")
            [[ -f "$bundle_path" ]] || return 0
            if [[ -d "$source_dir" ]] && find "$source_dir" -type f -newer "$bundle_path" 2>/dev/null | head -1 | grep -q .; then
                return 0
            fi

            local ui_dir
            ui_dir=$(dirname "$(dirname "$bundle_path")")
            for config_file in package.json vite.config.ts vite.config.js tsconfig.json index.html; do
                if [[ -f "$ui_dir/$config_file" && "$ui_dir/$config_file" -nt "$bundle_path" ]]; then
                    return 0
                fi
            done

            if [[ "$watch_file_dependencies" == "true" && -f "$ui_dir/package.json" ]]; then
                while IFS= read -r dep_spec; do
                    [[ -z "$dep_spec" ]] && continue
                    local dep_path="${dep_spec#file:}"
                    dep_path=$(setup::resolve_path "$ui_dir" "$dep_path")
                    [[ -d "$dep_path" ]] || return 0
                    if find "$dep_path" -type f \
                        -not -path '*/node_modules/*' \
                        -not -path '*/.git/*' \
                        -not -path '*/coverage/*' \
                        -newer "$bundle_path" 2>/dev/null | head -1 | grep -q .; then
                        return 0
                    fi
                done < <(jq -r '
                    [
                      .dependencies,
                      .devDependencies,
                      .peerDependencies,
                      .optionalDependencies
                    ]
                    | map(. // {})
                    | add
                    | to_entries[]
                    | select((.value | type) == "string" and (.value | startswith("file:")))
                    | .value
                ' "$ui_dir/package.json" 2>/dev/null || true)
            fi
            return 1
            ;;
        resources)
            local populated resources
            populated=$(echo "$check" | jq -r '.populated // false' 2>/dev/null)
            resources=$(echo "$check" | jq -r '.resources[]?' 2>/dev/null || true)
            local state_dir
            state_dir=$(setup::state_dir)
            if [[ "$populated" == "true" || -z "$resources" ]]; then
                setup::marker_exists "${state_dir}/.resources-populated" && return 1 || return 0
            fi
            local resource
            while IFS= read -r resource; do
                [[ -z "$resource" ]] && continue
                setup::marker_exists "${state_dir}/.${resource}-populated" || return 0
            done <<< "$resources"
            return 1
            ;;
        dependencies)
            local dep_path
            while IFS= read -r dep_path; do
                [[ -z "$dep_path" ]] && continue
                local resolved
                resolved=$(setup::resolve_path "$app_root" "$dep_path")
                case "$resolved" in
                    */package.json)
                        [[ -d "$(dirname "$resolved")/node_modules" ]] || return 0
                        ;;
                    */go.mod)
                        [[ -f "$(dirname "$resolved")/go.sum" || -d "$(dirname "$resolved")/vendor" ]] || return 0
                        ;;
                    */requirements.txt)
                        [[ -d "$(dirname "$resolved")/venv" || -d "$(dirname "$resolved")/.venv" ]] || return 0
                        ;;
                    */Cargo.toml)
                        [[ -d "$(dirname "$resolved")/target" ]] || return 0
                        ;;
                    *)
                        [[ -e "$resolved" ]] || return 0
                        ;;
                esac
            done < <(echo "$check" | jq -r '.paths[]?' 2>/dev/null)
            return 1
            ;;
        data)
            local data_path
            data_path=$(echo "$check" | jq -r '.path // "data"' 2>/dev/null)
            data_path=$(setup::resolve_path "$app_root" "$data_path")
            [[ -d "$data_path" && -n "$(ls -A "$data_path" 2>/dev/null)" ]] && return 1 || return 0
            ;;
        files)
            local file_path
            while IFS= read -r file_path; do
                [[ -z "$file_path" ]] && continue
                [[ -e "$(setup::resolve_path "$app_root" "$file_path")" ]] || return 0
            done < <(echo "$check" | jq -r '.paths[]?' 2>/dev/null)
            return 1
            ;;
        directories)
            local dir_path
            while IFS= read -r dir_path; do
                [[ -z "$dir_path" ]] && continue
                [[ -d "$(setup::resolve_path "$app_root" "$dir_path")" ]] || return 0
            done < <(echo "$check" | jq -r '.targets[]?' 2>/dev/null)
            return 1
            ;;
        *)
            log::warn "Unsupported setup condition type '$check_type' in shell compatibility layer"
            return 0
            ;;
    esac
}

#######################################
# Check if app needs setup based on service.json conditions
# Sets SETUP_REASONS global array with specific reasons
#
# Logic: Runs checks defined in service.json lifecycle.setup.condition.checks
# Returns:
#   0 if setup is needed
#   1 if setup is not needed
#######################################
setup::is_needed() {
    # Accept optional path parameter for service.json location
    local check_path="${1:-$APP_ROOT}"

    # Reset global array for setup reasons
    SETUP_REASONS=()

    # Track if force setup is active and whether it applies to this scenario
    local force_setup="${FORCE_SETUP:-false}"
    local force_setup_target="${FORCE_SETUP_SCENARIO:-}"
    local scenario_name_from_path
    scenario_name_from_path=$(basename "$check_path")
    scenario_name_from_path=${scenario_name_from_path:-$check_path}

    # FORCE_SETUP_SCENARIO scopes forced rebuilds to the scenario being
    # restarted so dependencies aren't rebuilt just because a parent restarts.
    local force_setup_applies=false
    if [[ "$force_setup" == "true" ]]; then
        if [[ -z "$force_setup_target" || "$force_setup_target" == "$scenario_name_from_path" ]]; then
            force_setup_applies=true
        fi
    fi

    if [[ "$force_setup_applies" == "true" ]]; then
        log::debug "FORCE_SETUP=true for scenario '$scenario_name_from_path', forcing setup verification"
        SETUP_REASONS+=("Forced rebuild (restart)")
    fi

    # Get service.json path
    local service_json="${check_path}/.vrooli/service.json"

    if [[ ! -f "$service_json" ]]; then
        log::debug "No service.json found at $check_path, assuming setup not needed"
        # Unless FORCE_SETUP is true
        [[ "$force_setup_applies" == "true" ]] && return 0
        return 1
    fi

    # Check if setup has a condition defined
    local has_condition
    has_condition=$(jq -r '.lifecycle.setup.condition // empty' "$service_json" 2>/dev/null)

    if [[ -z "$has_condition" ]]; then
        log::debug "No setup condition defined, setup not needed"
        # Unless FORCE_SETUP is true
        [[ "$force_setup_applies" == "true" ]] && return 0
        return 1
    fi

    # Check for checks array
    local checks
    checks=$(jq -c '.lifecycle.setup.condition.checks // []' "$service_json" 2>/dev/null)

    if [[ "$checks" == "[]" ]]; then
        log::debug "No setup checks defined, setup not needed"
        # Unless FORCE_SETUP is true
        [[ "$force_setup_applies" == "true" ]] && return 0
        return 1
    fi

    # Run each check
    local setup_needed="$force_setup_applies"  # Initialize to force_setup value
    local check_count=0
    
    while IFS= read -r check; do
        ((check_count++))
        
        local check_type
        check_type=$(echo "$check" | jq -r '.type // empty')
        
        if [[ -z "$check_type" ]]; then
            log::warn "Check #$check_count has no type, skipping"
            continue
        fi
        
        # Run the check inline (returns 0 if setup needed, 1 if not)
        if setup::check_condition "$check_path" "$check" "$check_type"; then
            log::debug "Check '$check_type' indicates setup is needed"
            
            # Add descriptive reason based on check type
            case "$check_type" in
                binaries)
                    local targets
                    targets=$(echo "$check" | jq -r '.targets[]?' 2>/dev/null | head -3 | paste -sd, -)
                    SETUP_REASONS+=("Missing binaries: $targets")
                    ;;
                cli)
                    local cmd
                    cmd=$(echo "$check" | jq -r '.command // "unknown"')
                    SETUP_REASONS+=("CLI not installed: $cmd")
                    ;;
                ui-bundle)
                    local bundle_path
                    bundle_path=$(echo "$check" | jq -r '.bundle_path // "ui/dist/index.html"')
                    SETUP_REASONS+=("UI bundle outdated: $bundle_path")
                    ;;
                resources)
                    SETUP_REASONS+=("Resources not populated")
                    ;;
                dependencies)
                    SETUP_REASONS+=("Dependencies not installed")
                    ;;
                data)
                    SETUP_REASONS+=("Data directory missing")
                    ;;
                files)
                    SETUP_REASONS+=("Required files missing")
                    ;;
                directories)
                    local targets
                    targets=$(echo "$check" | jq -r '.targets[]?' 2>/dev/null | head -3 | paste -sd, -)
                    SETUP_REASONS+=("Missing directories: $targets")
                    ;;
                *)
                    SETUP_REASONS+=("Check failed: $check_type")
                    ;;
            esac
            
            setup_needed=true
        else
            log::debug "Check '$check_type' passed"
        fi
    done <<< "$(echo "$checks" | jq -c '.[]' 2>/dev/null)"
    
    if [[ "$setup_needed" == "true" ]]; then
        log::debug "Setup needed based on condition checks"
        return 0
    else
        log::debug "All setup checks passed, no setup needed"
        return 1
    fi
}

#######################################
# Get completed setup steps from service.json for state tracking
# Returns:
#   JSON array of setup step names
#######################################
setup::get_steps_list() {
    local steps
    steps=$(json::get_value ".lifecycle.setup.steps" "[]" 2>/dev/null || echo "[]")
    
    if [[ "$steps" == "[]" ]]; then
        echo "[]"
        return 0
    fi
    
    echo "$steps" | jq -c '[.[].name // "unnamed"]' 2>/dev/null || echo "[]"
}

#######################################
# Mark setup as complete with markers for resource population
# This helps the condition checks know setup has been run
#######################################
setup::mark_complete() {
    local state_dir
    state_dir=$(setup::state_dir)
    mkdir -p "$state_dir"
    
    # Create a general setup completion marker
    local setup_steps
    setup_steps=$(setup::get_steps_list)
    
    cat > "$state_dir/.setup-complete" << EOF
{
  "setup_version": "2.0.0",
  "completed_at": "$(date -Iseconds)",
  "steps_completed": $setup_steps
}
EOF
    
    # Also create resource population marker if resources were populated
    # This is checked by the inline shell compatibility condition and native Go lifecycle checks
    if jq -e '.lifecycle.setup.steps[] | select(.name == "populate-resources" or .name == "add-data")' "${SERVICE_JSON:-${APP_ROOT}/.vrooli/service.json}" >/dev/null 2>&1; then
        touch "$state_dir/.resources-populated"
    fi
    
    log::debug "Setup marked as complete"
}
