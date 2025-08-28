# SearXNG Configuration Guide

This guide covers how to configure SearXNG for optimal performance and integration with Vrooli.

## 🛠️ Configuration Overview

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

## 🔐 Security Configuration

### Local Access Only
- **Default Binding**: Configuration binds to 127.0.0.1
- **No Tracking**: SearXNG doesn't track users or store searches
- **Secret Key**: Automatically generated for session security
- **Rate Limiting**: Disabled for local usage, enable for public access

### Recommended Security Settings
```yaml
server:
  bind_address: "127.0.0.1"   # Local access only
  port: 8280                  # Default port
  secret_key: "auto-generated" # Session security
  limiter: false              # Disable for local use

outgoing:
  request_timeout: 3.0        # Timeout for search requests
  max_request_timeout: 10.0   # Maximum allowed timeout
```

## 🔧 Advanced Configuration Options

### Engine-Specific Settings
Enable or disable specific search engines:

```yaml
engines:
  - name: google
    engine: google
    disabled: false
    
  - name: bing
    engine: bing
    disabled: false
    
  - name: duckduckgo
    engine: duckduckgo
    disabled: false

  - name: startpage
    engine: startpage
    disabled: false
```

### Custom Categories
Define custom search categories:

```yaml
categories_as_tabs:
  general:
    - google
    - bing
    - duckduckgo
  
  news:
    - google news
    - bing news
    
  science:
    - google scholar
    - semantic scholar
```

### Language Configuration
Set default language preferences:

```yaml
search:
  default_lang: "en"
  languages:
    - "en"
    - "es" 
    - "fr"
    - "de"
    - "auto"
```

## 🚀 Performance Tuning

### Request Optimization
```yaml
outgoing:
  request_timeout: 3.0        # Balance speed vs reliability
  max_request_timeout: 10.0   # Maximum timeout for slow engines
  useragent_suffix: ""        # Custom user agent
  pool_connections: 100       # Connection pool size
  pool_maxsize: 20           # Max connections per pool
```

### Result Processing
```yaml
search:
  max_page: 3                # Maximum page number for searches
  formats:                   # Enable required formats only
    - html
    - json
    - csv
```

## 📝 Template Files

### settings.yml.template
Location: `config/settings.yml.template`

This template provides a working configuration for local development and AI integration.

### limiter.toml.template  
Location: `docker/limiter.toml.template`

Rate limiting configuration (typically not needed for local usage).

## 🔄 Configuration Management

### View Current Configuration
```bash
# Check current settings
resource-searxng content execute --name config

# View specific setting
cat ~/.searxng/settings.yml | grep -A 5 "search:"
```

### Update Configuration
```bash
# Edit configuration
nano ~/.searxng/settings.yml

# Restart to apply changes
resource-searxng manage restart
```

### Reset to Defaults
```bash
# Backup current config
cp ~/.searxng/settings.yml ~/.searxng/settings.yml.backup

# Reset to template
resource-searxng content execute --name reset
```

## 🔗 Integration Settings

### For Vrooli AI Agents
Recommended settings for AI integration:

```yaml
search:
  safe_search: 1              # Moderate filtering
  default_lang: "en"          # English results
  formats:                    # Enable JSON for APIs
    - json
    - html

server:
  method: "GET"               # Enable GET requests
  limiter: false              # No rate limiting for local agents
  bind_address: "127.0.0.1"   # Local access only

outgoing:
  request_timeout: 5.0        # Reasonable timeout for agents
  max_request_timeout: 15.0   # Allow for complex queries
```

### For n8n Workflows
```yaml
# Same as above, plus:
server:
  cors_origins: 
    - "http://localhost:5678"  # Allow n8n access
```