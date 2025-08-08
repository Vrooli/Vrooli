# Deployment Guide: Converting Scenarios to Running Applications

## 🎯 From Validated Scenario to Live Application

Scenarios convert to running applications using the `scenario-to-app.sh` tool, which leverages existing resource infrastructure to create standalone deployable apps.

## 🚀 Quick Start

```bash
# Generate a standalone app from a scenario
./tools/scenario-to-app.sh research-assistant

# With verbose output
./tools/scenario-to-app.sh research-assistant --verbose

# Preview without creating files
./tools/scenario-to-app.sh research-assistant --dry-run
```

Generated apps are placed in `~/generated-apps/<scenario-name>/` and include the full Vrooli infrastructure.

## 🔧 How It Works

The conversion process is simple:

1. **Validates** the scenario's `.vrooli/service.json`
2. **Copies** scenario files and Vrooli infrastructure 
3. **Generates** a standalone app that uses standard Vrooli scripts

```
Scenario Structure              Generated App
├── .vrooli/service.json  →     ├── .vrooli/service.json (copied)
├── initialization/       →     ├── initialization/ (copied)
├── deployment/          →     ├── deployment/ (copied)
└── test.sh             →     ├── test.sh (copied)
                               ├── scripts/ (full Vrooli infrastructure)
                               └── README.md (generated)
```

## 📋 Configuration: Simplified Architecture

**Primary configuration is through `.vrooli/service.json`** with minimal scenario-specific configs for unique business logic only. 

Redundant configurations (app-config.json, feature-flags.json, resource-urls.json) have been removed - all this information is derived from service.json automatically.

## 🏗️ service.json Structure

The service.json file contains everything needed for deployment:

```json
{
  "service": {
    "name": "research-assistant",
    "displayName": "AI Research Assistant",
    "description": "Enterprise-grade AI research platform"
  },
  "resources": {
    "storage": {
      "postgres": {
        "enabled": true,
        "required": true,
        "initialization": {
          "data": [
            {
              "type": "schema", 
              "file": "initialization/storage/postgres/schema.sql"
            }
          ]
        }
      }
    },
    "ai": {
      "ollama": {
        "enabled": true,
        "required": true,
        "models": ["qwen2.5:32b", "nomic-embed-text"]
      }
    },
    "automation": {
      "windmill": {
        "enabled": true,
        "required": true,
        "initialization": {
          "apps": [
            {
              "path": "f/research-assistant/dashboard",
              "file": "initialization/automation/windmill/dashboard-app.json"
            }
          ]
        }
      }
    }
  },
  "deployment": {
    "urls": {
      "application": "${service.vrooli.url}",
      "windmill": "${service.windmill.url}"
    }
  }
}
```

## 🚀 Running Your Generated App

Once generated, run your app like any Vrooli instance:

```bash
# Navigate to generated app
cd ~/generated-apps/research-assistant

# Start the application (uses standard Vrooli scripts)
./scripts/manage.sh develop --target docker --detached yes

# The app will:
# 1. Start required resources based on service.json
# 2. Inject initialization data using resource inject.sh scripts  
# 3. Provide access URLs when ready
```

## 🔧 Customization

Scenarios use a simplified configuration architecture:

- **`.vrooli/service.json`** - Complete application configuration (resources, deployment, business metadata)
- **Domain-specific configs** - Business logic unique to each scenario (e.g., `research-config.json`, `compliance-config.json`)
- **Database schemas and workflows** - Core business functionality

All resource configuration, URLs, and feature flags are derived automatically from service.json.

## 🔍 Troubleshooting

### Common Issues

**App won't start?**
```bash
# Check service.json syntax
cd ~/generated-apps/your-scenario
jq . .vrooli/service.json

# Verify all referenced initialization files exist
find initialization/ -name "*.json" -o -name "*.sql"
```

**Resource connection errors?**
```bash
# Check resource status using Vrooli's resource scripts
./scripts/resources/ai/ollama/manage.sh --action status
./scripts/resources/storage/postgres/manage.sh --action status
```

**Need to reset everything?**
```bash
# Generated apps are self-contained - just delete and regenerate
rm -rf ~/generated-apps/your-scenario
./tools/scenario-to-app.sh your-scenario
```

## 🎯 Next Steps

Ready to deploy a scenario as a standalone application?

1. **Choose a scenario**: Browse `/scripts/scenarios/core/` for available scenarios
2. **Generate the app**: `./tools/scenario-to-app.sh <scenario-name>`  
3. **Run the app**: `cd ~/generated-apps/<scenario-name> && ./scripts/manage.sh develop`

That's it! The generated app is a complete, standalone business application ready for customer deployment.

