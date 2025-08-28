# SearXNG - Privacy-Respecting Metasearch Engine

SearXNG is a privacy-respecting metasearch engine that aggregates results from multiple search engines without tracking users. This resource provides a local instance perfect for AI agents and automation workflows.

## Quick Reference
- **Category**: Search
- **Port**: 8280 (defined in port_registry.sh)
- **Container**: searxng
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready

## When to Use
- **Privacy-sensitive searches** without tracking or data collection
- **Local search API** for AI agents and automation workflows
- **Aggregated results** from multiple search engines (Google, Bing, DuckDuckGo, Startpage)
- **Research and information gathering** workflows requiring diverse sources

**Alternative**: Direct search engine APIs for specific providers, cloud search services for scale

## 🚀 Quick Start

```bash
# Install SearXNG
resource-searxng manage install

# Check status
resource-searxng status

# View logs
resource-searxng logs

# Enhanced search capabilities
resource-searxng content execute --name search --query "AI" --limit 5 --output-format title-only
resource-searxng content execute --name headlines --topic "tech"
resource-searxng content execute --name lucky --query "Python documentation"
```

## Basic API Usage

```bash
# Simple search
curl "http://localhost:8280/search?q=artificial+intelligence&format=json"

# Search with specific parameters
curl "http://localhost:8280/search?q=tech+news&format=json&categories=general&language=en"
```

## 📚 Documentation

- 📖 [**Complete API Reference**](docs/API.md) - Full API documentation, enhanced management script, batch operations
- ⚙️ [**Configuration Guide**](docs/CONFIGURATION.md) - Settings, security, performance tuning
- 🔧 [**Troubleshooting**](docs/TROUBLESHOOTING.md) - Common issues, diagnostics, and solutions
- 🏗️ [**Advanced Integration**](docs/ADVANCED.md) - Programming examples, automation, multi-resource workflows

## Service Management

```bash
# Install and setup
resource-searxng manage install

# Start/stop/restart
resource-searxng manage start
resource-searxng manage stop
resource-searxng manage restart

# Status and health
resource-searxng status
resource-searxng logs

# Advanced operations
resource-searxng test integration
resource-searxng content execute --name benchmark
resource-searxng content execute --name diagnose
```

## Access Points

After installation:
- **Web Interface**: http://localhost:8280 (search interface)
- **API Endpoint**: http://localhost:8280/search (JSON API)
- **Stats/Health**: http://localhost:8280/stats (service statistics)

## Integration Examples

### With AI (Ollama)
```bash
# Search and analyze pipeline
RESULTS=$(resource-searxng content execute --name search --query "AI trends" --output-format json)
echo "$RESULTS" | curl -X POST http://localhost:11434/api/generate -d @-
```

### With n8n Workflows
```bash
# HTTP Request Node URL: http://localhost:8280/search
# Parameters: q={{ $json.query }}, format=json
```

### With Vrooli AI Tiers
- **Tier 1**: Strategic information discovery and research planning
- **Tier 2**: Automated research workflows and information filtering  
- **Tier 3**: Direct search execution and result processing

## Default Search Engines
- Google
- Bing
- DuckDuckGo
- Startpage
- Wikipedia

## Security Features
- **Local Access Only**: Binds to 127.0.0.1 by default
- **No Tracking**: Doesn't store searches or track users
- **Privacy-First**: Aggregates results without exposing user data
- **Rate Limiting**: Configurable for production use

For detailed usage instructions, integration patterns, and troubleshooting, see the documentation links above.