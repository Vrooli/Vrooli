#!/usr/bin/env bash
# Simple test of direct system execution without the complex loop

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/apis/http-client.sh"

echo "=== Testing Simple Direct Execution ==="

# Simple API request
api_request='{
    "model": "gpt-4o-mini",
    "messages": [
        {
            "role": "system",
            "content": "You are a helpful coding assistant. Write code as requested by the user."
        },
        {
            "role": "user",
            "content": "Write a hello world program in Rust"
        }
    ],
    "temperature": 0.1
}'

echo "Making API call..."
response=$(http_client::request "POST" "/chat/completions" "$api_request")

echo "Extracting content..."
content=$(echo "$response" | jq -r '.choices[0].message.content // "No response"')

echo "RESULT:"
echo "$content"