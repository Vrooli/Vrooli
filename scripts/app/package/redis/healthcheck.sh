#!/bin/sh
# Health check script for Redis that handles empty passwords

if [ -z "${REDIS_PASSWORD}" ]; then
    # No password set, use redis-cli without auth
    redis-cli ping
else
    # Password is set, use redis-cli with auth
    redis-cli -a "${REDIS_PASSWORD}" ping
fi