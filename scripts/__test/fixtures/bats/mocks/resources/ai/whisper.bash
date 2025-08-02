#!/usr/bin/env bash
# Whisper Resource Mock Implementation
# Provides realistic mock responses for Whisper speech-to-text service

# Prevent duplicate loading
if [[ "${WHISPER_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export WHISPER_MOCK_LOADED="true"

#######################################
# Setup Whisper mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::whisper::setup() {
    local state="${1:-healthy}"
    
    # Configure Whisper-specific environment
    export WHISPER_PORT="${WHISPER_PORT:-8090}"
    export WHISPER_BASE_URL="http://localhost:${WHISPER_PORT}"
    export WHISPER_CONTAINER_NAME="${TEST_NAMESPACE}_whisper"
    export WHISPER_MODEL="${WHISPER_MODEL:-base}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$WHISPER_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::whisper::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::whisper::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::whisper::setup_installing_endpoints
            ;;
        "stopped")
            mock::whisper::setup_stopped_endpoints
            ;;
        *)
            echo "[WHISPER_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[WHISPER_MOCK] Whisper mock configured with state: $state"
}

#######################################
# Setup healthy Whisper endpoints
#######################################
mock::whisper::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/health" \
        '{"status":"ok","model":"base","version":"1.0.0"}'
    
    # Transcribe endpoint
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/transcribe" \
        '{
            "text": "Hello, this is a test transcription of the provided audio.",
            "language": "en",
            "duration": 5.2,
            "segments": [
                {
                    "start": 0.0,
                    "end": 2.5,
                    "text": "Hello, this is a test"
                },
                {
                    "start": 2.5,
                    "end": 5.2,
                    "text": "transcription of the provided audio."
                }
            ],
            "word_timestamps": true
        }' \
        "POST"
    
    # Translate endpoint
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/translate" \
        '{
            "text": "This is the translated text in English.",
            "source_language": "es",
            "target_language": "en",
            "duration": 3.8,
            "confidence": 0.95
        }' \
        "POST"
    
    # Detect language endpoint
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/detect-language" \
        '{
            "detected_language": "en",
            "language_probability": {
                "en": 0.95,
                "es": 0.03,
                "fr": 0.02
            }
        }' \
        "POST"
    
    # Models endpoint
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/models" \
        '{
            "current_model": "base",
            "available_models": ["tiny", "base", "small", "medium", "large"],
            "model_info": {
                "base": {
                    "parameters": "74M",
                    "language_support": "multi",
                    "speed": "medium"
                }
            }
        }'
}

#######################################
# Setup unhealthy Whisper endpoints
#######################################
mock::whisper::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/health" \
        '{"status":"unhealthy","error":"Model not loaded"}' \
        "GET" \
        "503"
    
    # Transcribe endpoint returns error
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/transcribe" \
        '{"error":"Service unavailable","details":"Model initialization failed"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing Whisper endpoints
#######################################
mock::whisper::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/health" \
        '{"status":"installing","progress":60,"current_step":"Downloading model"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/transcribe" \
        '{"error":"Service is still initializing","eta_seconds":120}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped Whisper endpoints
#######################################
mock::whisper::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$WHISPER_BASE_URL"
}

#######################################
# Mock Whisper-specific operations
#######################################

# Mock audio processing with different qualities
mock::whisper::simulate_transcription() {
    local audio_duration="$1"
    local quality="${2:-high}"
    
    case "$quality" in
        "high")
            echo '{
                "text": "This is a high-quality transcription with excellent accuracy and proper punctuation.",
                "confidence": 0.98,
                "duration": '"$audio_duration"'
            }'
            ;;
        "medium")
            echo '{
                "text": "This is a medium quality transcription with good accuracy",
                "confidence": 0.85,
                "duration": '"$audio_duration"'
            }'
            ;;
        "low")
            echo '{
                "text": "this is low quality transcription some words missing",
                "confidence": 0.65,
                "duration": '"$audio_duration"'
            }'
            ;;
    esac
}

# Mock batch processing
mock::whisper::simulate_batch_transcription() {
    local file_count="$1"
    local results=()
    
    for i in $(seq 1 "$file_count"); do
        results+=("{\"file\":\"audio_$i.wav\",\"text\":\"Transcription for file $i\",\"status\":\"completed\"}")
    done
    
    echo "{\"batch_id\":\"batch_$(date +%s)\",\"total_files\":$file_count,\"completed\":$file_count,\"results\":[${results[*]}]}"
}

# Mock real-time transcription stream
mock::whisper::simulate_realtime_stream() {
    local chunks=(
        '{"partial":"Hello","is_final":false}'
        '{"partial":"Hello, how","is_final":false}'
        '{"partial":"Hello, how are","is_final":false}'
        '{"partial":"Hello, how are you","is_final":true}'
    )
    
    # In real implementation, this would stream responses
    # For testing, we return the final result
    echo "${chunks[-1]}"
}

#######################################
# Export mock functions
#######################################
export -f mock::whisper::setup
export -f mock::whisper::setup_healthy_endpoints
export -f mock::whisper::setup_unhealthy_endpoints
export -f mock::whisper::setup_installing_endpoints
export -f mock::whisper::setup_stopped_endpoints
export -f mock::whisper::simulate_transcription
export -f mock::whisper::simulate_batch_transcription
export -f mock::whisper::simulate_realtime_stream

echo "[WHISPER_MOCK] Whisper mock implementation loaded"