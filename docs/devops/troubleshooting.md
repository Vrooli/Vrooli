# Troubleshooting Guide

This comprehensive troubleshooting guide provides solutions for common issues encountered when working with Vrooli's sophisticated development, build, and deployment infrastructure.

> **Prerequisites**: See [Prerequisites Guide](./getting-started/prerequisites.md) for required tools installation.
> **SSH Setup**: For comprehensive SSH key configuration, see [SSH Setup Guide](./getting-started/ssh-setup.md).
> **Environment Variables**: For complete variable reference, see [Environment Variables Guide](./getting-started/environment-variables.md).
> **Development Environment**: For detailed setup instructions, see [Development Environment](./development-environment.md).
> **Server Deployment**: For deployment configuration, see [Server Deployment](./server-deployment.md).

## Table of Contents

- [Development Environment Issues](#development-environment-issues)
- [Scripting Infrastructure Issues](#scripting-infrastructure-issues)
- [Target-Specific Issues](#target-specific-issues)
- [Package Management Issues](#package-management-issues)
- [Environment & Secrets Issues](#environment--secrets-issues)
- [SSH Deployment Issues](#ssh-deployment-issues)
- [Vault Integration Issues](#vault-integration-issues)
- [Testing Framework Issues](#testing-framework-issues)
- [CI/CD Pipeline Issues](#cicd-pipeline-issues)
- [Build System Issues](#build-system-issues)
- [Database Issues](#database-issues)
- [Performance Issues](#performance-issues)
- [Container Issues](#container-issues)
- [Kubernetes Issues](#kubernetes-issues)

## Development Environment Issues

### Setup Script Fails

**Symptoms:**
- `vrooli develop --target local-services` fails
- Missing dependencies or permission errors
- Script exits with error codes

**Solutions:**

1. **Check prerequisites:**
   ```bash
   # Verify Git is installed
   git --version
   
   # Check internet connectivity
   ping -c 3 google.com
   
   # Verify sufficient disk space
   df -h
   ```

2. **Run setup manually:**
   ```bash
   # Run setup first, then develop
   ./scripts/manage.sh setup --target local-services
   vrooli develop --target local-services
   ```

3. **Check permissions:**
   ```bash
   # Ensure management script is executable
   chmod +x scripts/manage.sh
   chmod +x scripts/helpers/**/*.sh
   ```

4. **Enable debug mode:**
   ```bash
   # Run with verbose output
   DEBUG=1 vrooli develop --target local-services
   ```

### Environment File Not Created

**Symptoms:**
- `.env-dev` file missing after setup
- Environment variables not loading
- Application fails to start with configuration errors

**Solutions:**

1. **Manually create environment file:**
   ```bash
   # Copy example and customize
   cp .env-example .env-dev
   
   # Or regenerate via setup
   rm -f .env-dev
   ./scripts/manage.sh setup --target local-services
   ```

2. **Check file permissions:**
   ```bash
   # Ensure readable by application
   ls -la .env-dev
   chmod 644 .env-dev
   ```

3. **Validate environment file:**
   ```bash
   # Check for syntax errors
   source .env-dev && echo "Environment file is valid"
   
   # Check for required variables
   grep -E "(DB_|REDIS_|JWT_)" .env-dev
   ```

### Port Conflicts

**Symptoms:**
- "Port already in use" errors
- Services fail to start
- Cannot access application on expected ports

**Solutions:**

1. **Check port usage:**
   ```bash
   # Check common ports
   sudo lsof -i :3000    # UI port
   sudo lsof -i :5329    # API port
   sudo lsof -i :5432    # Database port
   sudo lsof -i :6379    # Redis port
   ```

2. **Kill conflicting processes:**
   ```bash
   # Kill specific port usage
   sudo kill -9 $(lsof -ti:3000)
   
   # Or kill Node.js processes
   pkill -f "node.*server"
   pkill -f "node.*ui"
   ```

3. **Use alternative ports:**
   ```bash
   # Edit .env-dev
   echo "PORT_UI=3001" >> .env-dev
   echo "PORT_SERVER=5330" >> .env-dev
   echo "PORT_DB=5433" >> .env-dev
   ```

## Scripting Infrastructure Issues

### Script Permission Denied

**Symptoms:**
- "Permission denied" when running scripts
- Scripts fail to execute

**Solutions:**

1. **Fix script permissions:**
   ```bash
   # Make all scripts executable
   find scripts/ -name "*.sh" -exec chmod +x {} \;
   
   # Or specific script
   chmod +x scripts/manage.sh
   ```

2. **Check shell compatibility:**
   ```bash
   # Ensure using bash
   echo $SHELL
   
   # Run with explicit bash
   vrooli develop --target local-services
   ```

### Helper Scripts Not Found

**Symptoms:**
- "No such file or directory" errors for helper scripts
- Functions not found errors

**Solutions:**

1. **Verify script structure:**
   ```bash
   # Check helper scripts exist
   ls -la scripts/helpers/utils/
   ls -la scripts/helpers/setup/
   ```

2. **Check sourcing paths:**
   ```bash
   # Debug script loading
   bash -x scripts/manage.sh setup 2>&1 | grep "source\|load"
   ```

3. **Reinstall if corrupted:**
   ```bash
   # Re-clone if scripts are corrupted
   git checkout scripts/
   # Or re-clone repository
   ```

### Argument Parsing Issues

**Symptoms:**
- "Unknown option" errors
- Scripts don't recognize valid arguments
- Help text not displaying correctly

**Solutions:**

1. **Check argument syntax:**
   ```bash
   # Use proper argument format
   vrooli develop --target local-services --yes
   
   # Not: vrooli develop -target local-services
   ```

2. **View available options:**
   ```bash
   # Show help
   vrooli develop --help
   ./scripts/manage.sh build --help
   ```

3. **Use full argument names:**
   ```bash
   # Use full names instead of abbreviations
   --environment development    # Not -e dev
   --target local-services     # Not -t local
   ```

## Target-Specific Issues

### Local Services Target Issues

**Symptoms:**
- PostgreSQL fails to start
- Redis connection errors
- Node.js services crash

**Solutions:**

1. **Check PostgreSQL:**
   ```bash
   # Check if PostgreSQL is running
   pg_isready -h localhost -p 5432
   
   # Start PostgreSQL (Ubuntu/Debian)
   sudo systemctl start postgresql
   
   # Start PostgreSQL (macOS with Homebrew)
   brew services start postgresql
   ```

2. **Check Redis:**
   ```bash
   # Check if Redis is running
   redis-cli ping
   
   # Start Redis (Ubuntu/Debian)
   sudo systemctl start redis-server
   
   # Start Redis (macOS with Homebrew)
   brew services start redis
   ```

3. **Check Node.js services:**
   ```bash
   # Check Node.js version
   node --version
   
   # Check for memory issues
   node --max-old-space-size=4096 packages/server/dist/index.js
   ```

### Docker Daemon Target Issues

**Symptoms:**
- Docker containers fail to start
- Container networking issues
- Volume mounting problems

**Solutions:**

1. **Check Docker daemon:**
   ```bash
   # Verify Docker is running
   docker info
   
   # Start Docker (Linux)
   sudo systemctl start docker
   
   # Restart Docker if needed
   sudo systemctl restart docker
   ```

2. **Check container status:**
   ```bash
   # View running containers
   docker ps
   
   # View all containers (including stopped)
   docker ps -a
   
   # Check container logs
   docker logs <container-name>
   ```

3. **Fix networking issues:**
   ```bash
   # Check Docker networks
   docker network ls
   
   # Recreate default network if needed
   docker network prune
   ```

### Kubernetes Cluster Target Issues

**Symptoms:**
- Minikube fails to start
- Pods stuck in pending state
- Helm deployment failures

**Solutions:**

1. **Check Minikube:**
   ```bash
   # Check Minikube status
   minikube status
   
   # Start Minikube if stopped
   minikube start
   
   # Check resources
   minikube ssh -- df -h
   minikube ssh -- free -m
   ```

2. **Check Kubernetes resources:**
   ```bash
   # Check pod status
   kubectl get pods
   
   # Check events for issues
   kubectl get events --sort-by=.metadata.creationTimestamp
   
   # Check resource usage
   kubectl top nodes
   kubectl top pods
   ```

## Package Management Issues

### PNPM Installation Issues

**Symptoms:**
- `pnpm: command not found`
- PNPM version conflicts
- Global vs local PNPM issues

**Solutions:**

1. **Install PNPM:**
   ```bash
   # Install via corepack (recommended)
   corepack enable
   corepack prepare pnpm@latest --activate
   
   # Or install globally
   npm install -g pnpm
   
   # Or via shell script
   curl -fsSL https://get.pnpm.io/install.sh | sh -
   ```

2. **Check PNPM version:**
   ```bash
   # Verify version (should be 8.x or later)
   pnpm --version
   
   # Update if needed
   corepack prepare pnpm@latest --activate
   ```

3. **Fix PATH issues:**
   ```bash
   # Add to PATH (if needed)
   echo 'export PATH="$HOME/.local/share/pnpm:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   ```

### Dependency Installation Issues

**Symptoms:**
- `pnpm install` fails
- Package resolution errors
- Workspace dependency issues

**Solutions:**

1. **Clear PNPM cache:**
   ```bash
   # Clear store and cache
   pnpm store prune
   rm -rf ~/.pnpm-store
   
   # Clear node_modules
   rm -rf node_modules packages/*/node_modules
   ```

2. **Fix lockfile issues:**
   ```bash
   # Delete and regenerate lockfile
   rm pnpm-lock.yaml
   pnpm install
   
   # Or install with frozen lockfile
   pnpm install --frozen-lockfile
   ```

3. **Fix workspace issues:**
   ```bash
   # Check workspace configuration
   cat pnpm-workspace.yaml
   
   # Install workspace dependencies
   pnpm install --filter shared
   pnpm install --filter server
   pnpm install --filter ui
   ```

### Build Dependency Issues

**Symptoms:**
- TypeScript compilation errors
- Missing build tools
- Native dependency compilation failures

**Solutions:**

1. **Install build tools:**
   ```bash
   # Ubuntu/Debian
   sudo apt-get install build-essential python3-dev
   
   # macOS
   xcode-select --install
   
   # Check installation
   gcc --version
   python3 --version
   ```

2. **Fix TypeScript issues:**
   ```bash
   # Clear TypeScript cache
   rm -rf packages/*/dist
   rm -rf packages/*/.tsbuildinfo
   
   # Rebuild all packages
   pnpm run build
   
   # Check TypeScript version
   pnpm list typescript
   ```

3. **Fix native dependencies:**
   ```bash
   # Rebuild native dependencies
   pnpm rebuild
   
   # Or install with specific node version
   pnpm config set target_platform linux
   pnpm install
   ```

## Environment & Secrets Issues

> **Complete Environment Guide**: For comprehensive environment variables documentation, see [Environment Variables Guide](./getting-started/environment-variables.md).

### Environment File Corruption

**Symptoms:**
- Invalid environment variable values
- Parsing errors when loading environment
- Application fails with configuration errors

**Solutions:**

1. **Validate environment file:**
   ```bash
   # Check for invalid characters
   cat -A .env-dev | grep -E '\^M|\r'
   
   # Remove carriage returns (Windows line endings)
   sed -i 's/\r$//' .env-dev
   
   # Check for missing quotes
   grep -E "^[A-Z_]+=.*[[:space:]]" .env-dev
   ```

2. **Regenerate environment file:**
   ```bash
   # Backup current file
   cp .env-dev .env-dev.backup
   
   # Regenerate
   rm .env-dev
   ./scripts/manage.sh setup --target local-services
   
   # Restore custom values if needed
   diff .env-dev.backup .env-dev
   ```

3. **Fix specific variable issues:**
   ```bash
   # Check JWT keys format
   echo "JWT_PRIV length: ${#JWT_PRIV}"
   echo "$JWT_PRIV" | head -1  # Should start with -----BEGIN
   
   # Check database URL construction
   echo "DB_URL: $DB_URL"
   ```

### Missing Required Variables

**Symptoms:**
- Application complains about missing configuration
- Features don't work (AI, payments, email)
- Database connection failures

**Solutions:**

1. **Check required variables:**
   ```bash
   # Core required variables
   grep -E "(ENVIRONMENT|NODE_ENV|DB_|REDIS_|JWT_)" .env-dev
   
   # Optional service variables
   grep -E "(OPENAI_|STRIPE_|EMAIL_)" .env-dev
   ```

2. **Add missing variables:**
   ```bash
   # Add missing core variables
   echo "ENVIRONMENT=development" >> .env-dev
   echo "NODE_ENV=development" >> .env-dev
   echo "SECRETS_SOURCE=file" >> .env-dev
   
   # Add service-specific variables
   echo "OPENAI_API_KEY=your-key-here" >> .env-dev
   echo "STRIPE_PUBLIC_KEY=pk_test_your-key" >> .env-dev
   ```

3. **Copy from template:**
   ```bash
   # Use environment template
   cp .env-example .env-dev
   # Edit with your specific values
   ```

## SSH Deployment Issues

> **Complete SSH Guide**: For comprehensive SSH key setup, testing, and troubleshooting, see [SSH Setup Guide](./getting-started/ssh-setup.md).

### SSH Connection Failures

**Symptoms:**
- "Permission denied (publickey)" errors
- "Connection refused" errors
- SSH timeouts during deployment

**Quick Troubleshooting:**

1. **Test SSH connection:**
   ```bash
   # Test basic connection
   ssh -vvv user@server-ip
   
   # Test with specific key
   ssh -i ~/.ssh/vrooli_staging_deploy user@server-ip
   ```

2. **Check key permissions:**
   ```bash
   # Fix key permissions
   chmod 600 ~/.ssh/vrooli_*_deploy
   chmod 644 ~/.ssh/vrooli_*_deploy.pub
   ```

3. **Verify key on server:**
   ```bash
   # Check authorized_keys on server
   ssh user@server "cat ~/.ssh/authorized_keys"
   ```

> **Note**: For detailed SSH troubleshooting including key format issues, server configuration, and advanced debugging, see [SSH Setup Guide](./getting-started/ssh-setup.md#troubleshooting-ssh-issues).

### CI/CD SSH Issues

**Symptoms:**
- GitHub Actions deployment fails with SSH errors
- Key format issues in CI/CD
- Host key verification failures

**Solutions:**

1. **Check GitHub Secrets format:**
   ```bash
   # Ensure private key is in OpenSSH format
   head -1 ~/.ssh/vrooli_staging_deploy
   # Should show: -----BEGIN OPENSSH PRIVATE KEY-----
   
   # Convert if needed
   ssh-keygen -p -m OpenSSH -f ~/.ssh/vrooli_staging_deploy
   ```

2. **Test CI/CD SSH setup locally:**
   ```bash
   # Simulate CI/CD SSH setup
   mkdir -p ~/.ssh
   echo -e "Host *\n \tStrictHostKeyChecking no\n" > ~/.ssh/config
   chmod 600 ~/.ssh/config
   
   # Test connection
   ssh -o StrictHostKeyChecking=no user@server-ip
   ```

## Vault Integration Issues

### Vault Connection Failures

**Symptoms:**
- Cannot connect to Vault server
- Authentication failures
- Secret retrieval errors

**Solutions:**

1. **Check Vault server:**
   ```bash
   # Check if Vault is running
   vault status
   
   # Start local Vault for development (see current Vault setup documentation)
   # Configuration may have changed - check current Vault resource setup
   ```

2. **Check authentication:**
   ```bash
   # Verify Vault token
   vault auth -method=token
   
   # Or use AppRole
   vault write auth/approle/login \
     role_id="$VAULT_ROLE_ID" \
     secret_id="$VAULT_SECRET_ID"
   ```

3. **Check secret paths:**
   ```bash
   # List available secrets
   vault kv list secret/data/vrooli/
   
   # Read specific secret
   vault kv get secret/data/vrooli/config/shared-all
   ```

### Vault Secret Format Issues

**Symptoms:**
- Secrets not loading correctly
- JSON parsing errors
- Missing secret values

**Solutions:**

1. **Check secret format:**
   ```bash
   # Verify secret structure
   vault kv get -format=json secret/data/vrooli/config/shared-all
   
   # Check for proper JSON format
   vault kv get secret/data/vrooli/config/shared-all | jq .
   ```

2. **Fix secret format:**
   ```bash
   # Write secrets in correct format
   vault kv put secret/data/vrooli/config/shared-all \
     DB_PASSWORD="secure-password" \
     REDIS_PASSWORD="secure-redis-password"
   ```

## Testing Framework Issues

### BATS Test Failures

**Symptoms:**
- Shell script tests fail
- BATS not found
- Test syntax errors

**Solutions:**

1. **Install BATS:**
   ```bash
   # Install BATS if missing
   git clone https://github.com/bats-core/bats-core.git
   cd bats-core
   sudo ./install.sh /usr/local
   
   # Verify installation
   bats --version
   ```

2. **Run tests with debug:**
   ```bash
   # Run with verbose output
   bash __test/__runTests.sh --verbose
   
   # Run specific test file
   bats __test/utils/log.bats
   ```

3. **Fix test syntax:**
   ```bash
   # Check test file syntax
   bash -n __test/utils/log.bats
   
   # Common BATS syntax
   @test "test description" {
     run command_to_test
     [ "$status" -eq 0 ]
     [[ "$output" =~ "expected text" ]]
   }
   ```

### Unit Test Issues

**Symptoms:**
- Vitest tests fail
- Import/export errors
- Database connection issues in tests

**Solutions:**

1. **Check test environment:**
   ```bash
   # Set test environment
   export NODE_ENV=test
   
   # Use test database
   export DB_NAME=vrooli_test
   ```

2. **Fix import issues:**
   ```bash
   # Check TypeScript configuration
   cat packages/server/tsconfig.json
   
   # Ensure proper file extensions
   import { something } from './module.js';  // Note .js extension
   ```

3. **Database test setup:**
   ```bash
   # Create test database
   createdb vrooli_test
   
   # Run migrations for test DB
   cd packages/server
   NODE_ENV=test pnpm run migrate
   ```

## CI/CD Pipeline Issues

### GitHub Actions Failures

**Symptoms:**
- Workflow fails to start
- Build steps fail
- Deployment steps fail

**Solutions:**

1. **Check workflow syntax:**
   ```bash
   # Validate YAML syntax locally
   yamllint .github/workflows/dev.yml
   yamllint .github/workflows/master.yml
   ```

2. **Check GitHub Secrets:**
   ```bash
   # Verify all required secrets are set:
   # - ENV_FILE_CONTENT
   # - JWT_PRIV_PEM
   # - JWT_PUB_PEM
   # - VPS_SSH_PRIVATE_KEY
   # - VPS_DEPLOY_USER
   # - VPS_DEPLOY_HOST
   ```

3. **Debug workflow locally:**
   ```bash
   # Use act to run GitHub Actions locally
   act -j build
   
   # Or test build script locally
   ./scripts/manage.sh build --environment development --ci-cd yes
   ```

### Build Failures in CI/CD

**Symptoms:**
- Build script fails in CI/CD but works locally
- Permission errors in CI/CD
- Missing dependencies in CI/CD

**Solutions:**

1. **Check CI/CD environment:**
   ```bash
   # Add debug output to workflow
   - name: Debug Environment
     run: |
       echo "Node version: $(node --version)"
       echo "PNPM version: $(pnpm --version)"
       echo "Docker version: $(docker --version)"
       echo "Current directory: $(pwd)"
       echo "Available disk space: $(df -h)"
   ```

2. **Fix permission issues:**
   ```bash
   # Ensure scripts are executable in CI/CD
   - name: Make scripts executable
     run: |
       find scripts/ -name "*.sh" -exec chmod +x {} \;
   ```

3. **Check dependency caching:**
   ```bash
   # Add proper caching to workflow
   - name: Cache PNPM store
     uses: actions/cache@v3
     with:
       path: ~/.pnpm-store
       key: ${{ runner.os }}-pnpm-${{ hashFiles('**/pnpm-lock.yaml') }}
   ```

### Deployment Failures

**Symptoms:**
- SSH deployment fails
- Services don't start after deployment
- Health checks fail

**Solutions:**

1. **Check SSH deployment:**
   ```bash
   # Test SSH connection from CI/CD
   ssh -o StrictHostKeyChecking=no user@server "echo 'SSH works'"
   
   # Check deployment script on server
   ssh user@server "cd ~/Vrooli && ./scripts/manage.sh deploy --help"
   ```

2. **Check service status:**
   ```bash
   # Check Docker containers on server
   ssh user@server "docker ps"
   
   # Check container logs
   ssh user@server "docker logs server"
   ssh user@server "docker logs ui"
   ```

3. **Check health endpoints:**
   ```bash
   # Test health endpoints
   curl -f http://server-ip:5329/healthcheck
   curl -f http://server-ip:3000/
   ```

## Build System Issues

### Build Script Failures

**Symptoms:**
- `build.sh` script fails
- Artifact generation issues
- Bundle creation problems

**Solutions:**

1. **Check build prerequisites:**
   ```bash
   # Verify all tools are installed
   docker --version
   pnpm --version
   node --version
   
   # Check disk space
   df -h
   ```

2. **Run build with debug:**
   ```bash
   # Enable debug mode
   DEBUG=1 ./scripts/manage.sh build --environment development
   
   # Check specific build steps
   ./scripts/manage.sh build --environment development --artifacts docker --verbose
   ```

3. **Check build artifacts:**
   ```bash
   # Verify artifacts were created
   ls -la dist/
   ls -la docker-images/
   
   # Check bundle contents
   unzip -l vrooli-bundle-*.zip
   ```

### Docker Build Issues

**Symptoms:**
- Docker image build failures
- Layer caching issues
- Image size problems

**Solutions:**

1. **Check Dockerfile syntax:**
   ```bash
   # Validate Dockerfile
   docker build --no-cache -t test-image .
   
   # Check for syntax errors
   hadolint Dockerfile
   ```

2. **Fix layer caching:**
   ```bash
   # Clear Docker cache
   docker builder prune
   
   # Build without cache
   docker build --no-cache -t vrooli/server:dev .
   ```

3. **Optimize image size:**
   ```bash
   # Check image layers
   docker history vrooli/server:dev
   
   # Use multi-stage builds
   # Minimize installed packages
   # Clean up package caches
   ```

## Database Issues

### PostgreSQL Connection Issues

**Symptoms:**
- Cannot connect to database
- Authentication failures
- Connection timeouts

**Solutions:**

1. **Check PostgreSQL status:**
   ```bash
   # Check if PostgreSQL is running
   sudo systemctl status postgresql
   
   # Check PostgreSQL logs
   sudo tail -f /var/log/postgresql/postgresql-*.log
   ```

2. **Check connection parameters:**
   ```bash
   # Test connection with psql
   psql -h localhost -U vrooli_dev -d postgres
   
   # Check environment variables
   echo "DB_HOST: $DB_HOST"
   echo "DB_USER: $DB_USER"
   echo "DB_NAME: $DB_NAME"
   ```

3. **Fix authentication:**
   ```bash
   # Check pg_hba.conf
   sudo cat /etc/postgresql/*/main/pg_hba.conf
   
   # Ensure user exists
   sudo -u postgres createuser vrooli_dev
   sudo -u postgres createdb -O vrooli_dev postgres
   ```

### Database Migration Issues

**Symptoms:**
- Migration scripts fail
- Schema version conflicts
- Data corruption during migration

**Solutions:**

1. **Check migration status:**
   ```bash
   # Check current schema version
   cd packages/server
   pnpm run migrate:status
   
   # Check migration files
   ls -la src/db/migrations/
   ```

2. **Fix migration conflicts:**
   ```bash
   # Rollback problematic migration
   pnpm run migrate:down
   
   # Re-run migrations
   pnpm run migrate:up
   ```

3. **Backup before migration:**
   ```bash
   # Create backup
   pg_dump -h localhost -U vrooli_dev postgres > backup.sql
   
   # Restore if needed
   psql -h localhost -U vrooli_dev -d postgres < backup.sql
   ```

### Redis Connection Issues

**Symptoms:**
- Cannot connect to Redis
- Redis authentication failures
- Memory issues

**Solutions:**

1. **Check Redis status:**
   ```bash
   # Check if Redis is running
   sudo systemctl status redis-server
   
   # Test connection
   redis-cli ping
   ```

2. **Check Redis configuration:**
   ```bash
   # Check Redis config
   redis-cli CONFIG GET "*"
   
   # Check memory usage
   redis-cli INFO memory
   ```

3. **Fix authentication:**
   ```bash
   # Check Redis password
   redis-cli -a "$REDIS_PASSWORD" ping
   
   # Update Redis config if needed
   sudo nano /etc/redis/redis.conf
   ```

## Performance Issues

### Slow Application Performance

**Symptoms:**
- Long response times
- High CPU/memory usage
- Database query slowness

**Solutions:**

1. **Monitor resource usage:**
   ```bash
   # Check system resources
   htop
   iotop
   
   # Check Docker container resources
   docker stats
   ```

2. **Optimize database:**
   ```bash
   # Check slow queries
   psql -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
   
   # Analyze query performance
   psql -c "EXPLAIN ANALYZE SELECT * FROM your_table;"
   ```

3. **Optimize application:**
   ```bash
   # Profile Node.js application
   node --prof packages/server/dist/index.js
   node --prof-process isolate-*.log > profile.txt
   
   # Check memory leaks
   node --inspect packages/server/dist/index.js
   ```

### Build Performance Issues

**Symptoms:**
- Slow build times
- High memory usage during builds
- Build timeouts

**Solutions:**

1. **Optimize build process:**
   ```bash
   # Use parallel builds
   export MAKEFLAGS="-j$(nproc)"
   
   # Increase Node.js memory
   export NODE_OPTIONS="--max-old-space-size=4096"
   ```

2. **Optimize Docker builds:**
   ```bash
   # Use BuildKit
   export DOCKER_BUILDKIT=1
   
   # Use multi-stage builds
   # Optimize layer caching
   # Use .dockerignore
   ```

3. **Optimize TypeScript compilation:**
   ```bash
   # Use incremental compilation
   # Enable project references
   # Use SWC for faster compilation
   ```

## Container Issues

### Docker Container Failures

**Symptoms:**
- Containers exit immediately
- Container networking issues
- Volume mounting problems

**Solutions:**

1. **Check container logs:**
   ```bash
   # View container logs
   docker logs <container-name>
   
   # Follow logs in real-time
   docker logs -f <container-name>
   ```

2. **Debug container startup:**
   ```bash
   # Run container interactively
   docker run -it vrooli/server:dev /bin/bash
   
   # Check container environment
   docker exec -it <container-name> env
   ```

3. **Fix networking issues:**
   ```bash
   # Check Docker networks
   docker network ls
   docker network inspect bridge
   
   # Test container connectivity
   docker exec -it <container-name> ping <other-container>
   ```

### Docker Compose Issues

**Symptoms:**
- Services fail to start
- Service dependency issues
- Configuration problems

**Solutions:**

1. **Check compose file:**
   ```bash
   # Validate compose file
   docker-compose config
   
   # Check service dependencies
   docker-compose ps
   ```

2. **Debug service startup:**
   ```bash
   # Start services individually
   docker-compose up postgres
   docker-compose up redis
   docker-compose up server
   ```

3. **Check service logs:**
   ```bash
   # View all service logs
   docker-compose logs
   
   # View specific service logs
   docker-compose logs server
   ```

## Kubernetes Issues

### Pod Startup Issues

**Symptoms:**
- Pods stuck in Pending state
- Pods crash on startup
- Image pull failures

**Solutions:**

1. **Check pod status:**
   ```bash
   # Get pod details
   kubectl describe pod <pod-name>
   
   # Check pod logs
   kubectl logs <pod-name>
   ```

2. **Check resource constraints:**
   ```bash
   # Check node resources
   kubectl top nodes
   
   # Check resource requests/limits
   kubectl describe pod <pod-name> | grep -A 5 "Requests\|Limits"
   ```

3. **Fix image issues:**
   ```bash
   # Check image availability
   docker pull <image-name>
   
   # Check image pull policy
   kubectl get pod <pod-name> -o yaml | grep imagePullPolicy
   ```

### Helm Deployment Issues

**Symptoms:**
- Helm install/upgrade failures
- Template rendering errors
- Value override issues

**Solutions:**

1. **Debug Helm templates:**
   ```bash
   # Render templates without installing
   helm template vrooli ./k8s/chart --values ./k8s/chart/values-dev.yaml
   
   # Check template syntax
   helm lint ./k8s/chart
   ```

2. **Check Helm values:**
   ```bash
   # Show computed values
   helm get values vrooli
   
   # Check value overrides
   helm upgrade vrooli ./k8s/chart --values ./k8s/chart/values-dev.yaml --dry-run
   ```

3. **Fix deployment issues:**
   ```bash
   # Check Helm release status
   helm status vrooli
   
   # Rollback if needed
   helm rollback vrooli <revision>
   ```

This comprehensive troubleshooting guide covers the most common issues encountered when working with Vrooli's sophisticated infrastructure. For additional help, consult the specific documentation sections referenced throughout this guide.

## Related Documentation

- **Prerequisites**: [Prerequisites Guide](./getting-started/prerequisites.md)
- **SSH Setup**: [SSH Setup Guide](./getting-started/ssh-setup.md)
- **Environment Variables**: [Environment Variables Reference](./getting-started/environment-variables.md)
- **Development Environment**: [Development Environment Setup](./development-environment.md)
- **Server Deployment**: [Server Deployment Guide](./server-deployment.md)
- **CI/CD Pipeline**: [CI/CD Pipeline Guide](./ci-cd.md)
- **Build System**: [Build System Guide](./build-system.md)
- **Kubernetes Deployment**: [Kubernetes Guide](./kubernetes.md) 