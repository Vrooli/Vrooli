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