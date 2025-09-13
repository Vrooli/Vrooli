#!/usr/bin/env bash
################################################################################
# MinIO + PostgreSQL Integration Example
# 
# Demonstrates backing up PostgreSQL databases to MinIO storage
# This example shows how MinIO can serve as a backup storage solution
################################################################################

set -euo pipefail

# Configuration
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minio123}"
BACKUP_BUCKET="postgres-backups"

# PostgreSQL configuration
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-postgres}"
PG_DATABASE="${PG_DATABASE:-vrooli}"

# Load MinIO credentials if available
CREDS_FILE="${HOME}/.minio/config/credentials"
if [[ -f "$CREDS_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$CREDS_FILE"
    MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-$MINIO_ACCESS_KEY}"
    MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-$MINIO_SECRET_KEY}"
fi

echo "=== PostgreSQL to MinIO Backup Example ==="
echo ""

# Function to check if MinIO is available
check_minio() {
    echo "Checking MinIO availability..."
    if timeout 5 curl -sf "${MINIO_ENDPOINT}/minio/health/live" &>/dev/null; then
        echo "✓ MinIO is available at ${MINIO_ENDPOINT}"
        return 0
    else
        echo "✗ MinIO is not available at ${MINIO_ENDPOINT}"
        echo "  Please ensure MinIO is running: vrooli resource minio develop"
        return 1
    fi
}

# Function to check if PostgreSQL is available
check_postgres() {
    echo "Checking PostgreSQL availability..."
    if pg_isready -h "$PG_HOST" -p "$PG_PORT" &>/dev/null; then
        echo "✓ PostgreSQL is available at ${PG_HOST}:${PG_PORT}"
        return 0
    else
        echo "✗ PostgreSQL is not available"
        echo "  Please ensure PostgreSQL is running: vrooli resource postgres develop"
        return 1
    fi
}

# Function to create backup bucket if needed
create_backup_bucket() {
    echo "Ensuring backup bucket exists..."
    
    # Use AWS CLI if available
    if command -v aws &>/dev/null; then
        export AWS_ACCESS_KEY_ID="$MINIO_ACCESS_KEY"
        export AWS_SECRET_ACCESS_KEY="$MINIO_SECRET_KEY"
        
        if aws s3 ls "s3://${BACKUP_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
            echo "✓ Backup bucket already exists"
        else
            if aws s3 mb "s3://${BACKUP_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
                echo "✓ Created backup bucket: ${BACKUP_BUCKET}"
            else
                echo "✗ Failed to create backup bucket"
                return 1
            fi
        fi
    else
        echo "⚠ AWS CLI not available, assuming bucket exists"
    fi
}

# Function to perform database backup
backup_database() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="/tmp/pg_backup_${PG_DATABASE}_${timestamp}.sql.gz"
    
    echo ""
    echo "Creating database backup..."
    
    # Create compressed backup
    if PGPASSWORD="${PG_PASSWORD:-}" pg_dump \
        -h "$PG_HOST" \
        -p "$PG_PORT" \
        -U "$PG_USER" \
        -d "$PG_DATABASE" \
        --no-password 2>/dev/null | gzip > "$backup_file"; then
        
        local file_size=$(du -h "$backup_file" | cut -f1)
        echo "✓ Database backed up to temporary file (${file_size})"
        
        # Upload to MinIO
        echo "Uploading backup to MinIO..."
        
        if command -v aws &>/dev/null; then
            export AWS_ACCESS_KEY_ID="$MINIO_ACCESS_KEY"
            export AWS_SECRET_ACCESS_KEY="$MINIO_SECRET_KEY"
            
            if aws s3 cp "$backup_file" "s3://${BACKUP_BUCKET}/$(basename "$backup_file")" \
                --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
                echo "✓ Backup uploaded to MinIO: ${BACKUP_BUCKET}/$(basename "$backup_file")"
                
                # List recent backups
                echo ""
                echo "Recent backups in MinIO:"
                aws s3 ls "s3://${BACKUP_BUCKET}/" --endpoint-url "$MINIO_ENDPOINT" 2>/dev/null | \
                    tail -5 | while read -r line; do
                    echo "  $line"
                done
            else
                echo "✗ Failed to upload backup to MinIO"
            fi
        else
            echo "✗ AWS CLI required for upload"
        fi
        
        # Cleanup temporary file
        rm -f "$backup_file"
    else
        echo "✗ Failed to create database backup"
        echo "  Ensure you have proper PostgreSQL credentials"
    fi
}

# Main execution
main() {
    echo "This example demonstrates using MinIO as backup storage for PostgreSQL"
    echo "----------------------------------------------------------------------"
    echo ""
    
    # Check prerequisites
    if ! check_minio; then
        exit 1
    fi
    
    if ! check_postgres; then
        exit 1
    fi
    
    # Setup bucket
    create_backup_bucket
    
    # Perform backup
    backup_database
    
    echo ""
    echo "=== Example Complete ==="
    echo ""
    echo "To restore a backup:"
    echo "1. Download from MinIO: aws s3 cp s3://${BACKUP_BUCKET}/backup.sql.gz . --endpoint-url ${MINIO_ENDPOINT}"
    echo "2. Restore to PostgreSQL: gunzip -c backup.sql.gz | psql -h ${PG_HOST} -U ${PG_USER} -d ${PG_DATABASE}"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi