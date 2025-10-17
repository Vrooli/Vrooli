-- Redis configuration script for Scenario Authenticator
-- Sets up Redis for session management and rate limiting

-- Configuration keys
local config = {
    -- Session configuration
    session_ttl = 86400,  -- 24 hours in seconds
    refresh_token_ttl = 604800,  -- 7 days in seconds
    
    -- Rate limiting windows
    rate_limit_window = 60,  -- 1 minute window
    rate_limit_max_requests = 100,  -- Max requests per window
    
    -- Token blacklist TTL (should match JWT expiry)
    blacklist_ttl = 3600,  -- 1 hour
    
    -- Cache TTLs
    user_cache_ttl = 300,  -- 5 minutes
    permission_cache_ttl = 600  -- 10 minutes
}

-- Store configuration in Redis
for key, value in pairs(config) do
    redis.call('SET', 'config:auth:' .. key, value)
end

-- Create rate limit script (stored procedure)
local rate_limit_script = [[
    local key = KEYS[1]
    local limit = tonumber(ARGV[1])
    local window = tonumber(ARGV[2])
    local current = tonumber(ARGV[3])
    
    local current_count = redis.call('GET', key)
    if current_count == false then
        redis.call('SETEX', key, window, 1)
        return {1, limit}
    end
    
    current_count = tonumber(current_count)
    if current_count < limit then
        redis.call('INCR', key)
        return {current_count + 1, limit}
    else
        return {current_count, limit}
    end
]]

-- Register the rate limit script
local script_sha = redis.sha1hex(rate_limit_script)
redis.call('SCRIPT', 'LOAD', rate_limit_script)
redis.call('SET', 'script:rate_limit', script_sha)

-- Create session management script
local session_script = [[
    local session_key = KEYS[1]
    local user_key = KEYS[2]
    local session_data = ARGV[1]
    local ttl = tonumber(ARGV[2])
    
    -- Store session
    redis.call('SETEX', session_key, ttl, session_data)
    
    -- Add to user's session set
    redis.call('SADD', user_key, session_key)
    redis.call('EXPIRE', user_key, ttl)
    
    return 'OK'
]]

-- Register the session script
local session_sha = redis.sha1hex(session_script)
redis.call('SCRIPT', 'LOAD', session_script)
redis.call('SET', 'script:session_create', session_sha)

-- Create token blacklist check script
local blacklist_script = [[
    local token = KEYS[1]
    local ttl = tonumber(ARGV[1])
    
    -- Check if token is blacklisted
    local blacklisted = redis.call('GET', 'blacklist:' .. token)
    if blacklisted then
        return 1
    end
    
    -- Add to blacklist if ARGV[2] is provided
    if ARGV[2] then
        redis.call('SETEX', 'blacklist:' .. token, ttl, '1')
    end
    
    return 0
]]

-- Register the blacklist script
local blacklist_sha = redis.sha1hex(blacklist_script)
redis.call('SCRIPT', 'LOAD', blacklist_script)
redis.call('SET', 'script:token_blacklist', blacklist_sha)

-- Initialize counters
redis.call('SET', 'stats:total_logins', 0)
redis.call('SET', 'stats:total_registrations', 0)
redis.call('SET', 'stats:active_sessions', 0)
redis.call('SET', 'stats:failed_logins', 0)

-- Create indexes for efficient queries
-- User sessions index
redis.call('SET', 'index:sessions:initialized', '1')

-- API key usage tracking
redis.call('SET', 'index:api_keys:initialized', '1')

-- Set initialization timestamp
redis.call('SET', 'auth:initialized', redis.call('TIME')[1])

-- Return success message
return 'Authentication Redis configuration completed'