# AI-Powered Content Creation Scripts

This directory contains scripts for using AI to generate and validate content for the Vrooli application.

## Scripts Overview

### Core AI Workflow Scripts
- **`maintenance-agent.sh`** - AI agent for maintenance tasks and general automation
- **`maintenance-supervisor.sh`** - Supervisor for AI maintenance workflows
- **`routine-generate.sh`** - Generate routine definitions using AI from backlog items
- **`routine-import.sh`** - Import and validate generated routines into the application

### Validation & Reference Scripts
- **`validate-routine.sh`** - Validate routine JSON structure, fields, and configuration
- **`validate-subroutines.sh`** - Validate subroutine references and check for TODO placeholders
- **`routine-reference-generator.sh`** - Generate routine-reference.json from staged files

## Usage

### Generate Routines
```bash
# Generate routines from backlog using AI
./scripts/ai-creation/routine-generate.sh

# Or use the enhanced generation (located in scripts/main/)
./scripts/main/routine-generate-enhanced.sh
```

### Validate Content
```bash
# Quick check for TODO placeholders
./scripts/ai-creation/validate-subroutines.sh --todo-only

# Full validation of routine structure
./scripts/ai-creation/validate-routine.sh docs/ai-creation/routine/staged/*.json

# Full validation of subroutine references
./scripts/ai-creation/validate-subroutines.sh
```

### Update Reference File
```bash
# Regenerate routine reference after adding new routines
./scripts/ai-creation/routine-reference-generator.sh
```

### Import Routines
```bash
# Import validated routines into the application
./scripts/ai-creation/routine-import.sh
```

## Related Documentation

- [AI Routine Creation System](../../docs/ai-creation/routine/README.md) - Complete guide to the routine creation pipeline
- [Routine Generation Prompts](../../docs/ai-creation/routine/prompts/) - AI prompts for different routine types
- [Staged Routines](../../docs/ai-creation/routine/staged/) - Generated routine definitions ready for import

## Workflow

1. **Generate** - Use AI to create routine definitions from backlog items
2. **Validate** - Check structure and subroutine references
3. **Import** - Load validated routines into the application
4. **Test** - Verify routines work correctly in the application

These scripts work together to provide an AI-powered content creation pipeline for Vrooli.