#!/usr/bin/env bash
# Qdrant Configuration Defaults
# All configuration constants and default values

# Default port constants for test reference
if [[ -z "${QDRANT_DEFAULT_PORT:-}" ]]; then
    readonly QDRANT_DEFAULT_PORT="6333"
fi
if [[ -z "${QDRANT_DEFAULT_GRPC_PORT:-}" ]]; then
    readonly QDRANT_DEFAULT_GRPC_PORT="6334"
fi

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
qdrant::export_config() {
    local qdrant_xdg_config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
    local qdrant_xdg_data_home="${XDG_DATA_HOME:-${HOME}/.local/share}"
    local qdrant_xdg_state_home="${XDG_STATE_HOME:-${HOME}/.local/state}"

    # Guard against multiple calls
    if [[ -n "${QDRANT_CONFIG_EXPORTED:-}" ]]; then
        return 0
    fi
    readonly QDRANT_CONFIG_EXPORTED="true"
    
    # Service configuration (only set if not already defined)
    if [[ -z "${QDRANT_PORT:-}" ]]; then
        readonly QDRANT_PORT="${QDRANT_CUSTOM_PORT:-$QDRANT_DEFAULT_PORT}"
    fi
    if [[ -z "${QDRANT_GRPC_PORT:-}" ]]; then
        readonly QDRANT_GRPC_PORT="${QDRANT_CUSTOM_GRPC_PORT:-6334}"
    fi
    if [[ -z "${QDRANT_BASE_URL:-}" ]]; then
        readonly QDRANT_BASE_URL="http://localhost:${QDRANT_PORT}"
    fi
    if [[ -z "${QDRANT_GRPC_URL:-}" ]]; then
        readonly QDRANT_GRPC_URL="grpc://localhost:${QDRANT_GRPC_PORT}"
    fi
    if [[ -z "${QDRANT_CONTAINER_NAME:-}" ]]; then
        readonly QDRANT_CONTAINER_NAME="qdrant"
    fi
    if [[ -z "${QDRANT_DATA_DIR:-}" ]]; then
        readonly QDRANT_DATA_DIR="${RESOURCE_DATA_DIR:-${qdrant_xdg_data_home}/vrooli/resources/qdrant}"
    fi
    if [[ -z "${QDRANT_CONFIG_DIR:-}" ]]; then
        readonly QDRANT_CONFIG_DIR="${RESOURCE_CONFIG_DIR:-${qdrant_xdg_config_home}/vrooli/resources/qdrant}"
    fi
    if [[ -z "${QDRANT_STATE_DIR:-}" ]]; then
        readonly QDRANT_STATE_DIR="${RESOURCE_STATE_DIR:-${qdrant_xdg_state_home}/vrooli/resources/qdrant}"
    fi
    if [[ -z "${QDRANT_SNAPSHOTS_DIR:-}" ]]; then
        readonly QDRANT_SNAPSHOTS_DIR="${QDRANT_STATE_DIR}/snapshots"
    fi
    if [[ -z "${QDRANT_IMAGE:-}" ]]; then
        readonly QDRANT_IMAGE="qdrant/qdrant:latest"
    fi
    if [[ -z "${QDRANT_VERSION:-}" ]]; then
        readonly QDRANT_VERSION="latest"
    fi

    # API Key configuration (only set if not already defined)
    if [[ -z "${QDRANT_API_KEY:-}" ]]; then
        # Use custom API key if provided, otherwise leave empty (no authentication)
        readonly QDRANT_API_KEY="${QDRANT_CUSTOM_API_KEY:-}"
    fi

    # Default collections for Vrooli
    if [[ -z "${QDRANT_DEFAULT_COLLECTIONS:-}" ]]; then
        readonly QDRANT_DEFAULT_COLLECTIONS=(
            "agent_memory"
            "code_embeddings" 
            "document_chunks"
            "conversation_history"
        )
    fi

    # Collection configuration templates
    if [[ -z "${QDRANT_COLLECTION_CONFIGS:-}" ]]; then
        # Format: collection_name:embedding_role_or_vector_size:distance_metric
        readonly QDRANT_COLLECTION_CONFIGS=(
            "agent_memory:embedding.default:Cosine"
            "code_embeddings:embedding.default:Dot"
            "document_chunks:embedding.default:Cosine"
            "conversation_history:embedding.default:Cosine"
        )
    fi

    # Network configuration (only set if not already defined)
    if [[ -z "${QDRANT_NETWORK_NAME:-}" ]]; then
        readonly QDRANT_NETWORK_NAME="qdrant-network"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${QDRANT_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly QDRANT_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${QDRANT_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly QDRANT_HEALTH_CHECK_MAX_ATTEMPTS=12
    fi
    if [[ -z "${QDRANT_API_TIMEOUT:-}" ]]; then
        readonly QDRANT_API_TIMEOUT=10
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${QDRANT_STARTUP_MAX_WAIT:-}" ]]; then
        readonly QDRANT_STARTUP_MAX_WAIT=60
    fi
    if [[ -z "${QDRANT_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly QDRANT_STARTUP_WAIT_INTERVAL=2
    fi
    if [[ -z "${QDRANT_INITIALIZATION_WAIT:-}" ]]; then
        readonly QDRANT_INITIALIZATION_WAIT=10
    fi

    # Qdrant specific settings
    if [[ -z "${QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE:-}" ]]; then
        readonly QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE=20000
    fi
    if [[ -z "${QDRANT_STORAGE_MEMMAP_THRESHOLD:-}" ]]; then
        readonly QDRANT_STORAGE_MEMMAP_THRESHOLD=10000
    fi
    if [[ -z "${QDRANT_STORAGE_INDEXING_THRESHOLD:-}" ]]; then
        readonly QDRANT_STORAGE_INDEXING_THRESHOLD=100
    fi

    # Performance settings
    if [[ -z "${QDRANT_MAX_REQUEST_SIZE_MB:-}" ]]; then
        readonly QDRANT_MAX_REQUEST_SIZE_MB=32
    fi
    if [[ -z "${QDRANT_MAX_WORKERS:-}" ]]; then
        readonly QDRANT_MAX_WORKERS=0  # 0 = auto-detect CPU cores
    fi

    # Storage limits
    if [[ -z "${QDRANT_MIN_DISK_SPACE_GB:-}" ]]; then
        readonly QDRANT_MIN_DISK_SPACE_GB=2
    fi

    # Export for global access
    export QDRANT_DEFAULT_PORT QDRANT_DEFAULT_GRPC_PORT
    export QDRANT_PORT QDRANT_GRPC_PORT QDRANT_BASE_URL QDRANT_GRPC_URL
    export QDRANT_CONTAINER_NAME QDRANT_DATA_DIR QDRANT_CONFIG_DIR QDRANT_STATE_DIR QDRANT_SNAPSHOTS_DIR QDRANT_IMAGE QDRANT_VERSION
    export QDRANT_API_KEY QDRANT_DEFAULT_COLLECTIONS QDRANT_COLLECTION_CONFIGS
    export QDRANT_NETWORK_NAME
    export QDRANT_HEALTH_CHECK_INTERVAL QDRANT_HEALTH_CHECK_MAX_ATTEMPTS
    export QDRANT_API_TIMEOUT QDRANT_STARTUP_MAX_WAIT
    export QDRANT_STARTUP_WAIT_INTERVAL QDRANT_INITIALIZATION_WAIT
    export QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE QDRANT_STORAGE_MEMMAP_THRESHOLD
    export QDRANT_STORAGE_INDEXING_THRESHOLD QDRANT_MAX_REQUEST_SIZE_MB QDRANT_MAX_WORKERS
    export QDRANT_MIN_DISK_SPACE_GB
}
