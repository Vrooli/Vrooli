#!/usr/bin/env bash
# Redis Backup and Restore Example
# Demonstrates backup and restore operations

set -euo pipefail

echo "=== Redis Backup and Restore Example ==="
echo

# Create some test data
echo "1. Creating test data..."
resource-redis inject "SET backup:test1 'Important data 1'"
resource-redis inject "SET backup:test2 'Important data 2'"
resource-redis inject "HSET backup:config setting1 value1"
resource-redis inject "HSET backup:config setting2 value2"
resource-redis inject "LPUSH backup:list item1 item2 item3"
echo "Test data created"
echo

# Show current keys
echo "2. Current keys in Redis:"
resource-redis inject "KEYS backup:*"
echo

# Create a backup
echo "3. Creating backup..."
BACKUP_NAME="example-backup-$(date +%Y%m%d-%H%M%S)"
resource-redis backup "$BACKUP_NAME"
echo

# List available backups
echo "4. Available backups:"
resource-redis list-backups
echo

# Simulate data loss - delete the test data
echo "5. Simulating data loss (deleting test data)..."
resource-redis inject "DEL backup:test1 backup:test2 backup:config backup:list"
echo "Data deleted"
echo

# Verify data is gone
echo "6. Verifying data is deleted:"
resource-redis inject "KEYS backup:*"
echo "(Should be empty or show '(empty array)')"
echo

# Restore from backup
echo "7. Restoring from backup: $BACKUP_NAME"
resource-redis restore "$BACKUP_NAME"
echo

# Verify data is restored
echo "8. Verifying restored data:"
resource-redis inject "GET backup:test1"
resource-redis inject "GET backup:test2"
resource-redis inject "HGETALL backup:config"
resource-redis inject "LRANGE backup:list 0 -1"
echo

# Clean up
echo "9. Cleaning up test data..."
resource-redis inject "DEL backup:test1 backup:test2 backup:config backup:list"
echo

echo "=== Example completed successfully ==="
echo "Note: Backup '$BACKUP_NAME' is still available in the backup directory"