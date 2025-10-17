#!/usr/bin/env bash
# Setup Qdrant collections for Web Scraper Manager

set -euo pipefail

echo "🔍 Setting up Qdrant vector collections..."

# Wait for Qdrant to be ready
echo "Waiting for Qdrant to be ready..."
timeout 60 bash -c 'until curl -sf http://localhost:6333/ > /dev/null 2>&1; do sleep 1; done'

# Create collections
echo "Creating vector collections..."

# Collection for scraped content similarity
curl -X PUT "http://localhost:6333/collections/scraped_content" \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 768,
      "distance": "Cosine"
    },
    "optimizers_config": {
      "default_segment_number": 2
    }
  }' || echo "Collection scraped_content may already exist"

# Collection for content deduplication
curl -X PUT "http://localhost:6333/collections/content_dedupe" \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 768,
      "distance": "Cosine"
    },
    "optimizers_config": {
      "default_segment_number": 2
    }
  }' || echo "Collection content_dedupe may already exist"

echo "✅ Qdrant collections configured successfully"