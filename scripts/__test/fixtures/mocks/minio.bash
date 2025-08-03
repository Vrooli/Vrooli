#!/usr/bin/env bash
# MinIO Resource Mock Implementation
# Provides realistic mock responses for MinIO object storage service

# Prevent duplicate loading
if [[ "${MINIO_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export MINIO_MOCK_LOADED="true"

#######################################
# Setup MinIO mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::minio::setup() {
    local state="${1:-healthy}"
    
    # Configure MinIO-specific environment
    export MINIO_PORT="${MINIO_PORT:-9000}"
    export MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"
    export MINIO_BASE_URL="http://localhost:${MINIO_PORT}"
    export MINIO_CONSOLE_URL="http://localhost:${MINIO_CONSOLE_PORT}"
    export MINIO_CONTAINER_NAME="${TEST_NAMESPACE}_minio"
    export MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
    export MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-minioadmin}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$MINIO_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::minio::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::minio::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::minio::setup_installing_endpoints
            ;;
        "stopped")
            mock::minio::setup_stopped_endpoints
            ;;
        *)
            echo "[MINIO_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[MINIO_MOCK] MinIO mock configured with state: $state"
}

#######################################
# Setup healthy MinIO endpoints
#######################################
mock::minio::setup_healthy_endpoints() {
    # Health endpoints
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/health/live" \
        '{"status":"ok"}'
    
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/health/ready" \
        '{"status":"ok"}'
    
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/health/cluster" \
        '{
            "status": "ok",
            "nodes": [
                {
                    "endpoint": "localhost:9000",
                    "status": "online",
                    "drives": [
                        {
                            "uuid": "abc-123",
                            "state": "ok",
                            "totalspace": 1099511627776,
                            "usedspace": 549755813888
                        }
                    ]
                }
            ]
        }'
    
    # S3 API endpoints
    # List buckets
    mock::http::set_endpoint_response "$MINIO_BASE_URL/" \
        '<?xml version="1.0" encoding="UTF-8"?>
        <ListAllMyBucketsResult>
            <Owner>
                <ID>minio</ID>
                <DisplayName>MinIO User</DisplayName>
            </Owner>
            <Buckets>
                <Bucket>
                    <Name>test-bucket</Name>
                    <CreationDate>2024-01-15T10:00:00.000Z</CreationDate>
                </Bucket>
                <Bucket>
                    <Name>backup-bucket</Name>
                    <CreationDate>2024-01-14T09:00:00.000Z</CreationDate>
                </Bucket>
            </Buckets>
        </ListAllMyBucketsResult>'
    
    # Bucket operations
    mock::http::set_endpoint_response "$MINIO_BASE_URL/test-bucket" \
        '<?xml version="1.0" encoding="UTF-8"?>
        <ListBucketResult>
            <Name>test-bucket</Name>
            <Prefix></Prefix>
            <MaxKeys>1000</MaxKeys>
            <IsTruncated>false</IsTruncated>
            <Contents>
                <Key>document.pdf</Key>
                <LastModified>2024-01-15T12:00:00.000Z</LastModified>
                <ETag>"d41d8cd98f00b204e9800998ecf8427e"</ETag>
                <Size>1048576</Size>
                <StorageClass>STANDARD</StorageClass>
            </Contents>
            <Contents>
                <Key>image.jpg</Key>
                <LastModified>2024-01-15T11:00:00.000Z</LastModified>
                <ETag>"098f6bcd4621d373cade4e832627b4f6"</ETag>
                <Size>524288</Size>
                <StorageClass>STANDARD</StorageClass>
            </Contents>
        </ListBucketResult>'
    
    # Create bucket response
    mock::http::set_endpoint_response "$MINIO_BASE_URL/new-bucket" \
        '' \
        "PUT" \
        "200"
    
    # Admin API endpoints
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/admin/v3/info" \
        '{
            "mode": "standalone",
            "deploymentID": "abc-123-def",
            "buckets": 2,
            "objects": 100,
            "usage": 104857600,
            "services": {
                "vault": {"status": "disabled"},
                "ldap": {"status": "disabled"},
                "logger": {"status": "enabled"}
            }
        }'
}

#######################################
# Setup unhealthy MinIO endpoints
#######################################
mock::minio::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/health/live" \
        '{"status":"error"}' \
        "GET" \
        "503"
    
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/health/ready" \
        '{"status":"error","error":"Storage backend unavailable"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing MinIO endpoints
#######################################
mock::minio::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/health/live" \
        '{"status":"starting"}'
    
    # Ready endpoint fails during install
    mock::http::set_endpoint_response "$MINIO_BASE_URL/minio/health/ready" \
        '{"status":"initializing","progress":60}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped MinIO endpoints
#######################################
mock::minio::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$MINIO_BASE_URL"
    mock::http::set_endpoint_unreachable "$MINIO_CONSOLE_URL"
}

#######################################
# Mock MinIO-specific operations
#######################################

# Mock file upload
mock::minio::upload_object() {
    local bucket="$1"
    local object_key="$2"
    local size="${3:-1048576}"
    
    echo '<?xml version="1.0" encoding="UTF-8"?>
    <CompleteMultipartUploadResult>
        <Location>http://localhost:9000/'$bucket'/'$object_key'</Location>
        <Bucket>'$bucket'</Bucket>
        <Key>'$object_key'</Key>
        <ETag>"'$(echo "$object_key" | md5sum | cut -d' ' -f1)'"</ETag>
    </CompleteMultipartUploadResult>'
}

# Mock presigned URL generation
mock::minio::generate_presigned_url() {
    local bucket="$1"
    local object_key="$2"
    local expiry="${3:-3600}"
    
    local signature=$(echo -n "${bucket}/${object_key}" | md5sum | cut -d' ' -f1)
    echo "http://localhost:9000/${bucket}/${object_key}?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Expires=${expiry}&X-Amz-Signature=${signature}"
}

# Mock bucket policy
mock::minio::set_bucket_policy() {
    local bucket="$1"
    local policy="${2:-public-read}"
    
    echo '{
        "status": "ok",
        "bucket": "'$bucket'",
        "policy": "'$policy'",
        "updated_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }'
}

# Mock multipart upload
mock::minio::initiate_multipart_upload() {
    local bucket="$1"
    local object_key="$2"
    
    echo '<?xml version="1.0" encoding="UTF-8"?>
    <InitiateMultipartUploadResult>
        <Bucket>'$bucket'</Bucket>
        <Key>'$object_key'</Key>
        <UploadId>upload-'$(date +%s)'</UploadId>
    </InitiateMultipartUploadResult>'
}

#######################################
# Export mock functions
#######################################
export -f mock::minio::setup
export -f mock::minio::setup_healthy_endpoints
export -f mock::minio::setup_unhealthy_endpoints
export -f mock::minio::setup_installing_endpoints
export -f mock::minio::setup_stopped_endpoints
export -f mock::minio::upload_object
export -f mock::minio::generate_presigned_url
export -f mock::minio::set_bucket_policy
export -f mock::minio::initiate_multipart_upload

echo "[MINIO_MOCK] MinIO mock implementation loaded"