# AI Maintenance Tracking System

This document defines a systematic approach for tracking repetitive AI maintenance tasks to avoid redundant work.

## Comment Format

```typescript
// AI_CHECK: TASK_ID=count,TASK_ID=count | LAST: YYYY-MM-DD
```

**Examples:**
```typescript
// AI_CHECK: TEST_QUALITY=3,REACT_PERF=1 | LAST: 2025-06-14
// AI_CHECK: TYPE_SAFETY=2 | LAST: 2025-06-13
// AI_CHECK: ERROR_HANDLING=1,ACCESSIBILITY=1 | LAST: 2025-06-12
```

## Standard Task IDs

| Task ID | Instructions (MUST READ) | When to Use |
|---------|-------------|-------------|
| `TEST_QUALITY` | [TEST_QUALITY.md](./tasks/TEST_QUALITY.md) | User asks to identify/fix tests written to pass |
| `TEST_COVERAGE` | [TEST_COVERAGE.md](./tasks/TEST_COVERAGE.md) | User asks to improve test coverage |
| `REACT_PERF` | [REACT_PERF.md](./tasks/REACT_PERF.md) | User asks to improve React component performance |
| `TYPE_SAFETY` | [TYPE_SAFETY.md](./tasks/TYPE_SAFETY.md) | User asks to improve type safety |
| `ERROR_HANDLING` | [ERROR_HANDLING.md](./tasks/ERROR_HANDLING.md) | User asks to improve error handling |
| `ACCESSIBILITY` | [ACCESSIBILITY.md](./tasks/ACCESSIBILITY.md) | User asks to improve accessibility |
| `SECURITY` | [SECURITY.md](./tasks/SECURITY.md) | User asks to review security |
| `CODE_QUALITY` | [CODE_QUALITY.md](./tasks/CODE_QUALITY.md) | User asks for code quality review |
| `EASY_WINS` | [EASY_WINS.md](./tasks/EASY_WINS.md) | User asks to suggest "easy wins" |
| `DEAD_CODE` | [DEAD_CODE.md](./tasks/DEAD_CODE.md) | User asks to identify unused code |
| `TODO_CLEANUP` | [TODO_CLEANUP.md](./tasks/TODO_CLEANUP.md) | User asks to address TODOs/FIXMEs |
| `DOCS_TO_CODE` | Update documentation to match actual code behavior | User asks to sync docs with current code |
| `CODE_TO_DOCS` | Update code to match documented intended behavior | User asks to sync code with documented specs |
| `PERF_GENERAL` | [PERF_GENERAL.md](./tasks/PERF_GENERAL.md) | User asks to improve general performance |
| `LOGGING` | Logging improvements and consistency | User asks to improve logging |
| `API_DESIGN` | API endpoint design improvements | User asks to improve API design |
| `DATABASE` | Database query optimization and design | User asks to optimize database usage |
| `BUNDLE_SIZE` | Bundle size optimization | User asks to optimize bundle size |
| `IMPORTS` | [IMPORTS.md](./tasks/IMPORTS.md) | User asks to clean up imports |
| `COMMENTS` | [COMMENTS.md](./tasks/COMMENTS.md) | User asks to improve/update comments |
| `FORMS` | Form validation, error handling, UX consistency | User asks to improve form implementations |
| `STATE_MGMT` | State management patterns and optimization | User asks to improve state management |
| `ASYNC_SAFETY` | Race conditions, proper async/await usage, error handling | User asks to fix async/concurrency issues |
| `MEMORY_LEAKS` | Event listeners, subscriptions, cleanup | User asks to identify/fix memory leaks |
| `LOADING_STATES` | [LOADING_STATES.md](./tasks/LOADING_STATES.md) | User asks to improve loading UX |
| `RESPONSIVE` | Mobile responsiveness and cross-device compatibility | User asks to improve responsive design |
| `INTERNATIONALIZATION` | i18n completeness and consistency | User asks to improve internationalization |
| `MONITORING` | Missing metrics, health checks, alerting | User asks to improve monitoring/observability |
| `STARTUP_ERRORS` | [STARTUP_ERRORS.md](./tasks/STARTUP_ERRORS.md) | User asks to identify and fix development environment startup issues |

## Query Commands

### Find files by task type
```bash
# Find all files checked for test quality
rg "AI_CHECK:.*TEST_QUALITY" --type ts --type tsx

# Find files never checked for React performance
rg -L "AI_CHECK:.*REACT_PERF" packages/ui/src --type ts --type tsx
```

### Sort by check frequency
```bash
# Find files with highest TEST_QUALITY check count
rg "AI_CHECK:.*TEST_QUALITY=(\d+)" -o -r '$1' --type ts | sort -nr | head -10

# Find files checked least recently
rg "AI_CHECK:.*LAST: (\d{4}-\d{2}-\d{2})" -o -r '$1' --type ts | sort | head -10
```

### Comprehensive file analysis
```bash
# Get all AI_CHECK comments with filenames
rg "AI_CHECK:" --type ts --type tsx -n

# Count total checks per task type
rg "AI_CHECK:.*TEST_QUALITY=(\d+)" -o -r '$1' --type ts | awk '{sum+=$1} END {print "TEST_QUALITY total:", sum}'
```

## Usage Instructions for Claude

### When starting a maintenance task:

1. **Query existing checks:**
   ```bash
   rg "AI_CHECK:.*TASK_ID" --type ts --type tsx
   ```

2. **Prioritize unchecked files:**
   - Files with no AI_CHECK comment (never checked)
   - Files with low count for the specific task
   - Files with old LAST dates

3. **After checking a file:**
   - Add/update the AI_CHECK comment
   - Increment the count for the task ID
   - Update the LAST date

### Example workflow:
```bash
# User asks: "Identify tests written to pass instead of testing expected behavior"

# 1. Find files already checked for TEST_QUALITY
rg "AI_CHECK:.*TEST_QUALITY" --type ts

# 2. Find test files not yet checked
find . -name "*.test.ts" -o -name "*.test.tsx" | xargs rg -L "AI_CHECK:.*TEST_QUALITY"

# 3. Prioritize files with no checks or oldest dates
# 4. After reviewing, add/update comment:
// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-15
```

## Comment Placement

- Place AI_CHECK comments at the top of the file, after imports
- For test files, place after describe block if more appropriate
- Keep consistent placement within each file type

## Maintenance

- Comments should be updated atomically with the actual maintenance work
- Remove outdated comments if files are significantly refactored
- Use git history to verify comment accuracy if needed