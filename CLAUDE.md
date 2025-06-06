# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Vrooli is a collaborative platform for creating, sharing, and running automated procedures. It features a three-tier AI architecture for autonomous operations, process management, and execution.

## Technology Stack

- **Frontend**: React + TypeScript, Material-UI, Vite, PWA-enabled
- **Backend**: Node.js + TypeScript, Express, PostgreSQL (with pgvector), Redis, Prisma ORM
- **Infrastructure**: Docker, Kubernetes with Helm, HashiCorp Vault
- **AI Integration**: Multi-provider support (OpenAI, Anthropic, Mistral, Google), MCP protocol
- **Package Manager**: pnpm with workspaces
- **Testing**: Mocha + Chai + Sinon, c8 for coverage
- **Scripts**: Bash-based automation in `/scripts/`

## Development Commands

### Setup and Environment
```bash
# Initial setup (installs dependencies, configures environment)
./scripts/main/setup.sh --target docker

# Start development environment with Docker (detached mode)
./scripts/main/develop.sh --target docker --detached yes

# Start development environment with Docker (interactive mode)
./scripts/main/develop.sh --target docker --detached no

# Start specific services only
./scripts/main/develop.sh --target docker --services "server ui redis postgresql"
```

### Building and Testing
```bash
# Build all packages
pnpm run build

# Run tests for a specific package
cd packages/[package-name] && pnpm test

# Run all tests with coverage
pnpm run test:coverage

# Lint all packages
pnpm run lint

# Type checking
pnpm run typecheck

# Run a single test file
cd packages/[package-name] && pnpm test -- path/to/test.test.ts
```

### Common Package Scripts
Each package typically has:
- `pnpm dev` - Start development server
- `pnpm build` - Build for production
- `pnpm test` - Run tests
- `pnpm lint` - Run linter
- `pnpm typecheck` - Run TypeScript type checking

## Architecture Overview

### Three-Tier AI Architecture
1. **Tier 1 - Coordination Intelligence**: Strategic planning, resource allocation, swarm management
2. **Tier 2 - Process Intelligence**: Task decomposition, routine navigation, execution monitoring
3. **Tier 3 - Execution Intelligence**: Direct task execution, tool integration, context management

### Key Patterns
- **Event-Driven Communication**: Redis-based event bus for inter-service communication
- **Resource Management**: Coordinated resource allocation across tiers
- **Error Handling**: Structured error propagation with circuit breakers
- **State Management**: Distributed state with Redis caching

### Project Structure
```
/packages/
  /ui/            # React frontend application
  /server/        # Express backend API
  /shared/        # Shared types and utilities
  /jobs/          # Background job processing
  /auth/          # Authentication service
  /agents/        # AI agent implementations
/scripts/         # Bash automation scripts
/docs/           # Comprehensive documentation
/k8s/            # Kubernetes/Helm configurations
```

## Key Development Guidelines

### Database Operations
- Use Prisma migrations for schema changes: `cd packages/server && pnpm prisma migrate dev`
- Access Prisma Studio: `cd packages/server && pnpm prisma studio`
- Generate Prisma client after schema changes: `cd packages/server && pnpm prisma generate`

### Environment Variables
- Development: `.env` files in package directories
- Production: Managed via HashiCorp Vault
- Never commit sensitive data; use Vault for secrets

### Testing Approach
- Write tests alongside code in `__tests__` directories
- Use descriptive test names following pattern: `should [expected behavior] when [condition]`
- Mock external dependencies using Sinon
- Aim for >80% code coverage

### Error Handling
- Use structured error types from `@vrooli/shared`
- Implement proper error boundaries in React components
- Log errors with appropriate severity levels
- Use circuit breakers for external service calls

### Import Requirements
- **IMPORTANT**: All TypeScript imports MUST include the `.js` extension (e.g., `import { foo } from "./bar.js"`)
- This is a hard requirement for the testing framework to work correctly
- Do NOT remove `.js` extensions from imports, even if TypeScript complains
- The build system is configured to handle `.js` extensions in TypeScript files

### Performance Considerations
- Implement pagination for list endpoints
- Use Redis caching for frequently accessed data
- Optimize database queries with proper indexes
- Use React.memo and useMemo for expensive computations

## Common Tasks

### Adding a New API Endpoint
1. Define types in `packages/shared/src/api/`
2. Add validation schema in `packages/shared/src/validation/`
3. Implement endpoint in `packages/server/src/endpoints/`
4. Add tests in corresponding `__tests__` directory
5. Update API documentation if needed

### Working with AI Services
1. AI providers configured in `packages/server/src/services/llm/`
2. Use the LLM service abstraction for provider-agnostic calls
3. Implement rate limiting and cost tracking
4. Handle provider-specific errors gracefully

### Debugging
- Server logs: Check Docker container logs or terminal output
- Frontend debugging: Use React Developer Tools
- Database queries: Enable Prisma query logging with `DEBUG=prisma:query`
- Network issues: Check Redis connection and event bus

## Script Utilities

The `/scripts/` directory contains comprehensive bash scripts:
- Use `--help` flag with any script for documentation
- Scripts support various targets (docker, k8s, local)
- Environment detection and validation built-in
- Automatic dependency installation when needed

## Security Notes
- All external URLs must be validated before use
- Implement proper authentication/authorization checks
- Sanitize user inputs, especially for database queries
- Use prepared statements/parameterized queries
- Follow OWASP guidelines for web security

## Performance Optimization
- Database queries should use appropriate indexes
- Implement request caching where appropriate
- Use connection pooling for database connections
- Optimize bundle sizes with code splitting
- Monitor memory usage in background jobs