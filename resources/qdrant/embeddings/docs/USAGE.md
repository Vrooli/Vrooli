# Usage Guide

## Overview

The Qdrant Embeddings system provides semantic knowledge extraction and search capabilities for Vrooli apps, enabling intelligent discovery of patterns, solutions, and knowledge across your entire ecosystem.

## Quick Start

```bash
# Navigate to your project
cd /path/to/your/app

# Initialize embeddings
resource-qdrant embeddings init

# Index your content
resource-qdrant embeddings refresh

# Search for knowledge
resource-qdrant embeddings search "email notification"
```

## Core Commands

### Initialization & Management

```bash
# Initialize app identity
resource-qdrant embeddings init [app-id]

# Refresh embeddings (auto-detects changes)
resource-qdrant embeddings refresh [app-id] [--force]

# Validate setup
resource-qdrant embeddings validate [directory]

# Show system status
resource-qdrant embeddings status

# Garbage collect orphaned embeddings
resource-qdrant embeddings gc [--force]
```

### Search Operations

```bash
# Search within current app
resource-qdrant embeddings search "query" [type]

# Search across all apps
resource-qdrant embeddings search-all "query" [type]

# Discover patterns
resource-qdrant embeddings patterns "authentication"

# Find solutions
resource-qdrant embeddings solutions "image processing"

# Analyze gaps
resource-qdrant embeddings gaps "security"

# Interactive explorer
resource-qdrant embeddings explore
```

## Content Types

Filter searches by content type:

- `all` - Search everything (default)
- `workflows` - N8n workflows and automations
- `scenarios` - PRDs and configurations
- `knowledge` - Documentation and guides
- `code` - Functions, APIs, and implementations
- `resources` - Resource capabilities

## Common Workflows

### New Project Setup

```bash
cd ~/projects/my-app
resource-qdrant embeddings init
resource-qdrant embeddings refresh
```

### Finding Existing Solutions

```bash
# Before implementing new feature
resource-qdrant embeddings solutions "payment processing"

# Check for patterns
resource-qdrant embeddings patterns "webhook handling"

# Learn from mistakes
resource-qdrant embeddings search-all "lessons learned authentication"
```

### Knowledge Discovery

```bash
# Explore what's available
resource-qdrant embeddings explore

# Find gaps in documentation
resource-qdrant embeddings gaps "deployment"

# Cross-app pattern analysis
resource-qdrant embeddings search-all "error handling" knowledge
```

## Advanced Features

### Custom App IDs

```bash
# Initialize with specific ID
resource-qdrant embeddings init "my-app-v2"

# Refresh specific app
resource-qdrant embeddings refresh "my-app-v2"
```

### Force Options

```bash
# Force refresh (ignore cache)
resource-qdrant embeddings refresh --force

# Force garbage collection
resource-qdrant embeddings gc --force yes
```

### Interactive Explorer

The explorer provides an interactive interface for discovery:

```bash
$ resource-qdrant embeddings explore

=== Qdrant Search Explorer ===
Commands: search, patterns, solutions, gaps, compare, quit

search> patterns authentication
# Shows authentication patterns across all apps

search> solutions "email system"
# Finds existing email implementations

search> quit
```

## Integration with AI Agents

Agents use these commands to:

1. **Discover existing solutions** before building new ones
2. **Learn from patterns** documented across projects
3. **Find reusable components** in the ecosystem
4. **Identify knowledge gaps** to address

Example agent workflow:

```bash
# Agent needs email functionality
resource-qdrant embeddings solutions "send email with attachments"
# Finds 3 implementations, reuses best match
# System gets smarter without redundant work
```

## Performance Tips

1. **Initialize early** - Set up embeddings when creating projects
2. **Refresh regularly** - System auto-detects when refresh is needed
3. **Use type filters** - Faster searches when you know the content type
4. **Leverage patterns** - Discover recurring solutions across apps
5. **Check gaps** - Identify missing documentation systematically

## Troubleshooting

### Common Issues

```bash
# No app identity found
resource-qdrant embeddings init

# No embeddings found
resource-qdrant embeddings refresh

# Cannot connect to Qdrant
docker ps | grep qdrant
docker start qdrant

# Model not available
ollama pull mxbai-embed-large
```

## See Also

- [Architecture](ARCHITECTURE.md) - System design and pipeline details
- [Configuration](CONFIGURATION.md) - Model selection and tuning
- [Troubleshooting](TROUBLESHOOTING.md) - Detailed problem resolution
- [API Reference](API_REFERENCE.md) - Complete command documentation