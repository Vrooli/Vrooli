# Product Requirements Document (PRD) - Claude Code

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
Claude Code provides an enterprise-grade AI coding agent with tool-using capabilities for autonomous software development. It integrates Anthropic's Claude models for advanced code generation, debugging, and refactoring while offering seamless fallback to local Ollama models for cost optimization and rate limit mitigation. This creates a resilient, always-available coding intelligence infrastructure.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Autonomous Development**: Enables scenarios to generate, modify, and test code without human intervention using Claude's tool capabilities
- **Intelligent Fallback**: Automatic switching to Ollama-Code models when rate limits are hit or for cost optimization
- **Tool Execution**: Unlike pure LLMs, Claude Code can actually create files, run commands, and modify systems
- **Multi-Provider Resilience**: Seamlessly switches between Claude (primary) and Ollama (fallback) without disrupting workflows
- **Session Persistence**: Maintains context across restarts and provider switches for continuous development
- **Batch Processing**: Enables bulk code operations across entire codebases
- **MCP Integration**: Leverages Model Context Protocol for enhanced tool usage and system integration

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Autonomous Application Development**: Generate complete applications from specifications with actual file creation
2. **Intelligent Code Refactoring**: Automated code quality improvements with direct file modifications
3. **Security Auditing & Fixing**: Vulnerability scanning and automatic patching across codebases
4. **Test Suite Generation**: Create comprehensive tests with actual file outputs
5. **Documentation Automation**: Generate and maintain documentation that stays in sync with code
6. **Code Migration**: Automated language/framework migrations with fallback safety
7. **Continuous Code Improvement**: 24/7 code optimization using Ollama during off-peak hours

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Core CLI interface (resource-claude-code)
  - [x] Installation and setup automation
  - [x] Health monitoring and status reporting
  - [x] Integration with Vrooli resource framework
  - [ ] Ollama-Code fallback mechanism for rate limit handling
  - [x] Tool execution capabilities (file creation, editing, command execution)
  - [x] Session management for persistent contexts
  - [x] Rate limit detection and automatic provider switching
  - [ ] Content management implementation (add/list/get/remove/execute)
  
- **Should Have (P1)**
  - [x] Batch processing framework
  - [x] MCP (Model Context Protocol) support
  - [x] Rate limiting and retry logic with intelligent backoff
  - [x] Error handling and recovery with provider fallback
  - [x] Usage tracking and cost estimation
  - [x] Model selection based on task complexity
  - [ ] JSON output format for all commands
  - [ ] Integration tests in test/ folder
  - [ ] Test results in status output
  
- **Nice to Have (P2)**
  - [x] Sandbox environment for safe code testing
  - [x] Template system for common development tasks
  - [ ] Web interface for monitoring sessions
  - [ ] Metrics and telemetry integration with OpenTelemetry
  - [ ] Fine-tuned Ollama models for specific languages
  - [ ] Code review workflow integration

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time (Claude) | < 5s for simple queries | Time from request to first token |
| Response Time (Ollama Fallback) | < 10s for simple queries | Time from request to first token |
| Provider Switch Time | < 2s | Time to detect rate limit and switch |
| Tool Execution | < 1s per operation | File operations and command execution |
| Session Recovery | < 10s | Time to restore session state |
| Batch Processing | 100 files/hour | Throughput measurement |
| Error Rate | < 1% | Failed requests / total requests |
| Fallback Success Rate | > 95% | Successful Ollama responses when Claude limited |

### Non-Functional Requirements
- **Security**
  - [x] API key management via Vault
  - [x] Secure session storage with encryption
  - [x] Sandboxed code execution environment
  - [ ] Audit logging for all code modifications
  - [ ] Code signing for generated outputs
  
- **Reliability**
  - [x] Automatic retry on failures with exponential backoff
  - [x] Session persistence across restarts
  - [x] Graceful degradation with Ollama fallback
  - [x] Rate limit prediction and preemptive switching
  - [ ] Health monitoring and alerting integration
  
- **Scalability**
  - [x] Support for multiple concurrent sessions
  - [x] Batch processing for bulk operations
  - [x] Distributed execution across multiple Ollama instances
  - [ ] Resource usage limits and quotas
  - [ ] Horizontal scaling support for high throughput

## 🔧 Technical Specifications

### Architecture
- **Runtime**: Node.js v18+ (Claude CLI), Bash (resource management)
- **Primary Provider**: Anthropic Claude (claude-3-5-sonnet-latest)
- **Fallback Provider**: Ollama-Code with optimized models
- **Dependencies**: 
  - Claude CLI (@anthropic-ai/claude-code)
  - Ollama resource (for fallback)
  - LiteLLM adapter (optional proxy)
  - Vault (for secure key storage)

### Provider Configuration
```yaml
primary_provider:
  name: Claude Code (Anthropic)
  models:
    - claude-3-5-sonnet-latest (default)
    - claude-3-opus-latest (complex tasks)
  capabilities:
    - Tool execution (file operations, commands)
    - Long context (200K tokens)
    - Code-specific optimizations
  limits:
    - 5-hour rolling window: ~45 requests (Pro)
    - Daily limits based on subscription
    - Weekly limits vary by tier

fallback_provider:
  name: Ollama-Code
  models:
    - qwen2.5-coder:14b (primary fallback)
    - qwen2.5-coder:7b (fast responses)
    - codellama:7b (specialized coding)
    - deepseek-r1:8b (reasoning tasks)
  capabilities:
    - Unlimited local inference
    - No rate limits
    - Privacy-preserving
    - Cost-free operation
  limitations:
    - No tool execution (text-only responses)
    - Smaller context windows
    - Variable performance based on hardware
```

### Interfaces
- **CLI Commands**:
  ```bash
  # Core management
  resource-claude-code status [--json] [--verbose]    # Health and configuration
  resource-claude-code start                          # Start the service
  resource-claude-code stop                           # Stop the service
  
  # Execution
  resource-claude-code run "<prompt>"                 # Execute with automatic fallback
  resource-claude-code batch --file tasks.json        # Batch processing
  
  # Session management
  resource-claude-code session list                   # List active sessions
  resource-claude-code session resume <id>            # Resume a session
  
  # Fallback control
  resource-claude-code for ollama connect            # Force Ollama fallback
  resource-claude-code for ollama disconnect         # Return to Claude
  resource-claude-code usage                         # Check rate limit status
  
  # Content management
  resource-claude-code content add <template>        # Add templates
  resource-claude-code content list                  # List templates
  resource-claude-code content execute <name>        # Execute template
  ```

### Configuration
```yaml
environment_variables:
  # Primary provider
  ANTHROPIC_API_KEY: Retrieved from Vault
  CLAUDE_SUBSCRIPTION_TIER: free|pro|teams|enterprise
  
  # Fallback configuration
  OLLAMA_BASE_URL: http://localhost:11434
  CLAUDE_CODE_FALLBACK_MODE: auto|manual|disabled
  CLAUDE_CODE_FALLBACK_THRESHOLD: 80  # Switch at 80% limit
  
  # Execution settings
  CLAUDE_CODE_DEFAULT_MODEL: claude-3-5-sonnet-latest
  CLAUDE_CODE_FALLBACK_MODEL: qwen2.5-coder:14b
  CLAUDE_CODE_MAX_RETRIES: 3
  CLAUDE_CODE_TIMEOUT: 600
  
  # Usage tracking
  CLAUDE_USAGE_FILE: ~/.claude/usage_tracking.json
  CLAUDE_CODE_TRACK_COSTS: true

config_files:
  - ~/.claude/config.json           # Global configuration
  - ~/.claude/litellm_config.json   # Fallback adapter config
  - ~/.claude/usage_tracking.json   # Usage and rate limit tracking
  - .claude/project.json            # Project-specific settings
```

## 📝 Implementation Status

### Completed
- [x] Basic CLI implementation with resource framework
- [x] Installation automation for Claude CLI
- [x] Status monitoring with health checks
- [x] Session management and persistence
- [x] Batch processing framework
- [x] MCP support for enhanced tools
- [x] Ollama fallback adapter implementation
- [x] Rate limit detection (JSON and text patterns)
- [x] Usage tracking and limit estimation
- [x] Automatic provider switching logic
- [x] Tool execution capabilities (native Claude)
- [x] Documentation structure

### In Progress
- [ ] Content management migration (inject → content)
- [ ] JSON output format for all commands
- [ ] Integration test suite with fallback scenarios
- [ ] Improved model selection heuristics

### Not Started
- [ ] Test result integration in status output
- [ ] Metrics and telemetry with OpenTelemetry
- [ ] Web monitoring interface
- [ ] Audit logging system for code changes
- [ ] Custom Ollama model fine-tuning
- [ ] Cost optimization recommendations

## 🚀 Deployment Notes

### Prerequisites
1. Node.js v18+ installed
2. Ollama resource installed and configured
3. API keys configured in Vault (ANTHROPIC_API_KEY)
4. Network access to Anthropic API
5. Sufficient disk space for sessions (~1GB)
6. GPU recommended for Ollama performance (optional)

### Installation Steps
```bash
# 1. Install the resource
vrooli resource install claude-code

# 2. Configure API keys in Vault
vrooli vault set ANTHROPIC_API_KEY "your-key-here"

# 3. Install Ollama for fallback support
vrooli resource install ollama

# 4. Pull recommended Ollama models
resource-ollama pull qwen2.5-coder:14b
resource-ollama pull codellama:7b

# 5. Verify installation
resource-claude-code status

# 6. Test with example prompt
resource-claude-code run "Write a hello world in Python"
```

### Monitoring
- Health checks: `resource-claude-code status`
- Usage tracking: `resource-claude-code usage`
- Session monitoring: `resource-claude-code session list`
- Logs: `~/.claude/logs/`
- Rate limit status: Automatic in status output

## 🔄 Fallback Mechanism

### Rate Limit Detection
```yaml
detection_patterns:
  json_format:
    - type: "result"
      is_error: true
      result: "Claude AI usage limit reached|<timestamp>"
  
  text_patterns:
    - "429" (HTTP status)
    - "rate_limit_error"
    - "usage limit"
    - "Too Many Requests"
    - "would exceed your.*limit"

detection_sources:
  - Claude CLI exit code (129 for rate limits)
  - Response content analysis
  - HTTP status codes
  - Error message parsing
```

### Fallback Strategy
```yaml
trigger_conditions:
  automatic:
    - Rate limit detected (immediate switch)
    - Usage > 80% of estimated limit (preemptive)
    - Claude service unavailable
    - Network connectivity issues
  
  manual:
    - User-initiated switch for cost savings
    - Testing/development mode
    - Privacy-sensitive operations

fallback_behavior:
  model_selection:
    - Primary: qwen2.5-coder:14b (best quality)
    - Fast: qwen2.5-coder:7b (quick responses)
    - Specialized: codellama:7b (code-specific)
  
  limitations:
    - No tool execution (text responses only)
    - Manual file operations required
    - Reduced context window
    - Different response format
  
  recovery:
    - Auto-return after 5-hour window
    - Manual switch back available
    - Usage tracking continues
```

## 📚 Documentation
- [API Documentation](docs/API.md)
- [Configuration Guide](docs/CONFIGURATION.md)
- [Fallback Guide](docs/FALLBACK.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Rate Limit Management](docs/RATE_LIMITS.md)
- [Automation Guide](docs/AUTOMATION.md)
- [Examples](examples/README.md)

## 💰 Infrastructure Value

### Technical Value
- **Always-Available Coding**: Fallback ensures 24/7 availability regardless of rate limits
- **Cost Optimization**: Automatic switch to free local models when appropriate
- **Privacy Options**: Sensitive code can be processed locally via Ollama
- **Tool Capabilities**: Claude's unique ability to execute actual file operations
- **Resilient Architecture**: Multiple providers ensure no single point of failure

### Resource Economics
- **Setup Cost**: ~30 minutes (including Ollama model downloads)
- **Operating Cost**: 
  - Claude: Based on subscription tier ($20-200/month)
  - Ollama: Free (local compute costs only)
- **Integration Value**: Multiplies productivity of all code-generation scenarios
- **ROI**: Typical 10x productivity gain for repetitive coding tasks

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Claude API rate limits | High | Medium | Automatic Ollama fallback with 95% success rate |
| Ollama model quality | Medium | Low | Multiple model options, user can review output |
| Network connectivity | Low | High | Local Ollama fallback, session persistence |
| API key exposure | Low | Critical | Vault integration, secure storage |
| Tool execution errors | Medium | Medium | Sandboxing, dry-run options, rollback support |

### Operational Risks
- **Configuration Drift**: Automated config validation on startup
- **Model Compatibility**: Regular testing of fallback models
- **Usage Overruns**: Preemptive switching at 80% threshold
- **Quality Degradation**: Clear indication when using fallback

## ✅ Validation Criteria

### Infrastructure Validation
- [x] Claude Code installs and authenticates successfully
- [x] Ollama fallback activates on rate limit
- [x] Tool execution works for file operations
- [x] Session persistence across restarts
- [x] Usage tracking accurately predicts limits
- [ ] All management actions documented

### Integration Validation
- [x] Seamless provider switching (<2s)
- [x] Ollama models provide useful responses
- [x] Rate limit detection works reliably
- [ ] Cost tracking matches actual usage
- [ ] Batch processing handles failures gracefully

### Operational Validation
- [x] Installation procedure documented
- [x] Fallback behavior predictable
- [ ] Performance metrics collected
- [ ] Troubleshooting guide complete
- [ ] Load testing completed

## 📝 Implementation Notes

### Design Decisions
**Ollama-Code as Primary Fallback**: 
- Alternative considered: OpenRouter, LiteLLM proxy
- Decision driver: Local execution, no costs, complete privacy
- Trade-offs: Lower capability but always available

**Direct API vs Proxy Approach**:
- Alternative considered: LiteLLM proxy for all requests
- Decision driver: Tool execution requires native Claude
- Trade-offs: More complex but preserves full functionality

**Usage Tracking Implementation**:
- Alternative considered: Server-side tracking
- Decision driver: Local tracking for privacy and immediacy
- Trade-offs: Estimates vs exact counts

### Known Limitations
- **Ollama Cannot Execute Tools**: Text-only responses during fallback
  - Workaround: Parse commands from response and execute manually
  - Future fix: Implement local tool execution layer
  
- **Context Loss on Provider Switch**: Full context not transferred
  - Workaround: Summary generation before switch
  - Future fix: Context compression and transfer

- **Model Quality Variance**: Ollama models vary in capability
  - Workaround: Multiple models available for selection
  - Future fix: Fine-tuned models for specific tasks

### Integration Considerations
- **Ollama Resource Dependency**: Must be installed for fallback
- **Network Isolation**: Can operate fully offline with Ollama
- **GPU Utilization**: Shared between Claude and Ollama operations
- **Session Storage**: ~100MB per active session

## 🔗 References

### Documentation
- README.md - Quick start and overview
- docs/FALLBACK.md - Detailed fallback configuration
- docs/RATE_LIMITS.md - Rate limit management strategies
- lib/common.sh - Core utility functions including rate detection
- adapters/litellm/ - Fallback adapter implementation

### Related Resources
- **ollama** - Provides local LLM inference for fallback
- **vault** - Secure API key storage
- **litellm** - Optional proxy for model routing
- **n8n** - Workflow automation using Claude Code

### External Resources
- [Claude Documentation](https://docs.anthropic.com/claude/docs)
- [Ollama Models](https://ollama.com/library)
- [Model Context Protocol](https://modelcontextprotocol.io)
- [Claude Code CLI](https://github.com/anthropics/claude-code)

---

**Last Updated**: 2025-08-22  
**Status**: Validated  
**Owner**: AI Infrastructure Team  
**Review Cycle**: Monthly