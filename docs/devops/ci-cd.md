# CI/CD for Generated Applications

This guide covers CI/CD patterns and strategies for applications generated from Vrooli scenarios. Since each generated app has its own deployment requirements, this focuses on common patterns and best practices.

> **Prerequisites**: See [Prerequisites Guide](./getting-started/prerequisites.md) for required tools installation.

## Overview

When Vrooli generates applications from scenarios, each app can implement its own CI/CD strategy based on its specific needs:

### Generated App CI/CD Characteristics

- 🎯 **Purpose-Built**: Each app's CI/CD matches its deployment targets (web platforms, app stores, cloud services)
- 🔄 **Lifecycle Integration**: Uses `vrooli build` and `vrooli deploy` commands defined in `.vrooli/service.json`
- 📦 **Artifact Variety**: Different apps generate different artifacts (web bundles, mobile packages, container images)
- 🚀 **Target Flexibility**: Apps can deploy to multiple platforms simultaneously
- 🛡️ **Security First**: Built-in secret management and secure deployment practices

## Common CI/CD Patterns

### Pattern 1: Web Application (SaaS/Website)

For apps that deploy to web platforms like Vercel, Netlify, or custom servers:

```yaml
# .github/workflows/deploy.yml for a generated SaaS app
name: Deploy Web Application

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '18'
          cache: 'npm'
          
      - name: Install dependencies
        run: npm ci
        
      - name: Build application
        run: vrooli build --environment production
        
      - name: Deploy to production
        if: github.ref == 'refs/heads/main'
        run: vrooli deploy --target vercel --environment production
        env:
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
          
      - name: Deploy to preview
        if: github.event_name == 'pull_request'
        run: vrooli deploy --target vercel --environment preview
        env:
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
```

### Pattern 2: Mobile Application

For apps targeting app stores:

```yaml
# .github/workflows/mobile-deploy.yml for a generated mobile app
name: Build and Deploy Mobile App

on:
  push:
    tags: ['v*']

jobs:
  build-android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Java
        uses: actions/setup-java@v4
        with:
          distribution: 'temurin'
          java-version: '17'
          
      - name: Setup Android SDK
        uses: android-actions/setup-android@v2
        
      - name: Build Android APK
        run: vrooli build --platform android
        
      - name: Sign and Deploy to Play Store
        run: vrooli deploy --target play-store
        env:
          ANDROID_SIGNING_KEY: ${{ secrets.ANDROID_SIGNING_KEY }}
          PLAY_STORE_SERVICE_ACCOUNT: ${{ secrets.PLAY_STORE_SERVICE_ACCOUNT }}

  build-ios:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Xcode
        uses: actions/setup-xcode@v1
        with:
          xcode-version: latest-stable
          
      - name: Build iOS IPA
        run: vrooli build --platform ios
        
      - name: Deploy to App Store
        run: vrooli deploy --target app-store
        env:
          APPLE_ID: ${{ secrets.APPLE_ID }}
          APPLE_ID_PASSWORD: ${{ secrets.APPLE_ID_PASSWORD }}
          APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
```

### Pattern 3: AI/ML Service

For AI applications deploying to cloud platforms:

```yaml
# .github/workflows/ai-deploy.yml for a generated AI service
name: Deploy AI Service

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.11'
          
      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install -r requirements-dev.txt
          
      - name: Build AI model package
        run: vrooli build --artifacts model,container
        
      - name: Deploy to cloud
        run: vrooli deploy --target aws --service lambda
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          AWS_REGION: us-east-1
```

### Pattern 4: Multi-Platform Deployment

For apps that deploy to multiple platforms:

```yaml
# .github/workflows/multi-deploy.yml
name: Multi-Platform Deployment

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.version.outputs.version }}
    steps:
      - uses: actions/checkout@v4
      
      - name: Generate version
        id: version
        run: echo "version=$(date +%Y%m%d-%H%M%S)" >> $GITHUB_OUTPUT
        
      - name: Build all platforms
        run: vrooli build --platforms web,android,desktop
        
      - name: Upload build artifacts
        uses: actions/upload-artifact@v3
        with:
          name: build-artifacts-${{ steps.version.outputs.version }}
          path: dist/

  deploy-web:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: build-artifacts-${{ needs.build.outputs.version }}
          path: dist/
          
      - name: Deploy web version
        run: vrooli deploy --target netlify --artifacts dist/web
        env:
          NETLIFY_AUTH_TOKEN: ${{ secrets.NETLIFY_AUTH_TOKEN }}

  deploy-desktop:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: build-artifacts-${{ needs.build.outputs.version }}
          path: dist/
          
      - name: Deploy desktop apps
        run: vrooli deploy --target github-releases --artifacts dist/desktop
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Application Configuration Integration

### Service Configuration

Generated apps define their CI/CD behavior in `.vrooli/service.json`:

```json
{
  "name": "my-generated-saas",
  "type": "web-application",
  "ci-cd": {
    "triggers": {
      "build": ["push", "pull_request"],
      "deploy": ["push:main", "tag:v*"]
    },
    "environments": {
      "development": {
        "auto-deploy": true,
        "targets": ["vercel-preview"]
      },
      "production": {
        "auto-deploy": false,
        "targets": ["vercel", "cloudflare"],
        "requires-approval": true
      }
    },
    "quality-gates": {
      "test": "required",
      "lint": "required", 
      "security-scan": "optional"
    }
  },
  "deployment": {
    "targets": {
      "vercel": {
        "command": "vercel deploy --prod",
        "secrets": ["VERCEL_TOKEN"]
      },
      "cloudflare": {
        "command": "wrangler deploy",
        "secrets": ["CLOUDFLARE_API_TOKEN"]
      }
    }
  }
}
```

### Environment Management

Generated apps can use standardized environment patterns:

```bash
# Development environment (.env.development)
NODE_ENV=development
API_BASE_URL=https://dev-api.myapp.com
DEBUG_MODE=true

# Production environment (.env.production)
NODE_ENV=production
API_BASE_URL=https://api.myapp.com
DEBUG_MODE=false
ANALYTICS_ENABLED=true
```

## Quality Gates and Testing

### Automated Testing Integration

```yaml
# Example quality gate workflow
jobs:
  quality-gates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install dependencies
        run: npm ci
        
      - name: Run linting
        run: vrooli test lint
        
      - name: Run unit tests
        run: vrooli test unit
        
      - name: Run integration tests
        run: vrooli test integration
        
      - name: Security scan
        run: vrooli test security
        
      - name: Performance test
        if: github.ref == 'refs/heads/main'
        run: vrooli scenario test performance

  deploy:
    needs: quality-gates
    if: success()
    runs-on: ubuntu-latest
    steps:
      - name: Deploy application
        run: vrooli deploy --environment production
```

### Test Configuration

Apps define their testing strategy in `.vrooli/service.json`:

```json
{
  "testing": {
    "lint": {
      "command": "eslint src/ --ext .ts,.tsx",
      "required": true
    },
    "unit": {
      "command": "jest --coverage",
      "coverage-threshold": 80
    },
    "integration": {
      "command": "playwright test",
      "browser-matrix": ["chromium", "firefox", "webkit"]
    },
    "security": {
      "command": "npm audit && snyk test",
      "fail-on": "high"
    }
  }
}
```

## Deployment Strategies by App Type

### SaaS Applications

**Common Deployment Targets:**
- Vercel/Netlify for frontend
- Railway/Render for backend APIs
- Planet Scale for databases
- Cloudflare for CDN and edge functions

**Typical Pipeline:**
1. Build frontend and backend separately
2. Deploy backend API first
3. Update frontend environment variables
4. Deploy frontend with new API endpoints
5. Run smoke tests against production

### E-commerce Platforms

**Common Deployment Targets:**
- Shopify app store
- WordPress plugin directory
- WooCommerce marketplace
- Custom hosting platforms

**Typical Pipeline:**
1. Build platform-specific packages
2. Run comprehensive testing (payment flows, inventory)
3. Deploy to staging environment
4. Manual QA and approval
5. Deploy to production marketplace

### AI/ML Applications

**Common Deployment Targets:**
- AWS Lambda/SageMaker
- Google Cloud Run/Vertex AI
- Azure Container Instances
- Hugging Face Spaces

**Typical Pipeline:**
1. Build and package models
2. Create container images
3. Run model validation tests
4. Deploy to inference endpoints
5. Gradual traffic shifting (canary deployment)

### Mobile Applications

**Common Deployment Targets:**
- Google Play Store
- Apple App Store
- Enterprise distribution (TestFlight, Firebase)
- Progressive Web App (PWA) platforms

**Typical Pipeline:**
1. Build platform-specific packages
2. Run automated UI testing
3. Upload to internal testing (TestFlight, Internal Testing)
4. Manual QA approval
5. Release to production stores

## Secret Management

### Generated App Secrets

Apps define required secrets in their configuration:

```json
{
  "secrets": {
    "required": [
      "DATABASE_URL",
      "JWT_SECRET", 
      "STRIPE_API_KEY"
    ],
    "optional": [
      "ANALYTICS_API_KEY",
      "SENTRY_DSN"
    ],
    "deployment": {
      "vercel": ["VERCEL_TOKEN"],
      "aws": ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"]
    }
  }
}
```

### Security Best Practices

```yaml
# Example secure deployment workflow
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production  # GitHub environment protection
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        
      - name: Configure secrets
        run: |
          echo "${{ secrets.ENV_FILE }}" > .env.production
          echo "${{ secrets.SERVICE_ACCOUNT_KEY }}" > service-account.json
          chmod 600 .env.production service-account.json
          
      - name: Deploy with secure cleanup
        run: |
          vrooli deploy --environment production
        always: |
          rm -f .env.production service-account.json
```

## Monitoring and Observability

### Deployment Monitoring

Generated apps can include monitoring configuration:

```json
{
  "monitoring": {
    "health-checks": {
      "endpoint": "/health",
      "timeout": 30,
      "retries": 3
    },
    "metrics": {
      "provider": "datadog",
      "custom-metrics": ["user_signups", "api_latency", "error_rate"]
    },
    "logging": {
      "provider": "sentry",
      "log-level": "info"
    },
    "alerts": {
      "deployment-failure": "slack://deployment-alerts",
      "high-error-rate": "pagerduty://on-call"
    }
  }
}
```

### Post-Deployment Verification

```yaml
jobs:
  verify-deployment:
    runs-on: ubuntu-latest
    needs: deploy
    steps:
      - name: Health check
        run: |
          curl -f ${{ env.HEALTH_CHECK_URL }} || exit 1
          
      - name: Integration test
        run: vrooli test integration --environment production
        
      - name: Performance baseline
        run: vrooli scenario test performance --baseline --environment production
        
      - name: Notify success
        if: success()
        run: |
          curl -X POST ${{ secrets.SLACK_WEBHOOK }} \
            -H 'Content-type: application/json' \
            --data '{"text":"✅ Deployment successful for ${{ github.repository }}"}'
```

## Benefits of Generated App CI/CD

### Flexibility
- Each app chooses its optimal deployment strategy
- No platform lock-in - apps can switch deployment targets
- Technology-agnostic - works with any tech stack

### Intelligence
- Apps learn from successful deployment patterns
- Common patterns become templates for new apps
- Failed deployments inform better practices

### Scaling
- Apps can start simple and evolve deployment complexity
- Infrastructure scales with application requirements
- Deployment knowledge compounds across all generated apps

This approach enables each generated application to have a CI/CD pipeline perfectly suited to its specific needs, deployment targets, and operational requirements.