#!/bin/bash
# Airbyte OAuth Authentication Library

set -euo pipefail

# Get OAuth token using client credentials flow
get_oauth_token() {
    local client_id="${1:-}"
    local client_secret="${2:-}"
    
    # If not provided, get from Kubernetes secrets
    if [[ -z "$client_id" ]] || [[ -z "$client_secret" ]]; then
        local auth_data
        auth_data=$(docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl get secret airbyte-auth-secrets -o json 2>/dev/null || echo "{}")
        
        if [[ "$auth_data" != "{}" ]]; then
            client_id=$(echo "$auth_data" | jq -r '.data["instance-admin-client-id"]' | base64 -d 2>/dev/null || echo "")
            client_secret=$(echo "$auth_data" | jq -r '.data["instance-admin-client-secret"]' | base64 -d 2>/dev/null || echo "")
        fi
    fi
    
    # Return empty if no credentials found
    if [[ -z "$client_id" ]] || [[ -z "$client_secret" ]]; then
        echo ""
        return 0
    fi
    
    # Get token from OAuth endpoint
    local token_response
    token_response=$(docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
        curl -s -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "grant_type=client_credentials&client_id=${client_id}&client_secret=${client_secret}" \
        "http://localhost:8001/api/public/v1/oauth2/token" 2>/dev/null || echo "{}")
    
    # Extract access token
    echo "$token_response" | jq -r '.access_token' 2>/dev/null || echo ""
}

# Make authenticated API call
authenticated_api_call() {
    local method="${1}"
    local endpoint="${2}"
    local data="${3:-}"
    
    # Get OAuth token
    local token
    token=$(get_oauth_token)
    
    if [[ -z "$token" ]]; then
        # Fall back to unauthenticated call
        if [[ -n "$data" ]]; then
            docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
                curl -s -X "${method}" -H "Content-Type: application/json" -d "${data}" \
                "http://localhost:8001/api/public/v1/${endpoint}"
        else
            docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
                curl -s "http://localhost:8001/api/public/v1/${endpoint}"
        fi
    else
        # Make authenticated call
        if [[ -n "$data" ]]; then
            docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
                curl -s -X "${method}" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer ${token}" \
                -d "${data}" \
                "http://localhost:8001/api/public/v1/${endpoint}"
        else
            docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
                curl -s -H "Authorization: Bearer ${token}" \
                "http://localhost:8001/api/public/v1/${endpoint}"
        fi
    fi
}

# Test OAuth authentication
test_oauth() {
    echo "Testing OAuth authentication..."
    
    local token
    token=$(get_oauth_token)
    
    if [[ -z "$token" ]]; then
        echo "❌ Could not obtain OAuth token"
        return 1
    fi
    
    echo "✅ OAuth token obtained successfully"
    
    # Test authenticated API call
    local response
    response=$(authenticated_api_call GET "workspaces/list")
    
    if echo "$response" | jq -e '.data' &>/dev/null; then
        echo "✅ Authenticated API call successful"
        return 0
    else
        echo "❌ Authenticated API call failed"
        echo "Response: $response"
        return 1
    fi
}