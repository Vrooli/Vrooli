# Qdrant Embeddings CLI Usage Guide

The Qdrant Semantic Knowledge System is now fully integrated into the `resource-qdrant` CLI!

## 🚀 Quick Start

```bash
# Navigate to your project
cd /path/to/your/app

# Initialize embeddings for your project
resource-qdrant embeddings init

# Refresh/index all content
resource-qdrant embeddings refresh

# Search your knowledge
resource-qdrant embeddings search "how to send emails"

# Search across ALL apps
resource-qdrant embeddings search-all "webhook processing"
```

## 📋 All Available Commands

### Management Commands

```bash
# Initialize app identity
resource-qdrant embeddings init [app-id]

# Refresh embeddings (auto-detects changes)
resource-qdrant embeddings refresh [app-id] [--force]

# Validate setup and coverage
resource-qdrant embeddings validate [directory]

# Show status of all embedding systems
resource-qdrant embeddings status

# Garbage collect orphaned embeddings
resource-qdrant embeddings gc [--force]
```

### Search Commands

```bash
# Search within current app
resource-qdrant embeddings search "query" [type]

# Search across all apps
resource-qdrant embeddings search-all "query" [type]

# Discover patterns
resource-qdrant embeddings patterns "authentication"

# Find reusable solutions
resource-qdrant embeddings solutions "image processing problem"

# Analyze knowledge gaps
resource-qdrant embeddings gaps "security topic"

# Interactive explorer
resource-qdrant embeddings explore
```

## 🎯 Real-World Examples

### Example 1: Initialize and Index a New Project

```bash
$ cd ~/projects/my-new-app

$ resource-qdrant embeddings init
[INFO] Initializing embeddings for project...
[SUCCESS] Initialized embeddings for app: my-new-app

$ resource-qdrant embeddings refresh
[INFO] Refreshing embeddings for app: my-new-app
[INFO] Processing workflows...
[INFO] Processing scenarios...
[INFO] Processing documentation...
[INFO] Processing code...
[INFO] Processing resources...
[SUCCESS] Embedding refresh complete!
[INFO] Total Embeddings: 523
[INFO] Duration: 45s
```

### Example 2: Search for Existing Solutions

```bash
$ resource-qdrant embeddings search "email notification"

=== Search Results ===
Query: "email notification"
App: my-new-app
Type: all

Found 3 result(s):

Score: 0.92
Type: workflow
Content: Workflow: Email Notification System
Description: Sends automated emails based on events
File: initialization/n8n/email-notifications.json
---

Score: 0.87
Type: code
Content: Function: sendEmailNotification
File: lib/notifications/email.ts
---
```

### Example 3: Cross-App Pattern Discovery

```bash
$ resource-qdrant embeddings patterns "webhook"

=== Pattern Discovery ===
Query: "webhook"

By Type:
  workflows: 8 occurrences
  code: 5 occurrences
  knowledge: 2 occurrences

By App:
  vrooli-main: 5 matches
  api-gateway: 4 matches
  event-processor: 3 matches
  notification-service: 3 matches

Top Patterns (score > 0.7):
[0.94] workflows in api-gateway:
  Webhook receiver for external events

[0.91] code in vrooli-main:
  Webhook validation middleware

[0.89] workflows in event-processor:
  Webhook-to-queue processor
```

### Example 4: Find Knowledge Gaps

```bash
$ resource-qdrant embeddings gaps "rate limiting"

=== Knowledge Gap Analysis: rate limiting ===

Current Coverage:
  • code: 2 item(s)
  • knowledge: 1 item(s)

Potential Gaps:
  ❌ No workflows found
  ❌ No scenarios found

App Coverage: 2/5 apps have relevant content
  Consider adding rate limiting content to more apps
```

### Example 5: Interactive Explorer

```bash
$ resource-qdrant embeddings explore

=== Qdrant Search Explorer ===
Commands: search, patterns, solutions, gaps, compare, quit

search> search
Query: database migration
Type (all/workflows/scenarios/knowledge/code/resources): code
Found 4 results across 3 apps

[0.93] code in vrooli-main:
  Database migration utility functions

[0.89] code in user-service:
  Automated migration runner

search> patterns
Query: authentication
Pattern Discovery:
By Type:
  code: 12 occurrences
  workflows: 4 occurrences
  knowledge: 3 occurrences

search> quit
Goodbye!
```

## 🔧 Options and Filters

### Type Filters
- `all` - Search all content types (default)
- `workflows` - Only n8n workflows
- `scenarios` - Only PRDs and configs
- `knowledge` - Only documentation
- `code` - Only functions and APIs
- `resources` - Only resource capabilities

### Force Options
- `--force yes` - Skip confirmations for refresh/gc
- `--force` - Same as above

### Search Limits
- Default: 10 results per search
- Use manage.sh directly for custom limits

## 🐛 Troubleshooting

### "No app identity found"
```bash
# Initialize first
resource-qdrant embeddings init
```

### "No embeddings found"
```bash
# Refresh to create embeddings
resource-qdrant embeddings refresh
```

### "Cannot connect to Qdrant"
```bash
# Ensure Qdrant is running
docker ps | grep qdrant
# If not, start it
docker start qdrant
```

### "Model not available"
```bash
# Install the embedding model
ollama pull mxbai-embed-large
```

## 🎨 Tips for Best Results

1. **Initialize Early**: Run `init` as soon as you create a new project
2. **Refresh Regularly**: Embeddings auto-detect when refresh is needed
3. **Use Type Filters**: Faster searches when you know what you're looking for
4. **Explore Patterns**: Use `patterns` command to discover recurring solutions
5. **Check Gaps**: Regularly run `gaps` to find missing documentation

## 🔮 Advanced Usage

### Custom App IDs
```bash
# Initialize with specific app ID
resource-qdrant embeddings init "my-app-v2"

# Refresh specific app
resource-qdrant embeddings refresh "my-app-v2"
```

### Validation Reports
```bash
# Validate current directory
resource-qdrant embeddings validate

# Validate specific directory
resource-qdrant embeddings validate /path/to/project
```

### Status Monitoring
```bash
# Check all apps and collections
resource-qdrant embeddings status

# Shows:
# - Current app identity
# - All registered apps
# - Collection sizes
# - Model availability
```

## 🎯 Integration with Agents

AI agents can use these commands to:
1. **Discover existing solutions** before building new ones
2. **Learn from documented patterns** and failures
3. **Find reusable components** across the ecosystem
4. **Identify knowledge gaps** to fill

Example agent workflow:
```bash
# Agent needs to implement email sending
$ resource-qdrant embeddings solutions "send email with attachments"

# Agent finds 3 existing implementations
# Agent reuses the best match instead of creating new code
# System gets smarter without redundant work!
```

## 📚 More Information

- Full documentation: `/scripts/resources/storage/qdrant/embeddings/README.md`
- Implementation details: `/scripts/resources/storage/qdrant/embeddings/IMPLEMENTATION_PLAN.md`
- Progress status: `/scripts/resources/storage/qdrant/embeddings/PROGRESS_SUMMARY.md`

---

*The Semantic Knowledge System is now live - every search makes the system smarter!* 🚀