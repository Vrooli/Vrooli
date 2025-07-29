# SearXNG API Reference

SearXNG provides a powerful JSON API for programmatic access, perfect for integration with Vrooli's AI agents.

## 🔌 Basic API Usage

### Simple Search
```bash
# Basic search
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