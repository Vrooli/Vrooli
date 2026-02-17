# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Real-time system monitoring with AI-driven anomaly detection and automated root cause analysis. This scenario provides infrastructure observability that enables proactive system health management, performance optimization, and intelligent incident response across all Vrooli resources and scenarios.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Provides performance baselines that agents use to optimize their own execution
- Creates anomaly detection patterns that identify issues before they cascade
- Establishes resource usage profiles that guide efficient task scheduling
- Enables predictive maintenance that prevents system failures
- Offers diagnostic workflows that agents apply to self-healing operations

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Auto-Scaling Orchestrator**: Dynamic resource allocation based on metrics
2. **Cost Optimization Advisor**: Cloud spend analysis using usage patterns
3. **Security Threat Detector**: Anomaly patterns applied to security monitoring
4. **Performance Tuner**: Automated system optimization recommendations
5. **Incident Response Manager**: Intelligent alerting and remediation workflows

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Real-time CPU, memory, disk, network, GPU monitoring
  - [x] Threshold-based anomaly detection with configurable triggers
  - [x] Automated investigation of system anomalies via agent-manager
  - [x] Time-series data storage in QuestDB (configured; API defaults to in-memory with PostgreSQL fallback)
  - [x] Configurable warning/critical thresholds (via API settings endpoints)
  - [x] Report generation (daily/weekly via API endpoint)
  - [x] Dark cyberpunk monitoring dashboard (Matrix-themed React UI)
  - [x] Investigation script execution (30 scripts in investigations/active/; API script endpoints are placeholders)
  - [x] Process monitoring and management (zombie detection, process kill — UI has kill dialog but API endpoint `/processes/{pid}/kill` not implemented)
  - [x] Infrastructure monitoring (database pools, HTTP pools, message queues, storage I/O)

- **Should Have (P1)**
  - [ ] Historical trend analysis (no timeline endpoint exists; trend analysis implemented only within report generation)
  - [x] Alert webhook support (configured, cooldown-based)
  - [x] Investigation cooldown management (configurable period, reset capability)
  - [x] Agent configuration management (runner type, model, max turns, timeout)
  - [ ] Custom metric collection via API (no custom metric ingestion endpoint; only built-in collectors)
  - [ ] Alert routing to multiple channels (webhook and email configured but email not implemented)
  - [ ] Resource prediction models (not implemented)
  - [ ] Correlation analysis between metrics (not implemented)

- **Nice to Have (P2)**
  - [ ] Distributed tracing integration
  - [ ] Custom dashboard builder
  - [ ] Mobile monitoring app
  - [ ] WebSocket real-time updates (type defined but UI uses HTTP polling)

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Metric Collection | < 100ms latency | Prometheus scrape duration |
| Anomaly Detection | < 30s from occurrence | Alert timestamp comparison |
| Dashboard Refresh | 5s current+detailed metrics, 60s process/infra/investigations, 4s agent status (WebSocket not implemented) | UI polling interval |
| Query Performance | < 500ms for 24h data | QuestDB query profiling |
| AI Investigation | < 2min per anomaly | Workflow execution time |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] QuestDB ingests 1000+ metrics/second
- [ ] Anomaly detection accuracy > 85%
- [ ] Dashboard renders smoothly with 50+ metric streams
- [ ] Zero data loss during 24-hour stress test

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store thresholds, investigations, configurations
    integration_pattern: Direct SQL for configuration management
    access_method: resource-postgres CLI for backups
    
  - resource_name: questdb
    purpose: High-performance time-series metrics storage
    integration_pattern: ILP (InfluxDB Line Protocol) for ingestion
    access_method: Direct API for metrics ingestion
    
  - resource_name: redis
    purpose: Real-time alerts and metric buffering
    integration_pattern: Pub/Sub for alert distribution
    access_method: resource-redis CLI for queue management
    
  - resource_name: node-red
    purpose: Orchestrate monitoring workflows
    integration_pattern: Scheduled and triggered flows
    access_method: resource-node-red CLI for flow management
    
  - resource_name: ollama
    purpose: AI analysis of anomalies (llama3.2:3b)
    integration_pattern: Investigation prompt execution via agent-manager
    access_method: Agent-manager API integration (api/internal/agentmanager/)

optional:
  - resource_name: grafana
    purpose: Advanced visualization dashboards
    integration_pattern: Optional dashboard integration
    access_method: GRAFANA_URL env var (disabled by default)

scenario_dependencies:
  - scenario_name: agent-manager
    purpose: Orchestrate AI-driven investigations
    integration_pattern: API calls for agent spawning, status polling, stopping
    access_method: api/internal/agentmanager/ package
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: metric-collector.json
      location: initialization/node-red/
      purpose: Collect host metrics and fan out to storage
      reused_by: [load-tester, api-monitor]
      
    - workflow: anomaly-detector.json
      location: initialization/node-red/
      purpose: Analyze metric trends and trigger investigations
      reused_by: [log-analyzer, incident-response-manager]
      
  2_resource_cli:
    - command: resource-questdb query "SELECT * FROM metrics WHERE time > now() - '1h'"
      purpose: Direct metric queries for analysis
      
    - command: resource-redis publish alerts "{"severity":"critical"}"
      purpose: Distribute alerts to subscribers
      
  3_direct_api:
    - justification: Real-time metric ingestion requires ILP protocol
      endpoint: tcp://localhost:9009 (QuestDB ILP)
      note: Configured but API primarily uses in-memory + PostgreSQL fallback

    - justification: Live dashboard updates via HTTP polling
      endpoint: http://localhost:${API_PORT}/api/v1/metrics/current (5s poll interval)
      note: WebSocket types defined but not implemented; UI uses HTTP polling

shared_workflow_validation:
  - metric-collector.json collects system telemetry for storage
  - anomaly-detector.json evaluates triggers and opens investigations
```

### Data Models
```yaml
# PostgreSQL tables (initialization/postgres/schema.sql):
primary_entities:
  - name: metrics
    storage: postgres
    schema: |
      { id: serial PK, metric_name: varchar(255), value: double precision,
        unit: varchar(50), source: varchar(100), tags: jsonb, timestamp: timestamptz }
    indexes: [idx_metrics_timestamp, idx_metrics_name]

  - name: anomalies
    storage: postgres
    schema: |
      { id: serial PK, anomaly_type: varchar(100), severity: varchar(20),
        detected_at: timestamptz, resolved_at: timestamptz, status: varchar(20) default 'open',
        description: text, metadata: jsonb }
    indexes: [idx_anomalies_status]

  - name: investigations
    storage: postgres
    schema: |
      { id: serial PK, anomaly_id: int FK→anomalies, started_at: timestamptz,
        completed_at: timestamptz, status: varchar(20) default 'pending',
        agent: varchar(100) default 'claude', findings: text, recommendations: text,
        metadata: jsonb }
    indexes: [idx_investigations_anomaly]

  - name: reports
    storage: postgres
    schema: |
      { id: serial PK, report_type: varchar(50), generated_at: timestamptz default now(),
        period_start: timestamptz, period_end: timestamptz, content: jsonb,
        metrics_snapshot: jsonb, status: varchar(20) default 'generated' }

  - name: thresholds
    storage: postgres
    schema: |
      { id: serial PK, metric_name: varchar(255) unique, warning_level: double precision,
        critical_level: double precision, enabled: boolean default true,
        created_at: timestamptz default now(), updated_at: timestamptz default now() }
    default_data: [cpu_usage 70/90, memory_usage 80/95, disk_usage 85/95, tcp_connections 500/1000]

  - name: system_health
    storage: postgres
    schema: |
      { id: serial PK, timestamp: timestamptz default now(), cpu_usage: double precision,
        memory_usage: double precision, disk_usage: double precision,
        network_rx_bytes: bigint, network_tx_bytes: bigint, tcp_connections: int,
        process_count: int, load_average: double precision }

# API in-memory models (api/internal/models/):
api_models:
  - MetricsResponse: { cpu_usage, memory_usage, tcp_connections, gpu_usage, timestamp }
  - DetailedMetrics: { cpu, memory, network, gpu, disk, processes, system_health }
  - Investigation: { id, status, anomaly_id, start/end time, findings, progress, steps, details }
  - EnhancedSystemReport: { id, type, time range, executive summary, performance, trends, recommendations }
  - Settings: { metric/anomaly/threshold intervals, thresholds, cooldown }
  - TriggerConfig: { id, name, enabled, auto_fix, threshold, unit, condition }

# QuestDB configured but API primarily uses in-memory + PostgreSQL fallback
```

### API Contract
```yaml
endpoints:
  # Health
  - method: GET
    path: /health
    purpose: Basic health check
  - method: GET
    path: /api/v1/health
    purpose: Detailed health check with dependency status

  # Metrics
  - method: GET
    path: /api/v1/metrics/current
    purpose: Current system metrics snapshot
    query_params: { fresh: "1|true for real-time collection" }
    output_schema: |
      { cpu_usage: float, memory_usage: float, tcp_connections: int, gpu_usage: float, timestamp: string }
  - method: GET
    path: /api/v1/metrics/detailed
    purpose: Comprehensive metrics (CPU, memory, network, GPU, disk, processes, system health)
  - method: GET
    path: /api/v1/metrics/processes
    purpose: Process monitoring data (zombies, high-thread, leak candidates)
  - method: GET
    path: /api/v1/metrics/infrastructure
    purpose: Infrastructure monitoring (DB pools, HTTP pools, queues, storage I/O)

  # Investigations
  - method: GET
    path: /api/v1/investigations
    purpose: List investigations (query param limit, default 20)
  - method: GET
    path: /api/v1/investigations/latest
    purpose: Get latest investigation
  - method: GET
    path: /api/v1/investigations/{id}
    purpose: Get investigation by ID
  - method: POST
    path: /api/v1/investigations/trigger
    purpose: Trigger new investigation
    input_schema: |
      { auto_fix: boolean, note: string }
    output_schema: |
      { id: string, status: "queued", message: string }
  - method: POST
    path: /api/v1/investigations/agent/spawn
    purpose: Alias for trigger investigation
  - method: GET
    path: /api/v1/investigations/agent/current
    purpose: Get currently running agent status
  - method: GET
    path: /api/v1/investigations/agent/{id}/status
    purpose: Get agent status by investigation ID
  - method: POST
    path: /api/v1/investigations/agent/{id}/stop
    purpose: Stop agent for investigation
  - method: PUT
    path: /api/v1/investigations/{id}/status
    purpose: Update investigation status
  - method: PUT
    path: /api/v1/investigations/{id}/findings
    purpose: Update findings
  - method: PUT
    path: /api/v1/investigations/{id}/progress
    purpose: Update progress (0-100)
  - method: POST
    path: /api/v1/investigations/{id}/step
    purpose: Add investigation step
  - method: GET
    path: /api/v1/investigations/cooldown
    purpose: Get cooldown status
  - method: POST
    path: /api/v1/investigations/cooldown/reset
    purpose: Reset cooldown period
  - method: PUT
    path: /api/v1/investigations/cooldown/period
    purpose: Update cooldown duration
  - method: GET
    path: /api/v1/investigations/triggers
    purpose: Get all investigation triggers
  - method: PUT
    path: /api/v1/investigations/triggers/{id}
    purpose: Update trigger config
  - method: PUT
    path: /api/v1/investigations/triggers/{id}/threshold
    purpose: Update trigger threshold only
  - method: GET
    path: /api/v1/investigations/scripts
    purpose: List investigation scripts (placeholder - returns empty array)
  - method: GET
    path: /api/v1/investigations/scripts/{id}
    purpose: Get script by ID (placeholder - returns not found)
  - method: POST
    path: /api/v1/investigations/scripts/{id}/execute
    purpose: Execute investigation script (placeholder - returns not found)

  # Reports
  - method: POST
    path: /api/v1/reports/generate
    purpose: Generate report
    input_schema: |
      { type: "daily"|"weekly" }
  - method: GET
    path: /api/v1/reports
    purpose: List all reports
  - method: GET
    path: /api/v1/reports/{id}
    purpose: Get report by ID

  # Settings
  - method: GET
    path: /api/v1/settings
    purpose: Get all settings
  - method: PUT
    path: /api/v1/settings
    purpose: Update settings
  - method: POST
    path: /api/v1/settings/reset
    purpose: Reset to defaults

  # Maintenance
  - method: GET
    path: /api/v1/maintenance/state
    purpose: Get maintenance state
  - method: POST
    path: /api/v1/maintenance/state
    purpose: Set maintenance state (active/inactive)

  # Agent Configuration
  - method: GET
    path: /api/v1/agent/config
    purpose: Get agent configuration
  - method: PUT
    path: /api/v1/agent/config
    purpose: Update agent config (runner, model, max turns, timeout, tools, skip_permissions, requires_sandbox, requires_approval)
  - method: GET
    path: /api/v1/agent/runners
    purpose: Get available runners
  - method: GET
    path: /api/v1/agent/status
    purpose: Get agent status

  # Tool Discovery Protocol
  - method: GET
    path: /api/v1/tools
    purpose: Get tool manifest
  - method: GET
    path: /api/v1/tools/{name}
    purpose: Get specific tool definition
  - method: POST
    path: /api/v1/tools/execute
    purpose: Execute a tool
```

### Event Interface
```yaml
# Note: The event system is configured via Node-RED flows and Redis pub/sub,
# but no formal event bus with named events is implemented in the Go API.
# The following describes the actual integration pattern:

node_red_flows:
  - name: metric-collector
    location: initialization/node-red/metric-collector.json
    trigger: Every 30 seconds
    action: Collect system metrics, store to PostgreSQL/QuestDB/Redis

  - name: anomaly-detector
    location: initialization/node-red/anomaly-detector.json
    trigger: Every 60 seconds
    action: Check thresholds, trigger alerts

redis_channels:
  - key_patterns: system_metrics:*, system_alerts:*, system_thresholds:*, system_investigations:*
  - keyspace_events: Enabled ("Ex") for expiry notifications

webhook_alerts:
  - configured: true (via ALERT_WEBHOOK_URL env var)
  - implemented: Basic webhook POST on threshold violations
  - cooldown: 5 minutes between alerts (configurable)

# These planned events from the original PRD are NOT implemented:
# - monitor.anomaly.detected (no formal event bus)
# - monitor.investigation.complete (no event publishing)
# - monitor.resource.critical (no event publishing)
# - scenario.performance.degraded (no event consumption)
# - resource.health.changed (no event consumption)
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: system-monitor
install_script: cli/install.sh
language: Bash
version: 2.0.0

global_flags:
  - name: -h, --help
    description: Display help message
  - name: -v, --version
    description: Display CLI version
  - name: -p, --port <port>
    description: Override API port (default 8080)
  - name: -j, --json
    description: Output in JSON format
  - name: -q, --quiet
    description: Suppress non-essential output (parsed but never checked in code - no effect)

commands:
  - name: version
    description: Show CLI version (2.0.0)
    api_endpoint: none (local only)

  - name: health
    description: Check API health status
    api_endpoint: GET /health
    flags: [--json]
    example: system-monitor health

  - name: metrics
    description: Get current system metrics (CPU, memory, TCP connections)
    api_endpoint: GET /api/v1/metrics/current
    flags: [--json]
    example: system-monitor metrics --json

  - name: status
    description: Show system status (HEALTHY/WARNING/CRITICAL/OFFLINE)
    api_endpoint: GET /api/v1/metrics/current
    thresholds: "CRITICAL: CPU>90% or Mem>95%; WARNING: CPU>80% or Mem>85%"
    example: system-monitor status

  - name: alerts
    description: List active alerts based on current metrics
    api_endpoint: GET /api/v1/metrics/current
    alert_conditions: "HIGH_CPU>80%, HIGH_MEMORY>85%, HIGH_CONNECTIONS>150"
    example: system-monitor alerts

  - name: investigate
    description: Fetch latest investigation results
    api_endpoint: GET /api/v1/investigations/latest
    flags: [--json]
    example: system-monitor investigate

  - name: report
    description: Generate system report
    api_endpoint: POST /api/v1/reports/generate
    note: "CLI BUG: script calls /api/reports/generate (missing /v1/ prefix)"
    arguments:
      - name: type
        values: [daily, weekly]
        required: true
    flags: [--json]
    example: system-monitor report daily

  - name: simulate
    description: Simulate CPU anomaly for testing
    api_endpoint: GET /api/test/anomaly/cpu (not implemented in API)
    flags: [--json]
    note: Test endpoint does not exist in API router

  - name: dashboard
    description: Open UI dashboard in browser
    api_endpoint: none (launches browser via xdg-open)
    example: system-monitor dashboard

  - name: watch
    description: Live monitoring with ASCII progress bars (2s refresh)
    api_endpoint: GET /api/v1/metrics/current (polling)
    example: system-monitor watch
```

### CLI-API Parity Requirements
- **Coverage**: Core monitoring endpoints accessible via CLI (metrics, investigations, reports)
- **Output**: Human-readable by default, JSON with --json flag
- **Configuration**: Environment variables only (API_PORT, UI_PORT); no config file
- **Authentication**: None (no API key support)
- **Note**: Several API endpoints have no CLI equivalent (settings, triggers, agent config, tools, maintenance, detailed metrics, processes, infrastructure)

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Bash shell script with curl-based API client
  - language: Bash (not Go as originally planned)
  - dependencies:
      - curl (HTTP client)
      - bc (floating-point math)
      - grep, cut (JSON parsing - fragile, no jq)
  - error_handling:
      - Exit 0: Success
      - Exit 1: General error or API unavailable
  - configuration:
      - Env: API_PORT (default 8080), UI_PORT (default 3003)
      - Flags: --port overrides API_PORT

installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - permissions: 755 on script
  - two_entry_points: system-monitor and vrooli-system-monitor
  - note: vrooli-system-monitor is missing the version command and -v flag
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: "The Matrix meets htop - cyberpunk system monitoring"
  
  visual_style:
    color_scheme: dark with neon green accents
    typography: monospace, terminal-style fonts
    layout: dense information grid with animated backgrounds
    animations: matrix rain effect, pulsing alerts, graph animations
  
  personality:
    tone: technical, authoritative, slightly ominous
    mood: intense focus, constant vigilance
    target_feeling: "I have god-mode visibility into the system"

ui_components:
  main_dashboard:
    - Matrix rain background (subtle, non-distracting)
    - Real-time metric cards with neon borders
    - Animated line graphs with glow effects
    - Alert feed with severity color coding
    
  metric_visualizations:
    - CPU: Animated core utilization bars
    - Memory: Liquid fill gauge effect
    - Disk: Circular progress with sectors
    - Network: Flowing particle visualization
    
  investigation_panel:
    - Terminal-style log viewer
    - AI analysis with typewriter effect
    - Dependency graph with force layout
    
  alert_system:
    - Flashing borders for critical alerts
    - Sound effects (optional): beeps, warnings
    - Full-screen takeover for emergencies

color_palette:
  primary: "#00FF41"     # Matrix green
  secondary: "#39FF14"   # Neon green
  warning: "#FFFF00"     # Yellow
  critical: "#FF0000"    # Red
  background: "#0D0208"  # Nearly black
  surface: "#1A1A1A"     # Dark gray
  text: "#00FF41"        # Green text
  accent: "#00FFFF"      # Cyan for highlights
```

### Target Audience Alignment
- **Primary Users**: DevOps engineers, SREs, system administrators
- **User Expectations**: Power user interface, information density, keyboard navigation
- **Accessibility**: High contrast mode available, screen reader support for alerts
- **Responsive Design**: Desktop-optimized, terminal mode for SSH sessions

### Brand Consistency Rules
- **Scenario Identity**: "The all-seeing eye of your infrastructure"
- **Vrooli Integration**: Technical scenarios use function-over-form approach
- **Professional vs Fun**: Technical but engaging - serious tool with personality
- **Differentiation**: More visual than Prometheus, more integrated than Grafana

## 💰 Value Proposition

### Business Value
- **Primary Value**: Prevents downtime (saves $10K+/hour for enterprises), reduces MTTR by 60%
- **Revenue Potential**: $25K - $35K per deployment
- **Cost Savings**: 30 hours/week saved on manual monitoring and investigation
- **Market Differentiator**: Only monitoring tool with built-in AI root cause analysis

### Technical Value
- **Reusability Score**: 10/10 - Every scenario benefits from performance monitoring
- **Complexity Reduction**: Consolidates 5+ monitoring tools into one
- **Innovation Enablement**: Enables self-healing and auto-scaling scenarios

## 🔄 Scenario Lifecycle Integration

### Scenario-to-App Conversion
```yaml
app_conversion:
  supported: true
  app_structure_compliance:
    - Complete service.json with monitoring configuration
    - QuestDB schema for time-series data
    - N8n workflows for investigations
    - Matrix-style dashboard UI
    
  deployment_targets:
    - local: Docker Compose with persistent metrics
    - kubernetes: DaemonSet for node monitoring
    - cloud: CloudWatch/Stackdriver integration
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        basic: $200/month (10 hosts)
        standard: $800/month (50 hosts)
        enterprise: $2500/month (unlimited)
    - trial_period: 14 days
    - value_proposition: "Replace Datadog + New Relic at 20% cost"
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: system-monitor
    category: infrastructure
    capabilities:
      - Real-time system metrics
      - AI anomaly detection
      - Automated investigations
      - Performance baselines
      - Alert management
    interfaces:
      - api: http://localhost:${API_PORT}/api/v1
      - cli: system-monitor
      - events: monitor.*
      - metrics: tcp://localhost:9009 (QuestDB)
      
  metadata:
    description: "AI-powered infrastructure monitoring with anomaly detection"
    keywords: [monitoring, metrics, anomaly, performance, infrastructure]
    dependencies: []
    enhances: [all scenarios benefit from monitoring]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  api_version: v1
  
  breaking_changes: []
  deprecations: []
  
  upgrade_path:
    from_0_9: "Migrate from InfluxDB to QuestDB"
```

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core system metrics monitoring
- AI anomaly detection with Ollama
- Matrix-style dashboard
- Basic alerting and reporting

### Version 2.0 (Planned)
- Distributed tracing integration
- Custom metric collectors
- Predictive failure analysis
- Multi-cluster monitoring
- Cost analysis features

### Long-term Vision
- Autonomous infrastructure management
- Self-healing orchestration
- Capacity planning AI
- Cross-scenario performance optimization

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Metric data loss | Low | Critical | QuestDB replication, Redis buffer |
| False positive alerts | Medium | Medium | ML model training, threshold tuning |
| Dashboard performance | Low | Low | WebSocket optimization, data sampling |
| AI investigation timeout | Medium | Low | Async processing, timeout limits |

### Operational Risks
- **Drift Prevention**: PRD validated against metrics accuracy weekly
- **Version Compatibility**: Metric format versioning in QuestDB
- **Resource Conflicts**: Dedicated QuestDB instance for isolation
- **Style Drift**: Matrix theme enforced by CSS framework
- **CLI Consistency**: Integration tests for all commands

## ✅ Validation Criteria

### Declarative Test Specification
```bash
# test/run-tests.sh (excerpt)
testing::runner::register_phase --name structure --script "test/phases/test-structure.sh"
testing::runner::register_phase --name dependencies --script "test/phases/test-dependencies.sh"
testing::runner::register_phase --name unit --script "test/phases/test-unit.sh"
testing::runner::register_phase --name integration --script "test/phases/test-integration.sh" --requires-runtime true
testing::runner::register_phase --name business --script "test/phases/test-business.sh" --requires-runtime true
testing::runner::register_phase --name performance --script "test/phases/test-performance.sh"
```
- **Structure** validates required files/directories, gofmt cleanliness, and optional UI lint checks.
- **Dependencies** dry-runs Go module analysis and `npm install` to ensure manifests stay resolvable.
- **Unit** leverages the shared multi-language runner with scenario-specific coverage thresholds.
- **Integration** executes `go test -tags=integration` against API services for workflow coverage.
- **Business** exercises `/health`, `/api/v1/metrics/current`, and report generation while verifying the React dashboard responds.
- **Performance** runs best-effort Go benchmarks and targeted performance tests, emitting warnings when regressions appear.

### Test Execution Gates
```bash
# Full suite with managed runtime
test/run-tests.sh comprehensive

# Quick developer loop (structure + unit)
test/run-tests.sh quick

# Integration/business when the scenario is already running
test/run-tests.sh business --allow-skip-missing-runtime

# Lifecycle entrypoint
vrooli scenario test system-monitor
```

### Performance Validation
- [ ] Metric ingestion < 100ms latency (not formally tested)
- [ ] Anomaly detection < 30s from occurrence (not formally tested)
- [x] Dashboard updates every 5s via HTTP polling for metrics (WebSocket not implemented; UI also polls process/infra/investigations every 60s, agent status every 4s when active)
- [ ] Metrics throughput benchmarks (not formally tested)
- [ ] Data retention with no data loss (not formally tested)

### Integration Validation
- [ ] Publishes alerts to Redis pub/sub (configured, not verified)
- [ ] Stores investigations in PostgreSQL (repository layer exists; defaults to in-memory)
- [ ] Ingests metrics to QuestDB via ILP (configured, not verified)
- [ ] Executes investigations via agent-manager (integration code exists)
- [ ] HTTP polling delivers metrics to dashboard (implemented, 5s interval)

### Capability Verification
- [x] Monitors CPU, memory, disk, network, GPU, process metrics
- [x] Threshold-based anomaly detection with configurable triggers
- [x] Investigation system with agent-manager integration
- [x] Matrix-themed UI with cyberpunk styling
- [x] 30 investigation scripts available in investigations/active/
- [ ] AI investigation accuracy benchmarks (not measured)
- [ ] Alerts fire within configured thresholds (not formally tested)

## 📝 Implementation Notes

### Design Decisions
**QuestDB over InfluxDB**: Superior performance for time-series
- Alternative considered: InfluxDB (more popular)
- Decision driver: QuestDB 10x faster for our query patterns
- Trade-offs: Less ecosystem support, better performance

**Matrix theme over traditional**: Memorable, engaging UI
- Alternative considered: Grafana-style traditional dashboard
- Decision driver: Differentiation and user engagement
- Trade-offs: Longer development, unique user experience

**Bash for CLI**: Rapid development with curl-based API client
- Original plan: Go CLI for performance
- Actual implementation: Bash script with curl, bc, grep/cut for JSON parsing
- Trade-offs: Fragile JSON parsing (no jq), but fast to develop and no compilation needed

### Known Limitations
- **Cloud Metrics**: Currently local systems only
  - Workaround: Agent installation on cloud instances
  - Future fix: Cloud provider API integration

- **Custom Metrics**: Limited to 6 built-in collectors (CPU, Memory, Network, Disk, Process, GPU)
  - No custom metric ingestion endpoint exists
  - Future fix: Plugin system for collectors

- **WebSocket**: UI type definitions exist but not implemented
  - Dashboard uses HTTP polling (5s current+detailed metrics+timeline, 60s process/infrastructure/investigations, 4s agent status when active)
  - UI activates maintenance state on mount and restores previous state on unmount
  - Future fix: Implement WebSocket server in Go API

- **Missing API Endpoints Referenced by UI**: UI calls endpoints not registered in the API router
  - `/api/v1/metrics/timeline`: UI sparkline history falls back to client-side accumulation
  - `/api/v1/metrics/disk/details`: UI disk detail view cannot fetch partition-level data
  - `/processes/{pid}/kill` (POST): UI ProcessMonitor has kill confirmation dialog but API has no kill endpoint — kill action silently fails
  - Future fix: Implement these endpoints in the Go API

- **CLI Fragility**: Bash CLI uses regex-based JSON parsing (no jq)
  - Can break on unexpected JSON structures
  - Future fix: Rewrite CLI in Go or add jq dependency

- **Storage**: API defaults to in-memory repository
  - PostgreSQL repository interface exists but may not be fully implemented
  - QuestDB integration configured but not directly used by API collectors

- **Authentication**: No auth on API endpoints
  - No API key, token, or session support
  - Future fix: Add middleware authentication

- **Investigation Script API**: Script endpoints (list, get, execute) are placeholders
  - ListScripts returns empty array; GetScript and ExecuteScript return 404
  - 30 scripts exist on disk in investigations/active/ but are not served via API
  - Scripts are executed by the investigation agent directly, not via the API

- **Test Coverage**: test/ directory is empty
  - Tests defined in service.json via test-genie but test phases not populated

### Security Considerations
- **Data Protection**: Metrics encrypted in transit (TLS)
- **Access Control**: Read-only by default, admin for thresholds
- **Audit Trail**: All investigations logged
- **Metric Privacy**: Sensitive data scrubbing options

## 🔗 References

### Documentation
- README.md - Quick start guide
- api/REFACTORING.md - API refactoring notes
- (api/docs/metrics.md - not created)
- (cli/docs/advanced.md - not created)
- (ui/docs/customization.md - not created)

### Related PRDs (planned scenarios, not yet created)
- scenarios/incident-manager/PRD.md - Would consume anomaly events
- scenarios/auto-scaler/PRD.md - Would use metrics for scaling
- scenarios/cost-optimizer/PRD.md - Would analyze resource usage

### External Resources
- [QuestDB Documentation](https://questdb.io/docs/)
- [Prometheus Metric Types](https://prometheus.io/docs/concepts/metric_types/)
- [Matrix Code](https://matrix.fandom.com/wiki/Matrix_code)

---

**Last Updated**: 2026-02-16 (spec-sync pass 4: documented missing process kill endpoint, corrected requirement counts)
**Status**: Implemented, Not Formally Tested
**Owner**: AI Agent - Infrastructure Intelligence Module
**Review Cycle**: Periodic spec-sync before archiving
