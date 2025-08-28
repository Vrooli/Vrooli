# SearXNG API Reference

SearXNG provides a powerful JSON API for programmatic access, perfect for integration with Vrooli's AI agents.

## 🔌 Basic API Usage

### Simple Search
```bash
# Basic search
curl "http://localhost:8280/search?q=artificial+intelligence&format=json"

# Search with specific parameters
curl "http://localhost:8280/search?q=tech+news&format=json&categories=general&language=en"
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

## 🎯 Enhanced CLI Commands

The SearXNG CLI provides powerful search capabilities beyond basic API calls, designed for AI agents and automation workflows.

### Advanced Search Parameters

```bash
# Basic search with enhanced formatting
resource-searxng content execute --name search --query "machine learning" --output title-only --limit 5

# Advanced search with all parameters
resource-searxng content execute --name search \
  --query "tech news" \
  --category news \
  --time-range day \
  --page 2 \
  --safesearch 1 \
  --output compact \
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
resource-searxng content execute --name search --query "AI breakthroughs" --output title-only

# Export to CSV for analysis
resource-searxng content execute --name search --query "market trends" --output csv --save results.csv

# Markdown for documentation
resource-searxng content execute --name search --query "tutorial" --output markdown --limit 3
```

### Quick Actions

#### Headlines
Get latest news headlines with optional topic filtering:

```bash
# General headlines
resource-searxng content execute --name headlines

# Topic-specific headlines
resource-searxng content execute --name headlines --topic "AI"
resource-searxng content execute --name headlines --topic "climate"
resource-searxng content execute --name headlines --topic "technology"
```

#### Lucky Search
Get the first result URL directly (like "I'm Feeling Lucky"):

```bash
# Get first result URL
resource-searxng content execute --name lucky --query "Python official documentation"
resource-searxng content execute --name lucky --query "React tutorial"

# Use in automation
DOCS_URL=$(resource-searxng content execute --name lucky --query "TypeScript handbook")
echo "Documentation: $DOCS_URL"
```

### Batch Operations

#### File-Based Batch Search
Process multiple queries from a file:

```bash
# Create query file
echo -e "AI trends\nmachine learning\nquantum computing" > queries.txt

# Process all queries
resource-searxng content execute --name batch-search --file queries.txt

# Results saved as: results_1_AI_trends.json, results_2_machine_learning.json, etc.
```

#### Inline Batch Search
Process comma-separated queries:

```bash
# Multiple queries at once
resource-searxng content execute --name batch-search --queries "AI,ML,data science,automation"

# Results saved as: batch_1_AI.json, batch_2_ML.json, etc.
```

### File Operations

#### Save Results
```bash
# Save to file
resource-searxng content execute --name search --query "research papers" --save research.json

# Append with timestamps (JSONL format)
resource-searxng content execute --name search --query "daily news" --append news-feed.jsonl
```

#### Result Processing
```bash
# Get specific number of results
resource-searxng content execute --name search --query "tutorials" --limit 3

# Pagination support
resource-searxng content execute --name search --query "articles" --page 2 --limit 5

# Time-filtered results
resource-searxng content execute --name search --query "breaking news" --time-range hour
```

### Advanced Features

#### Parameter Validation
The script validates all parameters and provides helpful error messages:

```bash
# Invalid page number
resource-searxng content execute --name search --query "test" --page 0
# ERROR: Page number must be a positive integer

# Invalid time range
resource-searxng content execute --name search --query "test" --time-range invalid
# ERROR: Invalid time range. Use: hour, day, week, month, year

# Invalid safe search
resource-searxng content execute --name search --query "test" --safesearch 5
# ERROR: Safe search must be 0, 1, or 2
```

#### API Testing & Monitoring
```bash
# Comprehensive API testing
resource-searxng content execute --name api-test

# Performance benchmarking
resource-searxng content execute --name benchmark --count 10

# Health diagnostics
resource-searxng content execute --name diagnose

# Monitor with custom interval
resource-searxng content execute --name monitor --interval 60
```

## 🔍 Advanced Usage

### Category-Specific Searches
```bash
# News only
curl "http://localhost:8280/search?q=AI&format=json&categories=news"

# Images
curl "http://localhost:8280/search?q=sunset&format=json&categories=images"

# Scientific papers
curl "http://localhost:8280/search?q=quantum&format=json&categories=science"
```

### Time-Range Filtering
```bash
# Last 24 hours
curl "http://localhost:8280/search?q=breaking+news&format=json&time_range=day"

# Last week
curl "http://localhost:8280/search?q=tech+updates&format=json&time_range=week"

# Last month
curl "http://localhost:8280/search?q=research+papers&format=json&time_range=month"
```

### Language-Specific Results
```bash
# Spanish results
curl "http://localhost:8280/search?q=tecnología&format=json&language=es"

# French results  
curl "http://localhost:8280/search?q=intelligence+artificielle&format=json&language=fr"
```

## 📊 Monitoring & Health

### Check Health Status
```bash
# Via management script
resource-searxng content execute --name status

# Direct health check
curl -s http://localhost:8280/stats | jq .

# Check if API is responding
curl -s "http://localhost:8280/search?q=test&format=json" | jq -r '.query'
```

### View Logs
```bash
# Recent logs
resource-searxng content execute --name logs

# Follow logs
docker logs -f searxng

# Filter for errors
docker logs searxng 2>&1 | grep -i error
```