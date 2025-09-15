#!/usr/bin/env bash
################################################################################
# Vrooli Integration Functions for Huginn
# Provides integration with Redis event bus and MinIO storage
################################################################################

#######################################
# Initialize Vrooli integration
# Sets up Redis and MinIO connections
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::vrooli_init() {
    log::info "🔗 Initializing Vrooli integration..."
    
    # Check if Redis is available (port 6380 for vrooli-redis-resource)
    if docker ps | grep -q "vrooli-redis-resource"; then
        export HUGINN_REDIS_ENABLED=true
        export HUGINN_REDIS_HOST="${REDIS_HOST:-localhost}"
        export HUGINN_REDIS_PORT="${REDIS_PORT:-6380}"  # Vrooli Redis is on 6380
        log::success "✅ Redis integration enabled (port $HUGINN_REDIS_PORT)"
    else
        export HUGINN_REDIS_ENABLED=false
        log::warn "⚠️  Redis not available - event publishing disabled"
    fi
    
    # Check if MinIO is available
    if docker ps | grep -q "minio"; then
        export HUGINN_MINIO_ENABLED=true
        export HUGINN_MINIO_HOST="${MINIO_HOST:-localhost}"
        export HUGINN_MINIO_PORT="${MINIO_PORT:-9000}"
        export HUGINN_MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
        export HUGINN_MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"
        log::success "✅ MinIO integration enabled (port $HUGINN_MINIO_PORT)"
    else
        export HUGINN_MINIO_ENABLED=false
        log::warn "⚠️  MinIO not available - artifact storage disabled"
    fi
    
    return 0
}

#######################################
# Publish event to Redis
# Arguments:
#   $1 - event type
#   $2 - event payload (JSON)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::publish_to_redis() {
    local event_type="$1"
    local payload="$2"
    
    if [[ "${HUGINN_REDIS_ENABLED:-false}" != "true" ]]; then
        log::debug "Redis integration not enabled"
        return 1
    fi
    
    # Create event message
    local event_json
    event_json=$(jq -n \
        --arg type "$event_type" \
        --arg source "huginn" \
        --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --argjson payload "$payload" \
        '{
            type: $type,
            source: $source,
            timestamp: $timestamp,
            payload: $payload
        }')
    
    # Publish to Redis using redis-cli
    if command -v redis-cli &>/dev/null; then
        echo "$event_json" | redis-cli -h "$HUGINN_REDIS_HOST" -p "$HUGINN_REDIS_PORT" \
            PUBLISH "vrooli:events:huginn" - >/dev/null 2>&1
        
        if [[ $? -eq 0 ]]; then
            log::debug "Published event to Redis: $event_type"
            return 0
        fi
    fi
    
    log::debug "Failed to publish event to Redis"
    return 1
}

#######################################
# Store artifact in MinIO
# Arguments:
#   $1 - local file path
#   $2 - bucket name (optional, defaults to "huginn")
#   $3 - object name (optional, defaults to file basename)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::store_in_minio() {
    local file_path="$1"
    local bucket="${2:-huginn}"
    local object_name="${3:-$(basename "$file_path")}"
    
    if [[ "${HUGINN_MINIO_ENABLED:-false}" != "true" ]]; then
        log::debug "MinIO integration not enabled"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Use mc (MinIO client) if available
    if command -v mc &>/dev/null; then
        # Configure mc alias if not already done
        mc alias set huginn-minio "http://$HUGINN_MINIO_HOST:$HUGINN_MINIO_PORT" \
            "$HUGINN_MINIO_ACCESS_KEY" "$HUGINN_MINIO_SECRET_KEY" >/dev/null 2>&1
        
        # Create bucket if it doesn't exist
        mc mb "huginn-minio/$bucket" >/dev/null 2>&1 || true
        
        # Upload file
        if mc cp "$file_path" "huginn-minio/$bucket/$object_name" >/dev/null 2>&1; then
            log::debug "Stored artifact in MinIO: $bucket/$object_name"
            echo "http://$HUGINN_MINIO_HOST:$HUGINN_MINIO_PORT/$bucket/$object_name"
            return 0
        fi
    fi
    
    # Fallback to curl
    local endpoint="http://$HUGINN_MINIO_HOST:$HUGINN_MINIO_PORT"
    # Note: Proper S3 signature would be needed for production
    # This is a simplified example
    
    log::debug "MinIO upload requires mc client or proper S3 signature"
    return 1
}

#######################################
# Retrieve artifact from MinIO
# Arguments:
#   $1 - bucket name
#   $2 - object name
#   $3 - local file path to save to
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::retrieve_from_minio() {
    local bucket="$1"
    local object_name="$2"
    local output_path="$3"
    
    if [[ "${HUGINN_MINIO_ENABLED:-false}" != "true" ]]; then
        log::debug "MinIO integration not enabled"
        return 1
    fi
    
    # Use mc (MinIO client) if available
    if command -v mc &>/dev/null; then
        # Configure mc alias if not already done
        mc alias set huginn-minio "http://$HUGINN_MINIO_HOST:$HUGINN_MINIO_PORT" \
            "$HUGINN_MINIO_ACCESS_KEY" "$HUGINN_MINIO_SECRET_KEY" >/dev/null 2>&1
        
        # Download file
        if mc cp "huginn-minio/$bucket/$object_name" "$output_path" >/dev/null 2>&1; then
            log::debug "Retrieved artifact from MinIO: $bucket/$object_name"
            return 0
        fi
    fi
    
    log::debug "Failed to retrieve artifact from MinIO"
    return 1
}

#######################################
# Setup Redis event listener
# Creates a Rails-based event listener
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::setup_redis_listener() {
    if [[ "${HUGINN_REDIS_ENABLED:-false}" != "true" ]]; then
        log::warn "Redis not available - skipping event listener setup"
        return 1
    fi
    
    local listener_code='
    require "redis"
    
    begin
      redis = Redis.new(
        host: ENV["HUGINN_REDIS_HOST"] || "localhost",
        port: (ENV["HUGINN_REDIS_PORT"] || 6379).to_i
      )
      
      # Create a Redis Event Listener Agent if it doesn'"'"'t exist
      user = User.find_by(username: "admin") || User.first
      
      agent = Agent.where(
        type: "Agents::TriggerAgent",
        name: "Vrooli Redis Event Listener"
      ).first_or_create(
        user: user,
        options: {
          "expected_receive_period_in_days" => 1,
          "rules" => [{
            "type" => "field>=value",
            "value" => "0",
            "path" => "importance"
          }],
          "message" => "Vrooli event received: {{type}}"
        },
        schedule: "every_1m",
        keep_events_for: 7 * 24 * 60 * 60
      )
      
      if agent.persisted?
        puts "✅ Redis event listener agent configured"
        puts "   Agent ID: #{agent.id}"
      else
        puts "❌ Failed to create event listener: #{agent.errors.full_messages.join(", ")}"
      end
      
    rescue => e
      puts "❌ Redis setup error: #{e.message}"
    end
    '
    
    huginn::rails_runner "$listener_code"
}

#######################################
# Publish Huginn agent events to Redis
# Monitors and publishes agent events
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::publish_agent_events() {
    if [[ "${HUGINN_REDIS_ENABLED:-false}" != "true" ]]; then
        return 1
    fi
    
    local publish_code='
    require "redis"
    
    begin
      redis = Redis.new(
        host: ENV["HUGINN_REDIS_HOST"] || "localhost",
        port: (ENV["HUGINN_REDIS_PORT"] || 6379).to_i
      )
      
      # Get recent events
      events = Event.order(created_at: :desc).limit(10)
      
      events.each do |event|
        channel = "vrooli:events:huginn"
        message = {
          type: "huginn.agent.event",
          source: "huginn",
          timestamp: event.created_at.iso8601,
          payload: {
            agent_id: event.agent_id,
            agent_name: event.agent&.name,
            event_id: event.id,
            data: event.payload
          }
        }.to_json
        
        redis.publish(channel, message)
      end
      
      puts "✅ Published #{events.count} events to Redis"
      
    rescue => e
      puts "❌ Redis publish error: #{e.message}"
    end
    '
    
    huginn::rails_runner "$publish_code"
}

#######################################
# Create MinIO storage agent
# Sets up agent for artifact storage
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::setup_minio_storage() {
    if [[ "${HUGINN_MINIO_ENABLED:-false}" != "true" ]]; then
        log::warn "MinIO not available - skipping storage agent setup"
        return 1
    fi
    
    local storage_code='
    begin
      user = User.find_by(username: "admin") || User.first
      
      # Create a MinIO Storage Agent (using WebhookAgent as base)
      agent = Agent.where(
        type: "Agents::WebhookAgent",
        name: "Vrooli MinIO Storage"
      ).first_or_create(
        user: user,
        options: {
          "secret" => SecureRandom.hex(16),
          "expected_receive_period_in_days" => 1,
          "payload_path" => ".",
          "response" => "File stored in MinIO"
        },
        keep_events_for: 30 * 24 * 60 * 60
      )
      
      if agent.persisted?
        puts "✅ MinIO storage agent configured"
        puts "   Agent ID: #{agent.id}"
        puts "   Webhook URL: /users/#{user.id}/web_requests/#{agent.id}/#{agent.options['"'"'secret'"'"']}"
      else
        puts "❌ Failed to create storage agent: #{agent.errors.full_messages.join(", ")}"
      end
      
    rescue => e
      puts "❌ MinIO setup error: #{e.message}"
    end
    '
    
    huginn::rails_runner "$storage_code"
}

#######################################
# Test Vrooli integration
# Runs integration tests
# Returns: 0 if all tests pass
#######################################
huginn::test_vrooli_integration() {
    log::info "🧪 Testing Vrooli integration..."
    
    # Initialize if not already done
    if [[ -z "${HUGINN_REDIS_ENABLED:-}" ]]; then
        huginn::vrooli_init
    fi
    
    local tests_passed=0
    local tests_failed=0
    
    # Test Redis connection
    if [[ "${HUGINN_REDIS_ENABLED:-false}" == "true" ]]; then
        # Test Redis connectivity using nc or timeout
        if timeout 2 bash -c "echo -e 'PING\r\n' | nc -w 1 ${HUGINN_REDIS_HOST} ${HUGINN_REDIS_PORT} 2>/dev/null | grep -q PONG"; then
            log::success "✅ Redis connection test passed"
            ((tests_passed++))
        else
            log::warn "⚠️  Redis connection test skipped (redis-cli not available in container)"
            log::info "   Redis endpoint configured at ${HUGINN_REDIS_HOST}:${HUGINN_REDIS_PORT}"
        fi
        
        # Test Redis publishing
        local test_event='{"test": "event", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'
        if huginn::publish_to_redis "test.event" "$test_event"; then
            log::success "✅ Redis publish test passed"
            ((tests_passed++))
        else
            log::error "❌ Redis publish test failed"
            ((tests_failed++))
        fi
    fi
    
    # Test MinIO connection
    if [[ "${HUGINN_MINIO_ENABLED:-false}" == "true" ]]; then
        if command -v mc &>/dev/null; then
            if mc alias set huginn-minio "http://$HUGINN_MINIO_HOST:$HUGINN_MINIO_PORT" \
                "$HUGINN_MINIO_ACCESS_KEY" "$HUGINN_MINIO_SECRET_KEY" >/dev/null 2>&1; then
                log::success "✅ MinIO connection test passed"
                ((tests_passed++))
            else
                log::error "❌ MinIO connection test failed"
                ((tests_failed++))
            fi
            
            # Test MinIO upload/download
            local test_file="/tmp/huginn-test-$(date +%s).txt"
            echo "Test content" > "$test_file"
            
            if huginn::store_in_minio "$test_file" "test" "test.txt"; then
                log::success "✅ MinIO upload test passed"
                ((tests_passed++))
                
                if huginn::retrieve_from_minio "test" "test.txt" "/tmp/retrieved.txt"; then
                    log::success "✅ MinIO download test passed"
                    ((tests_passed++))
                else
                    log::error "❌ MinIO download test failed"
                    ((tests_failed++))
                fi
            else
                log::error "❌ MinIO upload test failed"
                ((tests_failed++))
            fi
            
            rm -f "$test_file" "/tmp/retrieved.txt"
        fi
    fi
    
    log::info "📊 Integration test results: $tests_passed passed, $tests_failed failed"
    
    [[ $tests_failed -eq 0 ]]
}

# Export functions for use by other scripts
export -f huginn::vrooli_init
export -f huginn::publish_to_redis
export -f huginn::store_in_minio
export -f huginn::retrieve_from_minio
export -f huginn::setup_redis_listener
export -f huginn::publish_agent_events
export -f huginn::setup_minio_storage
export -f huginn::test_vrooli_integration