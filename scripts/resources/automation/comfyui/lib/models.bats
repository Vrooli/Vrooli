#!/usr/bin/env bats
# Tests for ComfyUI models.sh functions

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export COMFYUI_CUSTOM_PORT="8188"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_MODELS_DIR="/tmp/comfyui-test/models"
    export COMFYUI_DATA_DIR="/tmp/comfyui-test"
    export MODEL_NAME="test-model-v1.0.safetensors"
    export MODEL_URL="https://huggingface.co/test/model/resolve/main/model.safetensors"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$COMFYUI_MODELS_DIR"/{checkpoints,loras,vae,controlnet,embeddings,upscale_models}
    
    # Create test model files
    touch "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors"
    touch "$COMFYUI_MODELS_DIR/loras/test-lora.safetensors"
    touch "$COMFYUI_MODELS_DIR/vae/test-vae.safetensors"
    
    # Mock system functions
    
    # Mock curl/wget for downloads
    
    wget() {
        echo "WGET_DOWNLOAD: $*"
        # Create a fake downloaded file
        local filename=$(basename "$MODEL_URL")
        touch "$COMFYUI_MODELS_DIR/checkpoints/$filename"
        return 0
    }
    
    # Mock sha256sum
    sha256sum() {
        echo "abc123def456789 $1"
    }
    
    # Mock du (disk usage)
    du() {
        case "$*" in
            *"-h"*) echo "2.3G  $1" ;;
            *) echo "2415919104 $1" ;;
        esac
    }
    
    # Mock find
    find() {
        case "$*" in
            *"-name *.safetensors"*)
                echo "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors"
                echo "$COMFYUI_MODELS_DIR/loras/test-lora.safetensors"
                echo "$COMFYUI_MODELS_DIR/vae/test-vae.safetensors"
                ;;
            *"-name *.ckpt"*)
                echo "$COMFYUI_MODELS_DIR/checkpoints/legacy-model.ckpt"
                ;;
            *"-type f"*)
                echo "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors"
                echo "$COMFYUI_MODELS_DIR/loras/test-lora.safetensors"
                ;;
            *) echo "FIND: $*" ;;
        esac
    }
    
    # Mock Docker operations
    
    # Mock filesystem operations
    ls() {
        case "$*" in
            *"checkpoints"*)
                echo "test-checkpoint.safetensors"
                echo "legacy-model.ckpt"
                ;;
            *"loras"*)
                echo "test-lora.safetensors"
                ;;
            *"vae"*)
                echo "test-vae.safetensors"
                ;;
            *) command ls "$@" ;;
        esac
    }
    
    # Mock log functions
    
    # Mock ComfyUI utility functions
    comfyui::container_exists() { return 0; }
    comfyui::is_running() { return 0; }
    comfyui::is_healthy() { return 0; }
    
    # Load configuration and messages
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    comfyui::export_config
    comfyui::export_messages
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/models.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$COMFYUI_MODELS_DIR" --test-cleanup
    trash::safe_remove "$COMFYUI_DATA_DIR" --test-cleanup
}

# Test model directory listing
@test "comfyui::list_models shows available models" {
    result=$(comfyui::list_models)
    
    [[ "$result" =~ "test-checkpoint.safetensors" ]]
    [[ "$result" =~ "test-lora.safetensors" ]]
    [[ "$result" =~ "test-vae.safetensors" ]]
}

# Test model listing by type
@test "comfyui::list_models filters by model type" {
    result=$(comfyui::list_models "checkpoints")
    
    [[ "$result" =~ "test-checkpoint.safetensors" ]]
    [[ ! "$result" =~ "test-lora.safetensors" ]]
}

# Test model download
@test "comfyui::download_model downloads model from URL" {
    result=$(comfyui::download_model "$MODEL_URL" "checkpoints")
    
    [[ "$result" =~ "Downloading" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "downloaded" ]]
    [[ "$result" =~ "WGET_DOWNLOAD:" ]] || [[ "$result" =~ "CURL_DOWNLOAD:" ]]
}

# Test model download with invalid URL
@test "comfyui::download_model handles invalid URL" {
    # Override curl to fail
    curl() { return 1; }
    wget() { return 1; }
    
    run comfyui::download_model "invalid-url" "checkpoints"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test model verification
@test "comfyui::verify_model validates model file integrity" {
    result=$(comfyui::verify_model "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors")
    
    [[ "$result" =~ "verify" ]] || [[ "$result" =~ "valid" ]]
}

# Test model verification with corrupted file
@test "comfyui::verify_model detects corrupted model" {
    # Create empty file to simulate corruption
    touch "$COMFYUI_MODELS_DIR/checkpoints/corrupted.safetensors"
    
    result=$(comfyui::verify_model "$COMFYUI_MODELS_DIR/checkpoints/corrupted.safetensors" "expected_hash")
    
    [[ "$result" =~ "corrupted" ]] || [[ "$result" =~ "invalid" ]]
}

# Test model installation
@test "comfyui::install_model installs model to correct directory" {
    result=$(comfyui::install_model "test-model.safetensors" "checkpoints" "$MODEL_URL")
    
    [[ "$result" =~ "Installing" ]]
    [[ "$result" =~ "checkpoints" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "installed" ]]
}

# Test model removal
@test "comfyui::remove_model removes model file" {
    result=$(comfyui::remove_model "test-checkpoint.safetensors" "checkpoints")
    
    [[ "$result" =~ "Removing" ]] || [[ "$result" =~ "deleted" ]]
    [[ "$result" =~ "test-checkpoint.safetensors" ]]
}

# Test model removal with confirmation
@test "comfyui::remove_model handles user confirmation" {
    export YES="yes"
    
    result=$(comfyui::remove_model "test-checkpoint.safetensors" "checkpoints")
    
    [[ "$result" =~ "Removing" ]] || [[ "$result" =~ "deleted" ]]
}

# Test model size calculation
@test "comfyui::get_model_size returns model file size" {
    result=$(comfyui::get_model_size "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors")
    
    [[ "$result" =~ "2.3G" ]] || [[ "$result" =~ "size" ]]
}

# Test model metadata extraction
@test "comfyui::get_model_info extracts model information" {
    result=$(comfyui::get_model_info "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors")
    
    [[ "$result" =~ "test-checkpoint.safetensors" ]]
    [[ "$result" =~ "size" ]] || [[ "$result" =~ "hash" ]]
}

# Test model backup
@test "comfyui::backup_model creates model backup" {
    result=$(comfyui::backup_model "test-checkpoint.safetensors" "checkpoints")
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "backed up" ]]
}

# Test model restore
@test "comfyui::restore_model restores model from backup" {
    result=$(comfyui::restore_model "test-checkpoint.safetensors" "checkpoints")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "restored" ]]
}

# Test popular models installation
@test "comfyui::install_popular_models installs recommended models" {
    result=$(comfyui::install_popular_models)
    
    [[ "$result" =~ "popular" ]] || [[ "$result" =~ "recommended" ]]
    [[ "$result" =~ "Installing" ]]
}

# Test model format conversion
@test "comfyui::convert_model_format converts between formats" {
    result=$(comfyui::convert_model_format "legacy-model.ckpt" "safetensors")
    
    [[ "$result" =~ "convert" ]] || [[ "$result" =~ "conversion" ]]
    [[ "$result" =~ "safetensors" ]]
}

# Test model validation
@test "comfyui::validate_model_format checks model format" {
    result=$(comfyui::validate_model_format "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors")
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "safetensors" ]]
}

# Test model search
@test "comfyui::search_models finds models by pattern" {
    result=$(comfyui::search_models "test")
    
    [[ "$result" =~ "test-checkpoint.safetensors" ]]
    [[ "$result" =~ "test-lora.safetensors" ]]
}

# Test model organization
@test "comfyui::organize_models arranges models by type" {
    result=$(comfyui::organize_models)
    
    [[ "$result" =~ "organize" ]] || [[ "$result" =~ "organized" ]]
}

# Test model cleanup
@test "comfyui::cleanup_models removes unused models" {
    result=$(comfyui::cleanup_models)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "cleaned" ]]
}

# Test model dependency check
@test "comfyui::check_model_dependencies verifies model requirements" {
    result=$(comfyui::check_model_dependencies "test-checkpoint.safetensors")
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "requirements" ]]
}

# Test model compatibility check
@test "comfyui::check_model_compatibility verifies ComfyUI compatibility" {
    result=$(comfyui::check_model_compatibility "test-checkpoint.safetensors")
    
    [[ "$result" =~ "compatible" ]] || [[ "$result" =~ "compatibility" ]]
}

# Test model update check
@test "comfyui::check_model_updates checks for model updates" {
    result=$(comfyui::check_model_updates)
    
    [[ "$result" =~ "update" ]] || [[ "$result" =~ "latest" ]]
}

# Test model hash verification
@test "comfyui::verify_model_hash validates model checksum" {
    result=$(comfyui::verify_model_hash "$COMFYUI_MODELS_DIR/checkpoints/test-checkpoint.safetensors" "abc123def456789")
    
    [[ "$result" =~ "hash" ]] || [[ "$result" =~ "checksum" ]]
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "verified" ]]
}

# Test model cache management
@test "comfyui::manage_model_cache handles model caching" {
    result=$(comfyui::manage_model_cache)
    
    [[ "$result" =~ "cache" ]] || [[ "$result" =~ "cached" ]]
}

# Test model loading status
@test "comfyui::get_loaded_models shows currently loaded models" {
    result=$(comfyui::get_loaded_models)
    
    [[ "$result" =~ "loaded" ]] || [[ "$result" =~ "test-model" ]]
}

# Test model preloading
@test "comfyui::preload_model loads model into memory" {
    result=$(comfyui::preload_model "test-checkpoint.safetensors")
    
    [[ "$result" =~ "preload" ]] || [[ "$result" =~ "loading" ]]
}

# Test model unloading
@test "comfyui::unload_model removes model from memory" {
    result=$(comfyui::unload_model "test-checkpoint.safetensors")
    
    [[ "$result" =~ "unload" ]] || [[ "$result" =~ "unloaded" ]]
}

# Test model statistics
@test "comfyui::get_model_statistics provides model usage stats" {
    result=$(comfyui::get_model_statistics)
    
    [[ "$result" =~ "statistics" ]] || [[ "$result" =~ "stats" ]]
    [[ "$result" =~ "models" ]]
}

# Test model configuration
@test "comfyui::configure_model_settings configures model parameters" {
    result=$(comfyui::configure_model_settings "test-checkpoint.safetensors")
    
    [[ "$result" =~ "configure" ]] || [[ "$result" =~ "settings" ]]
}

# Test model export
@test "comfyui::export_model_list exports model inventory" {
    result=$(comfyui::export_model_list)
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "list" ]]
    [[ "$result" =~ "test-checkpoint.safetensors" ]]
}

# Test model import
@test "comfyui::import_model_list imports model inventory" {
    result=$(comfyui::import_model_list "/tmp/model-list.json")
    
    [[ "$result" =~ "import" ]] || [[ "$result" =~ "imported" ]]
}

# Test model directory validation
@test "comfyui::validate_model_directories checks directory structure" {
    result=$(comfyui::validate_model_directories)
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "directories" ]]
    [[ "$result" =~ "checkpoints" ]]
}

# Test model space usage
@test "comfyui::get_models_disk_usage calculates storage usage" {
    result=$(comfyui::get_models_disk_usage)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "usage" ]]
    [[ "$result" =~ "GB" ]] || [[ "$result" =~ "MB" ]]
}

# Test model synchronization
@test "comfyui::sync_models synchronizes model files" {
    result=$(comfyui::sync_models)
    
    [[ "$result" =~ "sync" ]] || [[ "$result" =~ "synchronized" ]]
}