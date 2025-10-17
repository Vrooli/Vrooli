#!/usr/bin/env bash
# MinIO Object Storage Parser for Qdrant Embeddings
# Extracts semantic information from MinIO bucket configuration files
#
# Handles:
# - Bucket definitions and policies
# - Access control and permissions
# - Lifecycle rules and retention
# - Versioning and encryption settings
# - Storage class configurations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract bucket metadata
# 
# Gets basic bucket information and settings
#
# Arguments:
#   $1 - Path to MinIO bucket config JSON file
# Returns: JSON with bucket metadata
#######################################
extractor::lib::minio::extract_bucket() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Validate JSON format
    if ! jq empty "$file" 2>/dev/null; then
        log::debug "Invalid JSON format in MinIO config: $file" >&2
        return 1
    fi
    
    # Extract bucket name and basic settings
    local bucket_name=$(jq -r '.bucket_name // .name // "unknown"' "$file" 2>/dev/null)
    local region=$(jq -r '.region // "us-east-1"' "$file" 2>/dev/null)
    local storage_class=$(jq -r '.storage_class // "STANDARD"' "$file" 2>/dev/null)
    
    # Extract versioning settings
    local versioning_enabled=$(jq -r '.versioning.enabled // false' "$file" 2>/dev/null)
    local mfa_delete=$(jq -r '.versioning.mfa_delete // false' "$file" 2>/dev/null)
    
    # Extract encryption settings
    local encryption_enabled=$(jq -r '.encryption.enabled // false' "$file" 2>/dev/null)
    local encryption_type=$(jq -r '.encryption.type // "none"' "$file" 2>/dev/null)
    local kms_key=$(jq -r '.encryption.kms_key // ""' "$file" 2>/dev/null)
    
    # Extract notification settings
    local has_notifications="false"
    if jq -e '.notifications' "$file" >/dev/null 2>/dev/null; then
        has_notifications="true"
    fi
    
    # Extract tags
    local tags=$(jq -c '.tags // {}' "$file" 2>/dev/null)
    local tag_count=$(echo "$tags" | jq 'keys | length' 2>/dev/null || echo "0")
    
    jq -n \
        --arg name "$bucket_name" \
        --arg region "$region" \
        --arg storage_class "$storage_class" \
        --arg versioning "$versioning_enabled" \
        --arg mfa_delete "$mfa_delete" \
        --arg encryption "$encryption_enabled" \
        --arg encryption_type "$encryption_type" \
        --arg kms_key "$kms_key" \
        --arg notifications "$has_notifications" \
        --arg tag_count "$tag_count" \
        --argjson tags "$tags" \
        '{
            bucket_name: $name,
            region: $region,
            storage_class: $storage_class,
            versioning_enabled: ($versioning == "true"),
            mfa_delete_enabled: ($mfa_delete == "true"),
            encryption_enabled: ($encryption == "true"),
            encryption_type: $encryption_type,
            kms_key: $kms_key,
            has_notifications: ($notifications == "true"),
            tag_count: ($tag_count | tonumber),
            tags: $tags
        }'
}

#######################################
# Extract access policies
# 
# Analyzes bucket access policies and permissions
#
# Arguments:
#   $1 - Path to MinIO bucket config file
# Returns: JSON with policy information
#######################################
extractor::lib::minio::extract_policies() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract bucket policy
    local has_bucket_policy="false"
    local policy_statements=0
    local public_read="false"
    local public_write="false"
    
    if jq -e '.policy' "$file" >/dev/null 2>/dev/null; then
        has_bucket_policy="true"
        policy_statements=$(jq '.policy.Statement | length' "$file" 2>/dev/null || echo "0")
        
        # Check for public access patterns
        if jq -e '.policy.Statement[] | select(.Effect == "Allow" and (.Principal == "*" or .Principal.AWS == "*"))' "$file" >/dev/null 2>/dev/null; then
            # Check if it's read access
            if jq -e '.policy.Statement[] | select(.Action | contains("s3:GetObject"))' "$file" >/dev/null 2>/dev/null; then
                public_read="true"
            fi
            # Check if it's write access
            if jq -e '.policy.Statement[] | select(.Action | contains("s3:PutObject"))' "$file" >/dev/null 2>/dev/null; then
                public_write="true"
            fi
        fi
    fi
    
    # Extract CORS configuration
    local has_cors="false"
    local cors_rules=0
    if jq -e '.cors' "$file" >/dev/null 2>/dev/null; then
        has_cors="true"
        cors_rules=$(jq '.cors.CORSRules | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Extract lifecycle rules
    local has_lifecycle="false"
    local lifecycle_rules=0
    if jq -e '.lifecycle' "$file" >/dev/null 2>/dev/null; then
        has_lifecycle="true"
        lifecycle_rules=$(jq '.lifecycle.Rules | length' "$file" 2>/dev/null || echo "0")
    fi
    
    jq -n \
        --arg has_policy "$has_bucket_policy" \
        --arg statements "$policy_statements" \
        --arg public_read "$public_read" \
        --arg public_write "$public_write" \
        --arg has_cors "$has_cors" \
        --arg cors_rules "$cors_rules" \
        --arg has_lifecycle "$has_lifecycle" \
        --arg lifecycle_rules "$lifecycle_rules" \
        '{
            has_bucket_policy: ($has_policy == "true"),
            policy_statements: ($statements | tonumber),
            public_read_access: ($public_read == "true"),
            public_write_access: ($public_write == "true"),
            has_cors_config: ($has_cors == "true"),
            cors_rules: ($cors_rules | tonumber),
            has_lifecycle_rules: ($has_lifecycle == "true"),
            lifecycle_rules: ($lifecycle_rules | tonumber)
        }'
}

#######################################
# Extract notification configuration
# 
# Analyzes event notification settings
#
# Arguments:
#   $1 - Path to MinIO bucket config file
# Returns: JSON with notification information
#######################################
extractor::lib::minio::extract_notifications() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract different notification types
    local has_lambda="false"
    local has_sns="false"
    local has_sqs="false"
    local has_webhook="false"
    
    local lambda_configs=0
    local sns_configs=0
    local sqs_configs=0
    local webhook_configs=0
    
    # Check for Lambda notifications
    if jq -e '.notifications.lambda' "$file" >/dev/null 2>/dev/null; then
        has_lambda="true"
        lambda_configs=$(jq '.notifications.lambda | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Check for SNS notifications
    if jq -e '.notifications.sns' "$file" >/dev/null 2>/dev/null; then
        has_sns="true"
        sns_configs=$(jq '.notifications.sns | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Check for SQS notifications
    if jq -e '.notifications.sqs' "$file" >/dev/null 2>/dev/null; then
        has_sqs="true"
        sqs_configs=$(jq '.notifications.sqs | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Check for webhook notifications
    if jq -e '.notifications.webhook' "$file" >/dev/null 2>/dev/null; then
        has_webhook="true"
        webhook_configs=$(jq '.notifications.webhook | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Extract common event types
    local event_types=()
    if jq -e '.notifications' "$file" >/dev/null 2>/dev/null; then
        while IFS= read -r event; do
            [[ -z "$event" ]] && continue
            event_types+=("$event")
        done < <(jq -r '.notifications | .. | .events? // empty | .[]?' "$file" 2>/dev/null | sort -u)
    fi
    
    local event_types_json="[]"
    if [[ ${#event_types[@]} -gt 0 ]]; then
        event_types_json=$(printf '%s\n' "${event_types[@]}" | jq -R . | jq -s '.')
    fi
    
    jq -n \
        --arg lambda "$has_lambda" \
        --arg sns "$has_sns" \
        --arg sqs "$has_sqs" \
        --arg webhook "$has_webhook" \
        --arg lambda_count "$lambda_configs" \
        --arg sns_count "$sns_configs" \
        --arg sqs_count "$sqs_configs" \
        --arg webhook_count "$webhook_configs" \
        --argjson events "$event_types_json" \
        '{
            has_lambda_notifications: ($lambda == "true"),
            has_sns_notifications: ($sns == "true"),
            has_sqs_notifications: ($sqs == "true"),
            has_webhook_notifications: ($webhook == "true"),
            lambda_configurations: ($lambda_count | tonumber),
            sns_configurations: ($sns_count | tonumber),
            sqs_configurations: ($sqs_count | tonumber),
            webhook_configurations: ($webhook_count | tonumber),
            event_types: $events
        }'
}

#######################################
# Analyze bucket purpose
# 
# Determines bucket usage based on configuration and name
#
# Arguments:
#   $1 - Path to MinIO bucket config file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::minio::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local filename=$(basename "$file" | tr '[:upper:]' '[:lower:]')
    local bucket_name=$(jq -r '.bucket_name // .name // ""' "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Analyze bucket name for hints
    if [[ "$bucket_name" == *"backup"* ]]; then
        purposes+=("backup_storage")
    elif [[ "$bucket_name" == *"log"* ]]; then
        purposes+=("log_storage")
    elif [[ "$bucket_name" == *"media"* ]] || [[ "$bucket_name" == *"asset"* ]]; then
        purposes+=("media_storage")
    elif [[ "$bucket_name" == *"data"* ]]; then
        purposes+=("data_storage")
    elif [[ "$bucket_name" == *"static"* ]] || [[ "$bucket_name" == *"web"* ]]; then
        purposes+=("static_website")
    elif [[ "$bucket_name" == *"archive"* ]]; then
        purposes+=("archive_storage")
    elif [[ "$bucket_name" == *"temp"* ]] || [[ "$bucket_name" == *"tmp"* ]]; then
        purposes+=("temporary_storage")
    elif [[ "$bucket_name" == *"upload"* ]]; then
        purposes+=("file_upload")
    fi
    
    # Analyze configuration for usage patterns
    local storage_class=$(jq -r '.storage_class // ""' "$file" 2>/dev/null)
    case "$storage_class" in
        "GLACIER"|"DEEP_ARCHIVE")
            purposes+=("long_term_archive")
            ;;
        "STANDARD_IA"|"ONEZONE_IA")
            purposes+=("infrequent_access")
            ;;
    esac
    
    # Check for public access (indicates web hosting or CDN)
    if echo "$content" | grep -qE '"effect"[[:space:]]*:[[:space:]]*"allow".*"principal"[[:space:]]*:[[:space:]]*"\*"'; then
        purposes+=("public_hosting")
    fi
    
    # Check for lifecycle rules (indicates data management)
    if jq -e '.lifecycle' "$file" >/dev/null 2>/dev/null; then
        purposes+=("data_lifecycle_management")
    fi
    
    # Check for notifications (indicates event-driven processing)
    if jq -e '.notifications' "$file" >/dev/null 2>/dev/null; then
        purposes+=("event_driven_processing")
    fi
    
    # Check for versioning (indicates important data)
    if jq -r '.versioning.enabled // false' "$file" 2>/dev/null | grep -q "true"; then
        purposes+=("versioned_storage")
    fi
    
    # Check for encryption (indicates sensitive data)
    if jq -r '.encryption.enabled // false' "$file" 2>/dev/null | grep -q "true"; then
        purposes+=("secure_storage")
    fi
    
    # Analyze filename for hints
    if [[ "$filename" == *"config"* ]]; then
        purposes+=("configuration_storage")
    elif [[ "$filename" == *"model"* ]]; then
        purposes+=("model_storage")
    elif [[ "$filename" == *"document"* ]]; then
        purposes+=("document_storage")
    fi
    
    # Determine primary purpose
    local primary_purpose="object_storage"
    if [[ ${#purposes[@]} -gt 0 ]]; then
        primary_purpose="${purposes[0]}"
    fi
    
    local purposes_json=$(printf '%s\n' "${purposes[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson purposes "$purposes_json" \
        --arg primary "$primary_purpose" \
        '{
            purposes: $purposes,
            primary_purpose: $primary
        }'
}

#######################################
# Extract all MinIO bucket information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - MinIO config file path or directory
#   $2 - Component type (bucket, storage, etc.)
#   $3 - Resource name
# Returns: JSON lines with all bucket information
#######################################
extractor::lib::minio::extract_all() {
    local path="$1"
    local component_type="${2:-bucket}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a supported file type
        case "$file_ext" in
            json)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Extract all components
        local bucket=$(extractor::lib::minio::extract_bucket "$file")
        local policies=$(extractor::lib::minio::extract_policies "$file")
        local notifications=$(extractor::lib::minio::extract_notifications "$file")
        local purpose=$(extractor::lib::minio::analyze_purpose "$file")
        
        # Get key metrics
        local bucket_name=$(echo "$bucket" | jq -r '.bucket_name')
        local region=$(echo "$bucket" | jq -r '.region')
        local storage_class=$(echo "$bucket" | jq -r '.storage_class')
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        local versioning=$(echo "$bucket" | jq -r '.versioning_enabled')
        local encryption=$(echo "$bucket" | jq -r '.encryption_enabled')
        
        # Build content summary
        local content="MinIO Bucket: $bucket_name | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose | Region: $region"
        [[ "$storage_class" != "STANDARD" ]] && content="$content | Storage: $storage_class"
        
        # Check for advanced features
        [[ "$versioning" == "true" ]] && content="$content | Versioned"
        [[ "$encryption" == "true" ]] && content="$content | Encrypted"
        
        local has_policy=$(echo "$policies" | jq -r '.has_bucket_policy')
        [[ "$has_policy" == "true" ]] && content="$content | Has Policies"
        
        local has_notifications=$(echo "$notifications" | jq -r '.has_lambda_notifications or .has_sns_notifications or .has_sqs_notifications or .has_webhook_notifications')
        [[ "$has_notifications" == "true" ]] && content="$content | Has Notifications"
        
        # Output comprehensive bucket analysis
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg file_size "$file_size" \
            --argjson bucket "$bucket" \
            --argjson policies "$policies" \
            --argjson notifications "$notifications" \
            --argjson purpose "$purpose" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    storage_type: "minio",
                    file_size: ($file_size | tonumber),
                    bucket: $bucket,
                    policies: $policies,
                    notifications: $notifications,
                    purpose: $purpose,
                    content_type: "minio_bucket",
                    extraction_method: "minio_parser"
                }
            }' | jq -c
            
        # Output entry for bucket access patterns (for better searchability)
        local public_access=$(echo "$policies" | jq -r '.public_read_access or .public_write_access')
        if [[ "$public_access" == "true" ]]; then
            local access_content="MinIO Public Bucket: $bucket_name | Resource: $resource_name | Purpose: $primary_purpose"
            
            jq -n \
                --arg content "$access_content" \
                --arg resource "$resource_name" \
                --arg source_file "$file" \
                --arg bucket_name "$bucket_name" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        bucket_name: $bucket_name,
                        component_type: $component_type,
                        access_level: "public",
                        content_type: "minio_access",
                        extraction_method: "minio_parser"
                    }
                }' | jq -c
        fi
        
    elif [[ -d "$path" ]]; then
        # Directory - find all MinIO config files
        local config_files=()
        while IFS= read -r file; do
            config_files+=("$file")
        done < <(find "$path" -type f -name "*.json" 2>/dev/null)
        
        if [[ ${#config_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${config_files[@]}"; do
            # Basic validation - check if it looks like a MinIO bucket config
            if jq -e '.bucket_name or .name' "$file" >/dev/null 2>/dev/null; then
                extractor::lib::minio::extract_all "$file" "$component_type" "$resource_name"
            fi
        done
    fi
}

#######################################
# Check if file is a MinIO bucket configuration
# 
# Validates if JSON file is a MinIO bucket definition
#
# Arguments:
#   $1 - File path
# Returns: 0 if MinIO config, 1 otherwise
#######################################
extractor::lib::minio::is_bucket_config() {
    local file="$1"
    
    if [[ ! -f "$file" ]] || [[ ! "$file" == *.json ]]; then
        return 1
    fi
    
    # Check for MinIO bucket structure
    if jq -e '.bucket_name or .name' "$file" >/dev/null 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::minio::extract_bucket
export -f extractor::lib::minio::extract_policies
export -f extractor::lib::minio::extract_notifications
export -f extractor::lib::minio::analyze_purpose
export -f extractor::lib::minio::extract_all
export -f extractor::lib::minio::is_bucket_config