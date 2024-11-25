-- Lua script for token bucket rate limiting. 
-- This script is used by the rate limiter middleware.
-- It works by storing the current number of tokens in a hash with a key, and refilling the bucket at a constant rate.
-- The script calculates the number of tokens to add based on the time elapsed since the last refill.
-- If the bucket is empty, that means the rate limit has been exceeded and the script returns a wait time until the next token is available.
-- If a token is consumed, the script returns 1 and 0 as the wait time.
--
-- KEYS - keys for the rate limiting bucket
-- ARGV[1, 2, ..., n-1] - pairs of maxTokens and refillRate for each key
-- ARGV[n] - now - the current time in milliseconds since the Unix epoch
-- Returns: 
--   {1, wait_time_ms} if a token was consumed (wait_time_ms will be 0)
--   {0, wait_time_ms} if no token was available, with wait_time_ms indicating milliseconds until next token

-- Enable command replication for Redis cluster
redis.replicate_commands()

-- Input validation
local numKeys = #KEYS
if numKeys < 1 then
    return redis.error_reply("At least 1 key is required")
end

-- Get 'now' from last argument
local now = tonumber(ARGV[#ARGV])
if not now then
    return redis.error_reply("now must be a valid timestamp")
end

-- Initialize result array
local results = {}

-- For each key, process the rate limiting
for i = 1, numKeys do
    local key = KEYS[i]
    local maxTokens = tonumber(ARGV[2*(i-1)+1])
    local refillRate = tonumber(ARGV[2*(i-1)+2])

    if not maxTokens or maxTokens <= 0 then
        return redis.error_reply("maxTokens must be a positive number")
    end
    if not refillRate or refillRate <= 0 then
        return redis.error_reply("refillRate must be a positive number")
    end

    -- Get the current token count and last refill time
    local tokensKey = key .. ':tokens'
    local lastRefillKey = key .. ':lastRefill'
    local state = redis.call('MGET', tokensKey, lastRefillKey)
    local tokens = tonumber(state[1]) or maxTokens
    local lastRefill = tonumber(state[2]) or now

    -- Calculate token refill (using millisecond precision)
    local elapsedTime = math.max(0, now - lastRefill)
    local refillTokens = (elapsedTime / 1000.0) * refillRate
    tokens = math.min(tokens + refillTokens, maxTokens)

    -- Calculate wait time if insufficient tokens
    local waitTime = 0
    local success = tokens >= 1
    if success then
        tokens = tokens - 1
    else
        waitTime = math.ceil((1 - tokens) / refillRate * 1000)
    end

    -- Calculate TTL based on time to refill to max + 24h buffer
    local ttl = math.ceil((maxTokens - tokens) / refillRate + 86400)

    -- Store updated values
    redis.call('SET', tokensKey, tokens, 'EX', ttl)
    redis.call('SET', lastRefillKey, now, 'EX', ttl)

    -- Append result for this key
    table.insert(results, success and 1 or 0)
    table.insert(results, waitTime)
end

return results