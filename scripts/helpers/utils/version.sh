#!/usr/bin/env bash
# Handles getting and setting the version of the project.
# Assumes yq and jq are installed and available in PATH, typically handled by scripts/helpers/setup/common_deps.sh
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/var.sh"

# Finds the project version using the root package.json file.
# Assumes jq is available for parsing package.json.
version::get_project_version() {
    local version
    local root_pkg_json="${var_ROOT_DIR}/package.json"

    if [ ! -f "$root_pkg_json" ]; then
        log::error "Root package.json not found at $root_pkg_json"
        exit "$ERROR_FATAL"
    fi

    # Attempt to get version, capture jq's stderr separately for better error reporting
    local jq_stderr_file
    jq_stderr_file=$(mktemp)
    # shellcheck disable=SC2015 # We check exit status after
    version=$(jq -r '.version' "$root_pkg_json" 2> "$jq_stderr_file")
    local jq_exit_status=$?
    local jq_error_output
    jq_error_output=$(<"$jq_stderr_file")
    rm -f "$jq_stderr_file"

    if [ $jq_exit_status -ne 0 ]; then
        log::error "jq command failed to extract version from $root_pkg_json. Exit status: $jq_exit_status"
        if [ -n "$jq_error_output" ]; then
            log::error "jq stderr: $jq_error_output"
        fi
        exit "$ERROR_FATAL"
    fi

    if [ -z "$version" ] || [ "$version" == "null" ]; then # Check for empty string or literal "null"
        log::error "Version field not found, is empty, or is 'null' in $root_pkg_json."
        log::info "jq successfully parsed the file but found no valid '.version' field or it was null."
        exit "$ERROR_FATAL"
    fi
    echo "$version"
}

# Updates the project version in all package.json files.
# Assumes jq is available for robustly modifying package.json files.
version::set_package_json_version() {
    local current_version="$1" # Used for context/logging if needed, not directly by jq update logic.
    local target_version="$2"

    if [ -z "$current_version" ]; then
        log::error "No current version supplied to set_package_json_version (for context)."
        exit "$ERROR_USAGE"
    fi
    if [ -z "$target_version" ]; then
        log::error "No target version supplied to set_package_json_version."
        exit "$ERROR_USAGE"
    fi

    log::info "Attempting to update package.json files from version $current_version to $target_version using jq..."

    local pkg_file_paths=()
    # Find all package.json files in subdirectories of 'packages'
    # Ensure find doesn't error if 'packages' dir or sub-pkg.json files don't exist
    if [ -d "${var_ROOT_DIR}/packages" ]; then
        while IFS= read -r -d $'\0' file; do
            pkg_file_paths+=("$file")
        done < <(find "${var_ROOT_DIR}/packages" -name "package.json" -print0 2>/dev/null || true)
    else
        log::info "Directory ${var_ROOT_DIR}/packages not found. Skipping search for package.json files within it."
    fi


    # Add the root package.json file
    if [ -f "${var_ROOT_DIR}/package.json" ]; then
        pkg_file_paths+=("${var_ROOT_DIR}/package.json")
    else
        log::warn "Root package.json not found at ${var_ROOT_DIR}/package.json. Skipping it for version update."
    fi

    if [ ${#pkg_file_paths[@]} -eq 0 ]; then
        log::warn "No package.json files found to update."
        # This is not an error if the project structure doesn't use them or they are not relevant at this stage.
        # The function's purpose is to update them if they exist.
        return 0 
    fi
    
    local all_successful=true
    for pkg_file in "${pkg_file_paths[@]}"; do
        if [ ! -f "$pkg_file" ]; then # Should ideally not happen if find worked, but good safeguard
            log::warn "Package file $pkg_file listed but not found. Skipping."
            continue
        fi
        log::debug "Processing $pkg_file..."
        local tmp_file="${pkg_file}.jq.tmp" # Temporary file for jq output

        local file_current_version
        file_current_version=$(jq -r '.version' "$pkg_file" 2>/dev/null) # Check current version, suppress jq error for this read

        if [ "$file_current_version" == "$target_version" ]; then
            log::info "Version in $pkg_file is already $target_version. Skipping update."
            continue
        fi
        
        local jq_set_stderr_file
        jq_set_stderr_file=$(mktemp)
        # shellcheck disable=SC2015
        if jq --arg new_ver "$target_version" '.version = $new_ver' "$pkg_file" > "$tmp_file" 2>"$jq_set_stderr_file"; then
            if mv "$tmp_file" "$pkg_file"; then
                log::info "Successfully updated version in $pkg_file to $target_version."
            else
                log::error "Failed to move temporary file $tmp_file to $pkg_file."
                rm -f "$tmp_file" # Attempt to clean up
                all_successful=false
            fi
        else
            local jq_set_error_output
            jq_set_error_output=$(<"$jq_set_stderr_file")
            log::error "jq command failed to update version in $pkg_file (target: $target_version). Original file untouched."
            if [ -n "$jq_set_error_output" ]; then
                log::error "jq stderr: $jq_set_error_output"
            fi
            # jq might create an empty or partial tmp_file on error, clean it up.
            [ -e "$tmp_file" ] && rm -f "$tmp_file"
            all_successful=false
        fi
        rm -f "$jq_set_stderr_file"
    done

    if [ "$all_successful" = false ]; then
        log::error "One or more package.json files could not be updated. Please check logs."
        exit "$ERROR_FATAL"
    fi

    log::success "All found package.json files checked/updated."
}

# Updates the appVersion in k8s/chart/Chart.yaml.
version::set_helm_chart_app_version() {
    local target_version="$1"
    if [ -z "$target_version" ]; then
        log::error "No version supplied"
        exit "$ERROR_USAGE"
    fi

    local chart_file="${var_ROOT_DIR}/k8s/chart/Chart.yaml"
    if [ ! -f "$chart_file" ]; then
        log::error "Chart.yaml not found at $chart_file"
        exit "$ERROR_FATAL"
    fi

    # Use yq to update the appVersion.
    # The strenv(TARGET_VERSION) ensures the version is treated as a string.
    # The -i flag modifies the file in-place.
    if ! TARGET_VERSION="$target_version" yq eval '.appVersion = strenv(TARGET_VERSION)' -i "$chart_file"; then
        log::error "Failed to execute yq command to update appVersion in $chart_file"
        exit "$ERROR_FATAL"
    fi
    
    # Check if the update was successful using yq.
    if ! TARGET_VERSION="$target_version" yq eval --exit-status '.appVersion == strenv(TARGET_VERSION)' "$chart_file" > /dev/null; then
        log::error "Failed to update appVersion in $chart_file to $target_version using yq. Current value: $(yq eval '.appVersion' "$chart_file")"
        exit "$ERROR_FATAL"
    fi
    log::info "appVersion in $chart_file updated to $target_version"
}

# Updates the service tags in k8s/chart/values-prod.yaml.
version::set_values_prod_service_tags() {
    local target_version="$1"
    if [ -z "$target_version" ]; then
        log::error "No version supplied"
        exit "$ERROR_USAGE"
    fi

    local values_file="${var_ROOT_DIR}/k8s/chart/values-prod.yaml"
    if [ ! -f "$values_file" ]; then
        log::error "values-prod.yaml not found at $values_file"
        exit "$ERROR_FATAL"
    fi

    log::info "Updating service tags in $values_file to $target_version using yq"

    # Update tags for ui, server, and jobs services using yq
    local services_to_update=("ui" "server" "jobs")
    for service_name in "${services_to_update[@]}"; do
        log::info "Updating $service_name tag to $target_version..."
        if ! TARGET_VERSION="$target_version" yq eval "(.services.${service_name}.tag = strenv(TARGET_VERSION))" -i "$values_file"; then
            log::error "Failed to execute yq command to update $service_name tag in $values_file"
            exit "$ERROR_FATAL"
        fi

        # Validate update for the current service
        if ! TARGET_VERSION="$target_version" yq eval --exit-status ".services.${service_name}.tag == strenv(TARGET_VERSION)" "$values_file" > /dev/null; then
            log::error "Failed to update $service_name tag to $target_version in $values_file. Current value: $(yq eval ".services.${service_name}.tag" "$values_file")"
            # Attempt to log the relevant block for easier debugging with yq
            local service_block_content
            service_block_content=$(yq eval ".services.${service_name}" "$values_file" 2>/dev/null || echo "Could not retrieve block with yq")
            log::error "Relevant ${service_name} block content in $values_file:"
            echo "${service_block_content}" >&2
            exit "$ERROR_FATAL"
        fi
        log::info "$service_name tag updated successfully to $target_version."
    done

    log::info "Service tags in $values_file updated successfully to $target_version."
}

# Updates all package.json files and the helm chart appVersion.
version::set_project_version() {
    local target_version="$1"
    if [ -z "$target_version" ]; then
        log::error "No version supplied"
        exit "$ERROR_USAGE"
    fi
    
    local current_version
    current_version=$(version::get_project_version)
    if [ "$current_version" != "$target_version" ]; then
        log::info "Updating project version from $current_version to $target_version"
        version::set_package_json_version "$current_version" "$target_version"
        version::set_helm_chart_app_version "$target_version"
        version::set_values_prod_service_tags "$target_version"
    else
        log::info "Version $target_version is already set, skipping"
    fi
}
