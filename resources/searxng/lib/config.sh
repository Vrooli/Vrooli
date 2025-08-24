#!/usr/bin/env bash
# SearXNG Configuration Management
# Generate and manage SearXNG configuration files

# Source trash module for safe cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SEARXNG_LIB_DIR="${APP_ROOT}/resources/searxng/lib"
# shellcheck disable=SC1091
source "${SEARXNG_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Generate SearXNG settings.yml from template
#######################################
searxng::generate_config() {
    local template_file="$SEARXNG_LIB_DIR/../config/settings.yml.template"
    local config_file="$SEARXNG_DATA_DIR/settings.yml"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Configuration template not found: $template_file"
        return 1
    fi
    
    log::info "Generating SearXNG configuration..."
    
    # Ensure data directory exists
    if ! searxng::ensure_data_dir; then
        return 1
    fi
    
    # Process template and substitute variables
    local temp_config
    temp_config=$(mktemp)
    
    # Export all variables for envsubst
    export SEARXNG_INSTANCE_NAME SEARXNG_CONTACT_URL SEARXNG_DONATION_URL
    export SEARXNG_SAFE_SEARCH SEARXNG_AUTOCOMPLETE SEARXNG_DEFAULT_LANG
    export SEARXNG_SECRET_KEY SEARXNG_BASE_URL
    export SEARXNG_REQUEST_TIMEOUT SEARXNG_MAX_REQUEST_TIMEOUT
    export SEARXNG_POOL_CONNECTIONS SEARXNG_POOL_MAXSIZE
    export SEARXNG_REDIS_HOST SEARXNG_REDIS_PORT
    
    # Use envsubst if available, otherwise fall back to sed
    if command -v envsubst >/dev/null 2>&1; then
        envsubst < "$template_file" > "$temp_config"
    else
        # Fallback to sed for safer substitution
        cp "$template_file" "$temp_config"
        sed -i "s|\${SEARXNG_INSTANCE_NAME}|$SEARXNG_INSTANCE_NAME|g" "$temp_config"
        sed -i "s|\${SEARXNG_CONTACT_URL}|$SEARXNG_CONTACT_URL|g" "$temp_config"
        sed -i "s|\${SEARXNG_DONATION_URL}|$SEARXNG_DONATION_URL|g" "$temp_config"
        sed -i "s|\${SEARXNG_SAFE_SEARCH}|$SEARXNG_SAFE_SEARCH|g" "$temp_config"
        sed -i "s|\${SEARXNG_AUTOCOMPLETE}|$SEARXNG_AUTOCOMPLETE|g" "$temp_config"
        sed -i "s|\${SEARXNG_DEFAULT_LANG}|$SEARXNG_DEFAULT_LANG|g" "$temp_config"
        sed -i "s|\${SEARXNG_SECRET_KEY}|$SEARXNG_SECRET_KEY|g" "$temp_config"
        sed -i "s|\${SEARXNG_BASE_URL}|$SEARXNG_BASE_URL|g" "$temp_config"
        sed -i "s|\${SEARXNG_REQUEST_TIMEOUT}|$SEARXNG_REQUEST_TIMEOUT|g" "$temp_config"
        sed -i "s|\${SEARXNG_MAX_REQUEST_TIMEOUT}|$SEARXNG_MAX_REQUEST_TIMEOUT|g" "$temp_config"
        sed -i "s|\${SEARXNG_POOL_CONNECTIONS}|$SEARXNG_POOL_CONNECTIONS|g" "$temp_config"
        sed -i "s|\${SEARXNG_POOL_MAXSIZE}|$SEARXNG_POOL_MAXSIZE|g" "$temp_config"
        sed -i "s|\${SEARXNG_REDIS_HOST}|$SEARXNG_REDIS_HOST|g" "$temp_config"
        sed -i "s|\${SEARXNG_REDIS_PORT}|$SEARXNG_REDIS_PORT|g" "$temp_config"
    fi
    
    # Copy to final location (handle permission issues with docker if needed)
    if cp "$temp_config" "$config_file" 2>/dev/null; then
        log::success "SearXNG configuration generated: $config_file"
    elif cat "$temp_config" > "$config_file" 2>/dev/null; then
        log::success "SearXNG configuration generated: $config_file"
    else
        # Use docker to copy file if permissions are an issue
        log::info "Using Docker to handle file permissions..."
        docker run --rm -v "$temp_config:/source" -v "$SEARXNG_DATA_DIR:/target" alpine sh -c "cp /source /target/settings.yml && chmod 666 /target/settings.yml" 2>/dev/null
        if [[ -f "$config_file" ]]; then
            log::success "SearXNG configuration generated via Docker: $config_file"
        else
            log::error "Failed to generate SearXNG configuration"
            trash::safe_remove "$temp_config" --temp
            return 1
        fi
    fi
    
    # Clean up temp file
    trash::safe_remove "$temp_config" --temp
    
    # Generate additional config files
    searxng::generate_limiter_config
    searxng::generate_uwsgi_config
    
    return 0
}

#######################################
# Generate rate limiter configuration
#######################################
searxng::generate_limiter_config() {
    if [[ "$SEARXNG_LIMITER_ENABLED" != "yes" ]]; then
        return 0
    fi
    
    local template_file="$SEARXNG_LIB_DIR/../docker/limiter.toml.template"
    local config_file="$SEARXNG_DATA_DIR/limiter.toml"
    
    if [[ ! -f "$template_file" ]]; then
        log::warn "Limiter template not found: $template_file"
        return 0
    fi
    
    log::info "Generating rate limiter configuration..."
    
    # Process template and substitute variables
    local temp_config
    temp_config=$(mktemp)
    
    # Export variable for envsubst
    export SEARXNG_RATE_LIMIT
    
    # Use envsubst if available, otherwise use sed
    if command -v envsubst >/dev/null 2>&1; then
        envsubst < "$template_file" > "$temp_config"
    else
        # Fallback to sed
        cp "$template_file" "$temp_config"
        sed -i "s|\${SEARXNG_RATE_LIMIT}|$SEARXNG_RATE_LIMIT|g" "$temp_config"
    fi
    
    if cp "$temp_config" "$config_file" 2>/dev/null || cat "$temp_config" > "$config_file" 2>/dev/null; then
        log::success "Rate limiter configuration generated: $config_file"
    else
        # Use docker to copy file if permissions are an issue
        docker run --rm -v "$temp_config:/source" -v "$SEARXNG_DATA_DIR:/target" alpine sh -c "cp /source /target/limiter.toml && chmod 666 /target/limiter.toml" 2>/dev/null
        if [[ -f "$config_file" ]]; then
            log::success "Rate limiter configuration generated via Docker: $config_file"
        else
            log::warn "Failed to generate rate limiter configuration"
        fi
    fi
    trash::safe_remove "$temp_config" --temp
    return 0
}

#######################################
# Generate uWSGI configuration for Searxng
#######################################
searxng::generate_uwsgi_config() {
    local config_file="$SEARXNG_DATA_DIR/uwsgi.ini"
    
    log::info "Generating uWSGI configuration..."
    
    # Create uwsgi.ini with proper HTTP binding
    cat > "$config_file" << 'EOF'
[uwsgi]
# SearXNG uwsgi configuration

# Who will run the code
uid = searxng
gid = searxng

# HTTP socket - bind to port 8080
http = :8080

# Number of workers (uses number of CPU cores)
workers = %k
threads = 4

# The right granted on the created socket
chmod-socket = 666

# Plugin to use and interpretor config
single-interpreter = true
master = true
plugin = python3
lazy-apps = true
enable-threads = true

# Module to import
module = searx.webapp

# Virtualenv and python path
pythonpath = /usr/local/searxng/
chdir = /usr/local/searxng/searx/

# Disable request logging
disable-logging = false
log-5xx = true

# Set process name
auto-procname = true

# Increase buffer size
buffer-size = 8192

# Max request size
post-buffering = 8192
EOF
    
    # Fix permissions if needed
    chmod 644 "$config_file" 2>/dev/null || true
    
    log::success "uWSGI configuration generated: $config_file"
    return 0
}

#######################################
# Update SearXNG engines configuration
# Arguments:
#   $1 - comma-separated list of engines
#######################################
searxng::update_engines() {
    local engines="$1"
    local config_file="$SEARXNG_DATA_DIR/settings.yml"
    
    if [[ ! -f "$config_file" ]]; then
        log::error "SearXNG configuration not found: $config_file"
        return 1
    fi
    
    if [[ -z "$engines" ]]; then
        log::error "Engine list cannot be empty"
        return 1
    fi
    
    log::info "Updating SearXNG engines: $engines"
    
    # Backup current configuration
    cp "$config_file" "${config_file}.backup"
    
    # This is a simplified implementation
    # In a full implementation, you'd want to properly parse and modify the YAML
    log::warn "Engine configuration update requires manual editing of $config_file"
    log::info "Supported engines: google, bing, duckduckgo, startpage, wikipedia"
    
    return 0
}

#######################################
# Update SearXNG search settings
#######################################
searxng::update_search_settings() {
    local config_file="$SEARXNG_DATA_DIR/settings.yml"
    
    if [[ ! -f "$config_file" ]]; then
        log::error "SearXNG configuration not found: $config_file"
        return 1
    fi
    
    log::info "Updating search settings..."
    
    # This would require a proper YAML parser for production use
    # For now, regenerate the entire configuration
    searxng::generate_config
}

#######################################
# Show current configuration
#######################################
searxng::show_config() {
    log::header "SearXNG Configuration"
    
    echo "Basic Settings:"
    echo "  Instance Name: $SEARXNG_INSTANCE_NAME"
    echo "  Base URL: $SEARXNG_BASE_URL"
    echo "  Port: $SEARXNG_PORT"
    echo "  Bind Address: $SEARXNG_BIND_ADDRESS"
    echo
    
    echo "Search Settings:"
    echo "  Default Engines: $SEARXNG_DEFAULT_ENGINES"
    echo "  Safe Search: $SEARXNG_SAFE_SEARCH"
    echo "  Autocomplete: $SEARXNG_AUTOCOMPLETE"
    echo "  Default Language: $SEARXNG_DEFAULT_LANG"
    echo
    
    echo "Performance Settings:"
    echo "  Request Timeout: ${SEARXNG_REQUEST_TIMEOUT}s"
    echo "  Max Request Timeout: ${SEARXNG_MAX_REQUEST_TIMEOUT}s"
    echo "  Pool Connections: $SEARXNG_POOL_CONNECTIONS"
    echo "  Pool Max Size: $SEARXNG_POOL_MAXSIZE"
    echo
    
    echo "Security & Limits:"
    echo "  Secret Key Length: ${#SEARXNG_SECRET_KEY} characters"
    echo "  Rate Limiting: $SEARXNG_LIMITER_ENABLED"
    if [[ "$SEARXNG_LIMITER_ENABLED" == "yes" ]]; then
        echo "  Rate Limit: $SEARXNG_RATE_LIMIT requests/minute"
    fi
    echo "  Public Access: $SEARXNG_ENABLE_PUBLIC_ACCESS"
    echo
    
    echo "Storage & Caching:"
    echo "  Data Directory: $SEARXNG_DATA_DIR"
    echo "  Redis Caching: $SEARXNG_ENABLE_REDIS"
    if [[ "$SEARXNG_ENABLE_REDIS" == "yes" ]]; then
        echo "  Redis Host: $SEARXNG_REDIS_HOST"
        echo "  Redis Port: $SEARXNG_REDIS_PORT"
    fi
    echo
    
    # Show file locations
    echo "Configuration Files:"
    if [[ -f "$SEARXNG_DATA_DIR/settings.yml" ]]; then
        echo "  settings.yml: ✅ Present"
    else
        echo "  settings.yml: ❌ Missing"
    fi
    
    if [[ "$SEARXNG_LIMITER_ENABLED" == "yes" ]]; then
        if [[ -f "$SEARXNG_DATA_DIR/limiter.toml" ]]; then
            echo "  limiter.toml: ✅ Present"
        else
            echo "  limiter.toml: ❌ Missing"
        fi
    fi
}

#######################################
# Validate configuration files
#######################################
searxng::validate_config_files() {
    local config_file="$SEARXNG_DATA_DIR/settings.yml"
    local errors=0
    
    log::info "Validating SearXNG configuration files..."
    
    # Check main configuration
    if [[ ! -f "$config_file" ]]; then
        log::error "Main configuration file missing: $config_file"
        ((errors++))
    else
        # Basic YAML syntax check (if yq is available)
        if command -v yq >/dev/null 2>&1; then
            if ! yq eval '.' "$config_file" >/dev/null 2>&1; then
                log::error "Invalid YAML syntax in: $config_file"
                ((errors++))
            else
                log::success "Configuration syntax is valid"
            fi
        else
            log::info "YAML validation skipped (yq not available)"
        fi
    fi
    
    # Check required fields in configuration
    if [[ -f "$config_file" ]]; then
        local required_fields=("server.secret_key" "server.base_url")
        for field in "${required_fields[@]}"; do
            if command -v yq >/dev/null 2>&1; then
                if ! yq eval ".$field" "$config_file" >/dev/null 2>&1; then
                    log::error "Required field missing: $field"
                    ((errors++))
                fi
            fi
        done
    fi
    
    # Check limiter configuration if enabled
    if [[ "$SEARXNG_LIMITER_ENABLED" == "yes" ]]; then
        local limiter_file="$SEARXNG_DATA_DIR/limiter.toml"
        if [[ ! -f "$limiter_file" ]]; then
            log::warn "Rate limiter enabled but configuration missing: $limiter_file"
        fi
    fi
    
    if [[ $errors -eq 0 ]]; then
        log::success "Configuration validation passed"
        return 0
    else
        log::error "Configuration validation failed with $errors errors"
        return 1
    fi
}

#######################################
# Export configuration for external use
# Arguments:
#   $1 - output file path
#######################################
searxng::export_config() {
    local output_file="$1"
    
    if [[ -z "$output_file" ]]; then
        log::error "Output file path is required"
        return 1
    fi
    
    log::info "Exporting SearXNG configuration..."
    
    {
        echo "# SearXNG Configuration Export"
        echo "# Generated: $(date)"
        echo
        searxng::show_config
    } > "$output_file"
    
    log::success "Configuration exported to: $output_file"
}

#######################################
# Reset configuration to defaults
#######################################
searxng::reset_config() {
    log::info "Resetting SearXNG configuration to defaults..."
    
    # Backup existing configuration
    if [[ -f "$SEARXNG_DATA_DIR/settings.yml" ]]; then
        local backup_file="$SEARXNG_DATA_DIR/settings.yml.backup.$(date +%Y%m%d-%H%M%S)"
        cp "$SEARXNG_DATA_DIR/settings.yml" "$backup_file"
        log::info "Existing configuration backed up to: $backup_file"
    fi
    
    # Regenerate configuration
    if searxng::generate_config; then
        log::success "Configuration reset to defaults"
        return 0
    else
        log::error "Failed to reset configuration"
        return 1
    fi
}