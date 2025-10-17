#!/bin/bash

# Test Resource with Hardcoded Secrets (for testing vulnerability scanner)

# These are intentional hardcoded secrets for testing the scanner
DATABASE_PASSWORD="hardcoded_db_password_123"
API_KEY="sk-test-abcdef1234567890123456789012345678901234567890"
JWT_SECRET="super_secret_jwt_key_for_testing_12345"
REDIS_URL="redis://admin:password123@localhost:6379/0"

# Environment variable usage (should NOT be flagged)
PROPER_PASSWORD="${DB_PASSWORD:-default}"
VAULT_SECRET=$(resource-vault get secret/test/key)

echo "Test resource with intentional hardcoded secrets for scanner testing"