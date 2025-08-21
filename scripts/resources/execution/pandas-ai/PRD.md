# Product Requirements Document (PRD) - Pandas AI

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
Pandas AI provides conversational AI-powered data analysis and manipulation infrastructure, enabling natural language queries over structured data. It transforms data analysis from code-writing to conversational interaction, making data insights accessible to non-technical users while maintaining the full power of pandas for advanced users.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Natural Language Data Analysis** - Scenarios can query and analyze data using plain English instead of code
- **Automated Insight Generation** - Automatically generates visualizations and statistical summaries from data
- **Cross-Resource Data Integration** - Seamlessly connects to PostgreSQL, Redis, MongoDB for unified analysis
- **Report Generation** - Creates professional data reports and dashboards without manual coding
- **Data Pipeline Simplification** - Reduces complex ETL pipelines to conversational commands

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Business Intelligence**: Automated report generation, KPI tracking, executive dashboards
2. **Data Science Workflows**: Exploratory data analysis, feature engineering, model preparation
3. **Customer Analytics**: Behavior analysis, segmentation, churn prediction with natural language
4. **Financial Analysis**: Portfolio analysis, risk assessment, automated financial reporting
5. **Healthcare Analytics**: Patient data analysis, outcome tracking, medical research support

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Natural language to pandas code generation
  - [x] Support for CSV, Excel, JSON data sources
  - [x] Database connectivity (PostgreSQL, Redis, MongoDB)
  - [x] Integration with Vrooli resource framework
  - [x] Standard CLI interface (resource-pandas-ai)
  - [ ] Health monitoring and status reporting
  - [ ] Docker containerization and networking
  
- **Should Have (P1)**
  - [ ] Visualization generation (matplotlib, seaborn, plotly)
  - [ ] Data cleaning and preparation suggestions
  - [ ] Multi-dataframe operations and joins
  - [ ] Performance optimization for large datasets
  - [ ] Advanced configuration options
  
- **Nice to Have (P2)**
  - [ ] Machine learning model suggestions
  - [ ] Real-time data streaming analysis
  - [ ] Integration with Jupyter notebooks

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Service Startup Time | < 10s | Container initialization |
| Health Check Response | < 500ms | API/CLI status checks |
| Resource Utilization | < 30% CPU/Memory | Resource monitoring |
| Availability | > 99% uptime | Service monitoring |
| Query Response Time | < 5s for typical queries | API response timing |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [ ] Integration tests pass with all dependent resources
- [ ] Performance targets met under expected load
- [ ] Security standards met for resource type
- [ ] Documentation complete and accurate

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: vault
    purpose: Secure storage of API keys for AI models
    integration_pattern: Read credentials at startup
    access_method: CLI
    
  - resource_name: postgres
    purpose: Primary data source for analysis
    integration_pattern: Direct SQL queries via psycopg2
    access_method: Direct connection
    
optional:
  - resource_name: openrouter
    purpose: Enhanced AI capabilities for complex analysis
    fallback: Use local models or basic pandas operations
    access_method: API
    
  - resource_name: redis
    purpose: Caching analysis results
    fallback: No caching, regenerate on each request
    access_method: Direct connection
    
  - resource_name: minio
    purpose: Storage for large datasets and reports
    fallback: Local filesystem storage
    access_method: S3 API
```

### Integration Standards
```yaml
resource_category: execution

standard_interfaces:
  management:
    - cli: cli.sh (using CLI framework)
    - actions: [help, install, uninstall, start, stop, restart, status, validate, test, content]
    - configuration: config/defaults.sh
    - documentation: README.md + docs/
    
  networking:
    - docker_networks: [vrooli-network]
    - port_registry: Port 8095 defined in scripts/resources/port_registry.sh
    - hostname: vrooli-pandas-ai
    
  monitoring:
    - health_check: GET /health endpoint
    - status_reporting: resource-pandas-ai status (uses status-args.sh framework)
    - logging: Application logs in data/pandas-ai/pandas-ai.log
    
  data_persistence:
    - volumes: [data/pandas-ai/scripts, data/pandas-ai/datasets]
    - backup_strategy: Script and dataset versioning
    - migration_support: Python package version management

integration_patterns:
  scenarios_using_resource:
    - scenario_name: business-intelligence-dashboard
      usage_pattern: Generate reports and visualizations from business data
      
    - scenario_name: customer-analytics
      usage_pattern: Analyze customer behavior and generate insights
      
  resource_to_resource:
    - postgres → pandas-ai: Data source for analysis
    - pandas-ai → n8n: Trigger workflows based on analysis results
    - pandas-ai → minio: Store generated reports and visualizations
```

### Configuration Schema
```yaml
resource_configuration:
  defaults:
    enabled: true
    port: 8095 (from port_registry.sh)
    networks: [vrooli-network]
    volumes: [data/pandas-ai]
    environment: [PANDAS_AI_PORT, OPENAI_API_KEY]
    
  templates:
    development:
      - description: Development environment with verbose logging
      - overrides: LOG_LEVEL=DEBUG, MAX_WORKERS=2
      
    production:
      - description: Production with optimized performance
      - overrides: LOG_LEVEL=INFO, MAX_WORKERS=8, CACHE_ENABLED=true
      
    testing:
      - description: Test environment with mock data
      - overrides: USE_MOCK_DATA=true, LOG_LEVEL=DEBUG
      
customization:
  user_configurable:
    - parameter: MAX_ROWS
      description: Maximum rows to process in a single query
      default: 100000
      
    - parameter: AI_MODEL
      description: AI model to use for analysis
      default: gpt-3.5-turbo
      
  environment_variables:
    - var: PANDAS_AI_PORT
      purpose: Port for API service
      
    - var: OPENAI_API_KEY
      purpose: API key for AI model access
```

### API Contract
```yaml
api_endpoints:
  - method: GET
    path: /health
    purpose: Health check endpoint
    output_schema: |
      {
        "status": "healthy",
        "version": "1.0.0",
        "uptime": 12345
      }
    
  - method: POST
    path: /analyze
    purpose: Analyze data with natural language query
    input_schema: |
      {
        "query": "string",
        "data_source": "string",
        "options": {}
      }
    output_schema: |
      {
        "result": "object",
        "visualization": "string (base64)",
        "code": "string"
      }
    authentication: Bearer token
    rate_limiting: 100 requests per minute
```

## 🖥️ Management Interface Contract

### Required Management Actions
```yaml
standard_actions:
  - name: install
    description: Install and configure Pandas AI
    flags: [--force, --template <name>]
    
  - name: start  
    description: Start the Pandas AI service
    flags: [--wait]
    
  - name: stop
    description: Stop the Pandas AI service gracefully
    flags: [--force]
    
  - name: status
    description: Show detailed Pandas AI status
    flags: [--json, --verbose]
    
  - name: uninstall
    description: Remove Pandas AI and cleanup
    flags: [--keep-data, --force]

resource_specific_actions:
  - name: analyze
    description: Run analysis on data with natural language
    flags: [--query, --source, --output]
    example: resource-pandas-ai analyze --query "show top 10 customers" --source customers.csv
    
  - name: content
    description: Manage analysis scripts and datasets
    flags: [add, list, get, remove, execute]
    example: resource-pandas-ai content add --file analysis.py
```

### Management Standards
```yaml
implementation_requirements:
  - cli_location: cli.sh (uses CLI framework)
  - configuration: config/defaults.sh
  - dependencies: lib/ directory with modular functions
  - error_handling: Exit codes (0=success, 1=error, 2=config error)
  - logging: Structured output with levels (INFO, WARN, ERROR)
  - idempotency: Safe to run commands multiple times
  
status_reporting:
  - health_status: healthy|degraded|unhealthy|unknown
  - service_info: version, uptime, resource usage
  - integration_status: database connectivity
  - configuration: current settings and AI model
  
output_formats:
  - default: Human-readable with color coding
  - json: Structured data with --json flag
  - verbose: Detailed diagnostics with --verbose
```

## 🔧 Operational Requirements

### Deployment Standards
```yaml
containerization:
  base_image: python:3.11-slim
  dockerfile_location: docker/Dockerfile
  build_requirements: Python packages installation
  
networking:
  required_networks:
    - vrooli-network: Primary inter-resource communication
    
  port_allocation:
    - internal: 8095
    - external: 8095 (from port_registry.sh)
    - protocol: tcp
    - purpose: API service endpoint
    - registry_integration:
        definition: Port defined in scripts/resources/port_registry.sh
        retrieval: Use port_registry functions to get assigned port
        no_hardcoding: Never hardcode ports in resource code
    
data_management:
  persistence:
    - volume: data/pandas-ai/scripts
      mount: /app/scripts
      purpose: Analysis scripts and injected code
      
    - volume: data/pandas-ai/datasets
      mount: /app/datasets
      purpose: Dataset storage for analysis
      
  backup_strategy:
    - method: Filesystem backup of scripts and datasets
    - frequency: Daily
    - retention: 30 days
    
  migration_support:
    - version_compatibility: Python 3.8+
    - upgrade_path: pip package updates
    - rollback_support: Virtual environment snapshots
```

### Performance Standards
```yaml
resource_requirements:
  minimum:
    cpu: 2 cores
    memory: 2GB RAM
    disk: 5GB
    
  recommended:
    cpu: 4 cores
    memory: 8GB RAM
    disk: 20GB
    
  scaling:
    horizontal: Multiple instances for different datasets
    vertical: More RAM for larger datasets
    limits: 64GB RAM for in-memory operations
    
monitoring_requirements:
  health_checks:
    - endpoint: GET /health
    - interval: 30s
    - timeout: 5s
    - failure_threshold: 3
    
  metrics:
    - metric: query_latency
      collection: API response times
      alerting: > 10s average
      
    - metric: memory_usage
      collection: Process memory monitoring
      alerting: > 80% of limit
```

### Security Standards
```yaml
security_requirements:
  authentication:
    - method: API key or Bearer token
    - credential_storage: Vault integration
    - session_management: Token expiration
    
  authorization:
    - access_control: Dataset-level permissions
    - role_based: analyst, admin roles
    - resource_isolation: User-specific workspaces
    
  data_protection:
    - encryption_at_rest: Optional for sensitive datasets
    - encryption_in_transit: HTTPS for API
    - key_management: Vault integration
    
  network_security:
    - port_exposure: Only API port 8095
    - firewall_requirements: Restrict to vrooli-network
    - ssl_tls: Optional TLS termination
    
compliance:
  standards: GDPR for personal data analysis
  auditing: Query and access logging
  data_retention: Configurable per dataset
```

## 🧪 Testing Strategy

### Test Categories
```yaml
unit_tests:
  location: Co-located with source files (lib/lifecycle.bats, lib/status.bats)
  coverage: Individual function testing
  framework: BATS (Bash Automated Testing System)
  
integration_tests:
  location: test/ directory
  coverage: Database connectivity, API endpoints, content management
  test_data: Uses shared fixtures from scripts/__test/fixtures/data/
  test_scenarios: 
    - Resource lifecycle (install, start, stop, uninstall)
    - Content management operations
    - Data analysis queries
    - Health checks and status reporting
  
system_tests:
  location: scripts/__test/resources/
  coverage: End-to-end analysis workflows
  automation: Integrated with Vrooli test framework
  
performance_tests:
  load_testing: Concurrent query handling
  stress_testing: Large dataset processing
  endurance_testing: Memory leak detection
```

### Test Specifications
```yaml
test_specification:
  resource_name: pandas-ai
  test_categories: [unit, integration, system, performance]
  
  test_structure:
    - BATS files co-located with source files
    - Integration tests in test/ directory
    - Shared fixtures from scripts/__test/fixtures/data/
    - Test results included in status output with timestamp
    - Examples in examples/ directory
  
  lifecycle_tests:
    - name: "Resource Installation"
      command: resource-pandas-ai install
      expect:
        exit_code: 0
        service_running: false
        packages_installed: true
        
    - name: "Resource Start"
      command: resource-pandas-ai start
      expect:
        exit_code: 0
        service_running: true
        health_status: healthy
        
    - name: "Content Management"
      command: resource-pandas-ai content add --file test_analysis.py
      fixture: scripts/__test/fixtures/data/python/analysis.py
      expect:
        exit_code: 0
        content_stored: true
        
  performance_tests:
    - name: "Query Response Time"
      measurement: api_latency
      target: < 5 seconds for 100k rows
      
    - name: "Memory Usage"
      measurement: memory
      target: < 2GB for typical queries
      
  failure_tests:
    - name: "Database Connection Loss"
      scenario: PostgreSQL becomes unavailable
      expect: Graceful fallback to file-based analysis
```

## 💰 Infrastructure Value

### Technical Value
- **Infrastructure Stability**: Provides reliable data analysis service with fallback options
- **Integration Simplicity**: Natural language interface reduces complexity for scenarios
- **Operational Efficiency**: Automates report generation and data exploration
- **Developer Experience**: No-code data analysis for non-technical users

### Resource Economics
- **Setup Cost**: 5 minutes installation, minimal configuration
- **Operating Cost**: 2GB RAM baseline, scales with dataset size
- **Integration Value**: Multiplies value of data storage resources
- **Maintenance Overhead**: Python package updates quarterly

## 🔄 Resource Lifecycle Integration

### Vrooli Integration Standards
```yaml
resource_discovery:
  registry_entry:
    name: pandas-ai
    category: execution
    capabilities: [data-analysis, visualization, natural-language]
    interfaces:
      - cli: resource-pandas-ai (installed via install-resource-cli.sh)
      - api: http://localhost:8095
      - health: GET /health
      
  metadata:
    description: Conversational AI for data analysis
    version: 1.0.0
    dependencies: [python3, pandas, openai]
    enables: [business-intelligence, data-science, analytics]

resource_framework_compliance:
  - Standard directory structure (/config, /lib, /docs, /test, /examples)
  - CLI framework integration (cli.sh as thin wrapper)
  - Port registry integration
  - Docker network integration
  - Health monitoring integration
  - Configuration management standards
  
deployment_integration:
  supported_targets:
    - local: Python virtual environment
    - docker: Container deployment
    - kubernetes: StatefulSet with persistent storage
    
  configuration_management:
    - Environment-based configuration
    - Template-based setup
    - Secret management via Vault
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  compatibility: Python 3.8+ required
  upgrade_path: pip install --upgrade pandasai
  
  breaking_changes: []
  deprecations: []
  migration_guide: Not required for initial version

release_management:
  release_cycle: Follow pandasai package releases
  testing_requirements: Integration tests before upgrade
  rollback_strategy: Virtual environment snapshots
```

## 🧬 Evolution Path

### Version 1.0.0 (Current)
- Basic natural language to pandas conversion
- PostgreSQL, Redis, MongoDB connectivity
- Simple visualization generation
- CLI and API interfaces

### Version 2.0.0 (Planned)
- Advanced ML model suggestions
- Real-time streaming data analysis
- Jupyter notebook integration
- Multi-user workspace support

### Long-term Vision
- Autonomous data pipeline generation
- Predictive analytics automation
- Cross-resource data federation
- AI-driven data quality improvement

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| API key exhaustion | Medium | High | Rate limiting and caching |
| Memory overflow with large datasets | Medium | Medium | Chunked processing |
| AI hallucination in code generation | Low | High | Code validation before execution |
| Database connection loss | Low | Medium | Fallback to file-based analysis |

### Operational Risks
- **Configuration Drift**: Version lock Python packages
- **Dependency Failures**: Fallback to basic pandas operations
- **Resource Conflicts**: Isolated virtual environment
- **Update Compatibility**: Test before production updates

## ✅ Validation Criteria

### Infrastructure Validation
- [x] Resource installs and starts successfully
- [ ] All management actions work correctly
- [ ] Integration with other resources functions properly
- [ ] Performance meets established targets
- [ ] Security requirements satisfied
- [ ] Documentation complete and accurate

### Integration Validation  
- [ ] Successfully enables dependent scenarios
- [x] Integrates properly with Vrooli resource framework
- [ ] Networking and discovery work correctly
- [ ] Configuration management functions properly
- [ ] Monitoring and alerting work as expected

### Operational Validation
- [ ] Deployment procedures documented and tested
- [ ] Backup and recovery procedures verified
- [ ] Upgrade and rollback procedures validated
- [ ] Troubleshooting documentation complete
- [ ] Performance under load verified

## 📝 Implementation Notes

### Design Decisions
**Virtual Environment vs System Python**: Use venv when possible, fallback to system
- Alternative considered: Docker-only deployment
- Decision driver: Flexibility for different environments
- Trade-offs: More complex but more adaptable

**API Service vs CLI-only**: Provide both interfaces
- Alternative considered: CLI-only interface
- Decision driver: Support both interactive and programmatic use
- Trade-offs: More code but better integration options

### Known Limitations
- **Python 3.12 venv issue**: System missing python3.12-venv package
  - Workaround: Use system Python with --user flag
  - Future fix: Include in Docker image
  
- **Large dataset processing**: Memory constraints
  - Workaround: Chunk processing
  - Future fix: Streaming analysis support

### Integration Considerations
- **Database Credentials**: Obtain from Vault or environment
- **AI Model Selection**: Configure based on available providers
- **Performance Impact**: Monitor memory usage with large datasets
- **Scaling Considerations**: Horizontal scaling for multiple datasets

## 🔗 References

### Documentation
- README.md - Quick start and overview
- docs/usage.md - Detailed usage examples
- config/defaults.sh - Configuration options
- lib/*.sh - Implementation modules

### Related Resources
- postgres - Primary data source
- openrouter - AI model provider
- n8n - Workflow automation integration

### External Resources
- [PandasAI Documentation](https://docs.pandas-ai.com)
- [Pandas Documentation](https://pandas.pydata.org/docs/)
- [OpenAI API Reference](https://platform.openai.com/docs)

---

**Last Updated**: 2025-08-21
**Status**: Draft
**Owner**: Vrooli Resource Team
**Review Cycle**: Quarterly