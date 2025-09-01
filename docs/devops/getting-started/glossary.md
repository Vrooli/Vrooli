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

## Deployment Targets

### local-services
**Definition**: Development target running services directly on the host machine.
**Usage**: Native Node.js, local PostgreSQL/Redis
**Command**: `vrooli develop --target local-services`

### docker-daemon
**Definition**: Development target using Docker containers for all services.
**Usage**: Docker Compose orchestration, containerized services
**Command**: `vrooli develop --target docker-daemon`

### k8s-cluster
**Definition**: Development target using Kubernetes (typically Minikube).
**Usage**: Helm charts, Kubernetes resources, container orchestration
**Command**: `vrooli develop --target k8s-cluster`

## Script Invocation Standards

### Standard Format
**Correct**: `./scripts/manage.sh <phase>` or `vrooli <command>`
**Incorrect**: Using deprecated `scripts/main/` paths

**Migration Reference:**
- `scripts/main/setup.sh` → `./scripts/manage.sh setup` or `vrooli setup`
- `scripts/main/develop.sh` → `vrooli develop`  
- `scripts/main/build.sh` → `./scripts/manage.sh build`
- `scripts/main/deploy.sh` → `./scripts/manage.sh deploy`

**Note**: The `scripts/main/` directory was removed in Phase 5 of the migration to unified script management. All functionality has been moved to the universal entry points listed above.

### Argument Format
**Correct**: `--environment development` (long form)
**Incorrect**: `-e dev` (short form or abbreviated values)

## Container and Image Terminology

### Image Tags
- **Development Images**: Tagged with `development`
- **Production Images**: Tagged with `production`
- **Never use**: `dev`, `prod`, `latest` in documentation examples

### Registry References
**Standard**: Use placeholder format with variables
```yaml
image: ${DOCKERHUB_USERNAME}/server:development
```

**Avoid**: Hardcoded usernames like `vrooli/server:prod`

## File and Directory Standards

### Environment Files
- `.env-dev` - Development environment configuration
- `.env-staging` - Staging environment configuration (optional)
- `.env-prod` - Production environment configuration

### Script Paths
**Standard**: Always reference from repository root
```bash
vrooli develop
./scripts/manage.sh setup
bash scripts/helpers/utils/log.sh
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

# Use manage.sh for lifecycle operations
./scripts/manage.sh build --environment development --artifacts docker

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