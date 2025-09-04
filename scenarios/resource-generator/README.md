# Resource Generator

Automated system for creating new Vrooli resources with v2.0 contract compliance, complete scaffolding, and seamless integration.

## Overview

The Resource Generator creates brand new resources for the Vrooli ecosystem, ensuring each resource:
- Meets v2.0 contract requirements
- Has complete scaffolding and structure
- Includes health checks and lifecycle management
- Integrates with CLI and Docker systems
- Is properly documented and tested

## Architecture

### Components
- **Scaffolding Engine**: Generates complete resource directory structure
- **Contract Validator**: Ensures v2.0 compliance
- **Port Allocator**: Manages port assignments without conflicts  
- **Integration System**: Sets up CLI, Docker, and resource registration
- **Documentation Generator**: Creates comprehensive README and API docs

### Resources
- **Claude Code**: AI-powered code generation
- **PostgreSQL**: Template storage and generation history
- **Redis**: Queue management and port allocation cache
- **Qdrant**: Semantic search for patterns and best practices
- **N8n**: Workflow automation for resource creation

## Key Features

### 1. v2.0 Contract Compliance
Every generated resource includes:
- Health check implementation
- Lifecycle hooks (setup, develop, test, stop)
- CLI integration
- Docker configuration
- Resource registration

### 2. Intelligent Port Allocation
- Automatic port assignment from designated ranges
- Conflict prevention
- Registry management
- Environment variable configuration

### 3. Template-Based Generation
Pre-built templates for:
- AI/ML resources
- Data processing systems
- Workflow automation
- Monitoring tools
- Communication platforms

### 4. Complete Integration
- Automatic CLI command registration
- Docker compose configuration
- Resource dependency management
- Cross-resource communication setup

## Usage

### Starting the Generator
```bash
# Setup (first time)
vrooli scenario resource-generator setup

# Run generator service
vrooli scenario resource-generator develop
```

### Creating a New Resource
```bash
# Using CLI
resource-generator create --name matrix-synapse --template communication --priority high

# Or add to queue manually
cp queue/templates/new-resource.yaml queue/pending/001-create-matrix.yaml
# Edit file with specifications
```

### Checking Generation Status
```bash
# View queue status
resource-generator status

# List available templates
resource-generator list-templates

# Check port allocations
resource-generator ports --available
```

## Queue System

The queue manages resource creation requests:

```
queue/
├── pending/        # Resources waiting to be created
├── in-progress/    # Currently being generated
├── completed/      # Successfully created resources
├── failed/         # Failed attempts with logs
└── templates/      # Request templates
```

## Resource Creation Process

### 1. Request Analysis
- Parse resource requirements
- Select appropriate template
- Check dependencies and conflicts
- Plan integration points

### 2. Scaffolding Generation
- Create directory structure
- Generate base files from template
- Customize for specific resource
- Add v2.0 contract files

### 3. Port Allocation
- Find available port in range
- Register in port registry
- Configure in service.json
- Set environment variables

### 4. Integration Setup
- Register CLI commands
- Configure Docker if needed
- Set up resource dependencies
- Create initialization scripts

### 5. Documentation
- Generate README with examples
- Document API endpoints
- Create troubleshooting guide
- Add to resource catalog

### 6. Testing
- Validate health checks
- Test lifecycle operations
- Verify CLI commands
- Check integration points

## Resource Catalog

### Ready for Creation

#### Communication
- **Matrix Synapse**: Federated chat/rooms with E2E encryption
- **Mattermost**: Team collaboration platform
- **Rocket.Chat**: Open source team chat

#### Security
- **OWASP ZAP**: Automated security scanning
- **Vault**: Secrets management
- **Trivy**: Container vulnerability scanning

#### Data Integration
- **Airbyte**: ELT data pipelines with 600+ connectors
- **Apache NiFi**: Data flow automation
- **Debezium**: Change data capture

#### AI/ML
- **DeepStack**: Computer vision API
- **Label Studio**: Data labeling platform
- **MLflow**: ML lifecycle management

#### Robotics
- **ROS2**: Robot Operating System
- **Webots**: Robot simulation
- **MQTT**: IoT messaging

## Port Allocation Ranges

- **20000-24999**: Core infrastructure (databases, queues)
- **25000-29999**: Supporting services (monitoring, security)
- **30000-34999**: Application scenarios
- **35000-39999**: Development tools
- **40000-49999**: Dynamic allocation pool

## Templates

### AI-Powered Template
```yaml
includes:
  - Model management UI
  - Inference API
  - Training pipeline
  - GPU configuration
  - Performance monitoring
```

### Data Processing Template
```yaml
includes:
  - ETL pipeline setup
  - Data validation
  - Batch processing
  - Stream processing
  - Output connectors
```

### Communication Template
```yaml
includes:
  - Message routing
  - Authentication
  - Webhook management
  - Real-time capabilities
  - Protocol bridges
```

## Examples

### Example: Creating Matrix Synapse
```yaml
# queue/pending/001-create-matrix.yaml
id: create-matrix-synapse-20250103
title: "Create Matrix Synapse resource for federated chat"
type: communication
priority: high
requirements:
  base_template: communication
  specific_needs:
    - PostgreSQL backend configuration
    - Federation support with .well-known
    - TURN server for voice/video
    - Bridge to Slack/Discord
  port_requirements:
    - Main server: 8008
    - Federation: 8448
    - TURN: 3478
```

### Example: Creating Security Scanner
```yaml
# queue/pending/002-create-zap.yaml
id: create-owasp-zap-20250103
title: "Create OWASP ZAP security scanner resource"
type: security
priority: medium
requirements:
  base_template: monitoring
  specific_needs:
    - API mode configuration
    - Scan policy templates
    - Report generation
    - CI/CD integration
```

## Development

### Adding New Templates
```bash
# Create new template
mkdir -p scripts/resources/templates/new-category
cp -r scripts/resources/templates/basic/* scripts/resources/templates/new-category/
# Customize template files
```

### Testing Generated Resources
```bash
# Run contract validation
resource-generator validate resources/new-resource

# Test integration
vrooli resource new-resource test
```

## Troubleshooting

### Port Allocation Issues
```bash
# Check port availability
resource-generator ports --check 28000

# Reset port registry
resource-generator ports --reset-range 40000-49999
```

### Generation Failures
- Check logs in `queue/failed/`
- Verify template exists
- Ensure dependencies are available
- Check Qdrant memory is accessible

### Contract Validation Errors
- Review v2.0 requirements
- Check health check implementation
- Verify lifecycle hooks
- Ensure CLI registration

## Success Metrics

Tracks for each resource:
- Time to generation
- Contract compliance score
- Test pass rate
- Documentation completeness
- Adoption by scenarios

## Related Documentation

- [v2.0 Resource Contract](/docs/resources/v2.0-contract.md)
- [Port Management](/scripts/resources/port-registry.sh)
- [Template Guide](/scripts/resources/templates/README.md)
- [Migration Plan](/auto/docs/MIGRATION_PLAN.md)