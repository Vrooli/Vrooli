# Vrooli Service Schema System

## Overview

The Vrooli Service Schema System provides a modular, extensible configuration framework for defining, deploying, and managing services. It enables Vrooli to function both as a platform and as a template for building derived services.

## Schema Architecture

The schema system is composed of seven modular JSON schemas that work together:

### 1. `service.schema.json` (Main Schema)
The primary schema that references all other schemas. Defines complete service configurations including metadata, resources, execution environments, deployment strategies, and lifecycle management.

**Key Sections:**
- `service`: Service metadata and identification
- `resources`: Resource dependencies and configurations
- `execution`: Code execution environments and sandboxing
- `scenarios`: Deployment scenarios and profiles
- `serve`: Deployment, monitoring, and scaling
- `inheritance`: Configuration inheritance and extension
- `lifecycle`: Service lifecycle management

### 2. `common.schema.json` (Shared Definitions)
Contains reusable definitions used across all schemas:
- Data types (semver, port, url, path, duration, memory, cpu)
- Common structures (environment variables, labels, annotations)
- Health checks and secret references
- Resource requirements and network policies
- Lifecycle hooks

### 3. `resources.schema.json` (Resource Management)
Defines resource dependencies and lifecycle:
- **Databases**: PostgreSQL, MySQL, MongoDB, etc.
- **Caches**: Redis, Memcached, Hazelcast
- **Queues**: RabbitMQ, Kafka, BullMQ
- **Storage**: S3, MinIO, cloud storage
- **Search**: Elasticsearch, Qdrant, Solr
- **AI/ML**: LLMs, embeddings, custom models
- **Monitoring**: Prometheus, Grafana, DataDog
- **External**: Third-party APIs and services

### Resource Dependency Catalog

Scenario resource dependency authoring is backed by the generated `resource-definitions.json` catalog, but that catalog is now derived from each active resource's `resource.json` manifest rather than `resources/*/config/schema.json`.

The manifest-native split is:
- `resource.json`: canonical runtime and dependency contract
- `resource.json.dependency_schema`: scenario-facing dependency authoring schema
- `resource.json.orchestration`: lifecycle ordering and startup hints
- `resource-definitions.json`: generated editor/schema artifact

### 4. `execution.schema.json` (Execution Environment)
Manages code execution, sandboxing, and security:
- **Sandboxing**: Process, container, VM, WASM isolation
- **Permissions**: Filesystem, network, process, system
- **Runtimes**: Language support and configuration
- **Security**: Authentication, authorization, encryption
- **Limits**: Resource constraints and quotas
- **Auditing**: Execution tracking and logging

### 5. `scenarios.schema.json` (Service Scenarios)
Defines deployment scenarios and workflows:
- **Scenarios**: Named deployment configurations
- **Profiles**: Reusable configuration sets
- **Templates**: Parameterized configurations
- **Workflows**: Multi-step deployment processes
- **Initialization**: Resource startup sequences
- **Validation**: Pre/post deployment checks

### 6. `serve.schema.json` (Deployment & Operations)
Handles deployment, monitoring, and scaling:
- **Deployment**: Targets, strategies, artifacts
- **Monitoring**: Metrics, logging, tracing, profiling
- **Scaling**: Horizontal, vertical, cluster autoscaling
- **Networking**: Ingress, service mesh, CDN, load balancing
- **Observability**: SLOs, SLIs, error budgets
- **Resilience**: Circuit breakers, retries, rate limiting

### 7. `repo-contract.schema.json` (Repository Structure Contract)
Defines the future-state cross-platform repository contract used by repo-aware tooling:
- **Root markers**: required top-level directories and files used for canonical repo detection
- **Layout**: canonical top-level directories such as `.vrooli/`, `scenarios/`, `resources/`, `packages/`, `cmd/`, and `internal/`
- **Scenario layout**: `scenarios/<name>/.vrooli/service.json` and other shared well-known scenario paths
- **Resource layout**: `resources/<name>/resource.json` plus stable optional subpaths
- **Glob semantics**: root-relative, slash-normalized, `doublestar`-style semantics
- **Shared env vars**: `VROOLI_ROOT`, `VROOLI_SOURCE_ROOT`, and sandbox env vars
- **Repo-aware profiles**: named include/exclude profiles for future adapter-backed bundle tooling

## Usage Examples

### Basic Service Configuration

```json
{
  "$schema": "./schemas/service.schema.json",
  "version": "1.0.0",
  "service": {
    "name": "my-service",
    "version": "1.0.0",
    "type": "api"
  },
  "resources": {
    "databases": {
      "postgres": {
        "type": "postgresql",
        "enabled": true,
        "connection": {
          "host": "localhost",
          "port": 5432,
          "database": "myapp"
        }
      }
    }
  }
}
```

### Inheritance and Extension

Services can inherit from Vrooli or other base configurations:

```json
{
  "$schema": "../schemas/service.schema.json",
  "version": "1.0.0",
  "service": {
    "name": "derived-service",
    "version": "1.0.0",
    "type": "platform"
  },
  "inheritance": {
    "extends": "../service.json",
    "overrides": {
      "resources": true,
      "execution": false
    }
  }
}
```

### Scenario-Based Deployment

Define multiple deployment scenarios:

```json
{
  "scenarios": {
    "development": {
      "name": "development",
      "category": "development",
      "resources": {
        "include": ["postgres", "redis"],
        "exclude": ["monitoring"]
      },
      "deployment": {
        "target": "local"
      }
    },
    "production": {
      "name": "production",
      "category": "production",
      "resources": {
        "required": ["postgres", "redis", "monitoring"]
      },
      "deployment": {
        "target": "kubernetes",
        "strategy": "blue-green"
      }
    }
  }
}
```

## Key Features

### 1. Modularity
Each schema focuses on a specific aspect of service configuration, allowing for:
- Independent evolution of schema components
- Selective inclusion of configuration sections
- Clear separation of concerns

### 2. Inheritance
Services can inherit from base configurations:
- Extend existing services (like Vrooli)
- Override specific sections
- Merge configurations with various strategies

### 3. Scenarios and Profiles
Support for multiple deployment scenarios:
- Development, testing, staging, production
- Different resource combinations
- Environment-specific configurations

### 4. Resource Abstraction
Service-centric approach aligned with Vrooli's architecture:
- AI models and ML services (Ollama, OpenRouter, Whisper)
- Automation platforms (n8n, Windmill, Node-RED)
- AI agents (Claude Code, Agent-S2)
- Storage systems (PostgreSQL, Redis, MinIO, Qdrant)
- Execution environments (Judge0, BullMQ queues)

Resources use flexible configurations with `additionalProperties: true`, allowing service-specific properties while maintaining common patterns for authentication, health checks, and monitoring.

### 5. Security and Sandboxing
Comprehensive security model:
- Multiple sandboxing strategies (process, container, VM, WASM)
- Fine-grained permission controls
- Audit logging and compliance

### 6. Lifecycle Management
Complete service lifecycle support:
- Pre/post hooks for all lifecycle events
- Dependency management
- Health checks and readiness probes

## Benefits

### For Vrooli as a Platform
- **Standardized Configuration**: Consistent configuration across all components
- **Multi-Environment Support**: Easy switching between development, testing, and production
- **Resource Management**: Centralized resource configuration and lifecycle
- **Deployment Flexibility**: Support for local, Docker, Kubernetes, and cloud deployments

### For Service Derivation
- **Template Reuse**: Use Vrooli as a template for new services
- **Configuration Inheritance**: Inherit and customize Vrooli's configuration
- **Selective Overrides**: Change only what's needed, inherit the rest
- **Ecosystem Compatibility**: Maintain compatibility with Vrooli tooling

### For Operations
- **Observability**: Built-in monitoring, logging, and tracing
- **Resilience**: Circuit breakers, retries, and rate limiting
- **Scaling**: Automatic horizontal and vertical scaling
- **Security**: Comprehensive security controls and auditing

## Schema Validation

Validate service configurations using JSON Schema validators:

```bash
# Install ajv-cli
npm install -g ajv-cli

# Validate a service configuration
ajv validate -s .vrooli/schemas/service.schema.json -d .vrooli/service.json
```

Validate the repository contract and its schema:

```bash
python3 .vrooli/schemas/validate-repo-contract.py
make validate-repo-contract
```

The repo contract is future-state only. Do not add transitional project-level shell paths, historical repo-root fallbacks, or other legacy structure to `.vrooli/repo-contract.json`.

## Migration from Existing Configuration

To migrate existing Vrooli configurations:

1. **Consolidate Resources**: Move resource configurations to the `resources` section
2. **Define Execution Environments**: Configure sandboxing and permissions
3. **Create Scenarios**: Define deployment scenarios for different environments
4. **Setup Monitoring**: Configure metrics, logging, and alerts
5. **Add Lifecycle Hooks**: Define initialization and cleanup procedures

## Best Practices

1. **Start Simple**: Begin with minimal configuration and add complexity as needed
2. **Use Inheritance**: Leverage base configurations to reduce duplication
3. **Define Scenarios**: Create specific scenarios for each deployment environment
4. **Version Everything**: Use semantic versioning for services and configurations
5. **Document Custom Fields**: Add descriptions for any custom configuration
6. **Validate Early**: Use schema validation in CI/CD pipelines
7. **Secure Secrets**: Use secret references, never hardcode credentials

## Future Enhancements

- **Schema Registry**: Central repository for schema versions
- **Configuration Generator**: UI/CLI for generating configurations
- **Validation Library**: Runtime validation and type safety
- **Migration Tools**: Automated migration from other formats
- **Cloud Templates**: Pre-built templates for cloud providers
- **Compliance Profiles**: Industry-specific compliance configurations

## Contributing

To contribute to the schema system:

1. Propose changes via GitHub issues
2. Maintain backward compatibility
3. Update documentation and examples
4. Add validation tests
5. Follow semantic versioning

## License

The Vrooli Service Schema System is part of the Vrooli project and is licensed under GPL-3.0.
