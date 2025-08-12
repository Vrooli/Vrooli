# Metareasoning API - Architecture Guide

## 📁 Clean Architecture Structure

The API has been refactored from a single 1467-line file into a modular architecture with clear separation of concerns:

```
api/
├── main.go         (111 lines)  - Application entry point and route setup
├── models.go       (218 lines)  - Data structures and type definitions
├── database.go     (490 lines)  - Database operations and queries
├── services.go     (406 lines)  - Business logic and external integrations
├── handlers.go     (465 lines)  - HTTP request handlers
├── middleware.go   (134 lines)  - Cross-cutting concerns (auth, logging, CORS)
├── utils.go        (82 lines)   - Helper functions and utilities
└── README.md       (this file)   - Architecture documentation
```

Total: **1906 lines** (expanded from 1467 due to better organization and documentation)

## 🏗️ Architecture Layers

### 1. **Main Layer** (`main.go`)
- Application initialization
- Dependency injection
- Route configuration
- Server lifecycle management
- Graceful shutdown handling

### 2. **Models Layer** (`models.go`)
- All data structures
- Request/response types
- External API models (Ollama, n8n, Windmill)
- No business logic - pure data

### 3. **Database Layer** (`database.go`)
- PostgreSQL connection management
- CRUD operations for workflows
- Execution history management
- Metrics and statistics queries
- Transaction handling
- Connection pooling

### 4. **Service Layer** (`services.go`)
- Business logic orchestration
- External service integration (n8n, Windmill, Ollama)
- Workflow execution engine
- Import/export functionality
- Generation and cloning logic

### 5. **Handler Layer** (`handlers.go`)
- HTTP request/response handling
- Input validation
- Error responses
- Status codes
- Delegates to services

### 6. **Middleware Layer** (`middleware.go`)
- Authentication/authorization
- Request logging
- CORS headers
- Panic recovery
- Response wrapping

### 7. **Utils Layer** (`utils.go`)
- Environment configuration
- Helper functions
- Common utilities
- Service health checks

## 🔄 Request Flow

```
HTTP Request
    ↓
Middleware Stack
    ├── Recovery (catch panics)
    ├── Logging (track requests)
    ├── CORS (handle cross-origin)
    └── Auth (validate tokens)
    ↓
Router (mux)
    ↓
Handler Function
    ├── Parse request
    ├── Validate input
    └── Call service
    ↓
Service Layer
    ├── Business logic
    ├── External calls
    └── Database ops
    ↓
Database Layer
    ├── Execute queries
    └── Return results
    ↓
Response
```

## 🎯 Design Principles

1. **Single Responsibility**: Each file has one clear purpose
2. **Dependency Injection**: Services injected into handlers
3. **Clean Interfaces**: Clear contracts between layers
4. **Error Handling**: Consistent error propagation
5. **Testability**: Each layer can be tested independently

## 🧪 Testing Strategy

With this modular structure, you can now:

1. **Unit test** each layer independently:
   ```go
   // database_test.go - Test database operations
   // services_test.go - Test business logic with mocked DB
   // handlers_test.go - Test HTTP handling with mocked services
   ```

2. **Integration test** specific flows:
   ```go
   // Test complete workflow execution
   // Test import/export pipeline
   // Test generation flow
   ```

3. **Mock dependencies** easily:
   ```go
   type MockWorkflowService struct {
       // Mock implementation
   }
   ```

## 🚀 Benefits of Refactoring

1. **Maintainability**: Easy to find and modify specific functionality
2. **Scalability**: New features can be added to appropriate layers
3. **Testability**: Each component can be tested in isolation
4. **Readability**: Clear structure makes onboarding easier
5. **Reusability**: Services can be reused across different handlers
6. **Performance**: Optimizations can target specific layers

## 📝 Adding New Features

To add a new feature:

1. **Define models** in `models.go`
2. **Add database operations** in `database.go`
3. **Implement business logic** in `services.go`
4. **Create handler** in `handlers.go`
5. **Add route** in `main.go`
6. **Add middleware** if needed in `middleware.go`

## 🔧 Configuration

The API uses environment variables (see `utils.go`):

```bash
PORT=8093                    # API port
DATABASE_URL=postgres://...  # PostgreSQL connection
N8N_BASE_URL=http://...      # n8n instance
WINDMILL_BASE_URL=http://... # Windmill instance
WINDMILL_WORKSPACE=demo      # Windmill workspace
```

## 📚 Dependencies

- `github.com/gorilla/mux` - HTTP routing
- `github.com/google/uuid` - UUID generation
- `github.com/lib/pq` - PostgreSQL driver

## 🏆 Best Practices

1. **Always use transactions** for multi-step database operations
2. **Return early** on errors to avoid nested code
3. **Use meaningful variable names** for clarity
4. **Add comments** for complex business logic
5. **Keep functions small** and focused
6. **Use interfaces** for mockable dependencies
7. **Handle all errors** explicitly

---

*This refactored architecture makes the Metareasoning API a true 5-star reference implementation for database-driven Go APIs in the Vrooli ecosystem.*