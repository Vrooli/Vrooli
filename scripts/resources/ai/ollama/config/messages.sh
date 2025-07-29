#!/usr/bin/env bash

# Ollama Resource User-Facing Messages
# This file contains all user-facing messages for consistent communication

#######################################
# Installation Messages
#######################################
readonly MSG_OLLAMA_INSTALLING="Installing Ollama..."
readonly MSG_OLLAMA_ALREADY_INSTALLED="Ollama is already installed (use --force yes to reinstall)"
readonly MSG_OLLAMA_INSTALL_SUCCESS="✅ Ollama installation completed successfully"
readonly MSG_OLLAMA_INSTALL_FAILED="❌ Ollama installation failed"

readonly MSG_BINARY_INSTALL_SUCCESS="✅ Ollama binary installed successfully"
readonly MSG_BINARY_INSTALL_FAILED="Ollama installation failed - binary not found after installation"
readonly MSG_USER_CREATE_SUCCESS="✅ User $OLLAMA_USER created"
readonly MSG_USER_CREATE_FAILED="Failed to create user $OLLAMA_USER"
readonly MSG_SERVICE_INSTALL_SUCCESS="✅ Ollama systemd service installed successfully"

readonly MSG_DOWNLOAD_FAILED="Failed to download Ollama installer"
readonly MSG_INSTALLER_EMPTY="Downloaded installer is empty"
readonly MSG_INSTALLER_SUCCESS="Ollama installer completed successfully"
readonly MSG_INSTALLER_FAILED="Ollama installer failed"
readonly MSG_SUDO_REQUIRED="Sudo privileges required to install Ollama"
readonly MSG_USER_SUDO_REQUIRED="Sudo privileges required to create $OLLAMA_USER user"

#######################################
# Status Messages
#######################################
readonly MSG_OLLAMA_RUNNING="✅ Ollama is running and healthy on port $OLLAMA_PORT"
readonly MSG_OLLAMA_STARTED_NO_API="⚠️  Ollama service started but API is not responding"
readonly MSG_OLLAMA_NOT_INSTALLED="Ollama binary not found"
readonly MSG_OLLAMA_API_UNAVAILABLE="Ollama API is not available"

readonly MSG_STATUS_BINARY_OK="✅ Ollama binary installed"
readonly MSG_STATUS_USER_OK="✅ Ollama user '$OLLAMA_USER' exists"
readonly MSG_STATUS_SERVICE_OK="✅ Ollama systemd service installed"
readonly MSG_STATUS_SERVICE_ENABLED="✅ Ollama service enabled for auto-start"
readonly MSG_STATUS_SERVICE_ACTIVE="✅ Ollama service is active"
readonly MSG_STATUS_PORT_OK="✅ Ollama listening on port $OLLAMA_PORT"
readonly MSG_STATUS_API_OK="✅ Ollama API is healthy and responsive"

#######################################
# Model Messages
#######################################
readonly MSG_MODELS_HEADER="📚 Available Models (from catalog)"
readonly MSG_MODELS_LEGEND="✅ = Default models    ⚠️  = Legacy models"
readonly MSG_MODELS_TOTAL_SIZE="Total default models size: \$(ollama::calculate_default_size)GB"

readonly MSG_MODEL_INSTALL_SUCCESS="✅ Model \$model installed successfully"
readonly MSG_MODEL_INSTALL_FAILED="❌ Failed to install model: \$model"
readonly MSG_MODEL_PULL_SUCCESS="Model \$model pulled successfully"
readonly MSG_MODEL_PULL_FAILED="Failed to pull model: \$model"
readonly MSG_MODEL_NOT_INSTALLED="Model '\$model' is not installed"
readonly MSG_MODEL_VALIDATION_FAILED="Model validation failed"
readonly MSG_MODEL_NONE_SPECIFIED="No models specified for installation"

readonly MSG_MODELS_VALIDATED="Validated \${#models[@]} models, total size: \$(printf \"%.1f\" \"\$total_size\")GB"
readonly MSG_MODELS_INSTALLED="• Installed models: \${installed_models[*]}"
readonly MSG_MODELS_FAILED="• Failed models: \${failed_models[*]}"
readonly MSG_MODELS_COUNT="✅ \$model_count model(s) installed and available"

#######################################
# API/Prompt Messages
#######################################
readonly MSG_PROMPT_SENDING="Sending prompt to \$model..."
readonly MSG_PROMPT_RESPONSE_HEADER="🤖 Response from \$model"
readonly MSG_PROMPT_NO_TEXT="No prompt text provided"
readonly MSG_PROMPT_USAGE="Use: \$0 --action prompt --text 'your prompt here'"
readonly MSG_PROMPT_API_ERROR="Ollama returned error: \$error_msg"
readonly MSG_PROMPT_NO_RESPONSE="No response text received"
readonly MSG_PROMPT_RESPONSE_TIME="⏱️  Response time: \${duration}s"
readonly MSG_PROMPT_TOKEN_COUNT="📊 Tokens: \${prompt_eval_count} prompt + \${eval_count} generated"
readonly MSG_PROMPT_PARAMETERS="🎛️  Parameters: \$params_info"

readonly MSG_MODEL_SELECTING="Selecting best available model for type: \$use_case"
readonly MSG_MODEL_USING="Using specified model: \$model"
readonly MSG_MODEL_SELECTED="Selected model: \$model"
readonly MSG_MODEL_NONE_SUITABLE="No suitable models found for type '\$use_case'"
readonly MSG_MODEL_INSTALL_FIRST="Install a model first with: ollama pull llama3.1:8b"

#######################################
# Warning Messages
#######################################
readonly MSG_UNKNOWN_MODELS="Unknown models (not in catalog): \${invalid_models[*]}"
readonly MSG_USE_AVAILABLE_MODELS="Use 'ollama::show_available_models' to see available models"
readonly MSG_LOW_DISK_SPACE="⚠️  Low disk space detected: \${available_space_gb}GB available"
readonly MSG_JQ_UNAVAILABLE="jq not available, showing raw response"
readonly MSG_MODELS_INSTALL_FAILED="Model installation failed, but Ollama service is running"
readonly MSG_CONFIG_UPDATE_FAILED="Failed to update Vrooli configuration, but Ollama is installed"
readonly MSG_VERIFICATION_FAILED="Some verification checks failed, but Ollama should still be functional"

#######################################
# Port/Network Messages
#######################################
readonly MSG_PORT_CONFLICT="Port validation failed for Ollama - critical conflict"
readonly MSG_PORT_WARNING="Port validation returned warnings for Ollama, but continuing"
readonly MSG_PORT_UNEXPECTED="Unexpected port validation result: \$port_validation_result"

#######################################
# Help/Info Messages
#######################################
readonly MSG_START_OLLAMA="Start Ollama with: \$0 --action start"
readonly MSG_CHECK_STATUS="Check if Ollama is running: \$0 --action status"
readonly MSG_AVAILABLE_MODELS="Available models:"
readonly MSG_INSTALL_MODEL="Install the model with: ollama pull \$model"
readonly MSG_FAILED_API_REQUEST="Failed to send request to Ollama API"
readonly MSG_LIST_MODELS_FAILED="Could not list models (this is usually temporary)"

#######################################
# Export messages function
#######################################
ollama::export_messages() {
    # Export all message variables for use in other modules
    # Installation messages
    export MSG_OLLAMA_INSTALLING MSG_OLLAMA_ALREADY_INSTALLED MSG_OLLAMA_INSTALL_SUCCESS
    export MSG_OLLAMA_INSTALL_FAILED MSG_BINARY_INSTALL_SUCCESS MSG_BINARY_INSTALL_FAILED
    export MSG_USER_CREATE_SUCCESS MSG_USER_CREATE_FAILED MSG_SERVICE_INSTALL_SUCCESS
    export MSG_DOWNLOAD_FAILED MSG_INSTALLER_EMPTY MSG_INSTALLER_SUCCESS MSG_INSTALLER_FAILED
    export MSG_SUDO_REQUIRED MSG_USER_SUDO_REQUIRED
    
    # Status messages
    export MSG_OLLAMA_RUNNING MSG_OLLAMA_STARTED_NO_API MSG_OLLAMA_NOT_INSTALLED
    export MSG_OLLAMA_API_UNAVAILABLE MSG_STATUS_BINARY_OK MSG_STATUS_USER_OK
    export MSG_STATUS_SERVICE_OK MSG_STATUS_SERVICE_ENABLED MSG_STATUS_SERVICE_ACTIVE
    export MSG_STATUS_PORT_OK MSG_STATUS_API_OK
    
    # Model messages
    export MSG_MODELS_HEADER MSG_MODELS_LEGEND MSG_MODELS_TOTAL_SIZE
    export MSG_MODEL_INSTALL_SUCCESS MSG_MODEL_INSTALL_FAILED MSG_MODEL_PULL_SUCCESS
    export MSG_MODEL_PULL_FAILED MSG_MODEL_NOT_INSTALLED MSG_MODEL_VALIDATION_FAILED
    export MSG_MODEL_NONE_SPECIFIED MSG_MODELS_VALIDATED MSG_MODELS_INSTALLED
    export MSG_MODELS_FAILED MSG_MODELS_COUNT
    
    # API/Prompt messages
    export MSG_PROMPT_SENDING MSG_PROMPT_RESPONSE_HEADER MSG_PROMPT_NO_TEXT
    export MSG_PROMPT_USAGE MSG_PROMPT_API_ERROR MSG_PROMPT_NO_RESPONSE
    export MSG_PROMPT_RESPONSE_TIME MSG_PROMPT_TOKEN_COUNT MSG_PROMPT_PARAMETERS
    export MSG_MODEL_SELECTING MSG_MODEL_USING MSG_MODEL_SELECTED
    export MSG_MODEL_NONE_SUITABLE MSG_MODEL_INSTALL_FIRST
    
    # Warning messages
    export MSG_UNKNOWN_MODELS MSG_USE_AVAILABLE_MODELS MSG_LOW_DISK_SPACE
    export MSG_JQ_UNAVAILABLE MSG_MODELS_INSTALL_FAILED MSG_CONFIG_UPDATE_FAILED
    export MSG_VERIFICATION_FAILED
    
    # Port/Network messages
    export MSG_PORT_CONFLICT MSG_PORT_WARNING MSG_PORT_UNEXPECTED
    
    # Help/Info messages
    export MSG_START_OLLAMA MSG_CHECK_STATUS MSG_AVAILABLE_MODELS MSG_INSTALL_MODEL
    export MSG_FAILED_API_REQUEST MSG_LIST_MODELS_FAILED
}