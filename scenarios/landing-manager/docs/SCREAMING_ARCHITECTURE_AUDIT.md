---
title: "Architecture Audit"
description: "Code organization and architectural patterns"
category: "technical"
order: 1
audience: ["developers"]
---

# Screaming Architecture Audit: landing-manager

**Date**: 2025-11-29 (Updated)
**Auditor**: Claude Code (Opus 4.5)
**Status**: PASSING - Architecture is well-aligned with domain

---

## Executive Summary

The landing-manager scenario has **excellent architecture alignment** following the previous refactoring effort. The codebase now "screams" its purpose as a Landing Page Factory:

- **Template Registry** - Browse and select landing page templates
- **Scenario Generation** - Create complete landing scenarios from templates
- **Lifecycle Management** - Start/stop/monitor generated scenarios
- **Agent Customization** - Trigger AI-based customization via issue tracker

**Previous audit findings have been addressed.** No critical architectural issues remain.

---

## 1. Domain & Mental Model

### What This Scenario Exists To Do

Landing Manager is a **meta-scenario (factory)** that:
1. **Template Registry** - Manages a catalog of landing page templates
2. **Scenario Generation** - Creates complete landing-page scenarios from templates
3. **Lifecycle Orchestration** - Starts/stops/manages generated scenarios in staging
4. **Agent Integration** - Triggers Claude Code for customization handoff
5. **Generation Analytics** - Tracks factory-level metrics

### Primary Actors

| Actor | Role |
|-------|------|
| **Factory User** | Uses factory UI to browse templates, generate scenarios, manage lifecycle |
| **Template** | JSON definition describing a landing page configuration |
| **Generated Scenario** | Complete standalone landing page app created from a template |
| **Agent** | Claude Code for customizing generated landing pages |
| **Persona** | Predefined agent personality/guidance for customization |

---

## 2. Current Architecture Assessment

### API Layer (Go)

| File | Lines | Responsibility | Status |
|------|-------|----------------|--------|
| `main.go` | 248 | Server wiring only | **Excellent** |
| `handlers/handlers.go` | 75 | Handler infrastructure | **Excellent** |
| `handlers/lifecycle.go` | 431 | Lifecycle operations | **Good** |
| `handlers/customize.go` | 205 | Agent customization | **Excellent** |
| `handlers/generate.go` | 113 | Generation endpoint | **Excellent** |
| `handlers/templates.go` | 41 | Template listing | **Excellent** |
| `handlers/personas.go` | 41 | Persona endpoints | **Excellent** |
| `handlers/analytics.go` | 27 | Analytics endpoints | **Excellent** |
| `handlers/health.go` | 30 | Health check | **Excellent** |
| `services/scenario_generator.go` | 588 | Generation logic | **Good** |
| `services/template_registry.go` | 282 | Template management | **Excellent** |
| `services/analytics_service.go` | 194 | Analytics tracking | **Excellent** |
| `services/persona_service.go` | 69 | Persona management | **Excellent** |
| `services/preview_service.go` | 76 | Preview links | **Excellent** |
| `validation/validation.go` | 111 | Input validation | **Excellent** |
| `util/files.go` | 104 | File operations | **Excellent** |
| `util/scenario.go` | 106 | Scenario resolution | **Excellent** |
| `util/logging.go` | 29 | Structured logging | **Excellent** |

### UI Layer (React)

| File | Lines | Responsibility | Status |
|------|-------|----------------|--------|
| `pages/FactoryHome.tsx` | 368 | Main factory page | **Good** |
| `components/` | 20 files | UI components | **Excellent** |
| `hooks/useScenarioLifecycle.ts` | - | Lifecycle state | **Excellent** |
| `lib/api.ts` | - | API client | **Excellent** |

---

## 3. Current Directory Structure

```
landing-manager/
├── api/
│   ├── main.go                   # 248 lines - Server wiring only
│   ├── handlers/
│   │   ├── handlers.go           # Handler infrastructure
│   │   ├── health.go             # Health check
│   │   ├── templates.go          # Template list/show
│   │   ├── generate.go           # Scenario generation
│   │   ├── customize.go          # Agent customization
│   │   ├── personas.go           # Persona list/show
│   │   ├── analytics.go          # Analytics endpoints
│   │   ├── lifecycle.go          # Start/stop/restart/status/logs/promote/delete
│   │   ├── middleware.go         # Request logging, size limits
│   │   └── template_only.go      # Placeholder for template features
│   ├── services/
│   │   ├── template_registry.go  # Template loading and caching
│   │   ├── scenario_generator.go # Generation and scaffolding
│   │   ├── persona_service.go    # Persona management
│   │   ├── preview_service.go    # Preview link generation
│   │   └── analytics_service.go  # Event tracking
│   ├── validation/
│   │   └── validation.go         # Centralized validation patterns
│   ├── util/
│   │   ├── files.go              # File copy/directory operations
│   │   ├── logging.go            # Structured logging
│   │   └── scenario.go           # Scenario path resolution
│   ├── templates/                # Template JSON definitions
│   ├── personas/                 # Agent persona catalog
│   └── analytics/                # Runtime analytics data
├── ui/
│   ├── src/
│   │   ├── pages/
│   │   │   └── FactoryHome.tsx   # Main factory page
│   │   ├── components/           # 20 well-organized components
│   │   ├── hooks/                # Custom React hooks
│   │   └── lib/
│   │       └── api.ts            # Clean API client
│   └── ...
├── generated/                    # Staging area for generated scenarios
├── docs/
├── test/
└── ...
```

---

## 4. How the Architecture "Screams" Its Purpose

Opening the `api/` folder, a new developer immediately sees:

```
handlers/     ← "This is where HTTP requests are handled"
services/     ← "This is where business logic lives"
validation/   ← "This is where input validation happens"
util/         ← "These are shared utilities"
templates/    ← "These are the landing page templates"
personas/     ← "These are the agent customization profiles"
```

Each file name clearly indicates its domain:
- `template_registry.go` - Template management
- `scenario_generator.go` - Scenario creation
- `persona_service.go` - Agent personas
- `lifecycle.go` - Scenario lifecycle operations

---

## 5. Previous Audit Improvements - Verified

| Previous Issue | Target | Actual | Status |
|----------------|--------|--------|--------|
| `main.go` god file | ~150 lines | 248 lines | **Fixed** - clean wiring only |
| `template_service.go` mixed concerns | Split into 4 services | Split into 5 services | **Fixed** |
| Validation duplication | 1 location | 1 location (`validation/`) | **Fixed** |
| Handlers in main | Extract to `handlers/` | 10 handler files | **Fixed** |
| File utilities embedded | Extract to `util/` | 3 utility files | **Fixed** |
| Personas misplaced | Separate service | `persona_service.go` | **Fixed** |

---

## 6. Remaining Minor Observations

### 6.1 `services/scenario_generator.go` (588 lines)

The largest service file. Could potentially be split into:
- `generation.go` - Core generation orchestration
- `scaffolding.go` - Template copying and structure creation

**Assessment**: Low priority. The file is cohesive - all functions relate to generation. Splitting might introduce unnecessary coordination complexity.

### 6.2 `handlers/lifecycle.go` (431 lines)

Contains 7 similar handler methods (start/stop/restart/status/logs/promote/delete). There's some repetition in:
- Validation patterns
- Error response formatting
- Logging structure

**Assessment**: Low priority. The repetition is explicit and domain-appropriate. Each handler has slightly different error handling needs. Abstracting further could reduce clarity.

### 6.3 UI: `FactoryHome.tsx` (368 lines)

Has been well-factored:
- Uses custom hooks (`useScenarioLifecycle`)
- Components extracted (20+ dedicated components)
- Clean state management

**Assessment**: Acceptable. Further splitting would likely be premature.

---

## 7. Recommendations

### No Critical Changes Needed

The architecture is now well-aligned. The previous refactoring addressed all major issues.

### Optional Future Improvements (Low Priority)

1. **Add tests to sub-packages**: Currently test files are only in the root package. Consider adding:
   - `handlers/handlers_test.go`
   - `services/scenario_generator_test.go`
   - `validation/validation_test.go`

2. **Document service contracts**: Consider adding interface definitions for services to clarify boundaries.

3. **Consider generic lifecycle handler**: If more lifecycle operations are added, abstract the common pattern.

---

## 8. Key Metrics Summary

| Metric | Before Refactoring | After Refactoring | Status |
|--------|-------------------|-------------------|--------|
| `main.go` lines | 805 | 248 | **Reduced 69%** |
| God files | 2 | 0 | **Eliminated** |
| Files mixing domains | 2 | 0 | **Eliminated** |
| Validation locations | 2 | 1 | **Centralized** |
| Clear package boundaries | Poor | Excellent | **Improved** |
| Time to find functionality | High | Low | **Improved** |

---

## 9. Conclusion

The landing-manager scenario architecture is **well-aligned** with its domain model. The codebase clearly expresses its purpose as a Landing Page Factory:

- Clear separation between handlers (HTTP), services (business logic), and utilities
- Domain concepts (templates, generation, personas, lifecycle) have dedicated homes
- Validation is centralized
- UI follows good React patterns with component extraction and custom hooks

**Recommendation**: No refactoring needed at this time. Focus development efforts on feature work rather than structural changes.
