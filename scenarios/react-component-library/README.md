# React Component Library

Central UI for designing, previewing, editing, and tracking shared React UI components across the Vrooli ecosystem.

## Overview

React Component Library provides a comprehensive workflow hub for component creation, AI-powered refinement, multi-viewport testing, and adoption tracking. It enables:

- **Component Registry**: Centralized registry of shared UI components with metadata and versioning
- **Code Editor & Preview**: Full-featured TSX editor with live preview in isolated iframes
- **Multi-Viewport Emulator**: Simultaneous preview across desktop, mobile, and tablet viewports
- **AI-Powered Editing**: Context-aware code suggestions and refactoring via resource-openrouter
- **Element Selection**: iframe-bridge integration for precise element targeting
- **Adoption Workflow**: Integration with app-issue-tracker for component adoption across scenarios
- **Version Tracking**: Track component versions, detect changes, and view diffs

## Purpose

This scenario eliminates component duplication across scenarios, accelerates UI development with AI assistance, ensures design consistency through a shared library, and enables systematic component evolution with version tracking and diff views.

## Quick Start

```bash
# From repo root
cd scenarios/react-component-library

# Setup (builds API, UI, installs CLI)
vrooli scenario run react-component-library --setup

# Start development servers
make dev
# OR
vrooli scenario run react-component-library --dev

# Access UI
open http://localhost:$(vrooli scenario port react-component-library UI_PORT)
```

## Running Tests

```bash
# Run all test phases
make test

# Or run specific phases
cd test
./run-tests.sh
```

Tests are organized by phase:
- **Structure**: Validates scenario structure, configuration, and basic health
- **Unit**: Tests individual components and functions
- **Integration**: Tests API endpoints and component interactions
- **Business**: Tests business logic and workflows
- **Performance**: Tests performance benchmarks and optimization

Tag tests with `[REQ:ID]` to link them to requirements in `requirements/`.

## CLI Usage

```bash
# Check status
react-component-library status

# List components
react-component-library components list

# Search components
react-component-library components search --query "button"

# Show adoption status
react-component-library adoptions list
```

## Architecture

### Frontend (ui/)
- React 18+ with TypeScript
- TailwindCSS + shadcn/ui components
- Lucide icons
- Monaco Editor or CodeMirror for code editing
- iframe-bridge for element selection
- Vite for fast HMR

### Backend (api/)
- Go API server
- PostgreSQL for component registry and adoption tracking
- REST API for component CRUD, search, and adoption workflows

### Dependencies
- **postgres**: Component registry, adoption records, version history
- **resource-openrouter**: AI-powered code editing and refactoring
- **app-issue-tracker**: Integration for adoption workflow issue generation
- **app-monitor**: Reference for iframe-bridge patterns

## Development Guidelines

1. **PRD First**: See [PRD.md](PRD.md) for operational targets and requirements
2. **Track Progress**: Update [docs/PROGRESS.md](docs/PROGRESS.md) when landing work
3. **Log Problems**: Document issues in [docs/PROBLEMS.md](docs/PROBLEMS.md)
4. **Link Tests**: Tag tests with `[REQ:ID]` to connect to requirements

## Component Header Format

All library components must include a standardized header comment:

```tsx
/**
 * @libraryId react-component-library:ButtonPrimary
 * @displayName Primary Button
 * @description High-emphasis CTA button for primary actions.
 * @version 1.2.0
 * @sourcePath /path/to/library/components/ButtonPrimary.tsx
 * @warning DO NOT REMOVE OR EDIT THIS COMMENT.
 *          Used by react-component-library to track shared component adoption.
 */
```

This header enables:
- Component registry indexing from disk
- Adoption tracking across scenarios
- Version management and change detection
- Diff views between library and adopted versions

## Documentation

- [PRD.md](PRD.md) - Product requirements and operational targets
- [docs/PROGRESS.md](docs/PROGRESS.md) - Development progress log
- [docs/PROBLEMS.md](docs/PROBLEMS.md) - Known issues and deferred ideas
- [docs/RESEARCH.md](docs/RESEARCH.md) - Research notes and references
- [requirements/](requirements/) - Technical requirements registry

## Related Scenarios

- **app-monitor**: Reference for iframe-bridge integration and emulator patterns
- **app-issue-tracker**: Integration target for adoption workflow
- **brand-manager**: Future integration for design tokens and theming
- **tidiness-manager**: Could leverage component library for UI consistency

## License

MIT
