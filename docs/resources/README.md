# Vrooli Resource Management System

A comprehensive ecosystem of specialized tools that extend Vrooli's AI capabilities through local services. Rather than building every feature into the core platform, Vrooli dynamically discovers and orchestrates external tools that provide AI, automation, agent, search, and storage capabilities.

## 🎯 Philosophy

The resource system enables **capability emergence** through tool orchestration:
- **AI Services**: Local models, cloud APIs, specialized inference
- **Automation Platforms**: Multi-step workflows, integrations, scheduling  
- **Agent Services**: Web/desktop interaction, autonomous navigation
- **Search Services**: Information retrieval, privacy-respecting search APIs
- **Storage Services**: File management, artifacts, persistent data

This architecture allows Vrooli's three-tier AI system to adapt to whatever resources are available, making the platform highly flexible without requiring all users to run the same infrastructure stack.

## 🚀 Quick Start

### Resource Discovery
```bash
# See what's currently running
./resources/index.sh --action discover

# Check enabled resources
cat ~/.vrooli/service.json | jq '.services'

# List all available resources
./resources/index.sh --action list
```

### Installation
```bash
# Install enabled resources (recommended)
vrooli setup

# Install specific resources
vrooli resource install ollama n8n agent-s2

# Install by category
vrooli resource start-all  # Start all AI resources
```

### Management
```bash
# Resource lifecycle
./resources/index.sh --action install --resources "ollama,n8n" 
./resources/index.sh --action status --resources ollama
./resources/index.sh --action logs --resources n8n
```

---

## 🎯 See Resources in Action

Our scenario system demonstrates real-world resource combinations and generates deployable applications:

### Popular Resource Combinations
- **Document Processing**: Unstructured-IO + Ollama + Qdrant + Vault
  → [Secure Document Processing](../scenarios/secure-document-processing/) ($20k-40k projects)
- **Content Creation**: ComfyUI + Ollama
  → [Campaign Content Studio](../scenarios/campaign-content-studio/) ($8k-20k projects)
- **App Monitoring**: PostgreSQL + Redis + n8n + Node-RED
  → [App Monitor](../scenarios/app-monitor/) ($10k-20k projects)

### Test Resource Integration
```bash
# Run all scenarios using a specific resource
./scenarios/tools/test-by-resource.sh --resource ollama

# Test multi-resource combinations
cd ../scenarios/app-monitor && ./test.sh

# List all available scenarios (dynamically discovered)
vrooli scenario list
```

### Run Scenarios Directly
```bash
# Run any scenario directly without conversion
vrooli scenario run research-assistant

# Or from the scenario directory
cd scenarios/research-assistant
vrooli scenario run <scenario-name>
```

📖 **Full Scenario Documentation**: [Scenario System](../scenarios/)

---

## 🧪 Testing & Examples Architecture

### Where to Find Tests
Our testing system is distributed across multiple layers for comprehensive validation:

#### **Individual Resource Tests**
- **Location**: Resource-local `test/` directories (see each resource README)
- **Purpose**: Test individual resource functionality, health checks, and API endpoints
- **Execution**: Run the resource’s `test/run-tests.sh` (when present) or the resource-specific scripts documented in its README

#### **Multi-Resource Integration Tests**  
- **Location**: `scenarios/scenario-name/test.sh`
- **Purpose**: Test complex business scenarios using multiple resources
- **Examples**: `scenarios/app-monitor/test.sh`
- **Documentation**: [Scenario Testing](../scenarios/README.md)

#### **Resource Unit Tests**
- **Location**: `resources/category/resource/test/phases/test-unit.sh`
- **Purpose**: Validate individual resource functions exist and behave correctly (<60s)
- **Examples**: `resources/unstructured-io/test/phases/test-unit.sh`
- **Documentation**: [Unit Testing Guide](UNIT_TESTING_GUIDE.md) | [Troubleshooting](UNIT_TEST_TROUBLESHOOTING.md)

**Critical Requirements**:
- ✅ Use safe arithmetic: `tests_passed=$((tests_passed + 1))` not `((tests_passed++))`
- ✅ Test function existence with `declare -f function_name`
- ✅ Source configuration before libraries
- ✅ Handle missing libraries gracefully
- ✅ Complete in <60 seconds without external services

### Where to Find Examples
Resource usage examples are organized by complexity and integration level:

#### **Working Code Examples**
- **Location**: `resources/category/resource/examples/`
- **Content**: Real working code samples demonstrating resource usage
- **Structure**: `basic/`, `integration/`, and `scenarios/` subdirectories
- **Examples**: `resources/automation/n8n/examples/webhook-workflow.json`

#### **Business Application Examples**
- **Location**: `scenarios/` (real applications)
- **Content**: Complete working business applications using resource combinations
- **Value**: Demonstrates real-world integration patterns and revenue potential

#### **Test Fixtures**
- **Location**: `__test/fixtures/`
- **Content**: Sample data for testing (audio, documents, images, workflows)
- **Usage**: Shared test data across all resource tests

### Quick Testing Commands
```bash
# Test business scenario
cd ./scenarios/research-assistant && ./test.sh

# Run resource-local tests (example)
cd ./resources/ollama && ./test/integration-test.sh

# Test scenarios using specific resource
./scenarios/tools/test-by-resource.sh --resource ollama
```

### Interface Validation System
Vrooli uses a **three-layer validation system** for resource quality assurance:

- **Layer 1: Syntax Validation** (< 1 second) - Static analysis of manage.sh scripts
- **Layer 2: Behavioral Testing** (< 30 seconds) - Function execution in controlled environment  
- **Layer 3: Integration Testing** (< 5 minutes) - Real-world functionality validation

```bash
# Quick syntax validation
./tools/validate-interfaces.sh --level quick --resource ollama

# Full three-layer validation
./tools/validate-interfaces.sh --level full --resource ollama
```

📖 **Complete Testing Strategy**: [TESTING_STRATEGY.md](TESTING_STRATEGY.md)

### Development Workflow
1. **Explore**: Browse scenarios in `scenarios/` for working examples
2. **Develop**: Create resource following [CLI Framework](cli-framework.md) and [Interface Standards](interface-standards.md)
3. **Unit Test**: Write comprehensive unit tests - see [Unit Testing Guide](UNIT_TESTING_GUIDE.md)
4. **Test**: Use `__test/resources/single/` for individual resource validation  
5. **Validate**: Run three-layer interface validation during development
6. **Integrate**: Build complex workflows following scenario patterns
7. **Deploy**: Run scenarios directly using `vrooli scenario run`

### Unit Testing During Development
**Before writing tests**, verify function names and config variables:
```bash
# Check what functions actually exist
grep -E "^[a-z_]+::[a-z_:]+\(\)" lib/*.sh

# Check configuration variables
grep "^export" config/defaults.sh

# Test your unit test doesn't use dangerous patterns
grep -n '((' test/phases/test-unit.sh
```

**If unit test fails immediately**, check for arithmetic expansion issues:
```bash
# Replace dangerous patterns
sed -i 's/((tests_passed++))/tests_passed=$((tests_passed + 1))/g' test/phases/test-unit.sh
```

📖 **Detailed Help**: [Unit Test Troubleshooting](UNIT_TEST_TROUBLESHOOTING.md)

---

# 🧠 AI Resources

## Ollama - Local LLM Inference
**Local AI models for privacy-sensitive tasks and offline inference**

**Use Cases**: 
- Private AI chat and code generation
- Offline inference without cloud dependencies
- Domain-specific model customization via Modelfiles
- Cost-effective local processing

**When to Use**: Privacy-sensitive data, offline environments, cost optimization, custom models  
**Alternative**: OpenRouter/Cloudflare for scale and cutting-edge models

**Quick Example**:
```bash
# Check available models
curl http://localhost:11434/api/tags

# Generate text with base model
curl -X POST http://localhost:11434/api/generate -d '{"model": "llama3.1:8b", "prompt": "Explain AI"}'

# Create specialized model (e.g., customer support agent)
echo 'FROM llama3.1:8b
SYSTEM "You are a professional customer support agent. Always be helpful and follow company policies."' > /tmp/support-agent
ollama create support-agent -f /tmp/support-agent
```
📖 **Details**: [resources/ai/ollama/README.md](../../resources/ollama/README.md)

## Whisper - Speech-to-Text
**OpenAI Whisper for audio transcription and voice interfaces**

**Use Cases**:
- Meeting and video transcription
- Voice command interfaces
- Audio content analysis
- Accessibility features

**When to Use**: Local audio processing, privacy-sensitive transcription  
**Alternative**: Cloud speech APIs for scale and real-time processing

**Quick Example**:
```bash
# Transcribe audio file
curl -X POST http://localhost:8090/transcribe -F "audio=@meeting.wav"
```
📖 **Details**: [resources/ai/whisper/README.md](../../resources/whisper/README.md)

## Unstructured-IO - Document Processing
**AI-powered document parsing and content extraction**

**Use Cases**:
- PDF and document text extraction
- Structured data extraction from unstructured documents
- Document preprocessing for AI workflows
- Content analysis and indexing

**When to Use**: Document processing pipelines, content extraction, AI document analysis  
**Alternative**: Manual parsing, cloud document APIs

**Quick Example**:
```bash
# Process document
curl -X POST http://localhost:11450/general/v0/general \
  -F "files=@document.pdf" \
  -F "strategy=fast"
```
📖 **Details**: [resources/ai/unstructured-io/README.md](../../resources/unstructured-io/README.md)

## ComfyUI - AI Image Generation
**Workflow-based AI image generation and manipulation**

**Use Cases**:
- AI art generation and editing
- Image processing pipelines
- Visual content creation
- Style transfer and enhancement

**When to Use**: Complex image workflows, local processing, custom pipelines  
**Alternative**: Cloud APIs for simple generation, web UIs for ease of use

**Quick Example**:
```bash
# Check ComfyUI status
curl http://localhost:5679/

# Submit workflow  
curl -X POST http://localhost:5679/api/prompt -H "Content-Type: application/json" -d @workflow.json
```
📖 **Details**: [resources/ai/comfyui/README.md](../../resources/comfyui/README.md)

---

# ⚙️ Automation Resources

## n8n - Visual Workflow Automation
**Business process automation with 300+ integrations and host system access**

**Use Cases**:
- API orchestration and data transformation
- Scheduled business workflows  
- SaaS service integrations
- Complex data pipelines

**When to Use**: Business processes, scheduled workflows, external API integration  
**Alternative**: Node-RED for real-time/IoT

**Quick Example**:
```bash
# Access n8n editor
open http://localhost:5678

# Execute workflow via API
curl -X POST http://localhost:5678/webhook/my-workflow
```
📖 **Details**: [resources/automation/n8n/README.md](../../resources/n8n/README.md)

## Node-RED - Real-time Flow Programming
**Event-driven automation with IoT focus and real-time processing**

**Use Cases**:
- Real-time system monitoring and dashboards
- IoT device integration and sensor data
- Event-driven automation and alerts
- System integration and API development

**When to Use**: Real-time processing, IoT integration, live dashboards, system monitoring  
**Alternative**: n8n for business workflows, custom code for complex logic

**Quick Example**:
```bash
# Access Node-RED editor
open http://localhost:1880

# Check Node-RED flows API
curl http://localhost:1880/flows | jq .
```
📖 **Details**: [resources/automation/node-red/README.md](../../resources/node-red/README.md)

## Huginn - Agent-based Event Processing
**Intelligent web monitoring and data aggregation**

**Use Cases**:
- Website change monitoring
- Data scraping and aggregation
- Intelligent alerts and notifications
- RSS/feed processing

**When to Use**: Web monitoring, content tracking, intelligent data collection  
**Alternative**: Node-RED for real-time processing, n8n for API-based workflows

**Quick Example**:
```bash
# Access Huginn interface
open http://localhost:4111

# Create monitoring agent via API
curl -X POST http://localhost:4111/agents -d @agent_config.json
```
📖 **Details**: [resources/automation/huginn/README.md](../../resources/huginn/README.md)

---

# 🤖 Agent Resources

## Agent-S2 - Autonomous Screen Interaction
**AI-powered desktop and web automation with visual reasoning**

**Use Cases**:
- Public website navigation (anti-bot handling)
- Desktop application automation
- Visual UI interaction and testing
- Adaptive automation requiring decision-making

**When to Use**: Public/complex websites, desktop apps, visual reasoning needed, unknown/changing interfaces  
**Alternative**: Browserless for predictable/internal sites, direct APIs when available

**Quick Example**:
```bash
# Take screenshot
curl -X POST "http://localhost:4113/screenshot?format=png&response_format=binary" -o screenshot.png

# Test with management script (recommended)
resource-agent-s2 usage --usage-type screenshot

# Core automation examples
curl -X POST http://localhost:4113/mouse/click -d '{"x": 500, "y": 300}'
curl -X POST http://localhost:4113/keyboard/type -d '{"text": "Hello World"}'
```
📖 **Details**: [resources/agents/agent-s2/README.md](../../resources/agent-s2/README.md)

## Browserless - Chrome-as-a-Service
**Fast, lightweight web automation for trusted environments**

**Use Cases**:
- Internal dashboard screenshots
- PDF generation from web content
- Local development server automation
- High-volume web scraping (trusted sites)

**When to Use**: Internal/local services, known page structures, high-speed automation  
**Alternative**: Agent-S2 for public sites or visual reasoning, direct HTTP for APIs

**Quick Example**:
```bash
# Screenshot local dashboard
curl -X POST http://localhost:4110/chrome/screenshot -H "Content-Type: application/json" -d '{"url": "http://localhost:3000/dashboard"}'

# Generate PDF report
curl -X POST http://localhost:4110/chrome/pdf -H "Content-Type: application/json" -d '{"url": "http://localhost:8080/report"}'
```
📖 **Details**: [resources/agents/browserless/README.md](../../resources/browserless/README.md)

## Claude Code - AI Development Assistant  
**Anthropic's CLI for AI-powered development and code analysis**

**Use Cases**:
- AI pair programming and code review
- Automated code analysis and refactoring
- Development workflow assistance
- Code documentation generation

**When to Use**: Development tasks, code analysis, refactoring assistance  
**Alternative**: Direct IDE integration, manual code review

**Quick Example**:
```bash
# Analyze code
claude-code analyze src/

# Generate documentation
claude-code document --file src/main.ts
```
📖 **Details**: [resources/agents/claude-code/README.md](../../resources/claude-code/README.md)

---

# 🔍 Search Resources

## SearXNG - Privacy-Respecting Metasearch
**Aggregated search results from multiple engines without tracking**

**Use Cases**:
- Privacy-focused web searches without tracking
- Aggregated results from Google, Bing, DuckDuckGo, Startpage
- Local search API for AI agents and automation
- Research and information gathering workflows

**When to Use**: Privacy-sensitive searches, local search API needs, avoiding tracking  
**Alternative**: Direct search engine APIs for specific providers, cloud search services

**Quick Example**:
```bash
# Access SearXNG search interface
open http://localhost:9200

# Search via API  
curl "http://localhost:9200/search?q=vrooli+ai&format=json"
```
📖 **Details**: [resources/search/searxng/README.md](../../resources/searxng/README.md)

---

# 💾 Storage Resources

## MinIO - S3-Compatible Object Storage
**Local object storage with S3 API compatibility**

**Use Cases**:
- AI model and artifact storage
- File uploads and downloads
- Backup and archival storage
- Multi-application file sharing

**When to Use**: Local file storage, S3-compatible needs, artifact management  
**Alternative**: Cloud storage for scale, local filesystem for simplicity

**Quick Example**:
```bash
# Access MinIO console
open http://localhost:9001

# Upload file via API
curl -X PUT http://localhost:9000/bucket/file.txt -T ./file.txt
```
📖 **Details**: [resources/storage/minio/README.md](../../resources/minio/README.md)

## QuestDB - Ultra-Fast Time-Series Database
**High-performance time-series database for metrics and analytics**

**Use Cases**:
- AI model performance tracking
- Resource health monitoring
- Workflow execution analytics
- Real-time metrics dashboards

**When to Use**: Time-series data, performance metrics, high-frequency events  
**Alternative**: InfluxDB for specific ecosystems, Prometheus for Kubernetes

**Quick Example**:
```bash
# Query recent AI metrics
curl -G "http://localhost:9010/exec" --data-urlencode "query=SELECT * FROM ai_inference ORDER BY timestamp DESC LIMIT 10"

# Ingest metrics via InfluxDB protocol
echo "cpu,host=server1 usage=45.2 $(date +%s)000000000" | nc localhost 9011
```
📖 **Details**: [resources/storage/questdb/README.md](../../resources/questdb/README.md)

## Vault - Secret Management
**Secure secret storage and dynamic credential management**

**Use Cases**:
- API key and credential storage
- Dynamic database credentials
- Certificate management
- Secure configuration storage

**When to Use**: Production deployments, credential rotation, secure secret access  
**Alternative**: Environment variables for development, cloud secret managers

**Quick Example**:
```bash
# Store secret
curl -X POST http://localhost:8200/v1/secret/data/myapp \
  -H "X-Vault-Token: your-token" \
  -d '{"data": {"api_key": "secret-value"}}'

# Retrieve secret
curl -H "X-Vault-Token: your-token" http://localhost:8200/v1/secret/data/myapp
```
📖 **Details**: [resources/storage/vault/README.md](../../resources/vault/README.md)

## Qdrant - Vector Database
**High-performance vector database for AI embeddings and semantic search**

**Use Cases**:
- Semantic search and similarity matching
- AI embeddings storage and retrieval
- Recommendation systems
- RAG (Retrieval-Augmented Generation) workflows

**When to Use**: AI applications requiring semantic search, embedding storage, similarity matching  
**Alternative**: Simple databases for non-vector data, cloud vector services for scale

**Quick Example**:
```bash
# Create collection
curl -X PUT http://localhost:6333/collections/documents \
  -H "Content-Type: application/json" \
  -d '{"vectors": {"size": 384, "distance": "Cosine"}}'

# Search vectors
curl -X POST http://localhost:6333/collections/documents/points/search \
  -H "Content-Type: application/json" \
  -d '{"vector": [0.1, 0.2, 0.3], "limit": 5}'
```
📖 **Details**: [resources/storage/qdrant/README.md](../../resources/qdrant/README.md)

## PostgreSQL - Relational Database
**Production-ready PostgreSQL database for structured data**

**Use Cases**:
- Application data storage
- Relational data modeling
- Complex queries and transactions
- Business logic and reporting

**When to Use**: Structured data, complex relationships, ACID transactions, SQL queries  
**Alternative**: QuestDB for time-series, Qdrant for vectors, MinIO for files

**Quick Example**:
```bash
# Connect with psql
psql -h localhost -p 5433 -U postgres -d vrooli

# Query via HTTP (if enabled)
curl -X POST http://localhost:5433/query \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT version();"}'
```
📖 **Details**: [resources/storage/postgres/README.md](../../resources/postgres/README.md)

## Redis - In-Memory Cache
**High-performance in-memory data store for caching and pub/sub**

**Use Cases**:
- Application caching and session storage
- Real-time messaging and pub/sub
- Task queue management
- Rate limiting and counters

**When to Use**: Caching, real-time features, session management, high-speed data access  
**Alternative**: PostgreSQL for persistent data, QuestDB for time-series metrics

**Quick Example**:
```bash
# Set and get value
redis-cli -h localhost -p 6380 SET mykey "myvalue"
redis-cli -h localhost -p 6380 GET mykey

# Pub/sub messaging
redis-cli -h localhost -p 6380 PUBLISH mychannel "Hello World"
```
📖 **Details**: [resources/storage/redis/README.md](../../resources/redis/README.md)

---

# 🔗 Integration Patterns

## Resource Selection Guide

**Choose by Use Case**:
- **Real-time monitoring**: Node-RED + Agent-S2 + MinIO
- **Business automation**: n8n + Browserless + external APIs  
- **AI processing**: Ollama + Whisper + ComfyUI + MinIO
- **Domain-specific AI**: Ollama Modelfiles + Qdrant + MinIO (custom chatbots, specialized assistants)
- **Information gathering**: SearXNG + Huginn + Agent-S2 + MinIO
- **Development workflows**: Claude Code + version control
- **Interactive AI interfaces**: Ollama + Agent-S2 + ComfyUI + Whisper

## 🎨 UI Development Patterns

**Automated interface generation for AI workflows**

Node-RED provides powerful capabilities for creating interactive user interfaces that orchestrate multi-resource AI workflows. Rather than building static UIs, this platform enables **automated UI generation** based on your AI service combinations.

### **Node-RED: Real-time AI Dashboards**  
```javascript
// Live AI monitoring and interaction
- WebSocket-based real-time updates
- Resource health monitoring dashboards
- Event-driven AI triggers
- Simple drag-and-drop configuration  
```

**When to Use**: Real-time monitoring, IoT integration, rapid prototyping, system dashboards

### **Multi-Modal AI Assistant Example**

A complete **voice-to-visual-to-action** workflow combining:
- **Whisper**: Voice command processing
- **Ollama**: Intent understanding and response generation  
- **ComfyUI**: Visual content creation
- **Agent-S2**: Screen automation and file management

```bash
# Test the complete workflow
./resources/tests/run.sh --scenarios "scenario=analytics-dashboard"

# Access the interfaces
open http://localhost:1880  # Node-RED real-time dashboard  
```

**Revenue Potential**: $10,000-25,000 per project for accessibility and enterprise productivity solutions

📖 **Detailed Guides**:
- [Node-RED Dashboard Creation](automation/node-red/docs/DASHBOARD_CREATION.md)

## Configuration Management

**Resource Configuration**: `~/.vrooli/service.json`
```json
{
  "services": {
    "ai": {
      "ollama": { "enabled": true, "baseUrl": "http://localhost:11434" }
    },
    "automation": {
      "n8n": { "enabled": true, "baseUrl": "http://localhost:5678" }
    }
  }
}
```

## Resource Categories

Install by logical groupings:
- `ai-only` - All AI resources (ollama, whisper, comfyui)
- `automation-only` - Workflow platforms (n8n, node-red, huginn)  
- `agents-only` - Interaction agents (agent-s2, browserless, claude-code)
- `search-only` - Search and information retrieval (searxng)
- `storage-only` - Storage solutions (minio, questdb, vault, qdrant)
- `essential` - Core set (ollama, n8n, agent-s2, minio, questdb)
- `enabled` - Only enabled resources (default)

---

# 🛠️ Management & Troubleshooting

## Health Monitoring
```bash
# Check all resource health
./resources/index.sh --action discover

# Resource-specific status
resource-ollama status
```

## Common Issues

**Resource Not Available**:
1. Check if enabled: `jq '.services.ai.ollama.enabled' ~/.vrooli/service.json`
2. Verify running: `docker ps | grep ollama` 
3. Check logs: `resource-ollama logs`

**Port Conflicts**:
```bash
# Find port usage
sudo lsof -i :5678

# Check port registry
./resources/port_registry.sh --action list
```

**Docker Issues**:
```bash
# Check Docker access
sudo usermod -aG docker $USER && newgrp docker

# Verify service
sudo systemctl status docker
```

**Service Status Issues**:
```bash
# SearXNG - Service may not be accessible (connection refused)
# If SearXNG search API fails, check service status:
resource-searxng status

# QuestDB - Port conflict resolved
# QuestDB now runs on corrected ports to avoid MinIO conflict:
curl http://localhost:9010/  # HTTP/REST API (was 9009)
echo "test_metric value=1" | nc localhost 9011  # InfluxDB line protocol (was 9003)
```

## Getting Help

- **Individual Resource Issues**: See specific resource README files
- **General Setup**: [Vrooli CLI documentation](../README.md)
- **Integration Questions**: [Architecture Overview](../ARCHITECTURE_OVERVIEW.md)

---

**🎯 This resource ecosystem enables Vrooli's AI tiers to dynamically adapt and evolve their capabilities based on available tools, making the platform both powerful and flexible for diverse use cases.**
