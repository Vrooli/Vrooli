#!/bin/bash

# Script to add more algorithm implementations via API

API_URL="http://localhost:16890"

# Function to add an implementation
add_implementation() {
    local algo_name=$1
    local language=$2
    local code=$3
    
    # First get the algorithm ID
    ALGO_ID=$(curl -s "${API_URL}/api/v1/algorithms/search?q=${algo_name}" | jq -r '.algorithms[0].id')
    
    if [ "$ALGO_ID" != "null" ] && [ -n "$ALGO_ID" ]; then
        # Check if implementation already exists
        EXISTING=$(curl -s "${API_URL}/api/v1/algorithms/${ALGO_ID}/implementations" | jq -r ".implementations[] | select(.language==\"${language}\") | .id")
        
        if [ -z "$EXISTING" ]; then
            echo "Adding ${language} implementation for ${algo_name}"
            
            # Create payload
            cat > /tmp/impl_payload.json << EOJSON
{
    "algorithm_id": "${ALGO_ID}",
    "language": "${language}",
    "code": $(echo "$code" | jq -Rs .),
    "is_primary": true,
    "validated": true,
    "version": "1.0.0"
}
EOJSON
            
            # Submit implementation (would need endpoint)
            echo "Would add implementation for ${algo_name} in ${language}"
        else
            echo "${algo_name} already has ${language} implementation"
        fi
    else
        echo "Algorithm ${algo_name} not found"
    fi
}

# Add Go implementation for insertion_sort
add_implementation "insertion_sort" "go" 'func insertionSort(arr []int) []int {
    result := make([]int, len(arr))
    copy(result, arr)

    for i := 1; i < len(result); i++ {
        key := result[i]
        j := i - 1
        for j >= 0 && result[j] > key {
            result[j+1] = result[j]
            j--
        }
        result[j+1] = key
    }
    return result
}'

echo "Script complete"
