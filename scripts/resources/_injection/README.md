# Resource Data Injection System

**Transform resource startup into complete application deployment through declarative scenario configurations.**

## 🎯 Overview

The Resource Data Injection System extends Vrooli's resource management to automatically inject workflows, configurations, data, and scripts into resources based on **scenario specifications**. This enables **declarative application deployment** where a single JSON file defines everything needed to deploy a complete multi-resource application.

### Key Benefits

- **🚀 Rapid Deployment**: Deploy complete applications from JSON specifications
- **📋 Standardized Templates**: Consistent patterns for SaaS, e-commerce, analytics platforms
- **🤖 AI Generation Ready**: Perfect foundation for automated application creation  
- **🔄 Reproducible**: Same scenario always produces identical resource configurations
- **🛡️ Safe Operations**: Comprehensive validation and rollback support

## 🏗️ Architecture

```
📁 _injection/
├── 🎛️ engine.sh              # Central orchestrator
├── 📋 schema-validator.sh     # Configuration validation
├── 📄 config/
│   ├── schema.json           # JSON schema for scenarios
│   └── defaults.json         # Example scenario templates
└── 📚 docs/                  # Documentation (this folder)
```

### Core Components

1. **🎛️ Engine (`engine.sh`)**: Orchestrates injection across resources
2. **📋 Validator (`schema-validator.sh`)**: Validates scenario configurations  
3. **🔌 Adapters**: Co-located with each resource (e.g., `n8n/inject.sh`)
4. **📄 Schema**: Comprehensive JSON schema for validation

## 🚀 Quick Start

### 1. Initialize Configuration

```bash
# Create scenarios configuration from defaults
./scripts/resources/_injection/schema-validator.sh --action init

# Or create custom location
./scripts/resources/_injection/schema-validator.sh --action init --config-file ./my-scenarios.json
```

### 2. Inject Scenarios

```bash
# Inject specific scenario
./scripts/resources/index.sh --action inject --scenario vrooli-core

# Inject all active scenarios
./scripts/resources/index.sh --action inject-all

# Dry run to see what would be injected
./scripts/resources/_injection/engine.sh --action inject --scenario test-scenario --dry-run yes
```

### 3. Manage Scenarios

```bash
# List available scenarios
./scripts/resources/_injection/engine.sh --action list-scenarios

# Validate configuration
./scripts/resources/_injection/schema-validator.sh --action validate

# Validate specific scenario
./scripts/resources/_injection/engine.sh --action validate --scenario vrooli-core
```

## 📋 Scenario Configuration Format

Scenarios are defined in `~/.vrooli/scenarios.json`:

```json
{
  "version": "1.0.0",
  "scenarios": {
    "my-app": {
      "description": "Complete SaaS application",
      "version": "1.0.0",
      "dependencies": ["vrooli-core"],
      "resources": {
        "n8n": {
          "workflows": [
            {
              "name": "user-onboarding",
              "file": "scenarios/my-app/workflows/onboarding.json",
              "enabled": true,
              "tags": ["users", "onboarding"]
            }
          ],
          "credentials": [
            {
              "name": "api-auth",
              "type": "httpAuth",
              "config": {"url": "http://localhost:3000/api"}
            }
          ]
        },
        "windmill": {
          "apps": [
            {
              "path": "f/myapp/dashboard",
              "file": "scenarios/my-app/apps/dashboard.json",
              "summary": "Main application dashboard"
            }
          ]
        },
        "postgres": {
          "data": [
            {
              "type": "schema",
              "file": "scenarios/my-app/sql/schema.sql"
            }
          ]
        }
      }
    }
  },
  "active": ["my-app"]
}
```

### Supported Resource Types

| Resource | Injection Types | Example |
|----------|----------------|---------|
| **n8n** | workflows, credentials | Automation workflows, API credentials |
| **Windmill** | scripts, apps | TypeScript/Python scripts, web applications |
| **Node-RED** | flows | Real-time data flows, dashboards |
| **PostgreSQL** | schema, seed, migration | Database tables, sample data |
| **MinIO** | buckets, files | S3-compatible storage, file uploads |
| **Ollama** | models, modelfiles | Local AI models, custom configurations |
| **Qdrant** | collections, vectors | Vector databases, embeddings |
| **QuestDB** | tables, schemas | Time-series data structures |

## 🔌 Resource Adapter Interface

Each resource implements a standardized injection adapter (`resource/inject.sh`):

```bash
# Standard interface for all adapters
./resource/inject.sh --validate CONFIG_JSON    # Validate injection config
./resource/inject.sh --inject CONFIG_JSON     # Perform injection  
./resource/inject.sh --status CONFIG_JSON     # Check injection status
./resource/inject.sh --rollback CONFIG_JSON   # Rollback injection
```

### Example: Direct Resource Injection

```bash
# Inject workflows directly into n8n
./scripts/resources/automation/n8n/inject.sh --inject '{
  "workflows": [
    {
      "name": "test-workflow",
      "file": "path/to/workflow.json",
      "enabled": true
    }
  ]
}'

# Validate before injection
./scripts/resources/automation/n8n/inject.sh --validate '{
  "workflows": [{"name": "test", "file": "test.json"}]
}'
```

## 📚 Configuration Schema

### Scenario Structure

```typescript
interface Scenario {
  description: string;           // Human-readable description
  version: string;              // Semantic version (X.Y.Z)
  dependencies?: string[];      // Other scenarios to inject first
  resources: {
    [resourceName: string]: ResourceInjectionConfig;
  };
}
```

### Resource Injection Types

```typescript
interface ResourceInjectionConfig {
  workflows?: WorkflowInjection[];      // n8n workflows
  scripts?: ScriptInjection[];          // Windmill scripts  
  apps?: AppInjection[];                // Windmill applications
  flows?: FlowInjection[];              // Node-RED flows
  data?: DataInjection[];               // Database/file data
  buckets?: BucketInjection[];          // Storage buckets
  models?: ModelInjection[];            // AI models
  credentials?: CredentialInjection[];  // Authentication configs
}
```

## 🎨 Scenario Templates

### Built-in Templates

The system includes ready-to-use templates in `config/defaults.json`:

- **`vrooli-core`**: Essential Vrooli platform workflows and monitoring
- **`example-ecommerce`**: Complete e-commerce store with orders, payments, inventory
- **`saas-analytics`**: Analytics platform with real-time dashboards

### Creating Custom Templates

```bash
# Start with defaults and customize
cp scripts/resources/_injection/config/defaults.json my-template.json

# Validate your template
./scripts/resources/_injection/schema-validator.sh --action validate --config-file my-template.json
```

## 🔄 Dependency Management

Scenarios can depend on other scenarios:

```json
{
  "scenarios": {
    "base-platform": {
      "description": "Core platform services",
      "resources": {"n8n": {"workflows": [...]}}
    },
    "ecommerce-store": {
      "description": "E-commerce application", 
      "dependencies": ["base-platform"],
      "resources": {"windmill": {"apps": [...]}}
    }
  }
}
```

Dependencies are resolved automatically and injected in the correct order.

## 🛡️ Error Handling & Rollback

### Validation

Every injection is validated before execution:

```bash
# Configuration validation
./scripts/resources/_injection/schema-validator.sh --action validate

# Scenario-specific validation  
./scripts/resources/_injection/engine.sh --action validate --scenario my-app

# Resource-specific validation
./scripts/resources/automation/n8n/inject.sh --validate CONFIG_JSON
```

### Rollback Support

Failed injections automatically trigger rollback:

```bash
# Manual rollback
./scripts/resources/_injection/engine.sh --action rollback

# Resource-specific rollback
./scripts/resources/automation/n8n/inject.sh --rollback CONFIG_JSON
```

### Safe Operations

- **Dry Run Mode**: See what would be injected without making changes
- **Validation First**: All configurations validated before injection
- **Atomic Operations**: Either all resources succeed or all are rolled back
- **Idempotent**: Same scenario can be injected multiple times safely

## 🚀 Integration with Resource Management

### Startup Integration

Add injection to resource startup:

```bash
# Start resources and inject active scenarios
./scripts/main/setup.sh --target native-linux --resources enabled
# (automatically injects active scenarios after resource startup)
```

### Manual Integration

```bash
# Inject specific scenario
./scripts/resources/index.sh --action inject --scenario my-app

# Inject all active scenarios
./scripts/resources/index.sh --action inject-all
```

## 🎯 Use Cases

### 1. **SaaS Application Templates**
Deploy complete SaaS applications with authentication, dashboards, and analytics:
```json
{
  "scenarios": {
    "saas-starter": {
      "resources": {
        "n8n": {"workflows": ["user-auth", "subscription-management"]},
        "windmill": {"apps": ["admin-dashboard", "user-portal"]},
        "postgres": {"data": [{"type": "schema", "file": "saas-schema.sql"}]}
      }
    }
  }
}
```

### 2. **E-commerce Platforms**
Complete online stores with inventory, orders, and payments:
```json
{
  "scenarios": {
    "ecommerce-store": {
      "resources": {
        "n8n": {"workflows": ["order-processing", "inventory-sync"]},
        "windmill": {"apps": ["store-dashboard", "product-catalog"]},
        "minio": {"buckets": [{"name": "product-images", "policy": "public-read"}]}
      }
    }
  }
}
```

### 3. **Analytics Platforms**
Real-time analytics with dashboards and data processing:
```json
{
  "scenarios": {
    "analytics-platform": {
      "resources": {
        "questdb": {"data": [{"type": "schema", "file": "analytics-tables.sql"}]},
        "node-red": {"flows": [{"name": "data-ingestion", "file": "ingestion.json"}]},
        "windmill": {"apps": [{"path": "f/analytics/dashboard", "file": "dashboard.json"}]}
      }
    }
  }
}
```

## 🔧 Extending the System

### Creating New Resource Adapters

1. **Create injection adapter**: `resource/inject.sh`
2. **Implement standard interface**: `--validate|--inject|--status|--rollback`
3. **Add to resource management**: Extend `resource/manage.sh`
4. **Document configuration format**: Update schema if needed

See [Adapter Development Guide](docs/adapter-development.md) for details.

### Adding New Injection Types

1. **Update JSON schema**: Add new injection type to `config/schema.json`
2. **Implement in adapters**: Handle new type in relevant `inject.sh` scripts
3. **Add examples**: Include in `config/defaults.json`
4. **Update documentation**: Document the new injection type

## 📖 Advanced Topics

- **[Adapter Development Guide](docs/adapter-development.md)**: Create injection adapters for new resources
- **[Scenario Design Patterns](docs/scenario-patterns.md)**: Best practices for scenario design
- **[Troubleshooting Guide](docs/troubleshooting.md)**: Common issues and solutions
- **[API Reference](docs/api-reference.md)**: Complete interface documentation

## 🤝 Contributing

When adding new features:

1. **Follow the standard adapter interface**
2. **Update the JSON schema for new configuration options**
3. **Add comprehensive validation**
4. **Include rollback support**
5. **Add examples to defaults.json**
6. **Update documentation**

## 🎯 Future Roadmap

- **🤖 AI Generation**: Automatic scenario generation from requirements
- **🌐 Remote Scenarios**: Pull scenarios from repositories
- **📊 Analytics**: Track scenario deployment success and performance
- **🔄 Hot Reload**: Dynamic scenario updates without restart
- **🎨 UI Generator**: Visual scenario builder interface

---

**The Resource Data Injection System transforms Vrooli from a resource orchestrator into a complete declarative application deployment platform.**