# Contributing Guide

## Development Setup

### Prerequisites

- Vrooli CLI installed: `vrooli help`
- Qdrant resource enabled in `.vrooli/service.json`
- Ollama with `mxbai-embed-large` model
- Working in a Vrooli app directory

### First-Time Setup

```bash
# Navigate to your app
cd /path/to/your/app

# Initialize embeddings
resource-qdrant embeddings init

# Index existing content
resource-qdrant embeddings refresh

# Verify setup
resource-qdrant embeddings status
```

## Development Workflow

### Daily Development

```bash
# Morning: Check existing knowledge
resource-qdrant search "feature requirements"
resource-qdrant search-all "similar implementations"

# Before implementing: Find solutions
resource-qdrant search-all "email validation" code
resource-qdrant search-all "authentication" workflows

# After implementing: Update knowledge
resource-qdrant embeddings refresh
```

### Git Integration

The system automatically detects changes:
- Checks git commit hashes
- Runs background refresh if needed
- No manual intervention required

### Knowledge-Driven Development

```bash
# Discovery phase
resource-qdrant search-all "payment processing" workflows

# Pattern identification
resource-qdrant search "webhook validation" code

# Gap analysis
resource-qdrant search "error handling" docs
```

## Documentation Standards

### Embedding Markers

Use markers to structure knowledge extraction:

```markdown
<!-- EMBED:DECISION:START -->
### Database Choice: PostgreSQL vs MongoDB
**Context:** Need persistence for user data
**Decision:** PostgreSQL for ACID compliance
**Rationale:** Financial data requires consistency
**Trade-offs:** Complex schema vs data integrity
<!-- EMBED:DECISION:END -->

<!-- EMBED:PATTERN:START -->
### Email Queue Pattern
**Problem:** Reliable email delivery
**Solution:** Redis-backed queue with backoff
**Implementation:** See lib/email-queue.sh
**Used in:** user-portal, notifications
<!-- EMBED:PATTERN:END -->

<!-- EMBED:LESSON:START -->
### Authentication Timing Attack
**Context:** Initial auth had timing issues
**Problem:** Different response times
**Solution:** Constant-time comparison
**Prevention:** Always use constant-time ops
<!-- EMBED:LESSON:END -->
```

### Code Documentation

**Shell Functions:**
```bash
# Send notification email
# Arguments:
#   $1 - Recipient email
#   $2 - Subject line
#   $3 - Message body
# Returns:
#   0 - Success
#   1 - Invalid format
#   2 - SMTP failed
email::send_notification() {
    local recipient="$1"
    # Implementation
}
```

**TypeScript/JavaScript:**
```typescript
/**
 * Process batch email categorization
 * @route POST /api/emails/batch/categorize
 * @param emails Array to categorize
 * @returns Categorized with confidence scores
 */
export async function batchCategorizeEmails() {
    // Implementation
}
```

### PRD Documentation

Keep PRDs current and searchable:

```markdown
# Product Requirements: Email Assistant

## Executive Summary
AI-powered email management system...

## Value Proposition
- 70% reduction in processing time
- Automated priority detection

## Technical Requirements
- React + TypeScript frontend
- Node.js + PostgreSQL backend
```

## Testing Guidelines

### Unit Tests

Test individual extraction functions:

```bash
# Run unit tests
bats test/extractors/test-code.bats
bats test/extractors/test-docs.bats
```

### Integration Tests

Test end-to-end flows:

```bash
# Test complete pipeline
resource-qdrant test integration

# Test specific flows
./test/integration-test.sh
```

### Performance Testing

Monitor extraction and search performance:

```bash
# Time extraction
time resource-qdrant embeddings refresh

# Measure search latency
time resource-qdrant search-all "complex query"

# Check metrics
cat ~/.vrooli/embeddings/performance.log
```

## Adding New Features

### Custom Extractors

Create extractors for new content types:

```bash
# Create extractor
cat > extractors/custom/logs.sh << 'EOF'
#!/usr/bin/env bash

custom::extract::logs() {
    local directory="$1"
    
    # Find log files
    find "$directory" -name "*.log"
    
    # Extract patterns
    grep -E "ERROR|WARN" *.log | \
    sort | uniq -c
}
EOF

# Register in CLI
# Add to extraction pipeline
```

### New Parsers

Add language-specific parsers:

```bash
# Create parser
cat > parsers/code/rust.sh << 'EOF'
#!/usr/bin/env bash

parse::rust::functions() {
    local file="$1"
    
    # Extract Rust functions
    grep -E "^(pub )?fn " "$file"
}
EOF
```

## Best Practices

### Documentation

1. **Use embedding markers** for important content
2. **Document patterns** as you discover them
3. **Keep PRDs current** and detailed
4. **Add function docs** with clear examples

### Search Strategy

1. **Start broad**, then narrow down
2. **Use type filters** for performance
3. **Search across apps** for patterns
4. **Document successful queries**

### Code Quality

1. **Follow shell best practices**
2. **Use consistent naming** conventions
3. **Add error handling** to all functions
4. **Include performance logging**

### Performance

1. **Batch operations** when possible
2. **Use parallel processing** for large datasets
3. **Cache frequently accessed** data
4. **Monitor resource usage**

## IDE Integration

### VS Code Task

```json
{
    "version": "2.0.0",
    "tasks": [{
        "label": "Search Knowledge",
        "type": "shell",
        "command": "resource-qdrant",
        "args": ["search-all", "${input:query}"]
    }],
    "inputs": [{
        "id": "query",
        "type": "promptString",
        "description": "Search query"
    }]
}
```

### Git Hooks

Pre-commit hook for similar code detection:

```bash
#!/usr/bin/env bash
# .git/hooks/pre-commit

new_functions=$(git diff --cached --name-only | \
    xargs grep -l "function\|def\|async")

if [[ -n "$new_functions" ]]; then
    echo "🔍 Checking for similar code..."
    resource-qdrant search-all "function_name" code
fi
```

## Troubleshooting Development

### Common Issues

**Embeddings not refreshing:**
```bash
resource-qdrant embeddings refresh --force
```

**No search results:**
```bash
# Check embeddings exist
resource-qdrant embeddings status

# Validate extraction
resource-qdrant embeddings validate

# Check Qdrant
curl http://localhost:6333/collections
```

**Slow refresh:**
```bash
# Verbose mode
resource-qdrant embeddings refresh --verbose

# Selective refresh
resource-qdrant embeddings refresh --type workflows

# Monitor resources
top -p $(pgrep qdrant)
```

## Submitting Changes

### Pull Request Process

1. **Test locally** with full pipeline
2. **Update documentation** with changes
3. **Add tests** for new features
4. **Run validation** before submitting

### Code Review Checklist

- [ ] Extraction logic tested
- [ ] Performance acceptable
- [ ] Documentation updated
- [ ] Error handling complete
- [ ] Tests passing

## Getting Help

- Check existing patterns: `resource-qdrant search-all "problem"`
- Review documentation: `/docs/` directory
- Ask in discussions: GitHub Discussions
- Report issues: GitHub Issues

---

*Remember: Every contribution makes the system smarter. Document your patterns, share your learnings, and help build collective intelligence.*