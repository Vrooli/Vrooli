#!/bin/bash
# Home Assistant Voice Control Integration

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HOME_ASSISTANT_VOICE_DIR="${APP_ROOT}/resources/home-assistant/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${HOME_ASSISTANT_VOICE_DIR}/core.sh"
source "${HOME_ASSISTANT_VOICE_DIR}/health.sh"

#######################################
# Configure voice assistant integration
# Arguments:
#   assistant: Type of assistant (alexa|google|custom)
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::voice::configure() {
    local assistant="${1:-}"
    
    if [[ -z "$assistant" ]]; then
        log::error "No voice assistant type specified"
        log::info "Usage: voice configure <alexa|google|custom>"
        log::info "  alexa  - Amazon Alexa integration"
        log::info "  google - Google Assistant integration"
        log::info "  custom - Custom voice assistant (Whisper/Piper)"
        return 1
    fi
    
    # Initialize environment
    home_assistant::init
    
    # Check if Home Assistant is running
    if ! home_assistant::health::is_healthy; then
        log::error "Home Assistant must be running to configure voice control"
        return 1
    fi
    
    case "$assistant" in
        alexa)
            home_assistant::voice::configure_alexa
            ;;
        google)
            home_assistant::voice::configure_google
            ;;
        custom)
            home_assistant::voice::configure_custom
            ;;
        *)
            log::error "Unknown voice assistant type: $assistant"
            return 1
            ;;
    esac
}

#######################################
# Configure Alexa integration
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::voice::configure_alexa() {
    log::info "Configuring Alexa integration for Home Assistant..."
    
    local config_file="$HOME_ASSISTANT_CONFIG_DIR/configuration.yaml"
    local alexa_config="$HOME_ASSISTANT_CONFIG_DIR/alexa_config.yaml"
    
    # Create Alexa configuration
    cat > "$alexa_config" << 'EOF'
# Alexa Smart Home Integration
alexa:
  smart_home:
    locale: en-US
    endpoint: https://api.amazonalexa.com/v3/events
    client_id: !secret alexa_client_id
    client_secret: !secret alexa_client_secret
    filter:
      include_entities:
        - light.living_room
        - switch.bedroom_light
        - climate.thermostat
        - cover.garage_door
      include_domains:
        - light
        - switch
        - climate
        - cover
        - fan
        - lock
        - sensor
      exclude_entities:
        - sensor.internal_*
    entity_config:
      light.living_room:
        name: Living Room Light
        description: Main living room light
        display_categories: LIGHT
      switch.bedroom_light:
        name: Bedroom Light
        description: Master bedroom light
        display_categories: SWITCH
EOF
    
    # Add include to main configuration if not present
    if ! grep -q "alexa_config.yaml" "$config_file" 2>/dev/null; then
        echo "" >> "$config_file"
        echo "# Alexa Integration" >> "$config_file"
        echo "alexa: !include alexa_config.yaml" >> "$config_file"
    fi
    
    # Create secrets placeholder
    local secrets_file="$HOME_ASSISTANT_CONFIG_DIR/secrets.yaml"
    if [[ ! -f "$secrets_file" ]]; then
        touch "$secrets_file"
    fi
    
    if ! grep -q "alexa_client_id" "$secrets_file" 2>/dev/null; then
        cat >> "$secrets_file" << 'EOF'

# Alexa Integration Secrets
alexa_client_id: "YOUR_ALEXA_CLIENT_ID"
alexa_client_secret: "YOUR_ALEXA_CLIENT_SECRET"
EOF
        log::warning "Please update secrets.yaml with your Alexa credentials"
    fi
    
    log::success "Alexa configuration created"
    log::info "Next steps:"
    log::info "  1. Update secrets.yaml with your Alexa Smart Home credentials"
    log::info "  2. Register your Home Assistant instance with Amazon Developer Console"
    log::info "  3. Configure the Alexa Smart Home skill"
    log::info "  4. Restart Home Assistant to apply changes"
    log::info ""
    log::info "Documentation: https://www.home-assistant.io/integrations/alexa/"
    
    return 0
}

#######################################
# Configure Google Assistant integration
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::voice::configure_google() {
    log::info "Configuring Google Assistant integration for Home Assistant..."
    
    local config_file="$HOME_ASSISTANT_CONFIG_DIR/configuration.yaml"
    local google_config="$HOME_ASSISTANT_CONFIG_DIR/google_assistant_config.yaml"
    
    # Create Google Assistant configuration
    cat > "$google_config" << 'EOF'
# Google Assistant Integration
google_assistant:
  project_id: !secret google_project_id
  service_account: !include SERVICE_ACCOUNT.json
  report_state: true
  expose_by_default: false
  exposed_domains:
    - light
    - switch
    - climate
    - cover
    - fan
    - lock
  entity_config:
    light.living_room:
      name: Living Room
      aliases:
        - Living Room Light
        - Main Light
      room: Living Room
    switch.bedroom_light:
      name: Bedroom Light
      room: Bedroom
    climate.thermostat:
      name: Thermostat
      aliases:
        - AC
        - Air Conditioning
        - Heater
      room: Living Room
EOF
    
    # Add include to main configuration if not present
    if ! grep -q "google_assistant_config.yaml" "$config_file" 2>/dev/null; then
        echo "" >> "$config_file"
        echo "# Google Assistant Integration" >> "$config_file"
        echo "google_assistant: !include google_assistant_config.yaml" >> "$config_file"
    fi
    
    # Create service account placeholder
    local service_account_file="$HOME_ASSISTANT_CONFIG_DIR/SERVICE_ACCOUNT.json"
    if [[ ! -f "$service_account_file" ]]; then
        cat > "$service_account_file" << 'EOF'
{
  "type": "service_account",
  "project_id": "YOUR_PROJECT_ID",
  "private_key_id": "YOUR_PRIVATE_KEY_ID",
  "private_key": "YOUR_PRIVATE_KEY",
  "client_email": "YOUR_CLIENT_EMAIL",
  "client_id": "YOUR_CLIENT_ID",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "YOUR_CERT_URL"
}
EOF
        log::warning "Please update SERVICE_ACCOUNT.json with your Google Cloud credentials"
    fi
    
    # Update secrets file
    local secrets_file="$HOME_ASSISTANT_CONFIG_DIR/secrets.yaml"
    if [[ ! -f "$secrets_file" ]]; then
        touch "$secrets_file"
    fi
    
    if ! grep -q "google_project_id" "$secrets_file" 2>/dev/null; then
        cat >> "$secrets_file" << 'EOF'

# Google Assistant Integration Secrets
google_project_id: "YOUR_GOOGLE_PROJECT_ID"
EOF
        log::warning "Please update secrets.yaml with your Google Project ID"
    fi
    
    log::success "Google Assistant configuration created"
    log::info "Next steps:"
    log::info "  1. Create a Google Cloud project and enable HomeGraph API"
    log::info "  2. Download service account credentials and replace SERVICE_ACCOUNT.json"
    log::info "  3. Update secrets.yaml with your project ID"
    log::info "  4. Configure OAuth 2.0 in Google Actions Console"
    log::info "  5. Restart Home Assistant to apply changes"
    log::info ""
    log::info "Documentation: https://www.home-assistant.io/integrations/google_assistant/"
    
    return 0
}

#######################################
# Configure custom voice assistant (Whisper/Piper)
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::voice::configure_custom() {
    log::info "Configuring custom voice assistant integration..."
    
    local config_file="$HOME_ASSISTANT_CONFIG_DIR/configuration.yaml"
    local assist_config="$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml"
    
    # Create Assist Pipeline configuration
    cat > "$assist_config" << 'EOF'
# Voice Assistant Pipeline Configuration
assist_pipeline:
  pipelines:
    - name: "Custom Voice Assistant"
      conversation_engine: conversation.home_assistant
      conversation_language: en
      stt_engine: stt.faster_whisper
      stt_language: en
      tts_engine: tts.piper
      tts_language: en
      tts_voice: en_US-lessac-medium
      wake_word_entity: wake_word.openwakeword
      wake_word_id: ok_nabu

# Speech-to-Text Configuration (Whisper)
stt:
  - platform: faster_whisper
    api_key: !secret whisper_api_key
    model: base-int8
    beam_size: 5
    language: en

# Text-to-Speech Configuration (Piper)
tts:
  - platform: piper
    voice: en_US-lessac-medium
    speaker: 0
    length_scale: 1.0
    noise_scale: 0.667
    noise_w: 0.333

# Wake Word Detection
wake_word:
  - platform: openwakeword
    model: ok_nabu_v0.1
    threshold: 0.5
    trigger_level: 1

# Conversation Processing
conversation:
  intents:
    TurnOnLight:
      speech:
        - "Turn on the {name} light"
        - "Switch on {name}"
        - "Light up {name}"
    TurnOffLight:
      speech:
        - "Turn off the {name} light"
        - "Switch off {name}"
        - "Lights out {name}"
    SetTemperature:
      speech:
        - "Set temperature to {temperature} degrees"
        - "Make it {temperature} degrees"
        - "Change temperature to {temperature}"
    GetStatus:
      speech:
        - "What's the status of {device}"
        - "Check {device}"
        - "How is {device}"
EOF
    
    # Add include to main configuration if not present
    if ! grep -q "assist_config.yaml" "$config_file" 2>/dev/null; then
        echo "" >> "$config_file"
        echo "# Voice Assistant Configuration" >> "$config_file"
        echo "assist_pipeline: !include assist_config.yaml" >> "$config_file"
    fi
    
    # Create intent scripts
    local intent_scripts="$HOME_ASSISTANT_CONFIG_DIR/intent_scripts.yaml"
    cat > "$intent_scripts" << 'EOF'
# Intent Script Handlers
intent_script:
  TurnOnLight:
    speech:
      text: "Turning on {{ name }}"
    action:
      service: light.turn_on
      data:
        entity_id: "light.{{ name | replace(' ', '_') | lower }}"
  
  TurnOffLight:
    speech:
      text: "Turning off {{ name }}"
    action:
      service: light.turn_off
      data:
        entity_id: "light.{{ name | replace(' ', '_') | lower }}"
  
  SetTemperature:
    speech:
      text: "Setting temperature to {{ temperature }} degrees"
    action:
      service: climate.set_temperature
      data:
        entity_id: climate.thermostat
        temperature: "{{ temperature }}"
  
  GetStatus:
    speech:
      text: >
        {% set entity = 'sensor.' + device | replace(' ', '_') | lower %}
        {% if states(entity) != 'unknown' %}
          {{ device }} is {{ states(entity) }} {{ state_attr(entity, 'unit_of_measurement') or '' }}
        {% else %}
          I couldn't find information about {{ device }}
        {% endif %}
EOF
    
    if ! grep -q "intent_scripts.yaml" "$config_file" 2>/dev/null; then
        echo "intent_script: !include intent_scripts.yaml" >> "$config_file"
    fi
    
    # Update secrets file
    local secrets_file="$HOME_ASSISTANT_CONFIG_DIR/secrets.yaml"
    if [[ ! -f "$secrets_file" ]]; then
        touch "$secrets_file"
    fi
    
    if ! grep -q "whisper_api_key" "$secrets_file" 2>/dev/null; then
        cat >> "$secrets_file" << 'EOF'

# Custom Voice Assistant Secrets
whisper_api_key: "optional_api_key_if_using_cloud"
EOF
    fi
    
    log::success "Custom voice assistant configuration created"
    log::info "Features configured:"
    log::info "  ✓ Whisper for speech-to-text"
    log::info "  ✓ Piper for text-to-speech"
    log::info "  ✓ OpenWakeWord for wake word detection"
    log::info "  ✓ Intent recognition and handling"
    log::info ""
    log::info "Next steps:"
    log::info "  1. Install Whisper addon from Home Assistant Add-on Store"
    log::info "  2. Install Piper addon for text-to-speech"
    log::info "  3. Configure microphone access for your Home Assistant instance"
    log::info "  4. Restart Home Assistant to apply changes"
    log::info ""
    log::info "The system will respond to 'Ok Nabu' wake word by default"
    
    return 0
}

#######################################
# Show voice control status
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::voice::status() {
    log::header "Voice Control Status"
    
    # Initialize environment
    home_assistant::init
    
    local config_file="$HOME_ASSISTANT_CONFIG_DIR/configuration.yaml"
    
    # Check for Alexa configuration
    if grep -q "alexa" "$config_file" 2>/dev/null || [[ -f "$HOME_ASSISTANT_CONFIG_DIR/alexa_config.yaml" ]]; then
        log::success "✓ Alexa integration configured"
    else
        log::info "○ Alexa integration not configured"
    fi
    
    # Check for Google Assistant configuration
    if grep -q "google_assistant" "$config_file" 2>/dev/null || [[ -f "$HOME_ASSISTANT_CONFIG_DIR/google_assistant_config.yaml" ]]; then
        log::success "✓ Google Assistant integration configured"
    else
        log::info "○ Google Assistant integration not configured"
    fi
    
    # Check for custom voice assistant
    if grep -q "assist_pipeline" "$config_file" 2>/dev/null || [[ -f "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" ]]; then
        log::success "✓ Custom voice assistant configured"
        
        # Check for specific components
        if grep -q "stt.faster_whisper" "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" 2>/dev/null; then
            log::info "  • Whisper STT enabled"
        fi
        if grep -q "tts.piper" "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" 2>/dev/null; then
            log::info "  • Piper TTS enabled"
        fi
        if grep -q "wake_word.openwakeword" "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" 2>/dev/null; then
            log::info "  • Wake word detection enabled"
        fi
    else
        log::info "○ Custom voice assistant not configured"
    fi
    
    # Check for intent scripts
    if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/intent_scripts.yaml" ]]; then
        local intent_count=$(grep -c "^  [A-Z]" "$HOME_ASSISTANT_CONFIG_DIR/intent_scripts.yaml" 2>/dev/null || echo "0")
        log::info "  • $intent_count intent handlers configured"
    fi
    
    log::info ""
    log::info "Use 'voice configure <type>' to set up voice control:"
    log::info "  alexa  - Amazon Alexa integration"
    log::info "  google - Google Assistant integration"
    log::info "  custom - Local voice assistant (Whisper/Piper)"
    
    return 0
}

#######################################
# Test voice control configuration
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::voice::test() {
    log::header "Testing Voice Control Configuration"
    
    # Initialize environment
    home_assistant::init
    
    # Check if Home Assistant is running
    if ! home_assistant::health::is_healthy; then
        log::error "Home Assistant must be running to test voice control"
        return 1
    fi
    
    log::info "Checking configuration validity..."
    
    # Use Home Assistant's config check via API
    local check_result
    check_result=$(docker exec home-assistant python -m homeassistant --script check_config --config /config 2>&1) || true
    
    if echo "$check_result" | grep -q "ERROR"; then
        log::error "Configuration has errors:"
        echo "$check_result" | grep "ERROR" | head -5
        return 1
    else
        log::success "✓ Configuration is valid"
    fi
    
    # Check for voice-related integrations via API
    log::info "Checking voice integrations..."
    
    local api_check
    api_check=$(timeout 5 curl -sf "http://localhost:${HOME_ASSISTANT_PORT}/api/" 2>&1) || true
    
    if [[ -n "$api_check" ]]; then
        log::success "✓ API is accessible"
    else
        log::warning "⚠ API check failed - authentication may be required"
    fi
    
    # Check specific voice configurations
    local voice_configs=0
    
    if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/alexa_config.yaml" ]]; then
        log::info "  • Alexa configuration found"
        ((voice_configs++))
    fi
    
    if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/google_assistant_config.yaml" ]]; then
        log::info "  • Google Assistant configuration found"
        ((voice_configs++))
    fi
    
    if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" ]]; then
        log::info "  • Custom voice assistant configuration found"
        ((voice_configs++))
    fi
    
    if [[ $voice_configs -eq 0 ]]; then
        log::warning "No voice configurations found. Run 'voice configure <type>' to set up."
        return 1
    else
        log::success "✓ Found $voice_configs voice configuration(s)"
    fi
    
    return 0
}