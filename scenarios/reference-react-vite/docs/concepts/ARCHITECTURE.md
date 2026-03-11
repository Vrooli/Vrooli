# Architecture

This document describes the domain mental model, key entities, main flows, and natural boundaries of the reference-react-vite scenario.

## Overview

The reference-react-vite scenario implements a simple task/project management domain. The business logic is deliberately minimal - the value is in demonstrating architectural patterns, not features.

## Domain Mental Model

```
┌─────────────────────────────────────────────────────────────────┐
│                        TASK MANAGEMENT                          │
│                                                                 │
│   ┌──────────┐     contains      ┌──────────┐                  │
│   │ Project  │ ───────────────── │   Task   │                  │
│   └──────────┘       1:N         └──────────┘                  │
│        │                              │                         │
│        │                              │ has                     │
│        │                              │ 1:N                     │
│        │                              ▼                         │
│        │                         ┌──────────┐                  │
│        │                         │   Note   │                  │
│        │                         └──────────┘                  │
│        │                                                        │
│   Status: active,                Status: pending,              │
│   archived, completed            in_progress,                  │
│                                  completed, blocked            │
└─────────────────────────────────────────────────────────────────┘
```

## Key Entities

### Project
[CODE: api/domain/projects/project.go]

A container for organizing tasks. Projects have:
- **Name**: Required, 1-200 characters
- **Description**: Optional text
- **Color**: Optional hex color code (#RRGGBB)
- **Status**: active (default), archived, or completed

### Task
[CODE: api/domain/tasks/task.go]

A work item within a project. Tasks have:
- **Title**: Required, 1-500 characters
- **Description**: Optional detailed text
- **Status**: pending, in_progress, completed, or blocked
- **Priority**: 1 (lowest) to 5 (highest)
- **Due Date**: Optional deadline
- **Project**: Optional foreign key to parent project

### Note
[CODE: api/domain/notes/note.go]

An annotation attached to a task. Notes have:
- **Content**: Required, 1-10000 characters
- **Task**: Required foreign key to parent task
- **Author**: Optional identifier

## Main Flows

### Create Task Flow
```
Client → POST /api/v1/tasks
       → TaskHandler.Create()
       → Validate request body
       → tasks.New() (domain factory with validation)
       → TaskRepository.Create() (persist to PostgreSQL)
       → Return created task with ID
```

### List Tasks with Pagination Flow
```
Client → GET /api/v1/tasks?page=1&limit=20&status=pending
       → TaskHandler.List()
       → Parse pagination/filter params
       → TaskRepository.List() (query with filters)
       → Return tasks + pagination metadata
```

### Add Note to Task Flow
```
Client → POST /api/v1/tasks/{id}/notes
       → NoteHandler.Create()
       → TaskRepository.GetByID() (verify task exists)
       → notes.New() (domain factory)
       → NoteRepository.Create()
       → Return created note
```

## Natural Boundaries

### Presentation Layer
**Location**: [CODE: api/handlers/]

Handles HTTP request/response translation. Does NOT contain business logic.

Responsibilities:
- Parse request parameters and body
- Validate input format (not business rules)
- Call domain/repository layers
- Format response (success or error)
- Handle pagination metadata

### Domain Layer
**Location**: [CODE: api/domain/]

Contains business rules and entity definitions. Does NOT know about HTTP or databases.

Responsibilities:
- Define entity types with validation rules
- Provide factory functions that enforce invariants
- Implement business rule methods (e.g., status transitions)

### Data Access Layer
**Location**: [CODE: api/repository/]

Handles database operations. Does NOT contain business logic.

Responsibilities:
- Define repository interfaces
- Implement PostgreSQL queries
- Handle pagination and filtering
- Translate between domain types and database rows

### Storage Layer
**Location**: [CODE: initialization/postgres/schema.sql]

Defines physical data structure.

Responsibilities:
- Table definitions with constraints
- Indexes for query performance
- Triggers for auto-updating timestamps

## Cross-Cutting Concerns

| Concern | Location | Notes |
|---------|----------|-------|
| CORS | [CODE: api/main.go:82-108] | Configurable via CORS_ALLOWED_ORIGINS |
| Logging | [CODE: api/main.go:133-139] | Simple request logging middleware |
| Request ID | [CODE: api/main.go:142-149] | Generated per-request for tracing |
| Error Format | [CODE: api/handlers/errors.go] | Consistent API error shape |
| Health Check | [CODE: api/main.go:51-57] | Uses api-core/health |
| Graceful Shutdown | [CODE: api/main.go:170-175] | Uses api-core/server |

## Technology Stack

- **HTTP Router**: gorilla/mux
- **Database**: PostgreSQL via database/sql
- **Connection Management**: api-core/database (retry, pooling)
- **Health Checks**: api-core/health
- **Server Lifecycle**: api-core/server (graceful shutdown)

## Dependency Flow

```
main.go (composition root)
    │
    ├── database.Connect() → *sql.DB
    │
    ├── repository.NewRepositories(db)
    │       └── PostgresTaskRepository
    │       └── PostgresProjectRepository
    │       └── PostgresNoteRepository
    │
    ├── handlers.NewTaskHandler(repos.Tasks)
    ├── handlers.NewProjectHandler(repos.Projects)
    └── handlers.NewNoteHandler(repos.Notes, repos.Tasks)
```

Dependencies flow inward: handlers → repositories → domain types. The domain layer has no external dependencies.

## Last Updated

2026-03-11 - Initial architecture documentation created
