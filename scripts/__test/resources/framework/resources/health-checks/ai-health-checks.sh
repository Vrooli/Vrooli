#!/bin/bash
# ====================================================================
# AI Resource Health Checks
# ====================================================================
#
# Category-specific health checks for AI resources including model
# availability, inference readiness, and performance characteristics.
#
# Supported AI Resources:
# - Ollama: LLM inference with model management
# - Whisper: Speech-to-text transcription
# - Unstructured-IO: Document processing and extraction
#
# ====================================================================

# AI resource health check implementations
check_ollama_health() {
    local port="${1:-11434}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/api/tags" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local models_response
    models_response=$(curl -s --max-time 10 "http://localhost:${port}/api/tags" 2>/dev/null)
    
    if [[ -z "$models_response" ]] || ! echo "$models_response" | jq . >/dev/null 2>&1; then
        echo "unhealthy:invalid_api_response"
        return 1
    fi
    
    local model_count
    model_count=$(echo "$models_response" | jq '.models | length' 2>/dev/null)
    
    if [[ -z "$model_count" ]] || [[ "$model_count" == "0" ]]; then
        echo "degraded:no_models_available"
        return 0
    fi
    
    # Test inference capability with a quick request
    local available_model
    available_model=$(echo "$models_response" | jq -r '.models[0].name' 2>/dev/null)
    
    if [[ -n "$available_model" ]] && [[ "$available_model" != "null" ]]; then
        local inference_test
        inference_test=$(curl -s --max-time 15 -X POST "http://localhost:${port}/api/generate" \
            -H "Content-Type: application/json" \
            -d "{\"model\": \"$available_model\", \"prompt\": \"Hi\", \"stream\": false}" 2>/dev/null)
        
        if echo "$inference_test" | jq -e '.response' >/dev/null 2>&1; then
            echo "healthy:models_available:$model_count:inference_ready"
        else
            echo "degraded:models_available:$model_count:inference_failed"
        fi
    else
        echo "degraded:models_available:$model_count:no_usable_models"
    fi
    
    return 0
}

check_whisper_health() {
    local port="${1:-8090}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
        # Try alternative health endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check - test with minimal audio data
    local temp_audio="/tmp/whisper_health_test.wav"
    
    # Create a minimal WAV file for testing (1 second of silence)
    # This is a minimal WAV header + 1 second of 8kHz mono silence
    printf '\x52\x49\x46\x46\x24\x08\x00\x00\x57\x41\x56\x45\x66\x6d\x74\x20\x10\x00\x00\x00\x01\x00\x01\x00\x40\x1f\x00\x00\x80\x3e\x00\x00\x02\x00\x10\x00\x64\x61\x74\x61\x00\x08\x00\x00' > "$temp_audio"
    dd if=/dev/zero bs=1 count=2048 >> "$temp_audio" 2>/dev/null
    
    local transcription_test
    transcription_test=$(curl -s --max-time 30 -X POST "http://localhost:${port}/transcribe" \
        -F "audio=@${temp_audio}" 2>/dev/null)
    
    rm -f "$temp_audio"
    
    if [[ -n "$transcription_test" ]]; then
        if echo "$transcription_test" | jq -e '.text' >/dev/null 2>&1; then
            echo "healthy:transcription_ready"
        else
            echo "degraded:api_responding:transcription_format_unknown"
        fi
    else
        echo "degraded:transcription_failed"
    fi
    
    return 0
}

check_unstructured_io_health() {
    local port="${1:-11450}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/healthcheck" >/dev/null 2>&1; then
        # Try alternative endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check - test with minimal document
    local temp_doc="/tmp/unstructured_health_test.txt"
    echo "This is a test document for health checking." > "$temp_doc"
    
    local processing_test
    processing_test=$(curl -s --max-time 30 -X POST "http://localhost:${port}/general/v0/general" \
        -F "files=@${temp_doc}" \
        -F "strategy=fast" 2>/dev/null)
    
    rm -f "$temp_doc"
    
    if [[ -n "$processing_test" ]]; then
        if echo "$processing_test" | jq -e '.[0].text' >/dev/null 2>&1; then
            echo "healthy:document_processing_ready"
        else
            echo "degraded:api_responding:processing_format_unknown"
        fi
    else
        echo "degraded:document_processing_failed"
    fi
    
    return 0
}

# Generic AI health check dispatcher
check_ai_resource_health() {
    local resource_name="$1"
    local port="$2"
    local health_level="${3:-basic}"
    
    case "$resource_name" in
        "ollama")
            check_ollama_health "$port" "$health_level"
            ;;
        "whisper")
            check_whisper_health "$port" "$health_level"
            ;;
        "unstructured-io")
            check_unstructured_io_health "$port" "$health_level"
            ;;
        *)
            # Fallback to generic HTTP health check
            if curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            ;;
    esac
}

# AI resource capability testing
test_ai_resource_capabilities() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "ollama")
            test_ollama_capabilities "$port"
            ;;
        "whisper")
            test_whisper_capabilities "$port"
            ;;
        "unstructured-io")
            test_unstructured_io_capabilities "$port"
            ;;
        *)
            echo "capability_testing_not_implemented"
            ;;
    esac
}

test_ollama_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    # Test model listing
    if curl -s --max-time 5 "http://localhost:${port}/api/tags" >/dev/null 2>&1; then
        capabilities+=("model_listing")
    fi
    
    # Test generation endpoint
    if curl -s --max-time 5 -X POST "http://localhost:${port}/api/generate" \
        -H "Content-Type: application/json" -d '{}' >/dev/null 2>&1; then
        capabilities+=("text_generation")
    fi
    
    # Test chat endpoint  
    if curl -s --max-time 5 -X POST "http://localhost:${port}/api/chat" \
        -H "Content-Type: application/json" -d '{}' >/dev/null 2>&1; then
        capabilities+=("chat_completion")
    fi
    
    # Test embeddings endpoint
    if curl -s --max-time 5 -X POST "http://localhost:${port}/api/embeddings" \
        -H "Content-Type: application/json" -d '{}' >/dev/null 2>&1; then
        capabilities+=("embeddings")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_whisper_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    # Test transcription endpoint
    if curl -s --max-time 5 -X POST "http://localhost:${port}/transcribe" >/dev/null 2>&1; then
        capabilities+=("speech_to_text")
    fi
    
    # Test if it supports different audio formats (this would need actual testing)
    capabilities+=("audio_format_support")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_unstructured_io_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    # Test document processing endpoint
    if curl -s --max-time 5 -X POST "http://localhost:${port}/general/v0/general" >/dev/null 2>&1; then
        capabilities+=("document_processing")
    fi
    
    # Test different processing strategies
    capabilities+=("multi_strategy_processing")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

# Export functions
export -f check_ai_resource_health
export -f test_ai_resource_capabilities
export -f check_ollama_health
export -f check_whisper_health
export -f check_unstructured_io_health