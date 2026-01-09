# Visited Tracker - Prompt Construction Guide

## Overview

This guide shows how to construct effective AI prompts that leverage visited-tracker to systematically analyze codebases. The visited-tracker ensures comprehensive coverage by tracking which files have been reviewed, preventing redundant work, and prioritizing neglected files.

## Core Concept

**The Problem:** When analyzing large codebases across multiple AI conversations, you lose track of which files have been reviewed, leading to:
- Duplicate analysis of the same files
- Files that never get reviewed
- No systematic coverage guarantee

**The Solution:** Use visited-tracker to maintain persistent state across conversations, prioritizing files based on:
- **Visit count:** How many times a file has been analyzed
- **Staleness score:** Time since last visit + file modification recency
- **Coverage tracking:** Percentage of files analyzed in a campaign

---

## Setting Up a Campaign

Before using visited-tracker in prompts, create a campaign:

```bash
# Create a campaign for Go API files
visited-tracker campaigns create \
  --name "api-bug-sweep-2024" \
  --pattern "**/*.go" \
  --description "Systematic bug detection across all Go files" \
  --from-agent "claude-sonnet"

# Export the campaign ID for future use
export VISITED_TRACKER_CAMPAIGN_ID="<returned-campaign-id>"
```

**Alternative:** Use the web UI to create campaigns visually at the dashboard.

---

## Prompt Pattern 1: Bug Detection Workflow

### Goal
Systematically check every file in a codebase for potential bugs without repeating work across sessions.

### Example Prompt

```
I want to perform a comprehensive bug sweep across all Go files in this project.

TASK:
1. Use visited-tracker to get the 10 least-visited files matching **/*.go
2. For each file:
   - Analyze for common bug patterns:
     * Race conditions and concurrency issues
     * Nil pointer dereferences
     * Resource leaks (unclosed files, connections)
     * Error handling gaps
     * Off-by-one errors
     * Type conversion issues
   - Document any findings
3. Record your visit for each file using visited-tracker
4. Show progress: "Analyzed 10 of 247 total files (4% coverage)"

CAMPAIGN: api-bug-sweep-2024 (VISITED_TRACKER_CAMPAIGN_ID already set)

After analyzing these 10 files, I'll run this prompt again in the next session
to continue where we left off with the next 10 least-visited files.
```

### Why This Works
- **Stateful:** Each session picks up where the last one left off
- **No duplicates:** Already-visited files won't be re-analyzed
- **Guaranteed coverage:** Eventually every file gets reviewed
- **Progress tracking:** You can see completion percentage

---

## Prompt Pattern 2: Performance Analysis

### Goal
Identify performance bottlenecks by analyzing files that haven't been reviewed recently or have been modified without re-analysis.

### Example Prompt

```
Perform performance analysis on stale files in the codebase.

TASK:
1. Use visited-tracker to get the 5 most-stale files with threshold 5.0
   (Files that have been modified recently but not visited, or visited long ago)
2. For each file, analyze for:
   - O(n²) or worse algorithm complexity
   - Unnecessary allocations in hot paths
   - Missing caching opportunities
   - Database query inefficiencies (N+1 queries, missing indexes)
   - Inefficient string concatenation
   - Memory leaks
3. Record performance findings with context "performance"
4. Mark each file as visited after analysis

CAMPAIGN: backend-performance-audit

Focus on files with high staleness scores first - these are likely modified
recently but haven't been reviewed, or were reviewed long ago and may have
accumulated technical debt.
```

### Advanced Variant

```
I need to prioritize performance analysis based on git activity.

TASK:
1. Get the 10 most-stale files from visited-tracker
2. For files modified in the last 30 days (check git log):
   - Deep performance analysis with profiling recommendations
   - Identify specific hot paths using algorithmic analysis
3. For older files:
   - Quick scan for obvious inefficiencies
4. Record all visits with context "performance" and agent "claude-sonnet"
5. Track conversation_id so I can see which files were analyzed together

This approach ensures recently-changed code gets thorough performance review
while still providing coverage for older code.
```

---

## Prompt Pattern 3: Documentation Coverage

### Goal
Ensure every file has sufficient documentation, docstrings, and comments.

### Example Prompt

```
Systematically improve documentation across the codebase.

TASK:
1. Use visited-tracker to get 10 least-visited files
2. For each file, check:
   - File-level documentation (purpose, main responsibilities)
   - Function/method docstrings (all exported functions documented)
   - Complex logic has explanatory comments
   - Public APIs have usage examples
   - Edge cases are documented
3. For files with insufficient docs:
   - Add comprehensive documentation
   - Follow project documentation style
4. Record visit with context "docs" after completing documentation
5. Show coverage: "Documented 10 of 156 files (6% complete)"

CAMPAIGN: docs-coverage-2024

Goal: Achieve 100% documentation coverage within 15 sessions (150 files per session).
```

---

## Prompt Pattern 4: Security Audit

### Goal
Systematically audit code for security vulnerabilities.

### Example Prompt

```
Conduct security audit using visited-tracker for systematic coverage.

TASK:
1. Get 8 least-visited files from visited-tracker
2. Security analysis checklist per file:
   - Input validation (injection attacks, XSS, SQL injection)
   - Authentication/authorization checks
   - Secrets management (no hardcoded credentials)
   - Cryptography usage (strong algorithms, proper key management)
   - Data exposure risks
   - CSRF/SSRF vulnerabilities
   - Dependency vulnerabilities
3. For each finding:
   - Severity: Critical/High/Medium/Low
   - Recommended fix
   - CWE reference if applicable
4. Record visit with context "security" and findings metadata
5. Generate summary report

CAMPAIGN: security-audit-q4-2024

This systematic approach ensures no file escapes security review.
```

---

## Prompt Pattern 5: Test Coverage Improvement

### Goal
Identify files that need test coverage and systematically add tests.

### Example Prompt

```
Improve test coverage systematically using visited-tracker.

TASK:
1. Get 10 least-visited files from pattern "**/*.go" (excluding *_test.go)
2. For each file:
   - Check if corresponding _test.go file exists
   - Analyze current test coverage (if tests exist)
   - Identify untested functions and edge cases
   - Write comprehensive tests for untested code
3. Record visit with context "testing"
4. Show: "Added tests for 10 of 89 untested files (11% complete)"

CAMPAIGN: test-coverage-improvement

Target: 80% test coverage across all Go files.
This systematic approach ensures we don't miss any files.
```

---

## Prompt Pattern 6: Code Quality Refactoring

### Goal
Incrementally improve code quality across a large codebase.

### Example Prompt

```
Perform systematic code quality improvements using visited-tracker.

TASK:
1. Get 5 most-stale files (these haven't been refactored recently)
2. For each file, check for:
   - Code complexity (cyclomatic complexity > 10)
   - Long functions (> 50 lines)
   - Duplicate code
   - Magic numbers
   - Poor naming
   - Missing error handling
   - Type safety issues
3. Refactor issues found (one file at a time)
4. Record visit with context "refactoring" and findings metadata
5. Run tests to ensure refactoring didn't break anything

CAMPAIGN: code-quality-initiative

This ensures every file gets attention over time, prioritizing files that
haven't been touched in a while (high staleness = technical debt accumulation).
```

---

## Advanced Techniques

### Using Context for Multi-Dimensional Tracking

You can track different types of analysis independently by using the `--context` flag:

```bash
# Security analysis
visited-tracker visit src/auth.go --context security

# Performance analysis (separate tracking)
visited-tracker visit src/auth.go --context performance

# Get least-visited for specific context
visited-tracker least-visited --context security --limit 10
```

This allows you to run multiple parallel campaigns on the same files (e.g., one for security, one for performance).

### Conversation Grouping

Track which files were analyzed together in a single conversation:

```bash
# Generate a conversation ID
CONV_ID=$(uuidgen)

# Record all visits in this conversation with the same ID
visited-tracker visit file1.go --conversation "$CONV_ID"
visited-tracker visit file2.go --conversation "$CONV_ID"
visited-tracker visit file3.go --conversation "$CONV_ID"
```

### Exporting Progress Reports

```bash
# Export campaign data for analysis
visited-tracker export campaign-report.json --format json

# Check coverage statistics
visited-tracker coverage --group-by directory

# View campaign status
visited-tracker status
```

---

## Integration with AI Workflows

### Multi-Session Systematic Review

**Session 1 Prompt:**
```
Start systematic code review campaign.

1. Create campaign: "api-review-2024" for pattern "api/**/*.go"
2. Get 10 least-visited files
3. Review each file for bugs, performance, security
4. Record visits
5. Show: "Session 1: Reviewed 10 of 127 files (8%)"
```

**Session 2 Prompt (hours/days later):**
```
Continue systematic code review campaign.

1. Using campaign "api-review-2024"
2. Get 10 least-visited files (will automatically skip Session 1 files)
3. Review each file
4. Record visits
5. Show: "Session 2: Reviewed 20 of 127 files (16%)"
```

This pattern continues until 100% coverage is achieved.

### Automated Campaign Tracking

```bash
#!/bin/bash
# Script to track systematic review progress

CAMPAIGN_ID="your-campaign-id"
TOTAL_FILES=$(visited-tracker coverage --json | jq '.total_files')
VISITED=$(visited-tracker coverage --json | jq '.visited_files')
COVERAGE=$(visited-tracker coverage --json | jq '.coverage_percentage')

echo "Campaign Progress: $VISITED/$TOTAL_FILES files ($COVERAGE%)"

if [ "$COVERAGE" -lt 100 ]; then
    echo "Still have files to review. Continue with next session."
else
    echo "✅ Campaign complete! All files have been analyzed."
fi
```

---

## Best Practices

### ✅ DO

- **Create focused campaigns** - Separate campaigns for different analysis types (bugs, perf, security)
- **Use consistent patterns** - Stick to the same glob patterns within a campaign
- **Record every visit** - Even if you find nothing, record the visit to track coverage
- **Check coverage regularly** - Monitor progress with `visited-tracker status`
- **Use context tags** - Tag visits with context (security, performance, etc.) for better tracking
- **Batch appropriately** - Process 5-10 files per session for thorough analysis

### ❌ DON'T

- **Don't skip recording visits** - Breaks the systematic coverage guarantee
- **Don't analyze files manually** - Always get files from visited-tracker to maintain order
- **Don't mix unrelated patterns** - Keep campaigns focused (e.g., don't mix Go and JS in one campaign)
- **Don't delete campaigns prematurely** - Keep historical data for trend analysis

---

## Troubleshooting

### "No campaigns available"
```bash
# Create a campaign first
visited-tracker campaigns create --name "my-campaign" --pattern "**/*.go"
```

### "API not running"
```bash
# Start visited-tracker
vrooli scenario start visited-tracker

# Verify it's running
visited-tracker status
```

### "Coverage stuck at X%"
```bash
# Check for deleted/moved files
visited-tracker sync --remove-deleted

# View remaining unvisited files
visited-tracker least-visited --limit 20
```

---

## API Reference Quick Guide

### Common CLI Commands

```bash
# Campaign management
visited-tracker campaigns list
visited-tracker campaigns create --name "NAME" --pattern "PATTERN"
visited-tracker campaigns delete CAMPAIGN_ID

# Get files to analyze
visited-tracker least-visited --limit 10
visited-tracker most-stale --threshold 5.0 --limit 10

# Record analysis
visited-tracker visit file1.go file2.go --context review
visited-tracker visit file.go --agent claude-sonnet --conversation CONV_ID

# Track progress
visited-tracker status
visited-tracker coverage
visited-tracker coverage --group-by directory

# Data management
visited-tracker sync --remove-deleted
visited-tracker export report.json
```

### HTTP API Endpoints

```bash
# Create campaign
curl -X POST http://localhost:17695/api/v1/campaigns \
  -H "Content-Type: application/json" \
  -d '{"name":"campaign","patterns":["**/*.go"],"from_agent":"manual"}'

# Get least visited files
curl http://localhost:17695/api/v1/campaigns/CAMPAIGN_ID/prioritize/least-visited?limit=10

# Record visit
curl -X POST http://localhost:17695/api/v1/campaigns/CAMPAIGN_ID/visit \
  -H "Content-Type: application/json" \
  -d '{"files":["src/main.go"],"context":"review"}'

# Get coverage stats
curl http://localhost:17695/api/v1/campaigns/CAMPAIGN_ID/coverage
```

---

## Real-World Example: Full Bug Sweep

This example shows a complete workflow for systematically finding and fixing bugs across a large codebase.

```bash
# Day 1: Setup
visited-tracker campaigns create \
  --name "bug-sweep-2024-q4" \
  --pattern "**/*.go" \
  --description "Comprehensive bug detection across Go codebase" \
  --from-agent "claude-sonnet"

export VISITED_TRACKER_CAMPAIGN_ID="<campaign-id>"

# Prompt for Claude:
"""
Start Day 1 of systematic bug sweep.

TASK:
1. Get 10 least-visited Go files
2. Analyze each for bugs (race conditions, nil pointers, resource leaks, error handling)
3. For each bug found:
   - Explain the bug
   - Show the problematic code
   - Provide a fix
4. Record all visits
5. Summary: "Day 1: Analyzed X files, found Y bugs in Z files"

Campaign: bug-sweep-2024-q4
Target: Complete 247 files over 25 sessions (10 files/session)
"""

# Day 2: Continue (using same campaign ID)
"""
Continue Day 2 of systematic bug sweep.

TASK: [same as Day 1]

This will automatically pick the next 10 least-visited files.
"""

# ... continue for 25 days ...

# Day 25: Final session
"""
Final session of systematic bug sweep.

TASK: [same as Day 1]

After this session, generate comprehensive report:
- Total files analyzed: X
- Bugs found: Y
- Bug categories breakdown
- Most common bug types
- Files with most issues
"""

# Generate final report
visited-tracker coverage --group-by directory > bug-sweep-report.txt
visited-tracker export bug-sweep-complete.json
```

---

## Conclusion

Visited-tracker transforms ad-hoc code analysis into systematic, stateful campaigns that guarantee comprehensive coverage. By using these prompt patterns, you ensure:

✅ **No file left behind** - Every file gets analyzed eventually
✅ **No duplicate work** - Files aren't re-analyzed unnecessarily
✅ **Measurable progress** - Track coverage percentage across sessions
✅ **Intelligent prioritization** - Stale files get attention when needed
✅ **Multi-dimensional analysis** - Track security, performance, bugs separately

Start with simple campaigns and evolve to complex multi-session workflows as you become familiar with the patterns.
