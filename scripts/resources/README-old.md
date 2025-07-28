# Vrooli Resource Management System

This directory provides a comprehensive ecosystem of specialized tools that extend Vrooli's AI capabilities through local services. Rather than building every possible feature into the core platform, Vrooli leverages external specialized tools that its three-tier AI system can dynamically discover, orchestrate, and integrate.

## 🎯 Philosophy

The resource system enables **capability emergence** through tool orchestration:
- **AI Services** provide inference capabilities (local models, cloud APIs, specialized AI tools)
- **Automation Platforms** handle complex multi-step workflows and integrations
- **Agent Services** enable interaction with external systems and interfaces
- **Storage Services** manage artifacts, files, and persistent data

This architecture allows Vrooli's AI tiers to adapt to whatever resources are available, making the platform highly flexible without requiring all users to run the same infrastructure stack.

## 📁 Directory Structure

```
scripts/resources/
├── README.md              # This documentation
├── index.sh               # Resource orchestrator and discovery
├── common.sh              # Shared utilities and patterns
├── port-registry.sh       # Port allocation management
├── ai/                    # AI and machine learning resources
│   ├── ollama/            # Local LLM inference server
│   ├── whisper/           # Speech-to-text transcription
│   └── comfyui/           # AI image generation workflows
├── automation/            # Workflow automation platforms
│   ├── n8n/               # Visual workflow automation with host access
│   ├── node-red/          # IoT and flow-based programming
│   ├── windmill/          # Code-first workflow automation
│   └── huginn/            # Agent-based event processing
├── agents/                # Interaction and automation agents
│   ├── agent-s2/          # Autonomous screen interaction with AI
│   ├── browserless/       # Chrome-as-a-service for web automation
│   └── claude-code/       # Anthropic's CLI for AI development
└── storage/               # Data storage and artifact management
    └── minio/             # S3-compatible object storage
```

## 🚀 Quick Start

### Resource Discovery

Before using resources, check what's available:

```bash
# See what resources are currently running
./scripts/resources/index.sh --action discover

# Check which resources are enabled in your configuration
cat ~/.vrooli/resources.local.json | jq '.services'

# List all available resources for installation
./scripts/resources/index.sh --action list
```

### Automatic Setup (Recommended)

Install resources through the main setup script:

```bash
# Install only enabled resources (reads ~/.vrooli/resources.local.json)
./scripts/main/setup.sh --target native-linux --resources enabled

# Install specific resource categories
./scripts/main/setup.sh --target native-linux --resources ai-only
./scripts/main/setup.sh --target native-linux --resources automation-only

# Install specific resources
./scripts/main/setup.sh --target native-linux --resources "ollama,n8n,agent-s2"

# Skip all resources (useful for CI/CD)
./scripts/main/setup.sh --target native-linux --resources none
```

### Manual Resource Management

Fine-grained control for development:

```bash
# Resource lifecycle
./scripts/resources/index.sh --action install --resources "ollama,n8n"
./scripts/resources/index.sh --action start --resources n8n
./scripts/resources/index.sh --action stop --resources n8n
./scripts/resources/index.sh --action status --resources ollama

# Health monitoring
./scripts/resources/ollama/manage.sh --action status
./scripts/resources/n8n/manage.sh --action logs
```

## 📋 Complete Resource Catalog

### 🧠 AI Resources (`ai`)

| Resource | Status | Description | Default Port | Use Cases |
|----------|--------|-------------|----|-----------| 
| **ollama** | ✅ Production Ready | Local LLM inference server | 11434 | Private AI chat, code generation, offline inference |
| **whisper** | ✅ Production Ready | OpenAI Whisper speech-to-text | 8090 | Audio transcription, voice interfaces, meeting notes |
| **comfyui** | ✅ Production Ready | AI image generation workflows | 8188 | Image generation, editing, AI art pipelines |
| **cloudflare** | ✅ Cloud Service | Cloudflare AI Gateway | N/A (Cloud) | Scalable cloud AI, cost optimization, rate limiting |
| **openrouter** | ✅ Cloud Service | Multi-provider AI API access | N/A (Cloud) | Access to GPT-4, Claude, diverse AI models |

**💡 When to use:** Local AI resources for privacy/offline needs, cloud services for scale and cutting-edge models.

### ⚙️ Automation Resources (`automation`)

| Resource | Status | Description | Default Port | Use Cases |
|----------|--------|-------------|----|-----------| 
| **n8n** | ✅ Production Ready | Visual workflow automation + host access | 5678 | Complex workflows, API integrations, system automation |
| **node-red** | ✅ Production Ready | IoT and flow-based programming | 1880 | Device integration, sensor data, real-time processing |
| **windmill** | ✅ Production Ready | Code-first workflow automation | 5681 | Developer workflows, CI/CD integration, script orchestration |
| **huginn** | ✅ Production Ready | Agent-based event processing | 4111 | Web monitoring, data aggregation, intelligent alerts |
| **activepieces** | ✅ Available | No-code automation platform | 8080 | Business process automation, SaaS integrations |
| **airflow** | ✅ Available | Data pipeline orchestration | 8080 | ETL workflows, data science pipelines, scheduled jobs |
| **temporal** | ✅ Available | Durable workflow execution | 7233 | Long-running processes, microservice orchestration |

**💡 When to use:** n8n for visual workflows with system access, windmill for code-first approach, node-red for IoT, huginn for web monitoring.

### 🤖 Agent Resources (`agents`)

| Resource | Status | Description | Default Port | Use Cases |
|----------|--------|-------------|----|-----------| 
| **agent-s2** | ✅ Production Ready | Autonomous screen interaction with AI | 4113 (API), 5900 (VNC) | Real web navigation, desktop app control, adaptive automation |
| **browserless** | ✅ Production Ready | Chrome-as-a-service automation | 4110 | Local dashboards, internal tools, predictable web automation |
| **claude-code** | ✅ Production Ready | Anthropic's CLI for development | N/A (CLI) | AI pair programming, code analysis, development tasks |

**💡 When to use:** 
- **browserless**: Local/internal web services, dashboards, development servers, ComfyUI, or any trusted site with predictable structure
- **agent-s2**: Public websites with anti-bot measures, desktop applications, tasks requiring visual reasoning or adaptation
- **claude-code**: Development tasks and code assistance

### 💾 Storage Resources (`storage`)

| Resource | Status | Description | Default Port | Use Cases |
|----------|--------|-------------|----|-----------| 
| **minio** | ✅ Production Ready | S3-compatible object storage | 9000 (API), 9001 (Console) | File uploads, AI artifacts, model caching, backups |
| **ipfs** | ✅ Available | Distributed file storage | 5001 (API), 8080 (Gateway) | Decentralized storage, content addressing, P2P sharing |
| **rclone** | ✅ Available | Cloud storage synchronization | 5572 | Multi-cloud sync, backup automation, file migration |

**💡 When to use:** MinIO for local S3-compatible storage, IPFS for distributed storage, rclone for cloud synchronization.

## 🎯 Use Cases and Integration Patterns

### 🤖 Choosing Between Browserless and Agent-S2

**Critical Decision Guide for Web/GUI Automation:**

#### Use Browserless When:
- 🏠 **Accessing local/internal services**: localhost, internal IPs, VPN-only resources
- 📊 **Capturing dashboards**: Grafana, Kibana, monitoring tools, analytics
- 🛠️ **Development/staging sites**: Your own applications under development
- 🎨 **AI tool interfaces**: ComfyUI, Stable Diffusion WebUI, Jupyter notebooks
- 📄 **Generating reports**: Converting known web content to PDF/images
- ✅ **Predictable automation**: When you know exact selectors, URLs, and page structure
- 🚀 **High-volume tasks**: Need speed and efficiency over adaptability

**Browserless characteristics:**
- Fast, lightweight REST API
- No anti-detection needed
- Direct DOM manipulation
- Assumes cooperative target
- Best for "friendly" automation

#### Use Agent-S2 When:
- 🌐 **Navigating public websites**: Google, Amazon, social media, news sites
- 🛡️ **Facing anti-bot measures**: CAPTCHAs, rate limiting, bot detection
- 🎯 **Unknown interfaces**: "Find and click the submit button" vs knowing exact selector
- 🧩 **Multi-step workflows**: Tasks requiring decision-making and adaptation
- 💻 **Desktop applications**: Beyond web browsers - Excel, games, native apps
- 👁️ **Visual tasks**: "Click the biggest blue button" or "find the price"
- 🔄 **Dynamic content**: Sites that change layout or have unpredictable elements

**Agent-S2 characteristics:**
- AI-powered understanding
- Human-like interaction patterns
- Visual reasoning capabilities
- Adapts to unexpected changes
- Best for "hostile" or unknown environments

#### Quick Decision Tree:
```
Is it a web page?
├─ Yes
│  ├─ Is it local/internal/trusted?
│  │  ├─ Yes → Use Browserless
│  │  └─ No
│  │     ├─ Has anti-bot protection?
│  │     │  ├─ Yes → Use Agent-S2
│  │     │  └─ No
│  │     │     ├─ Need visual reasoning?
│  │     │     │  ├─ Yes → Use Agent-S2
│  │     │     │  └─ No → Either works, prefer Browserless for speed
│  │     └─ Dynamic/unpredictable?
│  │        ├─ Yes → Use Agent-S2
│  │        └─ No → Use Browserless
└─ No (Desktop app)
   └─ Use Agent-S2
```

### Common Integration Workflows

**AI-Powered Web Automation (Public Sites):**
```bash
# For unpredictable public websites:
# 1. agent-s2 navigates complex sites with anti-bot measures
# 2. agent-s2 handles dynamic content and CAPTCHAs
# 3. whisper transcribes any audio content
# 4. minio stores artifacts and results
```

**Internal Dashboard Automation:**
```bash
# For trusted internal services:
# 1. browserless captures dashboard screenshots efficiently
# 2. browserless generates PDF reports from internal wikis
# 3. ollama analyzes the captured content
# 4. n8n orchestrates the workflow on schedule
# 5. minio stores all reports and artifacts
```

**Content Processing Pipeline:**
```bash
# 1. browserless scrapes web content
# 2. whisper transcribes video/audio
# 3. ollama processes and analyzes content
# 4. comfyui generates related images
# 5. n8n orchestrates the entire workflow
# 6. minio stores all artifacts
```

**Development Workflow Automation:**
```bash
# 1. claude-code assists with code development
# 2. windmill runs CI/CD workflows
# 3. huginn monitors deployment status
# 4. minio stores build artifacts
```

### Resource Categories

Install by logical groupings:

- **`ai-only`** - All AI and ML resources (ollama, whisper, comfyui)
- **`automation-only`** - All workflow platforms (n8n, node-red, windmill, huginn)
- **`agents-only`** - All interaction agents (agent-s2, browserless, claude-code)
- **`storage-only`** - All storage solutions (minio, ipfs, rclone)
- **`essential`** - Core set (ollama, n8n, agent-s2, minio)
- **`all`** - All available resources
- **`enabled`** - Only resources marked enabled in configuration
- **`none`** - Skip resource installation

## 🔍 Resource Discovery and Availability

### Programmatic Resource Discovery

For coding agents and automated systems:

**Important for AI Agents**: When deciding between browserless and agent-s2:
```bash
# Quick decision logic for automation tasks
if [[ "$TARGET_URL" =~ ^(http://localhost|http://127\.0\.0\.1|http://172\.|http://192\.168\.|http://10\.) ]]; then
    echo "Use browserless - internal/local resource"
elif [[ "$TASK_TYPE" == "desktop_app" ]]; then
    echo "Use agent-s2 - desktop automation"
elif [[ "$NEEDS_VISUAL_REASONING" == "true" ]] || [[ "$HAS_ANTI_BOT" == "true" ]]; then
    echo "Use agent-s2 - complex web automation"
else
    echo "Use browserless - simple web automation"
fi
```

```bash
# Check what's currently available
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "Ollama is available for local AI inference"
fi

# Parse configuration to see enabled resources
ENABLED_RESOURCES=$(jq -r '.services | to_entries[] | select(.value.enabled == true) | .key' ~/.vrooli/resources.local.json 2>/dev/null || echo "none")

# Health check multiple resources
for service in ollama n8n agent-s2 minio; do
    if ./scripts/resources/index.sh --action status --resources $service | grep -q "healthy"; then
        echo "✅ $service is available"
    else
        echo "❌ $service is not available"
    fi
done
```

### Configuration-Based Availability

Resources are controlled by the `enabled` flag in `~/.vrooli/resources.local.json`:

```json
{
  "services": {
    "ai": {
      "ollama": { "enabled": true, "baseUrl": "http://localhost:11434" },
      "whisper": { "enabled": false }
    },
    "automation": {
      "n8n": { "enabled": true, "baseUrl": "http://localhost:5678" }
    }
  }
}
```

**Key Points for Coding Agents:**
- Only `enabled: true` resources will be installed and available
- Check both configuration AND actual service health before using
- Use the `discover` action to get real-time availability
- Each resource exposes health check endpoints for monitoring

## 📖 Quick Resource References

### Essential Commands by Resource

**AI Resources:**
```bash
# Ollama - Local LLM inference
./scripts/resources/ai/ollama/manage.sh --action status
# See: scripts/resources/ai/ollama/README.md

# Whisper - Speech-to-text
./scripts/resources/ai/whisper/manage.sh --action transcribe --file audio.wav
# See: scripts/resources/ai/whisper/README.md

# ComfyUI - AI image generation
./scripts/resources/automation/comfyui/manage.sh --action status
# See: scripts/resources/automation/comfyui/README.md
```

**Automation Resources:**
```bash
# n8n - Visual workflows with host access
./scripts/resources/automation/n8n/manage.sh --action execute --workflow-id ID
# See: scripts/resources/automation/n8n/README.md

# Node-RED - IoT and flow programming
./scripts/resources/automation/node-red/manage.sh --action deploy-flow --file flow.json
# See: scripts/resources/automation/node-red/README.md

# Windmill - Code-first workflows
./scripts/resources/automation/windmill/manage.sh --action scale-workers --workers 5
# See: scripts/resources/automation/windmill/README.md
```

**Agent Resources:**
```bash
# agent-s2 - For public web & desktop apps (adaptive, AI-powered)
# Example: Navigate Amazon and find products
curl -X POST http://localhost:4113/ai/task -d '{"task": "go to amazon.com and search for coffee makers"}'
# Example: Interact with desktop application
curl -X POST http://localhost:4113/ai/task -d '{"task": "open calculator and compute 47 * 83"}'
# See: scripts/resources/agents/agent-s2/README.md

# Browserless - For local/internal web services (fast, predictable)
# Example: Capture your Grafana dashboard
curl -X POST http://localhost:4110/chrome/screenshot -d '{"url": "http://localhost:3000/dashboard"}'
# Example: Screenshot ComfyUI interface
curl -X POST http://localhost:4110/chrome/screenshot -d '{"url": "http://localhost:8188"}'
# See: scripts/resources/agents/browserless/README.md
```

**Storage Resources:**
```bash
# MinIO - S3-compatible storage
./scripts/resources/storage/minio/manage.sh --action create-bucket --bucket my-bucket
# See: scripts/resources/storage/minio/README.md
```

## ⚙️ Configuration and Resource Management

### Configuration Files

**Template Configuration:**
- Source: `.vrooli/resources.example.json` (version-controlled template)
- Local Config: `~/.vrooli/resources.local.json` (user-specific settings)
- Production: Environment variables override local settings

**Configuration Workflow:**
```bash
# 1. Copy template to create local configuration
cp .vrooli/resources.example.json ~/.vrooli/resources.local.json

# 2. Edit to enable desired resources
vim ~/.vrooli/resources.local.json  # Set "enabled": true for resources you want

# 3. Set API keys in environment
export OPENAI_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"

# 4. Install enabled resources
./scripts/main/setup.sh --target native-linux --resources enabled
```

### Resource Configuration Schema

Each resource follows a consistent configuration pattern:

```json
{
  "services": {
    "category": {
      "resource-name": {
        "enabled": true,
        "baseUrl": "http://localhost:PORT",
        "apiKey": "${ENV_VAR_NAME}",
        "capabilities": ["feature1", "feature2"],
        "healthCheck": {
          "endpoint": "/health",
          "intervalMs": 300000,
          "timeoutMs": 5000
        }
      }
    }
  }
}
```

### Environment Variable Patterns

**AI Services:**
```bash
# Required for cloud AI services
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export CLOUDFLARE_API_KEY="..."
export OPENROUTER_API_KEY="sk-or-..."

# Optional for enhanced agent capabilities
export AGENTS2_API_KEY="..."  # For agent-s2 AI features
```

**Service Customization:**
```bash
# Override default ports if needed
export OLLAMA_PORT="11435"
export N8N_PORT="5679"
export MINIO_PORT="9001"

# Custom resource paths
export VROOLI_RESOURCES_CONFIG="/custom/path/resources.json"
```

## 🏗️ System Integration with Vrooli

### Automatic Resource Discovery

Vrooli's ResourceRegistry automatically discovers and integrates available resources:

```typescript
// The three-tier AI system dynamically adapts to available resources
const availableResources = await resourceRegistry.getHealthyResources();

// Tier 1 (Strategic) can route work based on available capabilities
if (availableResources.includes('ollama')) {
    // Use local inference for privacy-sensitive tasks
    strategy = 'local-first';
} else if (availableResources.includes('openrouter')) {
    // Fall back to cloud services
    strategy = 'cloud-fallback';
}

// Tier 2 (Process) orchestrates multi-resource workflows
if (availableResources.includes('n8n') && availableResources.includes('minio')) {
    // Complex workflows with artifact storage
    await processOrchestrator.executeWorkflow('data-pipeline');
}

// Tier 3 (Execution) uses specific tools for tasks
if (availableResources.includes('agent-s2')) {
    await executionEngine.performScreenAutomation(task);
}
```

### Health Monitoring and Resilience

**Continuous Health Checking:**
- Resources report health every 5 minutes via configured endpoints
- Failed health checks trigger automatic retry and fallback logic
- Resource status changes are propagated to all AI tiers immediately

**Graceful Degradation:**
```bash
# If local Ollama fails, automatically fall back to cloud services
# If n8n is unavailable, use simpler automation or manual execution
# If MinIO is down, use temporary local storage with warnings
```

### Service Management

**Docker-based Services (Recommended):**
```bash
# Most resources run as Docker containers for isolation
docker ps --filter "label=vrooli-resource"

# Check logs for any resource
./scripts/resources/index.sh --action logs --resources n8n

# Restart unhealthy resources
./scripts/resources/index.sh --action restart --resources ollama
```

**SystemD Services (Alternative):**
```bash
# Some resources can optionally run as system services
sudo systemctl status ollama
journalctl -u ollama -f --since "1 hour ago"
```

**Integration with Vrooli Server:**
- Resources are automatically registered with the ResourceRegistry on startup
- Health status is continuously monitored and cached
- Failed resources are automatically retried with exponential backoff
- Resource capabilities are exposed through the GraphQL API

## 🔍 Troubleshooting and Diagnostics

### Resource Health Diagnostics

**Quick Health Check:**
```bash
# Check all resource health at once
./scripts/resources/index.sh --action discover

# Detailed status for specific resources
./scripts/resources/index.sh --action status --resources "ollama,n8n,minio"

# Individual resource diagnostics
./scripts/resources/automation/n8n/manage.sh --action status
./scripts/resources/ai/ollama/manage.sh --action models
./scripts/resources/storage/minio/manage.sh --action diagnose
```

### Common Issues and Solutions

**Resource Not Available:**
```bash
# 1. Check if enabled in configuration
jq '.services.ai.ollama.enabled' ~/.vrooli/resources.local.json

# 2. Check if service is running
docker ps | grep ollama
# OR
systemctl status ollama

# 3. Check port availability
ss -tlnp | grep 11434

# 4. Check logs for errors
./scripts/resources/ai/ollama/manage.sh --action logs
```

**Port Conflicts:**
```bash
# Find what's using a port
sudo lsof -i :5678

# Use alternative ports
N8N_CUSTOM_PORT=5679 ./scripts/resources/automation/n8n/manage.sh --action install

# Check port registry
./scripts/resources/port-registry.sh --action list
```

**Permission and Access Issues:**
```bash
# Ensure Docker access (most common issue)
sudo usermod -aG docker $USER
newgrp docker  # Apply immediately

# Check Docker daemon
sudo systemctl status docker

# Verify resource scripts are executable
find scripts/resources -name "manage.sh" -exec chmod +x {} \;
```

**Configuration Issues:**
```bash
# Validate configuration file
jq . ~/.vrooli/resources.local.json

# Reset to defaults
cp .vrooli/resources.example.json ~/.vrooli/resources.local.json

# Check environment variables
echo $OPENAI_API_KEY | cut -c1-10  # Show first 10 chars only
```

### Debug Mode and Logging

**Enable Verbose Debugging:**
```bash
# Global debug mode
export DEBUG=1
export LOG_LEVEL=debug

# Resource-specific debugging
DEBUG_RESOURCES=1 ./scripts/resources/index.sh --action install --resources ollama

# Docker container debugging
docker logs -f ollama  # Follow logs in real-time
docker exec -it n8n /bin/bash  # Interactive shell in container
```

**Log Locations:**
```bash
# Docker container logs
docker logs <container-name>

# Resource-specific logs
~/.vrooli/logs/resources/
~/.ollama/logs/
~/.n8n/logs/

# System service logs
journalctl -u ollama -f
journalctl -u docker -f
```

### Recovery and Cleanup

**Automatic Recovery:**
```bash
# Restart failed resources
./scripts/resources/index.sh --action restart --resources ollama

# Reinstall corrupted resources
./scripts/resources/index.sh --action uninstall --resources n8n
./scripts/resources/index.sh --action install --resources n8n
```

**Manual Cleanup (Last Resort):**
```bash
# Stop all resource containers
docker stop $(docker ps -q --filter "label=vrooli-resource")

# Remove resource containers and volumes
docker system prune -f --volumes

# Reset configuration
rm ~/.vrooli/resources.local.json
cp .vrooli/resources.example.json ~/.vrooli/resources.local.json

# Clean install
./scripts/main/setup.sh --target native-linux --resources enabled --force
```

## 🧩 Extending the Resource System

### Adding New Resources

Follow the established patterns when adding new resources:

**1. Directory Structure:**
```bash
# Create directory in appropriate category
mkdir -p scripts/resources/category/new-resource
cd scripts/resources/category/new-resource

# Required files
touch README.md manage.sh
mkdir -p config lib examples
```

**2. Follow the Standard Pattern:**
```bash
# Each resource needs:
# - README.md: Comprehensive documentation
# - manage.sh: Main management script
# - config/defaults.sh: Default configuration
# - config/messages.sh: User messages
# - lib/: Modular functionality (docker.sh, status.sh, etc.)
# - examples/: Usage examples and templates
```

**3. Implement Required Functions:**
```bash
# Standard management functions in manage.sh:
# - install: Setup and configure resource
# - uninstall: Complete cleanup
# - start/stop/restart: Service lifecycle
# - status: Health checking and diagnostics
# - logs: Access to service logs
```

**4. Integration Steps:**
```bash
# Add to port registry
echo "new-resource 8080" >> scripts/resources/port-registry.sh

# Update main orchestrator
vim scripts/resources/index.sh  # Add to AVAILABLE_RESOURCES

# Add to configuration template
vim .vrooli/resources.example.json  # Add service definition

# Test thoroughly
./scripts/resources/index.sh --action install --resources new-resource
./scripts/resources/index.sh --action status --resources new-resource
```

### Resource Development Guidelines

**Configuration Integration:**
- Resources must auto-register in `~/.vrooli/resources.local.json`
- Follow the standard schema with `enabled`, `baseUrl`, `healthCheck`
- Support environment variable overrides for sensitive data

**Health Monitoring:**
- Implement reliable health check endpoints
- Use consistent status reporting (healthy/unhealthy/unknown)
- Support both quick checks and detailed diagnostics

**Docker Best Practices:**
- Use official base images when possible
- Implement proper signal handling for graceful shutdown
- Use volume mounts for persistent data
- Set appropriate resource limits

**Documentation Standards:**
- Comprehensive README with all features documented
- Include practical examples and use cases
- Cross-reference related resources and integration patterns
- Provide troubleshooting section with common issues

**Testing Requirements:**
- Test all management functions (install/start/stop/status)
- Verify health checks work correctly
- Test resource discovery and integration
- Include example workflows and API usage

## 📊 Resource Status and Monitoring

**Real-time Status Dashboard:**
```bash
# Complete resource overview
./scripts/resources/index.sh --action discover

# Category-specific status
./scripts/resources/index.sh --action status --resources ai-only
./scripts/resources/index.sh --action status --resources automation-only

# Continuous monitoring (useful for development)
watch -n 5 './scripts/resources/index.sh --action discover'
```

**Health Check API Integration:**
```bash
# Check if resource is ready for use
curl -s http://localhost:11434/api/tags > /dev/null && echo "Ollama ready"
curl -s http://localhost:5678/api/v1/workflows > /dev/null && echo "n8n ready"
curl -s http://localhost:4113/health > /dev/null && echo "agent-s2 ready"
```

## 🔗 Related Documentation

### Core Documentation
- **[CLAUDE.md](/CLAUDE.md)** - Project instructions and quick commands
- **[Vrooli Resource Provider System](/packages/server/src/services/resources/README.md)** - Server-side resource integration
- **[AI Resource Integration Plan](/docs/architecture/ai-resource-integration-plan.md)** - Architecture and patterns

### Setup and Development
- **[Main Setup Script](/scripts/main/setup.sh)** - Automated environment setup
- **[Development Environment](/scripts/main/develop.sh)** - Development workflow
- **[Project Documentation](/docs/README.md)** - Complete project overview

### Individual Resource Documentation
**AI Resources:**
- [Ollama Local LLM Server](/scripts/resources/ai/ollama/README.md)
- [Whisper Speech-to-Text](/scripts/resources/ai/whisper/README.md)
- [ComfyUI Image Generation](/scripts/resources/automation/comfyui/README.md)

**Automation Resources:**
- [n8n Workflow Automation](/scripts/resources/automation/n8n/README.md)
- [Node-RED Flow Programming](/scripts/resources/automation/node-red/README.md)
- [Windmill Code-First Workflows](/scripts/resources/automation/windmill/README.md)
- [Huginn Agent-Based Processing](/scripts/resources/agents/huginn/README.md)

**Agent Resources:**
- [agent-s2 Autonomous Interaction](/scripts/resources/agents/agent-s2/README.md)
- [Browserless Web Automation](/scripts/resources/agents/browserless/README.md)
- [Claude Code Development Assistant](/scripts/resources/agents/claude-code/README.md)

**Storage Resources:**
- [MinIO Object Storage](/scripts/resources/storage/minio/README.md)

## 💡 Best Practices for Coding Agents

### Resource Selection Strategy
1. **Check availability first** - Use `discover` action before attempting to use resources
2. **Graceful fallback** - Design workflows that degrade gracefully when resources unavailable
3. **Match tool to task** - Choose the right resource for each specific use case:
   - **Web automation**: Browserless for internal/trusted sites, Agent-S2 for public/complex sites
   - **Desktop automation**: Always use Agent-S2
   - **API workflows**: Use n8n or Node-RED
   - **Monitoring**: Use Huginn for web monitoring, Agent-S2 for visual monitoring
4. **Resource chaining** - Combine multiple resources for complex workflows

### Web Automation Tool Selection (Critical for AI Agents)

**When given a web automation task, ALWAYS consider:**

1. **URL Analysis** - Check if the target is:
   - `localhost`, `127.0.0.1`, or private IPs → **Use Browserless**
   - Public domain (google.com, amazon.com) → **Use Agent-S2**
   - Development server (`:3000`, `:8080`) → **Use Browserless**
   - Internal tool (ComfyUI, Grafana, Jupyter) → **Use Browserless**

2. **Task Complexity** - Evaluate if you need:
   - Known selectors/structure → **Use Browserless**
   - Visual understanding ("click the biggest button") → **Use Agent-S2**
   - CAPTCHA handling → **Use Agent-S2**
   - Dynamic adaptation → **Use Agent-S2**

3. **Performance Requirements**:
   - High-speed, bulk operations → **Use Browserless**
   - Human-like interaction timing → **Use Agent-S2**

**Example Decision Process:**
```
User: "Take a screenshot of the Google search results for 'rabbits'"
Analysis: Public site (google.com) + likely has anti-bot → Use Agent-S2

User: "Capture the status from our monitoring dashboard"
Analysis: Internal service + predictable layout → Use Browserless

User: "Extract data from this e-commerce site"
Analysis: Public site + may need adaptation → Use Agent-S2

User: "Generate a PDF from our wiki at http://192.168.1.50:8080"
Analysis: Internal IP + known structure → Use Browserless
```

### Development Workflow
1. **Start with essential resources** - Install core set (ollama, n8n, minio) first
2. **Add resources incrementally** - Install additional resources as needed for specific tasks
3. **Monitor resource health** - Regularly check resource status during development
4. **Use configuration management** - Keep resource configuration in version control

### Integration Patterns
1. **Health checking** - Always verify resource health before use
2. **Error handling** - Implement retries and fallbacks for resource failures
3. **Resource discovery** - Use programmatic discovery for dynamic workflows
4. **Configuration-driven** - Make resource usage configurable and optional

### Performance Optimization
1. **Resource pooling** - Reuse connections and sessions when possible
2. **Caching** - Cache resource responses to reduce load
3. **Batch operations** - Group related operations for efficiency
4. **Resource monitoring** - Track resource usage and performance metrics

---

**🎯 This resource ecosystem enables Vrooli's AI tiers to dynamically adapt and evolve their capabilities based on available tools, making the platform both powerful and flexible for diverse use cases.**