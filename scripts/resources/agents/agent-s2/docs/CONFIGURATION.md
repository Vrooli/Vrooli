# Agent S2 Configuration Guide

This guide covers all configuration options for Agent S2, including environment variables, installation parameters, runtime settings, and mode management.

## Environment Variables

### Core Configuration

```bash
# AI Provider Configuration
# Default: Uses local Ollama (no configuration needed)
# Alternatives:
export ANTHROPIC_API_KEY="your_anthropic_api_key_here"  # Override to use Anthropic
export OPENAI_API_KEY="your_openai_api_key_here"       # Override to use OpenAI

# General API key (alternative to provider-specific keys)
export AGENTS2_API_KEY="your_api_key_here"

# AI Feature Control
export AGENTS2_ENABLE_AI=true                          # Enable/disable AI features
```

### Mode Configuration

```bash
# Host Mode Settings
export AGENT_S2_HOST_MODE_ENABLED=true                 # Enable host mode capabilities
export AGENT_S2_HOST_AUDIT_LOGGING=true                # Enable audit logging for host mode
export AGENT_S2_HOST_SECURITY_PROFILE=agent-s2-host    # AppArmor security profile

# Display Configuration
export AGENT_S2_DISPLAY_WIDTH=1920                     # Virtual display width
export AGENT_S2_DISPLAY_HEIGHT=1080                    # Virtual display height
export AGENT_S2_DISPLAY_DEPTH=24                       # Color depth
```

### Service Configuration

```bash
# API Server Settings
export AGENT_S2_API_PORT=4113                          # API server port
export AGENT_S2_API_HOST=0.0.0.0                       # API bind address
export AGENT_S2_API_AUTH=false                         # Enable API authentication
export AGENT_S2_API_KEY="your-secure-api-key"          # API authentication key

# VNC Settings
export AGENT_S2_VNC_PORT=5900                          # VNC server port
export AGENT_S2_VNC_PASSWORD="agents2vnc"              # VNC password
export AGENT_S2_VNC_SSL=false                          # Enable VNC SSL/TLS
export AGENT_S2_VNC_ALLOWED_IPS="127.0.0.1"            # Comma-separated allowed IPs

# Resource Limits
export AGENT_S2_MEMORY_LIMIT=2g                        # Container memory limit
export AGENT_S2_CPU_LIMIT=1.0                          # Container CPU limit
export AGENT_S2_SHM_SIZE=1g                           # Shared memory size
```

### Security Configuration

```bash
# Security Settings
export AGENT_S2_SECURITY_HARDENING=true                # Enable security hardening
export AGENT_S2_NETWORK_ISOLATION=true                 # Enable network isolation
export AGENT_S2_FILESYSTEM_PROTECTION=strict           # Filesystem protection level
export AGENT_S2_PROCESS_MONITORING=true                # Enable process monitoring

# Rate Limiting
export AGENT_S2_API_RATE_LIMIT=100                     # Requests per minute
export AGENT_S2_API_BURST_LIMIT=20                     # Burst allowance

# CORS Settings
export AGENT_S2_CORS_ORIGINS="*"                       # Allowed CORS origins
export AGENT_S2_CORS_METHODS="GET,POST"                # Allowed HTTP methods
```

### Stealth Mode Configuration

```bash
# Stealth Mode Settings
export AGENT_S2_STEALTH_MODE_ENABLED=true              # Enable/disable stealth mode (default: true)
export AGENT_S2_SESSION_STORAGE_PATH=/home/agents2/.agent-s2/sessions    # Session storage path
export AGENT_S2_SESSION_DATA_PERSISTENCE=true          # Enable session data persistence (default: true)
export AGENT_S2_SESSION_STATE_PERSISTENCE=false        # Enable session state persistence (default: false)
export AGENT_S2_SESSION_ENCRYPTION=true                # Enable session encryption (default: true)
export AGENT_S2_SESSION_TTL_DAYS=30                    # Session TTL in days (default: 30)
export AGENT_S2_STEALTH_PROFILE_TYPE=residential       # Profile type: residential, mobile, datacenter

# Stealth Feature Toggles (configured via API or manage.sh)
# - fingerprint_randomization: Randomize browser fingerprints
# - webdriver_hiding: Hide WebDriver automation flags
# - user_agent_rotation: Rotate user agents based on profile type
# - canvas_noise: Add noise to canvas fingerprinting
# - webgl_randomization: Randomize WebGL parameters
# - session_persistence: Maintain cookies and auth across sessions
```

## Installation Configuration

### Basic Installation Options

```bash
# Sandbox mode with AI (recommended - uses Ollama by default)
./manage.sh --action install \
  --mode sandbox

# Alternative: with Anthropic
./manage.sh --action install \
  --mode sandbox \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219

# OpenAI configuration
./manage.sh --action install \
  --mode sandbox \
  --llm-provider openai \
  --llm-model gpt-4o

# Core automation only (no AI)
./manage.sh --action install \
  --mode sandbox \
  --enable-ai no
```

### Advanced Installation Options

```bash
# Full feature installation with Ollama (default)
./manage.sh --action install \
  --mode sandbox \
  --enable-ai yes \
  --enable-search yes \
  --host-mode-enabled yes \
  --vnc-password mysecurepassword \
  --audit-logging yes \
  --security-hardening yes

# Alternative: Full feature installation with Anthropic
./manage.sh --action install \
  --mode sandbox \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219 \
  --enable-ai yes \
  --enable-search yes \
  --host-mode-enabled yes \
  --vnc-password mysecurepassword \
  --audit-logging yes \
  --security-hardening yes

# Stealth mode configuration
./manage.sh --action install \
  --mode sandbox \
  --stealth-enabled yes \
  --stealth-profile residential
```

### Host Mode Configuration

```bash
# Host mode with specific applications
./manage.sh --action install \
  --mode sandbox \
  --host-mode-enabled yes \
  --allowed-host-apps "firefox,code,gimp" \
  --host-mounts "/home/user/Documents,/home/user/Projects"

# Host mode with X11 forwarding
./manage.sh --action install \
  --mode sandbox \
  --host-mode-enabled yes \
  --enable-host-display yes \
  --x11-forwarding yes
```

## Installation Parameters Reference

| Parameter | Description | Default | Options |
|-----------|-------------|---------|---------|
| `--mode` | Operation mode | `sandbox` | `sandbox`, `host` |
| `--llm-provider` | AI provider | `ollama` | `ollama`, `anthropic`, `openai` |
| `--llm-model` | AI model | `llama3.2-vision:11b` | Provider-specific models |
| `--enable-ai` | Enable AI features | `yes` | `yes`, `no` |
| `--enable-search` | Enable web search | `no` | `yes`, `no` |
| `--host-mode-enabled` | Enable host mode | `no` | `yes`, `no` |
| `--vnc-password` | VNC password | `agents2vnc` | Any string |
| `--audit-logging` | Enable audit logging | `no` | `yes`, `no` |
| `--security-hardening` | Enable security hardening | `yes` | `yes`, `no` |
| `--api-auth` | Enable API authentication | `no` | `yes`, `no` |
| `--vnc-ssl` | Enable VNC SSL | `no` | `yes`, `no` |
| `--allowed-host-apps` | Allowed host applications | `none` | Comma-separated list |
| `--host-mounts` | Host mount points | `none` | Comma-separated paths |
| `--memory-limit` | Container memory limit | `2g` | Memory specification |
| `--cpu-limit` | Container CPU limit | `1.0` | CPU specification |
| `--stealth-enabled` | Enable stealth mode | `yes` | `yes`, `no` |
| `--stealth-profile` | Stealth profile type | `residential` | `residential`, `mobile`, `datacenter` |
| `--stealth-feature` | Configure stealth feature | `none` | Feature=value pairs |
| `--stealth-url` | Test URL for stealth | `https://bot.sannysoft.com/` | Any URL |
| `--force` | Force reinstallation | `no` | `yes`, `no` |

## Runtime Mode Management

### Mode Switching

```bash
# Switch between modes at runtime
./lib/modes.sh switch_mode host
./lib/modes.sh switch_mode sandbox

# Check current mode
./lib/modes.sh current_mode

# Validate mode configuration
./lib/modes.sh validate_mode host
```

### API-Based Mode Management

```bash
# Get current mode information
curl http://localhost:4113/modes/current

# Switch modes via API
curl -X POST http://localhost:4113/modes/switch \
  -H "Content-Type: application/json" \
  -d '{"new_mode": "host"}'

# Get mode-specific capabilities
curl http://localhost:4113/modes/environment

# Get security constraints
curl http://localhost:4113/modes/security

# List available applications
curl http://localhost:4113/modes/applications
```

## Docker Configuration

### Container Resource Limits

```bash
# Update resource limits for running container
docker update agent-s2 \
  --memory 8g \
  --cpus 4.0 \
  --shm-size 2g

# View current resource usage
docker stats agent-s2

# Inspect container configuration
docker inspect agent-s2 | jq '.Config'
```

### Custom Docker Options

```bash
# Advanced Docker configuration
export DOCKER_EXTRA_ARGS="--privileged --device /dev/dri:/dev/dri"

# Custom network configuration
export DOCKER_NETWORK="agent-s2-network"

# Volume mounts
export DOCKER_VOLUMES="-v /host/path:/container/path:ro"
```

## Configuration Files

### Main Configuration File

Location: `/home/agents2/.config/agent-s2/config.json`

```json
{
  "ai": {
    "provider": "anthropic",
    "model": "claude-3-7-sonnet-20250219",
    "enabled": true,
    "api_key": "encrypted_key_here"
  },
  "display": {
    "width": 1920,
    "height": 1080,
    "depth": 24,
    "vnc_enabled": true,
    "vnc_port": 5900
  },
  "security": {
    "mode": "sandbox",
    "host_mode_enabled": false,
    "audit_logging": false,
    "security_profile": "agent-s2-sandbox"
  },
  "api": {
    "port": 4113,
    "host": "0.0.0.0",
    "auth_enabled": false,
    "rate_limit": 100
  }
}
```

### Mode-Specific Configuration

#### Sandbox Mode Configuration

```json
{
  "mode": "sandbox",
  "restrictions": {
    "filesystem": ["/home/agents2", "/tmp", "/opt/agent-s2"],
    "network": ["https://"],
    "commands": "whitelist",
    "applications": "container_only"
  },
  "security": {
    "isolation": "full",
    "capabilities": "minimal",
    "user": "agents2"
  }
}
```

#### Host Mode Configuration

```json
{
  "mode": "host",
  "permissions": {
    "filesystem": ["/home/user/Documents", "/home/user/Projects"],
    "network": ["localhost", "192.168.1.0/24"],
    "applications": ["firefox", "code", "gimp"],
    "x11_forwarding": true
  },
  "security": {
    "apparmor_profile": "docker-agent-s2-host",
    "audit_logging": true,
    "monitoring": "enhanced"
  }
}
```

## Performance Configuration

### Memory Optimization

```bash
# Configure memory usage for different workloads
export AGENT_S2_MEMORY_LIMIT=4g    # High memory for AI tasks
export AGENT_S2_MEMORY_LIMIT=1g    # Low memory for basic automation
export AGENT_S2_SHM_SIZE=1g        # Browser shared memory
```

### CPU Optimization

```bash
# CPU allocation based on workload
export AGENT_S2_CPU_LIMIT=2.0      # Multi-core for AI processing
export AGENT_S2_CPU_LIMIT=0.5      # Single core for simple tasks
```

### Display Optimization

```bash
# Display settings for different use cases
export AGENT_S2_DISPLAY_WIDTH=3840   # 4K display
export AGENT_S2_DISPLAY_HEIGHT=2160
export AGENT_S2_DISPLAY_DEPTH=32     # Higher color depth

export AGENT_S2_DISPLAY_WIDTH=1280   # Low resource display
export AGENT_S2_DISPLAY_HEIGHT=720
export AGENT_S2_DISPLAY_DEPTH=16
```

## Validation and Testing

### Configuration Validation

```bash
# Validate current configuration
./manage.sh --action validate-config

# Test configuration changes
./manage.sh --action test-config --dry-run

# Verify security settings
./security/validate-profile.sh
```

### Configuration Testing

```bash
# Test API endpoints
curl http://localhost:4113/health
curl http://localhost:4113/capabilities

# Test AI functionality (if enabled)
curl -X POST http://localhost:4113/ai/action \
  -H "Content-Type: application/json" \
  -d '{"task": "take a screenshot"}'

# Test VNC connection
vncviewer localhost:5900
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup configuration
./manage.sh --action backup-config --output config-backup.tar.gz

# Export environment variables
./manage.sh --action export-env > agent-s2-env.sh
```

### Configuration Restore

```bash
# Restore from backup
./manage.sh --action restore-config --input config-backup.tar.gz

# Import environment variables
source agent-s2-env.sh
```

## Troubleshooting Configuration

### Common Configuration Issues

```bash
# Check configuration syntax
./manage.sh --action check-config

# Validate environment variables
./manage.sh --action check-env

# Debug configuration loading
./manage.sh --action debug-config --verbose
```

### Configuration Logs

```bash
# View configuration loading logs
./manage.sh --action logs --filter config

# Monitor configuration changes
tail -f /var/log/agent-s2/config.log
```