#!/usr/bin/env bash
set -euo pipefail

# Ollama AI Resource Setup and Management
# This script handles installation, configuration, and management of Ollama

DESCRIPTION="Install and manage Ollama AI resource"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/.."

# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Ollama configuration
readonly OLLAMA_PORT="${OLLAMA_CUSTOM_PORT:-$(resources::get_default_port "ollama")}"
readonly OLLAMA_BASE_URL="http://localhost:${OLLAMA_PORT}"
readonly OLLAMA_SERVICE_NAME="ollama"
readonly OLLAMA_INSTALL_DIR="/usr/local/bin"
readonly OLLAMA_USER="ollama"

# Model catalog with metadata
# Format: ["model:tag"]="size_gb|capabilities|description"
declare -A MODEL_CATALOG=(
    # Current/Recommended Models
    ["llama3.1:8b"]="4.9|general,chat,reasoning|Latest general-purpose model from Meta"
    ["deepseek-r1:8b"]="4.7|reasoning,math,code,chain-of-thought|Advanced reasoning model with explicit thinking process"
    ["qwen2.5-coder:7b"]="4.1|code,programming,debugging|Superior code generation model, replaces CodeLlama"
    
    # Alternative Sizes
    ["llama3.3:8b"]="4.9|general,chat,reasoning|Very latest from Meta (Dec 2024)"
    ["deepseek-r1:14b"]="8.1|reasoning,math,code,chain-of-thought|Larger reasoning model for complex problems"
    ["deepseek-r1:1.5b"]="0.9|reasoning,lightweight|Smallest reasoning model for resource-constrained environments"
    
    # Specialized Models
    ["phi-4:14b"]="8.2|general,multilingual,math,function-calling|Microsoft's efficient model with multilingual support"
    ["qwen2.5:14b"]="8.0|general,multilingual,reasoning|Strong multilingual model with excellent reasoning"
    ["mistral-small:22b"]="13.2|general,balanced,multilingual|Excellent balanced performance model"
    
    # Code-Focused Models
    ["qwen2.5-coder:32b"]="19.1|code,programming,architecture|Large code model for complex projects"
    ["deepseek-coder:6.7b"]="3.8|code,programming,documentation|Specialized programming model"
    
    # Vision/Multimodal
    ["llava:13b"]="7.3|vision,image-understanding,multimodal|Image understanding and visual reasoning"
    ["qwen2-vl:7b"]="4.2|vision,image-understanding,multimodal|Vision-language model for image analysis"
    
    # Legacy Models (for reference)
    ["llama2:7b"]="3.8|general,legacy|Legacy model, superseded by llama3.1"
    ["codellama:7b"]="3.8|code,legacy|Legacy code model, superseded by qwen2.5-coder"
)

# Default models to install (updated for 2025)
readonly DEFAULT_MODELS=(
    "llama3.1:8b"      # General purpose - proven and reliable
    "deepseek-r1:8b"   # Advanced reasoning - breakthrough model for complex thinking
    "qwen2.5-coder:7b" # Modern code generation - superior to CodeLlama
)

#######################################
# Parse command line arguments
#######################################
ollama::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|models|available|info" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Ollama appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "models" \
        --flag "m" \
        --desc "Comma-separated list of models to install (empty = default models)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "skip-models" \
        --desc "Skip model installation during setup" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    if args::is_asking_for_help "$@"; then
        ollama::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export MODELS_INPUT=$(args::get "models")
    export SKIP_MODELS=$(args::get "skip-models")
}

#######################################
# Display usage information
#######################################
ollama::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install Ollama with default models (llama3.1:8b, deepseek-r1:8b, qwen2.5-coder:7b)"
    echo "  $0 --action install --skip-models               # Install Ollama without models"
    echo "  $0 --action install --models 'llama3.1:8b,phi-4:14b'  # Install with specific models"
    echo "  $0 --action available                           # Show available models from catalog"
    echo "  $0 --action status                               # Check Ollama status"
    echo "  $0 --action models                               # List installed models"
    echo "  $0 --action uninstall                           # Remove Ollama"
}

#######################################
# Model catalog utility functions
#######################################

#######################################
# Get model information from catalog
# Arguments:
#   $1 - model name (e.g., "llama3.1:8b")
# Outputs: size_gb|capabilities|description
#######################################
ollama::get_model_info() {
    local model="$1"
    echo "${MODEL_CATALOG[$model]:-unknown|unknown|Model not found in catalog}"
}

#######################################
# Get model size from catalog
# Arguments:
#   $1 - model name
# Outputs: size in GB
#######################################
ollama::get_model_size() {
    local model="$1"
    local info
    info=$(ollama::get_model_info "$model")
    echo "$info" | cut -d'|' -f1
}

#######################################
# Check if model is in catalog
# Arguments:
#   $1 - model name
# Returns: 0 if in catalog, 1 otherwise
#######################################
ollama::is_model_known() {
    local model="$1"
    [[ -n "${MODEL_CATALOG[$model]:-}" ]]
}

#######################################
# Display available models from catalog
#######################################
ollama::show_available_models() {
    log::header "📚 Available Models (from catalog)"
    
    printf "%-20s %-8s %-30s %s\n" "MODEL" "SIZE" "CAPABILITIES" "DESCRIPTION"
    printf "%-20s %-8s %-30s %s\n" "----" "----" "----" "----"
    
    # Sort models by category and size
    local sorted_models=()
    
    # Current/Recommended models first
    for model in "llama3.1:8b" "deepseek-r1:8b" "qwen2.5-coder:7b"; do
        if [[ -n "${MODEL_CATALOG[$model]:-}" ]]; then
            sorted_models+=("$model")
        fi
    done
    
    # Then other models alphabetically
    for model in "${!MODEL_CATALOG[@]}"; do
        if [[ ! " ${sorted_models[*]} " =~ " $model " ]]; then
            sorted_models+=("$model")
        fi
    done
    
    for model in "${sorted_models[@]}"; do
        local info="${MODEL_CATALOG[$model]}"
        local size=$(echo "$info" | cut -d'|' -f1)
        local capabilities=$(echo "$info" | cut -d'|' -f2)
        local description=$(echo "$info" | cut -d'|' -f3)
        
        # Highlight default models
        local marker=""
        if [[ " ${DEFAULT_MODELS[*]} " =~ " $model " ]]; then
            marker="✅"
        elif [[ "$capabilities" == *"legacy"* ]]; then
            marker="⚠️ "
        fi
        
        printf "%-20s %-8s %-30s %s %s\n" "$model" "${size}GB" "$capabilities" "$marker" "$description"
    done
    
    echo
    log::info "✅ = Default models    ⚠️  = Legacy models"
    log::info "Total default models size: $(ollama::calculate_default_size)GB"
}

#######################################
# Calculate total size of default models
#######################################
ollama::calculate_default_size() {
    local total=0
    for model in "${DEFAULT_MODELS[@]}"; do
        local size
        size=$(ollama::get_model_size "$model")
        if [[ "$size" =~ ^[0-9]+\.?[0-9]*$ ]]; then
            total=$(echo "$total + $size" | bc -l 2>/dev/null || echo "$total")
        fi
    done
    printf "%.1f" "$total"
}

#######################################
# Validate model list against catalog
# Arguments:
#   $@ - list of models to validate
# Returns: 0 if all valid, 1 if any invalid
#######################################
ollama::validate_model_list() {
    local models=("$@")
    local invalid_models=()
    local total_size=0
    
    for model in "${models[@]}"; do
        if ! ollama::is_model_known "$model"; then
            invalid_models+=("$model")
        else
            local size
            size=$(ollama::get_model_size "$model")
            if [[ "$size" =~ ^[0-9]+\.?[0-9]*$ ]]; then
                total_size=$(echo "$total_size + $size" | bc -l 2>/dev/null || echo "$total_size")
            fi
        fi
    done
    
    if [[ ${#invalid_models[@]} -gt 0 ]]; then
        log::warn "Unknown models (not in catalog): ${invalid_models[*]}"
        log::info "Use 'ollama::show_available_models' to see available models"
        return 1
    fi
    
    log::info "Validated ${#models[@]} models, total size: $(printf "%.1f" "$total_size")GB"
    return 0
}

#######################################
# Check if Ollama is installed
# Returns: 0 if installed, 1 otherwise
#######################################
ollama::is_installed() {
    resources::binary_exists "ollama"
}

#######################################
# Check if Ollama service is running
# Returns: 0 if running, 1 otherwise
#######################################
ollama::is_running() {
    resources::is_service_running "$OLLAMA_PORT"
}

#######################################
# Check if Ollama API is responsive
# Returns: 0 if responsive, 1 otherwise
#######################################
ollama::is_healthy() {
    resources::check_http_health "$OLLAMA_BASE_URL" "/api/tags"
}

#######################################
# Install Ollama binary with rollback support
#######################################
ollama::install_binary() {
    if ollama::is_installed && [[ "$FORCE" != "yes" ]]; then
        log::info "Ollama is already installed (use --force yes to reinstall)"
        return 0
    fi
    
    log::info "Installing Ollama..."
    
    # Download and install using official installer
    local install_script
    install_script=$(mktemp) || {
        log::error "Failed to create temporary file"
        return 1
    }
    
    # Add rollback action for cleanup
    resources::add_rollback_action \
        "Clean up Ollama installer script" \
        "rm -f \"$install_script\"" \
        1
    
    if ! resources::download_file "https://ollama.com/install.sh" "$install_script"; then
        log::error "Failed to download Ollama installer"
        return 1
    fi
    
    # Verify the installer is valid
    if [[ ! -s "$install_script" ]]; then
        log::error "Downloaded installer is empty"
        return 1
    fi
    
    # Make executable and run
    chmod +x "$install_script"
    
    if resources::can_sudo; then
        if sudo bash "$install_script"; then
            log::success "Ollama installer completed successfully"
        else
            log::error "Ollama installer failed"
            return 1
        fi
    else
        log::error "Sudo privileges required to install Ollama"
        return 1
    fi
    
    # Clean up installer
    rm -f "$install_script"
    
    # Verify installation
    if ollama::is_installed; then
        log::success "Ollama binary installed successfully"
        
        # Add rollback action for binary removal
        resources::add_rollback_action \
            "Remove Ollama binary" \
            "sudo rm -f /usr/local/bin/ollama" \
            20
        
        return 0
    else
        log::error "Ollama installation failed - binary not found after installation"
        return 1
    fi
}

#######################################
# Create Ollama user if it doesn't exist
#######################################
ollama::create_user() {
    if id "$OLLAMA_USER" &>/dev/null; then
        log::info "User $OLLAMA_USER already exists"
        return 0
    fi
    
    if ! resources::can_sudo; then
        log::error "Sudo privileges required to create $OLLAMA_USER user"
        return 1
    fi
    
    log::info "Creating $OLLAMA_USER user..."
    
    if sudo useradd -r -s /bin/false -d /usr/share/ollama -c "Ollama service user" "$OLLAMA_USER"; then
        log::success "User $OLLAMA_USER created"
        
        # Add rollback action for user removal
        resources::add_rollback_action \
            "Remove Ollama user" \
            "sudo userdel $OLLAMA_USER 2>/dev/null || true" \
            15
        
        return 0
    else
        log::error "Failed to create user $OLLAMA_USER"
        return 1
    fi
}

#######################################
# Install Ollama systemd service with rollback support
#######################################
ollama::install_service() {
    # Check if systemd service already exists (installed by official installer)
    if systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service"; then
        log::info "Ollama systemd service already exists (installed by official installer)"
        
        # Just ensure it's enabled
        if resources::can_sudo; then
            sudo systemctl enable "$OLLAMA_SERVICE_NAME" 2>/dev/null || true
        fi
        
        return 0
    fi
    
    # Only create service if it doesn't exist
    local service_content="[Unit]
Description=Ollama Service
After=network-online.target

[Service]
ExecStart=/usr/local/bin/ollama serve
User=$OLLAMA_USER
Group=$OLLAMA_USER
Restart=always
RestartSec=3
Environment=\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"
Environment=\"OLLAMA_HOST=0.0.0.0:$OLLAMA_PORT\"

[Install]
WantedBy=default.target"
    
    log::info "Installing Ollama systemd service..."
    
    if ! resources::can_sudo; then
        resources::handle_error \
            "Sudo privileges required to install systemd service" \
            "permission" \
            "Run 'sudo -v' to authenticate and retry"
        return 1
    fi
    
    if resources::install_systemd_service "$OLLAMA_SERVICE_NAME" "$service_content"; then
        log::success "Ollama systemd service installed successfully"
        
        # Add rollback action for service removal
        resources::add_rollback_action \
            "Remove Ollama systemd service" \
            "sudo systemctl stop $OLLAMA_SERVICE_NAME 2>/dev/null || true; sudo systemctl disable $OLLAMA_SERVICE_NAME 2>/dev/null || true; sudo rm -f /etc/systemd/system/${OLLAMA_SERVICE_NAME}.service; sudo systemctl daemon-reload" \
            18
        
        return 0
    else
        resources::handle_error \
            "Failed to install Ollama systemd service" \
            "system" \
            "Check systemd status and permissions: systemctl status systemd"
        return 1
    fi
}

#######################################
# Start Ollama service with enhanced error handling
#######################################
ollama::start() {
    if ollama::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Ollama is already running on port $OLLAMA_PORT"
        return 0
    fi
    
    log::info "Starting Ollama service..."
    
    # Check if service exists
    if ! systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service"; then
        resources::handle_error \
            "Ollama systemd service not found" \
            "system" \
            "Install Ollama first with: $0 --action install"
        return 1
    fi
    
    # Validate prerequisites
    if ! resources::can_sudo; then
        resources::handle_error \
            "Sudo privileges required to start Ollama service" \
            "permission" \
            "Run 'sudo -v' to authenticate and retry"
        return 1
    fi
    
    # Check if port is already in use by another process
    if resources::is_service_running "$OLLAMA_PORT" && ! resources::is_service_active "$OLLAMA_SERVICE_NAME"; then
        resources::handle_error \
            "Port $OLLAMA_PORT is already in use by another process" \
            "system" \
            "Find and stop the process using: sudo lsof -ti:$OLLAMA_PORT | xargs sudo kill -9"
        return 1
    fi
    
    # Start the service
    if ! resources::start_service "$OLLAMA_SERVICE_NAME"; then
        resources::handle_error \
            "Failed to start Ollama systemd service" \
            "system" \
            "Check service logs: journalctl -u $OLLAMA_SERVICE_NAME -n 20"
        return 1
    fi
    
    # Wait for service to become available with progress indication
    log::info "Waiting for Ollama to start (this may take 10-30 seconds)..."
    if resources::wait_for_service "Ollama" "$OLLAMA_PORT" 30; then
        # Additional health check with retry
        sleep 2
        local health_attempts=3
        local health_success=false
        
        for ((i=1; i<=health_attempts; i++)); do
            if ollama::is_healthy; then
                health_success=true
                break
            fi
            
            if [[ $i -lt $health_attempts ]]; then
                log::info "Health check attempt $i failed, retrying in 3 seconds..."
                sleep 3
            fi
        done
        
        if [[ "$health_success" == "true" ]]; then
            log::success "✅ Ollama is running and healthy on port $OLLAMA_PORT"
            return 0
        else
            log::warn "⚠️  Ollama service started but API is not responding"
            log::info "Service may still be initializing. Check status with: $0 --action status"
            log::info "Or check logs with: journalctl -u $OLLAMA_SERVICE_NAME -f"
            return 0
        fi
    else
        resources::handle_error \
            "Ollama service failed to start within 30 seconds" \
            "system" \
            "Check service status: systemctl status $OLLAMA_SERVICE_NAME; journalctl -u $OLLAMA_SERVICE_NAME -n 20"
        return 1
    fi
}

#######################################
# Stop Ollama service
#######################################
ollama::stop() {
    resources::stop_service "$OLLAMA_SERVICE_NAME"
}

#######################################
# Restart Ollama service
#######################################
ollama::restart() {
    log::info "Restarting Ollama service..."
    ollama::stop
    sleep 2
    ollama::start
}

#######################################
# Parse model list from input
# Outputs: space-separated list of models
#######################################
ollama::parse_models() {
    local input="$MODELS_INPUT"
    
    if [[ -z "$input" ]]; then
        echo "${DEFAULT_MODELS[*]}"
    else
        # Parse comma-separated list and clean up
        echo "$input" | tr ',' ' ' | tr -s ' '
    fi
}

#######################################
# Pull a single model
# Arguments:
#   $1 - model name
#######################################
ollama::pull_model() {
    local model="$1"
    
    log::info "Pulling model: $model"
    
    if ollama pull "$model"; then
        log::success "Model $model pulled successfully"
    else
        log::error "Failed to pull model: $model"
        return 1
    fi
}

#######################################
# Install default models with enhanced error handling and rollback
#######################################
ollama::install_models() {
    if [[ "$SKIP_MODELS" == "yes" ]]; then
        log::info "Skipping model installation (--skip-models specified)"
        return 0
    fi
    
    log::header "📦 Installing Ollama Models"
    
    # Validate Ollama API availability with retries
    local api_check_attempts=3
    local api_available=false
    
    for ((i=1; i<=api_check_attempts; i++)); do
        if ollama::is_healthy; then
            api_available=true
            break
        fi
        
        if [[ $i -lt $api_check_attempts ]]; then
            log::info "Ollama API not ready (attempt $i/$api_check_attempts), waiting 5 seconds..."
            sleep 5
        fi
    done
    
    if [[ "$api_available" != "true" ]]; then
        resources::handle_error \
            "Ollama API is not available after $api_check_attempts attempts" \
            "system" \
            "Check service status: systemctl status $OLLAMA_SERVICE_NAME; journalctl -u $OLLAMA_SERVICE_NAME -n 20"
        return 1
    fi
    
    # Parse and validate models list
    local models
    models=$(ollama::parse_models)
    
    if [[ -z "$models" ]]; then
        log::warn "No models specified for installation"
        return 0
    fi
    
    # Convert to array for validation
    local models_array=()
    IFS=' ' read -ra models_array <<< "$models"
    
    # Validate against catalog and show information
    log::info "Validating models against catalog..."
    if ! ollama::validate_model_list "${models_array[@]}"; then
        log::error "Model validation failed"
        log::info "Run '$0 --action available' to see available models"
        return 1
    fi
    
    # Display model information
    log::info "📦 Models to install with catalog information:"
    for model in "${models_array[@]}"; do
        if ollama::is_model_known "$model"; then
            local info
            info=$(ollama::get_model_info "$model")
            local size=$(echo "$info" | cut -d'|' -f1)
            local capabilities=$(echo "$info" | cut -d'|' -f2)
            local description=$(echo "$info" | cut -d'|' -f3)
            log::info "  • $model (${size}GB) - $description"
        else
            log::info "  • $model (size unknown) - Not in catalog"
        fi
    done
    
    # Check available disk space (models can be several GB each)
    local available_space_gb
    if available_space_gb=$(df "$HOME" --output=avail --block-size=1G | tail -n1 | tr -d ' '); then
        if [[ $available_space_gb -lt 10 ]]; then
            log::warn "⚠️  Low disk space detected: ${available_space_gb}GB available"
            log::info "Each model typically requires 2-8GB. Consider freeing up space if downloads fail."
        fi
    fi
    
    local success_count=0
    local total_count=0
    local installed_models=()
    local failed_models=()
    
    # Track installed models for rollback
    for model in $models; do
        total_count=$((total_count + 1))
        
        log::info "[$total_count/$(echo $models | wc -w)] Installing model: $model"
        
        # Check if model already exists
        if ollama list 2>/dev/null | grep -q "^$model"; then
            log::info "Model $model already installed, skipping"
            success_count=$((success_count + 1))
            continue
        fi
        
        # Install model with progress
        if ollama::pull_model "$model"; then
            success_count=$((success_count + 1))
            installed_models+=("$model")
            
            # Add rollback action for model removal (low priority since models are large)
            resources::add_rollback_action \
                "Remove installed model: $model" \
                "ollama rm \\\"$model\\\" 2>/dev/null || true" \
                5
            
            log::success "✅ Model $model installed successfully"
        else
            failed_models+=("$model")
            log::error "❌ Failed to install model: $model"
        fi
        
        # Brief pause between models to avoid overwhelming the system
        if [[ $total_count -lt $(echo $models | wc -w) ]]; then
            sleep 1
        fi
    done
    
    # Summary report
    log::info "Model installation summary:"
    log::info "  • Successfully installed: $success_count/$total_count models"
    
    if [[ ${#installed_models[@]} -gt 0 ]]; then
        log::success "  • Installed models: ${installed_models[*]}"
    fi
    
    if [[ ${#failed_models[@]} -gt 0 ]]; then
        log::warn "  • Failed models: ${failed_models[*]}"
        log::info "  • Retry failed models with: ollama pull <model-name>"
    fi
    
    # Validate at least one model was installed successfully
    if [[ $success_count -eq 0 ]]; then
        resources::handle_error \
            "No models were installed successfully" \
            "network" \
            "Check internet connection and try installing models manually: ollama pull llama3.1:8b"
        return 1
    fi
    
    # Show final model list
    echo
    log::info "📋 Available models:"
    ollama list 2>/dev/null || log::warn "Could not list models (this is usually temporary)"
    
    return 0
}

#######################################
# List installed models
#######################################
ollama::list_models() {
    if ! ollama::is_healthy; then
        log::error "Ollama API is not available"
        return 1
    fi
    
    log::header "📋 Installed Ollama Models"
    
    if system::is_command "ollama"; then
        ollama list
    else
        log::error "Ollama binary not found"
        return 1
    fi
}

#######################################
# Update Ollama configuration
#######################################
ollama::update_config() {
    # Create properly formatted JSON in a single line for jq compatibility
    local additional_config='{"models":{"defaultModel":"llama3.1:8b","supportsFunctionCalling":true},"api":{"version":"v1","modelsEndpoint":"/api/tags","chatEndpoint":"/api/chat","generateEndpoint":"/api/generate"}}'
    
    resources::update_config "ai" "ollama" "$OLLAMA_BASE_URL" "$additional_config"
}

#######################################
# Complete Ollama installation with comprehensive error handling
#######################################
ollama::install() {
    log::header "🤖 Installing Ollama"
    
    # Start rollback context
    resources::start_rollback_context "install_ollama"
    
    # Check if already installed and running
    if ollama::is_installed && ollama::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Ollama is already installed and running"
        log::info "Use --force yes to reinstall, or --action status to check current state"
        return 0
    fi
    
    # Validate prerequisites
    if ! resources::can_sudo; then
        resources::handle_error \
            "Ollama installation requires sudo privileges for system service management" \
            "permission" \
            "Run 'sudo -v' to authenticate, then retry installation"
        return 1
    fi
    
    # Validate port assignment
    if ! resources::validate_port "ollama" "$OLLAMA_PORT" "$FORCE"; then
        log::error "Port validation failed for Ollama"
        log::info "You can set a custom port with: export OLLAMA_CUSTOM_PORT=<port>"
        return 1
    fi
    
    # Check network connectivity
    if ! curl -s --max-time 5 https://ollama.com > /dev/null; then
        resources::handle_error \
            "Cannot connect to ollama.com for installation" \
            "network" \
            "Check internet connection and firewall settings"
        return 1
    fi
    
    # Install binary with error handling
    if ! ollama::install_binary; then
        resources::handle_error \
            "Failed to install Ollama binary" \
            "system" \
            "Check system requirements and available disk space"
        return 1
    fi
    
    # Create user with rollback
    if ! ollama::create_user; then
        resources::handle_error \
            "Failed to create ollama user" \
            "system" \
            "Check user creation permissions and existing user conflicts"
        return 1
    fi
    
    # Install service with rollback
    if ! ollama::install_service; then
        resources::handle_error \
            "Failed to install Ollama systemd service" \
            "system" \
            "Check systemd status and service file permissions"
        return 1
    fi
    
    # Start service with error handling
    if ! ollama::start; then
        resources::handle_error \
            "Failed to start Ollama service" \
            "system" \
            "Check service logs: journalctl -u ollama -n 20"
        return 1
    fi
    
    # Install models with error handling
    if ! ollama::install_models; then
        log::warn "Model installation failed, but Ollama service is running"
        log::info "You can install models manually later with: ollama pull <model-name>"
    fi
    
    # At this point, Ollama is successfully installed
    # Clear rollback context to prevent removing it if config update fails
    log::info "Ollama core installation completed successfully"
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
    
    # Update Vrooli configuration (non-critical)
    if ! ollama::update_config; then
        log::warn "Failed to update Vrooli configuration, but Ollama is installed"
        log::info "You may need to configure Vrooli manually to use this Ollama instance"
    fi
    
    # Validate installation (informational only, don't fail)
    if ! ollama::verify_installation; then
        log::warn "Some verification checks failed, but Ollama should still be functional"
        log::info "Check service status with: systemctl status ollama"
    else
        log::success "✅ Ollama installation completed successfully"
    fi
    
    # Show status
    echo
    ollama::status
}

#######################################
# Verify Ollama installation is complete and functional
#######################################
ollama::verify_installation() {
    log::info "Verifying Ollama installation..."
    
    local verification_errors=()
    local verification_warnings=()
    
    # Check binary installation
    if ! ollama::is_installed; then
        verification_errors+=("Ollama binary not found in PATH")
    else
        log::success "✅ Ollama binary installed"
    fi
    
    # Check user exists
    if ! id "$OLLAMA_USER" &>/dev/null; then
        verification_errors+=("Ollama user '$OLLAMA_USER' does not exist")
    else
        log::success "✅ Ollama user '$OLLAMA_USER' exists"
    fi
    
    # Check systemd service (check if service exists anywhere in systemd)
    if ! systemctl status "$OLLAMA_SERVICE_NAME" &>/dev/null && ! systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service"; then
        verification_errors+=("Ollama systemd service not found")
    else
        log::success "✅ Ollama systemd service installed"
        
        # Check if service is enabled
        if ! systemctl is-enabled "$OLLAMA_SERVICE_NAME" &>/dev/null; then
            verification_warnings+=("Ollama service is not enabled for auto-start")
        else
            log::success "✅ Ollama service enabled for auto-start"
        fi
    fi
    
    # Check service is running
    if ! resources::is_service_active "$OLLAMA_SERVICE_NAME"; then
        verification_errors+=("Ollama service is not running")
    else
        log::success "✅ Ollama service is active"
    fi
    
    # Check port accessibility
    if ! resources::is_service_running "$OLLAMA_PORT"; then
        verification_errors+=("Ollama is not listening on port $OLLAMA_PORT")
    else
        log::success "✅ Ollama listening on port $OLLAMA_PORT"
    fi
    
    # Check API health
    if ! ollama::is_healthy; then
        verification_warnings+=("Ollama API is not responding to health checks")
    else
        log::success "✅ Ollama API is healthy and responsive"
    fi
    
    # Check models are available
    local model_count=0
    if system::is_command "ollama"; then
        # Count lines that contain model names (skip header)
        model_count=$(ollama list 2>/dev/null | tail -n +2 | wc -l || echo "0")
        if [[ "$model_count" =~ ^[0-9]+$ ]] && [[ $model_count -gt 0 ]]; then
            log::success "✅ $model_count model(s) installed and available"
        else
            verification_warnings+=("No models are installed")
        fi
    fi
    
    # Check configuration
    if [[ -f "$VROOLI_RESOURCES_CONFIG" ]]; then
        if system::is_command "jq"; then
            local config_exists
            config_exists=$(jq -r '.services.ai.ollama // empty' "$VROOLI_RESOURCES_CONFIG" 2>/dev/null)
            if [[ -n "$config_exists" && "$config_exists" != "null" ]]; then
                log::success "✅ Ollama configured in Vrooli resources"
            else
                verification_warnings+=("Ollama not found in Vrooli resource configuration")
            fi
        fi
    else
        verification_warnings+=("Vrooli resource configuration file not found")
    fi
    
    # Print verification summary
    echo
    log::header "🔍 Installation Verification Summary"
    
    if [[ ${#verification_errors[@]} -eq 0 ]]; then
        log::success "✅ Ollama installation verification passed!"
        
        if [[ ${#verification_warnings[@]} -gt 0 ]]; then
            echo
            log::warn "⚠️  Warnings found:"
            for warning in "${verification_warnings[@]}"; do
                log::warn "  • $warning"
            done
            
            echo
            log::info "💡 These warnings don't prevent Ollama from working but may affect functionality"
        fi
        
        echo
        log::info "🚀 Ollama is ready to use!"
        log::info "   Base URL: $OLLAMA_BASE_URL"
        log::info "   Test API: curl $OLLAMA_BASE_URL/api/tags"
        if [[ $model_count -gt 0 ]]; then
            log::info "   Chat with a model: ollama run llama3.1:8b"
        else
            log::info "   Install a model: ollama pull llama3.1:8b"
        fi
        
        return 0
    else
        log::error "❌ Ollama installation verification failed!"
        echo
        log::error "Errors found:"
        for error in "${verification_errors[@]}"; do
            log::error "  • $error"
        done
        
        if [[ ${#verification_warnings[@]} -gt 0 ]]; then
            echo
            log::warn "Additional warnings:"
            for warning in "${verification_warnings[@]}"; do
                log::warn "  • $warning"
            done
        fi
        
        echo
        log::info "💡 Try reinstalling with: $0 --action install --force yes"
        
        return 1
    fi
}

#######################################
# Uninstall Ollama
#######################################
ollama::uninstall() {
    log::header "🗑️  Uninstalling Ollama"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will completely remove Ollama, including all models and data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop service
    if resources::is_service_active "$OLLAMA_SERVICE_NAME"; then
        ollama::stop
    fi
    
    # Remove systemd service
    if resources::can_sudo && [[ -f "/etc/systemd/system/${OLLAMA_SERVICE_NAME}.service" ]]; then
        sudo systemctl disable "$OLLAMA_SERVICE_NAME" 2>/dev/null || true
        sudo rm -f "/etc/systemd/system/${OLLAMA_SERVICE_NAME}.service"
        sudo systemctl daemon-reload
        log::info "Systemd service removed"
    fi
    
    # Remove binary
    if resources::can_sudo && [[ -f "${OLLAMA_INSTALL_DIR}/ollama" ]]; then
        sudo rm -f "${OLLAMA_INSTALL_DIR}/ollama"
        log::info "Ollama binary removed"
    fi
    
    # Remove user (optional - may have other uses)
    if id "$OLLAMA_USER" &>/dev/null && resources::can_sudo; then
        read -p "Remove $OLLAMA_USER user? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            sudo userdel "$OLLAMA_USER" 2>/dev/null || true
            log::info "User $OLLAMA_USER removed"
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "ai" "ollama"
    
    log::success "✅ Ollama uninstalled successfully"
}

#######################################
# Show Ollama status
#######################################
ollama::status() {
    resources::print_status "ollama" "$OLLAMA_PORT" "$OLLAMA_SERVICE_NAME"
    
    # Additional Ollama-specific status
    if ollama::is_healthy; then
        echo
        log::info "API Endpoints:"
        log::info "  Base URL: $OLLAMA_BASE_URL"
        log::info "  Models: $OLLAMA_BASE_URL/api/tags"
        log::info "  Chat: $OLLAMA_BASE_URL/api/chat"
        log::info "  Generate: $OLLAMA_BASE_URL/api/generate"
        
        # Show installed models
        echo
        local model_count
        if system::is_command "ollama"; then
            model_count=$(ollama list 2>/dev/null | tail -n +2 | wc -l || echo "0")
            log::info "Installed models: $model_count"
        fi
    fi
}

#######################################
# Show Ollama information
#######################################
ollama::info() {
    cat << EOF
=== Ollama Resource Information ===

ID: ollama
Category: ai
Display Name: Ollama
Description: Local LLM inference engine

Service Details:
- Binary Location: $OLLAMA_INSTALL_DIR/ollama
- Service Port: $OLLAMA_PORT
- Service URL: $OLLAMA_BASE_URL
- Service Name: $OLLAMA_SERVICE_NAME
- Run User: $OLLAMA_USER

Endpoints:
- Health Check: $OLLAMA_BASE_URL/api/tags
- List Models: $OLLAMA_BASE_URL/api/tags
- Chat: $OLLAMA_BASE_URL/api/chat
- Generate: $OLLAMA_BASE_URL/api/generate
- Pull Model: $OLLAMA_BASE_URL/api/pull
- Show Model: $OLLAMA_BASE_URL/api/show

Configuration:
- Default Models (2025): ${DEFAULT_MODELS[@]}
- Model Catalog: $(echo "${!MODEL_CATALOG[@]}" | wc -w) models available
- Model Storage: ~/.ollama/models
- Total Default Size: $(ollama::calculate_default_size)GB

Ollama Features:
- Local LLM inference
- Multiple model support
- Streaming responses
- OpenAI-compatible API
- GPU acceleration support
- Model quantization
- Custom model creation
- Multi-modal support (vision models)

Example Usage:
# Show available models from catalog
$0 --action available

# List currently installed models
curl $OLLAMA_BASE_URL/api/tags

# Pull new models
ollama pull deepseek-r1:8b
ollama pull qwen2.5-coder:7b

# Run interactive chat with different models
ollama run llama3.1:8b      # General purpose
ollama run deepseek-r1:8b   # Advanced reasoning 
ollama run qwen2.5-coder:7b # Code generation

# Generate text via API
curl -X POST $OLLAMA_BASE_URL/api/generate \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "llama3.1:8b",
    "prompt": "Why is the sky blue?",
    "stream": false
  }'

# Chat completion (OpenAI-compatible)
curl -X POST $OLLAMA_BASE_URL/api/chat \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "llama3.1:8b",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'

For more information, visit: https://ollama.com/library
EOF
}

#######################################
# Main execution function
#######################################
ollama::main() {
    ollama::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            ollama::install
            ;;
        "uninstall")
            ollama::uninstall
            ;;
        "start")
            ollama::start
            ;;
        "stop")
            ollama::stop
            ;;
        "restart")
            ollama::restart
            ;;
        "status")
            ollama::status
            ;;
        "models")
            ollama::list_models
            ;;
        "available")
            ollama::show_available_models
            ;;
        "info")
            ollama::info
            ;;
        *)
            log::error "Unknown action: $ACTION"
            ollama::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    ollama::main "$@"
fi