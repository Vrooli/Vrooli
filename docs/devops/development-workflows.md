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
# 1. Initial project setup (choose your target)
./scripts/main/setup.sh --target docker

# Available targets:
# --target docker      # Docker Compose (recommended for development)
# --target k8s          # Kubernetes cluster
# --target native-linux # Native Linux/Ubuntu
# --target native-mac   # Native macOS
# --target native-win   # Native Windows

# 2. Start development environment
./scripts/main/develop.sh --target docker --detached no

# 3. Open in browser: http://localhost:3000
```

### Environment Verification

```bash
# Verify your setup is working
curl http://localhost:5555/health
curl http://localhost:3000

# Check logs
./scripts/main/develop.sh --target docker --detached yes
docker logs vrooli-server-1
docker logs vrooli-ui-1
```

> 📖 **Setup Guide**: See [Prerequisites](../setup/prerequisites.md) for detailed setup requirements

---

## 📦 Package-Level Development

### Working with Packages

```bash
# Navigate to package for focused development
cd packages/server

# Start development server with hot reload
pnpm dev

# Run tests with watch mode
pnpm test-watch

# Type checking
pnpm type-check

# Build for production
pnpm build
```

### Package-Specific Workflows

| Package | Primary Commands | Purpose |
|---------|------------------|---------|
| **packages/server** | `pnpm dev`, `pnpm test-watch` | API development and testing |
| **packages/ui** | `pnpm dev`, `pnpm storybook` | UI development and component testing |
| **packages/shared** | `pnpm test-watch`, `pnpm build` | Type and validation development |
| **packages/jobs** | `pnpm dev`, `pnpm test` | Background job development |

### Integrated Development

```bash
# Start all services with auto-restart
./scripts/main/develop.sh --target docker --detached no

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
./scripts/main/setup.sh --target docker
./scripts/main/develop.sh --target docker --detached no

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
# Setup cluster and deploy
./scripts/main/setup.sh --target k8s
./scripts/main/develop.sh --target k8s --detached yes

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
./scripts/main/setup.sh --target native-linux

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
./scripts/main/develop.sh --target docker --detached no

# Code changes automatically trigger:
# - TypeScript compilation
# - Test execution (changed files)
# - Hot module replacement (UI)
# - Server restart (API changes)

# Verify changes
curl http://localhost:5555/api/status
# Open http://localhost:3000 in browser

# Run tests for your changes
cd packages/server && pnpm test -- --grep "your feature"
cd packages/ui && pnpm test -- your-component
```

### 2. Database Development

```bash
# Access database directly
docker exec -it vrooli-postgres-1 psql -U postgres -d vrooli

# Run migrations
cd packages/server && pnpm prisma migrate dev

# Reset database (development only)
cd packages/server && pnpm prisma migrate reset

# Seed data
cd packages/server && pnpm prisma db seed
```

### 3. Testing Workflows

```bash
# Run all tests
pnpm test

# Package-specific testing
cd packages/server && pnpm test-coverage
cd packages/ui && pnpm test-watch
cd packages/shared && pnpm test

# Integration testing with Docker
./scripts/main/develop.sh --target docker --detached yes
# Run your integration tests
./scripts/main/develop.sh --target docker --detached no  # Switch back to logs

# E2E testing
cd packages/ui && pnpm test:e2e
```

### 4. Performance Development

```bash
# Start with performance monitoring
./scripts/main/develop.sh --target docker --detached no

# Monitor resources
docker stats

# Profile specific packages
cd packages/server && node --inspect=0.0.0.0:9229 dist/index.js
cd packages/ui && pnpm build && pnpm analyze

# Memory and CPU analysis
cd packages/server && pnpm test -- --coverage
```

### 5. Debugging Workflows

```bash
# Debug server with logs
docker logs -f vrooli-server-1

# Debug UI issues
./scripts/main/develop.sh --target docker --detached no
# Check browser console and network tab

# Debug database queries
cd packages/server && DEBUG=prisma:query pnpm dev

# Debug Redis connections
docker exec -it vrooli-redis-1 redis-cli monitor
```

---

## 🌐 Environment Management

### Environment Configuration

```bash
# Development (automatic)
ENVIRONMENT=development ./scripts/main/develop.sh --target docker

# Production testing
ENVIRONMENT=production ./scripts/main/develop.sh --target docker

# Custom environment
cp .env-example .env
# Edit .env with your configuration
./scripts/main/develop.sh --target docker
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
./scripts/main/develop.sh --target docker --environment development

# Staging environment  
./scripts/main/develop.sh --target k8s --environment staging

# Production testing
./scripts/main/develop.sh --target k8s --environment production
```

---

## 🧪 Testing Integration Workflows

### Test-Driven Development

```bash
# Start with test watching
cd packages/server && pnpm test-watch &
cd packages/ui && pnpm test-watch &
./scripts/main/develop.sh --target docker --detached no

# Write failing tests first
# Implement features to make tests pass
# Verify in browser/API
```

### CI/CD Simulation

```bash
# Simulate CI pipeline locally
./scripts/main/setup.sh --target docker --clean yes
./scripts/main/build.sh --target docker
./scripts/main/deploy.sh --target docker --environment staging

# Run full test suite
pnpm test:ci
```

### Performance Testing

```bash
# Start performance monitoring
./scripts/main/develop.sh --target docker --detached yes

# Load testing
cd packages/server && npm run test:load
cd packages/ui && npm run test:lighthouse

# Memory leak testing
cd packages/server && npm run test:memory
```

---

## 🐛 Troubleshooting & Debugging

### Common Issues & Solutions

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **Port Conflicts** | "Port already in use" | `./scripts/main/develop.sh --target docker --clean yes` |
| **Database Connection** | "Connection refused" | Check PostgreSQL container: `docker ps` |
| **Module Not Found** | Import errors | `pnpm install && pnpm build` |
| **Hot Reload Broken** | Changes not reflecting | Restart: `Ctrl+C` then restart develop script |
| **Container Issues** | Services not starting | `docker system prune && ./scripts/main/setup.sh --clean yes` |

### Debug Mode

```bash
# Enable verbose logging
DEBUG=vrooli:* ./scripts/main/develop.sh --target docker --detached no

# Database query debugging
DEBUG=prisma:query ./scripts/main/develop.sh --target docker --detached no

# Network debugging
DEBUG=socket.io:* ./scripts/main/develop.sh --target docker --detached no
```

### Health Checks

```bash
# Check all services
curl http://localhost:5329/healthcheck
curl http://localhost:3000

# Database connectivity
cd packages/server && pnpm prisma db pull

# Dependencies status
docker ps
docker logs vrooli-server-1
docker logs vrooli-ui-1
```

### Performance Debugging

```bash
# Monitor resource usage
docker stats

# Profile Node.js performance
cd packages/server && node --inspect=0.0.0.0:9229 dist/index.js
# Open chrome://inspect in Chrome

# Analyze bundle size
cd packages/ui && pnpm analyze

# Database performance
cd packages/server && DEBUG=prisma:query pnpm dev
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

# UI performance
cd packages/ui && pnpm build && pnpm preview
# Check Lighthouse scores in browser dev tools

# Database performance
cd packages/server && pnpm prisma studio
# Monitor query performance
```

---

## 🚀 Deployment Workflows

### Development to Staging

```bash
# Test locally first
./scripts/main/develop.sh --target docker --environment production

# Deploy to staging
./scripts/main/deploy.sh --target k8s --environment staging

# Verify deployment
kubectl get pods -n vrooli-staging
kubectl logs -f deployment/vrooli-server -n vrooli-staging
```

### Production Deployment

```bash
# Build production images
./scripts/main/build.sh --target docker --environment production

# Deploy with zero downtime
./scripts/main/deploy.sh --target k8s --environment production

# Monitor deployment
kubectl rollout status deployment/vrooli-server -n vrooli-production
```

---

## 📚 Script Reference

### Main Scripts

| Script | Purpose | Common Usage |
|--------|---------|--------------|
| **setup.sh** | Initial environment setup | `./scripts/main/setup.sh --target docker` |
| **develop.sh** | Start development environment | `./scripts/main/develop.sh --target docker --detached no` |
| **build.sh** | Build production artifacts | `./scripts/main/build.sh --target docker` |
| **deploy.sh** | Deploy to environments | `./scripts/main/deploy.sh --target k8s --environment staging` |

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
- **[Server Development](../../packages/server/README.md)** - API development patterns
- **[UI Development](../../packages/ui/README.md)** - Frontend development workflows
- **[Testing Strategy](../testing/test-strategy.md)** - Testing approaches and tools

### Deployment & Operations
- **[Kubernetes](../devops/kubernetes.md)** - K8s deployment and management
- **[CI/CD](../devops/ci-cd.md)** - Continuous integration workflows
- **[Monitoring](../devops/logging.md)** - Logging and observability

---

*Master these workflows to develop efficiently with Vrooli's sophisticated architecture. The script-based approach ensures consistency across all environments while providing the flexibility needed for complex development scenarios.*