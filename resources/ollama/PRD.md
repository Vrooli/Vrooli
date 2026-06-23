# Product Requirements Document (PRD) - Resource

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
Local Large Language Model (LLM) inference and management infrastructure that provides privacy-first AI capabilities without external dependencies. Ollama enables scenarios to leverage advanced AI reasoning, text generation, code assistance, and embedding creation while maintaining complete data sovereignty and cost predictability.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- Provides AI intelligence that transforms every scenario from manual tools into intelligent assistants
- Enables privacy-sensitive AI processing without external API dependencies or costs
- Creates consistent AI reasoning patterns that scenarios can rely on for decision-making
- Establishes local embedding generation for semantic search and knowledge organization
- Offers specialized model customization through Modelfiles for domain-specific intelligence

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **AI-Powered Analysis Scenarios**: Document intelligence, research assistance, content analysis
2. **Code Generation & Review**: Programming assistants, code analysis, technical documentation
3. **Creative Content Scenarios**: Writing assistants, idea generation, content transformation  
4. **Knowledge Management**: Semantic search, information extraction, concept mapping
5. **Business Intelligence**: Report generation, data analysis, decision support systems

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Local LLM inference with multiple model support (50+ models)
  - [x] RESTful API for programmatic access
  - [x] Model management (download, list, remove)
  - [x] Streaming response support for real-time interaction
  - [x] Integration with Vrooli resource framework
  - [x] Standard CLI interface (resource-ollama)
  - [x] System service installation with GPU support
  
- **Should Have (P1)**
  - [x] Custom model creation via Modelfiles
  - [x] Automatic model selection based on task type
  - [x] GPU acceleration for performance
  - [x] Embedding generation for vector operations
  - [x] Temperature and parameter control
  
- **Nice to Have (P2)**
  - [ ] Multi-model ensemble inference
  - [ ] Fine-tuning capabilities
  - [ ] Integration with external model repositories

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Service Startup Time | < 30s | Docker container startup |
| Health Check Response | < 100ms | API endpoint response |
| Model Loading Time | < 60s for 8B models | Ollama model loading |
| Inference Response | < 2s for short prompts | API response time |
| Resource Utilization | < 80% available RAM | Memory monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration tests pass with scenarios using Ollama
- [x] Performance targets met with default models
- [x] Model customization works reliably
- [x] GPU acceleration functional when available

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: docker
    purpose: Runs the Ollama daemon as a managed container
    integration_pattern: docker-service driver (internal/resources/drivers.go)
    access_method: vrooli resource {install,start,stop,status} ollama
    
  - resource_name: nvidia-drivers
    purpose: GPU acceleration support (when GPU available)
    integration_pattern: NVIDIA driver integration
    access_method: CUDA runtime environment
    
optional:
  - resource_name: cuda
    purpose: Enhanced GPU performance
    fallback: CPU-only inference
    access_method: NVIDIA CUDA toolkit
```

### Integration Standards
```yaml
resource_category: ai

standard_interfaces:
  management:
    - lifecycle: vrooli resource {install,start,stop,restart,status,logs} ollama (docker-service driver)
    - resource_cli: resource-ollama {ensure,gateway,capacity,policy} for model + inference workflows
    - configuration: resource.json (runtime image, env, ports, memory cap, health checks)
    - documentation: README.md + docs/
    
  networking:
    - port: 11434 (declared in resource.json ports[])
    - hostname: localhost
    - protocol: http
    
  monitoring:
    - health_check: http://127.0.0.1:11434/api/tags (declared in resource.json health_checks[])
    - status_reporting: vrooli resource status ollama
    - logging: docker logs ollama (container stdout/stderr)
    
  data_persistence:
    - storage: ${RESOURCE_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/vrooli/resources/ollama}
    - backup_strategy: Model files and configuration
    - migration_support: Model version management

integration_patterns:
  scenarios_using_resource:
    - scenario_name: research-assistant
      usage_pattern: Document analysis and knowledge extraction
      
    - scenario_name: idea-generator
      usage_pattern: Creative content generation and refinement
      
    - scenario_name: stream-of-consciousness-analyzer
      usage_pattern: Thought pattern analysis and insight extraction
      
    - scenario_name: study-buddy
      usage_pattern: Educational content generation and explanations
      
  resource_to_resource:
    - qdrant → ollama: Vector embeddings generation
    - n8n → ollama: Workflow automation with AI processing
    - postgres → ollama: Database content analysis and insights
```

### Configuration Schema
```yaml
resource_configuration:
  defaults:
    enabled: true
    port: 11434
    container_name: ollama
    image: ollama/ollama:0.30.10
    memory_limit: 12g
    environment:
      OLLAMA_HOST: "0.0.0.0:11434"
      OLLAMA_ORIGINS: "*"
    
  templates:
    development:
      - description: CPU-optimized for development
      - overrides:
          OLLAMA_NUM_PARALLEL: 1
          OLLAMA_MAX_LOADED_MODELS: 2
      
    production:
      - description: GPU-optimized for production
      - overrides:
          OLLAMA_NUM_PARALLEL: 4
          OLLAMA_MAX_LOADED_MODELS: 4
          NVIDIA_VISIBLE_DEVICES: all
      
    testing:
      - description: Minimal resource usage for testing
      - overrides:
          OLLAMA_NUM_PARALLEL: 1
          OLLAMA_MAX_LOADED_MODELS: 1
      
customization:
  user_configurable:
    - parameter: default_models
      description: >-
        Models to install automatically. Governed by the role policy in
        model-policy.json (resolved primaries for the embedding/chat/code
        roles), not a hard-coded list — see `resource-ollama policy roles`.
      
    - parameter: gpu_enabled
      description: Enable GPU acceleration
      default: auto-detect
      
  environment_variables:
    - var: OLLAMA_NUM_PARALLEL
      purpose: Number of parallel request processing
      
    - var: OLLAMA_MAX_LOADED_MODELS
      purpose: Maximum models kept in memory

```

### Host Stability Protections

Ollama runs as a Docker container with a hard memory ceiling declared in
`resource.json` (`runtime.memory_limit: 12g`, applied as `docker run --memory`).
The intent is that any caller — including agents that hit `localhost:11434`
directly — runs against a bounded ollama: under genuine memory pressure Docker
OOM-kills the container rather than letting the kernel hang. An embeddings-burst
pattern from such callers was a candidate cause of host hard-resets in the
2026-05-07 incident; the container memory cap is the Docker equivalent of the
cgroup `MemoryMax` limit the old host-systemd unit used to render. To adjust the
ceiling, edit `runtime.memory_limit` and `vrooli resource restart ollama`.

### API Contract
```yaml
api_endpoints:
  - method: GET
    path: /api/tags
    purpose: List available models
    output_schema: |
      {
        "models": [{
          "name": "string",
          "size": "integer",
          "digest": "string",
          "modified_at": "timestamp"
        }]
      }
    authentication: none
    rate_limiting: none
    
  - method: POST
    path: /api/generate
    purpose: Generate text completion
    input_schema: |
      {
        "model": "string",
        "prompt": "string", 
        "stream": "boolean",
        "options": {
          "temperature": "float",
          "top_p": "float",
          "top_k": "integer"
        }
      }
    output_schema: |
      {
        "model": "string",
        "response": "string",
        "done": "boolean",
        "context": "array",
        "total_duration": "integer",
        "load_duration": "integer"
      }
    authentication: none
    rate_limiting: concurrent request limits
    
  - method: POST
    path: /api/embeddings
    purpose: Generate text embeddings
    input_schema: |
      {
        "model": "string",
        "prompt": "string"
      }
    output_schema: |
      {
        "embedding": "float[]"
      }
    authentication: none
    rate_limiting: none
    
  - method: POST
    path: /api/pull
    purpose: Download model
    input_schema: |
      {
        "name": "string",
        "stream": "boolean"
      }
    output_schema: |
      {
        "status": "string",
        "digest": "string",
        "total": "integer",
        "completed": "integer"
      }
    authentication: none
    rate_limiting: bandwidth limits
```

## 🖥️ Management Interface Contract

### Required Management Actions
```yaml
standard_actions:
  - name: install
    description: Install Ollama and default models
    flags: [--gpu, --cpu-only, --models <list>]
    
  - name: start  
    description: Start Ollama service
    flags: [--wait, --gpu]
    
  - name: stop
    description: Stop Ollama service gracefully
    flags: [--force]
    
  - name: status
    description: Show Ollama service status and model info
    flags: [--json, --verbose]
    
  - name: uninstall
    description: Remove Ollama and cleanup models
    flags: [--keep-models, --force]

resource_specific_actions:
  - name: models
    description: List available and installed models
    flags: [--available, --installed]
    example: resource-ollama content list --type models
    
  - name: pull
    description: Download specific models
    flags: [--models <model_list>]
    example: resource-ollama pull --models "llama3.1:70b,codellama:13b"
    
  - name: prompt
    description: Send prompt to Ollama for testing
    flags: [--text <prompt>, --model <model>, --type <task_type>]
    example: resource-ollama content execute --prompt "Explain AI" --model llama3.1:8b
    
  - name: modelfile
    description: Create custom model from Modelfile
    flags: [--file <path>, --name <model_name>]
    example: resource-ollama content add --modelfile custom.modelfile --name my-specialist
```

### Management Standards
```yaml
implementation_requirements:
  - lifecycle: vrooli resource ... (docker-service driver)
  - resource_cli: resource-ollama (Go module under cli/)
  - configuration: resource.json, model-policy.json, config/capabilities.yaml
  - error_handling: Exit codes (0=success, 1=error, 2=model error, 3=GPU error)
  - logging: Structured output with levels (INFO, WARN, ERROR)
  - idempotency: Safe to run commands multiple times
  
status_reporting:
  - framework: vrooli resource status ollama (docker-service driver)
  - health_status: healthy|degraded|unhealthy|unknown
  - service_info: version, uptime, GPU status, memory usage
  - model_info: installed models, sizes, last used
  - performance: inference times, queue status
  
output_formats:
  - default: Human-readable with model summaries
  - json: Structured data with --json flag
  - verbose: Detailed model info and performance stats
  - fast: Basic health check with --fast flag
```

### Content Management Implementation
```yaml
content_storage:
  purpose: Manage AI models, custom configurations, and prompt templates
  
  implementation_patterns:
    - Model storage: ${RESOURCE_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/vrooli/resources/ollama}/models
    - Custom models: Created via Modelfiles
    - Prompt templates: Stored for reuse across scenarios
    - Metadata tracking: Model sizes, creation dates, usage stats
    
  content_types:
    models:
      - type: Base models from Ollama registry
      - operations: pull, list, remove, execute
      - sharing: Available to all scenarios
      
    custom_models:
      - type: Modelfiles for specialized models
      - operations: add, list, execute, remove
      - sharing: Named models accessible globally
      
    prompt_templates:
      - type: Reusable prompt configurations
      - operations: add, list, execute
      - sharing: Templates for consistent scenario interactions
```

## 🔧 Operational Requirements

### Deployment Standards
```yaml
installation:
  method: Docker image pull (ollama/ollama:<pin>) via the docker-service driver
  runtime: Docker container named "ollama"
  service_management: vrooli resource {start,stop,restart,status} ollama
  model_storage: ${RESOURCE_DATA_DIR} bind-mounted to /root/.ollama
  
networking:
  port_allocation:
    - internal: [port from registry]
    - external: [port from registry]
    - protocol: tcp
    - purpose: HTTP API for inference and management
    
data_management:
  persistence:
    - storage: ${RESOURCE_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/vrooli/resources/ollama}
      purpose: Model files and configuration storage
      
  backup_strategy:
    - method: Model files and configuration backup
    - frequency: Before model updates
    - retention: 3 model versions
    
  migration_support:
    - version_compatibility: Ollama version migrations
    - upgrade_path: Model format upgrades
    - rollback_support: Model version rollback
```

### Performance Standards
```yaml
resource_requirements:
  minimum:
    cpu: 2 cores
    memory: 8GB RAM
    disk: 10GB (base models)
    
  recommended:
    cpu: 4+ cores
    memory: 16GB+ RAM
    disk: 50GB+ (multiple models)
    gpu: NVIDIA with 8GB+ VRAM
    
  scaling:
    horizontal: Single instance per node
    vertical: Scales with RAM and GPU memory
    limits: Model size limited by available memory
    
monitoring_requirements:
  health_checks:
    - endpoint: http://localhost:$(port from registry)/api/tags
    - interval: 30s
    - timeout: 5s
    - failure_threshold: 3 consecutive failures
    
  metrics:
    - metric: inference_latency
      collection: API response time measurement
      alerting: > 10s response time
      
    - metric: model_load_time
      collection: Model loading duration
      alerting: > 120s load time
      
    - metric: memory_usage
      collection: Process memory monitoring
      alerting: > 90% memory utilization
```

### Security Standards
```yaml
security_requirements:
  authentication:
    - method: Network-based access control
    - credential_storage: Not applicable (open access on localhost)
    - session_management: Stateless API
    
  authorization:
    - access_control: Localhost-only access by default
    - role_based: Not applicable
    - resource_isolation: Process-based isolation
    
  data_protection:
    - encryption_at_rest: Model files stored unencrypted (performance)
    - encryption_in_transit: HTTP (localhost only)
    - key_management: Not applicable
    
  network_security:
    - port_exposure: Localhost only by default
    - firewall_requirements: Block external access to Ollama port
    - ssl_tls: Not required for localhost communication
    
compliance:
  standards: Data sovereignty (local processing)
  auditing: Container logs (docker logs ollama) and API access logs
  data_retention: Models and prompts not logged by default
```

## 🧪 Testing Strategy

### Test Categories
```yaml
unit_tests:
  location: Co-located with Go source (cli/ and cli/internal/*/*_test.go)
  coverage: Model resolution/ensure, gateway, capacity, policy logic
  framework: Go testing (go test ./...)
  
integration_tests:
  location: internal/resources (docker-service driver, incl. port-conflict preflight)
  coverage: Driver argument construction, host-port preflight, lifecycle behavior
  test_scenarios: 
    - docker run argument assembly (memory cap, ports, volumes)
    - host-port conflict preflight (occupied port → actionable failure)
    - model resolution via model-policy.json
  
system_tests:
  location: vrooli resource {install,start,status,stop} ollama against the live container
  coverage: Full Ollama lifecycle, health, failure scenarios
  automation: Integrated with Vrooli resource control plane
  
performance_tests:
  load_testing: Concurrent inference requests
  stress_testing: Memory limits with large models
  endurance_testing: 24-hour continuous inference
```

### Test Specifications
```yaml
test_specification:
  resource_name: ollama
  test_categories: [unit, integration, system, performance]
  
  test_structure:
    - BATS files co-located with source files
    - Integration tests in test/ directory
    - Shared fixtures from __test/fixtures/data/
    - Test results included in status output with timestamp
    - Usage examples documented in README.md or docs/
  
  lifecycle_tests:
    - name: "Ollama Installation"
      command: resource-ollama install
      expect:
        exit_code: 0
        service_running: true
        health_status: healthy
        
    - name: "Resource Status"  
      command: resource-ollama status --json
      expect:
        exit_code: 0
        json_valid: true
        health_status: healthy
        test_results: includes last test timestamp
        
    - name: "Content Management - Add Model"
      command: resource-ollama content add --modelfile custom.modelfile --name test-model
      fixture: __test/fixtures/data/documents/llm-prompt.json
      expect:
        exit_code: 0
        model_created: true
        
    - name: "Content Management - List"
      command: resource-ollama content list --type models
      expect:
        exit_code: 0
        output_contains: ["test-model"]
        
    - name: "Content Management - Execute"
      command: resource-ollama content execute --prompt "Hello" --model test-model
      expect:
        exit_code: 0
        response_generated: true
        
    - name: "API Functionality"
      command: curl -s http://localhost:$(port from registry)/api/tags
      expect:
        exit_code: 0
        json_response: true
        models_listed: true
        
  performance_tests:
    - name: "Inference Latency"
      measurement: api_response_time
      target: < 2s for short prompts
      
    - name: "Model Loading Time"
      measurement: model_load_duration
      target: < 60s for 8B models
      
  failure_tests:
    - name: "OOM Recovery"
      scenario: Out of memory during model loading
      expect: Graceful failure with error message
```

## 💰 Infrastructure Value

### Technical Value
- **AI Democratization**: Makes advanced AI accessible to all scenarios without API costs
- **Privacy Preservation**: Enables sensitive data processing without external exposure
- **Cost Predictability**: Eliminates per-token costs and usage-based pricing
- **Performance Consistency**: Local processing removes network latency and external dependencies

### Resource Economics
- **Setup Cost**: 30-60 minutes for installation and model downloads
- **Operating Cost**: Hardware resources (8-16GB RAM, optional GPU)
- **Integration Value**: Transforms every scenario from static tools to intelligent systems
- **Maintenance Overhead**: Model updates and performance monitoring

## 🔄 Resource Lifecycle Integration

### Vrooli Integration Standards
```yaml
resource_discovery:
  registry_entry:
    name: ollama
    category: ai
    capabilities: 
      - Local LLM inference
      - Model management
      - Custom model creation
      - Embedding generation
      - Privacy-first AI processing
    interfaces:
      - lifecycle: vrooli resource ... ollama (docker-service driver)
      - cli: resource-ollama (Go module, built/installed via cli/install.sh)
      - api: http://localhost:11434/api
      - health: http://localhost:11434/api/tags
      
  metadata:
    description: "Local LLM inference with privacy-first AI capabilities"
    version: "latest"
    dependencies: []
    enables: [research-assistant, idea-generator, stream-of-consciousness-analyzer, study-buddy]

resource_framework_compliance:
  - docker-service structure (resource.json + cli/ Go module + config/ + docs/)
  - Lifecycle via the shared vrooli resource control plane (no bash wrapper)
  - Ports + health checks declared in resource.json
  - Health monitoring integration
  - Configuration management standards
  
deployment_integration:
  supported_targets:
    - local: Docker container with GPU support
    - kubernetes: Container workload with GPU node affinity
    - cloud: GPU-enabled cloud instances
    
  configuration_management:
    - Environment-based configuration
    - Model template management
    - GPU/CPU mode switching
```

### Version Management
```yaml
versioning:
  current: "latest"
  compatibility: Backward compatible API
  upgrade_path: Model format migrations handled automatically
  
  breaking_changes: []
  deprecations: []
  migration_guide: Model updates handled by Ollama service

release_management:
  release_cycle: Follows Ollama upstream releases
  testing_requirements: Model compatibility and API functionality
  rollback_strategy: Service rollback with model version management
```

## 🧬 Evolution Path

### Version Latest (Current)
- Local LLM inference with 50+ supported models
- RESTful API for programmatic access
- Custom model creation via Modelfiles
- GPU acceleration support
- Integration with Vrooli resource framework

### Version Next (Planned)
- Enhanced model management with automatic updates
- Multi-model ensemble inference capabilities
- Advanced performance monitoring and optimization
- Integration with additional vector databases
- Fine-tuning capabilities for specialized models

### Long-term Vision
- Federated learning across Vrooli instances
- Automated model selection based on task requirements
- Advanced caching and optimization strategies
- Integration with edge computing deployments

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Model loading failures | Low | High | Model validation, retry logic, fallback models |
| Out of memory errors | Medium | High | Memory monitoring, model size limits, graceful degradation |
| GPU compatibility issues | Low | Medium | CPU fallback, GPU detection, driver validation |
| Performance degradation | Medium | Medium | Performance monitoring, resource limits, model optimization |

### Operational Risks
- **Model Storage Growth**: Implement model cleanup and size limits
- **Resource Exhaustion**: Monitor memory usage and implement safeguards
- **Network Connectivity**: Ensure reliable model download during setup
- **Version Compatibility**: Test model compatibility across Ollama versions

## ✅ Validation Criteria

### Infrastructure Validation
- [x] Ollama installs and starts successfully
- [x] Default models download and load correctly
- [x] API endpoints respond appropriately
- [x] Performance meets established targets
- [x] GPU acceleration works when available
- [x] Management interface functions properly

### Integration Validation  
- [x] Successfully enables AI capabilities in scenarios
- [x] Integrates properly with Vrooli resource framework
- [x] Network connectivity established correctly
- [x] Health monitoring functions properly
- [x] Model customization works reliably

### Operational Validation
- [x] Deployment procedures documented and tested
- [x] Model backup and management procedures verified
- [x] Performance monitoring and alerting functional
- [x] Troubleshooting documentation complete
- [x] Resource utilization within acceptable limits

## 📝 Implementation Notes

### Design Decisions
**Official Ollama Docker image**: Pin and run `ollama/ollama:<version>` as a container
- Alternative considered: Host install via upstream installer script (legacy, removed)
- Decision driver: Single deployment axis, reproducible runtime, host isolation
- Trade-offs: Requires Docker; no host `ollama` binary (use `docker exec ollama ollama …`)

**Docker container service management**: Manage lifecycle via the docker-service driver
- Alternative considered: Host-systemd service (removed 2026-06-22)
- Decision driver: One manager (Go driver + resource.json) instead of two drifting ones
- Trade-offs: Requires Docker; memory bounding via `--memory` instead of cgroup unit directives

**Network-only Security Model**: No authentication on localhost
- Alternative considered: API key authentication
- Decision driver: Simplicity and performance for local access
- Trade-offs: Less granular access control, simpler integration

### Known Limitations
- **Memory Requirements**: Large models require significant RAM
  - Workaround: Smaller models and CPU-only mode
  - Future fix: Model streaming and advanced memory management
  
- **Single Instance**: Cannot run multiple Ollama instances easily
  - Workaround: Model switching and management
  - Future fix: Multi-instance support with load balancing

### Integration Considerations
- **Model Selection**: Automatic model selection based on scenario needs
- **Performance Optimization**: GPU utilization and memory management
- **Error Handling**: Graceful degradation when models fail to load
- **Resource Sharing**: Efficient model caching across concurrent requests

## 🔗 References

### Documentation
- README.md - Quick start and model management
- docs/INSTALLATION.md - Setup and configuration guide
- docs/MODELS.md - Available models and selection guide
- docs/API.md - Complete API reference

### Related Resources
- qdrant - Vector embeddings storage and search
- n8n - Workflow automation with AI processing
- postgres - Knowledge storage and retrieval

### External Resources
- [Ollama Official Documentation](https://ollama.ai)
- [Ollama Model Library](https://ollama.ai/library)
- [Modelfile Syntax Guide](https://github.com/ollama/ollama/blob/main/docs/modelfile.md)

---

**Last Updated**: 2025-01-20  
**Status**: Validated  
**Owner**: AI Infrastructure Team  
**Review Cycle**: Monthly review of model performance and new model availability
