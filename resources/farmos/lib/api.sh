#!/usr/bin/env bash
# farmOS API Library Functions
# Handles OAuth2 authentication and API CRUD operations

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# OAuth2 token storage
FARMOS_TOKEN_FILE="${HOME}/.farmos/auth_token.json"
FARMOS_TOKEN=""
FARMOS_TOKEN_EXPIRES=""

# Initialize API environment
farmos::api::init() {
    # Create token storage directory
    mkdir -p "$(dirname "${FARMOS_TOKEN_FILE}")"
    
    # Check if service is running
    if ! farmos::health_check; then
        echo "Error: farmOS service is not running"
        return 1
    fi
    
    return 0
}

# Authentication using Basic Auth
# farmOS 3.x API supports both OAuth2 and Basic Auth for simplicity
farmos::api::authenticate() {
    local username="${1:-${FARMOS_ADMIN_USER}}"
    local password="${2:-${FARMOS_ADMIN_PASSWORD}}"
    
    echo "Authenticating with farmOS API..."
    
    # For farmOS 3.x, we'll use Basic Authentication for simplicity
    # Create base64 encoded credentials
    local auth_header=$(echo -n "${username}:${password}" | base64)
    FARMOS_TOKEN="Basic ${auth_header}"
    
    # Test authentication by accessing a protected endpoint
    local test_response
    test_response=$(curl -sf -X GET \
        "${FARMOS_API_BASE}/asset/land" \
        -H "Authorization: ${FARMOS_TOKEN}" \
        -H "Accept: application/vnd.api+json" \
        2>/dev/null)
    
    if [[ -n "$test_response" ]]; then
        # Store token for reuse
        echo "{\"token\": \"${FARMOS_TOKEN}\"}" > "${FARMOS_TOKEN_FILE}"
        echo "Authentication successful"
        return 0
    else
        echo "Error: Failed to authenticate with farmOS"
        return 1
    fi
}

# Check if token is valid
farmos::api::check_token() {
    # Load token from file if not in memory
    if [[ -z "$FARMOS_TOKEN" ]] && [[ -f "${FARMOS_TOKEN_FILE}" ]]; then
        local token_data=$(cat "${FARMOS_TOKEN_FILE}")
        FARMOS_TOKEN=$(echo "$token_data" | jq -r '.token // empty')
    fi
    
    # Check if token exists
    if [[ -z "$FARMOS_TOKEN" ]]; then
        return 1
    fi
    
    # Basic Auth tokens don't expire
    return 0
}

# Ensure valid authentication
farmos::api::ensure_auth() {
    if ! farmos::api::check_token; then
        farmos::api::authenticate
    fi
}

# Generic API request function
farmos::api::request() {
    local method="${1}"
    local endpoint="${2}"
    local data="${3:-}"
    local content_type="${4:-application/json}"
    
    # Ensure authenticated
    farmos::api::ensure_auth
    
    # Build curl command
    local curl_cmd=(curl -sf -X "$method")
    curl_cmd+=(-H "Authorization: ${FARMOS_TOKEN}")  # Supports both "Basic" and "Bearer" prefix
    curl_cmd+=(-H "Content-Type: ${content_type}")
    curl_cmd+=(-H "Accept: application/vnd.api+json")
    
    # Add data if provided
    if [[ -n "$data" ]]; then
        curl_cmd+=(-d "$data")
    fi
    
    # Execute request
    local response
    response=$("${curl_cmd[@]}" "${FARMOS_API_BASE}${endpoint}" 2>/dev/null)
    local exit_code=$?
    
    if [[ $exit_code -ne 0 ]]; then
        echo "Error: API request failed"
        return 1
    fi
    
    echo "$response"
    return 0
}

# CRUD Operations for Fields
farmos::api::field::create() {
    local name="${1}"
    local size="${2:-0}"
    local unit="${3:-acres}"
    local description="${4:-}"
    
    echo "Creating field: $name"
    
    # Build field data
    local field_data=$(cat <<EOF
{
  "data": {
    "type": "land--field",
    "attributes": {
      "name": "$name",
      "status": "active",
      "land_type": "field",
      "field_size": {
        "value": $size,
        "unit": "$unit"
      },
      "notes": {
        "value": "$description",
        "format": "plain_text"
      }
    }
  }
}
EOF
)
    
    local response
    response=$(farmos::api::request "POST" "/land/field" "$field_data")
    
    if [[ $? -eq 0 ]]; then
        local field_id=$(echo "$response" | jq -r '.data.id // empty')
        echo "Field created successfully (ID: $field_id)"
        return 0
    else
        echo "Failed to create field"
        return 1
    fi
}

farmos::api::field::list() {
    echo "Listing fields..."
    
    local response
    response=$(farmos::api::request "GET" "/asset/land")
    
    if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
        # Check if we have data
        local count=$(echo "$response" | jq '.data | length')
        if [[ "$count" -eq 0 ]]; then
            echo "No fields found"
        else
            echo "$response" | jq -r '.data[] | "\(.id): \(.attributes.name)"' 2>/dev/null || echo "Error parsing response"
        fi
        return 0
    else
        echo "Failed to list fields"
        return 1
    fi
}

farmos::api::field::get() {
    local field_id="${1}"
    
    echo "Getting field details: $field_id"
    
    local response
    response=$(farmos::api::request "GET" "/land/field/${field_id}")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.data'
        return 0
    else
        echo "Failed to get field"
        return 1
    fi
}

farmos::api::field::delete() {
    local field_id="${1}"
    
    echo "Deleting field: $field_id"
    
    farmos::api::request "DELETE" "/land/field/${field_id}" ""
    
    if [[ $? -eq 0 ]]; then
        echo "Field deleted successfully"
        return 0
    else
        echo "Failed to delete field"
        return 1
    fi
}

# CRUD Operations for Assets (Equipment, Livestock)
farmos::api::asset::create() {
    local type="${1}"  # equipment, animal, etc.
    local name="${2}"
    local description="${3:-}"
    
    echo "Creating asset: $name (type: $type)"
    
    # Build asset data
    local asset_data=$(cat <<EOF
{
  "data": {
    "type": "asset--${type}",
    "attributes": {
      "name": "$name",
      "status": "active",
      "notes": {
        "value": "$description",
        "format": "plain_text"
      }
    }
  }
}
EOF
)
    
    local response
    response=$(farmos::api::request "POST" "/asset/${type}" "$asset_data")
    
    if [[ $? -eq 0 ]]; then
        local asset_id=$(echo "$response" | jq -r '.data.id // empty')
        echo "Asset created successfully (ID: $asset_id)"
        return 0
    else
        echo "Failed to create asset"
        return 1
    fi
}

farmos::api::asset::list() {
    local type="${1:-}"  # Optional filter by type
    
    echo "Listing assets..."
    
    # Try different asset types
    if [[ -z "$type" ]]; then
        # List all asset types
        for asset_type in land equipment animal plant structure; do
            echo "=== ${asset_type^} Assets ==="
            local response
            response=$(farmos::api::request "GET" "/asset/${asset_type}")
            
            if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
                local count=$(echo "$response" | jq '.data | length' 2>/dev/null)
                if [[ "$count" -gt 0 ]]; then
                    echo "$response" | jq -r '.data[] | "\(.id): \(.attributes.name)"' 2>/dev/null
                else
                    echo "  No ${asset_type} assets found"
                fi
            fi
        done
    else
        # List specific type
        local response
        response=$(farmos::api::request "GET" "/asset/${type}")
        
        if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
            local count=$(echo "$response" | jq '.data | length' 2>/dev/null)
            if [[ "$count" -gt 0 ]]; then
                echo "$response" | jq -r '.data[] | "\(.id): \(.attributes.name)"' 2>/dev/null
            else
                echo "No ${type} assets found"
            fi
            return 0
        else
            echo "Failed to list assets"
            return 1
        fi
    fi
    
    return 0
}

# CRUD Operations for Logs (Activities, Observations)
farmos::api::log::create() {
    local log_type="${1}"  # activity, observation, harvest, etc.
    local name="${2}"
    local field_id="${3:-}"
    local notes="${4:-}"
    local timestamp="${5:-$(date -Iseconds)}"
    
    echo "Creating log: $name (type: $log_type)"
    
    # Build log data
    local log_data=$(cat <<EOF
{
  "data": {
    "type": "log--${log_type}",
    "attributes": {
      "name": "$name",
      "timestamp": "$timestamp",
      "status": "done",
      "notes": {
        "value": "$notes",
        "format": "plain_text"
      }
    },
    "relationships": {}
  }
}
EOF
)
    
    # Add field relationship if provided
    if [[ -n "$field_id" ]]; then
        log_data=$(echo "$log_data" | jq --arg fid "$field_id" '.data.relationships.location = {
            "data": [{
                "type": "land--field",
                "id": $fid
            }]
        }')
    fi
    
    local response
    response=$(farmos::api::request "POST" "/log/${log_type}" "$log_data")
    
    if [[ $? -eq 0 ]]; then
        local log_id=$(echo "$response" | jq -r '.data.id // empty')
        echo "Log created successfully (ID: $log_id)"
        return 0
    else
        echo "Failed to create log"
        return 1
    fi
}

farmos::api::log::list() {
    local log_type="${1:-}"  # Optional filter by type
    local limit="${2:-20}"
    
    echo "Listing logs..."
    
    local endpoint="/log"
    if [[ -n "$log_type" ]]; then
        endpoint="/log/${log_type}"
    fi
    endpoint="${endpoint}?page[limit]=${limit}"
    
    local response
    response=$(farmos::api::request "GET" "$endpoint")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.data[] | "\(.attributes.timestamp): \(.attributes.name) (type: \(.type))"'
        return 0
    else
        echo "Failed to list logs"
        return 1
    fi
}

# Export functionality
farmos::api::export() {
    local entity_type="${1:-all}"  # fields, assets, logs, or all
    local format="${2:-json}"      # json or csv
    local output="${3:-export.${format}}"
    
    echo "Exporting $entity_type to $output..."
    
    # Collect data based on entity type
    local data=""
    
    case "$entity_type" in
        fields|all)
            echo "Fetching fields..."
            local fields=$(farmos::api::request "GET" "/land/field")
            data="${data}${fields}"
            ;;
        assets|all)
            echo "Fetching assets..."
            local assets=$(farmos::api::request "GET" "/asset")
            data="${data}${assets}"
            ;;
        logs|all)
            echo "Fetching logs..."
            local logs=$(farmos::api::request "GET" "/log")
            data="${data}${logs}"
            ;;
        *)
            echo "Unknown entity type: $entity_type"
            return 1
            ;;
    esac
    
    # Format and save data
    if [[ "$format" == "csv" ]]; then
        # Convert JSON to CSV (simplified)
        echo "$data" | jq -r '.data[] | [.id, .type, .attributes.name, .attributes.status] | @csv' > "$output"
    else
        # Save as JSON
        echo "$data" > "$output"
    fi
    
    echo "Export completed: $output"
    return 0
}

# Demo data seeding
farmos::api::seed_demo() {
    echo "Seeding demo farm data..."
    
    # Authenticate first
    farmos::api::authenticate
    
    # Create demo fields
    echo "Creating demo fields..."
    farmos::api::field::create "North Field" "25" "acres" "Main crop field"
    farmos::api::field::create "South Field" "15" "acres" "Secondary field"
    farmos::api::field::create "Greenhouse A" "5000" "sq_ft" "Climate controlled greenhouse"
    
    # Create demo equipment
    echo "Creating demo equipment..."
    farmos::api::asset::create "equipment" "John Deere Tractor" "Model 5075E, 75HP"
    farmos::api::asset::create "equipment" "Combine Harvester" "Model S660"
    farmos::api::asset::create "equipment" "Irrigation System" "Center pivot irrigation"
    
    # Create demo livestock
    echo "Creating demo livestock..."
    farmos::api::asset::create "animal" "Dairy Herd" "20 Holstein cows"
    farmos::api::asset::create "animal" "Chicken Flock" "100 laying hens"
    
    # Create demo activity logs
    echo "Creating demo activity logs..."
    farmos::api::log::create "activity" "Field Preparation" "" "Tilled and prepared North Field"
    farmos::api::log::create "activity" "Planting Corn" "" "Planted corn in North Field"
    farmos::api::log::create "observation" "Soil Test" "" "pH: 6.5, Nitrogen: Medium"
    farmos::api::log::create "harvest" "Corn Harvest" "" "Harvested 150 bushels/acre"
    
    echo "Demo data seeding completed"
    return 0
}