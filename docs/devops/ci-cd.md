# CI/CD for Vrooli Scenarios

> **Scope:** Tier 1 only. These patterns assume scenarios run on a full Vrooli stack (local or dev server) and are accessed via app-monitor/Cloudflare tunnels. For desktop, mobile, SaaS, or enterprise deployment planning, pivot to the [Deployment Hub](../deployment/README.md) and coordinate through deployment-manager once available.

This guide covers CI/CD patterns for testing and deploying Vrooli scenarios using direct execution. Scenarios run directly from their source locations without conversion to standalone applications.

> **Prerequisites**: See [Prerequisites Guide](./getting-started/prerequisites.md) for required tools installation.

## Overview

Vrooli scenarios use direct execution for both development and production deployment:

### Direct Scenario CI/CD Characteristics

- 🎯 **Dual-Purpose**: Scenarios serve as both integration tests AND deployable applications
- 🔄 **Direct Execution**: No conversion step - scenarios run directly from `scenarios/` directory
- 📦 **Resource Orchestration**: Scenarios orchestrate local resources to create business capabilities
- 🚀 **Instant Deployment**: Changes take effect immediately without regeneration
- 🛡️ **Security First**: Process isolation and resource-based security model

## Common CI/CD Patterns

### Pattern 1: Scenario Integration Testing

Core pattern for validating scenario functionality and deployment readiness:

```yaml
# .github/workflows/test-scenarios.yml
name: Test Vrooli Scenarios

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test-scenarios:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Vrooli Environment
        run: |
          vrooli setup
          
      - name: Start Required Resources
        run: |
          vrooli resource start-all
          vrooli resource status
          
      - name: Test All Scenarios
        run: |
          for scenario in scenarios/*/; do
            name=$(basename "$scenario")
            echo "Testing scenario: $name"
            vrooli scenario test "$scenario" || exit 1
          done
          
      - name: Cleanup
        if: always()
        run: vrooli stop
```

### Pattern 2: Scenario Deployment Pipeline

For deploying validated scenarios to production environments:

```yaml
# .github/workflows/deploy-scenarios.yml
name: Deploy Validated Scenarios

on:
  push:
    tags: ['v*']

jobs:
  validate-scenarios:
    runs-on: ubuntu-latest
    outputs:
      scenarios: ${{ steps.list-scenarios.outputs.scenarios }}
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Environment
        run: vrooli setup
        
      - name: Start Resources
        run: vrooli resource start-all
        
      - name: List Deployable Scenarios
        id: list-scenarios
        run: |
          scenarios=$(find scenarios/ -name 'service.json' -exec dirname {} \; | xargs -I {} basename {} | jq -R . | jq -s . | tr -d '\n')
          echo "scenarios=$scenarios" >> $GITHUB_OUTPUT
          
      - name: Validate All Scenarios
        run: |
          for scenario in scenarios/*/; do
            name=$(basename "$scenario")
            echo "Validating $name for deployment readiness"
            vrooli scenario test "$scenario" || exit 1
          done

  deploy-to-production:
    needs: validate-scenarios
    runs-on: ubuntu-latest
    environment: production
    steps:
      - uses: actions/checkout@v4
      
      - name: Package Scenario Deployment (legacy)
        run: |
          echo "Legacy pipeline retained for reference. Consult docs/deployment/history/packaging-script.md"
            
      - name: Deploy to Production Cluster
        run: |
          kubectl apply -f ./dist/k8s/
        env:
          KUBECONFIG: ${{ secrets.KUBECONFIG_PROD }}
```

### Pattern 3: Resource Health Monitoring

For continuous monitoring of resource availability and health:

```yaml
# .github/workflows/resource-health.yml
name: Monitor Resource Health

on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours
  workflow_dispatch:

jobs:
  health-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Vrooli
        run: vrooli setup
        
      - name: Start Core Resources
        run: |
          vrooli resource start ollama postgres redis
          sleep 30  # Allow startup time
          
      - name: Validate Resource Health
        run: |
          vrooli resource status --check-all || {
            echo "Resource health check failed"
            vrooli resource logs --all
            exit 1
          }
          
      - name: Test Resource Integration
        run: |
          # Test AI resource
          curl -f http://localhost:11434/api/tags || exit 1
          
          # Test database resource
          resource-postgres status || exit 1
          
          # Test cache resource
          resource-redis ping || exit 1
```

### Pattern 4: Multi-Scenario Production Deployment

For deploying multiple scenarios as a cohesive business solution:

```yaml
# .github/workflows/production-deploy.yml
name: Deploy Scenario Suite to Production

on:
  push:
    tags: ['release-*']

jobs:
  validate-suite:
    runs-on: ubuntu-latest
    outputs:
      deployment-ready: ${{ steps.validate.outputs.ready }}
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Environment
        run: vrooli setup
        
      - name: Start Resources
        run: vrooli resource start-all
        
      - name: Validate Scenario Suite
        id: validate
        run: |
          # Test core business scenarios
          vrooli scenario test research-assistant || exit 1
          vrooli scenario test invoice-generator || exit 1
          vrooli scenario test customer-portal || exit 1
          
          echo "ready=true" >> $GITHUB_OUTPUT

  deploy-production:
    needs: validate-suite
    if: needs.validate-suite.outputs.deployment-ready == 'true'
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Deployment Blocked
        run: |
          echo "Scenario packaging is now governed by the tiered deployment roadmap."
          echo "See docs/deployment/README.md before attempting any production deploys."
```

## Scenario Configuration Integration

### Service Configuration

Scenarios define their testing and deployment requirements in `.vrooli/service.json`:

```json
{
  "metadata": {
    "name": "research-assistant",
    "displayName": "AI Research Assistant",
    "description": "Enterprise research automation platform"
  },
  "dependencies": {
    "resources": {
      "ollama": {"enabled": true, "required": true},
      "qdrant": {"enabled": true, "required": true},
      "postgres": {"enabled": true, "required": true},
    }
  },
  "spec": {
    "testing": {
      "timeout": 600,
      "requiresDisplay": false,
      "ciEnabled": true,
      "successCriteria": [
        "Resource health checks pass",
        "AI inference responds correctly",
        "Database integration works",
        "Workflow automation executes"
      ]
    },
    "deployment": {
      "productionReady": true,
      "resourceRequirements": {
        "cpu": "2 cores",
        "memory": "4GB",
        "storage": "20GB"
      }
    }
  }
}
```

### Environment Management

Scenarios use environment variables for different execution contexts:

```bash
# Development environment
SCENARIO_MODE=development
RESOURCE_PREFIX=http://localhost
DEBUG_SCENARIO=true
VROOLI_ENV=development

# Production environment
SCENARIO_MODE=production
RESOURCE_PREFIX=https://resources.vrooli.com
DEBUG_SCENARIO=false
VROOLI_ENV=production
KUBERNETES_NAMESPACE=vrooli-scenarios
```

## Quality Gates and Testing

### Scenario Validation Pipeline

```yaml
# Comprehensive scenario validation workflow
jobs:
  scenario-quality-gates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Vrooli Environment
        run: vrooli setup
        
      - name: Validate Scenario Structure
        run: |
          for scenario in scenarios/*/; do
            echo "Validating structure: $(basename "$scenario")"
            test-genie execute "$(basename "$scenario")" --phases structure --fail-fast || exit 1
          done
          
      - name: Start Resources for Testing
        run: vrooli resource start-all
        
      - name: Run Scenario Integration Tests
        run: |
          for scenario in scenarios/*/; do
            name=$(basename "$scenario")
            echo "Integration testing: $name"
            vrooli scenario test "$name" --ci || exit 1
          done
          
      - name: Validate Business Requirements
        run: |
          # Check that scenarios meet business criteria
          ./scripts/scenarios/tools/validate-business-model.sh --all

  deploy-validated-scenarios:
    needs: scenario-quality-gates
    if: success() && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - name: Deploy Scenarios
        run: |
          for scenario in scenarios/*/; do
            name=$(basename "$scenario")
            echo "Deploying scenario: $name"
            vrooli scenario run "$name" --environment production
          done
```

### Scenario Test Configuration

Scenarios define their testing requirements in `.vrooli/service.json`:

```json
{
  "spec": {
    "testing": {
      "timeout": 600,
      "requiresDisplay": false,
      "ciEnabled": true,
      "resourceHealth": {
        "required": ["ollama", "postgres", "redis"],
        "optional": ["whisper", "comfyui"]
      },
      "integrationTests": {
        "aiInference": "Test AI model responses",
        "databaseOps": "Validate data persistence",
        "workflowExecution": "Test automation workflows",
        "endToEnd": "Complete user journey validation"
      },
      "performanceBaseline": {
        "responseTime": "< 2 seconds",
        "resourceUsage": "< 2GB RAM",
        "concurrentUsers": 10
      }
    }
  }
}
```

## Scenario Deployment Strategies

### Local Development Deployment

**Target Environment:** Developer machines, local testing
**Deployment Method:** Direct scenario execution

**Typical Pipeline:**
1. Start required resources locally
2. Run scenario directly from source
3. Test integration functionality
4. Validate business logic works

```bash
# Local development deployment
vrooli resource start-all
vrooli scenario run research-assistant
vrooli scenario test research-assistant
```

### Production Cloud Deployment

Use the tier documentation instead of ad-hoc scripts:

- Confirm Tier 4 readiness via the [Deployment Hub](../deployment/README.md).
- Follow provider notes (`docs/deployment/providers/`) if a manual prototype is unavoidable.
- Archive any Kubernetes/Vault experiments under `docs/deployment/history/` so future automation can learn from them.

### Customer-Specific Deployment

Until deployment-manager exists, customer installs should ride on Tier 1 infrastructure (app-monitor + Cloudflare tunnels) or receive bespoke instructions captured in the deployment hub. Do **not** rely on `vrooli scenario deploy` or other legacy commands.

## Secret Management for Scenarios

### Scenario Secret Configuration

Scenarios define required secrets for resource access:

```json
{
  "spec": {
    "secrets": {
      "required": [
        "POSTGRES_CONNECTION_STRING",
        "OLLAMA_API_KEY",
        "QDRANT_API_KEY"
      ],
      "optional": [
        "OPENAI_API_KEY",
        "MONITORING_WEBHOOK_URL"
      ],
      "resourceSpecific": {
        "ollama": ["OLLAMA_MODELS_PATH"],
        "postgres": ["DATABASE_URL", "DATABASE_SSL_CERT"],
        "vault": ["VAULT_TOKEN", "VAULT_ADDR"]
      }
    }
  }
}
```

### Security Best Practices for Scenarios

```yaml
# Secure scenario deployment workflow
jobs:
  secure-deploy:
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        
      - name: Configure Resource Secrets
        run: |
          # Create secure configuration for resources
          echo "${{ secrets.RESOURCE_CONFIG }}" > .vrooli/resource-secrets.json
          echo "${{ secrets.VAULT_TOKEN }}" > .vault-token
          chmod 600 .vrooli/resource-secrets.json .vault-token
          
      - name: Deploy Scenarios Securely
        run: |
          # Deploy with secure resource configuration
          export VAULT_TOKEN=$(cat .vault-token)
          vrooli resource start-all --secure
          
          # Deploy scenarios with secret management
          for scenario in research-assistant invoice-generator; do
            vrooli scenario run "$scenario" --production --vault-integration
          done
          
      - name: Cleanup Secrets
        if: always()
        run: |
          rm -f .vrooli/resource-secrets.json .vault-token
          shred -vfz -n 3 /tmp/vrooli-* 2>/dev/null || true
```

## Monitoring and Observability

### Scenario Monitoring Configuration

Scenarios include monitoring for both resource health and business metrics:

```json
{
  "spec": {
    "monitoring": {
      "resourceHealth": {
        "checkInterval": 30,
        "timeout": 10,
        "retries": 3,
        "endpoints": {
          "ollama": "http://localhost:11434/api/tags",
          "postgres": "postgresql://localhost:5432/health",
          "qdrant": "http://localhost:6333/health"
        }
      },
      "businessMetrics": {
        "provider": "questdb",
        "metrics": [
          "scenario_execution_time",
          "resource_response_time", 
          "user_interaction_count",
          "ai_inference_accuracy"
        ]
      },
      "logging": {
        "level": "info",
        "location": "~/.vrooli/logs/scenarios/{scenario_name}/",
        "rotation": "daily"
      },
      "alerts": {
        "resource-failure": "restart-resource",
        "scenario-timeout": "escalate-to-admin",
        "business-metric-anomaly": "notify-stakeholders"
      }
    }
  }
}
```

### Post-Deployment Scenario Verification

```yaml
jobs:
  verify-scenario-deployment:
    runs-on: ubuntu-latest
    needs: deploy-validated-scenarios
    steps:
      - name: Resource Health Check
        run: |
          vrooli resource status --check-all || exit 1
          
      - name: Scenario Integration Validation
        run: |
          for scenario in research-assistant invoice-generator customer-portal; do
            echo "Validating deployed scenario: $scenario"
            vrooli scenario test "$scenario" --production --health-check || exit 1
          done
          
      - name: Business Functionality Verification
        run: |
          # Test end-to-end business workflows
          ./scripts/scenarios/tools/verify-business-functionality.sh --all
          
      - name: Performance Baseline Validation
        run: |
          for scenario in scenarios/*/; do
            name=$(basename "$scenario")
            vrooli scenario test "$name" --performance-baseline
          done
          
      - name: Notify Deployment Success
        if: success()
        run: |
          curl -X POST ${{ secrets.SLACK_WEBHOOK }} \
            -H 'Content-type: application/json' \
            --data '{"text":"✅ Scenario suite deployed successfully: $(echo scenarios/*/ | xargs -n1 basename | tr "\n" ", ")"}'
```

## Benefits of Direct Scenario CI/CD

### Simplicity
- No conversion step - scenarios run directly from source
- Single source of truth for both testing and deployment
- Immediate feedback without regeneration delays

### Resource Efficiency
- Scenarios share Vrooli's resource infrastructure
- No duplication of framework code or configuration
- Optimal resource utilization across multiple scenarios

### Business Alignment
- Scenarios represent actual business value ($10K-50K revenue potential)
- Testing proves deployment readiness and business viability
- Direct path from customer requirements to deployed solution

### Continuous Improvement
- Successful scenarios become permanent platform capabilities
- Resource orchestration patterns improve over time
- Business model validation enhances revenue predictability

This approach enables rapid deployment of business-ready applications while maintaining high quality through comprehensive resource integration testing.
