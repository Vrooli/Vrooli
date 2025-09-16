#!/bin/bash

# MinIO Replication Management
# Implements multi-instance data replication for MinIO

set -e

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source required libraries
if [[ -f "$RESOURCE_DIR/lib/common.sh" ]]; then
    source "$RESOURCE_DIR/lib/common.sh"
fi
if [[ -f "$RESOURCE_DIR/config/defaults.sh" ]]; then
    source "$RESOURCE_DIR/config/defaults.sh"
fi

# If common functions not available, define minimal ones
if ! command -v load_credentials &>/dev/null; then
    load_credentials() {
        # Load MinIO credentials
        local cred_file="${CONFIG_DIR}/credentials.conf"
        if [[ -f "$cred_file" ]]; then
            export MINIO_ACCESS_KEY=$(grep "^MINIO_ACCESS_KEY=" "$cred_file" | cut -d'=' -f2)
            export MINIO_SECRET_KEY=$(grep "^MINIO_SECRET_KEY=" "$cred_file" | cut -d'=' -f2)
            return 0
        fi
        return 1
    }
fi

# Replication configuration
CONFIG_DIR="${CONFIG_DIR:-$HOME/.minio/config}"
REPLICATION_CONFIG="$CONFIG_DIR/replication.json"
REPLICATION_ALIAS="minio-replica"

# ============================================================================
# REPLICATION SETUP
# ============================================================================

minio::replication::setup() {
    local remote_url="${1:-}"
    local remote_access_key="${2:-}"
    local remote_secret_key="${3:-}"
    local replication_type="${4:-active-active}"  # active-active or active-passive
    
    if [[ -z "$remote_url" || -z "$remote_access_key" || -z "$remote_secret_key" ]]; then
        echo "[ERROR] Usage: setup_replication <remote_url> <access_key> <secret_key> [type]"
        return 1
    fi
    
    echo "[INFO] Setting up replication to $remote_url..."
    
    # Load local credentials
    load_credentials || return 1
    
    # Configure remote MinIO instance
    docker exec minio mc alias set "$REPLICATION_ALIAS" "$remote_url" "$remote_access_key" "$remote_secret_key" || {
        echo "[ERROR] Failed to configure remote MinIO instance"
        return 1
    }
    
    # Verify connectivity to remote
    if ! docker exec minio mc admin info "$REPLICATION_ALIAS" &> /dev/null; then
        echo "[ERROR] Cannot connect to remote MinIO instance at $remote_url"
        return 1
    fi
    
    # Store replication configuration
    cat > "$REPLICATION_CONFIG" << EOF
{
    "remote_url": "$remote_url",
    "remote_alias": "$REPLICATION_ALIAS",
    "type": "$replication_type",
    "enabled": true,
    "created": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    echo "[SUCCESS] Replication configuration saved"
    
    # Enable versioning on both instances (required for replication)
    echo "[INFO] Enabling versioning on local and remote instances..."
    for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
        # Enable on local
        docker exec minio mc version enable "minio-local/$bucket" 2>/dev/null || true
        # Enable on remote
        docker exec minio mc version enable "$REPLICATION_ALIAS/$bucket" 2>/dev/null || true
    done
    
    # Configure site replication
    if [[ "$replication_type" == "active-active" ]]; then
        minio::replication::configure_active_active
    else
        minio::replication::configure_active_passive
    fi
}

# ============================================================================
# ACTIVE-ACTIVE REPLICATION
# ============================================================================

minio::replication::configure_active_active() {
    echo "[INFO] Configuring active-active (bi-directional) replication..."
    
    # Get site names
    local local_site="site-local"
    local remote_site="site-remote"
    
    # Add both sites to replication
    docker exec minio mc admin replicate add \
        "minio-local" \
        "$REPLICATION_ALIAS" \
        --site-name "$local_site" \
        --site-name "$remote_site" || {
        echo "[WARNING] Site replication might already be configured"
    }
    
    echo "[SUCCESS] Active-active replication configured"
}

# ============================================================================
# ACTIVE-PASSIVE REPLICATION
# ============================================================================

minio::replication::configure_active_passive() {
    echo "[INFO] Configuring active-passive (one-way) replication..."
    
    # Configure replication rules for each bucket
    for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
        echo "[INFO] Setting up replication for bucket: $bucket"
        
        # Create bucket on remote if it doesn't exist
        docker exec minio mc mb "$REPLICATION_ALIAS/$bucket" 2>/dev/null || true
        
        # Add replication rule
        docker exec minio mc replicate add \
            "minio-local/$bucket" \
            --remote-bucket "$REPLICATION_ALIAS/$bucket" \
            --priority 1 || {
            echo "[WARNING] Replication rule for $bucket might already exist"
        }
    done
    
    echo "[SUCCESS] Active-passive replication configured"
}

# ============================================================================
# REPLICATION STATUS
# ============================================================================

minio::replication::status() {
    echo "MinIO Replication Status"
    echo "========================"
    
    # Check if replication is configured
    if [[ ! -f "$REPLICATION_CONFIG" ]]; then
        echo "❌ Replication not configured"
        echo ""
        echo "To configure replication, use:"
        echo "  resource-minio replication setup <remote_url> <access_key> <secret_key>"
        return 0
    fi
    
    # Load configuration
    local remote_url=$(jq -r '.remote_url' "$REPLICATION_CONFIG")
    local replication_type=$(jq -r '.type' "$REPLICATION_CONFIG")
    local enabled=$(jq -r '.enabled' "$REPLICATION_CONFIG")
    
    echo "Configuration:"
    echo "  Remote URL: $remote_url"
    echo "  Type: $replication_type"
    echo "  Enabled: $enabled"
    echo ""
    
    if [[ "$enabled" == "false" ]]; then
        echo "⚠️ Replication is disabled"
        return 0
    fi
    
    # Check site replication status
    if [[ "$replication_type" == "active-active" ]]; then
        echo "Site Replication Status:"
        docker exec minio mc admin replicate info minio-local 2>/dev/null || {
            echo "  ❌ Site replication not active"
        }
    else
        # Check bucket replication status
        echo "Bucket Replication Status:"
        for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
            echo "  $bucket:"
            docker exec minio mc replicate status "minio-local/$bucket" 2>/dev/null || {
                echo "    ❌ No replication configured"
            }
        done
    fi
    
    # Check replication metrics
    echo ""
    echo "Replication Metrics:"
    docker exec minio mc admin replicate status minio-local 2>/dev/null || {
        echo "  No metrics available"
    }
}

# ============================================================================
# REPLICATION CONTROL
# ============================================================================

minio::replication::enable() {
    if [[ ! -f "$REPLICATION_CONFIG" ]]; then
        echo "[ERROR] Replication not configured"
        return 1
    fi
    
    echo "[INFO] Enabling replication..."
    jq '.enabled = true' "$REPLICATION_CONFIG" > "${REPLICATION_CONFIG}.tmp" && \
        mv "${REPLICATION_CONFIG}.tmp" "$REPLICATION_CONFIG"
    
    # Resume all replication rules
    for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
        docker exec minio mc replicate resume "minio-local/$bucket" 2>/dev/null || true
    done
    
    echo "[SUCCESS] Replication enabled"
}

minio::replication::disable() {
    if [[ ! -f "$REPLICATION_CONFIG" ]]; then
        echo "[ERROR] Replication not configured"
        return 1
    fi
    
    echo "[INFO] Disabling replication..."
    jq '.enabled = false' "$REPLICATION_CONFIG" > "${REPLICATION_CONFIG}.tmp" && \
        mv "${REPLICATION_CONFIG}.tmp" "$REPLICATION_CONFIG"
    
    # Pause all replication rules
    for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
        docker exec minio mc replicate pause "minio-local/$bucket" 2>/dev/null || true
    done
    
    echo "[SUCCESS] Replication disabled"
}

# ============================================================================
# REPLICATION SYNC
# ============================================================================

minio::replication::sync() {
    local direction="${1:-both}"  # push, pull, or both
    
    if [[ ! -f "$REPLICATION_CONFIG" ]]; then
        echo "[ERROR] Replication not configured"
        return 1
    fi
    
    local remote_alias=$(jq -r '.remote_alias' "$REPLICATION_CONFIG")
    
    echo "[INFO] Starting manual sync (direction: $direction)..."
    
    for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
        echo "[INFO] Syncing bucket: $bucket"
        
        case "$direction" in
            push)
                # Sync from local to remote
                docker exec minio mc mirror \
                    "minio-local/$bucket" \
                    "$remote_alias/$bucket" \
                    --overwrite \
                    --preserve || {
                    echo "[ERROR] Failed to push $bucket"
                }
                ;;
            pull)
                # Sync from remote to local
                docker exec minio mc mirror \
                    "$remote_alias/$bucket" \
                    "minio-local/$bucket" \
                    --overwrite \
                    --preserve || {
                    echo "[ERROR] Failed to pull $bucket"
                }
                ;;
            both)
                # Bi-directional sync (newer wins)
                docker exec minio mc mirror \
                    "minio-local/$bucket" \
                    "$remote_alias/$bucket" \
                    --overwrite \
                    --newer-than 1s || {
                    echo "[WARNING] Push sync for $bucket encountered issues"
                }
                docker exec minio mc mirror \
                    "$remote_alias/$bucket" \
                    "minio-local/$bucket" \
                    --overwrite \
                    --newer-than 1s || {
                    echo "[WARNING] Pull sync for $bucket encountered issues"
                }
                ;;
            *)
                echo "[ERROR] Invalid sync direction: $direction (use: push, pull, or both)"
                return 1
                ;;
        esac
    done
    
    echo "[SUCCESS] Manual sync completed"
}

# ============================================================================
# REPLICATION MONITORING
# ============================================================================

minio::replication::monitor() {
    local interval="${1:-5}"  # Default 5 second refresh
    
    if [[ ! -f "$REPLICATION_CONFIG" ]]; then
        echo "[ERROR] Replication not configured"
        return 1
    fi
    
    echo "Monitoring MinIO Replication (refresh every ${interval}s, Ctrl+C to stop)..."
    echo ""
    
    while true; do
        clear
        echo "MinIO Replication Monitor - $(date)"
        echo "====================================="
        echo ""
        
        # Show replication lag
        echo "Replication Lag:"
        for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
            local lag=$(docker exec minio mc replicate status "minio-local/$bucket" 2>/dev/null | \
                grep -oP 'Lag: \K[^\s]+' || echo "N/A")
            printf "  %-30s: %s\n" "$bucket" "$lag"
        done
        echo ""
        
        # Show pending operations
        echo "Pending Operations:"
        docker exec minio mc admin replicate status minio-local --json 2>/dev/null | \
            jq -r '.[] | "\(.bucket): \(.pendingSize) bytes pending"' 2>/dev/null || \
            echo "  No pending operations"
        
        echo ""
        echo "Press Ctrl+C to stop monitoring..."
        sleep "$interval"
    done
}

# ============================================================================
# FAILOVER MANAGEMENT
# ============================================================================

minio::replication::failover() {
    local action="${1:-status}"  # status, promote, demote
    
    if [[ ! -f "$REPLICATION_CONFIG" ]]; then
        echo "[ERROR] Replication not configured"
        return 1
    fi
    
    local replication_type=$(jq -r '.type' "$REPLICATION_CONFIG")
    
    if [[ "$replication_type" != "active-passive" ]]; then
        echo "[ERROR] Failover only supported for active-passive replication"
        return 1
    fi
    
    case "$action" in
        status)
            echo "Failover Status:"
            echo "  Current role: PRIMARY"
            echo "  Replication: $(jq -r '.enabled' "$REPLICATION_CONFIG")"
            ;;
        promote)
            echo "[INFO] Promoting local instance to primary..."
            enable_replication
            echo "[SUCCESS] Local instance promoted to primary"
            ;;
        demote)
            echo "[INFO] Demoting local instance from primary..."
            disable_replication
            echo "[SUCCESS] Local instance demoted from primary"
            ;;
        *)
            echo "[ERROR] Invalid failover action: $action (use: status, promote, or demote)"
            return 1
            ;;
    esac
}

# ============================================================================
# CLEANUP
# ============================================================================

minio::replication::cleanup() {
    echo "[INFO] Removing replication configuration..."
    
    # Remove all replication rules
    for bucket in vrooli-user-uploads vrooli-agent-artifacts vrooli-model-cache vrooli-temp-storage; do
        docker exec minio mc replicate rm "minio-local/$bucket" 2>/dev/null || true
    done
    
    # Remove remote alias
    docker exec minio mc alias rm "$REPLICATION_ALIAS" 2>/dev/null || true
    
    # Remove configuration file
    rm -f "$REPLICATION_CONFIG"
    
    echo "[SUCCESS] Replication configuration removed"
}

# Functions are available when sourced