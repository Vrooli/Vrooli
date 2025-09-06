# API Manager

The **API Manager** scenario provides automated API analysis, security auditing, and improvement capabilities for all Vrooli scenarios. It creates a permanent intelligence multiplier that ensures every API in the Vrooli ecosystem becomes more secure, performant, and well-documented over time.

## 🎯 Core Capability

This scenario adds the **permanent capability** to automatically audit, improve, and maintain the quality of all scenario APIs across the entire Vrooli platform. It provides:

- **Security vulnerability detection** and automated fixes
- **Performance baseline measurement** and optimization
- **OpenAPI documentation generation** for all APIs
- **Standards enforcement** across all scenarios
- **Cross-scenario compatibility** analysis

## 🚀 Quick Start

### Prerequisites
- PostgreSQL running (for API metadata storage)
- Ollama with llama3.1:8b model (for AI-powered analysis)
- N8n (for automated workflows)

### Installation & Setup

1. **Install the scenario:**
   ```bash
   vrooli scenario run api-manager
   ```

2. **Install the CLI tool:**
   ```bash
   cd scenarios/api-manager/cli
   ./install.sh
   ```

3. **Verify installation:**
   ```bash
   api-manager health
   ```

### Basic Usage

```bash
# Check system status
api-manager status

# Discover all scenario APIs
api-manager discover

# List managed scenarios  
api-manager list scenarios

# Scan a specific scenario for issues
api-manager scan my-scenario

# View all vulnerabilities
api-manager vulnerabilities

# View vulnerabilities for specific scenario
api-manager vulnerabilities my-scenario
```

## 📊 Web Dashboard

Access the web interface at: http://localhost:8420

The dashboard provides:
- **System overview** with key metrics
- **Scenario management** with one-click scanning
- **Vulnerability tracking** with severity indicators
- **Real-time status** monitoring

## 🔧 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | System health check |
| GET | `/api/v1/scenarios` | List all managed scenarios |
| GET | `/api/v1/scenarios/{name}` | Get scenario details |
| POST | `/api/v1/scenarios/{name}/scan` | Trigger scenario scan |
| GET | `/api/v1/scenarios/{name}/endpoints` | List scenario endpoints |
| GET | `/api/v1/vulnerabilities` | Get all vulnerabilities |
| GET | `/api/v1/vulnerabilities/{scenario}` | Get scenario vulnerabilities |
| POST | `/api/v1/system/discover` | Discover new scenarios |
| GET | `/api/v1/system/status` | System status and statistics |

## 🛡️ Security Analysis

The API Manager performs comprehensive security analysis:

### Automated Detection
- **SQL injection** vulnerabilities
- **Input validation** gaps  
- **CORS misconfigurations**
- **Missing rate limiting**
- **Authentication/authorization** weaknesses
- **Information disclosure** risks

### AI-Powered Analysis
Uses Ollama's llama3.1:8b model to provide:
- **Contextual vulnerability analysis**
- **Specific fix recommendations**
- **Code pattern recognition**
- **Best practice suggestions**

## 🔄 Automated Workflows

### N8n Workflows

1. **API Scanner** (`api-scanner.json`)
   - Discovers API files in scenarios
   - Catalogs endpoints and metadata
   - Triggers security analysis

2. **Security Auditor** (`security-auditor.json`)
   - Performs automated security scans
   - Uses AI for deep code analysis
   - Generates actionable recommendations

3. **Fix Generator** (`fix-generator.json`) *[Coming Soon]*
   - Generates automated security fixes
   - Validates fixes in isolated environments
   - Applies fixes with rollback capability

### Workflow Triggers
- **Manual execution** via N8n interface
- **API triggers** via webhook endpoints
- **Scheduled scans** (configurable intervals)
- **Event-driven** (on scenario updates)

## 📈 Intelligence Amplification

### Compound Learning
- **Pattern Recognition**: Learns successful API patterns across scenarios
- **Security Knowledge Base**: Accumulates vulnerability signatures and fixes
- **Performance Optimization**: Builds reusable performance improvements
- **Documentation Standards**: Enforces consistent API documentation

### Cross-Scenario Benefits
- **New scenarios** inherit established best practices
- **Existing scenarios** benefit from continuous improvements
- **Integration patterns** become standardized and reliable
- **Security vulnerabilities** are prevented proactively

## 🔗 Integration with Other Scenarios

### As a Service Provider
Other scenarios can leverage API Manager via:
```bash
# From scenario workflows
resource-n8n execute-workflow "api-scanner" '{"scenario": "my-scenario"}'

# From CLI tools  
api-manager scan my-scenario

# Via direct API calls
curl -X POST http://localhost:8421/api/v1/scenarios/my-scenario/scan
```

### As a Capability Multiplier
API Manager enhances scenarios like:
- **scenario-improver**: Uses API analysis for targeted improvements
- **agent-dashboard**: Monitors API health across all managed agents
- **security-hardening**: Applies systematic security improvements

## 📚 Database Schema

The API Manager uses PostgreSQL to store:
- **Scenario inventory** and metadata
- **API endpoint** catalog
- **Vulnerability scans** and results
- **Performance metrics** and baselines
- **Applied fixes** and rollback information
- **Improvement recommendations**

## 🛠️ Development

### Local Development
```bash
# Start database and dependencies
vrooli resource postgres start
vrooli resource ollama start  
vrooli resource n8n start

# Run API server
cd api && go run main.go

# Run web interface
cd ui && npm start

# Test CLI
cd cli && go run main.go health
```

### Adding New Analysis Types
1. Create new N8n workflow in `initialization/automation/n8n/`
2. Add corresponding API endpoints in `api/main.go`
3. Update database schema if needed
4. Add CLI commands for new functionality

## 🎯 Success Metrics

### Current Status
- [x] **API Discovery**: Automatically finds and catalogs scenario APIs
- [x] **Database Integration**: Stores all API metadata and scan results
- [x] **Security Scanning**: Detects common vulnerabilities
- [x] **Web Dashboard**: Provides comprehensive management interface
- [x] **CLI Tools**: Full command-line management capability

### Next Milestones
- [ ] **Automated Fix Generation**: AI-powered security fix creation
- [ ] **OpenAPI Documentation**: Auto-generate comprehensive API docs
- [ ] **Performance Monitoring**: Real-time API performance tracking
- [ ] **Integration Testing**: Cross-scenario API compatibility testing

## 💡 Future Enhancements

- **Real-time monitoring** dashboards with alerts
- **Automated client SDK** generation for scenarios
- **Load testing** automation for performance validation
- **API versioning** strategy enforcement
- **Compliance reporting** (OWASP, security standards)

---

**The API Manager creates permanent intelligence that compounds over time - every API improvement it makes enhances the capabilities of all future scenarios forever.**