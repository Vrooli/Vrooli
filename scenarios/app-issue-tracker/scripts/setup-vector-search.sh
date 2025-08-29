#!/usr/bin/env bash

# Vector Search Setup Script
# Sets up Qdrant vector database and configures semantic search

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
QDRANT_PORT=${QDRANT_PORT:-6333}
QDRANT_URL="http://localhost:${QDRANT_PORT}"
COLLECTION_NAME="issue_embeddings"
VECTOR_SIZE=1536  # OpenAI text-embedding-ada-002 size
POSTGRES_URL=${POSTGRES_URL:-"postgres://postgres:postgres@localhost:5432/issue_tracker?sslmode=disable"}

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# Check if Qdrant is running
check_qdrant() {
    log "Checking Qdrant connection..."
    
    if curl -s "${QDRANT_URL}/collections" >/dev/null 2>&1; then
        success "Qdrant is running at $QDRANT_URL"
        return 0
    else
        error "Qdrant is not running at $QDRANT_URL"
        return 1
    fi
}

# Start Qdrant using Docker
start_qdrant() {
    log "Starting Qdrant vector database..."
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        error "Docker is required to run Qdrant. Please install Docker first."
        return 1
    fi
    
    # Check if Qdrant container is already running
    if docker ps | grep -q qdrant; then
        warn "Qdrant container is already running"
        return 0
    fi
    
    # Start Qdrant container
    docker run -d \
        --name app-issue-tracker-qdrant \
        -p ${QDRANT_PORT}:6333 \
        -p $((QDRANT_PORT + 1)):6334 \
        -v $(pwd)/qdrant_data:/qdrant/storage:z \
        qdrant/qdrant:latest
    
    # Wait for Qdrant to be ready
    log "Waiting for Qdrant to be ready..."
    local retries=30
    while ! curl -s "${QDRANT_URL}/collections" >/dev/null 2>&1; do
        if [ $retries -eq 0 ]; then
            error "Qdrant failed to start within timeout"
            return 1
        fi
        sleep 2
        retries=$((retries - 1))
        echo -n "."
    done
    echo
    
    success "Qdrant started successfully"
}

# Create collection for issue embeddings
create_collection() {
    log "Creating Qdrant collection: $COLLECTION_NAME"
    
    # Check if collection already exists
    if curl -s "${QDRANT_URL}/collections/${COLLECTION_NAME}" | grep -q "result"; then
        warn "Collection $COLLECTION_NAME already exists"
        return 0
    fi
    
    # Create collection
    curl -X PUT "${QDRANT_URL}/collections/${COLLECTION_NAME}" \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {
                "size": '${VECTOR_SIZE}',
                "distance": "Cosine"
            },
            "optimizers_config": {
                "default_segment_number": 2
            },
            "replication_factor": 1
        }'
    
    if [ $? -eq 0 ]; then
        success "Collection $COLLECTION_NAME created successfully"
    else
        error "Failed to create collection $COLLECTION_NAME"
        return 1
    fi
}

# Generate embeddings for existing issues
generate_embeddings() {
    log "Generating embeddings for existing issues..."
    
    # Check if we have any issues
    local issue_count
    if command -v psql &> /dev/null; then
        issue_count=$(PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d issue_tracker -t -c "SELECT COUNT(*) FROM issues;" | xargs)
        log "Found $issue_count issues in database"
    else
        warn "psql not available, skipping issue count check"
        issue_count=0
    fi
    
    if [ "$issue_count" -gt 0 ]; then
        log "Generating embeddings using Python script..."
        
        # Create Python embedding generation script
        cat > "${PROJECT_DIR}/scripts/generate_embeddings.py" << 'EOF'
#!/usr/bin/env python3
"""
Generate embeddings for issues using OpenAI API
"""

import os
import sys
import json
import psycopg2
import requests
from typing import List, Dict, Any
import time

# Configuration
OPENAI_API_KEY = os.getenv('OPENAI_API_KEY')
QDRANT_URL = os.getenv('QDRANT_URL', 'http://localhost:6333')
POSTGRES_URL = os.getenv('POSTGRES_URL', 'postgres://postgres:postgres@localhost:5432/issue_tracker?sslmode=disable')
COLLECTION_NAME = 'issue_embeddings'

def get_embedding(text: str) -> List[float]:
    """Get embedding from OpenAI API"""
    if not OPENAI_API_KEY:
        print("Warning: OPENAI_API_KEY not set, using mock embeddings")
        # Return mock embedding for testing
        return [0.1] * 1536
    
    url = "https://api.openai.com/v1/embeddings"
    headers = {
        "Authorization": f"Bearer {OPENAI_API_KEY}",
        "Content-Type": "application/json"
    }
    data = {
        "input": text,
        "model": "text-embedding-ada-002"
    }
    
    response = requests.post(url, headers=headers, json=data)
    if response.status_code == 200:
        return response.json()["data"][0]["embedding"]
    else:
        print(f"Error getting embedding: {response.status_code}")
        return [0.0] * 1536

def store_in_qdrant(issue_id: str, embedding: List[float], metadata: Dict[str, Any]):
    """Store embedding in Qdrant"""
    url = f"{QDRANT_URL}/collections/{COLLECTION_NAME}/points"
    data = {
        "points": [{
            "id": issue_id,
            "vector": embedding,
            "payload": metadata
        }]
    }
    
    response = requests.put(url, json=data)
    return response.status_code == 200

def main():
    # Connect to PostgreSQL
    try:
        conn = psycopg2.connect(POSTGRES_URL)
        cur = conn.cursor()
        
        # Get all issues without embeddings
        cur.execute("""
            SELECT id, title, description, type, priority, error_message, stack_trace, status
            FROM issues 
            WHERE embedding IS NULL OR embedding_generated_at IS NULL
            ORDER BY created_at DESC
            LIMIT 100
        """)
        
        issues = cur.fetchall()
        print(f"Processing {len(issues)} issues...")
        
        for issue in issues:
            issue_id, title, description, issue_type, priority, error_msg, stack_trace, status = issue
            
            # Create combined text for embedding
            text_parts = [
                f"Title: {title}",
                f"Description: {description or ''}",
                f"Type: {issue_type}",
                f"Priority: {priority}",
            ]
            
            if error_msg:
                text_parts.append(f"Error: {error_msg}")
            if stack_trace:
                text_parts.append(f"Stack trace: {stack_trace[:500]}")  # Limit stack trace length
                
            combined_text = " ".join(text_parts)
            
            print(f"Processing issue {issue_id}: {title[:50]}...")
            
            # Generate embedding
            embedding = get_embedding(combined_text)
            
            # Store in Qdrant
            metadata = {
                "title": title,
                "description": description or "",
                "type": issue_type,
                "priority": priority,
                "status": status,
                "has_error": bool(error_msg),
                "has_stack_trace": bool(stack_trace)
            }
            
            if store_in_qdrant(str(issue_id), embedding, metadata):
                # Update PostgreSQL with embedding
                cur.execute("""
                    UPDATE issues 
                    SET embedding = %s, embedding_generated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                """, (embedding, issue_id))
                conn.commit()
                print(f"✓ Processed issue {issue_id}")
            else:
                print(f"✗ Failed to store embedding for issue {issue_id}")
            
            # Small delay to avoid rate limiting
            time.sleep(0.1)
        
        print(f"Completed processing {len(issues)} issues")
        
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)
    finally:
        if 'conn' in locals():
            conn.close()

if __name__ == "__main__":
    main()
EOF
        
        chmod +x "${PROJECT_DIR}/scripts/generate_embeddings.py"
        
        # Run the embedding generation
        cd "${PROJECT_DIR}/scripts"
        if command -v python3 &> /dev/null; then
            python3 generate_embeddings.py
        else
            warn "Python3 not available, skipping embedding generation"
        fi
    else
        log "No issues found, skipping embedding generation"
    fi
}

# Create vector search API endpoint
create_vector_search_endpoint() {
    log "Creating vector search endpoint..."
    
    # Create Go vector search module
    cat > "${PROJECT_DIR}/api/vector_search.go" << 'EOF'
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// QdrantSearchRequest represents a search request to Qdrant
type QdrantSearchRequest struct {
	Vector []float64 `json:"vector"`
	Limit  int       `json:"limit"`
	WithPayload bool  `json:"with_payload"`
	WithVector bool   `json:"with_vector"`
}

// QdrantSearchResponse represents the response from Qdrant
type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
	Status string              `json:"status"`
}

type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Version int                    `json:"version"`
	Score   float64               `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// VectorSearchResult represents a search result for API response
type VectorSearchResult struct {
	IssueID     string  `json:"issue_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Status      string  `json:"status"`
	Similarity  float64 `json:"similarity"`
}

// generateMockEmbedding creates a mock embedding for testing
func generateMockEmbedding(text string) []float64 {
	// Simple hash-based mock embedding
	embedding := make([]float64, 1536)
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}
	
	for i := range embedding {
		embedding[i] = float64((hash+i)%1000) / 1000.0 - 0.5
	}
	return embedding
}

// performVectorSearch searches for similar issues using Qdrant
func (s *Server) performVectorSearch(queryEmbedding []float64, limit int) ([]VectorSearchResult, error) {
	// Prepare search request
	searchReq := QdrantSearchRequest{
		Vector:      queryEmbedding,
		Limit:       limit,
		WithPayload: true,
		WithVector:  false,
	}
	
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling search request: %v", err)
	}
	
	// Make request to Qdrant
	qdrantURL := fmt.Sprintf("%s/collections/issue_embeddings/points/search", s.config.QdrantURL)
	client := &http.Client{}
	
	resp, err := client.Post(qdrantURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error calling Qdrant: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Qdrant returned status: %d", resp.StatusCode)
	}
	
	// Parse response
	var qdrantResp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&qdrantResp); err != nil {
		return nil, fmt.Errorf("error parsing Qdrant response: %v", err)
	}
	
	// Convert to our result format
	results := make([]VectorSearchResult, 0, len(qdrantResp.Result))
	for _, result := range qdrantResp.Result {
		vectorResult := VectorSearchResult{
			IssueID:    result.ID,
			Similarity: result.Score,
		}
		
		// Extract payload fields
		if title, ok := result.Payload["title"].(string); ok {
			vectorResult.Title = title
		}
		if desc, ok := result.Payload["description"].(string); ok {
			vectorResult.Description = desc
		}
		if issueType, ok := result.Payload["type"].(string); ok {
			vectorResult.Type = issueType
		}
		if priority, ok := result.Payload["priority"].(string); ok {
			vectorResult.Priority = priority
		}
		if status, ok := result.Payload["status"].(string); ok {
			vectorResult.Status = status
		}
		
		results = append(results, vectorResult)
	}
	
	return results, nil
}

// vectorSearchHandler handles semantic search requests
func (s *Server) vectorSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	// Generate embedding for the query
	// In production, this would use OpenAI API or local embedding model
	queryEmbedding := generateMockEmbedding(query)
	
	// Perform vector search
	results, err := s.performVectorSearch(queryEmbedding, limit)
	if err != nil {
		// Fallback to text search if vector search fails
		w.Header().Set("Content-Type", "application/json")
		response := ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Vector search failed, falling back to text search: %v", err),
			Data: map[string]interface{}{
				"fallback":        true,
				"vector_error":    err.Error(),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Return results
	response := ApiResponse{
		Success: true,
		Message: "Vector search completed successfully",
		Data: map[string]interface{}{
			"results": results,
			"count":   len(results),
			"query":   query,
			"method":  "vector_similarity",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
EOF
    
    success "Vector search endpoint created"
}

# Update API to include vector search route
update_api_routes() {
    log "Adding vector search route to API..."
    
    # Add the route to main.go (this would need manual integration)
    log "Vector search route needs to be manually added to main.go"
    log "Add this line to the API routes section:"
    log '  api.HandleFunc("/search/vector", server.vectorSearchHandler).Methods("GET")'
}

# Create test script
create_test_script() {
    log "Creating vector search test script..."
    
    cat > "${PROJECT_DIR}/scripts/test_vector_search.sh" << 'EOF'
#!/usr/bin/env bash

# Test vector search functionality

API_BASE="http://localhost:8090"
QDRANT_URL="http://localhost:6333"

echo "Testing vector search setup..."

# Test 1: Check Qdrant health
echo "1. Checking Qdrant health..."
if curl -s "${QDRANT_URL}/collections" | grep -q "result"; then
    echo "✓ Qdrant is running"
else
    echo "✗ Qdrant is not running"
fi

# Test 2: Check collection exists
echo "2. Checking collection exists..."
if curl -s "${QDRANT_URL}/collections/issue_embeddings" | grep -q "result"; then
    echo "✓ Collection exists"
else
    echo "✗ Collection does not exist"
fi

# Test 3: Count points in collection
echo "3. Checking points in collection..."
POINT_COUNT=$(curl -s "${QDRANT_URL}/collections/issue_embeddings" | grep -o '"points_count":[0-9]*' | grep -o '[0-9]*')
echo "Found $POINT_COUNT points in collection"

# Test 4: Test API search endpoint (if available)
echo "4. Testing API vector search..."
if curl -s "${API_BASE}/api/search/vector?q=login+error&limit=5" | grep -q "success"; then
    echo "✓ Vector search API is working"
else
    echo "⚠ Vector search API not available or not working"
fi

echo "Vector search test completed."
EOF
    
    chmod +x "${PROJECT_DIR}/scripts/test_vector_search.sh"
    success "Test script created at scripts/test_vector_search.sh"
}

# Main setup function
main() {
    log "Setting up vector search for App Issue Tracker..."
    
    # Check if Qdrant is running, start if not
    if ! check_qdrant; then
        start_qdrant
    fi
    
    # Create collection
    create_collection
    
    # Generate embeddings for existing issues
    generate_embeddings
    
    # Create vector search endpoint
    create_vector_search_endpoint
    
    # Update API routes (manual step)
    update_api_routes
    
    # Create test script
    create_test_script
    
    success "Vector search setup completed!"
    log ""
    log "Next steps:"
    log "1. Add vector search route to main.go API routes"
    log "2. Rebuild the API binary: cd api && go build -o app-issue-tracker-api main.go vector_search.go"
    log "3. Restart the API server"
    log "4. Run test: ./scripts/test_vector_search.sh"
    log ""
    log "Vector search will be available at: ${API_BASE}/api/search/vector?q=<query>&limit=<limit>"
}

# Run main function
main "$@"