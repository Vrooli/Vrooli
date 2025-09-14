#!/usr/bin/env bash
################################################################################
# Whisper API Functions
# 
# Functions for interacting with the Whisper API
################################################################################

#######################################
# Validate audio file format and size
# Arguments:
#   $1 - Audio file path
# Returns: 0 if valid, 1 otherwise
#######################################
whisper::validate_audio_file() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "No audio file specified"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "Audio file not found: $file"
        return 1
    fi
    
    # Check file size (Whisper has a 25MB limit for API)
    local file_size_bytes
    file_size_bytes=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
    local max_size=$((25 * 1024 * 1024))  # 25MB in bytes
    
    if [[ $file_size_bytes -gt $max_size ]]; then
        local file_size_mb=$((file_size_bytes / 1024 / 1024))
        log::error "File too large: ${file_size_mb}MB (maximum 25MB)"
        return 1
    fi
    
    # Check file extension
    local file_ext="${file##*.}"
    file_ext="${file_ext,,}"
    
    case "$file_ext" in
        wav|mp3|m4a|ogg|flac|aac|wma)
            return 0
            ;;
        *)
            log::error "Unsupported audio format: $file_ext"
            log::info "Supported formats: wav, mp3, m4a, ogg, flac, aac, wma"
            return 1
            ;;
    esac
}

#######################################
# Parse API error response
# Arguments:
#   $1 - Response JSON
#######################################
whisper::parse_api_error() {
    local response="${1:-}"
    
    if [[ -z "$response" ]]; then
        log::error "Empty response from Whisper API"
        return
    fi
    
    # Try to parse as JSON error
    if echo "$response" | jq . >/dev/null 2>&1; then
        local error_detail
        error_detail=$(echo "$response" | jq -r '.detail // empty' 2>/dev/null)
        
        if [[ -n "$error_detail" ]]; then
            log::error "Whisper API error: $error_detail"
        else
            log::error "Unknown API error response format"
            echo "Raw response: $response"
        fi
    else
        log::error "Invalid JSON response from Whisper API"
        echo "Raw response: $response"
    fi
}

#######################################
# Transcribe audio file using Whisper API
# Arguments:
#   $1 - Audio file path
#   $2 - Task (transcribe or translate) [optional, default: transcribe]
#   $3 - Language (language code or auto) [optional, default: auto]
# Outputs: JSON response from Whisper API
#######################################
whisper::transcribe_audio() {
    local file="${1:-}"
    local task="${2:-transcribe}"
    local language="${3:-auto}"
    
    # Setup agent tracking
    local agent_id
    agent_id=$(agents::generate_id)
    local command="whisper::transcribe_audio $file $task $language"
    
    # Register agent
    if ! agents::register "$agent_id" "$$" "$command"; then
        log::warn "Failed to register agent for tracking"
    fi
    
    # Setup cleanup trap
    whisper::setup_agent_cleanup "$agent_id"
    
    # Validate audio file
    if ! whisper::validate_audio_file "$file"; then
        return 1
    fi
    
    # Check if service is healthy
    if ! whisper::is_healthy; then
        log::error "Whisper service is not available"
        log::info "Start it with: resource-whisper start"
        return 1
    fi
    
    # Get file info for logging
    local file_size_mb
    file_size_mb=$(du -m "$file" | cut -f1)
    local file_ext="${file##*.}"
    file_ext="${file_ext,,}"
    
    log::info "Transcribing file: $file"
    log::info "Model: ${WHISPER_DEFAULT_MODEL:-base}, Task: $task, Language: $language"
    log::info "File size: ${file_size_mb}MB, Format: $file_ext"
    
    # Prepare curl command - output must be query parameter
    local curl_cmd=(
        curl -s -X POST
        "${WHISPER_BASE_URL:-http://localhost:9000}/asr?output=json"
        -F "audio_file=@${file}"
        -F "task=${task}"
    )
    
    # Add language if specified
    if [[ "$language" != "auto" ]]; then
        curl_cmd+=(-F "language=${language}")
    fi
    
    # Execute transcription with progress
    log::info "Processing... (this may take a few minutes for longer files)"
    local response
    local start_time=$(date +%s)
    
    if response=$("${curl_cmd[@]}" 2>/dev/null); then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        # Check if response is valid JSON and contains transcription
        if echo "$response" | jq . >/dev/null 2>&1; then
            # Check if this is an error response
            if echo "$response" | jq -e '.detail' >/dev/null 2>&1; then
                whisper::parse_api_error "$response"
                return 1
            fi
            
            # Check if response contains expected transcription fields
            if echo "$response" | jq -e '.text' >/dev/null 2>&1; then
                log::success "✅ Transcription completed in ${duration}s"
                echo "$response" | jq .
                
                # Show summary
                local text_length
                text_length=$(echo "$response" | jq -r '.text' | wc -c)
                log::info "Transcribed text length: ${text_length} characters"
                
                # Check for silent audio indication
                local no_speech_prob
                no_speech_prob=$(echo "$response" | jq -r '.segments[0].no_speech_prob // 0' 2>/dev/null)
                if command -v bc >/dev/null 2>&1 && (( $(echo "$no_speech_prob > 0.8" | bc -l 2>/dev/null || echo 0) )); then
                    log::warn "⚠️  High probability of no speech detected (${no_speech_prob})"
                    log::info "This may be a silent file or very quiet audio"
                fi
            else
                log::error "Unexpected response format from Whisper API"
                echo "Raw response: $response"
                return 1
            fi
        else
            whisper::parse_api_error "$response"
            return 1
        fi
    else
        log::error "Failed to connect to Whisper service"
        log::info "Check if Whisper is running: resource-whisper status"
        return 1
    fi
}

#######################################
# Get API information
#######################################
whisper::get_api_info() {
    echo "Whisper API Information:"
    echo ""
    echo "Base URL: ${WHISPER_BASE_URL:-http://localhost:9000}"
    echo "Port: ${WHISPER_PORT:-9000}"
    echo "Transcription endpoint: /asr"
    echo "Documentation: /docs"
    echo "OpenAPI spec: /openapi.json"
    echo ""
    echo "Supported audio formats:"
    echo "  wav, mp3, m4a, ogg, flac, aac, wma"
    echo ""
    echo "Maximum file size: 25MB"
    echo "Default model: ${WHISPER_DEFAULT_MODEL:-base}"
}

#######################################
# Test API connectivity
#######################################
whisper::test_api() {
    local base_url="${WHISPER_BASE_URL:-http://localhost:9000}"
    
    log::info "Testing Whisper API connectivity..."
    
    # Test basic connectivity
    if curl -sf "$base_url/docs" >/dev/null 2>&1; then
        log::success "✅ API is accessible"
    else
        log::error "❌ Cannot connect to API at $base_url"
        return 1
    fi
    
    # Test health endpoint if available
    if curl -sf "$base_url/health" >/dev/null 2>&1; then
        log::success "✅ Health check passed"
    else
        log::warn "⚠️  No health endpoint available"
    fi
    
    log::info "API test completed"
}

#######################################
# Show available models information
#######################################
whisper::show_models() {
    echo "Available Whisper models:"
    echo ""
    echo "  tiny.en     English-only,  ~39 MB,  ~32x realtime"
    echo "  tiny        Multilingual,  ~39 MB,  ~32x realtime"
    echo "  base.en     English-only,  ~74 MB,  ~16x realtime"
    echo "  base        Multilingual,  ~74 MB,  ~16x realtime"
    echo "  small.en    English-only, ~244 MB,   ~6x realtime"
    echo "  small       Multilingual, ~244 MB,   ~6x realtime"
    echo "  medium.en   English-only, ~769 MB,   ~2x realtime"
    echo "  medium      Multilingual, ~769 MB,   ~2x realtime"
    echo "  large-v1    Multilingual, ~1550 MB,  ~1x realtime"
    echo "  large-v2    Multilingual, ~1550 MB,  ~1x realtime"
    echo "  large-v3    Multilingual, ~1550 MB,  ~1x realtime"
    echo ""
    echo "Current default: ${WHISPER_DEFAULT_MODEL:-base}"
}

#######################################
# Show supported languages
#######################################
whisper::show_languages() {
    echo "Supported languages (99 languages):"
    echo ""
    echo "Afrikaans, Albanian, Amharic, Arabic, Armenian, Assamese, Azerbaijani,"
    echo "Bashkir, Basque, Belarusian, Bengali, Bosnian, Breton, Bulgarian,"
    echo "Burmese, Castilian, Catalan, Chinese, Croatian, Czech, Danish, Dutch,"
    echo "English, Estonian, Faroese, Finnish, Flemish, French, Galician,"
    echo "Georgian, German, Greek, Gujarati, Haitian, Haitian Creole, Hausa,"
    echo "Hawaiian, Hebrew, Hindi, Hungarian, Icelandic, Indonesian, Italian,"
    echo "Japanese, Javanese, Kannada, Kazakh, Khmer, Korean, Lao, Latin,"
    echo "Latvian, Lingala, Lithuanian, Luxembourgish, Macedonian, Malagasy,"
    echo "Malay, Malayalam, Maltese, Maori, Marathi, Mongolian, Myanmar,"
    echo "Nepali, Norwegian, Nynorsk, Occitan, Pashto, Persian, Polish,"
    echo "Portuguese, Punjabi, Romanian, Russian, Sanskrit, Serbian, Shona,"
    echo "Sindhi, Sinhala, Slovak, Slovenian, Somali, Spanish, Sundanese,"
    echo "Swahili, Swedish, Tagalog, Tajik, Tamil, Tatar, Telugu, Thai,"
    echo "Tibetan, Turkish, Turkmen, Ukrainian, Urdu, Uzbek, Vietnamese,"
    echo "Welsh, Yiddish, Yoruba"
    echo ""
    echo "Note: Language detection is automatic, or specify with language parameter"
}