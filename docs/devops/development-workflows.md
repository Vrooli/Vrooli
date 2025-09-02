# 🛠️ Development Workflows Guide

> **Complete development workflow integration** - Master Vrooli's script-based automation for efficient daily development across all deployment targets.

> 📖 **Quick Links**: [Getting Started](#getting-started) | [Daily Workflows](#daily-development-workflows) | [Deployment Targets](#deployment-targets) | [Troubleshooting](#troubleshooting--debugging)

---

## 🎯 Purpose & Workflow Philosophy

Vrooli's development workflows are designed around **script-based automation** that provides:

- **🎯 Target-Agnostic Development** - Same commands work across Docker, Kubernetes, and native environments
- **🔄 Consistent Environments** - Automated setup and configuration management
- **⚡ Fast Iteration Cycles** - Hot reloading, auto-restart, and intelligent caching
- **🛡️ Production Parity** - Development environments mirror production infrastructure
- **🐛 Comprehensive Debugging** - Built-in tools for troubleshooting across all tiers

The workflow scripts are located in `/scripts/` and provide a **unified interface** for all development operations.

---

## 🚀 Getting Started

### Initial Setup

```bash
# 1. Initial project setup
vrooli setup

# 2. Start development environment
vrooli develop

# 3. Open in browser: http://localhost:3000
```

### Environment Verification

```bash
# Verify your setup is working
curl http://localhost:5555/health
curl http://localhost:3000

# Check logs
vrooli status
```

> 📖 **Setup Guide**: See [Prerequisites](../setup/prerequisites.md) for detailed setup requirements

---

## 📦 Resource & Scenario Development

### Working with Resources

```bash
# Start all enabled resources
vrooli resource start-all

# Check resource status
vrooli resource status

# Test specific resource
resource-ollama test
resource-postgres test

# View resource logs
resource-<name> logs
```

### Scenario-Specific Workflows

| Component | Primary Commands | Purpose |
|---------|------------------|---------|
| **Resources** | `vrooli resource start`, `resource-<name> test` | Service management and testing |
| **Scenarios** | `vrooli scenario run`, `vrooli scenario test` | Business application development |
| **System** | `vrooli develop`, `vrooli status` | Overall system management |

### Integrated Development

```bash
# Start all services with auto-restart
vrooli develop

# This starts:
# - PostgreSQL (port 5432)
# - Redis (port 6379) 
# - Server API (port 5555)
# - UI Development Server (port 3000)
# - Background Jobs (internal)

# Development features:
# ✅ Hot module replacement (UI)
# ✅ Auto-restart on changes (Server)
# ✅ Live reloading (Both)
# ✅ Type checking (Real-time)
```

---

## 🎯 Deployment Targets

### Docker Compose (Recommended)

**Best for**: Daily development, testing, quick setup

```bash
# Setup and start
vrooli setup
vrooli develop

# Environment:
# - PostgreSQL + Redis in containers
# - Server + UI in development mode
# - All dependencies automatically managed
# - Volume mounts for live code changes

# Advantages:
# ✅ Fastest setup
# ✅ Production parity
# ✅ Isolated dependencies
# ✅ Easy cleanup
```

### Kubernetes Cluster

**Best for**: Production testing, advanced development, CI/CD

```bash
# Setup and deploy
vrooli setup
vrooli develop

# Environment:
# - Full Kubernetes deployment
# - Helm charts with development values
# - Ingress, services, and pods
# - Vault integration for secrets

# Advantages:
# ✅ Production-identical
# ✅ Scalability testing
# ✅ Advanced networking
# ✅ Secret management
```

### Native Development

**Best for**: Deep debugging, performance profiling, direct database access

```bash
# Setup native dependencies
vrooli setup

# Manual service management
# Start PostgreSQL: sudo systemctl start postgresql
# Start Redis: sudo systemctl start redis
# Start development servers manually

# Advantages:
# ✅ Direct debugging
# ✅ Best performance
# ✅ Full system access
# ✅ Custom configurations
```

---

## 🔄 Daily Development Workflows

### 1. Feature Development

```bash
# Start your day
vrooli develop

# Code changes automatically trigger:
# - Scenario configuration validation
# - Test execution (changed files)
# - Hot module replacement (UI)
# - Server restart (API changes)

# Verify changes
curl http://localhost:5555/api/status
# Open http://localhost:3000 in browser

# Run tests for your changes
vrooli scenario test <scenario-name>
vrooli test resources --grep "your feature"
```

### 2. Database Development

```bash
# Access database directly
docker exec -it vrooli-postgres-1 psql -U postgres -d vrooli

# PostgreSQL resource handles migrations
resource-postgres migrate

# Reset database (development only)
resource-postgres reset

# Seed data through scenario initialization
vrooli scenario init <scenario-name>
```

### 3. Testing Workflows

```bash
# Run all tests
pnpm test

# Scenario-specific testing
vrooli scenario test <scenario-name> --coverage
vrooli test scenarios --watch
vrooli test resources

# Integration testing with Docker
vrooli develop
# Run your integration tests
vrooli test

# E2E testing
vrooli test e2e --scenario <scenario-name>
```

### 4. Performance Development

```bash
# Start with performance monitoring
vrooli develop

# Monitor resources
docker stats

# Profile scenarios
vrooli scenario profile <scenario-name> --inspect

# Resource performance analysis
vrooli resource profile --all

# Memory and CPU analysis
vrooli test performance --scenario <scenario-name>
```

### 5. Debugging Workflows

```bash
# Debug server with logs
docker logs -f vrooli-server-1

# Debug UI issues
vrooli develop
# Check browser console and network tab

# Debug database queries
DEBUG=postgres:query vrooli develop

# Debug Redis connections
docker exec -it vrooli-redis-1 redis-cli monitor
```

---

## 🌐 Environment Management

### Environment Configuration

```bash
# Development (automatic)
vrooli develop

# Production testing (use appropriate config)
vrooli develop

# Custom environment
cp .env-example .env
# Edit .env with your configuration
vrooli develop
```

### Environment Variables by Service

| Service | Key Variables | Purpose |
|---------|---------------|---------|
| **Server** | `DATABASE_URL`, `REDIS_URL` | Database connections |
| **UI** | `VITE_SERVER_URL`, `VITE_NODE_ENV` | API communication |
| **Jobs** | `QUEUE_REDIS_URL`, `WORKER_CONCURRENCY` | Background processing |
| **Infrastructure** | `VAULT_ADDR`, `VAULT_TOKEN` | Secret management |

### Multi-Environment Development

```bash
# Development environment
vrooli develop

# For different environments, configure your .env files appropriately
# then run:
vrooli develop
```

---

## 🧪 Testing Integration Workflows

### Test-Driven Development

```bash
# Start with test watching
vrooli test --watch &
vrooli develop

# Write failing tests first
# Implement features to make tests pass
# Verify in browser/API
```

### CI/CD Simulation

```bash
# Simulate CI pipeline locally
vrooli clean
vrooli setup
vrooli build
vrooli deploy

# Run full test suite
pnpm test:ci
```

### Performance Testing

```bash
# Start performance monitoring
vrooli develop

# Load testing
vrooli test load --scenario <scenario-name>

# Memory leak testing
vrooli test memory --scenario <scenario-name>
```

---

## 🐛 Troubleshooting & Debugging

### Common Issues & Solutions

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **Port Conflicts** | "Port already in use" | `vrooli stop && vrooli clean && vrooli develop` |
| **Database Connection** | "Connection refused" | `vrooli resource status` |
| **Module Not Found** | Import errors | `pnpm install && pnpm build` |
| **Hot Reload Broken** | Changes not reflecting | `vrooli stop && vrooli develop` |
| **Container Issues** | Services not starting | `vrooli clean && vrooli setup` |

### Debug Mode

```bash
# Enable verbose logging
DEBUG=vrooli:* vrooli develop

# Database query debugging
DEBUG=prisma:query vrooli develop

# Network debugging
DEBUG=socket.io:* vrooli develop
```

### Health Checks

```bash
# Check all services
curl http://localhost:5329/healthcheck
curl http://localhost:3000

# Database connectivity
resource-postgres status
resource-postgres pull

# Dependencies status
docker ps
docker logs vrooli-server-1
docker logs vrooli-ui-1
```

### Performance Debugging

```bash
# Monitor resource usage
docker stats

# Profile scenario performance
vrooli scenario profile <scenario-name> --inspect
# Open chrome://inspect in Chrome

# Analyze resource usage
vrooli resource analyze

# Database performance
resource-postgres profile
```

---

## 📊 Monitoring & Observability

### Development Monitoring

```bash
# Application logs
docker logs -f vrooli-server-1
docker logs -f vrooli-ui-1

# Database monitoring
docker exec -it vrooli-postgres-1 psql -U postgres -c "SELECT * FROM pg_stat_activity;"

# Redis monitoring
docker exec -it vrooli-redis-1 redis-cli info stats
```

### Performance Metrics

```bash
# Server performance
curl http://localhost:5329/metrics

# Scenario UI performance
vrooli scenario analyze <scenario-name> --ui
# Check Lighthouse scores in browser dev tools

# Database performance monitoring
resource-postgres studio
# Monitor query performance
```

---

## 🚀 Deployment Workflows

### Development to Staging

```bash
# Test locally first
vrooli develop

# Deploy to staging
vrooli deploy

# Verify deployment
kubectl get pods -n vrooli-staging
kubectl logs -f deployment/vrooli-server -n vrooli-staging
```

### Production Deployment

```bash
# Build production images
vrooli build

# Deploy with zero downtime
vrooli deploy

# Monitor deployment
kubectl rollout status deployment/vrooli-server -n vrooli-production
```

---

## 📚 CLI Reference

### Main Commands

| Command | Purpose | Common Usage |
|--------|---------|--------------|
| **vrooli setup** | Initial environment setup | `vrooli setup` |
| **vrooli develop** | Start development environment | `vrooli develop` |
| **vrooli build** | Build scenario deployments | `vrooli build` |
| **vrooli deploy** | Deploy to environments | `vrooli deploy` |

### Helper Scripts

| Category | Scripts | Purpose |
|----------|---------|---------|
| **Setup** | `/scripts/helpers/setup/` | Dependency installation, configuration |
| **Development** | `/scripts/helpers/develop/` | Development server management |
| **Build** | `/scripts/helpers/build/` | Production build processes |
| **Deploy** | `/scripts/helpers/deploy/` | Deployment automation |
| **Utils** | `/scripts/helpers/utils/` | Shared utilities and functions |

### Script Arguments

```bash
# Common arguments across all scripts
--target        # docker|k8s|native-linux|native-mac|native-win
--environment   # development|staging|production
--detached      # yes|no (for develop.sh)
--clean         # yes|no (remove previous artifacts)
--sudo-mode     # yes|no (use sudo where needed)
--yes           # Auto-accept prompts
--help          # Show usage information
```

---

## 🎯 Best Practices

### Daily Development

1. **Start Fresh**: Use `--clean yes` periodically to avoid accumulated issues
2. **Monitor Logs**: Keep development logs visible for immediate feedback
3. **Test Early**: Run tests alongside development for faster iteration
4. **Use Hot Reload**: Leverage auto-restart and hot module replacement
5. **Check Health**: Verify service health before starting feature work

### Code Quality

1. **Run Lints**: Use `pnpm lint` before committing
2. **Type Check**: Ensure `pnpm type-check` passes
3. **Test Coverage**: Maintain >80% test coverage
4. **Performance**: Monitor bundle size and query performance
5. **Documentation**: Update docs alongside code changes

### Environment Management

1. **Use Docker**: Prefer Docker for development unless debugging requires native
2. **Match Production**: Use production-like configurations for testing
3. **Clean Regularly**: Reset environments to catch configuration drift
4. **Monitor Resources**: Watch CPU, memory, and disk usage
5. **Version Dependencies**: Keep package versions consistent across environments

---

## 🔗 Related Documentation

### Setup & Configuration
- **[Prerequisites](../setup/prerequisites.md)** - Required tools and dependencies
- **[Environment Variables](../setup/environment-variables.md)** - Configuration management
- **[Docker Setup](../setup/working_with_docker.md)** - Docker-specific configuration

### Development Guides
- **[Scenario Development](../scenarios/getting-started.md)** - Building business applications
- **[Resource Integration](../resources/README.md)** - Working with resources
- **[Testing Strategy](../testing/test-strategy.md)** - Testing approaches and tools

### Deployment & Operations
- **[Kubernetes](../devops/kubernetes.md)** - K8s deployment and management
- **[CI/CD](../devops/ci-cd.md)** - Continuous integration workflows
- **[Monitoring](../devops/logging.md)** - Logging and observability

---

*Master these workflows to develop efficiently with Vrooli's sophisticated architecture. The script-based approach ensures consistency across all environments while providing the flexibility needed for complex development scenarios.*