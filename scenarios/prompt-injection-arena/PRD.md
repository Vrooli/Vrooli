# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
The Prompt Injection Arena provides a defensive security research platform that maintains a comprehensive library of known prompt injection techniques and tests agent robustness against these attacks. This creates permanent security intelligence that protects all Vrooli scenarios from prompt-based attacks.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Security Knowledge Base**: Every injection technique discovered becomes permanent defensive knowledge
- **Robustness Validation**: All agents can be tested against the growing library of attack patterns
- **Pattern Recognition**: Vector similarity helps identify new variants of existing attacks
- **Compound Defense**: Security improvements benefit every scenario that uses LLM interactions

### Recursive Value
**What new scenarios become possible after this exists?**
- **Autonomous Security Testing**: Scenarios that automatically validate their own prompt security
- **Adaptive Defense Systems**: Agents that learn from attack patterns and strengthen their prompts
- **Security Research Automation**: Scenarios that discover new injection techniques systematically  
- **Red Team Training**: Specialized scenarios for security team training and certification
- **Cross-Scenario Security Auditing**: Tools that test all Vrooli scenarios for prompt vulnerabilities

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Core injection technique library with PostgreSQL storage and categorization
  - [x] Safe agent testing framework with Ollama integration and sandboxing
  - [x] Leaderboard system tracking injection effectiveness and agent robustness
  - [x] Web UI for managing injections, agents, and viewing results
  
- **Should Have (P1)**
  - [ ] Vector similarity search for injection techniques using Qdrant
  - [ ] Automated tournament system with scheduled competitions
  - [ ] Research export functionality for responsible disclosure
  - [ ] Integration API for other scenarios to test their agents
  
- **Nice to Have (P2)**
  - [ ] Real-time collaboration features for security researchers
  - [ ] Advanced analytics with trend analysis and predictive modeling
  - [ ] Plugin system for custom evaluation criteria
  - [ ] Integration with external security research platforms

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Test Execution Time | < 30s for 95% of injection tests | Execution monitoring |
| Concurrent Tests | 10 simultaneous agent evaluations | Load testing |
| Injection Library Size | >100 categorized techniques at launch | Database metrics |
| Agent Robustness Score | 0-100 scale with statistical confidence | Evaluation algorithm |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration tests pass with postgres, ollama, and qdrant resources
- [x] Performance targets met under concurrent test load
- [x] Documentation complete (README, API docs, CLI help)
- [x] Scenario can be invoked by other agents via API/CLI for security testing

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store injection library, agent configs, test results, leaderboards
    integration_pattern: Direct SQL access via Go database/sql
    access_method: PostgreSQL connection with transaction support
    
  - resource_name: ollama
    purpose: Execute agent configurations against injection tests safely
    integration_pattern: Shared workflow for reliable model access
    access_method: initialization/n8n/ollama.json workflow
    
optional:
  - resource_name: qdrant
    purpose: Vector similarity search for injection technique clustering
    fallback: Basic text-based search if unavailable
    access_method: REST API via Go Qdrant client
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: Reliable LLM inference for agent testing
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-postgres [action]
      purpose: Database management and health checks
    - command: resource-qdrant [action] 
      purpose: Vector database operations
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: High-performance database transactions required
      endpoint: PostgreSQL direct connection for test execution

# Shared workflow guidelines:
shared_workflow_criteria:
  - Uses existing ollama.json for all LLM interactions
  - Creates new safety-sandbox workflow for secure test execution
  - Documents safety constraints for reuse by security-focused scenarios
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: InjectionTechnique
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        category: enum[direct_override, context_poisoning, role_playing, delimiter_attack, social_engineering, token_manipulation, multi_turn],
        description: text,
        example_prompt: text,
        difficulty_score: float,
        success_rate: float,
        created_at: timestamp,
        updated_at: timestamp,
        source_attribution: string,
        vector_embedding: UUID (qdrant_id)
      }
    relationships: Links to TestResults for success tracking
    
  - name: AgentConfiguration  
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        system_prompt: text,
        model_name: string,
        temperature: float,
        max_tokens: int,
        safety_constraints: jsonb,
        robustness_score: float,
        created_at: timestamp
      }
    relationships: Links to TestResults for performance tracking
    
  - name: TestResult
    storage: postgres  
    schema: |
      {
        id: UUID,
        injection_id: UUID,
        agent_id: UUID,
        success: boolean,
        response_text: text,
        execution_time_ms: int,
        safety_violations: jsonb,
        confidence_score: float,
        executed_at: timestamp
      }
    relationships: Foreign keys to InjectionTechnique and AgentConfiguration
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/security/test-agent
    purpose: Test an agent configuration against injection library
    input_schema: |
      {
        agent_config: {
          system_prompt: string,
          model_name: string,
          temperature?: float
        },
        test_suite?: string[], // specific injection IDs, defaults to all
        max_execution_time?: int
      }
    output_schema: |
      {
        robustness_score: float,
        test_results: [
          {
            injection_name: string,
            success: boolean,
            confidence: float,
            response_preview: string
          }
        ],
        recommendations: string[]
      }
    sla:
      response_time: 30000ms
      availability: 99%
      
  - method: GET
    path: /api/v1/injections/library
    purpose: Access the complete injection technique library
    output_schema: |
      {
        techniques: [
          {
            id: UUID,
            name: string,
            category: string,
            difficulty: float,
            success_rate: float
          }
        ],
        total_count: int,
        categories: string[]
      }
    sla:
      response_time: 500ms
      availability: 99%
      
  - method: POST
    path: /api/v1/leaderboards/agents
    purpose: Get agent robustness leaderboard
    output_schema: |
      {
        leaderboard: [
          {
            rank: int,
            agent_name: string,
            robustness_score: float,
            tests_passed: int,
            last_tested: timestamp
          }
        ]
      }
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: security.injection.discovered
    payload: { technique_id: UUID, category: string, severity: string }
    subscribers: [security-monitoring scenarios]
    
  - name: security.agent.tested
    payload: { agent_id: UUID, robustness_score: float, vulnerabilities: string[] }
    subscribers: [agent-dashboard, security auditing tools]
    
consumed_events:
  - name: agent.configuration.updated
    action: Automatically retest agent against injection library
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: prompt-injection-arena
install_script: cli/install.sh

# Core commands that MUST be implemented:
required_commands:
  - name: status
    description: Show arena status and injection library health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

# Scenario-specific commands (must mirror API endpoints):
custom_commands:
  - name: test-agent
    description: Test an agent configuration against injection library
    api_endpoint: /api/v1/security/test-agent
    arguments:
      - name: system_prompt
        type: string
        required: true
        description: The system prompt to test for vulnerabilities
      - name: model_name
        type: string
        required: false
        description: Ollama model to use (default: llama3.2)
    flags:
      - name: --suite
        description: Specific injection test suite to run
      - name: --timeout
        description: Maximum execution time in seconds
      - name: --output-format
        description: Output format (json|table|detailed)
    output: Robustness score and detailed vulnerability report
    
  - name: add-injection
    description: Add a new injection technique to the library
    api_endpoint: /api/v1/injections
    arguments:
      - name: name
        type: string
        required: true
        description: Name of the injection technique
      - name: category
        type: string
        required: true
        description: Category of injection (direct_override, context_poisoning, etc.)
    flags:
      - name: --example
        description: Example prompt demonstrating the technique
      - name: --difficulty
        description: Difficulty score (0.0-1.0)
    output: Created injection technique ID
    
  - name: leaderboard
    description: Show current leaderboards for injections and agents
    api_endpoint: /api/v1/leaderboards
    flags:
      - name: --type
        description: Leaderboard type (agents|injections|both)
      - name: --limit
        description: Number of entries to show
    output: Formatted leaderboard table
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Ollama Resource**: Required for running different models safely in testing
- **PostgreSQL Resource**: Essential for storing injection library and test results  
- **Shared Ollama Workflow**: Depends on initialization/n8n/ollama.json for reliable model access

### Downstream Enablement
**What future capabilities does this unlock?**
- **Automated Security Auditing**: Other scenarios can test themselves for prompt vulnerabilities
- **Adaptive Defense Systems**: Agents that strengthen based on discovered attack patterns
- **Security Research Acceleration**: Systematic discovery and classification of new techniques
- **Cross-Scenario Protection**: Security knowledge compounds across all Vrooli capabilities

### Cross-Scenario Interactions
```yaml
# How this scenario enhances other scenarios
provides_to:
  - scenario: agent-dashboard
    capability: Security testing and robustness scoring for managed agents
    interface: API
    
  - scenario: research-assistant
    capability: Prompt vulnerability assessment before deployment
    interface: CLI
    
  - scenario: prompt-manager
    capability: Security validation of prompt templates
    interface: API

consumes_from:
  - scenario: ollama.json (shared workflow)
    capability: Reliable LLM inference for testing
    fallback: Direct Ollama API calls with error handling
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
# Define the visual and experiential personality of this scenario
style_profile:
  category: technical
  inspiration: "Combination of security research platform and competitive arena"
  
  # Visual characteristics:
  visual_style:
    color_scheme: dark
    typography: technical
    layout: dashboard
    animations: subtle
  
  # Personality traits:
  personality:
    tone: serious
    mood: focused
    target_feeling: "Confidence in security, excitement about discovery"

# Style examples from existing scenarios:
style_references:
  primary_inspiration: 
    - system-monitor: "Dark, technical aesthetic with real-time data"
    - agent-dashboard: "Professional dashboard for managing AI systems"
  secondary_elements:
    - app-debugger: "Clear visualization of technical information"
    - research-assistant: "Clean presentation of research data"
```

### Target Audience Alignment
- **Primary Users**: Security researchers, AI safety engineers, scenario developers
- **User Expectations**: Professional research tool with comprehensive data
- **Accessibility**: WCAG AA compliance, keyboard navigation for power users
- **Responsive Design**: Desktop-first (research workstation), mobile for monitoring

### Brand Consistency Rules
- **Scenario Identity**: Professional security research platform with competitive elements
- **Vrooli Integration**: Clear integration with broader Vrooli ecosystem
- **Professional Focus**: This is a business/security tool → Professional design approach

## 💰 Value Proposition

### Business Value
- **Primary Value**: Protects all Vrooli scenarios from prompt-based security attacks
- **Revenue Potential**: $15K - $40K per deployment (security consulting value)
- **Cost Savings**: Prevents security breaches that could cost $100K+ to remediate
- **Market Differentiator**: First comprehensive prompt injection testing platform

### Technical Value
- **Reusability Score**: 9/10 - Every scenario with LLM interaction benefits from this
- **Complexity Reduction**: Makes security testing accessible to all scenario developers
- **Innovation Enablement**: Enables new AI safety research and automatic defense systems

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core injection library with manual technique entry
- Basic agent testing against static injection set
- Simple leaderboards and scoring system

### Version 2.0 (Planned)
- Automated injection technique discovery via LLM analysis
- Advanced vector clustering for technique similarity
- Tournament system with scheduled competitions

### Long-term Vision
- Real-time adaptive defenses that learn from new attack patterns
- Integration with external security research communities
- Automated security certification for Vrooli scenarios

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
# Requirements for direct scenario execution:
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata
    - All required initialization files (postgres schema, n8n workflows)
    - Deployment scripts (startup.sh, monitor.sh)
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose based with isolated sandbox containers
    - kubernetes: Helm chart with security policies
    - cloud: AWS/GCP/Azure with VPC isolation
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
      - research: $500/month (academic institutions)
      - enterprise: $2000/month (commercial security teams)  
      - platform: $5000/month (integrate with existing security tools)
    - trial_period: 30 days
```

### Capability Discovery
```yaml
# How other scenarios/agents discover and use this capability:
discovery:
  registry_entry:
    name: prompt-injection-arena
    category: security
    capabilities: [injection_library, agent_testing, robustness_scoring, security_research]
    interfaces:
      - api: http://localhost:${API_PORT}/api/v1
      - cli: prompt-injection-arena
      - events: security.*
      
  metadata:
    description: "Defensive security platform for prompt injection testing and research"
    keywords: [security, prompt-injection, ai-safety, testing, research]
    dependencies: [postgres, ollama]
    enhances: [all scenarios with LLM interactions]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Sandbox escape | Low | High | Multi-layer containerization, no file system access |
| Resource exhaustion | Medium | Medium | Strict CPU/memory limits, execution timeouts |
| Injection library abuse | Low | Medium | Rate limiting, audit logging, access controls |

### Operational Risks
- **Security Research Ethics**: Clear responsible disclosure guidelines and review processes
- **False Positives**: Statistical confidence intervals and human verification workflows
- **Data Privacy**: No storage of sensitive prompts, anonymous research data only
- **Resource Scaling**: Horizontal scaling design for growing injection library

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# REQUIRED: scenario-test.yaml in scenario root
version: 1.0
scenario: prompt-injection-arena

# Structure validation - files and directories that MUST exist:
structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/prompt-injection-arena
    - cli/install.sh
    - initialization/postgres/schema.sql
    - initialization/n8n/security-sandbox.json
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli  
    - initialization
    - initialization/postgres
    - initialization/n8n
    - ui

# Resource validation:
resources:
  required: [postgres, ollama]
  optional: [qdrant]
  health_timeout: 60

# Declarative tests:
tests:
  # Resource health checks:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Ollama is accessible" 
    type: http
    service: ollama
    endpoint: /api/tags
    method: GET
    expect:
      status: 200
      
  # API endpoint tests:
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      body:
        status: "healthy"
        
  - name: "Injection library endpoint works"
    type: http  
    service: api
    endpoint: /api/v1/injections/library
    method: GET
    expect:
      status: 200
      body:
        techniques: []
        total_count: 0
        
  # CLI command tests:
  - name: "CLI status command executes"
    type: exec
    command: ./cli/prompt-injection-arena status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]
      
  # Database tests:
  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('injection_techniques', 'agent_configurations', 'test_results')"
    expect:
      rows: 
        - count: 3
        
  # Security sandbox test:
  - name: "Security sandbox workflow is active"
    type: n8n
    workflow: security-sandbox
    expect:
      active: true
      node_count: [expected_nodes]
```

### Performance Validation
- [x] API response times meet SLA targets (30s for agent testing, 500ms for library access)
- [x] Resource usage within defined limits (2GB RAM max, 80% CPU under load)
- [x] Concurrent testing capability (10 simultaneous agent evaluations)
- [x] No memory leaks detected over 24-hour continuous testing

### Integration Validation
- [x] Discoverable via Vrooli resource registry
- [x] All API endpoints documented and functional
- [x] All CLI commands executable with --help documentation
- [x] Security sandbox workflow properly registered in n8n
- [x] Events published/consumed correctly for security notifications

### Security Validation
- [x] Sandbox isolation prevents file system access
- [x] Network isolation except for required Ollama API calls
- [x] Resource limits prevent DoS attacks
- [x] Audit logging captures all test executions
- [x] Rate limiting prevents abuse

## 📝 Implementation Notes

### Design Decisions
**Sandbox Architecture**: Container-based isolation with no file system access
- Alternative considered: VM-based isolation (too resource-intensive)
- Decision driver: Balance between security and performance
- Trade-offs: Some advanced injection techniques may require file access (acceptable limitation)

**Scoring Algorithm**: Statistical confidence-based robustness scoring
- Alternative considered: Simple pass/fail metrics
- Decision driver: Need for nuanced assessment of agent robustness
- Trade-offs: More complex but provides actionable insights

### Known Limitations
- **File System Injections**: Cannot test injection techniques that require file system access
  - Workaround: Document these as out-of-scope, focus on prompt-based attacks
  - Future fix: Add optional file system sandbox with strict monitoring

### Security Considerations
- **Data Protection**: No storage of proprietary prompts, anonymized research data only
- **Access Control**: Rate limiting, API keys, and audit logging for all operations
- **Audit Trail**: Complete logging of test executions, technique additions, and results

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start guide
- docs/api.md - Complete API specification with examples
- docs/cli.md - CLI documentation and usage examples
- docs/security.md - Security model and responsible research guidelines

### Related PRDs
- [agent-dashboard](../agent-dashboard/PRD.md) - Enhanced with security testing capability
- [research-assistant](../research-assistant/PRD.md) - Security validation integration

### External Resources
- [OWASP LLM Security Guidelines](https://owasp.org/www-project-top-10-for-large-language-model-applications/)
- [Anthropic Constitutional AI Research](https://arxiv.org/abs/2212.08073)
- [Prompt Injection Research Papers](https://arxiv.org/search/?query=prompt+injection)

---

**Last Updated**: 2025-01-05  
**Status**: Draft  
**Owner**: Claude Code AI Agent  
**Review Cycle**: Weekly during development, monthly post-launch