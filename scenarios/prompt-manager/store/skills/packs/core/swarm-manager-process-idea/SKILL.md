# Process: Idea

## Purpose

Create or improve a Vrooli scenario based on a fully-refined idea. Transform the enhanced specification into working code.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — shared processing workflow and decision hierarchy.

## Output Requirements

### For New Scenario
- Create scenario in `scenarios/[scenario-name]/`
- Include `service.json` for lifecycle management
- Include API and/or UI as specified
- Write `notes.md` in item folder with completion summary

### For Scenario Improvement
- Modify existing scenario at specified path
- Preserve existing functionality unless explicitly changing
- Write `notes.md` in item folder with completion summary

## Success Criteria

- [ ] Scenario builds without errors
- [ ] API responds to health checks
- [ ] UI loads in browser (if applicable)
- [ ] All accepted suggestions implemented
- [ ] All answered questions respected
- [ ] service.json valid and complete
- [ ] Completion summary written

## Instructions

You are processing an idea to create or improve a Vrooli scenario. Your goal is to produce working, production-quality code that matches the refined specification.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Read all context**
   - Start with `enhance/summary.md` (the refined plan)
   - Review `clarify/questions.json` for answered questions
   - Check `suggest/suggestions.json` for accepted suggestions
   - Read `research/summary.md` if available
   - Note any user-added context files

2. **Determine operation type**
   - **New scenario**: No existing scenario, create from scratch
   - **Improve existing**: Target scenario exists, enhance it

3. **Plan the implementation**
   - List all files to create/modify
   - Identify integration points
   - Note dependencies on resources/other scenarios

4. **Implement systematically**

   For new scenarios:
   ```
   scenarios/[name]/
   ├── service.json      # Lifecycle configuration
   ├── api/              # Backend (if needed)
   │   ├── main.go
   │   ├── go.mod
   │   └── internal/
   ├── ui/               # Frontend (if needed)
   │   ├── package.json
   │   ├── vite.config.ts
   │   └── src/
   └── README.md         # Basic documentation
   ```

5. **Integrate with Vrooli**
   - Use existing resources (postgres, redis, etc.)
   - Follow established patterns from similar scenarios
   - Register in `.vrooli/service.json` if needed

6. **Verify the implementation**
   - Build passes
   - Services start
   - Basic functionality works
   - No security issues

7. **Write completion summary**
   - Document what was created
   - Note any deviations from plan
   - List verification steps completed

### Implementation Guidelines

#### API Implementation
```go
// Follow standard patterns
package main

import (
    "net/http"
    "log"
)

func main() {
    // Health check endpoint
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })

    // API routes
    // ...

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

#### UI Implementation
```typescript
// Use React + Vite
// Follow existing UI patterns
// Use Tailwind for styling
```

#### service.json
```json
{
  "name": "{{ITEM_NAME}}",
  "version": "1.0.0",
  "description": "{{ITEM_TITLE}}",
  "services": {
    "api": {
      "type": "go",
      "port": 8080,
      "health": "/health"
    },
    "ui": {
      "type": "vite",
      "port": 3000
    }
  },
  "resources": ["postgres", "redis"]
}
```

### Quality Standards

- **Code quality**: Clean, readable, follows Go/TypeScript best practices
- **Security**: No hardcoded secrets, input validation, no injection vulnerabilities
- **Performance**: Efficient queries, appropriate caching
- **Maintainability**: Clear structure, documented APIs
- **Testing**: Include basic tests where appropriate

## Quality Guidelines

**Good implementation:**
- Matches the refined specification exactly
- Uses existing resources appropriately
- Clean, maintainable code
- Complete service.json
- Thorough completion summary

**Poor implementation:**
- Deviates from spec without explanation
- Reinvents existing resources
- Hacky or unmaintainable code
- Missing configuration
- No documentation of what was done

## Anti-Patterns

- **Don't** implement rejected suggestions
- **Don't** add features not in the specification
- **Don't** ignore answered questions
- **Don't** skip the completion summary
- **Don't** leave code that doesn't build
- **Don't** hardcode credentials or secrets
