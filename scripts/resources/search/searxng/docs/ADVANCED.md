# SearXNG Advanced Integration Guide

This guide covers advanced integration patterns, automation examples, and architectural considerations for SearXNG in Vrooli.

## 💻 Programming Language Integration

### Python
```python
import requests
import json

def search_searxng(query, **kwargs):
    """Search using local SearXNG instance"""
    params = {
        'q': query,
        'format': 'json',
        **kwargs
    }
    
    response = requests.get('http://localhost:9200/search', params=params)
    return response.json()

# Example usage
results = search_searxng('quantum computing', categories='science', time_range='month')
for result in results['results'][:5]:
    print(f"• {result['title']}")
    print(f"  {result['url']}")
    print(f"  {result['content'][:100]}...")
    print()
```

### Node.js
```javascript
const axios = require('axios');

async function searchSearXNG(query, options = {}) {
    const params = {
        q: query,
        format: 'json',
        ...options
    };
    
    const response = await axios.get('http://localhost:9200/search', { params });
    return response.data;
}

// Example usage
searchSearXNG('climate change', { categories: 'news', language: 'en' })
    .then(data => {
        data.results.slice(0, 5).forEach(result => {
            console.log(`• ${result.title}`);
            console.log(`  ${result.url}`);
            console.log(`  ${result.content.substring(0, 100)}...`);
            console.log();
        });
    });
```

### Bash/CLI
```bash
#!/bin/bash

# Function to search and format results
search_web() {
    local query="$1"
    local results=$(curl -s "http://localhost:9200/search?q=${query// /+}&format=json")
    
    echo "$results" | jq -r '.results[:5] | .[] | "📰 \(.title)\n   🔗 \(.url)\n   📝 \(.content[:100])...\n"'
}

# Example usage
search_web "renewable energy innovations"
```

## 🔄 Automation Examples

### Shell Script Integration
```bash
#!/bin/bash

# Daily news aggregation
get_daily_news() {
    local topic="$1"
    ./manage.sh --action headlines --topic "$topic" > "news_${topic}_$(date +%Y%m%d).txt"
}

# Research assistant
research_topic() {
    local topic="$1"
    
    # Get overview
    ./manage.sh --action search --query "$topic overview" --limit 5 --output-format compact
    
    # Get recent developments
    ./manage.sh --action search --query "$topic" --time-range month --category news --limit 3
    
    # Get official documentation
    local docs_url=$(./manage.sh --action lucky --query "$topic official documentation")
    echo "Official docs: $docs_url"
}

# Usage
get_daily_news "technology"
research_topic "machine learning"
```

### Cron Job Integration
```bash
# Add to crontab: crontab -e
# Daily news collection at 8 AM
0 8 * * * /path/to/searxng/manage.sh --action headlines --topic "tech" --save ~/daily-tech-news.json

# Hourly monitoring of specific keywords
0 * * * * /path/to/searxng/manage.sh --action search --query "security breach" --time-range hour --append ~/security-alerts.jsonl
```

## 🤖 n8n Integration

### Basic HTTP Request Node
SearXNG can be easily integrated with n8n workflows:

1. **HTTP Request Node Configuration**:
   - Method: GET
   - URL: `http://localhost:9200/search`
   - Query Parameters:
     - q: `{{ $json.searchQuery }}`
     - format: `json`
   - Authentication: None required for local instance

### Advanced n8n Workflow
```json
{
  "name": "Enhanced Search Workflow",
  "nodes": [
    {
      "name": "Get Headlines",
      "type": "n8n-nodes-base.executeCommand",
      "parameters": {
        "command": "./manage.sh",
        "arguments": "--action headlines --topic {{ $json.topic }} --output-format json"
      }
    },
    {
      "name": "Process Results",
      "type": "n8n-nodes-base.function",
      "parameters": {
        "functionCode": "const results = JSON.parse($input.first().json.stdout);\nreturn results.map(item => ({ title: item.title, url: item.url }));"
      }
    }
  ]
}
```

### Complete Search and Summarize Workflow
```json
{
  "name": "Search and Summarize",
  "nodes": [
    {
      "name": "Search Web",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "http://localhost:9200/search",
        "qs": {
          "q": "={{ $json.topic }}",
          "format": "json",
          "categories": "general,news"
        }
      }
    },
    {
      "name": "Extract Content",
      "type": "n8n-nodes-base.function",
      "parameters": {
        "functionCode": "return $input.all().map(item => {\n  const results = item.json.results.slice(0, 5);\n  return results.map(result => ({\n    title: result.title,\n    url: result.url,\n    summary: result.content\n  }));\n}).flat();"
      }
    },
    {
      "name": "AI Summarization",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "http://localhost:11434/api/generate",
        "method": "POST",
        "body": {
          "model": "llama3.1:8b",
          "prompt": "Summarize these search results: {{ $json.summary }}",
          "stream": false
        }
      }
    }
  ]
}
```

## 🏗️ Architecture Integration

### With Vrooli's AI Tiers

SearXNG integrates seamlessly with Vrooli's three-tier AI system:

#### **Tier 1 (Coordination Intelligence)**
- **Strategic Information Discovery**: High-level research planning
- **Knowledge Gap Analysis**: Identifying information needs
- **Source Prioritization**: Determining which information sources to query

```python
# Tier 1 Integration Example
def coordinate_research(research_topic):
    # Identify key areas to research
    subtopics = [
        f"{research_topic} overview",
        f"{research_topic} recent developments", 
        f"{research_topic} technical specifications",
        f"{research_topic} industry applications"
    ]
    
    # Queue searches for Tier 2
    return subtopics
```

#### **Tier 2 (Process Intelligence)**
- **Automated Research Workflows**: Orchestrating search sequences
- **Information Filtering**: Processing and ranking results
- **Context Building**: Connecting related information

```python
# Tier 2 Integration Example  
def execute_research_workflow(subtopics):
    results = {}
    for topic in subtopics:
        # Execute search via SearXNG
        search_results = search_searxng(topic, limit=10)
        
        # Process and filter results
        filtered_results = filter_relevant_results(search_results)
        results[topic] = filtered_results
        
    return consolidate_research(results)
```

#### **Tier 3 (Execution Intelligence)**
- **Direct Search Queries**: Individual search execution
- **Result Processing**: Extracting specific information
- **Data Validation**: Verifying information accuracy

```python
# Tier 3 Integration Example
def execute_search_task(query, parameters):
    # Direct SearXNG API call
    return search_searxng(query, **parameters)
```

### Event-Driven Integration

```python
# Integration with Vrooli's event bus
class SearXNGService:
    def __init__(self, event_bus):
        self.event_bus = event_bus
        self.event_bus.subscribe('search.request', self.handle_search_request)
        
    def handle_search_request(self, event):
        query = event.data['query']
        options = event.data.get('options', {})
        
        # Execute search
        results = search_searxng(query, **options)
        
        # Publish results
        self.event_bus.publish('search.results', {
            'query': query,
            'results': results,
            'source': 'searxng'
        })
```

## 🔗 Multi-Resource Integration

### SearXNG + Ollama (AI Analysis)
```bash
#!/bin/bash
# Search and analyze pipeline

# 1. Search for information
SEARCH_RESULTS=$(./manage.sh --action search --query "$1" --output-format json --limit 5)

# 2. Extract summaries
SUMMARIES=$(echo "$SEARCH_RESULTS" | jq -r '.results[] | .content' | head -5)

# 3. AI analysis via Ollama
curl -X POST http://localhost:11434/api/generate -d "{
    \"model\": \"llama3.1:8b\",
    \"prompt\": \"Analyze and summarize these search results: $SUMMARIES\",
    \"stream\": false
}" | jq -r '.response'
```

### SearXNG + MinIO (Result Storage)
```python
import boto3
from datetime import datetime

def store_search_results(query, results):
    # Store in MinIO for later analysis
    s3_client = boto3.client(
        's3',
        endpoint_url='http://localhost:9000',
        aws_access_key_id='vrooli',
        aws_secret_access_key='password'
    )
    
    timestamp = datetime.now().isoformat()
    filename = f"searches/{timestamp}_{query.replace(' ', '_')}.json"
    
    s3_client.put_object(
        Bucket='vrooli-search-results',
        Key=filename,
        Body=json.dumps(results),
        ContentType='application/json'
    )
```

### SearXNG + Agent-S2 (Visual Verification)
```python
def verify_search_results_visually(url):
    # Use Agent-S2 to take screenshot and verify content
    agent_request = {
        "task": f"take a screenshot of {url} and verify the content matches the search result"
    }
    
    response = requests.post(
        'http://localhost:4113/ai/task',
        json=agent_request
    )
    
    return response.json()
```

## 🚀 Performance Optimization

### Parallel Search Execution
```python
import asyncio
import aiohttp

async def parallel_search(queries):
    async with aiohttp.ClientSession() as session:
        tasks = []
        for query in queries:
            task = session.get(
                'http://localhost:9200/search',
                params={'q': query, 'format': 'json'}
            )
            tasks.append(task)
        
        responses = await asyncio.gather(*tasks)
        return [await r.json() for r in responses]

# Usage
queries = ["AI trends", "machine learning", "quantum computing"]
results = asyncio.run(parallel_search(queries))
```

### Caching Layer
```python
import redis
import json
from datetime import timedelta

class SearXNGCache:
    def __init__(self):
        self.redis_client = redis.Redis(host='localhost', port=6379, db=0)
        self.cache_ttl = timedelta(hours=1)
    
    def get_cached_results(self, query):
        cached = self.redis_client.get(f"search:{query}")
        return json.loads(cached) if cached else None
    
    def cache_results(self, query, results):
        self.redis_client.setex(
            f"search:{query}",
            self.cache_ttl,
            json.dumps(results)
        )
    
    def search_with_cache(self, query, **kwargs):
        # Check cache first
        cached_results = self.get_cached_results(query)
        if cached_results:
            return cached_results
        
        # Execute search
        results = search_searxng(query, **kwargs)
        
        # Cache results
        self.cache_results(query, results)
        
        return results
```

## 📊 Analytics and Monitoring

### Search Analytics
```python
def track_search_metrics(query, results, response_time):
    metrics = {
        'timestamp': datetime.now().isoformat(),
        'query': query,
        'result_count': len(results.get('results', [])),
        'response_time_ms': response_time * 1000,
        'engines_used': list(set(r.get('engine') for r in results.get('results', [])))
    }
    
    # Send to monitoring system
    send_metrics(metrics)
```

### Health Monitoring
```bash
#!/bin/bash
# Health monitoring script

while true; do
    # Test basic functionality
    RESPONSE=$(curl -s -w "%{http_code}" "http://localhost:9200/search?q=test&format=json")
    HTTP_CODE="${RESPONSE: -3}"
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "$(date): SearXNG unhealthy - HTTP $HTTP_CODE"
        # Send alert
    else
        echo "$(date): SearXNG healthy"
    fi
    
    sleep 60
done
```

## 🔐 Security Considerations

### Access Control
```yaml
# Restrict access to local networks only
server:
  bind_address: "127.0.0.1"
  limiter: true  # Enable for production
```

### Query Sanitization
```python
import re

def sanitize_query(query):
    # Remove potentially harmful characters
    sanitized = re.sub(r'[<>"\']', '', query)
    
    # Limit query length
    return sanitized[:500]

def safe_search(query, **kwargs):
    clean_query = sanitize_query(query)
    return search_searxng(clean_query, **kwargs)
```

This advanced integration guide provides the foundation for building sophisticated search-enabled AI workflows within the Vrooli ecosystem.