# Service Configuration Examples

This directory contains modular examples demonstrating the Vrooli Service Schema System. The examples are organized into clear sections to make learning and development easier.

## 📁 Directory Structure

```
examples/
├── README.md                    # This documentation
├── components/                  # Individual schema component examples
│   ├── service.example.json     # Service metadata and identification
│   ├── resources.example.json   # Resource dependencies (ai, automation, agents, storage, execution)
│   ├── execution.example.json   # Code execution environments, sandboxing, and security
│   ├── scenarios.example.json   # Deployment scenarios, initialization sequences, and workflows
│   └── serve.example.json       # Deployment targets, monitoring, scaling, and networking
└── complete/                    # Complete service examples
    ├── ai-assistant.service.json        # Complete AI assistant application
    └── enterprise-platform.service.json # Enterprise automation platform
```

## 📚 Learning Path

### 1. **Start with Components** (`components/`)
Learn individual schema sections:
- **Service metadata**: Basic identification and project info
- **Resources**: AI models, databases, automation tools
- **Execution**: Sandboxing and security environments  
- **Scenarios**: Deployment configurations for different environments
- **Serve**: Production monitoring, scaling, and networking

### 2. **Study Complete Examples** (`complete/`)
See how components work together:
- **AI Assistant**: Basic chat application with local LLM
- **Enterprise Platform**: Advanced automation with multiple integrations

## 🎯 Resource Categories

The examples demonstrate Vrooli's service-centric resource categories:

### **AI Resources** (`resources.ai`)
- **Ollama**: Local LLM hosting (llama3.1, codellama, embeddings)
- **OpenRouter**: Cloud LLM access (Claude, GPT-4, etc.)
- **Custom AI Services**: Internal or specialized AI models

### **Automation Resources** (`resources.automation`)
- **n8n**: Visual workflow automation
- **Windmill**: Developer-first automation platform
- **Zapier**: SaaS automation and integrations

### **Agent Resources** (`resources.agents`)
- **Claude Code**: AI coding assistant
- **Agent-S2**: Web browsing and interaction agent
- **Custom Agents**: Specialized AI agents for specific tasks

### **Storage Resources** (`resources.storage`)
- **PostgreSQL**: Primary relational database
- **Redis**: Caching and session storage
- **MinIO**: Object storage (S3-compatible)
- **Qdrant**: Vector database for embeddings
- **MongoDB**: Document database for analytics
- **Elasticsearch**: Search and log aggregation

### **Execution Resources** (`resources.execution`)
- **BullMQ**: Task queue processing
- **Judge0**: Secure code execution sandbox
- **Custom Executors**: Specialized execution environments

## 🔧 Key Features Demonstrated

### 1. **Flexible Resource Configuration**
Resources support service-specific properties while maintaining common patterns:

```json
{
  "ai": {
    "ollama": {
      "type": "ollama",
      "enabled": true,
      "baseUrl": "http://localhost:11434",
      "models": [...],
      "keepAlive": "5m",
      "temperature": 0.7
    }
  }
}
```

### 2. **Inheritance and Extension**
Services can inherit from Vrooli and override specific sections:

```json
{
  "inheritance": {
    "extends": "../service.json",
    "overrides": {
      "resources": true,
      "scenarios": true
    }
  }
}
```

### 3. **Scenario-Based Deployment**
Different scenarios for different environments:

```json
{
  "scenarios": {
    "local-development": {
      "resources": {
        "include": ["storage.postgres", "ai.ollama"]
      },
      "deployment": {
        "target": "local"
      }
    },
    "production": {
      "resources": {
        "required": ["storage.postgres", "storage.redis", "ai.openai"]
      },
      "deployment": {
        "target": "kubernetes",
        "strategy": "rolling"
      }
    }
  }
}
```

### 4. **Progressive Security**
Different security levels for different environments:

- **Development**: Process sandboxing, relaxed permissions
- **Production**: Container sandboxing, strict permissions
- **WASM**: Maximum security with WebAssembly isolation

### 5. **Resource Profiles**
Named configuration sets for easy environment switching:

```json
{
  "profiles": {
    "minimal": {
      "enabled": ["storage.postgres", "ai.ollama"],
      "disabled": ["ai.openrouter", "automation.n8n"]
    },
    "full": {
      "enabled": ["storage.*", "ai.*", "automation.*"]
    }
  }
}
```

## 🚀 Usage Patterns

### For Learning
1. Start with `components/` to understand individual schema sections
2. Study `complete/ai-assistant.service.json` to see basic integration
3. Examine `complete/enterprise-platform.service.json` for advanced patterns

### For Development
1. Copy relevant examples from `components/` as starting points
2. Customize resource configurations for your needs
3. Define scenarios for your deployment environments
4. Reference `complete/` examples for integration patterns

### For Production
1. Use inheritance to extend Vrooli's configuration
2. Override only what you need to change
3. Define comprehensive validation rules
4. Set up proper monitoring, alerting, and scaling

## 📖 Schema Validation

Validate your configurations using JSON Schema:

```bash
# Install ajv-cli
npm install -g ajv-cli

# Validate individual components
ajv validate -s ../schemas/resources.schema.json -d components/resources.example.json
ajv validate -s ../schemas/execution.schema.json -d components/execution.example.json
ajv validate -s ../schemas/scenarios.schema.json -d components/scenarios.example.json
ajv validate -s ../schemas/serve.schema.json -d components/serve.example.json

# Validate complete services
ajv validate -s ../schemas/service.schema.json -d complete/ai-assistant.service.json
ajv validate -s ../schemas/service.schema.json -d complete/enterprise-platform.service.json
```

## 🔄 From Examples to Applications

These examples show the path from minimal configurations to full applications:

1. **Learn Components**: Start with `components/` to understand each schema section
2. **Basic Integration**: Study `complete/ai-assistant.service.json` for simple applications
3. **Advanced Patterns**: Examine `complete/enterprise-platform.service.json` for complex scenarios  
4. **Production Ready**: Add comprehensive monitoring, scaling, and security
5. **Enterprise Scale**: Add compliance, audit trails, and advanced integrations

The organized structure means you can start with individual components and progressively build to full enterprise applications.