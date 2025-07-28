# SearXNG - Privacy-Respecting Metasearch Engine

SearXNG is a privacy-respecting metasearch engine that aggregates results from multiple search engines without tracking users. This resource provides a local instance perfect for AI agents and automation workflows.

## 🚀 Quick Start

```bash
# Install SearXNG
./manage.sh --action install

# Check status
./manage.sh --action status

# View logs
./manage.sh --action logs

# Enhanced search capabilities
./manage.sh --action search --query "AI" --limit 5 --output-format title-only
./manage.sh --action headlines --topic "tech"
./manage.sh --action lucky --query "Python documentation"
```

## 🔌 API Usage

SearXNG provides a powerful JSON API for programmatic access, perfect for integration with Vrooli's AI agents.

### Basic Search

```bash
# Simple search
curl "http://localhost:8100/search?q=artificial+intelligence&format=json"

# Search with specific parameters
curl "http://localhost:8100/search?q=tech+news&format=json&categories=general&language=en"
```

### API Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `q` | Search query (required) | `q=machine+learning` |
| `format` | Response format | `format=json` |
| `categories` | Search categories | `categories=general,news` |
| `language` | Language preference | `language=en` |
| `pageno` | Page number | `pageno=2` |
| `time_range` | Time filter | `time_range=day` |
| `safesearch` | Safe search level | `safesearch=0` |

### Response Format

```json
{
  "query": "AI breakthrough",
  "number_of_results": 1000000,
  "results": [
    {
      "title": "Latest AI Breakthrough...",
      "url": "https://example.com/article",
      "content": "Description of the result...",
      "engine": "google",
      "score": 1.0,
      "category": "general"
    }
  ],
  "suggestions": ["AI breakthrough 2025", "AI research"],
  "answers": [],
  "infoboxes": []
}
```

## 🎯 Enhanced Management Script

The `manage.sh` script provides powerful search capabilities beyond basic API calls, designed for AI agents and automation workflows.

### Advanced Search Parameters

```bash
# Basic search with enhanced formatting
./manage.sh --action search --query "machine learning" --output-format title-only --limit 5

# Advanced search with all parameters
./manage.sh --action search \
  --query "tech news" \
  --category news \
  --time-range day \
  --pageno 2 \
  --safesearch 1 \
  --output-format compact \
  --limit 10
```

### Output Formats

| Format | Description | Example Use Case |
|--------|-------------|------------------|
| `json` | Full JSON response (default) | API integration, debugging |
| `title-only` | Just titles, one per line | Quick scanning, AI processing |
| `title-url` | Title and URL pairs | Link collection |
| `csv` | CSV format | Spreadsheet import |
| `markdown` | Markdown formatted | Documentation |
| `compact` | Human-readable condensed | Terminal display |

```bash
# Get just titles for AI processing
./manage.sh --action search --query "AI breakthroughs" --output-format title-only

# Export to CSV for analysis
./manage.sh --action search --query "market trends" --output-format csv --save results.csv

# Markdown for documentation
./manage.sh --action search --query "tutorial" --output-format markdown --limit 3
```

### Quick Actions

#### Headlines
Get latest news headlines with optional topic filtering:

```bash
# General headlines
./manage.sh --action headlines

# Topic-specific headlines
./manage.sh --action headlines --topic "AI"
./manage.sh --action headlines --topic "climate"
./manage.sh --action headlines --topic "technology"
```

#### Lucky Search
Get the first result URL directly (like "I'm Feeling Lucky"):

```bash
# Get first result URL
./manage.sh --action lucky --query "Python official documentation"
./manage.sh --action lucky --query "React tutorial"

# Use in automation
DOCS_URL=$(./manage.sh --action lucky --query "TypeScript handbook")
echo "Documentation: $DOCS_URL"
```

### Batch Operations

#### File-Based Batch Search
Process multiple queries from a file:

```bash
# Create query file
echo -e "AI trends\nmachine learning\nquantum computing" > queries.txt

# Process all queries
./manage.sh --action batch-search --file queries.txt

# Results saved as: results_1_AI_trends.json, results_2_machine_learning.json, etc.
```

#### Inline Batch Search
Process comma-separated queries:

```bash
# Multiple queries at once
./manage.sh --action batch-search --queries "AI,ML,data science,automation"

# Results saved as: batch_1_AI.json, batch_2_ML.json, etc.
```

### File Operations

#### Save Results
```bash
# Save to file
./manage.sh --action search --query "research papers" --save research.json

# Append with timestamps (JSONL format)
./manage.sh --action search --query "daily news" --append news-feed.jsonl
```

#### Result Processing
```bash
# Get specific number of results
./manage.sh --action search --query "tutorials" --limit 3

# Pagination support
./manage.sh --action search --query "articles" --pageno 2 --limit 5

# Time-filtered results
./manage.sh --action search --query "breaking news" --time-range hour
```

### Advanced Features

#### Parameter Validation
The script validates all parameters and provides helpful error messages:

```bash
# Invalid page number
./manage.sh --action search --query "test" --pageno 0
# ERROR: Page number must be a positive integer

# Invalid time range
./manage.sh --action search --query "test" --time-range invalid
# ERROR: Invalid time range. Use: hour, day, week, month, year

# Invalid safe search
./manage.sh --action search --query "test" --safesearch 5
# ERROR: Safe search must be 0, 1, or 2
```

#### API Testing & Monitoring
```bash
# Comprehensive API testing
./manage.sh --action api-test

# Performance benchmarking
./manage.sh --action benchmark --count 10

# Health diagnostics
./manage.sh --action diagnose

# Monitor with custom interval
./manage.sh --action monitor --interval 60
```

### Automation Examples

#### Shell Script Integration
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

#### n8n Workflow Integration
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

## 💻 Integration Examples

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
    
    response = requests.get('http://localhost:8100/search', params=params)
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
    
    const response = await axios.get('http://localhost:8100/search', { params });
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
    local results=$(curl -s "http://localhost:8100/search?q=${query// /+}&format=json")
    
    echo "$results" | jq -r '.results[:5] | .[] | "📰 \(.title)\n   🔗 \(.url)\n   📝 \(.content[:100])...\n"'
}

# Example usage
search_web "renewable energy innovations"
```

## 🤖 n8n Integration

SearXNG can be easily integrated with n8n workflows:

1. **HTTP Request Node**:
   - Method: GET
   - URL: `http://localhost:8100/search`
   - Query Parameters:
     - q: `{{ $json.searchQuery }}`
     - format: `json`
   - Authentication: None required for local instance

2. **Example Workflow**:
   ```json
   {
     "name": "Search and Summarize",
     "nodes": [
       {
         "name": "Search Web",
         "type": "n8n-nodes-base.httpRequest",
         "parameters": {
           "url": "http://localhost:8100/search",
           "qs": {
             "q": "={{ $json.topic }}",
             "format": "json",
             "categories": "general,news"
           }
         }
       }
     ]
   }
   ```

## 🛠️ Configuration

### Search Engines
SearXNG aggregates results from multiple engines. Default configuration includes:
- Google
- Bing  
- DuckDuckGo
- Startpage
- Wikipedia

### Settings Location
- Configuration: `~/.searxng/settings.yml`
- Rate Limiter: `~/.searxng/limiter.toml` (optional)

### Key Settings
```yaml
search:
  safe_search: 1              # 0=off, 1=moderate, 2=strict
  autocomplete: ""            # Autocomplete provider
  default_lang: "en"          # Default language
  formats:                    # Enabled output formats
    - html
    - json
    - csv
    - rss

server:
  method: "GET"               # Allow GET requests for API
  limiter: false              # Rate limiting disabled for local use
```

## 🔍 Advanced Usage

### Category-Specific Searches
```bash
# News only
curl "http://localhost:8100/search?q=AI&format=json&categories=news"

# Images
curl "http://localhost:8100/search?q=sunset&format=json&categories=images"

# Scientific papers
curl "http://localhost:8100/search?q=quantum&format=json&categories=science"
```

### Time-Range Filtering
```bash
# Last 24 hours
curl "http://localhost:8100/search?q=breaking+news&format=json&time_range=day"

# Last week
curl "http://localhost:8100/search?q=tech+updates&format=json&time_range=week"

# Last month
curl "http://localhost:8100/search?q=research+papers&format=json&time_range=month"
```

### Language-Specific Results
```bash
# Spanish results
curl "http://localhost:8100/search?q=tecnología&format=json&language=es"

# French results  
curl "http://localhost:8100/search?q=intelligence+artificielle&format=json&language=fr"
```

## 📊 Monitoring & Health

### Check Health Status
```bash
# Via management script
./manage.sh --action status

# Direct health check
curl -s http://localhost:8100/stats | jq .

# Check if API is responding
curl -s "http://localhost:8100/search?q=test&format=json" | jq -r '.query'
```

### View Logs
```bash
# Recent logs
./manage.sh --action logs

# Follow logs
docker logs -f searxng

# Filter for errors
docker logs searxng 2>&1 | grep -i error
```

## 🚨 Troubleshooting

### Common Issues

1. **Permission Denied Errors**
   ```bash
   # Fix permissions on config directory
   docker run --rm -v ~/.searxng:/fix alpine chmod -R 777 /fix
   ```

2. **Container Keeps Restarting**
   - Check logs: `docker logs searxng`
   - Verify config syntax: `cat ~/.searxng/settings.yml`
   - Ensure no port conflicts: `lsof -i :8100`

3. **403 Forbidden on API**
   - Ensure `method: "GET"` in settings.yml
   - Check `limiter: false` for local usage
   - Verify `formats` includes `json`

4. **No Results Returned**
   - Test individual engines are accessible
   - Check network connectivity
   - Verify search engines aren't blocking requests

### Reset Installation
```bash
# Complete cleanup and reinstall
./manage.sh --action uninstall --remove-data yes
./manage.sh --action install
```

## 🔐 Security Considerations

- **Local Access Only**: Default configuration binds to 127.0.0.1
- **No Tracking**: SearXNG doesn't track users or store searches
- **Secret Key**: Automatically generated for session security
- **Rate Limiting**: Disabled for local usage, enable for public access

## 📚 Additional Resources

- [SearXNG Documentation](https://docs.searxng.org/)
- [Supported Engines](https://docs.searxng.org/admin/engines.html)
- [API Reference](https://docs.searxng.org/dev/rest-api.html)
- [Configuration Guide](https://docs.searxng.org/admin/settings.html)

## 🤝 Integration with Vrooli

SearXNG integrates seamlessly with Vrooli's AI tiers:

1. **Tier 3 (Execution)**: Direct search queries for information gathering
2. **Tier 2 (Process)**: Automated research workflows
3. **Tier 1 (Coordination)**: Strategic information discovery

The JSON API enables agents to:
- Research topics autonomously
- Gather real-time information
- Verify facts and data
- Monitor news and updates
- Aggregate diverse perspectives