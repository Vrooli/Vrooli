#!/bin/bash
# Eclipse Ditto Digital Twin Management Functions

set -euo pipefail

# Twin command dispatcher
twin_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        create)
            twin_create "$@"
            ;;
        update)
            twin_update "$@"
            ;;
        query)
            twin_query "$@"
            ;;
        watch)
            twin_watch "$@"
            ;;
        command)
            twin_send_command "$@"
            ;;
        *)
            echo "Error: Unknown twin subcommand: $subcommand" >&2
            echo "Valid subcommands: create, update, query, watch, command" >&2
            exit 1
            ;;
    esac
}

# Create a new digital twin
twin_create() {
    local twin_id="${1:-}"
    local definition="${2:-}"
    
    if [[ -z "$twin_id" ]]; then
        log_error "Twin ID required. Usage: twin create <id> [definition.json]"
        exit 1
    fi
    
    local data
    if [[ -n "$definition" && -f "$definition" ]]; then
        data=$(cat "$definition")
    else
        # Create default twin structure
        data=$(cat << EOF
{
  "thingId": "$twin_id",
  "policyId": "$twin_id",
  "definition": "org.eclipse.ditto:device:1.0.0",
  "attributes": {
    "name": "Digital Twin $twin_id",
    "created": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "status": "active",
    "type": "device"
  },
  "features": {
    "telemetry": {
      "properties": {
        "temperature": null,
        "humidity": null,
        "pressure": null,
        "lastUpdate": null
      }
    },
    "control": {
      "properties": {
        "state": "idle",
        "mode": "auto"
      }
    },
    "metadata": {
      "properties": {
        "location": null,
        "manufacturer": null,
        "model": null,
        "serialNumber": null
      }
    }
  }
}
EOF
)
    fi
    
    log_info "Creating digital twin: $twin_id"
    
    # First create policy
    create_policy "$twin_id"
    
    # Then create thing
    curl -X PUT \
        -H "Content-Type: application/json" \
        -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        -d "$data" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$twin_id" \
        | jq .
}

# Create policy for twin
create_policy() {
    local policy_id="${1:-}"
    
    local policy=$(cat << EOF
{
  "policyId": "$policy_id",
  "entries": {
    "DEFAULT": {
      "subjects": {
        "nginx:${DITTO_USERNAME}": {
          "type": "nginx basic auth user"
        }
      },
      "resources": {
        "thing:/": {
          "grant": ["READ", "WRITE"],
          "revoke": []
        },
        "policy:/": {
          "grant": ["READ", "WRITE"],
          "revoke": []
        },
        "message:/": {
          "grant": ["READ", "WRITE"],
          "revoke": []
        }
      }
    }
  }
}
EOF
)
    
    curl -X PUT \
        -H "Content-Type: application/json" \
        -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        -d "$policy" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/policies/$policy_id" \
        &>/dev/null || true
}

# Update digital twin properties
twin_update() {
    local twin_id="${1:-}"
    local property_path="${2:-}"
    local value="${3:-}"
    
    if [[ -z "$twin_id" || -z "$property_path" ]]; then
        log_error "Usage: twin update <id> <property.path> <value>"
        exit 1
    fi
    
    log_info "Updating twin $twin_id: $property_path = $value"
    
    # Determine if value is JSON or plain string
    if [[ "$value" =~ ^\{.*\}$ ]] || [[ "$value" =~ ^\[.*\]$ ]]; then
        # JSON value
        curl -X PUT \
            -H "Content-Type: application/json" \
            -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
            -d "$value" \
            "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$twin_id/$property_path" \
            | jq .
    else
        # Plain value
        curl -X PUT \
            -H "Content-Type: application/json" \
            -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
            -d "\"$value\"" \
            "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$twin_id/$property_path" \
            | jq .
    fi
}

# Query digital twins
twin_query() {
    local filter="${1:-}"
    
    if [[ -z "$filter" ]]; then
        # List all twins
        log_info "Querying all digital twins..."
        curl -sf \
            -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
            "http://localhost:${DITTO_GATEWAY_PORT}/api/2/search/things" \
            | jq .
    else
        # Query with RQL filter
        log_info "Querying twins with filter: $filter"
        curl -sf \
            -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
            -G --data-urlencode "filter=$filter" \
            "http://localhost:${DITTO_GATEWAY_PORT}/api/2/search/things" \
            | jq .
    fi
}

# Watch digital twin changes via WebSocket
twin_watch() {
    local twin_id="${1:-}"
    
    if [[ -z "$twin_id" ]]; then
        log_error "Twin ID required. Usage: twin watch <id>"
        exit 1
    fi
    
    log_info "Watching changes for twin: $twin_id"
    log_info "Press Ctrl+C to stop watching"
    
    # Check if wscat is available
    if ! command -v wscat &> /dev/null; then
        log_info "Installing wscat for WebSocket support..."
        npm install -g wscat 2>/dev/null || {
            log_error "Failed to install wscat. Install it manually: npm install -g wscat"
            exit 1
        }
    fi
    
    # Connect to WebSocket endpoint
    wscat -c "ws://localhost:${DITTO_GATEWAY_PORT}/ws/2" \
        -H "Authorization: Basic $(echo -n "${DITTO_USERNAME}:${DITTO_PASSWORD}" | base64)" \
        -x "{\"topic\":\"$twin_id/things/twin/events\",\"action\":\"START-SEND-EVENTS\"}"
}

# Send command to digital twin
twin_send_command() {
    local twin_id="${1:-}"
    local feature="${2:-}"
    local command="${3:-}"
    local payload="${4:-{}}"
    
    if [[ -z "$twin_id" || -z "$feature" || -z "$command" ]]; then
        log_error "Usage: twin command <id> <feature> <command> [payload]"
        exit 1
    fi
    
    log_info "Sending command to $twin_id/$feature: $command"
    
    curl -X POST \
        -H "Content-Type: application/json" \
        -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        -d "$payload" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$twin_id/features/$feature/inbox/messages/$command" \
        | jq .
}

# Seed example digital twins
seed_example_twins() {
    log_info "Seeding example digital twins..."
    
    # Industrial sensor twin
    twin_create "industrial:sensor:temp-001" << 'EOF'
{
  "thingId": "industrial:sensor:temp-001",
  "policyId": "industrial:sensor:temp-001",
  "attributes": {
    "name": "Temperature Sensor 001",
    "location": "Factory Floor A",
    "type": "temperature-sensor",
    "manufacturer": "SensorCorp",
    "model": "TC-500"
  },
  "features": {
    "temperature": {
      "properties": {
        "value": 22.5,
        "unit": "celsius",
        "lastUpdate": "2024-01-16T10:00:00Z",
        "status": "normal",
        "threshold": {
          "min": -10,
          "max": 50
        }
      }
    },
    "diagnostics": {
      "properties": {
        "batteryLevel": 85,
        "signalStrength": -45,
        "uptime": 86400,
        "errors": []
      }
    }
  }
}
EOF

    # Vehicle twin
    twin_create "vehicle:car:tesla-001" << 'EOF'
{
  "thingId": "vehicle:car:tesla-001",
  "policyId": "vehicle:car:tesla-001",
  "attributes": {
    "name": "Tesla Model 3",
    "vin": "5YJ3E1EA1JF000001",
    "type": "electric-vehicle",
    "manufacturer": "Tesla",
    "model": "Model 3",
    "year": 2024
  },
  "features": {
    "battery": {
      "properties": {
        "level": 75,
        "range": 280,
        "charging": false,
        "temperature": 25
      }
    },
    "location": {
      "properties": {
        "latitude": 37.7749,
        "longitude": -122.4194,
        "speed": 0,
        "heading": 90
      }
    },
    "climate": {
      "properties": {
        "interiorTemp": 21,
        "targetTemp": 22,
        "hvacOn": false
      }
    }
  }
}
EOF

    # Smart building twin
    twin_create "building:office:main-hq" << 'EOF'
{
  "thingId": "building:office:main-hq",
  "policyId": "building:office:main-hq",
  "attributes": {
    "name": "Main Headquarters",
    "address": "123 Tech Street",
    "type": "smart-building",
    "floors": 5,
    "rooms": 150
  },
  "features": {
    "energy": {
      "properties": {
        "consumption": 450,
        "solar": 120,
        "grid": 330,
        "efficiency": 0.85
      }
    },
    "hvac": {
      "properties": {
        "zones": 12,
        "activeZones": 8,
        "averageTemp": 22,
        "mode": "cooling"
      }
    },
    "security": {
      "properties": {
        "armed": true,
        "cameras": 48,
        "activeCameras": 48,
        "accessPoints": 24,
        "alerts": []
      }
    }
  }
}
EOF

    log_info "Example twins created successfully"
}