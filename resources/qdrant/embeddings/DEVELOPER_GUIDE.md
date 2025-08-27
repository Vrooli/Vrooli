# Developer Integration Guide

This guide shows how to integrate the Qdrant Semantic Knowledge System into your development workflow and leverage it for building smarter agents and applications.

## 🚀 Getting Started

### Prerequisites
- Vrooli CLI installed and working: `vrooli help`
- Qdrant resource enabled: Check `.vrooli/service.json`
- Working in a Vrooli app directory

### First-Time Setup

```bash
# 1. Navigate to your app directory
cd /path/to/your/vrooli/app

# 2. Initialize semantic knowledge system
resource-qdrant embeddings init

# 3. Index your existing content
resource-qdrant embeddings refresh

# 4. Verify setup worked
resource-qdrant embeddings status
```

## 🔧 CLI Integration

### Using with Vrooli CLI

The system integrates seamlessly with the Vrooli CLI:

```bash
# Initialize embeddings for current app
resource-qdrant embeddings init

# Refresh embeddings (detects changes automatically)
resource-qdrant embeddings refresh

# Search your app
resource-qdrant search "send email notifications"

# Search across all apps  
resource-qdrant search-all "authentication patterns"

# Validate your knowledge structure
resource-qdrant embeddings validate

# View status of all apps
resource-qdrant embeddings status
```

### Direct Shell Integration

For advanced usage, source the management script directly:

```bash
# Load functions into current shell
source resources/qdrant/embeddings/manage.sh

# Now use functions directly
qdrant::embeddings::init "my-app-v2"
qdrant::search::all_apps "database migrations" knowledge
qdrant::search::discover_patterns "error handling"
```

## 🏗️ Development Workflow Integration

### 1. Daily Development Workflow

```bash
# Morning routine - check what knowledge exists
resource-qdrant search "today's feature requirements"
resource-qdrant search-all "similar implementations"

# Before implementing - find existing solutions
resource-qdrant search-all "email validation" code
resource-qdrant search-all "user authentication" workflows

# After implementing - refresh knowledge  
resource-qdrant embeddings refresh
```

### 2. Git Integration (Automatic)

The system automatically detects git changes and refreshes embeddings:

```bash
# When you run vrooli develop, it automatically:
# 1. Checks for git commit changes
# 2. Runs background embedding refresh if needed
# 3. Continues with development without delays
```

### 3. Knowledge-Driven Development

```bash
# Start with discovery
resource-qdrant search-all "payment processing" workflows

# Find patterns
resource-qdrant search "webhook validation" code

# Check for gaps
resource-qdrant search "error handling" docs

# Document as you go (will be auto-indexed)
echo "<!-- EMBED:PATTERN:START -->" >> docs/PATTERNS.md
echo "### Payment Validation Pattern" >> docs/PATTERNS.md
echo "**Context:** User payment validation" >> docs/PATTERNS.md
echo "<!-- EMBED:PATTERN:END -->" >> docs/PATTERNS.md
```

## 📚 Knowledge Management Best Practices

### 1. Structuring Knowledge

**Use embedding markers in documentation:**

```markdown
<!-- EMBED:DECISION:START -->
### 2025-01-22 Database Choice: PostgreSQL vs MongoDB
**Context:** Need persistence for user data and transactions
**Decision:** Use PostgreSQL for ACID compliance
**Rationale:** Financial data requires strong consistency
**Trade-offs:** Slightly more complex schema vs guaranteed data integrity
<!-- EMBED:DECISION:END -->
```

**Pattern documentation:**

```markdown
<!-- EMBED:PATTERN:START -->
### Email Queue Pattern
**Problem:** Reliable email delivery with retries
**Solution:** Redis-backed queue with exponential backoff
**Implementation:** See `lib/email-queue.sh`
**Used in:** user-portal-v1, notification-service
<!-- EMBED:PATTERN:END -->
```

### 2. Code Documentation

**Function documentation:**

```bash
# Send notification email to user
# Arguments:
#   $1 - Recipient email address
#   $2 - Subject line
#   $3 - Message body (HTML or plain text)
# Returns:
#   0 - Success
#   1 - Invalid email format
#   2 - SMTP connection failed
email::send_notification() {
    local recipient="$1"
    local subject="$2"
    local body="$3"
    
    # Implementation...
}
```

**API endpoint documentation:**

```typescript
/**
 * Process batch email categorization
 * @route POST /api/emails/batch/categorize
 * @param emails Array of email objects to categorize
 * @param model AI model to use for categorization
 * @returns Categorized emails with confidence scores
 */
export async function batchCategorizeEmails(req: Request, res: Response) {
    // Implementation...
}
```

### 3. Scenario Documentation

**Keep PRDs current:**

```markdown
# Product Requirements Document: Smart Email Assistant

## Executive Summary
A lightweight email management assistant that uses AI-powered categorization...

## Value Proposition
- 70% reduction in email processing time
- Automated priority detection
- Smart folder organization

## Technical Requirements
- React + TypeScript frontend
- Node.js backend with PostgreSQL
- Integration with Gmail API
```

## 🔍 Advanced Search Patterns

### 1. Multi-Step Problem Solving

```bash
# Step 1: Find similar problems
resource-qdrant search-all "user onboarding flow" scenarios

# Step 2: Find implementation patterns  
resource-qdrant search-all "email verification" workflows

# Step 3: Find code examples
resource-qdrant search-all "JWT token validation" code

# Step 4: Check for lessons learned
resource-qdrant search-all "onboarding mistakes" knowledge
```

### 2. Cross-App Pattern Discovery

```bash
# Find authentication patterns across all apps
resource-qdrant search-all "OAuth integration" workflows

# Discover error handling approaches
resource-qdrant search-all "retry logic" code

# Find monitoring patterns
resource-qdrant search-all "health checks" resources
```

### 3. Gap Analysis and Planning

```bash
# Before starting new feature
resource-qdrant search-all "notification system" scenarios
resource-qdrant search-all "push notifications" workflows
resource-qdrant search-all "FCM integration" code

# If gaps found, check other apps
resource-qdrant search-all "mobile notifications" 
```

## 🤖 AI Agent Integration

### 1. Agent Discovery Workflow

```bash
# Agent starting new task
function agent_discover_solutions() {
    local task_description="$1"
    
    echo "🔍 Searching for existing solutions..."
    
    # Search workflows first
    local workflows=$(resource-qdrant search-all "$task_description" workflows)
    
    # Then search code patterns
    local code_patterns=$(resource-qdrant search-all "$task_description" code)
    
    # Check documentation
    local knowledge=$(resource-qdrant search-all "$task_description" knowledge)
    
    # Present findings
    echo "Found solutions:"
    echo "$workflows"
    echo "$code_patterns"
    echo "$knowledge"
}
```

### 2. Incremental Learning

```bash
# After agent completes task, document learnings
function agent_document_learnings() {
    local task="$1"
    local solution="$2"
    local outcome="$3"
    
    # Document in lessons learned
    cat >> docs/LESSONS_LEARNED.md << EOF
<!-- EMBED:SUCCESS:START -->
### $(date) $task Success
**Context:** $task
**Implementation:** $solution  
**Result:** $outcome
**Reusable:** Pattern can be applied to similar tasks
<!-- EMBED:SUCCESS:END -->
EOF
    
    # Refresh embeddings to capture new knowledge
    resource-qdrant embeddings refresh
}
```

### 3. Pattern Recognition

```bash
# Agent analyzing patterns before implementation
function agent_analyze_patterns() {
    local domain="$1"
    
    # Discover existing patterns
    local patterns=$(resource-qdrant search-all "$domain" | grep -E "(Pattern|pattern)")
    
    # Find related implementations
    local implementations=$(resource-qdrant search-all "$domain" code)
    
    # Check for anti-patterns or failures
    local lessons=$(resource-qdrant search-all "$domain failure" knowledge)
    
    echo "Pattern Analysis for $domain:"
    echo "Existing patterns: $patterns"
    echo "Implementations: $implementations"
    echo "Lessons learned: $lessons"
}
```

## 🛠️ Custom Extractors

### Creating Custom Content Extractors

For specialized content types, create custom extractors:

```bash
# Create custom extractor for your domain
cat > extractors/custom-logs.sh << 'EOF'
#!/usr/bin/env bash

# Extract insights from application logs
custom::extract::log_insights() {
    local log_file="$1"
    
    # Extract error patterns
    grep -E "(ERROR|WARN|FATAL)" "$log_file" | \
    sort | uniq -c | sort -nr | \
    head -20
    
    # Extract performance metrics
    grep -E "duration:|latency:|response_time:" "$log_file"
}

# Find log files in project
custom::extract::find_logs() {
    local directory="$1"
    find "$directory" -name "*.log" -o -name "application.log*"
}
EOF

# Integrate into main extraction process
# Add call to manage.sh extraction workflow
```

## 📊 Performance Optimization

### 1. Efficient Search Strategies

```bash
# Use type filters for faster searches
resource-qdrant search "authentication" workflows  # Faster
resource-qdrant search "authentication"            # Slower

# Use specific apps when known
resource-qdrant search "user-portal-v1" "login flow"  # Faster

# Limit results for exploration
resource-qdrant search "database" code 5  # Top 5 results only
```

### 2. Batch Operations

```bash
# Refresh multiple apps efficiently
for app in app1 app2 app3; do
    resource-qdrant embeddings refresh "$app" &
done
wait  # Parallel refresh
```

### 3. Monitoring Usage

```bash
# Check embedding statistics
resource-qdrant embeddings status

# Monitor search performance
time resource-qdrant search-all "complex query"

# Check collection sizes
curl -s http://localhost:6333/collections | jq '.result[].vectors_count'
```

## 🔗 Integration Examples

### 1. VS Code Integration

Create a VS Code task for semantic search:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Search Knowledge",
            "type": "shell",
            "command": "vrooli",
            "args": ["resource-qdrant", "search-all", "${input:searchQuery}"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "new"
            }
        }
    ],
    "inputs": [
        {
            "id": "searchQuery",
            "description": "Enter search query",
            "default": "",
            "type": "promptString"
        }
    ]
}
```

### 2. Git Hooks Integration

Add pre-commit hook to suggest similar implementations:

```bash
#!/usr/bin/env bash
# .git/hooks/pre-commit

# Check for new functions being added
new_functions=$(git diff --cached --name-only | xargs grep -l "function\|def\|async function" 2>/dev/null || true)

if [[ -n "$new_functions" ]]; then
    echo "🔍 Checking for similar implementations..."
    
    # Extract function names and search for similar
    for file in $new_functions; do
        grep -E "(function|def|async function)" "$file" | while read -r line; do
            func_name=$(echo "$line" | sed -E 's/.*function\s+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/')
            if [[ -n "$func_name" ]]; then
                echo "Searching for similar to: $func_name"
                resource-qdrant search-all "$func_name" code 3
            fi
        done
    done
fi
```

### 3. CI/CD Integration

Add knowledge validation to CI pipeline:

```yaml
# .github/workflows/knowledge-validation.yml
name: Knowledge Validation

on: [push, pull_request]

jobs:
  validate-knowledge:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Vrooli
        run: |
          # Install Vrooli CLI
          curl -sSL https://install.vrooli.com | bash
          
      - name: Validate Knowledge Structure  
        run: |
          resource-qdrant embeddings validate
          
      - name: Check for Missing Documentation
        run: |
          resource-qdrant embeddings validate --strict
```

## 🔧 Troubleshooting Development Issues

### Common Integration Problems

**1. Embeddings not refreshing automatically:**

```bash
# Check git hook is installed
ls -la .git/hooks/post-commit

# Check manage.sh integration
grep -n "refresh_embeddings_on_changes" scripts/manage.sh

# Manual refresh
resource-qdrant embeddings refresh --force
```

**2. Search returning no results:**

```bash
# Check if embeddings exist
resource-qdrant embeddings status

# Validate content extraction
resource-qdrant embeddings validate

# Check Qdrant connection
curl http://localhost:6333/collections
```

**3. Slow embedding refresh:**

```bash
# Check what's being processed
resource-qdrant embeddings refresh --verbose

# Consider selective refresh
resource-qdrant embeddings refresh --type workflows

# Monitor resource usage
top -p $(pgrep qdrant)
```

## 📈 Best Practices Summary

### 1. Documentation Strategy
- Use embedding markers in all important docs
- Document patterns as you discover them
- Keep PRDs and technical docs current
- Add function/API documentation

### 2. Search Strategy
- Start broad, then narrow down
- Use type filters for performance
- Search across apps for patterns
- Document search patterns that work

### 3. Maintenance Strategy
- Let auto-refresh handle most updates
- Validate periodically with `validate` command
- Monitor performance with `status` command
- Clean up with `gc` when needed

### 4. Team Collaboration
- Share successful search patterns
- Document anti-patterns in lessons learned
- Use consistent embedding marker styles
- Regular knowledge validation in CI/CD

---

*Remember: The semantic knowledge system gets smarter as you use it. Every piece of knowledge you add becomes a building block for future solutions.*