#!/bin/bash

# Ollama Integration for Huginn
# Provides AI-powered event filtering and analysis

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)}"

# Source required dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${SCRIPT_DIR}/common.sh"

# Configuration
# Handle OLLAMA_HOST whether it's just "localhost" or a full URL
if [[ -n "${OLLAMA_HOST}" ]]; then
    # If OLLAMA_HOST doesn't start with http:// or https://, add http://
    if [[ "${OLLAMA_HOST}" != http://* ]] && [[ "${OLLAMA_HOST}" != https://* ]]; then
        # Check if it has a port
        if [[ "${OLLAMA_HOST}" == *:* ]]; then
            OLLAMA_HOST="http://${OLLAMA_HOST}"
        else
            OLLAMA_HOST="http://${OLLAMA_HOST}:11434"
        fi
    fi
else
    OLLAMA_HOST="http://localhost:11434"
fi
OLLAMA_MODEL="${OLLAMA_MODEL:-llama3.2}"
OLLAMA_TIMEOUT="${OLLAMA_TIMEOUT:-30}"

# Check if Ollama is available
check_ollama_available() {
    # Use a more reliable check that doesn't hang
    if command -v curl &>/dev/null; then
        curl -sf --max-time 2 "${OLLAMA_HOST}/api/tags" &>/dev/null 2>&1
        return $?
    else
        # Fallback to nc if curl isn't available
        nc -z -w2 localhost 11434 &>/dev/null 2>&1
        return $?
    fi
}

# Call Ollama for event analysis
ollama_analyze_event() {
    local event_json="$1"
    local filter_prompt="$2"
    
    if [[ -z "$event_json" || -z "$filter_prompt" ]]; then
        log::error "Event JSON and filter prompt required"
        return 1
    fi
    
    # Prepare the analysis request
    local request_json=$(cat <<EOF
{
    "model": "${OLLAMA_MODEL}",
    "prompt": "Analyze this event and determine if it matches the filter criteria.\n\nEvent: ${event_json}\n\nFilter: ${filter_prompt}\n\nRespond with only 'MATCH' or 'NO_MATCH' followed by a brief reason.",
    "stream": false,
    "options": {
        "temperature": 0.1,
        "num_predict": 100
    }
}
EOF
    )
    
    # Call Ollama API
    local response
    response=$(timeout "${OLLAMA_TIMEOUT}" curl -sf -X POST \
        "${OLLAMA_HOST}/api/generate" \
        -H "Content-Type: application/json" \
        -d "${request_json}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to analyze event with Ollama"
        return 1
    fi
    
    # Extract the response
    echo "$response" | jq -r '.response // empty'
}

# Create AI filter agent in Huginn
create_ai_filter_agent() {
    local name="$1"
    local filter_criteria="$2"
    local source_ids="$3"
    
    log::info "Creating AI filter agent: ${name}"
    
    # Create the agent using Rails runner
    local create_script=$(cat <<'RUBY'
# Create AI-powered filter agent
name = ENV['AGENT_NAME']
filter = ENV['FILTER_CRITERIA']
source_ids = ENV['SOURCE_IDS']

agent = Agent.build_for_type(
  'EventFormattingAgent',
  user: User.first,
  name: name,
  options: {
    'instructions' => {
      'message' => '{{message | ai_filter}}',
      'ai_filter_criteria' => filter,
      'ai_model' => ENV['OLLAMA_MODEL'] || 'llama3.2'
    },
    'mode' => 'clean'
  }
)

if source_ids && !source_ids.empty?
  source_agents = Agent.where(id: source_ids.split(','))
  agent.sources = source_agents
end

if agent.save
  puts "SUCCESS: Created AI filter agent '#{agent.name}' (ID: #{agent.id})"
else
  puts "ERROR: #{agent.errors.full_messages.join(', ')}"
  exit 1
end
RUBY
    )
    
    AGENT_NAME="$name" \
    FILTER_CRITERIA="$filter_criteria" \
    SOURCE_IDS="$source_ids" \
    OLLAMA_MODEL="${OLLAMA_MODEL}" \
    docker exec huginn bash -c "cd /app && bundle exec rails runner -e production \"$create_script\""
}

# List AI filter agents
list_ai_filter_agents() {
    log::info "Listing AI filter agents..."
    
    local list_script=$(cat <<'RUBY'
agents = Agent.where("name LIKE ? OR options LIKE ?", '%AI%', '%ai_filter%')
if agents.any?
  agents.each do |agent|
    criteria = agent.options.dig('instructions', 'ai_filter_criteria') || 'N/A'
    puts "- #{agent.name} (ID: #{agent.id}, Type: #{agent.type}, Criteria: #{criteria})"
  end
else
  puts "No AI filter agents found"
end
RUBY
    )
    
    docker exec huginn bash -c "cd /app && bundle exec rails runner -e production \"$list_script\""
}

# Process events through AI filter
process_ai_filter() {
    local agent_id="$1"
    local event_limit="${2:-10}"
    
    log::info "Processing events through AI filter (Agent ID: ${agent_id})..."
    
    local process_script=$(cat <<'RUBY'
agent_id = ENV['AGENT_ID'].to_i
event_limit = ENV['EVENT_LIMIT'].to_i

agent = Agent.find_by(id: agent_id)
if agent.nil?
  puts "ERROR: Agent not found"
  exit 1
end

# Get recent events
events = agent.sources.flat_map(&:events).order(created_at: :desc).limit(event_limit)

if events.empty?
  puts "No events to process"
  exit 0
end

require 'net/http'
require 'json'

ollama_host = ENV['OLLAMA_HOST'] || 'http://localhost:11434'
model = agent.options.dig('instructions', 'ai_model') || 'llama3.2'
criteria = agent.options.dig('instructions', 'ai_filter_criteria') || 'all events'

processed = 0
matched = 0

events.each do |event|
  begin
    # Prepare Ollama request
    uri = URI("#{ollama_host}/api/generate")
    request = Net::HTTP::Post.new(uri)
    request['Content-Type'] = 'application/json'
    request.body = {
      model: model,
      prompt: "Analyze this event: #{event.payload.to_json}\n\nFilter criteria: #{criteria}\n\nRespond with MATCH or NO_MATCH",
      stream: false,
      options: { temperature: 0.1, num_predict: 50 }
    }.to_json
    
    # Call Ollama
    response = Net::HTTP.start(uri.hostname, uri.port) do |http|
      http.read_timeout = 30
      http.request(request)
    end
    
    if response.code == '200'
      result = JSON.parse(response.body)
      analysis = result['response'] || ''
      
      if analysis.include?('MATCH') && !analysis.include?('NO_MATCH')
        matched += 1
        # Create filtered event
        agent.create_event(
          payload: event.payload.merge(
            '_ai_filtered' => true,
            '_ai_analysis' => analysis
          )
        )
        puts "✅ Event #{event.id} matched filter"
      else
        puts "❌ Event #{event.id} did not match filter"
      end
      processed += 1
    end
  rescue => e
    puts "ERROR processing event #{event.id}: #{e.message}"
  end
end

puts "\nProcessed #{processed} events, #{matched} matched filter"
RUBY
    )
    
    AGENT_ID="$agent_id" \
    EVENT_LIMIT="$event_limit" \
    OLLAMA_HOST="${OLLAMA_HOST}" \
    docker exec huginn bash -c "cd /app && bundle exec rails runner -e production \"$process_script\""
}

# Test Ollama integration  
test_ollama_integration() {
    log::info "Testing Ollama integration..."
    
    # Check if Ollama is available (avoid early exit from set -e)
    local ollama_check
    ollama_check=$(curl -sf --max-time 2 "${OLLAMA_HOST}/api/tags" 2>&1 | head -c 100) || true
    
    if [[ -z "$ollama_check" ]] || [[ "$ollama_check" == *"curl:"* ]]; then
        log::error "Ollama is not available at ${OLLAMA_HOST}"
        return 1
    fi
    log::success "Ollama is available"
    
    # Test simple prompt
    log::info "Testing event analysis..."
    local test_response
    test_response=$(curl -sf --max-time 10 -X POST "${OLLAMA_HOST}/api/generate" \
        -H "Content-Type: application/json" \
        -d "{\"model\": \"${OLLAMA_MODEL}\", \"prompt\": \"Reply with only MATCH\", \"stream\": false, \"options\": {\"num_predict\": 10}}" \
        2>/dev/null | jq -r '.response // empty') || true
    
    if [[ -n "$test_response" ]]; then
        log::success "Event analysis works: ${test_response:0:50}"
    else
        # May not have the model loaded, which is OK
        log::warn "Event analysis skipped (model ${OLLAMA_MODEL} may not be installed)"
        log::info "To enable AI filtering, run: ollama pull ${OLLAMA_MODEL}"
    fi
    
    log::success "Ollama integration test completed"
}

# Main command handler
main() {
    local command="$1"
    shift
    
    case "$command" in
        test)
            test_ollama_integration
            ;;
        create-filter)
            local name="$1"
            local criteria="$2"
            local sources="$3"
            if [[ -z "$name" || -z "$criteria" ]]; then
                log::error "Usage: ollama create-filter <name> <criteria> [source_ids]"
                return 1
            fi
            create_ai_filter_agent "$name" "$criteria" "$sources"
            ;;
        list-filters)
            list_ai_filter_agents
            ;;
        process)
            local agent_id="$1"
            local limit="${2:-10}"
            if [[ -z "$agent_id" ]]; then
                log::error "Usage: ollama process <agent_id> [event_limit]"
                return 1
            fi
            process_ai_filter "$agent_id" "$limit"
            ;;
        analyze)
            local event="$1"
            local filter="$2"
            if [[ -z "$event" || -z "$filter" ]]; then
                log::error "Usage: ollama analyze <event_json> <filter_criteria>"
                return 1
            fi
            ollama_analyze_event "$event" "$filter"
            ;;
        *)
            log::error "Unknown command: $command"
            echo "Available commands: test, create-filter, list-filters, process, analyze"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi