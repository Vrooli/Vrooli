# DevOps Glossary

This glossary provides standardized definitions for all terms used in Vrooli's DevOps documentation to ensure consistency across all guides.

## Environment Types

### development
**Definition**: The local development environment where code is written and initially tested.
**Usage**: Always use `development` (not `dev`)
**Examples**:
```bash
ENVIRONMENT=development
NODE_ENV=development
```

### staging
**Definition**: A production-like environment for testing before production deployment.
**Usage**: Always use `staging` (not `stage` or `test`)
**Examples**:
```bash
ENVIRONMENT=staging
# Branch: dev (maps to staging environment)
```

### production
**Definition**: The live environment serving real users.
**Usage**: Always use `production` (not `prod`)
**Examples**:
```bash
ENVIRONMENT=production
NODE_ENV=production
```

## Execution Models

### Direct Scenario Execution (Primary)
**Definition**: Scenarios run directly from their source folders without build steps.
**Usage**: Standard way to run any business application
**Commands**: 
```bash
vrooli scenario run <name>              # Run any scenario
cd scenarios/<name> && ../../scripts/manage.sh develop
```

### Resource Management
**Definition**: Independent services that provide capabilities to scenarios.
**Usage**: Start/stop resources that scenarios need
**Commands**:
```bash
vrooli resource start-all               # Start all enabled resources
resource-<name> start                   # Start specific resource
```

### Production Deployment
**Definition**: Deploy scenario suites to Kubernetes for customers.
**Usage**: Package multiple scenarios for production use
**Command**: `./scripts/deployment/package-scenario-deployment.sh`

## Script Invocation Standards

### Standard Format
**Primary Commands**: Use `vrooli` CLI for all common operations
```bash
vrooli setup                    # Initial setup
vrooli develop                  # Start development
vrooli scenario run <name>      # Run scenarios directly
vrooli resource start-all       # Start resources
```

**Direct Resource Commands**: Use resource-specific CLIs
```bash
resource-postgres start         # Start PostgreSQL
resource-ollama logs           # View Ollama logs
```

## ⚠️ Deprecated Terms (DO NOT USE)

**These terms reflect obsolete Vrooli architecture:**
- ❌ **"Converting scenarios"** - Scenarios run directly, no conversion
- ❌ **"Build process"** - No build steps, direct execution only
- ❌ **"Standalone apps"** - Scenarios are not converted to apps
- ❌ **"Packages"** - No monorepo structure, use resources/scenarios
- ❌ **"scripts/main/"** - Directory no longer exists, use `vrooli` CLI
- ❌ **"Compilation/Transpilation"** - Direct execution from source
- ❌ **"Generated code"** - No code generation, scenarios run as-is

### Argument Format
**Correct**: `--environment development` (long form)
**Incorrect**: `-e dev` (short form or abbreviated values)

## Container Terminology (Resources Only)

**Note**: Scenarios run directly from source and don't use containers. This section applies only to containerized resources.

### Resource Containers
- **Local Resources**: May run in Docker containers (PostgreSQL, Redis, etc.)
- **Production Resources**: Deployed as Kubernetes pods
- **Scenarios**: Run directly from `scenarios/` folder - no containers needed!

### Image References (If Using Containerized Resources)
```yaml
# For resources that use containers:
image: postgres:15-alpine       # PostgreSQL resource
image: redis:7-alpine           # Redis resource
image: ollama/ollama:latest     # Ollama AI resource
```

## File and Directory Standards

### Environment Files
- `.env-dev` - Development environment configuration
- `.env-staging` - Staging environment configuration (optional)
- `.env-prod` - Production environment configuration

### Script Paths
**Primary Interface**: Use `vrooli` CLI for all operations
```bash
vrooli setup                    # Initial setup
vrooli develop                  # Development environment
vrooli scenario run <name>      # Direct scenario execution
vrooli resource start-all       # Resource management
```

**Direct Execution**: Scenarios run from their folders
```bash
cd scenarios/my-app
../../scripts/manage.sh develop  # Run scenario directly
```

## Service and Component Names

### Core Services
- **server** - API server service
- **ui** - Frontend/UI service  
- **jobs** - Background job processing service
- **postgres** - PostgreSQL database service
- **redis** - Redis cache service

### Infrastructure Components
- **nginx-proxy** - Reverse proxy for Docker deployments
- **letsencrypt** - SSL certificate management
- **vault** - HashiCorp Vault for secrets management
- **vso** - Vault Secrets Operator

## Configuration Management

### Secrets Management
- **file** - File-based secrets (`.env` files)
- **vault** - HashiCorp Vault integration

### Location Types
- **local** - Running on developer machine
- **remote** - Running on server/cloud instance

## Build and Deployment Terms

### Artifact Types
- **docker** - Docker images and related assets
- **k8s** - Kubernetes manifests and Helm charts
- **zip** - Compressed deployment bundles
- **cli** - Command-line interface executables

### Bundle Types
- **all** - All available bundle types
- **zip** - ZIP compressed bundles
- **cli** - CLI executable bundles

### Binary Platforms
- **all** - All supported platforms
- **windows** - Windows executable
- **mac** - macOS executable  
- **linux** - Linux executable
- **android** - Android package
- **ios** - iOS package

## Network and Security

### Domains and URLs
- **apex domain** - Root domain (example.com)
- **subdomain** - Subdomain (api.example.com)
- **origin server** - Backend server (do-origin.example.com)

### SSL/TLS
- **Let's Encrypt** - Free SSL certificate service
- **ACME** - Automatic Certificate Management Environment
- **SSL termination** - Ending SSL at load balancer/proxy

### Authentication Methods
- **token** - Bearer token authentication
- **approle** - HashiCorp Vault AppRole authentication
- **kubernetes** - Kubernetes service account authentication

## Database and Cache

### Database Terms
- **primary** - Main database instance
- **replica** - Read-only database copy
- **migration** - Database schema change
- **seed data** - Initial/test data

### Cache Terms
- **master** - Primary Redis instance
- **replica** - Redis read replica
- **sentinel** - Redis monitoring service
- **failover** - Automatic switching to backup

## Monitoring and Observability

### Metrics and Monitoring
- **prometheus** - Metrics collection system
- **grafana** - Metrics visualization dashboard
- **alerting** - Automated notification system
- **health checks** - Service status verification

### Logging
- **structured logging** - JSON-formatted logs
- **log aggregation** - Centralized log collection
- **correlation ID** - Request tracking identifier

## Version Control and Branches

### Branch Names
- **main** - Primary development branch
- **dev** - Development branch (deploys to staging)
- **master** - Production branch (deploys to production)

### Version Format
**Standard**: Semantic versioning (MAJOR.MINOR.PATCH)
**Examples**: `1.0.0`, `2.1.3`, `1.0.0-beta.1`

## Testing Terminology

### Test Types
- **unit tests** - Individual component tests
- **integration tests** - Component interaction tests
- **e2e tests** - End-to-end user workflow tests
- **shell tests** - BATS shell script tests

### Test Frameworks
- **BATS** - Bash Automated Testing System
- **Vitest** - JavaScript test framework

## Infrastructure as Code

### Tools and Platforms
- **Terraform** - Infrastructure provisioning tool
- **Helm** - Kubernetes package manager
- **Docker Compose** - Container orchestration tool
- **Kubernetes** - Container orchestration platform

### Cloud Providers
- **AWS** - Amazon Web Services
- **GCP** - Google Cloud Platform
- **DigitalOcean** - Cloud hosting provider
- **Linode** - Cloud hosting provider

## Operational Terms

### Deployment Strategies
- **rolling update** - Gradual replacement deployment
- **blue-green** - Parallel environment deployment
- **canary** - Partial traffic deployment
- **hotfix** - Emergency production fix

### Backup and Recovery
- **backup** - Data copy for restoration
- **disaster recovery** - Full system restoration plan
- **RTO** - Recovery Time Objective
- **RPO** - Recovery Point Objective

## Common Acronyms and Abbreviations

### Technical Acronyms
- **API** - Application Programming Interface
- **CLI** - Command Line Interface
- **CI/CD** - Continuous Integration/Continuous Deployment
- **DNS** - Domain Name System
- **HTTP/HTTPS** - HyperText Transfer Protocol (Secure)
- **JWT** - JSON Web Token
- **JSON** - JavaScript Object Notation
- **REST** - Representational State Transfer
- **SSH** - Secure Shell
- **SSL/TLS** - Secure Sockets Layer/Transport Layer Security
- **URL** - Uniform Resource Locator
- **UUID** - Universally Unique Identifier
- **YAML** - YAML Ain't Markup Language

### Infrastructure Acronyms
- **VPS** - Virtual Private Server
- **LB** - Load Balancer
- **CDN** - Content Delivery Network
- **IAM** - Identity and Access Management
- **RBAC** - Role-Based Access Control
- **PVC** - Persistent Volume Claim (Kubernetes)
- **CR** - Custom Resource (Kubernetes)
- **CRD** - Custom Resource Definition (Kubernetes)

### Database Acronyms
- **ACID** - Atomicity, Consistency, Isolation, Durability
- **ORM** - Object-Relational Mapping
- **CRUD** - Create, Read, Update, Delete
- **SQL** - Structured Query Language
- **NoSQL** - Not Only SQL

## File Extensions and Formats

### Configuration Files
- `.md` - Markdown documentation
- `.yaml` / `.yml` - YAML configuration
- `.json` - JSON configuration
- `.env` - Environment variables file
- `.sh` - Shell script
- `.bats` - BATS test file

### Container and Kubernetes
- `Dockerfile` - Docker image build instructions
- `.dockerignore` - Docker build exclusions
- `.helmignore` - Helm package exclusions
- `Chart.yaml` - Helm chart metadata
- `values.yaml` - Helm chart configuration

## Command Patterns

### Script Execution
```bash
# Use vrooli CLI for common operations
vrooli develop --target local-services

# Use manage.sh for scenario operations
./scripts/manage.sh develop  # Start development environment
vrooli scenario run <name>   # Run scenarios directly

# Use consistent quoting
export ENVIRONMENT="development"
```

### Environment Variables
```bash
# Correct quoting and expansion
DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"

# Consistent naming convention
ENVIRONMENT=development  # Not ENVIRONMENT=dev
NODE_ENV=development     # Not NODE_ENV=dev
```

## Documentation Standards

### Cross-References
**Format**: `[Link Text](./relative/path.md)`
**Section Links**: `[Section](./file.md#section-name)`

### Code Examples
**Language Tags**: Always specify language
```bash
# Shell commands
```

```yaml
# YAML configuration
```

```typescript
// TypeScript code
```

### Prerequisites References
**Standard**: Always link to consolidated prerequisites
```markdown
> **Prerequisites**: See [Prerequisites Guide](../getting-started/prerequisites.md)
```

This glossary serves as the authoritative reference for all terminology used in Vrooli's DevOps documentation. When updating documentation, always refer to this glossary to ensure consistency. 