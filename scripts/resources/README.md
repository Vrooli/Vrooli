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
./scripts/resources/index.sh --action discover

# Check enabled resources
cat ~/.vrooli/resources.local.json | jq '.services'

# List all available resources
./scripts/resources/index.sh --action list
```

### Installation
```bash
# Install enabled resources (recommended)
./scripts/main/setup.sh --target native-linux --resources enabled

# Install specific resources
./scripts/main/setup.sh --target native-linux --resources "ollama,n8n,agent-s2"

# Install by category
./scripts/main/setup.sh --target native-linux --resources ai-only
```

### Management
```bash
# Resource lifecycle
./scripts/resources/index.sh --action install --resources "ollama,n8n" 
./scripts/resources/index.sh --action status --resources ollama
./scripts/resources/index.sh --action logs --resources n8n
```

---

# 🧠 AI Resources

## Ollama - Local LLM Inference
**Local AI models for privacy-sensitive tasks and offline inference**

**Use Cases**: 
- Private AI chat and code generation
- Offline inference without cloud dependencies
- Custom model fine-tuning and experimentation
- Cost-effective local processing

**When to Use**: Privacy-sensitive data, offline environments, cost optimization, custom models  
**Alternative**: OpenRouter/Cloudflare for scale and cutting-edge models

**Quick Example**:
```bash
# Check available models
curl http://localhost:11434/api/tags

# Generate text
curl -X POST http://localhost:11434/api/generate -d '{"model": "llama3.1:8b", "prompt": "Explain AI"}'
```
📖 **Details**: [scripts/resources/ai/ollama/README.md](ai/ollama/README.md)

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
📖 **Details**: [scripts/resources/ai/whisper/README.md](ai/whisper/README.md)

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
curl http://localhost:8188/

# Submit workflow
curl -X POST http://localhost:8188/api/v1/queue -d @workflow.json
```
📖 **Details**: [scripts/resources/ai/comfyui/README.md](automation/comfyui/README.md)

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
**Alternative**: Node-RED for real-time/IoT, Windmill for code-first approach

**Quick Example**:
```bash
# Access n8n editor
open http://localhost:5678

# Execute workflow via API
curl -X POST http://localhost:5678/webhook/my-workflow
```
📖 **Details**: [scripts/resources/automation/n8n/README.md](automation/n8n/README.md)

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

# Check resource monitoring API
curl http://localhost:1880/api/resources/status | jq .
```
📖 **Details**: [scripts/resources/automation/node-red/README.md](automation/node-red/README.md)

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
📖 **Details**: [scripts/resources/automation/huginn/README.md](agents/huginn/README.md)

## Windmill - Code-first Workflows
**Developer-focused workflow automation with script orchestration**

**Use Cases**:
- CI/CD pipeline automation
- Developer workflow orchestration  
- Script and code execution
- Infrastructure automation

**When to Use**: Developer workflows, CI/CD, infrastructure automation, code-heavy tasks  
**Alternative**: n8n for visual workflows, direct scripting for simple tasks

**Quick Example**:
```bash
# Access Windmill interface
open http://localhost:5681

# Execute script
curl -X POST http://localhost:5681/api/jobs/run -d '{"script": "my_script"}'
```
📖 **Details**: [scripts/resources/automation/windmill/README.md](automation/windmill/README.md)

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
# Take screenshot and analyze
curl -X POST http://localhost:4113/ai/task -d '{"task": "take a screenshot"}'

# Automate web navigation  
curl -X POST http://localhost:4113/ai/task -d '{"task": "go to google.com and search for cats"}'
```
📖 **Details**: [scripts/resources/agents/agent-s2/README.md](agents/agent-s2/README.md)

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
curl -X POST http://localhost:4110/screenshot -d '{"url": "http://localhost:3000/dashboard"}'

# Generate PDF report
curl -X POST http://localhost:4110/pdf -d '{"url": "http://localhost:8080/report"}'
```
📖 **Details**: [scripts/resources/agents/browserless/README.md](agents/browserless/README.md)

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
📖 **Details**: [scripts/resources/agents/claude-code/README.md](agents/claude-code/README.md)

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
open http://localhost:8100

# Search via API  
curl "http://localhost:8100/search?q=vrooli+ai&format=json"
```
📖 **Details**: [scripts/resources/search/searxng/README.md](search/searxng/README.md)

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
📖 **Details**: [scripts/resources/storage/minio/README.md](storage/minio/README.md)

---

# 🔗 Integration Patterns

## Resource Selection Guide

**Choose by Use Case**:
- **Real-time monitoring**: Node-RED + Agent-S2 + MinIO
- **Business automation**: n8n + Browserless + external APIs  
- **AI processing**: Ollama + Whisper + ComfyUI + MinIO
- **Information gathering**: SearXNG + Huginn + Agent-S2 + MinIO
- **Development workflows**: Claude Code + Windmill + version control

## Configuration Management

**Resource Configuration**: `~/.vrooli/resources.local.json`
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
- `automation-only` - Workflow platforms (n8n, node-red, windmill, huginn)  
- `agents-only` - Interaction agents (agent-s2, browserless, claude-code)
- `search-only` - Search and information retrieval (searxng)
- `storage-only` - Storage solutions (minio, ipfs, rclone)
- `essential` - Core set (ollama, n8n, agent-s2, minio)
- `enabled` - Only enabled resources (default)

---

# 🛠️ Management & Troubleshooting

## Health Monitoring
```bash
# Check all resource health
./scripts/resources/index.sh --action discover

# Resource-specific status
./scripts/resources/ai/ollama/manage.sh --action status
```

## Common Issues

**Resource Not Available**:
1. Check if enabled: `jq '.services.ai.ollama.enabled' ~/.vrooli/resources.local.json`
2. Verify running: `docker ps | grep ollama` 
3. Check logs: `./scripts/resources/ai/ollama/manage.sh --action logs`

**Port Conflicts**:
```bash
# Find port usage
sudo lsof -i :5678

# Check port registry
./scripts/resources/port-registry.sh --action list
```

**Docker Issues**:
```bash
# Check Docker access
sudo usermod -aG docker $USER && newgrp docker

# Verify service
sudo systemctl status docker
```

## Getting Help

- **Individual Resource Issues**: See specific resource README files
- **General Setup**: [scripts/main/setup.sh documentation](../main/README.md)
- **Integration Questions**: [docs/architecture/ai-resource-integration-plan.md](../../docs/architecture/ai-resource-integration-plan.md)

---

**🎯 This resource ecosystem enables Vrooli's AI tiers to dynamically adapt and evolve their capabilities based on available tools, making the platform both powerful and flexible for diverse use cases.**