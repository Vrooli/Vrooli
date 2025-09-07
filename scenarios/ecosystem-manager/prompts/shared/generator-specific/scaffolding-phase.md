# Scaffolding Phase

## Purpose
Scaffolding creates the minimal viable structure that future improvers can build upon. The goal is NOT to implement everything, but to create a solid foundation with correct structure and patterns.

## Scaffolding Allocation: 20% of Total Effort

### Scaffolding Philosophy

Think of scaffolding as **planting a seed**:
- Create the right structure
- Implement core patterns
- Provide clear extension points
- Leave room for growth

**Quality over Completeness** - Better to have 20% perfectly structured than 80% poorly organized.

## Scaffolding Process

### Step 1: Template Selection

Based on research, choose approach:

```bash
# Option A: Copy from template
cp -r scripts/scenarios/templates/[template-type]/* scenarios/[new-name]/
# OR
cp -r scripts/resources/templates/[template-type]/* resources/[new-name]/

# Option B: Copy from similar existing
cp -r scenarios/[similar-scenario]/* scenarios/[new-name]/
# OR  
cp -r resources/[similar-resource]/* resources/[new-name]/

# Option C: Hybrid approach
# Take structure from template, patterns from existing
```

### Step 2: Structure Creation

#### For Scenarios
```bash
scenarios/[name]/
├── .vrooli/
│   └── service.json          # Resource dependencies
├── api/
│   ├── main.go              # API server
│   ├── go.mod               # Dependencies
│   └── README.md            # API documentation
├── cli/
│   ├── [name]-cli           # CLI executable
│   ├── install.sh           # Installation script
│   └── README.md            # CLI documentation
├── ui/
│   ├── index.html           # Main UI
│   ├── package.json         # Dependencies
│   ├── server.js            # Dev server
│   └── README.md            # UI documentation
├── initialization/
│   └── [any seed data]      # Initial content
├── prompts/
│   └── prompt.md            # Agent prompts
├── tests/
│   └── test.sh              # Basic tests
├── PRD.md                   # Product requirements
└── README.md                # Main documentation
```

#### For Resources
```bash
resources/[name]/
├── lib/
│   ├── core.sh              # Core functions
│   ├── health.sh            # Health checks
│   ├── lifecycle.sh         # Setup/start/stop
│   └── content.sh           # Content management
├── cli.sh                   # CLI entry point
├── inject.sh                # Content injection
├── manage.sh                # Management scripts
├── service.json             # Configuration
├── Dockerfile               # If containerized
├── docker-compose.yml       # If using compose
├── PRD.md                   # Product requirements
└── README.md                # Documentation
```

### Step 3: Core Implementation

#### Minimal Viable Functionality

Implement ONLY:
1. **Health check endpoint** - Must respond to health checks
2. **Basic lifecycle** - Must start/stop cleanly
3. **One P0 requirement** - Prove the concept works
4. **Basic CLI command** - Minimum interaction

Example minimal API (Go):
```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }

    // Health check - REQUIRED
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]bool{"healthy": true})
    })

    // One core endpoint - prove it works
    http.HandleFunc("/api/core", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]string{
            "status": "operational",
            "message": "Core functionality placeholder",
        })
    })

    log.Printf("Starting server on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

Example minimal CLI (Bash):
```bash
#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/core.sh"

case "$1" in
    setup)
        echo "Setting up [name]..."
        # Minimal setup
        ;;
    start|develop)
        echo "Starting [name]..."
        # Start the service
        ;;
    stop)
        echo "Stopping [name]..."
        # Stop the service
        ;;
    health)
        check_health
        ;;
    help|*)
        echo "Usage: $0 {setup|start|stop|health|help}"
        ;;
esac
```

### Step 4: Configuration

#### service.json Template
```json
{
  "name": "[name]",
  "version": "0.1.0",
  "description": "[One line description]",
  "status": "scaffold",
  "type": "[scenario|resource]",
  "category": "[category]",
  "ports": {
    "api": 3000,
    "ui": 3001
  },
  "dependencies": {
    "required": ["postgres", "qdrant"],
    "optional": ["redis"]
  },
  "resources": {
    "memory": "256MB",
    "cpu": "0.5"
  }
}
```

### Step 5: Documentation

#### Minimal README.md
```markdown
# [Name]

[One paragraph description based on PRD]

## Status
🚧 **Scaffold** - Basic structure in place, ready for improvement

## Quick Start
\`\`\`bash
# Setup
vrooli scenario [name] setup

# Run
vrooli scenario [name] run

# Test
vrooli scenario [name] test
\`\`\`

## Completed Features
- ✅ Basic structure
- ✅ Health checks
- ✅ [One P0 feature]

## TODO
See PRD.md for complete requirements list.

## Development
This is a scaffold ready for improvement. See PRD.md for requirements and priorities.
```

## Scaffolding Quality Checklist

### Structure
- [ ] Correct directory structure for type
- [ ] All required files present
- [ ] Consistent naming throughout
- [ ] Proper file permissions

### Functionality
- [ ] Health check responds
- [ ] Service starts/stops cleanly
- [ ] One P0 requirement works
- [ ] Basic CLI commands function

### Configuration
- [ ] service.json valid and complete
- [ ] Ports allocated correctly
- [ ] Dependencies listed
- [ ] Resource limits set

### Documentation
- [ ] PRD.md complete and detailed
- [ ] README.md explains status
- [ ] Setup instructions work
- [ ] TODOs clearly marked

## Scaffolding Anti-Patterns

### Over-Implementation
❌ **Bad**: Trying to implement all P0 requirements
✅ **Good**: One working P0 + solid structure

### Under-Documentation
❌ **Bad**: "TODO: Add documentation"
✅ **Good**: Complete PRD + basic README

### Poor Structure
❌ **Bad**: Everything in one file
✅ **Good**: Proper separation of concerns

### No Extension Points
❌ **Bad**: Hardcoded, rigid implementation
✅ **Good**: Clear places to add features

## Scaffolding Success Metrics

- **Can improvers understand it?** Clear structure and docs
- **Can it be extended easily?** Good patterns in place
- **Does it actually run?** Basic functionality works
- **Is the vision clear?** PRD tells the story

## Remember for Scaffolding

**You're planting a seed** - Focus on strong roots, not full growth

**Structure matters most** - Right patterns enable everything

**Document the vision** - PRD guides all future work  

**Prove the concept** - One working feature validates approach

**Leave room to grow** - Don't over-constrain the future

The scaffold is the foundation. Make it solid, clear, and extensible. Future improvers will thank you.