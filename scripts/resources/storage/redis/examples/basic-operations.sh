#!/usr/bin/env bash
# Redis Basic Operations Example
# Demonstrates common Redis operations using the resource CLI

set -euo pipefail

echo "=== Redis Basic Operations Example ==="
echo

# Check Redis status
echo "1. Checking Redis status..."
resource-redis status
echo

# Test connection
echo "2. Testing Redis connection..."
resource-redis ping
echo

# Set a key-value pair
echo "3. Setting a key-value pair..."
resource-redis inject SET example:key "Hello_Redis"
echo

# Get the value
echo "4. Getting the value..."
resource-redis inject GET example:key
echo

# Set a key with expiration (10 seconds)
echo "5. Setting a key with 10 second expiration..."
resource-redis inject SETEX example:temp 10 "Temporary_value"
resource-redis inject TTL example:temp
echo

# Work with lists
echo "6. Working with lists..."
resource-redis inject LPUSH example:list item1
resource-redis inject LPUSH example:list item2
resource-redis inject RPUSH example:list item3
resource-redis inject LRANGE example:list 0 -1
echo

# Work with sets
echo "7. Working with sets..."
resource-redis inject SADD example:set member1
resource-redis inject SADD example:set member2
resource-redis inject SADD example:set member3
resource-redis inject SMEMBERS example:set
echo

# Work with hashes
echo "8. Working with hashes..."
resource-redis inject HSET example:user name "John_Doe"
resource-redis inject HSET example:user email "john@example.com"
resource-redis inject HSET example:user age 30
resource-redis inject HGETALL example:user
echo

# Show Redis info
echo "9. Redis server info..."
resource-redis info | head -20
echo

# Clean up example keys
echo "10. Cleaning up example keys..."
resource-redis inject DEL example:key example:temp example:list example:set example:user
echo

echo "=== Example completed successfully ==="