#!/bin/sh

# Check for required variables
if [ -z "$PROJECT_DIR" ]; then
  echo "ERROR: PROJECT_DIR environment variable is not set." >&2
  exit 1
fi

# Create a new directory for Redis appendonly if it does not exist
mkdir -p ${PROJECT_DIR}/data/redis/appendonlydir

# Remove the existing Redis database dump file if it exists
rm -f ${PROJECT_DIR}/data/redis/dump.rdb

# Start the Redis server with the specified configuration
# --maxmemory 256mb: Set a maximum memory limit for Redis
# --maxmemory-policy noeviction: Do not evict keys when memory is full; return errors instead
# --appendonly yes: Enable Append Only File persistence mode
# --dbfilename dump.rdb: Name of the database dump file
# --dir ${PROJECT_DIR}/data/redis/: Directory for database dumps
# --requirepass your_password: Set a password for Redis
redis-server --maxmemory 256mb --maxmemory-policy noeviction --appendonly yes --dbfilename dump.rdb --dir ${PROJECT_DIR}/data/redis/ --requirepass ${REDIS_PASSWORD}
