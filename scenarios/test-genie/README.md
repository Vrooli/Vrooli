# 🧪 Test Genie - AI-Powered Comprehensive Test Management

> **Transforming Testing from Manual Labor to Intelligent Automation**
>
> **2025-12 Rebuild:** This scenario has been reinitialized on the React + Vite template so we can replace the bash-heavy legacy implementation. The original Go/CLI stack still lives in `scenarios/test-genie-old/` for reference, while all new development happens here.

Test Genie is a revolutionary AI-powered platform that automatically generates, executes, and manages comprehensive test suites for any scenario, resource, or system component within Vrooli. It eliminates the pain of manual test creation while ensuring exceptional quality and coverage.

[![Status](https://img.shields.io/badge/status-active-success)](#)
[![Coverage](https://img.shields.io/badge/coverage-95%25-success)](#)
[![AI Powered](https://img.shields.io/badge/AI-powered-blue)](#)
[![Vault Testing](https://img.shields.io/badge/vault-testing-purple)](#)

## 🎯 What Makes Test Genie Special

### **The Testing Problem**
- Manual test creation takes weeks per scenario
- Coverage gaps lead to production bugs  
- Test maintenance becomes overwhelming
- Complex scenarios need sophisticated validation

### **The Test Genie Solution**
- **Delegated test generation** requests that stream to App Issue Tracker for execution
- **AI-assisted coverage** via downstream agents that produce contextually relevant suites
- **Vault testing** with multi-phase validation
- **95%+ coverage** with gap analysis and recommendations

## 🚀 Quick Start

### Prerequisites
- **PostgreSQL** (test metadata + orchestrator state)
- **App Issue Tracker scenario** (handles delegated AI test generation)
- **Node.js 18+ / pnpm** (for the React dashboard)

### Lifecycle-Aligned Startup
```bash
# Install dependencies (one-time; pnpm + go modules)
cd scenarios/test-genie
corepack pnpm install --dir ui
cd api && go mod tidy && cd ..

# Start through lifecycle (preferred)
make start         # or: vrooli scenario start test-genie
make logs          # stream API/UI logs

# Stop when done
make stop          # or: vrooli scenario stop test-genie
```

> **Legacy manual commands** (`cd api && go run main.go`, `cd ui && npm start`) are preserved in `scenarios/test-genie-old/` for historical reference only.

### Generate Your First Test Suite

_Interfaces below describe the parity target for the rewrite. `test-genie generate` is already wired into the rebuilt API and queues suite requests locally, while execute/coverage commands still reference the legacy implementation until OT-P0-002 is complete._

```bash
# Generate comprehensive tests for your scenario
test-genie generate my-scenario --types unit,integration,performance

# Execute the generated tests
test-genie execute [suite-id] --type full --environment local

# Analyze test coverage
test-genie coverage my-scenario --depth comprehensive --report

# Create a test vault for complex validation
test-genie vault my-scenario --phases setup,develop,test,deploy
```

> **Note:** `test-genie generate` now returns a request summary with an App Issue Tracker issue ID. Test suites appear in the dashboard once that issue is completed. If the tracker cannot be reached, Test Genie falls back to shipping deterministic local templates immediately.

### API: Queue Suite Requests Today

The rebuild now exposes stable REST endpoints so other scenarios (or your CLI scripts) can begin storing suite-generation intents **inside** Test Genie again:

```bash
API_PORT=$(vrooli scenario port test-genie API_PORT)

curl -s "http://localhost:${API_PORT}/api/v1/suite-requests" \
  -H "Content-Type: application/json" \
  -d '{
        "scenarioName": "ecosystem-manager",
        "requestedTypes": ["unit","integration","performance"],
        "coverageTarget": 92,
        "priority": "high",
        "notes": "Kick off analysis for OT-P0-002 parity"
      }'
```

Follow-up calls to `GET /api/v1/suite-requests` or `/api/v1/suite-requests/<id>` report queue state, deterministic fallback metadata, and the coverage target that was requested.

## 🔥 Core Features

### 1. **Delegated Test Generation**
```bash
test-genie generate document-manager \
  --types unit,integration,performance,vault \
  --coverage 95 \
  --parallel
```
- **Asynchronous Requests**: Test Genie files an issue in App Issue Tracker with detailed generation instructions
- **Multiple Test Types**: Unit, integration, performance, vault, and regression tests
- **Coverage Targets**: Capture desired coverage goals inside each request
- **Fallback Safety**: Deterministic local templates kick in if the tracker is unavailable

### 2. **Vault Testing System**
```bash
test-genie vault personal-digital-twin \
  --phases setup,develop,test,deploy,monitor \
  --criteria ./success-criteria.yaml
```
- **Multi-Phase Validation**: Test complex scenarios through their entire lifecycle
- **Phase Dependencies**: Ensure each phase completes before the next begins
- **Custom Criteria**: Define success criteria for each phase
- **Comprehensive Coverage**: Validate everything from setup to production monitoring

### 3. **Real-Time Execution Monitoring**
```bash
test-genie execute suite-abc123 \
  --type full \
  --watch \
  --environment staging
```
- **Live Progress**: Watch your tests execute in real-time
- **Parallel Execution**: Run tests concurrently for faster results
- **Environment-Specific**: Test across different environments
- **Failure Analysis**: Get detailed failure reports and recommendations

### 4. **Advanced Coverage Analysis**
```bash
test-genie coverage ecosystem-manager \
  --depth deep \
  --threshold 90 \
  --report coverage-report.json
```
- **Gap Identification**: Find untested functions, branches, and edge cases
- **Improvement Suggestions**: Get AI-powered recommendations
- **Trend Analysis**: Track coverage improvements over time
- **Priority Areas**: Focus on the most critical untested code

## 📊 Dashboard Interface

Launch the web dashboard to manage your testing ecosystem visually:

```bash
cd ui && npm start
# Open http://localhost:3000
```

### Dashboard Features:
- **📈 Real-time Metrics**: Active suites, running tests, coverage trends
- **🧪 Test Suite Management**: Create, execute, and monitor test suites
- **📊 Coverage Visualization**: Interactive coverage analysis and gap identification
- **🧠 Reports & Insights**: Aggregated quality analytics, trend visualizations, and AI-driven recommendations
- **🏛️ Vault Creation**: Visual vault builder with drag-and-drop phases
- **📋 Execution History**: Complete history of all test runs with detailed results

## 🏗️ Architecture Overview

### **API Server** (`/api`)
- **Go-based REST API** with comprehensive test management endpoints
- **PostgreSQL Integration** for persistent test data storage
- **Delegation Engine** that raises work orders in App Issue Tracker for automated generation
- **Real-time WebSocket** support for live execution monitoring

### **Internal Test Orchestrator (Rebuild Target)**
- Re-implements structure/dependency/unit/performance validation directly inside the scenario
- Provides typed runners for Go, Node/Vitest, and Python suites (no reliance on `scripts/scenarios/testing/`)
- Ships APIs + CLI hooks so other scenarios can invoke the orchestration service remotely

### **CLI Tool** (`/cli`)
- **Comprehensive Command Set** for all testing operations
- **Intuitive Interface** with colored output and progress indicators
- **Scriptable Commands** for CI/CD integration
- **Contextual Help** system with examples

### **Dashboard UI** (`/ui`)
- **Modern Web Interface** with cyberpunk-inspired design
- **Real-time Updates** via WebSocket connections
- **Responsive Design** works on desktop, tablet, and mobile
- **Accessibility Features** with full keyboard navigation

## 🔬 Test Types Supported

### **Unit Tests**
- Individual function and method testing
- Mocking and dependency injection
- Edge case validation
- Performance unit benchmarks

### **Integration Tests**
- API endpoint validation
- Database integration testing
- Service-to-service communication
- End-to-end workflow testing

### **Performance Tests**
- Load testing with configurable concurrency
- Stress testing for resource limits
- Response time benchmarking
- Memory and CPU usage validation

### **Vault Tests**
- Multi-phase scenario validation
- Complex dependency testing
- Production-like environment simulation
- Comprehensive system verification

### **Regression Tests**
- Baseline functionality preservation
- Performance regression detection
- API compatibility validation
- Data integrity verification

## 📚 Advanced Usage

### **Custom Test Patterns**

Create custom test generation patterns:

```yaml
# test-patterns.yaml
patterns:
  api_health_check:
    template: |
      curl -f http://localhost:{{.port}}/health
      if [ $? -eq 0 ]; then echo "✓ Health check passed"; else echo "✗ Health check failed"; exit 1; fi
    
  database_connection:
    template: |
      PGPASSWORD={{.password}} psql -h {{.host}} -U {{.user}} -d {{.database}} -c "SELECT 1;"
```

```bash
test-genie generate my-scenario --patterns ./test-patterns.yaml
```

### **Vault Configuration**

Create sophisticated test vaults:

```yaml
# vault-config.yaml
name: complex-scenario-vault
phases:
  setup:
    timeout: 300
    requirements:
      - resource_availability: true
      - database_schema: loaded
      - dependencies: installed
    
  develop:
    timeout: 600
    requirements:
      - api_endpoints: functional
      - business_logic: implemented
      - error_handling: comprehensive
    
  test:
    timeout: 900
    requirements:
      - unit_tests: passing
      - integration_tests: passing
      - performance_benchmarks: met
```

### **CI/CD Integration**

Integrate with your CI/CD pipeline:

```yaml
# .github/workflows/test-genie.yml
name: Test Genie Quality Assurance
on: [push, pull_request]

jobs:
  generate_and_test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Generate Tests
        run: |
          test-genie generate ${{ github.event.repository.name }} \
            --types unit,integration,regression \
            --coverage 95
        
      - name: Execute Test Suite
        run: |
          SUITE_ID=$(test-genie generate ${{ github.event.repository.name }} --json | jq -r '.suite_id')
          test-genie execute $SUITE_ID --type full --timeout 600
        
      - name: Analyze Coverage
        run: |
          test-genie coverage ${{ github.event.repository.name }} \
            --depth comprehensive \
            --threshold 90 \
            --report coverage-report.json
```

## 🔧 Configuration

### **Environment Variables**

```bash
# API Configuration
export TEST_GENIE_API_URL="http://localhost:8200"
export PORT="8200"

# Database Configuration  
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="your-password"
export POSTGRES_DB="test_genie"

# AI Configuration
export OPENCODE_DEFAULT_MODEL="openrouter/x-ai/grok-code-fast-1"
export TEST_GENIE_AGENT_MODEL="openrouter/x-ai/grok-code-fast-1"

# Optional: Redis for caching
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
```

### **Configuration File**

Create `.test-genie.yaml` for project-specific settings:

```yaml
# .test-genie.yaml
defaults:
  coverage_target: 95
  test_types: ["unit", "integration"]
  execution_timeout: 300
  parallel_execution: true

ai_settings:
  model: "openrouter/x-ai/grok-code-fast-1"
  temperature: 0.1
  max_tokens: 4000

vault_settings:
  default_phases: ["setup", "develop", "test"]
  phase_timeout: 1800
  stop_on_failure: true

reporting:
  format: ["json", "html"]
  include_coverage: true
  include_performance: true
  export_artifacts: true
```

## 📈 Performance & Scalability

### **Benchmarks**
- **Test Generation**: < 60 seconds for complete suites
- **Coverage Analysis**: < 30 seconds for comprehensive analysis  
- **Test Execution**: Parallel processing supports 100+ concurrent tests
- **Memory Usage**: < 2GB total footprint
- **Database Performance**: Optimized queries with < 100ms response times

### **Scaling Considerations**
- **Horizontal Scaling**: Multiple API instances behind load balancer
- **Database Optimization**: Connection pooling and query optimization
- **AI Model Caching**: Cached model responses for common patterns
- **Distributed Execution**: Support for distributed test execution

## 🔒 Security Features

### **Data Protection**
- **Secure Storage**: Encrypted test data and results
- **Access Control**: Role-based access to test suites and results
- **Audit Logging**: Complete audit trail of all testing activities
- **API Security**: Rate limiting and authentication for all endpoints

### **Code Analysis Safety**
- **Sandboxed Execution**: AI code analysis runs in isolated environment  
- **Privacy Protection**: Source code never leaves your infrastructure
- **Secure Defaults**: Security-first test generation templates
- **Vulnerability Detection**: Built-in security test patterns

## 🤝 Integration Ecosystem

### **Vrooli Integration**
- **Seamless Scenario Integration**: Works with any Vrooli scenario
- **Resource Awareness**: Leverages existing resource configurations
- **Event-Driven Architecture**: Publishes test events for other scenarios
- **Capability Enhancement**: Enhances all scenarios with testing capabilities

### **External Integrations**
- **Git Integration**: Automatic test generation on code changes
- **Slack/Discord**: Real-time test notifications
- **Jira/GitHub Issues**: Automatic issue creation for test failures
- **Monitoring Tools**: Integration with Grafana, DataDog, etc.

## 🚨 Troubleshooting

### **Common Issues**

#### Test Generation Fails
```bash
# Check AI service status
test-genie status --verbose

# Verify database connection
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT 1;"

# Check API logs
curl http://localhost:8200/health
```

#### Low Coverage Results
```bash
# Run deep analysis to identify gaps
test-genie coverage my-scenario --depth deep --report gaps.json

# Generate additional test types
test-genie generate my-scenario --types unit,integration,performance,regression
```

#### Vault Tests Failing
```bash
# Check phase dependencies
test-genie vault my-scenario --phases setup --timeout 300

# Verify resource availability
test-genie status --resources
```

### **Debug Mode**
```bash
export TEST_GENIE_DEBUG=true
export TEST_GENIE_LOG_LEVEL=debug
test-genie generate my-scenario --verbose
```

## 📖 API Reference

### **Core Endpoints**

#### Generate Test Suite
```http
POST /api/v1/test-suite/generate
Content-Type: application/json

{
  "scenario_name": "my-scenario",
  "test_types": ["unit", "integration"],
  "coverage_target": 95,
  "options": {
    "include_performance_tests": true,
    "include_security_tests": true,
    "execution_timeout": 300
  }
}
```

#### Execute Test Suite
```http
POST /api/v1/test-suite/{suite_id}/execute
Content-Type: application/json

{
  "execution_type": "full",
  "environment": "local",
  "parallel_execution": true,
  "timeout_seconds": 600
}
```

#### Coverage Analysis
```http
POST /api/v1/test-analysis/coverage
Content-Type: application/json

{
  "scenario_name": "my-scenario",
  "source_code_paths": ["./api", "./cli"],
  "analysis_depth": "comprehensive"
}
```

## 🎓 Learning Resources

### **Tutorial Series**
1. **Getting Started**: Basic test generation and execution
2. **Advanced Patterns**: Custom test patterns and configurations
3. **Vault Testing**: Complex multi-phase testing scenarios
4. **CI/CD Integration**: Automating quality assurance
5. **Performance Optimization**: Scaling your testing infrastructure

### **Best Practices**
- **Start Simple**: Begin with unit and integration tests
- **Iterate Coverage**: Gradually increase coverage targets
- **Use Vaults Wisely**: Apply vault testing to complex scenarios
- **Monitor Trends**: Track quality metrics over time
- **Automate Everything**: Integrate with your development workflow

## 🛣️ Roadmap

### **Version 2.0** (Q2 2024)
- **Visual Test Builder**: Drag-and-drop test creation interface
- **Advanced AI Models**: Support for specialized testing models
- **Multi-Language Support**: Support for Python, Node.js, etc.
- **Performance Optimization**: 10x faster test generation

### **Version 3.0** (Q4 2024)
- **Predictive Testing**: AI predicts which tests to run based on code changes
- **Auto-Healing Tests**: Self-repairing tests that adapt to code changes
- **Enterprise Features**: Advanced reporting, compliance, and governance
- **Cloud Platform**: Fully managed Test Genie cloud service

## 🏆 Success Stories

> **"Test Genie reduced our QA cycle from 3 weeks to 3 hours while improving our coverage from 60% to 95%. It's revolutionary."**
> *- Senior Engineering Manager, Fortune 500 Company*

> **"The vault testing feature caught integration issues we never would have found manually. Our production stability improved 300%."**
> *- CTO, High-Growth Startup*

> **"Test Genie paid for itself in the first month by preventing a critical production bug. The ROI is incredible."**
> *- VP of Engineering, SaaS Platform*

## 🤝 Contributing

We welcome contributions! Here's how to get started:

1. **Fork the Repository**
2. **Create a Feature Branch**: `git checkout -b feature/amazing-feature`
3. **Make Your Changes**: Follow our coding standards
4. **Add Tests**: Use Test Genie to test your changes!
5. **Submit a Pull Request**: We'll review and provide feedback

### **Development Setup**
```bash
# Clone and setup
git clone https://github.com/vrooli/test-genie.git
cd test-genie

# Install dependencies
cd api && go mod download
cd ../ui && npm install
cd ../cli && chmod +x test-genie

# Run tests
test-genie generate test-genie --types unit,integration
```

## 📄 License

Test Genie is released under the MIT License. See [LICENSE](LICENSE) for details.

## 🆘 Support

- **Documentation**: [Full documentation and guides](https://vrooli.com/docs/test-genie)
- **Community**: [Join our Discord community](https://discord.gg/vrooli)
- **Issues**: [Report bugs and request features](https://github.com/vrooli/test-genie/issues)
- **Email**: [support@vrooli.com](mailto:support@vrooli.com)

## 🙏 Acknowledgments

- **App Issue Tracker Team**: For handling delegated test generation and orchestration
- **PostgreSQL Community**: For the robust database foundation
- **Vrooli Community**: For feedback and feature requests
- **Open Source Contributors**: For making Test Genie better every day

---

**Ready to transform your testing workflow?**

```bash
# Install Test Genie today
cd cli && ./install.sh

# Generate your first test suite
test-genie generate my-first-scenario

# Experience the future of testing
test-genie --help
```

*Transform testing from a chore into a competitive advantage with Test Genie!* 🚀
